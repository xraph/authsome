// Package vpndetect provides VPN, proxy, and Tor exit node detection for
// authentication events. It reads GeoLocation flags from the geoip plugin
// and optionally blocks or flags suspicious connections.
package vpndetect

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
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
	// SettingBlockVPN controls whether VPN connections are blocked.
	SettingBlockVPN = settings.Define("vpndetect.block_vpn", false,
		settings.WithDisplayName("Block VPN"),
		settings.WithDescription("Block authentication attempts from VPN connections"),
		settings.WithCategory("VPN Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Block authentication attempts from VPN connections"),
		settings.WithOrder(10),
	)

	// SettingBlockProxy controls whether proxy connections are blocked.
	SettingBlockProxy = settings.Define("vpndetect.block_proxy", false,
		settings.WithDisplayName("Block Proxy"),
		settings.WithDescription("Block authentication attempts from proxy connections"),
		settings.WithCategory("VPN Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Block authentication attempts from proxy connections"),
		settings.WithOrder(20),
	)

	// SettingBlockTor controls whether Tor exit node connections are blocked.
	SettingBlockTor = settings.Define("vpndetect.block_tor", true,
		settings.WithDisplayName("Block Tor"),
		settings.WithDescription("Block authentication attempts from Tor exit nodes"),
		settings.WithCategory("VPN Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Block authentication attempts from Tor exit nodes"),
		settings.WithOrder(30),
	)

	// SettingBlockMessage is the error message shown when access is blocked.
	SettingBlockMessage = settings.Define("vpndetect.block_message", "",
		settings.WithDisplayName("Block Message"),
		settings.WithDescription("Error message shown when access is blocked due to VPN/proxy/Tor detection"),
		settings.WithCategory("VPN Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Error message shown when access is blocked due to VPN/proxy/Tor detection"),
		settings.WithPlaceholder("Access denied: VPN or proxy detected"),
		settings.WithOrder(40),
	)
)

// Config configures the VPN/Proxy/Tor detection plugin.
type Config struct {
	// BlockVPN blocks VPN connections (default: false).
	BlockVPN bool

	// BlockProxy blocks proxy connections (default: false).
	BlockProxy bool

	// BlockTor blocks Tor exit node connections (default: true).
	BlockTor bool

	// BlockMessage is the error message returned on block.
	BlockMessage string
}

func (c *Config) defaults() {
	if c.BlockMessage == "" {
		c.BlockMessage = "connection type not allowed"
	}
}

// Plugin detects and optionally blocks VPN/proxy/Tor connections.
type Plugin struct {
	config      Config
	geoIP       *geoip.Plugin
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager
}

// New creates a new VPN detection plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "vpndetect" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all VPN-detection-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "vpndetect", SettingBlockVPN); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "vpndetect", SettingBlockProxy); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "vpndetect", SettingBlockTor); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "vpndetect", SettingBlockMessage)
}

// OnInit discovers engine dependencies.
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
		p.logger.Warn("vpndetect: geoip plugin not found, detection will be inactive")
	}

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	return nil
}

// OnBeforeSignIn checks the connection type.
func (p *Plugin) OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

// OnBeforeSignUp checks the connection type.
func (p *Plugin) OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

func (p *Plugin) check(ctx context.Context, ipAddress, appID string) error {
	if p.geoIP == nil || ipAddress == "" {
		return nil
	}

	loc := p.geoIP.Resolve(ipAddress)
	if loc == nil {
		return nil
	}

	var reason string
	switch {
	case loc.IsVPN && p.config.BlockVPN:
		reason = "vpn"
	case loc.IsProxy && p.config.BlockProxy:
		reason = "proxy"
	case loc.IsTor && p.config.BlockTor:
		reason = "tor"
	default:
		return nil
	}

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
			Action:   "vpndetect_blocked",
			Resource: "auth",
			Tenant:   appID,
			Outcome:  bridge.OutcomeFailure,
			Severity: bridge.SeverityWarning,
			Metadata: map[string]string{
				"reason":  reason,
				"ip":      loc.IP,
				"country": loc.Country,
			},
		})
	}
	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{
			Type:     "security.vpndetect_blocked",
			TenantID: appID,
			Data: map[string]string{
				"reason": reason,
				"ip":     loc.IP,
			},
		})
	}

	return fmt.Errorf("vpndetect: %s: %s", reason, p.config.BlockMessage)
}
