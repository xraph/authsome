package subscription

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/xraph/forge"
	"github.com/xraph/ledger"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
)

// The tenant routes are the self-service surface: one subscription per
// tenant, addressed by the tenant rather than by ID, so a client never
// carries a subscription ID and can't aim a write at someone else's. The
// tenant comes from the session (see resolveTenant); tenant_id is an
// operator override.

// CurrentSubscriptionRequest reads the caller's tenant's subscription.
type CurrentSubscriptionRequest struct {
	AppID    string `query:"app_id,omitempty"`
	TenantID string `query:"tenant_id,omitempty"`
}

// CurrentSubscriptionResponse carries the subscription, null when the tenant
// has none, and whether the caller may change it.
type CurrentSubscriptionResponse struct {
	Subscription *SubscriptionView `json:"subscription"`
	CanManage    bool              `json:"can_manage"`
}

// SubscribeRequest starts a subscription. Plan is a plan slug or ID.
type SubscribeRequest struct {
	Plan      string `json:"plan"`
	Seats     *int64 `json:"seats,omitempty"`
	ReturnURL string `json:"return_url,omitempty"`
	CancelURL string `json:"cancel_url,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
}

// UpdateSubscriptionRequest changes the plan, the seat count, or both.
type UpdateSubscriptionRequest struct {
	Plan      string `json:"plan,omitempty"`
	Seats     *int64 `json:"seats,omitempty"`
	ReturnURL string `json:"return_url,omitempty"`
	CancelURL string `json:"cancel_url,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
}

// CancelCurrentRequest cancels at the period end (the default) or now.
type CancelCurrentRequest struct {
	When     string `json:"when,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
}

// TenantRequest names nothing but an optional tenant override.
type TenantRequest struct {
	TenantID string `json:"tenant_id,omitempty"`
}

// SubscriptionMutationResponse holds the updated subscription, or a
// checkout URL when a checkout provider takes payment first. Exactly one is
// set.
type SubscriptionMutationResponse struct {
	Subscription *SubscriptionView `json:"subscription,omitempty"`
	CheckoutURL  string            `json:"checkout_url,omitempty"`
}

// ListEntitlementsRequest reads every entitlement of the caller's tenant.
type ListEntitlementsRequest struct {
	AppID    string `query:"app_id,omitempty"`
	TenantID string `query:"tenant_id,omitempty"`
}

// EntitlementsResponse maps each feature on the active plan to its check.
type EntitlementsResponse struct {
	Entitlements map[string]EntitlementResponse `json:"entitlements"`
}

const (
	cancelAtPeriodEnd = "period_end"
	cancelNow         = "now"
)

func (p *Plugin) registerTenantRoutes(router forge.Router) error {
	prefix := p.config.PathPrefix
	group := router.Group(prefix+"/subscription",
		forge.WithGroupTags("Billing"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)

	if err := group.GET("", p.handleGetCurrentSubscription,
		forge.WithSummary("Get your subscription"),
		forge.WithDescription("Returns the subscription of your active organization (or your own in user mode), null when there is none, and whether you may change it."),
		forge.WithOperationID("getCurrentSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Current subscription", CurrentSubscriptionResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := group.POST("", p.handleSubscribe,
		forge.WithSummary("Subscribe to a plan"),
		forge.WithOperationID("subscribe"),
		forge.WithResponseSchema(http.StatusOK, "Subscription or checkout URL", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := group.PATCH("", p.handleUpdateSubscription,
		forge.WithSummary("Change plan or seats"),
		forge.WithOperationID("updateSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Subscription or checkout URL", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := group.POST("/cancel", p.handleCancelCurrentSubscription,
		forge.WithSummary("Cancel your subscription"),
		forge.WithDescription("Cancels at the end of the current period unless when is \"now\"."),
		forge.WithOperationID("cancelCurrentSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Updated subscription", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := group.POST("/resume", p.handleResumeCurrentSubscription,
		forge.WithSummary("Keep your subscription"),
		forge.WithDescription("Takes back a cancellation scheduled for the end of the period."),
		forge.WithOperationID("resumeCurrentSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Updated subscription", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	ent := router.Group(prefix+"/entitlements",
		forge.WithGroupTags("Billing"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)
	return ent.GET("", p.handleListEntitlements,
		forge.WithSummary("List your entitlements"),
		forge.WithDescription("Checks every feature on your active plan in one call."),
		forge.WithOperationID("listEntitlements"),
		forge.WithResponseSchema(http.StatusOK, "Entitlements by feature key", EntitlementsResponse{}),
		forge.WithErrorResponses(),
	)
}

// tenantScope resolves the app and tenant of a tenant route and checks the
// caller may read that tenant.
func (p *Plugin) tenantScope(ctx forge.Context, appID, tenantID string) (app, tenant string, err error) {
	if app, err = p.requestApp(ctx, appID); err != nil {
		return "", "", err
	}
	if tenant, err = p.resolveTenant(ctx, tenantID); err != nil {
		return "", "", err
	}
	if err = p.requireTenant(ctx, tenant); err != nil {
		return "", "", err
	}
	return app, tenant, nil
}

// writeScope is tenantScope for the self-service writes.
func (p *Plugin) writeScope(ctx forge.Context, tenantID string) (app, tenant string, err error) {
	if app, err = p.requestApp(ctx, ""); err != nil {
		return "", "", err
	}
	if tenant, err = p.resolveTenant(ctx, tenantID); err != nil {
		return "", "", err
	}
	if err = p.requireSelfService(ctx, tenant); err != nil {
		return "", "", err
	}
	return app, tenant, nil
}

// activeSubscription answers nil, nil when the tenant has none.
func (p *Plugin) activeSubscription(ctx forge.Context, tenantID, appID string) (*lsub.Subscription, error) {
	sub, err := p.service.GetActiveSubscription(ctx.Context(), tenantID, appID)
	if errors.Is(err, ledger.ErrNoActiveSubscription) {
		return nil, nil
	}
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("load subscription: %w", err))
	}
	return sub, nil
}

func (p *Plugin) mutationResponse(ctx forge.Context, sub *lsub.Subscription) (*SubscriptionMutationResponse, error) {
	view, err := p.toSubscriptionView(ctx.Context(), sub)
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	return &SubscriptionMutationResponse{Subscription: view}, nil
}

func (p *Plugin) handleGetCurrentSubscription(ctx forge.Context, req *CurrentSubscriptionRequest) (*CurrentSubscriptionResponse, error) {
	appID, tenantID, err := p.tenantScope(ctx, req.AppID, req.TenantID)
	if err != nil {
		return nil, err
	}

	resp := &CurrentSubscriptionResponse{}
	if userID, ok := middleware.UserIDFrom(ctx.Context()); ok {
		manage, err := p.canManageTenant(ctx.Context(), userID, tenantID)
		if err != nil {
			return nil, middleware.InternalError(ctx, err)
		}
		resp.CanManage = manage && (p.selfServiceOn(ctx.Context()) || p.holdsBillingManage(ctx.Context(), userID))
	}

	sub, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil || sub == nil {
		return resp, err
	}
	if resp.Subscription, err = p.toSubscriptionView(ctx.Context(), sub); err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	return resp, nil
}

func (p *Plugin) handleSubscribe(ctx forge.Context, req *SubscribeRequest) (*SubscriptionMutationResponse, error) {
	if req.Plan == "" {
		return nil, forge.BadRequest("plan is required")
	}
	appID, tenantID, err := p.writeScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	current, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, forge.NewHTTPError(http.StatusConflict, "already subscribed: use PATCH to change plan")
	}

	pl, err := p.resolvePlan(ctx, req.Plan, appID)
	if err != nil {
		return nil, err
	}
	quantity, err := seatQuantity(pl, req.Seats)
	if err != nil {
		return nil, err
	}

	return p.applyPlanChange(ctx, planChange{
		tenantID: tenantID, appID: appID, plan: pl, quantity: quantity,
		returnURL: req.ReturnURL, cancelURL: req.CancelURL,
	})
}

func (p *Plugin) handleUpdateSubscription(ctx forge.Context, req *UpdateSubscriptionRequest) (*SubscriptionMutationResponse, error) {
	if req.Plan == "" && req.Seats == nil {
		return nil, forge.BadRequest("nothing to change: pass plan, seats or both")
	}
	appID, tenantID, err := p.writeScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	current, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, forge.NotFound("no active subscription")
	}

	var pl *plan.Plan
	if req.Plan != "" {
		pl, err = p.resolvePlan(ctx, req.Plan, appID)
	} else {
		pl, err = p.service.GetPlan(ctx.Context(), current.PlanID)
	}
	if err != nil {
		return nil, err
	}
	quantity, err := seatQuantity(pl, req.Seats)
	if err != nil {
		return nil, err
	}

	return p.applyPlanChange(ctx, planChange{
		tenantID: tenantID, appID: appID, current: current, plan: pl, quantity: quantity,
		returnURL: req.ReturnURL, cancelURL: req.CancelURL,
	})
}

func (p *Plugin) handleCancelCurrentSubscription(ctx forge.Context, req *CancelCurrentRequest) (*SubscriptionMutationResponse, error) {
	var immediately bool
	switch req.When {
	case "", cancelAtPeriodEnd:
	case cancelNow:
		immediately = true
	default:
		return nil, forge.BadRequest(`when must be "period_end" or "now"`)
	}

	appID, tenantID, err := p.writeScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	current, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, forge.NotFound("no active subscription")
	}

	if err := p.service.CancelSubscription(ctx.Context(), current.ID, immediately); err != nil {
		return nil, writeErr(ctx, err, "cancel subscription")
	}
	p.audit(ctx.Context(), "subscription.canceled", "subscription", current.ID.String(), p.actor(ctx), tenantID, bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "subscription.canceled", tenantID, map[string]string{"immediately": boolStr(immediately)})

	sub, err := p.service.GetSubscription(ctx.Context(), current.ID)
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	return p.mutationResponse(ctx, sub)
}

func (p *Plugin) handleResumeCurrentSubscription(ctx forge.Context, req *TenantRequest) (*SubscriptionMutationResponse, error) {
	appID, tenantID, err := p.writeScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	current, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, forge.NotFound("no active subscription")
	}

	sub, err := p.service.KeepSubscription(ctx.Context(), current.ID)
	if err != nil {
		return nil, writeErr(ctx, err, "resume subscription")
	}
	p.audit(ctx.Context(), "subscription.resumed", "subscription", sub.ID.String(), p.actor(ctx), tenantID, bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "subscription.resumed", tenantID, nil)
	return p.mutationResponse(ctx, sub)
}

func (p *Plugin) handleListEntitlements(ctx forge.Context, req *ListEntitlementsRequest) (*EntitlementsResponse, error) {
	appID, tenantID, err := p.tenantScope(ctx, req.AppID, req.TenantID)
	if err != nil {
		return nil, err
	}

	resp := &EntitlementsResponse{Entitlements: map[string]EntitlementResponse{}}
	sub, err := p.activeSubscription(ctx, tenantID, appID)
	if err != nil || sub == nil {
		return resp, err
	}
	pl, err := p.service.GetPlan(ctx.Context(), sub.PlanID)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("load plan: %w", err))
	}

	entCtx := entitlementContext(ctx.Context(), tenantID, appID)
	for _, f := range pl.Features {
		result, err := p.service.CheckEntitlement(entCtx, f.Key)
		if err != nil {
			return nil, middleware.InternalError(ctx, fmt.Errorf("check entitlement %q: %w", f.Key, err))
		}
		resp.Entitlements[f.Key] = EntitlementResponse{
			Allowed:   result.Allowed,
			Feature:   result.Feature,
			Used:      result.Used,
			Limit:     result.Limit,
			Remaining: result.Remaining,
			Reason:    result.Reason,
		}
	}
	return resp, nil
}

// planChange is one subscribe or plan/seat change. current is nil for a new
// subscription.
type planChange struct {
	tenantID, appID      string
	current              *lsub.Subscription
	plan                 *plan.Plan
	quantity             map[string]int64
	returnURL, cancelURL string
}

// applyPlanChange subscribes the tenant, or moves its subscription to the
// plan and seat count, and answers with the updated subscription.
func (p *Plugin) applyPlanChange(ctx forge.Context, c planChange) (*SubscriptionMutationResponse, error) {
	var (
		sub *lsub.Subscription
		err error
	)
	if c.current == nil {
		sub, err = p.service.Subscribe(ctx.Context(), c.tenantID, c.plan.ID, c.appID, c.quantity)
		if err != nil {
			return nil, writeErr(ctx, err, "subscribe")
		}
		p.audit(ctx.Context(), "subscription.create", "subscription", sub.ID.String(), p.actor(ctx), c.tenantID, bridge.OutcomeSuccess)
		p.relayEvent(ctx.Context(), "subscription.created", c.tenantID, map[string]string{"plan_slug": c.plan.Slug})
	} else {
		sub, err = p.service.ChangePlan(ctx.Context(), c.current.ID, c.plan.ID, c.quantity)
		if err != nil {
			return nil, writeErr(ctx, err, "change plan")
		}
		p.audit(ctx.Context(), "subscription.plan_changed", "subscription", sub.ID.String(), p.actor(ctx), c.tenantID, bridge.OutcomeSuccess)
		p.relayEvent(ctx.Context(), "subscription.plan_changed", c.tenantID, map[string]string{"plan_slug": c.plan.Slug})
	}
	return p.mutationResponse(ctx, sub)
}

// resolvePlan finds a plan a tenant may subscribe to by slug or ID: in the
// request's app, active, and not an add-on.
func (p *Plugin) resolvePlan(ctx forge.Context, ref, appID string) (*plan.Plan, error) {
	var (
		pl  *plan.Plan
		err error
	)
	if planID, parseErr := ledgerid.ParsePlanID(ref); parseErr == nil {
		pl, err = p.service.GetPlan(ctx.Context(), planID)
	} else {
		pl, err = p.service.GetPlanBySlug(ctx.Context(), ref, appID)
	}
	if err != nil || pl.AppID != appID {
		return nil, forge.NotFound("plan not found")
	}
	if pl.Status != plan.StatusActive {
		return nil, forge.BadRequest("plan is not available")
	}
	if isAddon(pl) {
		return nil, forge.BadRequest("an add-on can't be subscribed to on its own")
	}
	return pl, nil
}

// seatQuantity turns a requested seat count into a ledger quantity. A nil
// count keeps whatever the subscription has.
func seatQuantity(pl *plan.Plan, seats *int64) (map[string]int64, error) {
	if seats == nil {
		return nil, nil
	}
	feat := seatFeature(pl)
	if feat == nil {
		return nil, forge.BadRequest(fmt.Sprintf("plan %q has no seats", pl.Slug))
	}
	if *seats < 0 {
		return nil, forge.BadRequest("seats can't be negative")
	}
	return map[string]int64{feat.Key: *seats}, nil
}

func isAddon(pl *plan.Plan) bool {
	return pl.Metadata != nil && pl.Metadata["addon"] == "true"
}

// actor is the caller's user ID for audit records, empty when unknown.
func (p *Plugin) actor(ctx forge.Context) string {
	if userID, ok := middleware.UserIDFrom(ctx.Context()); ok {
		return userID.String()
	}
	return ""
}
