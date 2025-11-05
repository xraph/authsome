package user

import (
	"context"
	"github.com/rs/xid"
	"time"
)

// Repository defines the user repository interface
type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id xid.ID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id xid.ID) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*User, error)
	Count(ctx context.Context) (int, error)
	CountSearch(ctx context.Context, query string) (int, error)
	CountCreatedSince(ctx context.Context, since time.Time) (int, error)
}
