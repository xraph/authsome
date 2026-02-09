// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// SubscriptionPlan represents a subscription plan in the database.
type SubscriptionPlan struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_plans,alias:sp"`

	ID              xid.ID         `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID           xid.ID         `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	Name            string         `bun:"name,notnull"                    json:"name"`
	Slug            string         `bun:"slug,notnull"                    json:"slug"` // Unique within app
	Description     string         `bun:"description"                     json:"description"`
	BillingPattern  string         `bun:"billing_pattern,notnull"         json:"billingPattern"`
	BillingInterval string         `bun:"billing_interval,notnull"        json:"billingInterval"`
	BasePrice       int64          `bun:"base_price,notnull,default:0"    json:"basePrice"`
	Currency        string         `bun:"currency,notnull,default:'USD'"  json:"currency"`
	TrialDays       int            `bun:"trial_days,notnull,default:0"    json:"trialDays"`
	TierMode        string         `bun:"tier_mode,default:'graduated'"   json:"tierMode"`
	Metadata        map[string]any `bun:"metadata,type:jsonb"             json:"metadata"`
	IsActive        bool           `bun:"is_active,notnull,default:true"  json:"isActive"`
	IsPublic        bool           `bun:"is_public,notnull,default:true"  json:"isPublic"`
	DisplayOrder    int            `bun:"display_order,notnull,default:0" json:"displayOrder"`
	ProviderPlanID  string         `bun:"provider_plan_id"                json:"providerPlanId"`
	ProviderPriceID string         `bun:"provider_price_id"               json:"providerPriceId"`

	// Relations
	App          *mainschema.App           `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Features     []SubscriptionPlanFeature `bun:"rel:has-many,join:id=plan_id"  json:"features,omitempty"`
	Tiers        []SubscriptionPlanTier    `bun:"rel:has-many,join:id=plan_id"  json:"tiers,omitempty"`
	FeatureLinks []PlanFeatureLink         `bun:"rel:has-many,join:id=plan_id"  json:"featureLinks,omitempty"`
}

// SubscriptionPlanFeature represents a feature limit on a plan.
type SubscriptionPlanFeature struct {
	bun.BaseModel `bun:"table:subscription_plan_features,alias:spf"`

	ID          xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	PlanID      xid.ID    `bun:"plan_id,notnull,type:varchar(20)"                      json:"planId"`
	Key         string    `bun:"key,notnull"                                           json:"key"`
	Name        string    `bun:"name,notnull"                                          json:"name"`
	Description string    `bun:"description"                                           json:"description"`
	Type        string    `bun:"type,notnull"                                          json:"type"`  // boolean, limit, unlimited
	Value       string    `bun:"value"                                                 json:"value"` // JSON-encoded value
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Plan *SubscriptionPlan `bun:"rel:belongs-to,join:plan_id=id" json:"plan,omitempty"`
}

// SubscriptionPlanTier represents a pricing tier for a plan.
type SubscriptionPlanTier struct {
	bun.BaseModel `bun:"table:subscription_plan_tiers,alias:spt"`

	ID         xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	PlanID     xid.ID    `bun:"plan_id,notnull,type:varchar(20)"                      json:"planId"`
	TierOrder  int       `bun:"tier_order,notnull"                                    json:"tierOrder"`
	UpTo       int64     `bun:"up_to,notnull"                                         json:"upTo"` // -1 for infinite
	UnitAmount int64     `bun:"unit_amount,notnull"                                   json:"unitAmount"`
	FlatAmount int64     `bun:"flat_amount,notnull,default:0"                         json:"flatAmount"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Plan *SubscriptionPlan `bun:"rel:belongs-to,join:plan_id=id" json:"plan,omitempty"`
}

// Subscription represents an organization's subscription.
type Subscription struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscriptions,alias:sub"`

	ID                 xid.ID         `bun:"id,pk,type:varchar(20)"                   json:"id"`
	OrganizationID     xid.ID         `bun:"organization_id,notnull,type:varchar(20)" json:"organizationId"`
	PlanID             xid.ID         `bun:"plan_id,notnull,type:varchar(20)"         json:"planId"`
	Status             string         `bun:"status,notnull"                           json:"status"`
	Quantity           int            `bun:"quantity,notnull,default:1"               json:"quantity"`
	CurrentPeriodStart time.Time      `bun:"current_period_start,notnull"             json:"currentPeriodStart"`
	CurrentPeriodEnd   time.Time      `bun:"current_period_end,notnull"               json:"currentPeriodEnd"`
	TrialStart         *time.Time     `bun:"trial_start"                              json:"trialStart"`
	TrialEnd           *time.Time     `bun:"trial_end"                                json:"trialEnd"`
	CancelAt           *time.Time     `bun:"cancel_at"                                json:"cancelAt"`
	CanceledAt         *time.Time     `bun:"canceled_at"                              json:"canceledAt"`
	EndedAt            *time.Time     `bun:"ended_at"                                 json:"endedAt"`
	PausedAt           *time.Time     `bun:"paused_at"                                json:"pausedAt"`
	ResumeAt           *time.Time     `bun:"resume_at"                                json:"resumeAt"`
	ProviderSubID      string         `bun:"provider_sub_id"                          json:"providerSubId"`
	ProviderCustomerID string         `bun:"provider_customer_id"                     json:"providerCustomerId"`
	Metadata           map[string]any `bun:"metadata,type:jsonb"                      json:"metadata"`

	// Relations
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	Plan         *SubscriptionPlan        `bun:"rel:belongs-to,join:plan_id=id"         json:"plan,omitempty"`
	AddOns       []SubscriptionAddOnItem  `bun:"rel:has-many,join:id=subscription_id"   json:"addOns,omitempty"`
}

// SubscriptionAddOn represents an add-on product.
type SubscriptionAddOn struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_addons,alias:sao"`

	ID              xid.ID         `bun:"id,pk,type:varchar(20)"                     json:"id"`
	AppID           xid.ID         `bun:"app_id,notnull,type:varchar(20)"            json:"appId"`
	Name            string         `bun:"name,notnull"                               json:"name"`
	Slug            string         `bun:"slug,notnull"                               json:"slug"`
	Description     string         `bun:"description"                                json:"description"`
	BillingPattern  string         `bun:"billing_pattern,notnull"                    json:"billingPattern"`
	BillingInterval string         `bun:"billing_interval,notnull"                   json:"billingInterval"`
	Price           int64          `bun:"price,notnull,default:0"                    json:"price"`
	Currency        string         `bun:"currency,notnull,default:'USD'"             json:"currency"`
	TierMode        string         `bun:"tier_mode,default:'graduated'"              json:"tierMode"`
	Metadata        map[string]any `bun:"metadata,type:jsonb"                        json:"metadata"`
	IsActive        bool           `bun:"is_active,notnull,default:true"             json:"isActive"`
	IsPublic        bool           `bun:"is_public,notnull,default:true"             json:"isPublic"`
	DisplayOrder    int            `bun:"display_order,notnull,default:0"            json:"displayOrder"`
	RequiresPlanIDs []string       `bun:"requires_plan_ids,array,type:varchar(20)[]" json:"requiresPlanIds"`
	ExcludesPlanIDs []string       `bun:"excludes_plan_ids,array,type:varchar(20)[]" json:"excludesPlanIds"`
	MaxQuantity     int            `bun:"max_quantity,notnull,default:0"             json:"maxQuantity"`
	ProviderPriceID string         `bun:"provider_price_id"                          json:"providerPriceId"`

	// Relations
	App      *mainschema.App            `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Features []SubscriptionAddOnFeature `bun:"rel:has-many,join:id=addon_id" json:"features,omitempty"`
	Tiers    []SubscriptionAddOnTier    `bun:"rel:has-many,join:id=addon_id" json:"tiers,omitempty"`
}

// SubscriptionAddOnFeature represents a feature on an add-on.
type SubscriptionAddOnFeature struct {
	bun.BaseModel `bun:"table:subscription_addon_features,alias:saf"`

	ID          xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	AddOnID     xid.ID    `bun:"addon_id,notnull,type:varchar(20)"                     json:"addOnId"`
	Key         string    `bun:"key,notnull"                                           json:"key"`
	Name        string    `bun:"name,notnull"                                          json:"name"`
	Description string    `bun:"description"                                           json:"description"`
	Type        string    `bun:"type,notnull"                                          json:"type"`
	Value       string    `bun:"value"                                                 json:"value"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	AddOn *SubscriptionAddOn `bun:"rel:belongs-to,join:addon_id=id" json:"addOn,omitempty"`
}

// SubscriptionAddOnTier represents a pricing tier for an add-on.
type SubscriptionAddOnTier struct {
	bun.BaseModel `bun:"table:subscription_addon_tiers,alias:sat"`

	ID         xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	AddOnID    xid.ID    `bun:"addon_id,notnull,type:varchar(20)"                     json:"addOnId"`
	TierOrder  int       `bun:"tier_order,notnull"                                    json:"tierOrder"`
	UpTo       int64     `bun:"up_to,notnull"                                         json:"upTo"`
	UnitAmount int64     `bun:"unit_amount,notnull"                                   json:"unitAmount"`
	FlatAmount int64     `bun:"flat_amount,notnull,default:0"                         json:"flatAmount"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	AddOn *SubscriptionAddOn `bun:"rel:belongs-to,join:addon_id=id" json:"addOn,omitempty"`
}

// SubscriptionAddOnItem represents an add-on attached to a subscription.
type SubscriptionAddOnItem struct {
	bun.BaseModel `bun:"table:subscription_addon_items,alias:sai"`

	ID                xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	SubscriptionID    xid.ID    `bun:"subscription_id,notnull,type:varchar(20)"              json:"subscriptionId"`
	AddOnID           xid.ID    `bun:"addon_id,notnull,type:varchar(20)"                     json:"addOnId"`
	Quantity          int       `bun:"quantity,notnull,default:1"                            json:"quantity"`
	ProviderSubItemID string    `bun:"provider_sub_item_id"                                  json:"providerSubItemId"`
	CreatedAt         time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Subscription *Subscription      `bun:"rel:belongs-to,join:subscription_id=id" json:"subscription,omitempty"`
	AddOn        *SubscriptionAddOn `bun:"rel:belongs-to,join:addon_id=id"        json:"addOn,omitempty"`
}

// SubscriptionInvoice represents a billing invoice.
type SubscriptionInvoice struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_invoices,alias:si"`

	ID                xid.ID         `bun:"id,pk,type:varchar(20)"                   json:"id"`
	SubscriptionID    xid.ID         `bun:"subscription_id,notnull,type:varchar(20)" json:"subscriptionId"`
	OrganizationID    xid.ID         `bun:"organization_id,notnull,type:varchar(20)" json:"organizationId"`
	Number            string         `bun:"number,notnull,unique"                    json:"number"`
	Status            string         `bun:"status,notnull"                           json:"status"`
	Currency          string         `bun:"currency,notnull,default:'USD'"           json:"currency"`
	Subtotal          int64          `bun:"subtotal,notnull,default:0"               json:"subtotal"`
	Tax               int64          `bun:"tax,notnull,default:0"                    json:"tax"`
	Total             int64          `bun:"total,notnull,default:0"                  json:"total"`
	AmountPaid        int64          `bun:"amount_paid,notnull,default:0"            json:"amountPaid"`
	AmountDue         int64          `bun:"amount_due,notnull,default:0"             json:"amountDue"`
	Description       string         `bun:"description"                              json:"description"`
	PeriodStart       time.Time      `bun:"period_start,notnull"                     json:"periodStart"`
	PeriodEnd         time.Time      `bun:"period_end,notnull"                       json:"periodEnd"`
	DueDate           time.Time      `bun:"due_date,notnull"                         json:"dueDate"`
	PaidAt            *time.Time     `bun:"paid_at"                                  json:"paidAt"`
	VoidedAt          *time.Time     `bun:"voided_at"                                json:"voidedAt"`
	ProviderInvoiceID string         `bun:"provider_invoice_id"                      json:"providerInvoiceId"`
	ProviderPDFURL    string         `bun:"provider_pdf_url"                         json:"providerPdfUrl"`
	HostedInvoiceURL  string         `bun:"hosted_invoice_url"                       json:"hostedInvoiceUrl"`
	Metadata          map[string]any `bun:"metadata,type:jsonb"                      json:"metadata"`

	// Relations
	Subscription *Subscription             `bun:"rel:belongs-to,join:subscription_id=id" json:"subscription,omitempty"`
	Organization *mainschema.Organization  `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	Items        []SubscriptionInvoiceItem `bun:"rel:has-many,join:id=invoice_id"        json:"items,omitempty"`
}

// SubscriptionInvoiceItem represents a line item on an invoice.
type SubscriptionInvoiceItem struct {
	bun.BaseModel `bun:"table:subscription_invoice_items,alias:sii"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	InvoiceID      xid.ID         `bun:"invoice_id,notnull,type:varchar(20)"                   json:"invoiceId"`
	Description    string         `bun:"description,notnull"                                   json:"description"`
	Quantity       int64          `bun:"quantity,notnull,default:1"                            json:"quantity"`
	UnitAmount     int64          `bun:"unit_amount,notnull"                                   json:"unitAmount"`
	Amount         int64          `bun:"amount,notnull"                                        json:"amount"`
	PlanID         *xid.ID        `bun:"plan_id,type:varchar(20)"                              json:"planId"`
	AddOnID        *xid.ID        `bun:"addon_id,type:varchar(20)"                             json:"addOnId"`
	PeriodStart    time.Time      `bun:"period_start,notnull"                                  json:"periodStart"`
	PeriodEnd      time.Time      `bun:"period_end,notnull"                                    json:"periodEnd"`
	Proration      bool           `bun:"proration,notnull,default:false"                       json:"proration"`
	Metadata       map[string]any `bun:"metadata,type:jsonb"                                   json:"metadata"`
	ProviderItemID string         `bun:"provider_item_id"                                      json:"providerItemId"`
	CreatedAt      time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Invoice *SubscriptionInvoice `bun:"rel:belongs-to,join:invoice_id=id" json:"invoice,omitempty"`
	Plan    *SubscriptionPlan    `bun:"rel:belongs-to,join:plan_id=id"    json:"plan,omitempty"`
	AddOn   *SubscriptionAddOn   `bun:"rel:belongs-to,join:addon_id=id"   json:"addOn,omitempty"`
}

// SubscriptionUsageRecord represents a usage record for metered billing.
type SubscriptionUsageRecord struct {
	bun.BaseModel `bun:"table:subscription_usage_records,alias:sur"`

	ID               xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	SubscriptionID   xid.ID         `bun:"subscription_id,notnull,type:varchar(20)"              json:"subscriptionId"`
	OrganizationID   xid.ID         `bun:"organization_id,notnull,type:varchar(20)"              json:"organizationId"`
	MetricKey        string         `bun:"metric_key,notnull"                                    json:"metricKey"`
	Quantity         int64          `bun:"quantity,notnull"                                      json:"quantity"`
	Action           string         `bun:"action,notnull"                                        json:"action"` // set, increment, decrement
	Timestamp        time.Time      `bun:"timestamp,notnull"                                     json:"timestamp"`
	IdempotencyKey   string         `bun:"idempotency_key"                                       json:"idempotencyKey"`
	Metadata         map[string]any `bun:"metadata,type:jsonb"                                   json:"metadata"`
	ProviderRecordID string         `bun:"provider_record_id"                                    json:"providerRecordId"`
	Reported         bool           `bun:"reported,notnull,default:false"                        json:"reported"`
	ReportedAt       *time.Time     `bun:"reported_at"                                           json:"reportedAt"`
	CreatedAt        time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Subscription *Subscription            `bun:"rel:belongs-to,join:subscription_id=id" json:"subscription,omitempty"`
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

// SubscriptionPaymentMethod represents a stored payment method.
type SubscriptionPaymentMethod struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_payment_methods,alias:spm"`

	ID                 xid.ID `bun:"id,pk,type:varchar(20)"                   json:"id"`
	OrganizationID     xid.ID `bun:"organization_id,notnull,type:varchar(20)" json:"organizationId"`
	Type               string `bun:"type,notnull"                             json:"type"` // card, bank_account, sepa_debit
	IsDefault          bool   `bun:"is_default,notnull,default:false"         json:"isDefault"`
	ProviderMethodID   string `bun:"provider_method_id,notnull"               json:"providerMethodId"`
	ProviderCustomerID string `bun:"provider_customer_id,notnull"             json:"providerCustomerId"`

	// Card fields
	CardBrand    string `bun:"card_brand"     json:"cardBrand"`
	CardLast4    string `bun:"card_last4"     json:"cardLast4"`
	CardExpMonth int    `bun:"card_exp_month" json:"cardExpMonth"`
	CardExpYear  int    `bun:"card_exp_year"  json:"cardExpYear"`
	CardFunding  string `bun:"card_funding"   json:"cardFunding"`

	// Bank fields
	BankName          string `bun:"bank_name"           json:"bankName"`
	BankLast4         string `bun:"bank_last4"          json:"bankLast4"`
	BankRoutingNumber string `bun:"bank_routing_number" json:"bankRoutingNumber"`
	BankAccountType   string `bun:"bank_account_type"   json:"bankAccountType"`

	// Billing details
	BillingName         string `bun:"billing_name"          json:"billingName"`
	BillingEmail        string `bun:"billing_email"         json:"billingEmail"`
	BillingPhone        string `bun:"billing_phone"         json:"billingPhone"`
	BillingAddressLine1 string `bun:"billing_address_line1" json:"billingAddressLine1"`
	BillingAddressLine2 string `bun:"billing_address_line2" json:"billingAddressLine2"`
	BillingCity         string `bun:"billing_city"          json:"billingCity"`
	BillingState        string `bun:"billing_state"         json:"billingState"`
	BillingPostalCode   string `bun:"billing_postal_code"   json:"billingPostalCode"`
	BillingCountry      string `bun:"billing_country"       json:"billingCountry"`

	Metadata map[string]any `bun:"metadata,type:jsonb" json:"metadata"`

	// Relations
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

// SubscriptionCustomer represents the billing customer for an organization.
type SubscriptionCustomer struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_customers,alias:sc"`

	ID                 xid.ID  `bun:"id,pk,type:varchar(20)"                          json:"id"`
	OrganizationID     xid.ID  `bun:"organization_id,notnull,unique,type:varchar(20)" json:"organizationId"`
	ProviderCustomerID string  `bun:"provider_customer_id,notnull,unique"             json:"providerCustomerId"`
	Email              string  `bun:"email,notnull"                                   json:"email"`
	Name               string  `bun:"name"                                            json:"name"`
	Phone              string  `bun:"phone"                                           json:"phone"`
	TaxID              string  `bun:"tax_id"                                          json:"taxId"`
	TaxExempt          bool    `bun:"tax_exempt,notnull,default:false"                json:"taxExempt"`
	Currency           string  `bun:"currency,notnull,default:'USD'"                  json:"currency"`
	Balance            int64   `bun:"balance,notnull,default:0"                       json:"balance"`
	DefaultPaymentID   *xid.ID `bun:"default_payment_id,type:varchar(20)"             json:"defaultPaymentId"`

	// Billing address
	BillingAddressLine1 string `bun:"billing_address_line1" json:"billingAddressLine1"`
	BillingAddressLine2 string `bun:"billing_address_line2" json:"billingAddressLine2"`
	BillingCity         string `bun:"billing_city"          json:"billingCity"`
	BillingState        string `bun:"billing_state"         json:"billingState"`
	BillingPostalCode   string `bun:"billing_postal_code"   json:"billingPostalCode"`
	BillingCountry      string `bun:"billing_country"       json:"billingCountry"`

	Metadata map[string]any `bun:"metadata,type:jsonb" json:"metadata"`

	// Relations
	Organization   *mainschema.Organization   `bun:"rel:belongs-to,join:organization_id=id"    json:"organization,omitempty"`
	DefaultPayment *SubscriptionPaymentMethod `bun:"rel:belongs-to,join:default_payment_id=id" json:"defaultPayment,omitempty"`
}

// SubscriptionEvent represents an audit event for subscription changes.
type SubscriptionEvent struct {
	bun.BaseModel `bun:"table:subscription_events,alias:se"`

	ID              xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	SubscriptionID  *xid.ID        `bun:"subscription_id,type:varchar(20)"                      json:"subscriptionId"`
	OrganizationID  xid.ID         `bun:"organization_id,notnull,type:varchar(20)"              json:"organizationId"`
	EventType       string         `bun:"event_type,notnull"                                    json:"eventType"`
	EventData       map[string]any `bun:"event_data,type:jsonb"                                 json:"eventData"`
	ProviderEventID string         `bun:"provider_event_id"                                     json:"providerEventId"`
	IPAddress       string         `bun:"ip_address"                                            json:"ipAddress"`
	UserAgent       string         `bun:"user_agent"                                            json:"userAgent"`
	ActorID         *xid.ID        `bun:"actor_id,type:varchar(20)"                             json:"actorId"`
	CreatedAt       time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Subscription *Subscription            `bun:"rel:belongs-to,join:subscription_id=id" json:"subscription,omitempty"`
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}
