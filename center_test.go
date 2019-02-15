package queen

import (
	"testing"

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
