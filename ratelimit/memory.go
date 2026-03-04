package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryLimiter is an in-memory sliding window rate limiter.
type MemoryLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
}

type window struct {
	timestamps []time.Time
}

// NewMemoryLimiter creates a new in-memory rate limiter.
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		windows: make(map[string]*window),
	}
}

var _ Limiter = (*MemoryLimiter)(nil)

// Allow checks if a request is allowed under the sliding window.
func (l *MemoryLimiter) Allow(_ context.Context, key string, limit int, dur time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w := l.getOrCreate(key)
	w.prune(now, dur)

	if len(w.timestamps) >= limit {
		return false, nil
	}

	w.timestamps = append(w.timestamps, now)
	return true, nil
}

// Remaining returns how many requests are left in the current window.
func (l *MemoryLimiter) Remaining(_ context.Context, key string, limit int, dur time.Duration) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w := l.getOrCreate(key)
	w.prune(now, dur)

	remaining := limit - len(w.timestamps)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (l *MemoryLimiter) getOrCreate(key string) *window {
	w, ok := l.windows[key]
	if !ok {
		w = &window{}
		l.windows[key] = w
	}
	return w
}

// prune removes timestamps outside the sliding window.
func (w *window) prune(now time.Time, dur time.Duration) {
	cutoff := now.Add(-dur)
	i := 0
	for i < len(w.timestamps) && w.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		w.timestamps = w.timestamps[i:]
	}
}
