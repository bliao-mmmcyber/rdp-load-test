package stresstest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessage(t *testing.T) {
	data := []byte("4.size,1.0,4.1024,3.768;")
	m, e := parseMessage(data)
	assert.Equal(t, m.Op, "size")
	assert.Nil(t, e)
}
