package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// LogsPage renders the SCIM event logs page.
func LogsPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			Div(
				H1(Class("text-2xl font-bold tracking-tight"), g.Text("SCIM Event Logs")),
				P(Class("text-muted-foreground"), g.Text("Monitor SCIM provisioning and synchronization events")),
			),
			Div(
				Class("flex items-center gap-2"),
				button.Button(
					Div(Class("flex items-center gap-2"), lucide.Download(Class("size-4")), g.Text("Export")),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "exportLogs()")),
				),
				button.Button(
					Div(Class("flex items-center gap-2"), lucide.RefreshCw(Class("size-4")), g.Text("Refresh")),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "loadLogs()")),
				),
			),
		),

		// Alpine.js container
		Div(
			g.Attr("x-data", logsPageData(appID)),
			g.Attr("x-init", "loadLogs()"),

			// Filters
			card.Card(
				card.Content(
					Class("space-y-4"),
					Div(
						Class("flex flex-wrap items-center gap-4"),
						// Event type filter
						Div(
							Class("w-40"),
							Select(
								Class("w-full px-3 py-2 border rounded-md text-sm bg-background"),
								g.Attr("x-model", "filters.eventType"),
								g.Attr("@change", "applyFilters()"),
								Option(Value(""), g.Text("All Events")),
								Option(Value("user.created"), g.Text("User Created")),
								Option(Value("user.updated"), g.Text("User Updated")),
								Option(Value("user.deleted"), g.Text("User Deleted")),
								Option(Value("group.created"), g.Text("Group Created")),
								Option(Value("group.updated"), g.Text("Group Updated")),
								Option(Value("group.deleted"), g.Text("Group Deleted")),
								Option(Value("sync.started"), g.Text("Sync Started")),
								Option(Value("sync.completed"), g.Text("Sync Completed")),
								Option(Value("sync.failed"), g.Text("Sync Failed")),
							),
						),
						// Status filter
						Div(
							Class("w-32"),
							Select(
								Class("w-full px-3 py-2 border rounded-md text-sm bg-background"),
								g.Attr("x-model", "filters.status"),
								g.Attr("@change", "applyFilters()"),
								Option(Value(""), g.Text("All Status")),
								Option(Value("success"), g.Text("Success")),
								Option(Value("error"), g.Text("Error")),
								Option(Value("pending"), g.Text("Pending")),
							),
						),
						// Date range
						Div(
							Class("flex items-center gap-2"),
							input.Input(
								input.WithType("date"),
								input.WithAttrs(
									g.Attr("x-model", "filters.startDate"),
									g.Attr("@change", "applyFilters()"),
									Class("w-36"),
								),
							),
							Span(Class("text-muted-foreground"), g.Text("to")),
							input.Input(
								input.WithType("date"),
								input.WithAttrs(
									g.Attr("x-model", "filters.endDate"),
									g.Attr("@change", "applyFilters()"),
									Class("w-36"),
								),
							),
						),
						// Clear filters
						g.El("template",
							g.Attr("x-if", "hasFilters"),
							button.Button(
								Div(Class("flex items-center gap-1"), lucide.X(Class("size-4")), g.Text("Clear")),
								button.WithVariant("ghost"),
								button.WithAttrs(g.Attr("@click", "clearFilters()")),
							),
						),
					),
				),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
			),

			// Error state
			g.El("template",
				g.Attr("x-if", "error && !loading"),
				card.Card(
					card.Content(
						Class("flex items-center gap-3 text-destructive"),
						lucide.CircleAlert(Class("size-5")),
						Span(g.Attr("x-text", "error")),
					),
				),
			),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-4"),

				// Empty state
				Div(
					g.Attr("x-show", "logs.length === 0"),
					card.Card(
						card.Content(
							Class("text-center py-12"),
							lucide.FileText(Class("size-16 mx-auto text-muted-foreground mb-4")),
							H3(Class("text-lg font-semibold mb-2"), g.Text("No events found")),
							P(Class("text-muted-foreground"), g.Text("Events will appear here as SCIM operations occur")),
						),
					),
				),

				// Logs list
				Div(
					g.Attr("x-show", "logs.length > 0"),
					card.Card(
						Class("overflow-hidden"),
						Table(
							Class("w-full"),
							THead(
								Class("bg-muted/50"),
								Tr(
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("Event")),
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("Status")),
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("Provider")),
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("Resource")),
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("Timestamp")),
									Th(Class("px-4 py-3 text-left text-sm font-medium"), g.Text("")),
								),
							),
							TBody(
								g.El("template",
									g.Attr("x-for", "log in logs"),
									g.Attr(":key", "log.id"),
									Tr(
										Class("border-t hover:bg-muted/30 transition-colors"),
										// Event type
										Td(
											Class("px-4 py-3"),
											Div(
												Class("flex items-center gap-2"),
												Div(
													Class("p-1.5 rounded-full"),
													g.Attr(":class", "getEventTypeClass(log.eventType)"),
													lucide.Activity(Class("size-3")),
												),
												Span(
													Class("text-sm font-medium"),
													g.Attr("x-text", "formatEventType(log.eventType)"),
												),
											),
										),
										// Status
										Td(
											Class("px-4 py-3"),
											Span(
												Class("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"),
												g.Attr(":class", "log.status === 'success' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : log.status === 'error' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'"),
												g.Attr("x-text", "log.status"),
											),
										),
										// Provider
										Td(
											Class("px-4 py-3 text-sm text-muted-foreground"),
											g.Attr("x-text", "log.provider || '-'"),
										),
										// Resource
										Td(
											Class("px-4 py-3 text-sm"),
											Span(g.Attr("x-text", "log.resource || '-'")),
											Span(
												Class("text-muted-foreground ml-1"),
												g.Attr("x-show", "log.resourceId"),
												g.Attr("x-text", "'(' + log.resourceId + ')'")),
										),
										// Timestamp
										Td(
											Class("px-4 py-3 text-sm text-muted-foreground"),
											g.Attr("x-text", "formatTimestamp(log.timestamp)"),
										),
										// Actions
										Td(
											Class("px-4 py-3"),
											button.Button(
												lucide.Eye(Class("size-4")),
												button.WithVariant("ghost"),
												button.WithSize("icon"),
												button.WithAttrs(g.Attr("@click", "showDetails(log)")),
											),
										),
									),
								),
							),
						),
					),
				),

				// Pagination
				Div(
					g.Attr("x-show", "pagination.totalPages > 1"),
					Class("flex items-center justify-between pt-4"),
					Div(
						Class("text-sm text-muted-foreground"),
						g.Text("Showing "),
						Span(g.Attr("x-text", "((pagination.page - 1) * pagination.pageSize) + 1")),
						g.Text(" to "),
						Span(g.Attr("x-text", "Math.min(pagination.page * pagination.pageSize, pagination.total)")),
						g.Text(" of "),
						Span(g.Attr("x-text", "pagination.total")),
						g.Text(" events"),
					),
					Div(
						Class("flex items-center gap-2"),
						button.Button(
							Div(Class("flex items-center gap-1"), lucide.ChevronLeft(Class("size-4")), g.Text("Previous")),
							button.WithVariant("outline"),
							button.WithAttrs(
								g.Attr(":disabled", "pagination.page <= 1"),
								g.Attr("@click", "goToPage(pagination.page - 1)"),
							),
						),
						button.Button(
							Div(Class("flex items-center gap-1"), g.Text("Next"), lucide.ChevronRight(Class("size-4"))),
							button.WithVariant("outline"),
							button.WithAttrs(
								g.Attr(":disabled", "pagination.page >= pagination.totalPages"),
								g.Attr("@click", "goToPage(pagination.page + 1)"),
							),
						),
					),
				),
			),

			// Details modal
			logDetailsModal(),
		),
	)
}

// logDetailsModal renders the log details modal.
func logDetailsModal() g.Node {
	return g.El("template",
		g.Attr("x-if", "selectedLog"),
		Div(
			Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
			g.Attr("@click.self", "selectedLog = null"),
			card.Card(
				Class("w-full max-w-lg mx-4 max-h-[80vh] overflow-y-auto"),
				card.Header(
					Class("flex items-center justify-between"),
					Div(
						Span(Class("text-lg font-semibold"), g.Text("Event Details")),
						Div(Class("text-sm text-muted-foreground"), g.Attr("x-text", "formatTimestamp(selectedLog.timestamp)")),
					),
					button.Button(
						lucide.X(Class("size-4")),
						button.WithVariant("ghost"),
						button.WithSize("icon"),
						button.WithAttrs(g.Attr("@click", "selectedLog = null")),
					),
				),
				card.Content(
					Class("space-y-4"),
					// Event type
					Div(
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Event Type")),
						Div(Class("font-medium"), g.Attr("x-text", "formatEventType(selectedLog.eventType)")),
					),
					// Status
					Div(
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Status")),
						Div(
							Span(
								Class("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"),
								g.Attr(":class", "selectedLog.status === 'success' ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'"),
								g.Attr("x-text", "selectedLog.status"),
							),
						),
					),
					// Provider
					Div(
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Provider")),
						Div(Class("font-medium"), g.Attr("x-text", "selectedLog.provider || '-'")),
					),
					// Resource
					Div(
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Resource")),
						Div(Class("font-medium"), g.Attr("x-text", "(selectedLog.resource || '-') + (selectedLog.resourceId ? ' (' + selectedLog.resourceId + ')' : '')")),
					),
					// Details
					Div(
						g.Attr("x-show", "selectedLog.details"),
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Details")),
						Div(
							Class("p-3 bg-muted rounded-lg text-sm font-mono whitespace-pre-wrap"),
							g.Attr("x-text", "selectedLog.details"),
						),
					),
					// IP Address
					Div(
						g.Attr("x-show", "selectedLog.ipAddress"),
						Label(Class("text-sm font-medium text-muted-foreground"), g.Text("IP Address")),
						Div(Class("font-medium"), g.Attr("x-text", "selectedLog.ipAddress")),
					),
				),
			),
		),
	)
}

// logsPageData returns the Alpine.js data for the logs page.
func logsPageData(appID string) string {
	return fmt.Sprintf(`{
		logs: [],
		pagination: {
			page: 1,
			pageSize: 25,
			total: 0,
			totalPages: 0
		},
		filters: {
			eventType: '',
			status: '',
			startDate: '',
			endDate: ''
		},
		loading: true,
		error: null,
		selectedLog: null,
		
		get hasFilters() {
			return this.filters.eventType || this.filters.status || this.filters.startDate || this.filters.endDate;
		},
		
		async loadLogs() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('scim.getLogs', {
					appId: '%s',
					page: this.pagination.page,
					pageSize: this.pagination.pageSize,
					eventType: this.filters.eventType,
					status: this.filters.status,
					startDate: this.filters.startDate,
					endDate: this.filters.endDate
				});
				this.logs = result.logs || [];
				this.pagination.total = result.total || 0;
				this.pagination.totalPages = result.totalPages || 0;
			} catch (err) {
				console.error('Failed to load logs:', err);
				this.error = err.message || 'Failed to load logs';
			} finally {
				this.loading = false;
			}
		},
		
		applyFilters() {
			this.pagination.page = 1;
			this.loadLogs();
		},
		
		clearFilters() {
			this.filters = {
				eventType: '',
				status: '',
				startDate: '',
				endDate: ''
			};
			this.applyFilters();
		},
		
		goToPage(page) {
			if (page >= 1 && page <= this.pagination.totalPages) {
				this.pagination.page = page;
				this.loadLogs();
			}
		},
		
		showDetails(log) {
			this.selectedLog = log;
		},
		
		exportLogs() {
			alert('Export functionality coming soon');
		},
		
		formatEventType(type) {
			if (!type) return '-';
			return type.split('.').map(s => s.charAt(0).toUpperCase() + s.slice(1)).join(' ');
		},
		
		getEventTypeClass(type) {
			if (!type) return 'bg-gray-100 dark:bg-gray-800';
			if (type.includes('created')) return 'bg-emerald-100 dark:bg-emerald-900/30';
			if (type.includes('updated')) return 'bg-blue-100 dark:bg-blue-900/30';
			if (type.includes('deleted')) return 'bg-red-100 dark:bg-red-900/30';
			if (type.includes('failed')) return 'bg-red-100 dark:bg-red-900/30';
			return 'bg-violet-100 dark:bg-violet-900/30';
		},
		
		formatTimestamp(timestamp) {
			if (!timestamp) return '-';
			return new Date(timestamp).toLocaleString();
		}
	}`, appID)
}
