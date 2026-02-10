package pages

import (
	"fmt"

	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// ClientsListPage shows the list of OAuth2/OIDC clients.
func (p *PagesManager) ClientsListPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Page header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("OAuth & OIDC Clients")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Manage OAuth2 and OpenID Connect clients")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						Span(g.Text("New Client")),
					),
					button.WithAttrs(
						g.Attr("@click", fmt.Sprintf("window.location.href = '%s/app/%s/oauth/clients/new'", p.baseUIPath, appID)),
					),
				),
			),

			// Clients list with Alpine.js
			Div(
				g.Attr("x-data", `{
					clients: [],
					loading: true,
					error: null,
					search: '',
					page: 1,
					pageSize: 20,
					pagination: null,
					
					async loadClients() {
						this.loading = true;
						this.error = null;
						try {
							const result = await $bridge.call('oidcprovider.getClients', {
								appId: '`+appID+`',
								page: this.page,
								pageSize: this.pageSize,
								search: this.search
							});
							this.clients = result.data || [];
							this.pagination = result.pagination;
						} catch (err) {
							console.error('Failed to load clients:', err);
							this.error = err.message || 'Failed to load OAuth clients';
						} finally {
							this.loading = false;
						}
					},
					
					async deleteClient(clientId) {
						if (!confirm('Are you sure you want to delete this client? This will revoke all associated tokens.')) {
							return;
						}
						try {
							await $bridge.call('oidcprovider.deleteClient', {
								clientId: clientId
							});
							await this.loadClients();
						} catch (err) {
							alert('Failed to delete client: ' + (err.message || 'Unknown error'));
						}
					},
					
					formatDate(dateStr) {
						if (!dateStr) return 'N/A';
						const date = new Date(dateStr);
						return date.toLocaleDateString('en-US', { 
							year: 'numeric', 
							month: 'short', 
							day: 'numeric' 
						});
					}
				}`),
				g.Attr("x-init", "loadClients()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(
						Class("flex items-center justify-center py-12"),
						Div(
							Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary"),
						),
					),
				),

				// Error state
				g.El("template", g.Attr("x-if", "!loading && error"),
					card.Card(
						card.Content(
							Div(
								Class("text-center py-8"),
								icons.AlertCircle(icons.WithSize(48), icons.WithClass("mx-auto text-red-500 mb-4")),
								P(Class("text-red-600 dark:text-red-400 font-medium"), g.Attr("x-text", "error")),
								button.Button(
									g.Text("Retry"),
									button.WithVariant("outline"),
									button.WithAttrs(g.Attr("@click", "loadClients()"), g.Attr("class", "mt-4")),
								),
							),
						),
					),
				),

				// Empty state
				g.El("template", g.Attr("x-if", "!loading && !error && clients.length === 0"),
					card.Card(
						card.Content(
							Div(
								Class("text-center py-12"),
								icons.Shield(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
								H3(Class("text-lg font-semibold text-gray-900 dark:text-gray-100"), g.Text("No OAuth clients yet")),
								P(Class("text-gray-500 dark:text-gray-400 mt-1 mb-4"), g.Text("Create your first OAuth2/OIDC client to get started.")),
								button.Button(
									Div(
										Class("flex items-center gap-2"),
										icons.Plus(icons.WithSize(16)),
										Span(g.Text("Create Client")),
									),
									button.WithAttrs(
										g.Attr("@click", fmt.Sprintf("window.location.href = '%s/app/%s/oauth/clients/new'", p.baseUIPath, appID)),
									),
								),
							),
						),
					),
				),

				// Clients grid
				g.El("template", g.Attr("x-if", "!loading && !error && clients.length > 0"),
					Div(
						Class("space-y-4"),

						// Search bar
						Div(
							Class("flex items-center gap-4"),
							input.Input(
								input.WithType("text"),
								input.WithPlaceholder("Search clients..."),
								input.WithAttrs(
									g.Attr("x-model", "search"),
									g.Attr("@keyup.enter", "page = 1; loadClients()"),
								),
							),
							button.Button(
								g.Text("Search"),
								button.WithAttrs(g.Attr("@click", "page = 1; loadClients()")),
							),
						),

						// Clients list
						Div(
							Class("grid grid-cols-1 gap-4"),
							g.El("template", g.Attr("x-for", "client in clients"),
								card.Card(
									card.Header(
										Div(
											Class("flex items-start justify-between"),
											Div(
												Class("flex-1"),
												Div(
													Class("flex items-center gap-2"),
													card.Title("", card.WithAttrs(g.Attr("x-text", "client.clientName"))),
													// Org-level badge
													g.El("template", g.Attr("x-if", "client.isOrgLevel"),
														badge.Badge("Org-Level", badge.WithVariant("secondary")),
													),
												),
												card.Description("", card.WithAttrs(g.Attr("x-text", "client.clientId"))),
											),
											Div(
												Class("flex items-center gap-2"),
												button.Button(
													icons.Eye(icons.WithSize(16)),
													button.WithVariant("ghost"),
													button.WithSize("sm"),
													button.WithAttrs(
														g.Attr("@click", fmt.Sprintf("window.location.href = '%s/app/%s/oauth/clients/' + client.clientId", p.baseUIPath, appID)),
													),
												),
												button.Button(
													icons.Trash2(icons.WithSize(16)),
													button.WithVariant("ghost"),
													button.WithSize("sm"),
													button.WithAttrs(
														g.Attr("@click", "deleteClient(client.clientId)"),
													),
												),
											),
										),
									),
									card.Content(
										Div(
											Class("grid grid-cols-2 md:grid-cols-4 gap-4 text-sm"),
											Div(
												Div(Class("text-muted-foreground"), g.Text("Type")),
												Div(Class("font-medium capitalize"), g.Attr("x-text", "client.applicationType")),
											),
											Div(
												Div(Class("text-muted-foreground"), g.Text("Grant Types")),
												Div(
													Class("flex flex-wrap gap-1"),
													g.El("template", g.Attr("x-for", "grant in client.grantTypes"),
														Span(Class("inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium"),
															g.Attr("x-text", "grant"),
														),
													),
												),
											),
											Div(
												Div(Class("text-muted-foreground"), g.Text("PKCE Required")),
												Div(
													g.El("template", g.Attr("x-if", "client.requirePkce"),
														Span(Class("text-green-600 dark:text-green-400"), g.Text("Yes")),
													),
													g.El("template", g.Attr("x-if", "!client.requirePkce"),
														Span(Class("text-gray-600 dark:text-gray-400"), g.Text("No")),
													),
												),
											),
											Div(
												Div(Class("text-muted-foreground"), g.Text("Created")),
												Div(Class("font-medium"), g.Attr("x-text", "formatDate(client.createdAt)")),
											),
										),
									),
								),
							),
						),

						// Pagination
						g.El("template", g.Attr("x-if", "pagination && pagination.totalPages > 1"),
							Div(
								Class("flex items-center justify-center gap-2 mt-6"),
								button.Button(
									Div(
										Class("flex items-center gap-1"),
										icons.ChevronLeft(icons.WithSize(16)),
										g.Text("Previous"),
									),
									button.WithVariant("outline"),
									button.WithSize("sm"),
									button.WithAttrs(
										g.Attr("x-show", "page > 1"),
										g.Attr("@click", "page--; loadClients()"),
									),
								),
								Span(
									Class("text-sm text-muted-foreground"),
									g.Attr("x-text", "`Page ${page} of ${pagination.totalPages}`"),
								),
								button.Button(
									Div(
										Class("flex items-center gap-1"),
										g.Text("Next"),
										icons.ChevronRight(icons.WithSize(16)),
									),
									button.WithVariant("outline"),
									button.WithSize("sm"),
									button.WithAttrs(
										g.Attr("x-show", "page < pagination.totalPages"),
										g.Attr("@click", "page++; loadClients()"),
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
