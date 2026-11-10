package authsome_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// A provider saved for one app (what the admin API does with ?app_id)
// has to reach that app's sign-in page, start a flow there, and stay out
// of every other app. The lookups used to read the global list only, so
// app-scoped providers were saved and then never found.
func TestSocial_AppScopedProviderReachesOnlyItsApp(t *testing.T) {
	ctx := context.Background()
	sp := social.New()
	eng, st := newTestEngine(t, authsome.WithPlugin(sp))

	appA := testAppID(t)
	appB := id.NewAppID()
	now := time.Now()
	require.NoError(t, st.CreateApp(ctx, &app.App{ID: appB, Name: "Other", Slug: "other", CreatedAt: now, UpdatedAt: now}))

	raw, err := json.Marshal([]social.ProviderSetting{{
		Name: "github", ClientID: "app-a-client", ClientSecret: "s", Enabled: true,
	}})
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, social.SettingSocialProviders.Def.Key, raw,
		settings.ScopeApp, appA.String(), appA.String(), "", "test"))

	socialIDs := func(appID id.AppID) []string {
		cfg := eng.ClientConfig(ctx, appID)
		if cfg.Social == nil {
			return nil
		}
		var out []string
		for _, p := range cfg.Social.Providers {
			out = append(out, p.ID)
		}
		return out
	}
	assert.Equal(t, []string{"github"}, socialIDs(appA))
	assert.Empty(t, socialIDs(appB))

	userA := &user.User{ID: id.NewUserID(), AppID: appA, Email: "a@example.com", CreatedAt: now, UpdatedAt: now}
	userB := &user.User{ID: id.NewUserID(), AppID: appB, Email: "b@example.com", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, st.CreateUser(ctx, userA))
	require.NoError(t, st.CreateUser(ctx, userB))
	assert.True(t, sp.CanUnlink(ctx, userA.ID, "github"))
	assert.False(t, sp.CanUnlink(ctx, userB.ID, "github"))

	router := forge.NewRouter()
	require.NoError(t, sp.RegisterRoutes(router))
	start := func(appID id.AppID) *httptest.ResponseRecorder {
		reqCtx := forge.WithScope(ctx, forge.NewAppScope(appID.String()))
		req := httptest.NewRequestWithContext(reqCtx, http.MethodPost, "/v1/social/github", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	recA := start(appA)
	require.Equal(t, http.StatusOK, recA.Code, recA.Body.String())
	assert.Contains(t, recA.Body.String(), "client_id=app-a-client")
	assert.Equal(t, http.StatusBadRequest, start(appB).Code)
}

// The callback looks the provider up again, for the app the flow started
// in. A provider removed from that app between start and callback is gone.
func TestSocial_CallbackResolvesTheProviderForTheFlowsApp(t *testing.T) {
	ctx := context.Background()
	sp := social.New()
	eng, _ := newTestEngine(t, authsome.WithPlugin(sp))
	appA := testAppID(t)

	save := func(list []social.ProviderSetting) {
		raw, err := json.Marshal(list)
		require.NoError(t, err)
		require.NoError(t, eng.Settings().Set(ctx, social.SettingSocialProviders.Def.Key, raw,
			settings.ScopeApp, appA.String(), appA.String(), "", "test"))
	}
	save([]social.ProviderSetting{{Name: "github", ClientID: "app-a-client", ClientSecret: "s", Enabled: true}})

	router := forge.NewRouter()
	require.NoError(t, sp.RegisterRoutes(router))
	scoped := forge.WithScope(ctx, forge.NewAppScope(appA.String()))
	serve := func(method, target string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(scoped, method, target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	stateFrom := func() string {
		rec := serve(http.MethodPost, "/v1/social/github")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			AuthURL string `json:"auth_url"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		u, err := url.Parse(body.AuthURL)
		require.NoError(t, err)
		return u.Query().Get("state")
	}

	// Resolved for app A, so the request gets as far as the provider
	// error, which is answered without a token exchange.
	rec := serve(http.MethodGet, "/v1/social/github/callback?error=access_denied&state="+url.QueryEscape(stateFrom()))
	assert.NotContains(t, rec.Body.String(), "unsupported provider")

	state := stateFrom()
	save([]social.ProviderSetting{})
	rec = serve(http.MethodGet, "/v1/social/github/callback?code=x&state="+url.QueryEscape(state))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported provider")
}

// labelOnlySocial is a social plugin from before ProvidersForApp: the
// engine still lists its providers.
type labelOnlySocial struct{}

func (labelOnlySocial) Name() string        { return "social" }
func (labelOnlySocial) Providers() []string { return []string{"google"} }

func TestClientConfig_OlderSocialPluginStillListed(t *testing.T) {
	eng, _ := newTestEngine(t, authsome.WithPlugin(labelOnlySocial{}))
	cfg := eng.ClientConfig(context.Background(), testAppID(t))
	require.NotNil(t, cfg.Social)
	assert.Equal(t, []authsome.ClientConfigSocialProvider{{ID: "google", Name: "Google"}}, cfg.Social.Providers)
}
