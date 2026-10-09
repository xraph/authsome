package oauth2provider

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ──────────────────────────────────────────────────
// Memory
// ──────────────────────────────────────────────────

func (s *MemoryStore) DeleteExpiredAuthCodes(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, c := range s.codes {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if c.ExpiresAt.Before(before) {
			delete(s.codes, k)
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) DeleteExpiredDeviceCodesBefore(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, dc := range s.deviceCodes {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if dc.ExpiresAt.Before(before) {
			delete(s.deviceCodes, k)
			n++
		}
	}
	return n, nil
}

// ──────────────────────────────────────────────────
// Postgres
// ──────────────────────────────────────────────────

// pgSweep runs one batched delete; LIMIT NULL is postgres for "no limit".
func (s *PostgresStore) pgSweep(ctx context.Context, query string, before time.Time, batch int) (int64, error) {
	var limit any
	if batch > 0 {
		limit = batch
	}
	res, err := s.pg.NewRaw(query, before, limit).Exec(ctx)
	if err != nil {
		return 0, oauth2PgError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // pgx always reports rows affected
	return n, nil
}

func (s *PostgresStore) DeleteExpiredAuthCodes(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.pgSweep(ctx, `
DELETE FROM authsome_oauth2_auth_codes WHERE id IN (
    SELECT id FROM authsome_oauth2_auth_codes WHERE expires_at < $1 LIMIT $2)`, before, batch)
}

func (s *PostgresStore) DeleteExpiredDeviceCodesBefore(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.pgSweep(ctx, `
DELETE FROM authsome_oauth2_device_codes WHERE id IN (
    SELECT id FROM authsome_oauth2_device_codes WHERE expires_at < $1 LIMIT $2)`, before, batch)
}

// ──────────────────────────────────────────────────
// SQLite
// ──────────────────────────────────────────────────

// sqliteSweep runs one batched delete; a negative LIMIT is sqlite for "no
// limit". The cutoff is bound in UTC and placed on the left because
// expires_at is TEXT here and compares as a string.
func (s *SqliteStore) sqliteSweep(ctx context.Context, query string, before time.Time, batch int) (int64, error) {
	limit := -1
	if batch > 0 {
		limit = batch
	}
	res, err := s.sdb.NewRaw(query, utc(before), limit).Exec(ctx)
	if err != nil {
		return 0, oauth2SqliteError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // sqlite always reports rows affected
	return n, nil
}

func (s *SqliteStore) DeleteExpiredAuthCodes(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sqliteSweep(ctx, `
DELETE FROM authsome_oauth2_auth_codes WHERE id IN (
    SELECT id FROM authsome_oauth2_auth_codes WHERE ? > expires_at LIMIT ?)`, before, batch)
}

func (s *SqliteStore) DeleteExpiredDeviceCodesBefore(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sqliteSweep(ctx, `
DELETE FROM authsome_oauth2_device_codes WHERE id IN (
    SELECT id FROM authsome_oauth2_device_codes WHERE ? > expires_at LIMIT ?)`, before, batch)
}

// ──────────────────────────────────────────────────
// Mongo
// ──────────────────────────────────────────────────

// mongoSweep deletes at most batch documents that expired before the cutoff.
// A delete cannot take a limit, so a bounded batch reads its ids first.
func (s *MongoStore) mongoSweep(ctx context.Context, col string, before time.Time, batch int) (int64, error) {
	coll := s.mdb.Collection(col)
	filter := bson.M{"expires_at": bson.M{"$lt": before}}
	if batch <= 0 {
		res, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			return 0, oauth2MongoError(err)
		}
		return res.DeletedCount, nil
	}
	cur, err := coll.Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(int64(batch)))
	if err != nil {
		return 0, oauth2MongoError(err)
	}
	// _id is carried back as whatever the collection uses (a string id or
	// a native ObjectID) and handed straight back to the delete.
	var docs []struct {
		ID any `bson:"_id"`
	}
	if err = cur.All(ctx, &docs); err != nil {
		return 0, oauth2MongoError(err)
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
		return 0, oauth2MongoError(err)
	}
	return res.DeletedCount, nil
}

func (s *MongoStore) DeleteExpiredAuthCodes(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.mongoSweep(ctx, oauth2AuthCodesColl, before, batch)
}

func (s *MongoStore) DeleteExpiredDeviceCodesBefore(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.mongoSweep(ctx, oauth2DeviceCodesColl, before, batch)
}
