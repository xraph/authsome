package authsome

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"crypto/rand"
	"encoding/hex"

	"github.com/xraph/forge"
	"github.com/xraph/warden"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// SignUp creates a new user account and returns the user + session.
func (e *Engine) SignUp(ctx context.Context, req *account.SignUpRequest) (*user.User, *session.Session, error) {
	if err := e.requireStarted(); err != nil {
		return nil, nil, err
	}

	// Normalize email early so all downstream checks use the canonical form.
	if req.Email != "" {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}

	// Validate password
	policy := e.passwordPolicy()
	if err := policy.ValidatePassword(req.Password); err != nil {
		return nil, nil, err
	}

	// Breach check (fail-open on network errors)
	if policy.CheckBreached {
		if breached, _ := account.NewBreachChecker().IsBreached(req.Password); breached { //nolint:errcheck // best-effort check
			return nil, nil, account.ErrPasswordBreached
		}
	}

	// Plugin: before signup
	if err := e.plugins.EmitBeforeSignUp(ctx, req); err != nil {
		return nil, nil, fmt.Errorf("authsome: before signup: %w", err)
	}

	// Validate custom form fields against the active form config (if any).
	if len(req.Metadata) > 0 {
		fc, err := e.store.GetFormConfig(ctx, req.AppID, formconfig.FormTypeSignup)
		if err == nil && fc != nil && fc.Active {
			if errs := formconfig.ValidateSubmission(fc.Fields, req.Metadata); len(errs) > 0 {
				// Return the first validation error.
				for _, msg := range errs {
					return nil, nil, fmt.Errorf("authsome: form validation: %s", msg)
				}
			}
		}
	}

	// Check email uniqueness
	_, err := e.store.GetUserByEmail(ctx, req.AppID, req.Email)
	if err == nil {
		return nil, nil, account.ErrEmailTaken
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, nil, fmt.Errorf("authsome: check email: %w", err)
	}

	// Check username uniqueness (if provided)
	if req.Username != "" {
		_, lookupErr := e.store.GetUserByUsername(ctx, req.AppID, req.Username)
		if lookupErr == nil {
			return nil, nil, account.ErrUsernameTaken
		}
		if !errors.Is(lookupErr, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("authsome: check username: %w", lookupErr)
		}
	}

	// Resolve default environment when not explicitly provided.
	if req.EnvID.IsNil() {
		if env, _ := e.GetDefaultEnvironment(ctx, req.AppID); env != nil {
			req.EnvID = env.ID
		}
	}

	// Hash password using configured algorithm
	hash, err := account.HashPasswordWithPolicy(req.Password, policy)
	if err != nil {
		return nil, nil, err
	}

	// Create user
	u := account.NewUser(req, hash)

	// Plugin: before user create
	if hookErr := e.plugins.EmitBeforeUserCreate(ctx, u); hookErr != nil {
		return nil, nil, fmt.Errorf("authsome: before user create: %w", hookErr)
	}

	if createErr := e.store.CreateUser(ctx, u); createErr != nil {
		return nil, nil, fmt.Errorf("authsome: create user: %w", createErr)
	}

	// Plugin: after user create
	e.plugins.EmitAfterUserCreate(ctx, u)

	// Assign default Warden role to the new user.
	e.EnsureDefaultRole(ctx, req.AppID, u.ID)

	// If this is the first user for the platform app, promote to platform_owner.
	e.promoteFirstUserToOwner(ctx, req.AppID, u.ID)

	// Create session (using per-app + per-env config; JWT if configured)
	sess, err := e.newSession(req.AppID, u.ID, e.sessionConfigForApp(ctx, req.AppID, req.EnvID))
	if err != nil {
		return nil, nil, fmt.Errorf("authsome: create session token: %w", err)
	}

	// Bind session to device (registers or finds device via fingerprint upsert)
	e.bindSessionToDevice(ctx, sess, req.AppID, req.EnvID, req.IPAddress, req.UserAgent)

	// Plugin: before session create
	if hookErr := e.plugins.EmitBeforeSessionCreate(ctx, sess); hookErr != nil {
		return nil, nil, fmt.Errorf("authsome: before session create: %w", hookErr)
	}

	if storeErr := e.store.CreateSession(ctx, sess); storeErr != nil {
		return nil, nil, fmt.Errorf("authsome: create session: %w", storeErr)
	}

	// Plugin: after session create + after signup
	e.plugins.EmitAfterSessionCreate(ctx, sess)
	e.plugins.EmitAfterSignUp(ctx, u, sess)

	// Global hook bus
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSignUp,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     req.AppID.String(),
	})

	// Audit
	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "signup", "user", u.ID.String(), u.ID.String(), req.AppID.String(), "auth", nil)

	// Relay
	e.relayEvent(ctx, "user.created", req.AppID.String(), map[string]string{
		"user_id": u.ID.String(),
		"email":   u.Email,
	})

	return u, sess, nil
}

// SignIn authenticates a user and returns the user + session.
func (e *Engine) SignIn(ctx context.Context, req *account.SignInRequest) (*user.User, *session.Session, error) {
	if err := e.requireStarted(); err != nil {
		return nil, nil, err
	}

	// Build lockout key from identifier + appID
	lockoutKey := e.lockoutKey(req)

	// Check account lockout before proceeding
	if e.lockout != nil {
		locked, until, err := e.lockout.IsLocked(ctx, lockoutKey)
		if err != nil {
			e.logger.Warn("authsome: lockout check failed", log.String("error", err.Error()))
		}
		if locked {
			e.audit(ctx, bridge.SeverityWarning, bridge.OutcomeFailure, "signin", "session", "", "", req.AppID.String(), "auth", map[string]string{
				"reason":       "account_locked",
				"locked_until": until.Format(time.RFC3339),
			})
			return nil, nil, account.ErrAccountLocked
		}
	}

	// Plugin: before signin
	if err := e.plugins.EmitBeforeSignIn(ctx, req); err != nil {
		return nil, nil, fmt.Errorf("authsome: before signin: %w", err)
	}

	// Normalize email for case-insensitive lookup (emails are stored lowercase).
	if req.Email != "" {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}

	// Lookup user
	var u *user.User
	var err error
	switch {
	case req.Email != "":
		u, err = e.store.GetUserByEmail(ctx, req.AppID, req.Email)
	case req.Username != "":
		u, err = e.store.GetUserByUsername(ctx, req.AppID, req.Username)
	default:
		return nil, nil, account.ErrInvalidCredentials
	}

	if err != nil {
		e.recordFailedSignin(ctx, req, lockoutKey)
		return nil, nil, account.ErrInvalidCredentials
	}

	// Resolve default environment when not explicitly provided.
	if req.EnvID.IsNil() {
		if env, _ := e.GetDefaultEnvironment(ctx, req.AppID); env != nil {
			req.EnvID = env.ID
		}
	}

	// Check banned
	if u.Banned {
		if u.BanExpires == nil || u.BanExpires.After(time.Now()) {
			e.recordFailedSignin(ctx, req, lockoutKey)
			return nil, nil, account.ErrUserBanned
		}
	}

	// Verify password
	if checkErr := account.CheckPassword(u.PasswordHash, req.Password); checkErr != nil {
		e.recordFailedSignin(ctx, req, lockoutKey)
		return nil, nil, account.ErrInvalidCredentials
	}

	// Reset lockout on successful authentication
	if e.lockout != nil {
		_ = e.lockout.Reset(ctx, lockoutKey) //nolint:errcheck // best-effort reset
	}

	// Enforce email verification. Users whose email is not verified are
	// blocked from signing in unless enforcement is explicitly disabled.
	//
	// Resolution order (first non-nil wins):
	// 1. Per-app client config override (RequireEmailVerification)
	// 2. Environment setting (SkipEmailVerification — inverted)
	// 3. Default: require verification (true)
	if !u.EmailVerified {
		requireVerif := true // default: enforce

		// Check per-app override first.
		if appCfg, cfgErr := e.store.GetAppClientConfig(ctx, req.AppID); cfgErr == nil && appCfg.RequireEmailVerification != nil {
			requireVerif = *appCfg.RequireEmailVerification
		} else {
			// Fall back to environment setting.
			if env, _ := e.GetDefaultEnvironment(ctx, req.AppID); env != nil && env.Settings != nil {
				if env.Settings.SkipEmailVerificationEnabled() {
					requireVerif = false
				}
			}
		}

		if requireVerif {
			return u, nil, account.ErrEmailNotVerified
		}
	}

	// Check password expiration
	if e.config.Password.MaxAgeDays > 0 && u.PasswordChangedAt != nil {
		maxAge := time.Duration(e.config.Password.MaxAgeDays) * 24 * time.Hour
		if time.Since(*u.PasswordChangedAt) > maxAge {
			// Reset lockout so the user can change their password
			if e.lockout != nil {
				_ = e.lockout.Reset(ctx, lockoutKey) //nolint:errcheck // best-effort reset
			}
			return u, nil, account.ErrPasswordExpired
		}
	}

	// Transparent rehash if the password algorithm has changed (e.g. bcrypt→argon2id).
	policy := e.passwordPolicy()
	if account.NeedsRehash(u.PasswordHash, policy) {
		if newHash, hashErr := account.HashPasswordWithPolicy(req.Password, policy); hashErr == nil {
			u.PasswordHash = newHash
			u.UpdatedAt = time.Now()
			if updateErr := e.store.UpdateUser(ctx, u); updateErr != nil {
				e.logger.Warn("authsome: rehash on login failed", log.String("error", updateErr.Error()))
			}
		}
	}

	// Create session (using per-app + per-env config; JWT if configured)
	sess, err := e.newSession(req.AppID, u.ID, e.sessionConfigForApp(ctx, req.AppID, req.EnvID))
	if err != nil {
		return nil, nil, fmt.Errorf("authsome: create session token: %w", err)
	}

	// Bind session to device (registers or finds device via fingerprint upsert)
	e.bindSessionToDevice(ctx, sess, req.AppID, req.EnvID, req.IPAddress, req.UserAgent)

	// Plugin: before session create
	if hookErr := e.plugins.EmitBeforeSessionCreate(ctx, sess); hookErr != nil {
		return nil, nil, fmt.Errorf("authsome: before session create: %w", hookErr)
	}

	if storeErr := e.store.CreateSession(ctx, sess); storeErr != nil {
		return nil, nil, fmt.Errorf("authsome: create session: %w", storeErr)
	}

	// Plugin: after session create + after signin
	e.plugins.EmitAfterSessionCreate(ctx, sess)
	e.plugins.EmitAfterSignIn(ctx, u, sess)

	// Global hook bus
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSignIn,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     req.AppID.String(),
		Metadata: map[string]string{
			"email":      u.Email,
			"user_name":  u.Name(),
			"session_id": sess.ID.String(),
		},
	})

	// Audit
	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "signin", "session", sess.ID.String(), u.ID.String(), req.AppID.String(), "auth", nil)

	// Relay
	e.relayEvent(ctx, "auth.signin", req.AppID.String(), map[string]string{
		"user_id":    u.ID.String(),
		"session_id": sess.ID.String(),
	})

	return u, sess, nil
}

// SignOut terminates a session.
func (e *Engine) SignOut(ctx context.Context, sessionID id.SessionID) error {
	if err := e.requireStarted(); err != nil {
		return err
	}

	sess, err := e.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("authsome: get session: %w", err)
	}

	// Plugin: before signout
	if hookErr := e.plugins.EmitBeforeSignOut(ctx, sessionID); hookErr != nil {
		return fmt.Errorf("authsome: before signout: %w", hookErr)
	}

	if deleteErr := e.store.DeleteSession(ctx, sessionID); deleteErr != nil {
		return fmt.Errorf("authsome: delete session: %w", deleteErr)
	}

	// Plugin: after signout
	e.plugins.EmitAfterSignOut(ctx, sessionID)
	e.plugins.EmitAfterSessionRevoke(ctx, sessionID)

	// Global hook bus
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSignOut,
		Resource:   hook.ResourceSession,
		ResourceID: sessionID.String(),
		ActorID:    sess.UserID.String(),
		Tenant:     sess.AppID.String(),
	})

	// Audit
	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "signout", "session", sessionID.String(), sess.UserID.String(), sess.AppID.String(), "auth", nil)

	// Relay
	e.relayEvent(ctx, "auth.signout", sess.AppID.String(), map[string]string{
		"user_id":    sess.UserID.String(),
		"session_id": sessionID.String(),
	})

	return nil
}

// Refresh generates new tokens for an existing session using the refresh token.
func (e *Engine) Refresh(ctx context.Context, refreshToken string) (*session.Session, error) {
	sess, err := e.store.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, account.ErrInvalidCredentials
	}

	// Check if refresh token is expired
	if time.Now().After(sess.RefreshTokenExpiresAt) {
		_ = e.store.DeleteSession(ctx, sess.ID) //nolint:errcheck // best-effort cleanup
		return nil, account.ErrSessionExpired
	}

	// Generate new tokens (using per-app + per-env config if available)
	cfg := e.sessionConfigForApp(ctx, sess.AppID, sess.EnvID)
	if err := account.RefreshSession(sess, cfg); err != nil {
		return nil, fmt.Errorf("authsome: refresh session: %w", err)
	}

	if err := e.store.UpdateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("authsome: update session: %w", err)
	}

	// Plugin: after session refresh
	e.plugins.EmitAfterSessionRefresh(ctx, sess)

	// Global hook bus
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRefresh,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    sess.UserID.String(),
		Tenant:     sess.AppID.String(),
	})

	return sess, nil
}

// GetMe returns the current user by ID.
func (e *Engine) GetMe(ctx context.Context, userID id.UserID) (*user.User, error) {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authsome: get user: %w", err)
	}
	return u, nil
}

// UpdateMe updates the current user.
func (e *Engine) UpdateMe(ctx context.Context, u *user.User) error {
	if err := e.plugins.EmitBeforeUserUpdate(ctx, u); err != nil {
		return fmt.Errorf("authsome: before user update: %w", err)
	}

	u.UpdatedAt = time.Now()
	if err := e.store.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("authsome: update user: %w", err)
	}

	e.plugins.EmitAfterUserUpdate(ctx, u)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionUserUpdate,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.relayEvent(ctx, "user.updated", u.AppID.String(), map[string]string{
		"user_id": u.ID.String(),
	})

	return nil
}

// ListSessions returns all sessions for a user.
func (e *Engine) ListSessions(ctx context.Context, userID id.UserID) ([]*session.Session, error) {
	return e.store.ListUserSessions(ctx, userID)
}

// ListAllSessions returns the most recent sessions across all users, up to limit.
func (e *Engine) ListAllSessions(ctx context.Context, limit int) ([]*session.Session, error) {
	return e.store.ListSessions(ctx, limit)
}

// RevokeSession deletes a specific session.
func (e *Engine) RevokeSession(ctx context.Context, sessionID id.SessionID) error {
	sess, err := e.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("authsome: get session: %w", err)
	}

	if err := e.store.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("authsome: delete session: %w", err)
	}

	e.plugins.EmitAfterSessionRevoke(ctx, sessionID)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSessionRevoke,
		Resource:   hook.ResourceSession,
		ResourceID: sessionID.String(),
		ActorID:    sess.UserID.String(),
		Tenant:     sess.AppID.String(),
	})

	e.relayEvent(ctx, "session.revoked", sess.AppID.String(), map[string]string{
		"user_id":    sess.UserID.String(),
		"session_id": sessionID.String(),
	})

	return nil
}

// ResolveSessionByToken resolves a session from its token (for middleware).
func (e *Engine) ResolveSessionByToken(token string) (*session.Session, error) {
	ctx := context.Background()
	sess, err := e.store.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, account.ErrSessionExpired
	}
	return sess, nil
}

// ResolveUser resolves a user by ID string (for middleware).
func (e *Engine) ResolveUser(userIDStr string) (*user.User, error) {
	ctx := context.Background()
	userID, err := id.ParseUserID(userIDStr)
	if err != nil {
		return nil, err
	}
	return e.store.GetUser(ctx, userID)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func (e *Engine) passwordPolicy() account.PasswordPolicy {
	return account.PasswordPolicy{
		MinLength:        e.config.Password.MinLength,
		RequireUppercase: e.config.Password.RequireUppercase,
		RequireLowercase: e.config.Password.RequireLowercase,
		RequireDigit:     e.config.Password.RequireDigit,
		RequireSpecial:   e.config.Password.RequireSpecial,
		BcryptCost:       e.config.Password.BcryptCost,
		Algorithm:        e.config.Password.Algorithm,
		Argon2Params: account.Argon2Params{
			Memory:      e.config.Password.Argon2.Memory,
			Iterations:  e.config.Password.Argon2.Iterations,
			Parallelism: e.config.Password.Argon2.Parallelism,
			SaltLength:  e.config.Password.Argon2.SaltLength,
			KeyLength:   e.config.Password.Argon2.KeyLength,
		},
		CheckBreached: e.config.Password.CheckBreached,
	}
}

func (e *Engine) sessionConfig() account.SessionConfig {
	return account.SessionConfig{
		TokenTTL:           e.config.Session.TokenTTL,
		RefreshTokenTTL:    e.config.Session.RefreshTokenTTL,
		MaxActiveSessions:  e.config.Session.MaxActiveSessions,
		RotateRefreshToken: e.config.Session.ShouldRotateRefreshToken(),
	}
}

// sessionConfigForApp returns the session config for a specific app, applying
// per-app overrides on top of the global defaults. Resolution order:
//  1. Global engine config (base)
//  2. Per-app session config (stored in DB or seeded from YAML/code)
//  3. Per-environment settings (if envID is provided)
func (e *Engine) sessionConfigForApp(ctx context.Context, appID id.AppID, envIDs ...id.EnvironmentID) account.SessionConfig {
	cfg := e.sessionConfig()

	// Apply per-app overrides.
	if appCfg, err := e.store.GetAppSessionConfig(ctx, appID); err == nil && appCfg != nil {
		appCfg.ApplyTo(&cfg)
	}

	// Apply per-environment overrides (highest priority).
	if len(envIDs) > 0 && !envIDs[0].IsNil() {
		if env, err := e.store.GetEnvironment(ctx, envIDs[0]); err == nil && env.Settings != nil {
			env.Settings.ApplySessionOverrides(&cfg)
		}
	}

	return cfg
}

// newSession creates a new session, optionally generating a JWT access token
// when the app is configured for JWT token format. Falls back to opaque tokens.
func (e *Engine) newSession(appID id.AppID, userID id.UserID, cfg account.SessionConfig) (*session.Session, error) {
	sess, err := account.NewSession(appID, userID, cfg)
	if err != nil {
		return nil, err
	}

	// Check if app uses JWT for access tokens.
	tokFmt := e.TokenFormatForApp(appID.String())
	if tokFmt.Name() == "jwt" {
		jwtToken, genErr := tokFmt.GenerateAccessToken(tokenformat.TokenClaims{
			UserID:    userID.String(),
			AppID:     appID.String(),
			SessionID: sess.ID.String(),
			IssuedAt:  sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
		if genErr != nil {
			return nil, genErr
		}
		sess.Token = jwtToken
	}

	return sess, nil
}

// bindSessionToDevice populates connection info on a session and registers
// (or finds) the associated device via fingerprint-based upsert. Device
// registration failure is non-fatal so it never blocks authentication.
func (e *Engine) bindSessionToDevice(ctx context.Context, sess *session.Session, appID id.AppID, envID id.EnvironmentID, ipAddress, userAgent string) {
	sess.EnvID = envID
	sess.IPAddress = ipAddress
	sess.UserAgent = userAgent

	if userAgent == "" {
		return
	}

	browser, os, devType := device.ParseUserAgent(userAgent)
	dev, err := e.RegisterDevice(ctx, &device.Device{
		UserID:      sess.UserID,
		AppID:       appID,
		EnvID:       envID,
		Browser:     browser,
		OS:          os,
		Type:        devType,
		IPAddress:   ipAddress,
		Fingerprint: userAgent,
	})
	if err != nil {
		e.logger.Warn("authsome: bind session to device failed", log.String("error", err.Error()))
		return
	}
	sess.DeviceID = dev.ID
}

func (e *Engine) audit(ctx context.Context, severity, outcome, action, resource, resourceID, actorID, tenant, category string, metadata map[string]string) {
	if e.chronicle == nil {
		return
	}
	if err := e.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    outcome,
		Severity:   severity,
		Category:   category,
		Metadata:   metadata,
	}); err != nil {
		e.logger.Warn("authsome: audit record failed",
			log.String("action", action),
			log.String("error", err.Error()),
		)
	}
}

// checkPasswordHistory verifies that the new password does not match any
// of the user's recent passwords. Returns ErrPasswordReused on match.
func (e *Engine) checkPasswordHistory(ctx context.Context, userID id.UserID, newPassword string) error {
	if e.passwordHistory == nil || e.config.Password.HistoryCount <= 0 {
		return nil
	}
	entries, err := e.passwordHistory.GetPasswordHistory(ctx, userID, e.config.Password.HistoryCount)
	if err != nil {
		e.logger.Warn("authsome: password history lookup failed", log.String("error", err.Error()))
		return nil // fail open — don't block the user
	}
	for _, entry := range entries {
		if account.CheckPassword(entry.Hash, newPassword) == nil {
			return account.ErrPasswordReused
		}
	}
	return nil
}

// savePasswordHistory records the old password hash in the history store.
func (e *Engine) savePasswordHistory(ctx context.Context, userID id.UserID, oldHash string) {
	if e.passwordHistory == nil || e.config.Password.HistoryCount <= 0 || oldHash == "" {
		return
	}
	if err := e.passwordHistory.SavePasswordHash(ctx, userID, oldHash); err != nil {
		e.logger.Warn("authsome: save password history failed", log.String("error", err.Error()))
	}
}

// lockoutKey builds a scoped lockout key from a sign-in request.
func (e *Engine) lockoutKey(req *account.SignInRequest) string {
	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}
	return req.AppID.String() + ":" + identifier
}

// recordFailedSignin audits, emits hooks, records lockout failure, and fires
// the account-locked event when the threshold is crossed.
func (e *Engine) recordFailedSignin(ctx context.Context, req *account.SignInRequest, lockoutKey string) {
	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}

	// Audit + hook + relay (same as before)
	e.audit(ctx, bridge.SeverityWarning, bridge.OutcomeFailure, "signin", "session", "", "", req.AppID.String(), "auth", map[string]string{
		"identifier": identifier,
	})
	e.hooks.Emit(ctx, &hook.Event{
		Action:   hook.ActionSignIn,
		Resource: hook.ResourceSession,
		Tenant:   req.AppID.String(),
		Err:      account.ErrInvalidCredentials,
		Metadata: map[string]string{"identifier": identifier},
	})
	e.relayEvent(ctx, "auth.signin.failed", req.AppID.String(), map[string]string{
		"identifier": identifier,
	})

	// Record failure in lockout tracker
	if e.lockout != nil {
		attempts, err := e.lockout.RecordFailure(ctx, lockoutKey)
		if err != nil {
			e.logger.Warn("authsome: lockout record failure failed", log.String("error", err.Error()))
			return
		}

		// Check if this failure caused a lockout
		maxAttempts := e.config.Lockout.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 5
		}
		if attempts >= maxAttempts {
			e.hooks.Emit(ctx, &hook.Event{
				Action:   hook.ActionAccountLocked,
				Resource: hook.ResourceUser,
				Tenant:   req.AppID.String(),
				Metadata: map[string]string{
					"identifier": identifier,
					"attempts":   fmt.Sprintf("%d", attempts),
				},
			})
			e.audit(ctx, bridge.SeverityCritical, bridge.OutcomeFailure, "account_locked", "user", "", "", req.AppID.String(), "auth", map[string]string{
				"identifier": identifier,
				"attempts":   fmt.Sprintf("%d", attempts),
			})
			e.relayEvent(ctx, "auth.account_locked", req.AppID.String(), map[string]string{
				"identifier": identifier,
			})
		}
	}
}

// ──────────────────────────────────────────────────
// Password Management
// ──────────────────────────────────────────────────

// ForgotPassword creates a password reset token for the given email.
// Returns the reset record (token can be sent via email by the caller).
// Returns nil, nil if user not found (avoids email enumeration).
func (e *Engine) ForgotPassword(ctx context.Context, appID id.AppID, email string) (*account.PasswordReset, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}

	email = strings.ToLower(strings.TrimSpace(email))

	u, err := e.store.GetUserByEmail(ctx, appID, email)
	if err != nil {
		return nil, nil //nolint:nilerr // intentionally returning nil on auth failure
	}

	ttl := 1 * time.Hour
	pr, err := account.NewPasswordReset(ctx, appID, u.ID, ttl)
	if err != nil {
		return nil, fmt.Errorf("authsome: create password reset: %w", err)
	}

	if storeErr := e.store.CreatePasswordReset(ctx, pr); storeErr != nil {
		return nil, fmt.Errorf("authsome: store password reset: %w", storeErr)
	}

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "forgot_password", "user", u.ID.String(), u.ID.String(), appID.String(), "auth", nil)
	e.relayEvent(ctx, "auth.forgot_password", appID.String(), map[string]string{
		"user_id": u.ID.String(),
		"email":   u.Email,
	})

	return pr, nil
}

// ResetPassword resets a user's password using a reset token.
func (e *Engine) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := e.requireStarted(); err != nil {
		return err
	}

	pr, err := e.store.GetPasswordReset(ctx, token)
	if err != nil {
		return account.ErrInvalidCredentials
	}

	if pr.Consumed || time.Now().After(pr.ExpiresAt) {
		return account.ErrInvalidCredentials
	}

	policy := e.passwordPolicy()
	if validateErr := policy.ValidatePassword(newPassword); validateErr != nil {
		return validateErr
	}

	// Breach check
	if policy.CheckBreached {
		if breached, _ := account.NewBreachChecker().IsBreached(newPassword); breached { //nolint:errcheck // best-effort check
			return account.ErrPasswordBreached
		}
	}

	u, err := e.store.GetUser(ctx, pr.UserID)
	if err != nil {
		return fmt.Errorf("authsome: get user: %w", err)
	}

	// Password history check
	if historyErr := e.checkPasswordHistory(ctx, u.ID, newPassword); historyErr != nil {
		return historyErr
	}

	hash, err := account.HashPasswordWithPolicy(newPassword, policy)
	if err != nil {
		return err
	}

	oldHash := u.PasswordHash
	now := time.Now()
	u.PasswordHash = hash
	u.PasswordChangedAt = &now
	u.UpdatedAt = now
	if updateErr := e.store.UpdateUser(ctx, u); updateErr != nil {
		return fmt.Errorf("authsome: update user: %w", updateErr)
	}

	e.savePasswordHistory(ctx, u.ID, oldHash)

	if consumeErr := e.store.ConsumePasswordReset(ctx, token); consumeErr != nil {
		return fmt.Errorf("authsome: consume reset: %w", consumeErr)
	}

	_ = e.store.DeleteUserSessions(ctx, pr.UserID) //nolint:errcheck // best-effort cleanup

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "reset_password", "user", u.ID.String(), u.ID.String(), pr.AppID.String(), "auth", nil)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionPasswordReset,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     pr.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.relayEvent(ctx, "auth.password_reset", pr.AppID.String(), map[string]string{
		"user_id": u.ID.String(),
	})

	return nil
}

// ChangePassword changes a user's password (requires current password).
func (e *Engine) ChangePassword(ctx context.Context, userID id.UserID, currentPassword, newPassword string) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: get user: %w", err)
	}

	if verifyErr := account.CheckPassword(u.PasswordHash, currentPassword); verifyErr != nil {
		return account.ErrInvalidCredentials
	}

	policy := e.passwordPolicy()
	if validateErr := policy.ValidatePassword(newPassword); validateErr != nil {
		return validateErr
	}

	// Breach check
	if policy.CheckBreached {
		if breached, _ := account.NewBreachChecker().IsBreached(newPassword); breached { //nolint:errcheck // best-effort check
			return account.ErrPasswordBreached
		}
	}

	// Password history check
	if historyErr := e.checkPasswordHistory(ctx, userID, newPassword); historyErr != nil {
		return historyErr
	}

	hash, err := account.HashPasswordWithPolicy(newPassword, policy)
	if err != nil {
		return err
	}

	oldHash := u.PasswordHash
	now := time.Now()
	u.PasswordHash = hash
	u.PasswordChangedAt = &now
	u.UpdatedAt = now
	if updateErr := e.store.UpdateUser(ctx, u); updateErr != nil {
		return fmt.Errorf("authsome: update user: %w", updateErr)
	}

	e.savePasswordHistory(ctx, userID, oldHash)

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "change_password", "user", u.ID.String(), u.ID.String(), u.AppID.String(), "auth", nil)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionPasswordChange,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.relayEvent(ctx, "auth.password_changed", u.AppID.String(), map[string]string{
		"user_id": u.ID.String(),
	})

	return nil
}

// VerifyEmail verifies a user's email using a verification token.
func (e *Engine) VerifyEmail(ctx context.Context, token string) error {
	if err := e.requireStarted(); err != nil {
		return err
	}

	v, err := e.store.GetVerification(ctx, token)
	if err != nil {
		return account.ErrInvalidCredentials
	}

	if v.Consumed || time.Now().After(v.ExpiresAt) {
		return account.ErrInvalidCredentials
	}

	if consumeErr := e.store.ConsumeVerification(ctx, token); consumeErr != nil {
		return fmt.Errorf("authsome: consume verification: %w", consumeErr)
	}

	u, err := e.store.GetUser(ctx, v.UserID)
	if err != nil {
		return fmt.Errorf("authsome: get user: %w", err)
	}

	u.EmailVerified = true
	u.UpdatedAt = time.Now()
	if updateErr := e.store.UpdateUser(ctx, u); updateErr != nil {
		return fmt.Errorf("authsome: update user: %w", updateErr)
	}

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "verify_email", "user", u.ID.String(), u.ID.String(), v.AppID.String(), "auth", nil)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEmailVerify,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     v.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.relayEvent(ctx, "user.email_verified", v.AppID.String(), map[string]string{
		"user_id": u.ID.String(),
	})

	return nil
}

// ──────────────────────────────────────────────────
// Device Management
// ──────────────────────────────────────────────────

// ListUserDevices returns all devices for a user.
func (e *Engine) ListUserDevices(ctx context.Context, userID id.UserID) ([]*device.Device, error) {
	return e.store.ListUserDevices(ctx, userID)
}

// ListAllDevices returns the most recent devices across all users, up to limit.
func (e *Engine) ListAllDevices(ctx context.Context, limit int) ([]*device.Device, error) {
	return e.store.ListDevices(ctx, limit)
}

// GetDevice returns a device by ID.
func (e *Engine) GetDevice(ctx context.Context, deviceID id.DeviceID) (*device.Device, error) {
	return e.store.GetDevice(ctx, deviceID)
}

// DeleteDevice removes a device.
func (e *Engine) DeleteDevice(ctx context.Context, deviceID id.DeviceID) error {
	if err := e.store.DeleteDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("authsome: delete device: %w", err)
	}
	return nil
}

// TrustDevice marks a device as trusted.
func (e *Engine) TrustDevice(ctx context.Context, deviceID id.DeviceID) (*device.Device, error) {
	d, err := e.store.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("authsome: trust device: %w", err)
	}

	d.Trusted = true
	d.UpdatedAt = time.Now()
	if err := e.store.UpdateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("authsome: trust device: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     "device.trust",
		Resource:   hook.ResourceDevice,
		ResourceID: d.ID.String(),
		ActorID:    d.UserID.String(),
	})
	e.relayEvent(ctx, "device.trusted", d.AppID.String(), map[string]string{
		"device_id": d.ID.String(),
		"user_id":   d.UserID.String(),
	})

	return d, nil
}

// RegisterDevice creates or updates a device using fingerprint-based upsert.
func (e *Engine) RegisterDevice(ctx context.Context, d *device.Device) (*device.Device, error) {
	// Try to find an existing device by fingerprint
	if d.Fingerprint != "" {
		existing, err := e.store.GetDeviceByFingerprint(ctx, d.UserID, d.Fingerprint)
		if err == nil {
			// Device already exists — update last seen
			existing.LastSeenAt = time.Now()
			existing.IPAddress = d.IPAddress
			existing.UpdatedAt = time.Now()
			if err := e.store.UpdateDevice(ctx, existing); err != nil {
				return nil, fmt.Errorf("authsome: register device: update: %w", err)
			}
			return existing, nil
		}
		// Not found — fall through to create
	}

	// Create new device
	if d.ID.String() == "" {
		d.ID = id.NewDeviceID()
	}
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
		d.UpdatedAt = now
		d.LastSeenAt = now
	}

	if err := e.store.CreateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("authsome: register device: create: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     "device.register",
		Resource:   hook.ResourceDevice,
		ResourceID: d.ID.String(),
		ActorID:    d.UserID.String(),
	})
	e.relayEvent(ctx, "device.registered", d.AppID.String(), map[string]string{
		"device_id": d.ID.String(),
		"user_id":   d.UserID.String(),
	})

	return d, nil
}

// ──────────────────────────────────────────────────
// Webhook Management
// ──────────────────────────────────────────────────

// CreateWebhook creates a new webhook endpoint registration.
func (e *Engine) CreateWebhook(ctx context.Context, w *webhook.Webhook) error {
	// Generate a signing secret if not provided
	if w.Secret == "" {
		secret, err := generateWebhookSecret()
		if err != nil {
			return fmt.Errorf("authsome: create webhook: generate secret: %w", err)
		}
		w.Secret = secret
	}

	if w.ID.String() == "" {
		w.ID = id.NewWebhookID()
	}
	now := time.Now()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
		w.UpdatedAt = now
	}
	w.Active = true

	if err := e.store.CreateWebhook(ctx, w); err != nil {
		return fmt.Errorf("authsome: create webhook: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookCreate,
		Resource:   hook.ResourceWebhook,
		ResourceID: w.ID.String(),
		Tenant:     w.AppID.String(),
	})
	e.relayEvent(ctx, "webhook.created", w.AppID.String(), map[string]string{
		"webhook_id": w.ID.String(),
		"url":        w.URL,
	})

	return nil
}

// GetWebhook returns a webhook by ID.
func (e *Engine) GetWebhook(ctx context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	return e.store.GetWebhook(ctx, webhookID)
}

// UpdateWebhook updates an existing webhook.
func (e *Engine) UpdateWebhook(ctx context.Context, w *webhook.Webhook) error {
	w.UpdatedAt = time.Now()
	if err := e.store.UpdateWebhook(ctx, w); err != nil {
		return fmt.Errorf("authsome: update webhook: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookUpdate,
		Resource:   hook.ResourceWebhook,
		ResourceID: w.ID.String(),
		Tenant:     w.AppID.String(),
	})

	return nil
}

// DeleteWebhook deletes a webhook.
func (e *Engine) DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error {
	if err := e.store.DeleteWebhook(ctx, webhookID); err != nil {
		return fmt.Errorf("authsome: delete webhook: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookDelete,
		Resource:   hook.ResourceWebhook,
		ResourceID: webhookID.String(),
	})

	return nil
}

// ListWebhooks returns all webhooks for an app.
func (e *Engine) ListWebhooks(ctx context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	return e.store.ListWebhooks(ctx, appID)
}

// generateWebhookSecret generates a random hex secret for webhook signing.
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(b), nil
}

// rbacStore returns the RBAC store backed by Warden. Warden is required for
// all RBAC operations. Callers should guard with hasRBACStore() first.
func (e *Engine) rbacStore() rbac.Store {
	if e.wardenEng == nil {
		panic("authsome: rbacStore() called but warden is not configured; warden is required for RBAC")
	}
	return rbac.NewWardenStore(e.wardenEng)
}

// hasRBACStore reports whether the Warden engine is available for RBAC operations.
func (e *Engine) hasRBACStore() bool {
	return e.wardenEng != nil
}

// ──────────────────────────────────────────────────
// RBAC Management
// ──────────────────────────────────────────────────

// CreateRole creates a new RBAC role.
func (e *Engine) CreateRole(ctx context.Context, r *rbac.Role) error {
	if r.CreatedAt.IsZero() {
		now := time.Now()
		r.CreatedAt = now
		r.UpdatedAt = now
	}

	if err := e.rbacStore().CreateRole(ctx, r); err != nil {
		return fmt.Errorf("authsome: create role: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRoleCreate,
		Resource:   hook.ResourceRole,
		ResourceID: r.ID,
		Tenant:     r.AppID,
	})
	e.relayEvent(ctx, "rbac.role.created", r.AppID, map[string]string{
		"role_id":   r.ID,
		"role_slug": r.Slug,
	})

	return nil
}

// GetRole returns a role by ID.
func (e *Engine) GetRole(ctx context.Context, roleID id.RoleID) (*rbac.Role, error) {
	return e.rbacStore().GetRole(ctx, roleID.String())
}

// GetRoleBySlug returns a role by slug within an app.
func (e *Engine) GetRoleBySlug(ctx context.Context, appID id.AppID, slug string) (*rbac.Role, error) {
	return e.rbacStore().GetRoleBySlug(ctx, appID.String(), slug)
}

// UpdateRole updates an existing RBAC role.
func (e *Engine) UpdateRole(ctx context.Context, r *rbac.Role) error {
	r.UpdatedAt = time.Now()
	if err := e.rbacStore().UpdateRole(ctx, r); err != nil {
		return fmt.Errorf("authsome: update role: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRoleUpdate,
		Resource:   hook.ResourceRole,
		ResourceID: r.ID,
		Tenant:     r.AppID,
	})

	return nil
}

// DeleteRole deletes an RBAC role and cascades to its permissions and assignments.
func (e *Engine) DeleteRole(ctx context.Context, roleID id.RoleID) error {
	if err := e.rbacStore().DeleteRole(ctx, roleID.String()); err != nil {
		return fmt.Errorf("authsome: delete role: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRoleDelete,
		Resource:   hook.ResourceRole,
		ResourceID: roleID.String(),
	})

	return nil
}

// ListRoles returns all roles for an app.
func (e *Engine) ListRoles(ctx context.Context, appID id.AppID) ([]*rbac.Role, error) {
	return e.rbacStore().ListRoles(ctx, appID.String())
}

// AddPermission adds a permission to a role.
func (e *Engine) AddPermission(ctx context.Context, p *rbac.Permission) error {
	if err := e.rbacStore().AddPermission(ctx, p); err != nil {
		return fmt.Errorf("authsome: add permission: %w", err)
	}
	return nil
}

// RemovePermission removes a permission from a role.
func (e *Engine) RemovePermission(ctx context.Context, permID id.PermissionID) error {
	if err := e.rbacStore().RemovePermission(ctx, permID.String()); err != nil {
		return fmt.Errorf("authsome: remove permission: %w", err)
	}
	return nil
}

// ListRolePermissions returns all permissions for a role.
func (e *Engine) ListRolePermissions(ctx context.Context, roleID id.RoleID) ([]*rbac.Permission, error) {
	return e.rbacStore().ListRolePermissions(ctx, roleID.String())
}

// AssignUserRole assigns a role to a user.
func (e *Engine) AssignUserRole(ctx context.Context, ur *rbac.UserRole) error {
	if ur.AssignedAt.IsZero() {
		ur.AssignedAt = time.Now()
	}

	if err := e.rbacStore().AssignUserRole(ctx, ur); err != nil {
		return fmt.Errorf("authsome: assign user role: %w", err)
	}

	// Resolve names for notification template variables (best-effort).
	hookMeta := e.buildRoleHookMetadata(ctx, ur.UserID, ur.RoleID)
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRoleAssign,
		Resource:   hook.ResourceRole,
		ResourceID: ur.RoleID,
		ActorID:    ur.UserID,
		Metadata:   hookMeta,
	})

	return nil
}

// UnassignUserRole removes a role from a user.
func (e *Engine) UnassignUserRole(ctx context.Context, userID id.UserID, roleID id.RoleID) error {
	if err := e.rbacStore().UnassignUserRole(ctx, userID.String(), roleID.String()); err != nil {
		return fmt.Errorf("authsome: unassign user role: %w", err)
	}

	// Resolve names for notification template variables (best-effort).
	hookMeta := e.buildRoleHookMetadata(ctx, userID.String(), roleID.String())
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionRoleUnassign,
		Resource:   hook.ResourceRole,
		ResourceID: roleID.String(),
		ActorID:    userID.String(),
		Metadata:   hookMeta,
	})

	return nil
}

// buildRoleHookMetadata resolves user and role names for notification templates.
// All lookups are best-effort — missing fields are simply omitted.
func (e *Engine) buildRoleHookMetadata(ctx context.Context, userIDStr, roleIDStr string) map[string]string {
	meta := make(map[string]string, 4)
	if uid, err := id.ParseUserID(userIDStr); err == nil {
		if u, err := e.store.GetUser(ctx, uid); err == nil {
			meta["user_name"] = u.Name()
			meta["email"] = u.Email
		}
	}
	if role, err := e.rbacStore().GetRole(ctx, roleIDStr); err == nil {
		meta["new_role"] = role.Name
	}
	return meta
}

// ListUserRoles returns all roles assigned to a user within the platform app scope.
func (e *Engine) ListUserRoles(ctx context.Context, userID id.UserID) ([]*rbac.Role, error) {
	if appID := e.PlatformAppID(); !appID.IsNil() {
		return e.rbacStore().ListUserRolesForApp(ctx, appID.String(), userID.String())
	}
	// Fallback to config.AppID (e.g. tests that skip bootstrap).
	if e.config.AppID != "" {
		return e.rbacStore().ListUserRolesForApp(ctx, e.config.AppID, userID.String())
	}
	return e.rbacStore().ListUserRoles(ctx, userID.String())
}

// GetRoleChildren returns the direct child roles of a parent role.
func (e *Engine) GetRoleChildren(ctx context.Context, roleID id.RoleID) ([]*rbac.Role, error) {
	return e.rbacStore().GetRoleChildren(ctx, roleID.String())
}

// HasPermission checks whether a user has a specific permission.
// The check walks the role hierarchy so permissions from parent roles are inherited.
func (e *Engine) HasPermission(ctx context.Context, userID id.UserID, action, resource string) (bool, error) {
	ctx = e.ensureWardenScope(ctx)

	result, err := e.wardenEng.Check(ctx, &warden.CheckRequest{
		Subject:  warden.Subject{Kind: warden.SubjectUser, ID: userID.String()},
		Action:   warden.Action{Name: action},
		Resource: warden.Resource{Type: resource},
	})
	if err != nil {
		e.logger.Warn("authsome: HasPermission error",
			log.String("user_id", userID.String()),
			log.String("action", action),
			log.String("resource", resource),
			log.String("error", err.Error()),
		)
		return false, err
	}

	if !result.Allowed {
		// Log tenant and decision for diagnostics.
		forgeAppID := ""
		forgeOrgID := ""
		if s, ok := forge.ScopeFrom(ctx); ok {
			forgeAppID = s.AppID()
			forgeOrgID = s.OrgID()
		}

		e.logger.Warn("authsome: HasPermission denied",
			log.String("user_id", userID.String()),
			log.String("action", action),
			log.String("resource", resource),
			log.String("decision", string(result.Decision)),
			log.String("reason", result.Reason),
			log.String("forge_app_id", forgeAppID),
			log.String("forge_org_id", forgeOrgID),
			log.String("platform_app_id", e.PlatformAppID().String()),
		)
	}

	return result.Allowed, nil
}

// EnsureDefaultRole assigns the default Warden role to a user if they don't
// already have one. For the platform app this is "platform_user"; for regular
// apps it is "user". Errors are silently ignored to avoid blocking user creation.
func (e *Engine) EnsureDefaultRole(ctx context.Context, appID id.AppID, userID id.UserID) {
	if !e.hasRBACStore() {
		return
	}

	// Determine default role slug based on app type.
	slug := rbac.AppUserSlug // "user"
	if appID == e.PlatformAppID() {
		slug = rbac.PlatformUserSlug // "platform_user"
	}

	role, err := e.GetRoleBySlug(ctx, appID, slug)
	if err != nil || role == nil {
		return
	}

	// Assign role (ignore duplicate assignment errors).
	_ = e.AssignUserRole(ctx, &rbac.UserRole{ //nolint:errcheck // best-effort role assign
		UserID: userID.String(),
		RoleID: role.ID,
	})
}

// promoteFirstUserToOwner assigns the platform_owner (or app owner) role to
// the first user created for an app. This must live in the engine so it works
// regardless of entry point (API handler, dashboard, SDK, etc.).
func (e *Engine) promoteFirstUserToOwner(ctx context.Context, appID id.AppID, userID id.UserID) {
	if !e.hasRBACStore() {
		return
	}

	platformID := e.PlatformAppID()
	if appID.IsNil() || platformID.IsNil() {
		return
	}

	// Only promote for the platform app.
	if appID != platformID {
		return
	}

	list, err := e.store.ListUsers(ctx, &user.Query{AppID: appID, Limit: 2})
	if err != nil || list == nil || len(list.Users) != 1 {
		return // Not the first user.
	}

	ownerRole, err := e.GetRoleBySlug(ctx, appID, rbac.PlatformOwnerSlug)
	if err != nil || ownerRole == nil {
		e.logger.Warn("authsome: could not find platform_owner role for first-user promotion",
			log.String("app_id", appID.String()),
			log.String("error", fmt.Sprintf("%v", err)),
		)
		return
	}

	if err := e.AssignUserRole(ctx, &rbac.UserRole{
		UserID: userID.String(),
		RoleID: ownerRole.ID,
	}); err != nil {
		e.logger.Warn("authsome: failed to promote first user to platform_owner",
			log.String("user_id", userID.String()),
			log.String("error", err.Error()),
		)
		return
	}

	e.logger.Info("authsome: promoted first user to platform_owner",
		log.String("user_id", userID.String()),
		log.String("app_id", appID.String()),
	)
}

// ensureWardenScope ensures the context has a warden tenant scope set.
// Warden's scopeFromContext uses forge.Scope.OrgID() as the tenant, but for
// app-scoped sessions (no org) this is empty while roles are stored with
// tenant_id = appID. We always inject the explicit warden tenant values so
// that Warden falls back to the app ID when OrgID is absent.
func (e *Engine) ensureWardenScope(ctx context.Context) context.Context {
	if e.wardenEng == nil {
		return ctx
	}

	// If forge scope is set, derive the warden tenant from it. Use OrgID if
	// present (org-scoped), otherwise fall back to AppID (app-scoped).
	if s, ok := forge.ScopeFrom(ctx); ok {
		tenantID := s.OrgID()
		if tenantID == "" {
			tenantID = s.AppID()
		}
		return warden.WithTenant(ctx, s.AppID(), tenantID)
	}

	// No forge scope at all — inject the platform app ID as tenant.
	appID := e.PlatformAppID()
	if !appID.IsNil() {
		return warden.WithTenant(ctx, appID.String(), appID.String())
	}
	// Fallback to config.AppID (e.g. tests that skip bootstrap).
	if e.config.AppID != "" {
		return warden.WithTenant(ctx, e.config.AppID, e.config.AppID)
	}
	return ctx
}

// ──────────────────────────────────────────────────
// Admin Operations
// ──────────────────────────────────────────────────

// AdminListUsers returns a paginated list of users for the given app.
func (e *Engine) AdminListUsers(ctx context.Context, q *user.Query) (*user.List, error) {
	return e.store.ListUsers(ctx, q)
}

// AdminGetUser returns a user by ID (admin access — no ownership check).
func (e *Engine) AdminGetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authsome: admin get user: %w", err)
	}
	return u, nil
}

// AdminBanUser bans a user account. Optionally sets ban reason and expiration.
func (e *Engine) AdminBanUser(ctx context.Context, adminID, userID id.UserID, reason string, expiresAt *time.Time) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: admin ban user: %w", err)
	}

	u.Banned = true
	u.BanReason = reason
	u.BanExpires = expiresAt
	u.UpdatedAt = time.Now()

	if err := e.store.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("authsome: admin ban user: %w", err)
	}

	// Revoke all active sessions for the banned user
	_ = e.store.DeleteUserSessions(ctx, userID) //nolint:errcheck // best-effort cleanup

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAdminBanUser,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    adminID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"reason":    reason,
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.audit(ctx, bridge.SeverityWarning, bridge.OutcomeSuccess, "admin_ban_user", "user", userID.String(), adminID.String(), u.AppID.String(), "admin", map[string]string{
		"reason": reason,
	})

	e.relayEvent(ctx, "admin.user.banned", u.AppID.String(), map[string]string{
		"user_id":  userID.String(),
		"admin_id": adminID.String(),
		"reason":   reason,
	})

	return nil
}

// AdminUnbanUser removes a ban from a user account.
func (e *Engine) AdminUnbanUser(ctx context.Context, adminID, userID id.UserID) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: admin unban user: %w", err)
	}

	u.Banned = false
	u.BanReason = ""
	u.BanExpires = nil
	u.UpdatedAt = time.Now()

	if err := e.store.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("authsome: admin unban user: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAdminUnbanUser,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    adminID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "admin_unban_user", "user", userID.String(), adminID.String(), u.AppID.String(), "admin", nil)

	e.relayEvent(ctx, "admin.user.unbanned", u.AppID.String(), map[string]string{
		"user_id":  userID.String(),
		"admin_id": adminID.String(),
	})

	return nil
}

// AdminDeleteUser permanently deletes a user and all associated data.
func (e *Engine) AdminDeleteUser(ctx context.Context, adminID, userID id.UserID) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: admin delete user: %w", err)
	}

	appID := u.AppID

	// Cascade delete via plugin hooks (MFA, Passkey, OAuth cleanup)
	if err := e.plugins.EmitBeforeUserDelete(ctx, userID); err != nil {
		return fmt.Errorf("authsome: admin delete user: before delete: %w", err)
	}

	// Revoke all sessions
	_ = e.store.DeleteUserSessions(ctx, userID) //nolint:errcheck // best-effort cleanup

	// Delete the user
	if err := e.store.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("authsome: admin delete user: %w", err)
	}

	// Notify plugins of completion
	e.plugins.EmitAfterUserDelete(ctx, userID)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAdminDeleteUser,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    adminID.String(),
		Tenant:     appID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.audit(ctx, bridge.SeverityCritical, bridge.OutcomeSuccess, "admin_delete_user", "user", userID.String(), adminID.String(), appID.String(), "admin", nil)

	e.relayEvent(ctx, "admin.user.deleted", appID.String(), map[string]string{
		"user_id":  userID.String(),
		"admin_id": adminID.String(),
	})

	return nil
}

// ──────────────────────────────────────────────────
// Bulk Admin Operations
// ──────────────────────────────────────────────────

// BulkImportResult holds the results of a bulk user import operation.
type BulkImportResult struct {
	Created int         `json:"created"`
	Skipped int         `json:"skipped"`
	Errors  []BulkError `json:"errors,omitempty"`
}

// BulkError records an error for a single item in a bulk operation.
type BulkError struct {
	Index int    `json:"index"`
	Email string `json:"email,omitempty"`
	Error string `json:"error"`
}

// AdminBulkImportUsers creates multiple users in a single operation. Users that
// already exist (duplicate email or username) are skipped. The operation is
// best-effort: individual failures don't abort the entire import.
func (e *Engine) AdminBulkImportUsers(ctx context.Context, adminID id.UserID, users []*user.User) (*BulkImportResult, error) {
	result := &BulkImportResult{}
	policy := e.passwordPolicy()

	for i, u := range users {
		// Validate email is required
		if u.Email == "" {
			result.Errors = append(result.Errors, BulkError{Index: i, Error: "email is required"})
			result.Skipped++
			continue
		}

		// Set defaults
		if u.ID.Prefix() == "" {
			u.ID = id.NewUserID()
		}
		now := time.Now()
		if u.CreatedAt.IsZero() {
			u.CreatedAt = now
		}
		u.UpdatedAt = now

		// Hash password if provided as plaintext (PasswordHash is empty)
		if u.PasswordHash == "" {
			result.Errors = append(result.Errors, BulkError{Index: i, Email: u.Email, Error: "password_hash is required for import"})
			result.Skipped++
			continue
		}

		// Check email uniqueness
		if _, err := e.store.GetUserByEmail(ctx, u.AppID, u.Email); err == nil {
			result.Skipped++
			continue
		}

		// Check username uniqueness
		if u.Username != "" {
			if _, err := e.store.GetUserByUsername(ctx, u.AppID, u.Username); err == nil {
				result.Errors = append(result.Errors, BulkError{Index: i, Email: u.Email, Error: "username already taken"})
				result.Skipped++
				continue
			}
		}

		if err := e.store.CreateUser(ctx, u); err != nil {
			result.Errors = append(result.Errors, BulkError{Index: i, Email: u.Email, Error: err.Error()})
			result.Skipped++
			continue
		}

		result.Created++
	}

	_ = policy // keep linter happy; available for future validation

	e.hooks.Emit(ctx, &hook.Event{
		Action:   "admin.bulk_import",
		Resource: hook.ResourceUser,
		ActorID:  adminID.String(),
		Metadata: map[string]string{
			"created": fmt.Sprintf("%d", result.Created),
			"skipped": fmt.Sprintf("%d", result.Skipped),
		},
	})

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "admin_bulk_import", "user", "", adminID.String(), "", "admin", map[string]string{
		"created": fmt.Sprintf("%d", result.Created),
		"skipped": fmt.Sprintf("%d", result.Skipped),
	})

	return result, nil
}

// AdminBulkRevokeSessions revokes all sessions for a specific user. Returns
// the number of sessions that were revoked.
func (e *Engine) AdminBulkRevokeSessions(ctx context.Context, adminID, userID id.UserID) (int, error) {
	sessions, err := e.store.ListUserSessions(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("authsome: bulk revoke sessions: %w", err)
	}

	count := len(sessions)
	if err := e.store.DeleteUserSessions(ctx, userID); err != nil {
		return 0, fmt.Errorf("authsome: bulk revoke sessions: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:   "admin.bulk_revoke_sessions",
		Resource: hook.ResourceSession,
		ActorID:  adminID.String(),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"count":   fmt.Sprintf("%d", count),
		},
	})

	e.audit(ctx, bridge.SeverityWarning, bridge.OutcomeSuccess, "admin_bulk_revoke_sessions", "session", "", adminID.String(), "", "admin", map[string]string{
		"user_id": userID.String(),
		"count":   fmt.Sprintf("%d", count),
	})

	e.relayEvent(ctx, "admin.sessions.bulk_revoked", "", map[string]string{
		"user_id":  userID.String(),
		"admin_id": adminID.String(),
		"count":    fmt.Sprintf("%d", count),
	})

	return count, nil
}

// DeleteAccount performs a self-service account deletion (GDPR right to erasure).
// This soft-deletes the user, revokes all sessions, and fires cascade cleanup via plugin hooks.
func (e *Engine) DeleteAccount(ctx context.Context, userID id.UserID) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: delete account: %w", err)
	}

	// Capture original email before anonymization for notification delivery.
	originalEmail := u.Email
	originalName := u.Name()

	// Cascade delete via plugin hooks (MFA, Passkey, OAuth cleanup)
	if err := e.plugins.EmitBeforeUserDelete(ctx, userID); err != nil {
		return fmt.Errorf("authsome: delete account: before delete: %w", err)
	}

	// Revoke all sessions
	_ = e.store.DeleteUserSessions(ctx, userID) //nolint:errcheck // best-effort cleanup

	// Soft-delete: set deleted_at timestamp
	now := time.Now()
	u.DeletedAt = &now
	u.Email = "deleted_" + userID.String() + "@deleted.local" // anonymize
	u.FirstName = ""
	u.LastName = ""
	u.Username = ""
	u.DisplayUsername = ""
	u.Image = ""
	u.Phone = ""
	u.PasswordHash = ""
	u.Metadata = nil
	u.UpdatedAt = now

	if err := e.store.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("authsome: delete account: %w", err)
	}

	// Notify plugins of completion
	e.plugins.EmitAfterUserDelete(ctx, userID)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAccountDeletion,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    userID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"email":     originalEmail,
			"user_name": originalName,
		},
	})

	e.audit(ctx, bridge.SeverityCritical, bridge.OutcomeSuccess, "account_deletion", "user", userID.String(), userID.String(), u.AppID.String(), "account", nil)

	e.relayEvent(ctx, "user.account_deleted", u.AppID.String(), map[string]string{
		"user_id": userID.String(),
	})

	return nil
}

// UserExport is the complete data export for a user (GDPR data portability).
type UserExport struct {
	User     *user.User         `json:"user"`
	Sessions []*session.Session `json:"sessions"`
	Devices  []*device.Device   `json:"devices"`
	Extra    map[string]any     `json:"extra,omitempty"`
}

// ExportUserData returns all data associated with a user for GDPR data portability.
func (e *Engine) ExportUserData(ctx context.Context, userID id.UserID) (*UserExport, error) {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authsome: export user data: %w", err)
	}

	sessions, err := e.store.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authsome: export user data: sessions: %w", err)
	}

	devices, err := e.store.ListUserDevices(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authsome: export user data: devices: %w", err)
	}

	// Collect plugin-contributed export data.
	extra := e.plugins.CollectExportData(ctx, userID)

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionDataExport,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    userID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"email":     u.Email,
			"user_name": u.Name(),
		},
	})

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "data_export", "user", userID.String(), userID.String(), u.AppID.String(), "account", nil)

	return &UserExport{
		User:     u,
		Sessions: sessions,
		Devices:  devices,
		Extra:    extra,
	}, nil
}

// ──────────────────────────────────────────────────
// Impersonation
// ──────────────────────────────────────────────────

// Impersonate creates a new session for the target user, marked as impersonated
// by the admin. The resulting session behaves as if the target user is signed in,
// but carries the impersonator's identity for audit purposes.
func (e *Engine) Impersonate(ctx context.Context, adminID, targetID id.UserID) (*user.User, *session.Session, error) {
	// Prevent self-impersonation
	if adminID == targetID {
		return nil, nil, fmt.Errorf("authsome: cannot impersonate yourself")
	}

	// Verify target user exists
	u, err := e.store.GetUser(ctx, targetID)
	if err != nil {
		return nil, nil, fmt.Errorf("authsome: impersonate: get target user: %w", err)
	}

	// Create an impersonation session (short-lived: 1 hour, non-refreshable)
	cfg := e.sessionConfigForApp(ctx, u.AppID)
	cfg.TokenTTL = 1 * time.Hour
	cfg.RefreshTokenTTL = 1 * time.Hour // same as token — not meant to be refreshed

	sess, err := e.newSession(u.AppID, u.ID, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("authsome: impersonate: create session: %w", err)
	}
	sess.ImpersonatedBy = adminID

	if err := e.store.CreateSession(ctx, sess); err != nil {
		return nil, nil, fmt.Errorf("authsome: impersonate: store session: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionImpersonate,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    adminID.String(),
		Tenant:     u.AppID.String(),
		Metadata: map[string]string{
			"target_user_id": targetID.String(),
		},
	})

	e.audit(ctx, bridge.SeverityCritical, bridge.OutcomeSuccess, "impersonate", "session", sess.ID.String(), adminID.String(), u.AppID.String(), "admin", map[string]string{
		"target_user_id": targetID.String(),
	})

	e.relayEvent(ctx, "admin.impersonate", u.AppID.String(), map[string]string{
		"admin_id":   adminID.String(),
		"target_id":  targetID.String(),
		"session_id": sess.ID.String(),
	})

	return u, sess, nil
}

// StopImpersonation terminates an impersonation session.
func (e *Engine) StopImpersonation(ctx context.Context, sessionID id.SessionID) error {
	sess, err := e.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("authsome: stop impersonation: %w", err)
	}

	if sess.ImpersonatedBy.Prefix() == "" {
		return fmt.Errorf("authsome: session is not an impersonation session")
	}

	if err := e.store.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("authsome: stop impersonation: delete session: %w", err)
	}

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "stop_impersonation", "session", sessionID.String(), sess.ImpersonatedBy.String(), sess.AppID.String(), "admin", map[string]string{
		"target_user_id": sess.UserID.String(),
	})

	return nil
}

// ──────────────────────────────────────────────────
// App operations
// ──────────────────────────────────────────────────

// GetApp retrieves an app by ID.
func (e *Engine) GetApp(ctx context.Context, appID id.AppID) (*app.App, error) {
	return e.store.GetApp(ctx, appID)
}

// GetAppBySlug retrieves an app by its slug.
func (e *Engine) GetAppBySlug(ctx context.Context, slug string) (*app.App, error) {
	return e.store.GetAppBySlug(ctx, slug)
}

// ListApps returns all apps.
func (e *Engine) ListApps(ctx context.Context) ([]*app.App, error) {
	return e.store.ListApps(ctx)
}

// CreateApp creates a new application with default environments and roles.
func (e *Engine) CreateApp(ctx context.Context, a *app.App) error {
	if a.ID.IsNil() {
		a.ID = id.NewAppID()
	}
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	// Generate a publishable key if not provided.
	if a.PublishableKey == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err == nil {
			a.PublishableKey = apikey.PublicKeyMarker(environment.TypeProduction) + hex.EncodeToString(b)
		}
	}

	if err := e.store.CreateApp(ctx, a); err != nil {
		return fmt.Errorf("authsome: create app: %w", err)
	}

	// Bootstrap default environments and roles for the new app.
	if err := e.bootstrapApp(ctx, a.ID); err != nil {
		e.logger.Warn("authsome: bootstrap new app failed",
			log.String("app_id", a.ID.String()),
			log.String("error", err.Error()),
		)
	}

	// Assign the owner role to the creating user (if user context is present).
	if userID, ok := middleware.UserIDFrom(ctx); ok && !userID.IsNil() {
		ownerRole, roleErr := e.GetRoleBySlug(ctx, a.ID, rbac.AppOwnerSlug)
		if roleErr == nil && ownerRole != nil {
			_ = e.AssignUserRole(ctx, &rbac.UserRole{ //nolint:errcheck // best-effort role assign
				UserID: userID.String(),
				RoleID: ownerRole.ID,
			})
		}
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAppCreate,
		Resource:   hook.ResourceApp,
		ResourceID: a.ID.String(),
		Metadata:   map[string]string{"app_name": a.Name, "app_slug": a.Slug},
	})

	return nil
}

// UpdateApp updates an existing application.
func (e *Engine) UpdateApp(ctx context.Context, a *app.App) error {
	a.UpdatedAt = time.Now()
	if err := e.store.UpdateApp(ctx, a); err != nil {
		return fmt.Errorf("authsome: update app: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAppUpdate,
		Resource:   hook.ResourceApp,
		ResourceID: a.ID.String(),
		Metadata:   map[string]string{"app_name": a.Name},
	})

	return nil
}

// DeleteApp removes an application.
func (e *Engine) DeleteApp(ctx context.Context, appID id.AppID) error {
	if err := e.store.DeleteApp(ctx, appID); err != nil {
		return fmt.Errorf("authsome: delete app: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionAppDelete,
		Resource:   hook.ResourceApp,
		ResourceID: appID.String(),
	})

	return nil
}

// ──────────────────────────────────────────────────
// Environment operations
// ──────────────────────────────────────────────────

// CreateEnvironment creates a new environment for an app.
func (e *Engine) CreateEnvironment(ctx context.Context, env *environment.Environment) error {
	if env.ID.Prefix() == "" {
		env.ID = id.NewEnvironmentID()
	}
	now := time.Now()
	if env.CreatedAt.IsZero() {
		env.CreatedAt = now
	}
	if env.UpdatedAt.IsZero() {
		env.UpdatedAt = now
	}
	if env.Color == "" {
		env.Color = env.Type.DefaultColor()
	}

	if err := e.store.CreateEnvironment(ctx, env); err != nil {
		return fmt.Errorf("authsome: create environment: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEnvironmentCreate,
		Resource:   hook.ResourceEnvironment,
		ResourceID: env.ID.String(),
		Tenant:     env.AppID.String(),
		Metadata:   map[string]string{"env_type": string(env.Type), "env_slug": env.Slug},
	})

	e.relayEvent(ctx, "environment.created", env.AppID.String(), map[string]string{
		"env_id":   env.ID.String(),
		"env_name": env.Name,
		"env_type": string(env.Type),
	})

	return nil
}

// GetEnvironment retrieves an environment by ID.
func (e *Engine) GetEnvironment(ctx context.Context, envID id.EnvironmentID) (*environment.Environment, error) {
	return e.store.GetEnvironment(ctx, envID)
}

// GetDefaultEnvironment retrieves the default environment for an app.
func (e *Engine) GetDefaultEnvironment(ctx context.Context, appID id.AppID) (*environment.Environment, error) {
	return e.store.GetDefaultEnvironment(ctx, appID)
}

// ListEnvironments returns all environments for an app.
func (e *Engine) ListEnvironments(ctx context.Context, appID id.AppID) ([]*environment.Environment, error) {
	return e.store.ListEnvironments(ctx, appID)
}

// GetEnvironmentBySlug retrieves an environment by app ID and slug.
func (e *Engine) GetEnvironmentBySlug(ctx context.Context, appID id.AppID, slug string) (*environment.Environment, error) {
	return e.store.GetEnvironmentBySlug(ctx, appID, slug)
}

// UpdateEnvironment updates an existing environment.
func (e *Engine) UpdateEnvironment(ctx context.Context, env *environment.Environment) error {
	env.UpdatedAt = time.Now()
	if err := e.store.UpdateEnvironment(ctx, env); err != nil {
		return fmt.Errorf("authsome: update environment: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEnvironmentUpdate,
		Resource:   hook.ResourceEnvironment,
		ResourceID: env.ID.String(),
		Tenant:     env.AppID.String(),
	})

	e.relayEvent(ctx, "environment.updated", env.AppID.String(), map[string]string{
		"env_id":   env.ID.String(),
		"env_name": env.Name,
	})

	return nil
}

// DeleteEnvironment removes an environment.
func (e *Engine) DeleteEnvironment(ctx context.Context, envID id.EnvironmentID) error {
	env, err := e.store.GetEnvironment(ctx, envID)
	if err != nil {
		return fmt.Errorf("authsome: delete environment: %w", err)
	}

	if env.IsDefault {
		return fmt.Errorf("authsome: cannot delete the default environment")
	}

	if err := e.store.DeleteEnvironment(ctx, envID); err != nil {
		return fmt.Errorf("authsome: delete environment: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEnvironmentDelete,
		Resource:   hook.ResourceEnvironment,
		ResourceID: envID.String(),
		Tenant:     env.AppID.String(),
	})

	e.relayEvent(ctx, "environment.deleted", env.AppID.String(), map[string]string{
		"env_id":   envID.String(),
		"env_name": env.Name,
	})

	return nil
}

// SetDefaultEnvironment sets an environment as the default for its app.
func (e *Engine) SetDefaultEnvironment(ctx context.Context, appID id.AppID, envID id.EnvironmentID) error {
	if err := e.store.SetDefaultEnvironment(ctx, appID, envID); err != nil {
		return fmt.Errorf("authsome: set default environment: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEnvironmentUpdate,
		Resource:   hook.ResourceEnvironment,
		ResourceID: envID.String(),
		Tenant:     appID.String(),
		Metadata:   map[string]string{"is_default": "true"},
	})

	return nil
}

// CloneEnvironment clones an environment's config and structure (roles,
// permissions, webhooks) into a new environment. User data is NOT cloned.
func (e *Engine) CloneEnvironment(ctx context.Context, req environment.CloneRequest) (*environment.CloneResult, error) {
	adapter := &storeCloneAdapter{store: e.store, rbacStore: e.rbacStore()}
	cloner := environment.NewCloner(e.store, adapter, adapter)

	result, err := cloner.Clone(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("authsome: clone environment: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionEnvironmentClone,
		Resource:   hook.ResourceEnvironment,
		ResourceID: result.Environment.ID.String(),
		Tenant:     result.Environment.AppID.String(),
		Metadata: map[string]string{
			"source_env_id": req.SourceEnvID.String(),
			"env_type":      string(req.Type),
			"env_slug":      req.Slug,
			"roles_cloned":  fmt.Sprintf("%d", result.RolesCloned),
		},
	})

	e.relayEvent(ctx, "environment.cloned", result.Environment.AppID.String(), map[string]string{
		"env_id":        result.Environment.ID.String(),
		"env_name":      result.Environment.Name,
		"source_env_id": req.SourceEnvID.String(),
	})

	return result, nil
}

// storeCloneAdapter bridges the store.Store and rbac.Store interfaces to the
// environment.CloneSource and environment.CloneTarget interfaces.
type storeCloneAdapter struct {
	store     store.Store
	rbacStore rbac.Store
}

func (a *storeCloneAdapter) ListRolesForClone(ctx context.Context, appID id.AppID, envID id.EnvironmentID) ([]*environment.RoleForClone, error) {
	roles, err := a.rbacStore.ListRoles(ctx, appID.String())
	if err != nil {
		return nil, err
	}
	var out []*environment.RoleForClone
	for _, r := range roles {
		if r.EnvID != envID.String() {
			continue
		}
		out = append(out, &environment.RoleForClone{
			ID:          r.ID,
			AppID:       r.AppID,
			EnvID:       r.EnvID,
			ParentID:    r.ParentID,
			Name:        r.Name,
			Slug:        r.Slug,
			Description: r.Description,
		})
	}
	return out, nil
}

func (a *storeCloneAdapter) ListPermissionsForClone(ctx context.Context, roleID string) ([]*environment.PermissionForClone, error) {
	perms, err := a.rbacStore.ListRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	out := make([]*environment.PermissionForClone, len(perms))
	for i, p := range perms {
		out[i] = &environment.PermissionForClone{
			ID:       p.ID,
			RoleID:   p.RoleID,
			Action:   p.Action,
			Resource: p.Resource,
		}
	}
	return out, nil
}

func (a *storeCloneAdapter) ListWebhooksForClone(ctx context.Context, appID id.AppID, envID id.EnvironmentID) ([]*environment.WebhookForClone, error) {
	hooks, err := a.store.ListWebhooks(ctx, appID)
	if err != nil {
		return nil, err
	}
	var out []*environment.WebhookForClone
	for _, wh := range hooks {
		if wh.EnvID != envID {
			continue
		}
		out = append(out, &environment.WebhookForClone{
			ID:     wh.ID.String(),
			AppID:  wh.AppID.String(),
			EnvID:  wh.EnvID.String(),
			URL:    wh.URL,
			Events: wh.Events,
			Secret: wh.Secret,
			Active: wh.Active,
		})
	}
	return out, nil
}

func (a *storeCloneAdapter) CreateClonedRole(ctx context.Context, r *environment.RoleForClone) error {
	now := time.Now()
	return a.rbacStore.CreateRole(ctx, &rbac.Role{
		ID:          r.ID,
		AppID:       r.AppID,
		EnvID:       r.EnvID,
		ParentID:    r.ParentID,
		Name:        r.Name,
		Slug:        r.Slug,
		Description: r.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (a *storeCloneAdapter) CreateClonedPermission(ctx context.Context, p *environment.PermissionForClone) error {
	return a.rbacStore.AddPermission(ctx, &rbac.Permission{
		ID:       p.ID,
		RoleID:   p.RoleID,
		Action:   p.Action,
		Resource: p.Resource,
	})
}

func (a *storeCloneAdapter) CreateClonedWebhook(ctx context.Context, w *environment.WebhookForClone) error {
	now := time.Now()
	return a.store.CreateWebhook(ctx, &webhook.Webhook{
		ID:        id.MustParse(w.ID),
		AppID:     id.MustParse(w.AppID),
		EnvID:     id.MustParse(w.EnvID),
		URL:       w.URL,
		Events:    w.Events,
		Secret:    w.Secret,
		Active:    w.Active,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

// Verify storeCloneAdapter implements both interfaces.
var (
	_ environment.CloneSource = (*storeCloneAdapter)(nil)
	_ environment.CloneTarget = (*storeCloneAdapter)(nil)
)

// ──────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────

func (e *Engine) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if e.relay == nil {
		return
	}
	if err := e.relay.Send(ctx, &bridge.WebhookEvent{
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	}); err != nil {
		e.logger.Warn("authsome: relay event failed",
			log.String("type", eventType),
			log.String("error", err.Error()),
		)
	}
}
