package user

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// USER REPOSITORY INTERFACE (ISP Compliant)
// =============================================================================

// Repository defines the interface for user storage operations
// This follows the Interface Segregation Principle from core/app architecture.
type Repository interface {
	// Create creates a new user
	Create(ctx context.Context, user *schema.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.User, error)

	// FindByEmail finds a user by email (global search)
	FindByEmail(ctx context.Context, email string) (*schema.User, error)

	// FindByAppAndEmail finds a user by app ID and email (app-scoped search)
	FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*schema.User, error)

	// FindByUsername finds a user by username
	FindByUsername(ctx context.Context, username string) (*schema.User, error)

	// Update updates a user
	Update(ctx context.Context, user *schema.User) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id xid.ID) error

	// ListUsers lists users with pagination and filtering
	ListUsers(ctx context.Context, filter *ListUsersFilter) (*pagination.PageResponse[*schema.User], error)

	// CountUsers counts users with filtering
	CountUsers(ctx context.Context, filter *CountUsersFilter) (int, error)
}
