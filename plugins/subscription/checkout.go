package subscription

import (
	"context"
	"maps"
	"net/http"
	"sync"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
)

// CheckoutProvider decides whether a self-service plan change has to be paid
// for before it happens.
//
// When one is configured, every subscribe and every plan or seat change goes
// through Checkout first. Answer with a URL to send the customer to your
// payment page; nothing changes until your webhook calls
// Plugin.CompleteCheckout. Answer with an empty URL when there's nothing to
// pay (a free plan, a downgrade, fewer seats) and the change is applied
// straight away. Cancellations and resumes never go through the provider:
// keep your payment provider in step by listening for the
// subscription.canceled, subscription.plan_changed and subscription.resumed
// events.
type CheckoutProvider interface {
	Checkout(ctx context.Context, req CheckoutRequest) (CheckoutResult, error)
}

// CheckoutResult is a CheckoutProvider's answer.
type CheckoutResult struct {
	// URL is where to send the customer to pay. Empty applies the change now.
	URL string
}

// CheckoutRequest describes one plan change.
type CheckoutRequest struct {
	TenantID string
	AppID    string
	// Plan is the plan being moved to.
	Plan *plan.Plan
	// Current is the tenant's subscription, nil for a new one, and
	// CurrentPlan its plan.
	Current     *lsub.Subscription
	CurrentPlan *plan.Plan
	// Quantity holds seat counts by feature key, nil to keep the current ones.
	Quantity map[string]int64
	// ReturnURL and CancelURL are where the client asked to land after
	// paying or backing out. Either may be empty.
	ReturnURL string
	CancelURL string
	// UserID is who asked for the change.
	UserID id.UserID
}

// PortalProvider is optional. A CheckoutProvider that also implements it
// backs POST /v1/billing/portal: a hosted page where customers update their
// payment details and see their invoices.
type PortalProvider interface {
	Portal(ctx context.Context, tenantID, appID, returnURL string) (url string, err error)
}

// SetCheckoutProvider sets the checkout provider after construction. Call it
// before the engine starts serving; nil turns checkout off.
func (p *Plugin) SetCheckoutProvider(cp CheckoutProvider) {
	p.config.Checkout = cp
}

// startCheckout asks the provider about a change and answers its URL, empty
// to apply the change now. A provider error is logged in full and answered
// with a 502: its text can carry keys or account details the client has no
// business seeing.
func (p *Plugin) startCheckout(ctx forge.Context, c planChange) (string, error) {
	userID, _ := middleware.UserIDFrom(ctx.Context())
	req := CheckoutRequest{
		TenantID:  c.tenantID,
		AppID:     c.appID,
		Plan:      c.plan,
		Current:   c.current,
		Quantity:  c.quantity,
		ReturnURL: c.returnURL,
		CancelURL: c.cancelURL,
		UserID:    userID,
	}
	if c.current != nil {
		if currentPlan, err := p.service.GetPlan(ctx.Context(), c.current.PlanID); err == nil {
			req.CurrentPlan = currentPlan
		}
	}

	result, err := p.config.Checkout.Checkout(ctx.Context(), req)
	if err != nil {
		p.logProviderError("checkout", c.tenantID, err)
		return "", forge.NewHTTPError(http.StatusBadGateway, "checkout failed")
	}
	return result.URL, nil
}

// CompleteCheckout applies a plan change once its payment has landed. Call
// it from your payment provider's webhook with the CheckoutRequest you got
// in Checkout (keep what you need from it in the checkout session's
// metadata).
//
// It is safe to call more than once for the same payment: when the tenant is
// already on that plan with those seats, it changes nothing and returns the
// subscription. It moves the tenant's current subscription when there is
// one, even if a different one was current at checkout, and records the same
// audit entry and event the self-service routes do.
func (p *Plugin) CompleteCheckout(ctx context.Context, req CheckoutRequest) (*lsub.Subscription, error) {
	unlock := p.tenantLock(req.TenantID, req.AppID)
	defer unlock()

	current, err := p.service.CurrentSubscription(ctx, req.TenantID, req.AppID)
	if err != nil {
		return nil, err
	}
	quantity := req.Quantity
	if current != nil {
		if current.PlanID == req.Plan.ID && (quantity == nil || maps.Equal(quantity, current.Quantity)) {
			return current, nil
		}
		if quantity == nil {
			quantity = carriedQuantity(current.Quantity, req.Plan)
		}
	}
	return p.commitPlanChange(ctx, planChange{
		tenantID: req.TenantID, appID: req.AppID, current: current, plan: req.Plan, quantity: quantity,
	}, req.UserID.String())
}

// commitPlanChange subscribes the tenant, or moves its subscription to the
// plan and seat count, and records the audit entry and event.
func (p *Plugin) commitPlanChange(ctx context.Context, c planChange, actor string) (*lsub.Subscription, error) {
	if c.current == nil {
		sub, err := p.service.Subscribe(ctx, c.tenantID, c.plan.ID, c.appID, c.quantity)
		if err != nil {
			return nil, err
		}
		p.audit(ctx, "subscription.create", "subscription", sub.ID.String(), actor, c.tenantID, bridge.OutcomeSuccess)
		p.relayEvent(ctx, "subscription.created", c.tenantID, map[string]string{"plan_slug": c.plan.Slug})
		return sub, nil
	}

	sub, err := p.service.ChangePlan(ctx, c.current.ID, c.plan.ID, c.quantity)
	if err != nil {
		return nil, err
	}
	p.audit(ctx, "subscription.plan_changed", "subscription", sub.ID.String(), actor, c.tenantID, bridge.OutcomeSuccess)
	p.relayEvent(ctx, "subscription.plan_changed", c.tenantID, map[string]string{"plan_slug": c.plan.Slug})
	return sub, nil
}

// tenantLock serializes subscription creation for one tenant in one app, so
// two requests can't both see "no subscription" and both create one. It
// holds within this process only: ledger has no per-tenant uniqueness, so
// replicas behind a load balancer can still race. Route a tenant's billing
// writes to one replica, or check for duplicates in your webhook, if that
// matters to you.
func (p *Plugin) tenantLock(tenantID, appID string) (unlock func()) {
	v, _ := p.tenantLocks.LoadOrStore(tenantID+"\x00"+appID, &sync.Mutex{})
	mu := v.(*sync.Mutex) //nolint:errcheck,forcetypeassert // only *sync.Mutex is ever stored
	mu.Lock()
	return mu.Unlock
}

// PortalRequest opens the billing portal for the caller's tenant.
type PortalRequest struct {
	ReturnURL string `json:"return_url,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
}

// PortalResponse carries the portal URL to send the customer to.
type PortalResponse struct {
	URL string `json:"url"`
}

func (p *Plugin) registerPortalRoute(router forge.Router, opts ...forge.RouteOption) error {
	group := router.Group(p.config.PathPrefix+"/portal",
		forge.WithGroupTags("Billing"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)
	return group.POST("", p.handleCreatePortalSession, append([]forge.RouteOption{
		forge.WithSummary("Open the billing portal"),
		forge.WithDescription("Returns a URL to a hosted page where you manage payment details and invoices. Needs a checkout provider that offers a portal."),
		forge.WithOperationID("createBillingPortalSession"),
		forge.WithResponseSchema(http.StatusOK, "Portal URL", PortalResponse{}),
		forge.WithErrorResponses(),
	}, opts...)...)
}

func (p *Plugin) handleCreatePortalSession(ctx forge.Context, req *PortalRequest) (*PortalResponse, error) {
	appID, tenantID, err := p.writeScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	portal, ok := p.config.Checkout.(PortalProvider)
	if !ok {
		return nil, forge.NotFound("billing portal is not configured")
	}

	url, err := portal.Portal(ctx.Context(), tenantID, appID, req.ReturnURL)
	if err != nil || url == "" {
		if err == nil {
			err = errNoPortalURL
		}
		p.logProviderError("billing portal", tenantID, err)
		return nil, forge.NewHTTPError(http.StatusBadGateway, "billing portal failed")
	}
	return &PortalResponse{URL: url}, nil
}

// logProviderError keeps a provider's own error text in the logs, where an
// operator can act on it, and out of the response.
func (p *Plugin) logProviderError(what, tenantID string, err error) {
	if p.logger == nil {
		return
	}
	p.logger.Error("subscription: "+what+" provider failed",
		log.String("tenant_id", tenantID), log.Error(err))
}
