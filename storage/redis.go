package storage

import (
    "context"
    "time"

    rl "github.com/xraph/authsome/core/ratelimit"
    "github.com/redis/go-redis/v9"
)

// RedisStorage implements rl.Storage using Redis counters with windowed expiry.
type RedisStorage struct {
    client *redis.Client
}

// NewRedisStorage creates a new RedisStorage.
func NewRedisStorage(client *redis.Client) *RedisStorage {
    return &RedisStorage{client: client}
}

// Increment increases the counter for key within the given window and returns the current value.
// If the key is new or has no expiry, it sets the expiry to the window duration.
func (s *RedisStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
    // Use a pipeline to INCR and ensure expiry is set if missing.
    pipe := s.client.Pipeline()
    incr := pipe.Incr(ctx, key)
    ttl := pipe.TTL(ctx, key)
    if _, err := pipe.Exec(ctx); err != nil {
        return 0, err
    }

    // If no expiry (-1) or key does not exist (-2 before INCR), set expiry.
    t, err := ttl.Result()
    if err != nil {
        // Some Redis versions may return error on TTL for new keys; set expiry regardless.
        _ = s.client.Expire(ctx, key, window).Err()
    } else if t <= 0 {
        _ = s.client.Expire(ctx, key, window).Err()
    }

    n, err := incr.Result()
    if err != nil {
        return 0, err
    }
    return int(n), nil
}

// Assert RedisStorage implements rl.Storage
var _ rl.Storage = (*RedisStorage)(nil)