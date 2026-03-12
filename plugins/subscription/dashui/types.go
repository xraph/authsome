package dashui

import "time"

// ──────────────────────────────────────────────────
// Plan view types
// ──────────────────────────────────────────────────

// PlanView is a display-only view of a billing plan.
type PlanView struct {
	ID            string
	Name          string
	Slug          string
	Description   string
	Currency      string
	Status        string
	TrialDays     int
	BaseAmount    string
	BillingPeriod string
	FeaturesCount int
	IsAddon       bool
	Metadata      map[string]string
}

// PlanDetailView extends PlanView with feature and pricing data.
type PlanDetailView struct {
	PlanView
	Features        []FeatureView
	Tiers           []TierView
	SubscriberCount int
}

// FeatureView is a display-only view of a plan feature.
type FeatureView struct {
	ID        string
	CatalogID string // links to catalog feature (if created from catalog)
	Key       string
	Name      string
	Type      string
	Limit     int64
	Period    string
	SoftLimit bool
}

// TierView is a display-only view of a pricing tier.
type TierView struct {
	FeatureKey string
	Type       string
	UpTo       int64
	UnitAmount string
	FlatAmount string
}

// ──────────────────────────────────────────────────
// Subscription view types
// ──────────────────────────────────────────────────

// SubscriptionView is a display-only view of a subscription.
type SubscriptionView struct {
	ID                 string
	TenantID           string
	PlanID             string
	PlanName           string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	TrialStart         *time.Time
	TrialEnd           *time.Time
	CanceledAt         *time.Time
	CancelAt           *time.Time
	EndedAt            *time.Time
	AppID              string
	ProviderName       string
	Metadata           map[string]string
}

// ──────────────────────────────────────────────────
// Invoice view types
// ──────────────────────────────────────────────────

// InvoiceView is a display-only view of an invoice.
type InvoiceView struct {
	ID             string
	TenantID       string
	SubscriptionID string
	Status         string
	Currency       string
	Subtotal       string
	TaxAmount      string
	DiscountAmount string
	Total          string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	DueDate        *time.Time
	PaidAt         *time.Time
	PaymentRef     string
	VoidReason     string
}

// LineItemView is a display-only view of an invoice line item.
type LineItemView struct {
	Description string
	Type        string
	FeatureKey  string
	Quantity    int64
	UnitAmount  string
	Amount      string
}

// ──────────────────────────────────────────────────
// Coupon view types
// ──────────────────────────────────────────────────

// CouponView is a display-only view of a coupon.
type CouponView struct {
	ID             string
	Code           string
	Name           string
	Type           string // "percentage" or "amount"
	Amount         string // formatted for display
	Percentage     int
	Currency       string
	MaxRedemptions int
	TimesRedeemed  int
	ValidFrom      *time.Time
	ValidUntil     *time.Time
	IsExpired      bool
	IsExhausted    bool
}

// ──────────────────────────────────────────────────
// Usage view types
// ──────────────────────────────────────────────────

// UsageView displays feature usage vs limits.
type UsageView struct {
	FeatureKey  string
	FeatureName string
	FeatureType string
	Used        int64
	Limit       int64
	Remaining   int64
	Period      string
	Percentage  int // 0-100
}

// ──────────────────────────────────────────────────
// Page data types
// ──────────────────────────────────────────────────

// PlansPageData holds all data for the plans management page.
type PlansPageData struct {
	Plans         []PlanView
	Addons        []PlanView
	ActiveCount   int
	ArchivedCount int
	TotalFeatures int
	Error         string
	Success       string
	FormNonce     string
	Tab           string // "plans" or "addons"
}

// PlanDetailPageData holds data for the plan detail page.
type PlanDetailPageData struct {
	Plan            PlanDetailView
	CatalogFeatures []CatalogFeatureView // for quick-add from catalog
	TierForm        TierFormView         // for add-tier dropdown
	Error           string
	Success         string
	FormNonce       string
}

// SubscriptionsPageData holds all data for the subscriptions page.
type SubscriptionsPageData struct {
	Subscriptions []SubscriptionView
	Plans         []PlanView // for create/change-plan dropdown
	ActiveCount   int
	TrialCount    int
	PastDueCount  int
	CanceledCount int
	Error         string
	Success       string
	FormNonce     string
	StatusFilter  string
}

// SubscriptionDetailPageData holds data for subscription detail page.
type SubscriptionDetailPageData struct {
	Subscription SubscriptionView
	Plan         PlanDetailView
	Usage        []UsageView
	Invoices     []InvoiceView
	Plans        []PlanView // for change-plan dropdown
	Error        string
	Success      string
	FormNonce    string
	ActiveTab    string // "overview", "usage", "invoices"
}

// InvoicesPageData holds all data for the invoices page.
type InvoicesPageData struct {
	Invoices     []InvoiceView
	PaidCount    int
	PendingCount int
	OverdueCount int
	Error        string
	StatusFilter string
}

// InvoiceDetailPageData holds data for the invoice detail page.
type InvoiceDetailPageData struct {
	Invoice   InvoiceView
	LineItems []LineItemView
	Error     string
	Success   string
	FormNonce string
}

// CouponsPageData holds data for the coupons management page.
type CouponsPageData struct {
	Coupons   []CouponView
	Error     string
	Success   string
	FormNonce string
}

// OverviewWidgetData holds data for the subscription overview widget.
type OverviewWidgetData struct {
	ActiveCount  int
	TrialCount   int
	PastDueCount int
	PlanCount    int
	MRR          string // Monthly recurring revenue
}

// SettingsPanelData holds data for the settings panel.
type SettingsPanelData struct {
	DefaultPlan      string
	TenantMode       string
	AutoSubscribeOrg bool
	AutoSubscribeUsr bool
	TrialDays        int
	SelfService      bool
	GracePeriodDays  int
}

// ──────────────────────────────────────────────────
// Feature catalog view types
// ──────────────────────────────────────────────────

// CatalogFeatureView is a display-only view of a catalog feature.
type CatalogFeatureView struct {
	ID           string
	Key          string
	Name         string
	Description  string
	Type         string // "metered", "boolean", "seat"
	DefaultLimit int64
	Period       string
	SoftLimit    bool
	Status       string // "active", "draft", "archived"
}

// CatalogFeaturesPageData holds data for the feature catalog page.
type CatalogFeaturesPageData struct {
	Features      []CatalogFeatureView
	ActiveCount   int
	DraftCount    int
	ArchivedCount int
	Error         string
	Success       string
	FormNonce     string
}

// TierFormView holds available feature keys for the add-tier form.
type TierFormView struct {
	FeatureKeys []string // from plan.Features
}

// ──────────────────────────────────────────────────
// Compact section views
// ──────────────────────────────────────────────────

// ActiveSubView is a compact view for user/org detail sections.
type ActiveSubView struct {
	PlanName  string
	Status    string
	PeriodEnd time.Time
	HasSub    bool
}

// OrgBillingTabData holds all billing data for the org detail tab.
type OrgBillingTabData struct {
	Subscription *SubscriptionView
	Plan         *PlanView
	Usage        []UsageView
	Invoices     []InvoiceView
	HasSub       bool
}
