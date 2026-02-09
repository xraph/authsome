package environment

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// ENVIRONMENT REPOSITORY INTERFACE (ISP Compliant)
// =============================================================================

// Repository defines the interface for environment data access
// This follows the Interface Segregation Principle from core/app and core/jwt architecture.
type Repository interface {
	// Environment CRUD
	Create(ctx context.Context, env *schema.Environment) error
	FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error)
	FindByAppAndSlug(ctx context.Context, appID xid.ID, slug string) (*schema.Environment, error)
	FindDefaultByApp(ctx context.Context, appID xid.ID) (*schema.Environment, error)

	// ListEnvironments lists environments with pagination and filtering
	ListEnvironments(ctx context.Context, filter *ListEnvironmentsFilter) (*pagination.PageResponse[*schema.Environment], error)

	// CountByApp counts environments for an app
	CountByApp(ctx context.Context, appID xid.ID) (int, error)

	Update(ctx context.Context, env *schema.Environment) error
	Delete(ctx context.Context, id xid.ID) error

	// Promotion operations
	CreatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error
	FindPromotionByID(ctx context.Context, id xid.ID) (*schema.EnvironmentPromotion, error)

	// ListPromotions lists promotions with pagination and filtering
	ListPromotions(ctx context.Context, filter *ListPromotionsFilter) (*pagination.PageResponse[*schema.EnvironmentPromotion], error)

	UpdatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error
}
