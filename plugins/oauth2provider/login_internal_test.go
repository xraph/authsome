package oauth2provider

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
)

func TestOnInit_RefusesBadLoginURL(t *testing.T) {
	p := New(Config{Issuer: "https://auth.example.com", LoginURL: "sign-in"})
	p.SetOAuth2Store(NewMemoryStore())
	err := p.OnInit(context.Background(), secutil.NewTestEngine(t))
	assert.ErrorContains(t, err, "login URL")
}

func TestDeclareSettings_ReportsAClash(t *testing.T) {
	for _, taken := range []string{SettingAppIssuer.Def.Key, SettingAppLoginURL.Def.Key} {
		eng := secutil.NewTestEngine(t)
		require.NoError(t, settings.RegisterTyped(eng.Settings(), settingsNamespace,
			settings.Define(taken, "", settings.WithScopes(settings.ScopeApp))))
		p := New()
		assert.ErrorIs(t, p.DeclareSettings(eng.Settings()), settings.ErrKeyAlreadyRegistered, "key %s", taken)
	}
}

func TestLoginURLFor_UndeclaredSettingUsesConfig(t *testing.T) {
	ctx := context.Background()
	p := New(Config{Issuer: "https://auth.example.com", LoginURL: "/sign-in"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, secutil.NewTestEngine(t)))

	u, err := p.loginURLFor(ctx, id.NewAppID())
	require.NoError(t, err)
	assert.Equal(t, "/sign-in", u.String())
}

func TestLoginURLFor_InvalidStoredValueFallsBack(t *testing.T) {
	// A value saved before the setting validated: declare the key without
	// validation, store a bad value, and check the deployment page is used.
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com", LoginURL: "/sign-in"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	tl := log.NewTestLogger()
	p.logger = tl

	require.NoError(t, settings.RegisterTyped(eng.Settings(), settingsNamespace,
		settings.Define(SettingAppLoginURL.Def.Key, "", settings.WithScopes(settings.ScopeApp))))
	appID := id.NewAppID()
	raw, err := json.Marshal("javascript:alert(1)")
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, SettingAppLoginURL.Def.Key, raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	u, err := p.loginURLFor(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "/sign-in", u.String())
	assert.Len(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"), 1)
}

func TestLoginURLFor_BadConfigIsAnError(t *testing.T) {
	// OnInit refuses this, so it only happens to a plugin that skipped it.
	p := New(Config{LoginURL: "sign-in"})
	_, err := p.loginURLFor(context.Background(), id.NewAppID())
	assert.Error(t, err)
}

func TestLoginRedirect_UnparsableIssuerStaysRelative(t *testing.T) {
	p := New(Config{Issuer: "://not a url"})
	login, err := ValidateLoginURL("https://login.example.com/in")
	require.NoError(t, err)
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/oauth/authorize", nil)
	assert.Equal(t, "https://login.example.com/in?redirect=%2Fv1%2Foauth%2Fauthorize", p.loginRedirect(login, r))
}

func TestErrorRedirect(t *testing.T) {
	_, err := errorRedirect("://bad", errLoginRequired, "", "")
	assert.Error(t, err)

	to, err := errorRedirect("https://app.example.com/cb?x=1", errLoginRequired, "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/cb?error=login_required&x=1", to)
}

func TestAppLoginURL_NoSettingsManager(t *testing.T) {
	p := New()
	p.engine = &authsome.Engine{}
	assert.Empty(t, p.appLoginURL(context.Background(), id.NewAppID()))
}

func TestAppLoginURL_ReadErrorCountsAsUnset(t *testing.T) {
	// The key holds a number, so reading it as a string fails with
	// something other than ErrUnknownKey. With or without a logger, the
	// read counts as unset.
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))

	require.NoError(t, settings.RegisterTyped(eng.Settings(), settingsNamespace,
		settings.Define(SettingAppLoginURL.Def.Key, 0, settings.WithScopes(settings.ScopeApp))))
	appID := id.NewAppID()
	require.NoError(t, eng.Settings().Set(ctx, SettingAppLoginURL.Def.Key, json.RawMessage("5"),
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	tl := log.NewTestLogger()
	p.logger = tl
	assert.Empty(t, p.appLoginURL(ctx, appID))
	assert.Len(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"), 1)

	p.logger = nil
	assert.Empty(t, p.appLoginURL(ctx, appID))
}

func TestLoginURLFor_InvalidStoredValueWithoutLogger(t *testing.T) {
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	p.logger = nil

	require.NoError(t, settings.RegisterTyped(eng.Settings(), settingsNamespace,
		settings.Define(SettingAppLoginURL.Def.Key, "", settings.WithScopes(settings.ScopeApp))))
	appID := id.NewAppID()
	require.NoError(t, eng.Settings().Set(ctx, SettingAppLoginURL.Def.Key, json.RawMessage(`"sign-in"`),
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	u, err := p.loginURLFor(ctx, appID)
	require.NoError(t, err)
	assert.Nil(t, u)
}
