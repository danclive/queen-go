package queen

import (
	"testing"

	nson "github.com/danclive/nson-go"
	"github.com/stretchr/testify/assert"
)

func TestNewDataBus(t *testing.T) {
	dataBus := NewDataBus()
	assert.NotNil(t, dataBus.maps)
	assert.NotNil(t, dataBus.handles)
}

func TestInitDataBus(t *testing.T) {
	dataBus := DataBus{}
	dataBus.InitDataBus()
	assert.NotNil(t, dataBus.maps)
	assert.NotNil(t, dataBus.handles)
}

func TestDataBusOn(t *testing.T) {
	dataBus := NewDataBus()

	dataBus.On("hello", func(context DataBusContext) {})
	handles, ok := dataBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(dataBus.handles))

	dataBus.On("hello", func(context DataBusContext) {})
	handles, ok = dataBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(dataBus.handles))

	_, ok = dataBus.handles["unkonw"]
	assert.False(t, ok)
}

func TestDataBusAll(t *testing.T) {
	dataBus := NewDataBus()
	dataBus.All(func(context DataBusContext) {})

	assert.Exactly(t, 1, len(dataBus.all))

	dataBus.All(func(context DataBusContext) {})
	dataBus.All(func(context DataBusContext) {})

	assert.Exactly(t, 3, len(dataBus.all))
}

func TestDataBusOff(t *testing.T) {
	dataBus := NewDataBus()

	id := dataBus.On("hello", func(context DataBusContext) {})

	handles, ok := dataBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(dataBus.handles))

	has := dataBus.Off(id)
	assert.True(t, has)

	handles, ok = dataBus.handles["hello"]
	assert.False(t, ok)
	assert.Exactly(t, 0, len(handles))
	assert.Exactly(t, 0, len(dataBus.handles))
}

func TestdataBusOff2(t *testing.T) {
	dataBus := NewDataBus()

	id1 := dataBus.On("hello", func(context DataBusContext) {})
	id2 := dataBus.On("hello", func(context DataBusContext) {})

	handles, ok := dataBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 2, len(handles))
	assert.Exactly(t, 1, len(dataBus.handles))

	has := dataBus.Off(id1)
	assert.True(t, has)
	_ = id2

	handles, ok = dataBus.handles["hello"]
	assert.True(t, ok)
	assert.Exactly(t, 1, len(handles))
	assert.Exactly(t, 1, len(dataBus.handles))
}

func TestDataBusOff3(t *testing.T) {
	dataBus := NewDataBus()
	has := dataBus.Off(123)
	assert.False(t, has)
}

func TestdataBusOff4(t *testing.T) {
	dataBus := NewDataBus()

	id := dataBus.All(func(context DataBusContext) {})
	assert.Exactly(t, 1, len(dataBus.all))

	dataBus.Off(id)
	assert.Exactly(t, 0, len(dataBus.all))

	dataBus.All(func(context DataBusContext) {})
	id2 := dataBus.All(func(context DataBusContext) {})
	dataBus.All(func(context DataBusContext) {})
	dataBus.Off(id2)

	assert.Exactly(t, 2, len(dataBus.all))
}

func TestdataBusInsert(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.On("hello", func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	value, has := dataBus.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, nil, value)
	assert.Exactly(t, false, has)
	assert.Exactly(t, 1, counter)
}

func TestDataBusInsert2(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.On("hello", func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.On("hello", func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestDataBusInsert3(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.On("hello", func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	value, has := dataBus.Insert("hello", nson.I32(1))
	value2, has2 := dataBus.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, nil, value)
	assert.Exactly(t, false, has)
	assert.Exactly(t, nson.I32(1), value2)
	assert.Exactly(t, true, has2)
	assert.Exactly(t, 1, counter)
}

func TestDataBusInsert4(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.All(func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestDataBusInsert5(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.All(func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.All(func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestDataBusInsert6(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.All(func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.Insert("hello", nson.I32(1))
	dataBus.Insert("hello", nson.I32(1))
	<-ch

	assert.Exactly(t, 1, counter)
}

func TestDataBusInsert7(t *testing.T) {
	dataBus := NewDataBus()

	counter := 0
	ch := make(chan int)

	dataBus.On("hello", func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.All(func(context DataBusContext) {
		counter += int(context.Value.(nson.I32))
		ch <- 1
	})

	dataBus.Insert("hello", nson.I32(1))
	<-ch
	<-ch

	assert.Exactly(t, 2, counter)
}

func TestDataBusGet(t *testing.T) {
	dataBus := NewDataBus()

	dataBus.Insert("hello", nson.I32(123))
	value, has := dataBus.Get("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	_, has = dataBus.Get("unknow")
	assert.Exactly(t, false, has)
}

func TestDataBusRemove(t *testing.T) {
	dataBus := NewDataBus()

	dataBus.Insert("hello", nson.I32(123))
	value, has := dataBus.Get("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	value, has = dataBus.Remove("hello")
	assert.Exactly(t, true, has)
	assert.Exactly(t, nson.I32(123), value.(nson.I32))

	_, has = dataBus.Get("hello")
	assert.Exactly(t, false, has)
}

func TestDataBusSet(t *testing.T) {
	dataBus := NewDataBus()

	value, has := dataBus.Set("hello", nson.I32(123))

	assert.Exactly(t, nil, value)
	assert.Exactly(t, false, has)

	value2, has2 := dataBus.Get("hello")
	assert.Exactly(t, true, has2)
	assert.Exactly(t, nson.I32(123), value2.(nson.I32))
}
