package oidcprovider

import (
	"context"
	"fmt"
	"strings"

	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// JWTValidationStrategy validates OAuth/OIDC JWT access tokens
// This allows any AuthSome endpoint to accept JWT tokens from the OIDC provider.
type JWTValidationStrategy struct {
	oidcJWTSvc *JWTService // OIDC provider's JWT service (has the JWKS keys!)
	userSvc    user.ServiceInterface
	issuer     string // Expected issuer (e.g., "http://localhost:4000")
	audience   string // Expected audience (optional)
	logger     forge.Logger
}

// Ensure JWTValidationStrategy implements AuthStrategy.
var _ middleware.AuthStrategy = (*JWTValidationStrategy)(nil)

// NewJWTValidationStrategy creates a new JWT validation strategy.
func NewJWTValidationStrategy(
	oidcJWTSvc *JWTService, // Use OIDC provider's JWT service, not core JWT service!
	userSvc user.ServiceInterface,
	issuer string,
	audience string,
	logger forge.Logger,
) *JWTValidationStrategy {
	return &JWTValidationStrategy{
		oidcJWTSvc: oidcJWTSvc,
		userSvc:    userSvc,
		issuer:     issuer,
		audience:   audience,
		logger:     logger,
	}
}

// ID returns the strategy identifier.
func (s *JWTValidationStrategy) ID() string {
	return "oidc-jwt"
}

// Priority returns 15 (after API keys 10, before bearer session tokens 20).
func (s *JWTValidationStrategy) Priority() int {
	return 15
}

// Extract attempts to extract a JWT from Authorization header.
func (s *JWTValidationStrategy) Extract(c forge.Context) (any, bool) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}

	// Must be Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return nil, false
	}

	// Don't process API keys (let API key strategy handle them)
	if strings.HasPrefix(token, "pk_") ||
		strings.HasPrefix(token, "sk_") ||
		strings.HasPrefix(token, "rk_") {
		return nil, false
	}

	// Quick check: JWTs have 3 parts separated by dots
	if strings.Count(token, ".") != 2 {
		return nil, false // Not a JWT, let bearer strategy try as session token
	}

	return token, true
}

// Authenticate validates the JWT and builds auth context.
func (s *JWTValidationStrategy) Authenticate(ctx context.Context, credentials any) (*contexts.AuthContext, error) {
	token, ok := credentials.(string)
	if !ok {
		return nil, &JWTAuthError{Message: "invalid credentials type"}
	}

	// Validate JWT signature using OIDC provider's JWT service
	// This service has the JWKS keys that signed the token
	jwtToken, err := s.oidcJWTSvc.VerifyToken(token)
	if err != nil {
		s.logger.Warn("JWT verification failed", forge.F("error", err.Error()))

		return nil, &JWTAuthError{
			Message: "JWT verification failed",
			Err:     err,
		}
	}

	if !jwtToken.Valid {
		s.logger.Warn("JWT token invalid")

		return nil, &JWTAuthError{
			Message: "invalid JWT token",
		}
	}

	// Extract claims from verified token
	claims, ok := jwtToken.Claims.(jwt2.MapClaims)
	if !ok {
		s.logger.Warn("failed to parse JWT claims")

		return nil, &JWTAuthError{
			Message: "failed to parse JWT claims",
		}
	}

	// Verify issuer
	issuer, _ := claims["iss"].(string)
	if issuer != s.issuer {
		return nil, &JWTAuthError{
			Message: fmt.Sprintf("invalid issuer: expected %s, got %s", s.issuer, issuer),
		}
	}

	// Verify audience if configured
	if s.audience != "" {
		audValid := false

		switch aud := claims["aud"].(type) {
		case string:
			audValid = aud == s.audience
		case []any:
			for _, a := range aud {
				if audStr, ok := a.(string); ok && audStr == s.audience {
					audValid = true

					break
				}
			}
		}

		if !audValid {
			return nil, &JWTAuthError{
				Message: "invalid audience: expected " + s.audience,
			}
		}
	}

	// Extract user ID from subject claim
	subject, _ := claims["sub"].(string)
	if subject == "" {
		return nil, &JWTAuthError{
			Message: "missing subject claim",
		}
	}

	userID, err := xid.FromString(subject)
	if err != nil {
		s.logger.Warn("invalid user ID in JWT", forge.F("subject", subject), forge.F("error", err.Error()))

		return nil, &JWTAuthError{
			Message: "invalid user ID in token",
			Err:     err,
		}
	}

	// Load user
	usr, err := s.userSvc.FindByID(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to load user from JWT", forge.F("user_id", userID.String()), forge.F("error", err.Error()))

		return nil, &JWTAuthError{
			Message: "user not found",
			Err:     err,
		}
	}

	if usr == nil {
		s.logger.Warn("user not found", forge.F("user_id", userID.String()))

		return nil, &JWTAuthError{
			Message: "user not found",
		}
	}

	// Build auth context
	authCtx := &contexts.AuthContext{
		User:            usr,
		Method:          contexts.AuthMethodSession,
		IsAuthenticated: true,
		IsUserAuth:      true,
	}

	// TODO: Extract app_id, environment_id, organization_id from claims if present
	// TODO: Load user RBAC roles/permissions

	return authCtx, nil
}

// JWTAuthError represents a JWT authentication error.
type JWTAuthError struct {
	Message string
	Err     error
}

func (e *JWTAuthError) Error() string {
	if e.Err != nil {
		return "jwt auth: " + e.Message + ": " + e.Err.Error()
	}

	return "jwt auth: " + e.Message
}

func (e *JWTAuthError) Unwrap() error {
	return e.Err
}
