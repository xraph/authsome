package subscription

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	g "maragu.dev/gomponents"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forgeui/router"
)

// HandleCreatePlan handles plan creation form submission
func (e *DashboardExtension) HandleCreatePlan(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Parse form
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	form := ctx.Request.Form
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
		return nil, errs.BadRequest("Plan name is required")
	}

	// Generate slug from name if not provided
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return nil, errs.BadRequest("Invalid price")
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

	plan, err := e.plugin.planSvc.Create(reqCtx, currentApp.ID, req)
	if err != nil {
		return nil, errs.InternalServerError("Failed to create plan", err)
	}

	// Link selected features to the plan
	e.linkFeaturesFromForm(reqCtx, plan.ID, form, currentApp.ID)

	// Redirect to plan detail page
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String(), http.StatusFound)
	return nil, nil
}

// HandleUpdatePlan handles plan update form submission
func (e *DashboardExtension) HandleUpdatePlan(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	// Verify plan exists
	existingPlan, err := e.plugin.planSvc.GetByID(reqCtx, planID)
	if err != nil {
		return nil, errs.NotFound("Plan not found")
	}

	// Parse form
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	form := ctx.Request.Form
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

	_, err = e.plugin.planSvc.Update(reqCtx, planID, updateReq)
	if err != nil {
		return nil, errs.InternalServerError("Failed to update plan", err)
	}

	// Update feature links
	e.linkFeaturesFromForm(reqCtx, planID, form, currentApp.ID)

	// Redirect to plan detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String(), http.StatusFound)
	return nil, nil
}

// HandleArchivePlan handles plan archiving (deactivation)
func (e *DashboardExtension) HandleArchivePlan(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	// Use SetActive(false) to deactivate the plan
	err = e.plugin.planSvc.SetActive(reqCtx, planID, false)
	if err != nil {
		return nil, errs.InternalServerError("Failed to archive plan", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans", http.StatusFound)
	return nil, nil
}

// HandleSyncPlan handles syncing a plan to the payment provider
func (e *DashboardExtension) HandleSyncPlan(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	// Sync plan to provider
	err = e.plugin.planSvc.SyncToProvider(reqCtx, planID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to sync plan to provider", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String(), http.StatusFound)
	return nil, nil
}

// HandleSyncPlanFromProvider syncs a single plan from the payment provider
func (e *DashboardExtension) HandleSyncPlanFromProvider(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	// Get the existing plan to find its provider plan ID
	existingPlan, err := e.plugin.planSvc.GetByID(reqCtx, planID)
	if err != nil {
		return nil, errs.NotFound("Plan not found")
	}

	if existingPlan.ProviderPlanID == "" {
		return nil, errs.BadRequest("Plan is not synced to provider - cannot sync from provider")
	}

	// Sync plan from provider
	_, err = e.plugin.planSvc.SyncFromProvider(reqCtx, existingPlan.ProviderPlanID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to sync plan from provider", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planID.String(), http.StatusFound)
	return nil, nil
}

// HandleSyncAllPlansFromProvider syncs all plans from the payment provider
func (e *DashboardExtension) HandleSyncAllPlansFromProvider(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Sync all plans from provider for this app
	syncedPlans, err := e.plugin.planSvc.SyncAllFromProvider(reqCtx, currentApp.ID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to sync plans from provider", err)
	}

	// Redirect to plans list with success message (plans count synced)
	_ = syncedPlans // Could show success message with count
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans", http.StatusFound)
	return nil, nil
}

// HandleDeletePlan handles permanent plan deletion
func (e *DashboardExtension) HandleDeletePlan(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	// Delete the plan
	err = e.plugin.planSvc.Delete(reqCtx, planID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to delete plan", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans", http.StatusFound)
	return nil, nil
}

// HandleCancelSubscription handles subscription cancellation
func (e *DashboardExtension) HandleCancelSubscription(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	subID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid subscription ID")
	}

	// Cancel at end of period by default
	cancelReq := &core.CancelSubscriptionRequest{
		Immediate: false, // Cancel at end of period
		Reason:    "Canceled via dashboard",
	}
	err = e.plugin.subscriptionSvc.Cancel(reqCtx, subID, cancelReq)
	if err != nil {
		return nil, errs.InternalServerError("Failed to cancel subscription", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions/"+subID.String(), http.StatusFound)
	return nil, nil
}

// HandleCreateAddOn handles add-on creation form submission
func (e *DashboardExtension) HandleCreateAddOn(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Parse form
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	form := ctx.Request.Form
	name := strings.TrimSpace(form.Get("name"))
	description := strings.TrimSpace(form.Get("description"))
	priceStr := form.Get("price")
	currency := form.Get("currency")
	billingType := form.Get("billing_type")
	featureKey := strings.TrimSpace(form.Get("feature_key"))
	isActive := form.Get("is_active") == "true"

	if name == "" {
		return nil, errs.BadRequest("Add-on name is required")
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return nil, errs.BadRequest("Invalid price")
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

	_, err = e.plugin.addOnSvc.Create(reqCtx, currentApp.ID, addOnReq)
	if err != nil {
		return nil, errs.InternalServerError("Failed to create add-on", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/addons", http.StatusFound)
	return nil, nil
}

// HandleSyncInvoices handles syncing invoices from Stripe
func (e *DashboardExtension) HandleSyncInvoices(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Call the sync function from handlers.go
	syncedCount, err := e.plugin.SyncInvoicesFromStripe(reqCtx, nil)
	if err != nil {
		// Redirect back with error message
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/invoices?error=sync_failed", http.StatusFound)
		return nil, nil
	}

	// Redirect back with success message
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/invoices?success=synced&count="+fmt.Sprintf("%d", syncedCount), http.StatusFound)
	return nil, nil
}

// HandleMarkInvoicePaid handles marking an invoice as paid
func (e *DashboardExtension) HandleMarkInvoicePaid(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	invoiceID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid invoice ID")
	}

	err = e.plugin.invoiceSvc.MarkPaid(reqCtx, invoiceID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to mark invoice as paid", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/invoices/"+invoiceID.String(), http.StatusFound)
	return nil, nil
}

// HandleCreateCoupon handles coupon creation form submission
// TODO: Implement when couponSvc is added to Plugin
func (e *DashboardExtension) HandleCreateCoupon(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Coupon service not yet integrated into Plugin
	// For now, redirect back to coupons page
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/coupons", http.StatusFound)
	return nil, nil
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
func (e *DashboardExtension) HandleExportFeaturesAndPlans(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid request")
	}

	reqCtx := ctx.Request.Context()

	// Export features and plans
	exportData, err := e.plugin.exportImportSvc.ExportFeaturesAndPlans(reqCtx, currentApp.ID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to export data", err)
	}

	// Set headers for JSON download
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	ctx.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=features-plans-export-%s.json", time.Now().Format("2006-01-02")))

	_ = exportData  // TODO: Actually write the export data to response
	return nil, nil // Success
}

// HandleImportFeaturesAndPlans imports features and plans from JSON
func (e *DashboardExtension) HandleImportFeaturesAndPlans(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Parse import data
	var importData service.ExportData
	// TODO: Parse JSON from request body properly
	_ = importData

	// Get overwrite flag from query param
	overwriteExisting := ctx.Request.URL.Query().Get("overwrite") == "true"

	// Import features and plans
	_, err = e.plugin.exportImportSvc.ImportFeaturesAndPlans(reqCtx, currentApp.ID, &importData, overwriteExisting)
	if err != nil {
		return nil, fmt.Errorf("failed to import data: %w", err)
	}

	// Return result as JSON for AJAX requests or redirect for form submissions
	if ctx.Request.Header.Get("Accept") == "application/json" {
		return nil, nil // Success
	}

	// Redirect with success message
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features", http.StatusFound)
	return nil, nil
}

// HandleShowImportForm displays the import form page
func (e *DashboardExtension) HandleShowImportForm(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

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
    <a href="%s/app/%s/billing/features" class="back-link">&larr; Back to Features</a>
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
                
                const response = await fetch('%s/app/%s/billing/import?overwrite=' + overwrite, {
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
                        window.location.href = '%s/app/%s/billing/features';
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

	ctx.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = ctx.ResponseWriter.Write([]byte(html))
	return nil, nil
}

// HandleSyncFeature handles syncing a feature to the provider
func (e *DashboardExtension) HandleSyncFeature(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	featureIDStr := ctx.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid feature ID")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Sync feature to provider
	err = e.plugin.featureSvc.SyncToProvider(reqCtx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync feature to provider: %w", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr, http.StatusFound)
	return nil, nil
}

// HandleSyncFeatureFromProvider syncs a feature from the payment provider
func (e *DashboardExtension) HandleSyncFeatureFromProvider(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	featureIDStr := ctx.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid feature ID")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Get the existing feature to find its provider feature ID
	existingFeature, err := e.plugin.featureSvc.GetByID(reqCtx, featureID)
	if err != nil {
		return nil, nil // TODO: Proper g.Node return
	}

	if existingFeature.ProviderFeatureID == "" {
		return nil, errs.BadRequest("Feature is not synced to provider - cannot sync from provider")
	}

	// Sync feature from provider
	_, err = e.plugin.featureSvc.SyncFromProvider(reqCtx, existingFeature.ProviderFeatureID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync feature from provider: %w", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr, http.StatusFound)
	return nil, nil
}

// HandleSyncAllFeaturesFromProvider syncs all features from the payment provider
func (e *DashboardExtension) HandleSyncAllFeaturesFromProvider(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	// Get product ID from query param
	productID := ctx.Query("productId")
	if productID == "" {
		return nil, errs.BadRequest("productId parameter is required")
	}

	// Sync all features from provider for this product
	syncedFeatures, err := e.plugin.featureSvc.SyncAllFromProvider(reqCtx, productID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features?error=sync_failed", http.StatusFound)
		return nil, nil
	}

	// Redirect to features list with success message (features count synced)
	_ = syncedFeatures // Could show success message with count
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features", http.StatusFound)
	return nil, nil
}

// Suppress unused variable warnings
var _ = func() {
	var ctx *router.PageContext
	var e *DashboardExtension
	_ = e.getUserFromContext(ctx)
}
