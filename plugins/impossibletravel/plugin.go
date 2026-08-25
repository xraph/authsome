// Package impossibletravel detects geographically impossible logins based on
// the time and distance between consecutive user logins.
package impossibletravel

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
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin             = (*Plugin)(nil)
	_ plugin.OnInit             = (*Plugin)(nil)
	_ plugin.AfterSignIn        = (*Plugin)(nil)
	_ plugin.AfterPrincipalAuth = (*Plugin)(nil)
	_ plugin.SettingsProvider   = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingMaxSpeedKmH is the maximum plausible travel speed in km/h.
	SettingMaxSpeedKmH = settings.Define("impossibletravel.max_speed_kmh", float64(900),
		settings.WithDisplayName("Max Speed (km/h)"),
		settings.WithDescription("Maximum plausible travel speed in km/h"),
		settings.WithCategory("Impossible Travel"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Maximum plausible travel speed in km/h. Logins exceeding this are flagged."),
		settings.WithOrder(10),
	)

	// SettingMinDistanceKm is the minimum distance in km between logins to trigger a check.
	SettingMinDistanceKm = settings.Define("impossibletravel.min_distance_km", float64(500),
		settings.WithDisplayName("Min Distance (km)"),
		settings.WithDescription("Minimum distance in km between logins to trigger check"),
		settings.WithCategory("Impossible Travel"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Minimum distance in km between logins to trigger check"),
		settings.WithOrder(20),
	)

	// SettingLookbackWindowHours is how far back to look for previous logins.
	SettingLookbackWindowHours = settings.Define("impossibletravel.lookback_window_hours", 24,
		settings.WithDisplayName("Lookback Window (hours)"),
		settings.WithDescription("How far back to look for previous logins, in hours"),
		settings.WithCategory("Impossible Travel"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("How far back to look for previous logins, in hours"),
		settings.WithOrder(30),
	)

	// SettingAction is the action to take when impossible travel is detected.
	SettingAction = settings.Define("impossibletravel.action", "flag",
		settings.WithDisplayName("Detection Action"),
		settings.WithDescription("Action to take when impossible travel is detected"),
		settings.WithCategory("Impossible Travel"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Action to take when impossible travel is detected"),
		settings.WithOrder(40),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Label: "Flag Only", Value: "flag"},
			formconfig.SelectOption{Label: "Block", Value: "block"},
		),
	)
)

// Config configures the impossible travel detection plugin.
type Config struct {
	// MaxSpeedKmH is the maximum realistic travel speed (default: 900 km/h = airplane).
	MaxSpeedKmH float64

	// MinDistanceKm is the minimum distance to trigger a check (default: 500 km).
	MinDistanceKm float64

	// LookbackWindow is how far back to check login history (default: 24h).
	LookbackWindow time.Duration

	// Action is what to do on detection: "flag" (default), "block".
	Action string
}

func (c *Config) defaults() {
	if c.MaxSpeedKmH == 0 {
		c.MaxSpeedKmH = 900
	}
	if c.MinDistanceKm == 0 {
		c.MinDistanceKm = 500
	}
	if c.LookbackWindow == 0 {
		c.LookbackWindow = 24 * time.Hour
	}
	if c.Action == "" {
		c.Action = "flag"
	}
}

// LoginLocation records a login position for travel calculation.
type LoginLocation struct {
	// Principal is the caller this location belongs to: a user for sign-in,
	// an agent or workload for machine traffic.
	Principal principal.Ref
	IP        string
	Country   string
	City      string
	Latitude  float64
	Longitude float64
	LoginAt   time.Time
}

// TravelAlert is emitted when impossible travel is detected.
type TravelAlert struct {
	Principal       principal.Ref
	SessionID       id.SessionID
	FromLocation    LoginLocation
	ToLocation      LoginLocation
	DistanceKm      float64
	TimeDeltaMin    float64
	RequiredSpeedKm float64
	RiskLevel       string
}

// Plugin detects impossible travel between consecutive logins.
type Plugin struct {
	config      Config
	geoIP       *geoip.Plugin
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager

	// In-memory last-login cache, keyed by principal.Ref.String() rather
	// than by user ID, so a machine caller gets its own travel history
	// instead of sharing (or worse, colliding with) another principal's.
	mu         sync.RWMutex
	lastLogins map[string]*LoginLocation

	// events records the alerts raised so far, for tests to assert against.
	events []TravelAlert
}

// New creates a new impossible travel detection plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{
		config:     c,
		lastLogins: make(map[string]*LoginLocation),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "impossibletravel" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all impossible-travel-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "impossibletravel", SettingMaxSpeedKmH); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "impossibletravel", SettingMinDistanceKm); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "impossibletravel", SettingLookbackWindowHours); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "impossibletravel", SettingAction)
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

// OnAfterSignIn records a person's login location and checks for impossible
// travel.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	return p.recordLocation(ctx, principal.UserRef(u.ID), u.AppID.String(), s.ID, s.IPAddress)
}

// OnAfterPrincipalAuth records a machine caller's login location and checks
// for impossible travel.
//
// Keyed by principal rather than user. An agent has no user, and two agents
// sharing one travel history would score each other's movements as travel,
// which on a fleet spread across regions means a permanent false positive.
func (p *Plugin) OnAfterPrincipalAuth(ctx context.Context, a *principal.AuthAttempt, s *session.Session) error {
	ip := a.IPAddress
	var sessionID id.SessionID
	if s != nil {
		if ip == "" {
			ip = s.IPAddress
		}
		sessionID = s.ID
	}
	return p.recordLocation(ctx, a.Subject, a.AppID.String(), sessionID, ip)
}

// recordLocation is the body shared by OnAfterSignIn and
// OnAfterPrincipalAuth: it records ref's login location and checks it
// against ref's own last known location for impossible travel.
func (p *Plugin) recordLocation(ctx context.Context, ref principal.Ref, appID string, sessionID id.SessionID, ipAddress string) error {
	if p.geoIP == nil || ipAddress == "" {
		return nil
	}

	loc := p.geoIP.Resolve(ipAddress)
	if loc == nil || (loc.Latitude == 0 && loc.Longitude == 0) {
		return nil
	}

	current := &LoginLocation{
		Principal: ref,
		IP:        ipAddress,
		Country:   loc.Country,
		City:      loc.City,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		LoginAt:   time.Now(),
	}

	// Check against last login.
	refKey := ref.String()
	p.mu.RLock()
	prev := p.lastLogins[refKey]
	p.mu.RUnlock()

	if prev != nil {
		timeDelta := current.LoginAt.Sub(prev.LoginAt)

		// Only check within lookback window.
		if timeDelta <= p.config.LookbackWindow && timeDelta > 0 {
			distance := geoip.Haversine(prev.Latitude, prev.Longitude, current.Latitude, current.Longitude)

			if distance >= p.config.MinDistanceKm {
				timeDeltaHours := timeDelta.Hours()
				requiredSpeed := distance / timeDeltaHours

				if requiredSpeed > p.config.MaxSpeedKmH {
					alert := &TravelAlert{
						Principal:       ref,
						SessionID:       sessionID,
						FromLocation:    *prev,
						ToLocation:      *current,
						DistanceKm:      distance,
						TimeDeltaMin:    timeDelta.Minutes(),
						RequiredSpeedKm: requiredSpeed,
						RiskLevel:       riskLevel(requiredSpeed, p.config.MaxSpeedKmH),
					}
					p.handleAlert(ctx, appID, alert)
				}
			}
		}
	}

	// Update last login.
	p.mu.Lock()
	p.lastLogins[refKey] = current
	p.mu.Unlock()

	return nil
}

// RecordedEvents returns the impossible-travel alerts raised so far, for
// tests to assert against.
func (p *Plugin) RecordedEvents() []TravelAlert {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]TravelAlert, len(p.events))
	copy(out, p.events)
	return out
}

func riskLevel(speed, threshold float64) string {
	ratio := speed / threshold
	switch {
	case ratio > 5:
		return "critical"
	case ratio > 2:
		return "high"
	default:
		return "medium"
	}
}

func (p *Plugin) handleAlert(ctx context.Context, appID string, alert *TravelAlert) {
	p.logger.Warn("impossibletravel: alert detected",
		log.String("principal", alert.Principal.String()),
		log.String("from", fmt.Sprintf("%s, %s", alert.FromLocation.City, alert.FromLocation.Country)),
		log.String("to", fmt.Sprintf("%s, %s", alert.ToLocation.City, alert.ToLocation.Country)),
		log.String("risk", alert.RiskLevel),
	)

	p.mu.Lock()
	p.events = append(p.events, *alert)
	p.mu.Unlock()

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
			Action:     "impossible_travel",
			Resource:   "session",
			ResourceID: alert.SessionID.String(),
			ActorID:    alert.Principal.String(),
			Tenant:     appID,
			Outcome:    bridge.OutcomeFailure,
			Severity:   bridge.SeverityCritical,
			Metadata: map[string]string{
				"from_country": alert.FromLocation.Country,
				"from_city":    alert.FromLocation.City,
				"to_country":   alert.ToLocation.Country,
				"to_city":      alert.ToLocation.City,
				"distance_km":  fmt.Sprintf("%.0f", alert.DistanceKm),
				"time_min":     fmt.Sprintf("%.0f", alert.TimeDeltaMin),
				"speed_kmh":    fmt.Sprintf("%.0f", alert.RequiredSpeedKm),
				"risk_level":   alert.RiskLevel,
			},
		})
	}

	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
			Type:     "security.impossible_travel",
			TenantID: appID,
			Data: map[string]string{
				"principal":    alert.Principal.String(),
				"session_id":   alert.SessionID.String(),
				"from_country": alert.FromLocation.Country,
				"to_country":   alert.ToLocation.Country,
				"distance_km":  fmt.Sprintf("%.0f", alert.DistanceKm),
				"risk_level":   alert.RiskLevel,
			},
		})
	}
}
