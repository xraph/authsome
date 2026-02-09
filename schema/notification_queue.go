package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationQueueStatus represents the status of a queued notification.
type NotificationQueueStatus string

const (
	NotificationQueueStatusPending    NotificationQueueStatus = "pending"
	NotificationQueueStatusProcessing NotificationQueueStatus = "processing"
	NotificationQueueStatusSucceeded  NotificationQueueStatus = "succeeded"
	NotificationQueueStatusFailed     NotificationQueueStatus = "failed"
)

// NotificationQueue represents a notification queued for retry in the database.
type NotificationQueue struct {
	bun.BaseModel `bun:"table:notification_queue,alias:nq"`

	ID    xid.ID `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID xid.ID `bun:"app_id,notnull,type:varchar(20)" json:"appId"`

	// Notification details
	Type        string `bun:"type,notnull"      json:"type"`                  // email, sms, push
	Priority    string `bun:"priority,notnull"  json:"priority"`              // critical, high, normal, low
	Recipient   string `bun:"recipient,notnull" json:"recipient"`             // Email address or phone number
	Subject     string `bun:"subject"           json:"subject,omitempty"`     // Email subject
	Body        string `bun:"body"              json:"body,omitempty"`        // Direct body content
	TemplateKey string `bun:"template_key"      json:"templateKey,omitempty"` // Template key if using template

	// Payload for retry (JSON serialized request)
	Payload []byte `bun:"payload,type:bytea" json:"payload,omitempty"`

	// Retry state
	Attempts    int                     `bun:"attempts,notnull,default:0"       json:"attempts"`
	MaxAttempts int                     `bun:"max_attempts,notnull,default:3"   json:"maxAttempts"`
	LastError   string                  `bun:"last_error"                       json:"lastError,omitempty"`
	Status      NotificationQueueStatus `bun:"status,notnull,default:'pending'" json:"status"`
	NextRetryAt *time.Time              `bun:"next_retry_at"                    json:"nextRetryAt,omitempty"`
	ProcessedAt *time.Time              `bun:"processed_at"                     json:"processedAt,omitempty"`

	// Audit fields
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

// NotificationQueueStats holds aggregate statistics for the notification queue.
type NotificationQueueStats struct {
	PendingCount    int64 `json:"pendingCount"`
	ProcessingCount int64 `json:"processingCount"`
	SucceededCount  int64 `json:"succeededCount"`
	FailedCount     int64 `json:"failedCount"`
	TotalCount      int64 `json:"totalCount"`
}
