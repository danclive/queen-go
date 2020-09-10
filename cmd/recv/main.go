package main

import (
	"log"

	"github.com/danclive/queen-go/client"
	"github.com/danclive/queen-go/conn"
	"github.com/danclive/queen-go/crypto"
)

func main() {
	config := conn.Config{
		Addrs:        []string{"snple.com:8888"},
		EnableCrypto: true,
		CryptoMethod: crypto.Aes128Gcm,
		AccessKey:    "fcbd6ea1e8c94dfc6b84405e",
		SecretKey:    "b14cd7bf94f0e3374e7fc4d4",
		Debug:        false,
	}

	c, err := client.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	client.Close()
	// }()

	c.OnConnect(func() {
		err = c.Attach("dev.data", nil, 0)
		if err != nil {
			log.Fatalln(err)
		}
	})

	recvChan := c.Recv()

	for recv := range recvChan {
		log.Println(recv)

		if back := recv.Back(); back != nil {
			c.Send(back, 0)
		}
	}

	// var e = make(chan bool)
	// <-e
}
