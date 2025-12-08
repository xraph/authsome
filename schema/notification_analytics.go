package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationEvent represents the type of analytics event
type NotificationEvent string

const (
	NotificationEventSent       NotificationEvent = "sent"       // Notification sent
	NotificationEventDelivered  NotificationEvent = "delivered"  // Provider confirmed delivery
	NotificationEventOpened     NotificationEvent = "opened"     // Recipient opened (email tracking pixel)
	NotificationEventClicked    NotificationEvent = "clicked"    // Recipient clicked link (tracked URL)
	NotificationEventConverted  NotificationEvent = "converted"  // Recipient completed desired action
	NotificationEventBounced    NotificationEvent = "bounced"    // Delivery failed permanently
	NotificationEventComplained NotificationEvent = "complained" // Recipient marked as spam
	NotificationEventFailed     NotificationEvent = "failed"     // General failure
)

// NotificationAnalytics represents a single analytics event for a notification
type NotificationAnalytics struct {
	bun.BaseModel `bun:"table:notification_analytics,alias:na"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	NotificationID xid.ID                 `bun:"notification_id,notnull,type:varchar(20)" json:"notificationId"`
	TemplateID     *xid.ID                `bun:"template_id,type:varchar(20)" json:"templateId,omitempty"`
	AppID          xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	OrganizationID *xid.ID                `bun:"organization_id,type:varchar(20)" json:"organizationId,omitempty"`
	Event          string                 `bun:"event,notnull" json:"event"`                       // sent, delivered, opened, clicked, converted, bounced, complained
	EventData      map[string]interface{} `bun:"event_data,type:jsonb" json:"eventData,omitempty"` // Additional event-specific data (e.g., link clicked, conversion value)
	UserAgent      string                 `bun:"user_agent" json:"userAgent,omitempty"`
	IPAddress      string                 `bun:"ip_address" json:"ipAddress,omitempty"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Notification *Notification         `bun:"rel:belongs-to,join:notification_id=id" json:"notification,omitempty"`
	Template     *NotificationTemplate `bun:"rel:belongs-to,join:template_id=id" json:"template,omitempty"`
	App          *App                  `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Organization *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}
