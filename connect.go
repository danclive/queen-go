package queen

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"time"

	nson "github.com/danclive/nson-go"
)

func InitConnect(queen *Queen) {
	queen.On(LINK, link)
	queen.On(UNLINK, unlink)
	queen.On(REMOVE, remove)
	queen.On(HAND, hand)
	queen.On(HANDED, handed)
	queen.On(ATTACH, attach)
	queen.On(DETACH, detach)
	queen.On(SEND, send)
	queen.On(RECV, recv)
}

const (
	LINK   = "sys:link"
	UNLINK = "sys:unlink"
	REMOVE = "sys:remove"
	HAND   = "sys:hand"
	HANDED = "sys:handed"
	ATTACH = "sys:attach"
	DETACH = "sys:detach"
	SEND   = "sys:send"
	RECV   = "sys:recv"
)

var control struct {
	lock      sync.Mutex
	islink    bool
	handshake bool
	conn      net.Conn
	addr      string
}

func link(context Context) {
	msg, ok := context.Message.(nson.Message)
	if !ok {
		return
	}

	if msg.Contains("ok") {
		return
	}

	protocol, err := msg.GetString("protocol")
	if err != nil {
		msg.Insert("ok", nson.Bool(false))
		msg.Insert("error", nson.String("Message format error: can't not get protocol!"))

		context.Queen.Emit(LINK, msg)
		return
	}

	if protocol == "tcp" {
		addr, err := msg.GetString("addr")
		if err != nil {
			msg.Insert("ok", nson.Bool(false))
			msg.Insert("error", nson.String("Message format error: can't not get addr!"))

			context.Queen.Emit(LINK, msg)
			return
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			msg.Insert("ok", nson.Bool(false))
			msg.Insert("error", nson.String(err.Error()))
		}

		control.lock.Lock()
		control.conn = conn
		control.islink = true
		control.addr = addr
		control.lock.Unlock()

		go func() {

			buffer := new(bytes.Buffer)

			for {
				control.lock.Lock()
				islink := control.islink
				control.lock.Unlock()

				if !islink {
					return
				}

				t := time.Now()

				err := conn.SetReadDeadline(t.Add(10 * time.Second))
				if err != nil {
					// control.lock.Lock()
					// control.islink = false
					// control.handshake = false
					// control.lock.Unlock()

					msg := nson.Message{
						"event":    nson.String(REMOVE),
						"protocol": nson.String("tcp"),
						"addr":     nson.String(control.addr),
						"error":    nson.String(err.Error()),
					}

					context.Queen.Emit(REMOVE, msg)
					return
				}

				buf := make([]byte, 0, 4*1024)
				n, err := control.conn.Read(buf)
				if err != nil {
					// control.lock.Lock()
					// control.islink = false
					// control.handshake = false
					// control.lock.Unlock()

					msg := nson.Message{
						"event":    nson.String(REMOVE),
						"protocol": nson.String("tcp"),
						"addr":     nson.String(control.addr),
						"error":    nson.String(err.Error()),
					}

					context.Queen.Emit(REMOVE, msg)
					return
				}

				buffer.Write(buf[:n])

				for l := buffer.Len(); l > 4; {

					i := int(GetI32(buffer.Bytes(), 0))

					if l >= i {
						message, err := nson.Message{}.Decode(buffer)
						if err != nil {

							// control.lock.Lock()
							// control.islink = false
							// control.handshake = false
							// control.lock.Unlock()

							msg := nson.Message{
								"event":    nson.String(REMOVE),
								"protocol": nson.String("tcp"),
								"addr":     nson.String(control.addr),
								"error":    nson.String("Message decode error!"),
							}

							context.Queen.Emit(REMOVE, msg)
							return
						}

						context.Queen.Emit(RECV, message)
					} else {
						break
					}
				}
			}
		}()

		msg.Insert("ok", nson.Bool(true))
		return
	}

	msg.Insert("ok", nson.Bool(false))
	msg.Insert("error", nson.String("unimplemented!"))

	context.Queen.Emit(LINK, msg)
}

func unlink(context Context) {
	msg, ok := context.Message.(nson.Message)
	if !ok {
		return
	}

	if msg.Contains("ok") {
		return
	}

	control.lock.Lock()

	if control.islink {
		control.islink = false
		control.handshake = false
		control.conn.Close()

		msg.Insert("ok", nson.Bool(true))
		msg.Insert("protocol", nson.String("tcp"))
		msg.Insert("addr", nson.String(control.addr))
	} else {
		msg.Insert("ok", nson.Bool(false))
		msg.Insert("error", nson.String("Already unlink!"))
	}

	control.lock.Unlock()
}

func remove(context Context) {
	control.lock.Lock()

	if control.islink {
		control.islink = false
		control.handshake = false
		control.conn.Close()
	}

	control.lock.Unlock()
}

func hand(context Context) {

}

func handed(context Context) {

}

func attach(context Context) {

}

func detach(context Context) {

}

func send(context Context) {
	control.lock.Lock()
	islink := control.islink
	control.lock.Unlock()

	if !islink {
		return
	}

	msg, ok := context.Message.(nson.Message)
	if !ok {
		return
	}

	if msg.Contains("ok") {
		return
	}

	event, err := msg.GetString("event")
	if err != nil {
		return
	}

	if !strings.HasPrefix(event, "pub:") {
		return
	}

	buf := new(bytes.Buffer)

	err = msg.Encode(buf)
	if err != nil {
		msg.Insert("ok", nson.Bool(true))
		msg.Insert("error", nson.String("Message encode error!"))
		return
	}

	_, err = control.conn.Write(buf.Bytes())
	if err != nil {
		msg := nson.Message{
			"event":    nson.String(REMOVE),
			"protocol": nson.String("tcp"),
			"addr":     nson.String(control.addr),
			"error":    nson.String(err.Error()),
		}

		context.Queen.Emit(REMOVE, msg)
		return
	}

	msg.Insert("ok", nson.Bool(true))
	context.Queen.Emit(SEND, msg)
}

func recv(context Context) {

}

func GetI32(b []byte, pos int) int32 {
	return (int32(b[pos+0])) |
		(int32(b[pos+1]) << 8) |
		(int32(b[pos+2]) << 16) |
		(int32(b[pos+3]) << 24)
}
