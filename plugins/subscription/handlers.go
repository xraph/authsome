package subscription

import (
	"io"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers/types"
	"github.com/xraph/forge"
)

// ctxQueryInt gets an integer query parameter with a default value
func ctxQueryInt(c forge.Context, name string, defaultValue int) int {
	str := c.QueryDefault(name, "")
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// ctxBody reads the request body
func ctxBody(c forge.Context) ([]byte, error) {
	return io.ReadAll(c.Request().Body)
}

// Request/Response DTOs

type createPlanRequest struct {
	Name            string             `json:"name" validate:"required,min=1,max=100"`
	Slug            string             `json:"slug" validate:"required,min=1,max=50"`
	Description     string             `json:"description"`
	BillingPattern  string             `json:"billingPattern" validate:"required"`
	BillingInterval string             `json:"billingInterval" validate:"required"`
	BasePrice       int64              `json:"basePrice"`
	Currency        string             `json:"currency"`
	TrialDays       int                `json:"trialDays"`
	Features        []core.PlanFeature `json:"features"`
	PriceTiers      []core.PriceTier   `json:"priceTiers"`
	TierMode        string             `json:"tierMode"`
	Metadata        map[string]any     `json:"metadata"`
	IsActive        bool               `json:"isActive"`
	IsPublic        bool               `json:"isPublic"`
	DisplayOrder    int                `json:"displayOrder"`
}

type updatePlanRequest struct {
	Name         *string            `json:"name"`
	Description  *string            `json:"description"`
	BasePrice    *int64             `json:"basePrice"`
	TrialDays    *int               `json:"trialDays"`
	Features     []core.PlanFeature `json:"features"`
	PriceTiers   []core.PriceTier   `json:"priceTiers"`
	TierMode     *string            `json:"tierMode"`
	Metadata     map[string]any     `json:"metadata"`
	IsActive     *bool              `json:"isActive"`
	IsPublic     *bool              `json:"isPublic"`
	DisplayOrder *int               `json:"displayOrder"`
}

type createSubscriptionRequest struct {
	OrganizationID string         `json:"organizationId" validate:"required"`
	PlanID         string         `json:"planId" validate:"required"`
	Quantity       int            `json:"quantity"`
	StartTrial     bool           `json:"startTrial"`
	TrialDays      int            `json:"trialDays"`
	Metadata       map[string]any `json:"metadata"`
}

type updateSubscriptionRequest struct {
	PlanID   *string        `json:"planId"`
	Quantity *int           `json:"quantity"`
	Metadata map[string]any `json:"metadata"`
}

type cancelSubscriptionRequest struct {
	Immediate bool   `json:"immediate"`
	Reason    string `json:"reason"`
}

type pauseSubscriptionRequest struct {
	ResumeAt *int64 `json:"resumeAt"` // Unix timestamp
	Reason   string `json:"reason"`
}

type recordUsageRequest struct {
	SubscriptionID string         `json:"subscriptionId" validate:"required"`
	MetricKey      string         `json:"metricKey" validate:"required"`
	Quantity       int64          `json:"quantity" validate:"required"`
	Action         string         `json:"action" validate:"required"`
	Timestamp      *int64         `json:"timestamp"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Metadata       map[string]any `json:"metadata"`
}

type checkoutRequest struct {
	OrganizationID  string `json:"organizationId" validate:"required"`
	PlanID          string `json:"planId" validate:"required"`
	Quantity        int    `json:"quantity"`
	SuccessURL      string `json:"successUrl" validate:"required"`
	CancelURL       string `json:"cancelUrl" validate:"required"`
	AllowPromoCodes bool   `json:"allowPromoCodes"`
	TrialDays       int    `json:"trialDays"`
}

type portalRequest struct {
	OrganizationID string `json:"organizationId" validate:"required"`
	ReturnURL      string `json:"returnUrl" validate:"required"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type successResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Plan Handlers

func (p *Plugin) handleCreatePlan(c forge.Context) error {
	var req createPlanRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// Get app ID from context
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	tierMode := core.TierMode(req.TierMode)
	if req.TierMode == "" {
		tierMode = core.TierModeGraduated
	}

	plan, err := p.planSvc.Create(c.Context(), appID, &core.CreatePlanRequest{
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		BillingPattern:  core.BillingPattern(req.BillingPattern),
		BillingInterval: core.BillingInterval(req.BillingInterval),
		BasePrice:       req.BasePrice,
		Currency:        req.Currency,
		TrialDays:       req.TrialDays,
		Features:        req.Features,
		PriceTiers:      req.PriceTiers,
		TierMode:        tierMode,
		Metadata:        req.Metadata,
		IsActive:        req.IsActive,
		IsPublic:        req.IsPublic,
		DisplayOrder:    req.DisplayOrder,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, plan)
}

func (p *Plugin) handleListPlans(c forge.Context) error {
	appID := getAppID(c)
	activeOnly := c.Query("active") == "true"
	publicOnly := c.Query("public") == "true"
	page := ctxQueryInt(c, "page", 1)
	pageSize := ctxQueryInt(c, "pageSize", 20)

	plans, total, err := p.planSvc.List(c.Context(), appID, activeOnly, publicOnly, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"plans": plans,
		"total": total,
		"page":  page,
	})
}

func (p *Plugin) handleGetPlan(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	plan, err := p.planSvc.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, plan)
}

func (p *Plugin) handleUpdatePlan(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	var req updatePlanRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	var tierMode *core.TierMode
	if req.TierMode != nil {
		tm := core.TierMode(*req.TierMode)
		tierMode = &tm
	}

	plan, err := p.planSvc.Update(c.Context(), id, &core.UpdatePlanRequest{
		Name:         req.Name,
		Description:  req.Description,
		BasePrice:    req.BasePrice,
		TrialDays:    req.TrialDays,
		Features:     req.Features,
		PriceTiers:   req.PriceTiers,
		TierMode:     tierMode,
		Metadata:     req.Metadata,
		IsActive:     req.IsActive,
		IsPublic:     req.IsPublic,
		DisplayOrder: req.DisplayOrder,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, plan)
}

func (p *Plugin) handleDeletePlan(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	if err := p.planSvc.Delete(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "plan deleted"})
}

func (p *Plugin) handleSyncPlan(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	if err := p.planSvc.SyncToProvider(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "plan synced to provider"})
}

// Subscription Handlers

func (p *Plugin) handleCreateSubscription(c forge.Context) error {
	var req createSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	orgID, err := xid.FromString(req.OrganizationID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_org_id", Message: "invalid organization ID"})
	}

	planID, err := xid.FromString(req.PlanID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_plan_id", Message: "invalid plan ID"})
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	sub, err := p.subscriptionSvc.Create(c.Context(), &core.CreateSubscriptionRequest{
		OrganizationID: orgID,
		PlanID:         planID,
		Quantity:       quantity,
		StartTrial:     req.StartTrial,
		TrialDays:      req.TrialDays,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, sub)
}

func (p *Plugin) handleListSubscriptions(c forge.Context) error {
	page := ctxQueryInt(c, "page", 1)
	pageSize := ctxQueryInt(c, "pageSize", 20)
	status := c.Query("status")

	var orgID, planID *xid.ID
	if orgIDStr := c.Query("organizationId"); orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err == nil {
			orgID = &id
		}
	}
	if planIDStr := c.Query("planId"); planIDStr != "" {
		id, err := xid.FromString(planIDStr)
		if err == nil {
			planID = &id
		}
	}

	subs, total, err := p.subscriptionSvc.List(c.Context(), nil, orgID, planID, status, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"subscriptions": subs,
		"total":         total,
		"page":          page,
	})
}

func (p *Plugin) handleGetSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	sub, err := p.subscriptionSvc.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, sub)
}

func (p *Plugin) handleGetOrganizationSubscription(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	sub, err := p.subscriptionSvc.GetByOrganizationID(c.Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, sub)
}

func (p *Plugin) handleUpdateSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	var req updateSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	var planID *xid.ID
	if req.PlanID != nil {
		id, err := xid.FromString(*req.PlanID)
		if err == nil {
			planID = &id
		}
	}

	sub, err := p.subscriptionSvc.Update(c.Context(), id, &core.UpdateSubscriptionRequest{
		PlanID:   planID,
		Quantity: req.Quantity,
		Metadata: req.Metadata,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, sub)
}

func (p *Plugin) handleCancelSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	var req cancelSubscriptionRequest
	c.BindJSON(&req)

	if err := p.subscriptionSvc.Cancel(c.Context(), id, &core.CancelSubscriptionRequest{
		Immediate: req.Immediate,
		Reason:    req.Reason,
	}); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "subscription canceled"})
}

func (p *Plugin) handlePauseSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	var req pauseSubscriptionRequest
	c.BindJSON(&req)

	if err := p.subscriptionSvc.Pause(c.Context(), id, &core.PauseSubscriptionRequest{
		Reason: req.Reason,
	}); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "subscription paused"})
}

func (p *Plugin) handleResumeSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	if err := p.subscriptionSvc.Resume(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "subscription resumed"})
}

// Add-on Handlers

func (p *Plugin) handleCreateAddOn(c forge.Context) error {
	var req createPlanRequest // Reuse structure
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	appID := getAppID(c)

	addon, err := p.addOnSvc.Create(c.Context(), appID, &core.CreateAddOnRequest{
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		BillingPattern:  core.BillingPattern(req.BillingPattern),
		BillingInterval: core.BillingInterval(req.BillingInterval),
		Price:           req.BasePrice,
		Currency:        req.Currency,
		Features:        req.Features,
		PriceTiers:      req.PriceTiers,
		TierMode:        core.TierMode(req.TierMode),
		Metadata:        req.Metadata,
		IsActive:        req.IsActive,
		IsPublic:        req.IsPublic,
		DisplayOrder:    req.DisplayOrder,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, addon)
}

func (p *Plugin) handleListAddOns(c forge.Context) error {
	appID := getAppID(c)
	activeOnly := c.Query("active") == "true"
	publicOnly := c.Query("public") == "true"
	page := ctxQueryInt(c, "page", 1)
	pageSize := ctxQueryInt(c, "pageSize", 20)

	addons, total, err := p.addOnSvc.List(c.Context(), appID, activeOnly, publicOnly, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"addons": addons,
		"total":  total,
		"page":   page,
	})
}

func (p *Plugin) handleGetAddOn(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid add-on ID"})
	}

	addon, err := p.addOnSvc.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, addon)
}

func (p *Plugin) handleUpdateAddOn(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid add-on ID"})
	}

	var req updatePlanRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	var tierMode *core.TierMode
	if req.TierMode != nil {
		tm := core.TierMode(*req.TierMode)
		tierMode = &tm
	}

	addon, err := p.addOnSvc.Update(c.Context(), id, &core.UpdateAddOnRequest{
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.BasePrice,
		Features:     req.Features,
		PriceTiers:   req.PriceTiers,
		TierMode:     tierMode,
		Metadata:     req.Metadata,
		IsActive:     req.IsActive,
		IsPublic:     req.IsPublic,
		DisplayOrder: req.DisplayOrder,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, addon)
}

func (p *Plugin) handleDeleteAddOn(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid add-on ID"})
	}

	if err := p.addOnSvc.Delete(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "add-on deleted"})
}

// Invoice Handlers

func (p *Plugin) handleListInvoices(c forge.Context) error {
	page := ctxQueryInt(c, "page", 1)
	pageSize := ctxQueryInt(c, "pageSize", 20)
	status := c.Query("status")

	var orgID, subID *xid.ID
	if orgIDStr := c.Query("organizationId"); orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err == nil {
			orgID = &id
		}
	}
	if subIDStr := c.Query("subscriptionId"); subIDStr != "" {
		id, err := xid.FromString(subIDStr)
		if err == nil {
			subID = &id
		}
	}

	invoices, total, err := p.invoiceSvc.List(c.Context(), orgID, subID, status, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"invoices": invoices,
		"total":    total,
		"page":     page,
	})
}

func (p *Plugin) handleGetInvoice(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid invoice ID"})
	}

	invoice, err := p.invoiceSvc.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, invoice)
}

// Usage Handlers

func (p *Plugin) handleRecordUsage(c forge.Context) error {
	var req recordUsageRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	subID, err := xid.FromString(req.SubscriptionID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	record, err := p.usageSvc.RecordUsage(c.Context(), &core.RecordUsageRequest{
		SubscriptionID: subID,
		MetricKey:      req.MetricKey,
		Quantity:       req.Quantity,
		Action:         core.UsageAction(req.Action),
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, record)
}

func (p *Plugin) handleGetUsageSummary(c forge.Context) error {
	subIDStr := c.Query("subscriptionId")
	if subIDStr == "" {
		return c.JSON(400, errorResponse{Error: "missing_param", Message: "subscriptionId required"})
	}

	subID, err := xid.FromString(subIDStr)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	metricKey := c.Query("metricKey")
	if metricKey == "" {
		return c.JSON(400, errorResponse{Error: "missing_param", Message: "metricKey required"})
	}

	// Get subscription for period
	sub, err := p.subscriptionSvc.GetByID(c.Context(), subID)
	if err != nil {
		return handleError(c, err)
	}

	summary, err := p.usageSvc.GetSummary(c.Context(), &core.GetUsageSummaryRequest{
		SubscriptionID: subID,
		MetricKey:      metricKey,
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, summary)
}

// Checkout Handlers

func (p *Plugin) handleCreateCheckout(c forge.Context) error {
	var req checkoutRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	orgID, err := xid.FromString(req.OrganizationID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_org_id", Message: "invalid organization ID"})
	}

	planID, err := xid.FromString(req.PlanID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_plan_id", Message: "invalid plan ID"})
	}

	// Get plan to get price ID
	plan, err := p.planSvc.GetByID(c.Context(), planID)
	if err != nil {
		return handleError(c, err)
	}

	// Get or create customer
	customer, err := p.customerSvc.GetOrCreate(c.Context(), orgID, "", "")
	if err != nil {
		return handleError(c, err)
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	session, err := p.provider.CreateCheckoutSession(c.Context(), &types.CheckoutRequest{
		CustomerID:      customer.ProviderCustomerID,
		PriceID:         plan.ProviderPriceID,
		Quantity:        quantity,
		SuccessURL:      req.SuccessURL,
		CancelURL:       req.CancelURL,
		AllowPromoCodes: req.AllowPromoCodes,
		TrialDays:       req.TrialDays,
		Metadata: map[string]interface{}{
			"organization_id": orgID.String(),
			"plan_id":         planID.String(),
		},
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, session)
}

func (p *Plugin) handleCreatePortal(c forge.Context) error {
	var req portalRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	orgID, err := xid.FromString(req.OrganizationID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_org_id", Message: "invalid organization ID"})
	}

	customer, err := p.customerSvc.GetByOrganizationID(c.Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}

	url, err := p.provider.CreatePortalSession(c.Context(), customer.ProviderCustomerID, req.ReturnURL)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]string{"url": url})
}

// Webhook Handler

func (p *Plugin) handleStripeWebhook(c forge.Context) error {
	payload, err := ctxBody(c)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "bad_request", Message: "Failed to read request body"})
	}
	signature := c.Header("Stripe-Signature")

	event, err := p.provider.HandleWebhook(c.Context(), payload, signature)
	if err != nil {
		p.logger.Error("webhook verification failed", forge.F("error", err.Error()))
		return c.JSON(400, errorResponse{Error: "webhook_error", Message: err.Error()})
	}

	p.logger.Info("webhook received", forge.F("type", event.Type), forge.F("id", event.ID))

	// Handle webhook events
	switch event.Type {
	case "checkout.session.completed":
		// Handle successful checkout
	case "customer.subscription.created":
		// Sync subscription
	case "customer.subscription.updated":
		// Update subscription status
	case "customer.subscription.deleted":
		// Handle cancellation
	case "invoice.paid":
		// Mark invoice paid
	case "invoice.payment_failed":
		// Handle failed payment
	}

	return c.JSON(200, map[string]string{"received": "true"})
}

// Feature/Limit Handlers

func (p *Plugin) handleCheckFeature(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	feature := c.Param("feature")

	hasAccess, err := p.enforcementSvc.CheckFeatureAccess(c.Context(), orgID, feature)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"feature":   feature,
		"hasAccess": hasAccess,
	})
}

func (p *Plugin) handleGetLimits(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	limits, err := p.enforcementSvc.GetAllLimits(c.Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, limits)
}

// Helper functions

func getAppID(c forge.Context) xid.ID {
	// Get app ID from context (set by auth middleware)
	if appID, ok := c.Get("appID").(xid.ID); ok {
		return appID
	}
	// Try to get from header
	if appIDStr := c.Header("X-App-ID"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			return id
		}
	}
	return xid.ID{}
}

func handleError(c forge.Context, err error) error {
	if IsNotFoundError(err) {
		return c.JSON(404, errorResponse{Error: "not_found", Message: err.Error()})
	}
	if IsValidationError(err) {
		return c.JSON(400, errorResponse{Error: "validation_error", Message: err.Error()})
	}
	if IsConflictError(err) {
		return c.JSON(409, errorResponse{Error: "conflict", Message: err.Error()})
	}
	if IsLimitError(err) {
		return c.JSON(403, errorResponse{Error: "limit_exceeded", Message: err.Error()})
	}
	if IsPaymentError(err) {
		return c.JSON(402, errorResponse{Error: "payment_required", Message: err.Error()})
	}

	return c.JSON(500, errorResponse{Error: "internal_error", Message: err.Error()})
}

// CheckoutRequest for internal use
type CheckoutRequest struct {
	CustomerID      string
	PriceID         string
	Quantity        int
	SuccessURL      string
	CancelURL       string
	AllowPromoCodes bool
	TrialDays       int
	Metadata        map[string]interface{}
}
