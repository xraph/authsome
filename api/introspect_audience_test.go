package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

const (
	audResAPI   = "https://api.example.com"
	audResFiles = "https://files.example.com"
)

// newIntrospectAudienceEngine builds an engine like newTestAPI's, but lets
// the caller optionally wire in a JWT token format. newTestAPI always leaves
// the engine on opaque tokens, which is no good for the JWT half of the
// audience test: a.engine.ValidateJWT only succeeds against a format it
// actually knows about.
func newIntrospectAudienceEngine(t *testing.T, jwtFmt *tokenformat.JWT) *authsome.Engine {
	t.Helper()
	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	opts := []authsome.Option{
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
	}
	if jwtFmt != nil {
		opts = append(opts, authsome.WithDefaultTokenFormat(jwtFmt))
	}

	eng, err := authsome.NewEngine(opts...)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)
	return eng
}

func newHMACJWTFormat(t *testing.T) *tokenformat.JWT {
	t.Helper()
	j, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("introspect-audience-test-key-000"),
		Issuer:        "https://auth.example.com",
	})
	require.NoError(t, err)
	return j
}

func postIntrospect(t *testing.T, router http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"token": token})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// signUpForAudience creates a fresh user/session on eng, independent of the
// signUp helper in api_test.go: that helper hands back only strings, but
// these tests need the *session.Session itself so they can mutate Audience
// and persist it back through the store.
func signUpForAudience(t *testing.T, eng *authsome.Engine, email string) string {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss1",
		FirstName: "Aud",
	})
	require.NoError(t, err)
	return sess.Token
}

// TestIntrospectAudience is the discrimination test for Task 12: it must
// fail if either introspection branch stops copying the audience across,
// and it must fail if the omitempty guard on IntrospectResponse.Audience
// ever regresses into emitting "aud": null or "aud": [].
func TestIntrospectAudience(t *testing.T) {
	t.Run("a JWT carrying aud introspects with the same audience array", func(t *testing.T) {
		jwtFmt := newHMACJWTFormat(t)
		eng := newIntrospectAudienceEngine(t, jwtFmt)
		router := newAPIWithRouter(t, eng)

		token, err := jwtFmt.GenerateAccessToken(tokenformat.TokenClaims{
			UserID:    id.NewUserID().String(),
			AppID:     testAppIDStr,
			SessionID: id.NewSessionID().String(),
			Audience:  []string{audResAPI, audResFiles},
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		rec := postIntrospect(t, router, token)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp api.IntrospectResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Active)
		assert.Equal(t, []string{audResAPI, audResFiles}, resp.Audience)
	})

	t.Run("an opaque session carrying Audience introspects with the same array", func(t *testing.T) {
		eng := newIntrospectAudienceEngine(t, nil)
		router := newAPIWithRouter(t, eng)

		token := signUpForAudience(t, eng, "aud-opaque@example.com")

		sess, err := eng.Store().GetSessionByToken(context.Background(), token)
		require.NoError(t, err)
		sess.Audience = []string{audResAPI, audResFiles}
		require.NoError(t, eng.Store().UpdateSession(context.Background(), sess))

		rec := postIntrospect(t, router, token)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp api.IntrospectResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Active)
		assert.Equal(t, []string{audResAPI, audResFiles}, resp.Audience)
	})

	t.Run("a token with no audience omits aud entirely", func(t *testing.T) {
		eng := newIntrospectAudienceEngine(t, nil)
		router := newAPIWithRouter(t, eng)

		token := signUpForAudience(t, eng, "aud-none@example.com")

		rec := postIntrospect(t, router, token)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		// A struct unmarshal can't distinguish an absent key from a present
		// key holding a zero value, so this has to be checked against the
		// raw response, not a decoded IntrospectResponse. The backwards-
		// compatibility rule is that a token with no audience introspects
		// exactly as it did before this field existed: no "aud" key at all,
		// not "aud": null and not "aud": [].
		assert.NotContains(t, rec.Body.String(), `"aud"`,
			"aud must be absent from the response body, not null or an empty array")

		var raw map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
		_, present := raw["aud"]
		assert.False(t, present, "aud key must not be present in the decoded response")
	})
}
