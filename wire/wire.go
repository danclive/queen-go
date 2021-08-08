package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/circ"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/dict"
	"github.com/danclive/queen-go/util"
)

var (
	// defaultKeepalive is the default connection keepalive value in seconds.
	defaultKeepalive uint16 = 10

	MAX_MESSAGE_LEN = 64 * 1024 * 1024 // 64MB

	ErrConnectionClosed = errors.New("Connection not open")
)

type Wire struct {
	sync.RWMutex
	conn      net.Conn
	r         *circ.Reader   // a reader for reading incoming bytes.
	w         *circ.Writer   // a writer for writing outgoing bytes.
	ID        nson.MessageId // the wire id.
	keepalive uint16         // the number of seconds the connection can wait.
	State     State          // the operational state of the wire.
	crypto    *crypto.Crypto // encrypt/decrypt
	close     chan struct{}  // used by heartbeat
}

// State tracks the state of the Wire.
type State struct {
	Done    int32           // atomic counter which indicates that the wire has closed.
	started *sync.WaitGroup // tracks the goroutines which have been started.
	endedW  *sync.WaitGroup // tracks when the writer has ended.
	endedR  *sync.WaitGroup // tracks when the reader has ended.
	endOnce sync.Once       // only end once.
}

// NewWire returns a new instance of Wire.
func NewWire(c net.Conn, r *circ.Reader, w *circ.Writer, id nson.MessageId, crypto *crypto.Crypto, keepalive uint16) *Wire {
	if keepalive == 0 {
		keepalive = defaultKeepalive
	}

	wire := &Wire{
		conn:      c,
		r:         r,
		w:         w,
		ID:        id,
		keepalive: keepalive,
		State: State{
			started: new(sync.WaitGroup),
			endedW:  new(sync.WaitGroup),
			endedR:  new(sync.WaitGroup),
		},

		crypto: crypto,
		close:  make(chan struct{}),
	}

	wire.refreshReadDeadline(wire.keepalive)
	wire.refreshWriteDeadline(wire.keepalive)

	return wire
}

// refreshDeadline refreshes the read/write deadline for the net.Conn connection.
func (c *Wire) refreshReadDeadline(keepalive uint16) {
	if c.conn != nil {
		var expiry time.Time // Nil time can be used to disable deadline if keepalive = 0
		if keepalive > 0 {
			expiry = time.Now().Add(time.Duration(keepalive+(keepalive/2)) * time.Second)
		}
		c.conn.SetReadDeadline(expiry)
	}
}

func (c *Wire) refreshWriteDeadline(keepalive uint16) {
	if c.conn != nil {
		var expiry time.Time // Nil time can be used to disable deadline if keepalive = 0
		if keepalive > 0 {
			expiry = time.Now().Add(time.Duration(keepalive+(keepalive/2)) * time.Second)
		}
		c.conn.SetWriteDeadline(expiry)
	}
}

func (c *Wire) heartbeat() error {
	ticker := time.NewTicker(time.Duration(c.keepalive) * time.Second)

	for {
		select {
		case <-c.close:
			return nil
		case <-ticker.C:
			message := nson.Message{
				dict.CHAN: nson.String(dict.KEEP_ALIVE),
			}
			// fmt.Println("keep_alive")
			_, err := c.Write(message)
			if err != nil {
				return err
			}
		}
	}
}

// Start begins the wire goroutines reading and writing packets.
func (c *Wire) start() {
	c.State.started.Add(2)

	go func() {
		c.State.started.Done()
		c.w.WriteTo(c.conn)
		c.State.endedW.Done()
		c.Stop()
	}()
	c.State.endedW.Add(1)

	go func() {
		c.State.started.Done()
		c.r.ReadFrom(c.conn)
		c.State.endedR.Done()
		c.Stop()
	}()
	c.State.endedR.Add(1)

	c.State.started.Wait()

	// start heartbeat
	go func() {
		c.heartbeat()
		c.Stop()
	}()
}

// Stop instructs the wire to shut down all processing goroutines and disconnect.
func (c *Wire) Stop() {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return
	}

	c.State.endOnce.Do(func() {
		// stop heartbead
		close(c.close)

		c.r.Stop()
		c.w.Stop()
		c.State.endedW.Wait()

		c.conn.Close()

		c.State.endedR.Wait()
		atomic.StoreInt32(&c.State.Done, 1)
	})
}

func (c *Wire) Handshake(attr nson.Message) error {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return ErrConnectionClosed
	}

	// handshake
	message := nson.Message{
		dict.CHAN:    nson.String(dict.HAND),
		dict.SLOT_ID: c.ID,
	}

	if c.crypto != nil {
		message.Insert(dict.METHOD, nson.String(c.crypto.Method().ToString()))
	}

	message.Extend(attr)

	wbuff := new(bytes.Buffer)
	err := message.Encode(wbuff)
	if err != nil {
		return err
	}

	c.conn.SetReadDeadline(time.Now().Add(time.Duration(10) * time.Second))

	_, err = c.conn.Write(wbuff.Bytes())
	if err != nil {
		return err
	}

	p := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, p[:]); err != nil {
		return err
	}

	l := int(util.GetU32(p[:], 0))
	if l < 5 || l > MAX_MESSAGE_LEN {
		return errors.New("invalid data")
	}

	b := make([]byte, l)
	if _, err := io.ReadFull(c.conn, b[4:]); err != nil {
		return err
	}

	for i := 0; i < len(p); i++ {
		b[i] = p[i]
	}

	rbuff := bytes.NewBuffer(b)
	value, err := nson.Message{}.Decode(rbuff)
	if err != nil {
		return err
	}

	if value.Tag() != nson.TAG_MESSAGE {
		return errors.New("invalid data")
	}

	replyMessage := value.(nson.Message)
	code, err := replyMessage.GetI32(dict.CODE)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("error code %v", code)
	}

	c.start()

	return nil
}

// Read reads new packets from a wire connection
func (c *Wire) Read(h func(*Wire, string, nson.Message) error) error {
	for {
		if atomic.LoadInt32(&c.State.Done) == 1 && c.r.CapDelta() == 0 {
			return nil
		}

		c.refreshReadDeadline(c.keepalive)

		var err error

		p, err := c.r.Read(4)
		if err != nil {
			return err
		}

		l := int(util.GetU32(p[:], 0))
		if l < 5 || l > MAX_MESSAGE_LEN {
			return errors.New("invalid data")
		}

		b, err := c.r.Read(l)
		if err != nil {
			return err
		}

		var buff *bytes.Buffer

		if c.crypto != nil {
			plaindata, err := c.crypto.Decrypt(b)
			if err != nil {

				return err
			}

			buff = bytes.NewBuffer(plaindata)
		} else {
			p = append(p, b...)

			buff = bytes.NewBuffer(p)
		}

		value, err := nson.Message{}.Decode(buff)
		if err != nil {
			return err
		}

		if value.Tag() != nson.TAG_MESSAGE {
			return errors.New("invalid data")
		}

		c.r.CommitTail(l)

		message := value.(nson.Message)

		// fmt.Println(message)

		if ch, err := message.GetString(dict.CHAN); err == nil {
			if ch == dict.KEEP_ALIVE {
				if !message.Contains(dict.CODE) {
					message.Insert(dict.CODE, nson.I32(0))

					_, err = c.Write(message)
					if err != nil {
						return err
					}

					continue
				}
			} else {
				err = h(c, ch, message) // Process inbound packet.
				if err != nil {
					return err
				}
			}
		}
	}
}

func (c *Wire) Write(message nson.Message) (n int, err error) {
	if atomic.LoadInt32(&c.State.Done) == 1 {
		return 0, ErrConnectionClosed
	}

	c.w.Mu.Lock()
	defer c.w.Mu.Unlock()

	buf := new(bytes.Buffer)

	err = message.Encode(buf)
	if err != nil {
		return
	}

	bytes := buf.Bytes()

	if c.crypto != nil {
		bytes, err = c.crypto.Encrypt(bytes)
		if err != nil {
			return
		}
	}

	n, err = c.w.Write(bytes)
	if err != nil {
		return
	}

	c.refreshWriteDeadline(c.keepalive)

	return
}
