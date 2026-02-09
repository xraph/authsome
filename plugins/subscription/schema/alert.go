package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SubscriptionAlertConfig represents an alert configuration in the database.
type SubscriptionAlertConfig struct {
	bun.BaseModel `bun:"table:subscription_alert_configs,alias:sac"`

	ID               xid.ID     `bun:"id,pk,type:char(20)"`
	AppID            xid.ID     `bun:"app_id,notnull,type:char(20)"`
	OrganizationID   xid.ID     `bun:"organization_id,notnull,type:char(20)"`
	AlertType        string     `bun:"alert_type,notnull"`
	IsEnabled        bool       `bun:"is_enabled,notnull,default:true"`
	ThresholdPercent float64    `bun:"threshold_percent"`
	MetricKey        string     `bun:"metric_key"`
	DaysBeforeEnd    int        `bun:"days_before_end"`
	Channels         []string   `bun:"channels,array"`
	Recipients       []string   `bun:"recipients,array"`
	WebhookURL       string     `bun:"webhook_url"`
	SlackChannel     string     `bun:"slack_channel"`
	MinInterval      int        `bun:"min_interval,notnull,default:60"`
	MaxAlertsPerDay  int        `bun:"max_alerts_per_day,notnull,default:5"`
	SnoozedUntil     *time.Time `bun:"snoozed_until"`
	CreatedAt        time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt        time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionAlert represents an alert in the database.
type SubscriptionAlert struct {
	bun.BaseModel `bun:"table:subscription_alerts,alias:sa"`

	ID             xid.ID     `bun:"id,pk,type:char(20)"`
	AppID          xid.ID     `bun:"app_id,notnull,type:char(20)"`
	OrganizationID xid.ID     `bun:"organization_id,notnull,type:char(20)"`
	ConfigID       *xid.ID    `bun:"config_id,type:char(20)"`
	Type           string     `bun:"type,notnull"`
	Severity       string     `bun:"severity,notnull,default:'info'"`
	Status         string     `bun:"status,notnull,default:'pending'"`
	Title          string     `bun:"title,notnull"`
	Message        string     `bun:"message,notnull"`
	MetricKey      string     `bun:"metric_key"`
	CurrentValue   float64    `bun:"current_value"`
	ThresholdValue float64    `bun:"threshold_value"`
	LimitValue     float64    `bun:"limit_value"`
	SubscriptionID *xid.ID    `bun:"subscription_id,type:char(20)"`
	InvoiceID      *xid.ID    `bun:"invoice_id,type:char(20)"`
	Channels       []string   `bun:"channels,array"`
	SentAt         *time.Time `bun:"sent_at"`
	DeliveryStatus string     `bun:"delivery_status,type:jsonb"`
	AcknowledgedAt *time.Time `bun:"acknowledged_at"`
	AcknowledgedBy string     `bun:"acknowledged_by"`
	ResolvedAt     *time.Time `bun:"resolved_at"`
	Resolution     string     `bun:"resolution"`
	Metadata       string     `bun:"metadata,type:jsonb"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionAlertTemplate represents an alert template in the database.
type SubscriptionAlertTemplate struct {
	bun.BaseModel `bun:"table:subscription_alert_templates,alias:sat"`

	ID            xid.ID    `bun:"id,pk,type:char(20)"`
	AppID         xid.ID    `bun:"app_id,notnull,type:char(20)"`
	AlertType     string    `bun:"alert_type,notnull"`
	Channel       string    `bun:"channel,notnull"`
	Subject       string    `bun:"subject"`
	TitleTemplate string    `bun:"title_template,notnull"`
	BodyTemplate  string    `bun:"body_template,notnull"`
	IsDefault     bool      `bun:"is_default,notnull,default:false"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
