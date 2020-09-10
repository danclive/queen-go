package main

import (
	"fmt"
	"log"
	"time"

	"github.com/danclive/nson-go"
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

	_ = c
	// for {
	//time.Sleep(1 * time.Second)
	time1 := time.Now()

	slotId, err := nson.MessageIdFromHex("017477033867f215f0c5341e")
	if err != nil {
		panic(err)
	}

	msg := client.NewSendMessage("dev.meta").Call(true).To(slotId)

	data := nson.Message{
		"method": nson.String("PullSlots"),
		"params": nson.Message{},
	}
	msg.Body().Insert("data", data)

	log.Println(c.Send(msg, 0))

	fmt.Println(time.Now().Sub(time1))
	//}

	var e = make(chan bool)
	<-e
}
