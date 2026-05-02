package mongo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// Refresh-token replay detection — native MongoDB implementation.
//
// Backed by the `authsome_revoked_refresh_tokens` collection (migration 20)
// and a `family_id` index on `authsome_sessions` (migration 19). Inserts
// are idempotent: duplicate-key errors on token_hash are silently swallowed.
// RevokeRefreshTokenFamily uses a MongoTx (requires a replica set or sharded
// cluster — standalone mongod returns ErrUnsupportedDeployment).

// IsRefreshTokenRevoked reports whether tokenHash has been recorded as revoked.
func (s *Store) IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	if tokenHash == "" {
		return false, nil
	}
	var m revokedRefreshTokenModel
	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": tokenHash}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return false, nil
		}
		return false, fmt.Errorf("authsome/mongo: lookup revoked refresh token: %w", err)
	}
	return true, nil
}

// MarkRefreshTokenRevoked records tokenHash as revoked. Idempotent: a
// duplicate-key error is swallowed (the original record wins).
func (s *Store) MarkRefreshTokenRevoked(ctx context.Context, tokenHash string, familyID id.SessionFamilyID, reason string) error {
	if tokenHash == "" {
		return nil
	}
	m := &revokedRefreshTokenModel{
		TokenHash: tokenHash,
		FamilyID:  familyID.String(),
		RevokedAt: time.Now().UTC(),
		Reason:    reason,
	}
	if _, err := s.mdb.NewInsert(m).Exec(ctx); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("authsome/mongo: insert revoked refresh token: %w", err)
	}
	return nil
}

// GetRevokedRefreshTokenFamily returns the family bound to a revoked token,
// or store.ErrNotFound if the hash is unknown.
func (s *Store) GetRevokedRefreshTokenFamily(ctx context.Context, tokenHash string) (id.SessionFamilyID, error) {
	if tokenHash == "" {
		return id.Nil, store.ErrNotFound
	}
	var m revokedRefreshTokenModel
	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": tokenHash}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return id.Nil, store.ErrNotFound
		}
		return id.Nil, fmt.Errorf("authsome/mongo: lookup revoked refresh token: %w", err)
	}
	if m.FamilyID == "" {
		return id.Nil, nil
	}
	famID, err := id.ParseSessionFamilyID(m.FamilyID)
	if err != nil {
		return id.Nil, err
	}
	return famID, nil
}

// RevokeRefreshTokenFamily revokes every active session sharing familyID,
// recording each refresh-token hash as revoked. Runs inside a MongoTx.
//
// MongoDB transactions require a replica set or sharded cluster. On a
// standalone mongod this returns ErrUnsupportedDeployment.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID id.SessionFamilyID, reason string) error {
	if familyID.IsNil() {
		return nil
	}
	famStr := familyID.String()
	now := time.Now().UTC()

	mtx, err := s.mdb.GroveTx(ctx, 0, false)
	if err != nil {
		return fmt.Errorf("authsome/mongo: begin tx for refresh-family revoke: %w", err)
	}
	tx, ok := mtx.(*mongodriver.MongoTx)
	if !ok {
		return fmt.Errorf("authsome/mongo: unexpected tx type %T", mtx)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback() //nolint:errcheck // best-effort rollback
		}
	}()

	var sessions []sessionModel
	if err := tx.NewFind(&sessions).
		Filter(bson.M{"family_id": famStr}).
		Scan(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: list family sessions: %w", err)
	}

	for i := range sessions {
		sess := sessions[i]
		if sess.RefreshToken == "" {
			continue
		}
		h := hashRefreshTokenMongo(sess.RefreshToken)
		rec := &revokedRefreshTokenModel{
			TokenHash: h,
			FamilyID:  famStr,
			RevokedAt: now,
			Reason:    reason,
		}
		if _, err := tx.NewInsert(rec).Exec(ctx); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return fmt.Errorf("authsome/mongo: insert revoked sibling: %w", err)
		}
	}

	if _, err := tx.NewDelete((*sessionModel)(nil)).
		Many().
		Filter(bson.M{"family_id": famStr}).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: delete family sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("authsome/mongo: commit refresh-family revoke: %w", err)
	}
	committed = true
	return nil
}

// hashRefreshTokenMongo returns hex(SHA-256(tok)).
func hashRefreshTokenMongo(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

