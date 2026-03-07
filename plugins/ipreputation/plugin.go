// Package ipreputation checks IP addresses against threat intelligence
// sources and blocks or flags high-risk IPs during authentication.
package ipreputation

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/plugin"
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

func intPtr(i int) *int { return &i }

var (
	// SettingBlockThreshold is the score at or above which IPs are blocked.
	SettingBlockThreshold = settings.Define("ipreputation.block_threshold", 80,
		settings.WithDisplayName("Block Threshold"),
		settings.WithDescription("Score at or above which IPs are blocked"),
		settings.WithCategory("IP Reputation"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score at or above which IPs are blocked (0-100)"),
		settings.WithOrder(10),
	)

	// SettingWarnThreshold is the score at or above which IPs are flagged with a warning.
	SettingWarnThreshold = settings.Define("ipreputation.warn_threshold", 50,
		settings.WithDisplayName("Warn Threshold"),
		settings.WithDescription("Score at or above which IPs are flagged with a warning"),
		settings.WithCategory("IP Reputation"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score at or above which IPs are flagged with a warning (0-100)"),
		settings.WithOrder(20),
	)

	// SettingCacheTTLHours is how long IP reputation results are cached, in hours.
	SettingCacheTTLHours = settings.Define("ipreputation.cache_ttl_hours", 6,
		settings.WithDisplayName("Cache TTL (Hours)"),
		settings.WithDescription("How long IP reputation results are cached"),
		settings.WithCategory("IP Reputation"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("How long IP reputation results are cached, in hours"),
		settings.WithOrder(30),
	)

	// SettingBlockMessage is the error message shown when an IP is blocked.
	SettingBlockMessage = settings.Define("ipreputation.block_message", "access denied due to IP reputation",
		settings.WithDisplayName("Block Message"),
		settings.WithDescription("Error message shown when an IP is blocked"),
		settings.WithCategory("IP Reputation"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Error message shown when an IP is blocked"),
		settings.WithOrder(40),
	)
)

// IPReputation represents the reputation score for an IP address.
type IPReputation struct {
	IP            string   `json:"ip"`
	Score         int      `json:"score"` // 0 (clean) to 100 (malicious)
	Categories    []string `json:"categories,omitempty"`
	ISP           string   `json:"isp,omitempty"`
	ASN           int      `json:"asn,omitempty"`
	IsBlacklisted bool     `json:"is_blacklisted"`
	Source        string   `json:"source"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Provider checks the reputation of an IP address.
type Provider interface {
	CheckIP(ctx context.Context, ip string) (*IPReputation, error)
}

// Config configures the IP reputation plugin.
type Config struct {
	// Provider is the reputation data source.
	Provider Provider

	// BlockThreshold is the score at/above which IPs are blocked (default: 80).
	BlockThreshold int

	// WarnThreshold is the score at/above which IPs are flagged (default: 50).
	WarnThreshold int

	// CacheTTL is how long reputation results are cached (default: 6h).
	CacheTTL time.Duration

	// BlockMessage is the error message on block.
	BlockMessage string
}

func (c *Config) defaults() {
	if c.BlockThreshold == 0 {
		c.BlockThreshold = 80
	}
	if c.WarnThreshold == 0 {
		c.WarnThreshold = 50
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = 6 * time.Hour
	}
	if c.BlockMessage == "" {
		c.BlockMessage = "access denied due to IP reputation"
	}
}

// Plugin checks IP reputation before authentication.
type Plugin struct {
	config      Config
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager

	mu    sync.RWMutex
	cache map[string]*cachedReputation
}

type cachedReputation struct {
	rep       *IPReputation
	expiresAt time.Time
}

// New creates a new IP reputation plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{
		config: c,
		cache:  make(map[string]*cachedReputation),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "ipreputation" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all IP reputation-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "ipreputation", SettingBlockThreshold); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "ipreputation", SettingWarnThreshold); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "ipreputation", SettingCacheTTLHours); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "ipreputation", SettingBlockMessage)
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

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	return nil
}

// OnBeforeSignIn checks the IP reputation.
func (p *Plugin) OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

// OnBeforeSignUp checks the IP reputation.
func (p *Plugin) OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	return p.check(ctx, req.IPAddress, req.AppID.String())
}

func (p *Plugin) check(ctx context.Context, ipAddress, appID string) error {
	if p.config.Provider == nil || ipAddress == "" {
		return nil
	}

	rep := p.getCached(ipAddress)
	if rep == nil {
		var err error
		rep, err = p.config.Provider.CheckIP(ctx, ipAddress)
		if err != nil {
			p.logger.Warn("ipreputation: check failed",
				log.String("ip", ipAddress),
				log.String("error", err.Error()),
			)
			return nil // fail open
		}
		p.setCache(ipAddress, rep)
	}

	if rep.Score >= p.config.BlockThreshold || rep.IsBlacklisted {
		if p.chronicle != nil {
			_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
				Action:   "ipreputation_blocked",
				Resource: "auth",
				Tenant:   appID,
				Outcome:  bridge.OutcomeFailure,
				Severity: bridge.SeverityCritical,
				Metadata: map[string]string{
					"ip":    ipAddress,
					"score": fmt.Sprintf("%d", rep.Score),
				},
			})
		}
		if p.relay != nil {
			_ = p.relay.Send(ctx, &bridge.WebhookEvent{
				Type:     "security.ipreputation_blocked",
				TenantID: appID,
				Data: map[string]string{
					"ip":    ipAddress,
					"score": fmt.Sprintf("%d", rep.Score),
				},
			})
		}
		return fmt.Errorf("ipreputation: %s", p.config.BlockMessage)
	}

	if rep.Score >= p.config.WarnThreshold {
		if p.chronicle != nil {
			_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
				Action:   "ipreputation_warning",
				Resource: "auth",
				Tenant:   appID,
				Outcome:  bridge.OutcomeSuccess,
				Severity: bridge.SeverityWarning,
				Metadata: map[string]string{
					"ip":    ipAddress,
					"score": fmt.Sprintf("%d", rep.Score),
				},
			})
		}
	}

	return nil
}

func (p *Plugin) getCached(ip string) *IPReputation {
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, ok := p.cache[ip]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.rep
}

func (p *Plugin) setCache(ip string, rep *IPReputation) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[ip] = &cachedReputation{
		rep:       rep,
		expiresAt: time.Now().Add(p.config.CacheTTL),
	}
}
