package subscription

import (
	"context"
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/internal/errs"
	core "github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServeBillingOverviewPage renders the billing overview dashboard.
func (e *DashboardExtension) ServeBillingOverviewPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Get summary data
	plans, planCount, _ := e.plugin.planSvc.List(reqCtx, currentApp.ID, false, false, 1, 100)
	subs, subCount, _ := e.plugin.subscriptionSvc.List(reqCtx, nil, nil, nil, "", 1, 100)

	// Count by status
	var (
		activeCount, trialingCount, canceledCount int
		mrr                                       int64
	)

	for _, sub := range subs {
		switch sub.Status {
		case "active":
			activeCount++

			if sub.Plan != nil {
				switch sub.Plan.BillingInterval {
				case "monthly":
					mrr += sub.Plan.BasePrice * int64(sub.Quantity)
				case "yearly":
					mrr += (sub.Plan.BasePrice * int64(sub.Quantity)) / 12
				}
			}
		case "trialing":
			trialingCount++
		case "canceled", "cancelled":
			canceledCount++
		}
	}

	// Calculate ARR
	arr := mrr * 12

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Subscription Overview")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Monitor your subscription business metrics")),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "billing"),

		// Stats cards
		Div(
			Class("grid gap-6 md:grid-cols-2 lg:grid-cols-4"),
			e.statsCard("Monthly Recurring Revenue", formatMoney(mrr, "USD"), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Annual Recurring Revenue", formatMoney(arr, "USD"), lucide.TrendingUp(Class("size-5 text-violet-600"))),
			e.statsCard("Active Subscriptions", fmt.Sprintf("%d", activeCount), lucide.Users(Class("size-5 text-violet-600"))),
			e.statsCard("Trial Subscriptions", fmt.Sprintf("%d", trialingCount), lucide.Clock(Class("size-5 text-violet-600"))),
		),

		// Quick stats row
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Total Plans", fmt.Sprintf("%d", planCount), lucide.Package(Class("size-5 text-violet-600"))),
			e.statsCard("Total Subscriptions", fmt.Sprintf("%d", subCount), lucide.FileText(Class("size-5 text-violet-600"))),
			e.statsCard("Canceled", fmt.Sprintf("%d", canceledCount), lucide.X(Class("size-5 text-violet-600"))),
		),

		// Recent subscriptions
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Recent Subscriptions")),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions"),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
					g.Text("View all →"),
				),
			),
			Div(
				Class("px-6 py-4"),
				g.If(len(subs) == 0,
					Div(
						Class("text-center py-8 text-slate-500 dark:text-gray-400"),
						lucide.Inbox(Class("mx-auto h-12 w-12 mb-3")),
						P(g.Text("No subscriptions yet")),
					),
				),
				g.If(len(subs) > 0,
					e.renderRecentSubscriptionsTable(reqCtx, subs[:min(5, len(subs))], currentApp, basePath),
				),
			),
		),

		// Plans overview
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Plans Overview")),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
					g.Text("Manage plans →"),
				),
			),
			Div(
				Class("px-6 py-4"),
				g.If(len(plans) == 0,
					Div(
						Class("text-center py-8 text-slate-500 dark:text-gray-400"),
						lucide.Package(Class("mx-auto h-12 w-12 mb-3")),
						P(g.Text("No plans created yet")),
						A(
							Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/create"),
							Class("mt-4 inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
							lucide.Plus(Class("size-4")),
							g.Text("Create your first plan"),
						),
					),
				),
				g.If(len(plans) > 0,
					e.renderPlansOverviewGrid(plans[:min(4, len(plans))], currentApp, basePath),
				),
			),
		),

		// Payment Methods Widget
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Payment Methods")),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/payment-methods"),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
					g.Text("Manage all →"),
				),
			),
			Div(
				Class("px-6 py-4"),
				e.renderPaymentMethodsOverviewWidget(reqCtx, currentApp, basePath),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServePlansListPage renders the plans list page.
func (e *DashboardExtension) ServePlansListPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)

	plans, total, _ := e.plugin.planSvc.List(reqCtx, currentApp.ID, false, false, page, pageSize)
	totalPages := int((int64(total) + int64(pageSize) - 1) / int64(pageSize))

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Subscription Plans")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage your pricing plans and features")),
			),
			Div(
				Class("flex items-center gap-3"),
				// Sync from provider button
				Form(
					Method("POST"),
					Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/sync-all-from-provider"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
						lucide.CloudDownload(Class("size-4")),
						g.Text("Sync from Stripe"),
					),
				),
				// Create plan button
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/create"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					lucide.Plus(Class("size-4")),
					g.Text("Create Plan"),
				),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "plans"),

		// Stats row
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Total Plans", fmt.Sprintf("%d", total), lucide.Package(Class("size-5 text-violet-600"))),
			e.statsCard("Active Plans", fmt.Sprintf("%d", countActivePlans(plans)), lucide.Check(Class("size-5 text-violet-600"))),
			e.statsCard("Public Plans", fmt.Sprintf("%d", countPublicPlans(plans)), lucide.Globe(Class("size-5 text-violet-600"))),
		),

		// Plans table
		e.renderPlansTable(reqCtx, plans, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeSubscriptionsListPage renders the subscriptions list page.
func (e *DashboardExtension) ServeSubscriptionsListPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)
	statusFilter := ctx.Request.URL.Query().Get("status")

	subs, total, _ := e.plugin.subscriptionSvc.List(reqCtx, nil, nil, nil, statusFilter, page, pageSize)
	totalPages := int((int64(total) + int64(pageSize) - 1) / int64(pageSize))

	// Count by status for filters
	allSubs, _, _ := e.plugin.subscriptionSvc.List(reqCtx, nil, nil, nil, "", 1, 10000)

	statusCounts := make(map[string]int)
	for _, sub := range allSubs {
		statusCounts[string(sub.Status)]++
	}

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("Subscriptions")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("View and manage all customer subscriptions")),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "subscriptions"),

		// Status filter tabs
		e.renderStatusFilterTabs(statusFilter, statusCounts, currentApp, basePath, "/billing/subscriptions"),

		// Subscriptions table
		e.renderSubscriptionsTable(reqCtx, subs, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAddOnsListPage renders the add-ons list page.
func (e *DashboardExtension) ServeAddOnsListPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)

	addons, total, _ := e.plugin.addOnSvc.List(reqCtx, currentApp.ID, false, false, page, pageSize)
	totalPages := int((int64(total) + int64(pageSize) - 1) / int64(pageSize))

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Add-ons")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage additional features and products")),
			),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Add-on"),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "addons"),

		// Add-ons table
		e.renderAddOnsTable(reqCtx, addons, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/addons"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeInvoicesListPage renders the invoices list page.
func (e *DashboardExtension) ServeInvoicesListPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)
	statusFilter := ctx.Request.URL.Query().Get("status")

	// Check for success/error messages
	successMsg := ctx.Request.URL.Query().Get("success")
	errorMsg := ctx.Request.URL.Query().Get("error")
	syncCount := ctx.Request.URL.Query().Get("count")

	invoices, total, _ := e.plugin.invoiceSvc.List(reqCtx, nil, nil, statusFilter, page, pageSize)
	totalPages := int((int64(total) + int64(pageSize) - 1) / int64(pageSize))

	// Count by status
	allInvoices, _, _ := e.plugin.invoiceSvc.List(reqCtx, nil, nil, "", 1, 10000)
	statusCounts := make(map[string]int)

	var totalRevenue int64

	for _, inv := range allInvoices {
		statusCounts[string(inv.Status)]++
		if inv.Status == core.InvoiceStatusPaid {
			totalRevenue += inv.Total
		}
	}

	content := Div(
		Class("space-y-2"),

		// Success/Error alerts
		g.If(successMsg == "synced",
			Div(
				Class("rounded-lg bg-green-50 p-4 text-sm text-green-800 dark:bg-green-900/20 dark:text-green-300"),
				g.Textf("✅ Successfully synced %s invoices from Stripe!", syncCount),
			),
		),
		g.If(errorMsg == "sync_failed",
			Div(
				Class("rounded-lg bg-red-50 p-4 text-sm text-red-800 dark:bg-red-900/20 dark:text-red-300"),
				g.Text("❌ Failed to sync invoices from Stripe. Please try again."),
			),
		),

		// Page header with sync button
		Div(
			Class("flex items-start justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Invoices")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("View and manage billing invoices")),
			),
			Form(
				Method("POST"),
				Action(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices/sync"),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-2 px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 transition-colors"),
					lucide.RefreshCw(Class("size-4")),
					g.Text("Sync from Stripe"),
				),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "invoices"),

		// Stats row
		Div(
			Class("grid gap-6 md:grid-cols-4"),
			e.statsCard("Total Revenue", formatMoney(totalRevenue, "USD"), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Paid Invoices", fmt.Sprintf("%d", statusCounts["paid"]), lucide.CircleCheck(Class("size-5 text-violet-600"))),
			e.statsCard("Pending", fmt.Sprintf("%d", statusCounts["open"]+statusCounts["draft"]), lucide.Clock(Class("size-5 text-violet-600"))),
			e.statsCard("Overdue", fmt.Sprintf("%d", statusCounts["past_due"]), lucide.CircleAlert(Class("size-5 text-violet-600"))),
		),

		// Status filter tabs
		e.renderStatusFilterTabs(statusFilter, statusCounts, currentApp, basePath, "/billing/invoices"),

		// Invoices table
		e.renderInvoicesTable(reqCtx, invoices, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/invoices"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeUsageDashboardPage renders the usage dashboard page.
func (e *DashboardExtension) ServeUsageDashboardPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Parse date range filter
	dateRange := ctx.Request.URL.Query().Get("range")
	if dateRange == "" {
		dateRange = "30d"
	}

	startDate, endDate := calculateDateRange(dateRange)

	// Get real usage data
	currentUsage, _ := e.plugin.featureUsageRepo.GetCurrentUsageSnapshot(reqCtx, currentApp.ID)
	orgStats, _ := e.plugin.featureUsageRepo.GetUsageByOrg(reqCtx, currentApp.ID, startDate, endDate)
	typeStats, _ := e.plugin.featureUsageRepo.GetUsageByFeatureType(reqCtx, currentApp.ID, startDate, endDate)
	topConsumers, _ := e.plugin.featureUsageRepo.GetTopConsumers(reqCtx, currentApp.ID, nil, startDate, endDate, 10)

	// Current path for filters
	currentPath := basePath + "/app/" + currentApp.ID.String() + "/billing/usage"

	content := Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Usage Dashboard")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Monitor feature usage across all subscriptions")),
			),
			// Date range filter
			renderDateRangeFilter(currentPath, dateRange),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "usage"),

		// Current usage cards
		g.If(len(currentUsage) > 0,
			Div(
				H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Current Usage")),
				Div(
					Class("grid gap-6 md:grid-cols-2 lg:grid-cols-3"),
					g.Group(g.Map(currentUsage, func(u *core.CurrentUsage) g.Node {
						icon := lucide.Activity(Class("size-6 text-violet-600"))
						// Pick icon based on feature type
						switch u.FeatureType {
						case "metered":
							icon = lucide.Zap(Class("size-6 text-violet-600"))
						case "limit":
							icon = lucide.Users(Class("size-6 text-violet-600"))
						default:
							icon = lucide.Activity(Class("size-6 text-violet-600"))
						}

						used := fmt.Sprintf("%s %s", formatNumber(u.CurrentUsage), u.Unit)

						limit := formatLimit(u.Limit)
						if u.Limit > 0 {
							limit = fmt.Sprintf("%s %s", formatNumber(u.Limit), u.Unit)
						}

						return e.renderUsageCard(u.FeatureName, used, limit, u.PercentUsed, icon)
					})),
				),
			),
		),

		// Empty state
		g.If(len(currentUsage) == 0,
			renderEmptyState(
				lucide.Activity(Class("mx-auto h-16 w-16 text-slate-300 dark:text-gray-600")),
				"No Usage Data",
				"Start using features to see usage statistics appear here",
			),
		),

		// Usage by organization table
		g.If(len(orgStats) > 0,
			Div(
				Class("mt-6"),
				H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Usage by Organization")),
				Div(
					Class("overflow-x-auto rounded-lg border border-slate-200 dark:border-gray-800"),
					Table(
						Class("w-full"),
						THead(
							Class("bg-slate-50 dark:bg-gray-800"),
							Tr(
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Organization")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Feature")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Usage")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Limit")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("% Used")),
							),
						),
						TBody(
							Class("bg-white divide-y divide-slate-200 dark:bg-gray-900 dark:divide-gray-800"),
							g.Group(g.Map(orgStats, func(stat *core.OrgUsageStats) g.Node {
								return Tr(
									Td(Class("px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white"),
										g.Text(stat.OrgName)),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-gray-400"),
										g.Text(stat.FeatureName)),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-900 dark:text-white"),
										g.Text(fmt.Sprintf("%s %s", formatNumber(stat.Usage), stat.Unit))),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-gray-400"),
										g.Text(formatLimit(stat.Limit))),
									Td(Class("px-6 py-4 whitespace-nowrap"),
										renderProgressBar(stat.PercentUsed)),
								)
							})),
						),
					),
				),
			),
		),

		// Usage by feature type and top consumers
		g.If(len(typeStats) > 0 || len(topConsumers) > 0,
			Div(
				Class("mt-6 grid gap-6 lg:grid-cols-2"),
				// Feature type breakdown
				g.If(len(typeStats) > 0,
					Div(
						Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
							g.Text("Usage by Feature Type")),
						g.Group(func() []g.Node {
							nodes := make([]g.Node, 0, len(typeStats))
							for featureType, stats := range typeStats {
								ft := featureType
								st := stats
								nodes = append(nodes, Div(
									Class("flex items-center justify-between py-3 border-b border-slate-100 dark:border-gray-800 last:border-0"),
									Div(
										Span(Class("text-sm font-medium text-slate-900 dark:text-white capitalize"),
											g.Text(string(ft))),
										Span(Class("text-xs text-slate-500 dark:text-gray-400 ml-2"),
											g.Text(fmt.Sprintf("(%d orgs)", st.TotalOrgs))),
									),
									Span(Class("text-sm font-semibold text-slate-900 dark:text-white"),
										g.Text(formatNumber(st.TotalUsage))),
								))
							}

							return nodes
						}()),
					),
				),
				// Top consumers
				g.If(len(topConsumers) > 0,
					Div(
						Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
							g.Text("Top 10 Consumers")),
						g.Group(g.Map(topConsumers, func(consumer *core.OrgUsageStats) g.Node {
							return Div(
								Class("flex items-center justify-between py-3 border-b border-slate-100 dark:border-gray-800 last:border-0"),
								Div(
									Div(Class("text-sm font-medium text-slate-900 dark:text-white"),
										g.Text(consumer.OrgName)),
									Div(Class("text-xs text-slate-500 dark:text-gray-400"),
										g.Text(consumer.FeatureName)),
								),
								Span(Class("text-sm font-semibold text-slate-900 dark:text-white"),
									g.Text(fmt.Sprintf("%s %s", formatNumber(consumer.Usage), consumer.Unit))),
							)
						})),
					),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeCouponsListPage renders the coupons list page.
func (e *DashboardExtension) ServeCouponsListPage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)

	// TODO: Add couponSvc to Plugin - for now return empty list
	var (
		coupons []*core.Coupon
		total   int64 = 0
	)

	_ = reqCtx // suppress unused warning
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Coupons & Discounts")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage promotional codes and discounts")),
			),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Coupon"),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "coupons"),

		// Coupons table
		e.renderCouponsTable(reqCtx, coupons, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/coupons"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAnalyticsDashboardPage renders the analytics dashboard.
func (e *DashboardExtension) ServeAnalyticsDashboardPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Parse date range filter
	dateRange := ctx.Request.URL.Query().Get("range")
	if dateRange == "" {
		dateRange = "30d"
	}

	startDate, endDate := calculateDateRange(dateRange)
	currentPath := basePath + "/app/" + currentApp.ID.String() + "/billing/analytics"

	// Get real analytics data
	metrics, _ := e.plugin.analyticsSvc.GetDashboardMetrics(reqCtx, currentApp.ID, startDate, endDate, "USD")
	if metrics == nil {
		metrics = &core.DashboardMetrics{
			Currency: "USD",
		}
	}

	// Get MRR history for chart
	mrrHistory, _ := e.plugin.analyticsSvc.GetMRRHistory(reqCtx, currentApp.ID, startDate, endDate, "USD")

	// Get per-org revenue breakdown
	orgRevenue, _ := e.plugin.analyticsSvc.GetRevenueByOrg(reqCtx, currentApp.ID, startDate, endDate)

	content := Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Billing Analytics")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Insights into your subscription business")),
			),
			// Date range filter
			renderDateRangeFilter(currentPath, dateRange),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "analytics"),

		// Key metrics
		Div(
			Class("grid gap-6 md:grid-cols-2 lg:grid-cols-4"),
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						P(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text("MRR")),
						P(Class("text-2xl font-bold text-slate-900 dark:text-white mt-2"), g.Text(formatMoney(metrics.TotalMRR, "USD"))),
						renderMetricChange(metrics.MRRGrowth),
					),
					lucide.DollarSign(Class("size-6 text-violet-600")),
				),
			),
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						P(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text("ARR")),
						P(Class("text-2xl font-bold text-slate-900 dark:text-white mt-2"), g.Text(formatMoney(metrics.TotalARR, "USD"))),
					),
					lucide.TrendingUp(Class("size-6 text-violet-600")),
				),
			),
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						P(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text("Active Customers")),
						P(Class("text-2xl font-bold text-slate-900 dark:text-white mt-2"), g.Text(fmt.Sprintf("%d", metrics.ActiveSubscriptions))),
						renderMetricChange(metrics.SubscriptionGrowth),
					),
					lucide.Users(Class("size-6 text-violet-600")),
				),
			),
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						P(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text("Churn Rate")),
						P(Class("text-2xl font-bold text-slate-900 dark:text-white mt-2"), g.Text(fmt.Sprintf("%.1f%%", metrics.ChurnRate))),
						g.If(metrics.ChurnRate > 0, renderMetricChange(-metrics.ChurnRate)),
					),
					lucide.UserMinus(Class("size-6 text-red-600")),
				),
			),
		),

		// Charts row - MRR History
		g.If(len(mrrHistory) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("MRR Over Time")),
				Div(
					Class("space-y-2"),
					g.Group(g.Map(mrrHistory, func(breakdown *core.MRRBreakdown) g.Node {
						return Div(
							Class("flex items-center justify-between py-2 border-b border-slate-100 dark:border-gray-800 last:border-0"),
							Span(Class("text-sm text-slate-600 dark:text-gray-400"),
								g.Text(breakdown.Date.Format("Jan 02"))),
							Div(
								Class("flex items-center gap-4"),
								Span(Class("text-sm font-semibold text-slate-900 dark:text-white"),
									g.Text(formatMoney(breakdown.TotalMRR, "USD"))),
								g.If(breakdown.NetNewMRR != 0,
									Span(
										Class(fmt.Sprintf("text-xs %s",
											func() string {
												if breakdown.NetNewMRR > 0 {
													return "text-green-600 dark:text-green-400"
												}

												return "text-red-600 dark:text-red-400"
											}(),
										)),
										g.Text(formatMoney(breakdown.NetNewMRR, "USD")),
									),
								),
							),
						)
					})),
				),
			),
		),

		// Revenue by organization table
		g.If(len(orgRevenue) > 0,
			Div(
				Class("mt-6"),
				H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Revenue by Organization")),
				Div(
					Class("overflow-x-auto rounded-lg border border-slate-200 dark:border-gray-800"),
					Table(
						Class("w-full"),
						THead(
							Class("bg-slate-50 dark:bg-gray-800"),
							Tr(
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Organization")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("MRR")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("ARR")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Plan")),
								Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
									g.Text("Status")),
							),
						),
						TBody(
							Class("bg-white divide-y divide-slate-200 dark:bg-gray-900 dark:divide-gray-800"),
							g.Group(g.Map(orgRevenue, func(rev *core.OrgRevenue) g.Node {
								return Tr(
									Td(Class("px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white"),
										g.Text(rev.OrgName)),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-900 dark:text-white"),
										g.Text(formatMoney(rev.MRR, "USD"))),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-gray-400"),
										g.Text(formatMoney(rev.ARR, "USD"))),
									Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-gray-400"),
										g.Text(rev.PlanName)),
									Td(Class("px-6 py-4 whitespace-nowrap"),
										renderStatusBadge(rev.Status)),
								)
							})),
						),
					),
				),
			),
		),

		// Trial conversion metrics
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("Subscription Health")),
			Div(
				Class("grid gap-6 md:grid-cols-3"),
				e.statsCard("Trialing", fmt.Sprintf("%d", metrics.TrialingSubscriptions), lucide.UserPlus(Class("size-5 text-blue-600"))),
				e.statsCard("New MRR", formatMoney(metrics.NewMRR, "USD"), lucide.TrendingUp(Class("size-5 text-green-600"))),
				e.statsCard("Churned MRR", formatMoney(metrics.ChurnedMRR, "USD"), lucide.TrendingDown(Class("size-5 text-red-600"))),
			),
		),

		// Empty state
		g.If(len(orgRevenue) == 0 && metrics.ActiveSubscriptions == 0,
			renderEmptyState(
				lucide.TrendingUp(Class("mx-auto h-16 w-16 text-slate-300 dark:text-gray-600")),
				"No Analytics Data",
				"Start creating subscriptions to see analytics appear here",
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAlertsListPage renders the alerts list page.
func (e *DashboardExtension) ServeAlertsListPage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	content := Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Usage Alerts")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Configure notifications for usage thresholds")),
			),
			Button(
				Type("button"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Alert"),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "alerts"),

		// Alert configuration cards
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("text-center py-12"),
				lucide.Bell(Class("mx-auto h-16 w-16 text-slate-300 dark:text-gray-600 mb-4")),
				H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"),
					g.Text("Usage Alerts")),
				P(Class("text-slate-500 dark:text-gray-400 max-w-md mx-auto mb-4"),
					g.Text("Set up alerts to notify customers when they approach their usage limits.")),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					lucide.Plus(Class("size-4")),
					g.Text("Configure Alert"),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// Helper functions for counting plans.
func countActivePlans(plans []*core.Plan) int {
	count := 0

	for _, p := range plans {
		if p.IsActive {
			count++
		}
	}

	return count
}

func countPublicPlans(plans []*core.Plan) int {
	count := 0

	for _, p := range plans {
		if p.IsPublic {
			count++
		}
	}

	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// Table rendering helpers

func (e *DashboardExtension) renderRecentSubscriptionsTable(ctx context.Context, subs []*core.Subscription, currentApp *app.App, basePath string) g.Node {
	rows := make([]g.Node, 0, len(subs))
	for _, sub := range subs {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-4 py-3 text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(sub.ID.String()[:8]+"...")),
			Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
				g.If(sub.Plan != nil, g.Text(sub.Plan.Name)),
				g.If(sub.Plan == nil, g.Text("-")),
			),
			Td(Class("px-4 py-3"), e.subscriptionStatusBadge(sub)),
			Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(sub.CreatedAt.Format("Jan 2, 2006"))),
			Td(Class("px-4 py-3 text-right"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions/"+sub.ID.String()),
					Class("text-violet-600 hover:text-violet-700 text-sm"),
					g.Text("View"),
				)),
		))
	}

	return Table(
		Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
		THead(
			Class("bg-slate-50 dark:bg-gray-800"),
			Tr(
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase"), g.Text("ID")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase"), g.Text("Plan")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase"), g.Text("Status")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase"), g.Text("Created")),
				Th(Class("px-4 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase"), g.Text("Actions")),
			),
		),
		TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
	)
}

func (e *DashboardExtension) renderPlansOverviewGrid(plans []*core.Plan, currentApp *app.App, basePath string) g.Node {
	cards := make([]g.Node, 0, len(plans))
	for _, plan := range plans {
		cards = append(cards, Div(
			Class("rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-gray-700 dark:bg-gray-800"),
			Div(
				Class("flex items-center justify-between mb-2"),
				H4(Class("font-medium text-slate-900 dark:text-white"), g.Text(plan.Name)),
				e.planStatusBadge(plan),
			),
			Div(
				Class("text-2xl font-bold text-violet-600 dark:text-violet-400 mb-1"),
				g.Text(formatMoney(plan.BasePrice, plan.Currency)),
				Span(Class("text-sm font-normal text-slate-500 dark:text-gray-400"),
					g.Text("/"+string(plan.BillingInterval))),
			),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
				Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
				g.Text("View details →"),
			),
		))
	}

	return Div(Class("grid gap-4 md:grid-cols-2 lg:grid-cols-4"), g.Group(cards))
}

func (e *DashboardExtension) renderPlansTable(ctx context.Context, plans []*core.Plan, currentApp *app.App, basePath string) g.Node {
	if len(plans) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
			lucide.Package(Class("mx-auto h-12 w-12 text-slate-300 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"), g.Text("No plans yet")),
			P(Class("text-slate-500 dark:text-gray-400 mb-4"), g.Text("Create your first subscription plan to start accepting payments")),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Plan"),
			),
		)
	}

	rows := make([]g.Node, 0, len(plans))
	for _, plan := range plans {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-6 py-4"),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(plan.Name)),
					Div(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(plan.Description)),
				)),
			Td(Class("px-6 py-4 text-sm"),
				Span(Class("font-medium text-slate-900 dark:text-white"),
					g.Text(formatMoney(plan.BasePrice, plan.Currency))),
				Span(Class("text-slate-500 dark:text-gray-400"),
					g.Text("/"+string(plan.BillingInterval))),
			),
			Td(Class("px-6 py-4"), e.statusBadge(string(plan.BillingPattern))),
			Td(Class("px-6 py-4"),
				Div(Class("flex flex-col gap-1"),
					e.planStatusBadge(plan),
					e.planSyncStatusBadge(plan),
				),
			),
			Td(Class("px-6 py-4 text-right"),
				Div(
					Class("flex items-center justify-end gap-2"),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
						Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
						g.Text("View"),
					),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/edit"),
						Class("text-slate-600 hover:text-slate-700 dark:text-gray-400 text-sm"),
						g.Text("Edit"),
					),
					// Sync button (only show if not synced)
					g.If(plan.ProviderPlanID == "" || plan.ProviderPriceID == "",
						Form(
							Method("POST"),
							Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/sync"),
							Class("inline"),
							Button(
								Type("submit"),
								Class("text-blue-600 hover:text-blue-700 dark:text-blue-400 text-sm"),
								g.Text("Sync"),
							),
						),
					),
				)),
		))
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Plan")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Price")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Type")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
					Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
				),
			),
			TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
		),
	)
}

func (e *DashboardExtension) renderSubscriptionsTable(ctx context.Context, subs []*core.Subscription, currentApp *app.App, basePath string) g.Node {
	if len(subs) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
			lucide.Users(Class("mx-auto h-12 w-12 text-slate-300 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"), g.Text("No subscriptions yet")),
			P(Class("text-slate-500 dark:text-gray-400"), g.Text("Subscriptions will appear here when customers sign up")),
		)
	}

	rows := make([]g.Node, 0, len(subs))
	for _, sub := range subs {
		planName := "-"
		if sub.Plan != nil {
			planName = sub.Plan.Name
		}

		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-6 py-4"),
				Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(sub.ID.String()[:8]+"..."))),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"), g.Text(planName)),
			Td(Class("px-6 py-4"), e.subscriptionStatusBadge(sub)),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"), g.Text(fmt.Sprintf("%d", sub.Quantity))),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"), g.Text(sub.CurrentPeriodEnd.Format("Jan 2, 2006"))),
			Td(Class("px-6 py-4 text-right"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions/"+sub.ID.String()),
					Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
					g.Text("View"),
				)),
		))
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Subscription")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Plan")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Quantity")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Renews")),
					Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
				),
			),
			TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
		),
	)
}

func (e *DashboardExtension) renderAddOnsTable(ctx context.Context, addons []*core.AddOn, currentApp *app.App, basePath string) g.Node {
	if len(addons) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
			lucide.Puzzle(Class("mx-auto h-12 w-12 text-slate-300 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"), g.Text("No add-ons yet")),
			P(Class("text-slate-500 dark:text-gray-400 mb-4"), g.Text("Create add-ons to offer additional features to your customers")),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Add-on"),
			),
		)
	}

	rows := make([]g.Node, 0, len(addons))
	for _, addon := range addons {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-6 py-4"),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(addon.Name)),
					Div(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(addon.Description)),
				),
			),
			Td(Class("px-6 py-4 text-sm"),
				g.Text(formatMoney(addon.Price, addon.Currency)),
			),
			Td(Class("px-6 py-4"), e.statusBadge(string(addon.BillingPattern))),
			Td(Class("px-6 py-4"),
				g.If(addon.IsActive, e.statusBadge("active")),
				g.If(!addon.IsActive, e.statusBadge("inactive")),
			),
			Td(Class("px-6 py-4 text-right"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons/"+addon.ID.String()),
					Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
					g.Text("View"),
				),
			),
		))
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Add-on")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Price")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Type")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
					Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
				),
			),
			TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
		),
	)
}

func (e *DashboardExtension) renderInvoicesTable(ctx context.Context, invoices []*core.Invoice, currentApp *app.App, basePath string) g.Node {
	if len(invoices) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
			lucide.FileText(Class("mx-auto h-12 w-12 text-slate-300 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"), g.Text("No invoices yet")),
			P(Class("text-slate-500 dark:text-gray-400"), g.Text("Invoices will be generated when subscriptions are billed")),
		)
	}

	rows := make([]g.Node, 0, len(invoices))
	for _, inv := range invoices {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-6 py-4"),
				Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(inv.Number))),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(inv.SubscriptionID.String()[:8]+"...")),
			Td(Class("px-6 py-4 text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(formatMoney(inv.Total, inv.Currency))),
			Td(Class("px-6 py-4"), e.invoiceStatusBadge(inv)),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(inv.CreatedAt.Format("Jan 2, 2006"))),
			Td(Class("px-6 py-4 text-right"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices/"+inv.ID.String()),
					Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
					g.Text("View"),
				)),
		))
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Invoice")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Subscription")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Amount")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Date")),
					Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
				),
			),
			TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
		),
	)
}

func (e *DashboardExtension) renderCouponsTable(ctx context.Context, coupons []*core.Coupon, currentApp *app.App, basePath string) g.Node {
	if len(coupons) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
			lucide.Ticket(Class("mx-auto h-12 w-12 text-slate-300 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-medium text-slate-900 dark:text-white mb-2"), g.Text("No coupons yet")),
			P(Class("text-slate-500 dark:text-gray-400 mb-4"), g.Text("Create coupons to offer discounts to your customers")),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Coupon"),
			),
		)
	}

	rows := make([]g.Node, 0, len(coupons))
	for _, coupon := range coupons {
		discountText := ""
		switch coupon.Type {
		case core.CouponTypePercentage:
			discountText = fmt.Sprintf("%.0f%% off", coupon.PercentOff)
		case core.CouponTypeFixedAmount:
			discountText = formatMoney(coupon.AmountOff, coupon.Currency) + " off"
		case core.CouponTypeTrialExtension:
			discountText = fmt.Sprintf("+%d trial days", coupon.TrialDays)
		case core.CouponTypeFreeMonths:
			discountText = fmt.Sprintf("%d free months", coupon.FreeMonths)
		}

		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-6 py-4"),
				Div(
					Div(Class("font-mono font-medium text-slate-900 dark:text-white"), g.Text(coupon.Code)),
					Div(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(coupon.Name)),
				)),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(discountText)),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(fmt.Sprintf("%d / %d", coupon.TimesRedeemed, coupon.MaxRedemptions))),
			Td(Class("px-6 py-4"),
				g.If(coupon.Status == core.CouponStatusActive, e.statusBadge("active")),
				g.If(coupon.Status != core.CouponStatusActive, e.statusBadge(string(coupon.Status))),
			),
			Td(Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
				g.If(coupon.ValidUntil != nil, g.Text(coupon.ValidUntil.Format("Jan 2, 2006"))),
				g.If(coupon.ValidUntil == nil, g.Text("Never")),
			),
			Td(Class("px-6 py-4 text-right"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons/"+coupon.ID.String()),
					Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
					g.Text("View"),
				)),
		))
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Code")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Discount")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Redemptions")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
					Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Expires")),
					Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
				),
			),
			TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
		),
	)
}

func (e *DashboardExtension) renderStatusFilterTabs(currentStatus string, counts map[string]int, currentApp *app.App, basePath, path string) g.Node {
	statuses := []struct {
		value string
		label string
	}{
		{"", "All"},
		{"active", "Active"},
		{"trialing", "Trialing"},
		{"past_due", "Past Due"},
		{"canceled", "Canceled"},
	}

	tabs := make([]g.Node, 0, len(statuses))
	for _, s := range statuses {
		isActive := currentStatus == s.value

		count := counts[s.value]
		if s.value == "" {
			count = 0
			for _, c := range counts {
				count += c
			}
		}

		classes := "px-4 py-2 text-sm font-medium rounded-lg transition-colors "
		if isActive {
			classes += "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
		} else {
			classes += "text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800"
		}

		href := basePath + "/app/" + currentApp.ID.String() + path
		if s.value != "" {
			href += "?status=" + s.value
		}

		tabs = append(tabs, A(
			Href(href),
			Class(classes),
			g.Text(fmt.Sprintf("%s (%d)", s.label, count)),
		))
	}

	return Div(
		Class("flex flex-wrap gap-2 mb-4"),
		g.Group(tabs),
	)
}

func (e *DashboardExtension) renderUsageCard(title, current, limit string, percent float64, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between mb-4"),
			Div(Class("flex items-center gap-3"),
				Div(Class("rounded-full bg-violet-100 p-2 dark:bg-violet-900/30"), icon),
				H4(Class("font-medium text-slate-900 dark:text-white"), g.Text(title)),
			),
		),
		Div(
			Class("mb-2"),
			Span(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text(current)),
			Span(Class("text-slate-500 dark:text-gray-400"), g.Text(" / "+limit)),
		),
		Div(
			Class("w-full bg-slate-200 rounded-full h-2 dark:bg-gray-700"),
			Div(
				Class("bg-violet-600 h-2 rounded-full"),
				StyleAttr(fmt.Sprintf("width: %.1f%%", percent)),
			),
		),
		Div(
			Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
			g.Text(fmt.Sprintf("%.1f%% used", percent)),
		),
	)
}

func (e *DashboardExtension) renderMetricCard(title, value string, change float64, icon g.Node) g.Node {
	changeColor := "text-green-600 dark:text-green-400"
	changePrefix := "+"

	if change < 0 {
		changeColor = "text-red-600 dark:text-red-400"
		changePrefix = ""
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("mt-1 text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
				g.If(change != 0,
					Div(Class("mt-1 text-sm "+changeColor),
						g.Text(fmt.Sprintf("%s%.1f%%", changePrefix, change))),
				),
			),
			Div(
				Class("rounded-full bg-violet-100 p-3 dark:bg-violet-900/30 text-violet-600 dark:text-violet-400"),
				icon,
			),
		),
	)
}

// renderPaymentMethodsOverviewWidget renders the payment methods widget for billing overview.
func (e *DashboardExtension) renderPaymentMethodsOverviewWidget(ctx context.Context, currentApp *app.App, basePath string) g.Node {
	// Get payment methods for this organization
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(ctx, currentApp.ID)
	if err != nil || len(paymentMethods) == 0 {
		return Div(
			Class("text-center py-8 text-slate-500 dark:text-gray-400"),
			lucide.CreditCard(Class("mx-auto h-12 w-12 mb-3")),
			P(g.Text("No payment methods added")),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/payment-methods/add"),
				Class("mt-4 inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Add payment method"),
			),
		)
	}

	// Find default payment method
	var defaultPM *core.PaymentMethod

	for _, pm := range paymentMethods {
		if pm.IsDefault {
			defaultPM = pm

			break
		}
	}

	// If no default but have methods, use first one
	if defaultPM == nil && len(paymentMethods) > 0 {
		defaultPM = paymentMethods[0]
	}

	return Div(
		Class("space-y-4"),
		// Default payment method card
		g.If(defaultPM != nil,
			Div(
				Class("flex items-center gap-4 p-4 rounded-lg bg-slate-50 dark:bg-gray-800"),
				Div(
					Class("flex h-10 w-10 items-center justify-center rounded-lg bg-white dark:bg-gray-700"),
					g.If(defaultPM.IsCard(),
						lucide.CreditCard(Class("size-5 text-slate-600 dark:text-gray-400")),
					),
					g.If(defaultPM.IsBankAccount(),
						lucide.Building2(Class("size-5 text-slate-600 dark:text-gray-400")),
					),
				),
				Div(
					Class("flex-1"),
					Div(
						Class("flex items-center gap-2"),
						P(Class("font-medium text-slate-900 dark:text-white"), g.Text(defaultPM.DisplayName())),
						Span(
							Class("inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"),
							g.Text("Default"),
						),
						g.If(defaultPM.IsExpired(),
							Span(
								Class("inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-200"),
								g.Text("Expired"),
							),
						),
						g.If(defaultPM.WillExpireSoon(30) && !defaultPM.IsExpired(),
							Span(
								Class("inline-flex items-center rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"),
								g.Text("Expiring Soon"),
							),
						),
					),
					P(Class("text-sm text-slate-600 dark:text-gray-400 mt-1"),
						g.If(defaultPM.IsCard(),
							g.Textf("Expires %02d/%d", defaultPM.CardExpMonth, defaultPM.CardExpYear),
						),
						g.If(defaultPM.IsBankAccount() && defaultPM.BankAccountType != "",
							g.Textf("%s account", defaultPM.BankAccountType),
						),
					),
				),
			),
		),

		// Additional payment methods count
		g.If(len(paymentMethods) > 1,
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Textf("+%d more payment method", len(paymentMethods)-1),
				g.If(len(paymentMethods) > 2, g.Text("s")),
			),
		),
	)
}
