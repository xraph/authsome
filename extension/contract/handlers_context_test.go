package contract

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
)

// switchCtx is a command ctx carrying a recorder, so the test can read back
// the cookies a switch command wrote.
func switchCtx(t *testing.T) (context.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/dashboard/v1", nil)
	return dashauth.WithHTTP(context.Background(), w, r), w
}

// cookiesByName indexes the Set-Cookie headers a recorder collected.
func cookiesByName(w *httptest.ResponseRecorder) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range w.Result().Cookies() {
		out[c.Name] = c
	}
	return out
}

// defaultEnvOf returns the first environment bootstrap made for appID.
func defaultEnvOf(t *testing.T, eng *authsome.Engine, appID id.AppID) id.EnvironmentID {
	t.Helper()
	envs, err := eng.ListEnvironments(context.Background(), appID)
	require.NoError(t, err)
	require.NotEmpty(t, envs)
	for _, e := range envs {
		if e.IsDefault {
			return e.ID
		}
	}
	return envs[0].ID
}

func TestAppsSwitch_WritesTheAppCookieAndClearsTheEnv(t *testing.T) {
	eng := newSetupEngine(t)
	tenant := &app.App{Name: "Tenant", Slug: "tenant"}
	require.NoError(t, eng.CreateApp(context.Background(), tenant))

	ctx, w := switchCtx(t)
	got, err := appsSwitchHandler(Deps{Engine: eng})(ctx, AppSwitchInput{AppID: tenant.ID.String()}, dashcontract.Principal{})

	require.NoError(t, err)
	assert.True(t, got.OK)
	cookies := cookiesByName(w)
	require.Contains(t, cookies, AppSwitcherCookie)
	assert.Equal(t, tenant.ID.String(), cookies[AppSwitcherCookie].Value)
	require.Contains(t, cookies, EnvSwitcherCookie, "the old env belongs to the old app, so it is cleared")
	assert.Negative(t, cookies[EnvSwitcherCookie].MaxAge)
}

func TestAppsSwitch_EmptyIDClearsTheAppCookie(t *testing.T) {
	eng := newSetupEngine(t)

	ctx, w := switchCtx(t)
	got, err := appsSwitchHandler(Deps{Engine: eng})(ctx, AppSwitchInput{AppID: "  "}, dashcontract.Principal{})

	require.NoError(t, err)
	assert.True(t, got.OK)
	cookies := cookiesByName(w)
	require.Contains(t, cookies, AppSwitcherCookie)
	assert.Negative(t, cookies[AppSwitcherCookie].MaxAge)
	assert.NotContains(t, cookies, EnvSwitcherCookie)
}

func TestEnvironmentsSwitch_WritesTheEnvCookie(t *testing.T) {
	eng := newSetupEngine(t)
	platform := eng.PlatformAppID()
	envID := defaultEnvOf(t, eng, platform)

	ctx, w := switchCtx(t)
	p := dashcontract.Principal{Claims: map[string]any{"app_id": platform.String()}}
	got, err := environmentsSwitchHandler(Deps{Engine: eng})(ctx, EnvSwitchInput{EnvID: envID.String()}, p)

	require.NoError(t, err)
	assert.True(t, got.OK)
	cookies := cookiesByName(w)
	require.Contains(t, cookies, EnvSwitcherCookie)
	assert.Equal(t, envID.String(), cookies[EnvSwitcherCookie].Value)
}

func TestEnvironmentsSwitch_EmptyIDClearsTheEnvCookie(t *testing.T) {
	eng := newSetupEngine(t)

	ctx, w := switchCtx(t)
	got, err := environmentsSwitchHandler(Deps{Engine: eng})(ctx, EnvSwitchInput{}, dashcontract.Principal{})

	require.NoError(t, err)
	assert.True(t, got.OK)
	cookies := cookiesByName(w)
	require.Contains(t, cookies, EnvSwitcherCookie)
	assert.Negative(t, cookies[EnvSwitcherCookie].MaxAge)
}

func TestEnvironmentsSwitch_RefusesAnotherAppsEnvironment(t *testing.T) {
	eng := newSetupEngine(t)
	tenant := &app.App{Name: "Tenant", Slug: "tenant"}
	require.NoError(t, eng.CreateApp(context.Background(), tenant))
	platformEnv := defaultEnvOf(t, eng, eng.PlatformAppID())

	ctx, w := switchCtx(t)
	p := dashcontract.Principal{Claims: map[string]any{"app_id": tenant.ID.String()}}
	_, err := environmentsSwitchHandler(Deps{Engine: eng})(ctx, EnvSwitchInput{EnvID: platformEnv.String()}, p)

	var cerr *dashcontract.Error
	require.True(t, errors.As(err, &cerr), "want a contract error, got %v", err)
	assert.Equal(t, dashcontract.CodeBadRequest, cerr.Code)
	assert.Empty(t, cookiesByName(w), "a refused switch writes no cookie")
}
