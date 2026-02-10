package subscription

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers/types"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
	"github.com/xraph/forge"
)

// ctxQueryInt gets an integer query parameter with a default value.
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

// ctxBody reads the request body.
func ctxBody(c forge.Context) ([]byte, error) {
	return io.ReadAll(c.Request().Body)
}

// Request/Response DTOs

type createPlanRequest struct {
	Name            string             `json:"name"            validate:"required,min=1,max=100"`
	Slug            string             `json:"slug"            validate:"required,min=1,max=50"`
	Description     string             `json:"description"`
	BillingPattern  string             `json:"billingPattern"  validate:"required"`
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
	PlanID         string         `json:"planId"         validate:"required"`
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
	MetricKey      string         `json:"metricKey"      validate:"required"`
	Quantity       int64          `json:"quantity"       validate:"required"`
	Action         string         `json:"action"         validate:"required"`
	Timestamp      *int64         `json:"timestamp"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Metadata       map[string]any `json:"metadata"`
}

type checkoutRequest struct {
	OrganizationID  string `json:"organizationId"  validate:"required"`
	PlanID          string `json:"planId"          validate:"required"`
	Quantity        int    `json:"quantity"`
	SuccessURL      string `json:"successUrl"      validate:"required"`
	CancelURL       string `json:"cancelUrl"       validate:"required"`
	AllowPromoCodes bool   `json:"allowPromoCodes"`
	TrialDays       int    `json:"trialDays"`
}

type portalRequest struct {
	OrganizationID string `json:"organizationId" validate:"required"`
	ReturnURL      string `json:"returnUrl"      validate:"required"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type successResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
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

	return c.JSON(200, map[string]any{
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

func (p *Plugin) handleSyncSubscription(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	if err := p.subscriptionSvc.SyncToProvider(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "subscription synced to provider"})
}

func (p *Plugin) handleSyncSubscriptionFromProvider(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
	}

	sub, err := p.subscriptionSvc.SyncFromProviderByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, sub)
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

	return c.JSON(200, map[string]any{
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
	if err := c.BindJSON(&req); err != nil {
		return err
	}

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
	if err := c.BindJSON(&req); err != nil {
		return err
	}

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

	return c.JSON(200, map[string]any{
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

	return c.JSON(200, map[string]any{
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

func (p *Plugin) handleSyncInvoices(c forge.Context) error {
	ctx := c.Context()

	// Get optional subscription ID filter
	var subID *xid.ID

	if subIDStr := c.Query("subscriptionId"); subIDStr != "" {
		id, err := xid.FromString(subIDStr)
		if err != nil {
			return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid subscription ID"})
		}

		subID = &id
	}

	// Sync invoices
	syncedCount, err := p.SyncInvoicesFromStripe(ctx, subID)
	if err != nil {
		return c.JSON(500, errorResponse{Error: "sync_failed", Message: err.Error()})
	}

	return c.JSON(200, map[string]any{
		"synced":  syncedCount,
		"message": fmt.Sprintf("Successfully synced %d invoices from Stripe", syncedCount),
	})
}

// SyncInvoicesFromStripe syncs invoices from Stripe to the local database (exported for dashboard use).
func (p *Plugin) SyncInvoicesFromStripe(ctx context.Context, subID *xid.ID) (int, error) {
	var (
		subscriptions []*schema.Subscription
		err           error
	)

	if subID != nil {
		// Sync for specific subscription
		sub, err := p.subRepo.FindByID(ctx, *subID)
		if err != nil {
			return 0, fmt.Errorf("subscription not found: %w", err)
		}

		subscriptions = []*schema.Subscription{sub}
	} else {
		// Sync for all active subscriptions
		filter := &repository.SubscriptionFilter{
			Status:   "active",
			Page:     1,
			PageSize: 1000,
		}

		subscriptions, _, err = p.subRepo.List(ctx, filter)
		if err != nil {
			return 0, fmt.Errorf("failed to list subscriptions: %w", err)
		}
	}

	syncedCount := 0

	for _, sub := range subscriptions {
		if sub.ProviderSubID == "" {
			continue
		}

		// Get invoices from Stripe for this subscription
		providerInvoices, err := p.provider.ListSubscriptionInvoices(ctx, sub.ProviderSubID, 100)
		if err != nil {
			p.logger.Error("failed to list invoices from provider",
				forge.F("subscriptionId", sub.ID.String()),
				forge.F("error", err.Error()))

			continue
		}

		for _, providerInv := range providerInvoices {
			// Check if invoice already exists
			existing, err := p.invoiceRepo.FindByProviderID(ctx, providerInv.ID)
			if err == nil && existing != nil {
				// Invoice exists, update it
				existing.Status = mapStripeStatusToInvoiceStatus(providerInv.Status)
				existing.Subtotal = providerInv.Subtotal
				existing.Tax = providerInv.Tax
				existing.Total = providerInv.Total
				existing.AmountPaid = providerInv.AmountPaid
				existing.AmountDue = providerInv.AmountDue
				existing.ProviderPDFURL = providerInv.PDFURL
				existing.HostedInvoiceURL = providerInv.HostedURL
				existing.UpdatedAt = time.Now()

				if err := p.invoiceRepo.Update(ctx, existing); err != nil {
					p.logger.Error("failed to update invoice",
						forge.F("invoiceId", existing.ID.String()),
						forge.F("error", err.Error()))

					continue
				}

				syncedCount++

				continue
			}

			// Create new invoice
			number, err := p.invoiceRepo.GetNextInvoiceNumber(ctx, sub.Plan.AppID)
			if err != nil {
				p.logger.Error("failed to generate invoice number", forge.F("error", err.Error()))

				continue
			}

			now := time.Now()
			invoice := &schema.SubscriptionInvoice{
				ID:                xid.New(),
				SubscriptionID:    sub.ID,
				OrganizationID:    sub.OrganizationID,
				Number:            number,
				Status:            mapStripeStatusToInvoiceStatus(providerInv.Status),
				Currency:          providerInv.Currency,
				Subtotal:          providerInv.Subtotal,
				Tax:               providerInv.Tax,
				Total:             providerInv.Total,
				AmountPaid:        providerInv.AmountPaid,
				AmountDue:         providerInv.AmountDue,
				PeriodStart:       time.Unix(providerInv.PeriodStart, 0),
				PeriodEnd:         time.Unix(providerInv.PeriodEnd, 0),
				DueDate:           time.Unix(providerInv.PeriodStart, 0).AddDate(0, 0, 14),
				ProviderInvoiceID: providerInv.ID,
				ProviderPDFURL:    providerInv.PDFURL,
				HostedInvoiceURL:  providerInv.HostedURL,
				Metadata:          make(map[string]any),
			}

			if providerInv.Status == "paid" {
				invoice.PaidAt = &now
			}

			if err := p.invoiceRepo.Create(ctx, invoice); err != nil {
				p.logger.Error("failed to create invoice",
					forge.F("providerInvoiceId", providerInv.ID),
					forge.F("error", err.Error()))

				continue
			}

			syncedCount++
		}
	}

	return syncedCount, nil
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
		Metadata: map[string]any{
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
		// TODO: Implement checkout completion
	case "customer.subscription.created":
		// Sync subscription
		// TODO: Implement subscription sync
	case "customer.subscription.updated":
		// Update subscription status
		// TODO: Implement subscription update
	case "customer.subscription.deleted":
		// Handle cancellation
		// TODO: Implement subscription cancellation
	case "invoice.created", "invoice.finalized":
		if err := p.handleInvoiceCreatedOrFinalized(c.Context(), event); err != nil {
			p.logger.Error("failed to handle invoice created/finalized", forge.F("error", err.Error()))
		}
	case "invoice.paid":
		if err := p.handleInvoicePaid(c.Context(), event); err != nil {
			p.logger.Error("failed to handle invoice paid", forge.F("error", err.Error()))
		}
	case "invoice.payment_failed":
		if err := p.handleInvoicePaymentFailed(c.Context(), event); err != nil {
			p.logger.Error("failed to handle invoice payment failed", forge.F("error", err.Error()))
		}
	case "invoice.updated":
		if err := p.handleInvoiceUpdated(c.Context(), event); err != nil {
			p.logger.Error("failed to handle invoice updated", forge.F("error", err.Error()))
		}
	}

	return c.JSON(200, map[string]string{"received": "true"})
}

// Webhook event handlers

func (p *Plugin) handleInvoiceCreatedOrFinalized(ctx context.Context, event *types.WebhookEvent) error {
	// Extract invoice data from webhook
	invoiceData, ok := event.Data["object"].(map[string]any)
	if !ok {
		return errs.BadRequest("invalid invoice data in webhook")
	}

	providerInvoiceID, _ := invoiceData["id"].(string)
	if providerInvoiceID == "" {
		return errs.BadRequest("missing invoice ID in webhook")
	}

	// Check if invoice already exists
	existingInvoice, err := p.invoiceRepo.FindByProviderID(ctx, providerInvoiceID)
	if err == nil && existingInvoice != nil {
		// Invoice already exists, update it instead
		return p.syncInvoiceFromProvider(ctx, providerInvoiceID)
	}

	// Get subscription ID from webhook
	providerSubID, _ := invoiceData["subscription"].(string)
	if providerSubID == "" {
		p.logger.Warn("invoice has no subscription", forge.F("invoiceId", providerInvoiceID))

		return nil // Not all invoices are subscription-based
	}

	// Find our subscription by provider ID
	sub, err := p.subRepo.FindByProviderID(ctx, providerSubID)
	if err != nil {
		return fmt.Errorf("subscription not found for provider ID %s: %w", providerSubID, err)
	}

	// Generate invoice number
	number, err := p.invoiceRepo.GetNextInvoiceNumber(ctx, sub.Plan.AppID)
	if err != nil {
		return fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// Parse invoice status
	status, _ := invoiceData["status"].(string)
	invoiceStatus := mapStripeStatusToInvoiceStatus(status)

	// Parse amounts (Stripe sends in cents)
	total := int64(0)
	if t, ok := invoiceData["total"].(float64); ok {
		total = int64(t)
	}

	subtotal := int64(0)
	if s, ok := invoiceData["subtotal"].(float64); ok {
		subtotal = int64(s)
	}

	tax := int64(0)
	if tx, ok := invoiceData["tax"].(float64); ok {
		tax = int64(tx)
	}

	amountDue := int64(0)
	if a, ok := invoiceData["amount_due"].(float64); ok {
		amountDue = int64(a)
	}

	amountPaid := int64(0)
	if a, ok := invoiceData["amount_paid"].(float64); ok {
		amountPaid = int64(a)
	}

	// Parse timestamps
	periodStart := time.Unix(int64(invoiceData["period_start"].(float64)), 0)
	periodEnd := time.Unix(int64(invoiceData["period_end"].(float64)), 0)

	dueDate := time.Now().AddDate(0, 0, 14) // Default 14 days
	if dd, ok := invoiceData["due_date"].(float64); ok && dd > 0 {
		dueDate = time.Unix(int64(dd), 0)
	}

	// Parse URLs
	hostedURL, _ := invoiceData["hosted_invoice_url"].(string)
	pdfURL, _ := invoiceData["invoice_pdf"].(string)
	currency, _ := invoiceData["currency"].(string)

	invoice := &schema.SubscriptionInvoice{
		ID:                xid.New(),
		SubscriptionID:    sub.ID,
		OrganizationID:    sub.OrganizationID,
		Number:            number,
		Status:            string(invoiceStatus),
		Currency:          currency,
		Subtotal:          subtotal,
		Tax:               tax,
		Total:             total,
		AmountPaid:        amountPaid,
		AmountDue:         amountDue,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		DueDate:           dueDate,
		ProviderInvoiceID: providerInvoiceID,
		ProviderPDFURL:    pdfURL,
		HostedInvoiceURL:  hostedURL,
		Metadata:          make(map[string]any),
	}

	// Create invoice in database
	if err := p.invoiceRepo.Create(ctx, invoice); err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	p.logger.Info("invoice created from webhook",
		forge.F("invoiceId", invoice.ID.String()),
		forge.F("providerInvoiceId", providerInvoiceID),
		forge.F("subscriptionId", sub.ID.String()))

	return nil
}

func (p *Plugin) handleInvoicePaid(ctx context.Context, event *types.WebhookEvent) error {
	invoiceData, ok := event.Data["object"].(map[string]any)
	if !ok {
		return errs.BadRequest("invalid invoice data in webhook")
	}

	providerInvoiceID, _ := invoiceData["id"].(string)
	if providerInvoiceID == "" {
		return errs.BadRequest("missing invoice ID in webhook")
	}

	// Find invoice by provider ID
	invoice, err := p.invoiceRepo.FindByProviderID(ctx, providerInvoiceID)
	if err != nil {
		// Invoice doesn't exist, create it first
		if err := p.handleInvoiceCreatedOrFinalized(ctx, event); err != nil {
			return err
		}
		// Try to find it again
		invoice, err = p.invoiceRepo.FindByProviderID(ctx, providerInvoiceID)
		if err != nil {
			return fmt.Errorf("invoice not found after creation: %w", err)
		}
	}

	// Update invoice status to paid
	now := time.Now()
	invoice.Status = string(core.InvoiceStatusPaid)
	invoice.PaidAt = &now
	invoice.AmountPaid = invoice.Total
	invoice.AmountDue = 0
	invoice.UpdatedAt = now

	if err := p.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	p.logger.Info("invoice marked as paid",
		forge.F("invoiceId", invoice.ID.String()),
		forge.F("providerInvoiceId", providerInvoiceID))

	return nil
}

func (p *Plugin) handleInvoicePaymentFailed(ctx context.Context, event *types.WebhookEvent) error {
	invoiceData, ok := event.Data["object"].(map[string]any)
	if !ok {
		return errs.BadRequest("invalid invoice data in webhook")
	}

	providerInvoiceID, _ := invoiceData["id"].(string)
	if providerInvoiceID == "" {
		return errs.BadRequest("missing invoice ID in webhook")
	}

	// Find invoice by provider ID
	invoice, err := p.invoiceRepo.FindByProviderID(ctx, providerInvoiceID)
	if err != nil {
		p.logger.Warn("invoice not found for payment failed event",
			forge.F("providerInvoiceId", providerInvoiceID))

		return nil
	}

	// Update invoice status
	invoice.Status = string(core.InvoiceStatusOpen) // Keep it open for retry
	invoice.UpdatedAt = time.Now()

	if err := p.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	p.logger.Warn("invoice payment failed",
		forge.F("invoiceId", invoice.ID.String()),
		forge.F("providerInvoiceId", providerInvoiceID))

	// TODO: Send notification to organization about payment failure
	// TODO: Create alert for payment failure

	return nil
}

func (p *Plugin) handleInvoiceUpdated(ctx context.Context, event *types.WebhookEvent) error {
	invoiceData, ok := event.Data["object"].(map[string]any)
	if !ok {
		return errs.BadRequest("invalid invoice data in webhook")
	}

	providerInvoiceID, _ := invoiceData["id"].(string)
	if providerInvoiceID == "" {
		return errs.BadRequest("missing invoice ID in webhook")
	}

	// Sync invoice from provider to get latest data
	return p.syncInvoiceFromProvider(ctx, providerInvoiceID)
}

func (p *Plugin) syncInvoiceFromProvider(ctx context.Context, providerInvoiceID string) error {
	// Get invoice from provider
	providerInv, err := p.provider.GetInvoice(ctx, providerInvoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice from provider: %w", err)
	}

	// Find local invoice
	invoice, err := p.invoiceRepo.FindByProviderID(ctx, providerInvoiceID)
	if err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// Update invoice with provider data
	invoice.Status = mapStripeStatusToInvoiceStatus(providerInv.Status)
	invoice.Subtotal = providerInv.Subtotal
	invoice.Tax = providerInv.Tax
	invoice.Total = providerInv.Total
	invoice.AmountPaid = providerInv.AmountPaid
	invoice.AmountDue = providerInv.AmountDue
	invoice.ProviderPDFURL = providerInv.PDFURL
	invoice.HostedInvoiceURL = providerInv.HostedURL
	invoice.UpdatedAt = time.Now()

	if err := p.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	return nil
}

func mapStripeStatusToInvoiceStatus(stripeStatus string) string {
	switch stripeStatus {
	case "draft":
		return string(core.InvoiceStatusDraft)
	case "open":
		return string(core.InvoiceStatusOpen)
	case "paid":
		return string(core.InvoiceStatusPaid)
	case "void":
		return string(core.InvoiceStatusVoid)
	case "uncollectible":
		return string(core.InvoiceStatusUncollectible)
	default:
		return string(core.InvoiceStatusDraft)
	}
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

	return c.JSON(200, map[string]any{
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

// CheckoutRequest for internal use.
type CheckoutRequest struct {
	CustomerID      string
	PriceID         string
	Quantity        int
	SuccessURL      string
	CancelURL       string
	AllowPromoCodes bool
	TrialDays       int
	Metadata        map[string]any
}
