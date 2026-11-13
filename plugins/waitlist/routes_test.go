package waitlist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
)

// The extension groups the router at the app's base path before plugins
// see it, and the UI client posts to {baseURL}/v1/waitlist/join. The
// plugin must therefore register under /v1 only. Prefixing the base path
// again put the routes at /authsome/authsome/waitlist/join, where nothing
// calls them.
func TestWaitlist_RoutesSitUnderV1NotTheBasePathAgain(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.BasePath = "/authsome"
	p := New()
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithPlugin(p))

	root := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(root.Group(cfg.BasePath)))
	h := root.Handler()

	post := func(path string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(`{"email":"joiner@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.NotEqual(t, http.StatusNotFound, post("/authsome/v1/waitlist/join"), "join must be where the UI client posts")
	assert.Equal(t, http.StatusNotFound, post("/authsome/authsome/waitlist/join"), "the base path must not appear twice")
}

// The UI client joins with only a publishable key, which the publishable-key
// middleware turns into an app on authsome's own context key. Reading only
// forge's scope missed it, so every join from the sign-up form failed with
// "unable to determine app_id" on a host with more than one app.
func TestWaitlist_JoinTakesTheAppFromThePublishableKey(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	p := New()
	_ = secutil.NewTestEngine(t, authsome.WithConfig(cfg), authsome.WithPlugin(p))
	p.defaultAppID = "" // a multi-app host has no single default to fall back on

	keyApp := id.NewAppID()
	root := forge.NewRouter()
	// What PublishableKeyMiddleware does with a valid key.
	root.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.WithContext(middleware.WithAppID(ctx.Context(), keyApp))
			return next(ctx)
		}
	})
	require.NoError(t, p.RegisterRoutes(root))
	h := root.Handler()

	join := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/waitlist/join", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := join(`{"email":"joiner@example.com"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	entry, err := p.store.GetEntryByEmail(context.Background(), keyApp, "joiner@example.com")
	require.NoError(t, err)
	assert.Equal(t, keyApp, entry.AppID, "the entry belongs to the key's app")
}
