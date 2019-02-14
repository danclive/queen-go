package queen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEventEmiter(t *testing.T) {
	event_emiter := NewEventEmiter()
	assert.NotNil(t, event_emiter.handles)
}

func TestInitEventEmiter(t *testing.T) {
	event_emiter := EventEmiter{}
	event_emiter.InitEventEmiter()
	assert.NotNil(t, event_emiter.handles)
}

func TestOn(t *testing.T) {
	event_emiter := NewEventEmiter()

	event_emiter.On("hello", func(context Context) {})

	handles, ok := event_emiter.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(event_emiter.handles))

	event_emiter.On("hello", func(context Context) {})
	handles, ok = event_emiter.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(event_emiter.handles))

	_, ok = event_emiter.handles["unkonw"]
	assert.False(t, ok)
}

func TestOff(t *testing.T) {
	event_emiter := NewEventEmiter()

	id := event_emiter.On("hello", func(context Context) {})

	handles, ok := event_emiter.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(event_emiter.handles))

	has := event_emiter.Off(id)
	assert.True(t, has)

	handles, ok = event_emiter.handles["hello"]
	assert.False(t, ok)
	assert.Exactly(t, 0, len(handles))
	assert.Exactly(t, 0, len(event_emiter.handles))
}

func TestOff2(t *testing.T) {
	event_emiter := NewEventEmiter()

	id1 := event_emiter.On("hello", func(context Context) {})
	id2 := event_emiter.On("hello", func(context Context) {})

	handles, ok := event_emiter.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(event_emiter.handles))

	has := event_emiter.Off(id1)
	assert.True(t, has)
	_ = id2

	handles, ok = event_emiter.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(event_emiter.handles))
}

func TestOff3(t *testing.T) {
	event_emiter := NewEventEmiter()
	has := event_emiter.Off(123)
	assert.False(t, has)
}

func TestEmit(t *testing.T) {
	event_emiter := NewEventEmiter()

	counter := 0
	ch := make(chan int)

	event_emiter.On("hello", func(context Context) {
		counter += context.Message.(int)
		ch <- 1
	})

	event_emiter.Emit("hello", 1)
	<-ch

	assert.Exactly(t, 1, counter)

	event_emiter.Emit("unknow", 1)
	assert.Exactly(t, 1, counter)

	event_emiter.Emit("hello", 2)
	<-ch

	assert.Exactly(t, 3, counter)
}

func TestEmit2(t *testing.T) {
	event_emiter := NewEventEmiter()

	counter := 0
	ch := make(chan int)

	event_emiter.On("hello", func(context Context) {
		counter += context.Message.(int)
		ch <- 1
	})

	event_emiter.Emit("hello", 1)
	<-ch

	assert.Exactly(t, 1, counter)

	event_emiter.Emit("unknow", 1)
	assert.Exactly(t, 1, counter)

	event_emiter.Emit("hello", 2)
	<-ch

	assert.Exactly(t, 3, counter)
}
