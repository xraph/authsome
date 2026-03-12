package subscription

import (
	"context"
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

// GetPlan retrieves a plan by ID.
func (s *Service) GetPlan(ctx context.Context, planID ledgerid.PlanID) (*plan.Plan, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.GetPlan(ctx, planID)
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

// Subscribe creates a new subscription for a tenant on a plan.
func (s *Service) Subscribe(ctx context.Context, tenantID string, planID ledgerid.PlanID, appID string) (*subscription.Subscription, error) {
	if s.ledger == nil {
		return nil, fmt.Errorf("subscription: ledger not available")
	}

	// Resolve trial days from settings.
	trialDays, _ := settings.Get(ctx, s.settings, SettingTrialDays, settings.ResolveOpts{AppID: appID}) //nolint:errcheck // best-effort settings

	now := time.Now()
	sub := &subscription.Subscription{
		TenantID:           tenantID,
		PlanID:             planID,
		Status:             subscription.StatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		AppID:              appID,
	}

	if trialDays > 0 {
		sub.Status = subscription.StatusTrialing
		trialEnd := now.AddDate(0, 0, trialDays)
		sub.TrialStart = &now
		sub.TrialEnd = &trialEnd
		sub.CurrentPeriodEnd = trialEnd
	}

	if err := s.ledger.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// CancelSubscription cancels a subscription.
func (s *Service) CancelSubscription(ctx context.Context, subID ledgerid.SubscriptionID, immediately bool) error {
	if s.ledger == nil {
		return fmt.Errorf("subscription: ledger not available")
	}
	return s.ledger.CancelSubscription(ctx, subID, immediately)
}

// ChangePlan changes a subscription to a different plan.
func (s *Service) ChangePlan(ctx context.Context, subID ledgerid.SubscriptionID, newPlanID ledgerid.PlanID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}

	sub, err := s.ledger.GetSubscription(ctx, subID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}

	sub.PlanID = newPlanID
	sub.Touch()

	return s.ledgerStore.UpdateSubscription(ctx, sub)
}

// PauseSubscription pauses a subscription.
func (s *Service) PauseSubscription(ctx context.Context, subID ledgerid.SubscriptionID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	sub, err := s.ledger.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	sub.Status = subscription.StatusPaused
	sub.Touch()
	return s.ledgerStore.UpdateSubscription(ctx, sub)
}

// ResumeSubscription resumes a paused subscription.
func (s *Service) ResumeSubscription(ctx context.Context, subID ledgerid.SubscriptionID) error {
	if s.ledgerStore == nil {
		return fmt.Errorf("subscription: ledger store not available")
	}
	sub, err := s.ledger.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	sub.Status = subscription.StatusActive
	sub.Touch()
	return s.ledgerStore.UpdateSubscription(ctx, sub)
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
