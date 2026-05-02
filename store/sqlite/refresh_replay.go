package sqlite

import (
	"context"
	"errors"
	"sync"

	"github.com/xraph/authsome/id"
)

// Refresh-token replay detection — STUB.
//
// TODO(replay): implement for SQL/mongo. See store/postgres/refresh_replay.go
// for rationale. SQLite mirrors the postgres stub.

var sqliteReplayWarnOnce sync.Once

func sqliteReplayStubWarn() {
	sqliteReplayWarnOnce.Do(func() {
		// nolint:forbidigo // intentional one-shot stderr line; package has no logger handle
		println("authsome/sqlite: refresh-replay detection not yet implemented")
	})
}

// IsRefreshTokenRevoked is a stub that always returns false.
func (s *Store) IsRefreshTokenRevoked(_ context.Context, _ string) (bool, error) {
	sqliteReplayStubWarn()
	return false, nil
}

// MarkRefreshTokenRevoked is a stub no-op.
func (s *Store) MarkRefreshTokenRevoked(_ context.Context, _ string, _ id.SessionFamilyID, _ string) error {
	sqliteReplayStubWarn()
	return nil
}

// GetRevokedRefreshTokenFamily is a stub returning "not implemented".
func (s *Store) GetRevokedRefreshTokenFamily(_ context.Context, _ string) (id.SessionFamilyID, error) {
	sqliteReplayStubWarn()
	return id.Nil, errors.New("authsome/sqlite: GetRevokedRefreshTokenFamily not implemented")
}

// RevokeRefreshTokenFamily is a stub no-op.
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, _ id.SessionFamilyID, _ string) error {
	sqliteReplayStubWarn()
	return nil
}
