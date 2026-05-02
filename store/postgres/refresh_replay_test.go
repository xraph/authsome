//go:build integration

package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

func hashForTest(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// TestRefreshReplay_MarkAndCheck exercises the basic mark/check round trip
// against a live postgres container.
func TestRefreshReplay_MarkAndCheck(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	hash := "deadbeefcafef00d"
	fam := id.NewSessionFamilyID()

	// Initially the hash is not revoked.
	revoked, err := s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.False(t, revoked)

	// Mark it revoked.
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, fam, session.RevokeReasonRotated))

	revoked, err = s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.True(t, revoked)

	got, err := s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), got.String())

	// Idempotent: a duplicate insert with a different family is a silent no-op
	// (original wins).
	other := id.NewSessionFamilyID()
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, other, session.RevokeReasonReplayDetected))
	got, err = s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), got.String(), "first writer wins")
}

// TestRefreshReplay_GetUnknownReturnsNotFound checks the miss path.
func TestRefreshReplay_GetUnknownReturnsNotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	_, err := s.GetRevokedRefreshTokenFamily(ctx, "no-such-hash")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// TestRefreshReplay_RevokeFamilyCascades verifies that RevokeRefreshTokenFamily
// records every sibling session's refresh token as revoked and deletes the
// underlying sessions.
func TestRefreshReplay_RevokeFamilyCascades(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	a := createTestApp(t, s, "refresh-replay")
	u := createTestUser(t, s, a.ID, "refresh-user@test.com")

	fam := id.NewSessionFamilyID()
	other := id.NewSessionFamilyID()

	mkSess := func(family id.SessionFamilyID, suffix string) *session.Session {
		return &session.Session{
			ID:                    id.NewSessionID(),
			AppID:                 a.ID,
			UserID:                u.ID,
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

	// Family sessions deleted.
	_, err := s.GetSession(ctx, famSess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetSession(ctx, famSess2.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Other family untouched.
	_, err = s.GetSession(ctx, otherSess.ID)
	require.NoError(t, err)

	// Both family refresh tokens recorded as revoked.
	for _, rt := range []string{famSess.RefreshToken, famSess2.RefreshToken} {
		h := hashForTest(rt)
		revoked, err := s.IsRefreshTokenRevoked(ctx, h)
		require.NoError(t, err)
		assert.True(t, revoked, "token %q should be revoked", rt)
	}

	// Other family's refresh token still un-revoked.
	revoked, err := s.IsRefreshTokenRevoked(ctx, hashForTest(otherSess.RefreshToken))
	require.NoError(t, err)
	assert.False(t, revoked)
}
