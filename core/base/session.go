package base

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Session represents a user session (DTO)
type Session struct {
	ID             xid.ID    `json:"id"`
	Token          string    `json:"token"`
	AppID          xid.ID    `json:"appID"`
	EnvironmentID  *xid.ID   `json:"environmentID,omitempty"`
	OrganizationID *xid.ID   `json:"organizationID,omitempty"`
	UserID         xid.ID    `json:"userId"`
	ExpiresAt      time.Time `json:"expiresAt"`
	IPAddress      string    `json:"ipAddress"`
	UserAgent      string    `json:"userAgent"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ToSchema converts Session DTO to schema.Session
func (s *Session) ToSchema() *schema.Session {
	return &schema.Session{
		ID:             s.ID,
		Token:          s.Token,
		AppID:          s.AppID,
		EnvironmentID:  s.EnvironmentID,
		OrganizationID: s.OrganizationID,
		UserID:         s.UserID,
		ExpiresAt:      s.ExpiresAt,
		IPAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
		AuditableModel: schema.AuditableModel{
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			CreatedBy: s.UserID,
			UpdatedBy: s.UserID,
		},
	}
}
