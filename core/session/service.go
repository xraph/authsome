package session

import (
	"context"
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
	token, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, err
	}
	id := xid.New()
	ttl := s.config.DefaultTTL
	if req.Remember {
		ttl = s.config.RememberTTL
	}
	now := time.Now().UTC()
	sess := &Session{
		ID:        id,
		Token:     token,
		UserID:    req.UserID,
		ExpiresAt: now.Add(ttl),
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, err
	}

	// Emit webhook event for user login
	if s.webhookSvc != nil {
		data := map[string]interface{}{
			"session_id": sess.ID.String(),
			"user_id":    sess.UserID.String(),
			"ip_address": sess.IPAddress,
			"user_agent": sess.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, "user.login", "default", data) // TODO: Get orgID from context
	}

	return sess, nil
}

// FindByToken retrieves a session by token
func (s *Service) FindByToken(ctx context.Context, token string) (*Session, error) {
	return s.repo.FindByToken(ctx, token)
}

// FindByID retrieves a session by ID
func (s *Service) FindByID(ctx context.Context, id xid.ID) (*Session, error) {
	return s.repo.FindByID(ctx, id)
}

// ListAll retrieves all sessions (for admin dashboard)
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Session, error) {
	return s.repo.ListAll(ctx, limit, offset)
}

// ListByUser retrieves sessions for a specific user
func (s *Service) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*Session, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

// RevokeByID revokes a session by ID
func (s *Service) RevokeByID(ctx context.Context, id xid.ID) error {
	// Get session before revocation for webhook event
	session, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.RevokeByID(ctx, id); err != nil {
		return err
	}

	// Emit webhook event for session revocation
	if s.webhookSvc != nil && session != nil {
		data := map[string]interface{}{
			"session_id": session.ID.String(),
			"user_id":    session.UserID.String(),
			"ip_address": session.IPAddress,
			"user_agent": session.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, "session.revoked", "default", data) // TODO: Get orgID from context
	}

	return nil
}

// Revoke revokes a session by token
func (s *Service) Revoke(ctx context.Context, token string) error {
	// Get session before revocation for webhook event
	session, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	if err := s.repo.Revoke(ctx, token); err != nil {
		return err
	}

	// Emit webhook event for user logout
	if s.webhookSvc != nil && session != nil {
		data := map[string]interface{}{
			"session_id": session.ID.String(),
			"user_id":    session.UserID.String(),
			"ip_address": session.IPAddress,
			"user_agent": session.UserAgent,
		}
		go s.webhookSvc.EmitEvent(ctx, "user.logout", "default", data) // TODO: Get orgID from context
	}

	return nil
}
