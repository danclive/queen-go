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

	queent := queen.NewQueen()

	queent.On("hello", func(context queen.Context) {
		fmt.Println(context.Message)
	})

	queent.On("pub:queen/center", func(context queen.Context) {
		fmt.Println(context.Message.String())
		fmt.Println("1111111111111")
	})

	queent.Emit("hello", nson.Message{"hello": nson.String("world")})
	queent.Emit("hello", nson.Message{"hello": nson.String("world"), "_delay": nson.I32(1000 * 2)})

	center.Run(queent, "pub:queen/center")

	center.Insert("hello", nson.String("22222222222222222"))

	center2 := queen.NewCenter()
	center2.Run(queent, "pub:queen/center")

	center2.Insert("hello", nson.String("ffffffffff"))

	select {}
}
