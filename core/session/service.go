package session

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/crypto"
)

// Service provides session-related operations
type Service struct {
	repo         Repository
	config       Config
	webhookSvc   *webhook.Service
	hookExecutor HookExecutor
}

// Config represents session service configuration
type Config struct {
	// Basic TTL settings
	DefaultTTL      time.Duration
	RememberTTL     time.Duration
	AllowMultiple   bool
	RequireUserAuth bool

	// Sliding session renewal (Option 1)
	EnableSlidingWindow bool          // Enable automatic session renewal
	SlidingRenewalAfter time.Duration // Only renew if session age > this (default: 5 min)

	// Refresh token support (Option 3)
	EnableRefreshTokens bool          // Enable refresh token pattern
	RefreshTokenTTL     time.Duration // Refresh token lifetime (default: 30 days)
	AccessTokenTTL      time.Duration // Short-lived access token (default: 15 min)
}

// NewService creates a new session service
func NewService(repo Repository, cfg Config, webhookSvc *webhook.Service, hookExecutor HookExecutor) *Service {
	// default sensible values
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 24 * time.Hour
	}
	if cfg.RememberTTL == 0 {
		cfg.RememberTTL = 7 * 24 * time.Hour
	}

	// Sliding window defaults
	if cfg.EnableSlidingWindow && cfg.SlidingRenewalAfter == 0 {
		cfg.SlidingRenewalAfter = 5 * time.Minute // Only renew if session is >5 min old
	}

	// Refresh token defaults
	if cfg.EnableRefreshTokens {
		if cfg.RefreshTokenTTL == 0 {
			cfg.RefreshTokenTTL = 30 * 24 * time.Hour // 30 days
		}
		if cfg.AccessTokenTTL == 0 {
			cfg.AccessTokenTTL = 15 * time.Minute // 15 minutes
		}
	}

	return &Service{
		repo:         repo,
		config:       cfg,
		webhookSvc:   webhookSvc,
		hookExecutor: hookExecutor,
	}
}

// Create creates a new session for a user
func (s *Service) Create(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	// Execute before session create hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSessionCreate(ctx, req); err != nil {
			return nil, err
		}
	}

	// Validate app context
	if req.AppID.IsNil() {
		return nil, MissingAppContext()
	}

	token, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, SessionCreationFailed(err)
	}

	id := xid.New()
	now := time.Now().UTC()

	// Determine TTL based on refresh token config
	var ttl time.Duration
	if s.config.EnableRefreshTokens {
		// Use short-lived access token when refresh tokens are enabled
		ttl = s.config.AccessTokenTTL
	} else {
		// Use standard TTL for regular sessions
		ttl = s.config.DefaultTTL
		if req.Remember {
			ttl = s.config.RememberTTL
		}
	}

	sess := &Session{
		ID:             id,
		Token:          token,
		AppID:          req.AppID,
		EnvironmentID:  req.EnvironmentID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		ExpiresAt:      now.Add(ttl),
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Generate refresh token if enabled
	if s.config.EnableRefreshTokens {
		refreshToken, err := crypto.GenerateToken(32)
		if err != nil {
			return nil, SessionCreationFailed(err)
		}

		refreshExpiresAt := now.Add(s.config.RefreshTokenTTL)
		sess.RefreshToken = &refreshToken
		sess.RefreshTokenExpiresAt = &refreshExpiresAt
	}

	if err := s.repo.CreateSession(ctx, sess.ToSchema()); err != nil {
		return nil, SessionCreationFailed(err)
	}

	// Execute after session create hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSessionCreate(ctx, sess); err != nil {
			// Log error but don't fail - session is already created
		}
	}

	// Emit webhook event for user login
	if s.webhookSvc != nil {
		envID := xid.ID{}
		if sess.EnvironmentID != nil {
			envID = *sess.EnvironmentID
		}
		data := map[string]interface{}{
			"session_id": sess.ID.String(),
			"user_id":    sess.UserID.String(),
			"app_id":     sess.AppID.String(),
			"ip_address": sess.IPAddress,
			"user_agent": sess.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, sess.AppID, envID, "user.login", data)
	}

	return sess, nil
}

// FindByToken retrieves a session by token
func (s *Service) FindByToken(ctx context.Context, token string) (*Session, error) {
	schemaSession, err := s.repo.FindSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, SessionNotFound()
		}
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(schemaSession.ExpiresAt) {
		return nil, SessionExpired()
	}

	return FromSchemaSession(schemaSession), nil
}

// TouchSession extends the session expiry time if sliding window is enabled
// Returns the updated session and whether it was actually updated
func (s *Service) TouchSession(ctx context.Context, sess *Session) (*Session, bool, error) {
	if !s.config.EnableSlidingWindow {
		return sess, false, nil // Sliding window disabled
	}

	now := time.Now().UTC()

	// Calculate time since last update
	timeSinceUpdate := now.Sub(sess.UpdatedAt)

	// Only update if enough time has passed (throttling)
	if timeSinceUpdate < s.config.SlidingRenewalAfter {
		return sess, false, nil // Too soon to renew
	}

	// Calculate new expiry time
	ttl := s.config.DefaultTTL
	if sess.ExpiresAt.Sub(sess.CreatedAt) > s.config.DefaultTTL {
		// Session was created with RememberTTL, maintain that
		ttl = s.config.RememberTTL
	}

	newExpiresAt := now.Add(ttl)

	// Update session in database
	if err := s.repo.UpdateSessionExpiry(ctx, sess.ID, newExpiresAt); err != nil {
		return nil, false, err
	}

	// Return updated session
	sess.ExpiresAt = newExpiresAt
	sess.UpdatedAt = now

	return sess, true, nil
}

// FindByID retrieves a session by ID
func (s *Service) FindByID(ctx context.Context, id xid.ID) (*Session, error) {
	schemaSession, err := s.repo.FindSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, SessionNotFound()
		}
		return nil, err
	}

	return FromSchemaSession(schemaSession), nil
}

// ListSessions retrieves sessions with filtering and pagination
func (s *Service) ListSessions(ctx context.Context, filter *ListSessionsFilter) (*ListSessionsResponse, error) {
	pageResp, err := s.repo.ListSessions(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema sessions to DTOs
	dtoSessions := FromSchemaSessions(pageResp.Data)

	return &ListSessionsResponse{
		Data:       dtoSessions,
		Pagination: pageResp.Pagination,
	}, nil
}

// RevokeByID revokes a session by ID
func (s *Service) RevokeByID(ctx context.Context, id xid.ID) error {
	// Get session before revocation for webhook event and hooks
	schemaSession, err := s.repo.FindSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionNotFound()
		}
		return err
	}

	// Execute before session revoke hooks (using empty token string for ID-based revocation)
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSessionRevoke(ctx, ""); err != nil {
			return err
		}
	}

	if err := s.repo.RevokeSessionByID(ctx, id); err != nil {
		return SessionRevocationFailed(err)
	}

	// Execute after session revoke hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSessionRevoke(ctx, id); err != nil {
			// Log error but don't fail - session is already revoked
		}
	}

	// Emit webhook event for session revocation
	if s.webhookSvc != nil && schemaSession != nil {
		envID := xid.ID{}
		if schemaSession.EnvironmentID != nil {
			envID = *schemaSession.EnvironmentID
		}
		data := map[string]interface{}{
			"session_id": schemaSession.ID.String(),
			"user_id":    schemaSession.UserID.String(),
			"app_id":     schemaSession.AppID.String(),
			"ip_address": schemaSession.IPAddress,
			"user_agent": schemaSession.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, schemaSession.AppID, envID, "session.revoked", data)
	}

	return nil
}

// Revoke revokes a session by token
func (s *Service) Revoke(ctx context.Context, token string) error {
	// Get session before revocation for webhook event and hooks
	schemaSession, err := s.repo.FindSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionNotFound()
		}
		return err
	}

	// Execute before session revoke hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSessionRevoke(ctx, token); err != nil {
			return err
		}
	}

	if err := s.repo.RevokeSession(ctx, token); err != nil {
		return SessionRevocationFailed(err)
	}

	// Execute after session revoke hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSessionRevoke(ctx, schemaSession.ID); err != nil {
			// Log error but don't fail - session is already revoked
		}
	}

	// Emit webhook event for user logout
	if s.webhookSvc != nil && schemaSession != nil {
		envID := xid.ID{}
		if schemaSession.EnvironmentID != nil {
			envID = *schemaSession.EnvironmentID
		}
		data := map[string]interface{}{
			"session_id": schemaSession.ID.String(),
			"user_id":    schemaSession.UserID.String(),
			"app_id":     schemaSession.AppID.String(),
			"ip_address": schemaSession.IPAddress,
			"user_agent": schemaSession.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, schemaSession.AppID, envID, "user.logout", data)
	}

	return nil
}

// RefreshSession refreshes an access token using a refresh token (Option 3)
// This implements the refresh token pattern for long-lived sessions
func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	if !s.config.EnableRefreshTokens {
		return nil, errors.New("refresh tokens are not enabled")
	}

	// Find session by refresh token
	schemaSession, err := s.repo.FindSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, err
	}

	// Check if refresh token is expired
	now := time.Now().UTC()
	if schemaSession.RefreshTokenExpiresAt != nil && now.After(*schemaSession.RefreshTokenExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Generate new access token
	newAccessToken, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, err
	}

	// Calculate new access token expiry
	accessTokenExpiresAt := now.Add(s.config.AccessTokenTTL)

	// Optional: Rotate refresh token for enhanced security
	var newRefreshToken string
	var refreshTokenExpiresAt time.Time

	if s.config.RefreshTokenTTL > 0 {
		// Keep the same refresh token but extend its expiry (safer option)
		newRefreshToken = refreshToken
		refreshTokenExpiresAt = now.Add(s.config.RefreshTokenTTL)
	} else {
		// Rotate refresh token (more secure but requires client to handle token rotation)
		newRefreshToken, err = crypto.GenerateToken(32)
		if err != nil {
			return nil, err
		}
		refreshTokenExpiresAt = now.Add(s.config.RefreshTokenTTL)
	}

	// Update session in database
	if err := s.repo.RefreshSessionTokens(ctx, schemaSession.ID, newAccessToken, accessTokenExpiresAt, newRefreshToken, refreshTokenExpiresAt); err != nil {
		return nil, err
	}

	// Build updated session
	sess := FromSchemaSession(schemaSession)
	sess.Token = newAccessToken
	sess.ExpiresAt = accessTokenExpiresAt
	sess.RefreshToken = &newRefreshToken
	sess.RefreshTokenExpiresAt = &refreshTokenExpiresAt
	sess.LastRefreshedAt = &now
	sess.UpdatedAt = now

	return &RefreshResponse{
		Session:          sess,
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		ExpiresAt:        accessTokenExpiresAt,
		RefreshExpiresAt: refreshTokenExpiresAt,
	}, nil
}
