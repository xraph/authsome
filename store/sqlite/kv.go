package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/store"
)

// KVModel is one authsome_kv row: a caller-chosen key, an opaque value, a
// counter for increments, and an expiry in unix milliseconds.
type KVModel struct {
	grove.BaseModel `grove:"table:authsome_kv,alias:kv"`

	Key       string `grove:"key,pk"`
	Value     []byte `grove:"value"`
	Counter   int64  `grove:"counter,notnull"`
	ExpiresAt int64  `grove:"expires_at,notnull"`
}

// KVGet implements store.KV.
func (s *Store) KVGet(ctx context.Context, key string) ([]byte, error) {
	m := new(KVModel)
	err := s.sdb.NewSelect(m).
		Where("key = ?", key).
		// expires_at is an integer of unix milliseconds, not a text timestamp,
		// so no zone binding applies; the operand order keeps the store's
		// timestamp guard from reading it as one.
		Where("? < expires_at", time.Now().UnixMilli()).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return m.Value, nil
}

// KVSet implements store.KV.
func (s *Store) KVSet(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if value == nil {
		value = []byte{}
	}
	_, err := s.sdb.NewRaw(`
INSERT INTO authsome_kv (key, value, counter, expires_at) VALUES (?, ?, 0, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value, counter = 0, expires_at = excluded.expires_at`,
		key, value, store.KVExpiry(time.Now(), ttl)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: kv set: %w", sqliteError(err))
	}
	return nil
}

// KVSetNX implements store.KV.
func (s *Store) KVSetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	now := time.Now()
	if value == nil {
		value = []byte{}
	}
	res, err := s.sdb.NewRaw(`
INSERT INTO authsome_kv (key, value, counter, expires_at) VALUES (?, ?, 0, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value, counter = 0, expires_at = excluded.expires_at
WHERE authsome_kv.expires_at <= ?`,
		key, value, store.KVExpiry(now, ttl), now.UnixMilli()).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("authsome/sqlite: kv setnx: %w", sqliteError(err))
	}
	n, _ := res.RowsAffected() //nolint:errcheck // sqlite always reports rows affected
	return n > 0, nil
}

// KVIncrement implements store.KV.
func (s *Store) KVIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	var counter int64
	err := s.sdb.NewRaw(`
INSERT INTO authsome_kv (key, value, counter, expires_at) VALUES (?, X'', 1, ?)
ON CONFLICT (key) DO UPDATE SET
    counter    = CASE WHEN authsome_kv.expires_at <= ? THEN 1 ELSE authsome_kv.counter + 1 END,
    expires_at = CASE WHEN authsome_kv.expires_at <= ? THEN excluded.expires_at ELSE authsome_kv.expires_at END
RETURNING counter`,
		key, store.KVExpiry(now, ttl), nowMs, nowMs).Scan(ctx, &counter)
	if err != nil {
		return 0, fmt.Errorf("authsome/sqlite: kv increment: %w", sqliteError(err))
	}
	return counter, nil
}

// KVCounter implements store.KV.
func (s *Store) KVCounter(ctx context.Context, key string) (int64, error) {
	m := new(KVModel)
	err := s.sdb.NewSelect(m).
		Where("key = ?", key).
		Where("? < expires_at", time.Now().UnixMilli()).
		Scan(ctx)
	if err != nil {
		if errors.Is(sqliteError(err), store.ErrNotFound) {
			return 0, nil
		}
		return 0, sqliteError(err)
	}
	return m.Counter, nil
}

// KVDelete implements store.KV.
func (s *Store) KVDelete(ctx context.Context, key string) error {
	_, err := s.sdb.NewDelete((*KVModel)(nil)).Where("key = ?", key).Exec(ctx)
	if err != nil && !errors.Is(sqliteError(err), store.ErrNotFound) {
		return fmt.Errorf("authsome/sqlite: kv delete: %w", sqliteError(err))
	}
	return nil
}

// KVDeleteExpired implements store.KV.
func (s *Store) KVDeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.sdb.NewDelete((*KVModel)(nil)).Where("? >= expires_at", now.UnixMilli()).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("authsome/sqlite: kv delete expired: %w", sqliteError(err))
	}
	n, _ := res.RowsAffected() //nolint:errcheck // sqlite always reports rows affected
	return n, nil
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "create_kv",
		Version: "20260922000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			if _, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_kv (
    key        TEXT PRIMARY KEY,
    value      BLOB NOT NULL DEFAULT X'',
    counter    INTEGER NOT NULL DEFAULT 0,
    expires_at INTEGER NOT NULL
);`); err != nil {
				return err
			}
			_, err := exec.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_authsome_kv_expires_at ON authsome_kv (expires_at);`)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_kv;`)
			return err
		},
	})
}
