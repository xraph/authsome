// Package deviceverify provides new device detection and challenge-response
// verification. When a user signs in from an unrecognized device, the plugin
// can flag it or require verification via email/SMS.
package deviceverify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
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
	// SettingNotifyOnNewDevice controls whether a notification is sent when
	// a new device is detected.
	SettingNotifyOnNewDevice = settings.Define("deviceverify.notify_on_new_device", true,
		settings.WithDisplayName("Notify on New Device"),
		settings.WithDescription("Send a notification when a new device is detected"),
		settings.WithCategory("Device Verification"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Send a notification when a new device is detected"),
		settings.WithOrder(10),
	)

	// SettingChallengeTTLMinutes controls how long a device verification
	// challenge is valid, in minutes.
	SettingChallengeTTLMinutes = settings.Define("deviceverify.challenge_ttl_minutes", 10,
		settings.WithDisplayName("Challenge TTL (minutes)"),
		settings.WithDescription("How long a device verification challenge is valid, in minutes"),
		settings.WithCategory("Device Verification"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("How long a device verification challenge is valid, in minutes"),
		settings.WithOrder(20),
	)
)

// Config configures the device verification plugin.
type Config struct {
	// NotifyOnNewDevice sends a notification when a new device is detected
	// (default: true).
	NotifyOnNewDevice bool

	// ChallengeTTL is how long a device verification challenge is valid
	// (default: 10m).
	ChallengeTTL time.Duration
}

func (c *Config) defaults() {
	if c.ChallengeTTL == 0 {
		c.ChallengeTTL = 10 * time.Minute
	}
}

// Plugin detects new devices on sign-in and optionally sends notifications.
type Plugin struct {
	config      Config
	store       store.Store
	herald      bridge.Herald
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	logger      log.Logger
	settingsMgr *settings.Manager
}

// New creates a new device verification plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	} else {
		c.NotifyOnNewDevice = true
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "deviceverify" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all device-verification-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "deviceverify", SettingNotifyOnNewDevice); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "deviceverify", SettingChallengeTTLMinutes)
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

	type storeGetter interface{ Store() store.Store }
	if sg, ok := engine.(storeGetter); ok {
		p.store = sg.Store()
	}

	type heraldGetter interface{ Herald() bridge.Herald }
	if hg, ok := engine.(heraldGetter); ok {
		p.herald = hg.Herald()
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

// OnAfterSignIn checks if the device used for sign-in is new/unrecognized.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	if p.store == nil {
		return nil
	}

	// If no device was bound to the session, nothing to verify.
	if s.DeviceID.Prefix() == "" {
		return nil
	}

	dev, err := p.store.GetDevice(ctx, s.DeviceID)
	if err != nil {
		return nil // can't verify without the device record
	}

	// A device is "new" if it was just created (CreatedAt ≈ now).
	if time.Since(dev.CreatedAt) > 30*time.Second {
		// Device existed before — not new.
		return nil
	}

	p.handleNewDevice(ctx, u, s, dev)
	return nil
}

func (p *Plugin) handleNewDevice(ctx context.Context, u *user.User, s *session.Session, dev *device.Device) {
	p.logger.Info("deviceverify: new device detected",
		log.String("user_id", u.ID.String()),
		log.String("device_id", dev.ID.String()),
		log.String("browser", dev.Browser),
		log.String("os", dev.OS),
	)

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
			Action:     "new_device_detected",
			Resource:   "device",
			ResourceID: dev.ID.String(),
			ActorID:    u.ID.String(),
			Tenant:     u.AppID.String(),
			Outcome:    bridge.OutcomeSuccess,
			Severity:   bridge.SeverityInfo,
			Metadata: map[string]string{
				"browser":    dev.Browser,
				"os":         dev.OS,
				"ip_address": dev.IPAddress,
				"type":       dev.Type,
			},
		})
	}

	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{
			Type:     "device.new_detected",
			TenantID: u.AppID.String(),
			Data: map[string]string{
				"user_id":   u.ID.String(),
				"device_id": dev.ID.String(),
				"browser":   dev.Browser,
				"os":        dev.OS,
			},
		})
	}

	// Send notification to user about the new device.
	if p.config.NotifyOnNewDevice && p.herald != nil {
		name := u.Name()
		if name == "" {
			name = u.Email
		}
		_ = p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
			Template: "new-device-login",
			Channels: []string{"email"},
			To:       []string{u.Email},
			UserID:   u.ID.String(),
			Locale:   "en",
			Async:    true,
			Data: map[string]any{
				"user_name":  name,
				"browser":    dev.Browser,
				"os":         dev.OS,
				"ip_address": dev.IPAddress,
				"device_type": dev.Type,
				"login_time": time.Now().Format(time.RFC1123),
			},
		})
	}
}

// GenerateToken creates a random hex token for device challenges.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Ensure unused imports are used.
var _ = id.NewDeviceID
