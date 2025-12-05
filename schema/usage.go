package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// UsageEvent represents the usage_events table for tracking API usage
// Note: Indexes should be created in migrations for the following columns:
// user_id, organization_id, session_id, api_key_id, method, endpoint,
// status_code, auth_method, country, plugin, feature
type UsageEvent struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:usage_events,alias:ue"`

	ID             xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID         *xid.ID `json:"userId,omitempty" bun:"user_id,type:varchar(20)"`
	OrganizationID *xid.ID `json:"organizationId,omitempty" bun:"organization_id,type:varchar(20)"`
	SessionID      *xid.ID `json:"sessionId,omitempty" bun:"session_id,type:varchar(20)"`
	APIKeyID       *xid.ID `json:"apiKeyId,omitempty" bun:"api_key_id,type:varchar(20)"`

	// Request details
	Method     string `json:"method" bun:"method,notnull"`
	Path       string `json:"path" bun:"path,notnull"`
	Endpoint   string `json:"endpoint" bun:"endpoint"` // Normalized endpoint
	StatusCode int    `json:"statusCode" bun:"status_code"`

	// Authentication context
	AuthMethod string `json:"authMethod,omitempty" bun:"auth_method"` // session, apikey, jwt, anonymous

	// Network information
	IPAddress string `json:"ipAddress,omitempty" bun:"ip_address"`
	UserAgent string `json:"userAgent,omitempty" bun:"user_agent,type:text"`
	Country   string `json:"country,omitempty" bun:"country"`
	City      string `json:"city,omitempty" bun:"city"`

	// Performance metrics
	ResponseTimeMs int64 `json:"responseTimeMs" bun:"response_time_ms"`
	RequestSize    int64 `json:"requestSize" bun:"request_size"`
	ResponseSize   int64 `json:"responseSize" bun:"response_size"`

	// Feature tracking
	Plugin  string `json:"plugin,omitempty" bun:"plugin"`
	Feature string `json:"feature,omitempty" bun:"feature"`

	// Error tracking
	Error     string `json:"error,omitempty" bun:"error,type:text"`
	ErrorCode string `json:"errorCode,omitempty" bun:"error_code"`

	// Metadata
	Metadata string `json:"metadata,omitempty" bun:"metadata,type:text"` // JSON string
}
