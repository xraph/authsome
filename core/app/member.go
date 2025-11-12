package app

import (
	"time"

	"github.com/rs/xid"
)

// Member represents an app member
type Member struct {
	ID        xid.ID    `json:"id"`
	AppID     xid.ID    `json:"appId"`
	UserID    xid.ID    `json:"userId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// InviteMemberRequest represents an invite member request
type InviteMemberRequest struct {
	AppID     xid.ID `json:"appId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	InviterID xid.ID `json:"inviterId"`
}
