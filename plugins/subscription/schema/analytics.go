package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SubscriptionBillingMetric represents a billing metric in the database
type SubscriptionBillingMetric struct {
	bun.BaseModel `bun:"table:subscription_billing_metrics,alias:sbm"`

	ID        xid.ID    `bun:"id,pk,type:char(20)"`
	AppID     xid.ID    `bun:"app_id,notnull,type:char(20)"`
	Type      string    `bun:"type,notnull"` // mrr, arr, churn_rate, etc.
	Period    string    `bun:"period,notnull"` // daily, weekly, monthly, yearly
	Value     float64   `bun:"value,notnull"`
	Currency  string    `bun:"currency,notnull"`
	Date      time.Time `bun:"date,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// SubscriptionMovement represents a subscription movement event in the database
type SubscriptionMovement struct {
	bun.BaseModel `bun:"table:subscription_movements,alias:sm"`

	ID             xid.ID    `bun:"id,pk,type:char(20)"`
	AppID          xid.ID    `bun:"app_id,notnull,type:char(20)"`
	SubscriptionID xid.ID    `bun:"subscription_id,notnull,type:char(20)"`
	OrganizationID xid.ID    `bun:"organization_id,notnull,type:char(20)"`
	MovementType   string    `bun:"movement_type,notnull"` // new, upgrade, downgrade, churn, reactivation
	PreviousMRR    int64     `bun:"previous_mrr,notnull,default:0"`
	NewMRR         int64     `bun:"new_mrr,notnull,default:0"`
	MRRChange      int64     `bun:"mrr_change,notnull,default:0"`
	Currency       string    `bun:"currency,notnull"`
	PreviousPlanID *xid.ID   `bun:"previous_plan_id,type:char(20)"`
	NewPlanID      *xid.ID   `bun:"new_plan_id,type:char(20)"`
	Reason         string    `bun:"reason"`
	Notes          string    `bun:"notes"`
	OccurredAt     time.Time `bun:"occurred_at,notnull"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp"`

	// Relations
	Subscription *Subscription     `bun:"rel:belongs-to,join:subscription_id=id"`
	PreviousPlan *SubscriptionPlan `bun:"rel:belongs-to,join:previous_plan_id=id"`
	NewPlan      *SubscriptionPlan `bun:"rel:belongs-to,join:new_plan_id=id"`
}

// SubscriptionCohort represents cohort data in the database
type SubscriptionCohort struct {
	bun.BaseModel `bun:"table:subscription_cohorts,alias:sco"`

	ID             xid.ID    `bun:"id,pk,type:char(20)"`
	AppID          xid.ID    `bun:"app_id,notnull,type:char(20)"`
	CohortMonth    time.Time `bun:"cohort_month,notnull"`
	TotalCustomers int       `bun:"total_customers,notnull"`
	MonthNumber    int       `bun:"month_number,notnull"` // 0 = signup month, 1 = first month after, etc.
	ActiveCustomers int      `bun:"active_customers,notnull"`
	RetentionRate  float64   `bun:"retention_rate,notnull"`
	Revenue        int64     `bun:"revenue,notnull"`
	Currency       string    `bun:"currency,notnull"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

