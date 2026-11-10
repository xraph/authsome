package subscription

import (
	"fmt"
	"maps"
	"net/http"

	"github.com/xraph/forge"
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
// operator override. The app is always the request's app, for reads and
// writes alike, so a read can never show one app while a write changes
// another.

// CurrentSubscriptionRequest reads the caller's tenant's subscription.
type CurrentSubscriptionRequest struct {
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
type SubscriptionMutationResponse struct { //nolint:revive // the SDKs name their type after this one; "MutationResponse" says nothing on its own there
	Subscription *SubscriptionView `json:"subscription,omitempty"`
	CheckoutURL  string            `json:"checkout_url,omitempty"`
}

// ListEntitlementsRequest reads every entitlement of the caller's tenant.
type ListEntitlementsRequest struct {
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
	// Changing a plan is a human decision, the way writeGuard keeps the
	// operator writes. The deny-by-default interceptor refuses an agent
	// too, but only when the routes are mounted through the extension.
	humanOnly := forge.WithMiddleware(middleware.DenyDelegatedSession())

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
		humanOnly,
	); err != nil {
		return err
	}
	if err := group.PATCH("", p.handleUpdateSubscription,
		forge.WithSummary("Change plan or seats"),
		forge.WithOperationID("updateSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Subscription or checkout URL", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
		humanOnly,
	); err != nil {
		return err
	}
	if err := group.POST("/cancel", p.handleCancelCurrentSubscription,
		forge.WithSummary("Cancel your subscription"),
		forge.WithDescription("Cancels at the end of the current period unless when is \"now\"."),
		forge.WithOperationID("cancelCurrentSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Updated subscription", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
		humanOnly,
	); err != nil {
		return err
	}
	if err := group.POST("/resume", p.handleResumeCurrentSubscription,
		forge.WithSummary("Keep your subscription"),
		forge.WithDescription("Takes back a cancellation scheduled for the end of the period."),
		forge.WithOperationID("resumeCurrentSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Updated subscription", SubscriptionMutationResponse{}),
		forge.WithErrorResponses(),
		humanOnly,
	); err != nil {
		return err
	}

	ent := router.Group(prefix+"/entitlements",
		forge.WithGroupTags("Billing"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)
	if err := ent.GET("", p.handleListEntitlements,
		forge.WithSummary("List your entitlements"),
		forge.WithDescription("Checks every feature on your active plan in one call."),
		forge.WithOperationID("listEntitlements"),
		forge.WithResponseSchema(http.StatusOK, "Entitlements by feature key", EntitlementsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return p.registerPortalRoute(router, humanOnly)
}

// tenantScope resolves the app and tenant of a tenant route and checks the
// caller may read that tenant.
func (p *Plugin) tenantScope(ctx forge.Context, tenantID string) (app, tenant string, err error) {
	app, appErr := p.requestApp(ctx, "")
	if appErr != nil {
		return "", "", appErr
	}
	tenant, tenantErr := p.resolveTenant(ctx, tenantID)
	if tenantErr != nil {
		return "", "", tenantErr
	}
	if readErr := p.requireTenant(ctx, tenant); readErr != nil {
		return "", "", readErr
	}
	return app, tenant, nil
}

// writeScope is tenantScope for the self-service writes.
func (p *Plugin) writeScope(ctx forge.Context, tenantID string) (app, tenant string, err error) {
	app, appErr := p.requestApp(ctx, "")
	if appErr != nil {
		return "", "", appErr
	}
	tenant, tenantErr := p.resolveTenant(ctx, tenantID)
	if tenantErr != nil {
		return "", "", tenantErr
	}
	if writeErr := p.requireSelfService(ctx, tenant); writeErr != nil {
		return "", "", writeErr
	}
	return app, tenant, nil
}

// currentSubscription answers the tenant's subscription that hasn't ended,
// past_due and paused included, or nil, nil when it has none.
func (p *Plugin) currentSubscription(ctx forge.Context, tenantID, appID string) (*lsub.Subscription, error) {
	sub, err := p.service.CurrentSubscription(ctx.Context(), tenantID, appID)
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
	appID, tenantID, err := p.tenantScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	resp := &CurrentSubscriptionResponse{}
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if ok && middleware.RefuseDelegated(ctx.Context()) == nil {
		manage, manageErr := p.canManageTenant(ctx.Context(), userID, tenantID)
		if manageErr != nil {
			return nil, middleware.InternalError(ctx, manageErr)
		}
		resp.CanManage = manage && (p.selfServiceOn(ctx.Context()) || p.holdsBillingManage(ctx.Context(), userID))
	}

	sub, err := p.currentSubscription(ctx, tenantID, appID)
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

	// Check-then-create runs under the tenant's lock, so a double click
	// can't leave two live subscriptions. See tenantLock for its limits.
	unlock := p.tenantLock(tenantID, appID)
	defer unlock()

	current, err := p.currentSubscription(ctx, tenantID, appID)
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

	current, err := p.currentSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, forge.NotFound("no active subscription")
	}
	// A paused subscription is held by an operator, often for an unpaid
	// invoice; changing its plan would route around that.
	if statusErr := requireStatus(current, lsub.StatusActive, lsub.StatusTrialing, lsub.StatusPastDue); statusErr != nil {
		return nil, statusErr
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
	if quantity == nil {
		quantity = carriedQuantity(current.Quantity, pl)
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
	current, err := p.currentSubscription(ctx, tenantID, appID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, forge.NotFound("no active subscription")
	}

	if cancelErr := p.service.CancelSubscription(ctx.Context(), current.ID, immediately); cancelErr != nil {
		return nil, writeErr(ctx, cancelErr, "cancel subscription")
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
	current, err := p.currentSubscription(ctx, tenantID, appID)
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
	appID, tenantID, err := p.tenantScope(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	resp := &EntitlementsResponse{Entitlements: map[string]EntitlementResponse{}}
	sub, err := p.currentSubscription(ctx, tenantID, appID)
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

// applyPlanChange asks the checkout provider, when there is one, whether the
// change needs paying for first. Without a provider, or when it answers with
// no URL, the change is applied now.
func (p *Plugin) applyPlanChange(ctx forge.Context, c planChange) (*SubscriptionMutationResponse, error) {
	if p.config.Checkout != nil {
		url, err := p.startCheckout(ctx, c)
		if err != nil {
			return nil, err
		}
		if url != "" {
			return &SubscriptionMutationResponse{CheckoutURL: url}, nil
		}
	}

	sub, err := p.commitPlanChange(ctx.Context(), c, p.actor(ctx))
	if err != nil {
		return nil, writeErr(ctx, err, "change plan")
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
// count answers nil: keep what the subscription has.
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

// carriedQuantity is the seat count a plan change keeps when the caller
// didn't name one. Ledger keeps the current quantity on a nil map, and
// refuses a key the new plan has no seat feature for, so seats bought on
// the old plan would block a move to a plan without them. Keys the new plan
// still has carry over; the rest are dropped. nil means nothing changed.
func carriedQuantity(current map[string]int64, pl *plan.Plan) map[string]int64 {
	if len(current) == 0 {
		return nil
	}
	kept := make(map[string]int64, len(current))
	for key, n := range current {
		if f := pl.FindFeature(key); f != nil && f.Type == plan.FeatureSeat {
			kept[key] = n
		}
	}
	if maps.Equal(kept, current) {
		return nil
	}
	return kept
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
