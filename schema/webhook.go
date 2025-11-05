package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Webhook represents a webhook subscription
type Webhook struct {
	bun.BaseModel `bun:"table:webhooks,alias:w"`

	ID             xid.ID            `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID string            `bun:"organization_id,notnull" json:"organization_id"`
	URL            string            `bun:"url,notnull" json:"url"`
	Events         []string          `bun:"events,array" json:"events"`
	Secret         string            `bun:"secret,notnull" json:"-"`
	Active         bool              `bun:"active,notnull,default:true" json:"active"`
	Headers        map[string]string `bun:"headers,type:jsonb" json:"headers,omitempty"`
	Metadata       map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time         `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time         `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt      *time.Time        `bun:"deleted_at,soft_delete,nullzero" json:"-"`
}

// Event represents a webhook event
type Event struct {
	bun.BaseModel `bun:"table:webhook_events,alias:we"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID string                 `bun:"organization_id,notnull" json:"organization_id"`
	Type           string                 `bun:"type,notnull" json:"type"`
	Data           map[string]interface{} `bun:"data,type:jsonb" json:"data"`
	UserID         *xid.ID                `bun:"user_id,type:varchar(20)" json:"user_id,omitempty"`
	SessionID      *xid.ID                `bun:"session_id,type:varchar(20)" json:"session_id,omitempty"`
	IPAddress      string                 `bun:"ip_address" json:"ip_address,omitempty"`
	UserAgent      string                 `bun:"user_agent" json:"user_agent,omitempty"`
	CreatedAt      time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// Delivery represents a webhook delivery attempt
type Delivery struct {
	bun.BaseModel `bun:"table:webhook_deliveries,alias:wd"`

	ID              xid.ID     `bun:"id,pk,type:varchar(20)" json:"id"`
	WebhookID       xid.ID     `bun:"webhook_id,notnull,type:varchar(20)" json:"webhook_id"`
	EventID         xid.ID     `bun:"event_id,notnull,type:varchar(20)" json:"event_id"`
	URL             string     `bun:"url,notnull" json:"url"`
	HTTPMethod      string     `bun:"http_method,notnull,default:'POST'" json:"http_method"`
	Headers         []byte     `bun:"headers,type:jsonb" json:"headers,omitempty"`
	Body            []byte     `bun:"body" json:"body,omitempty"`
	Status          string     `bun:"status,notnull" json:"status"`
	StatusCode      *int       `bun:"status_code" json:"status_code,omitempty"`
	ResponseHeaders []byte     `bun:"response_headers,type:jsonb" json:"response_headers,omitempty"`
	ResponseBody    []byte     `bun:"response_body" json:"response_body,omitempty"`
	Error           *string    `bun:"error" json:"error,omitempty"`
	AttemptNumber   int        `bun:"attempt_number,notnull,default:1" json:"attempt_number"`
	NextRetryAt     *time.Time `bun:"next_retry_at" json:"next_retry_at,omitempty"`
	DeliveredAt     *time.Time `bun:"delivered_at" json:"delivered_at,omitempty"`
	CreatedAt       time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Webhook *Webhook `bun:"rel:belongs-to,join:webhook_id=id" json:"webhook,omitempty"`
	Event   *Event   `bun:"rel:belongs-to,join:event_id=id" json:"event,omitempty"`
}
