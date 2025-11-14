package session

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Repository defines session persistence operations
// Following ISP - works with schema types
type Repository interface {
	// Create/Read operations
	CreateSession(ctx context.Context, s *schema.Session) error
	FindSessionByID(ctx context.Context, id xid.ID) (*schema.Session, error)
	FindSessionByToken(ctx context.Context, token string) (*schema.Session, error)

	// List with pagination
	ListSessions(ctx context.Context, filter *ListSessionsFilter) (*pagination.PageResponse[*schema.Session], error)

	// Update/Delete operations
	RevokeSession(ctx context.Context, token string) error
	RevokeSessionByID(ctx context.Context, id xid.ID) error

	// Count operations
	CountSessions(ctx context.Context, appID xid.ID, userID *xid.ID) (int, error)

	// Maintenance
	CleanupExpiredSessions(ctx context.Context) (int, error)
}
