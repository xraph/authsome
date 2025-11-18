package session

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/base"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Session represents a user session (DTO)
type Session = base.Session

// FromSchemaSession converts schema.Session to Session DTO
func FromSchemaSession(s *schema.Session) *Session {
	if s == nil {
		return nil
	}
	return &Session{
		ID:             s.ID,
		Token:          s.Token,
		AppID:          s.AppID,
		EnvironmentID:  s.EnvironmentID,
		OrganizationID: s.OrganizationID,
		UserID:         s.UserID,
		ExpiresAt:      s.ExpiresAt,
		IPAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// FromSchemaSessions converts multiple schema.Session to Session DTOs
func FromSchemaSessions(sessions []*schema.Session) []*Session {
	result := make([]*Session, len(sessions))
	for i, s := range sessions {
		result[i] = FromSchemaSession(s)
	}
	return result
}

// ListSessionsResponse is a type alias for paginated response
type ListSessionsResponse = pagination.PageResponse[*Session]

// CreateSessionRequest represents the data to create a session
type CreateSessionRequest struct {
	AppID          xid.ID  `json:"appID"`
	EnvironmentID  *xid.ID `json:"environmentID,omitempty"`
	OrganizationID *xid.ID `json:"organizationID,omitempty"`
	UserID         xid.ID  `json:"userId"`
	IPAddress      string  `json:"ipAddress"`
	UserAgent      string  `json:"userAgent"`
	Remember       bool    `json:"remember"`
}
