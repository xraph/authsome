package bearer

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// BearerStrategy implements authentication via Bearer tokens in Authorization header
// This strategy extracts session tokens from "Authorization: Bearer <token>" headers
// and validates them as session tokens (not JWT tokens)
type BearerStrategy struct {
	sessionSvc session.ServiceInterface
	userSvc    user.ServiceInterface
	config     Config
	logger     forge.Logger
}

// NewBearerStrategy creates a new bearer token authentication strategy
func NewBearerStrategy(
	sessionSvc session.ServiceInterface,
	userSvc user.ServiceInterface,
	config Config,
	logger forge.Logger,
) *BearerStrategy {
	return &BearerStrategy{
		sessionSvc: sessionSvc,
		userSvc:    userSvc,
		config:     config,
		logger:     logger,
	}
}

// ID returns the strategy identifier
func (s *BearerStrategy) ID() string {
	return "bearer"
}

// Priority returns the strategy priority (20 = medium-high priority for bearer tokens)
// This runs after API keys (10) but before cookies (30)
func (s *BearerStrategy) Priority() int {
	return 20
}

// Extract attempts to extract a bearer token from the Authorization header
func (s *BearerStrategy) Extract(c forge.Context) (interface{}, bool) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}

	// Check if it's a bearer token with configured prefix
	prefix := s.config.TokenPrefix + " "

	var hasPrefix bool
	if !s.config.CaseSensitive {
		hasPrefix = len(authHeader) >= len(prefix) &&
			strings.EqualFold(authHeader[:len(prefix)], prefix)
	} else {
		hasPrefix = strings.HasPrefix(authHeader, prefix)
	}

	if !hasPrefix {
		return nil, false
	}

	// Extract token
	token := strings.TrimPrefix(authHeader, prefix)
	if !s.config.CaseSensitive {
		// Re-extract with case-insensitive prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 {
			token = strings.TrimSpace(parts[1])
		}
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, false
	}

	// Don't treat API keys as bearer tokens (they have their own strategy)
	if strings.HasPrefix(token, "pk_") ||
		strings.HasPrefix(token, "sk_") ||
		strings.HasPrefix(token, "rk_") {
		return nil, false
	}

	return token, true
}

// Authenticate validates the bearer token and builds auth context
func (s *BearerStrategy) Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error) {
	token, ok := credentials.(string)
	if !ok {
		return nil, &BearerAuthError{
			Message: "invalid credentials type",
		}
	}

	// Validate the token using session service
	sess, err := s.sessionSvc.FindByToken(ctx, token)
	if err != nil {
		s.logger.Debug("failed to find session by bearer token",
			forge.F("error", err.Error()))
		return nil, &BearerAuthError{
			Message: "invalid or expired bearer token",
			Err:     err,
		}
	}

	// Check if session is valid
	if sess == nil {
		return nil, &BearerAuthError{
			Message: "session not found",
		}
	}

	// Check expiration
	if time.Now().After(sess.ExpiresAt) {
		s.logger.Debug("bearer token session expired",
			forge.F("session_id", sess.ID.String()),
			forge.F("expires_at", sess.ExpiresAt))
		return nil, &BearerAuthError{
			Message: "session expired",
		}
	}

	// Get user information
	usr, err := s.userSvc.FindByID(ctx, sess.UserID)
	if err != nil {
		s.logger.Warn("failed to find user for bearer token session",
			forge.F("user_id", sess.UserID.String()),
			forge.F("error", err.Error()))
		return nil, &BearerAuthError{
			Message: "user not found",
			Err:     err,
		}
	}

	if usr == nil {
		s.logger.Warn("user not found for valid bearer token session",
			forge.F("user_id", sess.UserID.String()))
		return nil, &BearerAuthError{
			Message: "user not found",
		}
	}

	// Build AuthContext
	authCtx := &contexts.AuthContext{
		Session:         sess,
		User:            usr,
		AppID:           sess.AppID,
		OrganizationID:  sess.OrganizationID,
		Method:          contexts.AuthMethodSession,
		IsAuthenticated: true,
		IsUserAuth:      true,
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

// BearerAuthError represents a bearer authentication error
type BearerAuthError struct {
	Message string
	Err     error
}

func (e *BearerAuthError) Error() string {
	if e.Err != nil {
		return "bearer auth: " + e.Message + ": " + e.Err.Error()
	}
	return "bearer auth: " + e.Message
}

func (e *BearerAuthError) Unwrap() error {
	return e.Err
}
