// Package boundedmap is a map with a capacity: once full, putting a new key
// evicts the least recently used one. Plugins that keep per-principal or
// per-address state in memory use it so that state cannot grow without
// bound over the life of a process.
//
// A Map is not safe for concurrent use; callers hold their own lock, as
// they already did around the plain maps this replaces.
package boundedmap

// Map holds at most capacity entries in least-recently-used order.
type Map[K comparable, V any] struct {
	capacity int
	items    map[K]*node[K, V]
	// front is the most recently used entry, back the least.
	front, back *node[K, V]
}

// node is one entry on a doubly linked list kept in recency order. The list
// is typed rather than container/list so no value ever crosses an interface.
type node[K comparable, V any] struct {
	key        K
	value      V
	prev, next *node[K, V]
}

// New returns an empty map that holds at most capacity entries. A capacity
// of zero or less means no bound, for callers that opt out.
func New[K comparable, V any](capacity int) *Map[K, V] {
	return &Map[K, V]{capacity: capacity, items: make(map[K]*node[K, V])}
}

// Get returns the value under key and marks it recently used.
func (m *Map[K, V]) Get(key K) (V, bool) {
	n, ok := m.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	m.moveToFront(n)
	return n.value, true
}

// Put stores value under key as the most recently used entry, evicting the
// least recently used one when the map is full.
func (m *Map[K, V]) Put(key K, value V) {
	if n, ok := m.items[key]; ok {
		n.value = value
		m.moveToFront(n)
		return
	}
	n := &node[K, V]{key: key, value: value}
	m.items[key] = n
	m.pushFront(n)
	for m.capacity > 0 && len(m.items) > m.capacity {
		oldest := m.back
		m.unlink(oldest)
		delete(m.items, oldest.key)
	}
}

// Delete removes key. Deleting an absent key does nothing.
func (m *Map[K, V]) Delete(key K) {
	if n, ok := m.items[key]; ok {
		m.unlink(n)
		delete(m.items, key)
	}
}

// Len returns how many entries the map holds.
func (m *Map[K, V]) Len() int { return len(m.items) }

// Range calls fn for each entry, most recently used first, until fn returns
// false. fn must not modify the map.
func (m *Map[K, V]) Range(fn func(key K, value V) bool) {
	for n := m.front; n != nil; n = n.next {
		if !fn(n.key, n.value) {
			return
		}
	}
}

func (m *Map[K, V]) pushFront(n *node[K, V]) {
	n.prev, n.next = nil, m.front
	if m.front != nil {
		m.front.prev = n
	}
	m.front = n
	if m.back == nil {
		m.back = n
	}
}

func (m *Map[K, V]) unlink(n *node[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		m.front = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		m.back = n.prev
	}
	n.prev, n.next = nil, nil
}

func (m *Map[K, V]) moveToFront(n *node[K, V]) {
	if m.front == n {
		return
	}
	m.unlink(n)
	m.pushFront(n)
}
