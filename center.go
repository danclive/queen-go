package queen

import (
	"math"
	"sync"

	"github.com/danclive/nson-go"
)

type Center struct {
	maps    map[string]nson.Value
	handles map[string][]CHandler
	all     []CHandler
	next_id int32
	lock    sync.RWMutex
}

type CHandler struct {
	id     int32
	handle func(CContext)
}

type CContext struct {
	Center   *Center
	Id       int32
	Key      string
	OldValue nson.Value
	Value    nson.Value
}

func NewCenter() Center {
	return Center{
		maps:    make(map[string]nson.Value),
		handles: make(map[string][]CHandler),
		all:     make([]CHandler, 0),
		next_id: 0,
	}
}

func (self *Center) InitCenter() {
	self.maps = make(map[string]nson.Value)
	self.handles = make(map[string][]CHandler)
	self.all = make([]CHandler, 0)
	self.next_id = 0
}

func (self *Center) On(key string, fn func(CContext)) (id int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.next_id == math.MaxInt32 {
		self.next_id = 0
	}

	self.next_id += 1
	id = self.next_id

	handler := CHandler{
		id:     id,
		handle: fn,
	}

	if handlers, ok := self.handles[key]; ok {
		handlers = append(handlers, handler)
		self.handles[key] = handlers
	} else {
		self.handles[key] = []CHandler{handler}
	}

	return
}

func (self *Center) All(fn func(CContext)) (id int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.next_id == math.MaxInt32 {
		self.next_id = 0
	}

	self.next_id += 1
	id = self.next_id

	handler := CHandler{
		id:     id,
		handle: fn,
	}

	self.all = append(self.all, handler)

	return
}

func (self *Center) Off(id int32) (ok bool) {
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
				handles := append(handles[:pos], handles[pos+1:]...)
				self.handles[event] = handles
			} else {
				delete(self.handles, event)
			}

			return
		}
	}

	for i, handler := range self.all {
		if handler.id == id {
			self.all = append(self.all[:i], self.all[i+1:]...)
			ok = true

			return
		}
	}

	return
}

func (self *Center) Insert(key string, value nson.Value) {
	self.lock.Lock()

	v, has := self.maps[key]
	self.maps[key] = value
	handlers, ok := self.handles[key]
	all := self.all

	self.lock.Unlock()

	if (!has || v != value) && len(all) > 0 {
		go func(center *Center, handlers []CHandler) {
			for _, handler := range handlers {
				context := CContext{
					Center:   center,
					Id:       handler.id,
					Key:      key,
					OldValue: v,
					Value:    value,
				}
				handler.handle(context)
			}
		}(self, all)
	}

	if (!has || v != value) && ok && len(handlers) > 0 {
		go func(center *Center, handlers []CHandler) {
			for _, handler := range handlers {
				context := CContext{
					Center:   center,
					Id:       handler.id,
					Key:      key,
					OldValue: v,
					Value:    value,
				}
				handler.handle(context)
			}
		}(self, handlers)
	}
}

func (self *Center) Get(key string) (nson.Value, bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	value, has := self.maps[key]
	return value, has
}

func (self *Center) Remove(key string) (nson.Value, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	value, has := self.maps[key]

	if has {
		delete(self.maps, key)
	}

	return value, has
}
