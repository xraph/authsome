package memory

import (
	"context"
	"time"

	"github.com/xraph/authsome/store"
)

// kvEntry is one authsome_kv row.
type kvEntry struct {
	value     []byte
	counter   int64
	expiresAt int64 // unix milliseconds
}

func (e *kvEntry) expired(nowMs int64) bool { return e.expiresAt <= nowMs }

// KVGet implements store.KV.
func (s *Store) KVGet(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.kv[key]
	if !ok || e.expired(time.Now().UnixMilli()) {
		return nil, store.ErrNotFound
	}
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, nil
}

// KVSet implements store.KV.
func (s *Store) KVSet(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := make([]byte, len(value))
	copy(v, value)
	s.kv[key] = &kvEntry{value: v, expiresAt: store.KVExpiry(time.Now(), ttl)}
	return nil
}

// KVSetNX implements store.KV.
func (s *Store) KVSetNX(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if e, ok := s.kv[key]; ok && !e.expired(now.UnixMilli()) {
		return false, nil
	}
	v := make([]byte, len(value))
	copy(v, value)
	s.kv[key] = &kvEntry{value: v, expiresAt: store.KVExpiry(now, ttl)}
	return true, nil
}

// KVIncrement implements store.KV.
func (s *Store) KVIncrement(_ context.Context, key string, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	e, ok := s.kv[key]
	if !ok || e.expired(now.UnixMilli()) {
		e = &kvEntry{expiresAt: store.KVExpiry(now, ttl)}
		s.kv[key] = e
	}
	e.counter++
	return e.counter, nil
}

// KVDelete implements store.KV.
func (s *Store) KVDelete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.kv, key)
	return nil
}

// KVDeleteExpired implements store.KV.
func (s *Store) KVDeleteExpired(_ context.Context, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nowMs := now.UnixMilli()
	var n int64
	for k, e := range s.kv {
		if e.expired(nowMs) {
			delete(s.kv, k)
			n++
		}
	}
	return n, nil
}
