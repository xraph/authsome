package user

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/types"
)

// ServiceInterface defines the contract for user service operations
// This allows plugins to decorate the service with additional behavior
type ServiceInterface interface {
	Create(ctx context.Context, req *CreateUserRequest) (*User, error)
	FindByID(ctx context.Context, id xid.ID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, u *User, req *UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id xid.ID) error
	List(ctx context.Context, opts types.PaginationOptions) ([]*User, int, error)
	Search(ctx context.Context, query string, opts types.PaginationOptions) ([]*User, int, error)
	Count(ctx context.Context) (int, error)
	CountCreatedToday(ctx context.Context) (int, error)
}

// Ensure Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)
