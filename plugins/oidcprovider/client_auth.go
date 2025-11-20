package oidcprovider

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// ClientAuthenticator handles OAuth2/OIDC client authentication
type ClientAuthenticator struct {
	clientRepo *repo.OAuthClientRepository
}

// NewClientAuthenticator creates a new client authenticator
func NewClientAuthenticator(clientRepo *repo.OAuthClientRepository) *ClientAuthenticator {
	return &ClientAuthenticator{
		clientRepo: clientRepo,
	}
}

// AuthenticateClient authenticates an OAuth2 client using various methods
// Supports: client_secret_basic, client_secret_post, and none (for public clients with PKCE)
func (c *ClientAuthenticator) AuthenticateClient(ctx context.Context, r *http.Request) (*ClientAuthResult, *schema.OAuthClient, error) {
	// Try HTTP Basic Authentication first (client_secret_basic)
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Basic ") {
			return c.authenticateBasic(ctx, auth)
		}
	}
	
	// Try POST body authentication (client_secret_post)
	if err := r.ParseForm(); err != nil {
		return nil, nil, errs.BadRequest("failed to parse form data")
	}
	
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	
	if clientID == "" {
		return nil, nil, errs.BadRequest("client_id is required")
	}
	
	// If no client secret provided, check if client allows 'none' auth method
	if clientSecret == "" {
		client, err := c.clientRepo.FindByClientID(ctx, clientID)
		if err != nil {
			return nil, nil, errs.InternalError(err)
		}
		if client == nil {
			return nil, nil, errs.UnauthorizedWithMessage("invalid client credentials")
		}
		
		// Only allow 'none' method for public clients (PKCE-enabled)
		if client.TokenEndpointAuthMethod == "none" && client.RequirePKCE {
			return &ClientAuthResult{
				ClientID:      clientID,
				Authenticated: true,
				Method:        "none",
			}, client, nil
		}
		
		return nil, nil, errs.UnauthorizedWithMessage("client authentication required")
	}
	
	// Authenticate with client_secret_post
	return c.authenticatePost(ctx, clientID, clientSecret)
}

// authenticateBasic authenticates a client using HTTP Basic Authentication
func (c *ClientAuthenticator) authenticateBasic(ctx context.Context, authHeader string) (*ClientAuthResult, *schema.OAuthClient, error) {
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, nil, errs.UnauthorizedWithMessage("invalid authorization header")
	}
	
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, nil, errs.UnauthorizedWithMessage("invalid authorization header format")
	}
	
	clientID, clientSecret := parts[0], parts[1]
	
	// Verify client credentials
	client, err := c.clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return nil, nil, errs.InternalError(err)
	}
	if client == nil {
		return nil, nil, errs.UnauthorizedWithMessage("invalid client credentials")
	}
	
	// Verify client secret (use constant-time comparison in production)
	if client.ClientSecret != clientSecret {
		return nil, nil, errs.UnauthorizedWithMessage("invalid client credentials")
	}
	
	// Check if client supports this auth method
	if client.TokenEndpointAuthMethod != "client_secret_basic" && client.TokenEndpointAuthMethod != "" {
		return nil, nil, errs.UnauthorizedWithMessage("client authentication method not supported")
	}
	
	return &ClientAuthResult{
		ClientID:      clientID,
		Authenticated: true,
		Method:        "client_secret_basic",
	}, client, nil
}

// authenticatePost authenticates a client using POST body parameters
func (c *ClientAuthenticator) authenticatePost(ctx context.Context, clientID, clientSecret string) (*ClientAuthResult, *schema.OAuthClient, error) {
	// Verify client credentials
	client, err := c.clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return nil, nil, errs.InternalError(err)
	}
	if client == nil {
		return nil, nil, errs.UnauthorizedWithMessage("invalid client credentials")
	}
	
	// Verify client secret (use constant-time comparison in production)
	if client.ClientSecret != clientSecret {
		return nil, nil, errs.UnauthorizedWithMessage("invalid client credentials")
	}
	
	// Check if client supports this auth method
	if client.TokenEndpointAuthMethod != "client_secret_post" && client.TokenEndpointAuthMethod != "" {
		return nil, nil, errs.UnauthorizedWithMessage("client authentication method not supported")
	}
	
	return &ClientAuthResult{
		ClientID:      clientID,
		Authenticated: true,
		Method:        "client_secret_post",
	}, client, nil
}

// ValidateClientForEndpoint validates that a client can access a specific endpoint
func (c *ClientAuthenticator) ValidateClientForEndpoint(client *schema.OAuthClient, endpoint string) error {
	// Endpoint-specific validation rules
	switch endpoint {
	case "token":
		// All authenticated clients can access token endpoint
		return nil
		
	case "introspect":
		// Only confidential clients can introspect
		if client.TokenEndpointAuthMethod == "none" {
			return errs.PermissionDenied("introspect", "confidential clients only")
		}
		return nil
		
	case "revoke":
		// All authenticated clients can revoke their own tokens
		return nil
		
	default:
		return nil
	}
}

// IsPublicClient checks if a client is a public client (no client secret)
func (c *ClientAuthenticator) IsPublicClient(client *schema.OAuthClient) bool {
	return client.TokenEndpointAuthMethod == "none"
}

// IsConfidentialClient checks if a client is a confidential client (has client secret)
func (c *ClientAuthenticator) IsConfidentialClient(client *schema.OAuthClient) bool {
	return !c.IsPublicClient(client)
}

