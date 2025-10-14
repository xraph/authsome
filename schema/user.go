package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// User represents the user table
type User struct {
    AuditableModel `bun:",inline"`
    bun.BaseModel  `bun:"table:users,alias:u"`

    ID              xid.ID     `bun:"id,pk,type:varchar(20)"`
    Email           string     `bun:"email,notnull,unique"`
    EmailVerified   bool       `bun:"email_verified,notnull,default:false"`
    EmailVerifiedAt *time.Time `bun:"email_verified_at"`
    Name            string     `bun:"name"`
    Image           string     `bun:"image"`
    PasswordHash    string     `bun:"password_hash"`

    // Username support (Phase 6)
    Username        string     `bun:"username,unique"`
    DisplayUsername string     `bun:"display_username"`

    // Soft delete
    DeletedAt *time.Time `bun:"deleted_at"`
}
