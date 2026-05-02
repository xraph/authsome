package memory_test

import (
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

func hashTok(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func TestRefreshReplay_NotRevokedByDefault(t *testing.T) {
	s := newStore()
	revoked, err := s.IsRefreshTokenRevoked(ctx(), hashTok("never-seen"))
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestRefreshReplay_MarkAndCheck(t *testing.T) {
	s := newStore()
	fam := id.NewSessionFamilyID()
	h := hashTok("rotated-token")

	require.NoError(t, s.MarkRefreshTokenRevoked(ctx(), h, fam, session.RevokeReasonRotated))

	revoked, err := s.IsRefreshTokenRevoked(ctx(), h)
	require.NoError(t, err)
	assert.True(t, revoked)

	gotFam, err := s.GetRevokedRefreshTokenFamily(ctx(), h)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), gotFam.String())
}

func TestRefreshReplay_MarkIdempotent(t *testing.T) {
	s := newStore()
	fam := id.NewSessionFamilyID()
	h := hashTok("dup")

	require.NoError(t, s.MarkRefreshTokenRevoked(ctx(), h, fam, session.RevokeReasonRotated))
	// Second call with a *different* family should be a no-op (keep first).
	other := id.NewSessionFamilyID()
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx(), h, other, session.RevokeReasonReplayDetected))

	gotFam, err := s.GetRevokedRefreshTokenFamily(ctx(), h)
	require.NoError(t, err)
	assert.Equal(t, fam.String(), gotFam.String(), "idempotent: first family wins")
}

func TestRefreshReplay_GetRevokedFamily_NotFound(t *testing.T) {
	s := newStore()
	_, err := s.GetRevokedRefreshTokenFamily(ctx(), hashTok("missing"))
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestRefreshReplay_RevokeFamilyCascade(t *testing.T) {
	s := newStore()
	fam := id.NewSessionFamilyID()
	otherFam := id.NewSessionFamilyID()
	now := time.Now()

	// Two sibling sessions in the family + one unrelated session.
	siblingA := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 id.NewAppID(),
		UserID:                id.NewUserID(),
		FamilyID:              fam,
		Token:                 "tokA",
		RefreshToken:          "refreshA",
		ExpiresAt:             now.Add(time.Hour),
		RefreshTokenExpiresAt: now.Add(24 * time.Hour),
	}
	siblingB := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 siblingA.AppID,
		UserID:                siblingA.UserID,
		FamilyID:              fam,
		Token:                 "tokB",
		RefreshToken:          "refreshB",
		ExpiresAt:             now.Add(time.Hour),
		RefreshTokenExpiresAt: now.Add(24 * time.Hour),
	}
	unrelated := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 siblingA.AppID,
		UserID:                siblingA.UserID,
		FamilyID:              otherFam,
		Token:                 "tokC",
		RefreshToken:          "refreshC",
		ExpiresAt:             now.Add(time.Hour),
		RefreshTokenExpiresAt: now.Add(24 * time.Hour),
	}
	require.NoError(t, s.CreateSession(ctx(), siblingA))
	require.NoError(t, s.CreateSession(ctx(), siblingB))
	require.NoError(t, s.CreateSession(ctx(), unrelated))

	require.NoError(t, s.RevokeRefreshTokenFamily(ctx(), fam, session.RevokeReasonReplayDetected))

	// Both family sessions are gone.
	_, errA := s.GetSession(ctx(), siblingA.ID)
	_, errB := s.GetSession(ctx(), siblingB.ID)
	assert.ErrorIs(t, errA, store.ErrNotFound)
	assert.ErrorIs(t, errB, store.ErrNotFound)

	// Unrelated session survives.
	got, err := s.GetSession(ctx(), unrelated.ID)
	require.NoError(t, err)
	assert.Equal(t, unrelated.ID.String(), got.ID.String())

	// Both sibling refresh-token hashes are now in the revoked set.
	for _, tok := range []string{"refreshA", "refreshB"} {
		revoked, err := s.IsRefreshTokenRevoked(ctx(), hashTok(tok))
		require.NoError(t, err)
		assert.Truef(t, revoked, "expected refresh hash for %q to be revoked", tok)
	}

	// The unrelated token is NOT revoked by the cascade.
	revoked, err := s.IsRefreshTokenRevoked(ctx(), hashTok("refreshC"))
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestRefreshReplay_RevokeFamilyNoop_NilFamily(t *testing.T) {
	s := newStore()
	require.NoError(t, s.RevokeRefreshTokenFamily(ctx(), id.Nil, session.RevokeReasonReplayDetected))
}
