package magiclink

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// VerificationTypeMagicLink is the verification type used for magic link tokens.
const VerificationTypeMagicLink account.VerificationType = "magic_link"

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.RouteProvider    = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingTokenTTLSeconds controls the magic link token lifetime in seconds.
	SettingTokenTTLSeconds = settings.Define("magiclink.token_ttl_seconds", 600,
		settings.WithDisplayName("Token TTL (seconds)"),
		settings.WithDescription("Lifetime of magic link tokens in seconds"),
		settings.WithCategory("Magic Link"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(60), Max: intPtr(3600)}),
		settings.WithHelpText("How long magic link tokens remain valid. Default: 600 (10 minutes)"),
		settings.WithOrder(10),
	)

	// SettingSessionTokenTTLSeconds controls the session token lifetime for magic link sessions.
	SettingSessionTokenTTLSeconds = settings.Define("magiclink.session_token_ttl_seconds", 3600,
		settings.WithDisplayName("Session Token TTL (seconds)"),
		settings.WithDescription("Lifetime of sessions created via magic link in seconds"),
		settings.WithCategory("Magic Link"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(300), Max: intPtr(86400)}),
		settings.WithHelpText("How long sessions created via magic link remain valid. Default: 3600 (1 hour)"),
		settings.WithOrder(20),
	)

	// SettingSessionRefreshTTLSeconds controls the refresh token lifetime for magic link sessions.
	SettingSessionRefreshTTLSeconds = settings.Define("magiclink.session_refresh_ttl_seconds", 2592000,
		settings.WithDisplayName("Refresh Token TTL (seconds)"),
		settings.WithDescription("Lifetime of refresh tokens for magic link sessions in seconds"),
		settings.WithCategory("Magic Link"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(3600), Max: intPtr(7776000)}),
		settings.WithHelpText("How long refresh tokens remain valid. Default: 2592000 (30 days)"),
		settings.WithOrder(30),
	)
)

func intPtr(v int) *int { return &v }

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "magiclink", SettingTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "magiclink", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "magiclink", SettingSessionRefreshTTLSeconds)
}

// Mailer sends magic link emails.
type Mailer interface {
	SendMagicLink(ctx context.Context, email, token string) error
}

// MailerFunc is an adapter to use a plain function as a Mailer.
type MailerFunc func(ctx context.Context, email, token string) error

// SendMagicLink implements Mailer.
func (f MailerFunc) SendMagicLink(ctx context.Context, email, token string) error {
	return f(ctx, email, token)
}

// Config configures the magic link plugin.
type Config struct {
	// Mailer sends magic link emails. Required.
	Mailer Mailer

	// TokenTTL is the lifetime of magic link tokens (default: 10 minutes).
	TokenTTL time.Duration

	// SessionTokenTTL is the lifetime of sessions created via magic link (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens for magic link sessions (default: 30 days).
	SessionRefreshTTL time.Duration
}

// sessionConfigResolver resolves per-app session configuration.
type sessionConfigResolver interface {
	SessionConfigForApp(ctx context.Context, appID id.AppID) account.SessionConfig
}

// Plugin is the magic link authentication plugin.
type Plugin struct {
	config        Config
	store         store.Store
	appID         string
	sessionConfig sessionConfigResolver
}

// New creates a new magic link plugin.
func New(cfg Config) *Plugin {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 10 * time.Minute
	}
	if cfg.SessionTokenTTL == 0 {
		cfg.SessionTokenTTL = 1 * time.Hour
	}
	if cfg.SessionRefreshTTL == 0 {
		cfg.SessionRefreshTTL = 30 * 24 * time.Hour
	}
	return &Plugin{config: cfg}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "magiclink" }

// OnInit captures the store reference from the engine.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type storeGetter interface {
		Store() store.Store
	}
	if sg, ok := engine.(storeGetter); ok {
		p.store = sg.Store()
	}
	if scr, ok := engine.(sessionConfigResolver); ok {
		p.sessionConfig = scr
	}
	return nil
}

// RegisterRoutes registers magic link HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("magiclink: expected forge.Router, got %T", r)
	}

	g := router.Group("/v1/auth/magic-link", forge.WithGroupTags("Magic Link"))

	if err := g.POST("/send", p.handleSend,
		forge.WithSummary("Send magic link"),
		forge.WithOperationID("sendMagicLink"),
		forge.WithResponseSchema(http.StatusOK, "Magic link sent", SendResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/verify", p.handleVerify,
		forge.WithSummary("Verify magic link"),
		forge.WithOperationID("verifyMagicLink"),
		forge.WithResponseSchema(http.StatusOK, "Verified", VerifyResponse{}),
		forge.WithErrorResponses(),
	)
}

// SetStore allows direct store injection for testing.
func (p *Plugin) SetStore(s store.Store) {
	p.store = s
}

// SetAppID sets the app ID for the plugin (used in testing).
func (p *Plugin) SetAppID(appID string) {
	p.appID = appID
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// SendRequest is the request body for sending a magic link.
type SendRequest struct {
	Email string `json:"email"`
	AppID string `json:"app_id,omitempty"`
}

// SendResponse is returned when a magic link is sent.
type SendResponse struct {
	Status string `json:"status"`
}

// VerifyRequest is the request body for verifying a magic link.
type VerifyRequest struct {
	Token string `json:"token"`
	AppID string `json:"app_id,omitempty"`
}

// VerifyResponse is returned when a magic link is verified.
type VerifyResponse struct {
	User         any    `json:"user"`
	SessionToken string `json:"session_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    any    `json:"expires_at"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleSend(ctx forge.Context, req *SendRequest) (*SendResponse, error) {
	if req.Email == "" {
		return nil, forge.BadRequest("email required")
	}

	appIDStr := req.AppID
	if appIDStr == "" {
		appIDStr = p.appID
	}
	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	// Create verification token
	v, err := account.NewVerification(ctx.Context(), appID, id.NewUserID(), VerificationTypeMagicLink, p.config.TokenTTL)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create magic link token: %w", err))
	}

	if err := p.store.CreateVerification(ctx.Context(), v); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create magic link: %w", err))
	}

	// Send email
	if p.config.Mailer != nil {
		if err := p.config.Mailer.SendMagicLink(ctx.Context(), req.Email, v.Token); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to send magic link email: %w", err))
		}
	}

	return &SendResponse{Status: "magic link sent"}, nil
}

func (p *Plugin) handleVerify(ctx forge.Context, req *VerifyRequest) (*VerifyResponse, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token required")
	}

	// Look up verification
	v, err := p.store.GetVerification(ctx.Context(), req.Token)
	if err != nil {
		return nil, forge.Unauthorized("invalid or expired magic link")
	}

	// Check if expired
	if time.Now().After(v.ExpiresAt) {
		return nil, forge.Unauthorized("magic link expired")
	}

	// Check if already consumed
	if v.Consumed {
		return nil, forge.Unauthorized("magic link already used")
	}

	// Consume the verification
	if err := p.store.ConsumeVerification(ctx.Context(), req.Token); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to verify magic link: %w", err))
	}

	// Look up user
	u, err := p.store.GetUser(ctx.Context(), v.UserID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to resolve user: %w", err))
	}

	// Resolve per-app session config, falling back to plugin config.
	sessCfg := account.SessionConfig{
		TokenTTL:        p.config.SessionTokenTTL,
		RefreshTokenTTL: p.config.SessionRefreshTTL,
	}
	if p.sessionConfig != nil {
		sessCfg = p.sessionConfig.SessionConfigForApp(ctx.Context(), v.AppID)
	}
	sess, err := account.NewSession(v.AppID, u.ID, sessCfg)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create session: %w", err))
	}

	if err := p.store.CreateSession(ctx.Context(), sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create session: %w", err))
	}

	return &VerifyResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
	}, nil
}
