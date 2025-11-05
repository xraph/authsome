package schema

import (
	"time"

	"github.com/uptrace/bun"
)

// UserBan represents a user ban record in the database
type UserBan struct {
	bun.BaseModel `bun:"table:user_bans,alias:ub"`

	// Primary key
	ID string `bun:"id,pk" json:"id"`

	// Foreign keys
	UserID       string `bun:"user_id,notnull" json:"user_id"`
	BannedByID   string `bun:"banned_by_id,notnull" json:"banned_by_id"`
	UnbannedByID string `bun:"unbanned_by_id" json:"unbanned_by_id,omitempty"`

	// Ban details
	Reason    string     `bun:"reason,notnull" json:"reason"`
	IsActive  bool       `bun:"is_active,notnull,default:true" json:"is_active"`
	ExpiresAt *time.Time `bun:"expires_at" json:"expires_at,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	UnbannedAt *time.Time `bun:"unbanned_at" json:"unbanned_at,omitempty"`

	// Relations
	User       *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	BannedBy   *User `bun:"rel:belongs-to,join:banned_by_id=id" json:"banned_by,omitempty"`
	UnbannedBy *User `bun:"rel:belongs-to,join:unbanned_by_id=id" json:"unbanned_by,omitempty"`
}

// TableName returns the table name for the UserBan model
func (UserBan) TableName() string {
	return "user_bans"
}

// IsExpired checks if the ban has expired
func (ub *UserBan) IsExpired() bool {
	if ub.ExpiresAt == nil {
		return false // Permanent ban
	}
	return time.Now().After(*ub.ExpiresAt)
}

// IsCurrentlyActive checks if the ban is currently active
func (ub *UserBan) IsCurrentlyActive() bool {
	return ub.IsActive && !ub.IsExpired()
}
