package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServeFeaturesListPage renders the features list dashboard
func (e *DashboardExtension) ServeFeaturesListPage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath
	reqCtx := ctx.Request.Context()

	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 20)
	typeFilter := ctx.Query("type")
	publicFilter := ctx.Query("public") == "true"

	features, total, _ := e.plugin.featureSvc.List(reqCtx, currentApp.ID, typeFilter, publicFilter, page, pageSize)
	totalPages := int((int64(total) + int64(pageSize) - 1) / int64(pageSize))

	// Count by type
	allFeatures, _, _ := e.plugin.featureSvc.List(reqCtx, currentApp.ID, "", false, 1, 10000)
	typeCounts := make(map[string]int)
	for _, f := range allFeatures {
		typeCounts[string(f.Type)]++
	}

	// Check for success messages
	successMsg := ctx.Query("success")

	content := Div(
		Class("space-y-2"),

		// Success message
		g.If(successMsg != "",
			Div(
				Class("rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"),
				Div(
					Class("flex items-center gap-3"),
					lucide.Check(Class("size-5 text-green-600 dark:text-green-400")),
					P(Class("text-sm text-green-700 dark:text-green-400"), g.Text(successMsg)),
				),
			),
		),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Features")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage feature definitions for your subscription plans")),
			),
			Div(
				Class("flex items-center gap-2"),
				Form(
					Method("POST"),
					Action(basePath+"/app/"+currentApp.ID.String()+"/billing/features/sync-all-from-provider?productId="+currentApp.ID.String()),
					Class("inline"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg border border-blue-200 bg-white px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-700 dark:bg-slate-800 dark:text-blue-300"),
						lucide.RefreshCw(Class("size-4")),
						g.Text("Sync from Provider"),
					),
				),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/export"),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"),
					lucide.Download(Class("size-4")),
					g.Text("Export"),
				),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/import"),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"),
					lucide.Upload(Class("size-4")),
					g.Text("Import"),
				),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/create"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					lucide.Plus(Class("size-4")),
					g.Text("Create Feature"),
				),
			),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "features"),

		// Type filter tabs
		e.renderFeatureTypeFilterTabs(typeFilter, typeCounts, currentApp, basePath),

		// Features table
		e.renderFeaturesTable(reqCtx, features, currentApp, basePath),

		// Pagination
		e.renderPagination(page, totalPages, basePath+"/app/"+currentApp.ID.String()+"/billing/features"),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeFeatureDetailPage renders the feature detail page
func (e *DashboardExtension) ServeFeatureDetailPage(ctx *router.PageContext) (g.Node, error) {
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

	feature, err := e.plugin.featureSvc.GetByID(reqCtx, featureID)
	if err != nil {
		return nil, fmt.Errorf("feature not found")
	}

	// Get plans using this feature
	planLinks, _ := e.plugin.featureSvc.GetPlanFeatures(reqCtx, featureID) // This returns plans linked TO the feature

	// Check for error/success messages
	errorMsg := ctx.Query("error")
	successMsg := ctx.Query("success")

	content := Div(
		Class("space-y-2"),

		// Error message
		g.If(errorMsg != "",
			Div(
				Class("rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20"),
				Div(
					Class("flex items-center gap-3"),
					lucide.X(Class("size-5 text-red-600 dark:text-red-400")),
					Div(
						P(Class("font-medium text-red-800 dark:text-red-300"), g.Text("Error")),
						P(Class("text-sm text-red-700 dark:text-red-400"), g.Text(errorMsg)),
					),
				),
			),
		),

		// Success message
		g.If(successMsg != "",
			Div(
				Class("rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"),
				Div(
					Class("flex items-center gap-3"),
					lucide.Check(Class("size-5 text-green-600 dark:text-green-400")),
					P(Class("text-sm text-green-700 dark:text-green-400"), g.Text(successMsg)),
				),
			),
		),

		// Breadcrumb
		Nav(
			Class("flex"),
			Ol(
				Class("inline-flex items-center space-x-1 md:space-x-3"),
				Li(Class("inline-flex items-center"),
					A(Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features"),
						Class("text-sm font-medium text-gray-500 hover:text-violet-600"),
						g.Text("Features"))),
				Li(Class("flex items-center"),
					lucide.ChevronRight(Class("size-4 text-gray-400 mx-1")),
					Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
						g.Text(feature.Name))),
			),
		),

		// Feature header
		Div(
			Class("flex items-center justify-between"),
			Div(
				Class("flex items-center gap-4"),
				e.featureIcon(feature.Type),
				Div(
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
						g.Text(feature.Name)),
					P(Class("mt-1 text-slate-600 dark:text-gray-400"),
						g.Text(feature.Key)),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				// Sync button - show if not synced yet
				g.If(feature.ProviderFeatureID == "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/sync"),
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
				g.If(feature.ProviderFeatureID != "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/sync"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-blue-300 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-600 dark:text-blue-400 dark:hover:bg-blue-900/20"),
							lucide.CloudUpload(Class("size-4")),
							g.Text("Re-sync to Provider"),
						),
					),
				),
				// Pull from provider button (when already synced) - pulls from provider
				g.If(feature.ProviderFeatureID != "",
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/sync-from-provider"),
						Class("inline"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-blue-300 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-600 dark:text-blue-400 dark:hover:bg-blue-900/20"),
							lucide.CloudDownload(Class("size-4")),
							g.Text("Sync from Provider"),
						),
					),
				),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/edit"),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"),
					lucide.Pencil(Class("size-4")),
					g.Text("Edit"),
				),
				g.If(len(planLinks) > 0,
					// Show disabled delete button with tooltip if feature is in use
					Div(
						Class("relative group"),
						Button(
							Type("button"),
							Disabled(),
							Class("inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-100 px-4 py-2 text-sm font-medium text-gray-400 cursor-not-allowed dark:border-gray-700 dark:bg-gray-800"),
							lucide.Trash2(Class("size-4")),
							g.Text("Delete"),
						),
						Div(
							Class("absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block w-64 p-2 bg-gray-900 text-white text-xs rounded shadow-lg"),
							g.Text(fmt.Sprintf("Cannot delete: feature is linked to %d plan(s). Remove from plans first.", len(planLinks))),
						),
					),
				),
				g.If(len(planLinks) == 0,
					// Show active delete button if feature is not in use
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/delete"),
						g.Attr("onsubmit", "return confirm('Are you sure you want to delete this feature? This action cannot be undone.')"),
						Button(
							Type("submit"),
							Class("inline-flex items-center gap-2 rounded-lg border border-red-200 bg-white px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 dark:border-red-800 dark:bg-slate-800 dark:text-red-400"),
							lucide.Trash2(Class("size-4")),
							g.Text("Delete"),
						),
					),
				),
			),
		),

		// Feature details card
		Div(
			Class("grid gap-6 md:grid-cols-2"),
			// Info card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Feature Details")),
				Dl(
					Class("space-y-3"),
					e.detailRow("Type", e.featureTypeBadge(string(feature.Type))),
					e.detailRow("Unit", g.Text(feature.Unit)),
					e.detailRow("Reset Period", e.resetPeriodBadge(string(feature.ResetPeriod))),
					e.detailRow("Public", e.boolBadge(feature.IsPublic)),
					e.detailRow("Display Order", g.Text(fmt.Sprintf("%d", feature.DisplayOrder))),
					e.detailRow("Created", g.Text(feature.CreatedAt.Format("Jan 2, 2006 15:04"))),
				),
			),

			// Description card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Description")),
				g.If(feature.Description != "",
					P(Class("text-slate-600 dark:text-gray-400"), g.Text(feature.Description)),
				),
				g.If(feature.Description == "",
					P(Class("text-slate-400 dark:text-gray-500 italic"), g.Text("No description")),
				),
			),
		),

		// Tiers section (if tiered feature)
		g.If(feature.Type == core.FeatureTypeTiered && len(feature.Tiers) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Feature Tiers")),
				e.renderFeatureTiersTable(feature.Tiers),
			),
		),

		// Plans using this feature
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Plans Using This Feature")),
			g.If(len(planLinks) > 0,
				e.renderFeaturePlansTable(reqCtx, planLinks, currentApp, basePath),
			),
			g.If(len(planLinks) == 0,
				P(Class("text-slate-400 dark:text-gray-500 italic"),
					g.Text("No plans are using this feature yet")),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeFeatureUsagePage renders the feature usage monitoring page
func (e *DashboardExtension) ServeFeatureUsagePage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	content := Div(
		Class("space-y-2"),

		// Page header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("Feature Usage")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Monitor feature usage across organizations")),
		),

		// Billing sub-navigation
		e.renderBillingNav(currentApp, basePath, "usage"),

		// Info card
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800"),
			Div(
				Class("flex items-center gap-4"),
				lucide.TrendingUp(Class("size-8 text-violet-600")),
				Div(
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Feature Usage Analytics")),
					P(Class("text-slate-600 dark:text-gray-400"),
						g.Text("View detailed feature usage per organization. Select an organization to see their feature consumption.")),
				),
			),
		),

		// Coming soon placeholder
		Div(
			Class("rounded-lg border-2 border-dashed border-slate-300 p-12 text-center dark:border-slate-600"),
			lucide.Construction(Class("mx-auto size-12 text-slate-400")),
			H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Usage Dashboard Coming Soon")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("The feature usage dashboard with detailed analytics is under development.")),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// Helper rendering functions

func (e *DashboardExtension) renderFeatureTypeFilterTabs(currentType string, counts map[string]int, currentApp *app.App, basePath string) g.Node {
	types := []struct {
		key   string
		label string
	}{
		{"", "All"},
		{"boolean", "Boolean"},
		{"limit", "Limit"},
		{"unlimited", "Unlimited"},
		{"metered", "Metered"},
		{"tiered", "Tiered"},
	}

	var tabs []g.Node
	for _, t := range types {
		isActive := currentType == t.key
		count := counts[t.key]
		if t.key == "" {
			count = 0
			for _, c := range counts {
				count += c
			}
		}

		href := basePath + "/app/" + currentApp.ID.String() + "/billing/features"
		if t.key != "" {
			href += "?type=" + t.key
		}

		tabClass := "px-4 py-2 text-sm font-medium text-gray-500 hover:text-gray-700"
		if isActive {
			tabClass = "px-4 py-2 text-sm font-medium text-violet-600 border-b-2 border-violet-600"
		}
		tabs = append(tabs, A(
			Href(href),
			Class(tabClass),
			g.Text(fmt.Sprintf("%s (%d)", t.label, count)),
		))
	}

	return Div(
		Class("flex gap-1 border-b border-slate-200 dark:border-slate-700"),
		g.Group(tabs),
	)
}

func (e *DashboardExtension) renderFeaturesTable(ctx context.Context, features []*core.Feature, currentApp *app.App, basePath string) g.Node {
	if len(features) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-slate-700 dark:bg-slate-800"),
			lucide.Package(Class("mx-auto size-12 text-slate-400")),
			H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("No features yet")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Create your first feature to start managing plan capabilities.")),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/create"),
				Class("mt-4 inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Feature"),
			),
		)
	}

	var rows []g.Node
	for _, f := range features {
		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-slate-700"),
			Td(Class("px-6 py-4"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+f.ID.String()),
					Class("font-medium text-slate-900 hover:text-violet-600 dark:text-white"),
					g.Text(f.Name),
				),
			),
			Td(Class("px-6 py-4"),
				Span(Class("font-mono text-sm text-slate-600 dark:text-gray-400"), g.Text(f.Key)),
			),
			Td(Class("px-6 py-4"), e.featureTypeBadge(string(f.Type))),
			Td(Class("px-6 py-4"),
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text(f.Unit)),
			),
			Td(Class("px-6 py-4"), e.resetPeriodBadge(string(f.ResetPeriod))),
			Td(Class("px-6 py-4"), e.boolBadge(f.IsPublic)),
			Td(Class("px-6 py-4 text-right"),
				Div(
					Class("flex items-center justify-end gap-2"),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+f.ID.String()),
						Class("text-slate-400 hover:text-violet-600"),
						lucide.Eye(Class("size-4")),
					),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+f.ID.String()+"/edit"),
						Class("text-slate-400 hover:text-violet-600"),
						lucide.Pencil(Class("size-4")),
					),
				),
			),
		))
	}

	return Div(
		Class("overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-slate-700"),
			THead(
				Class("bg-slate-50 dark:bg-slate-900"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Name")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Key")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Type")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Unit")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Reset")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Public")),
					Th(Class("px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500"),
						g.Text("Actions")),
				),
			),
			TBody(
				Class("divide-y divide-slate-200 dark:divide-slate-700"),
				g.Group(rows),
			),
		),
	)
}

func (e *DashboardExtension) renderFeatureTiersTable(tiers []core.FeatureTier) g.Node {
	var rows []g.Node
	for _, t := range tiers {
		upToText := fmt.Sprintf("%d", t.UpTo)
		if t.UpTo == -1 {
			upToText = "Unlimited"
		}

		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-slate-700"),
			Td(Class("px-6 py-4"), g.Text(fmt.Sprintf("%d", t.TierOrder+1))),
			Td(Class("px-6 py-4"), g.Text(upToText)),
			Td(Class("px-6 py-4"), g.Text(t.Value)),
			Td(Class("px-6 py-4"), g.Text(t.Label)),
		))
	}

	return Table(
		Class("min-w-full divide-y divide-slate-200 dark:divide-slate-700"),
		THead(
			Class("bg-slate-50 dark:bg-slate-900"),
			Tr(
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Order")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Up To")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Value")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Label")),
			),
		),
		TBody(
			Class("divide-y divide-slate-200 dark:divide-slate-700"),
			g.Group(rows),
		),
	)
}

func (e *DashboardExtension) renderFeaturePlansTable(ctx context.Context, links []*core.PlanFeatureLink, currentApp *app.App, basePath string) g.Node {
	var rows []g.Node
	for _, link := range links {
		plan, err := e.plugin.planSvc.GetByID(ctx, link.PlanID)
		if err != nil {
			continue
		}

		// Parse value
		valueDisplay := link.Value
		if link.IsBlocked {
			valueDisplay = "Blocked"
		} else if link.Value != "" {
			var val interface{}
			if json.Unmarshal([]byte(link.Value), &val) == nil {
				switch v := val.(type) {
				case bool:
					if v {
						valueDisplay = "Enabled"
					} else {
						valueDisplay = "Disabled"
					}
				case float64:
					if v == -1 {
						valueDisplay = "Unlimited"
					} else {
						valueDisplay = fmt.Sprintf("%.0f", v)
					}
				}
			}
		}

		rows = append(rows, Tr(
			Class("hover:bg-slate-50 dark:hover:bg-slate-700"),
			Td(Class("px-6 py-4"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
					Class("font-medium text-slate-900 hover:text-violet-600 dark:text-white"),
					g.Text(plan.Name),
				),
			),
			Td(Class("px-6 py-4"), g.Text(valueDisplay)),
			Td(Class("px-6 py-4"), e.boolBadge(link.IsHighlighted)),
			Td(Class("px-6 py-4"), e.boolBadge(!link.IsBlocked)),
		))
	}

	return Table(
		Class("min-w-full divide-y divide-slate-200 dark:divide-slate-700"),
		THead(
			Class("bg-slate-50 dark:bg-slate-900"),
			Tr(
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Plan")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Value")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Highlighted")),
				Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"),
					g.Text("Enabled")),
			),
		),
		TBody(
			Class("divide-y divide-slate-200 dark:divide-slate-700"),
			g.Group(rows),
		),
	)
}

// Badge helpers

func (e *DashboardExtension) featureTypeBadge(featureType string) g.Node {
	colors := map[string]string{
		"boolean":   "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
		"limit":     "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300",
		"unlimited": "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
		"metered":   "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
		"tiered":    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-300",
	}

	color := colors[featureType]
	if color == "" {
		color = "bg-gray-100 text-gray-800"
	}

	return Span(
		Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+color),
		g.Text(featureType),
	)
}

func (e *DashboardExtension) resetPeriodBadge(period string) g.Node {
	if period == "" || period == "none" {
		return Span(Class("text-slate-400"), g.Text("Never"))
	}

	labels := map[string]string{
		"daily":          "Daily",
		"weekly":         "Weekly",
		"monthly":        "Monthly",
		"yearly":         "Yearly",
		"billing_period": "Billing Cycle",
	}

	label := labels[period]
	if label == "" {
		label = period
	}

	return Span(
		Class("inline-flex items-center rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-800 dark:bg-slate-700 dark:text-slate-300"),
		g.Text(label),
	)
}

func (e *DashboardExtension) boolBadge(value bool) g.Node {
	if value {
		return Span(
			Class("inline-flex items-center gap-1 text-green-600"),
			lucide.Check(Class("size-4")),
			g.Text("Yes"),
		)
	}
	return Span(
		Class("inline-flex items-center gap-1 text-slate-400"),
		lucide.X(Class("size-4")),
		g.Text("No"),
	)
}

func (e *DashboardExtension) featureIcon(featureType core.FeatureType) g.Node {
	iconClass := "size-10 rounded-lg bg-slate-100 p-2 dark:bg-slate-700"

	switch featureType {
	case core.FeatureTypeBoolean:
		return Div(Class(iconClass), lucide.ToggleLeft(Class("size-full text-blue-600")))
	case core.FeatureTypeLimit:
		return Div(Class(iconClass), lucide.Gauge(Class("size-full text-amber-600")))
	case core.FeatureTypeUnlimited:
		return Div(Class(iconClass), lucide.Infinity(Class("size-full text-green-600")))
	case core.FeatureTypeMetered:
		return Div(Class(iconClass), lucide.Activity(Class("size-full text-purple-600")))
	case core.FeatureTypeTiered:
		return Div(Class(iconClass), lucide.Layers(Class("size-full text-indigo-600")))
	default:
		return Div(Class(iconClass), lucide.Package(Class("size-full text-slate-600")))
	}
}

func (e *DashboardExtension) detailRow(label string, value g.Node) g.Node {
	return Div(
		Class("flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-700 last:border-0"),
		Dt(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(label)),
		Dd(Class("text-sm text-slate-900 dark:text-white"), value),
	)
}

// ServeFeatureCreatePage renders the feature creation form
func (e *DashboardExtension) ServeFeatureCreatePage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	content := Div(
		Class("space-y-6"),

		// Back button and header
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Features"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Create Feature")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Define a new feature that can be linked to subscription plans")),

		// Form
		e.renderFeatureForm(currentApp, basePath, nil, false),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeFeatureEditPage renders the feature edit form
func (e *DashboardExtension) ServeFeatureEditPage(ctx *router.PageContext) (g.Node, error) {
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

	reqCtx := ctx.Request.Context()
	basePath := e.baseUIPath

	feature, err := e.plugin.featureSvc.GetByID(reqCtx, featureID)
	if err != nil {
		return nil, fmt.Errorf("feature not found")
	}

	content := Div(
		Class("space-y-6"),

		// Back button and header
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Feature"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Edit Feature")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Update the feature definition")),

		// Form
		e.renderFeatureForm(currentApp, basePath, feature, true),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// HandleCreateFeature handles the feature creation form submission
func (e *DashboardExtension) HandleCreateFeature(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Parse form
	key := ctx.Request.FormValue("key")
	name := ctx.Request.FormValue("name")
	description := ctx.Request.FormValue("description")
	featureType := ctx.Request.FormValue("type")
	unit := ctx.Request.FormValue("unit")
	resetPeriod := ctx.Request.FormValue("resetPeriod")
	icon := ctx.Request.FormValue("icon")
	isPublic := ctx.Request.FormValue("isPublic") == "on"
	displayOrder, _ := strconv.Atoi(ctx.Request.FormValue("displayOrder"))

	req := &core.CreateFeatureRequest{
		Key:          key,
		Name:         name,
		Description:  description,
		Type:         core.FeatureType(featureType),
		Unit:         unit,
		ResetPeriod:  core.ResetPeriod(resetPeriod),
		IsPublic:     isPublic,
		DisplayOrder: displayOrder,
		Icon:         icon,
	}

	feature, err := e.plugin.featureSvc.Create(reqCtx, currentApp.ID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/create?error="+err.Error(), http.StatusFound)
		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+feature.ID.String(), http.StatusFound)
	return nil, nil
}

// HandleUpdateFeature handles the feature update form submission
func (e *DashboardExtension) HandleUpdateFeature(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
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

	// Parse form
	name := ctx.Request.FormValue("name")
	description := ctx.Request.FormValue("description")
	unit := ctx.Request.FormValue("unit")
	resetPeriod := ctx.Request.FormValue("resetPeriod")
	icon := ctx.Request.FormValue("icon")
	isPublic := ctx.Request.FormValue("isPublic") == "on"
	displayOrder, _ := strconv.Atoi(ctx.Request.FormValue("displayOrder"))

	rp := core.ResetPeriod(resetPeriod)
	req := &core.UpdateFeatureRequest{
		Name:         &name,
		Description:  &description,
		Unit:         &unit,
		ResetPeriod:  &rp,
		IsPublic:     &isPublic,
		DisplayOrder: &displayOrder,
		Icon:         &icon,
	}

	_, err = e.plugin.featureSvc.Update(reqCtx, featureID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"/edit?error="+err.Error(), http.StatusFound)
		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr, http.StatusFound)
	return nil, nil
}

// HandleDeleteFeature handles feature deletion
func (e *DashboardExtension) HandleDeleteFeature(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
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

	err = e.plugin.featureSvc.Delete(reqCtx, featureID)
	if err != nil {
		// URL encode the error message
		errorMsg := err.Error()
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features/"+featureIDStr+"?error="+errorMsg, http.StatusFound)
		return nil, nil
	}

	// Success - redirect to features list with success message
	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/features?success=Feature+deleted+successfully", http.StatusFound)
	return nil, nil
}

// ServePlanFeaturesPage renders the plan features management page
func (e *DashboardExtension) ServePlanFeaturesPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	planIDStr := ctx.Param("id")
	planID, err := xid.FromString(planIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	basePath := e.baseUIPath

	plan, err := e.plugin.planSvc.GetByID(reqCtx, planID)
	if err != nil {
		return nil, errs.NotFound("Plan not found")
	}

	// Get all features for this app
	allFeatures, _, _ := e.plugin.featureSvc.List(reqCtx, currentApp.ID, "", false, 1, 1000)

	// Get features linked to this plan
	linkedFeatures, _ := e.plugin.featureSvc.GetPlanFeatures(reqCtx, planID)

	// Create a map of linked feature IDs
	linkedMap := make(map[string]*core.PlanFeatureLink)
	for _, link := range linkedFeatures {
		linkedMap[link.FeatureID.String()] = link
	}

	content := Div(
		Class("space-y-6"),

		// Breadcrumb
		Nav(
			Class("flex"),
			Ol(
				Class("inline-flex items-center space-x-1 md:space-x-3"),
				Li(Class("inline-flex items-center"),
					A(Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
						Class("text-sm font-medium text-gray-500 hover:text-violet-600"),
						g.Text("Plans"))),
				Li(Class("flex items-center"),
					lucide.ChevronRight(Class("size-4 text-gray-400 mx-1")),
					A(Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr),
						Class("text-sm font-medium text-gray-500 hover:text-violet-600"),
						g.Text(plan.Name))),
				Li(Class("flex items-center"),
					lucide.ChevronRight(Class("size-4 text-gray-400 mx-1")),
					Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
						g.Text("Features"))),
			),
		),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Manage Plan Features")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text(fmt.Sprintf("Configure features for %s", plan.Name))),
			),
		),

		// Features configuration
		Div(
			Class("rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800"),

			// Available features
			Div(
				Class("p-6 border-b border-slate-200 dark:border-slate-700"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Available Features")),
				g.If(len(allFeatures) == 0,
					P(Class("text-slate-500"), g.Text("No features defined. Create features first.")),
				),
				g.If(len(allFeatures) > 0,
					e.renderPlanFeatureConfigList(reqCtx, allFeatures, linkedMap, plan, currentApp, basePath),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// HandleLinkFeatureToPlan handles linking a feature to a plan
func (e *DashboardExtension) HandleLinkFeatureToPlan(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	planIDStr := ctx.Param("id")
	planID, err := xid.FromString(planIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	basePath := e.baseUIPath

	// Parse form
	featureIDStr := ctx.Request.FormValue("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features?error=Invalid feature ID", http.StatusFound)
		return nil, nil
	}

	value := ctx.Request.FormValue("value")
	isBlocked := ctx.Request.FormValue("isBlocked") == "on"
	isHighlighted := ctx.Request.FormValue("isHighlighted") == "on"

	req := &core.LinkFeatureRequest{
		FeatureID:     featureID,
		Value:         value,
		IsBlocked:     isBlocked,
		IsHighlighted: isHighlighted,
	}

	_, err = e.plugin.featureSvc.LinkToPlan(reqCtx, planID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features?error="+err.Error(), http.StatusFound)
		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features", http.StatusFound)
	return nil, nil
}

// HandleUnlinkFeatureFromPlan handles unlinking a feature from a plan
func (e *DashboardExtension) HandleUnlinkFeatureFromPlan(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	planIDStr := ctx.Param("id")
	planID, err := xid.FromString(planIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	featureIDStr := ctx.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid feature ID")
	}

	basePath := e.baseUIPath

	err = e.plugin.featureSvc.UnlinkFromPlan(reqCtx, planID, featureID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features?error="+err.Error(), http.StatusFound)
		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features", http.StatusFound)
	return nil, nil
}

// HandleUpdatePlanFeatureLink handles updating a feature-plan link
func (e *DashboardExtension) HandleUpdatePlanFeatureLink(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	planIDStr := ctx.Param("id")
	planID, err := xid.FromString(planIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	featureIDStr := ctx.Param("featureId")
	featureID, err := xid.FromString(featureIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid feature ID")
	}

	basePath := e.baseUIPath

	// Parse form
	value := ctx.Request.FormValue("value")
	isBlocked := ctx.Request.FormValue("isBlocked") == "on"
	isHighlighted := ctx.Request.FormValue("isHighlighted") == "on"

	req := &core.UpdateLinkRequest{
		Value:         &value,
		IsBlocked:     &isBlocked,
		IsHighlighted: &isHighlighted,
	}

	_, err = e.plugin.featureSvc.UpdatePlanLink(reqCtx, planID, featureID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features?error="+err.Error(), http.StatusFound)
		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+planIDStr+"/features", http.StatusFound)
	return nil, nil
}

// Form rendering helpers

func (e *DashboardExtension) renderFeatureForm(currentApp *app.App, basePath string, feature *core.Feature, isEdit bool) g.Node {
	formAction := basePath + "/app/" + currentApp.ID.String() + "/billing/features/create"
	if isEdit && feature != nil {
		formAction = basePath + "/app/" + currentApp.ID.String() + "/billing/features/" + feature.ID.String() + "/update"
	}

	// Default values
	key := ""
	name := ""
	description := ""
	featureType := "boolean"
	unit := ""
	resetPeriod := "none"
	icon := ""
	isPublic := true
	displayOrder := 0

	if feature != nil {
		key = feature.Key
		name = feature.Name
		description = feature.Description
		featureType = string(feature.Type)
		unit = feature.Unit
		resetPeriod = string(feature.ResetPeriod)
		icon = feature.Icon
		isPublic = feature.IsPublic
		displayOrder = feature.DisplayOrder
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Form(
			Method("POST"),
			Action(formAction),
			ID("feature-form"),
			Class("space-y-6"),

			// Name and Key row (only show key on create)
			g.If(!isEdit,
				Div(
					Class("grid gap-4 md:grid-cols-2"),
					// Name
					Div(
						Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Name")),
						Input(
							Type("text"), Name("name"), ID("name"), Required(),
							Value(name),
							g.Attr("oninput", "generateFeatureKey(this.value)"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("Maximum Team Members"),
						),
					),
					// Key
					Div(
						Label(For("key"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Feature Key")),
						Input(
							Type("text"), Name("key"), ID("key"), Required(),
							Value(key),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("max_team_members"),
							g.Attr("pattern", "[a-z0-9_]+"),
						),
						P(Class("mt-1 text-xs text-slate-500"), g.Text("Auto-generated from name. Lowercase, underscores only.")),
					),
				),
			),

			// Name only (on edit)
			g.If(isEdit,
				Div(
					Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Name")),
					Input(
						Type("text"), Name("name"), ID("name"), Required(),
						Value(name),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Maximum Team Members"),
					),
				),
			),

			// Description
			Div(
				Label(For("description"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Description")),
				Textarea(
					Name("description"), ID("description"), Rows("3"),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					Placeholder("The maximum number of team members allowed"),
					g.Text(description),
				),
			),

			// Type (only on create)
			g.If(!isEdit,
				Div(
					Label(For("type"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Feature Type")),
					Select(
						Name("type"), ID("type"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("boolean"), g.If(featureType == "boolean", g.Attr("selected", "")), g.Text("Boolean (On/Off)")),
						Option(Value("limit"), g.If(featureType == "limit", g.Attr("selected", "")), g.Text("Limit (Numeric Cap)")),
						Option(Value("unlimited"), g.If(featureType == "unlimited", g.Attr("selected", "")), g.Text("Unlimited")),
						Option(Value("metered"), g.If(featureType == "metered", g.Attr("selected", "")), g.Text("Metered (Usage-Based)")),
						Option(Value("tiered"), g.If(featureType == "tiered", g.Attr("selected", "")), g.Text("Tiered (Levels)")),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Type cannot be changed after creation.")),
				),
			),

			// Unit and Reset Period row
			Div(
				Class("grid gap-4 md:grid-cols-2"),
				// Unit
				Div(
					Label(For("unit"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Unit Label")),
					Input(
						Type("text"), Name("unit"), ID("unit"),
						Value(unit),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("members, GB, API calls"),
					),
				),
				// Reset Period
				Div(
					Label(For("resetPeriod"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Usage Reset Period")),
					Select(
						Name("resetPeriod"), ID("resetPeriod"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("none"), g.If(resetPeriod == "none", g.Attr("selected", "")), g.Text("Never")),
						Option(Value("daily"), g.If(resetPeriod == "daily", g.Attr("selected", "")), g.Text("Daily")),
						Option(Value("weekly"), g.If(resetPeriod == "weekly", g.Attr("selected", "")), g.Text("Weekly")),
						Option(Value("monthly"), g.If(resetPeriod == "monthly", g.Attr("selected", "")), g.Text("Monthly")),
						Option(Value("yearly"), g.If(resetPeriod == "yearly", g.Attr("selected", "")), g.Text("Yearly")),
						Option(Value("billing_period"), g.If(resetPeriod == "billing_period", g.Attr("selected", "")), g.Text("Billing Cycle")),
					),
				),
			),

			// Icon and Display Order row
			Div(
				Class("grid gap-4 md:grid-cols-2"),
				// Icon
				Div(
					Label(For("icon"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Icon (optional)")),
					Input(
						Type("text"), Name("icon"), ID("icon"),
						Value(icon),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("users, database, zap"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Lucide icon name for UI display")),
				),
				// Display Order
				Div(
					Label(For("displayOrder"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Order")),
					Input(
						Type("number"), Name("displayOrder"), ID("displayOrder"),
						Value(fmt.Sprintf("%d", displayOrder)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),
			),

			// Public checkbox
			Div(
				Class("flex items-center gap-2"),
				Input(
					Type("checkbox"), Name("isPublic"), ID("isPublic"),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
					g.If(isPublic, g.Attr("checked", "")),
				),
				Label(For("isPublic"), Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Show in public pricing pages")),
			),

			// Actions
			Div(
				Class("flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features"),
					Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-gray-300"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.If(isEdit, g.Text("Save Changes")),
					g.If(!isEdit, g.Text("Create Feature")),
				),
			),
		),

		// JavaScript for auto-generating feature key from name
		g.If(!isEdit,
			Script(g.Raw(`
				function generateFeatureKey(name) {
					const keyField = document.getElementById('key');
					if (keyField && !keyField.dataset.manual) {
						const key = name
							.toLowerCase()
							.trim()
							.replace(/[^\w\s]/g, '')    // Remove special characters except spaces
							.replace(/\s+/g, '_')       // Replace spaces with underscores
							.replace(/_+/g, '_')        // Replace multiple underscores with single
							.replace(/^_|_$/g, '');     // Remove leading/trailing underscores
						keyField.value = key;
					}
				}
				
				// Mark key as manually edited if user modifies it
				document.addEventListener('DOMContentLoaded', function() {
					const keyField = document.getElementById('key');
					if (keyField) {
						keyField.addEventListener('input', function() {
							this.dataset.manual = 'true';
						});
					}
				});
			`)),
		),
	)
}

func (e *DashboardExtension) renderPlanFeatureConfigList(ctx context.Context, features []*core.Feature, linkedMap map[string]*core.PlanFeatureLink, plan *core.Plan, currentApp *app.App, basePath string) g.Node {
	var rows []g.Node

	for _, feature := range features {
		link, isLinked := linkedMap[feature.ID.String()]

		var valueDisplay string
		var valueInput g.Node

		if isLinked && link != nil {
			// Parse value for display
			valueDisplay = link.Value
			if link.IsBlocked {
				valueDisplay = "Blocked"
			}
		}

		// Create value input based on feature type
		switch feature.Type {
		case core.FeatureTypeBoolean:
			checked := false
			if link != nil && link.Value != "" {
				json.Unmarshal([]byte(link.Value), &checked)
			}
			valueInput = Div(
				Class("flex items-center gap-2"),
				Input(
					Type("checkbox"), Name("value"), ID("value-"+feature.ID.String()),
					Value("true"),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
					g.If(checked, g.Attr("checked", "")),
				),
				Label(For("value-"+feature.ID.String()), Class("text-sm text-slate-600"), g.Text("Enabled")),
			)
		case core.FeatureTypeLimit, core.FeatureTypeMetered:
			var limitVal string
			if link != nil && link.Value != "" {
				var val float64
				if json.Unmarshal([]byte(link.Value), &val) == nil {
					limitVal = fmt.Sprintf("%.0f", val)
				}
			}
			valueInput = Input(
				Type("number"), Name("value"),
				Value(limitVal),
				Placeholder("e.g., 10"),
				Class("w-24 rounded-md border-slate-300 text-sm shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
			)
		case core.FeatureTypeUnlimited:
			valueInput = Span(Class("text-sm text-green-600"), g.Text("Unlimited"))
		default:
			valueInput = Input(
				Type("text"), Name("value"),
				Value(valueDisplay),
				Class("w-32 rounded-md border-slate-300 text-sm shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
			)
		}

		row := Tr(
			Class("hover:bg-slate-50 dark:hover:bg-slate-700"),
			// Feature info
			Td(Class("px-4 py-3"),
				Div(
					Class("flex items-center gap-3"),
					e.featureIcon(feature.Type),
					Div(
						Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(feature.Name)),
						Div(Class("text-xs text-slate-500"), g.Text(feature.Key)),
					),
				),
			),
			Td(Class("px-4 py-3"), e.featureTypeBadge(string(feature.Type))),
			// Value/Config
			Td(Class("px-4 py-3"),
				g.If(isLinked,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/features/"+feature.ID.String()+"/update"),
						Class("flex items-center gap-2"),
						valueInput,
						Div(
							Class("flex items-center gap-2 ml-2"),
							Input(
								Type("checkbox"), Name("isHighlighted"),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
								g.If(link != nil && link.IsHighlighted, g.Attr("checked", "")),
							),
							Label(Class("text-xs text-slate-500"), g.Text("Highlight")),
						),
						Button(
							Type("submit"),
							Class("text-xs text-violet-600 hover:text-violet-700"),
							g.Text("Save"),
						),
					),
				),
				g.If(!isLinked,
					Span(Class("text-slate-400 italic text-sm"), g.Text("Not linked")),
				),
			),
			// Actions
			Td(Class("px-4 py-3 text-right"),
				g.If(isLinked,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/features/"+feature.ID.String()+"/unlink"),
						Button(
							Type("submit"),
							Class("text-sm text-red-600 hover:text-red-700"),
							g.Text("Remove"),
						),
					),
				),
				g.If(!isLinked,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/features/link"),
						Input(Type("hidden"), Name("featureId"), Value(feature.ID.String())),
						Input(Type("hidden"), Name("value"), Value(e.getDefaultValue(feature.Type))),
						Button(
							Type("submit"),
							Class("text-sm text-violet-600 hover:text-violet-700"),
							g.Text("+ Add"),
						),
					),
				),
			),
		)

		rows = append(rows, row)
	}

	return Table(
		Class("min-w-full"),
		THead(
			Class("bg-slate-50 dark:bg-slate-900"),
			Tr(
				Th(Class("px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"), g.Text("Feature")),
				Th(Class("px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"), g.Text("Type")),
				Th(Class("px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500"), g.Text("Value/Configuration")),
				Th(Class("px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500"), g.Text("Actions")),
			),
		),
		TBody(
			Class("divide-y divide-slate-200 dark:divide-slate-700"),
			g.Group(rows),
		),
	)
}

func (e *DashboardExtension) getDefaultValue(featureType core.FeatureType) string {
	switch featureType {
	case core.FeatureTypeBoolean:
		return "true"
	case core.FeatureTypeLimit, core.FeatureTypeMetered:
		return "10"
	case core.FeatureTypeUnlimited:
		return "-1"
	default:
		return ""
	}
}
