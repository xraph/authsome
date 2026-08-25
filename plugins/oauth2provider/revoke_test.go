package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"

	"golang.org/x/crypto/bcrypt"
)

// newRevokeFixture mirrors newFixture but hands back the core store too, which
// a revoke test needs in order to see whether a session actually died. It is
// spelled out here rather than widening newFixture, whose signature is used by
// every other test file in this package.
func newRevokeFixture(t *testing.T) (store.Store, forge.Router) {
	t.Helper()
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	core := memory.New()
	p.SetOAuth2Store(st)
	p.SetStore(core)

	hashed, err := bcrypt.GenerateFromPassword([]byte(confidentialSecret), bcrypt.MinCost)
	require.NoError(t, err)

	appID := id.NewAppID()
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     confidentialID,
		ClientSecret: string(hashed),
		Name:         "Confidential",
		RedirectURIs: []string{registeredURI, otherURI},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code"},
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return core, mux
}

// mintSession runs the authorization code flow and returns the access token.
func mintSession(t *testing.T, mux forge.Router) string {
	t.Helper()
	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))
	rec := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	return resp.AccessToken
}

// postRevoke posts to the revocation endpoint. caller, when non-nil, stands in
// for the session the auth middleware would have resolved from a bearer token.
func postRevoke(t *testing.T, mux forge.Router, caller *session.Session, authHeader string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	ctx := context.Background()
	if caller != nil {
		ctx = middleware.WithSession(ctx, caller)
	}
	req := httptest.NewRequestWithContext(ctx, "POST", "/v1/oauth/revoke", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// tokenStillValid reports whether the session behind a token survived.
func tokenStillValid(t *testing.T, core store.Store, token string) bool {
	t.Helper()
	_, err := core.GetSessionByToken(context.Background(), token)
	return err == nil
}

// ── the anonymous hole is closed (RFC 7009 §2.1) ────────────────────────

// The endpoint used to revoke whatever token it was handed, from anybody. A
// caller that proves nothing now gets turned away and the token survives.
func TestRevoke_RejectsAnonymousCaller(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	rec := postRevoke(t, mux, nil, "", map[string]string{"token": token})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.True(t, tokenStillValid(t, core, token), "an anonymous request must not revoke anything")
}

// ── the client credential path ──────────────────────────────────────────

func TestRevoke_AcceptsConfidentialClientWithSecret(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	rec := postRevoke(t, mux, nil, "", map[string]string{
		"token":         token,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.False(t, tokenStillValid(t, core, token), "the token should be gone")
}

func TestRevoke_RejectsConfidentialClientWithWrongSecret(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	rec := postRevoke(t, mux, nil, "", map[string]string{
		"token":         token,
		"client_id":     confidentialID,
		"client_secret": "not-the-secret",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.True(t, tokenStillValid(t, core, token))
}

// A client_id on its own is not authentication, so a confidential client that
// sends only its id is refused the same as an anonymous caller.
func TestRevoke_RejectsConfidentialClientIDWithoutSecret(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	rec := postRevoke(t, mux, nil, "", map[string]string{
		"token":     token,
		"client_id": confidentialID,
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.True(t, tokenStillValid(t, core, token))
}

// client_secret_basic works here for the same reason it works on /token.
func TestRevoke_AcceptsBasicAuth(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	rec := postRevoke(t, mux, nil, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"token": token,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.False(t, tokenStillValid(t, core, token))
}

// ── the bearer path, which is what the SDKs use ─────────────────────────

// The generated SDKs send a bearer session and no client credentials. Revoking
// your own token has to keep working or every SDK caller breaks.
func TestRevoke_AllowsSessionOwnerToRevokeOwnToken(t *testing.T) {
	core, mux := newRevokeFixture(t)
	token := mintSession(t, mux)

	caller, err := core.GetSessionByToken(context.Background(), token)
	require.NoError(t, err)

	rec := postRevoke(t, mux, caller, "Bearer "+token, map[string]string{"token": token})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.False(t, tokenStillValid(t, core, token))
}

// A signed-in user must not be able to revoke somebody else's token by holding
// a copy of it. The answer is the same 200 an unknown token gets, so the
// endpoint stays silent about tokens the caller does not control.
func TestRevoke_SilentlyIgnoresAnotherUsersToken(t *testing.T) {
	core, mux := newRevokeFixture(t)
	victimToken := mintSession(t, mux)

	stranger := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
	}

	rec := postRevoke(t, mux, stranger, "Bearer whatever", map[string]string{"token": victimToken})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.True(t, tokenStillValid(t, core, victimToken), "another user's token must survive")
}

// ── RFC 7009 §2.2: an unknown token is still a 200 ──────────────────────

func TestRevoke_UnknownTokenStillReturnsOK(t *testing.T) {
	_, mux := newRevokeFixture(t)

	rec := postRevoke(t, mux, nil, "", map[string]string{
		"token":         "no-such-token",
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	})

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}
