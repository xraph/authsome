package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/table"
)

// SessionsListPage renders the sessions list page with dynamic data loading
func SessionsListPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Page header
		PageHeader(
			"Session Management",
			"Monitor and manage active sessions across all devices",
			RefreshButton("loadSessions()"),
		),

		// Dynamic content with Alpine.js
		Div(
			g.Attr("x-data", sessionsListData(appID)),
			g.Attr("x-init", "loadSessions()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

			// Error state
			ErrorMessage("error && !loading"),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Stats cards
				Div(
					Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-4"),
					StatsCard("Total Sessions", "stats.totalSessions", "blue", lucide.Layers(Class("size-5"))),
					StatsCard("Active Sessions", "stats.activeCount", "emerald", lucide.Activity(Class("size-5"))),
					StatsCard("Mobile Devices", "stats.mobileCount", "violet", lucide.Smartphone(Class("size-5"))),
					StatsCard("Unique Users", "stats.uniqueUsers", "amber", lucide.Users(Class("size-5"))),
				),

				// Filters bar
				card.Card(
					card.Content(
						Class("p-4"),
						Div(
							Class("flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"),

							// Status filter tabs
							Div(
								Class("flex items-center gap-2 bg-muted rounded-lg p-1"),
								filterTab("All", "", "filters.status"),
								filterTab("Active", "active", "filters.status"),
								filterTab("Expiring", "expiring", "filters.status"),
							),

							// Right side controls
							Div(
								Class("flex flex-wrap items-center gap-3"),

								// View toggle
								ViewToggle("filters.view"),

								// Device filter
								FilterSelect("Device:", "filters.device", "applyFilters()", []FilterOption{
									{Value: "", Label: "All Devices"},
									{Value: "desktop", Label: "Desktop"},
									{Value: "mobile", Label: "Mobile"},
									{Value: "tablet", Label: "Tablet"},
								}),

								// Search
								SearchInput("Search by user ID...", "filters.search", "applyFilters()"),
							),
						),
					),
				),

				// Sessions content
				Div(
					// Empty state
					Div(
						g.Attr("x-show", "sessions.length === 0"),
						EmptyState(
							lucide.MonitorSmartphone(Class("size-12 text-muted-foreground")),
							"No Active Sessions",
							"There are no active sessions matching your filters. Sessions will appear here when users sign in to your application.",
						),
					),

					// Sessions grid view
					Div(
						g.Attr("x-show", "sessions.length > 0 && filters.view === 'grid'"),
						sessionsGrid(appBase),
					),

					// Sessions list view
					Div(
						g.Attr("x-show", "sessions.length > 0 && filters.view === 'list'"),
						sessionsTable(appBase),
					),
				),

				// Pagination
				Pagination("goToPage"),
			),
		),
	)
}

// sessionsListData returns the Alpine.js data object for sessions list
func sessionsListData(appID string) string {
	return fmt.Sprintf(`{
		sessions: [],
		stats: {
			totalSessions: 0,
			activeCount: 0,
			expiringCount: 0,
			expiredCount: 0,
			mobileCount: 0,
			desktopCount: 0,
			tabletCount: 0,
			uniqueUsers: 0
		},
		pagination: {
			currentPage: 1,
			pageSize: 25,
			totalItems: 0,
			totalPages: 0
		},
		filters: {
			status: '',
			device: '',
			search: '',
			view: 'grid'
		},
		loading: true,
		error: null,
		
		get visiblePages() {
			const current = this.pagination.currentPage;
			const total = this.pagination.totalPages;
			const range = [];
			const start = Math.max(1, current - 2);
			const end = Math.min(total, current + 2);
			for (let i = start; i <= end; i++) {
				range.push(i);
			}
			return range;
		},
		
		async loadSessions() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('multisession.getSessions', {
					appId: '%s',
					page: this.pagination.currentPage,
					pageSize: this.pagination.pageSize,
					status: this.filters.status,
					device: this.filters.device,
					search: this.filters.search
				});
				
				this.sessions = result.sessions || [];
				this.stats = result.stats || {};
				this.pagination = result.pagination || { currentPage: 1, pageSize: 25, totalItems: 0, totalPages: 0 };
			} catch (err) {
				console.error('Failed to load sessions:', err);
				this.error = err.message || 'Failed to load sessions';
			} finally {
				this.loading = false;
			}
		},
		
		applyFilters() {
			this.pagination.currentPage = 1;
			this.loadSessions();
		},
		
		setStatusFilter(status) {
			this.filters.status = status;
			this.applyFilters();
		},
		
		goToPage(page) {
			if (page >= 1 && page <= this.pagination.totalPages) {
				this.pagination.currentPage = page;
				this.loadSessions();
			}
		},
		
		async revokeSession(sessionId) {
			if (!confirm('Are you sure you want to revoke this session? The user will be logged out.')) return;
			try {
				const result = await $bridge.call('multisession.revokeSession', { 
					appId: '%s',
					sessionId: sessionId 
				});
				if (result.message) {
					alert(result.message);
				}
				await this.loadSessions();
			} catch (err) {
				alert(err.message || 'Failed to revoke session');
			}
		}
	}`, appID, appID)
}

// filterTab renders a filter tab button
func filterTab(label, value, xModel string) g.Node {
	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf("setStatusFilter('%s')", value)),
		g.Attr(":class", fmt.Sprintf("%s === '%s' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'", xModel, value)),
		Class("inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-all"),
		g.Text(label),
	)
}

// sessionsGrid renders the sessions grid view
func sessionsGrid(appBase string) g.Node {
	return Div(
		Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-3"),
		g.El("template", g.Attr("x-for", "session in sessions"), g.Attr(":key", "session.id"),
			sessionCard(appBase),
		),
	)
}

// sessionCard renders a single session card
func sessionCard(appBase string) g.Node {
	return card.Card(
		Class("hover:shadow-md transition-shadow hover:border-primary/50"),
		card.Content(
			Class("p-5"),

			// Header
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex items-center gap-4"),
					// Device icon
					Div(
						g.Attr(":class", `{
							'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400': session.deviceType === 'mobile',
							'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400': session.deviceType === 'tablet',
							'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400': session.deviceType !== 'mobile' && session.deviceType !== 'tablet'
						}`),
						Class("flex h-12 w-12 items-center justify-center rounded-xl"),
						// Icon based on device type
						Div(
							g.Attr("x-show", "session.deviceType === 'mobile'"),
							lucide.Smartphone(Class("size-6")),
						),
						Div(
							g.Attr("x-show", "session.deviceType === 'tablet'"),
							lucide.Tablet(Class("size-6")),
						),
						Div(
							g.Attr("x-show", "session.deviceType !== 'mobile' && session.deviceType !== 'tablet'"),
							lucide.Monitor(Class("size-6")),
						),
					),
					Div(
						H3(Class("font-semibold"), g.Attr("x-text", "session.deviceInfo")),
						P(Class("text-sm text-muted-foreground"), g.Attr("x-text", "session.os")),
					),
				),
				// Status badge
				DynamicStatusBadge(),
			),

			// Info grid
			Div(
				Class("mt-5 grid grid-cols-2 gap-4 border-t pt-4"),

				// User ID
				Div(
					Class("col-span-2"),
					P(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("User")),
					Div(
						Class("mt-1 flex items-center gap-2"),
						Div(Class("flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold"), g.Text("ID")),
						Span(Class("truncate font-mono text-sm"), g.Attr("x-text", "session.userId")),
					),
				),

				// IP Address
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("IP Address")),
					P(Class("mt-1 text-sm font-medium"), g.Attr("x-text", "session.ipAddress")),
				),

				// Created
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("Created")),
					P(Class("mt-1 text-sm"), g.Attr("x-text", "session.lastUsed")),
				),
			),
		),

		// Footer
		card.Footer(
			Class("flex items-center justify-between bg-muted/50 px-5 py-3"),
			Span(Class("text-xs text-muted-foreground"),
				g.El("template", g.Attr("x-if", "session.isActive"),
					Span(g.Text("Expires "), Span(g.Attr("x-text", "session.expiresIn"))),
				),
				g.El("template", g.Attr("x-if", "!session.isActive"),
					Span(g.Text("Expired")),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				// View details
				A(
					g.Attr(":href", fmt.Sprintf("'%s/multisession/session/' + session.id", appBase)),
					Class("rounded-lg p-2 text-muted-foreground hover:bg-background hover:text-primary transition-colors"),
					Title("View Details"),
					lucide.Eye(Class("size-4")),
				),
				// Revoke button
				g.El("template", g.Attr("x-if", "session.isActive"),
					Button(
						Type("button"),
						g.Attr("@click", "revokeSession(session.id)"),
						Class("rounded-lg p-2 text-muted-foreground hover:bg-background hover:text-destructive transition-colors"),
						Title("Revoke Session"),
						lucide.LogOut(Class("size-4")),
					),
				),
			),
		),
	)
}

// sessionsTable renders the sessions table view
func sessionsTable(appBase string) g.Node {
	return card.Card(
		Div(
			Class("overflow-x-auto"),
			table.Table()(
				table.TableHeader()(
					table.TableRow()(
						table.TableHeaderCell()(g.Text("Device")),
						table.TableHeaderCell()(g.Text("User")),
						table.TableHeaderCell()(g.Text("IP Address")),
						table.TableHeaderCell()(g.Text("Status")),
						table.TableHeaderCell()(g.Text("Activity")),
						table.TableHeaderCell(table.WithAlign(table.AlignRight))(g.Text("Actions")),
					),
				),
				table.TableBody()(
					g.El("template", g.Attr("x-for", "session in sessions"), g.Attr(":key", "session.id"),
						table.TableRow()(
							// Device
							table.TableCell()(
								Div(
									Class("flex items-center gap-3"),
									Div(
										g.Attr(":class", `{
											'bg-purple-100 dark:bg-purple-900/30 text-purple-600': session.deviceType === 'mobile',
											'bg-amber-100 dark:bg-amber-900/30 text-amber-600': session.deviceType === 'tablet',
											'bg-blue-100 dark:bg-blue-900/30 text-blue-600': session.deviceType !== 'mobile' && session.deviceType !== 'tablet'
										}`),
										Class("flex h-10 w-10 items-center justify-center rounded-lg"),
										Div(g.Attr("x-show", "session.deviceType === 'mobile'"), lucide.Smartphone(Class("size-5"))),
										Div(g.Attr("x-show", "session.deviceType === 'tablet'"), lucide.Tablet(Class("size-5"))),
										Div(g.Attr("x-show", "session.deviceType !== 'mobile' && session.deviceType !== 'tablet'"), lucide.Monitor(Class("size-5"))),
									),
									Div(
										Div(Class("font-medium"), g.Attr("x-text", "session.deviceInfo")),
										Div(Class("text-xs text-muted-foreground"), g.Attr("x-text", "session.os")),
									),
								),
							),
							// User
							table.TableCell()(
								Div(
									Class("flex items-center gap-2"),
									Div(Class("flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-bold"), g.Text("ID")),
									Span(Class("font-mono text-sm text-muted-foreground"),
										g.Attr("x-text", "session.userId.substring(0, 8) + '...' + session.userId.substring(session.userId.length - 4)")),
								),
							),
							// IP Address
							table.TableCell(table.WithCellClass("text-muted-foreground"))(
								Span(g.Attr("x-text", "session.ipAddress")),
							),
							// Status
							table.TableCell()(
								DynamicStatusBadge(),
							),
							// Activity
							table.TableCell()(
								Div(
									Div(Class("text-sm"), g.Text("Created "), Span(g.Attr("x-text", "session.lastUsed"))),
									Div(Class("text-xs text-muted-foreground"),
										g.El("template", g.Attr("x-if", "session.isActive"),
											Span(g.Text("Expires "), Span(g.Attr("x-text", "session.expiresIn"))),
										),
										g.El("template", g.Attr("x-if", "!session.isActive"),
											Span(g.Text("Expired")),
										),
									),
								),
							),
							// Actions
							table.TableCell(table.WithAlign(table.AlignRight))(
								Div(
									Class("flex items-center justify-end gap-2"),
									A(
										g.Attr(":href", fmt.Sprintf("'%s/multisession/session/' + session.id", appBase)),
										Class("rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-primary transition-colors"),
										Title("View Details"),
										lucide.Eye(Class("size-4")),
									),
									g.El("template", g.Attr("x-if", "session.isActive"),
										button.Button(
											lucide.LogOut(Class("size-4")),
											button.WithVariant("ghost"),
											button.WithSize("icon"),
											button.WithAttrs(
												g.Attr("@click", "revokeSession(session.id)"),
												Title("Revoke Session"),
											),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}
