// Package device defines the device tracking domain entity and its store interface.
package device

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Device represents a tracked user device.
type Device struct {
	ID          id.DeviceID      `json:"id"`
	UserID      id.UserID        `json:"user_id"`
	AppID       id.AppID         `json:"app_id"`
	EnvID       id.EnvironmentID `json:"env_id"`
	Name        string           `json:"name,omitempty"`
	Type        string           `json:"type,omitempty"`
	Browser     string           `json:"browser,omitempty"`
	OS          string           `json:"os,omitempty"`
	IPAddress   string           `json:"ip_address,omitempty"`
	Fingerprint string           `json:"fingerprint,omitempty"`
	Trusted     bool             `json:"trusted"`
	LastSeenAt  time.Time        `json:"last_seen_at"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
