package scim

import (
	"time"

	"github.com/xraph/authsome/internal/errs"
)

// Config holds the SCIM plugin configuration.
type Config struct {
	// Service configuration
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Authentication
	AuthMethod  string        `json:"auth_method"  yaml:"auth_method"` // "bearer" or "oauth2"
	TokenExpiry time.Duration `json:"token_expiry" yaml:"token_expiry"`

	// Rate limiting
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit"`

	// User provisioning
	UserProvisioning UserProvisioningConfig `json:"user_provisioning" yaml:"user_provisioning"`

	// Group synchronization
	GroupSync GroupSyncConfig `json:"group_sync" yaml:"group_sync"`

	// Attribute mapping
	AttributeMapping AttributeMappingConfig `json:"attribute_mapping" yaml:"attribute_mapping"`

	// JIT provisioning
	JITProvisioning JITProvisioningConfig `json:"jit_provisioning" yaml:"jit_provisioning"`

	// Webhooks
	Webhooks WebhookConfig `json:"webhooks" yaml:"webhooks"`

	// Bulk operations
	BulkOperations BulkOperationsConfig `json:"bulk_operations" yaml:"bulk_operations"`

	// Filtering and search
	Search SearchConfig `json:"search" yaml:"search"`

	// Compliance and security
	Security SecurityConfig `json:"security" yaml:"security"`
}

// RateLimitConfig configures rate limiting for SCIM endpoints.
type RateLimitConfig struct {
	Enabled        bool `json:"enabled"          yaml:"enabled"`
	RequestsPerMin int  `json:"requests_per_min" yaml:"requests_per_min"`
	BurstSize      int  `json:"burst_size"       yaml:"burst_size"`
}

// UserProvisioningConfig configures user provisioning behavior.
type UserProvisioningConfig struct {
	Enabled                 bool     `json:"enabled"                    yaml:"enabled"`
	AutoActivate            bool     `json:"auto_activate"              yaml:"auto_activate"`      // Activate users immediately
	SendWelcomeEmail        bool     `json:"send_welcome_email"         yaml:"send_welcome_email"` // Send welcome email on creation
	DefaultRole             string   `json:"default_role"               yaml:"default_role"`       // Default role for provisioned users
	RequiredAttributes      []string `json:"required_attributes"        yaml:"required_attributes"`
	PreventDuplicates       bool     `json:"prevent_duplicates"         yaml:"prevent_duplicates"`         // Prevent duplicate emails
	SoftDeleteOnDeProvision bool     `json:"soft_delete_on_deprovision" yaml:"soft_delete_on_deprovision"` // Soft delete vs hard delete
}

// GroupSyncConfig configures group synchronization with teams/roles.
type GroupSyncConfig struct {
	Enabled             bool              `json:"enabled"               yaml:"enabled"`
	SyncToTeams         bool              `json:"sync_to_teams"         yaml:"sync_to_teams"`         // Sync SCIM groups to teams
	SyncToRoles         bool              `json:"sync_to_roles"         yaml:"sync_to_roles"`         // Sync SCIM groups to roles
	GroupMapping        map[string]string `json:"group_mapping"         yaml:"group_mapping"`         // Map SCIM group ID to team/role ID
	CreateMissingGroups bool              `json:"create_missing_groups" yaml:"create_missing_groups"` // Auto-create teams/roles
	DeleteEmptyGroups   bool              `json:"delete_empty_groups"   yaml:"delete_empty_groups"`   // Delete teams/roles with no members
}

// AttributeMappingConfig configures custom attribute mapping.
type AttributeMappingConfig struct {
	Enabled       bool              `json:"enabled"        yaml:"enabled"`
	CustomMapping map[string]string `json:"custom_mapping" yaml:"custom_mapping"` // Map SCIM attribute to AuthSome field

	// Standard SCIM User schema mappings (RFC 7643)
	UserNameField    string `json:"username_field"     yaml:"username_field"`     // Default: "userName"
	EmailField       string `json:"email_field"        yaml:"email_field"`        // Default: "emails[0].value"
	GivenNameField   string `json:"given_name_field"   yaml:"given_name_field"`   // Default: "name.givenName"
	FamilyNameField  string `json:"family_name_field"  yaml:"family_name_field"`  // Default: "name.familyName"
	DisplayNameField string `json:"display_name_field" yaml:"display_name_field"` // Default: "displayName"
	ActiveField      string `json:"active_field"       yaml:"active_field"`       // Default: "active"

	// Enterprise schema extension (urn:ietf:params:scim:schemas:extension:enterprise:2.0:User)
	EmployeeNumberField string `json:"employee_number_field" yaml:"employee_number_field"` // Default: "employeeNumber"
	DepartmentField     string `json:"department_field"      yaml:"department_field"`      // Default: "department"
	ManagerField        string `json:"manager_field"         yaml:"manager_field"`         // Default: "manager.value"
}

// JITProvisioningConfig configures Just-In-Time provisioning.
type JITProvisioningConfig struct {
	Enabled            bool     `json:"enabled"               yaml:"enabled"`
	CreateOnFirstLogin bool     `json:"create_on_first_login" yaml:"create_on_first_login"` // Create user on first SSO login
	UpdateOnLogin      bool     `json:"update_on_login"       yaml:"update_on_login"`       // Update user attributes on each login
	RequiredAttributes []string `json:"required_attributes"   yaml:"required_attributes"`
}

// WebhookConfig configures provisioning event webhooks.
type WebhookConfig struct {
	Enabled           bool     `json:"enabled"              yaml:"enabled"`
	NotifyOnCreate    bool     `json:"notify_on_create"     yaml:"notify_on_create"`
	NotifyOnUpdate    bool     `json:"notify_on_update"     yaml:"notify_on_update"`
	NotifyOnDelete    bool     `json:"notify_on_delete"     yaml:"notify_on_delete"`
	NotifyOnGroupSync bool     `json:"notify_on_group_sync" yaml:"notify_on_group_sync"`
	WebhookURLs       []string `json:"webhook_urls"         yaml:"webhook_urls"`
	RetryAttempts     int      `json:"retry_attempts"       yaml:"retry_attempts"`
	TimeoutSeconds    int      `json:"timeout_seconds"      yaml:"timeout_seconds"`
}

// BulkOperationsConfig configures bulk operation limits.
type BulkOperationsConfig struct {
	Enabled         bool `json:"enabled"           yaml:"enabled"`
	MaxOperations   int  `json:"max_operations"    yaml:"max_operations"`    // Max operations per bulk request
	MaxPayloadBytes int  `json:"max_payload_bytes" yaml:"max_payload_bytes"` // Max payload size in bytes
}

// SearchConfig configures search and filtering behavior.
type SearchConfig struct {
	MaxResults     int      `json:"max_results"     yaml:"max_results"`     // Max results per page
	DefaultResults int      `json:"default_results" yaml:"default_results"` // Default page size
	AllowedFilters []string `json:"allowed_filters" yaml:"allowed_filters"` // Allowed filter attributes
	AllowedSortBy  []string `json:"allowed_sort_by" yaml:"allowed_sort_by"` // Allowed sort attributes
}

// SecurityConfig configures security and compliance features.
type SecurityConfig struct {
	RequireHTTPS         bool     `json:"require_https"          yaml:"require_https"`
	IPWhitelist          []string `json:"ip_whitelist"           yaml:"ip_whitelist"`
	AuditAllOperations   bool     `json:"audit_all_operations"   yaml:"audit_all_operations"`
	MaskSensitiveData    bool     `json:"mask_sensitive_data"    yaml:"mask_sensitive_data"`    // Mask emails, phones in logs
	RequireOrgValidation bool     `json:"require_org_validation" yaml:"require_org_validation"` // Validate org access
}

// DefaultConfig returns the default SCIM configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		AuthMethod:  "bearer",
		TokenExpiry: 90 * 24 * time.Hour, // 90 days

		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 600, // 10 req/sec
			BurstSize:      100,
		},

		UserProvisioning: UserProvisioningConfig{
			Enabled:                 true,
			AutoActivate:            true,
			SendWelcomeEmail:        false, // Let IdP handle welcome emails
			DefaultRole:             "member",
			RequiredAttributes:      []string{"userName", "emails"},
			PreventDuplicates:       true,
			SoftDeleteOnDeProvision: true, // Soft delete by default
		},

		GroupSync: GroupSyncConfig{
			Enabled:             true,
			SyncToTeams:         true,
			SyncToRoles:         false, // Roles managed separately
			GroupMapping:        make(map[string]string),
			CreateMissingGroups: true,
			DeleteEmptyGroups:   false, // Keep empty groups
		},

		AttributeMapping: AttributeMappingConfig{
			Enabled:             true,
			CustomMapping:       make(map[string]string),
			UserNameField:       "userName",
			EmailField:          "emails[0].value",
			GivenNameField:      "name.givenName",
			FamilyNameField:     "name.familyName",
			DisplayNameField:    "displayName",
			ActiveField:         "active",
			EmployeeNumberField: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber",
			DepartmentField:     "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
			ManagerField:        "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager.value",
		},

		JITProvisioning: JITProvisioningConfig{
			Enabled:            false, // Disabled by default
			CreateOnFirstLogin: true,
			UpdateOnLogin:      true,
			RequiredAttributes: []string{"userName", "emails"},
		},

		Webhooks: WebhookConfig{
			Enabled:           false,
			NotifyOnCreate:    true,
			NotifyOnUpdate:    true,
			NotifyOnDelete:    true,
			NotifyOnGroupSync: true,
			WebhookURLs:       []string{},
			RetryAttempts:     3,
			TimeoutSeconds:    10,
		},

		BulkOperations: BulkOperationsConfig{
			Enabled:         true,
			MaxOperations:   100,         // 100 operations per request
			MaxPayloadBytes: 1024 * 1024, // 1MB
		},

		Search: SearchConfig{
			MaxResults:     1000,
			DefaultResults: 50,
			AllowedFilters: []string{"userName", "emails", "displayName", "active", "externalId"},
			AllowedSortBy:  []string{"userName", "displayName", "meta.created", "meta.lastModified"},
		},

		Security: SecurityConfig{
			RequireHTTPS:         true,
			IPWhitelist:          []string{}, // Empty = allow all
			AuditAllOperations:   true,
			MaskSensitiveData:    true,
			RequireOrgValidation: true,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if disabled
	}

	if c.AuthMethod != "bearer" && c.AuthMethod != "oauth2" {
		return errs.BadRequest("invalid auth_method: must be 'bearer' or 'oauth2'")
	}

	if c.TokenExpiry <= 0 {
		return errs.BadRequest("token_expiry must be positive")
	}

	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerMin <= 0 {
			return errs.BadRequest("rate_limit.requests_per_min must be positive")
		}

		if c.RateLimit.BurstSize <= 0 {
			return errs.BadRequest("rate_limit.burst_size must be positive")
		}
	}

	if c.BulkOperations.Enabled {
		if c.BulkOperations.MaxOperations <= 0 || c.BulkOperations.MaxOperations > 1000 {
			return errs.BadRequest("bulk_operations.max_operations must be between 1 and 1000")
		}

		if c.BulkOperations.MaxPayloadBytes <= 0 {
			return errs.BadRequest("bulk_operations.max_payload_bytes must be positive")
		}
	}

	if c.Search.MaxResults <= 0 || c.Search.MaxResults > 10000 {
		return errs.BadRequest("search.max_results must be between 1 and 10000")
	}

	if c.Search.DefaultResults <= 0 || c.Search.DefaultResults > c.Search.MaxResults {
		return errs.BadRequest("search.default_results must be between 1 and max_results")
	}

	if c.Webhooks.Enabled {
		if len(c.Webhooks.WebhookURLs) == 0 {
			return errs.BadRequest("webhooks.webhook_urls cannot be empty when webhooks are enabled")
		}

		if c.Webhooks.RetryAttempts < 0 || c.Webhooks.RetryAttempts > 10 {
			return errs.BadRequest("webhooks.retry_attempts must be between 0 and 10")
		}

		if c.Webhooks.TimeoutSeconds <= 0 || c.Webhooks.TimeoutSeconds > 300 {
			return errs.BadRequest("webhooks.timeout_seconds must be between 1 and 300")
		}
	}

	return nil
}
