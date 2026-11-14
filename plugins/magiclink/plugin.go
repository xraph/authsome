package magiclink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
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

	// SettingEmailCode turns on the 6-digit sign-in code that goes out in the
	// same email as the link. Unset, Config.EmailCode decides.
	SettingEmailCode = settings.Define("magiclink.email_code", false,
		settings.WithDisplayName("Email a sign-in code"),
		settings.WithDescription("Send a 6-digit code alongside the magic link"),
		settings.WithCategory("Magic Link"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldSwitch),
		settings.WithHelpText("The code and the link share one lifetime, and using either one uses up both. Needs a mailer that implements magiclink.CodeMailer."),
		settings.WithOrder(40),
	)

	// SettingAllowSignUp lets /v1/magic-link/send create an account for an
	// unknown address when the request asks for it. Unset,
	// Config.AllowSignUp decides.
	SettingAllowSignUp = settings.Define("magiclink.allow_signup", false,
		settings.WithDisplayName("Passwordless sign-up"),
		settings.WithDescription("Create an account with no password when someone asks for a link with create_user"),
		settings.WithCategory("Magic Link"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldSwitch),
		settings.WithHelpText("Sign-up still has to be enabled for the app, and the sign-up hooks (waitlist, captcha) still apply."),
		settings.WithOrder(50),
	)
)

func intPtr(v int) *int { return &v }

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

// Purpose says why a magic link email went out.
type Purpose string

const (
	// PurposeSignIn is a link for an account that already exists.
	PurposeSignIn Purpose = "signin"
	// PurposeSignUp is a link for an account created by this request, or
	// one created earlier that has never proved its inbox.
	PurposeSignUp Purpose = "signup"
)

// MagicLinkMessage is everything a CodeMailer needs to write the email.
type MagicLinkMessage struct { //nolint:revive // the name reads well beside Mailer.SendMagicLink at call sites
	// AppID is the app the link signs in to.
	AppID string
	// Email is the recipient.
	Email string
	// Token goes in the link. The client posts it to /v1/magic-link/verify.
	Token string
	// Code is the 6-digit code the user can type instead of clicking. It is
	// empty unless the email code is on for the app.
	Code string
	// Purpose is PurposeSignIn or PurposeSignUp, so the email can greet a
	// new account differently.
	Purpose Purpose
	// ExpiresAt is when the link and the code stop working.
	ExpiresAt time.Time
	// Name is the user's name, when the account has one.
	Name string
}

// CodeMailer is a Mailer that can also deliver the sign-in code and knows
// why the email is going out. When the configured Mailer implements it, the
// plugin calls SendMagicLinkWithCode instead of SendMagicLink. The email
// code is only minted for a mailer that implements it.
type CodeMailer interface {
	Mailer
	SendMagicLinkWithCode(ctx context.Context, msg MagicLinkMessage) error
}

// Config configures the magic link plugin.
type Config struct {
	// Mailer sends magic link emails. Required. Implement CodeMailer as well
	// to deliver the email code and the purpose of the email.
	Mailer Mailer

	// EmailCode also mints a 6-digit code with every link, delivered in the
	// same email, which the user can type at /v1/magic-link/verify-code.
	// The code and link share one lifetime and using either uses up both.
	// It needs a Mailer that implements CodeMailer. The
	// magiclink.email_code setting overrides it per app. Default: false.
	EmailCode bool

	// AllowSignUp lets /v1/magic-link/send create an account with no
	// password when the request carries create_user and no account has the
	// address. Sign-up must also be enabled for the app. The
	// magiclink.allow_signup setting overrides it per app. Default: false.
	AllowSignUp bool

	// TokenTTL is the lifetime of magic link tokens (default: 10 minutes).
	TokenTTL time.Duration

	// SessionTokenTTL is the lifetime of sessions created via magic link (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens for magic link sessions (default: 30 days).
	SessionRefreshTTL time.Duration
}

// Plugin is the magic link authentication plugin.
type Plugin struct {
	config      Config
	store       store.Store
	appID       string
	engine      plugin.Engine
	settingsMgr *settings.Manager

	// ceremonies holds code state when the plugin runs without a concrete
	// engine (unit tests). With one, the engine's ceremony store is used so
	// a code sent from one replica can be checked on another.
	ceremonies ceremony.Store
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "magiclink", SettingTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "magiclink", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "magiclink", SettingSessionRefreshTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "magiclink", SettingEmailCode); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "magiclink", SettingAllowSignUp)
}

// New creates a new magic link plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.TokenTTL == 0 {
		c.TokenTTL = 10 * time.Minute
	}
	if c.SessionTokenTTL == 0 {
		c.SessionTokenTTL = 1 * time.Hour
	}
	if c.SessionRefreshTTL == 0 {
		c.SessionRefreshTTL = 30 * 24 * time.Hour
	}
	return &Plugin{config: c, ceremonies: ceremony.NewMemory()}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "magiclink" }

// OnInit captures the store reference from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.store = engine.Store()
	p.engine = engine
	p.settingsMgr = engine.Settings()
	if p.config.EmailCode && p.config.Mailer != nil {
		if _, ok := p.config.Mailer.(CodeMailer); !ok {
			if logger := engine.Logger(); logger != nil {
				logger.Warn("magiclink: EmailCode is on but the Mailer does not implement magiclink.CodeMailer; no code will be sent")
			}
		}
	}
	return nil
}

// resolveBool reads a switch setting for the app. The Config value is the
// default: it holds until the setting is set at some scope.
func (p *Plugin) resolveBool(ctx context.Context, appID id.AppID, def settings.DefinitionTyped[bool], fallback bool) bool {
	if p.settingsMgr == nil {
		return fallback
	}
	opts := settings.ResolveOpts{}
	if !appID.IsNil() {
		opts.AppID = appID.String()
	}
	details, err := p.settingsMgr.ResolveWithDetails(ctx, def.Def.Key, opts)
	if err != nil || details == nil || (len(details.ScopeValues) == 0 && details.EnforcedAt == nil) {
		return fallback
	}
	var v bool
	if json.Unmarshal(details.EffectiveValue, &v) != nil {
		return fallback
	}
	return v
}

// resolveTTL reads a duration-in-seconds setting for the app, falling back to
// the compile-time Config value.
//
// The settings were declared and registered from the start but never read, so
// every dashboard control for this plugin silently did nothing. Reading them
// here is what makes DeclareSettings mean anything.
func (p *Plugin) resolveTTL(ctx context.Context, appID id.AppID, def settings.DefinitionTyped[int], fallback time.Duration) time.Duration {
	if p.settingsMgr == nil {
		return fallback
	}
	opts := settings.ResolveOpts{}
	if !appID.IsNil() {
		opts.AppID = appID.String()
	}
	secs, err := settings.Get(ctx, p.settingsMgr, def, opts)
	if err != nil || secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}

// RegisterRoutes registers magic link HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group("/v1/magic-link", forge.WithGroupTags("Magic Link"))

	// /send mints a token and emails it — capped like resend-verification so
	// it can't be used to bomb an inbox. /verify accepts a bearer-equivalent
	// secret, so it's capped like the other token-submission endpoints.
	sendRL := authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
		return c.ResendVerificationLimit
	})
	verifyRL := authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
		return c.VerifyEmailLimit
	})
	// A code is six digits, so /verify-code is capped like the other
	// code-submission routes on top of the per-code attempt bound.
	verifyCodeRL := authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
		return c.VerifyEmailLimit
	})

	if err := g.POST("/send", p.handleSend, append([]forge.RouteOption{
		forge.WithSummary("Send magic link"),
		forge.WithOperationID("sendMagicLink"),
		forge.WithResponseSchema(http.StatusOK, "Magic link sent", SendResponse{}),
		forge.WithErrorResponses(),
	}, sendRL...)...); err != nil {
		return err
	}

	if err := g.POST("/verify", p.handleVerify, append([]forge.RouteOption{
		forge.WithSummary("Verify magic link"),
		forge.WithOperationID("verifyMagicLink"),
		forge.WithResponseSchema(http.StatusOK, "Verified", VerifyResponse{}),
		forge.WithErrorResponses(),
	}, verifyRL...)...); err != nil {
		return err
	}

	return g.POST("/verify-code", p.handleVerifyCode, append([]forge.RouteOption{
		forge.WithSummary("Verify magic link code"),
		forge.WithDescription("Exchanges the 6-digit code from a magic link email for a session."),
		forge.WithOperationID("verifyMagicLinkCode"),
		forge.WithResponseSchema(http.StatusOK, "Verified", VerifyResponse{}),
		forge.WithErrorResponses(),
	}, verifyCodeRL...)...)
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
	// CreateUser asks for an account to be created, with no password, when
	// no account has the address. It needs passwordless sign-up to be on.
	CreateUser bool `json:"create_user,omitempty"`
	// Name is the new account's name, used only when one is created.
	Name string `json:"name,omitempty"`
	// CaptchaToken is the captcha answer for a sign-up, when the app
	// requires one. The X-Captcha-Token header works too.
	CaptchaToken string `json:"captcha_token,omitempty"`
}

// SendResponse is returned when a magic link is sent. It is the same
// whether or not an account has the address.
type SendResponse struct {
	Status string `json:"status"`
	// ExpiresIn is how many seconds the link (and the code) stay valid.
	ExpiresIn int `json:"expires_in"`
	// CodeLength is the number of digits in the emailed code, or absent
	// when no code is sent.
	CodeLength int `json:"code_length,omitempty"`
}

// VerifyCodeRequest is the request body for signing in with the emailed code.
type VerifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
	AppID string `json:"app_id,omitempty"`
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
	email := normalizeEmail(req.Email)
	if email == "" {
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

	// Everything in the response is decided before the address is looked
	// up, so an unknown address gets exactly what a known one does.
	tokenTTL := p.resolveTTL(ctx.Context(), appID, SettingTokenTTLSeconds, p.config.TokenTTL)
	withCode := p.codeEnabled(ctx.Context(), appID)
	resp := &SendResponse{Status: "magic link sent", ExpiresIn: int(tokenTTL / time.Second)}
	if withCode {
		resp.CodeLength = codeLength
	}

	// A sign-up is refused, or held for a captcha, before the address is
	// looked up, for the same reason.
	if req.CreateUser {
		if !p.signUpEnabled(ctx.Context(), appID) {
			return nil, forge.Forbidden("sign-up is disabled")
		}
		if rejected, captchaErr := p.checkSignUpCaptcha(ctx, appID, req.CaptchaToken); rejected {
			return nil, captchaErr
		}
	}

	// One address may receive only so many links per window, from however
	// many client addresses the requests come.
	if limitErr := authsome.PluginIdentifierLimit(ctx.Context(), p.engine, "magiclink", appID, email,
		func(c authsome.RateLimitConfig) int { return c.ResendVerificationLimit }); limitErr != nil {
		return nil, forge.NewHTTPError(http.StatusTooManyRequests, "too many links requested for this address, try again later")
	}

	purpose := PurposeSignIn
	var created *user.User
	if req.CreateUser {
		// The sign-up path runs whether or not the address has an account,
		// the way /v1/signup does, so a hook that refuses the sign-up
		// (the waitlist, say) refuses it either way and the answer doesn't
		// say which addresses are registered.
		var suErr error
		created, suErr = p.signUp(ctx, appID, email, req.Name)
		switch {
		case suErr == nil:
			purpose = PurposeSignUp
		case errors.Is(suErr, account.ErrEmailTaken):
			// An existing account gets an ordinary sign-in link.
		case errors.Is(suErr, account.ErrSignUpDisabled):
			return nil, forge.Forbidden("sign-up is disabled")
		case errors.Is(suErr, account.ErrSignUpRefused):
			return nil, forge.Forbidden("sign-up was refused")
		case errors.Is(suErr, account.ErrRateLimited):
			return nil, forge.NewHTTPError(http.StatusTooManyRequests, "too many attempts, try again later")
		default:
			return nil, middleware.InternalError(ctx, fmt.Errorf("magiclink: sign up: %w", suErr))
		}
	}

	// Resolve the real user by email. The verification token must be bound to
	// their actual user id so /verify can resolve them — binding it to a random
	// id (the previous behavior) made every real magic-link login fail.
	// Scope the lookup to the app's default environment, the way SSO and
	// password sign-in do. A magic link grants a session, so resolving an
	// address app-wide would let one environment mint a session for an
	// account that lives in another.
	u := created
	if u == nil {
		var lookupErr error
		u, lookupErr = p.lookupUser(ctx.Context(), appID, email)
		if lookupErr != nil {
			// Anti-enumeration: return the same success response whether or not the
			// email is registered, and never mint a token or send a link for an
			// address with no account.
			return resp, nil
		}
		// An account that never proved its inbox (an unfinished passwordless
		// sign-up) is still signing up.
		if req.CreateUser && !u.EmailVerified {
			purpose = PurposeSignUp
		}
	}

	// Create verification token bound to the resolved user.
	v, err := account.NewVerification(ctx.Context(), appID, u.ID, VerificationTypeMagicLink, tokenTTL)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create magic link token: %w", err))
	}
	// NewVerification leaves the environment empty, and the SQL stores hold
	// env_id to a foreign key, so an empty one fails the insert and every
	// send for a real account is a 500. The link lives in the user's own
	// environment.
	v.EnvID = u.EnvID

	if createErr := p.store.CreateVerification(ctx.Context(), v); createErr != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create magic link: %w", createErr))
	}

	var code string
	if withCode {
		code, err = p.issueCode(ctx.Context(), v)
		if err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create magic link code: %w", err))
		}
	}

	if sendErr := p.deliver(ctx.Context(), MagicLinkMessage{
		AppID:     appID.String(),
		Email:     email,
		Token:     v.Token,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: v.ExpiresAt,
		Name:      u.Name(),
	}); sendErr != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to send magic link email: %w", sendErr))
	}

	return resp, nil
}

// lookupUser resolves an address to its account in the app's default
// environment.
func (p *Plugin) lookupUser(ctx context.Context, appID id.AppID, email string) (*user.User, error) {
	var envID id.EnvironmentID
	if env, _ := p.store.GetDefaultEnvironment(ctx, appID); env != nil { //nolint:errcheck // best-effort env lookup
		envID = env.ID
	}
	return p.store.GetUserByEmail(ctx, appID, envID, email)
}

// deliver hands the email to the Mailer, preferring CodeMailer when the
// Mailer implements it.
func (p *Plugin) deliver(ctx context.Context, msg MagicLinkMessage) error {
	if p.config.Mailer == nil {
		return nil
	}
	if cm, ok := p.config.Mailer.(CodeMailer); ok {
		return cm.SendMagicLinkWithCode(ctx, msg)
	}
	return p.config.Mailer.SendMagicLink(ctx, msg.Email, msg.Token)
}

func (p *Plugin) handleVerify(ctx forge.Context, req *VerifyRequest) (_ *VerifyResponse, err error) {
	attempt := telemetry.From(p.engine).StartSignIn(telemetry.MethodMagicLink)
	defer func() { attempt.Done(err) }()

	if req.Token == "" {
		return nil, forge.BadRequest("token required")
	}

	// Look up verification
	v, err := p.store.GetVerification(ctx.Context(), req.Token)
	if err != nil {
		return nil, forge.Unauthorized("invalid or expired magic link")
	}

	// Verifications of every kind share one table and GetVerification matches
	// on token alone, so this endpoint must confirm it was handed a magic-link
	// token and not some other type. Email-verification records carry a
	// 6-digit OTP in the same Token column: without this check, posting
	// {"token":"123456"} here walks a 10^6 keyspace and mints a full session
	// on a hit, bypassing the attempt cap the email-verify path enforces.
	if v.Type != VerificationTypeMagicLink {
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

	// The link and its code are one credential. Whichever is used first
	// claims it; the other is refused from then on.
	won, claimErr := p.claim(ctx.Context(), v.ID, v.ExpiresAt)
	if claimErr != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to verify magic link: %w", claimErr))
	}
	if !won {
		return nil, forge.Unauthorized("magic link already used")
	}

	// Consume the verification. The store consumes only if still unconsumed,
	// so ErrNotFound here means another request won the race — the Consumed
	// check above and this write are two steps, and this is what makes one
	// link yield one session.
	if consumeErr := p.store.ConsumeVerification(ctx.Context(), req.Token); consumeErr != nil {
		if errors.Is(consumeErr, store.ErrNotFound) {
			return nil, forge.Unauthorized("magic link already used")
		}
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to verify magic link: %w", consumeErr))
	}
	p.dropCode(ctx.Context(), v.AppID, v.UserID, v.ID)

	// Look up user
	u, err := p.store.GetUser(ctx.Context(), v.UserID)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to resolve user: %w", err))
	}

	return p.signIn(ctx, attempt, u, v.AppID)
}

// signIn finishes a verified link or code: the inbox is proven, so the
// email is marked verified, and the session is minted through
// Engine.IssueSession so the MFA gate fires as it does for every other
// sign-in.
func (p *Plugin) signIn(ctx forge.Context, attempt *telemetry.SignInAttempt, u *user.User, appID id.AppID) (*VerifyResponse, error) {
	// Using a link or code proves email ownership — mark as verified.
	eng, isEngine := p.engine.(*authsome.Engine)
	if isEngine && eng != nil {
		if markErr := eng.MarkEmailVerified(ctx.Context(), u); markErr != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to mark email verified: %w", markErr))
		}
	} else if !u.EmailVerified {
		u.EmailVerified = true
		u.UpdatedAt = time.Now()
		_ = p.store.UpdateUser(ctx.Context(), u) //nolint:errcheck // best-effort
	}

	// Mint the session through Engine.IssueSession so the centralized
	// MFARequired gate fires for magic-link verifications. Falls back
	// to a direct mint if the engine isn't the concrete *authsome.Engine.
	var sess *session.Session
	if isEngine && eng != nil {
		// /verify is a POST from the SDK, not the browser following the link,
		// so the client holding the DPoP key is the caller here and can prove
		// possession. Resolve the binding before minting: under mode=required
		// IssueSession refuses an unbound session, and without this the whole
		// magic-link route would be refused rather than bound.
		dpopJKT, bindErr := eng.DPoPBindingForRequest(ctx, appID)
		if bindErr != nil {
			if errors.Is(bindErr, authsome.ErrDPoPBindingRequired) {
				attempt.Refuse(telemetry.SignInDenied)
			}
			return nil, bindErr
		}
		attempt.HandOff()
		result, issueErr := eng.IssueSession(ctx.Context(), &authsome.IssueSessionRequest{
			User:       u,
			AppID:      appID,
			AuthMethod: "magiclink",
			DPoPJKT:    dpopJKT,
			IPAddress:  ctx.Request().RemoteAddr,
			UserAgent:  ctx.Request().UserAgent(),
			SessionTTL: p.resolveTTL(ctx.Context(), appID, SettingSessionTokenTTLSeconds, p.config.SessionTokenTTL),
			RefreshTTL: p.resolveTTL(ctx.Context(), appID, SettingSessionRefreshTTLSeconds, p.config.SessionRefreshTTL),
		})
		if issueErr != nil {
			return nil, issueErr
		}
		sess = result.Session
	} else {
		sessCfg := account.SessionConfig{
			TokenTTL:        p.resolveTTL(ctx.Context(), appID, SettingSessionTokenTTLSeconds, p.config.SessionTokenTTL),
			RefreshTokenTTL: p.resolveTTL(ctx.Context(), appID, SettingSessionRefreshTTLSeconds, p.config.SessionRefreshTTL),
		}
		if p.engine != nil {
			sessCfg = p.engine.SessionConfigForApp(ctx.Context(), appID)
		}
		var newErr error
		sess, newErr = account.NewSession(appID, u.ID, sessCfg)
		if newErr != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create session: %w", newErr))
		}
		if storeErr := p.store.CreateSession(ctx.Context(), sess); storeErr != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create session: %w", storeErr))
		}
	}

	return &VerifyResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
	}, nil
}
