package decorators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
)

// Ensure MultiTenantSessionService implements session.ServiceInterface.
var _ session.ServiceInterface = (*MultiTenantSessionService)(nil)

// sessionCacheEntry holds cached session validation results.
type sessionCacheEntry struct {
	session   *session.Session
	valid     bool
	timestamp time.Time
}

// MultiTenantSessionService decorates the core session service with multi-tenancy capabilities.
type MultiTenantSessionService struct {
	sessionService session.ServiceInterface
	appService     *coreapp.ServiceImpl
	// In-memory cache to prevent redundant queries within the same request
	// This cache has a very short TTL (1 second) and is primarily to handle
	// multiple authentication strategies trying to validate the same session
	cache    sync.Map // key: token → sessionCacheEntry
	cacheTTL time.Duration
}

// NewMultiTenantSessionService creates a new multi-tenant session service.
func NewMultiTenantSessionService(sessionService session.ServiceInterface, appService *coreapp.ServiceImpl) *MultiTenantSessionService {
	svc := &MultiTenantSessionService{
		sessionService: sessionService,
		appService:     appService,
		cacheTTL:       time.Second, // Very short cache to handle concurrent auth attempts
	}

	// Start background goroutine to clean up stale cache entries
	go svc.cleanupCache()

	return svc
}

// cleanupCache periodically removes stale entries from the cache.
func (s *MultiTenantSessionService) cleanupCache() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		s.cache.Range(func(key, value any) bool {
			entry := value.(sessionCacheEntry)
			if now.Sub(entry.timestamp) > s.cacheTTL {
				s.cache.Delete(key)
			}

			return true
		})
	}
}

// Create creates a new session with app context.
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
			return s.sessionService.Create(ctx, req)
		}

		// Organizations exist - user might have just been added to one
		// Try to find which organization this user belongs to
		memberships, err := s.appService.GetUserMemberships(ctx, req.UserID)
		if err != nil || len(memberships) == 0 {
			return nil, errs.BadRequest("app context not found and user has no organizations")
		}

		// Use the first organization the user belongs to
		appID = memberships[0].AppID
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
		return nil, errs.BadRequest("user membership is not active")
	}

	// Create session using core service
	return s.sessionService.Create(ctx, req)
}

// FindByToken retrieves a session by token with app context.
func (s *MultiTenantSessionService) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	// Check cache first to avoid redundant queries within the same request
	if cached, ok := s.cache.Load(token); ok {
		entry := cached.(sessionCacheEntry)
		// Check if cache entry is still valid
		if time.Since(entry.timestamp) < s.cacheTTL {
			if !entry.valid {
				return nil, errs.BadRequest("cached invalid session")
			}

			return entry.session, nil
		}
		// Cache expired, remove it
		s.cache.Delete(token)
	}

	// Find session using core service first
	sess, err := s.sessionService.FindByToken(ctx, token)
	if err != nil {
		// Cache the error result
		s.cache.Store(token, sessionCacheEntry{
			session:   nil,
			valid:     false,
			timestamp: time.Now(),
		})

		return nil, err
	}

	if sess == nil {
		// Cache the not-found result
		s.cache.Store(token, sessionCacheEntry{
			session:   nil,
			valid:     false,
			timestamp: time.Now(),
		})

		return nil, nil
	}

	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)

	// If no app context or appID is nil, try to find one for the user
	if !ok || appID.IsNil() {
		// Use session's app/org context if available
		if !sess.AppID.IsNil() {
			// Session already has organization context - trust it
			s.cacheSession(token, sess, true)

			return sess, nil
		}

		// Check if there are any organizations
		response, err := s.appService.ListApps(ctx, &coreapp.ListAppsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1,
			},
		})
		if err != nil || len(response.Data) == 0 {
			// No organizations exist yet - allow (first user scenario)
			s.cacheSession(token, sess, true)

			return sess, nil
		}

		// Organizations exist - get user's memberships
		memberships, err := s.appService.GetUserMemberships(ctx, sess.UserID)
		if err != nil || len(memberships) == 0 {
			// User has no organizations - check if this is a newly created session
			// This handles the race condition where the organization hook hasn't completed yet
			// Allow sessions created in the last 10 seconds to give hooks time to complete
			if time.Since(sess.CreatedAt) < 10*time.Second {
				s.cacheSession(token, sess, true)

				return sess, nil
			}
			// User has no organizations and session is not newly created
			s.cacheSession(token, nil, false)

			return nil, errs.BadRequest("session user has no organization memberships")
		}

		// User has organizations - allow the session
		// (In a more sophisticated system, you might want to set org context here)
		s.cacheSession(token, sess, true)

		return sess, nil
	}

	// Organization context is present - verify user belongs to it
	member, err := s.appService.Member.FindMember(ctx, appID, sess.UserID)
	if err != nil {
		s.cacheSession(token, nil, false)

		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}

	if member.Status != coreapp.MemberStatusActive {
		s.cacheSession(token, nil, false)

		return nil, errs.BadRequest("user membership is not active")
	}

	s.cacheSession(token, sess, true)

	return sess, nil
}

// cacheSession stores a session validation result in the cache.
func (s *MultiTenantSessionService) cacheSession(token string, sess *session.Session, valid bool) {
	s.cache.Store(token, sessionCacheEntry{
		session:   sess,
		valid:     valid,
		timestamp: time.Now(),
	})
}

// Revoke revokes a session with app context.
func (s *MultiTenantSessionService) Revoke(ctx context.Context, token string) error {
	// First validate the session belongs to the organization
	sess, err := s.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	if sess == nil {
		return errs.NotFound("session not found")
	}

	// Revoke session using core service
	return s.sessionService.Revoke(ctx, token)
}

// FindByID retrieves a session by ID with app context.
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
			return nil, errs.BadRequest("session user has no organization memberships")
		}

		return sess, nil
	}

	// Organization context is present - verify user belongs to it
	member, err := s.appService.Member.FindMember(ctx, appID, sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, errs.BadRequest("user membership is not active")
	}

	return sess, nil
}

// ListSessions lists sessions with filtering within app context.
func (s *MultiTenantSessionService) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*session.ListSessionsResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, errs.BadRequest("app context not found")
	}

	// Verify the filter's app ID matches the context (or set it if not provided)
	if filter.AppID.IsNil() {
		filter.AppID = appID
	} else if filter.AppID != appID {
		return nil, errs.BadRequest("filter app ID does not match context")
	}

	// If filtering by user, verify user belongs to the organization
	if filter.UserID != nil && !filter.UserID.IsNil() {
		member, err := s.appService.Member.FindMember(ctx, appID, *filter.UserID)
		if err != nil {
			return nil, fmt.Errorf("user is not a member of this app: %w", err)
		}

		if member.Status != coreapp.MemberStatusActive {
			return nil, errs.BadRequest("user membership is not active")
		}
	}

	// List sessions using core service
	return s.sessionService.ListSessions(ctx, filter)
}

// RevokeByID revokes a session by ID with app context.
func (s *MultiTenantSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	// Get organization from context
	_, ok := ctx.Value(contexts.AppContextKey).(xid.ID)
	if !ok {
		return errs.BadRequest("app context not found")
	}

	// Get session to verify organization membership
	// Note: We need FindByID in the core service, or we get the session another way
	// For now, we'll trust the session exists and belongs to a valid user
	// In production, add FindByID to the interface and use it here

	// Revoke session using core service
	return s.sessionService.RevokeByID(ctx, id)
}

// TouchSession extends the session expiry time if sliding window is enabled.
func (s *MultiTenantSessionService) TouchSession(ctx context.Context, sess *session.Session) (*session.Session, bool, error) {
	// Get organization from context if present
	appID, ok := contexts.GetAppID(ctx)
	if ok && !appID.IsNil() {
		// Verify user is a member of the organization
		member, err := s.appService.Member.FindMember(ctx, appID, sess.UserID)
		if err != nil {
			return nil, false, fmt.Errorf("user is not a member of this app: %w", err)
		}

		if member.Status != coreapp.MemberStatusActive {
			return nil, false, errs.BadRequest("user membership is not active")
		}
	}

	// Touch session using core service
	return s.sessionService.TouchSession(ctx, sess)
}

// RefreshSession refreshes an access token using a refresh token within app context.
func (s *MultiTenantSessionService) RefreshSession(ctx context.Context, refreshToken string) (*session.RefreshResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.BadRequest("app context not found")
	}

	// Refresh session using core service
	response, err := s.sessionService.RefreshSession(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, response.Session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app: %w", err)
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, errs.BadRequest("user membership is not active")
	}

	return response, nil
}
