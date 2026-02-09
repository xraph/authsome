package jwt

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// JWT KEY REPOSITORY INTERFACE (ISP Compliant)
// =============================================================================

// Repository defines the interface for JWT key storage operations
// This follows the Interface Segregation Principle from core/app architecture.
type Repository interface {
	// CreateJWTKey creates a new JWT key
	CreateJWTKey(ctx context.Context, key *schema.JWTKey) error

	// FindJWTKeyByID finds a JWT key by ID
	FindJWTKeyByID(ctx context.Context, id xid.ID) (*schema.JWTKey, error)

	// FindJWTKeyByKeyID finds a JWT key by key ID and app ID
	FindJWTKeyByKeyID(ctx context.Context, keyID string, appID xid.ID) (*schema.JWTKey, error)

	// FindPlatformJWTKeyByKeyID finds a platform JWT key by key ID
	FindPlatformJWTKeyByKeyID(ctx context.Context, keyID string) (*schema.JWTKey, error)

	// ListJWTKeys lists JWT keys with pagination and filtering
	ListJWTKeys(ctx context.Context, filter *ListJWTKeysFilter) (*pagination.PageResponse[*schema.JWTKey], error)

	// ListPlatformJWTKeys lists platform JWT keys with pagination
	ListPlatformJWTKeys(ctx context.Context, filter *ListJWTKeysFilter) (*pagination.PageResponse[*schema.JWTKey], error)

	// UpdateJWTKey updates a JWT key
	UpdateJWTKey(ctx context.Context, key *schema.JWTKey) error

	// UpdateJWTKeyUsage updates the usage statistics for a JWT key
	UpdateJWTKeyUsage(ctx context.Context, keyID string) error

	// DeactivateJWTKey deactivates a JWT key
	DeactivateJWTKey(ctx context.Context, id xid.ID) error

	// DeleteJWTKey soft deletes a JWT key
	DeleteJWTKey(ctx context.Context, id xid.ID) error

	// CleanupExpiredJWTKeys removes expired JWT keys
	CleanupExpiredJWTKeys(ctx context.Context) (int64, error)

	// CountJWTKeys counts JWT keys for an app
	CountJWTKeys(ctx context.Context, appID xid.ID) (int, error)
}
