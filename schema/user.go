package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// User represents the user table
// In the new architecture:
// - Users are app-scoped (can exist in multiple apps with different IDs)
// - Same email can exist across different apps
// - Unique constraint is on (app_id, email) combination
// - User membership to apps is managed via the Member table in app service.
type User struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:users,alias:u"`

	ID              xid.ID     `bun:"id,pk,type:varchar(20)"`
	AppID           *xid.ID    `bun:"app_id,type:varchar(20),notnull"` // App association (required in new architecture)
	Email           string     `bun:"email,notnull"`                   // Unique per app, not globally
	EmailVerified   bool       `bun:"email_verified,notnull,default:false"`
	EmailVerifiedAt *time.Time `bun:"email_verified_at"`
	Name            string     `bun:"name"`
	Image           string     `bun:"image"`
	PasswordHash    string     `bun:"password_hash"`

	// Username support (Phase 6)
	Username        string `bun:"username,unique"`
	DisplayUsername string `bun:"display_username"`

	// Soft delete
	DeletedAt *time.Time `bun:"deleted_at"`
}

// Note: Database migration should add:
// - Index on app_id for performance
// - Composite unique index on (app_id, email) to replace single email unique constraint
// - Foreign key constraint on app_id referencing apps(id)
