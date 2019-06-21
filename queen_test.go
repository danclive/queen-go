package queen

import (
	"testing"

	"github.com/danclive/nson-go"

	"github.com/stretchr/testify/assert"
)

func TestNewQueen(t *testing.T) {
	queen := NewQueen()
	assert.NotNil(t, queen.handles)
}

func TestInitQueen(t *testing.T) {
	queen := Queen{}
	queen.InitQueen()
	assert.NotNil(t, queen.handles)
}

func TestOn(t *testing.T) {
	queen := NewQueen()

	queen.On("hello", func(context Context) {})

	handles, ok := queen.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(queen.handles))

	queen.On("hello", func(context Context) {})
	handles, ok = queen.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(queen.handles))

	_, ok = queen.handles["unkonw"]
	assert.False(t, ok)
}

func TestOff(t *testing.T) {
	queen := NewQueen()

	id := queen.On("hello", func(context Context) {})

	handles, ok := queen.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(queen.handles))

	has := queen.Off(id)
	assert.True(t, has)

	handles, ok = queen.handles["hello"]
	assert.False(t, ok)
	assert.Exactly(t, 0, len(handles))
	assert.Exactly(t, 0, len(queen.handles))
}

func TestOff2(t *testing.T) {
	queen := NewQueen()

	id1 := queen.On("hello", func(context Context) {})
	id2 := queen.On("hello", func(context Context) {})

	handles, ok := queen.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(queen.handles))

	has := queen.Off(id1)
	assert.True(t, has)
	_ = id2

	handles, ok = queen.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(queen.handles))
}

func TestOff3(t *testing.T) {
	queen := NewQueen()
	has := queen.Off(123)
	assert.False(t, has)
}

func TestEmit(t *testing.T) {
	queen := NewQueen()

	counter := 0
	ch := make(chan int)

	queen.On("hello", func(context Context) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	queen.Emit("hello", nson.Message{"counter": nson.I32(1)})
	<-ch

	assert.Exactly(t, 1, counter)

	queen.Emit("unknow", nson.Message{"counter": nson.I32(1)})
	assert.Exactly(t, 1, counter)

	queen.Emit("hello", nson.Message{"counter": nson.I32(2)})
	<-ch

	assert.Exactly(t, 3, counter)
}

func TestEmit2(t *testing.T) {
	queen := NewQueen()

	counter := 0
	ch := make(chan int)

	queen.On("hello", func(context Context) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	queen.On("hello", func(context Context) {
		t, _ := context.Message.GetI32("counter")
		counter += int(t)
		ch <- 1
	})

	queen.Emit("hello", nson.Message{"counter": nson.I32(1)})
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)

	queen.Emit("unknow", nson.Message{"counter": nson.I32(1)})
	assert.Exactly(t, 2, counter)

	queen.Emit("hello", nson.Message{"counter": nson.I32(2)})
	<-ch
	<-ch

	assert.Exactly(t, 6, counter)
}
