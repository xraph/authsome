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
	repo       Repository
	config     Config
	webhookSvc *webhook.Service
}

// Config represents session service configuration
type Config struct {
	DefaultTTL      time.Duration
	RememberTTL     time.Duration
	AllowMultiple   bool
	RequireUserAuth bool
}

// NewService creates a new session service
func NewService(repo Repository, cfg Config, webhookSvc *webhook.Service) *Service {
	// default sensible values
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 24 * time.Hour
	}
	if cfg.RememberTTL == 0 {
		cfg.RememberTTL = 7 * 24 * time.Hour
	}
	return &Service{
		repo:       repo,
		config:     cfg,
		webhookSvc: webhookSvc,
	}
}

// Create creates a new session for a user
func (s *Service) Create(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	// Validate app context
	if req.AppID.IsNil() {
		return nil, MissingAppContext()
	}

	token, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, SessionCreationFailed(err)
	}

	id := xid.New()
	ttl := s.config.DefaultTTL
	if req.Remember {
		ttl = s.config.RememberTTL
	}

	now := time.Now().UTC()
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

	if err := s.repo.CreateSession(ctx, sess.ToSchema()); err != nil {
		return nil, SessionCreationFailed(err)
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
	// Get session before revocation for webhook event
	schemaSession, err := s.repo.FindSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionNotFound()
		}
		return err
	}

	if err := s.repo.RevokeSessionByID(ctx, id); err != nil {
		return SessionRevocationFailed(err)
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
	// Get session before revocation for webhook event
	schemaSession, err := s.repo.FindSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionNotFound()
		}
		return err
	}

	if err := s.repo.RevokeSession(ctx, token); err != nil {
		return SessionRevocationFailed(err)
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
