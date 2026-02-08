package scim

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SCIM 2.0 Schema URNs (RFC 7643)
const (
	SchemaCore            = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaEnterprise      = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	SchemaGroup           = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SchemaServiceProvider = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	SchemaResourceType    = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	SchemaSchema          = "urn:ietf:params:scim:schemas:core:2.0:Schema"
	SchemaListResponse    = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SchemaError           = "urn:ietf:params:scim:api:messages:2.0:Error"
	SchemaBulkRequest     = "urn:ietf:params:scim:api:messages:2.0:BulkRequest"
	SchemaBulkResponse    = "urn:ietf:params:scim:api:messages:2.0:BulkResponse"
	SchemaPatchOp         = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// SCIMUser represents a SCIM 2.0 User resource (RFC 7643 Section 4.1)
type SCIMUser struct {
	// Common attributes
	Schemas    []string  `json:"schemas"`
	ID         string    `json:"id"`
	ExternalID string    `json:"externalId,omitempty"`
	Meta       *SCIMMeta `json:"meta"`

	// Core User Schema attributes
	UserName          string    `json:"userName"`
	Name              *SCIMName `json:"name,omitempty"`
	DisplayName       string    `json:"displayName,omitempty"`
	NickName          string    `json:"nickName,omitempty"`
	ProfileURL        string    `json:"profileUrl,omitempty"`
	Title             string    `json:"title,omitempty"`
	UserType          string    `json:"userType,omitempty"`
	PreferredLanguage string    `json:"preferredLanguage,omitempty"`
	Locale            string    `json:"locale,omitempty"`
	Timezone          string    `json:"timezone,omitempty"`
	Active            bool      `json:"active"`
	Password          string    `json:"password,omitempty"`

	// Multi-valued attributes
	Emails           []Email           `json:"emails,omitempty"`
	PhoneNumbers     []PhoneNumber     `json:"phoneNumbers,omitempty"`
	IMs              []IM              `json:"ims,omitempty"`
	Photos           []Photo           `json:"photos,omitempty"`
	Addresses        []SCIMAddress     `json:"addresses,omitempty"`
	Groups           []GroupReference  `json:"groups,omitempty"`
	Entitlements     []Entitlement     `json:"entitlements,omitempty"`
	Roles            []SCIMRole        `json:"roles,omitempty"`
	X509Certificates []X509Certificate `json:"x509Certificates,omitempty"`

	// Enterprise extension
	EnterpriseUser *EnterpriseUser `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
}

// SCIMGroup represents a SCIM 2.0 Group resource (RFC 7643 Section 4.2)
type SCIMGroup struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	ExternalID  string            `json:"externalId,omitempty"`
	Meta        *SCIMMeta         `json:"meta"`
	DisplayName string            `json:"displayName"`
	Members     []MemberReference `json:"members,omitempty"`
}

// SCIMMeta contains resource metadata (RFC 7643 Section 3.1)
type SCIMMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Location     string    `json:"location"`
	Version      string    `json:"version,omitempty"`
}

// SCIMName represents a user's name (RFC 7643 Section 4.1.1)
type SCIMName struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

// Email represents an email address (RFC 7643 Section 4.1.2)
type Email struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // work, home, other
	Primary bool   `json:"primary,omitempty"`
}

// PhoneNumber represents a phone number (RFC 7643 Section 4.1.2)
type PhoneNumber struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // work, home, mobile, fax, pager, other
	Primary bool   `json:"primary,omitempty"`
}

// IM represents an instant messaging address (RFC 7643 Section 4.1.2)
type IM struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // aim, gtalk, icq, xmpp, msn, skype, qq, yahoo
	Primary bool   `json:"primary,omitempty"`
}

// Photo represents a photo URL (RFC 7643 Section 4.1.2)
type Photo struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // photo, thumbnail
	Primary bool   `json:"primary,omitempty"`
}

// SCIMAddress represents a physical mailing address (RFC 7643 Section 4.1.2)
type SCIMAddress struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
	Type          string `json:"type,omitempty"` // work, home, other
	Primary       bool   `json:"primary,omitempty"`
}

// GroupReference represents a group membership (RFC 7643 Section 4.1.2)
type GroupReference struct {
	Value   string `json:"value"` // Group ID
	Ref     string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // direct, indirect
}

// Entitlement represents an entitlement (RFC 7643 Section 4.1.2)
type Entitlement struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// SCIMRole represents a role (RFC 7643 Section 4.1.2)
type SCIMRole struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// X509Certificate represents an X.509 certificate (RFC 7643 Section 4.1.2)
type X509Certificate struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// EnterpriseUser represents enterprise user extension (RFC 7643 Section 4.3)
type EnterpriseUser struct {
	EmployeeNumber string            `json:"employeeNumber,omitempty"`
	CostCenter     string            `json:"costCenter,omitempty"`
	Organization   string            `json:"organization,omitempty"`
	Division       string            `json:"division,omitempty"`
	Department     string            `json:"department,omitempty"`
	Manager        *ManagerReference `json:"manager,omitempty"`
}

// ManagerReference represents a manager reference (RFC 7643 Section 4.3.1)
type ManagerReference struct {
	Value       string `json:"value"` // Manager's ID
	Ref         string `json:"$ref,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// MemberReference represents a group member (RFC 7643 Section 4.2)
type MemberReference struct {
	Value   string `json:"value"` // User ID
	Ref     string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"` // User or Group
}

// ListResponse represents a SCIM list response (RFC 7644 Section 3.4.2)
type ListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
	Resources    []interface{} `json:"Resources"`
}

// ErrorResponse represents a SCIM error response (RFC 7644 Section 3.12)
type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Status   int      `json:"status"`
	ScimType string   `json:"scimType,omitempty"` // invalidFilter, tooMany, uniqueness, mutability, invalidSyntax, invalidPath, invalidValue, invalidVers, sensitive, notTarget
	Detail   string   `json:"detail,omitempty"`
}

// PatchOp represents a PATCH operation (RFC 7644 Section 3.5.2)
type PatchOp struct {
	Schemas    []string         `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

// PatchOperation represents a single patch operation (RFC 7644 Section 3.5.2)
type PatchOperation struct {
	Op    string      `json:"op"` // add, remove, replace
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// BulkRequest represents a bulk operation request (RFC 7644 Section 3.7)
type BulkRequest struct {
	Schemas      []string        `json:"schemas"`
	FailOnErrors int             `json:"failOnErrors,omitempty"`
	Operations   []BulkOperation `json:"Operations"`
}

// BulkOperation represents a single bulk operation (RFC 7644 Section 3.7)
type BulkOperation struct {
	Method  string      `json:"method"` // POST, PUT, PATCH, DELETE
	BulkID  string      `json:"bulkId,omitempty"`
	Version string      `json:"version,omitempty"`
	Path    string      `json:"path"`
	Data    interface{} `json:"data,omitempty"`
}

// BulkResponse represents a bulk operation response (RFC 7644 Section 3.7)
type BulkResponse struct {
	Schemas    []string              `json:"schemas"`
	Operations []BulkOperationResult `json:"Operations"`
}

// BulkOperationResult represents a single bulk operation result (RFC 7644 Section 3.7)
type BulkOperationResult struct {
	Method   string      `json:"method"`
	BulkID   string      `json:"bulkId,omitempty"`
	Version  string      `json:"version,omitempty"`
	Location string      `json:"location,omitempty"`
	Status   int         `json:"status"`
	Response interface{} `json:"response,omitempty"`
}

// ServiceProviderConfig represents the service provider configuration (RFC 7643 Section 5)
type ServiceProviderConfig struct {
	Schemas               []string               `json:"schemas"`
	DocumentationURI      string                 `json:"documentationUri,omitempty"`
	Patch                 *Supported             `json:"patch"`
	Bulk                  *BulkSupport           `json:"bulk"`
	Filter                *FilterSupport         `json:"filter"`
	ChangePassword        *Supported             `json:"changePassword"`
	Sort                  *Supported             `json:"sort"`
	Etag                  *Supported             `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme `json:"authenticationSchemes"`
	Meta                  *SCIMMeta              `json:"meta"`
}

// Supported indicates feature support (RFC 7643 Section 5)
type Supported struct {
	Supported bool `json:"supported"`
}

// BulkSupport indicates bulk operation support (RFC 7643 Section 5)
type BulkSupport struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

// FilterSupport indicates filter support (RFC 7643 Section 5)
type FilterSupport struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

// AuthenticationScheme represents an authentication scheme (RFC 7643 Section 5)
type AuthenticationScheme struct {
	Type             string `json:"type"` // oauth, oauth2, oauthbearertoken, httpbasic, httpdigest
	Name             string `json:"name"`
	Description      string `json:"description"`
	SpecURI          string `json:"specUri,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
}

// ResourceType represents a resource type (RFC 7643 Section 6)
type ResourceType struct {
	Schemas          []string          `json:"schemas"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Endpoint         string            `json:"endpoint"`
	Description      string            `json:"description,omitempty"`
	Schema           string            `json:"schema"`
	SchemaExtensions []SchemaExtension `json:"schemaExtensions,omitempty"`
	Meta             *SCIMMeta         `json:"meta"`
}

// SchemaExtension represents a schema extension (RFC 7643 Section 6)
type SchemaExtension struct {
	Schema   string `json:"schema"`
	Required bool   `json:"required"`
}

// Schema represents a SCIM schema (RFC 7643 Section 7)
type Schema struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Attributes  []Attribute `json:"attributes"`
	Meta        *SCIMMeta   `json:"meta,omitempty"`
}

// Attribute represents a schema attribute (RFC 7643 Section 7)
type Attribute struct {
	Name            string      `json:"name"`
	Type            string      `json:"type"` // string, boolean, decimal, integer, dateTime, reference, complex, binary
	MultiValued     bool        `json:"multiValued"`
	Description     string      `json:"description,omitempty"`
	Required        bool        `json:"required"`
	CanonicalValues []string    `json:"canonicalValues,omitempty"`
	CaseExact       bool        `json:"caseExact"`
	Mutability      string      `json:"mutability"` // readOnly, readWrite, immutable, writeOnly
	Returned        string      `json:"returned"`   // always, never, default, request
	Uniqueness      string      `json:"uniqueness"` // none, server, global
	SubAttributes   []Attribute `json:"subAttributes,omitempty"`
	ReferenceTypes  []string    `json:"referenceTypes,omitempty"`
}

// Database models for SCIM provisioning

// ProvisioningToken represents a SCIM provisioning token (Bearer token)
// Updated for 3-tier architecture: App → Environment → Organization
type ProvisioningToken struct {
	ID             xid.ID     `bun:"id,pk,type:varchar(20)"`
	AppID          xid.ID     `bun:"app_id,type:varchar(20),notnull"`          // Platform app
	EnvironmentID  xid.ID     `bun:"environment_id,type:varchar(20),notnull"`  // Target environment (dev, prod, etc.)
	OrganizationID xid.ID     `bun:"organization_id,type:varchar(20),notnull"` // User-created organization
	Name           string     `bun:"name,notnull"`
	Description    string     `bun:"description"`
	TokenHash      string     `bun:"token_hash,notnull,unique"` // bcrypt hash
	TokenPrefix    string     `bun:"token_prefix,notnull"`      // First 8 chars for identification
	Scopes         []string   `bun:"scopes,type:text[],notnull"`
	ExpiresAt      *time.Time `bun:"expires_at"`
	LastUsedAt     *time.Time `bun:"last_used_at"`
	CreatedBy      xid.ID     `bun:"created_by,type:varchar(20)"`
	CreatedAt      time.Time  `bun:"created_at,notnull"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull"`
	RevokedAt      *time.Time `bun:"revoked_at"`
}

// ProvisioningLog represents a log entry for provisioning operations
// Updated for 3-tier architecture: App → Environment → Organization
type ProvisioningLog struct {
	ID             xid.ID                 `bun:"id,pk,type:varchar(20)"`
	AppID          xid.ID                 `bun:"app_id,type:varchar(20),notnull"`          // Platform app
	EnvironmentID  xid.ID                 `bun:"environment_id,type:varchar(20),notnull"`  // Target environment
	OrganizationID xid.ID                 `bun:"organization_id,type:varchar(20),notnull"` // User-created organization
	TokenID        xid.ID                 `bun:"token_id,type:varchar(20)"`
	Operation      string                 `bun:"operation,notnull"`     // CREATE_USER, UPDATE_USER, DELETE_USER, etc.
	ResourceType   string                 `bun:"resource_type,notnull"` // User, Group
	ResourceID     string                 `bun:"resource_id"`
	ExternalID     string                 `bun:"external_id"`
	Method         string                 `bun:"method,notnull"` // POST, PUT, PATCH, DELETE
	Path           string                 `bun:"path,notnull"`
	StatusCode     int                    `bun:"status_code,notnull"`
	Success        bool                   `bun:"success,notnull"`
	ErrorMessage   string                 `bun:"error_message"`
	RequestBody    map[string]interface{} `bun:"request_body,type:jsonb"`
	ResponseBody   map[string]interface{} `bun:"response_body,type:jsonb"`
	IPAddress      string                 `bun:"ip_address"`
	UserAgent      string                 `bun:"user_agent"`
	DurationMS     int                    `bun:"duration_ms"`
	CreatedAt      time.Time              `bun:"created_at,notnull"`
}

// AttributeMapping represents custom attribute mappings per organization
// Updated for 3-tier architecture: App → Environment → Organization
type AttributeMapping struct {
	bun.BaseModel  `bun:"table:attribute_mappings,alias:am"`
	ID             xid.ID                 `bun:"id,pk,type:varchar(20)"`
	AppID          xid.ID                 `bun:"app_id,type:varchar(20),notnull"`                                    // Platform app
	EnvironmentID  xid.ID                 `bun:"environment_id,type:varchar(20),notnull"`                            // Target environment
	OrganizationID xid.ID                 `bun:"organization_id,type:varchar(20),notnull,unique:org_mapping_unique"` // User-created organization
	Mappings       map[string]string      `bun:"mappings,type:jsonb,notnull"`                                        // SCIM attr -> AuthSome field
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb"`
	CreatedAt      time.Time              `bun:"created_at,notnull"`
	UpdatedAt      time.Time              `bun:"updated_at,notnull"`
}

// GroupMapping represents SCIM group to user-created organization team/role mapping
// Updated for 3-tier architecture: App → Environment → Organization
type GroupMapping struct {
	bun.BaseModel  `bun:"table:group_mappings,alias:gm"`
	ID             xid.ID    `bun:"id,pk,type:varchar(20)"`
	AppID          xid.ID    `bun:"app_id,type:varchar(20),notnull"`          // Platform app
	EnvironmentID  xid.ID    `bun:"environment_id,type:varchar(20),notnull"`  // Target environment
	OrganizationID xid.ID    `bun:"organization_id,type:varchar(20),notnull"` // User-created organization
	SCIMGroupID    string    `bun:"scim_group_id,notnull"`
	SCIMGroupName  string    `bun:"scim_group_name,notnull"`
	MappingType    string    `bun:"mapping_type,notnull"`               // team, role (in user-created organization)
	TargetID       xid.ID    `bun:"target_id,type:varchar(20),notnull"` // Team ID or Role ID in user organization
	CreatedAt      time.Time `bun:"created_at,notnull"`
	UpdatedAt      time.Time `bun:"updated_at,notnull"`
}

// API Request/Response Types for SCIM Admin Endpoints

// CreateTokenRequest is the request body for creating a provisioning token
type CreateTokenRequest struct {
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description"`
	Scopes      []string   `json:"scopes" validate:"required,min=1"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

// TokenResponse is the response for token creation
type TokenResponse struct {
	Token   string `json:"token"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// TokenListResponse represents a list of provisioning tokens (without actual token values)
type TokenListResponse struct {
	Tokens []ProvisioningTokenInfo `json:"tokens"`
	Total  int                     `json:"total"`
}

// ProvisioningTokenInfo contains token metadata without the actual token
type ProvisioningTokenInfo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
}

// UpdateAttributeMappingsRequest is the request body for updating attribute mappings
type UpdateAttributeMappingsRequest struct {
	Mappings map[string]string `json:"mappings" validate:"required"`
}

// AttributeMappingsResponse is the response for attribute mappings
type AttributeMappingsResponse struct {
	ID       string            `json:"id"`
	Mappings map[string]string `json:"mappings"`
}

// SearchRequest represents a SCIM search request (RFC 7644 Section 3.4.3)
type SearchRequest struct {
	Schemas            []string `json:"schemas"`
	Attributes         []string `json:"attributes,omitempty"`
	ExcludedAttributes []string `json:"excludedAttributes,omitempty"`
	Filter             string   `json:"filter,omitempty"`
	SortBy             string   `json:"sortBy,omitempty"`
	SortOrder          string   `json:"sortOrder,omitempty"` // ascending, descending
	StartIndex         int      `json:"startIndex,omitempty"`
	Count              int      `json:"count,omitempty"`
}

// LogsResponse represents a list of provisioning logs
type LogsResponse struct {
	Logs  []ProvisioningLog `json:"logs"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// StatsResponse represents provisioning statistics
type StatsResponse struct {
	TotalOperations int               `json:"totalOperations"`
	SuccessCount    int               `json:"successCount"`
	FailureCount    int               `json:"failureCount"`
	SuccessRate     float64           `json:"successRate"`
	ByOperation     map[string]int    `json:"byOperation"`
	ByResourceType  map[string]int    `json:"byResourceType"`
	ByStatus        map[string]int    `json:"byStatus"`
	Recent          []ProvisioningLog `json:"recent"`
	StartDate       *time.Time        `json:"startDate,omitempty"`
	EndDate         *time.Time        `json:"endDate,omitempty"`
}

// UsersResponse wraps user list response for clarity
type UsersResponse struct {
	Users []SCIMUser `json:"users"`
	Total int        `json:"total"`
}

// GroupsResponse wraps group list response for clarity
type GroupsResponse struct {
	Groups []SCIMGroup `json:"groups"`
	Total  int         `json:"total"`
}
