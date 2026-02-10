package subscription

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// DashboardExtension implements ui.DashboardExtension for the subscription plugin.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui",
	}
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "subscription"
}

// NavigationItems returns the navigation items for the dashboard.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "subscription-billing",
			Label:    "Billing",
			Icon:     lucide.CreditCard(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    50,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/billing"
				}

				return basePath + "/app/" + currentApp.ID.String() + "/billing"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "billing" || activePage == "plans" || activePage == "subscriptions" ||
					activePage == "addons" || activePage == "invoices" || activePage == "usage" ||
					activePage == "coupons" || activePage == "analytics" || activePage == "alerts" ||
					activePage == "features" || activePage == "payment-methods"
			},
			RequiresPlugin: "subscription",
		},
	}
}

// Routes returns the dashboard routes.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Subscription Overview
		{
			Method:       "GET",
			Path:         "/billing",
			Handler:      e.ServeBillingOverviewPage,
			Name:         "subscription.billing.overview",
			Summary:      "Subscription Overview",
			Description:  "View billing overview and summary",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Plans
		{
			Method:       "GET",
			Path:         "/billing/plans",
			Handler:      e.ServePlansListPage,
			Name:         "subscription.plans.list",
			Summary:      "List Plans",
			Description:  "View all subscription plans",
			RequireAuth:  true,
			RequireAdmin: true,
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
			Method:       "POST",
			Path:         "/billing/plans/create",
			Handler:      e.HandleCreatePlan,
			Name:         "subscription.plans.create.submit",
			Summary:      "Submit Create Plan",
			Description:  "Process plan creation form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/plans/:id",
			Handler:      e.ServePlanDetailPage,
			Name:         "subscription.plans.detail",
			Summary:      "Plan Details",
			Description:  "View plan details",
			RequireAuth:  true,
			RequireAdmin: true,
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
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/update",
			Handler:      e.HandleUpdatePlan,
			Name:         "subscription.plans.update",
			Summary:      "Update Plan",
			Description:  "Process plan update form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/archive",
			Handler:      e.HandleArchivePlan,
			Name:         "subscription.plans.archive",
			Summary:      "Archive Plan",
			Description:  "Archive a subscription plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/sync",
			Handler:      e.HandleSyncPlan,
			Name:         "subscription.plans.sync",
			Summary:      "Sync Plan to Provider",
			Description:  "Sync a subscription plan to the payment provider (e.g., Stripe)",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/sync-from-provider",
			Handler:      e.HandleSyncPlanFromProvider,
			Name:         "subscription.plans.sync-from-provider",
			Summary:      "Sync Plan from Provider",
			Description:  "Pull plan data from the payment provider (e.g., Stripe) and update local record",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/sync-all-from-provider",
			Handler:      e.HandleSyncAllPlansFromProvider,
			Name:         "subscription.plans.sync-all-from-provider",
			Summary:      "Sync All Plans from Provider",
			Description:  "Fetch all AuthSome-managed plans from the payment provider and sync to local database",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/delete",
			Handler:      e.HandleDeletePlan,
			Name:         "subscription.plans.delete",
			Summary:      "Delete Plan",
			Description:  "Permanently delete a subscription plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Subscriptions
		{
			Method:       "GET",
			Path:         "/billing/subscriptions",
			Handler:      e.ServeSubscriptionsListPage,
			Name:         "subscription.subscriptions.list",
			Summary:      "List Subscriptions",
			Description:  "View all subscriptions",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/subscriptions/:id",
			Handler:      e.ServeSubscriptionDetailPage,
			Name:         "subscription.subscriptions.detail",
			Summary:      "Subscription Details",
			Description:  "View subscription details",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/subscriptions/:id/cancel",
			Handler:      e.HandleCancelSubscription,
			Name:         "subscription.subscriptions.cancel",
			Summary:      "Cancel Subscription",
			Description:  "Cancel a subscription",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Add-ons
		{
			Method:       "GET",
			Path:         "/billing/addons",
			Handler:      e.ServeAddOnsListPage,
			Name:         "subscription.addons.list",
			Summary:      "List Add-ons",
			Description:  "View all add-ons",
			RequireAuth:  true,
			RequireAdmin: true,
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
			Method:       "POST",
			Path:         "/billing/addons/create",
			Handler:      e.HandleCreateAddOn,
			Name:         "subscription.addons.create.submit",
			Summary:      "Submit Create Add-on",
			Description:  "Process add-on creation form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/addons/:id",
			Handler:      e.ServeAddOnDetailPage,
			Name:         "subscription.addons.detail",
			Summary:      "Add-on Details",
			Description:  "View add-on details",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Invoices
		{
			Method:       "GET",
			Path:         "/billing/invoices",
			Handler:      e.ServeInvoicesListPage,
			Name:         "subscription.invoices.list",
			Summary:      "List Invoices",
			Description:  "View all invoices",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/invoices/:id",
			Handler:      e.ServeInvoiceDetailPage,
			Name:         "subscription.invoices.detail",
			Summary:      "Invoice Details",
			Description:  "View invoice details",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/invoices/sync",
			Handler:      e.HandleSyncInvoices,
			Name:         "subscription.invoices.sync",
			Summary:      "Sync Invoices",
			Description:  "Sync invoices from Stripe",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/invoices/:id/mark-paid",
			Handler:      e.HandleMarkInvoicePaid,
			Name:         "subscription.invoices.mark-paid",
			Summary:      "Mark Invoice Paid",
			Description:  "Mark an invoice as paid",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Payment Methods
		{
			Method:       "GET",
			Path:         "/billing/payment-methods",
			Handler:      e.ServePaymentMethodsPage,
			Name:         "subscription.payment_methods.page",
			Summary:      "Payment Methods",
			Description:  "Manage payment methods",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/payment-methods/add",
			Handler:      e.ServeAddPaymentMethodPage,
			Name:         "subscription.payment_methods.add_page",
			Summary:      "Add Payment Method",
			Description:  "Add a new payment method",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/payment-methods/set-default/:id",
			Handler:      e.HandleSetDefaultPaymentMethod,
			Name:         "subscription.payment_methods.set_default_action",
			Summary:      "Set Default Payment Method",
			Description:  "Set a payment method as default",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "DELETE",
			Path:         "/billing/payment-methods/:id",
			Handler:      e.HandleRemovePaymentMethod,
			Name:         "subscription.payment_methods.remove_action",
			Summary:      "Remove Payment Method",
			Description:  "Remove a payment method",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Usage
		{
			Method:       "GET",
			Path:         "/billing/usage",
			Handler:      e.ServeUsageDashboardPage,
			Name:         "subscription.usage.dashboard",
			Summary:      "Usage Dashboard",
			Description:  "View usage metrics and reports",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Coupons
		{
			Method:       "GET",
			Path:         "/billing/coupons",
			Handler:      e.ServeCouponsListPage,
			Name:         "subscription.coupons.list",
			Summary:      "List Coupons",
			Description:  "View all coupons and discounts",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/coupons/create",
			Handler:      e.ServeCouponCreatePage,
			Name:         "subscription.coupons.create",
			Summary:      "Create Coupon",
			Description:  "Create a new coupon",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/coupons/create",
			Handler:      e.HandleCreateCoupon,
			Name:         "subscription.coupons.create.submit",
			Summary:      "Submit Create Coupon",
			Description:  "Process coupon creation form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Analytics
		{
			Method:       "GET",
			Path:         "/billing/analytics",
			Handler:      e.ServeAnalyticsDashboardPage,
			Name:         "subscription.analytics.dashboard",
			Summary:      "Billing Analytics",
			Description:  "View billing analytics and metrics",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Alerts
		{
			Method:       "GET",
			Path:         "/billing/alerts",
			Handler:      e.ServeAlertsListPage,
			Name:         "subscription.alerts.list",
			Summary:      "Usage Alerts",
			Description:  "View and manage usage alerts",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings
		{
			Method:       "GET",
			Path:         "/settings/billing",
			Handler:      e.ServeSettingsPage,
			Name:         "subscription.settings",
			Summary:      "Billing Settings",
			Description:  "Configure billing settings",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Features
		{
			Method:       "GET",
			Path:         "/billing/features",
			Handler:      e.ServeFeaturesListPage,
			Name:         "subscription.features.list",
			Summary:      "List Features",
			Description:  "View all feature definitions",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/features/create",
			Handler:      e.ServeFeatureCreatePage,
			Name:         "subscription.features.create",
			Summary:      "Create Feature",
			Description:  "Create a new feature",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/create",
			Handler:      e.HandleCreateFeature,
			Name:         "subscription.features.create.submit",
			Summary:      "Submit Create Feature",
			Description:  "Process feature creation form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/features/usage",
			Handler:      e.ServeFeatureUsagePage,
			Name:         "subscription.features.usage",
			Summary:      "Feature Usage",
			Description:  "Monitor feature usage across organizations",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/features/:featureId",
			Handler:      e.ServeFeatureDetailPage,
			Name:         "subscription.features.detail",
			Summary:      "Feature Details",
			Description:  "View feature details",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/features/:featureId/edit",
			Handler:      e.ServeFeatureEditPage,
			Name:         "subscription.features.edit",
			Summary:      "Edit Feature",
			Description:  "Edit an existing feature",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/:featureId/update",
			Handler:      e.HandleUpdateFeature,
			Name:         "subscription.features.update",
			Summary:      "Update Feature",
			Description:  "Process feature update form",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/:featureId/delete",
			Handler:      e.HandleDeleteFeature,
			Name:         "subscription.features.delete",
			Summary:      "Delete Feature",
			Description:  "Permanently delete a feature",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/:featureId/sync",
			Handler:      e.HandleSyncFeature,
			Name:         "subscription.features.sync",
			Summary:      "Sync Feature to Provider",
			Description:  "Manually sync a feature to the payment provider",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/:featureId/sync-from-provider",
			Handler:      e.HandleSyncFeatureFromProvider,
			Name:         "subscription.features.sync-from-provider",
			Summary:      "Sync Feature from Provider",
			Description:  "Pull feature data from the payment provider",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/features/sync-all-from-provider",
			Handler:      e.HandleSyncAllFeaturesFromProvider,
			Name:         "subscription.features.sync-all-from-provider",
			Summary:      "Sync All Features from Provider",
			Description:  "Fetch all features from the payment provider and sync locally",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Export/Import Features and Plans
		{
			Method:       "GET",
			Path:         "/billing/export",
			Handler:      e.HandleExportFeaturesAndPlans,
			Name:         "subscription.export",
			Summary:      "Export Features and Plans",
			Description:  "Export all features and plans as JSON for backup or migration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/billing/import",
			Handler:      e.HandleShowImportForm,
			Name:         "subscription.import.form",
			Summary:      "Show Import Form",
			Description:  "Display the import form for features and plans",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/import",
			Handler:      e.HandleImportFeaturesAndPlans,
			Name:         "subscription.import",
			Summary:      "Import Features and Plans",
			Description:  "Import features and plans from JSON",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Plan Features Management
		{
			Method:       "GET",
			Path:         "/billing/plans/:id/features",
			Handler:      e.ServePlanFeaturesPage,
			Name:         "subscription.plans.features",
			Summary:      "Plan Features",
			Description:  "Manage features linked to a plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/features/link",
			Handler:      e.HandleLinkFeatureToPlan,
			Name:         "subscription.plans.features.link",
			Summary:      "Link Feature to Plan",
			Description:  "Link a feature to a plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/features/:featureId/update",
			Handler:      e.HandleUpdatePlanFeatureLink,
			Name:         "subscription.plans.features.update",
			Summary:      "Update Plan Feature Link",
			Description:  "Update feature configuration for a plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/billing/plans/:id/features/:featureId/unlink",
			Handler:      e.HandleUnlinkFeatureFromPlan,
			Name:         "subscription.plans.features.unlink",
			Summary:      "Unlink Feature from Plan",
			Description:  "Remove a feature from a plan",
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead).
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages for the plugin.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "subscription-settings",
			Label:         "Subscription",
			Description:   "Configure subscription and billing settings",
			Icon:          lucide.CreditCard(Class("h-5 w-5")),
			Category:      "general",
			Order:         30,
			Path:          "billing",
			RequirePlugin: "subscription",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns dashboard widgets.
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "subscription-mrr",
			Title: "Monthly Revenue",
			Icon:  lucide.DollarSign(Class("size-5")),
			Order: 10,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderMRRWidget(currentApp)
			},
		},
		{
			ID:    "subscription-active-count",
			Title: "Active Subscriptions",
			Icon:  lucide.Users(Class("size-5")),
			Order: 11,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderActiveSubscriptionsWidget(currentApp)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for the subscription plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	// No bridge functions for this plugin yet
	return nil
}

// Helper methods

// getUserFromContext extracts the current user from the request context.
func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) *user.User {
	reqCtx := ctx.Request.Context()
	if u, ok := reqCtx.Value("user").(*user.User); ok {
		return u
	}

	return nil
}

// extractAppFromURL extracts the app from the URL parameter.
func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.New(errs.CodeInvalidInput, "app ID is required", http.StatusBadRequest)
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	// Return minimal app with ID - the dashboard handler will enrich it
	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path.
func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// queryIntDefault gets an integer query parameter with a default value.
func queryIntDefault(ctx *router.PageContext, name string, defaultValue int) int {
	str := ctx.QueryDefault(name, "")
	if str == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}

	return val
}

// formatMoney formats a price in cents to a human readable string.
func formatMoney(cents int64, currency string) string {
	if currency == "" {
		currency = "USD"
	}

	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

// formatPercent formats a decimal as percentage.
func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}

// Widget renderers

func (e *DashboardExtension) renderMRRWidget(currentApp *app.App) g.Node {
	ctx := context.Background()
	subs, _, _ := e.plugin.subscriptionSvc.List(ctx, nil, nil, nil, "active", 1, 1000)

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

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text(formatMoney(mrr, "USD")),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Monthly Recurring"),
		),
	)
}

func (e *DashboardExtension) renderActiveSubscriptionsWidget(currentApp *app.App) g.Node {
	ctx := context.Background()
	_, total, _ := e.plugin.subscriptionSvc.List(ctx, nil, nil, nil, "active", 1, 1)

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text(strconv.Itoa(total)),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Active Subscriptions"),
		),
	)
}

// Common UI components

func (e *DashboardExtension) statsCard(title, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("mt-1 text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
			),
			Div(
				Class("rounded-full bg-violet-100 p-3 dark:bg-violet-900/30"),
				icon,
			),
		),
	)
}

func (e *DashboardExtension) statusBadge(status string) g.Node {
	var classes string

	switch strings.ToLower(status) {
	case "active", "paid", "success":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	case "trialing", "pending", "draft":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
	case "canceled", "cancelled", "failed", "void":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
	case "past_due", "overdue":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400"
	default:
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}

	return Span(Class(classes), g.Text(status))
}

func (e *DashboardExtension) renderPagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	items := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Previous"),
		))
	}

	// Page numbers (simplified - show current and nearby)
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 border border-violet-600 rounded-md"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(
				Class("px-2 py-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Next"),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(items),
	)
}

// Billing sub-navigation.
func (e *DashboardExtension) renderBillingNav(currentApp *app.App, basePath, activePage string) g.Node {
	type navItem struct {
		label string
		path  string
		page  string
		icon  g.Node
	}

	items := []navItem{
		{"Overview", "/billing", "billing", lucide.LayoutDashboard(Class("size-4"))},
		{"Plans", "/billing/plans", "plans", lucide.Package(Class("size-4"))},
		{"Features", "/billing/features", "features", lucide.Settings(Class("size-4"))},
		{"Subscriptions", "/billing/subscriptions", "subscriptions", lucide.Users(Class("size-4"))},
		{"Add-ons", "/billing/addons", "addons", lucide.Puzzle(Class("size-4"))},
		{"Invoices", "/billing/invoices", "invoices", lucide.FileText(Class("size-4"))},
		{"Coupons", "/billing/coupons", "coupons", lucide.Tag(Class("size-4"))},
		{"Usage", "/billing/usage", "usage", lucide.Activity(Class("size-4"))},
		{"Analytics", "/billing/analytics", "analytics", lucide.TrendingUp(Class("size-4"))},
	}

	navItems := make([]g.Node, 0, len(items))
	for _, item := range items {
		isActive := activePage == item.page

		classes := "inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors "
		if isActive {
			classes += "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
		} else {
			classes += "text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800"
		}

		navItems = append(navItems, A(
			Href(basePath+"/app/"+currentApp.ID.String()+item.path),
			Class(classes),
			item.icon,
			g.Text(item.label),
		))
	}

	return Nav(
		Class("flex flex-wrap gap-2 mb-6 p-2 bg-slate-50 dark:bg-gray-800/50 rounded-lg"),
		g.Group(navItems),
	)
}

// Type conversion helpers for display

func (e *DashboardExtension) planStatusBadge(plan *core.Plan) g.Node {
	if plan.IsActive {
		return nil
	}

	return nil
}

// planSyncStatusBadge returns a badge indicating whether the plan is synced to the payment provider.
func (e *DashboardExtension) planSyncStatusBadge(plan *core.Plan) g.Node {
	if plan.ProviderPlanID != "" && plan.ProviderPriceID != "" {
		return Span(
			Class("inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"),
			lucide.Cloud(Class("size-3")),
			g.Text("Synced"),
		)
	}

	return Span(
		Class("inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 dark:bg-gray-700 dark:text-gray-400"),
		lucide.CloudOff(Class("size-3")),
		g.Text("Not Synced"),
	)
}

func (e *DashboardExtension) subscriptionStatusBadge(sub *core.Subscription) g.Node {
	return e.statusBadge(string(sub.Status))
}

func (e *DashboardExtension) invoiceStatusBadge(inv *core.Invoice) g.Node {
	return e.statusBadge(string(inv.Status))
}

// ServeSettingsPage renders the subscription settings page.
func (e *DashboardExtension) ServeSettingsPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	content := Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("mb-6"),
			H1(Class("text-2xl font-bold"), g.Text("Subscription Settings")),
			P(Class("text-sm text-muted-foreground"), g.Text("Configure subscription and billing behavior for your application")),
		),

		// Settings form
		Div(
			Class("rounded-lg border bg-card p-6"),
			Form(
				Method("POST"),
				Action(basePath+"/app/"+currentApp.ID.String()+"/settings/billing/update"),
				Class("space-y-6"),

				// Require subscription
				Div(
					Class("flex items-center justify-between py-3 border-b"),
					Div(
						Class("space-y-1"),
						Label(
							For("require_subscription"),
							Class("text-sm font-medium"),
							g.Text("Require subscription for organizations"),
						),
						P(Class("text-xs text-muted-foreground"),
							g.Text("Organizations must have an active subscription to access features")),
					),
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Name("require_subscription"),
							ID("require_subscription"),
							Class("sr-only peer"),
							g.If(e.plugin.config.RequireSubscription, g.Attr("checked", "")),
						),
						Div(
							Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary"),
						),
					),
				),

				// Default trial days
				Div(
					Class("space-y-2 py-3 border-b"),
					Label(
						For("trial_days"),
						Class("text-sm font-medium"),
						g.Text("Default Trial Days"),
					),
					Div(
						Class("flex items-center gap-3"),
						Input(
							Type("number"),
							Name("trial_days"),
							ID("trial_days"),
							Value(strconv.Itoa(e.plugin.config.DefaultTrialDays)),
							Min("0"),
							Max("90"),
							Class("flex h-10 w-[120px] rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
						),
						Span(Class("text-sm text-muted-foreground"), g.Text("days")),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("Number of days for trial period on new subscriptions")),
				),

				// Grace period days
				Div(
					Class("space-y-2 py-3 border-b"),
					Label(
						For("grace_days"),
						Class("text-sm font-medium"),
						g.Text("Grace Period Days"),
					),
					Div(
						Class("flex items-center gap-3"),
						Input(
							Type("number"),
							Name("grace_days"),
							ID("grace_days"),
							Value(strconv.Itoa(e.plugin.config.GracePeriodDays)),
							Min("0"),
							Max("30"),
							Class("flex h-10 w-[120px] rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
						),
						Span(Class("text-sm text-muted-foreground"), g.Text("days")),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("Days after payment failure before subscription is canceled")),
				),

				// Stripe configuration status
				Div(
					Class("py-3"),
					func() g.Node {
						if e.plugin.config.IsStripeConfigured() {
							return Div(
								Class("rounded-lg border border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-900/20 p-4"),
								Div(
									Class("flex items-center gap-3"),
									lucide.CircleCheck(Class("h-5 w-5 text-emerald-600 dark:text-emerald-400")),
									Div(
										Div(Class("font-medium"), g.Text("Stripe Integration")),
										Div(Class("text-sm text-muted-foreground"),
											g.Text("Stripe is configured and ready to process payments")),
									),
								),
							)
						}

						return Div(
							Class("rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20 p-4"),
							Div(
								Class("flex items-center gap-3"),
								lucide.CircleAlert(Class("h-5 w-5 text-amber-600 dark:text-amber-400")),
								Div(
									Div(Class("font-medium"), g.Text("Stripe Integration")),
									Div(Class("text-sm text-muted-foreground"),
										g.Text("Configure Stripe API keys in environment variables")),
								),
							),
						)
					}(),
				),

				// Submit button
				Div(
					Class("flex justify-end pt-4"),
					Button(
						Type("submit"),
						Class("inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"),
						lucide.Save(Class("size-4 mr-2")),
						g.Text("Save Settings"),
					),
				),
			),
		),
	)

	return content, nil
}
