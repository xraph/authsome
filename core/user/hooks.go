package user

import (
	"context"

	"github.com/rs/xid"
)

// HookExecutor defines the interface for executing user-related hooks
// This interface allows the user service to execute hooks without importing the hooks package,
// avoiding circular dependencies (hooks package imports user for types).
type HookExecutor interface {
	ExecuteBeforeUserCreate(ctx context.Context, req *CreateUserRequest) error
	ExecuteAfterUserCreate(ctx context.Context, user *User) error
	ExecuteBeforeUserUpdate(ctx context.Context, userID xid.ID, req *UpdateUserRequest) error
	ExecuteAfterUserUpdate(ctx context.Context, user *User) error
	ExecuteBeforeUserDelete(ctx context.Context, userID xid.ID) error
	ExecuteAfterUserDelete(ctx context.Context, userID xid.ID) error
}
