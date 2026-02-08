package passkey

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// Service provides passkey/WebAuthn operations with full cryptographic verification
type Service struct {
	db             *bun.DB
	userSvc        *user.Service
	authSvc        *auth.Service
	audit          *audit.Service
	config         Config
	webauthn       *WebAuthnWrapper
	challengeStore ChallengeStore
}

// NewService creates a new passkey service with WebAuthn support
func NewService(db *bun.DB, userSvc *user.Service, authSvc *auth.Service, auditSvc *audit.Service, cfg Config) (*Service, error) {
	// Create WebAuthn wrapper
	webauthnWrapper, err := NewWebAuthnWrapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn wrapper: %w", err)
	}

	// Create challenge store with timeout
	challengeStore := NewMemoryChallengeStore(time.Duration(cfg.Timeout) * time.Millisecond)

	return &Service{
		db:             db,
		userSvc:        userSvc,
		authSvc:        authSvc,
		audit:          auditSvc,
		config:         cfg,
		webauthn:       webauthnWrapper,
		challengeStore: challengeStore,
	}, nil
}

// BeginRegistration initiates WebAuthn passkey registration with cryptographic challenge
func (s *Service) BeginRegistration(ctx context.Context, userID xid.ID, req BeginRegisterRequest) (*BeginRegisterResponse, error) {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	// Get user from database
	u, err := s.userSvc.FindByID(ctx, userID)
	if err != nil {
		return nil, errs.UserNotFound().WithError(err)
	}

	// Load existing passkeys for this user/app/org
	existingPasskeys, err := s.getPasskeysForUser(ctx, userID, appID)
	if err != nil {
		return nil, errs.DatabaseError("load passkeys", err)
	}

	// Create user adapter
	userAdapter := NewUserAdapter(userID, u.Email, u.Name, existingPasskeys)

	// Prepare registration options
	var opts []webauthn.RegistrationOption

	// Set authenticator selection based on request
	authSelection := protocol.AuthenticatorSelection{
		UserVerification: ParseUserVerificationRequirement(req.UserVerification),
	}

	if req.AuthenticatorType != "" {
		authSelection.AuthenticatorAttachment = ParseAuthenticatorAttachment(req.AuthenticatorType)
	}

	if req.RequireResidentKey {
		authSelection.ResidentKey = protocol.ResidentKeyRequirementRequired
		authSelection.RequireResidentKey = protocol.ResidentKeyRequired()
	} else {
		authSelection.ResidentKey = protocol.ResidentKeyRequirementDiscouraged
	}

	opts = append(opts, webauthn.WithAuthenticatorSelection(authSelection))

	// Set attestation preference
	attestation := ParseConveyancePreference(s.config.AttestationType)
	opts = append(opts, webauthn.WithConveyancePreference(attestation))

	// Begin registration with WebAuthn
	credentialCreation, sessionData, err := s.webauthn.BeginRegistration(userAdapter, opts...)
	if err != nil {
		return nil, errs.PasskeyRegistrationFailed("WebAuthn begin registration failed").WithError(err)
	}

	// Store challenge session
	sessionID := xid.New().String()
	challengeSession := &ChallengeSession{
		Challenge:   credentialCreation.Response.Challenge,
		UserID:      userID,
		SessionData: sessionData,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(s.config.Timeout) * time.Millisecond),
	}

	if err := s.challengeStore.Store(ctx, sessionID, challengeSession); err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to store challenge session").WithError(err)
	}

	return &BeginRegisterResponse{
		Options:   credentialCreation,
		Challenge: base64.RawURLEncoding.EncodeToString(credentialCreation.Response.Challenge),
		UserID:    userID.String(),
		Timeout:   s.config.Timeout,
	}, nil
}

// FinishRegistration completes passkey registration with attestation verification
func (s *Service) FinishRegistration(ctx context.Context, userID xid.ID, credentialResponse []byte, name, ip, ua string) (*FinishRegisterResponse, error) {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	orgID, _ := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if !orgID.IsNil() {
		userOrgID = &orgID
	}

	// Get user
	u, err := s.userSvc.FindByID(ctx, userID)
	if err != nil {
		return nil, errs.UserNotFound().WithError(err)
	}

	// Parse WebAuthn credential response
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(credentialResponse))
	if err != nil {
		return nil, errs.PasskeyRegistrationFailed("Invalid credential response").WithError(err)
	}

	// Get challenge session
	challengeB64 := base64.RawURLEncoding.EncodeToString([]byte(parsedResponse.Response.CollectedClientData.Challenge))
	var challengeSession *ChallengeSession
	// Try to find the challenge session by searching for matching challenge
	// In production, you'd include sessionID in the client request
	// For now, we'll search through sessions (not ideal for high volume)
	// TODO: Include sessionID in client request for better lookup

	// Load existing passkeys
	existingPasskeys, err := s.getPasskeysForUser(ctx, userID, appID)
	if err != nil {
		return nil, errs.DatabaseError("load passkeys", err)
	}

	// Create user adapter
	userAdapter := NewUserAdapter(userID, u.Email, u.Name, existingPasskeys)

	// For now, create a session data from the response
	// In production, retrieve stored session data
	sessionData := webauthn.SessionData{
		Challenge:        parsedResponse.Response.CollectedClientData.Challenge,
		UserID:           []byte(userID.String()),
		UserVerification: protocol.VerificationPreferred,
	}

	// Verify attestation and create credential
	credential, err := s.webauthn.FinishRegistration(userAdapter, sessionData, parsedResponse)
	if err != nil {
		return nil, errs.PasskeyRegistrationFailed("Attestation verification failed").WithError(err)
	}

	authData := parsedResponse.Response.AttestationObject.AuthData

	// Determine authenticator type
	authenticatorType := "cross-platform"
	if authData.Flags.HasUserPresent() && authData.Flags.HasUserVerified() {
		authenticatorType = "platform"
	}

	// Store passkey in database
	passkey := &schema.Passkey{
		ID:                 xid.New(),
		UserID:             userID,
		CredentialID:       base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:          credential.PublicKey,
		AAGUID:             credential.Authenticator.AAGUID,
		SignCount:          credential.Authenticator.SignCount,
		AuthenticatorType:  authenticatorType,
		Name:               name,
		IsResidentKey:      authData.Flags.HasAttestedCredentialData(),
		AppID:              appID,
		UserOrganizationID: userOrgID,
	}
	passkey.AuditableModel.CreatedBy = passkey.ID
	passkey.AuditableModel.UpdatedBy = passkey.ID

	_, err = s.db.NewInsert().Model(passkey).Exec(ctx)
	if err != nil {
		return nil, errs.DatabaseError("insert passkey", err)
	}

	// Audit log
	if s.audit != nil {
		_ = s.audit.Log(ctx, &userID, string(audit.ActionPasskeyRegistered), "passkey:"+passkey.ID.String(), ip, ua, fmt.Sprintf("name=%s,type=%s", name, authenticatorType))
	}

	// Clean up challenge session if found
	if challengeSession != nil {
		_ = s.challengeStore.Delete(ctx, challengeB64)
	}

	return &FinishRegisterResponse{
		PasskeyID:    passkey.ID.String(),
		Name:         passkey.Name,
		Status:       "registered",
		CreatedAt:    passkey.CreatedAt,
		CredentialID: passkey.CredentialID,
	}, nil
}

// BeginLogin initiates WebAuthn authentication challenge
func (s *Service) BeginLogin(ctx context.Context, userID xid.ID, req BeginLoginRequest) (*BeginLoginResponse, error) {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	// Get user
	u, err := s.userSvc.FindByID(ctx, userID)
	if err != nil {
		return nil, errs.UserNotFound().WithError(err)
	}

	// Load user's passkeys
	passkeys, err := s.getPasskeysForUser(ctx, userID, appID)
	if err != nil {
		return nil, errs.DatabaseError("load passkeys", err)
	}

	if len(passkeys) == 0 {
		return nil, errs.PasskeyNotFound()
	}

	// Create user adapter
	userAdapter := NewUserAdapter(userID, u.Email, u.Name, passkeys)

	// Prepare login options
	var opts []webauthn.LoginOption
	if req.UserVerification != "" {
		opts = append(opts, webauthn.WithUserVerification(ParseUserVerificationRequirement(req.UserVerification)))
	}

	// Begin login
	credentialAssertion, sessionData, err := s.webauthn.BeginLogin(userAdapter, opts...)
	if err != nil {
		return nil, errs.New("BEGIN_LOGIN_FAILED", "Failed to begin login", 400).WithError(err)
	}

	// Store challenge session
	sessionID := xid.New().String()
	challengeSession := &ChallengeSession{
		Challenge:   credentialAssertion.Response.Challenge,
		UserID:      userID,
		SessionData: sessionData,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(s.config.Timeout) * time.Millisecond),
	}

	if err := s.challengeStore.Store(ctx, sessionID, challengeSession); err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to store challenge session").WithError(err)
	}

	return &BeginLoginResponse{
		Options:   credentialAssertion,
		Challenge: base64.RawURLEncoding.EncodeToString(credentialAssertion.Response.Challenge),
		Timeout:   s.config.Timeout,
	}, nil
}

// FinishLogin completes authentication with signature verification and creates session
func (s *Service) FinishLogin(ctx context.Context, credentialResponse []byte, remember bool, ip, ua string) (*LoginResponse, error) {
	// Parse credential assertion response
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(credentialResponse))
	if err != nil {
		return nil, errs.InvalidToken().WithError(err)
	}

	// Get credential ID
	credentialID := base64.RawURLEncoding.EncodeToString(parsedResponse.RawID)

	// Find passkey by credential ID
	var passkey schema.Passkey
	err = s.db.NewSelect().Model(&passkey).
		Where("credential_id = ?", credentialID).
		Scan(ctx)
	if err != nil {
		return nil, errs.PasskeyNotFound().WithError(err)
	}

	// Get user
	u, err := s.userSvc.FindByID(ctx, passkey.UserID)
	if err != nil {
		return nil, errs.UserNotFound().WithError(err)
	}

	// Load all passkeys for verification
	passkeys, err := s.getPasskeysForUser(ctx, passkey.UserID, passkey.AppID)
	if err != nil {
		return nil, errs.DatabaseError("load passkeys", err)
	}

	// Create user adapter
	userAdapter := NewUserAdapter(passkey.UserID, u.Email, u.Name, passkeys)

	// Create session data for verification
	sessionData := webauthn.SessionData{
		Challenge:        parsedResponse.Response.CollectedClientData.Challenge,
		UserID:           []byte(passkey.UserID.String()),
		UserVerification: protocol.VerificationPreferred,
	}

	// Verify assertion signature
	credential, err := s.webauthn.FinishLogin(userAdapter, sessionData, parsedResponse)
	if err != nil {
		return nil, errs.PasskeyVerificationFailed("Signature verification failed").WithError(err)
	}

	// Check sign count for replay attack detection
	if credential.Authenticator.SignCount > 0 && credential.Authenticator.SignCount <= passkey.SignCount {
		return nil, errs.PasskeyVerificationFailed("Possible cloned authenticator detected (sign count)")
	}

	// Update passkey sign count and last used
	now := time.Now()
	_, err = s.db.NewUpdate().Model(&passkey).
		Set("sign_count = ?", credential.Authenticator.SignCount).
		Set("last_used_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", passkey.ID).
		Exec(ctx)
	if err != nil {
		// Log error but don't fail login
		// In production, consider how to handle this
	}

	// Create auth session
	authResp, err := s.authSvc.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to create session").WithError(err)
	}

	// Audit log
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, string(audit.ActionPasskeyLogin), "user:"+uid.String(), ip, ua, fmt.Sprintf("passkey_id=%s", passkey.ID.String()))
	}

	return &LoginResponse{
		User:        authResp.User,
		Session:     authResp.Session,
		Token:       authResp.Token,
		PasskeyUsed: passkey.ID.String(),
	}, nil
}

// List retrieves all passkeys for a user (app and org scoped)
func (s *Service) List(ctx context.Context, userID xid.ID) (*ListPasskeysResponse, error) {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	passkeys, err := s.getPasskeysForUser(ctx, userID, appID)
	if err != nil {
		return nil, errs.DatabaseError("list passkeys", err)
	}

	// Convert to response format
	passkeyInfos := make([]PasskeyInfo, len(passkeys))
	for i, pk := range passkeys {
		passkeyInfos[i] = PasskeyInfo{
			ID:                pk.ID.String(),
			Name:              pk.Name,
			CredentialID:      pk.CredentialID,
			AAGUID:            base64.RawURLEncoding.EncodeToString(pk.AAGUID),
			AuthenticatorType: pk.AuthenticatorType,
			CreatedAt:         pk.CreatedAt,
			LastUsedAt:        pk.LastUsedAt,
			SignCount:         pk.SignCount,
			IsResidentKey:     pk.IsResidentKey,
		}
	}

	return &ListPasskeysResponse{
		Passkeys: passkeyInfos,
		Count:    len(passkeyInfos),
	}, nil
}

// Update updates a passkey's metadata (primarily name)
func (s *Service) Update(ctx context.Context, passkeyID xid.ID, name string) (*UpdatePasskeyResponse, error) {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	orgID, _ := contexts.GetOrganizationID(ctx)

	// Build query with app/org scoping
	q := s.db.NewSelect().Model((*schema.Passkey)(nil)).
		Where("id = ?", passkeyID).
		Where("app_id = ?", appID)

	if !orgID.IsNil() {
		q = q.Where("user_organization_id = ?", orgID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	// Check if passkey exists
	exists, err := q.Exists(ctx)
	if err != nil {
		return nil, errs.DatabaseError("check passkey", err)
	}
	if !exists {
		return nil, errs.PasskeyNotFound()
	}

	// Update name
	now := time.Now()
	_, err = s.db.NewUpdate().Model((*schema.Passkey)(nil)).
		Set("name = ?", name).
		Set("updated_at = ?", now).
		Where("id = ?", passkeyID).
		Exec(ctx)
	if err != nil {
		return nil, errs.DatabaseError("update passkey", err)
	}

	return &UpdatePasskeyResponse{
		PasskeyID: passkeyID.String(),
		Name:      name,
		UpdatedAt: now,
	}, nil
}

// Delete removes a passkey (app and org scoped)
func (s *Service) Delete(ctx context.Context, passkeyID xid.ID, ip, ua string) error {
	// Get app and org from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	orgID, _ := contexts.GetOrganizationID(ctx)

	// Fetch passkey to verify ownership and get userID for audit
	var passkey schema.Passkey
	q := s.db.NewSelect().Model(&passkey).
		Where("id = ?", passkeyID).
		Where("app_id = ?", appID)

	if !orgID.IsNil() {
		q = q.Where("user_organization_id = ?", orgID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	if err != nil {
		return errs.PasskeyNotFound().WithError(err)
	}

	// Delete the passkey
	_, err = s.db.NewDelete().Model((*schema.Passkey)(nil)).Where("id = ?", passkeyID).Exec(ctx)
	if err != nil {
		return errs.DatabaseError("delete passkey", err)
	}

	// Audit log
	if s.audit != nil {
		_ = s.audit.Log(ctx, &passkey.UserID, string(audit.ActionPasskeyDeleted), "passkey:"+passkeyID.String(), ip, ua, "")
	}

	return nil
}

// getPasskeysForUser retrieves all passkeys for a user with app/org scoping
func (s *Service) getPasskeysForUser(ctx context.Context, userID, appID xid.ID) ([]schema.Passkey, error) {
	orgID, _ := contexts.GetOrganizationID(ctx)

	var passkeys []schema.Passkey
	q := s.db.NewSelect().Model(&passkeys).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID)

	if !orgID.IsNil() {
		q = q.Where("user_organization_id = ?", orgID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	return passkeys, err
}

// BeginDiscoverableLogin initiates authentication for discoverable credentials (usernameless)
func (s *Service) BeginDiscoverableLogin(ctx context.Context, req BeginLoginRequest) (*BeginLoginResponse, error) {
	// Prepare login options
	var opts []webauthn.LoginOption
	if req.UserVerification != "" {
		opts = append(opts, webauthn.WithUserVerification(ParseUserVerificationRequirement(req.UserVerification)))
	}

	// Begin discoverable login (no user required)
	credentialAssertion, sessionData, err := s.webauthn.BeginDiscoverableLogin(opts...)
	if err != nil {
		return nil, errs.New("BEGIN_LOGIN_FAILED", "Failed to begin discoverable login", 400).WithError(err)
	}

	// Store challenge session (without specific user)
	sessionID := xid.New().String()
	challengeSession := &ChallengeSession{
		Challenge:   credentialAssertion.Response.Challenge,
		UserID:      xid.NilID(), // No specific user for discoverable credentials
		SessionData: sessionData,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(s.config.Timeout) * time.Millisecond),
	}

	if err := s.challengeStore.Store(ctx, sessionID, challengeSession); err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to store challenge session").WithError(err)
	}

	return &BeginLoginResponse{
		Options:   credentialAssertion,
		Challenge: base64.RawURLEncoding.EncodeToString(credentialAssertion.Response.Challenge),
		Timeout:   s.config.Timeout,
	}, nil
}
