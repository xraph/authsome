package scim

import (
	"context"

	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
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
	plugins      plugin.Registry
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
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	// Discover AuthSome store.
	type storeGetter interface {
		Store() store.Store
	}
	if sg, ok := engine.(storeGetter); ok {
		p.authStore = sg.Store()
	}

	// Discover chronicle bridge.
	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	// Discover relay bridge.
	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	// Discover hook bus.
	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	// Discover logger.
	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	// Discover settings manager.
	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settings = sg.Settings()
	}

	// Discover plugin registry.
	type pluginsGetter interface {
		Plugins() plugin.Registry
	}
	if pg, ok := engine.(pluginsGetter); ok {
		p.plugins = pg.Plugins()
	}

	// Discover default app ID.
	type configGetter interface {
		Config() authsome.Config
	}
	if cg, ok := engine.(configGetter); ok {
		p.defaultAppID = cg.Config().AppID
	}

	// Initialize in-memory store.
	p.scimStore = NewMemoryStore()

	// Initialize the service layer.
	p.service = &Service{
		store:     p.scimStore,
		authStore: p.authStore,
		settings:  p.settings,
		logger:    p.logger,
	}

	// Provide role ensurer for SCIM-provisioned users.
	if re, ok := engine.(roleEnsurer); ok {
		p.service.roleEnsurer = re
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
