package subscription

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// queryIntDefault gets an integer query parameter with a default value
func queryIntDefault(c forge.Context, name string, defaultValue int) int {
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

// getAppIDFromContext gets the app ID from the context
func getAppIDFromContext(c forge.Context) xid.ID {
	// Get app ID from context (set by auth middleware)
	appID := c.Get("appID")
	if appID != nil {
		if id, ok := appID.(xid.ID); ok {
			return id
		}
	}
	// Try to get from header
	if appIDStr := c.Request().Header.Get("X-App-ID"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			return id
		}
	}
	return xid.ID{}
}

// DashboardExtension implements ui.DashboardExtension for the subscription plugin
type DashboardExtension struct {
	plugin *Plugin
}

// NewDashboardExtension creates a new dashboard extension
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "subscription"
}

// NavigationItems returns the navigation items for the dashboard
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "subscription-billing",
			Label:    "Billing",
			Icon:     html.Span(g.Text("💳")), // Using emoji as placeholder, should use lucide icon
			Position: ui.NavPositionMain,
			Order:    50,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/dashboard/billing"
				}
				return basePath + "/dashboard/app/" + currentApp.ID.String() + "/billing"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "billing" || activePage == "plans" || activePage == "subscriptions" ||
					activePage == "addons" || activePage == "invoices" || activePage == "usage"
			},
		},
	}
}

// Routes returns the dashboard routes
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Billing Overview
		{
			Method:      "GET",
			Path:        "/billing",
			Handler:     e.ServeBillingOverviewPage,
			Name:        "subscription.billing.overview",
			Summary:     "Billing Overview",
			Description: "View billing overview and summary",
			RequireAuth: true,
		},
		// Plans
		{
			Method:      "GET",
			Path:        "/billing/plans",
			Handler:     e.ServePlansListPage,
			Name:        "subscription.plans.list",
			Summary:     "List Plans",
			Description: "View all subscription plans",
			RequireAuth: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/plans/create",
			Handler:      e.ServePlanCreatePage,
			Name:         "subscription.plans.create",
			Summary:      "Create Plan",
			Description:  "Create a new subscription plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:      "GET",
			Path:        "/billing/plans/:id",
			Handler:     e.ServePlanDetailPage,
			Name:        "subscription.plans.detail",
			Summary:     "Plan Details",
			Description: "View plan details",
			RequireAuth: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/plans/:id/edit",
			Handler:      e.ServePlanEditPage,
			Name:         "subscription.plans.edit",
			Summary:      "Edit Plan",
			Description:  "Edit an existing plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Subscriptions
		{
			Method:      "GET",
			Path:        "/billing/subscriptions",
			Handler:     e.ServeSubscriptionsListPage,
			Name:        "subscription.subscriptions.list",
			Summary:     "List Subscriptions",
			Description: "View all subscriptions",
			RequireAuth: true,
		},
		{
			Method:      "GET",
			Path:        "/billing/subscriptions/:id",
			Handler:     e.ServeSubscriptionDetailPage,
			Name:        "subscription.subscriptions.detail",
			Summary:     "Subscription Details",
			Description: "View subscription details",
			RequireAuth: true,
		},
		// Add-ons
		{
			Method:      "GET",
			Path:        "/billing/addons",
			Handler:     e.ServeAddOnsListPage,
			Name:        "subscription.addons.list",
			Summary:     "List Add-ons",
			Description: "View all add-ons",
			RequireAuth: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/addons/create",
			Handler:      e.ServeAddOnCreatePage,
			Name:         "subscription.addons.create",
			Summary:      "Create Add-on",
			Description:  "Create a new add-on",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:      "GET",
			Path:        "/billing/addons/:id",
			Handler:     e.ServeAddOnDetailPage,
			Name:        "subscription.addons.detail",
			Summary:     "Add-on Details",
			Description: "View add-on details",
			RequireAuth: true,
		},
		// Invoices
		{
			Method:      "GET",
			Path:        "/billing/invoices",
			Handler:     e.ServeInvoicesListPage,
			Name:        "subscription.invoices.list",
			Summary:     "List Invoices",
			Description: "View all invoices",
			RequireAuth: true,
		},
		{
			Method:      "GET",
			Path:        "/billing/invoices/:id",
			Handler:     e.ServeInvoiceDetailPage,
			Name:        "subscription.invoices.detail",
			Summary:     "Invoice Details",
			Description: "View invoice details",
			RequireAuth: true,
		},
		// Usage
		{
			Method:      "GET",
			Path:        "/billing/usage",
			Handler:     e.ServeUsageDashboardPage,
			Name:        "subscription.usage.dashboard",
			Summary:     "Usage Dashboard",
			Description: "View usage metrics and reports",
			RequireAuth: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages for the plugin
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:          "subscription-settings",
			Label:       "Billing & Subscription",
			Description: "Configure subscription and billing settings",
			Icon:        html.Span(g.Text("💳")), // Using emoji as placeholder
			Category:    "general",
			Order:       30,
			Path:        "billing",
		},
	}
}

// DashboardWidgets returns dashboard widgets
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "subscription-mrr",
			Title: "Monthly Recurring Revenue",
			Icon:  html.Span(g.Text("💰")),
			Order: 10,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				// This would render the MRR widget content
				return html.Div(g.Text("$0.00"))
			},
		},
		{
			ID:    "subscription-active-count",
			Title: "Active Subscriptions",
			Icon:  html.Span(g.Text("📊")),
			Order: 11,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return html.Div(g.Text("0"))
			},
		},
	}
}

// Page Handlers

func (e *DashboardExtension) ServeBillingOverviewPage(c forge.Context) error {
	appID := getAppIDFromContext(c)

	// Get summary data
	plans, planCount, _ := e.plugin.planSvc.List(c.Context(), appID, false, false, 1, 100)
	subs, subCount, _ := e.plugin.subscriptionSvc.List(c.Context(), nil, nil, nil, "", 1, 100)

	// Count active subscriptions
	var activeCount, trialingCount int
	for _, sub := range subs {
		if sub.Status == "active" {
			activeCount++
		} else if sub.Status == "trialing" {
			trialingCount++
		}
	}

	data := map[string]interface{}{
		"title":            "Billing Overview",
		"totalPlans":       planCount,
		"totalSubs":        subCount,
		"activeSubs":       activeCount,
		"trialingSubs":     trialingCount,
		"plans":            plans,
		"stripeConfigured": e.plugin.config.IsStripeConfigured(),
	}

	return e.renderPage(c, "billing_overview", data)
}

func (e *DashboardExtension) ServePlansListPage(c forge.Context) error {
	appID := getAppIDFromContext(c)
	page := queryIntDefault(c, "page", 1)
	pageSize := queryIntDefault(c, "pageSize", 20)

	plans, total, err := e.plugin.planSvc.List(c.Context(), appID, false, false, page, pageSize)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":    "Subscription Plans",
		"plans":    plans,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}

	return e.renderPage(c, "plans_list", data)
}

func (e *DashboardExtension) ServePlanCreatePage(c forge.Context) error {
	data := map[string]interface{}{
		"title":            "Create Plan",
		"billingPatterns":  []string{"flat", "per_seat", "tiered", "usage", "hybrid"},
		"billingIntervals": []string{"monthly", "yearly", "one_time"},
	}

	return e.renderPage(c, "plan_create", data)
}

func (e *DashboardExtension) ServePlanDetailPage(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return e.renderError(c, err)
	}

	plan, err := e.plugin.planSvc.GetByID(c.Context(), id)
	if err != nil {
		return e.renderError(c, err)
	}

	// Get subscription count for this plan
	subs, subCount, _ := e.plugin.subscriptionSvc.List(c.Context(), nil, nil, &id, "", 1, 1)
	_ = subs

	data := map[string]interface{}{
		"title":             "Plan: " + plan.Name,
		"plan":              plan,
		"subscriptionCount": subCount,
	}

	return e.renderPage(c, "plan_detail", data)
}

func (e *DashboardExtension) ServePlanEditPage(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return e.renderError(c, err)
	}

	plan, err := e.plugin.planSvc.GetByID(c.Context(), id)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":            "Edit Plan: " + plan.Name,
		"plan":             plan,
		"billingPatterns":  []string{"flat", "per_seat", "tiered", "usage", "hybrid"},
		"billingIntervals": []string{"monthly", "yearly", "one_time"},
	}

	return e.renderPage(c, "plan_edit", data)
}

func (e *DashboardExtension) ServeSubscriptionsListPage(c forge.Context) error {
	page := queryIntDefault(c, "page", 1)
	pageSize := queryIntDefault(c, "pageSize", 20)
	status := c.Query("status")

	subs, total, err := e.plugin.subscriptionSvc.List(c.Context(), nil, nil, nil, status, page, pageSize)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":         "Subscriptions",
		"subscriptions": subs,
		"total":         total,
		"page":          page,
		"pageSize":      pageSize,
		"statusFilter":  status,
	}

	return e.renderPage(c, "subscriptions_list", data)
}

func (e *DashboardExtension) ServeSubscriptionDetailPage(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return e.renderError(c, err)
	}

	sub, err := e.plugin.subscriptionSvc.GetByID(c.Context(), id)
	if err != nil {
		return e.renderError(c, err)
	}

	// Get usage data
	usageData, _ := e.plugin.usageSvc.GetCurrentPeriodUsage(c.Context(), id)

	// Get invoices
	invoices, _, _ := e.plugin.invoiceSvc.List(c.Context(), nil, &id, "", 1, 10)

	data := map[string]interface{}{
		"title":        "Subscription Details",
		"subscription": sub,
		"usage":        usageData,
		"invoices":     invoices,
	}

	return e.renderPage(c, "subscription_detail", data)
}

func (e *DashboardExtension) ServeAddOnsListPage(c forge.Context) error {
	appID := getAppIDFromContext(c)
	page := queryIntDefault(c, "page", 1)
	pageSize := queryIntDefault(c, "pageSize", 20)

	addons, total, err := e.plugin.addOnSvc.List(c.Context(), appID, false, false, page, pageSize)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":    "Add-ons",
		"addons":   addons,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}

	return e.renderPage(c, "addons_list", data)
}

func (e *DashboardExtension) ServeAddOnCreatePage(c forge.Context) error {
	appID := getAppIDFromContext(c)
	plans, _, _ := e.plugin.planSvc.List(c.Context(), appID, true, false, 1, 100)

	data := map[string]interface{}{
		"title":            "Create Add-on",
		"plans":            plans,
		"billingPatterns":  []string{"flat", "usage"},
		"billingIntervals": []string{"monthly", "yearly", "one_time"},
	}

	return e.renderPage(c, "addon_create", data)
}

func (e *DashboardExtension) ServeAddOnDetailPage(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return e.renderError(c, err)
	}

	addon, err := e.plugin.addOnSvc.GetByID(c.Context(), id)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title": "Add-on: " + addon.Name,
		"addon": addon,
	}

	return e.renderPage(c, "addon_detail", data)
}

func (e *DashboardExtension) ServeInvoicesListPage(c forge.Context) error {
	page := queryIntDefault(c, "page", 1)
	pageSize := queryIntDefault(c, "pageSize", 20)
	status := c.Query("status")

	invoices, total, err := e.plugin.invoiceSvc.List(c.Context(), nil, nil, status, page, pageSize)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":        "Invoices",
		"invoices":     invoices,
		"total":        total,
		"page":         page,
		"pageSize":     pageSize,
		"statusFilter": status,
	}

	return e.renderPage(c, "invoices_list", data)
}

func (e *DashboardExtension) ServeInvoiceDetailPage(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return e.renderError(c, err)
	}

	invoice, err := e.plugin.invoiceSvc.GetByID(c.Context(), id)
	if err != nil {
		return e.renderError(c, err)
	}

	data := map[string]interface{}{
		"title":   "Invoice: " + invoice.Number,
		"invoice": invoice,
	}

	return e.renderPage(c, "invoice_detail", data)
}

func (e *DashboardExtension) ServeUsageDashboardPage(c forge.Context) error {
	data := map[string]interface{}{
		"title": "Usage Dashboard",
	}

	return e.renderPage(c, "usage_dashboard", data)
}

func (e *DashboardExtension) ServeSettingsPage(c forge.Context) error {
	data := map[string]interface{}{
		"title":               "Billing Settings",
		"config":              e.plugin.config,
		"stripeConfigured":    e.plugin.config.IsStripeConfigured(),
		"requireSubscription": e.plugin.config.RequireSubscription,
		"defaultTrialDays":    e.plugin.config.DefaultTrialDays,
		"gracePeriodDays":     e.plugin.config.GracePeriodDays,
	}

	return e.renderPage(c, "settings", data)
}

// Widget Handlers

func (e *DashboardExtension) ServeMRRWidget(c forge.Context) error {
	// Calculate MRR from active subscriptions
	subs, _, _ := e.plugin.subscriptionSvc.List(c.Context(), nil, nil, nil, "active", 1, 1000)

	var mrr int64
	for _, sub := range subs {
		if sub.Plan != nil {
			switch sub.Plan.BillingInterval {
			case "monthly":
				mrr += sub.Plan.BasePrice * int64(sub.Quantity)
			case "yearly":
				mrr += (sub.Plan.BasePrice * int64(sub.Quantity)) / 12
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"value":     mrr,
		"currency":  "USD",
		"formatted": fmt.Sprintf("$%.2f", float64(mrr)/100),
	})
}

func (e *DashboardExtension) ServeActiveSubscriptionsWidget(c forge.Context) error {
	_, total, _ := e.plugin.subscriptionSvc.List(c.Context(), nil, nil, nil, "active", 1, 1)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"value": total,
	})
}

// Helper methods

func (e *DashboardExtension) renderPage(c forge.Context, template string, data map[string]interface{}) error {
	// Check if JSON response is requested
	if c.Header("Accept") == "application/json" || c.Query("format") == "json" {
		return c.JSON(http.StatusOK, data)
	}

	// For HTML, we'd use a template renderer
	// For now, return JSON that the frontend can consume
	return c.JSON(http.StatusOK, data)
}

func (e *DashboardExtension) renderError(c forge.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, map[string]interface{}{
		"error":   true,
		"message": err.Error(),
	})
}
