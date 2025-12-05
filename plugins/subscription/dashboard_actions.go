package subscription

import (
	"context"
	"net/http"
	"net/url"
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
	slug := strings.TrimSpace(form.Get("slug"))
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

	// Generate slug from name if not provided
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid price")
	}

	trialDays, _ := strconv.Atoi(trialDaysStr)

	// Build metadata based on billing pattern
	metadata := make(map[string]any)
	switch billingPattern {
	case "per_seat":
		if v := form.Get("seat_price"); v != "" {
			if seatPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["seat_price"] = seatPrice
			}
		}
		if v := form.Get("min_seats"); v != "" {
			if minSeats, err := strconv.Atoi(v); err == nil {
				metadata["min_seats"] = minSeats
			}
		}
		if v := form.Get("max_seats"); v != "" {
			if maxSeats, err := strconv.Atoi(v); err == nil {
				metadata["max_seats"] = maxSeats
			}
		}
	case "tiered":
		if v := form.Get("tier_unit"); v != "" {
			metadata["unit_name"] = v
		}
	case "usage":
		if v := form.Get("unit_name"); v != "" {
			metadata["unit_name"] = v
		}
		if v := form.Get("unit_price"); v != "" {
			if unitPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["unit_price"] = unitPrice
			}
		}
		if v := form.Get("usage_aggregation"); v != "" {
			metadata["usage_aggregation"] = v
		}
	case "hybrid":
		if v := form.Get("hybrid_base"); v != "" {
			if hybridBase, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["hybrid_base_price"] = hybridBase
			}
		}
		if v := form.Get("hybrid_unit_name"); v != "" {
			metadata["unit_name"] = v
		}
		if v := form.Get("hybrid_unit_price"); v != "" {
			if unitPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["unit_price"] = unitPrice
			}
		}
		if v := form.Get("included_units"); v != "" {
			if includedUnits, err := strconv.Atoi(v); err == nil {
				metadata["included_units"] = includedUnits
			}
		}
	}

	// Create plan using service request
	req := &core.CreatePlanRequest{
		Name:            name,
		Slug:            slug,
		Description:     description,
		BasePrice:       price,
		Currency:        currency,
		BillingInterval: core.BillingInterval(billingInterval),
		BillingPattern:  core.BillingPattern(billingPattern),
		TrialDays:       trialDays,
		IsActive:        isActive,
		IsPublic:        isPublic,
		Metadata:        metadata,
	}

	plan, err := e.plugin.planSvc.Create(ctx, currentApp.ID, req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create plan: "+err.Error())
	}

	// Link selected features to the plan
	e.linkFeaturesFromForm(ctx, plan.ID, form, currentApp.ID)

	// Redirect to plan detail page
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String())
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
	existingPlan, err := e.plugin.planSvc.GetByID(ctx, planID)
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
	billingPattern := form.Get("billing_pattern")

	// Build metadata based on billing pattern
	metadata := existingPlan.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}

	switch billingPattern {
	case "per_seat":
		if v := form.Get("seat_price"); v != "" {
			if seatPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["seat_price"] = seatPrice
			}
		}
		if v := form.Get("min_seats"); v != "" {
			if minSeats, err := strconv.Atoi(v); err == nil {
				metadata["min_seats"] = minSeats
			}
		}
		if v := form.Get("max_seats"); v != "" {
			if maxSeats, err := strconv.Atoi(v); err == nil {
				metadata["max_seats"] = maxSeats
			}
		}
	case "tiered":
		if v := form.Get("tier_unit"); v != "" {
			metadata["unit_name"] = v
		}
	case "usage":
		if v := form.Get("unit_name"); v != "" {
			metadata["unit_name"] = v
		}
		if v := form.Get("unit_price"); v != "" {
			if unitPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["unit_price"] = unitPrice
			}
		}
		if v := form.Get("usage_aggregation"); v != "" {
			metadata["usage_aggregation"] = v
		}
	case "hybrid":
		if v := form.Get("hybrid_base"); v != "" {
			if hybridBase, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["hybrid_base_price"] = hybridBase
			}
		}
		if v := form.Get("hybrid_unit_name"); v != "" {
			metadata["unit_name"] = v
		}
		if v := form.Get("hybrid_unit_price"); v != "" {
			if unitPrice, err := strconv.ParseInt(v, 10, 64); err == nil {
				metadata["unit_price"] = unitPrice
			}
		}
		if v := form.Get("included_units"); v != "" {
			if includedUnits, err := strconv.Atoi(v); err == nil {
				metadata["included_units"] = includedUnits
			}
		}
	}

	updateReq := &core.UpdatePlanRequest{
		Name:        &name,
		Description: &description,
		BasePrice:   &price,
		TrialDays:   &trialDays,
		IsActive:    &isActive,
		IsPublic:    &isPublic,
		Metadata:    metadata,
	}

	_, err = e.plugin.planSvc.Update(ctx, planID, updateReq)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update plan: "+err.Error())
	}

	// Update feature links
	e.linkFeaturesFromForm(ctx, planID, form, currentApp.ID)

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

// HandleSyncPlanFromProvider syncs a single plan from the payment provider
func (e *DashboardExtension) HandleSyncPlanFromProvider(c forge.Context) error {
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

	// Get the existing plan to find its provider plan ID
	existingPlan, err := e.plugin.planSvc.GetByID(ctx, planID)
	if err != nil {
		return c.String(http.StatusNotFound, "Plan not found")
	}

	if existingPlan.ProviderPlanID == "" {
		return c.String(http.StatusBadRequest, "Plan is not synced to provider - cannot sync from provider")
	}

	// Sync plan from provider
	_, err = e.plugin.planSvc.SyncFromProvider(ctx, existingPlan.ProviderPlanID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync plan from provider: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String())
}

// HandleSyncAllPlansFromProvider syncs all plans from the payment provider
func (e *DashboardExtension) HandleSyncAllPlansFromProvider(c forge.Context) error {
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

	// Sync all plans from provider for this app
	syncedPlans, err := e.plugin.planSvc.SyncAllFromProvider(ctx, currentApp.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync plans from provider: "+err.Error())
	}

	// Redirect to plans list with success message (plans count synced)
	_ = syncedPlans // Could show success message with count
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans")
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

// linkFeaturesFromForm processes the feature selection from the plan form
func (e *DashboardExtension) linkFeaturesFromForm(ctx context.Context, planID xid.ID, form url.Values, appID xid.ID) {
	// Get all features for the app
	features, _, _ := e.plugin.featureSvc.List(ctx, appID, "", false, 1, 1000)

	// Get existing linked features
	existingLinks, _ := e.plugin.featureSvc.GetPlanFeatures(ctx, planID)
	existingMap := make(map[string]bool)
	for _, link := range existingLinks {
		existingMap[link.FeatureID.String()] = true
	}

	// Process each feature
	for _, feature := range features {
		featureID := feature.ID.String()
		isSelected := form.Get("feature_"+featureID) != ""
		valueKey := "feature_value_" + featureID
		value := form.Get(valueKey)

		// Handle boolean features - checkbox value
		if feature.Type == core.FeatureTypeBoolean {
			if isSelected {
				value = "true"
			} else {
				value = "false"
			}
		}

		// Handle unlimited features
		if feature.Type == core.FeatureTypeUnlimited && isSelected {
			value = "-1"
		}

		if isSelected {
			// Link or update feature
			if existingMap[featureID] {
				// Update existing link
				updateReq := &core.UpdateLinkRequest{
					Value: &value,
				}
				e.plugin.featureSvc.UpdatePlanLink(ctx, planID, feature.ID, updateReq)
			} else {
				// Create new link
				linkReq := &core.LinkFeatureRequest{
					FeatureID: feature.ID,
					Value:     value,
				}
				e.plugin.featureSvc.LinkToPlan(ctx, planID, linkReq)
			}
		} else if existingMap[featureID] {
			// Remove link if previously linked but now unselected
			e.plugin.featureSvc.UnlinkFromPlan(ctx, planID, feature.ID)
		}
	}
}

// Suppress unused variable warnings
var _ = func() {
	var c forge.Context
	var e *DashboardExtension
	_ = e.getUserFromContext(c)
}
