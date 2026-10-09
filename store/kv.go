package store

import (
	"context"
	"time"
)

// KV is a small shared key-value space with expiry that every backend
// provides on the authsome_kv table. Rate limiting, lockout and ceremony
// state are built on it so those controls hold across replicas instead of
// living in one process's memory.
//
// Expiry is stored as unix milliseconds on every backend so comparisons do
// not depend on timestamp parsing. Expired rows are invisible to every read
// and are reclaimed by KVDeleteExpired.
type KV interface {
	// KVGet returns the value stored under key, or ErrNotFound when the key
	// is absent or expired.
	KVGet(ctx context.Context, key string) ([]byte, error)

	// KVSet stores value under key for ttl, replacing any existing value.
	KVSet(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// KVSetNX stores value under key for ttl only when the key is absent or
	// expired, and reports whether it did.
	KVSetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)

	// KVIncrement adds one to the counter under key and returns the new
	// count. An absent or expired key starts at one with the given ttl; an
	// existing key keeps its expiry.
	KVIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// KVCounter returns the counter under key without changing it, or zero
	// when the key is absent or expired.
	KVCounter(ctx context.Context, key string) (int64, error)

	// KVDelete removes key. Deleting an absent key is not an error.
	KVDelete(ctx context.Context, key string) error

	// KVDeleteExpired removes every row whose expiry is before now and
	// returns how many it removed.
	KVDeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// KVExpiry converts a TTL from now into the unix-millisecond expiry every
// backend stores. A zero or negative ttl yields a row that is already
// expired; every caller passes a positive ttl, and the conformance suite
// uses negative ones to plant expired rows.
func KVExpiry(now time.Time, ttl time.Duration) int64 {
	return now.Add(ttl).UnixMilli()
}
