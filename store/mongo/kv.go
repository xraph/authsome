package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/store"
)

// kvModel is one authsome_kv document: a caller-chosen key, an opaque value,
// a counter for increments, and an expiry in unix milliseconds.
type kvModel struct {
	grove.BaseModel `grove:"table:authsome_kv"`

	Key       string `grove:"key,pk"    bson:"_id"`
	Value     []byte `grove:"value"     bson:"value"`
	Counter   int64  `grove:"counter"   bson:"counter"`
	ExpiresAt int64  `grove:"expires_at" bson:"expires_at"`
}

func (s *Store) kvColl() *mongo.Collection { return s.mdb.Collection(colKV) }

// KVGet implements store.KV.
func (s *Store) KVGet(ctx context.Context, key string) ([]byte, error) {
	var m kvModel
	err := s.kvColl().FindOne(ctx, bson.M{"_id": key, "expires_at": bson.M{"$gt": time.Now().UnixMilli()}}).Decode(&m)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: kv get: %w", err)
	}
	return m.Value, nil
}

// KVSet implements store.KV.
func (s *Store) KVSet(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if value == nil {
		value = []byte{}
	}
	_, err := s.kvColl().UpdateOne(ctx, bson.M{"_id": key},
		bson.M{"$set": bson.M{"value": value, "counter": int64(0), "expires_at": store.KVExpiry(time.Now(), ttl)}},
		options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("authsome/mongo: kv set: %w", err)
	}
	return nil
}

// KVSetNX implements store.KV.
func (s *Store) KVSetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	now := time.Now()
	if value == nil {
		value = []byte{}
	}
	// Clear an expired row first so the insert below can win; a duplicate
	// key on the insert means a live row exists.
	if _, err := s.kvColl().DeleteOne(ctx, bson.M{"_id": key, "expires_at": bson.M{"$lte": now.UnixMilli()}}); err != nil {
		return false, fmt.Errorf("authsome/mongo: kv setnx: clear expired: %w", err)
	}
	_, err := s.kvColl().InsertOne(ctx, kvModel{Key: key, Value: value, ExpiresAt: store.KVExpiry(now, ttl)})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, fmt.Errorf("authsome/mongo: kv setnx: %w", err)
	}
	return true, nil
}

// KVIncrement implements store.KV.
func (s *Store) KVIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	now := time.Now()
	if _, err := s.kvColl().DeleteOne(ctx, bson.M{"_id": key, "expires_at": bson.M{"$lte": now.UnixMilli()}}); err != nil {
		return 0, fmt.Errorf("authsome/mongo: kv increment: clear expired: %w", err)
	}
	var m kvModel
	err := s.kvColl().FindOneAndUpdate(ctx, bson.M{"_id": key},
		bson.M{
			"$inc":         bson.M{"counter": int64(1)},
			"$setOnInsert": bson.M{"value": []byte{}, "expires_at": store.KVExpiry(now, ttl)},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&m)
	if err != nil {
		return 0, fmt.Errorf("authsome/mongo: kv increment: %w", err)
	}
	return m.Counter, nil
}

// KVDelete implements store.KV.
func (s *Store) KVDelete(ctx context.Context, key string) error {
	if _, err := s.kvColl().DeleteOne(ctx, bson.M{"_id": key}); err != nil {
		return fmt.Errorf("authsome/mongo: kv delete: %w", err)
	}
	return nil
}

// KVDeleteExpired implements store.KV.
func (s *Store) KVDeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.kvColl().DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lte": now.UnixMilli()}})
	if err != nil {
		return 0, fmt.Errorf("authsome/mongo: kv delete expired: %w", err)
	}
	return res.DeletedCount, nil
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "create_kv",
		Version: "20260922000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			if err := mexec.CreateCollection(ctx, (*kvModel)(nil)); err != nil {
				return err
			}
			return mexec.CreateIndexes(ctx, colKV, []mongo.IndexModel{
				{Keys: bson.D{{Key: "expires_at", Value: 1}}},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			return mexec.DB().Collection(colKV).Drop(ctx)
		},
	})
}
