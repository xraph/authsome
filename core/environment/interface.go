package environment

import (
	"context"

	"github.com/rs/xid"
)

// =============================================================================
// ENVIRONMENT SERVICE INTERFACE
// =============================================================================

// EnvironmentService defines the contract for environment service operations
// This allows plugins to decorate the service with additional behavior
// Following the pattern from core/jwt and core/app architecture
type EnvironmentService interface {
	// Environment operations
	CreateEnvironment(ctx context.Context, req *CreateEnvironmentRequest) (*Environment, error)
	CreateDefaultEnvironment(ctx context.Context, appID xid.ID) (*Environment, error)
	GetEnvironment(ctx context.Context, id xid.ID) (*Environment, error)
	GetEnvironmentBySlug(ctx context.Context, appID xid.ID, slug string) (*Environment, error)
	GetDefaultEnvironment(ctx context.Context, appID xid.ID) (*Environment, error)
	ListEnvironments(ctx context.Context, filter *ListEnvironmentsFilter) (*ListEnvironmentsResponse, error)
	UpdateEnvironment(ctx context.Context, id xid.ID, req *UpdateEnvironmentRequest) (*Environment, error)
	DeleteEnvironment(ctx context.Context, id xid.ID) error

	// Promotion operations
	PromoteEnvironment(ctx context.Context, req *PromoteEnvironmentRequest) (*Promotion, error)
	GetPromotion(ctx context.Context, id xid.ID) (*Promotion, error)
	ListPromotions(ctx context.Context, filter *ListPromotionsFilter) (*ListPromotionsResponse, error)
}

// Ensure Service implements EnvironmentService
var _ EnvironmentService = (*Service)(nil)
