package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
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
	
	// Special case: First user session creation or no context provided
	// If no organization context, check if user belongs to any organization yet
	if !ok || orgID == "" {
		// Check if there are any organizations
		orgs, err := s.orgService.ListOrganizations(ctx, 1, 0)
		if err != nil || len(orgs) == 0 {
			// No organizations exist yet - this is the first user
			// Allow session creation (organization will be created by hook)
			fmt.Printf("[MultiTenancy] First user session creation allowed (no orgs yet): %s\n", req.UserID)
			return s.sessionService.Create(ctx, req)
		}
		
		// Organizations exist - user might have just been added to one
		// Try to find which organization this user belongs to
		memberships, err := s.orgService.GetUserMemberships(ctx, req.UserID.String())
		if err != nil || len(memberships) == 0 {
			return nil, fmt.Errorf("organization context not found and user has no organizations")
		}
		
		// Use the first organization the user belongs to
		orgID = memberships[0].OrganizationID
		fmt.Printf("[MultiTenancy] No org context provided, using user's primary organization for session: %s\n", orgID)
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
	// Find session using core service first
	sess, err := s.sessionService.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if sess == nil {
		return nil, nil
	}

	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	
	// If no organization context, try to find one for the user
	if !ok || orgID == "" {
		// Check if there are any organizations
		orgs, err := s.orgService.ListOrganizations(ctx, 1, 0)
		if err != nil || len(orgs) == 0 {
			// No organizations exist yet - allow (first user scenario)
			fmt.Printf("[MultiTenancy] Session lookup without org context (no orgs yet): %s\n", sess.UserID)
			return sess, nil
		}
		
		// Organizations exist - get user's memberships
		memberships, err := s.orgService.GetUserMemberships(ctx, sess.UserID.String())
		if err != nil || len(memberships) == 0 {
			// User has no organizations
			return nil, fmt.Errorf("session user has no organization memberships")
		}
		
		// User has organizations - allow the session
		// (In a more sophisticated system, you might want to set org context here)
		fmt.Printf("[MultiTenancy] Session lookup without org context, user has orgs: %s\n", sess.UserID)
		return sess, nil
	}

	// Organization context is present - verify user belongs to it
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

// ListAll lists all sessions within organization context
func (s *MultiTenantSessionService) ListAll(ctx context.Context, limit, offset int) ([]*session.Session, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Get all sessions from core service
	allSessions, err := s.sessionService.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Filter sessions by organization membership
	filteredSessions := make([]*session.Session, 0)
	for _, sess := range allSessions {
		isMember, err := s.orgService.IsUserMember(ctx, orgID, sess.UserID.String())
		if err != nil {
			continue // Skip on error
		}
		if isMember {
			filteredSessions = append(filteredSessions, sess)
		}
	}

	return filteredSessions, nil
}

// ListByUser lists sessions for a user within organization context
func (s *MultiTenantSessionService) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*session.Session, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Verify user belongs to the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to verify organization membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user not found in organization")
	}

	// List sessions using core service
	return s.sessionService.ListByUser(ctx, userID, limit, offset)
}

// RevokeByID revokes a session by ID with organization context
func (s *MultiTenantSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	// Get organization from context
	_, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return fmt.Errorf("organization context not found")
	}

	// Get session to verify organization membership
	// Note: We need FindByID in the core service, or we get the session another way
	// For now, we'll trust the session exists and belongs to a valid user
	// In production, add FindByID to the interface and use it here
	
	// Revoke session using core service
	return s.sessionService.RevokeByID(ctx, id)
}
