package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

func seedVerificationExpiring(t *testing.T, s store.Store, tn tenant, userID id.UserID, token string, expiresAt time.Time) {
	t.Helper()
	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: userID,
		Token: token, Type: account.VerificationEmail, ExpiresAt: expiresAt, CreatedAt: now(),
	}
	require.NoError(t, s.CreateVerification(context.Background(), v))
}

func seedPasswordResetExpiring(t *testing.T, s store.Store, tn tenant, userID id.UserID, token string, expiresAt time.Time) {
	t.Helper()
	pr := &account.PasswordReset{
		ID: id.NewPasswordResetID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: userID,
		Token: token, ExpiresAt: expiresAt, CreatedAt: now(),
	}
	require.NoError(t, s.CreatePasswordReset(context.Background(), pr))
}

// testRetentionDeletesOnlyExpiredRows proves each sweep removes rows past
// the cutoff and nothing else: a session whose refresh token is still valid
// survives even though its access token has expired, and a revocation
// record survives until the cutoff passes its revocation time.
func testRetentionDeletesOnlyExpiredRows(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "retention@test.com")
	sfx := suffix(u.ID.String())
	cutoff := now().Add(-24 * time.Hour)

	dead := seedSession(t, s, tn, u.ID, "dead-"+sfx, "dead-r-"+sfx)
	dead.ExpiresAt, dead.RefreshTokenExpiresAt = cutoff.Add(-2*time.Hour), cutoff.Add(-time.Hour)
	require.NoError(t, s.UpdateSession(ctx, dead))
	refreshable := seedSession(t, s, tn, u.ID, "refr-"+sfx, "refr-r-"+sfx)
	refreshable.ExpiresAt, refreshable.RefreshTokenExpiresAt = cutoff.Add(-2*time.Hour), now().Add(time.Hour)
	require.NoError(t, s.UpdateSession(ctx, refreshable))
	live := seedSession(t, s, tn, u.ID, "live-"+sfx, "live-r-"+sfx)

	n, err := s.DeleteExpiredSessions(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "only the session with both tokens expired goes")
	_, err = s.GetSession(ctx, dead.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetSession(ctx, refreshable.ID)
	assert.NoError(t, err, "a session that can still be refreshed stays")
	_, err = s.GetSession(ctx, live.ID)
	assert.NoError(t, err)

	seedVerificationExpiring(t, s, tn, u.ID, "v-old-"+sfx, cutoff.Add(-time.Hour))
	seedVerificationExpiring(t, s, tn, u.ID, "v-new-"+sfx, now().Add(time.Hour))
	n, err = s.DeleteExpiredVerifications(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	_, err = s.GetVerification(ctx, "v-old-"+sfx)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetVerification(ctx, "v-new-"+sfx)
	assert.NoError(t, err)

	seedPasswordResetExpiring(t, s, tn, u.ID, "pr-old-"+sfx, cutoff.Add(-time.Hour))
	seedPasswordResetExpiring(t, s, tn, u.ID, "pr-new-"+sfx, now().Add(time.Hour))
	n, err = s.DeleteExpiredPasswordResets(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	_, err = s.GetPasswordReset(ctx, "pr-old-"+sfx)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetPasswordReset(ctx, "pr-new-"+sfx)
	assert.NoError(t, err)

	hash := store.HashToken("revoked-" + sfx)
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, id.NewSessionFamilyID(), session.RevokeReasonRotated))
	n, err = s.DeleteExpiredRevokedRefreshTokens(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.Zero(t, n, "a record revoked after the cutoff stays")
	revoked, err := s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.True(t, revoked)
	n, err = s.DeleteExpiredRevokedRefreshTokens(ctx, now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	revoked, err = s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.False(t, revoked, "once swept the token is no longer known as revoked")
}

// testRetentionHonoursBatch proves a bounded batch removes no more than it
// was asked to and reports what it removed, so a sweeper can loop until a
// batch comes back short, and that a batch of zero removes everything.
func testRetentionHonoursBatch(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "batch@test.com")
	sfx := suffix(u.ID.String())
	cutoff := now().Add(-24 * time.Hour)
	for _, name := range []string{"a", "b", "c"} {
		seedVerificationExpiring(t, s, tn, u.ID, "b-"+name+"-"+sfx, cutoff.Add(-time.Hour))
	}

	n, err := s.DeleteExpiredVerifications(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n, "a batch of two removes two")
	n, err = s.DeleteExpiredVerifications(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "the next batch removes what is left")
	n, err = s.DeleteExpiredVerifications(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.Zero(t, n, "nothing left")

	for _, name := range []string{"d", "e", "f"} {
		seedVerificationExpiring(t, s, tn, u.ID, "b-"+name+"-"+sfx, cutoff.Add(-time.Hour))
	}
	n, err = s.DeleteExpiredVerifications(ctx, cutoff, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(3), "a batch of zero removes everything expired")
}
