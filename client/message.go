package client

import (
	"github.com/danclive/nson-go"
	"github.com/danclive/queen-go/dict"
)

type SendMessage struct {
	ch   string
	body nson.Message
	id   *nson.MessageId
	tag  []string
	to   []nson.MessageId
	call bool
}

func NewSendMessage(ch string) *SendMessage {
	return &SendMessage{
		ch:   ch,
		body: nson.Message{},
	}
}

func (s *SendMessage) SetId(id nson.MessageId) {
	s.id = &id
}

func (s *SendMessage) Id() *nson.MessageId {
	return s.id
}

func (s *SendMessage) AddTag(tag string) {
	if s.tag == nil {
		s.tag = make([]string, 0)
	}

	s.tag = append(s.tag, tag)
}

func (s *SendMessage) Lable() []string {
	return s.tag
}

func (s *SendMessage) AddTo(to nson.MessageId) {
	if s.to == nil {
		s.to = make([]nson.MessageId, 0)
	}

	s.to = append(s.to, to)
}

func (s *SendMessage) To() []nson.MessageId {
	return s.to
}

func (s *SendMessage) SetCall(call bool) {
	s.call = call
}

func (s *SendMessage) Call() bool {
	return s.call
}

func (s *SendMessage) Body() *nson.Message {
	return &s.body
}

func (s *SendMessage) build() nson.Message {
	msg := s.body

	msg.Insert(dict.CHAN, nson.String(s.ch))

	if s.tag != nil && len(s.tag) > 0 {
		if len(s.tag) == 1 {
			msg.Insert(dict.TAG, nson.String(s.tag[0]))
		} else {
			array := make(nson.Array, 0)
			for _, v := range s.tag {
				array = append(array, nson.String(v))
			}
			msg.Insert(dict.TAG, nson.Array(array))
		}
	}

	if s.to != nil && len(s.to) > 0 {
		if len(s.to) == 1 {
			msg.Insert(dict.TO, s.to[0])
		} else {
			array := make(nson.Array, 0)
			for _, v := range s.to {
				array = append(array, v)
			}
			msg.Insert(dict.TO, nson.Array(array))
		}
	}

	if s.call {
		msg.Insert(dict.SHARE, nson.Bool(true))
	}

	return msg
}

type RecvMessage struct {
	Ch   string
	Body nson.Message
}

func (r *RecvMessage) GetCallId() (nson.MessageId, bool) {
	if id, err := r.Body.GetMessageId(CALL_ID); err == nil {
		return id, true
	}

	return nil, false
}

func (r *RecvMessage) GetFromId() (nson.MessageId, bool) {
	if id, err := r.Body.GetMessageId(dict.FROM); err == nil {
		return id, true
	}

	return nil, false
}

func (r *RecvMessage) Back() *SendMessage {
	callId, ok := r.GetCallId()
	if !ok {
		return nil
	}

	fromId, ok := r.GetFromId()
	if !ok {
		return nil
	}

	return &SendMessage{
		ch: r.Ch,
		body: nson.Message{
			CALL_ID:   callId,
			dict.CODE: nson.I32(0),
		},
		to: []nson.MessageId{fromId},
	}
}
