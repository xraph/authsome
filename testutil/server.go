// Package testutil provides a ready-to-use HTTP test server backed by a real
// AuthSome engine with in-memory stores and all standard plugins. It creates
// an httptest.Server that mirrors the production API surface so the generated
// SDK client can be exercised end-to-end over HTTP.
package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/id"
	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/consent"
	"github.com/xraph/authsome/plugins/oauth2provider"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/subscription"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// DefaultAppID is the TypeID used for the test application.
const DefaultAppID = "aapp_01jf0000000000000000000000"

// TestServer wraps a full AuthSome engine with an httptest.Server and a
// pre-configured SDK client. All stores are in-memory.
type TestServer struct {
	Engine    *authsome.Engine
	Server    *httptest.Server
	Client    *authclient.Client
	AppID     string
	Store     *memory.Store
	Warden    *warden.Engine
	OrgPlugin *orgplugin.Plugin
	Logger    log.Logger
}

// ServerOption configures the test server.
type ServerOption func(*serverConfig)

type serverConfig struct {
	appID   string
	plugins []plugin.Plugin
}

// WithAppID overrides the default test app ID.
func WithAppID(appID string) ServerOption {
	return func(c *serverConfig) { c.appID = appID }
}

// WithPlugins adds extra plugins beyond the default set (password, apikey, organization).
func WithPlugins(plugins ...plugin.Plugin) ServerOption {
	return func(c *serverConfig) { c.plugins = append(c.plugins, plugins...) }
}

// NewTestServer creates a fully wired AuthSome test server with in-memory
// stores, the standard plugin set, and an httptest.Server serving the API.
// The returned Client is pointed at the test server's URL.
//
// t.Cleanup automatically stops the server and engine.
func NewTestServer(t *testing.T, opts ...ServerOption) *TestServer {
	t.Helper()

	cfg := &serverConfig{appID: DefaultAppID}
	for _, opt := range opts {
		opt(cfg)
	}

	store := memory.New()
	logger := log.NewNoopLogger()

	wardenEng, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	if err != nil {
		t.Fatalf("testutil: create warden: %v", err)
	}

	orgPlugin := orgplugin.New()

	engineOpts := make([]authsome.Option, 0, 13+len(cfg.plugins)) //nolint:mnd // preallocate for base opts + plugins
	engineOpts = append(engineOpts,
		authsome.WithStore(store),
		authsome.WithLogger(logger),
		authsome.WithWarden(wardenEng),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(cfg.appID),
		// Core plugins
		authsome.WithPlugin(password.New()),
		authsome.WithPlugin(apikey.New()),
		authsome.WithPlugin(orgPlugin),
		// Additional plugins
		authsome.WithPlugin(scim.New()),
		authsome.WithPlugin(subscription.New()),
		authsome.WithPlugin(phone.New()),
		authsome.WithPlugin(consent.New()),
		authsome.WithPlugin(oauth2provider.New()),
	)

	for _, p := range cfg.plugins {
		engineOpts = append(engineOpts, authsome.WithPlugin(p))
	}

	engine, err := authsome.NewEngine(engineOpts...)
	if err != nil {
		t.Fatalf("testutil: create engine: %v", err)
	}

	ctx := context.Background()
	if startErr := engine.Start(ctx); startErr != nil {
		t.Fatalf("testutil: start engine: %v", startErr)
	}

	// Build the API handler with plugin routes included.
	// api.New(engine).Handler() creates its own router, registers core routes,
	// and returns an http.Handler. We also need to register plugin routes
	// (org, apikey, etc.) on the same router before building the handler.
	router := forge.NewRouter()
	apiHandler := api.New(engine, router)
	if routeErr := apiHandler.RegisterRoutes(router); routeErr != nil {
		t.Fatalf("testutil: register API routes: %v", routeErr)
	}
	// Register plugin routes (org, apikey, passkey, etc.)
	for _, rp := range engine.Plugins().RouteProviders() {
		if pluginErr := rp.RegisterRoutes(router); pluginErr != nil {
			t.Fatalf("testutil: register plugin routes (%T): %v", rp, pluginErr)
		}
	}
	if startErr2 := router.Start(ctx); startErr2 != nil {
		t.Fatalf("testutil: start router: %v", startErr2)
	}
	handler := router.Handler()

	// Wrap with raw HTTP middleware that resolves Bearer tokens into user
	// context the same way the Forge authsome extension does, but without
	// requiring Forge's context adapter.
	resolveSession := engine.ResolveSessionByToken
	resolveUser := func(userID string) (*user.User, error) {
		parsed, parseErr := id.ParseUserID(userID)
		if parseErr != nil {
			return nil, parseErr
		}
		return engine.GetUser(ctx, parsed)
	}
	wrappedHandler := authMiddlewareHTTP(handler, resolveSession, resolveUser)

	server := httptest.NewServer(wrappedHandler)

	client := authclient.NewClient(server.URL, authclient.WithSessionCookies())

	ts := &TestServer{
		Engine:    engine,
		Server:    server,
		Client:    client,
		AppID:     cfg.appID,
		Store:     store,
		Warden:    wardenEng,
		OrgPlugin: orgPlugin,
		Logger:    logger,
	}

	t.Cleanup(func() {
		server.Close()
		_ = engine.Stop(ctx) //nolint:errcheck // test cleanup
	})

	return ts
}

// Close shuts down the test server and engine.
func (ts *TestServer) Close() {
	ts.Server.Close()
	_ = ts.Engine.Stop(context.Background()) //nolint:errcheck // test cleanup
}

// authMiddlewareHTTP is a plain net/http middleware that extracts the Bearer
// token, resolves the session and user, and sets them in the request context
// using authsome/middleware context functions. This mirrors what the Forge
// AuthMiddleware does but works with raw http.Handlers.
func authMiddlewareHTTP(
	next http.Handler,
	resolveSession authmw.SessionResolver,
	resolveUser authmw.UserResolver,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		sess, err := resolveSession(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = authmw.WithSessionID(ctx, sess.ID)
		ctx = authmw.WithAppID(ctx, sess.AppID)
		if sess.OrgID != (id.OrgID{}) {
			ctx = authmw.WithOrgID(ctx, sess.OrgID)
		}

		u, err := resolveUser(sess.UserID.String())
		if err != nil {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx = authmw.WithUser(ctx, u)
		ctx = authmw.WithUserID(ctx, u.ID)
		ctx = authmw.WithAuthMethod(ctx, "session")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}
