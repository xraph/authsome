package sso

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/store"
)

// The one-time code a browser redeems after an identity provider callback
// names the session and carries no token, and the exchange hands out a
// rotated pair rather than the pair minted at the callback.

func TestExchange_CodeCarriesNoTokenAndRotatesTheSession(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	p := New()
	require.NoError(t, p.OnInit(ctx, eng))

	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "otc@example.com", Password: "SecureP@ss123", FirstName: "O"})
	require.NoError(t, err)

	code, err := p.mintOTC(ctx, appID, &CallbackResponse{
		User: u, SessionToken: sess.Token, RefreshToken: sess.RefreshToken, Provider: "okta", sessionID: sess.ID,
	})
	require.NoError(t, err)

	raw, err := p.ceremonies.Get(ctx, "sso:otc:"+code)
	require.NoError(t, err)
	var pl map[string]any
	require.NoError(t, json.Unmarshal(raw, &pl))
	assert.Equal(t, sess.ID.String(), pl["session_id"])
	_, hasToken := pl["session_token"]
	_, hasRefresh := pl["refresh_token"]
	assert.False(t, hasToken || hasRefresh, "the stashed payload must carry no credential")
	assert.NotContains(t, string(raw), sess.Token)
	assert.NotContains(t, string(raw), sess.RefreshToken)

	router := forge.NewRouter()
	require.NoError(t, router.POST("/exchange", p.handleExchange))
	exchange := func(code string) (int, CallbackResponse) {
		body, _ := json.Marshal(map[string]string{"code": code})
		req := httptest.NewRequestWithContext(ctx, "POST", "/exchange", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp CallbackResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec.Code, resp
	}
	status, resp := exchange(code)
	require.Equal(t, 200, status)
	assert.NotEmpty(t, resp.SessionToken)
	assert.NotEqual(t, sess.Token, resp.SessionToken, "the exchange returns a rotated access token")
	assert.NotEqual(t, sess.RefreshToken, resp.RefreshToken, "the exchange returns a rotated refresh token")

	live, err := eng.ResolveSessionByToken(context.Background(), resp.SessionToken)
	require.NoError(t, err, "the rotated token authenticates")
	assert.Equal(t, sess.ID.String(), live.ID.String(), "the same session, rotated")
	_, err = eng.ResolveSessionByToken(context.Background(), sess.Token)
	assert.Error(t, err, "the token minted at the callback is retired by the exchange")
	revoked, err := eng.Store().IsRefreshTokenRevoked(ctx, store.HashToken(sess.RefreshToken))
	require.NoError(t, err)
	assert.True(t, revoked, "the callback refresh token is revoked, not merely replaced")

	status, _ = exchange(code)
	assert.Equal(t, 400, status, "a one-time code redeems once")
}
