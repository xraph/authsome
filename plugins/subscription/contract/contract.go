// Package contract wires the subscription plugin's intent surface
// into the forge-dashboard contract registry. The `/plans` and
// `/plans/:id` pages stay declared on the auth contributor; only the
// intent handlers + their declarations move here.
package contract

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	"github.com/xraph/ledger/coupon"
	"github.com/xraph/ledger/feature"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// SubscriptionService is the surface this package needs from the
// subscription plugin. Declared as an interface here so the contract
// subpackage doesn't import plugins/subscription directly (cycle
// avoidance). The parent contract.go satisfies it by passing the
// plugin's Service().
type SubscriptionService interface {
	ListPlans(ctx context.Context, appID string) ([]*plan.Plan, error)
	GetPlan(ctx context.Context, planID ledgerid.PlanID) (*plan.Plan, error)
	CreatePlan(ctx context.Context, p *plan.Plan) error
	UpdatePlan(ctx context.Context, p *plan.Plan) error
	ArchivePlan(ctx context.Context, planID ledgerid.PlanID) error
	ActivatePlan(ctx context.Context, planID ledgerid.PlanID) error
	ListSubscriptions(ctx context.Context, tenantID, appID string, opts subscription.ListOpts) ([]*subscription.Subscription, error)
	GetSubscription(ctx context.Context, subID ledgerid.SubscriptionID) (*subscription.Subscription, error)
	GetActiveSubscription(ctx context.Context, tenantID, appID string) (*subscription.Subscription, error)
	Subscribe(ctx context.Context, tenantID string, planID ledgerid.PlanID, appID string) (*subscription.Subscription, error)
	ChangePlan(ctx context.Context, subID ledgerid.SubscriptionID, newPlanID ledgerid.PlanID) error
	PauseSubscription(ctx context.Context, subID ledgerid.SubscriptionID) error
	ResumeSubscription(ctx context.Context, subID ledgerid.SubscriptionID) error
	CancelSubscription(ctx context.Context, subID ledgerid.SubscriptionID, immediately bool) error
	ListAllInvoices(ctx context.Context, appID string) ([]*invoice.Invoice, error)
	ListInvoices(ctx context.Context, tenantID, appID string) ([]*invoice.Invoice, error)
	GetInvoice(ctx context.Context, invID ledgerid.InvoiceID) (*invoice.Invoice, error)
	GenerateInvoice(ctx context.Context, subID ledgerid.SubscriptionID) (*invoice.Invoice, error)
	MarkInvoicePaid(ctx context.Context, invID ledgerid.InvoiceID, paymentRef string) error
	MarkInvoiceVoided(ctx context.Context, invID ledgerid.InvoiceID, reason string) error
	ListCoupons(ctx context.Context, appID string) ([]*coupon.Coupon, error)
	GetCoupon(ctx context.Context, code, appID string) (*coupon.Coupon, error)
	CreateCoupon(ctx context.Context, c *coupon.Coupon) error
	DeleteCoupon(ctx context.Context, couponID ledgerid.CouponID) error
	ListCatalogFeatures(ctx context.Context, appID string) ([]*feature.Feature, error)
	GetCatalogFeature(ctx context.Context, featureID ledgerid.FeatureID) (*feature.Feature, error)
	CreateCatalogFeature(ctx context.Context, f *feature.Feature) error
	UpdateCatalogFeature(ctx context.Context, f *feature.Feature) error
	ArchiveCatalogFeature(ctx context.Context, featureID ledgerid.FeatureID) error
}

type Deps struct {
	Engine  *authsome.Engine
	Service SubscriptionService
	Usage   func(context.Context, string, string) ([]UsageSummary, error)

	// CanManageBilling reports whether userID holds manage/billing. The
	// plugin passes its own check so both surfaces fail closed the same way.
	// Every intent here refuses a caller it says no to, and a nil func says
	// no to everyone.
	CanManageBilling func(ctx context.Context, userID id.UserID) bool
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("subscription/contract: Engine is required")
	}
	if deps.Service == nil {
		return fmt.Errorf("subscription/contract: Service is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("subscription/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("subscription/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("subscription/contract: register manifest: %w", err)
	}

	const c = "subscription"
	if err := dispatcher.RegisterQuery(d, c, "plans.list", 1, billingManagers(deps, plansListHandler(deps))); err != nil {
		return fmt.Errorf("subscription/contract: register plans.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "plans.detail", 1, billingManagers(deps, plansDetailHandler(deps))); err != nil {
		return fmt.Errorf("subscription/contract: register plans.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "plans.archive", 1, billingManagers(deps, plansArchiveHandler(deps))); err != nil {
		return fmt.Errorf("subscription/contract: register plans.archive: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "plans.activate", 1, billingManagers(deps, plansActivateHandler(deps))); err != nil {
		return fmt.Errorf("subscription/contract: register plans.activate: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "subscriptions.list", 1, billingManagers(deps, subscriptionsListHandler(deps))); err != nil {
		return fmt.Errorf("subscription/contract: register subscriptions.list: %w", err)
	}
	registrations := []func() error{
		func() error {
			return dispatcher.RegisterCommand(d, c, "plans.create", 1, billingManagers(deps, plansCreateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "plans.update", 1, billingManagers(deps, plansUpdateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "subscriptions.all", 1, billingManagers(deps, subscriptionsAllHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "subscriptions.detail", 1, billingManagers(deps, subscriptionsDetailHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "subscriptions.create", 1, billingManagers(deps, subscriptionsCreateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "subscriptions.change", 1, billingManagers(deps, subscriptionsChangeHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "subscriptions.pause", 1, billingManagers(deps, subscriptionsPauseHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "subscriptions.resume", 1, billingManagers(deps, subscriptionsResumeHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "subscriptions.cancel", 1, billingManagers(deps, subscriptionsCancelHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "invoices.list", 1, billingManagers(deps, invoicesListHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "invoices.detail", 1, billingManagers(deps, invoicesDetailHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "invoices.generate", 1, billingManagers(deps, invoicesGenerateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "invoices.markPaid", 1, billingManagers(deps, invoicesMarkPaidHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "invoices.void", 1, billingManagers(deps, invoicesVoidHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "coupons.list", 1, billingManagers(deps, couponsListHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "coupons.create", 1, billingManagers(deps, couponsCreateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "coupons.delete", 1, billingManagers(deps, couponsDeleteHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterQuery(d, c, "features.list", 1, billingManagers(deps, featuresListHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "features.create", 1, billingManagers(deps, featuresCreateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "features.update", 1, billingManagers(deps, featuresUpdateHandler(deps)))
		},
		func() error {
			return dispatcher.RegisterCommand(d, c, "features.archive", 1, billingManagers(deps, featuresArchiveHandler(deps)))
		},
	}
	for _, register := range registrations {
		if err := register(); err != nil {
			return fmt.Errorf("subscription/contract: register billing intent: %w", err)
		}
	}
	return nil
}

// billingManagers puts h behind manage/billing. This contributor is the
// operator's billing console: it lists every tenant's subscriptions and
// invoices and changes them, so membership in a tenant isn't enough to use
// any of it. Register wraps every intent with it, reads and writes alike.
func billingManagers[I, O any](deps Deps, h func(context.Context, I, contract.Principal) (O, error)) func(context.Context, I, contract.Principal) (O, error) {
	return func(ctx context.Context, in I, p contract.Principal) (O, error) {
		if err := requireBillingManage(ctx, deps, p); err != nil {
			var zero O
			return zero, err
		}
		return h(ctx, in, p)
	}
}

// requireBillingManage refuses a principal with no user, and one whose user
// deps.CanManageBilling doesn't clear.
func requireBillingManage(ctx context.Context, deps Deps, p contract.Principal) error {
	if p.User == nil || p.User.Subject == "" {
		return &contract.Error{Code: contract.CodeUnauthenticated, Message: "billing needs a signed-in user"}
	}
	userID, err := id.ParseUserID(p.User.Subject)
	if err != nil {
		return &contract.Error{Code: contract.CodeUnauthenticated, Message: "principal subject is not a valid user id"}
	}
	if deps.CanManageBilling == nil || !deps.CanManageBilling(ctx, userID) {
		return &contract.Error{Code: contract.CodePermissionDenied, Message: "billing needs the manage billing permission"}
	}
	return nil
}
