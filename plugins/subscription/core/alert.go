package core

import (
	"time"

	"github.com/rs/xid"
)

// AlertType represents the type of usage alert
type AlertType string

const (
	AlertTypeUsageThreshold       AlertType = "usage_threshold"        // When usage reaches X%
	AlertTypeUsageLimit           AlertType = "usage_limit"            // When usage hits limit
	AlertTypePaymentFailed        AlertType = "payment_failed"         // Payment failure
	AlertTypeTrialEnding          AlertType = "trial_ending"           // Trial about to end
	AlertTypeSubscriptionExpiring AlertType = "subscription_expiring"  // Subscription expiring
	AlertTypeInvoicePastDue       AlertType = "invoice_past_due"       // Invoice past due
	AlertTypeSeatLimitApproaching AlertType = "seat_limit_approaching" // Seat limit approaching
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusPending      AlertStatus = "pending"      // Alert created, not yet sent
	AlertStatusSent         AlertStatus = "sent"         // Alert sent
	AlertStatusAcknowledged AlertStatus = "acknowledged" // Alert acknowledged by user
	AlertStatusResolved     AlertStatus = "resolved"     // Alert condition resolved
	AlertStatusSnoozed      AlertStatus = "snoozed"      // Alert snoozed
)

// AlertChannel represents how an alert is delivered
type AlertChannel string

const (
	AlertChannelEmail   AlertChannel = "email"
	AlertChannelWebhook AlertChannel = "webhook"
	AlertChannelInApp   AlertChannel = "in_app"
	AlertChannelSMS     AlertChannel = "sms"
	AlertChannelSlack   AlertChannel = "slack"
)

// AlertConfig represents alert configuration for an organization
type AlertConfig struct {
	ID             xid.ID `json:"id"`
	AppID          xid.ID `json:"appId"`
	OrganizationID xid.ID `json:"organizationId"`

	// Alert type settings
	AlertType AlertType `json:"alertType"`
	IsEnabled bool      `json:"isEnabled"`

	// Threshold settings (for usage alerts)
	ThresholdPercent float64 `json:"thresholdPercent"` // Trigger at X% (e.g., 80, 90, 100)
	MetricKey        string  `json:"metricKey"`        // Which metric to monitor

	// Timing settings
	DaysBeforeEnd int `json:"daysBeforeEnd"` // For trial/subscription ending alerts

	// Delivery settings
	Channels     []AlertChannel `json:"channels"`
	Recipients   []string       `json:"recipients"`   // Email addresses
	WebhookURL   string         `json:"webhookUrl"`   // For webhook channel
	SlackChannel string         `json:"slackChannel"` // For Slack channel

	// Frequency settings
	MinInterval     int `json:"minInterval"`     // Minimum minutes between alerts
	MaxAlertsPerDay int `json:"maxAlertsPerDay"` // Max alerts per day (0 = unlimited)

	// Snooze settings
	SnoozedUntil *time.Time `json:"snoozedUntil"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Alert represents an individual alert instance
type Alert struct {
	ID             xid.ID  `json:"id"`
	AppID          xid.ID  `json:"appId"`
	OrganizationID xid.ID  `json:"organizationId"`
	ConfigID       *xid.ID `json:"configId"` // Reference to AlertConfig

	// Alert details
	Type     AlertType     `json:"type"`
	Severity AlertSeverity `json:"severity"`
	Status   AlertStatus   `json:"status"`
	Title    string        `json:"title"`
	Message  string        `json:"message"`

	// Context data
	MetricKey      string  `json:"metricKey"`
	CurrentValue   float64 `json:"currentValue"`
	ThresholdValue float64 `json:"thresholdValue"`
	LimitValue     float64 `json:"limitValue"`

	// Related entities
	SubscriptionID *xid.ID `json:"subscriptionId"`
	InvoiceID      *xid.ID `json:"invoiceId"`

	// Delivery tracking
	Channels       []AlertChannel    `json:"channels"`
	SentAt         *time.Time        `json:"sentAt"`
	DeliveryStatus map[string]string `json:"deliveryStatus"` // Channel -> status

	// Acknowledgment
	AcknowledgedAt *time.Time `json:"acknowledgedAt"`
	AcknowledgedBy string     `json:"acknowledgedBy"`

	// Resolution
	ResolvedAt *time.Time `json:"resolvedAt"`
	Resolution string     `json:"resolution"`

	// Metadata
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// AlertTemplate represents an alert message template
type AlertTemplate struct {
	ID        xid.ID       `json:"id"`
	AppID     xid.ID       `json:"appId"`
	AlertType AlertType    `json:"alertType"`
	Channel   AlertChannel `json:"channel"`

	// Template content
	Subject       string `json:"subject"` // For email
	TitleTemplate string `json:"titleTemplate"`
	BodyTemplate  string `json:"bodyTemplate"`

	// Template variables available:
	// {{.OrganizationName}}, {{.MetricKey}}, {{.CurrentValue}}, {{.ThresholdValue}}
	// {{.LimitValue}}, {{.PercentUsed}}, {{.DaysRemaining}}, {{.InvoiceAmount}}

	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UsageSnapshot represents a point-in-time usage snapshot
type UsageSnapshot struct {
	MetricKey    string    `json:"metricKey"`
	CurrentValue int64     `json:"currentValue"`
	Limit        int64     `json:"limit"`
	PercentUsed  float64   `json:"percentUsed"`
	Timestamp    time.Time `json:"timestamp"`
}

// CreateAlertConfigRequest is used to create an alert configuration
type CreateAlertConfigRequest struct {
	OrganizationID   xid.ID         `json:"organizationId" validate:"required"`
	AlertType        AlertType      `json:"alertType" validate:"required"`
	ThresholdPercent float64        `json:"thresholdPercent"`
	MetricKey        string         `json:"metricKey"`
	DaysBeforeEnd    int            `json:"daysBeforeEnd"`
	Channels         []AlertChannel `json:"channels" validate:"required,min=1"`
	Recipients       []string       `json:"recipients"`
	WebhookURL       string         `json:"webhookUrl"`
	SlackChannel     string         `json:"slackChannel"`
	MinInterval      int            `json:"minInterval"`
	MaxAlertsPerDay  int            `json:"maxAlertsPerDay"`
}

// UpdateAlertConfigRequest is used to update an alert configuration
type UpdateAlertConfigRequest struct {
	IsEnabled        *bool          `json:"isEnabled"`
	ThresholdPercent *float64       `json:"thresholdPercent"`
	DaysBeforeEnd    *int           `json:"daysBeforeEnd"`
	Channels         []AlertChannel `json:"channels"`
	Recipients       []string       `json:"recipients"`
	WebhookURL       *string        `json:"webhookUrl"`
	SlackChannel     *string        `json:"slackChannel"`
	MinInterval      *int           `json:"minInterval"`
	MaxAlertsPerDay  *int           `json:"maxAlertsPerDay"`
}

// TriggerAlertRequest is used to manually trigger an alert
type TriggerAlertRequest struct {
	OrganizationID xid.ID                 `json:"organizationId" validate:"required"`
	Type           AlertType              `json:"type" validate:"required"`
	Severity       AlertSeverity          `json:"severity"`
	Title          string                 `json:"title" validate:"required"`
	Message        string                 `json:"message" validate:"required"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// AcknowledgeAlertRequest is used to acknowledge an alert
type AcknowledgeAlertRequest struct {
	AlertID        xid.ID `json:"alertId" validate:"required"`
	AcknowledgedBy string `json:"acknowledgedBy"`
}

// SnoozeAlertRequest is used to snooze an alert
type SnoozeAlertRequest struct {
	AlertID     xid.ID    `json:"alertId" validate:"required"`
	SnoozeUntil time.Time `json:"snoozeUntil" validate:"required"`
}

// ResolveAlertRequest is used to resolve an alert
type ResolveAlertRequest struct {
	AlertID    xid.ID `json:"alertId" validate:"required"`
	Resolution string `json:"resolution"`
}

// AlertSummary provides a summary of alerts for an organization
type AlertSummary struct {
	TotalAlerts        int                   `json:"totalAlerts"`
	PendingAlerts      int                   `json:"pendingAlerts"`
	SentAlerts         int                   `json:"sentAlerts"`
	AcknowledgedAlerts int                   `json:"acknowledgedAlerts"`
	BySeverity         map[AlertSeverity]int `json:"bySeverity"`
	ByType             map[AlertType]int     `json:"byType"`
	RecentAlerts       []Alert               `json:"recentAlerts"`
}

// DefaultAlertConfigs returns default alert configurations for a new organization
func DefaultAlertConfigs(appID, orgID xid.ID) []AlertConfig {
	return []AlertConfig{
		{
			AppID:            appID,
			OrganizationID:   orgID,
			AlertType:        AlertTypeUsageThreshold,
			IsEnabled:        true,
			ThresholdPercent: 80,
			MetricKey:        FeatureKeyMaxMembers,
			Channels:         []AlertChannel{AlertChannelEmail, AlertChannelInApp},
			MinInterval:      60, // 1 hour
			MaxAlertsPerDay:  3,
		},
		{
			AppID:            appID,
			OrganizationID:   orgID,
			AlertType:        AlertTypeUsageThreshold,
			IsEnabled:        true,
			ThresholdPercent: 100,
			MetricKey:        FeatureKeyMaxMembers,
			Channels:         []AlertChannel{AlertChannelEmail, AlertChannelInApp},
			MinInterval:      30,
			MaxAlertsPerDay:  5,
		},
		{
			AppID:           appID,
			OrganizationID:  orgID,
			AlertType:       AlertTypeTrialEnding,
			IsEnabled:       true,
			DaysBeforeEnd:   3,
			Channels:        []AlertChannel{AlertChannelEmail, AlertChannelInApp},
			MinInterval:     1440, // 24 hours
			MaxAlertsPerDay: 1,
		},
		{
			AppID:           appID,
			OrganizationID:  orgID,
			AlertType:       AlertTypePaymentFailed,
			IsEnabled:       true,
			Channels:        []AlertChannel{AlertChannelEmail, AlertChannelInApp},
			MinInterval:     60,
			MaxAlertsPerDay: 3,
		},
	}
}

// DefaultAlertTemplates returns default alert templates
func DefaultAlertTemplates(appID xid.ID) []AlertTemplate {
	return []AlertTemplate{
		{
			AppID:         appID,
			AlertType:     AlertTypeUsageThreshold,
			Channel:       AlertChannelEmail,
			Subject:       "Usage Alert: {{.MetricKey}} at {{.PercentUsed}}%",
			TitleTemplate: "Usage threshold reached",
			BodyTemplate:  "Your {{.MetricKey}} usage has reached {{.PercentUsed}}% of your limit ({{.CurrentValue}}/{{.LimitValue}}). Consider upgrading your plan to avoid disruption.",
			IsDefault:     true,
		},
		{
			AppID:         appID,
			AlertType:     AlertTypeTrialEnding,
			Channel:       AlertChannelEmail,
			Subject:       "Your trial ends in {{.DaysRemaining}} days",
			TitleTemplate: "Trial ending soon",
			BodyTemplate:  "Your trial period for {{.OrganizationName}} will end in {{.DaysRemaining}} days. Subscribe now to continue using all features.",
			IsDefault:     true,
		},
		{
			AppID:         appID,
			AlertType:     AlertTypePaymentFailed,
			Channel:       AlertChannelEmail,
			Subject:       "Payment failed for {{.OrganizationName}}",
			TitleTemplate: "Payment failed",
			BodyTemplate:  "We were unable to process your payment of {{.InvoiceAmount}}. Please update your payment method to avoid service interruption.",
			IsDefault:     true,
		},
	}
}
