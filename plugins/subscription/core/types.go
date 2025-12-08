// Package core defines the core domain types for the subscription plugin.
// These types have no external dependencies and define the business domain model.
package core

// BillingPattern defines how a plan or add-on is billed
type BillingPattern string

const (
	// BillingPatternFlat is a fixed price per billing period
	BillingPatternFlat BillingPattern = "flat"
	// BillingPatternPerSeat charges per user/member
	BillingPatternPerSeat BillingPattern = "per_seat"
	// BillingPatternTiered uses volume-based pricing tiers
	BillingPatternTiered BillingPattern = "tiered"
	// BillingPatternUsage is pay-per-use metered billing
	BillingPatternUsage BillingPattern = "usage"
	// BillingPatternHybrid combines base price with usage
	BillingPatternHybrid BillingPattern = "hybrid"
)

// String returns the string representation of the billing pattern
func (b BillingPattern) String() string {
	return string(b)
}

// IsValid checks if the billing pattern is valid
func (b BillingPattern) IsValid() bool {
	switch b {
	case BillingPatternFlat, BillingPatternPerSeat, BillingPatternTiered,
		BillingPatternUsage, BillingPatternHybrid:
		return true
	}
	return false
}

// BillingInterval defines the billing frequency
type BillingInterval string

const (
	// BillingIntervalMonthly bills every month
	BillingIntervalMonthly BillingInterval = "monthly"
	// BillingIntervalYearly bills every year
	BillingIntervalYearly BillingInterval = "yearly"
	// BillingIntervalOneTime is a one-time charge
	BillingIntervalOneTime BillingInterval = "one_time"
)

// String returns the string representation of the billing interval
func (b BillingInterval) String() string {
	return string(b)
}

// IsValid checks if the billing interval is valid
func (b BillingInterval) IsValid() bool {
	switch b {
	case BillingIntervalMonthly, BillingIntervalYearly, BillingIntervalOneTime:
		return true
	}
	return false
}

// SubscriptionStatus defines the current state of a subscription
type SubscriptionStatus string

const (
	// StatusTrialing is an active trial period
	StatusTrialing SubscriptionStatus = "trialing"
	// StatusActive is a paid, active subscription
	StatusActive SubscriptionStatus = "active"
	// StatusPastDue has failed payment but still active
	StatusPastDue SubscriptionStatus = "past_due"
	// StatusCanceled has been cancelled
	StatusCanceled SubscriptionStatus = "canceled"
	// StatusUnpaid has exhausted retry attempts
	StatusUnpaid SubscriptionStatus = "unpaid"
	// StatusPaused is temporarily paused
	StatusPaused SubscriptionStatus = "paused"
	// StatusIncomplete requires payment action to activate
	StatusIncomplete SubscriptionStatus = "incomplete"
)

// String returns the string representation of the subscription status
func (s SubscriptionStatus) String() string {
	return string(s)
}

// IsValid checks if the subscription status is valid
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case StatusTrialing, StatusActive, StatusPastDue, StatusCanceled,
		StatusUnpaid, StatusPaused, StatusIncomplete:
		return true
	}
	return false
}

// IsActiveOrTrialing returns true if the subscription is usable
func (s SubscriptionStatus) IsActiveOrTrialing() bool {
	return s == StatusActive || s == StatusTrialing
}

// FeatureType defines the type of a plan feature
type FeatureType string

const (
	// FeatureTypeBoolean is a simple on/off feature
	FeatureTypeBoolean FeatureType = "boolean"
	// FeatureTypeLimit is a numeric limit
	FeatureTypeLimit FeatureType = "limit"
	// FeatureTypeUnlimited is no limit on the feature
	FeatureTypeUnlimited FeatureType = "unlimited"
	// FeatureTypeMetered is a usage-based billing feature
	FeatureTypeMetered FeatureType = "metered"
	// FeatureTypeTiered provides different values at different usage levels
	FeatureTypeTiered FeatureType = "tiered"
)

// String returns the string representation of the feature type
func (f FeatureType) String() string {
	return string(f)
}

// IsValid checks if the feature type is valid
func (f FeatureType) IsValid() bool {
	switch f {
	case FeatureTypeBoolean, FeatureTypeLimit, FeatureTypeUnlimited,
		FeatureTypeMetered, FeatureTypeTiered:
		return true
	}
	return false
}

// IsConsumable returns true if the feature type can be consumed (has usage tracking)
func (f FeatureType) IsConsumable() bool {
	switch f {
	case FeatureTypeLimit, FeatureTypeMetered:
		return true
	}
	return false
}

// ResetPeriod defines when feature usage should be reset
type ResetPeriod string

const (
	// ResetPeriodNone means usage never resets (cumulative)
	ResetPeriodNone ResetPeriod = "none"
	// ResetPeriodDaily resets usage every day
	ResetPeriodDaily ResetPeriod = "daily"
	// ResetPeriodWeekly resets usage every week
	ResetPeriodWeekly ResetPeriod = "weekly"
	// ResetPeriodMonthly resets usage every month
	ResetPeriodMonthly ResetPeriod = "monthly"
	// ResetPeriodYearly resets usage every year
	ResetPeriodYearly ResetPeriod = "yearly"
	// ResetPeriodBillingCycle resets usage at each billing cycle
	ResetPeriodBillingCycle ResetPeriod = "billing_period"
)

// String returns the string representation of the reset period
func (r ResetPeriod) String() string {
	return string(r)
}

// IsValid checks if the reset period is valid
func (r ResetPeriod) IsValid() bool {
	switch r {
	case ResetPeriodNone, ResetPeriodDaily, ResetPeriodWeekly,
		ResetPeriodMonthly, ResetPeriodYearly, ResetPeriodBillingCycle:
		return true
	}
	return false
}

// FeatureGrantType defines the type of feature grant
type FeatureGrantType string

const (
	// FeatureGrantTypeAddon is a grant from an add-on purchase
	FeatureGrantTypeAddon FeatureGrantType = "addon"
	// FeatureGrantTypeOverride is a manual override
	FeatureGrantTypeOverride FeatureGrantType = "override"
	// FeatureGrantTypePromotion is a promotional grant
	FeatureGrantTypePromotion FeatureGrantType = "promotion"
	// FeatureGrantTypeTrial is a trial grant
	FeatureGrantTypeTrial FeatureGrantType = "trial"
	// FeatureGrantTypeManual is a manually added grant
	FeatureGrantTypeManual FeatureGrantType = "manual"
)

// String returns the string representation of the feature grant type
func (f FeatureGrantType) String() string {
	return string(f)
}

// IsValid checks if the feature grant type is valid
func (f FeatureGrantType) IsValid() bool {
	switch f {
	case FeatureGrantTypeAddon, FeatureGrantTypeOverride, FeatureGrantTypePromotion,
		FeatureGrantTypeTrial, FeatureGrantTypeManual:
		return true
	}
	return false
}

// FeatureUsageAction defines the type of usage action
type FeatureUsageAction string

const (
	// FeatureUsageActionConsume decrements the available quota
	FeatureUsageActionConsume FeatureUsageAction = "consume"
	// FeatureUsageActionGrant adds to the quota
	FeatureUsageActionGrant FeatureUsageAction = "grant"
	// FeatureUsageActionReset resets the usage counter
	FeatureUsageActionReset FeatureUsageAction = "reset"
	// FeatureUsageActionAdjust manually adjusts the usage
	FeatureUsageActionAdjust FeatureUsageAction = "adjust"
)

// String returns the string representation of the feature usage action
func (f FeatureUsageAction) String() string {
	return string(f)
}

// IsValid checks if the feature usage action is valid
func (f FeatureUsageAction) IsValid() bool {
	switch f {
	case FeatureUsageActionConsume, FeatureUsageActionGrant,
		FeatureUsageActionReset, FeatureUsageActionAdjust:
		return true
	}
	return false
}

// InvoiceStatus defines the state of an invoice
type InvoiceStatus string

const (
	// InvoiceStatusDraft is not yet finalized
	InvoiceStatusDraft InvoiceStatus = "draft"
	// InvoiceStatusOpen is awaiting payment
	InvoiceStatusOpen InvoiceStatus = "open"
	// InvoiceStatusPaid has been paid
	InvoiceStatusPaid InvoiceStatus = "paid"
	// InvoiceStatusVoid has been voided
	InvoiceStatusVoid InvoiceStatus = "void"
	// InvoiceStatusUncollectible cannot be collected
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
)

// String returns the string representation of the invoice status
func (i InvoiceStatus) String() string {
	return string(i)
}

// IsValid checks if the invoice status is valid
func (i InvoiceStatus) IsValid() bool {
	switch i {
	case InvoiceStatusDraft, InvoiceStatusOpen, InvoiceStatusPaid,
		InvoiceStatusVoid, InvoiceStatusUncollectible:
		return true
	}
	return false
}

// PaymentMethodType defines the type of payment method
type PaymentMethodType string

const (
	// PaymentMethodCard is a credit/debit card
	PaymentMethodCard PaymentMethodType = "card"
	// PaymentMethodBankAccount is a bank account
	PaymentMethodBankAccount PaymentMethodType = "bank_account"
	// PaymentMethodSepaDebit is SEPA direct debit
	PaymentMethodSepaDebit PaymentMethodType = "sepa_debit"
)

// String returns the string representation of the payment method type
func (p PaymentMethodType) String() string {
	return string(p)
}

// IsValid checks if the payment method type is valid
func (p PaymentMethodType) IsValid() bool {
	switch p {
	case PaymentMethodCard, PaymentMethodBankAccount, PaymentMethodSepaDebit:
		return true
	}
	return false
}

// EventType defines subscription-related event types
type EventType string

const (
	// EventSubscriptionCreated when a subscription is created
	EventSubscriptionCreated EventType = "subscription.created"
	// EventSubscriptionUpdated when a subscription is updated
	EventSubscriptionUpdated EventType = "subscription.updated"
	// EventSubscriptionCanceled when a subscription is canceled
	EventSubscriptionCanceled EventType = "subscription.canceled"
	// EventSubscriptionPaused when a subscription is paused
	EventSubscriptionPaused EventType = "subscription.paused"
	// EventSubscriptionResumed when a subscription is resumed
	EventSubscriptionResumed EventType = "subscription.resumed"
	// EventSubscriptionTrialEnding when trial is about to end
	EventSubscriptionTrialEnding EventType = "subscription.trial_ending"
	// EventPaymentSucceeded when payment is successful
	EventPaymentSucceeded EventType = "payment.succeeded"
	// EventPaymentFailed when payment fails
	EventPaymentFailed EventType = "payment.failed"
	// EventInvoiceCreated when invoice is created
	EventInvoiceCreated EventType = "invoice.created"
	// EventInvoicePaid when invoice is paid
	EventInvoicePaid EventType = "invoice.paid"
	// EventUsageRecorded when usage is recorded
	EventUsageRecorded EventType = "usage.recorded"
)

// String returns the string representation of the event type
func (e EventType) String() string {
	return string(e)
}

// TierMode defines how tiered pricing is applied
type TierMode string

const (
	// TierModeGraduated applies each tier's price to units in that tier
	TierModeGraduated TierMode = "graduated"
	// TierModeVolume applies the final tier's price to all units
	TierModeVolume TierMode = "volume"
)

// String returns the string representation of the tier mode
func (t TierMode) String() string {
	return string(t)
}

// IsValid checks if the tier mode is valid
func (t TierMode) IsValid() bool {
	return t == TierModeGraduated || t == TierModeVolume
}

// DefaultCurrency is the default currency if not specified
const DefaultCurrency = "USD"

// Feature keys for common plan features
const (
	FeatureKeyMaxMembers        = "max_members"
	FeatureKeyMaxTeams          = "max_teams"
	FeatureKeyMaxProjects       = "max_projects"
	FeatureKeyMaxStorage        = "max_storage_gb"
	FeatureKeyMaxAPICallsMonth  = "max_api_calls_month"
	FeatureKeyCustomDomain      = "custom_domain"
	FeatureKeyPrioritySupport   = "priority_support"
	FeatureKeySSO               = "sso"
	FeatureKeyAuditLogs         = "audit_logs"
	FeatureKeyAdvancedAnalytics = "advanced_analytics"
)
