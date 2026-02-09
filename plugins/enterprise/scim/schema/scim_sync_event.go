package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SCIMSyncEvent represents a SCIM synchronization event for monitoring.
type SCIMSyncEvent struct {
	bun.BaseModel `bun:"table:scim_sync_events"`

	ID         xid.ID `bun:"id,pk,type:varchar(20)"               json:"id"`
	ProviderID xid.ID `bun:"provider_id,notnull,type:varchar(20)" json:"provider_id"`
	TokenID    xid.ID `bun:"token_id,type:varchar(20)"            json:"token_id,omitempty"`

	EventType string `bun:"event_type,notnull" json:"event_type"` // "user_create", "user_update", "user_delete", "group_create", etc.
	Direction string `bun:"direction"          json:"direction"`  // "inbound", "outbound"
	Status    string `bun:"status"             json:"status"`     // "success", "failed", "pending"

	ResourceType string  `bun:"resource_type"                json:"resource_type"` // "User", "Group"
	ResourceID   *xid.ID `bun:"resource_id,type:varchar(20)" json:"resource_id,omitempty"`
	ExternalID   *string `bun:"external_id"                  json:"external_id,omitempty"`

	Details      map[string]any `bun:"details,type:jsonb" json:"details,omitempty"`
	ErrorMessage *string        `bun:"error_message"      json:"error_message,omitempty"`

	Duration  int64     `bun:"duration_ms"                                  json:"duration_ms"` // Operation duration in milliseconds
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// BeforeInsert hook to set ID and timestamp.
func (e *SCIMSyncEvent) BeforeInsert(ctx context.Context, db *bun.DB) error {
	if e.ID.IsNil() {
		e.ID = xid.New()
	}

	e.CreatedAt = time.Now()

	return nil
}
