package subscription

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/service"
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

// HandleExportFeaturesAndPlans exports features and plans as JSON
func (e *DashboardExtension) HandleExportFeaturesAndPlans(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid app context"})
	}

	ctx := c.Request().Context()

	// Export features and plans
	exportData, err := e.plugin.exportImportSvc.ExportFeaturesAndPlans(ctx, currentApp.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to export data: " + err.Error()})
	}

	// Set headers for JSON download
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=features-plans-export-%s.json", time.Now().Format("2006-01-02")))

	return c.JSON(http.StatusOK, exportData)
}

// HandleImportFeaturesAndPlans imports features and plans from JSON
func (e *DashboardExtension) HandleImportFeaturesAndPlans(c forge.Context) error {
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

	// Parse import data
	var importData service.ExportData
	if err := c.BindJSON(&importData); err != nil {
		return c.String(http.StatusBadRequest, "Invalid import data: "+err.Error())
	}

	// Get overwrite flag from query param
	overwriteExisting := c.Query("overwrite") == "true"

	// Import features and plans
	result, err := e.plugin.exportImportSvc.ImportFeaturesAndPlans(ctx, currentApp.ID, &importData, overwriteExisting)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to import data: "+err.Error())
	}

	// Return result as JSON for AJAX requests or redirect for form submissions
	if c.Header("Accept") == "application/json" {
		return c.JSON(http.StatusOK, result)
	}

	// Redirect with success message
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/features")
}

// HandleShowImportForm displays the import form page
func (e *DashboardExtension) HandleShowImportForm(c forge.Context) error {
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

	// Render import form page
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Import Features & Plans</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        textarea { width: 100%%; height: 400px; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace; }
        .checkbox-group { margin: 10px 0; }
        button { padding: 10px 20px; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .back-link { display: inline-block; margin-bottom: 20px; color: #007bff; text-decoration: none; }
        .back-link:hover { text-decoration: underline; }
        .info { background-color: #e7f3ff; padding: 15px; border-left: 4px solid #007bff; margin-bottom: 20px; }
        .file-input { margin-bottom: 10px; }
        #result { margin-top: 20px; padding: 15px; border-radius: 4px; display: none; }
        .success { background-color: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .error { background-color: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
    </style>
</head>
<body>
    <a href="%s/dashboard/app/%s/billing/features" class="back-link">&larr; Back to Features</a>
    <h1>Import Features & Plans</h1>
    
    <div class="info">
        <strong>Import Instructions:</strong>
        <ul>
            <li>Upload a JSON file exported from another environment</li>
            <li>Features will be created or updated based on their unique keys</li>
            <li>Plans will be created (existing plans with same slug will be skipped)</li>
            <li>Check "Overwrite Existing" to update existing features with imported data</li>
        </ul>
    </div>

    <form id="importForm">
        <div class="form-group">
            <label for="fileInput">Select Export File:</label>
            <input type="file" id="fileInput" accept=".json" class="file-input">
        </div>

        <div class="form-group">
            <label for="importData">Or Paste JSON Data:</label>
            <textarea id="importData" placeholder="Paste your exported JSON data here..."></textarea>
        </div>

        <div class="checkbox-group">
            <label>
                <input type="checkbox" id="overwrite" name="overwrite">
                Overwrite existing features
            </label>
        </div>

        <button type="submit">Import Data</button>
    </form>

    <div id="result"></div>

    <script>
        document.getElementById('fileInput').addEventListener('change', function(e) {
            const file = e.target.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = function(e) {
                    document.getElementById('importData').value = e.target.result;
                };
                reader.readAsText(file);
            }
        });

        document.getElementById('importForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const jsonData = document.getElementById('importData').value;
            const overwrite = document.getElementById('overwrite').checked;
            const resultDiv = document.getElementById('result');
            
            if (!jsonData.trim()) {
                resultDiv.className = 'error';
                resultDiv.textContent = 'Please select a file or paste JSON data';
                resultDiv.style.display = 'block';
                return;
            }

            try {
                const data = JSON.parse(jsonData);
                
                const response = await fetch('%s/dashboard/app/%s/billing/import?overwrite=' + overwrite, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Accept': 'application/json'
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    resultDiv.className = 'success';
                    let message = '<strong>Import Successful!</strong><br>';
                    message += 'Features created: ' + result.featuresCreated + '<br>';
                    message += 'Features skipped: ' + result.featuresSkipped + '<br>';
                    message += 'Plans created: ' + result.plansCreated + '<br>';
                    message += 'Plans skipped: ' + result.plansSkipped;
                    
                    if (result.errors && result.errors.length > 0) {
                        message += '<br><br><strong>Errors:</strong><br>';
                        result.errors.forEach(err => {
                            message += '- ' + err + '<br>';
                        });
                    }
                    
                    resultDiv.innerHTML = message;
                    resultDiv.style.display = 'block';

                    // Clear form after success
                    setTimeout(() => {
                        window.location.href = '%s/dashboard/app/%s/billing/features';
                    }, 3000);
                } else {
                    resultDiv.className = 'error';
                    resultDiv.textContent = 'Error: ' + (result.error || 'Import failed');
                    resultDiv.style.display = 'block';
                }
            } catch (err) {
                resultDiv.className = 'error';
                resultDiv.textContent = 'Error: ' + err.message;
                resultDiv.style.display = 'block';
            }
        });
    </script>
</body>
</html>
`, basePath, currentApp.ID.String(), basePath, currentApp.ID.String(), basePath, currentApp.ID.String())

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write([]byte(html))
	return err
}

// HandleSyncFeature handles syncing a feature to the provider
func (e *DashboardExtension) HandleSyncFeature(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	featureIDStr := c.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid feature ID")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	// Sync feature to provider
	err = e.plugin.featureSvc.SyncToProvider(ctx, featureID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync feature to provider: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr)
}

// HandleSyncFeatureFromProvider syncs a feature from the payment provider
func (e *DashboardExtension) HandleSyncFeatureFromProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	featureIDStr := c.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid feature ID")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	// Get the existing feature to find its provider feature ID
	existingFeature, err := e.plugin.featureSvc.GetByID(ctx, featureID)
	if err != nil {
		return c.String(http.StatusNotFound, "Feature not found")
	}

	if existingFeature.ProviderFeatureID == "" {
		return c.String(http.StatusBadRequest, "Feature is not synced to provider - cannot sync from provider")
	}

	// Sync feature from provider
	_, err = e.plugin.featureSvc.SyncFromProvider(ctx, existingFeature.ProviderFeatureID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync feature from provider: "+err.Error())
	}

	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr)
}

// HandleSyncAllFeaturesFromProvider syncs all features from the payment provider
func (e *DashboardExtension) HandleSyncAllFeaturesFromProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	// Get product ID from query param
	productID := c.Query("productId")
	if productID == "" {
		return c.String(http.StatusBadRequest, "productId parameter is required")
	}

	// Sync all features from provider for this product
	syncedFeatures, err := e.plugin.featureSvc.SyncAllFromProvider(ctx, productID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to sync features from provider: "+err.Error())
	}

	// Redirect to features list with success message (features count synced)
	_ = syncedFeatures // Could show success message with count
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/features")
}

// Suppress unused variable warnings
var _ = func() {
	var c forge.Context
	var e *DashboardExtension
	_ = e.getUserFromContext(c)
}
