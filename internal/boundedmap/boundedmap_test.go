package boundedmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvictsLeastRecentlyUsed(t *testing.T) {
	m := New[string, int](2)
	m.Put("a", 1)
	m.Put("b", 2)
	_, ok := m.Get("a") // a is now the most recently used
	assert.True(t, ok)
	m.Put("c", 3) // evicts b

	_, ok = m.Get("b")
	assert.False(t, ok, "the least recently used entry goes")
	v, ok := m.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
	assert.Equal(t, 2, m.Len())
}

func TestPutReplacesAndDeleteRemoves(t *testing.T) {
	m := New[string, int](3)
	m.Put("a", 1)
	m.Put("a", 2)
	assert.Equal(t, 1, m.Len(), "putting an existing key replaces it")
	v, _ := m.Get("a")
	assert.Equal(t, 2, v)

	m.Delete("a")
	m.Delete("missing")
	_, ok := m.Get("a")
	assert.False(t, ok)
	assert.Zero(t, m.Len())
}

func TestRangeMostRecentFirst(t *testing.T) {
	m := New[string, int](0) // unbounded
	for i, k := range []string{"a", "b", "c"} {
		m.Put(k, i)
	}
	var seen []string
	m.Range(func(k string, _ int) bool {
		seen = append(seen, k)
		return k != "b"
	})
	assert.Equal(t, []string{"c", "b"}, seen, "most recent first, and stops when told")
	for i := range 1000 {
		m.Put(fmt.Sprintf("k%d", i), i)
	}
	assert.Equal(t, 1003, m.Len(), "a capacity of zero means no bound")
}
