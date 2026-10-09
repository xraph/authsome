package lockout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xraph/authsome/store"
)

// KVTracker keeps lockout state in the shared KV store so a lockout holds
// across every replica.
//
// Per key it keeps a failure counter that expires after the reset window
// and, once the threshold is crossed, a lock marker that expires with the
// lockout. Keys are grouped under a prefix (app:identifier) with the
// client's network as the suffix; an index row per prefix records the
// suffixes seen so ResetPrefix can clear them without scanning the store.
type KVTracker struct {
	kv              store.KV
	maxAttempts     int
	lockoutDuration time.Duration
	resetAfter      time.Duration
}

// NewKVTracker builds a tracker on kv. The options are the same ones the
// memory tracker takes.
func NewKVTracker(kv store.KV, opts ...MemoryOption) *KVTracker {
	m := NewMemoryTracker(opts...)
	return &KVTracker{
		kv:              kv,
		maxAttempts:     m.maxAttempts,
		lockoutDuration: m.lockoutDuration,
		resetAfter:      m.resetAfter,
	}
}

var _ Tracker = (*KVTracker)(nil)

func attemptsKey(key string) string { return "lo:" + key + ":n" }
func untilKey(key string) string    { return "lo:" + key + ":until" }
func indexKey(prefix string) string { return "lo:idx:" + prefix }

// splitKey returns the index prefix (everything up to and including the
// last colon) and the suffix after it.
func splitKey(key string) (prefix, suffix string) {
	i := strings.LastIndexByte(key, ':')
	if i < 0 {
		return "", key
	}
	return key[:i+1], key[i+1:]
}

// RecordFailure counts a failed attempt and sets the lock once the
// threshold is reached.
func (t *KVTracker) RecordFailure(ctx context.Context, key string) (int, error) {
	n, err := t.kv.KVIncrement(ctx, attemptsKey(key), t.resetAfter)
	if err != nil {
		return 0, fmt.Errorf("lockout: record failure: %w", err)
	}
	if err := t.remember(ctx, key); err != nil {
		return int(n), err
	}
	if int(n) >= t.maxAttempts {
		until := time.Now().Add(t.lockoutDuration)
		if err := t.kv.KVSet(ctx, untilKey(key), []byte(strconv.FormatInt(until.UnixMilli(), 10)), t.lockoutDuration); err != nil {
			return int(n), fmt.Errorf("lockout: set lock: %w", err)
		}
	}
	return int(n), nil
}

// IsLocked reports whether key is locked and until when.
func (t *KVTracker) IsLocked(ctx context.Context, key string) (bool, time.Time, error) {
	raw, err := t.kv.KVGet(ctx, untilKey(key))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, fmt.Errorf("lockout: read lock: %w", err)
	}
	ms, parseErr := strconv.ParseInt(string(raw), 10, 64)
	if parseErr != nil {
		return false, time.Time{}, nil
	}
	until := time.UnixMilli(ms)
	if !time.Now().Before(until) {
		return false, time.Time{}, nil
	}
	return true, until, nil
}

// Reset clears the failure count and the lock for a key.
func (t *KVTracker) Reset(ctx context.Context, key string) error {
	if err := t.kv.KVDelete(ctx, attemptsKey(key)); err != nil {
		return fmt.Errorf("lockout: reset: %w", err)
	}
	if err := t.kv.KVDelete(ctx, untilKey(key)); err != nil {
		return fmt.Errorf("lockout: reset: %w", err)
	}
	return nil
}

// ResetPrefix clears every key recorded under prefix.
func (t *KVTracker) ResetPrefix(ctx context.Context, prefix string) error {
	raw, err := t.kv.KVGet(ctx, indexKey(prefix))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lockout: read index: %w", err)
	}
	var suffixes []string
	if err := json.Unmarshal(raw, &suffixes); err != nil {
		return fmt.Errorf("lockout: decode index: %w", err)
	}
	for _, s := range suffixes {
		if err := t.Reset(ctx, prefix+s); err != nil {
			return err
		}
	}
	return t.kv.KVDelete(ctx, indexKey(prefix))
}

// remember adds key's suffix to its prefix index so ResetPrefix can find
// it. The index lives as long as the longest window it covers.
func (t *KVTracker) remember(ctx context.Context, key string) error {
	prefix, suffix := splitKey(key)
	if prefix == "" {
		return nil
	}
	suffixes := make([]string, 0, 1)
	if raw, err := t.kv.KVGet(ctx, indexKey(prefix)); err == nil {
		_ = json.Unmarshal(raw, &suffixes) //nolint:errcheck // a corrupt index is rebuilt below
	}
	for _, s := range suffixes {
		if s == suffix {
			return nil
		}
	}
	suffixes = append(suffixes, suffix)
	raw, err := json.Marshal(suffixes)
	if err != nil {
		return fmt.Errorf("lockout: encode index: %w", err)
	}
	ttl := t.resetAfter
	if t.lockoutDuration > ttl {
		ttl = t.lockoutDuration
	}
	if err := t.kv.KVSet(ctx, indexKey(prefix), raw, ttl); err != nil {
		return fmt.Errorf("lockout: write index: %w", err)
	}
	return nil
}
