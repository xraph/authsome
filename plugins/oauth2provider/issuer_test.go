package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/settings"
)

const (
	// platformSlug is the slug secutil gives the fixture's app.
	platformSlug = "platform"
	otherSlug    = "other-app"
)

// setAppIssuer turns the per-app issuer on or off for appID the way the
// dashboard would.
func setAppIssuer(t *testing.T, f *oidcFixture, appID id.AppID, on bool) {
	t.Helper()
	raw, err := json.Marshal(on)
	require.NoError(t, err)
	require.NoError(t, f.eng.Settings().Set(context.Background(), "oauth2.app_issuer", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// addOtherApp creates a second app that never opts in.
func addOtherApp(t *testing.T, f *oidcFixture) {
	t.Helper()
	now := time.Now()
	appID := id.NewAppID()
	require.NoError(t, f.eng.Store().CreateApp(context.Background(), &app.App{
		ID: appID, Name: "Other", Slug: otherSlug, PublishableKey: "pk_test_other_app",
		CreatedAt: now, UpdatedAt: now,
	}))
}

const platformIssuer = oidcIssuer + "/apps/" + platformSlug

// verifyAppIDToken checks raw against the app's own JWKS and returns its claims.
func verifyAppIDToken(t *testing.T, f *oidcFixture, slug, raw string) (map[string]any, jwt.MapClaims) {
	t.Helper()
	rec := getPath(t, f.mux, "/apps/"+slug+"/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc struct {
		Keys []json.RawMessage `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(tok *jwt.Token) (any, error) {
		for _, k := range doc.Keys {
			pub, j, err := jwkutil.ParseJSON(k)
			if err != nil {
				return nil, err
			}
			if j.KID == tok.Header["kid"] {
				return pub, nil
			}
		}
		return nil, assert.AnError
	}, jwt.WithValidMethods([]string{"RS256"}))
	require.NoError(t, err)
	require.True(t, tok.Valid)
	return tok.Header, claims
}

func idTokenOf(t *testing.T, f *oidcFixture) string {
	t.Helper()
	body := decodeTokenBody(t, f.codeGrant(t, publicID, "openid", "n-1", time.Now()))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the response carries an id_token: %v", body)
	return raw
}

func TestIssuerFor_DefaultsToDeployment(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	_, claims := verifyIDToken(t, f.mux, idTokenOf(t, f))
	assert.Equal(t, oidcIssuer, claims["iss"])

	rec := getPath(t, f.mux, "/apps/"+platformSlug+"/v1/oauth/jwks")
	assert.Equal(t, http.StatusNotFound, rec.Code, "the app did not opt in")
}

func TestIssuerFor_AppOptIn(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	setAppIssuer(t, f, f.appID, true)

	header, claims := verifyAppIDToken(t, f, platformSlug, idTokenOf(t, f))
	assert.Equal(t, platformIssuer, claims["iss"])

	// The key is the app's own, not the deployment's.
	deployment := kids(fetchJWKS(t, f.mux))
	assert.NotContains(t, deployment, header["kid"])
}

func TestIDToken_IssuerChosenAtTokenTime(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)

	// Authorize while the app is on the deployment issuer...
	q := baseAuthorizeQuery(publicID)
	q.Set("scope", "openid")
	q.Set("code_challenge", s256(oidcVerifier))
	q.Set("code_challenge_method", "S256")
	code := codeFrom(t, authorizeWith(sessionCtx(f.user.ID, f.appID, time.Now()), f.mux, q))

	// ...opt in before the code is redeemed.
	setAppIssuer(t, f, f.appID, true)
	body := decodeTokenBody(t, postToken(t, f.mux, map[string]string{
		"grant_type": "authorization_code", "code": code, "redirect_uri": registeredURI,
		"client_id": publicID, "code_verifier": oidcVerifier,
	}))
	_, claims := verifyAppIDToken(t, f, platformSlug, body["id_token"].(string))
	assert.Equal(t, platformIssuer, claims["iss"])

	// And back off again.
	setAppIssuer(t, f, f.appID, false)
	_, claims = verifyIDToken(t, f.mux, idTokenOf(t, f))
	assert.Equal(t, oidcIssuer, claims["iss"])
}

func TestAppDiscovery_UnknownSlug404(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	rec := getPath(t, f.mux, "/apps/no-such-app/.well-known/openid-configuration")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAppDiscovery_NotOptedIn404(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	addOtherApp(t, f)

	root := forge.NewRouter()
	require.NoError(t, f.p.RegisterRootRoutes(root))

	cases := []struct {
		name string
		h    http.Handler
		path string
	}{
		{"prefixed oidc", f.mux, "/.well-known/openid-configuration"},
		{"prefixed jwks", f.mux, "/v1/oauth/jwks"},
		{"root oidc", root, "/.well-known/openid-configuration"},
		{"root rfc 8414", root, "/.well-known/oauth-authorization-server"},
	}
	for _, c := range cases {
		unknown := getPath(t, c.h, "/apps/no-such-app"+c.path)
		other := getPath(t, c.h, "/apps/"+otherSlug+c.path)
		assert.Equal(t, http.StatusNotFound, unknown.Code, c.name)
		assert.Equal(t, http.StatusNotFound, other.Code, c.name)
		assert.NotEmpty(t, unknown.Body.String(), c.name)
		assert.Equal(t, unknown.Body.String(), other.Body.String(),
			"%s: an unknown slug and an app that did not opt in must look the same", c.name)
	}
}

func TestAppDiscovery_PointsAtSharedEndpoints(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	setAppIssuer(t, f, f.appID, true)
	root := forge.NewRouter()
	require.NoError(t, f.p.RegisterRootRoutes(root))

	for name, path := range map[string]string{
		"oidc":     "/apps/" + platformSlug + "/.well-known/openid-configuration",
		"rfc 8414": "/apps/" + platformSlug + "/.well-known/oauth-authorization-server",
	} {
		code, got := getJSON(t, root, path)
		require.Equal(t, http.StatusOK, code, name)
		assert.Equal(t, platformIssuer, got["issuer"], name)
		assert.Equal(t, platformIssuer+"/v1/oauth/jwks", got["jwks_uri"], name)
		assert.Equal(t, oidcIssuer+"/v1/oauth/authorize", got["authorization_endpoint"], name)
		assert.Equal(t, oidcIssuer+"/v1/oauth/token", got["token_endpoint"], name)
		assert.Equal(t, oidcIssuer+"/v1/oauth/userinfo", got["userinfo_endpoint"], name)
	}

	// The OIDC document is mirrored under the mount prefix.
	code, got := getJSON(t, f.mux, "/apps/"+platformSlug+"/.well-known/openid-configuration")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, platformIssuer, got["issuer"])
}

func TestAppJWKS_UsesAppKeys(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	setAppIssuer(t, f, f.appID, true)

	deployment := kids(fetchJWKS(t, f.mux))
	rec := getPath(t, f.mux, "/apps/"+platformSlug+"/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "public, max-age=300", rec.Header().Get("Cache-Control"))
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.Len(t, doc.Keys, 1)
	assert.NotContains(t, deployment, doc.Keys[0]["kid"], "the app has its own key")
	assert.Equal(t, "RS256", doc.Keys[0]["alg"])
}

func TestAppJWKS_IncludesOnlyThisAppsAccessTokenKey(t *testing.T) {
	appKey := rsaJWTFormat(t, "app-at-key")
	otherKey := rsaJWTFormat(t, "other-at-key")
	f := newOIDCFixture(t, oauth2provider.Config{}, true,
		authsome.WithJWTFormat("aapp_01jf0000000000000000000000", appKey),
		authsome.WithJWTFormat("aapp_other", otherKey))
	setAppIssuer(t, f, f.appID, true)

	rec := getPath(t, f.mux, "/apps/"+platformSlug+"/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	got := kids(doc)
	assert.Contains(t, got, "app-at-key")
	assert.NotContains(t, got, "other-at-key")
}

func TestAppDiscovery_UnknownSlugCreatesNoKey(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	addOtherApp(t, f)

	for _, path := range []string{
		"/apps/no-such-app/.well-known/openid-configuration",
		"/apps/no-such-app/v1/oauth/jwks",
		"/apps/" + otherSlug + "/.well-known/openid-configuration",
		"/apps/" + otherSlug + "/v1/oauth/jwks",
	} {
		assert.Equal(t, http.StatusNotFound, getPath(t, f.mux, path).Code, path)
	}

	ctx := context.Background()
	for _, scope := range []string{"no-such-app", otherSlug, f.appID.String()} {
		keys, err := f.st.ListSigningKeys(ctx, scope)
		require.NoError(t, err)
		assert.Empty(t, keys, "scope %q must have no key", scope)
	}
	// Nor did anything land under any scope the app ids would use.
	apps, err := f.eng.Store().ListApps(ctx)
	require.NoError(t, err)
	for _, a := range apps {
		keys, err := f.st.ListSigningKeys(ctx, a.ID.String())
		require.NoError(t, err)
		assert.Empty(t, keys, "app %s must have no key", a.Slug)
	}
}
