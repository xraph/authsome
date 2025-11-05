package storage

import (
	"context"
	"sync"
	"time"

	rl "github.com/xraph/authsome/core/ratelimit"
)

// MemoryStorage is an in-memory rate limit storage implementation
type MemoryStorage struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	expiresAt time.Time
	count     int
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{buckets: make(map[string]*bucket)}
}

// Ensure MemoryStorage implements rl.Storage
var _ rl.Storage = (*MemoryStorage)(nil)

func (m *MemoryStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	_ = ctx
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.buckets[key]
	if !ok || now.After(b.expiresAt) {
		m.buckets[key] = &bucket{expiresAt: now.Add(window), count: 1}
		return 1, nil
	}
	b.count++
	return b.count, nil
}
