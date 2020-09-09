package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/conn"
)

const (
	disconnected uint32 = iota
	connected
)

const CALL_ID = "_call"

type Client struct {
	status uint32
	conn   *conn.Conn

	opMutex  sync.Mutex
	sending  map[string]*Token
	recvChan chan *RecvMessage

	onConnectCallback func()
}

func NewClient(config conn.Config) (*Client, error) {
	client := &Client{
		sending:  make(map[string]*Token),
		recvChan: make(chan *RecvMessage, 64),
	}

	config.OnConnect = func() {
		client.onConnect()
	}

	config.OnDisConnect = func() {
		client.onDisConnect()
	}

	bc, err := conn.Dial(config)
	if err != nil {
		return nil, err
	}

	client.conn = bc

	go client.recv()

	return client, nil
}

func (c *Client) onConnect() {
	atomic.StoreUint32(&c.status, uint32(connected))

	c.opMutex.Lock()
	callback := c.onConnectCallback
	c.opMutex.Unlock()

	if callback != nil {
		callback()
	}
}

func (c *Client) OnConnect(callback func()) {
	c.opMutex.Lock()
	c.onConnectCallback = callback
	c.opMutex.Unlock()
}

func (c *Client) onDisConnect() {
	atomic.StoreUint32(&c.status, uint32(disconnected))
}

func (c *Client) IsConnect() bool {
	status := atomic.LoadUint32(&c.status)
	if status == connected {
		return true
	}

	return false
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) recv() {
	for {
		msg, err := c.conn.RecvMessage()
		if err != nil {
			log.Println("recv exit")
			return
		}

		if ch, err := msg.GetString(conn.CHAN); err == nil {
			if code, err := msg.GetI32(conn.CODE); err == nil {
				if id, err := msg.GetMessageId(CALL_ID); err == nil {
					c.opMutex.Lock()
					sendToken, ok := c.sending[id.Hex()]
					if ok {
						delete(c.sending, id.Hex())
					}
					c.opMutex.Unlock()

					if ok {
						if code == 0 {
							sendToken.setMessage(msg)
						} else {
							sendToken.setError(fmt.Errorf("error code: %v", code))
						}

						continue
					}
				}
			} else {
				c.recvChan <- &RecvMessage{ch, msg}
			}

		} else {
			log.Printf("message format error: %s", msg.String())
		}
	}
}

func (c *Client) RawSend(msg nson.Message, timeout time.Duration) (nson.Message, error) {
	if msg == nil {
		return nil, errors.New("消息不能为 nil")
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
	c.opMutex.Lock()
	c.sending[id.Hex()] = token
	c.opMutex.Unlock()

	defer func() {
		c.opMutex.Lock()
		if token, ok := c.sending[id.Hex()]; ok {
			token.flowComplete()
			delete(c.sending, id.Hex())
		}
		c.opMutex.Unlock()
	}()

	err = c.conn.SendMessage(msg)
	if err != nil {
		return nil, err
	}

	if !token.WaitTimeout(timeout) {
		return nil, errors.New("timeout")
	}

	return token.Message(), token.Error()
}

func (c *Client) Detach(ch string, label []string, timeout time.Duration) error {
	msg := nson.Message{
		conn.CHAN:  nson.String(conn.DETACH),
		conn.VALUE: nson.String(ch),
	}

	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		msg.Insert(conn.LABEL, nson.Array(array))
	}

	_, err := c.RawSend(msg, timeout)
	return err
}

func (c *Client) Attach(ch string, label []string, timeout time.Duration) error {
	msg := nson.Message{
		conn.CHAN:  nson.String(conn.ATTACH),
		conn.VALUE: nson.String(ch),
	}

	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		msg.Insert(conn.LABEL, nson.Array(array))
	}

	_, err := c.RawSend(msg, timeout)
	return err
}

func (c *Client) Send(
	message *SendMessage,
	timeout time.Duration,
) (nson.Message, error) {

	msg := message.Build()

	if message.Call {
		return c.RawSend(msg, timeout)
	}

	return nil, c.conn.SendMessage(msg)
}

func (c *Client) Recv() <-chan *RecvMessage {
	return c.recvChan
}
