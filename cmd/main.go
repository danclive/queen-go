package main

import (
	queen "github.com/danclive/queen-go"
)

func main() {
	event_emiter := queen.NewEventEmiter()

	event_emiter.On("run", func(context queen.Context) {
		// fmt.Println(1)
		event_emiter.Emit("run", nil)
	})

	event_emiter.Emit("run", nil)

	select {}
}
