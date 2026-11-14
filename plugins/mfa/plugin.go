package mfa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/authsome/internal/mask"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"

	"github.com/xraph/grove/migrate"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
	_ plugin.BeforeUserDelete  = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingIssuer controls the issuer name shown in authenticator apps.
	SettingIssuer = settings.Define("mfa.issuer", "AuthSome",
		settings.WithDisplayName("Issuer Name"),
		settings.WithDescription("Name shown in authenticator apps (e.g. Google Authenticator)"),
		settings.WithCategory("Multi-Factor Auth"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithPlaceholder("AuthSome"),
		settings.WithHelpText("The issuer label users see in their authenticator app"),
		settings.WithOrder(10),
	)
)

// Config configures the MFA plugin.
type Config struct {
	// Issuer is the name shown in authenticator apps (default: "AuthSome").
	Issuer string

	// StepUpWindow is how recently the caller must have signed in to enrol
	// a first factor on session auth alone (default: 5 minutes). Once a
	// factor is verified, enrolling another one asks for that factor.
	StepUpWindow time.Duration

	// TrustedDeviceTTL is how long a device the user chose to trust at the
	// sign-in challenge skips the second factor (default 30 days). A
	// negative value turns trusting devices off.
	TrustedDeviceTTL time.Duration

	// DisableTrustedDevices turns off "trust this device": the challenge
	// ignores trust_device and no existing token skips the second factor.
	DisableTrustedDevices bool
}

// Plugin is the MFA authentication plugin.
type Plugin struct {
	config     Config
	store      Store
	sms        bridge.SMSSender
	chronicle  bridge.Chronicle
	relay      bridge.EventRelay
	hooks      *hook.Bus
	logger     log.Logger
	ceremonies ceremony.Store
	// engine is the host engine reference, captured during OnInit, used
	// by the /v1/mfa/challenge handler to issue real sessions after a
	// ticket+code round-trip via Engine.IssueSession.
	engine plugin.Engine
	// revoker ends the user's other sessions after a factor changes. Nil in
	// minimal test wiring, in which case nothing is revoked.
	revoker sessionRevoker
	// stamper records on the request's session that it passed a second
	// factor. Nil in minimal test wiring, in which case nothing is recorded.
	stamper sessionStamper
	// tel counts every code check. Nil when the engine records no metrics.
	tel *telemetry.Metrics
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "mfa", SettingIssuer); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "mfa", SettingTrustedDevicesEnabled)
}

// New creates a new MFA plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Issuer == "" {
		c.Issuer = "AuthSome"
	}
	if c.TrustedDeviceTTL == 0 {
		c.TrustedDeviceTTL = DefaultTrustedDeviceTTL
	}
	return &Plugin{
		config:     c,
		ceremonies: ceremony.NewMemory(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "mfa" }

// OnInit is called during engine initialization. It auto-discovers the SMS
// sender, chronicle, relay, hooks, logger, and ceremony store from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.sms = engine.SMSSender()
	p.chronicle = bridge.NewBusChronicle(engine.Hooks())
	p.relay = engine.Relay()
	p.hooks = engine.Hooks()
	p.logger = engine.Logger()
	p.ceremonies = engine.CeremonyStore()
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}
	p.engine = engine
	p.revoker = engine
	if st, ok := engine.(sessionStamper); ok {
		p.stamper = st
	}
	p.tel = telemetry.From(engine)
	p.subscribePasswordChanges()

	// Build a persistent, at-rest-encrypted enrollment store when one hasn't
	// been injected (tests inject via SetStore). TOTP/SMS secrets are wrapped
	// with the engine's token encryptor so a database leak does not expose
	// usable MFA seeds.
	if p.store == nil {
		var inner Store
		if db := engine.DB(); db != nil {
			switch db.Driver().Name() {
			case "pg":
				inner = NewPostgresStore(db)
			case "sqlite":
				inner = NewSqliteStore(db)
			case "mongo":
				inner = NewMongoStore(db)
			}
		}
		if inner == nil {
			inner = NewMemoryStore()
		}
		p.store = NewEncryptedStore(inner, engine.TokenEncryptor())
	}

	return nil
}

// MigrationGroups implements plugin.MigrationProvider.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// SetStore sets the MFA enrollment store for testing.
func (p *Plugin) SetStore(s Store) {
	p.store = s
}

// SetSMSSender sets the SMS sender for testing.
func (p *Plugin) SetSMSSender(s bridge.SMSSender) {
	p.sms = s
}

// RegisterRoutes registers MFA HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group("/v1/mfa", forge.WithGroupTags("MFA"))

	// withRL appends rate-limit middleware (keyed off the engine's limiter and
	// MFAChallengeLimit) to a route's options. Applied to the code-submission
	// endpoints so 6-digit TOTP/SMS/recovery codes cannot be brute-forced.
	rl := p.mfaRateLimit()
	withRL := func(opts ...forge.RouteOption) []forge.RouteOption {
		return append(opts, rl...)
	}

	// Enrolling a factor, turning MFA off and regenerating recovery codes
	// change the credentials on the account. An agent session reads as its
	// delegating user here and is freshly minted, so it would pass the
	// recent-sign-in step-up too. Only the user's own session may.
	ownSession := forge.WithMiddleware(middleware.DenyDelegatedSession())

	// Enroll and disable take a TOTP or recovery code as step-up, so they
	// are throttled like the other code-submission routes.
	if err := g.POST("/enroll", p.handleEnroll, withRL(
		ownSession,
		forge.WithSummary("Enroll in MFA"),
		forge.WithOperationID("enrollMFA"),
		forge.WithResponseSchema(http.StatusOK, "Enrollment started", EnrollResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	if err := g.POST("/verify", p.handleVerify, withRL(
		forge.WithSummary("Verify MFA code"),
		forge.WithOperationID("verifyMFA"),
		forge.WithResponseSchema(http.StatusOK, "Verified", VerifyMFAResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	if err := g.POST("/challenge", p.handleChallenge, withRL(
		forge.WithSummary("MFA challenge"),
		forge.WithOperationID("challengeMFA"),
		forge.WithResponseSchema(http.StatusOK, "Challenge passed", ChallengeResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	if err := g.GET("/enrollment", p.handleGetEnrollment,
		forge.WithSummary("Read MFA enrollment"),
		forge.WithDescription("Reports whether the signed-in user has a second factor set up, which kinds, and how many recovery codes are left. Never returns a secret or a recovery code."),
		forge.WithOperationID("getMFAEnrollment"),
		forge.WithResponseSchema(http.StatusOK, "Enrollment state", EnrollmentResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/enrollment", p.handleDisable, withRL(
		ownSession,
		forge.WithSummary("Disable MFA"),
		forge.WithOperationID("disableMFA"),
		forge.WithResponseSchema(http.StatusOK, "MFA disabled", DisableResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	if err := p.registerTrustedDeviceRoutes(g); err != nil {
		return err
	}

	// Recovery code routes
	if err := g.POST("/recovery/verify", p.handleRecoveryVerify, withRL(
		forge.WithSummary("Verify MFA recovery code"),
		forge.WithOperationID("verifyMFARecovery"),
		forge.WithResponseSchema(http.StatusOK, "Recovery code accepted", RecoveryVerifyResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	// The request can carry a TOTP code, so it is throttled like the other
	// code-submission routes.
	if err := g.POST("/recovery/regenerate", p.handleRecoveryRegenerate, withRL(
		ownSession,
		forge.WithSummary("Regenerate MFA recovery codes"),
		forge.WithOperationID("regenerateMFARecoveryCodes"),
		forge.WithResponseSchema(http.StatusOK, "New recovery codes generated", RecoveryRegenerateResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	// SMS MFA routes
	if err := g.POST("/sms/send", p.handleSMSSend, withRL(
		forge.WithSummary("Send SMS verification code"),
		forge.WithOperationID("sendSMSCode"),
		forge.WithResponseSchema(http.StatusOK, "SMS code sent", SMSSendResponse{}),
		forge.WithErrorResponses(),
	)...); err != nil {
		return err
	}

	return g.POST("/sms/verify", p.handleSMSVerify, withRL(
		forge.WithSummary("Verify SMS code"),
		forge.WithOperationID("verifySMSCode"),
		forge.WithResponseSchema(http.StatusOK, "SMS code verified", SMSVerifyResponse{}),
		forge.WithErrorResponses(),
	)...)
}

// mfaRateLimit returns rate-limit middleware options for MFA code-submission
// endpoints, keyed off the engine's limiter and the MFAChallengeLimit config.
// Returns nil when rate limiting is disabled or the host engine isn't the
// concrete *authsome.Engine (e.g. minimal test wiring).
//
// The MFA routes share one budget per client address, kept apart from the
// engine's default bucket (sign-up, email verification, sign-in): someone
// who enrolls straight after signing up would otherwise find their first
// code refused.
func (p *Plugin) mfaRateLimit() []forge.RouteOption {
	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil
	}
	rl := eng.RateLimiter()
	limit := eng.Config().RateLimit.MFAChallengeLimit
	if rl == nil || !eng.Config().RateLimit.Enabled || limit <= 0 {
		return nil
	}
	cfg := eng.RateLimitMiddlewareConfig(limit)
	cfg.KeyFunc = func(ctx forge.Context) string { return "mfa:" + middleware.ClientIP(ctx.Request()) }
	return []forge.RouteOption{forge.WithMiddleware(middleware.RateLimit(rl, cfg))}
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// EnrollRequest is the request body for MFA enrollment.
type EnrollRequest struct {
	Method string `json:"method,omitempty"` // "totp" or "sms"
	Phone  string `json:"phone,omitempty"`  // Required for SMS method
	// Code proves the factor the user already holds when enrolling a
	// further one: a current TOTP code or a recovery code.
	Code string `json:"code,omitempty"`
}

// EnrollResponse is returned when MFA enrollment starts.
type EnrollResponse struct {
	ID         string `json:"id"`
	Method     string `json:"method"`
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

// VerifyMFARequest is the request body for MFA verification.
type VerifyMFARequest struct {
	Code string `json:"code"`
}

// VerifyMFAResponse is returned on successful MFA verification.
type VerifyMFAResponse struct {
	Verified      bool     `json:"verified"`
	Method        string   `json:"method"`
	RecoveryCodes []string `json:"recovery_codes,omitempty"`
}

// ChallengeRequest is the request body for MFA challenge during the
// sign-in MFA gate. The mfa_ticket is what /v1/signin (or any other
// session-issuing endpoint) returned in the 403 body when MFA was
// required; the code is the user's TOTP digits or recovery code.
type ChallengeRequest struct {
	Ticket string `json:"mfa_ticket" description:"Ticket returned by /v1/signin when MFA was required"`
	Code   string `json:"code"       description:"TOTP code from the user's authenticator (6 digits) or recovery code"`
	// TrustDevice asks to skip the second factor on this device for the
	// configured trust period (30 days by default).
	TrustDevice bool `json:"trust_device,omitempty" description:"Trust this device: later sign-ins from it skip the second factor until the trust expires"`
}

// ChallengeResponse is the AuthResponse-shaped body returned on a
// successful MFA challenge. The shape matches /v1/signin's success
// response so SDK consumers can persist a session via the same code
// path regardless of whether the MFA gate fired.
type ChallengeResponse struct {
	User         any    `json:"user"          description:"User object"`
	SessionToken string `json:"session_token" description:"Session token"`
	RefreshToken string `json:"refresh_token" description:"Refresh token"`
	ExpiresAt    string `json:"expires_at"    description:"Token expiration time (RFC3339)"`

	// TrustedDeviceToken is set when the request asked to trust the device.
	// A browser also gets it as an httpOnly cookie; other clients send it
	// back in the X-Authsome-Trusted-Device header when they sign in.
	TrustedDeviceToken     string `json:"trusted_device_token,omitempty"      description:"Opaque token that lets this device skip the second factor; send it as X-Authsome-Trusted-Device on sign-in"`
	TrustedDeviceExpiresAt string `json:"trusted_device_expires_at,omitempty" description:"When the device stops being trusted (RFC3339)"`
}

// DisableRequest is an empty request for disabling MFA.
type DisableRequest struct {
	// Code re-proves possession of the second factor: a current TOTP code or
	// an unused recovery code. Required in practice, but optional to the
	// binder so an unauthenticated caller still gets 401 rather than a 400
	// that reports a missing field before authentication was even checked.
	Code string `json:"code,omitempty"`
}

// DisableResponse is returned when MFA is disabled.
type DisableResponse struct {
	Status string `json:"status"`
}

// EnrollmentRequest binds GET /mfa/enrollment. It takes nothing: the
// enrollment it reports is always the caller's own.
type EnrollmentRequest struct{}

// EnrolledFactor is one second factor the caller has set up.
type EnrolledFactor struct {
	Method     string    `json:"method" description:"Factor kind: totp or sms"`
	Verified   bool      `json:"verified" description:"Whether the factor completed its first verification"`
	EnrolledAt time.Time `json:"enrolled_at" description:"When the factor was set up"`
}

// EnrollmentResponse answers "is two-step on for me, and with what".
//
// It carries no secret, no recovery code and nothing about anyone else. The
// remaining recovery code count is a number rather than the codes, because
// an account settings page needs to warn someone who is running low without
// showing codes that only exist to be shown once.
type EnrollmentResponse struct {
	Enabled                bool             `json:"enabled" description:"Whether any factor is verified"`
	Factors                []EnrolledFactor `json:"factors" description:"Every factor set up, verified or not"`
	RecoveryCodesRemaining int              `json:"recovery_codes_remaining" description:"How many unused recovery codes are left"`
	TrustedDevices         int              `json:"trusted_devices" description:"How many devices currently skip the second factor"`
}

// RecoveryVerifyRequest is the request body for verifying an MFA recovery code.
type RecoveryVerifyRequest struct {
	Code string `json:"code"`
}

// RecoveryVerifyResponse is returned on successful recovery code verification.
type RecoveryVerifyResponse struct {
	ChallengePassed bool `json:"challenge_passed"`
	CodesRemaining  int  `json:"codes_remaining"`
}

// RecoveryRegenerateRequest regenerates recovery codes. Code is a current
// TOTP code; it is needed unless the calling session already passed a second
// factor.
type RecoveryRegenerateRequest struct {
	Code string `json:"code,omitempty"`
}

// RecoveryRegenerateResponse is returned with new recovery codes.
type RecoveryRegenerateResponse struct {
	Codes []string `json:"codes"`
}

// SMSSendRequest is the request body for sending an SMS verification code.
type SMSSendRequest struct {
	Phone string `json:"phone,omitempty"` // Optional: override the enrolled phone
}

// SMSSendResponse is returned when an SMS code is sent.
type SMSSendResponse struct {
	Sent      bool   `json:"sent"`
	ExpiresIn int    `json:"expires_in_seconds"`
	Phone     string `json:"phone_masked"`
}

// SMSVerifyRequest is the request body for verifying an SMS code.
type SMSVerifyRequest struct {
	Code string `json:"code"`
}

// SMSVerifyResponse is returned on successful SMS code verification.
type SMSVerifyResponse struct {
	Verified bool   `json:"verified"`
	Method   string `json:"method"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

// handleEnroll starts MFA enrollment for the authenticated user.
func (p *Plugin) handleEnroll(ctx forge.Context, req *EnrollRequest) (*EnrollResponse, error) {
	// The route refuses a delegated session too. Checked here as well so a
	// mount without that guard still fails closed.
	if middleware.RefuseDelegated(ctx.Context()) != nil {
		return nil, forge.Forbidden(middleware.DelegatedSessionMessage)
	}
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Method == "" {
		req.Method = "totp"
	}

	if req.Method != "totp" && req.Method != "sms" {
		return nil, forge.BadRequest("unsupported MFA method: supported methods are totp and sms")
	}

	// Adding a factor is a change to how the account is protected, so it
	// needs more than a session that may have been stolen: the factor the
	// user already holds, or, before any exists, a recent sign-in.
	if req.Method == "sms" {
		if err := p.requireStepUp(ctx, userID, req.Code); err != nil {
			return nil, err
		}
		return p.enrollSMS(ctx, userID, req)
	}

	// Check if already enrolled. Re-enrolling the factor the user already
	// holds changes nothing, so it is answered before the gate.
	existing, _ := p.store.GetEnrollment(ctx.Context(), userID, "totp") //nolint:errcheck // best-effort lookup
	if existing != nil && existing.Verified {
		return nil, forge.NewHTTPError(http.StatusConflict, "MFA already enrolled and verified")
	}
	if err := p.requireStepUp(ctx, userID, req.Code); err != nil {
		return nil, err
	}

	// Generate TOTP key
	u, hasUser := middleware.UserFrom(ctx.Context())
	accountName := "user"
	if hasUser && u != nil && u.Email != "" {
		accountName = u.Email
	}

	key, err := GenerateTOTPKey(TOTPConfig{
		Issuer:      p.config.Issuer,
		AccountName: accountName,
	})
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to generate TOTP key: %w", err))
	}

	now := time.Now()
	enrollment := &Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// If there's an existing unverified enrollment, delete it first
	if existing != nil {
		_ = p.store.DeleteEnrollment(ctx.Context(), existing.ID) //nolint:errcheck // best-effort cleanup
	}

	if err := p.store.CreateEnrollment(ctx.Context(), enrollment); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create enrollment: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.enrolled", "", map[string]string{"user_id": userID.String(), "method": enrollment.Method})
	p.emitHook(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "")

	return &EnrollResponse{
		ID:         enrollment.ID.String(),
		Method:     "totp",
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	}, nil
}

// totpReplayTTL bounds how long a used TOTP time-step is remembered. It must
// exceed the acceptance window (period × (2·skew+1) ≈ 90s) with margin.
const totpReplayTTL = 120 * time.Second

// totpStepUsed reports whether the given TOTP time-step has already been
// accepted for this enrollment, and records it as used otherwise. Backed by the
// ephemeral ceremony store (TTL-bounded), so no persistent schema is required.
func (p *Plugin) totpStepUsed(ctx context.Context, enrollmentID id.MFAID, step int64) bool {
	key := fmt.Sprintf("mfa:totp:used:%s:%d", enrollmentID.String(), step)
	// One atomic claim: two submissions of the same code racing on two
	// replicas cannot both pass, which a read followed by a write allowed.
	claimed, err := p.ceremonies.SetNX(ctx, key, []byte{1}, totpReplayTTL)
	if err != nil {
		// The marker could not be written, so the step cannot be proven
		// unused. Refuse rather than accept a possible replay.
		return true
	}
	return !claimed
}

// handleVerify verifies a TOTP code against the user's enrollment.
func (p *Plugin) handleVerify(ctx forge.Context, req *VerifyMFARequest) (*VerifyMFAResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Code == "" {
		return nil, forge.BadRequest("code required")
	}

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil {
		return nil, forge.NotFound("no MFA enrollment found")
	}

	ok2, step := ValidateTOTPStep(req.Code, enrollment.Secret)
	if !ok2 {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodTOTP, telemetry.MFAInvalid)
		return nil, forge.Unauthorized("invalid TOTP code")
	}
	if p.totpStepUsed(ctx.Context(), enrollment.ID, step) {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodTOTP, telemetry.MFAReplayed)
		return nil, forge.Unauthorized("TOTP code already used")
	}
	p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodTOTP, telemetry.MFAOK)

	// The code was good: this session has now passed a second factor.
	if err := p.markSessionVerified(ctx.Context()); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to record the verified session: %w", err))
	}

	// Mark as verified if this is the first successful verification
	if !enrollment.Verified {
		enrollment.Verified = true
		enrollment.UpdatedAt = time.Now()
		if err := p.store.UpdateEnrollment(ctx.Context(), enrollment); err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to verify enrollment: %w", err))
		}

		// Generate recovery codes on first successful verification
		codes, plaintexts, err := GenerateRecoveryCodes(userID, DefaultRecoveryCodeCount)
		if err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to generate recovery codes: %w", err))
		}

		// Delete any old codes for this user, then store new ones
		_ = p.store.DeleteRecoveryCodes(ctx.Context(), userID) //nolint:errcheck // best-effort cleanup
		if err := p.store.CreateRecoveryCodes(ctx.Context(), codes); err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to store recovery codes: %w", err))
		}

		p.audit(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
		p.relayEvent(ctx.Context(), "auth.mfa.verified", "", map[string]string{"user_id": userID.String(), "method": "totp"})
		p.revokeOtherSessions(ctx.Context(), userID)

		return &VerifyMFAResponse{
			Verified:      true,
			Method:        "totp",
			RecoveryCodes: plaintexts,
		}, nil
	}

	return &VerifyMFAResponse{
		Verified: true,
		Method:   "totp",
	}, nil
}

// handleChallenge completes the sign-in MFA round-trip. Accepts the
// ticket previously returned by /v1/signin (or any other session-
// issuing endpoint that hit the MFARequired gate) plus the user's
// TOTP code. On success, consumes the ticket and issues a real
// session via Engine.IssueSession with MFAJustVerified=true so the
// gate doesn't fire a second time.
//
// Endpoint is unauthenticated by design: the user is mid-signin and
// has no valid session yet — that's the whole point of the gate. The
// ticket itself is the auth credential for this single endpoint, and
// it's single-use after a correct code.
//
// On bad code: ticket is NOT consumed, so the user can retry within
// the 5-minute TTL. Brute-force protection comes from the existing
// rate-limit middleware on this route, not from one-shot ticket
// invalidation.
//
// Type-asserts p.engine to *authsome.Engine to reach IssueSession /
// LoadMFATicket / ConsumeMFATicket. The plugin.Engine interface
// deliberately doesn't surface those — adding them would force a
// public surface change every plugin would inherit. The assertion is
// safe in production wiring; in test wiring without a concrete engine
// the handler returns 501.
func (p *Plugin) handleChallenge(ctx forge.Context, req *ChallengeRequest) (*ChallengeResponse, error) {
	if req.Ticket == "" || req.Code == "" {
		return nil, forge.BadRequest("mfa_ticket and code required")
	}

	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil, forge.NewHTTPError(http.StatusNotImplemented,
			"mfa challenge requires the concrete authsome.Engine; this plugin was wired against a stub")
	}

	payload, err := eng.LoadMFATicket(ctx.Context(), req.Ticket)
	if err != nil {
		p.tel.MFAVerification(telemetry.MFAFlowSignIn, telemetry.MFAMethodTOTP, telemetry.MFAExpired)
		return nil, forge.Unauthorized("invalid or expired ticket")
	}
	userID := payload.UserID

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil || enrollment == nil {
		return nil, forge.Unauthorized("no MFA enrollment for user")
	}
	if !enrollment.Verified {
		return nil, forge.Unauthorized("MFA enrollment not yet verified")
	}

	// Accept either a TOTP code or one of the user's recovery codes. Recovery
	// codes exist precisely for the case where the authenticator is
	// unavailable, which is this endpoint — before the fix they could only be
	// redeemed from an already-authenticated session, i.e. never when actually
	// needed.
	//
	// A code that matches neither counts as method=totp, which is what the
	// endpoint reports back.
	method := "totp"
	ok2, step := ValidateTOTPStep(req.Code, enrollment.Secret)
	switch {
	case ok2:
		if p.totpStepUsed(ctx.Context(), enrollment.ID, step) {
			p.tel.MFAVerification(telemetry.MFAFlowSignIn, telemetry.MFAMethodTOTP, telemetry.MFAReplayed)
			return nil, forge.Unauthorized("TOTP code already used")
		}
	default:
		used, recErr := p.consumeRecoveryCode(ctx.Context(), userID, req.Code)
		if recErr != nil {
			p.tel.MFAVerification(telemetry.MFAFlowSignIn, telemetry.MFAMethodRecovery, telemetry.MFAError)
			return nil, recErr
		}
		if !used {
			// Charge the attempt. The ticket is partial auth over a 6-digit
			// secret, so it must bound its own guesses — the route limiter
			// is absent entirely when rate limiting is disabled.
			exhausted, failErr := eng.FailMFATicket(ctx.Context(), req.Ticket)
			if failErr != nil {
				p.logger.Warn("mfa: record failed challenge attempt",
					log.String("ticket_prefix", safePrefix(req.Ticket)),
					log.Error(failErr),
				)
			}
			if exhausted {
				p.tel.MFAVerification(telemetry.MFAFlowSignIn, telemetry.MFAMethodTOTP, telemetry.MFAExhausted)
				return nil, forge.Unauthorized("too many invalid codes; sign in again")
			}
			p.tel.MFAVerification(telemetry.MFAFlowSignIn, telemetry.MFAMethodTOTP, telemetry.MFAInvalid)
			return nil, forge.Unauthorized("invalid TOTP code")
		}
		method = "recovery"
	}
	p.tel.MFAVerification(telemetry.MFAFlowSignIn, method, telemetry.MFAOK)

	// Code is good. Consume the ticket (single-use) before minting the
	// real session so a race that re-uses the ticket can't double-spend.
	if consumeErr := eng.ConsumeMFATicket(ctx.Context(), req.Ticket); consumeErr != nil {
		p.logger.Warn("mfa: consume ticket failed",
			log.String("ticket_prefix", safePrefix(req.Ticket)),
			log.Error(consumeErr),
		)
		return nil, middleware.InternalError(ctx, consumeErr)
	}

	u, err := eng.GetUser(ctx.Context(), userID)
	if err != nil || u == nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("mfa: load user %s after challenge: %w", userID, err))
	}

	result, issueErr := eng.IssueSession(ctx.Context(), &authsome.IssueSessionRequest{
		User:            u,
		AppID:           payload.AppID,
		EnvID:           payload.EnvID,
		AuthMethod:      payload.AuthMethod + "+mfa",
		IPAddress:       ctx.Request().RemoteAddr,
		UserAgent:       ctx.Request().UserAgent(),
		MFAJustVerified: true,
		// Carried across the ceremony by the ticket. Sign-in proved this
		// thumbprint; without replaying it here the session minted after
		// the second factor would be unbound, which on a required-mode app
		// means enabling MFA quietly turns DPoP off for that user.
		DPoPJKT: payload.DPoPJKT,
	})
	if issueErr != nil {
		// A DPoP refusal is a 400 the client can act on. The app mandates a
		// binding and this ticket carries none, which happens when the mode
		// was raised while the ticket was in flight. It renders its own RFC
		// 9449 body, so bubble it rather than flattening it into a 500 that
		// tells the client nothing about what to retry.
		if errors.Is(issueErr, authsome.ErrDPoPBindingRequired) {
			return nil, issueErr
		}
		return nil, middleware.InternalError(ctx, fmt.Errorf("mfa: issue session: %w", issueErr))
	}

	appIDStr := payload.AppID.String()
	p.audit(ctx.Context(), hook.ActionMFAChallenge, "mfa", enrollment.ID.String(),
		userID.String(), appIDStr, bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.challenged", appIDStr, map[string]string{
		"user_id": userID.String(),
		"method":  method,
	})
	p.emitHook(ctx.Context(), hook.ActionMFAChallenge, "mfa", enrollment.ID.String(),
		userID.String(), appIDStr)

	expires := ""
	if result.Session != nil {
		expires = result.Session.ExpiresAt.UTC().Format(time.RFC3339)
		// The first leg of sign-in set no cookie, so a browser that keeps
		// its session in the cookie is only signed in once this one is.
		authsome.SetSessionCookie(ctx.Response(), ctx.Request(), eng.Settings(), payload.AppID.String(),
			result.Session.Token, result.Session.ExpiresAt)
	}

	resp := &ChallengeResponse{
		User:         result.User,
		SessionToken: result.Session.Token,
		RefreshToken: result.Session.RefreshToken,
		ExpiresAt:    expires,
	}
	if req.TrustDevice {
		p.trustChallengedDevice(ctx, eng, payload.AppID, userID, resp)
	}
	return resp, nil
}

// requireStepUp gates a factor change. A user who already holds a verified
// TOTP enrolment must present a current code (or a recovery code); a user
// with no verified factor yet must have signed in within StepUpWindow.
func (p *Plugin) requireStepUp(ctx forge.Context, userID id.UserID, code string) error {
	existing, _ := p.store.GetEnrollment(ctx.Context(), userID, "totp") //nolint:errcheck // best-effort lookup
	if existing != nil && existing.Verified {
		if code == "" {
			return forge.NewHTTPError(http.StatusUnauthorized, "code required: prove your current factor to change it")
		}
		return p.verifyStepUp(ctx.Context(), userID, existing, code)
	}
	var lookup middleware.SessionLookup
	if p.engine != nil && p.engine.Store() != nil {
		lookup = p.engine.Store().GetSession
	}
	if !middleware.SessionIsFresh(ctx.Context(), time.Now(), p.config.StepUpWindow, lookup) {
		return forge.Forbidden("recent sign-in required: sign in again to change your factors")
	}
	return nil
}

// sessionRevoker is the slice of the engine a factor change needs.
type sessionRevoker interface {
	RevokeOtherUserSessions(ctx context.Context, userID id.UserID, keep id.SessionID) error
}

// sessionStamper is the slice of the engine that records a verified session.
type sessionStamper interface {
	MarkSessionMFAVerified(ctx context.Context, sessionID id.SessionID, at time.Time) error
}

// sessionStamped reports whether the request's session passed a second
// factor. A session read from its row carries the stamp; one that came in on
// a JWT is read by id. Anything that cannot be read is not stamped.
func (p *Plugin) sessionStamped(ctx context.Context) bool {
	if s, ok := middleware.SessionFrom(ctx); ok && s != nil && !s.CreatedAt.IsZero() {
		return !s.MFAVerifiedAt.IsZero()
	}
	sid, ok := middleware.SessionIDFrom(ctx)
	if !ok || sid.IsNil() || p.engine == nil || p.engine.Store() == nil {
		return false
	}
	s, err := p.engine.Store().GetSession(ctx, sid)
	return err == nil && s != nil && !s.MFAVerifiedAt.IsZero()
}

// clearSessionVerified removes the request's own session's stamp, after the
// factor it proved has been removed. A request with no session records
// nothing.
func (p *Plugin) clearSessionVerified(ctx context.Context) error {
	if p.stamper == nil {
		return nil
	}
	sid, ok := middleware.SessionIDFrom(ctx)
	if !ok || sid.IsNil() {
		return nil
	}
	return p.stamper.MarkSessionMFAVerified(ctx, sid, time.Time{})
}

// markSessionVerified records that the request's own session just proved a
// second factor, so a host enforcing two-step reads it as verified. A request
// with no session (or no engine) records nothing.
func (p *Plugin) markSessionVerified(ctx context.Context) error {
	if p.stamper == nil {
		return nil
	}
	sid, ok := middleware.SessionIDFrom(ctx)
	if !ok || sid.IsNil() {
		return nil
	}
	return p.stamper.MarkSessionMFAVerified(ctx, sid, time.Now())
}

// revokeOtherSessions signs the user out everywhere but the session that
// made the change. Enrolling or removing a factor is the moment to drop any
// session an attacker may already hold; a failure is logged, not returned,
// because the factor change itself has already been persisted.
func (p *Plugin) revokeOtherSessions(ctx context.Context, userID id.UserID) {
	if p.revoker == nil {
		return
	}
	keep, _ := middleware.SessionIDFrom(ctx)
	if err := p.revoker.RevokeOtherUserSessions(ctx, userID, keep); err != nil && p.logger != nil {
		p.logger.Warn("mfa: revoke other sessions failed",
			log.String("user_id", userID.String()),
			log.String("error", err.Error()),
		)
	}
}

// verifyStepUp re-proves possession of the second factor for a sensitive
// change, accepting a current TOTP code or a recovery code. Replay protection
// applies as it does at sign-in, so the same code cannot be reused to authorise
// a second action within its window.
func (p *Plugin) verifyStepUp(ctx context.Context, userID id.UserID, enrollment *Enrollment, code string) error {
	if ok, step := ValidateTOTPStep(code, enrollment.Secret); ok {
		if p.totpStepUsed(ctx, enrollment.ID, step) {
			return forge.Unauthorized("TOTP code already used")
		}
		return nil
	}
	used, err := p.consumeRecoveryCode(ctx, userID, code)
	if err != nil {
		return err
	}
	if !used {
		return forge.Unauthorized("invalid code")
	}
	return nil
}

// consumeRecoveryCode matches plaintext against the user's unused recovery
// codes and burns the first hit. Reports whether a code matched; a false
// return with a nil error means "no match", not a failure.
//
// Every unused code is compared even after a match is found, so the work done
// is the same whether the code was valid or not — bcrypt is slow enough that
// short-circuiting would make "wrong code" measurably faster than "right one".
func (p *Plugin) consumeRecoveryCode(ctx context.Context, userID id.UserID, plaintext string) (bool, error) {
	codes, err := p.store.GetRecoveryCodes(ctx, userID)
	if err != nil {
		return false, middleware.InternalErrorCtx(ctx, fmt.Errorf("mfa: get recovery codes: %w", err))
	}

	var matched *RecoveryCode
	for _, c := range codes {
		if c.Used {
			continue
		}
		if VerifyRecoveryCode(plaintext, c) && matched == nil {
			matched = c
		}
	}
	if matched == nil {
		return false, nil
	}

	if consumeErr := p.store.ConsumeRecoveryCode(ctx, matched.ID); consumeErr != nil {
		return false, middleware.InternalErrorCtx(ctx, fmt.Errorf("mfa: consume recovery code: %w", consumeErr))
	}
	p.audit(ctx, hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx, hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "")
	return true, nil
}

// safePrefix returns the first 8 chars of a ticket so it can appear
// in logs without leaking the secret in full.
func safePrefix(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8] + "…"
}

// handleDisable removes MFA enrollment for the authenticated user.
// handleGetEnrollment answers whether two-step sign-in is on for the caller.
//
// A delegated session may read this. It is the one MFA route that changes
// nothing, and an agent acting for a user has a legitimate reason to know
// whether that user is protected.
func (p *Plugin) handleGetEnrollment(ctx forge.Context, _ *EnrollmentRequest) (*EnrollmentResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	resp := &EnrollmentResponse{Factors: []EnrolledFactor{}}
	for _, method := range []string{"totp", "sms"} {
		e, err := p.store.GetEnrollment(ctx.Context(), userID, method)
		if err != nil || e == nil {
			continue
		}
		resp.Factors = append(resp.Factors, EnrolledFactor{
			Method:     e.Method,
			Verified:   e.Verified,
			EnrolledAt: e.CreatedAt,
		})
		if e.Verified {
			resp.Enabled = true
		}
	}

	// A failure to read recovery codes leaves the count at zero rather than
	// failing the whole answer. The enrollment state is the part that matters
	// and it is already known by this point.
	if codes, err := p.store.GetRecoveryCodes(ctx.Context(), userID); err == nil {
		for _, c := range codes {
			if !c.Used {
				resp.RecoveryCodesRemaining++
			}
		}
	}
	// Likewise the trusted device count.
	if devices, err := p.liveTrustedDevices(ctx.Context(), userID); err == nil {
		resp.TrustedDevices = len(devices)
	}

	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleDisable(ctx forge.Context, req *DisableRequest) (*DisableResponse, error) {
	// The route refuses a delegated session too. Checked here as well so a
	// mount without that guard still fails closed.
	if middleware.RefuseDelegated(ctx.Context()) != nil {
		return nil, forge.Forbidden(middleware.DelegatedSessionMessage)
	}
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil {
		return nil, forge.NotFound("no MFA enrollment found")
	}

	// Turning MFA off is exactly the action MFA exists to protect, so it needs
	// a fresh proof of possession rather than session auth alone. Without this
	// a stolen or hijacked session can strip the second factor silently and
	// keep long-term access.
	if req == nil || req.Code == "" {
		return nil, forge.BadRequest("code required to disable MFA")
	}
	if enrollment.Verified {
		if err := p.verifyStepUp(ctx.Context(), userID, enrollment, req.Code); err != nil {
			return nil, err
		}
	}

	// Devices trusted to skip the factor go before the factor does, so a
	// failure here leaves MFA on rather than off with trust still standing.
	if err := p.store.DeleteTrustedDevices(ctx.Context(), userID); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to remove trusted devices: %w", err))
	}
	authsome.ClearTrustedDeviceCookie(ctx.Response(), ctx.Request())

	if err := p.store.DeleteEnrollment(ctx.Context(), enrollment.ID); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to remove enrollment: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFADisable, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.disabled", "", map[string]string{"user_id": userID.String(), "method": "totp"})
	p.emitHook(ctx.Context(), hook.ActionMFADisable, "mfa", enrollment.ID.String(), userID.String(), "")
	p.revokeOtherSessions(ctx.Context(), userID)
	// The factor this session may have proved is gone, so its stamp goes
	// too: a host enforcing two-step must ask again.
	if err := p.clearSessionVerified(ctx.Context()); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to clear the verified session: %w", err))
	}

	return &DisableResponse{Status: "mfa disabled"}, nil
}

// ──────────────────────────────────────────────────
// Recovery code handlers
// ──────────────────────────────────────────────────

// handleRecoveryVerify verifies a recovery code as a substitute for TOTP during MFA challenge.
func (p *Plugin) handleRecoveryVerify(ctx forge.Context, req *RecoveryVerifyRequest) (*RecoveryVerifyResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Code == "" {
		return nil, forge.BadRequest("recovery code required")
	}

	// Ensure user has verified MFA enrollment
	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil || !enrollment.Verified {
		return nil, forge.BadRequest("no verified MFA enrollment found")
	}

	// Fetch all unused recovery codes
	codes, err := p.store.GetRecoveryCodes(ctx.Context(), userID)
	if err != nil {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodRecovery, telemetry.MFAError)
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to get recovery codes: %w", err))
	}

	// Try to match the provided code
	var matched *RecoveryCode
	unused := 0
	for _, c := range codes {
		if !c.Used {
			unused++
			if matched == nil && VerifyRecoveryCode(req.Code, c) {
				matched = c
				unused-- // this one is about to be consumed
			}
		}
	}

	if matched == nil {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodRecovery, telemetry.MFAInvalid)
		return nil, forge.Unauthorized("invalid recovery code")
	}

	// Consume the matched code (one-time use)
	if err := p.store.ConsumeRecoveryCode(ctx.Context(), matched.ID); err != nil {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodRecovery, telemetry.MFAError)
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to consume recovery code: %w", err))
	}
	p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodRecovery, telemetry.MFAOK)

	// A spent recovery code is a second-factor proof for this session.
	if err := p.markSessionVerified(ctx.Context()); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to record the verified session: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "")

	return &RecoveryVerifyResponse{
		ChallengePassed: true,
		CodesRemaining:  unused,
	}, nil
}

// handleRecoveryRegenerate generates a fresh set of recovery codes, replacing any existing ones.
func (p *Plugin) handleRecoveryRegenerate(ctx forge.Context, req *RecoveryRegenerateRequest) (*RecoveryRegenerateResponse, error) {
	// The route refuses a delegated session too. Checked here as well so a
	// mount without that guard still fails closed.
	if middleware.RefuseDelegated(ctx.Context()) != nil {
		return nil, forge.Forbidden(middleware.DelegatedSessionMessage)
	}
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	// Ensure user has verified MFA enrollment
	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil || !enrollment.Verified {
		return nil, forge.BadRequest("no verified MFA enrollment found")
	}

	// Replacing the codes wipes the account's way back in, so the caller has
	// to have proved a second factor: a session that already did, or a
	// current TOTP code sent with the request. A recovery code verified on
	// this session stamps it, so without this an unverified session could
	// regenerate, verify a fresh code and stamp itself.
	if !p.sessionStamped(ctx.Context()) {
		if req == nil || req.Code == "" {
			return nil, forge.NewHTTPError(http.StatusUnauthorized, "code required: prove your current factor to regenerate recovery codes")
		}
		ok, step := ValidateTOTPStep(req.Code, enrollment.Secret)
		if !ok {
			return nil, forge.Unauthorized("invalid TOTP code")
		}
		if p.totpStepUsed(ctx.Context(), enrollment.ID, step) {
			return nil, forge.Unauthorized("TOTP code already used")
		}
	}

	// Generate new codes
	codes, plaintexts, err := GenerateRecoveryCodes(userID, DefaultRecoveryCodeCount)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to generate recovery codes: %w", err))
	}

	// Delete old codes, store new ones
	_ = p.store.DeleteRecoveryCodes(ctx.Context(), userID) //nolint:errcheck // best-effort cleanup
	if err := p.store.CreateRecoveryCodes(ctx.Context(), codes); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to store recovery codes: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFARecoveryRegenerated, "mfa", "", userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFARecoveryRegenerated, "mfa", "", userID.String(), "")

	return &RecoveryRegenerateResponse{
		Codes: plaintexts,
	}, nil
}

// ──────────────────────────────────────────────────
// HasMFA checks if a user has verified MFA enrollment.
// ──────────────────────────────────────────────────

// HasMFA returns true if the user has a verified MFA enrollment.
func (p *Plugin) HasMFA(ctx context.Context, userID id.UserID) bool {
	// Check TOTP enrollment
	enrollment, err := p.store.GetEnrollment(ctx, userID, "totp")
	if err == nil && enrollment.Verified {
		return true
	}
	// Check SMS enrollment
	enrollment, err = p.store.GetEnrollment(ctx, userID, "sms")
	if err == nil && enrollment.Verified {
		return true
	}
	return false
}

// Enrolled reports whether the user has a verified TOTP enrollment, the
// factor /v1/mfa/challenge checks at sign-in. SMS isn't counted: the
// challenge can't check an SMS code, so gating on it would lock the user
// out. Unlike HasMFA it returns lookup failures, so the engine's sign-in
// gate can refuse rather than skip the second factor.
func (p *Plugin) Enrolled(ctx context.Context, userID id.UserID) (bool, error) {
	if p.store == nil {
		return false, nil
	}
	enrollment, err := p.store.GetEnrollment(ctx, userID, "totp")
	if errors.Is(err, ErrEnrollmentNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("mfa: get totp enrollment: %w", err)
	}
	return enrollment.Verified, nil
}

// VerifyStepUp checks a current TOTP code or an unused recovery code from a
// user with two-step on, for a change that adds a way into the account. A
// recovery code is spent. A user without two-step on passes with no code.
func (p *Plugin) VerifyStepUp(ctx context.Context, userID id.UserID, code string) error {
	if p.store == nil {
		return nil
	}
	enrollment, err := p.store.GetEnrollment(ctx, userID, "totp")
	if errors.Is(err, ErrEnrollmentNotFound) || (err == nil && !enrollment.Verified) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mfa: get totp enrollment: %w", err)
	}
	if code == "" {
		return forge.NewHTTPError(http.StatusUnauthorized, "code required: two-step verification is on, so enter a code from your authenticator app")
	}
	return p.verifyStepUp(ctx, userID, enrollment, code)
}

// ──────────────────────────────────────────────────
// SMS MFA Handlers
// ──────────────────────────────────────────────────

// enrollSMS handles MFA enrollment via SMS.
func (p *Plugin) enrollSMS(ctx forge.Context, userID id.UserID, req *EnrollRequest) (*EnrollResponse, error) {
	if p.sms == nil {
		return nil, forge.BadRequest("SMS MFA is not configured")
	}

	if req.Phone == "" {
		return nil, forge.BadRequest("phone number is required for SMS MFA")
	}

	// Check if already enrolled
	existing, _ := p.store.GetEnrollment(ctx.Context(), userID, "sms") //nolint:errcheck // best-effort lookup
	if existing != nil && existing.Verified {
		return nil, forge.NewHTTPError(http.StatusConflict, "SMS MFA already enrolled and verified")
	}

	now := time.Now()
	enrollment := &Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "sms",
		Secret:    req.Phone, // Store phone number as the "secret"
		Verified:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Delete existing unverified enrollment
	if existing != nil {
		_ = p.store.DeleteEnrollment(ctx.Context(), existing.ID) //nolint:errcheck // best-effort cleanup
	}

	if err := p.store.CreateEnrollment(ctx.Context(), enrollment); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to create SMS enrollment: %w", err))
	}

	// Send initial verification code
	challenge, err := SendSMSChallenge(ctx.Context(), p.sms, req.Phone)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to send SMS code: %w", err))
	}

	challengeData, _ := json.Marshal(challenge)                                                //nolint:errcheck // marshaling known types
	_ = p.ceremonies.Set(ctx.Context(), "mfa:sms:"+userID.String(), challengeData, smsCodeTTL) //nolint:errcheck // best-effort cache

	p.audit(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.enrolled", "", map[string]string{"user_id": userID.String(), "method": enrollment.Method})
	p.emitHook(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "")

	return &EnrollResponse{
		ID:     enrollment.ID.String(),
		Method: "sms",
	}, nil
}

// handleSMSSend sends a new SMS verification code to the enrolled phone number.
func (p *Plugin) handleSMSSend(ctx forge.Context, req *SMSSendRequest) (*SMSSendResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	if p.sms == nil {
		return nil, forge.BadRequest("SMS MFA is not configured")
	}

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "sms")
	if err != nil {
		return nil, forge.NotFound("no SMS MFA enrollment found")
	}

	// Once the number is verified a code only ever goes there: an override
	// would let a session that has only the password text itself a code
	// and pass the second factor. Before then, an override is the
	// enrollment flow pointing the pending enrollment at a corrected
	// number, so it needs the same step-up as enrolling, and the
	// enrollment moves with it so the number confirmed is the one that
	// got the code.
	phone := enrollment.Secret // Phone stored as the enrollment secret
	if req.Phone != "" && req.Phone != phone && !enrollment.Verified {
		if stepErr := p.requireStepUp(ctx, userID, ""); stepErr != nil {
			return nil, stepErr
		}
		enrollment.Secret = req.Phone
		enrollment.UpdatedAt = time.Now()
		if updErr := p.store.UpdateEnrollment(ctx.Context(), enrollment); updErr != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to update SMS enrollment: %w", updErr))
		}
		phone = req.Phone
	}

	challenge, err := SendSMSChallenge(ctx.Context(), p.sms, phone)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to send SMS code: %w", err))
	}

	challengeData, _ := json.Marshal(challenge)                                                //nolint:errcheck // marshaling known types
	_ = p.ceremonies.Set(ctx.Context(), "mfa:sms:"+userID.String(), challengeData, smsCodeTTL) //nolint:errcheck // best-effort cache

	// Mask phone number for response
	masked := maskPhone(phone)

	return &SMSSendResponse{
		Sent:      true,
		ExpiresIn: int(smsCodeTTL.Seconds()),
		Phone:     masked,
	}, nil
}

// handleSMSVerify verifies an SMS code for enrollment confirmation or challenge.
func (p *Plugin) handleSMSVerify(ctx forge.Context, req *SMSVerifyRequest) (*SMSVerifyResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Code == "" {
		return nil, forge.BadRequest("code required")
	}

	challengeData, err := p.ceremonies.Get(ctx.Context(), "mfa:sms:"+userID.String())
	if err != nil {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodSMS, telemetry.MFAExpired)
		return nil, forge.BadRequest("no pending SMS challenge; send a code first")
	}

	var challenge SMSChallenge
	if unmarshalErr := json.Unmarshal(challengeData, &challenge); unmarshalErr != nil {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodSMS, telemetry.MFAError)
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to parse challenge: %w", unmarshalErr))
	}

	if !ValidateSMSCode(req.Code, &challenge) {
		p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodSMS, telemetry.MFAInvalid)
		return nil, forge.Unauthorized("invalid or expired SMS code")
	}
	p.tel.MFAVerification(telemetry.MFAFlowVerify, telemetry.MFAMethodSMS, telemetry.MFAOK)

	// Remove the used challenge
	_ = p.ceremonies.Delete(ctx.Context(), "mfa:sms:"+userID.String()) //nolint:errcheck // best-effort cleanup

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "sms")
	if err != nil {
		return nil, forge.NotFound("no SMS MFA enrollment found")
	}
	if !enrollment.Verified {
		// The code confirms the number, which makes it a factor from now
		// on. It is not a factor the session already held, so the session
		// is not stamped: it proves the factor with a fresh code next.
		enrollment.Verified = true
		enrollment.UpdatedAt = time.Now()
		if err := p.store.UpdateEnrollment(ctx.Context(), enrollment); err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("failed to verify SMS enrollment: %w", err))
		}
		return &SMSVerifyResponse{Verified: true, Method: "sms"}, nil
	}

	// The code went to the verified number: this session has now passed a
	// second factor.
	if err := p.markSessionVerified(ctx.Context()); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to record the verified session: %w", err))
	}

	return &SMSVerifyResponse{
		Verified: true,
		Method:   "sms",
	}, nil
}

// ──────────────────────────────────────────────────
// Audit / Relay / Hook helpers
// ──────────────────────────────────────────────────

//nolint:unparam // consistent audit/event API
func (p *Plugin) audit(ctx context.Context, action, resource, resourceID, actorID, tenant, outcome string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    outcome,
		Severity:   bridge.SeverityInfo,
		Category:   "auth",
	})
}

func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	})
}

//nolint:unparam // consistent audit/event API
func (p *Plugin) emitHook(ctx context.Context, action, resource, resourceID, actorID, tenant string) {
	if p.hooks == nil {
		return
	}
	p.hooks.Emit(ctx, &hook.Event{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
	})
}

// maskPhone masks all but the last 4 digits of a phone number.
func maskPhone(phone string) string { return mask.Phone(phone) }

// OnBeforeUserDelete removes a deleted user's MFA enrollments, recovery codes
// and trusted devices: a second factor for an account that no longer exists
// is a secret with no owner.
func (p *Plugin) OnBeforeUserDelete(ctx context.Context, userID id.UserID) error {
	if p.store == nil {
		return nil
	}
	enrollments, err := p.store.ListEnrollments(ctx, userID)
	if err != nil {
		return fmt.Errorf("mfa: list enrollments for erasure: %w", err)
	}
	for _, e := range enrollments {
		if err := p.store.DeleteEnrollment(ctx, e.ID); err != nil {
			return fmt.Errorf("mfa: delete enrollment %s: %w", e.ID, err)
		}
	}
	if err := p.store.DeleteRecoveryCodes(ctx, userID); err != nil {
		return fmt.Errorf("mfa: delete recovery codes: %w", err)
	}
	if err := p.store.DeleteTrustedDevices(ctx, userID); err != nil {
		return fmt.Errorf("mfa: delete trusted devices: %w", err)
	}
	return nil
}
