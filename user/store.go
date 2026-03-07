package user

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for user operations.
type Store interface {
	CreateUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, userID id.UserID) (*User, error)
	GetUserByEmail(ctx context.Context, appID id.AppID, email string) (*User, error)
	GetUserByPhone(ctx context.Context, appID id.AppID, phone string) (*User, error)
	GetUserByUsername(ctx context.Context, appID id.AppID, username string) (*User, error)
	UpdateUser(ctx context.Context, u *User) error
	DeleteUser(ctx context.Context, userID id.UserID) error
	ListUsers(ctx context.Context, q *UserQuery) (*UserList, error)
}
