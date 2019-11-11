package queen

import (
	"testing"

	"github.com/danclive/nson-go"

	"github.com/stretchr/testify/assert"
)

func TestEventBus(t *testing.T) {
	eventBus := NewEventBus()
	assert.NotNil(t, eventBus.handles)
}

func TestInitEventBus(t *testing.T) {
	eventBus := EventBus{}
	eventBus.InitEventBus()
	assert.NotNil(t, eventBus.handles)
}

func TestOn(t *testing.T) {
	eventBus := NewEventBus()

	eventBus.On("hello", func(context EventBusContext) {})

	handles, ok := eventBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(eventBus.handles))

	eventBus.On("hello", func(context EventBusContext) {})
	handles, ok = eventBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(eventBus.handles))

	_, ok = eventBus.handles["unkonw"]
	assert.False(t, ok)
}

func TestOff(t *testing.T) {
	eventBus := NewEventBus()

	id := eventBus.On("hello", func(context EventBusContext) {})

	handles, ok := eventBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(eventBus.handles))

	has := eventBus.Off(id)
	assert.True(t, has)

	handles, ok = eventBus.handles["hello"]
	assert.False(t, ok)
	assert.Exactly(t, 0, len(handles))
	assert.Exactly(t, 0, len(eventBus.handles))
}

func TestOff2(t *testing.T) {
	eventBus := NewEventBus()

	id1 := eventBus.On("hello", func(context EventBusContext) {})
	id2 := eventBus.On("hello", func(context EventBusContext) {})

	handles, ok := eventBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(eventBus.handles))

	has := eventBus.Off(id1)
	assert.True(t, has)
	_ = id2

	handles, ok = eventBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(eventBus.handles))
}

func TestOff3(t *testing.T) {
	eventBus := NewEventBus()
	has := eventBus.Off(123)
	assert.False(t, has)
}

func TestEmit(t *testing.T) {
	eventBus := NewEventBus()

	counter := 0
	ch := make(chan int)

	eventBus.On("hello", func(context EventBusContext) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	eventBus.Emit("hello", nson.Message{"counter": nson.I32(1)})
	<-ch

	assert.Exactly(t, 1, counter)

	eventBus.Emit("unknow", nson.Message{"counter": nson.I32(1)})
	assert.Exactly(t, 1, counter)

	eventBus.Emit("hello", nson.Message{"counter": nson.I32(2)})
	<-ch

	assert.Exactly(t, 3, counter)
}

func TestEmit2(t *testing.T) {
	eventBus := NewEventBus()

	counter := 0
	ch := make(chan int)

	eventBus.On("hello", func(context EventBusContext) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	eventBus.On("hello", func(context EventBusContext) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	eventBus.Emit("hello", nson.Message{"counter": nson.I32(1)})
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)

	eventBus.Emit("unknow", nson.Message{"counter": nson.I32(1)})
	assert.Exactly(t, 2, counter)

	eventBus.Emit("hello", nson.Message{"counter": nson.I32(2)})
	<-ch
	<-ch

	assert.Exactly(t, 6, counter)
}
