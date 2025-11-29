package subscription

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forge"
)

// HandleCreatePlan handles plan creation form submission
func (e *DashboardExtension) HandleCreatePlan(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := strings.TrimSpace(form.Get("name"))
	description := strings.TrimSpace(form.Get("description"))
	priceStr := form.Get("price")
	currency := form.Get("currency")
	billingInterval := form.Get("billing_interval")
	billingPattern := form.Get("billing_pattern")
	trialDaysStr := form.Get("trial_days")
	isActive := form.Get("is_active") == "true"
	isPublic := form.Get("is_public") == "true"

	if name == "" {
		return c.String(http.StatusBadRequest, "Plan name is required")
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid price")
	}

	trialDays, _ := strconv.Atoi(trialDaysStr)

	// Create plan using service request
	req := &core.CreatePlanRequest{
		Name:            name,
		Slug:            strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Description:     description,
		BasePrice:       price,
		Currency:        currency,
		BillingInterval: core.BillingInterval(billingInterval),
		BillingPattern:  core.BillingPattern(billingPattern),
		TrialDays:       trialDays,
		IsActive:        isActive,
		IsPublic:        isPublic,
	}

	_, err = e.plugin.planSvc.Create(ctx, currentApp.ID, req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create plan: "+err.Error())
	}

	// Redirect to plans list
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans")
}

// HandleUpdatePlan handles plan update form submission
func (e *DashboardExtension) HandleUpdatePlan(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	planID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid plan ID")
	}

	// Verify plan exists
	_, err = e.plugin.planSvc.GetByID(ctx, planID)
	if err != nil {
		return c.String(http.StatusNotFound, "Plan not found")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := strings.TrimSpace(form.Get("name"))
	description := strings.TrimSpace(form.Get("description"))
	price, _ := strconv.ParseInt(form.Get("price"), 10, 64)
	trialDays, _ := strconv.Atoi(form.Get("trial_days"))
	isActive := form.Get("is_active") == "true"
	isPublic := form.Get("is_public") == "true"

	updateReq := &core.UpdatePlanRequest{
		Name:        &name,
		Description: &description,
		BasePrice:   &price,
		TrialDays:   &trialDays,
		IsActive:    &isActive,
		IsPublic:    &isPublic,
	}

	_, err = e.plugin.planSvc.Update(ctx, planID, updateReq)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update plan: "+err.Error())
	}

	// Redirect to plan detail
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String())
}

// HandleArchivePlan handles plan archiving (deactivation)
func (e *DashboardExtension) HandleArchivePlan(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	planID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid plan ID")
	}

	// Use SetActive(false) to deactivate the plan
	err = e.plugin.planSvc.SetActive(ctx, planID, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to archive plan: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans")
}

// HandleSyncPlan handles syncing a plan to the payment provider
func (e *DashboardExtension) HandleSyncPlan(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	planID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid plan ID")
	}

	// Sync plan to provider
	err = e.plugin.planSvc.SyncToProvider(ctx, planID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync plan to provider: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String())
}

// HandleDeletePlan handles permanent plan deletion
func (e *DashboardExtension) HandleDeletePlan(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	planID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid plan ID")
	}

	// Delete the plan
	err = e.plugin.planSvc.Delete(ctx, planID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete plan: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans")
}

// HandleCancelSubscription handles subscription cancellation
func (e *DashboardExtension) HandleCancelSubscription(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	subID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid subscription ID")
	}

	// Cancel at end of period by default
	cancelReq := &core.CancelSubscriptionRequest{
		Immediate: false, // Cancel at end of period
		Reason:    "Canceled via dashboard",
	}
	err = e.plugin.subscriptionSvc.Cancel(ctx, subID, cancelReq)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to cancel subscription: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/subscriptions/"+subID.String())
}

// HandleCreateAddOn handles add-on creation form submission
func (e *DashboardExtension) HandleCreateAddOn(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := strings.TrimSpace(form.Get("name"))
	description := strings.TrimSpace(form.Get("description"))
	priceStr := form.Get("price")
	currency := form.Get("currency")
	billingType := form.Get("billing_type")
	featureKey := strings.TrimSpace(form.Get("feature_key"))
	isActive := form.Get("is_active") == "true"

	if name == "" {
		return c.String(http.StatusBadRequest, "Add-on name is required")
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid price")
	}

	_ = featureKey // Not used in CreateAddOnRequest

	addOnReq := &core.CreateAddOnRequest{
		Name:            name,
		Slug:            strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Description:     description,
		Price:           price,
		Currency:        currency,
		BillingPattern:  core.BillingPattern(billingType),
		BillingInterval: core.BillingIntervalMonthly,
		IsActive:        isActive,
		IsPublic:        true,
	}

	_, err = e.plugin.addOnSvc.Create(ctx, currentApp.ID, addOnReq)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create add-on: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/addons")
}

// HandleMarkInvoicePaid handles marking an invoice as paid
func (e *DashboardExtension) HandleMarkInvoicePaid(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	invoiceID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid invoice ID")
	}

	err = e.plugin.invoiceSvc.MarkPaid(ctx, invoiceID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to mark invoice as paid: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/invoices/"+invoiceID.String())
}

// HandleCreateCoupon handles coupon creation form submission
// TODO: Implement when couponSvc is added to Plugin
func (e *DashboardExtension) HandleCreateCoupon(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Coupon service not yet integrated into Plugin
	// For now, redirect back to coupons page
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/coupons")
}

// Suppress unused variable warnings
var _ = func() {
	var c forge.Context
	var e *DashboardExtension
	_ = e.getUserFromContext(c)
}
