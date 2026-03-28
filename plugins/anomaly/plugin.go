// Package anomaly provides login anomaly detection by analyzing patterns
// such as unusual login times, new countries, frequency spikes, and new
// device types.
package anomaly

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/geoip"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.AfterSignIn      = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingMinLoginHistory controls the minimum number of logins before
	// anomaly detection activates.
	SettingMinLoginHistory = settings.Define("anomaly.min_login_history", 10,
		settings.WithDisplayName("Minimum Login History"),
		settings.WithDescription("Minimum number of logins before anomaly detection kicks in"),
		settings.WithCategory("Anomaly Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Minimum number of logins before anomaly detection kicks in"),
		settings.WithOrder(10),
	)

	// SettingRiskThreshold controls the score above which an anomaly alert
	// is raised.
	SettingRiskThreshold = settings.Define("anomaly.risk_threshold", 70,
		settings.WithDisplayName("Risk Threshold"),
		settings.WithDescription("Score above which an anomaly alert is raised (0-100)"),
		settings.WithCategory("Anomaly Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score above which an anomaly alert is raised (0-100)"),
		settings.WithOrder(20),
	)

	// SettingEnableTimeAnomaly controls whether unusual login time detection
	// is enabled.
	SettingEnableTimeAnomaly = settings.Define("anomaly.enable_time_anomaly", true,
		settings.WithDisplayName("Enable Time Anomaly"),
		settings.WithDescription("Enable detection of unusual login times"),
		settings.WithCategory("Anomaly Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Enable detection of unusual login times"),
		settings.WithOrder(30),
	)

	// SettingEnableGeoAnomaly controls whether new country detection is
	// enabled.
	SettingEnableGeoAnomaly = settings.Define("anomaly.enable_geo_anomaly", true,
		settings.WithDisplayName("Enable Geo Anomaly"),
		settings.WithDescription("Enable detection of logins from new countries"),
		settings.WithCategory("Anomaly Detection"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Enable detection of logins from new countries"),
		settings.WithOrder(40),
	)
)

func intPtr(i int) *int { return &i }

// Config configures the anomaly detection plugin.
type Config struct {
	// MinLoginHistory is the minimum number of logins before anomaly
	// detection kicks in (default: 10).
	MinLoginHistory int

	// RiskThreshold is the score above which an anomaly alert is raised
	// (default: 70 out of 100).
	RiskThreshold int

	// EnableTimeAnomaly enables unusual login time detection (default: true).
	EnableTimeAnomaly bool

	// EnableGeoAnomaly enables new country detection (default: true).
	EnableGeoAnomaly bool
}

func (c *Config) defaults() {
	if c.MinLoginHistory == 0 {
		c.MinLoginHistory = 10
	}
	if c.RiskThreshold == 0 {
		c.RiskThreshold = 70
	}
}

// LoginPattern tracks a user's typical login behavior.
type LoginPattern struct {
	UserID           id.UserID
	LoginCount       int
	HourHistogram    [24]int        // login count by hour of day
	DayHistogram     [7]int         // login count by day of week
	CountryHistogram map[string]int // login count by country
	LastLoginAt      time.Time
}

// Alert is emitted when an anomalous login is detected.
type Alert struct {
	UserID    id.UserID
	SessionID id.SessionID
	Type      string // "unusual_time", "unusual_country", "frequency_spike"
	RiskScore int
	Details   map[string]string
}

// Plugin analyzes login patterns and detects anomalies.
type Plugin struct {
	config      Config
	geoIP       *geoip.Plugin
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager

	mu       sync.RWMutex
	patterns map[string]*LoginPattern // keyed by user ID
}

// New creates a new anomaly detection plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	// Default enables.
	if !c.EnableTimeAnomaly && !c.EnableGeoAnomaly {
		c.EnableTimeAnomaly = true
		c.EnableGeoAnomaly = true
	}
	return &Plugin{
		config:   c,
		patterns: make(map[string]*LoginPattern),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "anomaly" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all anomaly-detection-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "anomaly", SettingMinLoginHistory); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "anomaly", SettingRiskThreshold); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "anomaly", SettingEnableTimeAnomaly); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "anomaly", SettingEnableGeoAnomaly)
}

// OnInit discovers engine dependencies.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	p.chronicle = engine.Chronicle()
	p.relay = engine.Relay()
	p.settingsMgr = engine.Settings()

	if gp := engine.Plugin("geoip"); gp != nil {
		if geoPlugin, ok := gp.(*geoip.Plugin); ok {
			p.geoIP = geoPlugin
		}
	}

	return nil
}

// OnAfterSignIn records the login and checks for anomalies.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	now := time.Now()
	userKey := u.ID.String()

	p.mu.Lock()
	pattern, exists := p.patterns[userKey]
	if !exists {
		pattern = &LoginPattern{
			UserID:           u.ID,
			CountryHistogram: make(map[string]int),
		}
		p.patterns[userKey] = pattern
	}

	// Record this login.
	pattern.LoginCount++
	pattern.HourHistogram[now.Hour()]++
	pattern.DayHistogram[now.Weekday()]++
	pattern.LastLoginAt = now

	// Resolve country.
	var country string
	if p.geoIP != nil && s.IPAddress != "" {
		if loc := p.geoIP.Resolve(s.IPAddress); loc != nil {
			country = loc.Country
			pattern.CountryHistogram[country]++
		}
	}

	// Take snapshot for analysis (avoid holding lock during IO).
	snapshot := *pattern
	snapshot.CountryHistogram = make(map[string]int, len(pattern.CountryHistogram))
	for k, v := range pattern.CountryHistogram {
		snapshot.CountryHistogram[k] = v
	}
	p.mu.Unlock()

	// Skip anomaly checks until we have enough history.
	if snapshot.LoginCount < p.config.MinLoginHistory {
		return nil
	}

	// Check anomalies.
	var alerts []*Alert

	if p.config.EnableTimeAnomaly {
		if alert := p.checkTimeAnomaly(u.ID, s.ID, now, &snapshot); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	if p.config.EnableGeoAnomaly && country != "" {
		if alert := p.checkGeoAnomaly(u.ID, s.ID, country, &snapshot); alert != nil {
			alerts = append(alerts, alert)
		}
	}

	for _, alert := range alerts {
		if alert.RiskScore >= p.config.RiskThreshold {
			p.handleAlert(ctx, u.AppID, alert)
		}
	}

	return nil
}

func (p *Plugin) checkTimeAnomaly(userID id.UserID, sessID id.SessionID, now time.Time, pattern *LoginPattern) *Alert {
	hour := now.Hour()
	hourCount := pattern.HourHistogram[hour]
	totalLogins := pattern.LoginCount

	// Calculate percentage of logins at this hour.
	hourPct := float64(hourCount) / float64(totalLogins) * 100

	// If less than 2% of logins happen at this hour, flag it.
	if hourPct < 2.0 && totalLogins > p.config.MinLoginHistory {
		score := int(100 - hourPct*50) // 0% → 100 score, 2% → 0 score
		if score > 100 {
			score = 100
		}
		return &Alert{
			UserID:    userID,
			SessionID: sessID,
			Type:      "unusual_time",
			RiskScore: score,
			Details: map[string]string{
				"hour":         fmt.Sprintf("%d", hour),
				"hour_pct":     fmt.Sprintf("%.1f", hourPct),
				"total_logins": fmt.Sprintf("%d", totalLogins),
			},
		}
	}
	return nil
}

func (p *Plugin) checkGeoAnomaly(userID id.UserID, sessID id.SessionID, country string, pattern *LoginPattern) *Alert {
	countryCount := pattern.CountryHistogram[country]

	// First time from this country (and user has history).
	if countryCount <= 1 {
		return &Alert{
			UserID:    userID,
			SessionID: sessID,
			Type:      "unusual_country",
			RiskScore: 85,
			Details: map[string]string{
				"country":      country,
				"total_logins": fmt.Sprintf("%d", pattern.LoginCount),
			},
		}
	}
	return nil
}

func (p *Plugin) handleAlert(ctx context.Context, appID id.AppID, alert *Alert) {
	p.logger.Warn("anomaly: login anomaly detected",
		log.String("user_id", alert.UserID.String()),
		log.String("type", alert.Type),
	)

	if p.chronicle != nil {
		metadata := map[string]string{
			"anomaly_type": alert.Type,
			"risk_score":   fmt.Sprintf("%d", alert.RiskScore),
		}
		for k, v := range alert.Details {
			metadata[k] = v
		}
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
			Action:     "login_anomaly",
			Resource:   "session",
			ResourceID: alert.SessionID.String(),
			ActorID:    alert.UserID.String(),
			Tenant:     appID.String(),
			Outcome:    bridge.OutcomeSuccess,
			Severity:   bridge.SeverityWarning,
			Metadata:   metadata,
		})
	}

	if p.relay != nil {
		data := map[string]string{
			"user_id":      alert.UserID.String(),
			"session_id":   alert.SessionID.String(),
			"anomaly_type": alert.Type,
			"risk_score":   fmt.Sprintf("%d", alert.RiskScore),
		}
		for k, v := range alert.Details {
			data[k] = v
		}
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
			Type:     "security.login_anomaly",
			TenantID: appID.String(),
			Data:     data,
		})
	}
}
