// Package phone provides phone number + OTP authentication as a primary auth method.
// It reuses the SMS bridge, Twilio adapter, and MFA SMS code generation infrastructure.
package phone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.AuthMethodContributor = (*Plugin)(nil)
	_ plugin.SettingsProvider      = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingCodeTTLSeconds controls the lifetime of OTP codes in seconds.
	SettingCodeTTLSeconds = settings.Define("phone.code_ttl_seconds", 300,
		settings.WithDisplayName("OTP Code TTL (seconds)"),
		settings.WithDescription("Lifetime of OTP verification codes in seconds"),
		settings.WithCategory("Phone Auth"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(60), Max: intPtr(900)}),
		settings.WithHelpText("How long OTP codes remain valid. Default: 300 (5 minutes)"),
		settings.WithOrder(10),
	)

	// SettingAutoCreate controls whether to create new users for unknown phone numbers.
	SettingAutoCreate = settings.Define("phone.auto_create", true,
		settings.WithDisplayName("Auto-Create Users"),
		settings.WithDescription("Automatically create new users when an unregistered phone number is used"),
		settings.WithCategory("Phone Auth"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When disabled, only existing users can authenticate via phone"),
		settings.WithOrder(20),
	)
)

func intPtr(v int) *int { return &v }

// phoneRegex is a basic E.164 phone number pattern.
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// Config configures the phone auth plugin.
type Config struct {
	// SMSSender sends SMS messages. If nil, the engine's SMS bridge is used.
	SMSSender bridge.SMSSender

	// CodeTTL is the lifetime of OTP codes (default: 5 minutes).
	CodeTTL time.Duration

	// AutoCreate controls whether to create new users when the phone number
	// is not found. If false, unregistered phone numbers receive an error.
	// Default: true.
	AutoCreate *bool
}

// Plugin is the phone authentication plugin.
type Plugin struct {
	config     Config
	store      store.Store
	sms        bridge.SMSSender
	ceremonies ceremony.Store
	logger     log.Logger
	appID      string
	engine     plugin.Engine
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "phone", SettingCodeTTLSeconds); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "phone", SettingAutoCreate)
}

// New creates a new phone auth plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.CodeTTL == 0 {
		c.CodeTTL = 5 * time.Minute
	}
	if c.AutoCreate == nil {
		t := true
		c.AutoCreate = &t
	}
	// Honour an explicitly supplied sender immediately. OnInit re-applies the
	// same precedence over the engine's bridge; doing it here too means a
	// caller who passes one in Config isn't silently ignored before OnInit.
	return &Plugin{config: c, sms: c.SMSSender}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "phone" }

// OnInit captures dependencies from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.store = engine.Store()

	p.sms = engine.SMSSender()
	// Config-level SMS sender takes precedence.
	if p.config.SMSSender != nil {
		p.sms = p.config.SMSSender
	}

	p.ceremonies = engine.CeremonyStore()
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}

	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	return nil
}

// RegisterRoutes registers phone auth HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group("/v1/phone", forge.WithGroupTags("Phone Auth"))

	// /start sends an SMS, so it's capped like resend-verification — an
	// uncapped one bills the operator per request. /verify accepts a 6-digit
	// code; the per-challenge attempt cap is the primary defence there, with
	// this as defence in depth since rate limiting can be disabled.
	startRL := authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
		return c.ResendVerificationLimit
	})
	verifyRL := authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
		return c.MFAChallengeLimit
	})

	if err := g.POST("/start", p.handleStart, append([]forge.RouteOption{
		forge.WithSummary("Start phone authentication"),
		forge.WithDescription("Sends an OTP code to the given phone number."),
		forge.WithOperationID("phoneAuthStart"),
		forge.WithResponseSchema(http.StatusOK, "OTP sent", StartResponse{}),
		forge.WithErrorResponses(),
	}, startRL...)...); err != nil {
		return err
	}

	return g.POST("/verify", p.handleVerify, append([]forge.RouteOption{
		forge.WithSummary("Verify phone OTP"),
		forge.WithDescription("Verifies the OTP code and creates or finds the user, returning a session."),
		forge.WithOperationID("phoneAuthVerify"),
		forge.WithResponseSchema(http.StatusOK, "Authenticated", VerifyResponse{}),
		forge.WithErrorResponses(),
	}, verifyRL...)...)
}

// SetStore allows direct store injection for testing.
func (p *Plugin) SetStore(s store.Store) { p.store = s }

// SetAppID sets the app ID for the plugin (used in testing).
func (p *Plugin) SetAppID(appID string) { p.appID = appID }

// SetCeremonyStore injects the short-lived challenge store, which OnInit
// otherwise resolves from the engine (used in testing).
func (p *Plugin) SetCeremonyStore(s ceremony.Store) { p.ceremonies = s }

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// StartRequest is the request body for starting phone auth.
type StartRequest struct {
	Phone string `json:"phone"`
	AppID string `json:"app_id,omitempty"`
}

// StartResponse is returned when an OTP is sent.
type StartResponse struct {
	Status    string `json:"status"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

// VerifyRequest is the request body for verifying phone OTP.
type VerifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
	AppID string `json:"app_id,omitempty"`
}

// VerifyResponse is returned when phone auth succeeds.
type VerifyResponse struct {
	User         *user.User `json:"user"`
	SessionToken string     `json:"session_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    time.Time  `json:"expires_at"`
	NewUser      bool       `json:"new_user"`
}

// maxPhoneCodeAttempts caps wrong-code submissions against a single challenge
// before it is discarded and the user must request a fresh code. A 6-digit
// code is only 10^6 wide, so without a cap the 5-minute TTL is enough to walk
// the whole space — and route rate limiting can be switched off by config,
// which would leave nothing at all in the way.
const maxPhoneCodeAttempts = 5

// phoneChallenge is stored in the ceremony store.
type phoneChallenge struct {
	Code      string    `json:"code"`
	Phone     string    `json:"phone"`
	AppID     string    `json:"app_id"`
	ExpiresAt time.Time `json:"expires_at"`
	// Attempts counts failed code submissions against this challenge.
	Attempts int `json:"attempts"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleStart(ctx forge.Context, req *StartRequest) (*StartResponse, error) {
	if req.Phone == "" {
		return nil, forge.BadRequest("phone number required")
	}
	if !phoneRegex.MatchString(req.Phone) {
		return nil, forge.BadRequest("phone number must be in E.164 format (e.g. +14155551234)")
	}
	if p.sms == nil {
		return nil, forge.InternalError(fmt.Errorf("phone auth: SMS sender not configured"))
	}

	appIDStr := req.AppID
	if appIDStr == "" {
		appIDStr = p.appID
	}
	if appIDStr == "" {
		return nil, forge.BadRequest("app_id required")
	}

	// Generate and send OTP using the MFA SMS helper.
	challenge, err := mfa.SendSMSChallenge(ctx.Context(), p.sms, req.Phone)
	if err != nil {
		p.logger.Error("phone auth: failed to send OTP",
			log.String("phone", req.Phone),
			log.String("error", err.Error()),
		)
		return nil, forge.InternalError(fmt.Errorf("phone auth: failed to send OTP: %w", err))
	}

	// Store challenge in ceremony store keyed by phone+app.
	ceremonyKey := fmt.Sprintf("phoneauth:%s:%s", appIDStr, req.Phone)
	data, err := json.Marshal(&phoneChallenge{
		Code:      challenge.Code,
		Phone:     req.Phone,
		AppID:     appIDStr,
		ExpiresAt: challenge.ExpiresAt,
	})
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("phone auth: marshal challenge: %w", err))
	}
	if err := p.ceremonies.Set(ctx.Context(), ceremonyKey, data, p.config.CodeTTL); err != nil {
		return nil, forge.InternalError(fmt.Errorf("phone auth: store challenge: %w", err))
	}

	return &StartResponse{
		Status:    "otp_sent",
		ExpiresIn: int(p.config.CodeTTL.Seconds()),
	}, nil
}

// recordFailedAttempt charges one wrong guess against a challenge, discarding
// it once maxPhoneCodeAttempts is reached so the caller must request a fresh
// code.
//
// The rewritten entry keeps the challenge's original deadline rather than a
// full TTL — re-Setting with p.config.CodeTTL would let an attacker hold a
// challenge open indefinitely by submitting wrong codes, which is the opposite
// of the intent. If the remaining time is already gone, the entry is dropped.
//
// Best-effort by design: a ceremony-store failure here must not convert a
// wrong-code response into a 500, which would itself leak that the code was
// wrong via a distinguishable status.
func (p *Plugin) recordFailedAttempt(ctx context.Context, key string, challenge *phoneChallenge) {
	challenge.Attempts++

	remaining := time.Until(challenge.ExpiresAt)
	if challenge.Attempts >= maxPhoneCodeAttempts || remaining <= 0 {
		_ = p.ceremonies.Delete(ctx, key) //nolint:errcheck // best-effort
		return
	}

	data, err := json.Marshal(challenge)
	if err != nil {
		// Can't persist the count — drop the challenge rather than leave it
		// standing with the attempt uncounted.
		_ = p.ceremonies.Delete(ctx, key) //nolint:errcheck // best-effort
		return
	}
	if setErr := p.ceremonies.Set(ctx, key, data, remaining); setErr != nil {
		_ = p.ceremonies.Delete(ctx, key) //nolint:errcheck // best-effort
	}
}

func (p *Plugin) handleVerify(ctx forge.Context, req *VerifyRequest) (*VerifyResponse, error) {
	if req.Phone == "" {
		return nil, forge.BadRequest("phone number required")
	}
	if req.Code == "" {
		return nil, forge.BadRequest("verification code required")
	}

	appIDStr := req.AppID
	if appIDStr == "" {
		appIDStr = p.appID
	}
	if appIDStr == "" {
		return nil, forge.BadRequest("app_id required")
	}

	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	// Retrieve and validate challenge from ceremony store.
	ceremonyKey := fmt.Sprintf("phoneauth:%s:%s", appIDStr, req.Phone)
	data, err := p.ceremonies.Get(ctx.Context(), ceremonyKey)
	if err != nil {
		return nil, forge.Unauthorized("no pending verification for this phone number")
	}

	var challenge phoneChallenge
	if unmarshalErr := json.Unmarshal(data, &challenge); unmarshalErr != nil {
		return nil, forge.InternalError(fmt.Errorf("phone auth: unmarshal challenge: %w", unmarshalErr))
	}

	// Validate using the MFA helper.
	if !mfa.ValidateSMSCode(req.Code, &mfa.SMSChallenge{
		Code:      challenge.Code,
		ExpiresAt: challenge.ExpiresAt,
	}) {
		p.recordFailedAttempt(ctx.Context(), ceremonyKey, &challenge)
		return nil, forge.Unauthorized("invalid or expired verification code")
	}

	// Consume the challenge.
	_ = p.ceremonies.Delete(ctx.Context(), ceremonyKey) //nolint:errcheck // best-effort cleanup

	// Look up or create user by phone.
	u, newUser, err := p.resolveOrCreateUser(ctx.Context(), appID, req.Phone)
	if err != nil {
		return nil, err
	}

	// Mark phone as verified if not already.
	if !u.PhoneVerified {
		u.PhoneVerified = true
		if updateErr := p.store.UpdateUser(ctx.Context(), u); updateErr != nil {
			p.logger.Warn("phone auth: failed to mark phone verified",
				log.String("user_id", u.ID.String()),
				log.String("error", updateErr.Error()),
			)
		}
	}

	// Mint the session through Engine.IssueSession so the centralized
	// MFARequired gate fires for phone-OTP sign-ins too.
	var sess *session.Session
	if eng, ok := p.engine.(*authsome.Engine); ok && eng != nil {
		result, issueErr := eng.IssueSession(ctx.Context(), &authsome.IssueSessionRequest{
			User:       u,
			AppID:      appID,
			AuthMethod: "phone",
			IPAddress:  ctx.Request().RemoteAddr,
			UserAgent:  ctx.Request().UserAgent(),
		})
		if issueErr != nil {
			return nil, issueErr
		}
		sess = result.Session
	} else {
		sessCfg := account.SessionConfig{
			TokenTTL:        time.Hour,
			RefreshTokenTTL: 30 * 24 * time.Hour,
		}
		if p.engine != nil {
			sessCfg = p.engine.SessionConfigForApp(ctx.Context(), appID)
		}
		var newErr error
		sess, newErr = account.NewSession(appID, u.ID, sessCfg)
		if newErr != nil {
			return nil, forge.InternalError(fmt.Errorf("phone auth: create session: %w", newErr))
		}
		if storeErr := p.store.CreateSession(ctx.Context(), sess); storeErr != nil {
			return nil, forge.InternalError(fmt.Errorf("phone auth: save session: %w", storeErr))
		}
	}

	return &VerifyResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
		NewUser:      newUser,
	}, nil
}

// resolveOrCreateUser looks up a user by phone or creates a new one.
//
// Both the lookup and the create are scoped to the app's default environment,
// the same way SSO and password sign-ups place their users. A phone number is
// only unique within an environment, so an app-wide lookup here would let one
// environment's account answer for another's.
func (p *Plugin) resolveOrCreateUser(ctx context.Context, appID id.AppID, phone string) (*user.User, bool, error) {
	var envID id.EnvironmentID
	if env, _ := p.store.GetDefaultEnvironment(ctx, appID); env != nil { //nolint:errcheck // best-effort env lookup
		envID = env.ID
	}

	u, err := p.store.GetUserByPhone(ctx, appID, envID, phone)
	if err == nil {
		return u, false, nil
	}

	// If auto-create is disabled, reject unknown phone numbers.
	if !*p.config.AutoCreate {
		return nil, false, forge.Unauthorized("phone number not registered")
	}

	// Create a new user with the phone number.
	newUser := &user.User{
		ID:            id.NewUserID(),
		AppID:         appID,
		EnvID:         envID,
		Phone:         phone,
		PhoneVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := p.store.CreateUser(ctx, newUser); err != nil {
		return nil, false, forge.InternalError(fmt.Errorf("phone auth: create user: %w", err))
	}
	if p.engine != nil {
		p.engine.EnsureDefaultRole(ctx, appID, newUser.ID)
	}

	return newUser, true, nil
}

// ListUserAuthMethods reports phone as a linked auth method if the user
// has a verified phone number.
func (p *Plugin) ListUserAuthMethods(ctx context.Context, userID id.UserID) ([]*plugin.AuthMethod, error) {
	u, err := p.store.GetUser(ctx, userID)
	if err != nil {
		return nil, nil
	}
	if u.Phone == "" || !u.PhoneVerified {
		return nil, nil
	}
	return []*plugin.AuthMethod{{
		Type:     "phone",
		Provider: "phone",
		Label:    "Phone (" + u.Phone + ")",
		LinkedAt: u.CreatedAt.Format(time.RFC3339),
	}}, nil
}
