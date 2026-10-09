package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/authsome/store"
)

// ttlFields maps each retention kind to the collection and the date field
// MongoDB expires it on. Sessions expire on the refresh token's expiry,
// since a session with a live refresh token can still be refreshed.
var ttlFields = map[string]struct{ col, field string }{
	store.RetentionSessions:             {colSessions, "refresh_token_expires_at"},
	store.RetentionVerifications:        {colVerifications, "expires_at"},
	store.RetentionPasswordResets:       {colPasswordResets, "expires_at"},
	store.RetentionRevokedRefreshTokens: {colRevokedRefreshTokens, "revoked_at"},
}

// EnsureTTLIndexes implements store.TTLIndexer. MongoDB's TTL monitor then
// removes rows as they pass their window, and the engine's sweeper only
// has to catch what the monitor has not reached yet.
func (s *Store) EnsureTTLIndexes(ctx context.Context, ttl map[string]time.Duration) error {
	for kind, window := range ttl {
		spec, ok := ttlFields[kind]
		if !ok || window <= 0 {
			continue
		}
		secs := int32(window / time.Second)
		if err := s.ensureTTLIndex(ctx, spec.col, spec.field, secs); err != nil {
			return fmt.Errorf("authsome/mongo: ttl index on %s.%s: %w", spec.col, spec.field, err)
		}
	}
	return nil
}

// ensureTTLIndex leaves col with exactly one index on field, a TTL index of
// secs. An existing TTL index with another window is modified in place; a
// plain index on the same field is replaced, since a TTL index serves the
// same queries and MongoDB refuses two indexes on one key pattern.
func (s *Store) ensureTTLIndex(ctx context.Context, col, field string, secs int32) error {
	coll := s.mdb.Collection(col)
	cur, err := coll.Indexes().List(ctx)
	if err != nil {
		return err
	}
	var existing []struct {
		Name               string `bson:"name"`
		Key                bson.D `bson:"key"`
		ExpireAfterSeconds *int32 `bson:"expireAfterSeconds"`
	}
	if allErr := cur.All(ctx, &existing); allErr != nil {
		return allErr
	}
	for _, ix := range existing {
		if len(ix.Key) != 1 || ix.Key[0].Key != field {
			continue
		}
		if ix.ExpireAfterSeconds != nil {
			if *ix.ExpireAfterSeconds == secs {
				return nil
			}
			return s.mdb.Database().RunCommand(ctx, bson.D{
				{Key: "collMod", Value: col},
				{Key: "index", Value: bson.D{
					{Key: "name", Value: ix.Name},
					{Key: "expireAfterSeconds", Value: secs},
				}},
			}).Err()
		}
		if dropErr := coll.Indexes().DropOne(ctx, ix.Name); dropErr != nil {
			return dropErr
		}
	}
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: field, Value: 1}},
		Options: options.Index().SetName("ttl_" + field).SetExpireAfterSeconds(secs),
	})
	return err
}
