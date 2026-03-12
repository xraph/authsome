package email

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.AfterSignUp      = (*Plugin)(nil)
	_ plugin.AfterUserCreate  = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingFromAddress controls the default sender email address.
	SettingFromAddress = settings.Define("email.from_address", "noreply@authsome.local",
		settings.WithDisplayName("From Address"),
		settings.WithDescription("Default sender email address for outgoing emails"),
		settings.WithCategory("Email"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldEmail),
		settings.WithPlaceholder("noreply@example.com"),
		settings.WithHelpText("The sender address shown in outgoing emails"),
		settings.WithOrder(10),
	)

	// SettingAppName controls the application name used in email templates.
	SettingAppName = settings.Define("email.app_name", "AuthSome",
		settings.WithDisplayName("Application Name"),
		settings.WithDescription("Application name used in email subjects and bodies"),
		settings.WithCategory("Email"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithPlaceholder("My App"),
		settings.WithHelpText("Used in email templates for branding"),
		settings.WithOrder(20),
	)

	// SettingBaseURL controls the application root URL for email links.
	SettingBaseURL = settings.Define("email.base_url", "",
		settings.WithDisplayName("Base URL"),
		settings.WithDescription("Application root URL for building links in emails"),
		settings.WithCategory("Email"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldURL),
		settings.WithPlaceholder("https://example.com"),
		settings.WithHelpText("Used to build verification and action links in emails"),
		settings.WithOrder(30),
	)
)

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "email", SettingFromAddress); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "email", SettingAppName); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "email", SettingBaseURL)
}

// Config configures the email notification plugin.
type Config struct {
	// From is the default sender email address (e.g. "noreply@example.com").
	From string

	// AppName is used in email subjects and bodies (e.g. "My App").
	AppName string

	// BaseURL is the application root URL for building links in emails
	// (e.g. "https://example.com").
	BaseURL string
}

// Plugin is the email notification plugin.
type Plugin struct {
	config Config
	mailer bridge.Mailer
	logger log.Logger
}

// New creates a new email plugin with the given configuration.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.AppName == "" {
		c.AppName = "AuthSome"
	}
	if c.From == "" {
		c.From = "noreply@authsome.local"
	}
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "email" }

// OnInit extracts the mailer bridge from the engine during initialization.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type mailerGetter interface {
		Mailer() bridge.Mailer
	}
	if mg, ok := engine.(mailerGetter); ok {
		p.mailer = mg.Mailer()
	}

	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	return nil
}

// SetMailer allows direct mailer injection for testing.
func (p *Plugin) SetMailer(m bridge.Mailer) {
	p.mailer = m
}

// OnAfterSignUp sends a welcome email to the newly registered user.
func (p *Plugin) OnAfterSignUp(ctx context.Context, u *user.User, _ *session.Session) error {
	if p.mailer == nil {
		return nil
	}

	name := u.Name()
	if name == "" {
		name = u.Email
	}

	subject, html, text := WelcomeEmail(name, p.config.AppName)

	if err := p.mailer.SendEmail(ctx, &bridge.EmailMessage{
		To:      []string{u.Email},
		From:    p.config.From,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		p.logger.Warn("email plugin: failed to send welcome email",
			log.String("email", u.Email),
			log.String("error", err.Error()),
		)
	}

	return nil
}

// OnAfterUserCreate sends a verification email to the newly created user.
func (p *Plugin) OnAfterUserCreate(ctx context.Context, u *user.User) error {
	if p.mailer == nil {
		return nil
	}

	// Only send if user has not yet verified their email
	if u.EmailVerified {
		return nil
	}

	name := u.Name()
	if name == "" {
		name = u.Email
	}

	verifyURL := p.config.BaseURL + "/verify-email"
	subject, html, text := VerificationEmail(name, p.config.AppName, verifyURL)

	if err := p.mailer.SendEmail(ctx, &bridge.EmailMessage{
		To:      []string{u.Email},
		From:    p.config.From,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		p.logger.Warn("email plugin: failed to send verification email",
			log.String("email", u.Email),
			log.String("error", err.Error()),
		)
	}

	return nil
}

// SendPasswordReset sends a password reset email. This is typically called
// by the engine's password reset flow.
func (p *Plugin) SendPasswordReset(ctx context.Context, email, name, resetURL string) error {
	if p.mailer == nil {
		return bridge.ErrMailerNotAvailable
	}

	if name == "" {
		name = email
	}

	subject, html, text := PasswordResetEmail(name, p.config.AppName, resetURL)

	return p.mailer.SendEmail(ctx, &bridge.EmailMessage{
		To:      []string{email},
		From:    p.config.From,
		Subject: subject,
		HTML:    html,
		Text:    text,
	})
}

// SendInvitation sends an organization invitation email.
func (p *Plugin) SendInvitation(ctx context.Context, email, inviterName, orgName, acceptURL string) error {
	if p.mailer == nil {
		return bridge.ErrMailerNotAvailable
	}

	subject, html, text := InvitationEmail(inviterName, orgName, p.config.AppName, acceptURL)

	return p.mailer.SendEmail(ctx, &bridge.EmailMessage{
		To:      []string{email},
		From:    p.config.From,
		Subject: subject,
		HTML:    html,
		Text:    text,
	})
}
