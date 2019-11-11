package queen

/*
import (
	"bytes"
	"log"

	"github.com/danclive/nson-go"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func MqttRun(topic string, mqttClient mqtt.Client, queen *EventBus) {
	mqttClient.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		buf := bytes.NewBuffer(msg.Payload())
		m, err := nson.Message{}.Decode(buf)
		if err != nil {
			log.Println(err)
		}

		message, ok := m.(nson.Message)
		if !ok {
			return
		}

		event, err := message.GetString("event")
		if err != nil {
			log.Println(err)
			return
		}

		if event == "emit" {
			value, err := message.GetString("value")
			if err != nil {
				log.Println(err)
				return
			}

			inMsg, err := message.GetMessage("msg")
			if err != nil {
				log.Println(err)
				return
			}

			queen.Push(value, inMsg)
		}
	})

	queen.On("queen", func(ctx Context) {
		buf := new(bytes.Buffer)

		err := ctx.Message.Encode(buf)
		if err != nil {
			log.Println(err)
			return
		}

		mqttClient.Publish(topic, 1, false, buf.Bytes())
	})
}
*/
