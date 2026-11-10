package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"

	"github.com/xraph/ledger"
	"github.com/xraph/ledger/coupon"
	"github.com/xraph/ledger/entitlement"
	"github.com/xraph/ledger/feature"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	ledgerstore "github.com/xraph/ledger/store"
	"github.com/xraph/ledger/subscription"
)

// Service wraps ledger operations with AuthSome context.
type Service struct {
	ledger      *ledger.Ledger
	ledgerStore ledgerstore.Store
	ledgerBrdg  bridge.Ledger
	authStore   store.Store
	settings    *settings.Manager
	logger      log.Logger
}

// SetLedgerStore sets the ledger store directly, bypassing duck-type discovery.
func (s *Service) SetLedgerStore(st ledgerstore.Store) {
	s.ledgerStore = st
}

// ──────────────────────────────────────────────────
// Plan operations
// ──────────────────────────────────────────────────

// ListPlans returns all plans for the given app.
func (s *Service) ListPlans(ctx context.Context, appID string) ([]*plan.Plan, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListPlans(ctx, appID, plan.ListOpts{})
}

// ListPlansWithOpts returns the app's plans filtered by opts.
func (s *Service) ListPlansWithOpts(ctx context.Context, appID string, opts plan.ListOpts) ([]*plan.Plan, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListPlans(ctx, appID, opts)
}

// GetPlan retrieves a plan by ID.
func (s *Service) GetPlan(ctx context.Context, planID ledgerid.PlanID) (*plan.Plan, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GetPlan(ctx, planID)
}

// GetPlanBySlug retrieves a plan by its slug within an app.
func (s *Service) GetPlanBySlug(ctx context.Context, slug, appID string) (*plan.Plan, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GetPlanBySlug(ctx, slug, appID)
}

// CreatePlan creates a new billing plan.
func (s *Service) CreatePlan(ctx context.Context, p *plan.Plan) error {
	if s.ledger == nil {
		return fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.CreatePlan(ctx, p)
}

// UpdatePlan updates an existing plan.
func (s *Service) UpdatePlan(ctx context.Context, p *plan.Plan) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	p.Touch()
	return s.ledgerStore.UpdatePlan(ctx, p)
}

// ArchivePlan archives a plan by ID.
func (s *Service) ArchivePlan(ctx context.Context, planID ledgerid.PlanID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ArchivePlan(ctx, planID)
}

// ActivatePlan transitions a draft plan to active status.
func (s *Service) ActivatePlan(ctx context.Context, planID ledgerid.PlanID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	p, err := s.ledger.GetPlan(ctx, planID)
	if err != nil {
		return err
	}
	p.Status = plan.StatusActive
	p.Touch()
	return s.ledgerStore.UpdatePlan(ctx, p)
}

// CountSubscribers counts active subscriptions for a plan.
func (s *Service) CountSubscribers(ctx context.Context, planID ledgerid.PlanID, appID string) int {
	if s.ledgerStore == nil {
		return 0
	}
	subs, err := s.ledgerStore.ListSubscriptions(ctx, "", appID, subscription.ListOpts{Status: subscription.StatusActive})
	if err != nil {
		return 0
	}
	count := 0
	for _, sub := range subs {
		if sub.PlanID == planID {
			count++
		}
	}
	return count
}

// ──────────────────────────────────────────────────
// Subscription operations
// ──────────────────────────────────────────────────

// GetSubscription retrieves a subscription by ID.
func (s *Service) GetSubscription(ctx context.Context, subID ledgerid.SubscriptionID) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GetSubscription(ctx, subID)
}

// GetActiveSubscription retrieves the active subscription for a tenant.
func (s *Service) GetActiveSubscription(ctx context.Context, tenantID, appID string) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GetActiveSubscription(ctx, tenantID, appID)
}

// ListSubscriptions lists subscriptions for a tenant.
func (s *Service) ListSubscriptions(ctx context.Context, tenantID, appID string, opts subscription.ListOpts) ([]*subscription.Subscription, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListSubscriptions(ctx, tenantID, appID, opts)
}

// Subscribe creates a new subscription for a tenant on a plan. Ledger picks
// the trial from the plan's trial_days and the first period from the plan's
// billing period. quantity sets seat counts for the plan's seat features and
// may be nil.
func (s *Service) Subscribe(ctx context.Context, tenantID string, planID ledgerid.PlanID, appID string, quantity map[string]int64) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}

	sub := &subscription.Subscription{
		TenantID: tenantID,
		PlanID:   planID,
		AppID:    appID,
		Quantity: quantity,
	}
	if err := s.ledger.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// CancelSubscription cancels a subscription, now or at the end of its period.
func (s *Service) CancelSubscription(ctx context.Context, subID ledgerid.SubscriptionID, immediately bool) error {
	if s.ledger == nil {
		return fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.CancelSubscription(ctx, subID, immediately)
}

// ChangePlan moves a subscription to another plan through ledger, which
// refuses a canceled or expired subscription and a plan outside its app. A nil
// quantity keeps the current seat counts.
func (s *Service) ChangePlan(ctx context.Context, subID ledgerid.SubscriptionID, newPlanID ledgerid.PlanID, quantity map[string]int64) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.ChangePlan(ctx, subID, newPlanID, quantity)
}

// PauseSubscription pauses an active or trialing subscription.
func (s *Service) PauseSubscription(ctx context.Context, subID ledgerid.SubscriptionID) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.PauseSubscription(ctx, subID)
}

// ResumeSubscription resumes a paused subscription. Ledger stretches the
// period and any running trial by the length of the pause, and leaves a
// subscription that ended while paused alone.
func (s *Service) ResumeSubscription(ctx context.Context, subID ledgerid.SubscriptionID) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.ResumeSubscription(ctx, subID)
}

// ErrNoScheduledCancel is returned by KeepSubscription when the subscription
// has no cancellation waiting at the end of its period.
var ErrNoScheduledCancel = errors.New("subscription: no cancellation is scheduled")

// KeepSubscription takes back a cancellation scheduled for the end of the
// period, so the subscription renews as usual. A subscription that has already
// ended can't be kept.
func (s *Service) KeepSubscription(ctx context.Context, subID ledgerid.SubscriptionID) (*subscription.Subscription, error) {
	if s.ledger == nil || s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	sub, err := s.ledger.GetSubscription(ctx, subID)
	if err != nil {
		return nil, err
	}
	switch sub.Status {
	case subscription.StatusCanceled:
		return nil, ledger.ErrSubscriptionCanceled
	case subscription.StatusExpired:
		return nil, ledger.ErrSubscriptionExpired
	}
	if sub.CancelAt == nil {
		return nil, ErrNoScheduledCancel
	}
	sub.CancelAt = nil
	sub.Touch()
	if err := s.ledgerStore.UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// ──────────────────────────────────────────────────
// Invoice operations
// ──────────────────────────────────────────────────

// GetInvoice retrieves an invoice by ID.
func (s *Service) GetInvoice(ctx context.Context, invID ledgerid.InvoiceID) (*invoice.Invoice, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.GetInvoice(ctx, invID)
}

// ListInvoices lists invoices for a tenant.
func (s *Service) ListInvoices(ctx context.Context, tenantID, appID string) ([]*invoice.Invoice, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListInvoices(ctx, tenantID, appID, invoice.ListOpts{})
}

// ListAllInvoices lists all invoices for an app.
func (s *Service) ListAllInvoices(ctx context.Context, appID string) ([]*invoice.Invoice, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListInvoices(ctx, "", appID, invoice.ListOpts{})
}

// GenerateInvoice generates an invoice for the current billing period.
func (s *Service) GenerateInvoice(ctx context.Context, subID ledgerid.SubscriptionID) (*invoice.Invoice, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GenerateInvoice(ctx, subID)
}

// MarkInvoicePaid marks an invoice as paid.
func (s *Service) MarkInvoicePaid(ctx context.Context, invID ledgerid.InvoiceID, paymentRef string) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.MarkInvoicePaid(ctx, invID, time.Now(), paymentRef)
}

// MarkInvoiceVoided voids an invoice.
func (s *Service) MarkInvoiceVoided(ctx context.Context, invID ledgerid.InvoiceID, reason string) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.MarkInvoiceVoided(ctx, invID, reason)
}

// ──────────────────────────────────────────────────
// Coupon operations
// ──────────────────────────────────────────────────

// ListCoupons lists all coupons for an app.
func (s *Service) ListCoupons(ctx context.Context, appID string) ([]*coupon.Coupon, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListCoupons(ctx, appID, coupon.ListOpts{})
}

// GetCoupon retrieves a coupon by code.
func (s *Service) GetCoupon(ctx context.Context, code, appID string) (*coupon.Coupon, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.GetCoupon(ctx, code, appID)
}

// GetCouponByID retrieves a coupon by ID.
func (s *Service) GetCouponByID(ctx context.Context, couponID ledgerid.CouponID) (*coupon.Coupon, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.GetCouponByID(ctx, couponID)
}

// CreateCoupon creates a new coupon.
func (s *Service) CreateCoupon(ctx context.Context, c *coupon.Coupon) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.CreateCoupon(ctx, c)
}

// DeleteCoupon deletes a coupon.
func (s *Service) DeleteCoupon(ctx context.Context, couponID ledgerid.CouponID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.DeleteCoupon(ctx, couponID)
}

// ──────────────────────────────────────────────────
// Entitlement operations
// ──────────────────────────────────────────────────

// CheckEntitlement checks whether a tenant can use a feature.
func (s *Service) CheckEntitlement(ctx context.Context, featureKey string) (*entitlement.Result, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.Entitled(ctx, featureKey)
}

// ──────────────────────────────────────────────────
// Usage operations
// ──────────────────────────────────────────────────

// UsageSummary holds feature usage data for display.
type UsageSummary struct {
	FeatureKey  string
	FeatureName string
	FeatureType string
	Used        int64
	Limit       int64
	Remaining   int64
	Period      string
}

// GetUsageSummary returns usage breakdown for the active subscription.
func (s *Service) GetUsageSummary(ctx context.Context, tenantID, appID string) ([]UsageSummary, error) {
	if s.ledger == nil || s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}

	sub, err := s.ledger.GetActiveSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}

	p, err := s.ledger.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	summaries := make([]UsageSummary, 0, len(p.Features))
	for _, f := range p.Features {
		summary := UsageSummary{
			FeatureKey:  f.Key,
			FeatureName: f.Name,
			FeatureType: string(f.Type),
			Limit:       f.Limit,
			Period:      string(f.Period),
		}

		if f.Type == plan.FeatureMetered || f.Type == plan.FeatureSeat {
			used, err := s.ledgerStore.Aggregate(ctx, tenantID, appID, f.Key, f.Period)
			if err != nil {
				if s.logger != nil {
					s.logger.Warn("subscription: failed to aggregate usage",
						log.String("feature", f.Key),
						log.Error(err),
					)
				}
				continue
			}
			summary.Used = used
			if f.Limit > 0 {
				summary.Remaining = max(0, f.Limit-used)
			} else {
				summary.Remaining = -1 // unlimited
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// ──────────────────────────────────────────────────
// Feature catalog operations
// ──────────────────────────────────────────────────

// ListCatalogFeatures lists all catalog features for an app.
func (s *Service) ListCatalogFeatures(ctx context.Context, appID string) ([]*feature.Feature, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ListFeatures(ctx, appID, feature.ListOpts{})
}

// GetCatalogFeature retrieves a catalog feature by ID.
func (s *Service) GetCatalogFeature(ctx context.Context, featureID ledgerid.FeatureID) (*feature.Feature, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.GetFeature(ctx, featureID)
}

// GetCatalogFeatureByKey retrieves a catalog feature by key.
func (s *Service) GetCatalogFeatureByKey(ctx context.Context, key, appID string) (*feature.Feature, error) {
	if s.ledgerStore == nil {
		return nil, fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.GetFeatureByKey(ctx, key, appID)
}

// CreateCatalogFeature creates a new catalog feature.
func (s *Service) CreateCatalogFeature(ctx context.Context, f *feature.Feature) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.CreateFeature(ctx, f)
}

// UpdateCatalogFeature updates a catalog feature.
func (s *Service) UpdateCatalogFeature(ctx context.Context, f *feature.Feature) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.UpdateFeature(ctx, f)
}

// ArchiveCatalogFeature archives a catalog feature.
func (s *Service) ArchiveCatalogFeature(ctx context.Context, featureID ledgerid.FeatureID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.ArchiveFeature(ctx, featureID)
}

// DeleteCatalogFeature deletes a catalog feature.
func (s *Service) DeleteCatalogFeature(ctx context.Context, featureID ledgerid.FeatureID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	return s.ledgerStore.DeleteFeature(ctx, featureID)
}

// ──────────────────────────────────────────────────
// MRR calculation
// ──────────────────────────────────────────────────

// CalculateMRR estimates monthly recurring revenue across all active subscriptions.
func (s *Service) CalculateMRR(ctx context.Context, appID string) string {
	if s.ledgerStore == nil {
		return "0.00"
	}

	subs, err := s.ledgerStore.ListSubscriptions(ctx, "", appID, subscription.ListOpts{Status: subscription.StatusActive})
	if err != nil {
		return "0.00"
	}

	var totalCents int64
	for _, sub := range subs {
		pl, err := s.ledger.GetPlan(ctx, sub.PlanID)
		if err != nil || pl.Pricing == nil {
			continue
		}
		amount := pl.Pricing.BaseAmount.Amount
		if pl.Pricing.BillingPeriod == plan.PeriodYearly {
			amount /= 12
		}
		totalCents += amount
	}

	dollars := totalCents / 100
	cents := totalCents % 100
	return fmt.Sprintf("%d.%02d", dollars, cents)
}
