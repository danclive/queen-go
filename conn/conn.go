package conn

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/util"
)

type Config struct {
	Addrs             []string
	SlotId            nson.MessageId
	EnableCrypto      bool
	CryptoMethod      crypto.Method
	AccessKey         string
	SecretKey         string
	AuthMessage       nson.Message
	HandshakeTimeout  time.Duration
	ReconnWaitTimeout time.Duration
	ReconnInterval    time.Duration
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	onConnect         func()
	onDisConnect      func()
	Debug             bool
}

func (cfg *Config) init() {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"127.0.0.1:8888"}
	}

	if cfg.SlotId == nil {
		cfg.SlotId = nson.NewMessageId()
	}

	if cfg.AuthMessage == nil {
		cfg.AuthMessage = nson.Message{}
	}

	cfg.AuthMessage.Insert(CHAN, nson.String(AUTH))
	cfg.AuthMessage.Insert(SLOT_ID, cfg.SlotId)

	if cfg.EnableCrypto {
		if cfg.CryptoMethod == crypto.None || cfg.AccessKey == "" || cfg.SecretKey == "" {
			log.Fatalln("如果开启了加密，必须提供 CryptoMethod， AccessKey 和 SecretKey")
		}
	}

	if cfg.HandshakeTimeout == time.Duration(0) {
		cfg.HandshakeTimeout = time.Second * 10
	}

	if cfg.ReconnWaitTimeout == time.Duration(0) {
		cfg.ReconnWaitTimeout = time.Second * 3600
	}

	if cfg.ReconnInterval == time.Duration(0) {
		cfg.ReconnInterval = time.Second * 5
	}

	if cfg.HeartbeatInterval == time.Duration(0) {
		cfg.HeartbeatInterval = time.Second * 5
	}

	if cfg.HeartbeatTimeout == time.Duration(0) {
		cfg.HeartbeatTimeout = cfg.HeartbeatInterval * 5
	}
}

type Conn struct {
	base net.Conn

	enableCrypt bool

	closed    int32
	closeChan chan struct{}
	closeOnce sync.Once

	closeHeartbeatChan chan struct{}

	writeMutex sync.Mutex
	readMutex  sync.Mutex

	aead *crypto.Aead

	reconnMutex       sync.RWMutex
	reconnOpMutex     sync.Mutex
	readWaiting       bool
	writeWaiting      bool
	readWaitChan      chan struct{}
	writeWaitChan     chan struct{}
	reconnWaitTimeout time.Duration

	readCount  uint64
	writeCount uint64

	lastRecvTime time.Time

	config Config
}

func Dial(config Config, onConnect func(), onDisConnect func()) (*Conn, error) {
	config.onConnect = onConnect
	config.onDisConnect = onDisConnect

	config.init()

	conn, err := dial(&config)
	if err != nil {
		return nil, err
	}

	sconn, err := newConn(conn, config)
	if err != nil {
		return nil, err
	}

	if config.EnableCrypto {
		if err = sconn.init_crypto(); err != nil {
			conn.Close()
			return nil, err
		}
	}

	if err = sconn.handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	go sconn.heartbeat()

	return sconn, nil
}

func dial(config *Config) (net.Conn, error) {
	var conn net.Conn
	var err error
	for _, addr := range config.Addrs {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			return conn, nil
		}
	}

	// if conn != nil {
	// 	conn.(*net.TCPConn).SetKeepAlive(true)
	// 	conn.(*net.TCPConn).SetKeepAlivePeriod(config.KeepAlivePeriod)
	// }

	return conn, err
}

func newConn(base net.Conn, config Config) (conn *Conn, err error) {
	conn = &Conn{
		base:               base,
		enableCrypt:        config.EnableCrypto,
		reconnWaitTimeout:  config.ReconnWaitTimeout,
		closeChan:          make(chan struct{}),
		closeHeartbeatChan: make(chan struct{}),
		readWaitChan:       make(chan struct{}),
		writeWaitChan:      make(chan struct{}),
		config:             config,
	}

	return conn, nil
}

func (c *Conn) init_crypto() error {
	aead, err := crypto.NewAead(c.config.CryptoMethod, c.config.SecretKey)
	if err != nil {
		return err
	}

	c.aead = aead

	return nil
}

func (c *Conn) handshake() error {
	// 构建握手消息
	message := nson.Message{
		CHAN: nson.String(HAND),
	}

	if c.config.EnableCrypto {
		message.Insert(METHOD, nson.String(c.config.CryptoMethod.ToString()))
		message.Insert(ACCESS, nson.String(c.config.AccessKey))
	}

	// 编码并发送
	buf := new(bytes.Buffer)
	err := message.Encode(buf)
	if err != nil {
		return err
	}

	_, err = c.base.Write(buf.Bytes())
	if err != nil {
		return err
	}

	// 接收并验证
	var lbuf [4]byte
	if _, err := io.ReadFull(c.base, lbuf[:]); err != nil {
		return err
	}

	len := int(util.GetI32(lbuf[:], 0))
	if len < 5 || len > MAX_MESSAGE_LEN {
		return fmt.Errorf("消息长度不合适: %v", len)
	}

	rbuf := make([]byte, len)
	if _, err := io.ReadFull(c.base, rbuf[4:]); err != nil {
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
		return errors.New("错误的消息格式")
	}

	message2 := value.(nson.Message)

	code, err := message2.GetI32(CODE)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("error code %v", code)
	}

	// 握手成功

	// 认证
	err = c._write(c.base, c.config.AuthMessage)
	if err != nil {
		return err
	}

	rmsg, err := c._read(c.base)
	if err != nil {
		return err
	}

	code, err = rmsg.GetI32(CODE)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("error code %v", code)
	}

	// 认证成功

	c.lastRecvTime = time.Now()

	// 运行到这里说明已经连接成功
	if c.config.onConnect != nil {
		go c.config.onConnect()
	}

	return nil
}

func (c *Conn) heartbeat() {
	for {
		select {
		case <-c.closeHeartbeatChan:
			c.trace("heartbeat exit")
			return
		case <-time.After(c.config.HeartbeatInterval):
			d := time.Now().Sub(c.LastRecvTime())
			c.trace("heartbeat: %v", d)

			if d > c.config.HeartbeatInterval {
				message := nson.Message{
					CHAN: nson.String(PING),
				}

				c.SendMessage(message)
			}

			if d > c.config.HeartbeatTimeout {
				c.TryReconn()
			}
		}
	}
}

func (c *Conn) WrapBaseForTest(wrap func(net.Conn) net.Conn) {
	c.base = wrap(c.base)
}

func (c *Conn) RemoteAddr() net.Addr {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.base.RemoteAddr()
}

func (c *Conn) LocalAddr() net.Addr {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.base.LocalAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.base.SetDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.base.SetReadDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.base.SetWriteDeadline(t)
}

func (c *Conn) SetReconnWaitTimeout(d time.Duration) {
	c.reconnWaitTimeout = d
}

func (c *Conn) LastRecvTime() time.Time {
	c.reconnMutex.RLock()
	defer c.reconnMutex.RUnlock()
	return c.lastRecvTime
}

func (c *Conn) Close() error {
	c.trace("Close()")
	c.closeOnce.Do(func() {
		// c.closed = true
		atomic.StoreInt32(&c.closed, 1)
		close(c.closeChan)
		close(c.closeHeartbeatChan)
	})
	return c.base.Close()
}

func (c *Conn) TryReconn() {
	c.reconnMutex.RLock()
	base := c.base
	c.reconnMutex.RUnlock()
	go c.tryReconn(base)
}

func (c *Conn) _read(base net.Conn) (nson.Message, error) {
	var lbuf [4]byte
	_, err := io.ReadFull(base, lbuf[:])
	if err != nil {
		return nil, err
	}

	len := int(util.GetI32(lbuf[:], 0))
	if len < 5 || len > MAX_MESSAGE_LEN {
		return nil, fmt.Errorf("消息长度不合适: %v", len)
	}

	rbuf := make([]byte, len)
	if _, err := io.ReadFull(base, rbuf[4:]); err != nil {
		return nil, err
	}

	rbuf[0] = lbuf[0]
	rbuf[1] = lbuf[1]
	rbuf[2] = lbuf[2]
	rbuf[3] = lbuf[3]

	if c.config.EnableCrypto {
		rbuf, err = c.aead.Decrypt(rbuf)
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
		return nil, errors.New("错误的消息格式")
	}

	return value.(nson.Message), nil
}

func (c *Conn) RecvMessage() (msg nson.Message, err error) {
	c.trace("ReadReadMessage() wait write")
	c.readMutex.Lock()
	c.trace("ReadReadMessage() wait reconn")
	c.reconnMutex.RLock()
	c.readWaiting = true

	defer func() {
		c.readWaiting = false
		c.reconnMutex.RUnlock()
		c.readMutex.Unlock()
	}()

	for {
		if atomic.LoadInt32(&c.closed) != 0 {
			return nil, errors.New("conn closed")
		}

		base := c.base

		msg, err = c._read(base)

		if err == nil {
			c.trace("read from conn, msg: %+v", msg)
			c.lastRecvTime = time.Now()
			break
		}

		base.Close()

		go c.tryReconn(base)

		if !c.waitReconn('r', c.readWaitChan) {
			break
		}
	}

	if err == nil {
		c.readCount += 1
	}

	return
}

func (c *Conn) _write(base net.Conn, msg nson.Message) error {
	buf := new(bytes.Buffer)
	err := msg.Encode(buf)
	if err != nil {
		return err
	}

	if c.config.EnableCrypto {
		data, err := c.aead.Encrypt(buf.Bytes())
		if err != nil {
			return err
		}

		_, err = base.Write(data)
		if err != nil {
			return err
		}
	} else {
		_, err = base.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Conn) SendMessage(msg nson.Message) (err error) {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	c.trace("Write() %+v", msg)

	c.trace("Write() wait write")
	c.writeMutex.Lock()
	c.trace("Write() wait reconn")
	c.reconnMutex.RLock()
	c.writeWaiting = true

	defer func() {
		c.writeWaiting = false
		c.reconnMutex.RUnlock()
		c.writeMutex.Unlock()
	}()

	for {
		if atomic.LoadInt32(&c.closed) != 0 {
			return errors.New("conn closed")
		}

		base := c.base

		err = c._write(base, msg)
		if err == nil {
			break
		}

		base.Close()

		go c.tryReconn(base)

		if !c.waitReconn('w', c.writeWaitChan) {
			break
		}
	}

	if err == nil {
		c.writeCount += 1
	}

	return
}

func (c *Conn) waitReconn(who byte, waitChan chan struct{}) (done bool) {
	c.trace("waitReconn('%c', \"%s\")", who, c.reconnWaitTimeout)

	timeout := time.NewTimer(c.reconnWaitTimeout)
	defer timeout.Stop()

	c.reconnMutex.RUnlock()
	defer func() {
		c.reconnMutex.RLock()
		if done {
			<-waitChan
			c.trace("waitReconn('%c', \"%s\") done", who, c.reconnWaitTimeout)
		}
	}()

	select {
	case <-waitChan:
		done = true
		c.trace("waitReconn('%c', \"%s\") wake up", who, c.reconnWaitTimeout)
		return
	case <-c.closeChan:
		c.trace("waitReconn('%c', \"%s\") closed", who, c.reconnWaitTimeout)
		return
	case <-timeout.C:
		c.trace("waitReconn('%c', \"%s\") timeout", who, c.reconnWaitTimeout)
		c.Close()
		return
	}
}

func (c *Conn) tryReconn(badConn net.Conn) {
	// 运行到这里说明已经断开连接
	if c.config.onDisConnect != nil {
		// 调用回调函数
		go c.config.onDisConnect()
	}

	var done bool

	c.trace("tryReconn() wait tryReconn()")
	c.reconnOpMutex.Lock()
	defer c.reconnOpMutex.Unlock()

	c.trace("tryReconn() wait Read() or Write()")
	badConn.Close()
	c.reconnMutex.Lock()
	readWaiting := c.readWaiting
	writeWaiting := c.writeWaiting
	defer func() {
		c.reconnMutex.Unlock()
		if done {
			c.wakeUp(readWaiting, writeWaiting)
		}
	}()
	c.trace("tryReconn() begin")

	if badConn != c.base {
		c.trace("badConn != c.base")
		return
	}

	// 尝试重连
	for i := 0; atomic.LoadInt32(&c.closed) == 0; i++ {
		if i > 0 {
			time.Sleep(c.config.ReconnInterval)
		}

		c.trace("reconn dial")
		conn, err := dial(&c.config)
		if err != nil {
			c.trace("dial failed: %v", err)
			continue
		}

		if c.doReconn(conn) {
			c.trace("reconn success")
			done = true
			break
		}
		conn.Close()
	}
}

func (c *Conn) doReconn(conn net.Conn) bool {
	c.trace(
		"doReconn(\"%s\"), c.writeCount = %d, c.readCount = %d",
		conn.RemoteAddr(), c.writeCount, c.readCount,
	)

	c.base = conn

	if err := c.handshake(); err != nil {
		return false
	}

	return true
}

func (c *Conn) wakeUp(readWaiting, writeWaiting bool) {
	if readWaiting {
		c.trace("continue read")
		// make sure reader take over reconnMutex
		for i := 0; i < 2; i++ {
			select {
			case c.readWaitChan <- struct{}{}:
			case <-c.closeChan:
				c.trace("continue read closed")
				return
			}
		}
		c.trace("continue read done")
	}

	if writeWaiting {
		c.trace("continue write")
		// make sure writer take over reconnMutex
		for i := 0; i < 2; i++ {
			select {
			case c.writeWaitChan <- struct{}{}:
			case <-c.closeChan:
				c.trace("continue write closed")
				return
			}
		}
		c.trace("continue write done")
	}
}

func (c *Conn) trace(format string, args ...interface{}) {
	if c.config.Debug {
		format = fmt.Sprintf("Client conn %s", format)
		log.Printf(format, args...)
	}
}
