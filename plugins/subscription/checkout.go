package subscription

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
)

// CheckoutProvider takes payment before a tenant moves onto a paid plan.
//
// When one is configured, subscribing or changing to a plan with a price
// answers with the URL Checkout returns instead of changing anything. Send
// the customer there; once your payment provider confirms, complete the
// change from your webhook handler with Plugin.Service().Subscribe or
// ChangePlan, passing the same tenant, plan and quantity. Free plans,
// cancellations and resumes never go through the provider.
type CheckoutProvider interface {
	Checkout(ctx context.Context, req CheckoutRequest) (url string, err error)
}

// CheckoutRequest describes the change a checkout pays for.
type CheckoutRequest struct {
	TenantID string
	AppID    string
	// Plan is the plan being bought.
	Plan *plan.Plan
	// Current is the tenant's subscription, nil for a new one.
	Current *lsub.Subscription
	// Quantity holds seat counts by feature key, nil to keep the current ones.
	Quantity map[string]int64
	// ReturnURL and CancelURL are where the client asked to land after
	// paying or backing out. Either may be empty.
	ReturnURL string
	CancelURL string
	// UserID is who started the checkout.
	UserID id.UserID
}

// PortalProvider is optional. A CheckoutProvider that also implements it
// backs POST /v1/billing/portal: a hosted page where customers update their
// payment details and see their invoices.
type PortalProvider interface {
	Portal(ctx context.Context, tenantID, appID, returnURL string) (url string, err error)
}

// SetCheckoutProvider sets the checkout provider after construction, for
// hosts that build it once the engine is up. nil turns checkout off.
func (p *Plugin) SetCheckoutProvider(cp CheckoutProvider) {
	p.config.Checkout = cp
}

// needsCheckout reports whether moving onto pl has to be paid for first.
func (p *Plugin) needsCheckout(pl *plan.Plan) bool {
	return p.config.Checkout != nil && pl.Pricing != nil && pl.Pricing.BaseAmount.Amount > 0
}

// startCheckout asks the provider for a checkout URL. A provider error is
// logged in full and answered with a generic 502: its text can carry keys
// or account details the client has no business seeing.
func (p *Plugin) startCheckout(ctx forge.Context, c planChange) (*SubscriptionMutationResponse, error) {
	userID, _ := middleware.UserIDFrom(ctx.Context())
	url, err := p.config.Checkout.Checkout(ctx.Context(), CheckoutRequest{
		TenantID:  c.tenantID,
		AppID:     c.appID,
		Plan:      c.plan,
		Current:   c.current,
		Quantity:  c.quantity,
		ReturnURL: c.returnURL,
		CancelURL: c.cancelURL,
		UserID:    userID,
	})
	if err != nil || url == "" {
		if err == nil {
			err = fmt.Errorf("provider returned no URL")
		}
		p.logProviderError("checkout", c.tenantID, err)
		return nil, forge.NewHTTPError(http.StatusBadGateway, "checkout failed")
	}
	return &SubscriptionMutationResponse{CheckoutURL: url}, nil
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

func (p *Plugin) registerPortalRoute(router forge.Router) error {
	group := router.Group(p.config.PathPrefix+"/portal",
		forge.WithGroupTags("Billing"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)
	return group.POST("", p.handleCreatePortalSession,
		forge.WithSummary("Open the billing portal"),
		forge.WithDescription("Returns a URL to a hosted page where you manage payment details and invoices. Needs a checkout provider that offers a portal."),
		forge.WithOperationID("createBillingPortalSession"),
		forge.WithResponseSchema(http.StatusOK, "Portal URL", PortalResponse{}),
		forge.WithErrorResponses(),
	)
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
			err = fmt.Errorf("provider returned no URL")
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
