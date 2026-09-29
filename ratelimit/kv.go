package ratelimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/xraph/authsome/store"
)

// KVLimiter is a fixed-window limiter on the shared KV store, so a limit
// holds across every replica that shares the database.
//
// Each key gets one counter per window, keyed by the window's start, and
// the counter expires with the window. A fixed window lets up to twice the
// limit through around a boundary; the limits it guards are small, and one
// increment per request is what keeps the shared store cheap.
type KVLimiter struct {
	kv store.KV
}

// NewKVLimiter builds a limiter on kv.
func NewKVLimiter(kv store.KV) *KVLimiter { return &KVLimiter{kv: kv} }

var _ Limiter = (*KVLimiter)(nil)

func windowKey(key string, now time.Time, dur time.Duration) string {
	if dur <= 0 {
		dur = time.Minute
	}
	start := now.UnixNano() / int64(dur)
	return "rl:" + key + ":" + strconv.FormatInt(start, 10)
}

// Allow counts the request against the current window and reports whether
// it fits under limit.
func (l *KVLimiter) Allow(ctx context.Context, key string, limit int, dur time.Duration) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	n, err := l.kv.KVIncrement(ctx, windowKey(key, time.Now(), dur), dur)
	if err != nil {
		return false, err
	}
	return n <= int64(limit), nil
}

// Remaining reports how many requests are left in the current window
// without counting one. The KV contract exposes the count only through an
// increment, so this answers the full limit for an untouched window and
// zero once the window has been used; the middleware only reads it after a
// refusal, where zero is the truth.
func (l *KVLimiter) Remaining(ctx context.Context, key string, limit int, dur time.Duration) (int, error) {
	_, err := l.kv.KVGet(ctx, windowKey(key, time.Now(), dur))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return limit, nil
		}
		return 0, err
	}
	return 0, nil
}
