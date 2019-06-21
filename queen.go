package queen

import (
	"container/heap"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/danclive/nson-go"
)

type Queen struct {
	handles map[string][]Handler // HashMap<String, []Handler>
	next_id int32
	lock    sync.RWMutex
	timer   *Timer
}

type Handler struct {
	id     int32
	handle func(Context)
}

type Context struct {
	Queen   *Queen
	Id      int32
	Event   string
	Message nson.Message
}

func NewQueen() *Queen {
	queen := &Queen{
		handles: make(map[string][]Handler),
		next_id: 0,
		timer:   NewTimer(),
	}

	queen.timer.Run(queen)

	return queen
}

func (self *Queen) InitQueen() {
	self.handles = make(map[string][]Handler)
	self.next_id = 0
}

func (self *Queen) On(event string, fn func(Context)) (id int32) {
	self.lock.Lock()

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

	self.lock.Unlock()

	if strings.HasPrefix(event, "pub:") || strings.HasPrefix(event, "sys:") {
		go self.Emit("queen", nson.Message{
			"event": nson.String("on"),
			"value": nson.String("event"),
		})
	}

	return
}

func (self *Queen) Off(id int32) (ok bool) {
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

		if strings.HasPrefix(event, "pub:") || strings.HasPrefix(event, "sys:") {
			go self.Emit("queen", nson.Message{
				"event": nson.String("off"),
				"value": nson.String("event"),
			})
		}
	}

	return
}

func (self *Queen) Emit(event string, message nson.Message) {
	if message.Contains("_delay") {
		delay, err := message.GetI32("_delay")
		if err != nil {

		}

		message.Remove("_delay")

		t := time.Now().Add(time.Millisecond * time.Duration(delay))

		self.timer.Push(Task{
			event: event,
			msg:   message,
			time:  t,
		})

	} else if strings.HasPrefix(event, "pub:") || strings.HasPrefix(event, "sys:") {
		self.Emit("queen", nson.Message{
			"event": nson.String("emit"),
			"value": nson.String("event"),
			"msg":   message,
		})

		self.Push(event, message)
	} else {
		self.Push(event, message)
	}
}

func (self *Queen) Push(event string, message nson.Message) {
	self.lock.RLock()
	handlers, ok := self.handles[event]
	self.lock.RUnlock()

	if ok {
		go func(queen *Queen, handlers []Handler) {
			for _, handler := range handlers {
				context := Context{
					Queen:   queen,
					Id:      handler.id,
					Event:   event,
					Message: message,
				}
				handler.handle(context)
			}
		}(self, handlers)
	}
}

type Timer struct {
	lock  sync.Mutex
	tasks *Tasks
	run   bool
}

type Task struct {
	event string
	msg   nson.Message
	time  time.Time
}

type Tasks []Task

func (h Tasks) Len() int           { return len(h) }
func (h Tasks) Less(i, j int) bool { return h[i].time.Sub(h[j].time) < 0 }
func (h Tasks) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *Tasks) Push(x interface{}) {
	*h = append(*h, x.(Task))
}

func (h *Tasks) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func NewTimer() *Timer {
	h := &Tasks{}
	heap.Init(h)

	return &Timer{tasks: h, run: true}
}

func (t *Timer) Push(task Task) {
	heap.Push((*t).tasks, task)
}

func (t *Timer) Pop() interface{} {
	return heap.Pop((*t).tasks)
}

func (t *Timer) Run(queen *Queen) {
	go func(timer *Timer, queen *Queen) {
		for timer.run {
			sleep_duration := time.Second * 1

			for {
				timer.lock.Lock()

				if timer.tasks.Len() <= 0 {
					timer.lock.Unlock()
					break
				}

				i := timer.Pop()
				if i == nil {
					timer.lock.Unlock()
					break
				} else {
					task := i.(Task)

					diff := task.time.Sub(time.Now())
					if diff > 0 {
						timer.Push(task)
						sleep_duration = diff
						timer.lock.Unlock()
						break
					} else {
						queen.Emit(task.event, task.msg)
						timer.lock.Unlock()
					}
				}
			}

			time.Sleep(sleep_duration)
		}
	}(t, queen)
}
