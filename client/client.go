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

const CALL_ID = "_CALLID"
const CALL = "CALL"
const LLAC = "LLAC"

type Client struct {
	status uint32
	conn   *conn.Conn

	opMutex sync.Mutex
	sending map[string]*BaseToken
	calling map[string]*CallToken
	recvs   map[string]recv
	llacs   map[string]llac
}

type recv struct {
	label    []string
	callback func(nson.Message)
}

type llac struct {
	label    []string
	callback func(nson.Message) nson.Message
}

func NewClient(config conn.Config) (*Client, error) {
	client := &Client{
		sending: make(map[string]*BaseToken),
		calling: make(map[string]*CallToken),
		recvs:   make(map[string]recv),
		llacs:   make(map[string]llac),
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

func (c *Client) IsDisConnect() bool {
	status := atomic.LoadUint32(&c.status)
	if status == disconnected {
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

		// fmt.Println(msg.String())

		if ch, err := msg.GetString(conn.CHAN); err == nil {
			if code, err := msg.GetI32(conn.CODE); err == nil {
				if id, err := msg.GetMessageId(conn.ID); err == nil {
					c.opMutex.Lock()
					sendToken, ok := c.sending[id.Hex()]
					if ok {
						delete(c.sending, id.Hex())
					}
					c.opMutex.Unlock()

					if ok {
						if code == 0 {
							sendToken.flowComplete()
						} else {
							sendToken.setError(fmt.Errorf("error code: %v", code))
						}

						continue
					}
				} else if callId, err := msg.GetMessageId(CALL_ID); err == nil {
					c.opMutex.Lock()
					callToken, ok := c.calling[callId.Hex()]
					if ok {
						delete(c.calling, callId.Hex())
					}
					c.opMutex.Unlock()

					if ok {
						if code != 0 {
							callToken.setError(fmt.Errorf("error code: %v", code))
						}

						continue
					}
				}
			}

			if ch == LLAC {
				if callId, err := msg.GetMessageId(CALL_ID); err == nil {
					c.opMutex.Lock()
					if callToken, ok := c.calling[callId.Hex()]; ok {
						delete(c.calling, callId.Hex())
						callToken.setMessage(msg)
					}
					c.opMutex.Unlock()
				}
			} else {
				c.opMutex.Lock()
				if recv, ok := c.recvs[ch]; ok {
					go recv.callback(msg)
				} else if llac, ok := c.llacs[ch]; ok {
					go func() {
						if from, err := msg.GetMessageId(conn.FROM); err == nil {
							if callId, err := msg.GetMessageId(CALL_ID); err == nil {
								res_msg := llac.callback(msg)

								res_msg.Insert(conn.CHAN, nson.String(LLAC))
								res_msg.Insert(conn.TO, from)
								res_msg.Insert(CALL_ID, callId)

								c.conn.SendMessage(res_msg)
							}
						}
					}()
				}

				c.opMutex.Unlock()
			}
		} else {
			log.Printf("message format error: %s", msg.String())
		}
	}
}

func (c *Client) Send(
	ch string,
	message nson.Message,
	label []string,
	to []nson.MessageId,
	timeout time.Duration,
) error {
	if ch == "" {
		return errors.New("消息频道不能为空")
	}

	if message == nil {
		return errors.New("消息不能为 nil")
	}

	// 消息ID
	id, err := message.GetMessageId(conn.ID)
	if err != nil {
		id = nson.NewMessageId()

		message.Insert(conn.ID, id)
	}

	// CHAN
	message.Insert(conn.CHAN, nson.String(ch))

	// LABEL
	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		message.Insert(conn.LABEL, nson.Array(array))
	}

	// TO
	if to != nil && len(to) > 0 {
		array := make(nson.Array, 0)
		for _, v := range to {
			array = append(array, nson.String(v))
		}
		message.Insert(conn.TO, nson.Array(array))
	}

	if timeout == time.Duration(0) {
		return c.conn.SendMessage(message)
	}

	// message.Insert(conn.ACK, nson.Bool(true))

	token := newBaseToken()
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

	err = c.conn.SendMessage(message)
	if err != nil {
		return err
	}

	if !token.WaitTimeout(timeout) {
		return errors.New("timeout")
	}

	return token.Error()
}

func (c *Client) detach(ch string) error {
	msg := nson.Message{
		conn.CHAN:  nson.String(conn.DETACH),
		conn.VALUE: nson.String(ch),
	}

	return c.conn.SendMessage(msg)
}

func (c *Client) attach(ch string, label []string) error {
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

	return c.conn.SendMessage(msg)
}

func (c *Client) Recv(
	ch string,
	label []string,
	callback func(nson.Message),
) error {
	if ch == "" {
		return errors.New("消息频道不能为空")
	}

	err := c.RemoveRecv(ch)
	if err != nil {
		return err
	}

	c.opMutex.Lock()
	c.recvs[ch] = recv{
		label:    label,
		callback: callback,
	}
	c.opMutex.Unlock()

	return c.attach(ch, label)
}

func (c *Client) RemoveRecv(ch string) error {
	if ch == "" {
		return errors.New("消息频道不能为空")
	}

	c.opMutex.Lock()
	_, ok := c.recvs[ch]
	if ok {
		delete(c.recvs, ch)
	}
	c.opMutex.Unlock()

	if ok {
		return c.detach(ch)
	}

	return nil
}

func (c *Client) Call(
	ch string,
	message nson.Message,
	label []string,
	to []nson.MessageId,
	timeout time.Duration,
) (nson.Message, error) {
	if ch == "" {
		return nil, errors.New("消息频道不能为空")
	}

	if message == nil {
		return nil, errors.New("消息不能为 nil")
	}

	callId := nson.NewMessageId()

	message.Insert(CALL_ID, callId)

	// CHAN
	ch2 := CALL + "." + ch
	message.Insert(conn.CHAN, nson.String(ch2))

	// LABEL
	if label != nil && len(label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range label {
			array = append(array, nson.String(v))
		}
		message.Insert(conn.LABEL, nson.Array(array))
	}

	// TO
	if to != nil && len(to) > 0 {
		array := make(nson.Array, 0)
		for _, v := range to {
			array = append(array, nson.String(v))
		}
		message.Insert(conn.TO, nson.Array(array))
	}

	if timeout == time.Duration(0) {
		timeout = time.Second * 60
	}

	// share
	message.Insert(conn.SHARE, nson.Bool(true))

	token := newCallToken()
	c.opMutex.Lock()
	c.calling[callId.Hex()] = token
	c.opMutex.Unlock()
	defer func() {
		c.opMutex.Lock()
		if token, ok := c.calling[callId.Hex()]; ok {
			token.flowComplete()
			delete(c.calling, callId.Hex())
		}
		c.opMutex.Unlock()
	}()

	err := c.conn.SendMessage(message)
	if err != nil {
		return nil, err
	}

	if !token.WaitTimeout(timeout) {
		return nil, errors.New("timeout")
	}

	err = token.Error()
	if err != nil {
		return nil, err
	}

	return token.Message(), nil
}

func (c *Client) Llac(
	ch string,
	label []string,
	callback func(nson.Message) nson.Message,
) error {
	if ch == "" {
		return errors.New("消息频道不能为空")
	}

	err := c.RemoveLlac(ch)
	if err != nil {
		return err
	}

	ch = CALL + "." + ch

	c.opMutex.Lock()
	c.llacs[ch] = llac{
		label:    label,
		callback: callback,
	}
	c.opMutex.Unlock()

	return c.attach(ch, label)
}

func (c *Client) RemoveLlac(ch string) error {
	if ch == "" {
		return errors.New("消息频道不能为空")
	}

	ch = CALL + "." + ch

	c.opMutex.Lock()
	_, ok := c.llacs[ch]
	if ok {
		delete(c.llacs, ch)
	}
	c.opMutex.Unlock()

	if ok {
		return c.detach(ch)
	}

	return nil
}
