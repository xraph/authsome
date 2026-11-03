package oauth2provider

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
)

func TestIssuerFor_NoEngineFallsBackToDeployment(t *testing.T) {
	p := New(Config{Issuer: "https://auth.example.com"})
	appID := id.NewAppID()
	issuer, scope, err := p.issuerFor(context.Background(), appID)
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com", issuer)
	assert.Empty(t, scope)
}

func TestIssuerFor_UnregisteredSettingMeansOff(t *testing.T) {
	// OnInit without the engine registering the plugin: the setting was never
	// declared, so nobody can have turned it on.
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))

	issuer, scope, err := p.issuerFor(ctx, id.NewAppID())
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com", issuer)
	assert.Empty(t, scope)
}

func TestIssuerFor_OptedInAppWithoutRowIsAnError(t *testing.T) {
	// A failed app lookup must not turn into the deployment issuer.
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	require.NoError(t, p.DeclareSettings(eng.Settings()))

	ghost := id.NewAppID()
	raw, err := json.Marshal(true)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, "oauth2.app_issuer", raw,
		settings.ScopeApp, ghost.String(), ghost.String(), "", "test"))

	issuer, _, err := p.issuerFor(ctx, ghost)
	require.Error(t, err)
	assert.Empty(t, issuer)
}

func TestIssuerFor_OptedInUsesSlugAndAppScope(t *testing.T) {
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	require.NoError(t, p.DeclareSettings(eng.Settings()))

	a := &app.App{ID: id.NewAppID(), Name: "Acme", Slug: "acme", PublishableKey: "pk_test_acme",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, eng.Store().CreateApp(ctx, a))
	raw, err := json.Marshal(true)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, "oauth2.app_issuer", raw,
		settings.ScopeApp, a.ID.String(), a.ID.String(), "", "test"))

	issuer, scope, err := p.issuerFor(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com/apps/acme", issuer)
	assert.Equal(t, a.ID.String(), scope)

	// Another app is unaffected.
	issuer, scope, err = p.issuerFor(ctx, id.NewAppID())
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com", issuer)
	assert.Empty(t, scope)
}
