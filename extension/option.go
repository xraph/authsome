package extension

import (
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
