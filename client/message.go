package client

import (
	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/conn"
)

type SendMessage struct {
	Ch    string
	Body  nson.Message
	Id    *nson.MessageId
	Label []string
	To    []nson.MessageId
	Call  bool
}

func NewSendMessage(ch string, body nson.Message) *SendMessage {
	return &SendMessage{
		Ch:   ch,
		Body: body,
	}
}

func (s *SendMessage) WithId(id nson.MessageId) *SendMessage {
	s.Id = &id
	return s
}

func (s *SendMessage) WithLabel(label []string) *SendMessage {
	s.Label = label
	return s
}

func (s *SendMessage) WithTo(to []nson.MessageId) *SendMessage {
	s.To = to
	return s
}

func (s *SendMessage) WithCall(call bool) *SendMessage {
	s.Call = call
	return s
}

func (s *SendMessage) Build() nson.Message {
	msg := s.Body

	msg.Insert(conn.CHAN, nson.String(s.Ch))

	if s.Label != nil && len(s.Label) > 0 {
		array := make(nson.Array, 0)
		for _, v := range s.Label {
			array = append(array, nson.String(v))
		}
		msg.Insert(conn.LABEL, nson.Array(array))
	}

	if s.To != nil && len(s.To) > 0 {
		array := make(nson.Array, 0)
		for _, v := range s.To {
			array = append(array, v)
		}
		msg.Insert(conn.TO, nson.Array(array))
	}

	if s.Call {
		msg.Insert(conn.SHARE, nson.Bool(true))
	}

	return msg
}

type RecvMessage struct {
	Ch   string
	Body nson.Message
}

func (r *RecvMessage) CallId() (nson.MessageId, bool) {
	if id, err := r.Body.GetMessageId(CALL_ID); err == nil {
		return id, true
	}

	return nil, false
}

func (r *RecvMessage) FromId() (nson.MessageId, bool) {
	if id, err := r.Body.GetMessageId(conn.FROM); err == nil {
		return id, true
	}

	return nil, false
}

func (r *RecvMessage) Back() (*SendMessage, bool) {
	callId, ok := r.CallId()
	if !ok {
		return nil, false
	}

	fromId, ok := r.FromId()
	if !ok {
		return nil, false
	}

	return &SendMessage{
		Ch: r.Ch,
		Body: nson.Message{
			CALL_ID:   callId,
			conn.CODE: nson.I32(0),
		},
		To: []nson.MessageId{fromId},
	}, true
}
