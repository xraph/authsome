package mongo

import (
	"context"
	"errors"
	"sync"

	"github.com/xraph/authsome/id"
)

// Refresh-token replay detection — STUB.
//
// TODO(replay): implement for SQL/mongo. See store/postgres/refresh_replay.go
// for rationale. The Mongo follow-up will use a dedicated
// `revoked_refresh_tokens` collection with a unique index on token_hash
// plus a TTL index keyed off revoked_at to age out old rows.

var mongoReplayWarnOnce sync.Once

func mongoReplayStubWarn() {
	mongoReplayWarnOnce.Do(func() {
		// nolint:forbidigo // intentional one-shot stderr line; package has no logger handle
		println("authsome/mongo: refresh-replay detection not yet implemented")
	})
}

// IsRefreshTokenRevoked is a stub that always returns false.
func (s *Store) IsRefreshTokenRevoked(_ context.Context, _ string) (bool, error) {
	mongoReplayStubWarn()
	return false, nil
}

// MarkRefreshTokenRevoked is a stub no-op.
func (s *Store) MarkRefreshTokenRevoked(_ context.Context, _ string, _ id.SessionFamilyID, _ string) error {
	mongoReplayStubWarn()
	return nil
}

// GetRevokedRefreshTokenFamily is a stub returning "not implemented".
func (s *Store) GetRevokedRefreshTokenFamily(_ context.Context, _ string) (id.SessionFamilyID, error) {
	mongoReplayStubWarn()
	return id.Nil, errors.New("authsome/mongo: GetRevokedRefreshTokenFamily not implemented")
}

// RevokeRefreshTokenFamily is a stub no-op.
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, _ id.SessionFamilyID, _ string) error {
	mongoReplayStubWarn()
	return nil
}
