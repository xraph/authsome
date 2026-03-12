package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Context helper tests
// ──────────────────────────────────────────────────

func TestWithEnvID_EnvIDFrom(t *testing.T) {
	envID := id.NewEnvironmentID()
	ctx := middleware.WithEnvID(context.Background(), envID)
	got, ok := middleware.EnvIDFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, envID, got)
}

func TestEnvIDFrom_Missing(t *testing.T) {
	_, ok := middleware.EnvIDFrom(context.Background())
	assert.False(t, ok)
}

func TestWithEnvironment_EnvironmentFrom(t *testing.T) {
	env := &environment.Environment{
		ID:   id.NewEnvironmentID(),
		Name: "Production",
		Type: environment.TypeProduction,
	}

	ctx := middleware.WithEnvironment(context.Background(), env)
	got, ok := middleware.EnvironmentFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, env.ID, got.ID)
	assert.Equal(t, "Production", got.Name)
}

func TestEnvironmentFrom_Missing(t *testing.T) {
	_, ok := middleware.EnvironmentFrom(context.Background())
	assert.False(t, ok)
}

func TestWithEnvironmentSettings_EnvironmentSettingsFrom(t *testing.T) {
	s := &environment.Settings{
		RateLimitEnabled: boolPtr(true),
	}

	ctx := middleware.WithEnvironmentSettings(context.Background(), s)
	got, ok := middleware.EnvironmentSettingsFrom(ctx)
	require.True(t, ok)
	assert.True(t, *got.RateLimitEnabled)
}

func TestEnvironmentSettingsFrom_Missing(t *testing.T) {
	_, ok := middleware.EnvironmentSettingsFrom(context.Background())
	assert.False(t, ok)
}

// ──────────────────────────────────────────────────
// EnvironmentMiddleware tests
// ──────────────────────────────────────────────────

func TestEnvironmentMiddleware_HeaderResolution(t *testing.T) {
	testAppID := id.NewAppID()
	testEnvID := id.NewEnvironmentID()
	testEnv := &environment.Environment{
		ID:    testEnvID,
		AppID: testAppID,
		Name:  "Production",
		Type:  environment.TypeProduction,
	}

	var (
		gotEnvID id.EnvironmentID
		gotEnv   *environment.Environment
		envIDOK  bool
		envOK    bool
	)

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(envID id.EnvironmentID) (*environment.Environment, error) {
			if envID == testEnvID {
				return testEnv, nil
			}
			return nil, errors.New("not found")
		},
		ResolveDefault: func(_ id.AppID) (*environment.Environment, error) {
			t.Fatal("should not be called when header is present")
			return nil, nil
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	// Inject AppID into context first (EnvironmentMiddleware needs it).
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithAppID(ctx.Context(), testAppID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotEnvID, envIDOK = middleware.EnvIDFrom(ctx.Context())
		gotEnv, envOK = middleware.EnvironmentFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Environment-ID", testEnvID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, envIDOK)
	assert.True(t, envOK)
	assert.Equal(t, testEnvID, gotEnvID)
	assert.Equal(t, "Production", gotEnv.Name)
}

func TestEnvironmentMiddleware_FallbackToDefault(t *testing.T) {
	testAppID := id.NewAppID()
	defaultEnv := &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     testAppID,
		Name:      "Development",
		Type:      environment.TypeDevelopment,
		IsDefault: true,
	}

	var (
		gotEnv *environment.Environment
		envOK  bool
	)

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(_ id.EnvironmentID) (*environment.Environment, error) {
			return nil, errors.New("not found")
		},
		ResolveDefault: func(appID id.AppID) (*environment.Environment, error) {
			if appID == testAppID {
				return defaultEnv, nil
			}
			return nil, errors.New("no default")
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithAppID(ctx.Context(), testAppID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotEnv, envOK = middleware.EnvironmentFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, envOK)
	assert.Equal(t, "Development", gotEnv.Name)
	assert.True(t, gotEnv.IsDefault)
}

func TestEnvironmentMiddleware_ResolvesSettings(t *testing.T) {
	testAppID := id.NewAppID()
	defaultEnv := &environment.Environment{
		ID:    id.NewEnvironmentID(),
		AppID: testAppID,
		Name:  "Production",
		Type:  environment.TypeProduction,
		Settings: &environment.Settings{
			RateLimitEnabled: boolPtr(true),
		},
	}

	var (
		gotSettings *environment.Settings
		settingsOK  bool
	)

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(_ id.EnvironmentID) (*environment.Environment, error) {
			return nil, errors.New("not found")
		},
		ResolveDefault: func(_ id.AppID) (*environment.Environment, error) {
			return defaultEnv, nil
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithAppID(ctx.Context(), testAppID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotSettings, settingsOK = middleware.EnvironmentSettingsFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, settingsOK)
	require.NotNil(t, gotSettings)
	assert.True(t, *gotSettings.RateLimitEnabled)
}

func TestEnvironmentMiddleware_NoAppID(t *testing.T) {
	var called bool

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(_ id.EnvironmentID) (*environment.Environment, error) {
			t.Fatal("should not be called without AppID")
			return nil, nil
		},
		ResolveDefault: func(_ id.AppID) (*environment.Environment, error) {
			t.Fatal("should not be called without AppID")
			return nil, nil
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should still be called without AppID")
}

func TestEnvironmentMiddleware_InvalidHeader(t *testing.T) {
	testAppID := id.NewAppID()

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(_ id.EnvironmentID) (*environment.Environment, error) {
			t.Fatal("should not be called with invalid header")
			return nil, nil
		},
		ResolveDefault: func(_ id.AppID) (*environment.Environment, error) {
			t.Fatal("should not be called with invalid header")
			return nil, nil
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithAppID(ctx.Context(), testAppID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Environment-ID", "not-a-valid-id")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid X-Environment-ID")
}

func TestEnvironmentMiddleware_WrongApp(t *testing.T) {
	testAppID := id.NewAppID()
	otherAppID := id.NewAppID()
	testEnvID := id.NewEnvironmentID()
	testEnv := &environment.Environment{
		ID:    testEnvID,
		AppID: otherAppID, // Belongs to a different app.
		Name:  "Other",
		Type:  environment.TypeProduction,
	}

	mw := middleware.EnvironmentMiddleware(middleware.EnvironmentMiddlewareConfig{
		ResolveEnvironment: func(envID id.EnvironmentID) (*environment.Environment, error) {
			return testEnv, nil
		},
		ResolveDefault: func(appID id.AppID) (*environment.Environment, error) {
			return nil, errors.New("not found")
		},
		Logger: log.NewNoopLogger(),
	})

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithAppID(ctx.Context(), testAppID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Environment-ID", testEnvID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "does not belong")
}

// ──────────────────────────────────────────────────
// RequireEnvironment tests
// ──────────────────────────────────────────────────

func TestRequireEnvironment_WithEnv(t *testing.T) {
	var called bool

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithEnvID(ctx.Context(), id.NewEnvironmentID())
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(middleware.RequireEnvironment())
	router.GET("/test", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}

func TestRequireEnvironment_WithoutEnv(t *testing.T) {
	var called bool

	router := forge.NewRouter()
	router.Use(middleware.RequireEnvironment())
	router.GET("/test", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called, "handler should not be called without environment")
	assert.Contains(t, rec.Body.String(), "environment context required")
}

func boolPtr(b bool) *bool { return &b }
