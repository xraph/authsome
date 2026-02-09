package subscription

import (
	"fmt"
	"strconv"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers/types"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServePlanDetailPage renders the plan detail page.
func (e *DashboardExtension) ServePlanDetailPage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("invalid plan ID")
	}

	plan, err := e.plugin.planSvc.GetByID(reqCtx, planID)
	if err != nil {
		return nil, errs.NotFound("plan not found")
	}

	// Get subscription count for this plan
	_, subCount, _ := e.plugin.subscriptionSvc.List(reqCtx, nil, nil, &planID, "", 1, 1)

	content := Div(
		Class("space-y-2"),

		// Back button and header
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Plans"),
		),

		// Plan header
		Div(
			Class("flex items-start justify-between"),
			Div(
				Div(
					Class("flex items-center gap-3"),
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text(plan.Name)),
					e.planStatusBadge(plan),
					e.planSyncStatusBadge(plan),
				),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"), g.Text(plan.Description)),
			),
			Div(
				Class("flex gap-2"),
				// Sync button
				g.If(plan.ProviderPlanID == "" || plan.ProviderPriceID == "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/sync"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"),
							lucide.Cloud(Class("size-4")),
							g.Text("Sync to Provider"),
						),
					),
				),
				// Re-sync button (when already synced) - pushes to provider
				g.If(plan.ProviderPlanID != "" && plan.ProviderPriceID != "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/sync"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-blue-300 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-600 dark:text-blue-400 dark:hover:bg-blue-900/20"),
							lucide.CloudUpload(Class("size-4")),
							g.Text("Push to Stripe"),
						),
					),
				),
				// Pull from provider button (when already synced) - pulls from provider
				g.If(plan.ProviderPlanID != "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/sync-from-provider"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-green-300 px-4 py-2 text-sm font-medium text-green-700 hover:bg-green-50 dark:border-green-600 dark:text-green-400 dark:hover:bg-green-900/20"),
							lucide.CloudDownload(Class("size-4")),
							g.Text("Pull from Stripe"),
						),
					),
				),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/edit"),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
					lucide.Pencil(Class("size-4")),
					g.Text("Edit"),
				),
				g.If(plan.IsActive,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/archive"),
						Class("inline"),
						g.Attr("onsubmit", "return confirm('Are you sure you want to archive this plan?')"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-amber-300 px-4 py-2 text-sm font-medium text-amber-700 hover:bg-amber-50 dark:border-amber-600 dark:text-amber-400 dark:hover:bg-amber-900/20"),
							lucide.Archive(Class("size-4")),
							g.Text("Archive"),
						),
					),
				),
				// Delete button
				Form(
					Method("POST"),
					Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/delete"),
					Class("inline"),
					g.Attr("onsubmit", "return confirm('Are you sure you want to permanently delete this plan? This action cannot be undone.')"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 dark:border-red-600 dark:text-red-400 dark:hover:bg-red-900/20"),
						lucide.Trash2(Class("size-4")),
						g.Text("Delete"),
					),
				),
			),
		),

		// Plan info cards
		Div(
			Class("grid gap-6 md:grid-cols-4"),
			e.statsCard("Price", formatMoney(plan.BasePrice, plan.Currency)+"/"+string(plan.BillingInterval), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Subscribers", strconv.Itoa(subCount), lucide.Users(Class("size-5 text-violet-600"))),
			e.statsCard("Billing Pattern", string(plan.BillingPattern), lucide.Repeat(Class("size-5 text-violet-600"))),
			e.statsCard("Trial Days", fmt.Sprintf("%d days", plan.TrialDays), lucide.Clock(Class("size-5 text-violet-600"))),
		),

		// Plan details
		Div(
			Class("grid gap-6 lg:grid-cols-2"),
			// Features
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Features")),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/features"),
						Class("inline-flex items-center gap-1.5 text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"),
						lucide.Settings(Class("size-4")),
						g.Text("Manage"),
					),
				),
				Div(
					Class("px-6 py-4"),
					g.If(len(plan.Features) == 0,
						P(Class("text-slate-500 dark:text-gray-400"), g.Text("No features defined")),
					),
					g.If(len(plan.Features) > 0,
						Ul(
							Class("space-y-2"),
							g.Group(renderPlanFeaturesList(plan.Features)),
						),
					),
				),
			),
			// Metadata
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Plan Details")),
				),
				Div(
					Class("px-6 py-4 space-y-4"),
					detailRow("Plan ID", plan.ID.String()),
					detailRow("Created", plan.CreatedAt.Format("Jan 2, 2006 3:04 PM")),
					detailRow("Updated", plan.UpdatedAt.Format("Jan 2, 2006 3:04 PM")),
					detailRow("Provider Price ID", stringOrDash(plan.ProviderPriceID)),
					detailRow("Provider Product ID", stringOrDash(plan.ProviderPlanID)),
					detailRow("Public", boolToYesNo(plan.IsPublic)),
				),
			),
		),

		// Pricing tiers (if tiered pricing)
		g.If(plan.BillingPattern == core.BillingPatternTiered && len(plan.PriceTiers) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Pricing Tiers")),
				),
				Div(
					Class("px-6 py-4"),
					e.renderPricingTiersTable(plan),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeSubscriptionDetailPage renders the subscription detail page.
func (e *DashboardExtension) ServeSubscriptionDetailPage(ctx *router.PageContext) (g.Node, error) {
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

	sub, err := e.plugin.subscriptionSvc.GetByID(reqCtx, subID)
	if err != nil {
		return nil, errs.NotFound("subscription not found")
	}

	// Get usage data
	usageData, _ := e.plugin.usageSvc.GetCurrentPeriodUsage(reqCtx, subID)

	// Get recent invoices - fetch directly from provider on-demand
	var invoices []*core.Invoice

	if sub.ProviderSubID != "" && e.plugin.provider != nil {
		providerInvoices, err := e.plugin.provider.ListSubscriptionInvoices(reqCtx, sub.ProviderSubID, 5)
		if err == nil {
			invoices = convertProviderInvoicesToCore(providerInvoices)
		}
	}
	// Fallback to local if provider fetch fails or no provider ID
	if len(invoices) == 0 {
		invoices, _, _ = e.plugin.invoiceSvc.List(reqCtx, nil, &subID, "", 1, 5)
	}

	planName := "-"
	if sub.Plan != nil {
		planName = sub.Plan.Name
	}

	content := Div(
		Class("space-y-2"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Subscriptions"),
		),

		// Subscription header
		Div(
			Class("flex items-start justify-between"),
			Div(
				Div(
					Class("flex items-center gap-3"),
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Subscription Details")),
					e.subscriptionStatusBadge(sub),
					e.subscriptionSyncStatusBadge(sub),
				),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"), g.Text("Plan: "+planName)),
			),
			Div(
				Class("flex gap-2"),
				// Sync TO provider button - show if not synced yet
				g.If(sub.ProviderSubID == "",
					Form(
						Method("POST"),
						Action(basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync"),
						Class("inline"),
						g.Attr("hx-post", basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync"),
						g.Attr("hx-target", "body"),
						g.Attr("hx-swap", "none"),
						g.Attr("hx-on::after-request", "location.reload()"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"),
							lucide.CloudUpload(Class("size-4")),
							g.Text("Sync to Provider"),
						),
					),
				),
				// Sync FROM provider button - show if already synced
				g.If(sub.ProviderSubID != "",
					Form(
						Method("POST"),
						Action(basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync-from-provider"),
						Class("inline"),
						g.Attr("hx-post", basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync-from-provider"),
						g.Attr("hx-target", "body"),
						g.Attr("hx-swap", "none"),
						g.Attr("hx-on::after-request", "location.reload()"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-blue-300 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-600 dark:text-blue-400 dark:hover:bg-blue-900/20"),
							lucide.CloudDownload(Class("size-4")),
							g.Text("Pull from Provider"),
						),
					),
				),
				g.If(sub.Status == "active" || sub.Status == "trialing",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/subscriptions/"+sub.ID.String()+"/cancel"),
						Class("inline"),
						g.Attr("onsubmit", "return confirm('Are you sure you want to cancel this subscription?')"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 dark:border-red-600 dark:text-red-400 dark:hover:bg-red-900/20"),
							lucide.X(Class("size-4")),
							g.Text("Cancel Subscription"),
						),
					),
				),
			),
		),

		// Info cards
		Div(
			Class("grid gap-6 md:grid-cols-4"),
			e.statsCard("Monthly Cost", func() string {
				if sub.Plan != nil {
					return formatMoney(sub.Plan.BasePrice*int64(sub.Quantity), sub.Plan.Currency)
				}

				return "-"
			}(), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Quantity", strconv.Itoa(sub.Quantity), lucide.Hash(Class("size-5 text-violet-600"))),
			e.statsCard("Current Period End", sub.CurrentPeriodEnd.Format("Jan 2, 2006"), lucide.Calendar(Class("size-5 text-violet-600"))),
			e.statsCard("Created", sub.CreatedAt.Format("Jan 2, 2006"), lucide.Clock(Class("size-5 text-violet-600"))),
		),

		// Details grid
		Div(
			Class("grid gap-6 lg:grid-cols-2"),
			// Subscription details
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Subscription Details")),
				),
				Div(
					Class("px-6 py-4 space-y-4"),
					g.Group(func() []g.Node {
						nodes := []g.Node{
							detailRow("Subscription ID", sub.ID.String()),
							detailRow("Organization ID", sub.OrganizationID.String()),
							detailRow("Status", string(sub.Status)),
							detailRow("Current Period Start", sub.CurrentPeriodStart.Format("Jan 2, 2006")),
							detailRow("Current Period End", sub.CurrentPeriodEnd.Format("Jan 2, 2006")),
						}
						if sub.TrialEnd != nil && !sub.TrialEnd.IsZero() {
							nodes = append(nodes, detailRow("Trial Ends", sub.TrialEnd.Format("Jan 2, 2006")))
						}

						if sub.CanceledAt != nil && !sub.CanceledAt.IsZero() {
							nodes = append(nodes, detailRow("Canceled At", sub.CanceledAt.Format("Jan 2, 2006")))
						}

						return nodes
					}()),
					Div(
						Class("flex items-center justify-between py-3 border-t border-slate-200 dark:border-gray-800"),
						Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Provider Subscription ID")),
						Div(
							Class("flex items-center gap-2"),
							Span(Class("text-sm text-slate-900 dark:text-white"), g.Text(stringOrDash(sub.ProviderSubID))),
							g.If(sub.ProviderSubID != "",
								Div(
									Class("flex gap-1"),
									// Sync FROM provider button
									Form(
										Method("POST"),
										Action(basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync-from-provider"),
										Class("inline"),
										g.Attr("hx-post", basePath+"/subscription/subscriptions/"+sub.ID.String()+"/sync-from-provider"),
										g.Attr("hx-target", "body"),
										g.Attr("hx-swap", "none"),
										g.Attr("hx-on::after-request", "location.reload()"),
										Button(
											Type("submit"),
											Class("inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-blue-700 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/20"),
											g.Attr("title", "Pull latest data from payment provider"),
											lucide.CloudDownload(Class("size-3")),
											g.Text("Pull"),
										),
									),
									// View in provider button
									A(
										Href("https://dashboard.stripe.com/subscriptions/"+sub.ProviderSubID),
										Target("_blank"),
										Class("inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:text-gray-300 dark:hover:bg-gray-800"),
										g.Attr("title", "View in Stripe dashboard"),
										lucide.ExternalLink(Class("size-3")),
										g.Text("Stripe"),
									),
								),
							),
						),
					),
				),
			),
			// Usage (if applicable)
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Usage This Period")),
				),
				Div(
					Class("px-6 py-4"),
					g.If(len(usageData) == 0,
						P(Class("text-slate-500 dark:text-gray-400"), g.Text("No usage data available")),
					),
					g.If(len(usageData) > 0,
						e.renderUsageDataList(usageData),
					),
				),
			),
		),

		// Recent invoices
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Recent Invoices")),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices?subscription="+sub.ID.String()),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
					g.Text("View all →"),
				),
			),
			Div(
				Class("px-6 py-4"),
				g.If(len(invoices) == 0,
					P(Class("text-slate-500 dark:text-gray-400"), g.Text("No invoices yet")),
				),
				g.If(len(invoices) > 0,
					e.renderInvoicesList(invoices, currentApp, basePath),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAddOnDetailPage renders the add-on detail page.
func (e *DashboardExtension) ServeAddOnDetailPage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	addonID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid add-on ID")
	}

	addon, err := e.plugin.addOnSvc.GetByID(reqCtx, addonID)
	if err != nil {
		return nil, errs.NotFound("add-on not found")
	}

	content := Div(
		Class("space-y-2"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Add-ons"),
		),

		// Header
		Div(
			Class("flex items-start justify-between"),
			Div(
				Div(
					Class("flex items-center gap-3"),
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text(addon.Name)),
					g.If(addon.IsActive, e.statusBadge("active")),
					g.If(!addon.IsActive, e.statusBadge("inactive")),
				),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"), g.Text(addon.Description)),
			),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons/"+addon.ID.String()+"/edit"),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
				lucide.Pencil(Class("size-4")),
				g.Text("Edit"),
			),
		),

		// Info cards
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Price", formatMoney(addon.Price, addon.Currency), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Billing Pattern", string(addon.BillingPattern), lucide.Repeat(Class("size-5 text-violet-600"))),
			e.statsCard("Created", addon.CreatedAt.Format("Jan 2, 2006"), lucide.Calendar(Class("size-5 text-violet-600"))),
		),

		// Details
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Add-on Details")),
			),
			Div(
				Class("px-6 py-4 space-y-4"),
				detailRow("Add-on ID", addon.ID.String()),
				detailRow("Slug", addon.Slug),
				detailRow("Provider Price ID", stringOrDash(addon.ProviderPriceID)),
				detailRow("Active", boolToYesNo(addon.IsActive)),
				detailRow("Created", addon.CreatedAt.Format("Jan 2, 2006 3:04 PM")),
				detailRow("Updated", addon.UpdatedAt.Format("Jan 2, 2006 3:04 PM")),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeInvoiceDetailPage renders the invoice detail page.
func (e *DashboardExtension) ServeInvoiceDetailPage(ctx *router.PageContext) (g.Node, error) {
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

	invoice, err := e.plugin.invoiceSvc.GetByID(reqCtx, invoiceID)
	if err != nil {
		return nil, errs.NotFound("invoice not found")
	}

	content := Div(
		Class("space-y-2"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Invoices"),
		),

		// Header
		Div(
			Class("flex items-start justify-between"),
			Div(
				Div(
					Class("flex items-center gap-3"),
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Invoice "+invoice.Number)),
					e.invoiceStatusBadge(invoice),
				),
			),
			Div(
				Class("flex gap-2"),
				g.If(invoice.HostedInvoiceURL != "",
					A(
						Href(invoice.HostedInvoiceURL),
						Target("_blank"),
						Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
						lucide.ExternalLink(Class("size-4")),
						g.Text("View Invoice"),
					),
				),
				g.If(invoice.ProviderPDFURL != "",
					A(
						Href(invoice.ProviderPDFURL),
						Target("_blank"),
						Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
						lucide.Download(Class("size-4")),
						g.Text("Download PDF"),
					),
				),
				g.If(invoice.Status != "paid" && invoice.Status != "void",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices/"+invoice.ID.String()+"/mark-paid"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"),
							lucide.Check(Class("size-4")),
							g.Text("Mark as Paid"),
						),
					),
				),
			),
		),

		// Summary cards
		Div(
			Class("grid gap-6 md:grid-cols-4"),
			e.statsCard("Total", formatMoney(invoice.Total, invoice.Currency), lucide.DollarSign(Class("size-5 text-violet-600"))),
			e.statsCard("Subtotal", formatMoney(invoice.Subtotal, invoice.Currency), lucide.Calculator(Class("size-5 text-violet-600"))),
			e.statsCard("Tax", formatMoney(invoice.Tax, invoice.Currency), lucide.Receipt(Class("size-5 text-violet-600"))),
			e.statsCard("Due Date", func() string {
				if !invoice.DueDate.IsZero() {
					return invoice.DueDate.Format("Jan 2, 2006")
				}

				return "-"
			}(), lucide.Calendar(Class("size-5 text-violet-600"))),
		),

		// Invoice details
		Div(
			Class("grid gap-6 lg:grid-cols-2"),
			// Details
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Invoice Details")),
				),
				Div(
					Class("px-6 py-4 space-y-4"),
					detailRow("Invoice ID", invoice.ID.String()),
					detailRow("Invoice Number", invoice.Number),
					detailRow("Subscription", invoice.SubscriptionID.String()),
					detailRow("Status", string(invoice.Status)),
					detailRow("Created", invoice.CreatedAt.Format("Jan 2, 2006 3:04 PM")),
					g.If(invoice.PaidAt != nil, detailRow("Paid At", invoice.PaidAt.Format("Jan 2, 2006 3:04 PM"))),
					detailRow("Provider Invoice ID", stringOrDash(invoice.ProviderInvoiceID)),
				),
			),
			// Payment info
			Div(
				Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Payment Information")),
				),
				Div(
					Class("px-6 py-4 space-y-4"),
					detailRow("Currency", invoice.Currency),
					detailRow("Subtotal", formatMoney(invoice.Subtotal, invoice.Currency)),
					detailRow("Tax", formatMoney(invoice.Tax, invoice.Currency)),
					detailRow("Total", formatMoney(invoice.Total, invoice.Currency)),
					g.If(invoice.AmountPaid > 0, detailRow("Amount Paid", formatMoney(invoice.AmountPaid, invoice.Currency))),
					g.If(invoice.AmountDue > 0, detailRow("Amount Due", formatMoney(invoice.AmountDue, invoice.Currency))),
				),
			),
		),

		// Line items
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Line Items")),
			),
			Div(
				Class("px-6 py-4"),
				g.If(len(invoice.Items) == 0,
					P(Class("text-slate-500 dark:text-gray-400"), g.Text("No line items")),
				),
				g.If(len(invoice.Items) > 0,
					e.renderLineItemsTable(invoice),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// Helper components for detail pages

func detailRow(label, value string) g.Node {
	return Div(
		Class("flex justify-between py-2 border-b border-slate-100 dark:border-gray-800 last:border-0"),
		Span(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(label)),
		Span(Class("text-sm font-medium text-slate-900 dark:text-white"), g.Text(value)),
	)
}

func stringOrDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}

// convertProviderInvoicesToCore converts provider invoices to core invoice format for display.
func convertProviderInvoicesToCore(providerInvs []*types.ProviderInvoice) []*core.Invoice {
	if providerInvs == nil {
		return []*core.Invoice{}
	}

	result := make([]*core.Invoice, len(providerInvs))
	for i, inv := range providerInvs {
		// Parse ID as xid (or generate a new one for display purposes)
		invoiceID := xid.New() // Use generated ID for display since provider IDs may not be xid format

		// Convert status string to core status type
		status := core.InvoiceStatusDraft

		switch inv.Status {
		case "open":
			status = core.InvoiceStatusOpen
		case "paid":
			status = core.InvoiceStatusPaid
		case "void":
			status = core.InvoiceStatusVoid
		case "uncollectible":
			status = core.InvoiceStatusUncollectible
		}

		result[i] = &core.Invoice{
			ID:             invoiceID,
			Number:         inv.ID,      // Use provider ID as invoice number for display
			SubscriptionID: xid.NilID(), // We don't have the local subscription ID
			Status:         status,
			Currency:       inv.Currency,
			Total:          inv.Total,
			Subtotal:       inv.Subtotal,
			Tax:            inv.Tax,
			AmountDue:      inv.AmountDue,
			AmountPaid:     inv.AmountPaid,
			CreatedAt:      time.Unix(inv.PeriodStart, 0),
		}
	}

	return result
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}

	return "No"
}

func renderFeaturesList(features map[string]any) []g.Node {
	items := make([]g.Node, 0, len(features))
	for key, value := range features {
		items = append(items, Li(
			Class("flex items-center gap-2"),
			lucide.Check(Class("size-4 text-green-600 dark:text-green-400")),
			Span(Class("text-sm text-slate-700 dark:text-gray-300"),
				g.Text(fmt.Sprintf("%s: %v", key, value))),
		))
	}

	return items
}

func renderPlanFeaturesList(features []core.PlanFeature) []g.Node {
	items := make([]g.Node, 0, len(features))
	for _, feature := range features {
		items = append(items, Li(
			Class("flex items-center gap-2"),
			lucide.Check(Class("size-4 text-green-600 dark:text-green-400")),
			Span(Class("text-sm text-slate-700 dark:text-gray-300"),
				g.Text(fmt.Sprintf("%s: %v", feature.Name, feature.Value))),
		))
	}

	return items
}

func (e *DashboardExtension) renderPricingTiersTable(plan *core.Plan) g.Node {
	rows := make([]g.Node, 0, len(plan.PriceTiers))
	for i, tier := range plan.PriceTiers {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-4 py-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(fmt.Sprintf("Tier %d", i+1))),
			Td(Class("px-4 py-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(fmt.Sprintf("Up to %d", tier.UpTo))),
			Td(Class("px-4 py-2 text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(formatMoney(tier.UnitAmount, plan.Currency))),
			Td(Class("px-4 py-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(formatMoney(tier.FlatAmount, plan.Currency))),
		))
	}

	return Table(
		Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
		THead(
			Class("bg-slate-50 dark:bg-gray-800"),
			Tr(
				Th(Class("px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Tier")),
				Th(Class("px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Units")),
				Th(Class("px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Unit Price")),
				Th(Class("px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Flat Fee")),
			),
		),
		TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
	)
}

func (e *DashboardExtension) renderUsageDataList(usageData map[string]int64) g.Node {
	items := make([]g.Node, 0, len(usageData))
	for key, value := range usageData {
		items = append(items, Div(
			Class("flex justify-between py-2 border-b border-slate-100 dark:border-gray-800 last:border-0"),
			Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text(key)),
			Span(Class("text-sm font-medium text-slate-900 dark:text-white"), g.Text(strconv.FormatInt(value, 10))),
		))
	}

	return Div(g.Group(items))
}

func (e *DashboardExtension) subscriptionSyncStatusBadge(sub *core.Subscription) g.Node {
	if sub.ProviderSubID == "" {
		return Span(
			Class("inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400"),
			lucide.CloudOff(Class("size-3")),
			g.Text("Not Synced"),
		)
	}

	return Span(
		Class("inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"),
		lucide.Cloud(Class("size-3")),
		g.Text("Synced"),
	)
}

func (e *DashboardExtension) renderInvoicesList(invoices []*core.Invoice, currentApp *app.App, basePath string) g.Node {
	items := make([]g.Node, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, Div(
			Class("flex items-center justify-between py-3 border-b border-slate-100 dark:border-gray-800 last:border-0"),
			Div(
				Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(inv.Number)),
				Div(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(inv.CreatedAt.Format("Jan 2, 2006"))),
			),
			Div(
				Class("flex items-center gap-3"),
				e.invoiceStatusBadge(inv),
				Span(Class("font-medium text-slate-900 dark:text-white"), g.Text(formatMoney(inv.Total, inv.Currency))),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/invoices/"+inv.ID.String()),
					Class("text-violet-600 hover:text-violet-700 dark:text-violet-400 text-sm"),
					g.Text("View"),
				),
			),
		))
	}

	return Div(g.Group(items))
}

func (e *DashboardExtension) renderLineItemsTable(invoice *core.Invoice) g.Node {
	rows := make([]g.Node, 0, len(invoice.Items))
	for _, item := range invoice.Items {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
			Td(Class("px-4 py-3 text-sm text-slate-900 dark:text-white"), g.Text(item.Description)),
			Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"), g.Text(strconv.FormatInt(item.Quantity, 10))),
			Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"), g.Text(formatMoney(item.UnitAmount, invoice.Currency))),
			Td(Class("px-4 py-3 text-sm font-medium text-slate-900 dark:text-white"), g.Text(formatMoney(item.Amount, invoice.Currency))),
		))
	}

	return Table(
		Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
		THead(
			Class("bg-slate-50 dark:bg-gray-800"),
			Tr(
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Description")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Qty")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Unit Price")),
				Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase"), g.Text("Amount")),
			),
		),
		TBody(Class("divide-y divide-slate-200 dark:divide-gray-700"), g.Group(rows)),
	)
}
