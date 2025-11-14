package decorators

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
)

// Ensure MultiTenantSessionService implements session.ServiceInterface
var _ session.ServiceInterface = (*MultiTenantSessionService)(nil)

// MultiTenantSessionService decorates the core session service with multi-tenancy capabilities
type MultiTenantSessionService struct {
	sessionService session.ServiceInterface
	appService     *coreapp.ServiceImpl
}

// NewMultiTenantSessionService creates a new multi-tenant session service
func NewMultiTenantSessionService(sessionService session.ServiceInterface, appService *coreapp.ServiceImpl) *MultiTenantSessionService {
	return &MultiTenantSessionService{
		sessionService: sessionService,
		appService:     appService,
	}
}

// Create creates a new session with app context
func (s *MultiTenantSessionService) Create(ctx context.Context, req *session.CreateSessionRequest) (*session.Session, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)

	// Special case: First user session creation or no context provided
	// If no app context, check if user belongs to any organization yet
	if !ok || appID.IsNil() {
		// Check if there are any organizations
		response, err := s.appService.ListApps(ctx, &coreapp.ListAppsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1,
			},
		})
		if err != nil || len(response.Data) == 0 {
			// No organizations exist yet - this is the first user
			// Allow session creation (organization will be created by hook)
			fmt.Printf("[MultiTenancy] First user session creation allowed (no orgs yet): %s\n", req.UserID)
			return s.sessionService.Create(ctx, req)
		}

		// Organizations exist - user might have just been added to one
		// Try to find which organization this user belongs to
		memberships, err := s.appService.GetUserMemberships(ctx, req.UserID)
		if err != nil || len(memberships) == 0 {
			return nil, fmt.Errorf("app context not found and user has no organizations")
		}

		// Use the first organization the user belongs to
		appID = memberships[0].AppID
		fmt.Printf("[MultiTenancy] No app context provided, using user's primary app for session: %s\n", appID)
	}

	// Verify organization exists
	_, err := s.appService.App.FindAppByID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}

	// Verify user belongs to the organization
	member, err := s.appService.Member.FindMember(ctx, appID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}
	if member.Status != coreapp.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	// Create session using core service
	return s.sessionService.Create(ctx, req)
}

// FindByToken retrieves a session by token with app context
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
	appID, ok := contexts.GetAppID(ctx)

	// If no app context or appID is nil, try to find one for the user
	if !ok || appID.IsNil() {
		// Check if there are any organizations
		response, err := s.appService.ListApps(ctx, &coreapp.ListAppsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1,
			},
		})
		if err != nil || len(response.Data) == 0 {
			// No organizations exist yet - allow (first user scenario)
			fmt.Printf("[MultiTenancy] Session lookup without org context (no orgs yet): %s\n", sess.UserID)
			return sess, nil
		}

		// Organizations exist - get user's memberships
		memberships, err := s.appService.GetUserMemberships(ctx, sess.UserID)
		if err != nil || len(memberships) == 0 {
			// User has no organizations - check if this is a newly created session
			// This handles the race condition where the organization hook hasn't completed yet
			// Allow sessions created in the last 10 seconds to give hooks time to complete
			if time.Since(sess.CreatedAt) < 10*time.Second {
				fmt.Printf("[MultiTenancy] Session lookup: newly created session (created %v ago), allowing while org hook completes: %s\n", time.Since(sess.CreatedAt), sess.UserID)
				return sess, nil
			}
			// User has no organizations and session is not newly created
			return nil, fmt.Errorf("session user has no organization memberships")
		}

		// User has organizations - allow the session
		// (In a more sophisticated system, you might want to set org context here)
		fmt.Printf("[MultiTenancy] Session lookup without org context, user has orgs: %s\n", sess.UserID)
		return sess, nil
	}

	// Organization context is present - verify user belongs to it
	member, err := s.appService.Member.FindMember(ctx, appID, sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}
	if member.Status != coreapp.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return sess, nil
}

// Revoke revokes a session with app context
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

// FindByID retrieves a session by ID with app context
func (s *MultiTenantSessionService) FindByID(ctx context.Context, id xid.ID) (*session.Session, error) {
	// Get session using core service first
	sess, err := s.sessionService.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if sess == nil {
		return nil, nil
	}

	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		// No app context - allow if user has any organization memberships
		memberships, err := s.appService.GetUserMemberships(ctx, sess.UserID)
		if err != nil || len(memberships) == 0 {
			return nil, fmt.Errorf("session user has no organization memberships")
		}
		return sess, nil
	}

	// Organization context is present - verify user belongs to it
	member, err := s.appService.Member.FindMember(ctx, appID, sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}
	if member.Status != coreapp.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return sess, nil
}

// ListSessions lists sessions with filtering within app context
func (s *MultiTenantSessionService) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*session.ListSessionsResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, fmt.Errorf("app context not found")
	}

	// Verify the filter's app ID matches the context (or set it if not provided)
	if filter.AppID.IsNil() {
		filter.AppID = appID
	} else if filter.AppID != appID {
		return nil, fmt.Errorf("filter app ID does not match context")
	}

	// If filtering by user, verify user belongs to the organization
	if filter.UserID != nil && !filter.UserID.IsNil() {
		member, err := s.appService.Member.FindMember(ctx, appID, *filter.UserID)
		if err != nil {
			return nil, fmt.Errorf("user is not a member of this app: %w", err)
		}
		if member.Status != coreapp.MemberStatusActive {
			return nil, fmt.Errorf("user membership is not active")
		}
	}

	// List sessions using core service
	return s.sessionService.ListSessions(ctx, filter)
}

// RevokeByID revokes a session by ID with app context
func (s *MultiTenantSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	// Get organization from context
	_, ok := ctx.Value(contexts.AppContextKey).(xid.ID)
	if !ok {
		return fmt.Errorf("app context not found")
	}

	// Get session to verify organization membership
	// Note: We need FindByID in the core service, or we get the session another way
	// For now, we'll trust the session exists and belongs to a valid user
	// In production, add FindByID to the interface and use it here

	// Revoke session using core service
	return s.sessionService.RevokeByID(ctx, id)
}
