package main

import (
	"fmt"

	"github.com/danclive/nson-go"

	"github.com/danclive/queen-go"
)

func main() {
	center := queen.NewCenter()

	center.On("hello", func(context queen.CContext) {
		fmt.Println(context)
	})

	center.Insert("hello", nson.String("world"))
	center.Insert("hello", nson.String("world2"))

	select {}
}
