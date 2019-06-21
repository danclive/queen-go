package queen

import (
	"math"
	"sync"
	"time"

	"github.com/danclive/nson-go"
)

type Center struct {
	id      int32
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

func NewCenter() *Center {
	return &Center{
		id:      int32(time.Now().Nanosecond()),
		maps:    make(map[string]nson.Value),
		handles: make(map[string][]CHandler),
		all:     make([]CHandler, 0),
		next_id: 0,
	}
}

func (self *Center) InitCenter() {
	self.id = int32(time.Now().Nanosecond())
	self.maps = make(map[string]nson.Value)
	self.handles = make(map[string][]CHandler)
	self.all = make([]CHandler, 0)
	self.next_id = 0
}

func (self *Center) Run(queen *Queen, event string) {
	self.All(func(ctx CContext) {
		msg := nson.Message{
			"key":   nson.String(ctx.Key),
			"value": ctx.Value,
			"_id":   nson.I32(self.id),
		}

		if ctx.OldValue != nil {
			msg.Insert("old_value", ctx.OldValue)
		}

		queen.Emit(event, msg)
	})

	go queen.On(event, func(ctx Context) {
		key, err := ctx.Message.GetString("key")
		if err != nil {
			return
		}

		value, has := ctx.Message.Get("value")
		if !has {
			return
		}

		id, err := ctx.Message.GetI32("_id")
		if err != nil {
			return
		}

		if id != self.id {
			self.InsertSkipAll(key, value)
		}
	})
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

func (self *Center) Insert(key string, value nson.Value) (nson.Value, bool) {
	return self.insert(key, value, false)
}

func (self *Center) InsertSkipAll(key string, value nson.Value) (nson.Value, bool) {
	return self.insert(key, value, true)
}

func (self *Center) insert(key string, value nson.Value, skip_all bool) (nson.Value, bool) {
	self.lock.Lock()

	v, has := self.maps[key]
	self.maps[key] = value
	handlers, ok := self.handles[key]
	all := self.all

	self.lock.Unlock()

	if !skip_all && len(all) > 0 {
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

	return v, has
}

func (self *Center) Set(key string, value nson.Value) (nson.Value, bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	v, has := self.maps[key]
	self.maps[key] = value
	return v, has
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
