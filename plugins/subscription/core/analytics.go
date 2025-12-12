package core

import (
	"time"

	"github.com/rs/xid"
)

// MetricPeriod represents the time period for metrics
type MetricPeriod string

const (
	MetricPeriodDaily   MetricPeriod = "daily"
	MetricPeriodWeekly  MetricPeriod = "weekly"
	MetricPeriodMonthly MetricPeriod = "monthly"
	MetricPeriodYearly  MetricPeriod = "yearly"
)

// MetricType represents the type of billing metric
type MetricType string

const (
	MetricTypeMRR             MetricType = "mrr"              // Monthly Recurring Revenue
	MetricTypeARR             MetricType = "arr"              // Annual Recurring Revenue
	MetricTypeChurnRate       MetricType = "churn_rate"       // Customer churn rate
	MetricTypeRevenueChurn    MetricType = "revenue_churn"    // Revenue churn rate
	MetricTypeExpansionMRR    MetricType = "expansion_mrr"    // Expansion MRR (upgrades)
	MetricTypeContractionMRR  MetricType = "contraction_mrr"  // Contraction MRR (downgrades)
	MetricTypeNewMRR          MetricType = "new_mrr"          // New customer MRR
	MetricTypeLTV             MetricType = "ltv"              // Customer Lifetime Value
	MetricTypeCAC             MetricType = "cac"              // Customer Acquisition Cost
	MetricTypeARPU            MetricType = "arpu"             // Average Revenue Per User
	MetricTypeTrialConversion MetricType = "trial_conversion" // Trial to paid conversion rate
	MetricTypeNetRevenue      MetricType = "net_revenue"      // Net revenue retention
)

// BillingMetric represents a point-in-time billing metric
type BillingMetric struct {
	ID        xid.ID       `json:"id"`
	AppID     xid.ID       `json:"appId"`
	Type      MetricType   `json:"type"`
	Period    MetricPeriod `json:"period"`
	Value     float64      `json:"value"`
	Currency  string       `json:"currency"`
	Date      time.Time    `json:"date"` // The date this metric is for
	CreatedAt time.Time    `json:"createdAt"`
}

// MRRBreakdown provides a detailed breakdown of MRR
type MRRBreakdown struct {
	Date            time.Time `json:"date"`
	Currency        string    `json:"currency"`
	TotalMRR        int64     `json:"totalMrr"`
	NewMRR          int64     `json:"newMrr"`          // From new customers
	ExpansionMRR    int64     `json:"expansionMrr"`    // From upgrades
	ContractionMRR  int64     `json:"contractionMrr"`  // From downgrades (negative)
	ChurnedMRR      int64     `json:"churnedMrr"`      // Lost from cancellations (negative)
	ReactivationMRR int64     `json:"reactivationMrr"` // From reactivated customers
	NetNewMRR       int64     `json:"netNewMrr"`       // Total change
}

// ChurnAnalysis provides detailed churn analysis
type ChurnAnalysis struct {
	Period    MetricPeriod `json:"period"`
	StartDate time.Time    `json:"startDate"`
	EndDate   time.Time    `json:"endDate"`

	// Customer churn
	CustomersAtStart  int     `json:"customersAtStart"`
	CustomersAtEnd    int     `json:"customersAtEnd"`
	CustomersChurned  int     `json:"customersChurned"`
	CustomerChurnRate float64 `json:"customerChurnRate"` // Percentage

	// Revenue churn
	MRRAtStart          int64   `json:"mrrAtStart"`
	MRRAtEnd            int64   `json:"mrrAtEnd"`
	MRRChurned          int64   `json:"mrrChurned"`
	RevenueChurnRate    float64 `json:"revenueChurnRate"`    // Percentage
	NetRevenueRetention float64 `json:"netRevenueRetention"` // Percentage (can be >100%)

	// Breakdown by reason
	ChurnByReason map[string]int `json:"churnByReason"` // Reason -> count
	ChurnByPlan   map[string]int `json:"churnByPlan"`   // Plan slug -> count

	Currency string `json:"currency"`
}

// CohortAnalysis represents a cohort analysis for retention
type CohortAnalysis struct {
	CohortMonth    time.Time `json:"cohortMonth"` // The month customers signed up
	TotalCustomers int       `json:"totalCustomers"`
	Retention      []float64 `json:"retention"` // Retention rate for each subsequent month
	Revenue        []int64   `json:"revenue"`   // Revenue for each subsequent month
	Currency       string    `json:"currency"`
}

// RevenueByPlan shows revenue breakdown by plan
type RevenueByPlan struct {
	PlanID            xid.ID  `json:"planId"`
	PlanName          string  `json:"planName"`
	PlanSlug          string  `json:"planSlug"`
	ActiveSubscribers int     `json:"activeSubscribers"`
	MRR               int64   `json:"mrr"`
	ARR               int64   `json:"arr"`
	PercentOfTotal    float64 `json:"percentOfTotal"`
	Currency          string  `json:"currency"`
}

// RevenueBySegment shows revenue by customer segment
type RevenueBySegment struct {
	Segment     string  `json:"segment"`
	Subscribers int     `json:"subscribers"`
	MRR         int64   `json:"mrr"`
	ARPU        int64   `json:"arpu"`
	ChurnRate   float64 `json:"churnRate"`
	LTV         int64   `json:"ltv"`
	Currency    string  `json:"currency"`
}

// SubscriptionMovement tracks subscription changes
type SubscriptionMovement struct {
	ID             xid.ID `json:"id"`
	AppID          xid.ID `json:"appId"`
	SubscriptionID xid.ID `json:"subscriptionId"`
	OrganizationID xid.ID `json:"organizationId"`
	MovementType   string `json:"movementType"` // new, upgrade, downgrade, churn, reactivation

	// Revenue impact
	PreviousMRR int64  `json:"previousMrr"`
	NewMRR      int64  `json:"newMrr"`
	MRRChange   int64  `json:"mrrChange"`
	Currency    string `json:"currency"`

	// Plan details
	PreviousPlanID *xid.ID `json:"previousPlanId"`
	NewPlanID      *xid.ID `json:"newPlanId"`

	// Metadata
	Reason     string    `json:"reason"`
	Notes      string    `json:"notes"`
	OccurredAt time.Time `json:"occurredAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TrialMetrics provides trial conversion metrics
type TrialMetrics struct {
	Period             MetricPeriod `json:"period"`
	TrialsStarted      int          `json:"trialsStarted"`
	TrialsConverted    int          `json:"trialsConverted"`
	TrialsExpired      int          `json:"trialsExpired"`
	TrialsActive       int          `json:"trialsActive"`
	ConversionRate     float64      `json:"conversionRate"`
	AverageTrialLength float64      `json:"averageTrialLength"` // In days

	// Conversion by plan
	ConversionByPlan map[string]float64 `json:"conversionByPlan"` // Plan slug -> rate
}

// InvoiceMetrics provides invoice and payment metrics
type InvoiceMetrics struct {
	Period               MetricPeriod `json:"period"`
	TotalInvoiced        int64        `json:"totalInvoiced"`
	TotalCollected       int64        `json:"totalCollected"`
	TotalOutstanding     int64        `json:"totalOutstanding"`
	TotalOverdue         int64        `json:"totalOverdue"`
	CollectionRate       float64      `json:"collectionRate"`       // Percentage
	AverageTimeToPayment float64      `json:"averageTimeToPayment"` // In days
	Currency             string       `json:"currency"`
}

// DashboardMetrics provides overview metrics for the dashboard
type DashboardMetrics struct {
	// Current state
	TotalMRR              int64 `json:"totalMrr"`
	TotalARR              int64 `json:"totalArr"`
	ActiveSubscriptions   int   `json:"activeSubscriptions"`
	TrialingSubscriptions int   `json:"trialingSubscriptions"`

	// Changes
	MRRGrowth          float64 `json:"mrrGrowth"` // Percentage change
	SubscriptionGrowth float64 `json:"subscriptionGrowth"`

	// Health indicators
	ChurnRate           float64 `json:"churnRate"`
	TrialConversionRate float64 `json:"trialConversionRate"`
	NetRevenueRetention float64 `json:"netRevenueRetention"`

	// Revenue breakdown
	NewMRR       int64 `json:"newMrr"`
	ExpansionMRR int64 `json:"expansionMrr"`
	ChurnedMRR   int64 `json:"churnedMrr"`

	Currency string    `json:"currency"`
	AsOf     time.Time `json:"asOf"`
}

// GetMetricsRequest is used to request billing metrics
type GetMetricsRequest struct {
	MetricTypes []MetricType `json:"metricTypes"`
	Period      MetricPeriod `json:"period"`
	StartDate   time.Time    `json:"startDate"`
	EndDate     time.Time    `json:"endDate"`
	Currency    string       `json:"currency"`
	GroupBy     string       `json:"groupBy"` // plan, segment, etc.
}

// MetricsResponse contains the metrics response
type MetricsResponse struct {
	Metrics    []BillingMetric   `json:"metrics"`
	MRRHistory []MRRBreakdown    `json:"mrrHistory,omitempty"`
	Churn      *ChurnAnalysis    `json:"churn,omitempty"`
	Dashboard  *DashboardMetrics `json:"dashboard,omitempty"`
	StartDate  time.Time         `json:"startDate"`
	EndDate    time.Time         `json:"endDate"`
	Currency   string            `json:"currency"`
}

// GetCohortAnalysisRequest is used to request cohort analysis
type GetCohortAnalysisRequest struct {
	StartMonth time.Time `json:"startMonth"`
	NumMonths  int       `json:"numMonths"`
	Currency   string    `json:"currency"`
}

// CohortAnalysisResponse contains cohort analysis results
type CohortAnalysisResponse struct {
	Cohorts  []CohortAnalysis `json:"cohorts"`
	Currency string           `json:"currency"`
}

// ExportMetricsRequest is used to export metrics
type ExportMetricsRequest struct {
	MetricTypes []MetricType `json:"metricTypes"`
	Period      MetricPeriod `json:"period"`
	StartDate   time.Time    `json:"startDate"`
	EndDate     time.Time    `json:"endDate"`
	Format      string       `json:"format"` // csv, json, xlsx
}

// CalculateMRR calculates MRR from a subscription
func CalculateMRR(subscription *Subscription, plan *Plan) int64 {
	if subscription == nil || plan == nil {
		return 0
	}

	baseAmount := plan.BasePrice
	quantity := subscription.Quantity
	if quantity == 0 {
		quantity = 1
	}

	// Normalize to monthly
	switch plan.BillingInterval {
	case BillingIntervalYearly:
		return (baseAmount * int64(quantity)) / 12
	case BillingIntervalOneTime:
		return 0 // One-time charges don't contribute to MRR
	default: // Monthly
		return baseAmount * int64(quantity)
	}
}

// CalculateARR calculates ARR from MRR
func CalculateARR(mrr int64) int64 {
	return mrr * 12
}

// CalculateChurnRate calculates the churn rate
func CalculateChurnRate(customersChurned, customersAtStart int) float64 {
	if customersAtStart == 0 {
		return 0
	}
	return float64(customersChurned) / float64(customersAtStart) * 100
}

// CalculateNetRevenueRetention calculates NRR
func CalculateNetRevenueRetention(mrrAtStart, newMRR, expansionMRR, contractionMRR, churnedMRR int64) float64 {
	if mrrAtStart == 0 {
		return 100
	}
	endingMRRFromExisting := mrrAtStart + expansionMRR - contractionMRR - churnedMRR
	return float64(endingMRRFromExisting) / float64(mrrAtStart) * 100
}

// CalculateARPU calculates Average Revenue Per User
func CalculateARPU(totalMRR int64, totalCustomers int) int64 {
	if totalCustomers == 0 {
		return 0
	}
	return totalMRR / int64(totalCustomers)
}

// CalculateLTV calculates Customer Lifetime Value
func CalculateLTV(arpu int64, churnRate float64) int64 {
	if churnRate == 0 {
		// If no churn, cap at some reasonable multiple
		return arpu * 60 // 5 years
	}
	monthsUntilChurn := 100 / churnRate
	return int64(float64(arpu) * monthsUntilChurn)
}

// OrgUsageStats represents usage statistics per organization
type OrgUsageStats struct {
	OrgID        xid.ID      `json:"orgId"`
	OrgName      string      `json:"orgName"`
	FeatureID    xid.ID      `json:"featureId"`
	FeatureName  string      `json:"featureName"`
	FeatureType  FeatureType `json:"featureType"`
	Unit         string      `json:"unit"`
	Usage        int64       `json:"usage"`
	Limit        int64       `json:"limit"`
	PercentUsed  float64     `json:"percentUsed"`
}

// UsageTrend represents usage trends over time
type UsageTrend struct {
	Date   time.Time `json:"date"`
	Usage  int64     `json:"usage"`
	Action string    `json:"action"`
}

// CurrentUsage represents current feature usage state
type CurrentUsage struct {
	OrganizationID   xid.ID `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	FeatureID        xid.ID `json:"featureId"`
	FeatureName      string `json:"featureName"`
	FeatureType      string `json:"featureType"`
	Unit             string `json:"unit"`
	CurrentUsage     int64  `json:"currentUsage"`
	Limit            int64  `json:"limit"`
	PercentUsed      float64 `json:"percentUsed"`
}

// OrgRevenue represents revenue per organization
type OrgRevenue struct {
	OrgID    xid.ID `json:"orgId"`
	OrgName  string `json:"orgName"`
	MRR      int64  `json:"mrr"`
	ARR      int64  `json:"arr"`
	PlanID   xid.ID `json:"planId"`
	PlanName string `json:"planName"`
	Status   string `json:"status"`
}

// GrowthPoint represents subscription growth at a point in time
type GrowthPoint struct {
	Date        time.Time `json:"date"`
	NewSubs     int       `json:"newSubs"`
	ChurnedSubs int       `json:"churnedSubs"`
	ActiveSubs  int       `json:"activeSubs"`
	NetGrowth   int       `json:"netGrowth"`
}

// UsageStats aggregates usage statistics
type UsageStats struct {
	TotalUsage  int64   `json:"totalUsage"`
	TotalOrgs   int     `json:"totalOrgs"`
	AverageUsage float64 `json:"averageUsage"`
}
