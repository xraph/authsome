package ceremony

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/store"
)

// KVStore keeps ceremony state in the shared KV store so a ceremony begun
// on one replica can finish on another, and a replay marker claimed on one
// replica is seen by all.
type KVStore struct {
	kv store.KV
}

// NewKV builds a ceremony store on kv.
func NewKV(kv store.KV) *KVStore { return &KVStore{kv: kv} }

var _ Store = (*KVStore)(nil)

func kvKey(key string) string { return "cer:" + key }

// Set implements Store.
func (s *KVStore) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return s.kv.KVSet(ctx, kvKey(key), data, ttl)
}

// Get implements Store.
func (s *KVStore) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.kv.KVGet(ctx, kvKey(key))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Delete implements Store.
func (s *KVStore) Delete(ctx context.Context, key string) error {
	return s.kv.KVDelete(ctx, kvKey(key))
}

// SetNX implements Store.
func (s *KVStore) SetNX(ctx context.Context, key string, data []byte, ttl time.Duration) (bool, error) {
	return s.kv.KVSetNX(ctx, kvKey(key), data, ttl)
}

// Increment implements Store.
func (s *KVStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return s.kv.KVIncrement(ctx, kvKey(key), ttl)
}
