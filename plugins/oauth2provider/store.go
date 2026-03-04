package oauth2provider

import (
	"context"
	"errors"

	"github.com/xraph/authsome/id"
)

// Store errors.
var (
	ErrClientNotFound = errors.New("oauth2: client not found")
	ErrCodeNotFound   = errors.New("oauth2: authorization code not found")
)

// Store persists OAuth2 clients and authorization codes.
type Store interface {
	// Clients
	CreateClient(ctx context.Context, c *OAuth2Client) error
	GetClient(ctx context.Context, clientID string) (*OAuth2Client, error)
	GetClientByID(ctx context.Context, id id.OAuth2ClientID) (*OAuth2Client, error)
	ListClients(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error)
	DeleteClient(ctx context.Context, id id.OAuth2ClientID) error

	// Authorization codes
	CreateAuthCode(ctx context.Context, code *AuthorizationCode) error
	GetAuthCode(ctx context.Context, code string) (*AuthorizationCode, error)
	ConsumeAuthCode(ctx context.Context, code string) error
}
