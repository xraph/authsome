package oidcprovider

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
)

// RevokeTokenService handles RFC 7009 token revocation operations
type RevokeTokenService struct {
	tokenRepo *repo.OAuthTokenRepository
}

// NewRevokeTokenService creates a new token revocation service
func NewRevokeTokenService(tokenRepo *repo.OAuthTokenRepository) *RevokeTokenService {
	return &RevokeTokenService{
		tokenRepo: tokenRepo,
	}
}

// RevokeToken implements RFC 7009 token revocation
// Returns nil even if token doesn't exist (per RFC 7009 spec)
func (s *RevokeTokenService) RevokeToken(ctx context.Context, req *TokenRevocationRequest) error {
	if req.Token == "" {
		return errs.BadRequest("token parameter is required")
	}
	
	// Determine token type from hint or try both
	switch req.TokenTypeHint {
	case "access_token", "":
		// Try access token first
		if err := s.tokenRepo.RevokeToken(ctx, req.Token); err != nil {
			// If hint was explicit and failed, return error
			if req.TokenTypeHint == "access_token" {
				return errs.InternalError(err)
			}
			// Otherwise, try refresh token
			if err := s.tokenRepo.RevokeByRefreshToken(ctx, req.Token); err != nil {
				// Per RFC 7009, return success even if token not found
				return nil
			}
		}
		
	case "refresh_token":
		// Try refresh token
		if err := s.tokenRepo.RevokeByRefreshToken(ctx, req.Token); err != nil {
			// Per RFC 7009, return success even if token not found
			return nil
		}
		
	default:
		return errs.BadRequest("invalid token_type_hint")
	}
	
	return nil
}

// RevokeByJTI revokes a token by its JWT ID
func (s *RevokeTokenService) RevokeByJTI(ctx context.Context, jti string) error {
	if jti == "" {
		return errs.BadRequest("jti parameter is required")
	}
	
	return s.tokenRepo.RevokeByJTI(ctx, jti)
}

// AuthenticateClient performs client authentication for the revocation endpoint
// Supports client_secret_basic and client_secret_post methods
func (s *RevokeTokenService) AuthenticateClient(r *http.Request, clientRepo *repo.OAuthClientRepository) (*ClientAuthResult, error) {
	ctx := r.Context()
	
	// Try HTTP Basic Authentication first (client_secret_basic)
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Basic ") {
			encoded := strings.TrimPrefix(auth, "Basic ")
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, errs.Unauthorized()
			}
			
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				return nil, errs.Unauthorized()
			}
			
			clientID, clientSecret := parts[0], parts[1]
			
			// Verify client credentials
			client, err := clientRepo.FindByClientID(ctx, clientID)
			if err != nil {
				return nil, errs.InternalError(err)
			}
			if client == nil || client.ClientSecret != clientSecret {
				return nil, errs.Unauthorized()
			}
			
			return &ClientAuthResult{
				ClientID:      clientID,
				Authenticated: true,
				Method:        "client_secret_basic",
			}, nil
		}
	}
	
	// Try POST body authentication (client_secret_post)
	if err := r.ParseForm(); err != nil {
		return nil, errs.BadRequest("failed to parse form data")
	}
	
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	
	if clientID == "" {
		return nil, errs.BadRequest("client_id is required")
	}
	
	// If no client secret provided, check if client allows 'none' auth method
	if clientSecret == "" {
		client, err := clientRepo.FindByClientID(ctx, clientID)
		if err != nil {
			return nil, errs.InternalError(err)
		}
		if client == nil {
			return nil, errs.Unauthorized()
		}
		
		// Only allow 'none' method for public clients (PKCE-enabled)
		if client.TokenEndpointAuthMethod == "none" && client.RequirePKCE {
			return &ClientAuthResult{
				ClientID:      clientID,
				Authenticated: true,
				Method:        "none",
			}, nil
		}
		
		return nil, errs.Unauthorized()
	}
	
	// Verify client credentials
	client, err := clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	if client == nil || client.ClientSecret != clientSecret {
		return nil, errs.Unauthorized()
	}
	
	return &ClientAuthResult{
		ClientID:      clientID,
		Authenticated: true,
		Method:        "client_secret_post",
	}, nil
}

