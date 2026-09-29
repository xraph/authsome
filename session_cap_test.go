package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
)

// A cap of two means two: the third sign-in revokes the oldest session,
// and the newest two stay valid.
func TestMaxActiveSessions_EvictsOldest(t *testing.T) {
	cfg := testEngineConfig()
	cfg.Session.MaxActiveSessions = 2
	eng, _ := newTestEngine(t, authsome.WithConfig(cfg))
	secutil.RelaxAuthDefaults(t, eng)
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	_, first, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "cap@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)
	signIn := func() string {
		_, s, signInErr := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "cap@example.com", Password: "SecureP@ss123"})
		require.NoError(t, signInErr)
		return s.Token
	}
	second := signIn()
	_, err = eng.ResolveSessionByToken(ctx, first.Token)
	require.NoError(t, err, "two sessions fit under a cap of two")

	third := signIn()
	_, err = eng.ResolveSessionByToken(ctx, first.Token)
	assert.Error(t, err, "the oldest session is revoked to make room")
	_, err = eng.ResolveSessionByToken(ctx, second)
	assert.NoError(t, err)
	_, err = eng.ResolveSessionByToken(ctx, third)
	assert.NoError(t, err)

	sessions, err := eng.ListSessions(ctx, first.UserID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}
