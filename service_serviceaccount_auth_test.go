package authsome_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xraph/authsome/environment"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/ratelimit"
)

func TestEngine_DeletedServiceAccountAPIKeyIsRefused(t *testing.T) {
	ctx := context.Background()
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.APIKeyFailureLimit = 2
	p := apikeyPlugin.New()
	eng := secutil.NewTestEngine(t, authsome.WithConfig(cfg),
		authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	// NewTestEngine supplies the memory store and seeds its platform app.
	store := eng.APIKeyStore()
	appID, err := id.ParseAppID(cfg.AppID)
	require.NoError(t, err)
	env := &environment.Environment{ID: id.NewEnvironmentID(), AppID: appID, Name: "Production", Slug: "production", Type: environment.TypeProduction, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, eng.Store().CreateEnvironment(ctx, env))
	sa, err := eng.CreateServiceAccountInEnvironment(ctx, appID, env.ID, "build runner", "", nil)
	require.NoError(t, err)
	key, raw, err := eng.CreateServiceAccountAPIKey(ctx, sa.ID, "build key", nil, nil)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/data", nil)
	req.RemoteAddr = "203.0.113.9:4000"
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", appID.String())
	s := p.Strategy()
	result, err := s.Authenticate(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Session)
	assert.Equal(t, principal.Ref{Kind: principal.KindService, ID: sa.ID.String()}, result.Session.Subject())

	require.NoError(t, eng.DeleteServiceAccount(ctx, sa.ID))
	_, err = eng.ResolvePrincipal(ctx, result.Session.Subject())
	require.ErrorIs(t, err, principal.ErrNotFound)
	retained, err := store.GetAPIKey(ctx, key.ID)
	require.NoError(t, err, "deletion must leave the credential available to exercise principal validation")
	require.True(t, retained.IsValid())
	for range 2 {
		result, err = s.Authenticate(ctx, req)
		assert.Nil(t, result)
		assert.EqualError(t, err, "apikey: service account is not active")
	}
	result, err = s.Authenticate(ctx, req)
	assert.Nil(t, result)
	assert.EqualError(t, err, "apikey: too many failed attempts from this address")
}
