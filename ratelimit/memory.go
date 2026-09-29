package ratelimit

import (
	"context"
	"sync"
	"time"
)

// DefaultMaxKeys bounds the memory limiter. A flood of distinct keys (one
// per spoofed address, say) would otherwise grow the map without limit; at
// the bound, idle windows are dropped before a new key is admitted.
const DefaultMaxKeys = 100_000

// MemoryLimiter is an in-memory sliding window rate limiter.
type MemoryLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	maxKeys int
}

type window struct {
	timestamps []time.Time
}

// NewMemoryLimiter creates a new in-memory rate limiter.
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		windows: make(map[string]*window),
		maxKeys: DefaultMaxKeys,
	}
}

// NewMemoryLimiterWithLimit creates a memory limiter bounded to maxKeys.
func NewMemoryLimiterWithLimit(maxKeys int) *MemoryLimiter {
	l := NewMemoryLimiter()
	if maxKeys > 0 {
		l.maxKeys = maxKeys
	}
	return l
}

// Len reports how many keys the limiter tracks.
func (l *MemoryLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.windows)
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
		l.makeRoomLocked()
		w = &window{}
		l.windows[key] = w
	}
	return w
}

// makeRoomLocked keeps the map under its bound: windows with no timestamps
// in the last minute go first, then arbitrary ones. Called with the lock
// held, before a new key is added.
func (l *MemoryLimiter) makeRoomLocked() {
	if len(l.windows) < l.maxKeys {
		return
	}
	cutoff := time.Now().Add(-time.Minute)
	for k, w := range l.windows {
		if len(w.timestamps) == 0 || w.timestamps[len(w.timestamps)-1].Before(cutoff) {
			delete(l.windows, k)
		}
	}
	for k := range l.windows {
		if len(l.windows) < l.maxKeys {
			break
		}
		delete(l.windows, k)
	}
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
