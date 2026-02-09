package core

import (
	"time"

	"github.com/rs/xid"
)

// UsageRecord represents a metered usage record.
type UsageRecord struct {
	ID               xid.ID         `json:"id"`
	SubscriptionID   xid.ID         `json:"subscriptionId"`   // Related subscription
	OrganizationID   xid.ID         `json:"organizationId"`   // Organization for quick queries
	MetricKey        string         `json:"metricKey"`        // Usage metric identifier (e.g., "api_calls", "storage_gb")
	Quantity         int64          `json:"quantity"`         // Usage amount
	Action           UsageAction    `json:"action"`           // set, increment, decrement
	Timestamp        time.Time      `json:"timestamp"`        // When the usage occurred
	IdempotencyKey   string         `json:"idempotencyKey"`   // For deduplication
	Metadata         map[string]any `json:"metadata"`         // Additional context
	ProviderRecordID string         `json:"providerRecordId"` // Stripe Usage Record ID
	Reported         bool           `json:"reported"`         // Has been reported to provider
	ReportedAt       *time.Time     `json:"reportedAt"`       // When reported to provider
	CreatedAt        time.Time      `json:"createdAt"`
}

// UsageAction defines the type of usage update.
type UsageAction string

const (
	// UsageActionSet sets the usage to an absolute value.
	UsageActionSet UsageAction = "set"
	// UsageActionIncrement adds to the current usage.
	UsageActionIncrement UsageAction = "increment"
	// UsageActionDecrement subtracts from the current usage.
	UsageActionDecrement UsageAction = "decrement"
)

// String returns the string representation of usage action.
func (u UsageAction) String() string {
	return string(u)
}

// IsValid checks if the usage action is valid.
func (u UsageAction) IsValid() bool {
	switch u {
	case UsageActionSet, UsageActionIncrement, UsageActionDecrement:
		return true
	}

	return false
}

// UsageSummary provides aggregated usage data for a metric.
type UsageSummary struct {
	MetricKey     string    `json:"metricKey"`
	TotalQuantity int64     `json:"totalQuantity"`
	PeriodStart   time.Time `json:"periodStart"`
	PeriodEnd     time.Time `json:"periodEnd"`
	RecordCount   int64     `json:"recordCount"`
	FirstRecordAt time.Time `json:"firstRecordAt"`
	LastRecordAt  time.Time `json:"lastRecordAt"`
}

// UsageMetric defines a usage metric for metered billing.
type UsageMetric struct {
	Key           string `json:"key"`           // Metric identifier
	Name          string `json:"name"`          // Display name
	Description   string `json:"description"`   // Description
	Unit          string `json:"unit"`          // Unit of measurement (e.g., "calls", "GB")
	AggregateType string `json:"aggregateType"` // "sum", "max", "last_during_period"
}

// Common usage metric keys.
const (
	UsageMetricAPICalls    = "api_calls"
	UsageMetricStorageGB   = "storage_gb"
	UsageMetricBandwidthGB = "bandwidth_gb"
	UsageMetricActiveUsers = "active_users"
	UsageMetricMessages    = "messages"
	UsageMetricCompute     = "compute_hours"
)

// NewUsageRecord creates a new UsageRecord.
func NewUsageRecord(subID, orgID xid.ID, metricKey string, quantity int64, action UsageAction) *UsageRecord {
	now := time.Now()

	return &UsageRecord{
		ID:             xid.New(),
		SubscriptionID: subID,
		OrganizationID: orgID,
		MetricKey:      metricKey,
		Quantity:       quantity,
		Action:         action,
		Timestamp:      now,
		Metadata:       make(map[string]any),
		Reported:       false,
		CreatedAt:      now,
	}
}

// RecordUsageRequest represents a request to record usage.
type RecordUsageRequest struct {
	SubscriptionID xid.ID         `json:"subscriptionId" validate:"required"`
	MetricKey      string         `json:"metricKey"      validate:"required,min=1,max=50"`
	Quantity       int64          `json:"quantity"       validate:"required,min=0"`
	Action         UsageAction    `json:"action"         validate:"required"`
	Timestamp      *time.Time     `json:"timestamp"` // Optional, defaults to now
	IdempotencyKey string         `json:"idempotencyKey" validate:"max=100"`
	Metadata       map[string]any `json:"metadata"`
}

// GetUsageSummaryRequest represents a request to get usage summary.
type GetUsageSummaryRequest struct {
	SubscriptionID xid.ID    `json:"subscriptionId" validate:"required"`
	MetricKey      string    `json:"metricKey"      validate:"required"`
	PeriodStart    time.Time `json:"periodStart"    validate:"required"`
	PeriodEnd      time.Time `json:"periodEnd"      validate:"required"`
}

// ListUsageRecordsFilter defines filters for listing usage records.
type ListUsageRecordsFilter struct {
	SubscriptionID *xid.ID    `json:"subscriptionId"`
	OrganizationID *xid.ID    `json:"organizationId"`
	MetricKey      string     `json:"metricKey"`
	FromDate       *time.Time `json:"fromDate"`
	ToDate         *time.Time `json:"toDate"`
	Reported       *bool      `json:"reported"`
	Page           int        `json:"page"`
	PageSize       int        `json:"pageSize"`
}

// UsageLimit tracks current usage against limits.
type UsageLimit struct {
	MetricKey      string  `json:"metricKey"`
	CurrentUsage   int64   `json:"currentUsage"`
	Limit          int64   `json:"limit"`          // -1 for unlimited
	RemainingUsage int64   `json:"remainingUsage"` // -1 for unlimited
	IsExceeded     bool    `json:"isExceeded"`
	PercentUsed    float64 `json:"percentUsed"`
}

// NewUsageLimit creates a UsageLimit with calculated values.
func NewUsageLimit(metricKey string, current, limit int64) *UsageLimit {
	ul := &UsageLimit{
		MetricKey:    metricKey,
		CurrentUsage: current,
		Limit:        limit,
	}

	if limit == -1 {
		// Unlimited
		ul.RemainingUsage = -1
		ul.IsExceeded = false
		ul.PercentUsed = 0
	} else {
		ul.RemainingUsage = max(limit-current, 0)

		ul.IsExceeded = current > limit
		if limit > 0 {
			ul.PercentUsed = float64(current) / float64(limit) * 100
		}
	}

	return ul
}
