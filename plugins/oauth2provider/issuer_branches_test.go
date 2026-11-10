package oauth2provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"
)

// noSettingsEngine is an engine that has no settings manager to ask.
type noSettingsEngine struct{ plugin.Engine }

func (noSettingsEngine) Settings() *settings.Manager { return nil }

func TestIssuerFor_NoSettingsManagerIsDeployment(t *testing.T) {
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	p.engine = noSettingsEngine{eng}

	issuer, scope, err := p.issuerFor(ctx, id.NewAppID())
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com", issuer)
	assert.Empty(t, scope)
}

// issuerPlugin boots a plugin whose per-app issuer setting is declared, with
// one app, acme, opted in.
func issuerPlugin(t *testing.T) (*Plugin, *authsome.Engine, *app.App) {
	t.Helper()
	ctx := context.Background()
	eng := secutil.NewTestEngine(t, authsome.WithTokenEncryptor(testEncryptor(t)))
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	require.NoError(t, p.DeclareSettings(eng.Settings()))
	a := &app.App{ID: id.NewAppID(), Name: "Acme", Slug: "acme", PublishableKey: "pk_test_acme_issuer",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, eng.Store().CreateApp(ctx, a))
	setIssuerSetting(t, eng, a.ID, json.RawMessage(`true`))
	return p, eng, a
}

func setIssuerSetting(t *testing.T, eng *authsome.Engine, appID id.AppID, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, eng.Settings().Set(context.Background(), SettingAppIssuer.Def.Key, raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// A stored value that isn't a boolean can't say which issuer applies, so it
// is an error rather than a guess, everywhere the issuer is chosen.
func TestIssuerFor_UnreadableSettingIsAnError(t *testing.T) {
	ctx := context.Background()
	p, eng, a := issuerPlugin(t)
	setIssuerSetting(t, eng, a.ID, json.RawMessage(`"yes"`))

	issuer, _, err := p.issuerFor(ctx, a.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), SettingAppIssuer.Def.Key)
	assert.Empty(t, issuer)

	_, _, ok, err := p.appIssuerBySlug(ctx, "acme")
	require.Error(t, err)
	assert.False(t, ok)

	_, err = p.idTokenKeyFor(ctx, &OAuth2Client{AppID: a.ID, ClientID: "c"})
	require.Error(t, err)
}

// slugFailStore fails every app lookup by slug, as a database outage would.
type slugFailStore struct{ store.Store }

func (slugFailStore) GetAppBySlug(context.Context, string) (*app.App, error) {
	return nil, errStoreDown
}

// The per-app endpoints are unauthenticated. A failed lookup answers 500
// without the cause, and never the 404 an unknown slug gets.
func TestAppEndpoints_LookupFailureIs500(t *testing.T) {
	p, eng, _ := issuerPlugin(t)
	p.store = slugFailStore{eng.Store()}
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	root := forge.NewRouter()
	require.NoError(t, p.RegisterRootRoutes(root))

	for path, h := range map[string]http.Handler{
		"/apps/acme/.well-known/openid-configuration":       mux,
		"/apps/acme/v1/oauth/jwks":                          mux,
		"/apps/acme/.well-known/oauth-authorization-server": root,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		assert.Equal(t, http.StatusInternalServerError, rec.Code, path)
		assert.NotContains(t, rec.Body.String(), errStoreDown.Error(), path)
	}
}

func TestAppIssuerBySlug_LookupFailureIsAnError(t *testing.T) {
	p, eng, _ := issuerPlugin(t)
	p.store = slugFailStore{eng.Store()}
	_, _, ok, err := p.appIssuerBySlug(context.Background(), "acme")
	require.ErrorIs(t, err, errStoreDown)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), `look up app "acme"`)
}

// Discovery still advertises an alg when the keys can't be described, since
// an empty list makes OIDC clients refuse the document; the failure is
// logged.
func TestIDTokenAlgs_DescribeFailureFallsBack(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(st)
	require.NoError(t, p.initSigningKeys(context.Background(), testEncryptor(t)))
	tl := log.NewTestLogger()
	p.logger = tl

	assert.Equal(t, []string{"RS256"}, p.idTokenAlgs(context.Background(), ""))
	warns := tl.(*log.TestLogger).GetLogsByLevel("WARN")
	require.Len(t, warns, 1)
	assert.Contains(t, warns[0].Message, "describe signing keys for discovery")
}

func jwtFormat(t *testing.T, method jwt.SigningMethod, priv, pub any, kid string) *tokenformat.JWT {
	t.Helper()
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{SigningMethod: method, SigningKey: priv, VerifyKey: pub, KeyID: kid})
	require.NoError(t, err)
	return f
}

// The deployment JWKS carries the engine's default access token key when it
// is a JWT key, and skips (with a warning) one it cannot encode instead of
// failing the whole document.
func TestAccessTokenJWKs_DefaultFormatAndUnencodableKey(t *testing.T) {
	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	p224, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	require.NoError(t, err)

	eng := secutil.NewTestEngine(t,
		authsome.WithDefaultTokenFormat(jwtFormat(t, jwt.SigningMethodRS256, rk, &rk.PublicKey, "default-key")),
		authsome.WithJWTFormat("app-odd", jwtFormat(t, jwt.SigningMethodES256, p224, &p224.PublicKey, "odd-curve")),
	)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(context.Background(), eng))
	tl := log.NewTestLogger()
	p.logger = tl

	got := p.accessTokenJWKs("")
	kids := make([]string, 0, len(got))
	for _, j := range got {
		kids = append(kids, j.KID)
	}
	assert.Equal(t, []string{"default-key"}, kids)
	warns := tl.(*log.TestLogger).GetLogsByLevel("WARN")
	require.Len(t, warns, 1)
	assert.True(t, strings.Contains(warns[0].Message, "unencodable access token key"), warns[0].Message)

	// The default key belongs to the deployment issuer only.
	assert.Empty(t, p.accessTokenJWKs("app-other"))
}
