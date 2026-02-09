package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// UsageEvent represents the usage_events table for tracking API usage
// Note: Indexes should be created in migrations for the following columns:
// user_id, organization_id, session_id, api_key_id, method, endpoint,
// status_code, auth_method, country, plugin, feature.
type UsageEvent struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:usage_events,alias:ue"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"           json:"id"`
	UserID         *xid.ID `bun:"user_id,type:varchar(20)"         json:"userId,omitempty"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationId,omitempty"`
	SessionID      *xid.ID `bun:"session_id,type:varchar(20)"      json:"sessionId,omitempty"`
	APIKeyID       *xid.ID `bun:"api_key_id,type:varchar(20)"      json:"apiKeyId,omitempty"`

	// Request details
	Method     string `bun:"method,notnull" json:"method"`
	Path       string `bun:"path,notnull"   json:"path"`
	Endpoint   string `bun:"endpoint"       json:"endpoint"` // Normalized endpoint
	StatusCode int    `bun:"status_code"    json:"statusCode"`

	// Authentication context
	AuthMethod string `bun:"auth_method" json:"authMethod,omitempty"` // session, apikey, jwt, anonymous

	// Network information
	IPAddress string `bun:"ip_address"           json:"ipAddress,omitempty"`
	UserAgent string `bun:"user_agent,type:text" json:"userAgent,omitempty"`
	Country   string `bun:"country"              json:"country,omitempty"`
	City      string `bun:"city"                 json:"city,omitempty"`

	// Performance metrics
	ResponseTimeMs int64 `bun:"response_time_ms" json:"responseTimeMs"`
	RequestSize    int64 `bun:"request_size"     json:"requestSize"`
	ResponseSize   int64 `bun:"response_size"    json:"responseSize"`

	// Feature tracking
	Plugin  string `bun:"plugin"  json:"plugin,omitempty"`
	Feature string `bun:"feature" json:"feature,omitempty"`

	// Error tracking
	Error     string `bun:"error,type:text" json:"error,omitempty"`
	ErrorCode string `bun:"error_code"      json:"errorCode,omitempty"`

	// Metadata
	Metadata string `bun:"metadata,type:text" json:"metadata,omitempty"` // JSON string
}
