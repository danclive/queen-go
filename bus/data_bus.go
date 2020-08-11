package bus

import (
	"math"
	"sync"
	"time"

	"github.com/danclive/nson-go"
)

type DataBus struct {
	id      int32
	maps    map[string]nson.Value
	handles map[string][]DataBusHandler
	all     []DataBusHandler
	next_id int32
	lock    sync.RWMutex
}

type DataBusHandler struct {
	id     int32
	handle func(DataBusContext)
}

type DataBusContext struct {
	DataBus  *DataBus
	Id       int32
	Key      string
	OldValue nson.Value
	Value    nson.Value
}

func NewDataBus() *DataBus {
	return &DataBus{
		id:      int32(time.Now().Nanosecond()),
		maps:    make(map[string]nson.Value),
		handles: make(map[string][]DataBusHandler),
		all:     make([]DataBusHandler, 0),
		next_id: 0,
	}
}

func (self *DataBus) InitDataBus() {
	self.id = int32(time.Now().Nanosecond())
	self.maps = make(map[string]nson.Value)
	self.handles = make(map[string][]DataBusHandler)
	self.all = make([]DataBusHandler, 0)
	self.next_id = 0
}

func (self *DataBus) Run(eventBus *EventBus, event string) {
	self.All(func(ctx DataBusContext) {
		msg := nson.Message{
			"key":   nson.String(ctx.Key),
			"value": ctx.Value,
			"_id":   nson.I32(self.id),
		}

		if ctx.OldValue != nil {
			msg.Insert("old_value", ctx.OldValue)
		}

		eventBus.Emit(event, msg)
	})

	go eventBus.On(event, func(ctx EventBusContext) {
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

func (self *DataBus) On(key string, fn func(DataBusContext)) (id int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.next_id == math.MaxInt32 {
		self.next_id = 0
	}

	self.next_id += 1
	id = self.next_id

	handler := DataBusHandler{
		id:     id,
		handle: fn,
	}

	if handlers, ok := self.handles[key]; ok {
		handlers = append(handlers, handler)
		self.handles[key] = handlers
	} else {
		self.handles[key] = []DataBusHandler{handler}
	}

	return
}

func (self *DataBus) All(fn func(DataBusContext)) (id int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.next_id == math.MaxInt32 {
		self.next_id = 0
	}

	self.next_id += 1
	id = self.next_id

	handler := DataBusHandler{
		id:     id,
		handle: fn,
	}

	self.all = append(self.all, handler)

	return
}

func (self *DataBus) Off(id int32) (ok bool) {
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

func (self *DataBus) Insert(key string, value nson.Value) (nson.Value, bool) {
	return self.insert(key, value, false)
}

func (self *DataBus) InsertSkipAll(key string, value nson.Value) (nson.Value, bool) {
	return self.insert(key, value, true)
}

func (self *DataBus) insert(key string, value nson.Value, skip_all bool) (nson.Value, bool) {
	self.lock.Lock()

	v, has := self.maps[key]
	self.maps[key] = value
	handlers, ok := self.handles[key]
	all := self.all

	self.lock.Unlock()

	if !skip_all && len(all) > 0 {
		go func(dataBus *DataBus, handlers []DataBusHandler) {
			for _, handler := range handlers {
				context := DataBusContext{
					DataBus:  dataBus,
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
		go func(dataBus *DataBus, handlers []DataBusHandler) {
			for _, handler := range handlers {
				context := DataBusContext{
					DataBus:  dataBus,
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

func (self *DataBus) Set(key string, value nson.Value) (nson.Value, bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	v, has := self.maps[key]
	self.maps[key] = value
	return v, has
}

func (self *DataBus) Get(key string) (nson.Value, bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	value, has := self.maps[key]
	return value, has
}

func (self *DataBus) Remove(key string) (nson.Value, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	value, has := self.maps[key]

	if has {
		delete(self.maps, key)
	}

	return value, has
}
