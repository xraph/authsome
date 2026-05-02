//go:build integration

package sqlite_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// setupTestStore opens a fresh in-process sqlite store backed by a temp file
// (in-memory mode does not always survive grove's connection pooling).
func setupTestStore(t *testing.T) *sqlitestore.Store {
	t.Helper()
	ctx := context.Background()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?cache=shared&_pragma=foreign_keys(1)"

	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn), "open grove sqlite connection")

	db, err := grove.Open(sdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { _ = db.Close() })

	s := sqlitestore.New(db)
	require.NoError(t, s.Migrate(ctx), "run migrations")
	return s
}

func hashForTest(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

func TestRefreshReplay_MarkAndCheck(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	hash := "deadbeefcafef00d"
	fam := id.NewSessionFamilyID()

	revoked, err := s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.False(t, revoked)

	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, fam, session.RevokeReasonRotated))

	revoked, err = s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.True(t, revoked)

	got, err := s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), got.String())

	// Idempotent.
	other := id.NewSessionFamilyID()
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, other, session.RevokeReasonReplayDetected))
	got, err = s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), got.String(), "first writer wins")
}

func TestRefreshReplay_RevokeFamilyCascades(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Need an app and user for FK-free schemas (sqlite migrations don't
	// declare FKs on sessions; safe to insert sessions directly).
	fam := id.NewSessionFamilyID()
	other := id.NewSessionFamilyID()

	mkSess := func(family id.SessionFamilyID, suffix string) *session.Session {
		return &session.Session{
			ID:                    id.NewSessionID(),
			AppID:                 id.NewAppID(),
			EnvID:                 id.NewEnvironmentID(),
			UserID:                id.NewUserID(),
			FamilyID:              family,
			Token:                 "tok_" + suffix,
			RefreshToken:          "rtk_" + suffix,
			ExpiresAt:             time.Now().Add(time.Hour),
			RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}
	}

	famSess := mkSess(fam, "fam1")
	famSess2 := mkSess(fam, "fam2")
	otherSess := mkSess(other, "other1")

	require.NoError(t, s.CreateSession(ctx, famSess))
	require.NoError(t, s.CreateSession(ctx, famSess2))
	require.NoError(t, s.CreateSession(ctx, otherSess))

	require.NoError(t, s.RevokeRefreshTokenFamily(ctx, fam, session.RevokeReasonReplayDetected))

	// Family refresh tokens revoked.
	for _, rt := range []string{famSess.RefreshToken, famSess2.RefreshToken} {
		revoked, err := s.IsRefreshTokenRevoked(ctx, hashForTest(rt))
		require.NoError(t, err)
		assert.True(t, revoked, "token %q should be revoked", rt)
	}

	// Other family refresh token un-revoked.
	revoked, err := s.IsRefreshTokenRevoked(ctx, hashForTest(otherSess.RefreshToken))
	require.NoError(t, err)
	assert.False(t, revoked)

	// Family sessions deleted.
	sessions, err := s.ListUserSessions(ctx, famSess.UserID)
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Other family session still present.
	sessions, err = s.ListUserSessions(ctx, otherSess.UserID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}
