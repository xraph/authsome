package oidcprovider

import (
	"context"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// IntrospectionService handles RFC 7662 token introspection operations
type IntrospectionService struct {
	tokenRepo  *repo.OAuthTokenRepository
	clientRepo *repo.OAuthClientRepository
	userSvc    UserService
}

// UserService interface for getting user information during introspection
// Note: This uses interface{} for userID to allow xid.ID or string
type UserService interface {
	FindByID(ctx context.Context, userID xid.ID) (interface{}, error)
}

// NewIntrospectionService creates a new token introspection service
func NewIntrospectionService(tokenRepo *repo.OAuthTokenRepository, clientRepo *repo.OAuthClientRepository, userSvc UserService) *IntrospectionService {
	return &IntrospectionService{
		tokenRepo:  tokenRepo,
		clientRepo: clientRepo,
		userSvc:    userSvc,
	}
}

// IntrospectToken implements RFC 7662 token introspection
// Returns token metadata if active, or {active: false} if inactive/invalid
func (s *IntrospectionService) IntrospectToken(ctx context.Context, req *TokenIntrospectionRequest, requestingClientID string) (*TokenIntrospectionResponse, error) {
	if req.Token == "" {
		return nil, errs.BadRequest("token parameter is required")
	}
	
	// Try to find token based on hint or try both types
	var token *schema.OAuthToken
	var err error
	
	switch req.TokenTypeHint {
	case "access_token", "":
		// Try access token first
		token, err = s.tokenRepo.FindByAccessToken(ctx, req.Token)
		if err != nil {
			return nil, errs.InternalError(err)
		}
		
		// If not found and no explicit hint, try refresh token
		if token == nil && req.TokenTypeHint == "" {
			token, err = s.tokenRepo.FindByRefreshToken(ctx, req.Token)
			if err != nil {
				return nil, errs.InternalError(err)
			}
		}
		
	case "refresh_token":
		token, err = s.tokenRepo.FindByRefreshToken(ctx, req.Token)
		if err != nil {
			return nil, errs.InternalError(err)
		}
		
	default:
		return nil, errs.BadRequest("invalid token_type_hint")
	}
	
	// If token not found, return inactive
	if token == nil {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Check if token is valid (not expired, not revoked, nbf check)
	if !token.IsValid() {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Authorization check: client can only introspect their own tokens
	if token.ClientID != requestingClientID {
		// Return inactive instead of error for security (don't leak token existence)
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Build introspection response
	response := &TokenIntrospectionResponse{
		Active:    true,
		Scope:     token.Scope,
		ClientID:  token.ClientID,
		TokenType: token.TokenType,
		Exp:       token.ExpiresAt.Unix(),
		Iat:       token.CreatedAt.Unix(),
		Sub:       token.UserID.String(),
		Jti:       token.JTI,
		Iss:       token.Issuer,
		Aud:       token.Audience,
	}
	
	// Add optional fields
	if token.NotBefore != nil {
		response.Nbf = token.NotBefore.Unix()
	}
	
	// Get username if available
	if s.userSvc != nil {
		userObj, err := s.userSvc.FindByID(ctx, token.UserID)
		if err == nil && userObj != nil {
			// Extract username/email from user object using type assertion or reflection
			if userMap, ok := userObj.(map[string]interface{}); ok {
				if username, ok := userMap["username"].(string); ok && username != "" {
					response.Username = username
				} else if email, ok := userMap["email"].(string); ok && email != "" {
					response.Username = email
				}
			}
		}
	}
	
	return response, nil
}

// ValidateIntrospectionRequest validates the introspection request
func (s *IntrospectionService) ValidateIntrospectionRequest(req *TokenIntrospectionRequest) error {
	if req.Token == "" {
		return errs.BadRequest("token parameter is required")
	}
	
	// Validate token_type_hint if provided
	if req.TokenTypeHint != "" {
		validHints := []string{"access_token", "refresh_token"}
		valid := false
		for _, hint := range validHints {
			if req.TokenTypeHint == hint {
				valid = true
				break
			}
		}
		if !valid {
			return errs.BadRequest("invalid token_type_hint: must be 'access_token' or 'refresh_token'")
		}
	}
	
	return nil
}

// IntrospectByJTI introspects a token by its JWT ID
func (s *IntrospectionService) IntrospectByJTI(ctx context.Context, jti string, requestingClientID string) (*TokenIntrospectionResponse, error) {
	if jti == "" {
		return nil, errs.BadRequest("jti parameter is required")
	}
	
	token, err := s.tokenRepo.FindByJTI(ctx, jti)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	
	if token == nil {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Check validity
	if !token.IsValid() {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Authorization check
	if token.ClientID != requestingClientID {
		return &TokenIntrospectionResponse{Active: false}, nil
	}
	
	// Build response
	response := &TokenIntrospectionResponse{
		Active:    true,
		Scope:     token.Scope,
		ClientID:  token.ClientID,
		TokenType: token.TokenType,
		Exp:       token.ExpiresAt.Unix(),
		Iat:       token.CreatedAt.Unix(),
		Sub:       token.UserID.String(),
		Jti:       token.JTI,
		Iss:       token.Issuer,
		Aud:       token.Audience,
	}
	
	if token.NotBefore != nil {
		response.Nbf = token.NotBefore.Unix()
	}
	
	return response, nil
}

// GetTokenScopes parses and returns the scopes from a token
func GetTokenScopes(scope string) []string {
	if scope == "" {
		return []string{}
	}
	return strings.Fields(scope)
}

// HasScope checks if a token has a specific scope
func HasScope(tokenScope, requiredScope string) bool {
	scopes := GetTokenScopes(tokenScope)
	for _, s := range scopes {
		if s == requiredScope {
			return true
		}
	}
	return false
}

