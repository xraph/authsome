package scim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSCIMStore_returnsTheStoreTheHostCanRead(t *testing.T) {
	p := &Plugin{}
	assert.Nil(t, p.SCIMStore(), "no store before one is wired")
	s := NewMemoryStore()
	p.SetSCIMStore(s)
	assert.Same(t, s, p.SCIMStore())
}
