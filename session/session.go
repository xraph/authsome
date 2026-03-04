// Package session defines the session domain entity and its store interface.
package session

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Session represents an authenticated user session.
type Session struct {
	ID                    id.SessionID     `json:"id"`
	AppID                 id.AppID         `json:"app_id"`
	EnvID                 id.EnvironmentID `json:"env_id"`
	UserID                id.UserID        `json:"user_id"`
	OrgID                 id.OrgID         `json:"org_id,omitempty"`
	Token                 string           `json:"-"`
	RefreshToken          string           `json:"-"`
	IPAddress             string           `json:"ip_address,omitempty"`
	UserAgent             string           `json:"user_agent,omitempty"`
	DeviceID              id.DeviceID      `json:"device_id,omitempty"`
	ImpersonatedBy        id.UserID        `json:"impersonated_by,omitempty"`
	ExpiresAt             time.Time        `json:"expires_at"`
	RefreshTokenExpiresAt time.Time        `json:"refresh_token_expires_at"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
}
