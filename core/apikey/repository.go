package apikey

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the interface for API key storage operations
// Following Interface Segregation Principle (ISP) - works with schema types
type Repository interface {
	// Create/Read operations
	CreateAPIKey(ctx context.Context, key *schema.APIKey) error
	FindAPIKeyByID(ctx context.Context, id xid.ID) (*schema.APIKey, error)
	FindAPIKeyByPrefix(ctx context.Context, prefix string) (*schema.APIKey, error)

	// List with pagination
	ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) (*pagination.PageResponse[*schema.APIKey], error)

	// Update operations
	UpdateAPIKey(ctx context.Context, key *schema.APIKey) error
	UpdateAPIKeyUsage(ctx context.Context, id xid.ID, ip, userAgent string) error
	DeactivateAPIKey(ctx context.Context, id xid.ID) error
	DeleteAPIKey(ctx context.Context, id xid.ID) error

	// Count operations
	CountAPIKeys(ctx context.Context, appID xid.ID, envID *xid.ID, orgID *xid.ID, userID *xid.ID) (int, error)

	// Maintenance
	CleanupExpiredAPIKeys(ctx context.Context) (int, error)
}

// =============================================================================
// ENVIRONMENT REPOSITORY INTERFACE
// =============================================================================

// EnvironmentRepository provides environment lookup for prefix generation
// This is a minimal interface to avoid tight coupling with the environment service
type EnvironmentRepository interface {
	FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error)
}
