package client

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/dict"
	"go.uber.org/zap"
)

const CALL_ID = "_call"

type Client struct {
	options Options

	onConnect    func(*Client)
	onDisConnect func()
	onRecv       func(*Client, RecvMessage)
	opLock       sync.RWMutex
	connectOnce  sync.Once

	sending  map[string]*Token
	sendLock sync.Mutex

	wire     *Wire
	wireLock sync.RWMutex

	crypto *crypto.Crypto

	close chan struct{}
}

type Options struct {
	Addrs             []string
	EnableCrypto      bool
	CryptoMethod      crypto.Method
	SecretKey         string
	SlotId            nson.MessageId
	Root              bool
	Attr              nson.Message
	HandTimeout       uint32
	KeepAlive         uint32
	HeartbeatInterval uint32
	Logger            *zap.Logger
}

func NewClient(o Options) (*Client, error) {
	if o.Addrs == nil || len(o.Addrs) == 0 {
		return nil, errors.New("must provide addr")
	}

	if o.SlotId == nil {
		o.SlotId = nson.NewMessageId()
	}

	if o.Attr == nil {
		o.Attr = nson.Message{}
	}

	o.Attr.Insert(dict.SLOT_ID, o.SlotId)
	o.Attr.Insert(dict.ROOT, nson.Bool(o.Root))

	if o.HandTimeout == 0 {
		o.HandTimeout = 10
	}

	if o.KeepAlive == 0 {
		o.KeepAlive = 60
	}

	if o.HeartbeatInterval == 0 {
		o.HeartbeatInterval = 10
	}

	if o.Logger == nil {
		o.Logger = zap.NewNop()
	}

	c := &Client{
		options: o,
		sending: make(map[string]*Token),
		close:   make(chan struct{}),
	}

	if o.EnableCrypto {
		if o.CryptoMethod == crypto.None {
			o.CryptoMethod = crypto.ChaCha20Poly1305
		}

		crypto, err := crypto.NewCrypto(o.CryptoMethod, o.SecretKey)
		if err != nil {
			return nil, err
		}

		c.crypto = crypto
	}

	return c, nil
}

func (c *Client) IsConnect() bool {
	c.wireLock.RLock()
	defer c.wireLock.RUnlock()

	return c.wire != nil
}

func (c *Client) DisConnect() {
	c.wireLock.Lock()
	defer c.wireLock.Unlock()

	if c.wire != nil {
		c.wire.Close()
		c.wire = nil
	}
}

func (c *Client) OnDisConnect(callback func()) {
	c.opLock.Lock()
	c.onDisConnect = callback
	c.opLock.Unlock()
}

func (c *Client) Connect(callback func(*Client)) {
	c.opLock.Lock()
	c.onConnect = callback
	c.opLock.Unlock()

	c.connectOnce.Do(func() {
		go func() {
			for {
				select {
				case <-c.close:
					return
				default:
					if !c.IsConnect() {
						func() {
							c.wireLock.Lock()
							defer c.wireLock.Unlock()

							wire, err := NewWire(
								c.options.Addrs,
								c.crypto,
								c.options.Attr,
								c.options.HandTimeout,
								c.options.KeepAlive,
								c.options.HeartbeatInterval,
								c.recv,
								c.options.Logger,
							)

							if err != nil {
								c.options.Logger.Error(err.Error())
								return
							}

							c.options.Logger.Info("connect", zap.String("addrs", fmt.Sprintf("%s", c.options.Addrs)))
							err = wire.Connect()
							if err != nil {
								c.options.Logger.Error(err.Error())
								return
							}

							c.options.Logger.Info("connect success", zap.String("addr", fmt.Sprintf("%s", wire.conn.RemoteAddr())))

							c.wire = wire
						}()

						c.wireLock.RLock()
						wire := c.wire
						c.wireLock.RUnlock()

						if wire != nil {
							c.opLock.RLock()
							f := c.onConnect
							c.opLock.RUnlock()

							go f(c)

							// when the connection is broken, this function will return
							wire.Run()

							c.options.Logger.Debug("disconnect")

							c.wireLock.Lock()
							c.wire = nil
							c.wireLock.Unlock()

							// call disconnect callback
							c.opLock.RLock()
							f2 := c.onDisConnect
							c.opLock.RUnlock()

							if f2 != nil {
								f2()
							}
						}
					}

					time.Sleep(time.Duration(c.options.HeartbeatInterval) * time.Second)
				}
			}
		}()
	})
}

func (c *Client) Recv(callback func(*Client, RecvMessage)) {
	c.opLock.Lock()
	c.onRecv = callback
	c.opLock.Unlock()
}

func (c *Client) RawSend(msg nson.Message, timeout time.Duration) (nson.Message, error) {
	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	if timeout == time.Duration(0) {
		timeout = time.Second * 10
	}

	id, err := msg.GetMessageId(CALL_ID)
	if err != nil {
		id = nson.NewMessageId()

		msg.Insert(CALL_ID, id)
	}

	token := newToken()
	c.sendLock.Lock()
	c.sending[id.Hex()] = token
	c.sendLock.Unlock()

	defer func() {
		c.sendLock.Lock()
		if token, ok := c.sending[id.Hex()]; ok {
			token.flowComplete()
			delete(c.sending, id.Hex())
		}
		c.sendLock.Unlock()
	}()

	c.wireLock.RLock()
	if c.wire != nil {
		c.wire.Send(msg)
	} else {
		c.wireLock.RUnlock()
		return nil, errors.New("disconnected")
	}
	c.wireLock.RUnlock()

	if !token.WaitTimeout(timeout) {
		return nil, errors.New("timeout")
	}

	return token.Message(), token.Error()
}

func (c *Client) Detach(ch string, label []string, share bool) error {
	msg := nson.Message{
		dict.CHAN:  nson.String(dict.DETACH),
		dict.VALUE: nson.String(ch),
	}

	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		msg.Insert(dict.LABEL, nson.Array(array))
	}

	if share {
		msg.Insert(dict.SHARE, nson.Bool(true))
	}

	_, err := c.RawSend(msg, 0)
	return err
}

func (c *Client) Attach(ch string, label []string, share bool) error {
	msg := nson.Message{
		dict.CHAN:  nson.String(dict.ATTACH),
		dict.VALUE: nson.String(ch),
	}

	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		msg.Insert(dict.LABEL, nson.Array(array))
	}

	if share {
		msg.Insert(dict.SHARE, nson.Bool(true))
	}

	_, err := c.RawSend(msg, 0)
	return err
}

func (c *Client) Send(
	message *SendMessage,
	timeout time.Duration,
) (nson.Message, error) {
	if message == nil {
		return nil, errors.New("message cannot be nil")
	}

	msg := message.build()

	if message.Call() {
		return c.RawSend(msg, timeout)
	}

	c.wireLock.RLock()
	if c.wire != nil {
		c.wire.Send(msg)
	} else {
		c.wireLock.RUnlock()
		return nil, errors.New("disconnected")
	}
	c.wireLock.RUnlock()

	return nil, nil
}

func (c *Client) Close() {
	select {
	case <-c.close:
		return
	default:
		close(c.close)
		go c.DisConnect()
	}
}

func (c *Client) recv(ch string, msg nson.Message) {
	if code, err := msg.GetI32(dict.CODE); err == nil {
		if id, err := msg.GetMessageId(CALL_ID); err == nil {
			c.sendLock.Lock()
			sendToken, ok := c.sending[id.Hex()]
			if ok {
				delete(c.sending, id.Hex())
			}
			c.sendLock.Unlock()

			if ok {
				if code == 0 {
					sendToken.setMessage(msg, nil)
				} else {
					sendToken.setMessage(msg, fmt.Errorf("error code: %v", code))
				}
			}
		}
	} else {
		c.opLock.RLock()
		callback := c.onRecv
		c.opLock.RUnlock()
		if callback != nil {
			go callback(c, RecvMessage{ch, msg})
		}
	}
}
