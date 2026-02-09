package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// MemberRole represents the role of a member in an app.
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// IsValid checks if the role is valid.
func (r MemberRole) IsValid() bool {
	switch r {
	case MemberRoleOwner, MemberRoleAdmin, MemberRoleMember:
		return true
	}

	return false
}

// String returns the string representation.
func (r MemberRole) String() string {
	return string(r)
}

// MemberStatus represents the status of a member.
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "active"
	MemberStatusSuspended MemberStatus = "suspended"
	MemberStatusPending   MemberStatus = "pending"
)

// IsValid checks if the status is valid.
func (s MemberStatus) IsValid() bool {
	switch s {
	case MemberStatusActive, MemberStatusSuspended, MemberStatusPending:
		return true
	}

	return false
}

// String returns the string representation.
func (s MemberStatus) String() string {
	return string(s)
}

// Member represents the app member table.
type Member struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:members,alias:m"`

	ID       xid.ID       `bun:"id,pk,type:varchar(20)"                               json:"id"`
	AppID    xid.ID       `bun:"app_id,notnull,type:varchar(20)"                      json:"appID"` // App context
	UserID   xid.ID       `bun:"user_id,notnull,type:varchar(20)"                     json:"userID"`
	Role     MemberRole   `bun:"role,nullzero"                                        json:"role"`
	Status   MemberStatus `bun:"status,notnull,default:'active'"                      json:"status"`
	JoinedAt time.Time    `bun:"joined_at,nullzero,notnull,default:current_timestamp" json:"joinedAt"` // when the member joined

	// Relations
	App   *App   `bun:"rel:belongs-to,join:app_id=id"     json:"app,omitempty"`
	User  *User  `bun:"rel:belongs-to,join:user_id=id"    json:"user,omitempty"`
	Teams []Team `bun:"m2m:team_members,join:Member=Team" json:"teams,omitempty"`
}
