package user

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// ServiceInterface defines the contract for user service operations
// This allows plugins to decorate the service with additional behavior.
type ServiceInterface interface {
	// Create creates a new user in the specified app
	Create(ctx context.Context, req *CreateUserRequest) (*User, error)

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id xid.ID) (*User, error)

	// FindByEmail finds a user by email (global search)
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByAppAndEmail finds a user by app ID and email (app-scoped search)
	FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*User, error)

	// FindByUsername finds a user by username
	FindByUsername(ctx context.Context, username string) (*User, error)

	// Update updates a user
	Update(ctx context.Context, u *User, req *UpdateUserRequest) (*User, error)

	// UpdatePassword updates a user's password directly
	UpdatePassword(ctx context.Context, userID xid.ID, hashedPassword string) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id xid.ID) error

	// ListUsers lists users with pagination and filtering
	ListUsers(ctx context.Context, filter *ListUsersFilter) (*pagination.PageResponse[*User], error)

	// CountUsers counts users with filtering
	CountUsers(ctx context.Context, filter *CountUsersFilter) (int, error)

	// SetHookRegistry sets the hook registry for lifecycle events
	SetHookRegistry(registry any)

	// GetHookRegistry returns the hook registry
	GetHookRegistry() any

	// SetVerificationRepo sets the verification repository for password resets
	SetVerificationRepo(repo any)

	// GetVerificationRepo returns the verification repository
	GetVerificationRepo() any
}

// Ensure Service implements ServiceInterface.
var _ ServiceInterface = (*Service)(nil)
