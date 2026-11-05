package user

import (
	"context"

	"github.com/xraph/authsome/id"
)

// BatchGetter is implemented by stores that can read many users in one call.
// GetUsers returns the users with the given IDs that exist and are not
// deleted, in no particular order. An ID with no live user is left out,
// not an error. Duplicate IDs are read once. An empty list returns an
// empty result without touching the database.
type BatchGetter interface {
	GetUsers(ctx context.Context, userIDs []id.UserID) ([]*User, error)
}
