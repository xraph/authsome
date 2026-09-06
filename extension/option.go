package extension

import (
	"time"

	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
)

// ExtOption configures the Forge extension.
type ExtOption func(*Extension)

// WithLogger sets the logger for the extension.
func WithLogger(logger log.Logger) ExtOption {
	return func(e *Extension) {
		e.logger = logger
	}
}

// WithConfig sets the extension configuration.
func WithConfig(cfg Config) ExtOption {
	return func(e *Extension) {
		e.config = cfg
	}
}

// WithEngineOption adds an authsome.Option to be applied to the engine.
func WithEngineOption(opt authsome.Option) ExtOption {
	return func(e *Extension) {
		e.opts = append(e.opts, opt)
	}
}

// WithPlugin registers a plugin with the extension.
func WithPlugin(p plugin.Plugin) ExtOption {
	return func(e *Extension) {
		e.plugins = append(e.plugins, p)
	}
}

// WithPlugins registers multiple plugins with the extension.
func WithPlugins(plugins ...plugin.Plugin) ExtOption {
	return func(e *Extension) {
		e.plugins = append(e.plugins, plugins...)
	}
}

// WithEngineOptions adds multiple authsome.Options to be applied to the engine.
func WithEngineOptions(opts ...authsome.Option) ExtOption {
	return func(e *Extension) {
		e.opts = append(e.opts, opts...)
	}
}

// WithDisableRoutes prevents HTTP route registration.
func WithDisableRoutes() ExtOption {
	return func(e *Extension) { e.config.DisableRoutes = true }
}

// WithDisableMigrate prevents auto-migration on start.
func WithDisableMigrate() ExtOption {
	return func(e *Extension) { e.config.DisableMigrate = true }
}

// WithBasePath sets the URL prefix for auth routes.
func WithBasePath(path string) ExtOption {
	return func(e *Extension) { e.config.BasePath = path }
}

// WithRequireConfig requires config to be present in YAML files.
// If true and no config is found, Register returns an error.
func WithRequireConfig(require bool) ExtOption {
	return func(e *Extension) { e.config.RequireConfig = require }
}

// WithGroveDatabase sets the name of the grove.DB to resolve from the DI container.
// The extension will auto-construct the appropriate store backend (postgres/sqlite/mongo)
// based on the grove driver type. Pass an empty string to use the default (unnamed) grove.DB.
func WithGroveDatabase(name string) ExtOption {
	return func(e *Extension) {
		e.config.GroveDatabase = name
		e.useGrove = true
	}
}

// WithClientMode configures the extension as a client that delegates all
// auth operations to a remote authsome server via HTTP API. In this mode:
//   - No local engine is created
//   - No database connection is required
//   - Token validation uses the /v1/introspect endpoint on the remote server
//   - An authclient.Client is registered in the DI container for downstream use
//
// This is the intended mode for services that are NOT the auth authority
// (e.g. TwinOS instances that delegate to a shared Portal).
func WithClientMode(portalURL string) ExtOption {
	return func(e *Extension) {
		e.config.ClientMode = true
		e.config.PortalURL = portalURL
	}
}

// WithClientAutoRefresh tunes the client-mode session refresh. Refresh is
// already on by default; pass enabled=false to hand the job to something else,
// or a non-zero threshold to rotate earlier or later than five minutes before
// expiry.
//
// No effect outside client mode: engine mode resolves the same two knobs from
// the settings cascade (session.auto_refresh_enabled and
// session.auto_refresh_threshold_seconds).
func WithClientAutoRefresh(enabled bool, threshold time.Duration) ExtOption {
	return func(e *Extension) {
		e.config.ClientAutoRefreshDisabled = !enabled
		e.config.ClientAutoRefreshThreshold = threshold
	}
}

// WithServiceAPIKey sets the API key used to authenticate this service with
// the remote authsome server. Required for client mode bootstrap operations
// (e.g. creating organizations during startup).
func WithServiceAPIKey(key string) ExtOption {
	return func(e *Extension) {
		e.config.ServiceAPIKey = key
	}
}

// WithServiceAppID sets the App ID this service binds to. The
// resolved authclient stamps it as the X-App-ID header on every
// request — required by Authsome's API key authentication strategy
// (which keys API keys per (app_id, prefix) tuple) and by every
// App-scoped admin endpoint. Without this, service-account API
// key auth fails with 401 even when WithServiceAPIKey is set.
//
// In TwinOS deployments this is App `studio`'s ID; identity logs
// it at first boot and operators copy it into their service env
// (`TWINOS_AUTH_SERVICE_APP_ID`).
func WithServiceAppID(appID string) ExtOption {
	return func(e *Extension) {
		e.config.ServiceAppID = appID
	}
}
