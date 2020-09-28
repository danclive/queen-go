package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/dict"
	"github.com/danclive/queen-go/util"
	"go.uber.org/zap"
)

type Wire struct {
	// client        *Client
	wg            sync.WaitGroup
	conn          net.Conn
	in            chan nson.Message
	out           chan nson.Message
	close         chan struct{}
	closeComplete chan struct{}
	error         chan error
	err           error

	lastRecvTime time.Time
	opLock       sync.RWMutex

	// config
	addrs             []string
	crypto            *crypto.Crypto
	handMessage       nson.Message
	handTimeout       uint32
	keepAlive         uint32
	heartbeatInterval uint32
	onRecv            func(string, nson.Message)

	logger *zap.Logger
}

func NewWire(
	addrs []string,
	crypto *crypto.Crypto,
	handMessage nson.Message,
	handTimeout uint32,
	keepAlive uint32,
	heartbeatInterval uint32,
	onRecv func(string, nson.Message),
	logger *zap.Logger,
) (*Wire, error) {
	if addrs == nil || len(addrs) == 0 {
		return nil, errors.New("must provide addr")
	}

	if onRecv == nil {
		return nil, errors.New("must provide on recv callback")
	}

	if handTimeout == 0 {
		handTimeout = 10
	}

	if keepAlive == 0 {
		keepAlive = 60
	}

	if heartbeatInterval == 0 {
		heartbeatInterval = 5
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	w := &Wire{
		in:                make(chan nson.Message),
		out:               make(chan nson.Message),
		close:             make(chan struct{}),
		closeComplete:     make(chan struct{}),
		error:             make(chan error, 1),
		addrs:             addrs,
		crypto:            crypto,
		handMessage:       handMessage,
		handTimeout:       handTimeout,
		keepAlive:         keepAlive,
		heartbeatInterval: heartbeatInterval,
		onRecv:            onRecv,
		logger:            logger,
	}

	return w, nil
}

func (w *Wire) Close() <-chan struct{} {
	w.setError(nil)
	return w.closeComplete
}

func (w *Wire) setError(err error) {
	select {
	case w.error <- err:
		if err != nil && err != io.EOF {
			w.logger.Error("connection lost", zap.String("error_msg", err.Error()))
		}
	default:
	}
}

func (w *Wire) Connect() error {
	var err error

	// connect
	conn, err := dial(w.addrs)
	if err != nil {
		return err
	}

	// handshake
	msg := nson.Message{
		dict.CHAN: nson.String(dict.HAND),
	}

	if w.crypto != nil {
		msg.Insert(dict.METHOD, nson.String(w.crypto.Method().ToString()))
	}

	msg.Extend(w.handMessage)

	buf := new(bytes.Buffer)
	err = msg.Encode(buf)
	if err != nil {
		return err
	}

	conn.SetReadDeadline(time.Now().Add(time.Duration(w.handTimeout) * time.Second))

	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	var lbuf [4]byte
	if _, err := io.ReadFull(conn, lbuf[:]); err != nil {
		return err
	}

	len := int(util.GetU32(lbuf[:], 0))
	if len < 5 || len > dict.MAX_MESSAGE_LEN {
		return errors.New("invalid data")
	}

	rbuf := make([]byte, len)
	if _, err := io.ReadFull(conn, rbuf[4:]); err != nil {
		return err
	}

	rbuf[0] = lbuf[0]
	rbuf[1] = lbuf[1]
	rbuf[2] = lbuf[2]
	rbuf[3] = lbuf[3]

	rbuffer := bytes.NewBuffer(rbuf)
	value, err := nson.Message{}.Decode(rbuffer)
	if err != nil {
		return err
	}

	if value.Tag() != nson.TAG_MESSAGE {
		return errors.New("invalid data")
	}

	msg2 := value.(nson.Message)

	code, err := msg2.GetI32(dict.CODE)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("error code %v", code)
	}

	w.conn = conn

	// ping
	conn.SetReadDeadline(time.Now().Add(time.Duration(w.handTimeout) * time.Second))

	ping_msg := nson.Message{
		dict.CHAN: nson.String(dict.PING),
	}

	err = w.write(ping_msg)
	if err != nil {
		return err
	}

	rmsg, err := w.read()
	if err != nil {
		return err
	}

	code, err = rmsg.GetI32(dict.CODE)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("error code %v", code)
	}

	return nil
}

func dial(addrs []string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for _, addr := range addrs {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			return conn, nil
		}
	}

	return conn, err
}

func (w *Wire) write(msg nson.Message) error {
	buf := new(bytes.Buffer)
	err := msg.Encode(buf)
	if err != nil {
		return err
	}

	if w.crypto != nil {
		data, err := w.crypto.Encrypt(buf.Bytes())
		if err != nil {
			return err
		}

		_, err = w.conn.Write(data)
		if err != nil {
			return err
		}
	} else {
		_, err = w.conn.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Wire) read() (nson.Message, error) {
	var lbuf [4]byte
	_, err := io.ReadFull(w.conn, lbuf[:])
	if err != nil {
		return nil, err
	}

	len := int(util.GetU32(lbuf[:], 0))
	if len < 5 || len > dict.MAX_MESSAGE_LEN {
		return nil, errors.New("invalid data")
	}

	rbuf := make([]byte, len)
	if _, err := io.ReadFull(w.conn, rbuf[4:]); err != nil {
		return nil, err
	}

	rbuf[0] = lbuf[0]
	rbuf[1] = lbuf[1]
	rbuf[2] = lbuf[2]
	rbuf[3] = lbuf[3]

	if w.crypto != nil {
		rbuf, err = w.crypto.Decrypt(rbuf)
		if err != nil {
			return nil, err
		}
	}

	rbuffer := bytes.NewBuffer(rbuf)
	value, err := nson.Message{}.Decode(rbuffer)
	if err != nil {
		return nil, err
	}

	if value.Tag() != nson.TAG_MESSAGE {
		return nil, errors.New("invalid data")
	}

	w.setLastRecvTime()

	return value.(nson.Message), nil
}

func (w *Wire) setLastRecvTime() {
	w.opLock.Lock()
	w.lastRecvTime = time.Now()
	w.opLock.Unlock()
}

func (w *Wire) LastRecvTime() time.Time {
	w.opLock.RLock()
	defer w.opLock.RUnlock()
	return w.lastRecvTime
}

func (w *Wire) writeLoop() {
	var err error

	defer func() {
		if re := recover(); re != nil {
			err = errors.New(fmt.Sprint(re))
		}
		w.setError(err)
		w.wg.Done()
		w.logger.Debug("write loop thread exit")
	}()

	for {
		select {
		case <-w.close:
			return
		case msg := <-w.out:

			err = w.write(msg)
			if err != nil {
				return
			}
		}
	}
}

func (w *Wire) readLoop() {
	var err error

	defer func() {
		if re := recover(); re != nil {
			err = errors.New(fmt.Sprint(re))
		}
		w.setError(err)
		w.wg.Done()
		w.logger.Debug("read loop thread exit")
	}()

	for {
		w.conn.SetReadDeadline(time.Now().Add(time.Duration(w.keepAlive+w.keepAlive/2) * time.Second))

		var msg nson.Message

		msg, err = w.read()
		if err != nil {
			return
		}

		w.in <- msg
	}
}

func (w *Wire) errorWatch() {
	defer func() {
		w.wg.Done()
		w.logger.Debug("error watch thread exit")
	}()

	select {
	case <-w.close:
		return
	case err := <-w.error: //有错误关闭
		w.err = err
		w.conn.Close()
		close(w.close) //退出chanel
		return
	}
}

func (w *Wire) Send(msg nson.Message) {
	select {
	case <-w.close:
		return
	case w.out <- msg:
		w.logger.Debug("send", zap.String("msg:", msg.String()))
	}
}

func (w *Wire) recv() {
	var err error

	defer func() {
		if re := recover(); re != nil {
			err = errors.New(fmt.Sprint(re))
		}
		w.setError(err)
		w.wg.Done()
		w.logger.Debug("recv thread exit")
	}()

	for {
		select {
		case <-w.close:
			return
		case msg := <-w.in:
			w.logger.Debug("recv", zap.String("msg:", msg.String()))

			if ch, err := msg.GetString(dict.CHAN); err == nil {
				if ch == dict.KEEP_ALIVE {
					w.logger.Debug("recv keep alive msg")
					if !msg.Contains(dict.CODE) {
						msg.Insert(dict.CODE, nson.I32(0))
						w.Send(msg)
					}
				} else {
					w.onRecv(ch, msg)
				}
			} else {
				err = fmt.Errorf("invalid message format: %s", msg.String())
				return
			}
		}
	}
}

func (w *Wire) heartbeat() {
	defer func() {
		w.wg.Done()
		w.logger.Debug("heartbeat thread exit")
	}()

	ticker := time.NewTicker(time.Duration(w.heartbeatInterval) * time.Second)

	for {
		select {
		case <-w.close:
			return
		case t := <-ticker.C:
			w.logger.Debug("heartbeat")
			if t.Sub(w.LastRecvTime()) > time.Duration(w.keepAlive)*time.Second {
				w.logger.Debug("send keep alive msg")
				msg := nson.Message{
					dict.CHAN: nson.String(dict.KEEP_ALIVE),
				}

				w.Send(msg)
			}
		}
	}
}

func (w *Wire) Run() {
	defer close(w.closeComplete)

	w.wg.Add(5)

	go w.errorWatch()
	go w.readLoop()
	go w.writeLoop()
	go w.recv()
	go w.heartbeat()

	w.wg.Wait()
}
