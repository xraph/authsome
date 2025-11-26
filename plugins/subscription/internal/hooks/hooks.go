// Package hooks provides subscription-specific hook types and registry.
package hooks

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
)

// Hook type definitions for subscription events
// These hooks can be used by other plugins or the host application to react to subscription events

// BeforeSubscriptionCreateHook is called before creating a subscription
type BeforeSubscriptionCreateHook func(ctx context.Context, orgID, planID xid.ID) error

// AfterSubscriptionCreateHook is called after creating a subscription
type AfterSubscriptionCreateHook func(ctx context.Context, subscription *core.Subscription) error

// BeforeSubscriptionUpdateHook is called before updating a subscription
type BeforeSubscriptionUpdateHook func(ctx context.Context, subID xid.ID, req *core.UpdateSubscriptionRequest) error

// AfterSubscriptionUpdateHook is called after updating a subscription
type AfterSubscriptionUpdateHook func(ctx context.Context, subscription *core.Subscription) error

// BeforeSubscriptionCancelHook is called before canceling a subscription
type BeforeSubscriptionCancelHook func(ctx context.Context, subID xid.ID, immediate bool) error

// AfterSubscriptionCancelHook is called after canceling a subscription
type AfterSubscriptionCancelHook func(ctx context.Context, subID xid.ID) error

// BeforeSubscriptionPauseHook is called before pausing a subscription
type BeforeSubscriptionPauseHook func(ctx context.Context, subID xid.ID) error

// AfterSubscriptionPauseHook is called after pausing a subscription
type AfterSubscriptionPauseHook func(ctx context.Context, subID xid.ID) error

// BeforeSubscriptionResumeHook is called before resuming a subscription
type BeforeSubscriptionResumeHook func(ctx context.Context, subID xid.ID) error

// AfterSubscriptionResumeHook is called after resuming a subscription
type AfterSubscriptionResumeHook func(ctx context.Context, subID xid.ID) error

// OnSubscriptionStatusChangeHook is called when a subscription status changes
type OnSubscriptionStatusChangeHook func(ctx context.Context, subID xid.ID, oldStatus, newStatus core.SubscriptionStatus) error

// OnTrialEndingHook is called when a trial is about to end (typically 3 days before)
type OnTrialEndingHook func(ctx context.Context, subID xid.ID, daysRemaining int) error

// OnTrialEndedHook is called when a trial has ended
type OnTrialEndedHook func(ctx context.Context, subID xid.ID) error

// OnPaymentSuccessHook is called when a payment succeeds
type OnPaymentSuccessHook func(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string) error

// OnPaymentFailedHook is called when a payment fails
type OnPaymentFailedHook func(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string, failureReason string) error

// OnInvoiceCreatedHook is called when an invoice is created
type OnInvoiceCreatedHook func(ctx context.Context, invoiceID xid.ID) error

// OnInvoicePaidHook is called when an invoice is paid
type OnInvoicePaidHook func(ctx context.Context, invoiceID xid.ID) error

// OnUsageLimitApproachingHook is called when usage is approaching a limit
type OnUsageLimitApproachingHook func(ctx context.Context, orgID xid.ID, metricKey string, percentUsed float64) error

// OnUsageLimitExceededHook is called when usage exceeds a limit
type OnUsageLimitExceededHook func(ctx context.Context, orgID xid.ID, metricKey string, currentUsage, limit int64) error

// BeforePlanChangeHook is called before changing plans
type BeforePlanChangeHook func(ctx context.Context, subID, oldPlanID, newPlanID xid.ID) error

// AfterPlanChangeHook is called after changing plans
type AfterPlanChangeHook func(ctx context.Context, subID xid.ID, oldPlanID, newPlanID xid.ID) error

// BeforeAddOnAttachHook is called before attaching an add-on
type BeforeAddOnAttachHook func(ctx context.Context, subID, addOnID xid.ID, quantity int) error

// AfterAddOnAttachHook is called after attaching an add-on
type AfterAddOnAttachHook func(ctx context.Context, subID, addOnID xid.ID) error

// BeforeAddOnDetachHook is called before detaching an add-on
type BeforeAddOnDetachHook func(ctx context.Context, subID, addOnID xid.ID) error

// AfterAddOnDetachHook is called after detaching an add-on
type AfterAddOnDetachHook func(ctx context.Context, subID, addOnID xid.ID) error

// SubscriptionHookRegistry manages subscription-specific hooks
type SubscriptionHookRegistry struct {
	beforeSubscriptionCreate []BeforeSubscriptionCreateHook
	afterSubscriptionCreate  []AfterSubscriptionCreateHook
	beforeSubscriptionUpdate []BeforeSubscriptionUpdateHook
	afterSubscriptionUpdate  []AfterSubscriptionUpdateHook
	beforeSubscriptionCancel []BeforeSubscriptionCancelHook
	afterSubscriptionCancel  []AfterSubscriptionCancelHook
	beforeSubscriptionPause  []BeforeSubscriptionPauseHook
	afterSubscriptionPause   []AfterSubscriptionPauseHook
	beforeSubscriptionResume []BeforeSubscriptionResumeHook
	afterSubscriptionResume  []AfterSubscriptionResumeHook
	onStatusChange           []OnSubscriptionStatusChangeHook
	onTrialEnding            []OnTrialEndingHook
	onTrialEnded             []OnTrialEndedHook
	onPaymentSuccess         []OnPaymentSuccessHook
	onPaymentFailed          []OnPaymentFailedHook
	onInvoiceCreated         []OnInvoiceCreatedHook
	onInvoicePaid            []OnInvoicePaidHook
	onUsageLimitApproaching  []OnUsageLimitApproachingHook
	onUsageLimitExceeded     []OnUsageLimitExceededHook
	beforePlanChange         []BeforePlanChangeHook
	afterPlanChange          []AfterPlanChangeHook
	beforeAddOnAttach        []BeforeAddOnAttachHook
	afterAddOnAttach         []AfterAddOnAttachHook
	beforeAddOnDetach        []BeforeAddOnDetachHook
	afterAddOnDetach         []AfterAddOnDetachHook
}

// NewSubscriptionHookRegistry creates a new hook registry
func NewSubscriptionHookRegistry() *SubscriptionHookRegistry {
	return &SubscriptionHookRegistry{}
}

// Registration methods

func (r *SubscriptionHookRegistry) RegisterBeforeSubscriptionCreate(hook BeforeSubscriptionCreateHook) {
	r.beforeSubscriptionCreate = append(r.beforeSubscriptionCreate, hook)
}

func (r *SubscriptionHookRegistry) RegisterAfterSubscriptionCreate(hook AfterSubscriptionCreateHook) {
	r.afterSubscriptionCreate = append(r.afterSubscriptionCreate, hook)
}

func (r *SubscriptionHookRegistry) RegisterBeforeSubscriptionUpdate(hook BeforeSubscriptionUpdateHook) {
	r.beforeSubscriptionUpdate = append(r.beforeSubscriptionUpdate, hook)
}

func (r *SubscriptionHookRegistry) RegisterAfterSubscriptionUpdate(hook AfterSubscriptionUpdateHook) {
	r.afterSubscriptionUpdate = append(r.afterSubscriptionUpdate, hook)
}

func (r *SubscriptionHookRegistry) RegisterBeforeSubscriptionCancel(hook BeforeSubscriptionCancelHook) {
	r.beforeSubscriptionCancel = append(r.beforeSubscriptionCancel, hook)
}

func (r *SubscriptionHookRegistry) RegisterAfterSubscriptionCancel(hook AfterSubscriptionCancelHook) {
	r.afterSubscriptionCancel = append(r.afterSubscriptionCancel, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnStatusChange(hook OnSubscriptionStatusChangeHook) {
	r.onStatusChange = append(r.onStatusChange, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnTrialEnding(hook OnTrialEndingHook) {
	r.onTrialEnding = append(r.onTrialEnding, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnTrialEnded(hook OnTrialEndedHook) {
	r.onTrialEnded = append(r.onTrialEnded, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnPaymentSuccess(hook OnPaymentSuccessHook) {
	r.onPaymentSuccess = append(r.onPaymentSuccess, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnPaymentFailed(hook OnPaymentFailedHook) {
	r.onPaymentFailed = append(r.onPaymentFailed, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnUsageLimitApproaching(hook OnUsageLimitApproachingHook) {
	r.onUsageLimitApproaching = append(r.onUsageLimitApproaching, hook)
}

func (r *SubscriptionHookRegistry) RegisterOnUsageLimitExceeded(hook OnUsageLimitExceededHook) {
	r.onUsageLimitExceeded = append(r.onUsageLimitExceeded, hook)
}

func (r *SubscriptionHookRegistry) RegisterBeforePlanChange(hook BeforePlanChangeHook) {
	r.beforePlanChange = append(r.beforePlanChange, hook)
}

func (r *SubscriptionHookRegistry) RegisterAfterPlanChange(hook AfterPlanChangeHook) {
	r.afterPlanChange = append(r.afterPlanChange, hook)
}

// Execution methods

func (r *SubscriptionHookRegistry) ExecuteBeforeSubscriptionCreate(ctx context.Context, orgID, planID xid.ID) error {
	for _, hook := range r.beforeSubscriptionCreate {
		if err := hook(ctx, orgID, planID); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteAfterSubscriptionCreate(ctx context.Context, sub *core.Subscription) error {
	for _, hook := range r.afterSubscriptionCreate {
		if err := hook(ctx, sub); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteBeforeSubscriptionUpdate(ctx context.Context, subID xid.ID, req *core.UpdateSubscriptionRequest) error {
	for _, hook := range r.beforeSubscriptionUpdate {
		if err := hook(ctx, subID, req); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteAfterSubscriptionUpdate(ctx context.Context, sub *core.Subscription) error {
	for _, hook := range r.afterSubscriptionUpdate {
		if err := hook(ctx, sub); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteBeforeSubscriptionCancel(ctx context.Context, subID xid.ID, immediate bool) error {
	for _, hook := range r.beforeSubscriptionCancel {
		if err := hook(ctx, subID, immediate); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteAfterSubscriptionCancel(ctx context.Context, subID xid.ID) error {
	for _, hook := range r.afterSubscriptionCancel {
		if err := hook(ctx, subID); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnStatusChange(ctx context.Context, subID xid.ID, oldStatus, newStatus core.SubscriptionStatus) error {
	for _, hook := range r.onStatusChange {
		if err := hook(ctx, subID, oldStatus, newStatus); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnTrialEnding(ctx context.Context, subID xid.ID, daysRemaining int) error {
	for _, hook := range r.onTrialEnding {
		if err := hook(ctx, subID, daysRemaining); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnPaymentSuccess(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string) error {
	for _, hook := range r.onPaymentSuccess {
		if err := hook(ctx, subID, invoiceID, amount, currency); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnPaymentFailed(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string, reason string) error {
	for _, hook := range r.onPaymentFailed {
		if err := hook(ctx, subID, invoiceID, amount, currency, reason); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnUsageLimitApproaching(ctx context.Context, orgID xid.ID, metricKey string, percentUsed float64) error {
	for _, hook := range r.onUsageLimitApproaching {
		if err := hook(ctx, orgID, metricKey, percentUsed); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteOnUsageLimitExceeded(ctx context.Context, orgID xid.ID, metricKey string, currentUsage, limit int64) error {
	for _, hook := range r.onUsageLimitExceeded {
		if err := hook(ctx, orgID, metricKey, currentUsage, limit); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteBeforePlanChange(ctx context.Context, subID, oldPlanID, newPlanID xid.ID) error {
	for _, hook := range r.beforePlanChange {
		if err := hook(ctx, subID, oldPlanID, newPlanID); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubscriptionHookRegistry) ExecuteAfterPlanChange(ctx context.Context, subID, oldPlanID, newPlanID xid.ID) error {
	for _, hook := range r.afterPlanChange {
		if err := hook(ctx, subID, oldPlanID, newPlanID); err != nil {
			return err
		}
	}
	return nil
}

