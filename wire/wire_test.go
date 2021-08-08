package wire

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/circ"
	"github.com/danclive/queen-go/crypto"
	"github.com/danclive/queen-go/dict"
	"github.com/stretchr/testify/require"
)

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

func TestWire(t *testing.T) {
	var err error

	addrs := []string{"127.0.0.1:8888"}

	// connect
	conn, err := dial(addrs)
	require.Nil(t, err)

	slotID, err := nson.MessageIdFromHex("017aaafc146a5bdb3af115d7")
	require.Nil(t, err)

	crypto, err := crypto.NewCrypto(crypto.Aes128Gcm, "sht3x1pass")
	require.Nil(t, err)

	wire := NewWire(conn, circ.NewReader(128, 8), circ.NewWriter(128, 8), slotID, crypto, 0)
	require.NotNil(t, wire)

	attr := nson.Message{}

	attr.Insert("_acce", nson.String("sht3x1user"))

	wire.Handshake(attr)

	ping := nson.Message{dict.CHAN: nson.String(dict.PING)}
	_, err = wire.Write(ping)
	require.Nil(t, err)

	err = wire.Read(func(w *Wire, _ string, m nson.Message) error {
		fmt.Println(m)
		w.Stop()
		return err
	})

	require.Nil(t, err)
}
