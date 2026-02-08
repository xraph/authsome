package audit

import (
	"time"

	"github.com/rs/xid"
)

// AggregationFilter defines filters for aggregation queries
type AggregationFilter struct {
	// Scope filters
	AppID          *xid.ID `json:"appId,omitempty" query:"app_id"`
	OrganizationID *xid.ID `json:"organizationId,omitempty" query:"organization_id"`
	EnvironmentID  *xid.ID `json:"environmentId,omitempty" query:"environment_id"`

	// Time range
	Since *time.Time `json:"since,omitempty" query:"since"`
	Until *time.Time `json:"until,omitempty" query:"until"`

	// Limit for top N results
	Limit int `json:"limit,omitempty" query:"limit"`
}

// DistinctValue represents a distinct field value with count
type DistinctValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// ActionAggregation contains distinct actions with counts
type ActionAggregation struct {
	Actions []DistinctValue `json:"actions"`
	Total   int             `json:"total"`
}

// SourceAggregation contains distinct sources with counts
type SourceAggregation struct {
	Sources []DistinctValue `json:"sources"`
	Total   int             `json:"total"`
}

// ResourceAggregation contains distinct resources with counts
type ResourceAggregation struct {
	Resources []DistinctValue `json:"resources"`
	Total     int             `json:"total"`
}

// UserAggregation contains distinct users with counts
type UserAggregation struct {
	Users []DistinctValue `json:"users"`
	Total int             `json:"total"`
}

// IPAggregation contains distinct IPs with counts
type IPAggregation struct {
	IPAddresses []DistinctValue `json:"ipAddresses"`
	Total       int             `json:"total"`
}

// AppAggregation contains distinct apps with counts
type AppAggregation struct {
	Apps  []DistinctValue `json:"apps"`
	Total int             `json:"total"`
}

// OrgAggregation contains distinct organizations with counts
type OrgAggregation struct {
	Organizations []DistinctValue `json:"organizations"`
	Total         int             `json:"total"`
}

// AllAggregations combines all aggregations in one response
type AllAggregations struct {
	Actions       []DistinctValue `json:"actions"`
	Sources       []DistinctValue `json:"sources"`
	Resources     []DistinctValue `json:"resources"`
	Users         []DistinctValue `json:"users"`
	IPAddresses   []DistinctValue `json:"ipAddresses"`
	Apps          []DistinctValue `json:"apps"`
	Organizations []DistinctValue `json:"organizations"`
}
