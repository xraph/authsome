package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SessionsListPage shows active sessions with pagination and filtering
func (p *PagesManager) SessionsListPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Header
			Div(
				H1(Class("text-3xl font-bold"), g.Text("Sessions")),
				P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Monitor and manage user sessions")),
			),

			// Sessions container
			Div(
				Class("space-y-2"),
				g.Attr("x-data", `{
					sessions: [],
					loading: true,
					error: '',
					pagination: {
						currentPage: 1,
						pageSize: 20,
						total: 0,
						totalPages: 0
					},
					filters: {
						status: 'all',
						searchEmail: ''
					},
					async loadSessions() {
						this.loading = true;
						this.error = '';
						try {
							const result = await $go('getSessionsList', {
								appId: '`+appID+`',
								page: this.pagination.currentPage,
								pageSize: this.pagination.pageSize,
								status: this.filters.status,
								searchEmail: this.filters.searchEmail
							});
							this.sessions = result.sessions || [];
							this.pagination.total = result.total || 0;
							this.pagination.totalPages = result.totalPages || 0;
						} catch (err) {
							console.error('Failed to load sessions:', err);
							this.error = err.message || 'Failed to load sessions';
						} finally {
							this.loading = false;
						}
					},
					async revokeSession(sessionId) {
						if (!confirm('Are you sure you want to revoke this session?')) return;
						try {
							const result = await $go('revokeSession', { sessionId });
							if (result.message) {
								alert(result.message);
							}
							await this.loadSessions();
						} catch (err) {
							alert(err.message || 'Failed to revoke session');
						}
					},
					goToPage(page) {
						if (page >= 1 && page <= this.pagination.totalPages) {
							this.pagination.currentPage = page;
							this.loadSessions();
						}
					},
					applyFilters() {
						this.pagination.currentPage = 1;
						this.loadSessions();
					},
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
					get start() {
						if (this.pagination.total === 0) return 0;
						return (this.pagination.currentPage - 1) * this.pagination.pageSize + 1;
					},
					get end() {
						return Math.min(this.pagination.currentPage * this.pagination.pageSize, this.pagination.total);
					},
					get activeCount() {
						return this.sessions.filter(s => s.isActive).length;
					},
					get expiredCount() {
						return this.sessions.filter(s => !s.isActive).length;
					}
				}`),
				g.Attr("x-init", "loadSessions()"),

				// Filter controls
				Div(
					g.Attr("x-show", "!loading"),
					Class("space-y-4"),

					// Status filter and search
					card.Card(
						card.Content(
							Div(
								Class("flex flex-col md:flex-row gap-4 items-start md:items-center justify-between"),

								// Status filter radio buttons
								Div(
									Class("flex items-center gap-4"),
									Label(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Status:")),
									Div(
										Class("flex items-center gap-4"),
										Label(
											Class("flex items-center gap-2 cursor-pointer"),
											Input(
												Type("radio"),
												Name("status"),
												Value("all"),
												g.Attr("x-model", "filters.status"),
												g.Attr("@change", "applyFilters()"),
												Class("text-blue-600"),
											),
											Span(Class("text-sm"), g.Text("All")),
										),
										Label(
											Class("flex items-center gap-2 cursor-pointer"),
											Input(
												Type("radio"),
												Name("status"),
												Value("active"),
												g.Attr("x-model", "filters.status"),
												g.Attr("@change", "applyFilters()"),
												Class("text-blue-600"),
											),
											Span(Class("text-sm"), g.Text("Active")),
										),
										Label(
											Class("flex items-center gap-2 cursor-pointer"),
											Input(
												Type("radio"),
												Name("status"),
												Value("expired"),
												g.Attr("x-model", "filters.status"),
												g.Attr("@change", "applyFilters()"),
												Class("text-blue-600"),
											),
											Span(Class("text-sm"), g.Text("Expired")),
										),
									),
								),

								// Email search
								Div(
									Class("flex items-center gap-2 flex-1 max-w-md"),
									Div(
										Class("relative flex-1"),
										Div(
											Class("absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"),
											icons.Search(icons.WithSize(16), icons.WithClass("text-gray-400")),
										),
										Input(
											Type("text"),
											Placeholder("Search by email..."),
											g.Attr("x-model", "filters.searchEmail"),
											g.Attr("@keyup.enter", "applyFilters()"),
											Class("pl-10 w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
										),
									),
									button.Button(
										g.Text("Search"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "applyFilters()"),
										),
									),
								),
							),

							// Stats bar
							Div(
								g.Attr("x-show", "sessions.length > 0"),
								Class("mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400"),
								Span(
									g.Attr("x-text", "`Active: ${activeCount}`"),
									Class("px-2 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 rounded"),
								),
								Span(
									g.Attr("x-text", "`Expired: ${expiredCount}`"),
									Class("px-2 py-1 bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400 rounded"),
								),
							),
						),
					),
				),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					Class("flex items-center justify-center py-12"),
					Div(
						Class("animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"),
					),
				),

				// Error state
				Div(
					g.Attr("x-show", "!loading && error"),
					Class("p-4 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 mb-4"),
					P(g.Attr("x-text", "error")),
				),

				// Empty state
				Div(
					g.Attr("x-show", "!loading && !error && sessions.length === 0"),
					Class("text-center py-12"),
					Div(
						Class("text-gray-400 dark:text-gray-600 mb-4"),
						icons.Users(icons.WithSize(48)),
					),
					H3(Class("text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2"), g.Text("No sessions found")),
					P(Class("text-gray-600 dark:text-gray-400"), g.Text("No sessions match your current filters")),
				),

				// Sessions table
				Div(
					g.Attr("x-show", "!loading && !error && sessions.length > 0"),
					card.Card(
						card.Content(
							Div(
								Class("overflow-x-auto"),
								g.El("table",
									Class("w-full"),
									g.El("thead",
										Class("bg-gray-50 dark:bg-gray-800/50"),
										g.El("tr",
											g.El("th", Class("pl-6 pr-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("User")),
											g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
											g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Device")),
											g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("IP Address")),
											g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Last Active")),
											g.El("th", Class("px-3 pr-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
										),
									),
									g.El("tbody",
										Class("bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-800"),
										g.El("template", g.Attr("x-for", "session in sessions"),
											g.El("tr",
												Class("hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"),
												// User email (clickable)
												g.El("td",
													Class("pl-6 pr-3 py-4"),
													g.El("a",
														g.Attr("x-show", "session.userId && session.userEmail"),
														g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users/' + session.userId"),
														Class("text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"),
														g.Attr("x-text", "session.userEmail"),
													),
													Span(
														g.Attr("x-show", "!session.userId || !session.userEmail"),
														Class("text-gray-400"),
														g.Text("-"),
													),
												),
												// Status badge
												g.El("td",
													Class("px-3 py-4"),
													Span(
														g.Attr("x-show", "session.isActive"),
														Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"),
														Span(Class("mr-1.5"), g.Text("●")),
														g.Text("Active"),
													),
													Span(
														g.Attr("x-show", "!session.isActive"),
														Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400"),
														Span(Class("mr-1.5"), g.Text("○")),
														g.Text("Expired"),
													),
												),
												// Device
												g.El("td",
													Class("px-3 py-4 text-sm text-gray-900 dark:text-gray-100"),
													g.Attr("x-text", "session.device || 'Unknown'"),
												),
												// IP Address
												g.El("td",
													Class("px-3 py-4 text-sm text-gray-500 dark:text-gray-400 font-mono"),
													g.Attr("x-text", "session.ipAddress || '-'"),
												),
												// Last Active
												g.El("td",
													Class("px-3 py-4 text-sm text-gray-500 dark:text-gray-400"),
													g.Attr("x-text", "new Date(session.lastUsed).toLocaleString()"),
												),
												// Actions
												g.El("td",
													Class("px-3 pr-6 py-4"),
													button.Button(
														g.Text("Revoke"),
														button.WithVariant("destructive"),
														button.WithSize("sm"),
														button.WithAttrs(
															g.Attr("@click", "revokeSession(session.id)"),
														),
													),
												),
											),
										),
									),
								),
							),

							// Pagination controls
							Div(
								g.Attr("x-show", "pagination.totalPages > 1"),
								Class("flex items-center justify-between px-6 py-4 border-t border-gray-200 dark:border-gray-800"),

								// Pagination info
								Div(
									Class("text-sm text-gray-700 dark:text-gray-300"),
									Span(g.Text("Showing ")),
									Span(Class("font-medium"), g.Attr("x-text", "start")),
									Span(g.Text("-")),
									Span(Class("font-medium"), g.Attr("x-text", "end")),
									Span(g.Text(" of ")),
									Span(Class("font-medium"), g.Attr("x-text", "pagination.total")),
									Span(g.Text(" sessions")),
								),

								// Pagination buttons
								Div(
									Class("flex items-center gap-2"),
									button.Button(
										Div(
											Class("flex items-center gap-1"),
											icons.ChevronLeft(icons.WithSize(16)),
											Span(g.Text("Previous")),
										),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "goToPage(pagination.currentPage - 1)"),
											g.Attr(":disabled", "pagination.currentPage === 1"),
										),
									),

									// Page numbers
									g.El("template", g.Attr("x-for", "page in visiblePages"),
										button.Button(
											Span(g.Attr("x-text", "page")),
											button.WithSize("sm"),
											button.WithAttrs(
												g.Attr("@click", "goToPage(page)"),
												g.Attr(":class", "page === pagination.currentPage ? '' : 'bg-white dark:bg-gray-800'"),
												g.Attr(":variant", "page === pagination.currentPage ? 'default' : 'outline'"),
											),
										),
									),

									button.Button(
										Div(
											Class("flex items-center gap-1"),
											Span(g.Text("Next")),
											icons.ChevronRight(icons.WithSize(16)),
										),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "goToPage(pagination.currentPage + 1)"),
											g.Attr(":disabled", "pagination.currentPage >= pagination.totalPages"),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
