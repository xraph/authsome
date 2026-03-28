package scim

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"

	"github.com/xraph/grove/migrate"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
)

// Plugin is the SCIM 2.0 provisioning plugin for authsome.
type Plugin struct {
	config  Config
	service *Service

	// SCIM store (in-memory by default).
	scimStore Store

	// AuthSome references.
	authStore    store.Store
	chronicle    bridge.Chronicle
	relay        bridge.EventRelay
	hooks        *hook.Bus
	logger       log.Logger
	settings     *settings.Manager
	plugins      *plugin.Registry
	defaultAppID string
}

// New creates a new SCIM plugin with the given configuration.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "scim" }

// OnInit captures bridge and engine references.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.authStore = engine.Store()
	p.chronicle = engine.Chronicle()
	p.relay = engine.Relay()
	p.hooks = engine.Hooks()
	p.logger = engine.Logger()
	p.settings = engine.Settings()
	p.plugins = engine.Plugins()
	p.defaultAppID = engine.DefaultAppID()

	// Initialize in-memory store.
	p.scimStore = NewMemoryStore()

	// Initialize the service layer.
	p.service = &Service{
		store:       p.scimStore,
		authStore:   p.authStore,
		settings:    p.settings,
		logger:      p.logger,
		roleEnsurer: engine,
	}

	return nil
}

// MigrationGroups returns SCIM-specific database migrations.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres", "postgresql":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// DeclareSettings registers SCIM settings with the settings manager.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "scim", SettingSCIMEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "scim", SettingAutoCreateUsers); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "scim", SettingAutoSuspendUsers); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "scim", SettingGroupSync); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "scim", SettingDefaultRole); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "scim", SettingTokenExpiryDays)
}
