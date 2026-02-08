package jwt

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated jwt plugin

// Plugin implements the jwt plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new jwt plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "jwt"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateJWTKey CreateJWTKey creates a new JWT signing key
func (p *Plugin) CreateJWTKey(ctx context.Context, req *authsome.CreateJWTKeyRequest) error {
	path := "/jwt/keys"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListJWTKeys ListJWTKeys lists JWT signing keys
func (p *Plugin) ListJWTKeys(ctx context.Context, req *authsome.ListJWTKeysRequest) error {
	path := "/jwt/keys"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// GetJWKS GetJWKS returns the JSON Web Key Set
func (p *Plugin) GetJWKS(ctx context.Context) error {
	path := "/jwt/jwks"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GenerateToken GenerateToken generates a new JWT token
func (p *Plugin) GenerateToken(ctx context.Context, req *authsome.GenerateTokenRequest) error {
	path := "/jwt/generate"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// VerifyToken VerifyToken verifies a JWT token
func (p *Plugin) VerifyToken(ctx context.Context, req *authsome.VerifyTokenRequest) error {
	path := "/jwt/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

