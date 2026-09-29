package storetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// legacySessionSeeder is the test seam every backend exposes to write a
// session the way releases before token hashing did: plaintext, no hashes.
type legacySessionSeeder interface {
	SeedLegacySession(ctx context.Context, sess *session.Session) error
}

// testSessionTokensStoredAsHashes proves a session read back by id carries no
// plaintext, that both hashes are the shared digest, and that a lookup by
// either plaintext returns exactly that plaintext and nothing else.
func testSessionTokensStoredAsHashes(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "hashed@test.com")
	sess := seedSession(t, s, tn, u.ID, "plain-access", "plain-refresh")

	assert.Equal(t, store.HashToken("plain-access"), sess.TokenHash, "create fills the caller's access hash")
	assert.Equal(t, store.HashToken("plain-refresh"), sess.RefreshTokenHash, "create fills the caller's refresh hash")
	assert.Equal(t, "plain-access", sess.Token, "create leaves the caller's plaintext alone")

	byID, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "no access plaintext at rest")
	assert.Empty(t, byID.RefreshToken, "no refresh plaintext at rest")
	assert.Equal(t, store.HashToken("plain-access"), byID.TokenHash)
	assert.Equal(t, store.HashToken("plain-refresh"), byID.RefreshTokenHash)

	byTok, err := s.GetSessionByToken(ctx, "plain-access")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byTok.ID.String())
	assert.Equal(t, "plain-access", byTok.Token, "the presented plaintext rides back on the session")
	assert.Empty(t, byTok.RefreshToken, "the other credential stays hidden")
	assert.Equal(t, store.HashToken("plain-access"), byTok.TokenHash)

	byRef, err := s.GetSessionByRefreshToken(ctx, "plain-refresh")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byRef.ID.String())
	assert.Equal(t, "plain-refresh", byRef.RefreshToken)
	assert.Empty(t, byRef.Token)
	assert.Equal(t, store.HashToken("plain-access"), byRef.TokenHash, "the access hash is what the refresh CAS keys on")

	_, err = s.GetSessionByToken(ctx, store.HashToken("plain-access"))
	assert.ErrorIs(t, err, store.ErrNotFound, "presenting the stored digest itself must not authenticate")
	_, err = s.GetSessionByToken(ctx, "")
	assert.ErrorIs(t, err, store.ErrNotFound, "an empty token never matches a row with empty plaintext")

	// An update through a session read by id must not erase the hashes.
	byID.UserAgent = "updated"
	require.NoError(t, s.UpdateSession(ctx, byID))
	again, err := s.GetSessionByToken(ctx, "plain-access")
	require.NoError(t, err)
	assert.Equal(t, "updated", again.UserAgent)
}

// testLegacyPlaintextSessionUpgrades proves a row written before hashing is
// still found by its plaintext, is rewritten as hashes on first use, and is
// then found by hash alone.
func testLegacyPlaintextSessionUpgrades(t *testing.T, s store.Store) {
	seeder, ok := s.(legacySessionSeeder)
	require.True(t, ok, "every backend must expose SeedLegacySession for this case")
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "legacy@test.com")
	sess := seedSession(t, s, tn, u.ID, "legacy-access", "legacy-refresh")
	require.NoError(t, s.DeleteSession(ctx, sess.ID))
	sess.TokenHash, sess.RefreshTokenHash = "", ""
	require.NoError(t, seeder.SeedLegacySession(ctx, sess))

	got, err := s.GetSessionByToken(ctx, "legacy-access")
	require.NoError(t, err, "a legacy plaintext row must still authenticate")
	assert.Equal(t, sess.ID.String(), got.ID.String())
	assert.Equal(t, store.HashToken("legacy-access"), got.TokenHash, "the upgraded row's hash rides back")

	byID, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "the first lookup rewrites the row without plaintext")
	assert.Empty(t, byID.RefreshToken)
	assert.Equal(t, store.HashToken("legacy-access"), byID.TokenHash)
	assert.Equal(t, store.HashToken("legacy-refresh"), byID.RefreshTokenHash, "the refresh token is upgraded in the same write")

	byRef, err := s.GetSessionByRefreshToken(ctx, "legacy-refresh")
	require.NoError(t, err, "the upgraded row is found by hash")
	assert.Equal(t, sess.ID.String(), byRef.ID.String())

	// A legacy row first touched through its refresh token upgrades too.
	other := seedSession(t, s, tn, u.ID, "legacy-access-2", "legacy-refresh-2")
	require.NoError(t, s.DeleteSession(ctx, other.ID))
	other.TokenHash, other.RefreshTokenHash = "", ""
	require.NoError(t, seeder.SeedLegacySession(ctx, other))
	_, err = s.GetSessionByRefreshToken(ctx, "legacy-refresh-2")
	require.NoError(t, err)
	byID, err = s.GetSession(ctx, other.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token)
	assert.Equal(t, store.HashToken("legacy-access-2"), byID.TokenHash)
}

// testRevokeFamilyUsesStoredHashes proves family revocation marks each
// sibling's refresh hash revoked without ever needing the plaintext.
func testRevokeFamilyUsesStoredHashes(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "family@test.com")
	a := seedSession(t, s, tn, u.ID, "fam-access-a", "fam-refresh-a")
	b := seedSession(t, s, tn, u.ID, "fam-access-b", "fam-refresh-b")
	b.FamilyID = a.FamilyID
	require.NoError(t, s.UpdateSession(ctx, b))

	require.NoError(t, s.RevokeRefreshTokenFamily(ctx, a.FamilyID, session.RevokeReasonReplayDetected))

	for _, plain := range []string{"fam-refresh-a", "fam-refresh-b"} {
		revoked, err := s.IsRefreshTokenRevoked(ctx, store.HashToken(plain))
		require.NoError(t, err)
		assert.True(t, revoked, "sibling %s must be in the revoked set", plain)
	}
	_, err := s.GetSession(ctx, a.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetSession(ctx, b.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}
