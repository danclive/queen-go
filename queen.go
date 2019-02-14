package queen

import (
	"math"
	"sync"
)

type EventEmiter struct {
	handles map[string][]Handler // HashMap<String, []Handler>
	next_id int32
	lock    sync.RWMutex
}

type Handler struct {
	id     int32
	handle func(Context)
}

type Context struct {
	Event_emiter *EventEmiter
	Id           int32
	Event        string
	Message      interface{}
}

func NewEventEmiter() EventEmiter {
	return EventEmiter{
		handles: make(map[string][]Handler),
		next_id: 0,
	}
}

func (self *EventEmiter) InitEventEmiter() {
	self.handles = make(map[string][]Handler)
	self.next_id = 0
}

func (self *EventEmiter) On(event string, fn func(Context)) (id int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.next_id == math.MaxInt32 {
		self.next_id = 0
	}

	self.next_id += 1
	id = self.next_id

	handler := Handler{
		id:     id,
		handle: fn,
	}

	if handlers, ok := self.handles[event]; ok {
		handlers = append(handlers, handler)
		self.handles[event] = handlers
	} else {
		self.handles[event] = []Handler{handler}
	}

	return
}

func (self *EventEmiter) Off(id int32) (ok bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for event, handles := range self.handles {

		has := false
		pos := 0

		for i, handler := range handles {
			if handler.id == id {
				has = true
				pos = i
			}
		}

		if has {
			ok = true

			l := len(handles)

			if l > 1 {
				handles = append(handles[:pos], handles[pos+1:]...)
				self.handles[event] = handles
			} else {
				delete(self.handles, event)
			}
		}
	}

	return
}

func (self *EventEmiter) Emit(event string, message interface{}) {
	self.lock.RLock()
	handlers, ok := self.handles[event]
	self.lock.RUnlock()

	if ok {
		go func(event_emiter *EventEmiter, handlers []Handler) {
			for _, handler := range handlers {
				context := Context{
					Event_emiter: event_emiter,
					Id:           handler.id,
					Event:        event,
					Message:      message,
				}
				handler.handle(context)
			}
		}(self, handlers)
	}
}
