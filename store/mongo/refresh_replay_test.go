//go:build integration

package mongo_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	_ "github.com/xraph/grove/drivers/mongodriver/mongomigrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	mongostore "github.com/xraph/authsome/store/mongo"
)

// setupTestStore opens a fresh mongo-backed store. Skips when AUTHSOME_MONGO_URI
// is unset so the build-only path stays passable in CI without a live mongo.
func setupTestStore(t *testing.T) *mongostore.Store {
	t.Helper()
	uri := os.Getenv("AUTHSOME_MONGO_URI")
	if uri == "" {
		t.Skip("AUTHSOME_MONGO_URI not set; skipping mongo integration test")
	}
	ctx := context.Background()

	mdb := mongodriver.New()
	require.NoError(t, mdb.Open(ctx, uri), "open grove mongo connection")

	db, err := grove.Open(mdb)
	require.NoError(t, err, "open grove db")
	t.Cleanup(func() { _ = db.Close() })

	s := mongostore.New(db)
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

	hash := "deadbeefcafef00d-mongo"
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

	other := id.NewSessionFamilyID()
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, other, session.RevokeReasonReplayDetected))
	got, err = s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), got.String(), "first writer wins")
}

func TestRefreshReplay_RevokeFamilyCascades(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

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

	for _, rt := range []string{famSess.RefreshToken, famSess2.RefreshToken} {
		revoked, err := s.IsRefreshTokenRevoked(ctx, hashForTest(rt))
		require.NoError(t, err)
		assert.True(t, revoked, "token %q should be revoked", rt)
	}

	revoked, err := s.IsRefreshTokenRevoked(ctx, hashForTest(otherSess.RefreshToken))
	require.NoError(t, err)
	assert.False(t, revoked)
}
