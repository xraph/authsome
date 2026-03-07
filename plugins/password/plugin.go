package password

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.BeforeSignUp          = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.AuthMethodContributor = (*Plugin)(nil)
	_ plugin.SettingsProvider      = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingMinLength controls the minimum password length.
	SettingMinLength = settings.Define("password.min_length", 8,
		settings.WithDisplayName("Minimum Password Length"),
		settings.WithDescription("Minimum number of characters required for passwords"),
		settings.WithCategory("Password Policy"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(4), Max: intPtr(128)}),
		settings.WithHelpText("Minimum number of characters required. Default: 8"),
		settings.WithOrder(10),
	)

	// SettingRequireSpecial controls whether special characters are required.
	SettingRequireSpecial = settings.Define("password.require_special", false,
		settings.WithDisplayName("Require Special Character"),
		settings.WithDescription("Require at least one special character in passwords"),
		settings.WithCategory("Password Policy"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, passwords must contain at least one special character"),
		settings.WithOrder(20),
	)

	// SettingAllowedDomains restricts signup to specific email domains.
	SettingAllowedDomains = settings.Define("password.allowed_domains", "",
		settings.WithDisplayName("Allowed Email Domains"),
		settings.WithDescription("Comma-separated list of allowed email domains for signup (empty = all allowed)"),
		settings.WithCategory("Password Policy"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldTextarea),
		settings.WithPlaceholder("example.com, mycompany.org"),
		settings.WithHelpText("Leave empty to allow all domains. Separate multiple domains with commas."),
		settings.WithOrder(30),
	)
)

func intPtr(v int) *int { return &v }

// Config configures the password plugin.
type Config struct {
	// MinLength overrides the engine password policy minimum length.
	// Set to 0 to use the engine default.
	MinLength int

	// RequireSpecial requires at least one special character.
	RequireSpecial bool

	// AllowedDomains restricts signup to specific email domains.
	// Empty means all domains are allowed.
	AllowedDomains []string
}

// Plugin is the password authentication plugin.
type Plugin struct {
	config Config
	store  store.Store
}

// New creates a new password plugin with the given configuration.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "password" }

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "password", SettingMinLength); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "password", SettingRequireSpecial); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "password", SettingAllowedDomains)
}

// OnBeforeSignUp validates the signup request against the password plugin's rules.
func (p *Plugin) OnBeforeSignUp(_ context.Context, req *account.SignUpRequest) error {
	// Domain restriction check
	if len(p.config.AllowedDomains) > 0 {
		domain := emailDomain(req.Email)
		if !containsString(p.config.AllowedDomains, domain) {
			return fmt.Errorf("email domain %q is not allowed", domain)
		}
	}
	return nil
}

// RegisterRoutes is a no-op for the password plugin since the core signup/signin
// routes are already registered by the API handler.
func (p *Plugin) RegisterRoutes(_ any) error {
	return nil
}

// Strategy returns the password authentication strategy.
func (p *Plugin) Strategy() strategy.Strategy {
	return &passwordStrategy{}
}

// OnInit captures the store from the engine.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type storeGetter interface {
		Store() store.Store
	}
	if sg, ok := engine.(storeGetter); ok {
		p.store = sg.Store()
	}
	return nil
}

// ListUserAuthMethods implements plugin.AuthMethodContributor.
// It reports a "password" method if the user has a non-empty password hash.
func (p *Plugin) ListUserAuthMethods(ctx context.Context, userID id.UserID) ([]*plugin.AuthMethod, error) {
	if p.store == nil {
		return nil, nil
	}
	u, err := p.store.GetUser(ctx, userID)
	if err != nil {
		return nil, nil // user not found; no methods to report
	}
	if u.PasswordHash == "" {
		return nil, nil
	}
	return []*plugin.AuthMethod{{
		Type:     "password",
		Provider: "password",
		Label:    "Password",
		LinkedAt: u.CreatedAt.Format(time.RFC3339),
	}}, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func emailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return ""
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
