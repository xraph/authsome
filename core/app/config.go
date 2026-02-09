package app

import "github.com/rs/xid"

// Config represents app service configuration.
type Config struct {
	// PlatformAppID is the ID of the platform app (super admin)
	PlatformAppID xid.ID `json:"platformAppId"`

	// DefaultAppName is the name of the default app in standalone mode
	DefaultAppName string `json:"defaultAppName"`

	// EnableAppCreation allows users to create new apps (multitenancy mode)
	EnableAppCreation bool `json:"enableAppCreation"`

	// MaxMembersPerApp limits the number of members per app
	MaxMembersPerApp int `json:"maxMembersPerApp"`

	// MaxTeamsPerApp limits the number of teams per app
	MaxTeamsPerApp int `json:"maxTeamsPerApp"`

	// RequireInvitation requires invitation for joining apps
	RequireInvitation bool `json:"requireInvitation"`

	// InvitationExpiryHours sets how long invitations are valid
	InvitationExpiryHours int `json:"invitationExpiryHours"`

	// AutoCreateDefaultApp auto-creates default app on server start
	AutoCreateDefaultApp bool `json:"autoCreateDefaultApp"`

	// DefaultEnvironmentName is the name of the default dev environment
	DefaultEnvironmentName string `json:"defaultEnvironmentName"`
}
