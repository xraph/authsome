package middleware

import (
	"context"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// SessionStrategy implements authentication via session cookies
// This is the traditional cookie-based session authentication
type SessionStrategy struct {
	sessionSvc   session.ServiceInterface
	userSvc      user.ServiceInterface
	cookieName   string
	allowInQuery bool
}

// NewSessionStrategy creates a new session cookie authentication strategy
func NewSessionStrategy(
	sessionSvc session.ServiceInterface,
	userSvc user.ServiceInterface,
	cookieName string,
	allowInQuery bool,
) *SessionStrategy {
	if cookieName == "" {
		cookieName = "authsome_session"
	}
	return &SessionStrategy{
		sessionSvc:   sessionSvc,
		userSvc:      userSvc,
		cookieName:   cookieName,
		allowInQuery: allowInQuery,
	}
}

// ID returns the strategy identifier
func (s *SessionStrategy) ID() string {
	return "session-cookie"
}

// Priority returns the strategy priority (30 = medium priority for cookies)
func (s *SessionStrategy) Priority() int {
	return 30
}

// Extract attempts to extract a session token from cookies
func (s *SessionStrategy) Extract(c forge.Context) (interface{}, bool) {
	// Method 1: Cookie (primary method)
	cookie, err := c.Request().Cookie(s.cookieName)
	if err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value, true
	}

	// Method 2: Query parameter (if enabled, not recommended)
	if s.allowInQuery {
		if token := c.Request().URL.Query().Get("session_token"); token != "" {
			return token, true
		}
	}

	return nil, false
}

// Authenticate validates the session and builds auth context
func (s *SessionStrategy) Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error) {
	sessionToken, ok := credentials.(string)
	if !ok {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "invalid credentials type",
		}
	}

	// Validate session
	sess, err := s.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil || sess == nil {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "invalid or expired session",
			Err:      err,
		}
	}

	// Check expiration
	if time.Now().After(sess.ExpiresAt) {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "session expired",
		}
	}

	// Attempt to renew session if sliding window is enabled
	renewedSess, wasRenewed, err := s.sessionSvc.TouchSession(ctx, sess)
	if err == nil && wasRenewed {
		sess = renewedSess
	}

	// Load user
	usr, err := s.userSvc.FindByID(ctx, sess.UserID)
	if err != nil || usr == nil {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "user not found",
			Err:      err,
		}
	}

	// Build auth context
	authCtx := &contexts.AuthContext{
		Method:          contexts.AuthMethodSession,
		IsAuthenticated: true,
		IsUserAuth:      true,
		Session:         sess,
		User:            usr,
		AppID:           sess.AppID,
		OrganizationID:  sess.OrganizationID,
	}

	// Safely handle nullable EnvironmentID
	if sess.EnvironmentID != nil {
		authCtx.EnvironmentID = *sess.EnvironmentID
	}

	// TODO: Load RBAC roles and permissions
	// This requires RBAC service integration
	authCtx.UserRoles = []string{}
	authCtx.UserPermissions = []string{}

	return authCtx, nil
}

