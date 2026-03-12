package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
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
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	return settings.RegisterTyped(m, "mfa", SettingIssuer)
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
	return &Plugin{
		config:     c,
		ceremonies: ceremony.NewMemory(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "mfa" }

// OnInit is called during engine initialization. It auto-discovers the SMS
// sender, chronicle, relay, hooks, logger, and ceremony store from the engine.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type smsSenderGetter interface {
		SMSSender() bridge.SMSSender
	}
	if sg, ok := engine.(smsSenderGetter); ok {
		p.sms = sg.SMSSender()
	}

	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	type ceremonyGetter interface {
		CeremonyStore() ceremony.Store
	}
	if cg, ok := engine.(ceremonyGetter); ok {
		p.ceremonies = cg.CeremonyStore()
	}
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
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
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("mfa: expected forge.Router, got %T", r)
	}

	g := router.Group("/v1/auth/mfa", forge.WithGroupTags("MFA"))

	if err := g.POST("/enroll", p.handleEnroll,
		forge.WithSummary("Enroll in MFA"),
		forge.WithOperationID("enrollMFA"),
		forge.WithResponseSchema(http.StatusOK, "Enrollment started", EnrollResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/verify", p.handleVerify,
		forge.WithSummary("Verify MFA code"),
		forge.WithOperationID("verifyMFA"),
		forge.WithResponseSchema(http.StatusOK, "Verified", VerifyMFAResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/challenge", p.handleChallenge,
		forge.WithSummary("MFA challenge"),
		forge.WithOperationID("challengeMFA"),
		forge.WithResponseSchema(http.StatusOK, "Challenge passed", ChallengeResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/enrollment", p.handleDisable,
		forge.WithSummary("Disable MFA"),
		forge.WithOperationID("disableMFA"),
		forge.WithResponseSchema(http.StatusOK, "MFA disabled", DisableResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Recovery code routes
	if err := g.POST("/recovery/verify", p.handleRecoveryVerify,
		forge.WithSummary("Verify MFA recovery code"),
		forge.WithOperationID("verifyMFARecovery"),
		forge.WithResponseSchema(http.StatusOK, "Recovery code accepted", RecoveryVerifyResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/recovery/regenerate", p.handleRecoveryRegenerate,
		forge.WithSummary("Regenerate MFA recovery codes"),
		forge.WithOperationID("regenerateMFARecoveryCodes"),
		forge.WithResponseSchema(http.StatusOK, "New recovery codes generated", RecoveryRegenerateResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// SMS MFA routes
	if err := g.POST("/sms/send", p.handleSMSSend,
		forge.WithSummary("Send SMS verification code"),
		forge.WithOperationID("sendSMSCode"),
		forge.WithResponseSchema(http.StatusOK, "SMS code sent", SMSSendResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/sms/verify", p.handleSMSVerify,
		forge.WithSummary("Verify SMS code"),
		forge.WithOperationID("verifySMSCode"),
		forge.WithResponseSchema(http.StatusOK, "SMS code verified", SMSVerifyResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// EnrollRequest is the request body for MFA enrollment.
type EnrollRequest struct {
	Method string `json:"method,omitempty"` // "totp" or "sms"
	Phone  string `json:"phone,omitempty"`  // Required for SMS method
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

// ChallengeRequest is the request body for MFA challenge during sign-in.
type ChallengeRequest struct {
	Code string `json:"code"`
}

// ChallengeResponse is returned on successful MFA challenge.
type ChallengeResponse struct {
	ChallengePassed bool   `json:"challenge_passed"`
	Method          string `json:"method"`
}

// DisableRequest is an empty request for disabling MFA.
type DisableRequest struct{}

// DisableResponse is returned when MFA is disabled.
type DisableResponse struct {
	Status string `json:"status"`
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

// RecoveryRegenerateRequest is an empty request for regenerating recovery codes.
type RecoveryRegenerateRequest struct{}

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

	if req.Method == "sms" {
		return p.enrollSMS(ctx, userID, req)
	}

	// Check if already enrolled
	existing, _ := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if existing != nil && existing.Verified {
		return nil, forge.NewHTTPError(http.StatusConflict, "MFA already enrolled and verified")
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
		return nil, forge.InternalError(fmt.Errorf("failed to generate TOTP key: %w", err))
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
		_ = p.store.DeleteEnrollment(ctx.Context(), existing.ID)
	}

	if err := p.store.CreateEnrollment(ctx.Context(), enrollment); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create enrollment: %w", err))
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

	if !ValidateTOTP(req.Code, enrollment.Secret) {
		return nil, forge.Unauthorized("invalid TOTP code")
	}

	// Mark as verified if this is the first successful verification
	if !enrollment.Verified {
		enrollment.Verified = true
		enrollment.UpdatedAt = time.Now()
		if err := p.store.UpdateEnrollment(ctx.Context(), enrollment); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to verify enrollment: %w", err))
		}

		// Generate recovery codes on first successful verification
		codes, plaintexts, err := GenerateRecoveryCodes(userID, DefaultRecoveryCodeCount)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to generate recovery codes: %w", err))
		}

		// Delete any old codes for this user, then store new ones
		_ = p.store.DeleteRecoveryCodes(ctx.Context(), userID)
		if err := p.store.CreateRecoveryCodes(ctx.Context(), codes); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to store recovery codes: %w", err))
		}

		p.audit(ctx.Context(), hook.ActionMFAEnroll, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
		p.relayEvent(ctx.Context(), "auth.mfa.verified", "", map[string]string{"user_id": userID.String(), "method": "totp"})

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

// handleChallenge verifies a TOTP code for a challenge (used during sign-in MFA step).
func (p *Plugin) handleChallenge(ctx forge.Context, req *ChallengeRequest) (*ChallengeResponse, error) {
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

	if !enrollment.Verified {
		return nil, forge.BadRequest("MFA enrollment not yet verified")
	}

	if !ValidateTOTP(req.Code, enrollment.Secret) {
		return nil, forge.Unauthorized("invalid TOTP code")
	}

	p.audit(ctx.Context(), hook.ActionMFAChallenge, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.challenged", "", map[string]string{"user_id": userID.String(), "method": "totp"})
	p.emitHook(ctx.Context(), hook.ActionMFAChallenge, "mfa", enrollment.ID.String(), userID.String(), "")

	return &ChallengeResponse{
		ChallengePassed: true,
		Method:          "totp",
	}, nil
}

// handleDisable removes MFA enrollment for the authenticated user.
func (p *Plugin) handleDisable(ctx forge.Context, _ *DisableRequest) (*DisableResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil {
		return nil, forge.NotFound("no MFA enrollment found")
	}

	if err := p.store.DeleteEnrollment(ctx.Context(), enrollment.ID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to remove enrollment: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFADisable, "mfa", enrollment.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.mfa.disabled", "", map[string]string{"user_id": userID.String(), "method": "totp"})
	p.emitHook(ctx.Context(), hook.ActionMFADisable, "mfa", enrollment.ID.String(), userID.String(), "")

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
		return nil, forge.InternalError(fmt.Errorf("failed to get recovery codes: %w", err))
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
		return nil, forge.Unauthorized("invalid recovery code")
	}

	// Consume the matched code (one-time use)
	if err := p.store.ConsumeRecoveryCode(ctx.Context(), matched.ID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to consume recovery code: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFARecoveryUsed, "mfa", matched.ID.String(), userID.String(), "")

	return &RecoveryVerifyResponse{
		ChallengePassed: true,
		CodesRemaining:  unused,
	}, nil
}

// handleRecoveryRegenerate generates a fresh set of recovery codes, replacing any existing ones.
func (p *Plugin) handleRecoveryRegenerate(ctx forge.Context, _ *RecoveryRegenerateRequest) (*RecoveryRegenerateResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}

	// Ensure user has verified MFA enrollment
	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "totp")
	if err != nil || !enrollment.Verified {
		return nil, forge.BadRequest("no verified MFA enrollment found")
	}

	// Generate new codes
	codes, plaintexts, err := GenerateRecoveryCodes(userID, DefaultRecoveryCodeCount)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate recovery codes: %w", err))
	}

	// Delete old codes, store new ones
	_ = p.store.DeleteRecoveryCodes(ctx.Context(), userID)
	if err := p.store.CreateRecoveryCodes(ctx.Context(), codes); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to store recovery codes: %w", err))
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
	existing, _ := p.store.GetEnrollment(ctx.Context(), userID, "sms")
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
		_ = p.store.DeleteEnrollment(ctx.Context(), existing.ID)
	}

	if err := p.store.CreateEnrollment(ctx.Context(), enrollment); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create SMS enrollment: %w", err))
	}

	// Send initial verification code
	challenge, err := SendSMSChallenge(ctx.Context(), p.sms, req.Phone)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to send SMS code: %w", err))
	}

	challengeData, _ := json.Marshal(challenge)
	_ = p.ceremonies.Set(ctx.Context(), "mfa:sms:"+userID.String(), challengeData, smsCodeTTL)

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

	phone := enrollment.Secret // Phone stored as the enrollment secret
	if req.Phone != "" {
		phone = req.Phone
	}

	challenge, err := SendSMSChallenge(ctx.Context(), p.sms, phone)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to send SMS code: %w", err))
	}

	challengeData, _ := json.Marshal(challenge)
	_ = p.ceremonies.Set(ctx.Context(), "mfa:sms:"+userID.String(), challengeData, smsCodeTTL)

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
		return nil, forge.BadRequest("no pending SMS challenge; send a code first")
	}

	var challenge SMSChallenge
	if unmarshalErr := json.Unmarshal(challengeData, &challenge); unmarshalErr != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to parse challenge: %w", unmarshalErr))
	}

	if !ValidateSMSCode(req.Code, &challenge) {
		return nil, forge.Unauthorized("invalid or expired SMS code")
	}

	// Remove the used challenge
	_ = p.ceremonies.Delete(ctx.Context(), "mfa:sms:"+userID.String())

	// If the enrollment is not yet verified, mark it as verified
	enrollment, err := p.store.GetEnrollment(ctx.Context(), userID, "sms")
	if err == nil && !enrollment.Verified {
		enrollment.Verified = true
		enrollment.UpdatedAt = time.Now()
		_ = p.store.UpdateEnrollment(ctx.Context(), enrollment)
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
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
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

//nolint:unparam // consistent audit/event API
func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{
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
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "***" + phone[len(phone)-4:]
}
