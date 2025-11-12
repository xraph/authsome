package environment

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Repository defines the interface for environment data access
type Repository interface {
	// Environment CRUD
	Create(ctx context.Context, env *schema.Environment) error
	FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error)
	FindByAppAndSlug(ctx context.Context, appID xid.ID, slug string) (*schema.Environment, error)
	FindDefaultByApp(ctx context.Context, appID xid.ID) (*schema.Environment, error)
	ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.Environment, error)
	CountByApp(ctx context.Context, appID xid.ID) (int, error)
	Update(ctx context.Context, env *schema.Environment) error
	Delete(ctx context.Context, id xid.ID) error

	// Promotion operations
	CreatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error
	FindPromotionByID(ctx context.Context, id xid.ID) (*schema.EnvironmentPromotion, error)
	ListPromotionsByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.EnvironmentPromotion, error)
	UpdatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error
}
