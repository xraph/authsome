package impersonation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/schema"
)

var (
	ErrPermissionDenied        = errors.New("impersonation: permission denied")
	ErrUserNotFound            = errors.New("impersonation: user not found")
	ErrSessionNotFound         = errors.New("impersonation: session not found")
	ErrImpersonationNotFound   = errors.New("impersonation: impersonation session not found")
	ErrAlreadyImpersonating    = errors.New("impersonation: already impersonating another user")
	ErrCannotImpersonateSelf   = errors.New("impersonation: cannot impersonate yourself")
	ErrSessionExpired          = errors.New("impersonation: session expired")
	ErrInvalidReason           = errors.New("impersonation: reason must be at least 10 characters")
	ErrInvalidDuration         = errors.New("impersonation: duration must be between 1 minute and max allowed")
)

// Config holds impersonation service configuration
type Config struct {
	// Time limits
	DefaultDurationMinutes int           `json:"default_duration_minutes" yaml:"default_duration_minutes"`
	MaxDurationMinutes     int           `json:"max_duration_minutes" yaml:"max_duration_minutes"`
	MinDurationMinutes     int           `json:"min_duration_minutes" yaml:"min_duration_minutes"`
	
	// Security
	RequireReason          bool          `json:"require_reason" yaml:"require_reason"`
	RequireTicket          bool          `json:"require_ticket" yaml:"require_ticket"`
	MinReasonLength        int           `json:"min_reason_length" yaml:"min_reason_length"`
	
	// RBAC
	RequirePermission      bool          `json:"require_permission" yaml:"require_permission"`
	ImpersonatePermission  string        `json:"impersonate_permission" yaml:"impersonate_permission"`
	
	// Audit
	AuditAllActions        bool          `json:"audit_all_actions" yaml:"audit_all_actions"` // Log every action during impersonation
	
	// Auto-cleanup
	AutoCleanupEnabled     bool          `json:"auto_cleanup_enabled" yaml:"auto_cleanup_enabled"`
	CleanupIntervalMinutes int           `json:"cleanup_interval_minutes" yaml:"cleanup_interval_minutes"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DefaultDurationMinutes:  30,
		MaxDurationMinutes:      480, // 8 hours
		MinDurationMinutes:      1,
		RequireReason:           true,
		RequireTicket:           false,
		MinReasonLength:         10,
		RequirePermission:       true,
		ImpersonatePermission:   "impersonate:user",
		AuditAllActions:         true,
		AutoCleanupEnabled:      true,
		CleanupIntervalMinutes:  15,
	}
}

// Service handles impersonation business logic
type Service struct {
	repo        Repository
	userSvc     user.ServiceInterface
	sessionSvc  session.ServiceInterface
	auditSvc    *audit.Service
	rbacSvc     *rbac.Service
	config      Config
}

// NewService creates a new impersonation service
func NewService(
	repo Repository,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	auditSvc *audit.Service,
	rbacSvc *rbac.Service,
	config Config,
) *Service {
	return &Service{
		repo:       repo,
		userSvc:    userSvc,
		sessionSvc: sessionSvc,
		auditSvc:   auditSvc,
		rbacSvc:    rbacSvc,
		config:     config,
	}
}

// Start begins an impersonation session
func (s *Service) Start(ctx context.Context, req *StartRequest) (*StartResponse, error) {
	// Validate request
	if err := s.validateStartRequest(req); err != nil {
		return nil, err
	}

	// Check if impersonator and target are the same
	if req.ImpersonatorID == req.TargetUserID {
		return nil, ErrCannotImpersonateSelf
	}

	// Check if impersonator already has an active impersonation session
	existingSession, err := s.repo.GetActive(ctx, req.ImpersonatorID, req.OrganizationID)
	if err == nil && existingSession != nil {
		return nil, ErrAlreadyImpersonating
	}

	// Verify impersonator exists and has permission
	impersonator, err := s.userSvc.FindByID(ctx, req.ImpersonatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get impersonator: %w", err)
	}

	// Check RBAC permission if enabled
	if s.config.RequirePermission && s.rbacSvc != nil {
		rbacCtx := &rbac.Context{
			Subject:  req.ImpersonatorID.String(),
			Action:   s.config.ImpersonatePermission,
			Resource: req.TargetUserID.String(),
		}
		// Note: RBAC uses roles, so we'd need to fetch user's roles
		// For now, we'll check without roles - this can be enhanced
		hasPermission := s.rbacSvc.Allowed(rbacCtx)
		if !hasPermission {
			s.auditLog(ctx, nil, "impersonation_denied", req.OrganizationID, req.ImpersonatorID, req.IPAddress, req.UserAgent, map[string]string{
				"target_user_id": req.TargetUserID.String(),
				"reason":         "permission_denied",
			})
			return nil, ErrPermissionDenied
		}
	}

	// Verify target user exists
	targetUser, err := s.userSvc.FindByID(ctx, req.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target user: %w", err)
	}

	// Determine duration
	duration := s.config.DefaultDurationMinutes
	if req.DurationMinutes > 0 {
		if req.DurationMinutes < s.config.MinDurationMinutes || req.DurationMinutes > s.config.MaxDurationMinutes {
			return nil, ErrInvalidDuration
		}
		duration = req.DurationMinutes
	}

	// Create a new session for the target user (impersonated session)
	newSession, err := s.sessionSvc.Create(ctx, &session.CreateSessionRequest{
		UserID:    targetUser.ID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	// Create impersonation record
	impersonationSession := &schema.ImpersonationSession{
		ID:              xid.New(),
		OrganizationID:  req.OrganizationID,
		ImpersonatorID:  req.ImpersonatorID,
		TargetUserID:    req.TargetUserID,
		NewSessionID:    &newSession.ID,
		SessionToken:    newSession.Token, // Store token for later revocation
		Reason:          req.Reason,
		TicketNumber:    req.TicketNumber,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		Active:          true,
		ExpiresAt:       time.Now().UTC().Add(time.Duration(duration) * time.Minute),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, impersonationSession); err != nil {
		// Cleanup: revoke the created session
		_ = s.sessionSvc.Revoke(ctx, newSession.Token)
		return nil, fmt.Errorf("failed to create impersonation record: %w", err)
	}

	// Log audit event
	s.auditLog(ctx, &impersonationSession.ID, "impersonation_started", req.OrganizationID, req.ImpersonatorID, req.IPAddress, req.UserAgent, map[string]string{
		"target_user_id":      req.TargetUserID.String(),
		"target_user_email":   targetUser.Email,
		"impersonator_email":  impersonator.Email,
		"reason":              req.Reason,
		"ticket_number":       req.TicketNumber,
		"duration_minutes":    fmt.Sprintf("%d", duration),
		"expires_at":          impersonationSession.ExpiresAt.Format(time.RFC3339),
	})

	// Create detailed audit event in impersonation audit table
	auditEvent := &schema.ImpersonationAuditEvent{
		ID:              xid.New(),
		ImpersonationID: impersonationSession.ID,
		OrganizationID:  req.OrganizationID,
		EventType:       "started",
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		Details: map[string]string{
			"target_user_id":     req.TargetUserID.String(),
			"impersonator_id":    req.ImpersonatorID.String(),
			"reason":             req.Reason,
			"ticket_number":      req.TicketNumber,
			"duration_minutes":   fmt.Sprintf("%d", duration),
		},
		CreatedAt: time.Now().UTC(),
	}
	_ = s.repo.CreateAuditEvent(ctx, auditEvent)

	// Get session token (implementation depends on session service)
	// For now, we'll return the session ID as the token
	sessionToken := newSession.Token

	return &StartResponse{
		ImpersonationID: impersonationSession.ID,
		SessionID:       newSession.ID,
		SessionToken:    sessionToken,
		ExpiresAt:       impersonationSession.ExpiresAt,
		Message:         fmt.Sprintf("Impersonating %s until %s", targetUser.Email, impersonationSession.ExpiresAt.Format(time.RFC3339)),
	}, nil
}

// End terminates an impersonation session
func (s *Service) End(ctx context.Context, req *EndRequest) (*EndResponse, error) {
	// Get impersonation session
	impersonationSession, err := s.repo.Get(ctx, req.ImpersonationID, req.OrganizationID)
	if err != nil {
		return nil, ErrImpersonationNotFound
	}

	// Verify the requester is the impersonator
	if impersonationSession.ImpersonatorID != req.ImpersonatorID {
		return nil, ErrPermissionDenied
	}

	// Check if already ended
	if !impersonationSession.Active || impersonationSession.EndedAt != nil {
		return &EndResponse{
			Success:         true,
			ImpersonationID: req.ImpersonationID,
			EndedAt:         *impersonationSession.EndedAt,
			Message:         "Impersonation session already ended",
		}, nil
	}

	// Update session
	now := time.Now().UTC()
	impersonationSession.Active = false
	impersonationSession.EndedAt = &now
	impersonationSession.EndReason = req.Reason
	if impersonationSession.EndReason == "" {
		impersonationSession.EndReason = "manual"
	}
	impersonationSession.UpdatedAt = now

	if err := s.repo.Update(ctx, impersonationSession); err != nil {
		return nil, fmt.Errorf("failed to end impersonation: %w", err)
	}

	// Revoke the impersonated session
	if impersonationSession.SessionToken != "" {
		_ = s.sessionSvc.Revoke(ctx, impersonationSession.SessionToken)
	}

	// Log audit event
	s.auditLog(ctx, &impersonationSession.ID, "impersonation_ended", req.OrganizationID, req.ImpersonatorID, "", "", map[string]string{
		"target_user_id": impersonationSession.TargetUserID.String(),
		"end_reason":     impersonationSession.EndReason,
		"duration":       time.Since(impersonationSession.CreatedAt).String(),
	})

	// Create detailed audit event
	auditEvent := &schema.ImpersonationAuditEvent{
		ID:              xid.New(),
		ImpersonationID: impersonationSession.ID,
		OrganizationID:  req.OrganizationID,
		EventType:       "ended",
		Details: map[string]string{
			"end_reason":     impersonationSession.EndReason,
			"duration":       time.Since(impersonationSession.CreatedAt).String(),
		},
		CreatedAt: time.Now().UTC(),
	}
	_ = s.repo.CreateAuditEvent(ctx, auditEvent)

	return &EndResponse{
		Success:         true,
		ImpersonationID: req.ImpersonationID,
		EndedAt:         now,
		Message:         "Impersonation session ended successfully",
	}, nil
}

// Get retrieves an impersonation session
func (s *Service) Get(ctx context.Context, req *GetRequest) (*SessionInfo, error) {
	impersonationSession, err := s.repo.Get(ctx, req.ImpersonationID, req.OrganizationID)
	if err != nil {
		return nil, ErrImpersonationNotFound
	}

	return s.toSessionInfo(ctx, impersonationSession), nil
}

// List retrieves impersonation sessions with filters
func (s *Service) List(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	sessions, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list impersonation sessions: %w", err)
	}

	total, err := s.repo.Count(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to count impersonation sessions: %w", err)
	}

	infos := make([]*SessionInfo, len(sessions))
	for i, session := range sessions {
		infos[i] = s.toSessionInfo(ctx, session)
	}

	return &ListResponse{
		Sessions: infos,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

// Verify checks if a session is an impersonation session
func (s *Service) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	impersonationSession, err := s.repo.GetBySessionID(ctx, req.SessionID)
	if err != nil || impersonationSession == nil {
		return &VerifyResponse{
			IsImpersonating: false,
		}, nil
	}

	if !impersonationSession.IsActive() {
		return &VerifyResponse{
			IsImpersonating: false,
		}, nil
	}

	return &VerifyResponse{
		IsImpersonating: true,
		ImpersonationID: &impersonationSession.ID,
		ImpersonatorID:  &impersonationSession.ImpersonatorID,
		TargetUserID:    &impersonationSession.TargetUserID,
		ExpiresAt:       &impersonationSession.ExpiresAt,
	}, nil
}

// ListAuditEvents retrieves audit events
func (s *Service) ListAuditEvents(ctx context.Context, req *AuditListRequest) ([]*AuditEvent, int, error) {
	events, err := s.repo.ListAuditEvents(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit events: %w", err)
	}

	total, err := s.repo.CountAuditEvents(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	auditEvents := make([]*AuditEvent, len(events))
	for i, event := range events {
		auditEvents[i] = &AuditEvent{
			ID:              event.ID,
			ImpersonationID: event.ImpersonationID,
			OrganizationID:  event.OrganizationID,
			EventType:       event.EventType,
			Action:          event.Action,
			Resource:        event.Resource,
			IPAddress:       event.IPAddress,
			UserAgent:       event.UserAgent,
			Details:         event.Details,
			CreatedAt:       event.CreatedAt,
		}
	}

	return auditEvents, total, nil
}

// ExpireSessions expires old impersonation sessions (run as cron job)
func (s *Service) ExpireSessions(ctx context.Context) (int, error) {
	return s.repo.ExpireOldSessions(ctx)
}

// Helper methods

func (s *Service) validateStartRequest(req *StartRequest) error {
	if req.Reason == "" && s.config.RequireReason {
		return ErrInvalidReason
	}
	if len(req.Reason) < s.config.MinReasonLength && s.config.RequireReason {
		return ErrInvalidReason
	}
	if req.TicketNumber == "" && s.config.RequireTicket {
		return errors.New("impersonation: ticket number is required")
	}
	return nil
}

func (s *Service) toSessionInfo(ctx context.Context, session *schema.ImpersonationSession) *SessionInfo {
	info := &SessionInfo{
		ID:             session.ID,
		OrganizationID: session.OrganizationID,
		ImpersonatorID: session.ImpersonatorID,
		TargetUserID:   session.TargetUserID,
		Reason:         session.Reason,
		TicketNumber:   session.TicketNumber,
		Active:         session.Active,
		ExpiresAt:      session.ExpiresAt,
		EndedAt:        session.EndedAt,
		EndReason:      session.EndReason,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}

	// Enrich with user data (best effort, don't fail if can't fetch)
	if impersonator, err := s.userSvc.FindByID(ctx, session.ImpersonatorID); err == nil {
		info.ImpersonatorEmail = impersonator.Email
		info.ImpersonatorName = impersonator.Name
	}
	if targetUser, err := s.userSvc.FindByID(ctx, session.TargetUserID); err == nil {
		info.TargetEmail = targetUser.Email
		info.TargetName = targetUser.Name
	}

	return info
}

func (s *Service) auditLog(ctx context.Context, impersonationID *xid.ID, action string, orgID, userID xid.ID, ip, ua string, metadata map[string]string) {
	if s.auditSvc == nil {
		return
	}

	resource := "impersonation"
	if impersonationID != nil {
		resource = fmt.Sprintf("impersonation:%s", impersonationID.String())
	}

	metadataStr := ""
	if len(metadata) > 0 {
		// Simple key-value format
		for k, v := range metadata {
			metadataStr += fmt.Sprintf("%s=%s; ", k, v)
		}
	}

	_ = s.auditSvc.Log(ctx, &userID, action, resource, ip, ua, metadataStr)
}

