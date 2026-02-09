package scim

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Monitoring Handlers

// ServeMonitoringPage renders the main SCIM monitoring dashboard.
func (e *DashboardExtension) ServeMonitoringPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()

	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, e.baseUIPath+"/login", http.StatusFound)

		return nil, nil
	}

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	content := e.renderMonitoringPageContent(reqCtx, currentApp, orgID)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderMonitoringPageContent renders the monitoring page content.
func (e *DashboardExtension) renderMonitoringPageContent(reqCtx context.Context, currentApp any, orgID *xid.ID) g.Node {
	basePath := e.getBasePath()
	app := currentApp.(*app.App)
	appID := app.ID

	// Fetch sync status and stats
	stats, err := e.plugin.service.GetDashboardStats(reqCtx, appID, orgID)
	if err != nil {
		return alertBox("error", "Error", "Failed to load monitoring data: "+err.Error())
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("SCIM Monitoring")),
			P(Class("mt-1 text-slate-600 dark:text-gray-400"),
				g.Text("Monitor synchronization status and provisioning events")),
		),

		// Stats Cards
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"),
			statsCard(
				"Total Syncs",
				strconv.Itoa(stats.TotalSyncs),
				"All time",
				lucide.Activity(Class("size-5 text-violet-600 dark:text-violet-400")),
			),
			statsCard(
				"Success Rate",
				fmt.Sprintf("%.1f%%", stats.SuccessRate),
				"Last 30 days",
				lucide.TrendingUp(Class("size-5 text-green-600 dark:text-green-400")),
			),
			statsCard(
				"Failed Syncs",
				strconv.Itoa(stats.FailedSyncs),
				"Requires attention",
				lucide.Info(Class("size-5 text-red-600 dark:text-red-400")),
			),
			statsCard(
				"Last Sync",
				stats.LastSyncTime,
				stats.LastSyncStatus,
				lucide.Clock(Class("size-5 text-blue-600 dark:text-blue-400")),
			),
		),

		// Quick Actions
		Div(
			Class("flex gap-3"),
			A(
				Href(fmt.Sprintf("%s/app/%s/settings/scim-monitoring/logs", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.FileText(Class("size-4")),
				g.Text("View Logs"),
			),
			A(
				Href(fmt.Sprintf("%s/app/%s/settings/scim-monitoring/stats", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.TrendingUp(Class("size-4")),
				g.Text("View Statistics"),
			),
			A(
				Href(fmt.Sprintf("%s/app/%s/settings/scim-monitoring/export", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.Download(Class("size-4")),
				g.Text("Export Logs"),
			),
		),

		// Recent Activity
		e.renderRecentActivitySection(reqCtx, &appID, orgID),

		// Failed Operations
		g.If(stats.FailedSyncs > 0,
			e.renderFailedOperationsSection(reqCtx, &appID, orgID),
		),
	)
}

// renderRecentActivitySection renders the recent activity section.
func (e *DashboardExtension) renderRecentActivitySection(ctx context.Context, appID *xid.ID, orgID *xid.ID) g.Node {
	// Fetch recent events
	events, err := e.plugin.service.GetRecentActivity(ctx, *appID, orgID, 10)
	if err != nil {
		return alertBox("error", "Error", "Failed to load recent activity")
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("border-b border-slate-200 p-6 dark:border-gray-800"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
				g.Text("Recent Activity")),
		),
		g.If(len(events) == 0,
			Div(
				Class("p-6"),
				emptyState(
					lucide.Activity(Class("size-12 text-slate-400")),
					"No Recent Activity",
					"SCIM synchronization events will appear here",
					"",
					"",
				),
			),
		),
		g.If(len(events) > 0,
			Div(
				Class("overflow-x-auto"),
				Table(
					Class("w-full"),
					THead(
						Class("bg-slate-50 dark:bg-gray-800/50"),
						Tr(
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Event")),
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Resource")),
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Direction")),
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Duration")),
							Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Time")),
						),
					),
					TBody(
						g.Group(e.renderEventRows(events)),
					),
				),
			),
		),
	)
}

// renderFailedOperationsSection renders the failed operations section.
func (e *DashboardExtension) renderFailedOperationsSection(ctx context.Context, appID *xid.ID, orgID *xid.ID) g.Node {
	// Fetch failed events
	failedEvents, err := e.plugin.service.GetFailedEvents(ctx, *appID, orgID, 5)
	if err != nil {
		return g.Raw("") // Silent fail
	}

	return Div(
		Class("rounded-lg border border-red-200 bg-red-50 shadow-sm dark:border-red-800 dark:bg-red-900/20"),
		Div(
			Class("border-b border-red-200 p-6 dark:border-red-800"),
			Div(
				Class("flex items-center gap-3"),
				lucide.Info(Class("size-5 text-red-600 dark:text-red-400")),
				H2(Class("text-xl font-semibold text-red-900 dark:text-red-300"),
					g.Text("Failed Operations")),
			),
			P(Class("mt-1 text-sm text-red-700 dark:text-red-400"),
				g.Text("These operations require your attention")),
		),
		Div(
			Class("p-6 space-y-3"),
			g.Group(e.renderFailedEventCards(failedEvents)),
		),
	)
}

// renderEventRows renders event table rows.
func (e *DashboardExtension) renderEventRows(events []*SCIMSyncEvent) []g.Node {
	rows := make([]g.Node, len(events))
	for i, event := range events {
		rows[i] = syncEventRow(event)
	}

	return rows
}

// renderFailedEventCards renders failed event cards.
func (e *DashboardExtension) renderFailedEventCards(events []*SCIMSyncEvent) []g.Node {
	cards := make([]g.Node, len(events))
	for i, event := range events {
		cards[i] = Div(
			Class("rounded-lg bg-white border border-red-200 p-4 dark:bg-gray-900 dark:border-red-800"),
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex-1"),
					Div(
						Class("flex items-center gap-2"),
						statusIcon(event.Status),
						Span(Class("font-medium text-slate-900 dark:text-white"), g.Text(event.EventType)),
					),
					P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Textf("%s - %s", event.ResourceType, formatRelativeTime(event.CreatedAt))),
					g.If(event.ErrorMessage != nil,
						P(Class("mt-2 text-xs text-red-600 dark:text-red-400 font-mono"),
							g.Text(*event.ErrorMessage)),
					),
				),
				Button(
					Type("button"),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
					g.Text("Retry"),
				),
			),
		)
	}

	return cards
}

// ServeLogsPage renders the SCIM event logs page.
func (e *DashboardExtension) ServeLogsPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()

	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, e.baseUIPath+"/login", http.StatusFound)

		return nil, nil
	}

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	// Parse query parameters
	page := 1

	if pageStr := ctx.Request.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	statusFilter := ctx.Request.URL.Query().Get("status")
	eventTypeFilter := ctx.Request.URL.Query().Get("event_type")

	content := e.renderLogsPageContent(reqCtx, currentApp, orgID, page, statusFilter, eventTypeFilter)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderLogsPageContent renders the logs page content.
func (e *DashboardExtension) renderLogsPageContent(reqCtx context.Context, currentApp any, orgID *xid.ID, page int, statusFilter, eventTypeFilter string) g.Node {
	basePath := e.getBasePath()
	app := currentApp.(*app.App)
	appID := app.ID

	// Fetch logs
	perPage := 50

	events, total, err := e.plugin.service.GetSyncLogs(reqCtx, appID, orgID, page, perPage, statusFilter, eventTypeFilter)
	if err != nil {
		return alertBox("error", "Error", "Failed to load logs: "+err.Error())
	}

	totalPages := (total + perPage - 1) / perPage

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("SCIM Event Logs")),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Textf("Showing %d-%d of %d events", (page-1)*perPage+1, min(page*perPage, total), total)),
			),
			A(
				Href(fmt.Sprintf("%s/app/%s/settings/scim-monitoring", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Monitoring"),
			),
		),

		// Filters
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("GET"),
				Class("flex gap-3"),
				Div(
					Class("flex-1"),
					Label(
						For("status"),
						Class("block text-xs font-medium text-slate-700 dark:text-gray-300 mb-1"),
						g.Text("Status"),
					),
					Select(
						Name("status"),
						ID("status"),
						Class("block w-full rounded-md border-slate-300 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value(""), g.Text("All"), g.If(statusFilter == "", g.Attr("selected", ""))),
						Option(Value("success"), g.Text("Success"), g.If(statusFilter == "success", g.Attr("selected", ""))),
						Option(Value("failed"), g.Text("Failed"), g.If(statusFilter == "failed", g.Attr("selected", ""))),
						Option(Value("pending"), g.Text("Pending"), g.If(statusFilter == "pending", g.Attr("selected", ""))),
					),
				),
				Div(
					Class("flex-1"),
					Label(
						For("event_type"),
						Class("block text-xs font-medium text-slate-700 dark:text-gray-300 mb-1"),
						g.Text("Event Type"),
					),
					Select(
						Name("event_type"),
						ID("event_type"),
						Class("block w-full rounded-md border-slate-300 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value(""), g.Text("All"), g.If(eventTypeFilter == "", g.Attr("selected", ""))),
						Option(Value("user_create"), g.Text("User Create"), g.If(eventTypeFilter == "user_create", g.Attr("selected", ""))),
						Option(Value("user_update"), g.Text("User Update"), g.If(eventTypeFilter == "user_update", g.Attr("selected", ""))),
						Option(Value("user_delete"), g.Text("User Delete"), g.If(eventTypeFilter == "user_delete", g.Attr("selected", ""))),
						Option(Value("group_create"), g.Text("Group Create"), g.If(eventTypeFilter == "group_create", g.Attr("selected", ""))),
						Option(Value("group_update"), g.Text("Group Update"), g.If(eventTypeFilter == "group_update", g.Attr("selected", ""))),
						Option(Value("group_delete"), g.Text("Group Delete"), g.If(eventTypeFilter == "group_delete", g.Attr("selected", ""))),
					),
				),
				Div(
					Class("flex items-end"),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Filter"),
					),
				),
			),
		),

		// Logs Table
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm overflow-hidden dark:border-gray-800 dark:bg-gray-900"),
			g.If(len(events) == 0,
				Div(
					Class("p-6"),
					emptyState(
						lucide.FileText(Class("size-12 text-slate-400")),
						"No Logs Found",
						"No SCIM events match your filters",
						"",
						"",
					),
				),
			),
			g.If(len(events) > 0,
				Div(
					Class("overflow-x-auto"),
					Table(
						Class("w-full"),
						THead(
							Class("bg-slate-50 dark:bg-gray-800/50"),
							Tr(
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Event")),
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Resource")),
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Direction")),
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Duration")),
								Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Time")),
							),
						),
						TBody(
							g.Group(e.renderEventRows(events)),
						),
					),
				),
			),
		),

		// Pagination
		pagination(page, totalPages, fmt.Sprintf("%s/app/%s/settings/scim-monitoring/logs", basePath, appID.String())),
	)
}

// ServeStatsPage renders the SCIM statistics page.
func (e *DashboardExtension) ServeStatsPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()

	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, e.baseUIPath+"/login", http.StatusFound)

		return nil, nil
	}

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	content := e.renderStatsPageContent(reqCtx, currentApp, orgID)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderStatsPageContent renders the statistics page content.
func (e *DashboardExtension) renderStatsPageContent(reqCtx context.Context, currentApp any, orgID *xid.ID) g.Node {
	basePath := e.getBasePath()
	app := currentApp.(*app.App)
	appID := app.ID

	// Fetch statistics
	stats, err := e.plugin.service.GetDetailedStats(reqCtx, appID, orgID)
	if err != nil {
		return alertBox("error", "Error", "Failed to load statistics: "+err.Error())
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("SCIM Statistics")),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Text("Analytics and metrics for SCIM provisioning")),
			),
			A(
				Href(fmt.Sprintf("%s/app/%s/settings/scim-monitoring", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Monitoring"),
			),
		),

		// Summary Stats
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"),
			statsCard("Total Operations", strconv.Itoa(stats.TotalOperations), "All time", lucide.Activity(Class("size-5"))),
			statsCard("Success Rate", fmt.Sprintf("%.1f%%", stats.SuccessRate), "", lucide.TrendingUp(Class("size-5"))),
			statsCard("Avg Duration", fmt.Sprintf("%dms", stats.AvgDuration), "Per operation", lucide.Clock(Class("size-5"))),
			statsCard("Total Errors", strconv.Itoa(stats.TotalErrors), "", lucide.X(Class("size-5"))),
		),

		// Operations by Type
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Operations by Type")),
			Div(
				Class("space-y-3"),
				g.Group(e.renderOperationTypeStats(stats.OperationsByType)),
			),
		),

		// Operations by Status
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Operations by Status")),
			Div(
				Class("space-y-3"),
				g.Group(e.renderStatusStats(stats.OperationsByStatus)),
			),
		),
	)
}

// renderOperationTypeStats renders operation type statistics.
func (e *DashboardExtension) renderOperationTypeStats(stats map[string]int) []g.Node {
	items := make([]g.Node, 0, len(stats))

	total := 0
	for _, count := range stats {
		total += count
	}

	for opType, count := range stats {
		percentage := 0.0
		if total > 0 {
			percentage = float64(count) / float64(total) * 100
		}

		items = append(items, Div(
			Class("flex items-center justify-between"),
			Span(Class("text-sm text-slate-700 dark:text-gray-300"), g.Text(opType)),
			Div(
				Class("flex items-center gap-2"),
				Div(
					Class("w-32 bg-slate-200 rounded-full h-2 dark:bg-gray-700"),
					Div(
						Class("bg-violet-600 h-2 rounded-full"),
						Style(fmt.Sprintf("width: %.1f%%", percentage)),
					),
				),
				Span(Class("text-sm font-medium text-slate-900 dark:text-white w-16 text-right"),
					g.Textf("%d (%.0f%%)", count, percentage)),
			),
		))
	}

	return items
}

// renderStatusStats renders status statistics.
func (e *DashboardExtension) renderStatusStats(stats map[string]int) []g.Node {
	items := make([]g.Node, 0, len(stats))

	total := 0
	for _, count := range stats {
		total += count
	}

	for status, count := range stats {
		percentage := 0.0
		if total > 0 {
			percentage = float64(count) / float64(total) * 100
		}

		colorClass := "bg-gray-600"

		switch status {
		case "success":
			colorClass = "bg-green-600"
		case "failed":
			colorClass = "bg-red-600"
		case "pending":
			colorClass = "bg-yellow-600"
		}

		items = append(items, Div(
			Class("flex items-center justify-between"),
			Span(Class("text-sm text-slate-700 dark:text-gray-300"), g.Text(status)),
			Div(
				Class("flex items-center gap-2"),
				Div(
					Class("w-32 bg-slate-200 rounded-full h-2 dark:bg-gray-700"),
					Div(
						Class(colorClass+" h-2 rounded-full"),
						Style(fmt.Sprintf("width: %.1f%%", percentage)),
					),
				),
				Span(Class("text-sm font-medium text-slate-900 dark:text-white w-16 text-right"),
					g.Textf("%d (%.0f%%)", count, percentage)),
			),
		))
	}

	return items
}

// HandleExportLogs handles log export.
func (e *DashboardExtension) HandleExportLogs(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	_ = reqCtx // Unused in this handler

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.InternalServerError("Invalid app context", nil)
	}

	orgID, _ := e.getOrgFromContext(ctx)

	format := ctx.Request.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Fetch all logs
	events, _, err := e.plugin.service.GetSyncLogs(reqCtx, currentApp.ID, orgID, 1, 10000, "", "")
	if err != nil {
		return nil, errs.InternalServerError("Failed to fetch logs", nil)
	}

	if format == "json" {
		ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		ctx.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=scim-logs-%s.json", time.Now().Format("2006-01-02")))
		json.NewEncoder(ctx.ResponseWriter).Encode(events)

		return nil, nil
	}

	// Default to CSV
	ctx.ResponseWriter.Header().Set("Content-Type", "text/csv")
	ctx.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=scim-logs-%s.csv", time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(ctx.ResponseWriter)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Timestamp", "Event Type", "Resource Type", "Status", "Direction", "Duration (ms)", "Error"})

	// Write data
	for _, event := range events {
		errorMsg := ""
		if event.ErrorMessage != nil {
			errorMsg = *event.ErrorMessage
		}

		writer.Write([]string{
			event.CreatedAt.Format(time.RFC3339),
			event.EventType,
			event.ResourceType,
			event.Status,
			event.Direction,
			strconv.FormatInt(event.Duration, 10),
			errorMsg,
		})
	}

	return nil, nil
}
