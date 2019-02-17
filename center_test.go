package queen

import (
	"testing"

	nson "github.com/danclive/nson-go"
	"github.com/stretchr/testify/assert"
)

func TestNewCenter(t *testing.T) {
	center := NewCenter()
	assert.NotNil(t, center.maps)
	assert.NotNil(t, center.handles)
}

func TestInitCenter(t *testing.T) {
	center := Center{}
	center.InitCenter()
	assert.NotNil(t, center.maps)
	assert.NotNil(t, center.handles)
}

func TestCenterOn(t *testing.T) {
	center := NewCenter()

	center.On("hello", func(context CContext) {})
	handles, ok := center.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(center.handles))

	center.On("hello", func(context CContext) {})
	handles, ok = center.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(center.handles))

	_, ok = center.handles["unkonw"]
	assert.False(t, ok)
}

func TestCenterAll(t *testing.T) {
	center := NewCenter()
	center.All(func(context CContext) {})

	assert.Exactly(t, 1, len(center.all))

	center.All(func(context CContext) {})
	center.All(func(context CContext) {})

	assert.Exactly(t, 3, len(center.all))
}

func TestCenterOff(t *testing.T) {
	center := NewCenter()

	id := center.On("hello", func(context CContext) {})

	handles, ok := center.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(center.handles))

	has := center.Off(id)
	assert.True(t, has)

	handles, ok = center.handles["hello"]
	assert.False(t, ok)
	assert.Exactly(t, 0, len(handles))
	assert.Exactly(t, 0, len(center.handles))
}

func TestCenterOff2(t *testing.T) {
	center := NewCenter()

	id1 := center.On("hello", func(context CContext) {})
	id2 := center.On("hello", func(context CContext) {})

	handles, ok := center.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(center.handles))

	has := center.Off(id1)
	assert.True(t, has)
	_ = id2

	handles, ok = center.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(center.handles))
}

func TestCenterOff3(t *testing.T) {
	center := NewCenter()
	has := center.Off(123)
	assert.False(t, has)
}

func TestCenterOff4(t *testing.T) {
	center := NewCenter()

	id := center.All(func(context CContext) {})
	assert.Exactly(t, 1, len(center.all))

	center.Off(id)
	assert.Exactly(t, 0, len(center.all))

	center.All(func(context CContext) {})
	id2 := center.All(func(context CContext) {})
	center.All(func(context CContext) {})
	center.Off(id2)

	assert.Exactly(t, 2, len(center.all))
}

func TestCenterInsert(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.On("hello", func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestCenterInsert2(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.On("hello", func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.On("hello", func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestCenterInsert3(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.On("hello", func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	center.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestCenterInsert4(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.All(func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestCenterInsert5(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.All(func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.All(func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestCenterInsert6(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.All(func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	center.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestCenterInsert7(t *testing.T) {
	center := NewCenter()

	counter := 0
	ch := make(chan int)

	center.On("hello", func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.All(func(context CContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	center.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestCenterGet(t *testing.T) {
	center := NewCenter()

	center.Insert("hello", nson.I32(123))
	value, has := center.Get("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	_, has = center.Get("unknow")
	assert.Exactly(t, false, has)
}

func TestCenterRemove(t *testing.T) {
	center := NewCenter()

	center.Insert("hello", nson.I32(123))
	value, has := center.Get("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	value, has = center.Remove("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	_, has = center.Get("hello")
	assert.Exactly(t, false, has)
}
