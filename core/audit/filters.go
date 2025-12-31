package audit

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListEventsFilter defines filters for listing audit events with pagination
type ListEventsFilter struct {
	pagination.PaginationParams

	// ========== Full-Text Search ==========
	// Full-text search query across action, resource, metadata
	SearchQuery *string `json:"searchQuery,omitempty" query:"q"`
	// Fields to search (empty = all fields)
	SearchFields []string `json:"searchFields,omitempty" query:"search_fields"`

	// ========== Exact Match Filters ==========
	// Filter by environment
	EnvironmentID *xid.ID `json:"environmentId,omitempty" query:"environment_id"`

	// Filter by user (single)
	UserID *xid.ID `json:"userId,omitempty" query:"user_id"`

	// Filter by action (single, exact match)
	Action *string `json:"action,omitempty" query:"action"`

	// Filter by resource (single, exact match)
	Resource *string `json:"resource,omitempty" query:"resource"`

	// Filter by IP address (single, exact match)
	IPAddress *string `json:"ipAddress,omitempty" query:"ip_address"`

	// ========== Multiple Value Filters (IN clauses) ==========
	// Filter by multiple users
	UserIDs []xid.ID `json:"userIds,omitempty" query:"user_ids"`

	// Filter by multiple actions
	Actions []string `json:"actions,omitempty" query:"actions"`

	// Filter by multiple resources
	Resources []string `json:"resources,omitempty" query:"resources"`

	// Filter by multiple IP addresses
	IPAddresses []string `json:"ipAddresses,omitempty" query:"ip_addresses"`

	// ========== Pattern Matching Filters (ILIKE) ==========
	// Action pattern match (use % for wildcards)
	ActionPattern *string `json:"actionPattern,omitempty" query:"action_pattern"`

	// Resource pattern match (use % for wildcards)
	ResourcePattern *string `json:"resourcePattern,omitempty" query:"resource_pattern"`

	// ========== IP Range Filtering ==========
	// IP range in CIDR notation (e.g., "192.168.1.0/24")
	IPRange *string `json:"ipRange,omitempty" query:"ip_range"`

	// ========== Metadata Filtering ==========
	// Metadata key-value filters (for structured metadata)
	MetadataFilters []MetadataFilter `json:"metadataFilters,omitempty" query:"metadata_filters"`

	// ========== Time Range Filters ==========
	Since *time.Time `json:"since,omitempty" query:"since"`
	Until *time.Time `json:"until,omitempty" query:"until"`

	// ========== Sort Order ==========
	SortBy    *string `json:"sortBy,omitempty" query:"sort_by"`       // created_at, action, resource, rank (for search)
	SortOrder *string `json:"sortOrder,omitempty" query:"sort_order"` // asc, desc
}

// MetadataFilter defines a filter for metadata field
type MetadataFilter struct {
	Key      string      `json:"key"`               // Metadata key to filter on
	Value    interface{} `json:"value"`             // Value to match
	Operator string      `json:"operator"`          // equals, contains, exists, not_exists
}
