package authsome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/internal/secutil"
)

// TestEngine_SessionAuthProviderEnforcesDPoP drives the "session" provider the
// engine actually registers with the forge auth registry. That provider is a
// second path from a token to an authenticated context, reached by plugin
// authz and several plugins, and the binding has to hold on it without
// depending on the global auth middleware having run first.
func TestEngine_SessionAuthProviderEnforcesDPoP(t *testing.T) {
	t.Parallel()

	eng := secutil.NewTestEngine(t)
	ctx := context.Background()
	require.NoError(t, eng.InitPlugins(ctx))

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "provider-dpop@test.com",
		Password:  "SecureP@ss1",
		FirstName: "Provider",
	})
	require.NoError(t, err)

	key := dpoptest.Key(t)
	sess.DPoPJKT = dpoptest.Thumbprint(t, key)
	require.NoError(t, eng.Store().UpdateSession(ctx, sess))

	provider, err := eng.AuthRegistry().Get("session")
	require.NoError(t, err)

	newRequest := func(proof string) *http.Request {
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/test", nil)
		req.Header.Set("Authorization", "DPoP "+sess.Token)
		if proof != "" {
			req.Header.Set("DPoP", proof)
		}
		return req
	}

	t.Run("without a proof", func(t *testing.T) {
		_, authErr := provider.Authenticate(ctx, newRequest(""))
		require.Error(t, authErr, "a bound session with no proof must not authenticate")
	})

	t.Run("with a valid proof", func(t *testing.T) {
		claims := dpoptest.ValidClaims(http.MethodGet, "http://example.com/test")
		claims["ath"] = dpop.AccessTokenHash(sess.Token)

		authCtx, authErr := provider.Authenticate(ctx, newRequest(dpoptest.MintProof(t, key, "ES256", claims)))

		require.NoError(t, authErr, "the client holding the key must still authenticate")
		assert.NotNil(t, authCtx)
	})
}
