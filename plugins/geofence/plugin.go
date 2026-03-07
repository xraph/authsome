// Package geofence provides country-based access control for authentication.
// It allows/blocks authentication based on the geographic location of the
// request IP, using GeoIP data from the geoip plugin.
package geofence

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/geoip"
	"github.com/xraph/authsome/settings"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.BeforeSignIn     = (*Plugin)(nil)
	_ plugin.BeforeSignUp     = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingDefaultPolicy is the default access policy when no country rules match.
	SettingDefaultPolicy = settings.Define("geofence.default_policy", "allow_all",
		settings.WithDisplayName("Default Policy"),
		settings.WithDescription("Default access policy when no country rules match"),
		settings.WithCategory("Geofencing"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Default access policy when no country rules match"),
		settings.WithOrder(10),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Label: "Allow All", Value: "allow_all"},
			formconfig.SelectOption{Label: "Block All", Value: "block_all"},
		),
	)

	// SettingAllowedCountries is the list of ISO 3166-1 alpha-2 country codes allowed.
	SettingAllowedCountries = settings.Define("geofence.allowed_countries", []string{},
		settings.WithDisplayName("Allowed Countries"),
		settings.WithDescription("ISO 3166-1 alpha-2 country codes allowed"),
		settings.WithCategory("Geofencing"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("ISO 3166-1 alpha-2 country codes allowed. Empty means all allowed."),
		settings.WithOrder(20),
		settings.WithPlaceholder("US, GB, DE"),
	)

	// SettingBlockedCountries is the list of ISO 3166-1 alpha-2 country codes blocked.
	SettingBlockedCountries = settings.Define("geofence.blocked_countries", []string{},
		settings.WithDisplayName("Blocked Countries"),
		settings.WithDescription("ISO 3166-1 alpha-2 country codes blocked"),
		settings.WithCategory("Geofencing"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("ISO 3166-1 alpha-2 country codes blocked."),
		settings.WithOrder(30),
		settings.WithPlaceholder("CN, RU"),
	)

	// SettingBlockMessage is the error message shown when access is denied.
	SettingBlockMessage = settings.Define("geofence.block_message", "access denied based on location",
		settings.WithDisplayName("Block Message"),
		settings.WithDescription("Error message shown when access is denied by geofence rules"),
		settings.WithCategory("Geofencing"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Error message shown when access is denied by geofence rules"),
		settings.WithOrder(40),
	)
)

// Config configures the geofence plugin.
type Config struct {
	// DefaultPolicy is the default action when no rules match.
	// "allow_all" (default) or "deny_all".
	DefaultPolicy string

	// AllowedCountries is a whitelist of ISO 3166-1 alpha-2 country codes.
	// When non-empty, only these countries are allowed.
	AllowedCountries []string

	// BlockedCountries is a blacklist of ISO 3166-1 alpha-2 country codes.
	// Requests from these countries are blocked.
	BlockedCountries []string

	// BlockMessage is the error message returned when blocked (default: "access denied from your location").
	BlockMessage string
}

func (c *Config) defaults() {
	if c.DefaultPolicy == "" {
		c.DefaultPolicy = "allow_all"
	}
	if c.BlockMessage == "" {
		c.BlockMessage = "access denied from your location"
	}
}

// Plugin implements country-based geofencing for auth events.
type Plugin struct {
	config      Config
	geoIP       *geoip.Plugin
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager

	// Pre-computed lookup sets for O(1) checks.
	allowSet map[string]struct{}
	blockSet map[string]struct{}
}

// New creates a new geofence plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()

	allowSet := make(map[string]struct{}, len(c.AllowedCountries))
	for _, cc := range c.AllowedCountries {
		allowSet[cc] = struct{}{}
	}
	blockSet := make(map[string]struct{}, len(c.BlockedCountries))
	for _, cc := range c.BlockedCountries {
		blockSet[cc] = struct{}{}
	}

	return &Plugin{
		config:   c,
		allowSet: allowSet,
		blockSet: blockSet,
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "geofence" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all geofence-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "geofence", SettingDefaultPolicy); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "geofence", SettingAllowedCountries); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "geofence", SettingBlockedCountries); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "geofence", SettingBlockMessage)
}

// OnInit discovers the geoip plugin and engine bridges.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type loggerGetter interface{ Logger() log.Logger }
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	type chronicleGetter interface{ Chronicle() bridge.Chronicle }
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}
	type relayGetter interface{ Relay() bridge.EventRelay }
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	// Discover geoip plugin.
	type pluginLister interface {
		Plugin(name string) plugin.Plugin
	}
	if pl, ok := engine.(pluginLister); ok {
		if gp := pl.Plugin("geoip"); gp != nil {
			if geoPlugin, ok := gp.(*geoip.Plugin); ok {
				p.geoIP = geoPlugin
			}
		}
	}

	if p.geoIP == nil {
		p.logger.Warn("geofence: geoip plugin not found, geofencing will be inactive")
	}

	return nil
}

// OnBeforeSignIn checks if the sign-in IP is allowed by geofence rules.
func (p *Plugin) OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

// OnBeforeSignUp checks if the sign-up IP is allowed by geofence rules.
func (p *Plugin) OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

func (p *Plugin) check(ctx context.Context, ipAddress, appID string) error {
	if p.geoIP == nil || ipAddress == "" {
		return nil
	}

	loc := p.geoIP.Resolve(ipAddress)
	if loc == nil {
		// Can't resolve — follow default policy.
		if p.config.DefaultPolicy == "deny_all" {
			return fmt.Errorf("geofence: %s", p.config.BlockMessage)
		}
		return nil
	}

	country := loc.Country

	// Check blocklist first.
	if _, blocked := p.blockSet[country]; blocked {
		p.auditBlock(ctx, appID, loc)
		return fmt.Errorf("geofence: %s", p.config.BlockMessage)
	}

	// Check allowlist (if configured).
	if len(p.allowSet) > 0 {
		if _, allowed := p.allowSet[country]; !allowed {
			p.auditBlock(ctx, appID, loc)
			return fmt.Errorf("geofence: %s", p.config.BlockMessage)
		}
	}

	// Default policy.
	if p.config.DefaultPolicy == "deny_all" {
		p.auditBlock(ctx, appID, loc)
		return fmt.Errorf("geofence: %s", p.config.BlockMessage)
	}

	return nil
}

func (p *Plugin) auditBlock(ctx context.Context, appID string, loc *geoip.GeoLocation) {
	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
			Action:   "geofence_blocked",
			Resource: "auth",
			Tenant:   appID,
			Outcome:  bridge.OutcomeFailure,
			Severity: bridge.SeverityWarning,
			Metadata: map[string]string{
				"country": loc.Country,
				"city":    loc.City,
				"ip":      loc.IP,
			},
		})
	}
	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{
			Type:     "security.geofence_blocked",
			TenantID: appID,
			Data: map[string]string{
				"country": loc.Country,
				"ip":      loc.IP,
			},
		})
	}
}
