package decorators

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
)

// Ensure MultiTenantSessionService implements session.ServiceInterface
var _ session.ServiceInterface = (*MultiTenantSessionService)(nil)

// MultiTenantSessionService decorates the core session service with multi-tenancy capabilities
type MultiTenantSessionService struct {
	sessionService session.ServiceInterface
	orgService     *organization.Service
}

// NewMultiTenantSessionService creates a new multi-tenant session service
func NewMultiTenantSessionService(sessionService session.ServiceInterface, orgService *organization.Service) *MultiTenantSessionService {
	return &MultiTenantSessionService{
		sessionService: sessionService,
		orgService:     orgService,
	}
}

// Create creates a new session with organization context
func (s *MultiTenantSessionService) Create(ctx context.Context, req *session.CreateSessionRequest) (*session.Session, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Verify organization exists
	_, err := s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}

	// Verify user belongs to the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, req.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to verify organization membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user not found in organization")
	}

	// Create session using core service
	return s.sessionService.Create(ctx, req)
}

// FindByToken retrieves a session by token with organization context
func (s *MultiTenantSessionService) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Find session using core service
	sess, err := s.sessionService.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if sess == nil {
		return nil, nil
	}

	// Verify user belongs to the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, sess.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to verify organization membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("session user not found in organization")
	}

	return sess, nil
}

// Revoke revokes a session with organization context
func (s *MultiTenantSessionService) Revoke(ctx context.Context, token string) error {
	// First validate the session belongs to the organization
	sess, err := s.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	if sess == nil {
		return fmt.Errorf("session not found")
	}

	// Revoke session using core service
	return s.sessionService.Revoke(ctx, token)
}
