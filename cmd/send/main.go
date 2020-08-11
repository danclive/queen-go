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
		Addrs:        []string{"danclive.com:8888"},
		EnableCrypto: true,
		CryptoMethod: crypto.Aes128Gcm,
		AccessKey:    "fcbd6ea1e8c94dfc6b84405e",
		SecretKey:    "b14cd7bf94f0e3374e7fc4d4",
		Debug:        false,
	}

	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	_ = client

	err = client.Send("lala", nson.Message{"aaa": nson.String("bbb")}, nil, nil, time.Second*10)
	log.Println(err)

	msg, err := client.Call("lala", nson.Message{"aaa": nson.String("bbb")}, nil, nil, time.Second*10)
	log.Println(err)
	log.Println(msg)

	for {
		now := time.Now()
		msg, err := client.Call("ping", nson.Message{"aaa": nson.String("bbb")}, nil, nil, time.Second*10)
		//log.Println(err)
		//log.Println(msg)
		_ = err
		_ = msg
		now2 := time.Now()

		fmt.Println(now2.Sub(now))
		time.Sleep(time.Second)
	}

	// var e = make(chan bool)
	// <-e
}
