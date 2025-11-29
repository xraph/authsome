// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// SubscriptionPlan represents a subscription plan in the database
type SubscriptionPlan struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_plans,alias:sp"`

	ID              xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID           xid.ID `json:"appId" bun:"app_id,notnull,type:varchar(20)"`
	Name            string `json:"name" bun:"name,notnull"`
	Slug            string `json:"slug" bun:"slug,notnull"` // Unique within app
	Description     string `json:"description" bun:"description"`
	BillingPattern  string `json:"billingPattern" bun:"billing_pattern,notnull"`
	BillingInterval string `json:"billingInterval" bun:"billing_interval,notnull"`
	BasePrice       int64  `json:"basePrice" bun:"base_price,notnull,default:0"`
	Currency        string `json:"currency" bun:"currency,notnull,default:'USD'"`
	TrialDays       int    `json:"trialDays" bun:"trial_days,notnull,default:0"`
	TierMode        string `json:"tierMode" bun:"tier_mode,default:'graduated'"`
	Metadata        map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	IsActive        bool   `json:"isActive" bun:"is_active,notnull,default:true"`
	IsPublic        bool   `json:"isPublic" bun:"is_public,notnull,default:true"`
	DisplayOrder    int    `json:"displayOrder" bun:"display_order,notnull,default:0"`
	ProviderPlanID  string `json:"providerPlanId" bun:"provider_plan_id"`
	ProviderPriceID string `json:"providerPriceId" bun:"provider_price_id"`

	// Relations
	App          *mainschema.App           `json:"app,omitempty" bun:"rel:belongs-to,join:app_id=id"`
	Features     []SubscriptionPlanFeature `json:"features,omitempty" bun:"rel:has-many,join:id=plan_id"`
	Tiers        []SubscriptionPlanTier    `json:"tiers,omitempty" bun:"rel:has-many,join:id=plan_id"`
	FeatureLinks []PlanFeatureLink         `json:"featureLinks,omitempty" bun:"rel:has-many,join:id=plan_id"`
}

// SubscriptionPlanFeature represents a feature limit on a plan
type SubscriptionPlanFeature struct {
	bun.BaseModel `bun:"table:subscription_plan_features,alias:spf"`

	ID          xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	PlanID      xid.ID `json:"planId" bun:"plan_id,notnull,type:varchar(20)"`
	Key         string `json:"key" bun:"key,notnull"`
	Name        string `json:"name" bun:"name,notnull"`
	Description string `json:"description" bun:"description"`
	Type        string `json:"type" bun:"type,notnull"` // boolean, limit, unlimited
	Value       string `json:"value" bun:"value"`       // JSON-encoded value
	CreatedAt   time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Plan *SubscriptionPlan `json:"plan,omitempty" bun:"rel:belongs-to,join:plan_id=id"`
}

// SubscriptionPlanTier represents a pricing tier for a plan
type SubscriptionPlanTier struct {
	bun.BaseModel `bun:"table:subscription_plan_tiers,alias:spt"`

	ID         xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	PlanID     xid.ID `json:"planId" bun:"plan_id,notnull,type:varchar(20)"`
	TierOrder  int    `json:"tierOrder" bun:"tier_order,notnull"`
	UpTo       int64  `json:"upTo" bun:"up_to,notnull"` // -1 for infinite
	UnitAmount int64  `json:"unitAmount" bun:"unit_amount,notnull"`
	FlatAmount int64  `json:"flatAmount" bun:"flat_amount,notnull,default:0"`
	CreatedAt  time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Plan *SubscriptionPlan `json:"plan,omitempty" bun:"rel:belongs-to,join:plan_id=id"`
}

// Subscription represents an organization's subscription
type Subscription struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscriptions,alias:sub"`

	ID                 xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID     xid.ID     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	PlanID             xid.ID     `json:"planId" bun:"plan_id,notnull,type:varchar(20)"`
	Status             string     `json:"status" bun:"status,notnull"`
	Quantity           int        `json:"quantity" bun:"quantity,notnull,default:1"`
	CurrentPeriodStart time.Time  `json:"currentPeriodStart" bun:"current_period_start,notnull"`
	CurrentPeriodEnd   time.Time  `json:"currentPeriodEnd" bun:"current_period_end,notnull"`
	TrialStart         *time.Time `json:"trialStart" bun:"trial_start"`
	TrialEnd           *time.Time `json:"trialEnd" bun:"trial_end"`
	CancelAt           *time.Time `json:"cancelAt" bun:"cancel_at"`
	CanceledAt         *time.Time `json:"canceledAt" bun:"canceled_at"`
	EndedAt            *time.Time `json:"endedAt" bun:"ended_at"`
	PausedAt           *time.Time `json:"pausedAt" bun:"paused_at"`
	ResumeAt           *time.Time `json:"resumeAt" bun:"resume_at"`
	ProviderSubID      string     `json:"providerSubId" bun:"provider_sub_id"`
	ProviderCustomerID string     `json:"providerCustomerId" bun:"provider_customer_id"`
	Metadata           map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Plan         *SubscriptionPlan        `json:"plan,omitempty" bun:"rel:belongs-to,join:plan_id=id"`
	AddOns       []SubscriptionAddOnItem  `json:"addOns,omitempty" bun:"rel:has-many,join:id=subscription_id"`
}

// SubscriptionAddOn represents an add-on product
type SubscriptionAddOn struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_addons,alias:sao"`

	ID              xid.ID   `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID           xid.ID   `json:"appId" bun:"app_id,notnull,type:varchar(20)"`
	Name            string   `json:"name" bun:"name,notnull"`
	Slug            string   `json:"slug" bun:"slug,notnull"`
	Description     string   `json:"description" bun:"description"`
	BillingPattern  string   `json:"billingPattern" bun:"billing_pattern,notnull"`
	BillingInterval string   `json:"billingInterval" bun:"billing_interval,notnull"`
	Price           int64    `json:"price" bun:"price,notnull,default:0"`
	Currency        string   `json:"currency" bun:"currency,notnull,default:'USD'"`
	TierMode        string   `json:"tierMode" bun:"tier_mode,default:'graduated'"`
	Metadata        map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	IsActive        bool     `json:"isActive" bun:"is_active,notnull,default:true"`
	IsPublic        bool     `json:"isPublic" bun:"is_public,notnull,default:true"`
	DisplayOrder    int      `json:"displayOrder" bun:"display_order,notnull,default:0"`
	RequiresPlanIDs []string `json:"requiresPlanIds" bun:"requires_plan_ids,array,type:varchar(20)[]"`
	ExcludesPlanIDs []string `json:"excludesPlanIds" bun:"excludes_plan_ids,array,type:varchar(20)[]"`
	MaxQuantity     int      `json:"maxQuantity" bun:"max_quantity,notnull,default:0"`
	ProviderPriceID string   `json:"providerPriceId" bun:"provider_price_id"`

	// Relations
	App      *mainschema.App             `json:"app,omitempty" bun:"rel:belongs-to,join:app_id=id"`
	Features []SubscriptionAddOnFeature  `json:"features,omitempty" bun:"rel:has-many,join:id=addon_id"`
	Tiers    []SubscriptionAddOnTier     `json:"tiers,omitempty" bun:"rel:has-many,join:id=addon_id"`
}

// SubscriptionAddOnFeature represents a feature on an add-on
type SubscriptionAddOnFeature struct {
	bun.BaseModel `bun:"table:subscription_addon_features,alias:saf"`

	ID          xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	AddOnID     xid.ID `json:"addOnId" bun:"addon_id,notnull,type:varchar(20)"`
	Key         string `json:"key" bun:"key,notnull"`
	Name        string `json:"name" bun:"name,notnull"`
	Description string `json:"description" bun:"description"`
	Type        string `json:"type" bun:"type,notnull"`
	Value       string `json:"value" bun:"value"`
	CreatedAt   time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	AddOn *SubscriptionAddOn `json:"addOn,omitempty" bun:"rel:belongs-to,join:addon_id=id"`
}

// SubscriptionAddOnTier represents a pricing tier for an add-on
type SubscriptionAddOnTier struct {
	bun.BaseModel `bun:"table:subscription_addon_tiers,alias:sat"`

	ID         xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	AddOnID    xid.ID `json:"addOnId" bun:"addon_id,notnull,type:varchar(20)"`
	TierOrder  int    `json:"tierOrder" bun:"tier_order,notnull"`
	UpTo       int64  `json:"upTo" bun:"up_to,notnull"`
	UnitAmount int64  `json:"unitAmount" bun:"unit_amount,notnull"`
	FlatAmount int64  `json:"flatAmount" bun:"flat_amount,notnull,default:0"`
	CreatedAt  time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	AddOn *SubscriptionAddOn `json:"addOn,omitempty" bun:"rel:belongs-to,join:addon_id=id"`
}

// SubscriptionAddOnItem represents an add-on attached to a subscription
type SubscriptionAddOnItem struct {
	bun.BaseModel `bun:"table:subscription_addon_items,alias:sai"`

	ID                xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	SubscriptionID    xid.ID    `json:"subscriptionId" bun:"subscription_id,notnull,type:varchar(20)"`
	AddOnID           xid.ID    `json:"addOnId" bun:"addon_id,notnull,type:varchar(20)"`
	Quantity          int       `json:"quantity" bun:"quantity,notnull,default:1"`
	ProviderSubItemID string    `json:"providerSubItemId" bun:"provider_sub_item_id"`
	CreatedAt         time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Subscription *Subscription      `json:"subscription,omitempty" bun:"rel:belongs-to,join:subscription_id=id"`
	AddOn        *SubscriptionAddOn `json:"addOn,omitempty" bun:"rel:belongs-to,join:addon_id=id"`
}

// SubscriptionInvoice represents a billing invoice
type SubscriptionInvoice struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_invoices,alias:si"`

	ID                xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	SubscriptionID    xid.ID     `json:"subscriptionId" bun:"subscription_id,notnull,type:varchar(20)"`
	OrganizationID    xid.ID     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	Number            string     `json:"number" bun:"number,notnull,unique"`
	Status            string     `json:"status" bun:"status,notnull"`
	Currency          string     `json:"currency" bun:"currency,notnull,default:'USD'"`
	Subtotal          int64      `json:"subtotal" bun:"subtotal,notnull,default:0"`
	Tax               int64      `json:"tax" bun:"tax,notnull,default:0"`
	Total             int64      `json:"total" bun:"total,notnull,default:0"`
	AmountPaid        int64      `json:"amountPaid" bun:"amount_paid,notnull,default:0"`
	AmountDue         int64      `json:"amountDue" bun:"amount_due,notnull,default:0"`
	Description       string     `json:"description" bun:"description"`
	PeriodStart       time.Time  `json:"periodStart" bun:"period_start,notnull"`
	PeriodEnd         time.Time  `json:"periodEnd" bun:"period_end,notnull"`
	DueDate           time.Time  `json:"dueDate" bun:"due_date,notnull"`
	PaidAt            *time.Time `json:"paidAt" bun:"paid_at"`
	VoidedAt          *time.Time `json:"voidedAt" bun:"voided_at"`
	ProviderInvoiceID string     `json:"providerInvoiceId" bun:"provider_invoice_id"`
	ProviderPDFURL    string     `json:"providerPdfUrl" bun:"provider_pdf_url"`
	HostedInvoiceURL  string     `json:"hostedInvoiceUrl" bun:"hosted_invoice_url"`
	Metadata          map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	Subscription *Subscription             `json:"subscription,omitempty" bun:"rel:belongs-to,join:subscription_id=id"`
	Organization *mainschema.Organization  `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Items        []SubscriptionInvoiceItem `json:"items,omitempty" bun:"rel:has-many,join:id=invoice_id"`
}

// SubscriptionInvoiceItem represents a line item on an invoice
type SubscriptionInvoiceItem struct {
	bun.BaseModel `bun:"table:subscription_invoice_items,alias:sii"`

	ID             xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	InvoiceID      xid.ID     `json:"invoiceId" bun:"invoice_id,notnull,type:varchar(20)"`
	Description    string     `json:"description" bun:"description,notnull"`
	Quantity       int64      `json:"quantity" bun:"quantity,notnull,default:1"`
	UnitAmount     int64      `json:"unitAmount" bun:"unit_amount,notnull"`
	Amount         int64      `json:"amount" bun:"amount,notnull"`
	PlanID         *xid.ID    `json:"planId" bun:"plan_id,type:varchar(20)"`
	AddOnID        *xid.ID    `json:"addOnId" bun:"addon_id,type:varchar(20)"`
	PeriodStart    time.Time  `json:"periodStart" bun:"period_start,notnull"`
	PeriodEnd      time.Time  `json:"periodEnd" bun:"period_end,notnull"`
	Proration      bool       `json:"proration" bun:"proration,notnull,default:false"`
	Metadata       map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	ProviderItemID string     `json:"providerItemId" bun:"provider_item_id"`
	CreatedAt      time.Time  `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Invoice *SubscriptionInvoice `json:"invoice,omitempty" bun:"rel:belongs-to,join:invoice_id=id"`
	Plan    *SubscriptionPlan    `json:"plan,omitempty" bun:"rel:belongs-to,join:plan_id=id"`
	AddOn   *SubscriptionAddOn   `json:"addOn,omitempty" bun:"rel:belongs-to,join:addon_id=id"`
}

// SubscriptionUsageRecord represents a usage record for metered billing
type SubscriptionUsageRecord struct {
	bun.BaseModel `bun:"table:subscription_usage_records,alias:sur"`

	ID               xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	SubscriptionID   xid.ID     `json:"subscriptionId" bun:"subscription_id,notnull,type:varchar(20)"`
	OrganizationID   xid.ID     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	MetricKey        string     `json:"metricKey" bun:"metric_key,notnull"`
	Quantity         int64      `json:"quantity" bun:"quantity,notnull"`
	Action           string     `json:"action" bun:"action,notnull"` // set, increment, decrement
	Timestamp        time.Time  `json:"timestamp" bun:"timestamp,notnull"`
	IdempotencyKey   string     `json:"idempotencyKey" bun:"idempotency_key"`
	Metadata         map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	ProviderRecordID string     `json:"providerRecordId" bun:"provider_record_id"`
	Reported         bool       `json:"reported" bun:"reported,notnull,default:false"`
	ReportedAt       *time.Time `json:"reportedAt" bun:"reported_at"`
	CreatedAt        time.Time  `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Subscription *Subscription            `json:"subscription,omitempty" bun:"rel:belongs-to,join:subscription_id=id"`
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// SubscriptionPaymentMethod represents a stored payment method
type SubscriptionPaymentMethod struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_payment_methods,alias:spm"`

	ID                  xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID      xid.ID `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	Type                string `json:"type" bun:"type,notnull"` // card, bank_account, sepa_debit
	IsDefault           bool   `json:"isDefault" bun:"is_default,notnull,default:false"`
	ProviderMethodID    string `json:"providerMethodId" bun:"provider_method_id,notnull"`
	ProviderCustomerID  string `json:"providerCustomerId" bun:"provider_customer_id,notnull"`

	// Card fields
	CardBrand    string `json:"cardBrand" bun:"card_brand"`
	CardLast4    string `json:"cardLast4" bun:"card_last4"`
	CardExpMonth int    `json:"cardExpMonth" bun:"card_exp_month"`
	CardExpYear  int    `json:"cardExpYear" bun:"card_exp_year"`
	CardFunding  string `json:"cardFunding" bun:"card_funding"`

	// Bank fields
	BankName          string `json:"bankName" bun:"bank_name"`
	BankLast4         string `json:"bankLast4" bun:"bank_last4"`
	BankRoutingNumber string `json:"bankRoutingNumber" bun:"bank_routing_number"`
	BankAccountType   string `json:"bankAccountType" bun:"bank_account_type"`

	// Billing details
	BillingName         string `json:"billingName" bun:"billing_name"`
	BillingEmail        string `json:"billingEmail" bun:"billing_email"`
	BillingPhone        string `json:"billingPhone" bun:"billing_phone"`
	BillingAddressLine1 string `json:"billingAddressLine1" bun:"billing_address_line1"`
	BillingAddressLine2 string `json:"billingAddressLine2" bun:"billing_address_line2"`
	BillingCity         string `json:"billingCity" bun:"billing_city"`
	BillingState        string `json:"billingState" bun:"billing_state"`
	BillingPostalCode   string `json:"billingPostalCode" bun:"billing_postal_code"`
	BillingCountry      string `json:"billingCountry" bun:"billing_country"`

	Metadata map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// SubscriptionCustomer represents the billing customer for an organization
type SubscriptionCustomer struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_customers,alias:sc"`

	ID                 xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID     xid.ID  `json:"organizationId" bun:"organization_id,notnull,unique,type:varchar(20)"`
	ProviderCustomerID string  `json:"providerCustomerId" bun:"provider_customer_id,notnull,unique"`
	Email              string  `json:"email" bun:"email,notnull"`
	Name               string  `json:"name" bun:"name"`
	Phone              string  `json:"phone" bun:"phone"`
	TaxID              string  `json:"taxId" bun:"tax_id"`
	TaxExempt          bool    `json:"taxExempt" bun:"tax_exempt,notnull,default:false"`
	Currency           string  `json:"currency" bun:"currency,notnull,default:'USD'"`
	Balance            int64   `json:"balance" bun:"balance,notnull,default:0"`
	DefaultPaymentID   *xid.ID `json:"defaultPaymentId" bun:"default_payment_id,type:varchar(20)"`

	// Billing address
	BillingAddressLine1 string `json:"billingAddressLine1" bun:"billing_address_line1"`
	BillingAddressLine2 string `json:"billingAddressLine2" bun:"billing_address_line2"`
	BillingCity         string `json:"billingCity" bun:"billing_city"`
	BillingState        string `json:"billingState" bun:"billing_state"`
	BillingPostalCode   string `json:"billingPostalCode" bun:"billing_postal_code"`
	BillingCountry      string `json:"billingCountry" bun:"billing_country"`

	Metadata map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	Organization   *mainschema.Organization    `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	DefaultPayment *SubscriptionPaymentMethod  `json:"defaultPayment,omitempty" bun:"rel:belongs-to,join:default_payment_id=id"`
}

// SubscriptionEvent represents an audit event for subscription changes
type SubscriptionEvent struct {
	bun.BaseModel `bun:"table:subscription_events,alias:se"`

	ID             xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	SubscriptionID *xid.ID                `json:"subscriptionId" bun:"subscription_id,type:varchar(20)"`
	OrganizationID xid.ID                 `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	EventType      string                 `json:"eventType" bun:"event_type,notnull"`
	EventData      map[string]interface{} `json:"eventData" bun:"event_data,type:jsonb"`
	ProviderEventID string                `json:"providerEventId" bun:"provider_event_id"`
	IPAddress      string                 `json:"ipAddress" bun:"ip_address"`
	UserAgent      string                 `json:"userAgent" bun:"user_agent"`
	ActorID        *xid.ID                `json:"actorId" bun:"actor_id,type:varchar(20)"`
	CreatedAt      time.Time              `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Subscription *Subscription            `json:"subscription,omitempty" bun:"rel:belongs-to,join:subscription_id=id"`
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

