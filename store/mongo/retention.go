package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// sweep deletes at most batch documents of col matching filter and reports
// how many went. A batch of zero or less deletes every match in one call;
// otherwise the ids are read first, since a delete cannot take a limit.
func (s *Store) sweep(ctx context.Context, col string, filter bson.M, batch int) (int64, error) {
	coll := s.mdb.Collection(col)
	if batch <= 0 {
		res, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			return 0, fmt.Errorf("authsome/mongo: delete expired %s: %w", col, err)
		}
		return res.DeletedCount, nil
	}
	cur, err := coll.Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(int64(batch)))
	if err != nil {
		return 0, fmt.Errorf("authsome/mongo: find expired %s: %w", col, err)
	}
	// _id is carried back as whatever the collection uses (a string id or
	// a native ObjectID) and handed straight back to the delete.
	var docs []struct {
		ID any `bson:"_id"`
	}
	if err = cur.All(ctx, &docs); err != nil {
		return 0, fmt.Errorf("authsome/mongo: read expired %s: %w", col, err)
	}
	if len(docs) == 0 {
		return 0, nil
	}
	ids := make([]any, len(docs))
	for i, d := range docs {
		ids[i] = d.ID
	}
	res, err := coll.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return 0, fmt.Errorf("authsome/mongo: delete expired %s: %w", col, err)
	}
	return res.DeletedCount, nil
}

// DeleteExpiredSessions implements store.Retention.
func (s *Store) DeleteExpiredSessions(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, colSessions, bson.M{
		"expires_at":               bson.M{"$lt": before},
		"refresh_token_expires_at": bson.M{"$lt": before},
	}, batch)
}

// DeleteExpiredVerifications implements store.Retention.
func (s *Store) DeleteExpiredVerifications(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, colVerifications, bson.M{"expires_at": bson.M{"$lt": before}}, batch)
}

// DeleteExpiredPasswordResets implements store.Retention.
func (s *Store) DeleteExpiredPasswordResets(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, colPasswordResets, bson.M{"expires_at": bson.M{"$lt": before}}, batch)
}

// DeleteExpiredRevokedRefreshTokens implements store.Retention.
func (s *Store) DeleteExpiredRevokedRefreshTokens(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, colRevokedRefreshTokens, bson.M{"revoked_at": bson.M{"$lt": before}}, batch)
}
