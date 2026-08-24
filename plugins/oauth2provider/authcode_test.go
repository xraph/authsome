package oauth2provider_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"

	"golang.org/x/crypto/bcrypt"
)

const (
	confidentialID     = "conf-client"
	confidentialSecret = "conf-secret-value"
	publicID           = "pub-client"
	registeredURI      = "https://app.example.com/cb"
	otherURI           = "https://app.example.com/other"
)

// newFixture builds a plugin backed by the in-memory store with one
// confidential and one public client registered.
func newFixture(t *testing.T) (*oauth2provider.Plugin, oauth2provider.Store, forge.Router) {
	t.Helper()
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	p.SetStore(memory.New())

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
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		ClientID:                publicID,
		Name:                    "Public",
		RedirectURIs:            []string{registeredURI},
		Scopes:                  []string{"openid", "profile"},
		GrantTypes:              []string{"authorization_code"},
		Public:                  true,
		TokenEndpointAuthMethod: "none",
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return p, st, mux
}

// authorize drives GET /v1/oauth/authorize as a signed-in user.
func authorize(t *testing.T, mux forge.Router, q url.Values) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(context.Background(), id.NewUserID())
	req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func baseAuthorizeQuery(clientID string) url.Values {
	return url.Values{
		"response_type": {"code"},
		"client_id":     {clientID},
		"redirect_uri":  {registeredURI},
	}
}

// codeFrom extracts the authorization code from a 302 Location header.
func codeFrom(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusFound, rec.Code, "body: %s", rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

func postToken(t *testing.T, mux forge.Router, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func s256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ── redirect_uri binding (RFC 6749 §4.1.3) ────────────

// The code is bound to the redirect_uri it was issued against. Without the
// check at token exchange, a code obtained via one registered URI can be
// redeemed while claiming another, breaking the tie between the two legs.
func TestTokenExchange_RejectsMismatchedRedirectURI(t *testing.T) {
	_, _, mux := newFixture(t)

	q := baseAuthorizeQuery(confidentialID)
	code := codeFrom(t, authorize(t, mux, q))

	rec := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  otherURI, // registered, but not the one used above
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "redirect_uri")
	assert.NotContains(t, rec.Body.String(), "access_token")
}

// A code minted for one client must not be redeemable by another.
func TestTokenExchange_RejectsMismatchedClientID(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	rec := postToken(t, mux, map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"client_id":    publicID,
		"redirect_uri": registeredURI,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

// ── PKCE (RFC 7636 / RFC 8252 §8.1) ───────────────────

// A public client holds no secret, so an intercepted code is redeemable by
// anyone. Enforcing PKCE only when a challenge happens to be present lets the
// client opt out of its own protection by omitting it.
func TestAuthorize_PublicClientMustSendCodeChallenge(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := authorize(t, mux, baseAuthorizeQuery(publicID))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "code_challenge")
}

// "plain" leaves the verifier recoverable wherever the challenge is observable
// — the authorize URL reaches browser history, logs and Referer headers.
func TestAuthorize_PublicClientMustUseS256(t *testing.T) {
	_, _, mux := newFixture(t)

	q := baseAuthorizeQuery(publicID)
	q.Set("code_challenge", "some-verifier")
	q.Set("code_challenge_method", "plain")

	rec := authorize(t, mux, q)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "S256")
}

func TestPKCE_HappyPathAndWrongVerifier(t *testing.T) {
	_, _, mux := newFixture(t)

	const verifier = "a-sufficiently-long-random-code-verifier-value"
	q := baseAuthorizeQuery(publicID)
	q.Set("code_challenge", s256(verifier))
	q.Set("code_challenge_method", "S256")

	code := codeFrom(t, authorize(t, mux, q))

	bad := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     publicID,
		"redirect_uri":  registeredURI,
		"code_verifier": "not-the-verifier",
	})
	assert.Equal(t, http.StatusBadRequest, bad.Code)
	assert.NotContains(t, bad.Body.String(), "access_token")

	// A wrong verifier must not have burned the code.
	missing := postToken(t, mux, map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"client_id":    publicID,
		"redirect_uri": registeredURI,
	})
	assert.Equal(t, http.StatusBadRequest, missing.Code)
	assert.Contains(t, missing.Body.String(), "code_verifier")
}

// ── scope and grant confinement ───────────────────────

// The code must carry only scopes the client is registered for, or a caller
// can name any scope it likes and have it honoured at exchange.
func TestAuthorize_RejectsUnregisteredScope(t *testing.T) {
	_, _, mux := newFixture(t)

	q := baseAuthorizeQuery(confidentialID)
	q.Set("scope", "openid admin:all")

	rec := authorize(t, mux, q)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "admin:all")
}

func TestAuthorize_OmittedScopeYieldsRegisteredSet(t *testing.T) {
	_, st, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	stored, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile"}, stored.Scopes)
}

// GrantTypes was recorded at registration but never enforced, so any client
// could run any flow.
func TestAuthorize_RejectsUnregisteredGrantType(t *testing.T) {
	_, st, mux := newFixture(t)

	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        id.NewAppID(),
		ClientID:     "cc-only",
		Name:         "Client credentials only",
		RedirectURIs: []string{registeredURI},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"client_credentials"},
	}))

	rec := authorize(t, mux, baseAuthorizeQuery("cc-only"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "authorization_code")
}

// ── single use ────────────────────────────────────────

// Checking Consumed and writing it are two statements; only an atomic
// compare-and-set in the store stops two concurrent exchanges from both
// succeeding against one code.
func TestTokenExchange_CodeIsSingleUseUnderConcurrency(t *testing.T) {
	_, st, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	const racers = 8
	var wg sync.WaitGroup
	results := make([]int, racers)
	start := make(chan struct{})

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = postToken(t, mux, map[string]string{
				"grant_type":    "authorization_code",
				"code":          code,
				"client_id":     confidentialID,
				"client_secret": confidentialSecret,
				"redirect_uri":  registeredURI,
			}).Code
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for _, c := range results {
		if c == http.StatusOK {
			succeeded++
		}
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, c,
			"unexpected status %d in %v", c, results)
	}
	assert.Equal(t, 1, succeeded,
		"exactly one concurrent exchange may succeed, got %d (%v)", succeeded, results)

	stored, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.True(t, stored.Consumed)
}

// The store primitive itself: only the first call reports success.
func TestConsumeAuthCode_IsCompareAndSet(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	require.NoError(t, st.CreateAuthCode(ctx, &oauth2provider.AuthorizationCode{
		ID:        id.NewAuthCodeID(),
		Code:      "abc",
		ClientID:  confidentialID,
		ExpiresAt: time.Now().Add(time.Minute),
	}))

	first, err := st.ConsumeAuthCode(ctx, "abc")
	require.NoError(t, err)
	assert.True(t, first, "first consume should win")

	second, err := st.ConsumeAuthCode(ctx, "abc")
	require.NoError(t, err)
	assert.False(t, second, "second consume must report the code as already used")
}

// ── redirect construction ─────────────────────────────

// Concatenating "?code=" corrupts a registered URI that already carries a
// query, and an unescaped state can smuggle extra parameters into the
// redirect the client parses.
func TestAuthorize_RedirectPreservesQueryAndEscapesState(t *testing.T) {
	_, st, mux := newFixture(t)

	const withQuery = "https://app.example.com/cb?tenant=acme"
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        id.NewAppID(),
		ClientID:     "queried",
		Name:         "Has query",
		RedirectURIs: []string{withQuery},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
	}))

	q := url.Values{
		"response_type": {"code"},
		"client_id":     {"queried"},
		"redirect_uri":  {withQuery},
		"state":         {"a&b=c d"},
	}
	rec := authorize(t, mux, q)
	require.Equal(t, http.StatusFound, rec.Code, "body: %s", rec.Body.String())

	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	got := loc.Query()

	assert.Equal(t, "acme", got.Get("tenant"), "existing query param must survive")
	assert.NotEmpty(t, got.Get("code"))
	assert.Equal(t, "a&b=c d", got.Get("state"), "state must round-trip, not split into params")
	assert.Empty(t, got.Get("b"), "state must not inject an extra parameter")
}

// Omitting redirect_uri is legal only with exactly one registered, and must
// resolve to it — echoing back "" produced a relative redirect that never
// reached the client.
func TestAuthorize_OmittedRedirectURI(t *testing.T) {
	_, _, mux := newFixture(t)

	// Public client has exactly one registered URI.
	verifier := "another-sufficiently-long-code-verifier-value"
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {publicID},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
	}
	rec := authorize(t, mux, q)
	require.Equal(t, http.StatusFound, rec.Code, "body: %s", rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", loc.Host)
	assert.Equal(t, "/cb", loc.Path)

	// Confidential client has two, so the caller must say which.
	ambiguous := authorize(t, mux, url.Values{
		"response_type": {"code"},
		"client_id":     {confidentialID},
	})
	assert.Equal(t, http.StatusBadRequest, ambiguous.Code)
}

// losingStore simulates the outcome of losing a consume race: the code looked
// unconsumed on read, but the atomic compare-and-set reports that another
// caller got there first.
type losingStore struct {
	oauth2provider.Store
}

func (s *losingStore) ConsumeAuthCode(context.Context, string) (bool, error) {
	return false, nil
}

// The concurrency test above cannot isolate this: the earlier Consumed read
// usually rejects the losers before they reach the compare-and-set, so the
// suite would still pass with the store's answer ignored. Driving the losing
// outcome directly pins the guard that turns it into a refusal.
func TestTokenExchange_RefusesWhenConsumeReportsReplay(t *testing.T) {
	p, st, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))
	p.SetOAuth2Store(&losingStore{Store: st})

	rec := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"a lost compare-and-set must be treated as a replay")
	assert.NotContains(t, rec.Body.String(), "access_token")
}
