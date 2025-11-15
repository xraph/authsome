package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// UserBan represents a user ban record in the database
type UserBan struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:user_bans,alias:ub"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// App context
	AppID xid.ID `bun:"app_id,notnull,type:varchar(20)" json:"appID"`

	// Foreign keys
	UserID       xid.ID  `bun:"user_id,notnull,type:varchar(20)" json:"userID"`
	BannedByID   xid.ID  `bun:"banned_by_id,notnull,type:varchar(20)" json:"bannedByID"`
	UnbannedByID *xid.ID `bun:"unbanned_by_id,type:varchar(20)" json:"unbannedByID,omitempty"`

	// Ban details
	Reason    string     `bun:"reason,notnull" json:"reason"`
	IsActive  bool       `bun:"is_active,notnull,default:true" json:"isActive"`
	ExpiresAt *time.Time `bun:"expires_at" json:"expiresAt,omitempty"`

	// Timestamps
	UnbannedAt *time.Time `bun:"unbanned_at" json:"unbannedAt,omitempty"`

	// Relations
	App        *App  `bun:"rel:belongs-to,join:app_id=id"`
	User       *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	BannedBy   *User `bun:"rel:belongs-to,join:banned_by_id=id" json:"bannedBy,omitempty"`
	UnbannedBy *User `bun:"rel:belongs-to,join:unbanned_by_id=id" json:"unbannedBy,omitempty"`
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
