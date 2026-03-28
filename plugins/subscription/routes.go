package subscription

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/ledger/coupon"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"
	"github.com/xraph/ledger/types"

	"github.com/xraph/authsome/bridge"
)

// RegisterRoutes registers billing API routes on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	prefix := p.config.PathPrefix

	// Plan routes
	plans := router.Group(prefix+"/plans", forge.WithGroupTags("Billing Plans"))

	if err := plans.GET("", p.handleListPlans,
		forge.WithSummary("List billing plans"),
		forge.WithOperationID("listBillingPlans"),
		forge.WithResponseSchema(http.StatusOK, "Billing plans list", ListPlansResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := plans.GET("/:planId", p.handleGetPlan,
		forge.WithSummary("Get billing plan"),
		forge.WithOperationID("getBillingPlan"),
		forge.WithResponseSchema(http.StatusOK, "Billing plan details", PlanResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := plans.POST("", p.handleCreatePlan,
		forge.WithSummary("Create billing plan"),
		forge.WithOperationID("createBillingPlan"),
		forge.WithCreatedResponse(PlanResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := plans.POST("/:planId/archive", p.handleArchivePlan,
		forge.WithSummary("Archive a billing plan"),
		forge.WithOperationID("archiveBillingPlan"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := plans.POST("/:planId/activate", p.handleActivatePlan,
		forge.WithSummary("Activate a billing plan"),
		forge.WithOperationID("activateBillingPlan"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Subscription routes
	subs := router.Group(prefix+"/subscriptions", forge.WithGroupTags("Billing Subscriptions"))

	if err := subs.GET("", p.handleListSubscriptions,
		forge.WithSummary("List subscriptions"),
		forge.WithOperationID("listSubscriptions"),
		forge.WithResponseSchema(http.StatusOK, "Subscriptions list", ListSubscriptionsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.GET("/active", p.handleGetActiveSubscription,
		forge.WithSummary("Get active subscription"),
		forge.WithOperationID("getActiveSubscription"),
		forge.WithResponseSchema(http.StatusOK, "Active subscription", Response{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.POST("", p.handleCreateSubscription,
		forge.WithSummary("Create subscription"),
		forge.WithOperationID("createSubscription"),
		forge.WithCreatedResponse(Response{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.POST("/:subId/change-plan", p.handleChangePlan,
		forge.WithSummary("Change subscription plan"),
		forge.WithOperationID("changeSubscriptionPlan"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.POST("/:subId/cancel", p.handleCancelSubscription,
		forge.WithSummary("Cancel subscription"),
		forge.WithOperationID("cancelSubscription"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.POST("/:subId/pause", p.handlePauseSubscription,
		forge.WithSummary("Pause subscription"),
		forge.WithOperationID("pauseSubscription"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := subs.POST("/:subId/resume", p.handleResumeSubscription,
		forge.WithSummary("Resume subscription"),
		forge.WithOperationID("resumeSubscription"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Invoice routes
	invoices := router.Group(prefix+"/invoices", forge.WithGroupTags("Billing Invoices"))

	if err := invoices.GET("", p.handleListInvoices,
		forge.WithSummary("List invoices"),
		forge.WithOperationID("listInvoices"),
		forge.WithResponseSchema(http.StatusOK, "Invoices list", ListInvoicesResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := invoices.GET("/:invoiceId", p.handleGetInvoice,
		forge.WithSummary("Get invoice"),
		forge.WithOperationID("getInvoice"),
		forge.WithResponseSchema(http.StatusOK, "Invoice details", InvoiceResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := invoices.POST("/:invoiceId/pay", p.handleMarkInvoicePaid,
		forge.WithSummary("Mark invoice as paid"),
		forge.WithOperationID("markInvoicePaid"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := invoices.POST("/:invoiceId/void", p.handleVoidInvoice,
		forge.WithSummary("Void an invoice"),
		forge.WithOperationID("voidInvoice"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Coupon routes
	coupons := router.Group(prefix+"/coupons", forge.WithGroupTags("Billing Coupons"))

	if err := coupons.GET("", p.handleListCoupons,
		forge.WithSummary("List coupons"),
		forge.WithOperationID("listCoupons"),
		forge.WithResponseSchema(http.StatusOK, "Coupons list", ListCouponsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := coupons.POST("", p.handleCreateCoupon,
		forge.WithSummary("Create coupon"),
		forge.WithOperationID("createCoupon"),
		forge.WithCreatedResponse(CouponResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := coupons.DELETE("/:couponId", p.handleDeleteCoupon,
		forge.WithSummary("Delete coupon"),
		forge.WithOperationID("deleteCoupon"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Entitlement routes
	ent := router.Group(prefix+"/entitlements", forge.WithGroupTags("Billing Entitlements"))

	if err := ent.GET("/:featureKey", p.handleCheckEntitlement,
		forge.WithSummary("Check feature entitlement"),
		forge.WithOperationID("checkEntitlement"),
		forge.WithResponseSchema(http.StatusOK, "Entitlement result", EntitlementResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Usage routes
	usage := router.Group(prefix+"/usage", forge.WithGroupTags("Billing Usage"))

	return usage.GET("", p.handleGetUsageSummary,
		forge.WithSummary("Get usage summary"),
		forge.WithOperationID("getUsageSummary"),
		forge.WithResponseSchema(http.StatusOK, "Usage summary", UsageSummaryResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// Plan types

type ListPlansRequest struct {
	AppID string `query:"app_id"`
}

type ListPlansResponse struct {
	Plans []PlanResponse `json:"plans"`
	Total int            `json:"total"`
}

type GetPlanRequest struct {
	PlanID string `path:"planId"`
}

type CreatePlanRequest struct {
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	Currency    string            `json:"currency"`
	TrialDays   int               `json:"trial_days"`
	BaseAmount  int64             `json:"base_amount"`
	Period      string            `json:"period"`
	AppID       string            `json:"app_id"`
	IsAddon     bool              `json:"is_addon"`
	Features    []FeatureInput    `json:"features,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type FeatureInput struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Limit     int64  `json:"limit"`
	Period    string `json:"period"`
	SoftLimit bool   `json:"soft_limit"`
}

type PlanResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	Description   string            `json:"description"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	TrialDays     int               `json:"trial_days"`
	BaseAmount    string            `json:"base_amount,omitempty"`
	BillingPeriod string            `json:"billing_period,omitempty"`
	FeaturesCount int               `json:"features_count"`
	IsAddon       bool              `json:"is_addon"`
	AppID         string            `json:"app_id"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type PlanIDRequest struct {
	PlanID string `path:"planId"`
}

// Subscription types

type ListSubscriptionsRequest struct {
	AppID    string `query:"app_id"`
	TenantID string `query:"tenant_id"`
	Status   string `query:"status,omitempty"`
}

type ListSubscriptionsResponse struct {
	Subscriptions []Response `json:"subscriptions"`
	Total         int        `json:"total"`
}

type GetActiveSubRequest struct {
	AppID    string `query:"app_id"`
	TenantID string `query:"tenant_id"`
}

type CreateSubscriptionRequest struct {
	TenantID string `json:"tenant_id"`
	PlanID   string `json:"plan_id"`
	AppID    string `json:"app_id"`
}

type ChangePlanRequest struct {
	SubID     string `path:"subId"`
	NewPlanID string `json:"new_plan_id"`
}

type CancelSubscriptionRequest struct {
	SubID       string `path:"subId"`
	Immediately bool   `json:"immediately"`
}

type SubIDRequest struct {
	SubID string `path:"subId"`
}

type Response struct {
	ID                 string  `json:"id"`
	TenantID           string  `json:"tenant_id"`
	PlanID             string  `json:"plan_id"`
	Status             string  `json:"status"`
	CurrentPeriodStart string  `json:"current_period_start"`
	CurrentPeriodEnd   string  `json:"current_period_end"`
	TrialStart         *string `json:"trial_start,omitempty"`
	TrialEnd           *string `json:"trial_end,omitempty"`
	CanceledAt         *string `json:"canceled_at,omitempty"`
	AppID              string  `json:"app_id"`
}

// Invoice types

type ListInvoicesRequest struct {
	AppID    string `query:"app_id"`
	TenantID string `query:"tenant_id"`
}

type ListInvoicesResponse struct {
	Invoices []InvoiceResponse `json:"invoices"`
	Total    int               `json:"total"`
}

type GetInvoiceRequest struct {
	InvoiceID string `path:"invoiceId"`
}

type InvoiceIDRequest struct {
	InvoiceID string `path:"invoiceId"`
}

type MarkInvoicePaidRequest struct {
	InvoiceID  string `path:"invoiceId"`
	PaymentRef string `json:"payment_ref"`
}

type VoidInvoiceRequest struct {
	InvoiceID string `path:"invoiceId"`
	Reason    string `json:"reason"`
}

type InvoiceResponse struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	SubscriptionID string `json:"subscription_id"`
	Status         string `json:"status"`
	Currency       string `json:"currency"`
	Total          string `json:"total"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
}

// Coupon types

type ListCouponsRequest struct {
	AppID string `query:"app_id"`
}

type ListCouponsResponse struct {
	Coupons []CouponResponse `json:"coupons"`
	Total   int              `json:"total"`
}

type CreateCouponRequest struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Amount         int64   `json:"amount"`
	Percentage     int     `json:"percentage"`
	Currency       string  `json:"currency"`
	MaxRedemptions int     `json:"max_redemptions"`
	ValidFrom      *string `json:"valid_from,omitempty"`
	ValidUntil     *string `json:"valid_until,omitempty"`
	AppID          string  `json:"app_id"`
}

type DeleteCouponRequest struct {
	CouponID string `path:"couponId"`
}

type CouponResponse struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Amount         string  `json:"amount,omitempty"`
	Percentage     int     `json:"percentage,omitempty"`
	Currency       string  `json:"currency"`
	MaxRedemptions int     `json:"max_redemptions"`
	TimesRedeemed  int     `json:"times_redeemed"`
	ValidFrom      *string `json:"valid_from,omitempty"`
	ValidUntil     *string `json:"valid_until,omitempty"`
	AppID          string  `json:"app_id"`
}

// Entitlement types

type CheckEntitlementRequest struct {
	FeatureKey string `path:"featureKey"`
}

type EntitlementResponse struct {
	Allowed   bool   `json:"allowed"`
	Feature   string `json:"feature"`
	Used      int64  `json:"used"`
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
	Reason    string `json:"reason,omitempty"`
}

// Usage types

type UsageSummaryRequest struct {
	AppID    string `query:"app_id"`
	TenantID string `query:"tenant_id"`
}

type UsageSummaryResponse struct {
	Usage []UsageItemResponse `json:"usage"`
}

type UsageItemResponse struct {
	FeatureKey  string `json:"feature_key"`
	FeatureName string `json:"feature_name"`
	FeatureType string `json:"feature_type"`
	Used        int64  `json:"used"`
	Limit       int64  `json:"limit"`
	Remaining   int64  `json:"remaining"`
	Period      string `json:"period"`
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleListPlans(ctx forge.Context, req *ListPlansRequest) (*ListPlansResponse, error) {
	appID := req.AppID
	if appID == "" {
		appID = p.defaultAppID
	}

	plans, err := p.service.ListPlans(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list plans: %w", err))
	}

	items := make([]PlanResponse, 0, len(plans))
	for _, pl := range plans {
		items = append(items, toPlanResponse(pl))
	}

	return &ListPlansResponse{Plans: items, Total: len(items)}, nil
}

func (p *Plugin) handleGetPlan(ctx forge.Context, req *GetPlanRequest) (*PlanResponse, error) {
	planID, err := ledgerid.ParsePlanID(req.PlanID)
	if err != nil {
		return nil, forge.BadRequest("invalid plan_id")
	}

	pl, err := p.service.GetPlan(ctx.Context(), planID)
	if err != nil {
		return nil, forge.NotFound("plan not found")
	}

	resp := toPlanResponse(pl)
	return &resp, nil
}

func (p *Plugin) handleCreatePlan(ctx forge.Context, req *CreatePlanRequest) (*PlanResponse, error) {
	if req.Name == "" || req.Slug == "" || req.AppID == "" {
		return nil, forge.BadRequest("name, slug, and app_id are required")
	}

	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	pl := &plan.Plan{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Currency:    currency,
		Status:      plan.StatusActive,
		TrialDays:   req.TrialDays,
		AppID:       req.AppID,
		Metadata:    req.Metadata,
	}

	if req.IsAddon {
		if pl.Metadata == nil {
			pl.Metadata = make(map[string]string)
		}
		pl.Metadata["addon"] = "true"
	}

	// Set pricing.
	if req.BaseAmount > 0 {
		period := plan.PeriodMonthly
		if req.Period == "yearly" {
			period = plan.PeriodYearly
		}
		pl.Pricing = &plan.Pricing{
			BaseAmount:    types.Money{Amount: req.BaseAmount, Currency: currency},
			BillingPeriod: period,
		}
	}

	// Add features.
	for _, f := range req.Features {
		pl.Features = append(pl.Features, plan.Feature{
			ID:        ledgerid.NewFeatureID(),
			Key:       f.Key,
			Name:      f.Name,
			Type:      plan.FeatureType(f.Type),
			Limit:     f.Limit,
			Period:    plan.Period(f.Period),
			SoftLimit: f.SoftLimit,
		})
	}

	if err := p.service.CreatePlan(ctx.Context(), pl); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create plan: %w", err))
	}

	p.audit(ctx.Context(), "plan.create", "plan", pl.ID.String(), "", "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "plan.created", "", map[string]string{"plan_slug": pl.Slug})

	resp := toPlanResponse(pl)
	return nil, ctx.JSON(http.StatusCreated, resp)
}

func (p *Plugin) handleArchivePlan(ctx forge.Context, req *PlanIDRequest) (*struct{}, error) {
	planID, err := ledgerid.ParsePlanID(req.PlanID)
	if err != nil {
		return nil, forge.BadRequest("invalid plan_id")
	}

	if err := p.service.ArchivePlan(ctx.Context(), planID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to archive plan: %w", err))
	}

	p.audit(ctx.Context(), "plan.archive", "plan", planID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleActivatePlan(ctx forge.Context, req *PlanIDRequest) (*struct{}, error) {
	planID, err := ledgerid.ParsePlanID(req.PlanID)
	if err != nil {
		return nil, forge.BadRequest("invalid plan_id")
	}

	if err := p.service.ActivatePlan(ctx.Context(), planID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to activate plan: %w", err))
	}

	p.audit(ctx.Context(), "plan.activate", "plan", planID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleListSubscriptions(ctx forge.Context, req *ListSubscriptionsRequest) (*ListSubscriptionsResponse, error) {
	appID := req.AppID
	if appID == "" {
		appID = p.defaultAppID
	}

	opts := lsub.ListOpts{}
	if req.Status != "" {
		opts.Status = lsub.Status(req.Status)
	}

	subs, err := p.service.ListSubscriptions(ctx.Context(), req.TenantID, appID, opts)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list subscriptions: %w", err))
	}

	items := make([]Response, 0, len(subs))
	for _, s := range subs {
		items = append(items, toSubResponse(s))
	}

	return &ListSubscriptionsResponse{Subscriptions: items, Total: len(items)}, nil
}

func (p *Plugin) handleGetActiveSubscription(ctx forge.Context, req *GetActiveSubRequest) (*Response, error) {
	if req.AppID == "" || req.TenantID == "" {
		return nil, forge.BadRequest("app_id and tenant_id are required")
	}

	sub, err := p.service.GetActiveSubscription(ctx.Context(), req.TenantID, req.AppID)
	if err != nil {
		return nil, forge.NotFound("no active subscription found")
	}

	resp := toSubResponse(sub)
	return &resp, nil
}

func (p *Plugin) handleCreateSubscription(ctx forge.Context, req *CreateSubscriptionRequest) (*Response, error) {
	if req.TenantID == "" || req.PlanID == "" || req.AppID == "" {
		return nil, forge.BadRequest("tenant_id, plan_id, and app_id are required")
	}

	planID, err := ledgerid.ParsePlanID(req.PlanID)
	if err != nil {
		return nil, forge.BadRequest("invalid plan_id")
	}

	sub, err := p.service.Subscribe(ctx.Context(), req.TenantID, planID, req.AppID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create subscription: %w", err))
	}

	p.audit(ctx.Context(), "subscription.create", "subscription", sub.ID.String(), req.TenantID, req.TenantID, bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "subscription.created", req.TenantID, map[string]string{"plan_id": req.PlanID})

	resp := toSubResponse(sub)
	return nil, ctx.JSON(http.StatusCreated, resp)
}

func (p *Plugin) handleChangePlan(ctx forge.Context, req *ChangePlanRequest) (*struct{}, error) {
	subID, err := ledgerid.ParseSubscriptionID(req.SubID)
	if err != nil {
		return nil, forge.BadRequest("invalid subscription id")
	}

	newPlanID, err := ledgerid.ParsePlanID(req.NewPlanID)
	if err != nil {
		return nil, forge.BadRequest("invalid new_plan_id")
	}

	if err := p.service.ChangePlan(ctx.Context(), subID, newPlanID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to change plan: %w", err))
	}

	p.audit(ctx.Context(), "subscription.plan_changed", "subscription", subID.String(), "", "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "subscription.plan_changed", "", map[string]string{"new_plan_id": req.NewPlanID})

	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleCancelSubscription(ctx forge.Context, req *CancelSubscriptionRequest) (*struct{}, error) {
	subID, err := ledgerid.ParseSubscriptionID(req.SubID)
	if err != nil {
		return nil, forge.BadRequest("invalid subscription id")
	}

	if err := p.service.CancelSubscription(ctx.Context(), subID, req.Immediately); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to cancel subscription: %w", err))
	}

	p.audit(ctx.Context(), "subscription.canceled", "subscription", subID.String(), "", "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "subscription.canceled", "", map[string]string{
		"immediately": boolStr(req.Immediately),
	})

	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handlePauseSubscription(ctx forge.Context, req *SubIDRequest) (*struct{}, error) {
	subID, err := ledgerid.ParseSubscriptionID(req.SubID)
	if err != nil {
		return nil, forge.BadRequest("invalid subscription id")
	}

	if err := p.service.PauseSubscription(ctx.Context(), subID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to pause subscription: %w", err))
	}

	p.audit(ctx.Context(), "subscription.paused", "subscription", subID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleResumeSubscription(ctx forge.Context, req *SubIDRequest) (*struct{}, error) {
	subID, err := ledgerid.ParseSubscriptionID(req.SubID)
	if err != nil {
		return nil, forge.BadRequest("invalid subscription id")
	}

	if err := p.service.ResumeSubscription(ctx.Context(), subID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to resume subscription: %w", err))
	}

	p.audit(ctx.Context(), "subscription.resumed", "subscription", subID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleListInvoices(ctx forge.Context, req *ListInvoicesRequest) (*ListInvoicesResponse, error) {
	appID := req.AppID
	if appID == "" {
		appID = p.defaultAppID
	}

	invoices, err := p.service.ListAllInvoices(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list invoices: %w", err))
	}

	items := make([]InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, InvoiceResponse{
			ID:             inv.ID.String(),
			TenantID:       inv.TenantID,
			SubscriptionID: inv.SubscriptionID.String(),
			Status:         string(inv.Status),
			Currency:       inv.Currency,
			Total:          inv.Total.FormatMajor(),
			PeriodStart:    inv.PeriodStart.Format("2006-01-02"),
			PeriodEnd:      inv.PeriodEnd.Format("2006-01-02"),
		})
	}

	return &ListInvoicesResponse{Invoices: items, Total: len(items)}, nil
}

func (p *Plugin) handleGetInvoice(ctx forge.Context, req *GetInvoiceRequest) (*InvoiceResponse, error) {
	invID, err := ledgerid.ParseInvoiceID(req.InvoiceID)
	if err != nil {
		return nil, forge.BadRequest("invalid invoice_id")
	}

	inv, err := p.service.GetInvoice(ctx.Context(), invID)
	if err != nil {
		return nil, forge.NotFound("invoice not found")
	}

	return &InvoiceResponse{
		ID:             inv.ID.String(),
		TenantID:       inv.TenantID,
		SubscriptionID: inv.SubscriptionID.String(),
		Status:         string(inv.Status),
		Currency:       inv.Currency,
		Total:          inv.Total.FormatMajor(),
		PeriodStart:    inv.PeriodStart.Format("2006-01-02"),
		PeriodEnd:      inv.PeriodEnd.Format("2006-01-02"),
	}, nil
}

func (p *Plugin) handleMarkInvoicePaid(ctx forge.Context, req *MarkInvoicePaidRequest) (*struct{}, error) {
	invID, err := ledgerid.ParseInvoiceID(req.InvoiceID)
	if err != nil {
		return nil, forge.BadRequest("invalid invoice_id")
	}

	paymentRef := req.PaymentRef
	if paymentRef == "" {
		paymentRef = "api"
	}

	if err := p.service.MarkInvoicePaid(ctx.Context(), invID, paymentRef); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to mark invoice paid: %w", err))
	}

	p.audit(ctx.Context(), "invoice.paid", "invoice", invID.String(), "", "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "invoice.paid", "", map[string]string{"invoice_id": invID.String()})

	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleVoidInvoice(ctx forge.Context, req *VoidInvoiceRequest) (*struct{}, error) {
	invID, err := ledgerid.ParseInvoiceID(req.InvoiceID)
	if err != nil {
		return nil, forge.BadRequest("invalid invoice_id")
	}

	if err := p.service.MarkInvoiceVoided(ctx.Context(), invID, req.Reason); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to void invoice: %w", err))
	}

	p.audit(ctx.Context(), "invoice.voided", "invoice", invID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleListCoupons(ctx forge.Context, req *ListCouponsRequest) (*ListCouponsResponse, error) {
	appID := req.AppID
	if appID == "" {
		appID = p.defaultAppID
	}

	coupons, err := p.service.ListCoupons(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list coupons: %w", err))
	}

	items := make([]CouponResponse, 0, len(coupons))
	for _, c := range coupons {
		items = append(items, toCouponResponse(c))
	}

	return &ListCouponsResponse{Coupons: items, Total: len(items)}, nil
}

func (p *Plugin) handleCreateCoupon(ctx forge.Context, req *CreateCouponRequest) (*CouponResponse, error) {
	if req.Code == "" || req.Name == "" || req.AppID == "" {
		return nil, forge.BadRequest("code, name, and app_id are required")
	}

	c := &coupon.Coupon{
		Code:           strings.ToUpper(req.Code),
		Name:           req.Name,
		MaxRedemptions: req.MaxRedemptions,
		AppID:          req.AppID,
	}

	if req.Currency != "" {
		c.Currency = req.Currency
	} else {
		c.Currency = "usd"
	}

	switch req.Type {
	case "percentage":
		c.Type = coupon.CouponTypePercentage
		c.Percentage = req.Percentage
	default:
		c.Type = coupon.CouponTypeAmount
		c.Amount = types.Money{Amount: req.Amount, Currency: c.Currency}
	}

	if req.ValidFrom != nil {
		t, err := time.Parse("2006-01-02", *req.ValidFrom)
		if err == nil {
			c.ValidFrom = &t
		}
	}
	if req.ValidUntil != nil {
		t, err := time.Parse("2006-01-02", *req.ValidUntil)
		if err == nil {
			c.ValidUntil = &t
		}
	}

	if err := p.service.CreateCoupon(ctx.Context(), c); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create coupon: %w", err))
	}

	p.audit(ctx.Context(), "coupon.create", "coupon", c.ID.String(), "", "", bridge.OutcomeSuccess)

	resp := toCouponResponse(c)
	return nil, ctx.JSON(http.StatusCreated, resp)
}

func (p *Plugin) handleDeleteCoupon(ctx forge.Context, req *DeleteCouponRequest) (*struct{}, error) {
	couponID, err := ledgerid.ParseCouponID(req.CouponID)
	if err != nil {
		return nil, forge.BadRequest("invalid coupon_id")
	}

	if err := p.service.DeleteCoupon(ctx.Context(), couponID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to delete coupon: %w", err))
	}

	p.audit(ctx.Context(), "coupon.delete", "coupon", couponID.String(), "", "", bridge.OutcomeSuccess)
	return nil, ctx.NoContent(http.StatusNoContent)
}

func (p *Plugin) handleCheckEntitlement(ctx forge.Context, req *CheckEntitlementRequest) (*EntitlementResponse, error) {
	result, err := p.service.CheckEntitlement(ctx.Context(), req.FeatureKey)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to check entitlement: %w", err))
	}

	return &EntitlementResponse{
		Allowed:   result.Allowed,
		Feature:   result.Feature,
		Used:      result.Used,
		Limit:     result.Limit,
		Remaining: result.Remaining,
		Reason:    result.Reason,
	}, nil
}

func (p *Plugin) handleGetUsageSummary(ctx forge.Context, req *UsageSummaryRequest) (*UsageSummaryResponse, error) {
	if req.AppID == "" || req.TenantID == "" {
		return nil, forge.BadRequest("app_id and tenant_id are required")
	}

	summaries, err := p.service.GetUsageSummary(ctx.Context(), req.TenantID, req.AppID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to get usage summary: %w", err))
	}

	items := make([]UsageItemResponse, 0, len(summaries))
	for _, u := range summaries {
		items = append(items, UsageItemResponse(u))
	}

	return &UsageSummaryResponse{Usage: items}, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func toPlanResponse(pl *plan.Plan) PlanResponse {
	resp := PlanResponse{
		ID:            pl.ID.String(),
		Name:          pl.Name,
		Slug:          pl.Slug,
		Description:   pl.Description,
		Currency:      pl.Currency,
		Status:        string(pl.Status),
		TrialDays:     pl.TrialDays,
		FeaturesCount: len(pl.Features),
		AppID:         pl.AppID,
		Metadata:      pl.Metadata,
	}
	if pl.Metadata != nil && pl.Metadata["addon"] == "true" {
		resp.IsAddon = true
	}
	if pl.Pricing != nil {
		resp.BaseAmount = pl.Pricing.BaseAmount.FormatMajor()
		resp.BillingPeriod = string(pl.Pricing.BillingPeriod)
	}
	return resp
}

func toSubResponse(s *lsub.Subscription) Response {
	resp := Response{
		ID:                 s.ID.String(),
		TenantID:           s.TenantID,
		PlanID:             s.PlanID.String(),
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.Format("2006-01-02"),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.Format("2006-01-02"),
		AppID:              s.AppID,
	}
	if s.TrialStart != nil {
		ts := s.TrialStart.Format("2006-01-02")
		resp.TrialStart = &ts
	}
	if s.TrialEnd != nil {
		te := s.TrialEnd.Format("2006-01-02")
		resp.TrialEnd = &te
	}
	if s.CanceledAt != nil {
		ca := s.CanceledAt.Format("2006-01-02")
		resp.CanceledAt = &ca
	}
	return resp
}

func toCouponResponse(c *coupon.Coupon) CouponResponse {
	resp := CouponResponse{
		ID:             c.ID.String(),
		Code:           c.Code,
		Name:           c.Name,
		Type:           string(c.Type),
		Currency:       c.Currency,
		MaxRedemptions: c.MaxRedemptions,
		TimesRedeemed:  c.TimesRedeemed,
		AppID:          c.AppID,
	}
	if c.Type == coupon.CouponTypePercentage {
		resp.Percentage = c.Percentage
	} else {
		resp.Amount = c.Amount.FormatMajor()
	}
	if c.ValidFrom != nil {
		vf := c.ValidFrom.Format("2006-01-02")
		resp.ValidFrom = &vf
	}
	if c.ValidUntil != nil {
		vu := c.ValidUntil.Format("2006-01-02")
		resp.ValidUntil = &vu
	}
	return resp
}

// parseAmountCents parses a dollar string like "9.99" into cents (999).
func parseAmountCents(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.SplitN(s, ".", 2)
	dollars, _ := strconv.ParseInt(parts[0], 10, 64) //nolint:errcheck // best-effort parse
	var cents int64
	if len(parts) == 2 {
		c := parts[1]
		if len(c) == 1 {
			c += "0"
		} else if len(c) > 2 {
			c = c[:2]
		}
		cents, _ = strconv.ParseInt(c, 10, 64) //nolint:errcheck // best-effort parse
	}
	return dollars*100 + cents
}
