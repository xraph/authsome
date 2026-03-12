package geoip

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.OnShutdown       = (*Plugin)(nil)
	_ plugin.BeforeSignIn     = (*Plugin)(nil)
	_ plugin.AfterSignIn      = (*Plugin)(nil)
	_ plugin.BeforeSignUp     = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingCacheTTLHours controls how long GeoIP lookup results are cached.
	SettingCacheTTLHours = settings.Define("geoip.cache_ttl_hours", 24,
		settings.WithDisplayName("Cache TTL (hours)"),
		settings.WithDescription("How long GeoIP lookup results are cached, in hours"),
		settings.WithCategory("GeoIP"),
		settings.WithScopes(settings.ScopeGlobal),
		settings.WithHelpText("How long GeoIP lookup results are cached, in hours"),
		settings.WithOrder(10),
	)
)

// Config configures the GeoIP plugin.
type Config struct {
	// DatabasePath is the path to the MaxMind GeoLite2-City.mmdb file.
	DatabasePath string

	// CacheTTL is how long GeoIP results are cached (default: 24h).
	CacheTTL time.Duration
}

func (c *Config) defaults() {
	if c.CacheTTL == 0 {
		c.CacheTTL = 24 * time.Hour
	}
}

// Plugin is the GeoIP location resolution plugin. It resolves IP addresses
// to geographic locations and stores results in context for downstream
// geo-security plugins.
type Plugin struct {
	config      Config
	provider    Provider
	cache       *Cache
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager
}

// New creates a new GeoIP plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "geoip" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all geoip-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	return settings.RegisterTyped(m, "geoip", SettingCacheTTLHours)
}

// OnInit opens the MaxMind database and discovers engine dependencies.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	// Discover logger.
	type loggerGetter interface{ Logger() log.Logger }
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	// Discover bridges.
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

	// Open MaxMind database.
	if p.config.DatabasePath != "" {
		provider, err := NewMaxMindProvider(p.config.DatabasePath)
		if err != nil {
			return fmt.Errorf("geoip: init: %w", err)
		}
		p.provider = provider
	}

	p.cache = NewCache(p.config.CacheTTL)
	return nil
}

// OnShutdown closes the GeoIP provider.
func (p *Plugin) OnShutdown(_ context.Context) error {
	if p.provider != nil {
		return p.provider.Close()
	}
	return nil
}

// Resolve looks up the geographic location for an IP string. Results are
// cached. Returns nil for private/loopback IPs or if no provider is configured.
func (p *Plugin) Resolve(ipStr string) *GeoLocation {
	if p.provider == nil || ipStr == "" {
		return nil
	}

	// Strip port if present.
	if strings.Contains(ipStr, ":") {
		host, _, err := net.SplitHostPort(ipStr)
		if err == nil {
			ipStr = host
		}
	}

	// Check cache.
	if loc := p.cache.Get(ipStr); loc != nil {
		return loc
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() {
		return nil
	}

	loc, err := p.provider.Lookup(ip)
	if err != nil {
		p.logger.Warn("geoip: lookup failed",
			log.String("ip", ipStr),
			log.String("error", err.Error()),
		)
		return nil
	}

	p.cache.Set(ipStr, loc)
	return loc
}

// OnBeforeSignIn resolves geo for the sign-in request and stores it in context.
func (p *Plugin) OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	// Resolve geo for the sign-in request. Downstream plugins will call Resolve themselves.
	_ = p.Resolve(req.IPAddress)
	return nil
}

// OnBeforeSignUp resolves geo for the sign-up request.
func (p *Plugin) OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	// Resolve but don't block — downstream plugins will call Resolve themselves.
	_ = p.Resolve(req.IPAddress)
	return nil
}

// OnAfterSignIn enriches audit/relay events with geo metadata.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	loc := p.Resolve(s.IPAddress)
	if loc == nil {
		return nil
	}

	p.auditGeo(ctx, "signin_geo", u.ID, u.AppID, loc)
	p.relayGeo(ctx, "auth.signin.geo", u.AppID.String(), u.ID.String(), s.ID.String(), loc)
	return nil
}

func (p *Plugin) auditGeo(ctx context.Context, action string, userID id.UserID, appID id.AppID, loc *GeoLocation) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:   action,
		Resource: "session",
		ActorID:  userID.String(),
		Tenant:   appID.String(),
		Outcome:  bridge.OutcomeSuccess,
		Severity: bridge.SeverityInfo,
		Metadata: map[string]string{
			"country":      loc.Country,
			"country_name": loc.CountryName,
			"city":         loc.City,
			"ip":           loc.IP,
		},
	})
}

func (p *Plugin) relayGeo(ctx context.Context, eventType, appID, userID, sessionID string, loc *GeoLocation) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{
		Type:     eventType,
		TenantID: appID,
		Data: map[string]string{
			"user_id":      userID,
			"session_id":   sessionID,
			"country":      loc.Country,
			"country_name": loc.CountryName,
			"city":         loc.City,
			"ip":           loc.IP,
		},
	})
}
