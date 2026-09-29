package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/session"
)

// A session ends thirty days after it was issued, however active it is, and
// neither the sliding window nor a refresh may carry it past that.

func agedSession(t *testing.T, age time.Duration) (*authsome.Engine, *session.Session) {
	t.Helper()
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "aged@example.com", Password: "SecureP@ss123", FirstName: "A",
	})
	require.NoError(t, err)

	// Age the row: issued long ago, but still active by its own clocks.
	sess.CreatedAt = time.Now().Add(-age)
	sess.ExpiresAt = time.Now().Add(time.Hour)
	sess.RefreshTokenExpiresAt = time.Now().Add(24 * time.Hour)
	require.NoError(t, eng.Store().UpdateSession(context.Background(), sess))
	return eng, sess
}

func TestAbsoluteLifetime_DefaultsToThirtyDays(t *testing.T) {
	eng, _ := newTestEngine(t)
	assert.Equal(t, 30*24*time.Hour, eng.AbsoluteLifetimeFor(context.Background(), id.Nil))
}

func TestAbsoluteLifetime_ResolveRefusesAnOldSession(t *testing.T) {
	eng, sess := agedSession(t, 31*24*time.Hour)
	_, err := eng.ResolveSessionByToken(sess.Token)
	assert.ErrorIs(t, err, account.ErrSessionExpired)
}

func TestAbsoluteLifetime_RefreshRefusesAnOldSession(t *testing.T) {
	eng, sess := agedSession(t, 31*24*time.Hour)
	_, err := eng.Refresh(context.Background(), sess.RefreshToken)
	assert.ErrorIs(t, err, account.ErrSessionExpired)
}

func TestAbsoluteLifetime_RefreshClampsToTheDeadline(t *testing.T) {
	eng, sess := agedSession(t, 29*24*time.Hour+23*time.Hour)
	deadline := sess.CreatedAt.Add(30 * 24 * time.Hour)

	rotated, err := eng.Refresh(context.Background(), sess.RefreshToken)
	require.NoError(t, err)
	assert.False(t, rotated.ExpiresAt.After(deadline), "the rotated access token stops at the deadline")
	assert.False(t, rotated.RefreshTokenExpiresAt.After(deadline), "the rotated refresh token stops at the deadline")
}

func TestAbsoluteLifetime_YoungSessionIsUntouched(t *testing.T) {
	eng, sess := agedSession(t, time.Hour)
	got, err := eng.ResolveSessionByToken(sess.Token)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
}
