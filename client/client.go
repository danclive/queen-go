package client

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/circ"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/dict"
	"github.com/danclive/queen-go/wire"
	"go.uber.org/zap"
)

const (
	CALL_ID = "_call"
)

var (
	BufferSize = 1024 * 1024 * 8 // the size of the buffer in bytes.
	BlockSize  = 1024 * 8        // the size per R/W block in bytes.

	ErrClientClosed     = errors.New("Client not open")
	ErrClientDisConnect = errors.New("Client not connect")
)

type Client struct {
	options Options

	onConnect    func(*Client)
	onDisConnect func(*Client)
	onRecv       func(*Client, RecvMessage)
	opLock       sync.RWMutex
	connectOnce  sync.Once

	sending  map[string]*Token
	sendLock sync.Mutex

	wire     *wire.Wire
	wireLock sync.RWMutex

	crypto *crypto.Crypto

	State State // the operational state of the client.
	close chan struct{}
}

type Options struct {
	Addrs        []string
	EnableCrypto bool
	CryptoMethod crypto.Method
	SecretKey    string
	SlotId       nson.MessageId
	Root         bool
	Attr         nson.Message
	KeepAlive    uint32
	Logger       *zap.Logger
}

// State tracks the state of the Wire.
type State struct {
	Done    int32     // atomic counter which indicates that the wire has closed.
	endOnce sync.Once // only end once.
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

	if o.KeepAlive == 0 {
		o.KeepAlive = 30
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

func (c *Client) OnDisConnect(callback func(*Client)) {
	c.opLock.Lock()
	c.onDisConnect = callback
	c.opLock.Unlock()
}

func (c *Client) OnConnect(callback func(*Client)) {
	c.opLock.Lock()
	c.onConnect = callback
	c.opLock.Unlock()
}

func (c *Client) OnRecv(callback func(*Client, RecvMessage)) {
	c.opLock.Lock()
	c.onRecv = callback
	c.opLock.Unlock()
}

func (c *Client) Connect() {
	c.connectOnce.Do(func() {
		for {
			select {
			case <-c.close:
				return
			default:
				ticker := backoff.NewTicker(backoff.NewExponentialBackOff())

				var conn net.Conn
				var err error

				for range ticker.C {
					if conn, err = dial(c.options.Addrs); err != nil {
						c.options.Logger.Error("dial error, will retry...", zap.Error(err))
						continue
					}

					ticker.Stop()
					break
				}

				w := wire.NewWire(conn, circ.NewReader(BufferSize, BlockSize), circ.NewWriter(BufferSize, BlockSize), c.options.SlotId, c.crypto, 0)

				if err = w.Handshake(c.options.Attr); err != nil {
					c.options.Logger.Error("handshake error, will retry...", zap.Error(err))
					continue
				}

				ping := nson.Message{dict.CHAN: nson.String(dict.PING)}
				if _, err = w.Write(ping); err != nil {
					c.options.Logger.Error("ping error, will retry...", zap.Error(err))
					continue
				}

				c.wireLock.Lock()
				c.wire = w
				c.wireLock.Unlock()

				c.opLock.RLock()
				onConnect := c.onConnect
				c.opLock.RUnlock()

				if onConnect != nil {
					go onConnect(c)
				}

				err = w.Read(c.recv)
				fmt.Println(err)

				c.wireLock.Lock()
				c.wire = nil
				c.wireLock.Unlock()

				if err != nil {
					c.options.Logger.Error("read error, will retry...", zap.Error(err))
				}

				c.opLock.RLock()
				onDisConnect := c.onDisConnect
				c.opLock.RUnlock()

				if onDisConnect != nil {
					go onDisConnect(c)
				}
			}
		}
	})
}

func (c *Client) Stop() {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return
	}

	c.State.endOnce.Do(func() {
		close(c.close)

		c.wireLock.Lock()
		c.wire.Stop()
		c.wireLock.Unlock()

		atomic.StoreInt32(&c.State.Done, 1)
	})
}

func (c *Client) Call(msg nson.Message, timeout time.Duration) (nson.Message, error) {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return nil, ErrClientClosed
	}

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
		c.wire.Write(msg)
	} else {
		c.wireLock.RUnlock()
		return nil, ErrClientDisConnect
	}
	c.wireLock.RUnlock()

	if !token.WaitTimeout(timeout) {
		return nil, errors.New("timeout")
	}

	return token.Message(), token.Error()
}

func (c *Client) Detach(ch string, share bool) error {
	msg := nson.Message{
		dict.CHAN:  nson.String(dict.DETACH),
		dict.VALUE: nson.String(ch),
	}

	if share {
		msg.Insert(dict.SHARE, nson.Bool(true))
	}

	_, err := c.Call(msg, 0)
	return err
}

func (c *Client) Attach(ch string, share bool) error {
	msg := nson.Message{
		dict.CHAN:  nson.String(dict.ATTACH),
		dict.VALUE: nson.String(ch),
	}

	if share {
		msg.Insert(dict.SHARE, nson.Bool(true))
	}

	_, err := c.Call(msg, 0)
	return err
}

func (c *Client) Send(
	message *SendMessage,
	timeout time.Duration,
) (nson.Message, error) {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return nil, ErrClientClosed
	}

	if message == nil {
		return nil, errors.New("message cannot be nil")
	}

	msg := message.build()

	if message.Call() {
		return c.Call(msg, timeout)
	}

	c.wireLock.RLock()
	defer c.wireLock.RUnlock()
	if c.wire != nil {
		c.wire.Write(msg)
	} else {
		return nil, ErrClientDisConnect
	}

	return nil, nil
}

func (c *Client) recv(_ *wire.Wire, ch string, msg nson.Message) error {
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

	return nil
}

func dial(addrs []string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for _, addr := range addrs {
		conn, err = net.DialTimeout("tcp", addr, time.Second*10)
		if err == nil {
			return conn, nil
		}
	}

	return conn, err
}
