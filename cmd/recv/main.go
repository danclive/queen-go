package main

import (
	"fmt"
	"log"

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

	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	_ = client

	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	client.Close()
	// }()

	client.Recv("lala", nil, func(ch string, message nson.Message) {
		fmt.Println(ch)
		fmt.Println(message)
	})

	client.Llac("lala", nil, func(ch string, message nson.Message) nson.Message {
		fmt.Println(ch)
		fmt.Println(message)

		return message
	})

	i := 0
	for {
		i++
		fmt.Println(i)
		err = client.Send("lala", nson.Message{"aaa": nson.String("bbb")}, nil, nil, 0)
		log.Println(err)
	}

	var e = make(chan bool)
	<-e
}
