package subscription

import (
	"github.com/xraph/authsome/plugins/subscription/internal/hooks"
)

// Re-export hook types for convenience.
type (
	BeforeSubscriptionCreateHook = hooks.BeforeSubscriptionCreateHook
	AfterSubscriptionCreateHook  = hooks.AfterSubscriptionCreateHook
	BeforeSubscriptionUpdateHook = hooks.BeforeSubscriptionUpdateHook
	AfterSubscriptionUpdateHook  = hooks.AfterSubscriptionUpdateHook
	BeforeSubscriptionCancelHook = hooks.BeforeSubscriptionCancelHook
	AfterSubscriptionCancelHook  = hooks.AfterSubscriptionCancelHook
	BeforeSubscriptionPauseHook  = hooks.BeforeSubscriptionPauseHook
	AfterSubscriptionPauseHook   = hooks.AfterSubscriptionPauseHook
	BeforeSubscriptionResumeHook = hooks.BeforeSubscriptionResumeHook
	AfterSubscriptionResumeHook  = hooks.AfterSubscriptionResumeHook

	OnSubscriptionStatusChangeHook = hooks.OnSubscriptionStatusChangeHook
	OnTrialEndingHook              = hooks.OnTrialEndingHook
	OnTrialEndedHook               = hooks.OnTrialEndedHook
	OnPaymentSuccessHook           = hooks.OnPaymentSuccessHook
	OnPaymentFailedHook            = hooks.OnPaymentFailedHook
	OnInvoiceCreatedHook           = hooks.OnInvoiceCreatedHook
	OnInvoicePaidHook              = hooks.OnInvoicePaidHook
	OnUsageLimitApproachingHook    = hooks.OnUsageLimitApproachingHook
	OnUsageLimitExceededHook       = hooks.OnUsageLimitExceededHook

	BeforePlanChangeHook  = hooks.BeforePlanChangeHook
	AfterPlanChangeHook   = hooks.AfterPlanChangeHook
	BeforeAddOnAttachHook = hooks.BeforeAddOnAttachHook
	AfterAddOnAttachHook  = hooks.AfterAddOnAttachHook
	BeforeAddOnDetachHook = hooks.BeforeAddOnDetachHook
	AfterAddOnDetachHook  = hooks.AfterAddOnDetachHook

	SubscriptionHookRegistry = hooks.SubscriptionHookRegistry
)

// NewSubscriptionHookRegistry creates a new hook registry.
var NewSubscriptionHookRegistry = hooks.NewSubscriptionHookRegistry
