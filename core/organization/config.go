package organization

// Config holds the organization service configuration
type Config struct {
	MaxOrganizationsPerUser   int  `json:"maxOrganizationsPerUser"`
	MaxMembersPerOrganization int  `json:"maxMembersPerOrganization"`
	MaxTeamsPerOrganization   int  `json:"maxTeamsPerOrganization"`
	EnableUserCreation        bool `json:"enableUserCreation"`
	RequireInvitation         bool `json:"requireInvitation"`
	InvitationExpiryHours     int  `json:"invitationExpiryHours"`
	EnforceUniqueSlug         bool `json:"enforceUniqueSlug"`     // Enforce unique slugs within app+environment scope
	AllowAppLevelRoles        bool `json:"allowAppLevelRoles"`    // Allow app-level (global) RBAC roles for org membership
}

// DefaultConfig returns sensible default configuration values
func DefaultConfig() Config {
	return Config{
		MaxOrganizationsPerUser:   5,
		MaxMembersPerOrganization: 50,
		MaxTeamsPerOrganization:   20,
		EnableUserCreation:        true,
		RequireInvitation:         false,
		InvitationExpiryHours:     72,   // 3 days
		EnforceUniqueSlug:         true, // Enforce unique slugs within app+environment by default
		AllowAppLevelRoles:        true, // Allow app-level roles by default for backward compatibility
	}
}
