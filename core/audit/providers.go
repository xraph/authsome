package audit

import (
	"context"
	"time"
)

// =============================================================================
// PROVIDER INTERFACES - Generic abstractions for external systems
// =============================================================================

// UserProvider provides access to user data from any source (authsome, LDAP, custom DB)
// This breaks the tight coupling to authsome's internal user service
type UserProvider interface {
	// GetUser retrieves a single user by ID
	GetUser(ctx context.Context, scope *Scope, userID string) (*GenericUser, error)

	// ListUsers retrieves users matching filter criteria
	ListUsers(ctx context.Context, scope *Scope, filter *UserFilter) ([]*GenericUser, error)

	// QueryUserMetrics retrieves aggregated metrics about users
	QueryUserMetrics(ctx context.Context, scope *Scope, query *MetricsQuery) (*UserMetrics, error)
}

// OrganizationProvider provides access to organization/tenant data
type OrganizationProvider interface {
	// GetOrganization retrieves organization details
	GetOrganization(ctx context.Context, orgID string) (*GenericOrganization, error)

	// ListOrganizations retrieves organizations matching filter
	ListOrganizations(ctx context.Context, filter *OrgFilter) ([]*GenericOrganization, error)
}

// AuditProvider allows external systems to receive audit events
// Useful for forwarding to SIEM, analytics engines, etc.
type AuditProvider interface {
	// OnEvent is called when an audit event is created
	OnEvent(ctx context.Context, event *Event) error

	// Healthy checks if the provider is operational
	Healthy(ctx context.Context) error
}

// =============================================================================
// SCOPE - Hierarchical scoping for compliance profiles
// =============================================================================

// ScopeType defines the level of compliance scope
type ScopeType string

const (
	ScopeTypeSystem ScopeType = "system" // Global defaults
	ScopeTypeApp    ScopeType = "app"    // Customer/tenant level
	ScopeTypeOrg    ScopeType = "org"    // User-created workspace
	ScopeTypeTeam   ScopeType = "team"   // Department/team
	ScopeTypeRole   ScopeType = "role"   // Role-based (admin, user)
	ScopeTypeUser   ScopeType = "user"   // Individual user overrides
)

// Scope represents a hierarchical compliance scope
type Scope struct {
	Type     ScopeType `json:"type"`
	ID       string    `json:"id"`
	ParentID *string   `json:"parentId,omitempty"` // For inheritance
}

// =============================================================================
// GENERIC USER - Provider-agnostic user representation
// =============================================================================

// GenericUser represents a user from any system
type GenericUser struct {
	ID               string                 `json:"id"`
	Email            string                 `json:"email"`
	Username         string                 `json:"username,omitempty"`
	DisplayName      string                 `json:"displayName,omitempty"`
	MFAEnabled       bool                   `json:"mfaEnabled"`
	MFAMethods       []string               `json:"mfaMethods"`         // ["totp", "sms", "webauthn"]
	PasswordChanged  time.Time              `json:"passwordChanged"`
	LastLogin        time.Time              `json:"lastLogin"`
	LoginCount       int                    `json:"loginCount"`
	FailedLoginCount int                    `json:"failedLoginCount"`
	Status           string                 `json:"status"`             // "active", "suspended", "deleted", "locked"
	Roles            []string               `json:"roles"`
	Groups           []string               `json:"groups,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"` // Extensible
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// UserFilter defines criteria for filtering users
type UserFilter struct {
	IDs               []string   `json:"ids,omitempty"`
	Emails            []string   `json:"emails,omitempty"`
	Status            *string    `json:"status,omitempty"`
	MFAEnabled        *bool      `json:"mfaEnabled,omitempty"`
	Roles             []string   `json:"roles,omitempty"`
	CreatedAfter      *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore     *time.Time `json:"createdBefore,omitempty"`
	LastLoginAfter    *time.Time `json:"lastLoginAfter,omitempty"`
	LastLoginBefore   *time.Time `json:"lastLoginBefore,omitempty"`
	PasswordExpired   *bool      `json:"passwordExpired,omitempty"`
	PasswordExpiryAge *int       `json:"passwordExpiryAge,omitempty"` // Days
	Limit             int        `json:"limit"`
	Offset            int        `json:"offset"`
}

// MetricsQuery defines aggregated metrics to retrieve
type MetricsQuery struct {
	Metrics      []string   `json:"metrics"` // ["total_users", "mfa_adoption", "inactive_users"]
	GroupBy      []string   `json:"groupBy,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	EndDate      *time.Time `json:"endDate,omitempty"`
	Granularity  string     `json:"granularity,omitempty"` // "day", "week", "month"
}

// UserMetrics contains aggregated user metrics
type UserMetrics struct {
	TotalUsers         int                       `json:"totalUsers"`
	ActiveUsers        int                       `json:"activeUsers"`
	InactiveUsers      int                       `json:"inactiveUsers"`
	MFAAdoptionRate    float64                   `json:"mfaAdoptionRate"`    // 0-100
	UsersWithMFA       int                       `json:"usersWithMFA"`
	UsersWithoutMFA    int                       `json:"usersWithoutMFA"`
	ExpiredPasswords   int                       `json:"expiredPasswords"`
	LockedAccounts     int                       `json:"lockedAccounts"`
	SuspendedAccounts  int                       `json:"suspendedAccounts"`
	ByRole             map[string]int            `json:"byRole,omitempty"`
	ByStatus           map[string]int            `json:"byStatus,omitempty"`
	CustomMetrics      map[string]interface{}    `json:"customMetrics,omitempty"`
}

// =============================================================================
// GENERIC ORGANIZATION
// =============================================================================

// GenericOrganization represents an organization/tenant from any system
type GenericOrganization struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName,omitempty"`
	Status      string                 `json:"status"` // "active", "suspended", "deleted"
	ParentID    *string                `json:"parentId,omitempty"`
	Members     int                    `json:"members"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// OrgFilter defines criteria for filtering organizations
type OrgFilter struct {
	IDs      []string  `json:"ids,omitempty"`
	Status   *string   `json:"status,omitempty"`
	ParentID *string   `json:"parentId,omitempty"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

// =============================================================================
// PROVIDER REGISTRY - Manages provider instances
// =============================================================================

// ProviderRegistry manages all provider instances
type ProviderRegistry struct {
	userProvider UserProvider
	orgProvider  OrganizationProvider
	auditProviders []AuditProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		auditProviders: make([]AuditProvider, 0),
	}
}

// SetUserProvider registers a user provider
func (r *ProviderRegistry) SetUserProvider(provider UserProvider) {
	r.userProvider = provider
}

// GetUserProvider returns the registered user provider
func (r *ProviderRegistry) GetUserProvider() UserProvider {
	return r.userProvider
}

// SetOrgProvider registers an organization provider
func (r *ProviderRegistry) SetOrgProvider(provider OrganizationProvider) {
	r.orgProvider = provider
}

// GetOrgProvider returns the registered organization provider
func (r *ProviderRegistry) GetOrgProvider() OrganizationProvider {
	return r.orgProvider
}

// AddAuditProvider registers an audit event consumer
func (r *ProviderRegistry) AddAuditProvider(provider AuditProvider) {
	r.auditProviders = append(r.auditProviders, provider)
}

// GetAuditProviders returns all registered audit providers
func (r *ProviderRegistry) GetAuditProviders() []AuditProvider {
	return r.auditProviders
}

// NotifyAuditEvent sends event to all audit providers (non-blocking)
func (r *ProviderRegistry) NotifyAuditEvent(ctx context.Context, event *Event) {
	for _, provider := range r.auditProviders {
		go func(p AuditProvider) {
			// Non-blocking notification
			// Errors are logged but don't block audit creation
			if err := p.OnEvent(ctx, event); err != nil {
				// TODO: Add structured logging
				_ = err
			}
		}(provider)
	}
}

// =============================================================================
// PROVIDER OPTIONS - Functional options pattern for service configuration
// =============================================================================

// ServiceOption is a functional option for configuring the audit service
type ServiceOption func(*ServiceConfig)

// ServiceConfig holds configuration for the audit service
type ServiceConfig struct {
	Providers *ProviderRegistry
}

// WithProviders configures the service to use custom providers
func WithProviders(providers *ProviderRegistry) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Providers = providers
	}
}

// WithUserProvider configures a user provider
func WithUserProvider(provider UserProvider) ServiceOption {
	return func(cfg *ServiceConfig) {
		if cfg.Providers == nil {
			cfg.Providers = NewProviderRegistry()
		}
		cfg.Providers.SetUserProvider(provider)
	}
}

// WithOrgProvider configures an organization provider
func WithOrgProvider(provider OrganizationProvider) ServiceOption {
	return func(cfg *ServiceConfig) {
		if cfg.Providers == nil {
			cfg.Providers = NewProviderRegistry()
		}
		cfg.Providers.SetOrgProvider(provider)
	}
}

// WithAuditProvider adds an audit event consumer
func WithAuditProvider(provider AuditProvider) ServiceOption {
	return func(cfg *ServiceConfig) {
		if cfg.Providers == nil {
			cfg.Providers = NewProviderRegistry()
		}
		cfg.Providers.AddAuditProvider(provider)
	}
}

