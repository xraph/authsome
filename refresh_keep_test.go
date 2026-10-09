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

// A rotation that keeps the refresh token mints a new access token only:
// the refresh token the client holds still refreshes afterwards, and is
// not recorded as spent.
func TestRefreshBySessionToken_KeepRefreshToken(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "keep@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)
	originalRefresh := sess.RefreshToken

	kept, err := eng.RefreshBySessionToken(ctx, sess.Token, authsome.RefreshOpts{KeepRefreshToken: true})
	require.NoError(t, err)
	assert.NotEqual(t, sess.Token, kept.Token, "the access token rotates")
	_, err = eng.ResolveSessionByToken(ctx, sess.Token)
	assert.Error(t, err, "the old access token is gone")

	again, err := eng.Refresh(ctx, originalRefresh)
	require.NoError(t, err, "the refresh token the client holds is still good")
	assert.NotEqual(t, originalRefresh, again.RefreshToken, "an explicit refresh rotates it as usual")
}
