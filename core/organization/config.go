package organization

// Config holds the organization service configuration
type Config struct {
	MaxOrganizationsPerUser   int  `json:"maxOrganizationsPerUser"`
	MaxMembersPerOrganization int  `json:"maxMembersPerOrganization"`
	MaxTeamsPerOrganization   int  `json:"maxTeamsPerOrganization"`
	EnableUserCreation        bool `json:"enableUserCreation"`
	RequireInvitation         bool `json:"requireInvitation"`
	InvitationExpiryHours     int  `json:"invitationExpiryHours"`
}

// DefaultConfig returns sensible default configuration values
func DefaultConfig() Config {
	return Config{
		MaxOrganizationsPerUser:   5,
		MaxMembersPerOrganization: 50,
		MaxTeamsPerOrganization:   20,
		EnableUserCreation:        true,
		RequireInvitation:         false,
		InvitationExpiryHours:     72, // 3 days
	}
}
