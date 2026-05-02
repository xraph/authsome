package postgres

import (
	"context"
	"errors"
	"sync"

	"github.com/xraph/authsome/id"
)

// Refresh-token replay detection — STUB.
//
// TODO(replay): implement for SQL/mongo. Phase 3E.2 ships replay detection
// only on the in-memory backend; the PostgreSQL store keeps a no-op
// implementation so the build stays green and existing deployments retain
// their current "no replay detection" behaviour. The follow-up phase will
// add a `revoked_refresh_tokens(token_hash, family_id, revoked_at, reason)`
// table and a `sessions.family_id` column with the appropriate migration.

var pgReplayWarnOnce sync.Once

func pgReplayStubWarn() {
	pgReplayWarnOnce.Do(func() {
		// nolint:forbidigo // intentional one-shot stderr line; this package has no logger handle
		println("authsome/postgres: refresh-replay detection not yet implemented")
	})
}

// IsRefreshTokenRevoked is a stub that always reports the token is not
// revoked, so the engine treats every refresh as a legitimate rotation
// (no false replay alarms).
func (s *Store) IsRefreshTokenRevoked(_ context.Context, _ string) (bool, error) {
	pgReplayStubWarn()
	return false, nil
}

// MarkRefreshTokenRevoked is a stub that silently discards the record.
func (s *Store) MarkRefreshTokenRevoked(_ context.Context, _ string, _ id.SessionFamilyID, _ string) error {
	pgReplayStubWarn()
	return nil
}

// GetRevokedRefreshTokenFamily is a stub that returns "not implemented".
func (s *Store) GetRevokedRefreshTokenFamily(_ context.Context, _ string) (id.SessionFamilyID, error) {
	pgReplayStubWarn()
	return id.Nil, errors.New("authsome/postgres: GetRevokedRefreshTokenFamily not implemented")
}

// RevokeRefreshTokenFamily is a stub that silently no-ops.
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, _ id.SessionFamilyID, _ string) error {
	pgReplayStubWarn()
	return nil
}
