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

// UsersListPage shows list of users with search, filters, pagination, and bulk actions
func (p *PagesManager) UsersListPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				users: [],
				loading: true,
				error: '',
				pagination: {
					currentPage: 1,
					pageSize: 20,
					total: 0,
					totalPages: 0
				},
				filters: {
					searchTerm: '',
					status: null,
					emailVerified: null,
					roleFilter: ''
				},
				selectedUsers: [],
				selectAll: false,
				availableRoles: [],
				async loadUsers() {
					this.loading = true;
					this.error = '';
					try {
						const result = await $go('getUsersList', {
							appId: '`+appID+`',
							page: this.pagination.currentPage,
							pageSize: this.pagination.pageSize,
							searchTerm: this.filters.searchTerm,
							status: this.filters.status,
							emailVerified: this.filters.emailVerified,
							roleFilter: this.filters.roleFilter
						});
						this.users = result.users || [];
						this.pagination.total = result.total || 0;
						this.pagination.totalPages = result.totalPages || 0;
						this.selectedUsers = [];
						this.selectAll = false;
					} catch (err) {
						console.error('Failed to load users:', err);
						this.error = err.message || 'Failed to load users';
					} finally {
						this.loading = false;
					}
				},
				async loadRoles() {
					try {
						const result = await $go('listRoles', {
							appId: '`+appID+`'
						});
						this.availableRoles = result.roles || [];
					} catch (err) {
						console.error('Failed to load roles:', err);
					}
				},
				goToPage(page) {
					if (page >= 1 && page <= this.pagination.totalPages) {
						this.pagination.currentPage = page;
						this.loadUsers();
					}
				},
				applyFilters() {
					this.pagination.currentPage = 1;
					this.loadUsers();
				},
				toggleUserSelection(userId) {
					const index = this.selectedUsers.indexOf(userId);
					if (index > -1) {
						this.selectedUsers.splice(index, 1);
					} else {
						this.selectedUsers.push(userId);
					}
					this.selectAll = this.selectedUsers.length === this.users.length;
				},
				toggleSelectAll() {
					if (this.selectAll) {
						this.selectedUsers = [];
						this.selectAll = false;
					} else {
						this.selectedUsers = this.users.map(u => u.id);
						this.selectAll = true;
					}
				},
				async bulkDelete() {
					if (this.selectedUsers.length === 0) {
						alert('Please select users to delete');
						return;
					}
					if (!confirm('Are you sure you want to delete ' + this.selectedUsers.length + ' user(s)? This action cannot be undone.')) {
						return;
					}
					try {
						const result = await $go('bulkDeleteUsers', {
							appId: '`+appID+`',
							userIds: this.selectedUsers
						});
						alert('Deleted ' + result.successCount + ' user(s)' + 
							(result.failedCount > 0 ? '. Failed: ' + result.failedCount : ''));
						this.selectedUsers = [];
						this.selectAll = false;
						await this.loadUsers();
					} catch (err) {
						alert(err.message || 'Failed to delete users');
					}
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
					return this.users.filter(u => u.status === 'active').length;
				},
				get inactiveCount() {
					return this.users.filter(u => u.status === 'inactive').length;
				},
				get verifiedCount() {
					return this.users.filter(u => u.emailVerified).length;
				}
			}`),
			g.Attr("x-init", "loadUsers(); loadRoles()"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold tracking-tight text-slate-900 dark:text-white"), g.Text("Users")),
					P(Class("text-sm text-slate-600 dark:text-gray-400 mt-1"), g.Text("Manage application users")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.UserPlus(icons.WithSize(16)),
						Span(g.Text("Add User")),
					),
				),
			),

			// Bulk action toolbar (shown when users are selected)
			Div(
				g.Attr("x-show", "selectedUsers.length > 0"),
				Class("bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 flex items-center justify-between"),
				Div(
					Class("flex items-center gap-4"),
					Span(
						Class("text-sm font-medium text-blue-900 dark:text-blue-100"),
						g.Attr("x-text", "`${selectedUsers.length} user(s) selected`"),
					),
				),
				Div(
					Class("flex items-center gap-2"),
					button.Button(
						Div(
							Class("flex items-center gap-2"),
							icons.Trash(icons.WithSize(14)),
							Span(g.Text("Delete Selected")),
						),
						button.WithVariant("destructive"),
						button.WithSize("sm"),
						button.WithAttrs(g.Attr("@click", "bulkDelete()")),
					),
				),
			),

			// Filter controls
			Div(
				g.Attr("x-show", "!loading"),
				card.Card(
					card.Content(
						Div(
							Class("space-y-4"),

							// Search and filter row
							Div(
								Class("flex flex-col md:flex-row gap-4 items-start md:items-end"),

								// Search input
								Div(
									Class("flex-1 max-w-md"),
									Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Search")),
									Div(
										Class("relative"),
										Div(
											Class("absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"),
											icons.Search(icons.WithSize(16), icons.WithClass("text-gray-400")),
										),
										Input(
											Type("text"),
											Placeholder("Search by email..."),
											g.Attr("x-model", "filters.searchTerm"),
											g.Attr("@keyup.enter", "applyFilters()"),
											Class("pl-10 w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
										),
									),
								),

								// Status filter
								Div(
									Class("w-full md:w-40"),
									Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Status")),
									g.El("select",
										g.Attr("x-model", "filters.status"),
										g.Attr("@change", "applyFilters()"),
										Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
										g.El("option", Value(""), g.Text("All Status")),
										g.El("option", Value("active"), g.Text("Active")),
										g.El("option", Value("inactive"), g.Text("Inactive")),
										g.El("option", Value("banned"), g.Text("Banned")),
									),
								),

								// Email verified filter
								Div(
									Class("w-full md:w-40"),
									Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Email Status")),
									g.El("select",
										g.Attr("x-model", "filters.emailVerified"),
										g.Attr("@change", "applyFilters()"),
										Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
										g.El("option", Value(""), g.Text("All")),
										g.El("option", Value("true"), g.Text("Verified")),
										g.El("option", Value("false"), g.Text("Unverified")),
									),
								),

								// Role filter
								Div(
									Class("w-full md:w-48"),
									Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Role")),
									g.El("select",
										g.Attr("x-model", "filters.roleFilter"),
										g.Attr("@change", "applyFilters()"),
										Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
										g.El("option", Value(""), g.Text("All Roles")),
										g.El("template", g.Attr("x-for", "role in availableRoles"),
											g.El("option", g.Attr(":value", "role.name"), g.Attr("x-text", "role.displayName")),
										),
									),
								),

								// Search button
								button.Button(
									Div(
										Class("flex items-center gap-2"),
										icons.Search(icons.WithSize(16)),
										Span(g.Text("Search")),
									),
									button.WithAttrs(g.Attr("@click", "applyFilters()")),
								),
							),

							// Stats bar
							Div(
								g.Attr("x-show", "users.length > 0"),
								Class("pt-4 border-t border-gray-200 dark:border-gray-700 flex items-center gap-4 text-sm"),
								Span(
									g.Attr("x-text", "`Active: ${activeCount}`"),
									Class("px-2 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 rounded"),
								),
								Span(
									g.Attr("x-text", "`Inactive: ${inactiveCount}`"),
									Class("px-2 py-1 bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-400 rounded"),
								),
								Span(
									g.Attr("x-text", "`Verified: ${verifiedCount}`"),
									Class("px-2 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 rounded"),
								),
							),
						),
					),
				),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-16"),
				Div(
					Class("text-center"),
					Div(Class("animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto mb-4")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading users...")),
				),
			),

			// Error state
			Div(
				g.Attr("x-show", "!loading && error"),
				Class("p-4 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400"),
				P(g.Attr("x-text", "error")),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!loading && !error && users.length === 0"),
				Class("flex flex-col items-center justify-center py-16"),
				Div(
					Class("rounded-full bg-slate-100 dark:bg-gray-800 p-6 mb-4"),
					icons.Users(icons.WithSize(48), icons.WithClass("text-slate-400 dark:text-gray-500")),
				),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text("No users found")),
				P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Try adjusting your search criteria")),
			),

			// Users table
			Div(
				g.Attr("x-show", "!loading && !error && users.length > 0"),
				card.Card(
					card.Content(
						Div(
							Class("overflow-x-auto"),
							g.El("table",
								Class("w-full"),
								g.El("thead",
									Class("bg-gray-50 dark:bg-gray-800/50"),
									g.El("tr",
										// Select all checkbox
										g.El("th", Class("pl-6 pr-3 py-3 text-left"),
											Input(
												Type("checkbox"),
												g.Attr("x-model", "selectAll"),
												g.Attr("@change", "toggleSelectAll()"),
												Class("rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
											),
										),
										g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Email")),
										g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Name")),
										g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Email Verified")),
										g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
										g.El("th", Class("px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Created")),
										g.El("th", Class("px-3 pr-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
									),
								),
								g.El("tbody",
									Class("bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-800"),
									g.El("template", g.Attr("x-for", "user in users"), g.Attr(":key", "user.id"),
										g.El("tr",
											Class("hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"),
											// Checkbox
											g.El("td", Class("pl-6 pr-3 py-4"),
												Input(
													Type("checkbox"),
													g.Attr(":value", "user.id"),
													g.Attr("@change", "toggleUserSelection(user.id)"),
													g.Attr(":checked", "selectedUsers.includes(user.id)"),
													Class("rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
												),
											),
											// Email with avatar
											g.El("td", Class("px-3 py-4"),
												Div(
													Class("flex items-center gap-3"),
													Div(
														Class("flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white font-semibold text-sm"),
														g.Attr("x-text", "user.email ? user.email.charAt(0).toUpperCase() : '?'"),
													),
													Span(
														Class("text-sm font-medium text-slate-900 dark:text-white"),
														g.Attr("x-text", "user.email"),
													),
												),
											),
											// Name
											g.El("td", Class("px-3 py-4"),
												Span(
													Class("text-sm text-slate-700 dark:text-gray-300"),
													g.Attr("x-text", "user.name || '-'"),
												),
											),
											// Verified badge
											g.El("td", Class("px-3 py-4"),
												Span(
													Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium"),
													g.Attr(":class", "user.emailVerified ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400' : 'bg-slate-100 dark:bg-slate-500/20 text-slate-600 dark:text-slate-400'"),
													Span(
														Class("w-1.5 h-1.5 rounded-full"),
														g.Attr(":class", "user.emailVerified ? 'bg-emerald-500' : 'bg-slate-400'"),
													),
													Span(g.Attr("x-text", "user.emailVerified ? 'Verified' : 'Unverified'")),
												),
											),
											// Status badge
											g.El("td", Class("px-3 py-4"),
												Span(
													Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium"),
													g.Attr(":class", "user.status === 'active' ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400' : 'bg-slate-100 dark:bg-slate-500/20 text-slate-600 dark:text-slate-400'"),
													Span(
														Class("w-1.5 h-1.5 rounded-full"),
														g.Attr(":class", "user.status === 'active' ? 'bg-emerald-500' : 'bg-slate-400'"),
													),
													Span(g.Attr("x-text", "user.status || 'inactive'")),
												),
											),
											// Created date
											g.El("td", Class("px-3 py-4"),
												Span(
													Class("text-sm text-slate-600 dark:text-gray-400"),
													g.Attr("x-text", "new Date(user.createdAt).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })"),
												),
											),
											// Actions
											g.El("td", Class("px-3 pr-6 py-4"),
												Div(
													Class("flex items-center gap-2 justify-end"),
													button.Button(
														Div(
															Class("flex items-center gap-1.5"),
															icons.Eye(icons.WithSize(14)),
															Span(g.Text("View")),
														),
														button.WithVariant("ghost"),
														button.WithSize("sm"),
														button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users/' + user.id")),
													),
													button.Button(
														Div(
															Class("flex items-center gap-1.5"),
															icons.Pencil(icons.WithSize(14)),
															Span(g.Text("Edit")),
														),
														button.WithVariant("outline"),
														button.WithSize("sm"),
														button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users/' + user.id + '/edit'")),
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
									Span(g.Text(" users")),
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

// UserDetailPage shows detailed user information
func (p *PagesManager) UserDetailPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	userID := ctx.Param("userId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				user: null,
				loading: true,
				error: null,
				async loadUser() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $go('getUserDetail', {
							userId: '`+userID+`',
							appId: '`+appID+`'
						});
						this.user = result;
					} catch (err) {
						console.error('Failed to load user:', err);
						this.error = err.message || 'Failed to load user';
					} finally {
						this.loading = false;
					}
				}
			}`),
			g.Attr("x-init", "loadUser()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-16"),
				Div(
					Class("text-center"),
					Div(Class("animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto mb-4")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading user...")),
				),
			),

			// Error state
			Div(
				g.Attr("x-show", "!loading && error"),
				Class("space-y-4"),
				card.Card(
					card.Content(
						Div(
							Class("text-center py-12"),
							Div(
								Class("rounded-full bg-red-100 dark:bg-red-900/20 p-6 w-20 h-20 mx-auto mb-4 flex items-center justify-center"),
								icons.AlertCircle(icons.WithSize(32), icons.WithClass("text-red-600 dark:text-red-400")),
							),
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text("User Not Found")),
							P(Class("text-sm text-slate-600 dark:text-gray-400 mb-4"), g.Attr("x-text", "error")),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									icons.ArrowLeft(icons.WithSize(16)),
									Span(g.Text("Back to Users")),
								),
								button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users'")),
							),
						),
					),
				),
			),

			// User detail content
			Div(
				g.Attr("x-show", "!loading && !error && user"),
				Class("space-y-6"),

				// Header with actions
				Div(
					Class("flex items-center justify-between"),
					Div(
						button.Button(
							Div(
								Class("flex items-center gap-2"),
								icons.ArrowLeft(icons.WithSize(16)),
								Span(g.Text("Back to Users")),
							),
							button.WithVariant("ghost"),
							button.WithSize("sm"),
							button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users'")),
						),
					),
					Div(
						Class("flex items-center gap-2"),
						button.Button(
							Div(
								Class("flex items-center gap-2"),
								icons.Pencil(icons.WithSize(16)),
								Span(g.Text("Edit User")),
							),
							button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users/"+userID+"/edit'")),
						),
					),
				),

				// User header card
				card.Card(
					card.Content(
						Div(
							Class("flex items-start gap-6"),
							// Avatar
							Div(
								Class("flex-shrink-0 w-20 h-20 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white font-bold text-3xl"),
								g.Attr("x-text", "user && user.email ? user.email.charAt(0).toUpperCase() : '?'"),
							),
							// User info
							Div(
								Class("flex-1"),
								H1(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "user ? user.name || user.email : 'User'"),
								),
								P(
									Class("text-sm text-slate-600 dark:text-gray-400 mt-1"),
									g.Attr("x-text", "user ? user.email : ''"),
								),
								// Badges
								Div(
									Class("flex items-center gap-2 mt-3"),
									Span(
										Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium"),
										g.Attr("x-show", "user && user.emailVerified"),
										g.Attr(":class", "user && user.emailVerified ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400' : ''"),
										Span(Class("w-1.5 h-1.5 rounded-full bg-emerald-500")),
										Span(g.Text("Email Verified")),
									),
									Span(
										Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium"),
										g.Attr(":class", "user && user.status === 'active' ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400' : 'bg-slate-100 dark:bg-slate-500/20 text-slate-600 dark:text-slate-400'"),
										Span(
											Class("w-1.5 h-1.5 rounded-full"),
											g.Attr(":class", "user && user.status === 'active' ? 'bg-emerald-500' : 'bg-slate-400'"),
										),
										Span(g.Attr("x-text", "user ? (user.status || 'inactive') : ''")),
									),
								),
							),
						),
					),
				),

				// User details grid
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 gap-6"),

					// Basic information
					card.Card(
						card.Header(
							card.Title("Basic Information"),
						),
						card.Content(
							Div(
								Class("space-y-4"),
								// ID
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("User ID")),
									P(Class("text-sm font-mono text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user ? user.id : ''")),
								),
								// Email
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Email")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user ? user.email : ''")),
								),
								// Name
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Name")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user && user.name ? user.name : '-'")),
								),
								// Created
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Created At")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user && user.createdAt ? new Date(user.createdAt).toLocaleString() : ''")),
								),
								// Updated
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Updated At")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user && user.updatedAt ? new Date(user.updatedAt).toLocaleString() : '-'")),
								),
							),
						),
					),

					// Activity information
					card.Card(
						card.Header(
							card.Title("Activity"),
						),
						card.Content(
							Div(
								Class("space-y-4"),
								// Last login
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Last Login")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user && user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleString() : 'Never'")),
								),
								// Active sessions
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Active Sessions")),
									P(Class("text-sm text-slate-900 dark:text-white mt-1"), g.Attr("x-text", "user ? user.sessionCount : 0")),
								),
								// Roles
								Div(
									Label(Class("text-sm font-medium text-gray-500 dark:text-gray-400"), g.Text("Roles")),
									Div(
										Class("flex flex-wrap gap-2 mt-2"),
										g.El("template", g.Attr("x-for", "role in (user ? user.roles : [])"),
											Span(
												Class("inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-500/20 text-blue-700 dark:text-blue-400"),
												g.Attr("x-text", "role"),
											),
										),
										Span(
											g.Attr("x-show", "!user || !user.roles || user.roles.length === 0"),
											Class("text-sm text-slate-600 dark:text-gray-400"),
											g.Text("No roles assigned"),
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
