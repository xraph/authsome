package pages

import (
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ClientDetailPage shows OAuth client details with tabs
func (p *PagesManager) ClientDetailPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	clientID := ctx.Param("clientId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				client: null,
				stats: null,
				loading: true,
				error: null,
				activeTab: 'overview',
				
				async loadClient() {
					this.loading = true;
					this.error = null;
					try {
						const clientResult = await $bridge.call('oidcprovider.getClient', {
							clientId: '`+clientID+`'
						});
						this.client = clientResult.data;
						
						const statsResult = await $bridge.call('oidcprovider.getClientStats', {
							clientId: '`+clientID+`'
						});
						this.stats = statsResult.data;
					} catch (err) {
						this.error = err.message || 'Failed to load client';
					} finally {
						this.loading = false;
					}
				},
				
				async regenerateSecret() {
					if (!confirm('Are you sure? This will invalidate the current secret.')) {
						return;
					}
					try {
						const result = await $bridge.call('oidcprovider.regenerateSecret', {
							clientId: '`+clientID+`'
						});
						alert('New secret: ' + result.data.clientSecret + '\n\nSave this secret now - it will not be shown again!');
					} catch (err) {
						alert('Failed to regenerate secret: ' + err.message);
					}
				}
			}`),
			g.Attr("x-init", "loadClient()"),

			// Back link
			A(
				Href(p.baseUIPath+"/app/"+appID+"/oauth/clients"),
				Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"),
				icons.ArrowLeft(icons.WithSize(16)),
				g.Text("Back to Clients"),
			),

			// Loading state
			g.El("template", g.Attr("x-if", "loading"),
				Div(Class("flex items-center justify-center py-12"),
					Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
				),
			),

			// Error state
			g.El("template", g.Attr("x-if", "!loading && error"),
				card.Card(
					card.Content(
						Div(Class("text-center py-8"),
							icons.AlertCircle(icons.WithSize(48), icons.WithClass("mx-auto text-red-500 mb-4")),
							P(Class("text-red-600 dark:text-red-400"), g.Attr("x-text", "error")),
						),
					),
				),
			),

			// Content
			g.El("template", g.Attr("x-if", "!loading && !error && client"),
				Div(Class("space-y-2"),
					// Header
					Div(Class("flex items-start justify-between"),
						Div(
							H1(Class("text-3xl font-bold"), g.Attr("x-text", "client.clientName")),
							P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Attr("x-text", "client.clientId")),
						),
						button.Button(
							Div(Class("flex items-center gap-2"),
								icons.Key(icons.WithSize(16)),
								g.Text("Regenerate Secret"),
							),
							button.WithVariant("outline"),
							button.WithAttrs(g.Attr("@click", "regenerateSecret()")),
						),
					),

					// Tab buttons
					Div(Class("border-b border-gray-200 dark:border-gray-700 flex gap-4"),
						Button(
							Class("px-4 py-2 border-b-2 transition-colors"),
							g.Attr(":class", "activeTab === 'overview' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground'"),
							g.Attr("@click", "activeTab = 'overview'"),
							g.Text("Overview"),
						),
						Button(
							Class("px-4 py-2 border-b-2 transition-colors"),
							g.Attr(":class", "activeTab === 'security' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground'"),
							g.Attr("@click", "activeTab = 'security'"),
							g.Text("Security"),
						),
						Button(
							Class("px-4 py-2 border-b-2 transition-colors"),
							g.Attr(":class", "activeTab === 'tokens' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground'"),
							g.Attr("@click", "activeTab = 'tokens'"),
							g.Text("Grant Types & Scopes"),
						),
						Button(
							Class("px-4 py-2 border-b-2 transition-colors"),
							g.Attr(":class", "activeTab === 'stats' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground'"),
							g.Attr("@click", "activeTab = 'stats'"),
							g.Text("Statistics"),
						),
					),

					// Tab content areas
					Div(Class("mt-4"),
						// Overview
						Div(
							g.Attr("x-show", "activeTab === 'overview'"),
							Div(Class("grid gap-6 md:grid-cols-2"),
								card.Card(
									card.Header(card.Title("Basic Information")),
									card.Content(
										Div(Class("space-y-4"),
											Div(
												Div(Class("text-sm text-muted-foreground"), g.Text("Client Name")),
												Div(Class("font-medium"), g.Attr("x-text", "client.clientName")),
											),
											Div(
												Div(Class("text-sm text-muted-foreground"), g.Text("Application Type")),
												Div(Class("font-medium capitalize"), g.Attr("x-text", "client.applicationType")),
											),
											Div(
												Div(Class("text-sm text-muted-foreground"), g.Text("Logo URI")),
												Div(Class("font-medium text-sm break-all"), g.Attr("x-text", "client.logoUri || 'Not set'")),
											),
										),
									),
								),
								card.Card(
									card.Header(card.Title("Redirect URIs")),
									card.Content(
										Div(Class("space-y-2"),
											g.El("template", g.Attr("x-if", "client.redirectUris && client.redirectUris.length > 0"),
												Div(Class("space-y-2"),
													g.El("template", g.Attr("x-for", "uri in client.redirectUris"),
														Div(Class("text-sm font-mono bg-muted px-3 py-2 rounded border"),
															g.Attr("x-text", "uri"),
														),
													),
												),
											),
											g.El("template", g.Attr("x-if", "!client.redirectUris || client.redirectUris.length === 0"),
												P(Class("text-sm text-muted-foreground"), g.Text("No redirect URIs configured")),
											),
										),
									),
								),
							),
						),

						// Security
						Div(
							g.Attr("x-show", "activeTab === 'security'"),
							card.Card(
								card.Header(card.Title("Security Settings")),
								card.Content(
									Div(Class("space-y-4"),
										Div(
											Div(Class("text-sm text-muted-foreground"), g.Text("Token Endpoint Auth")),
											Div(Class("font-medium"), g.Attr("x-text", "client.tokenEndpointAuth")),
										),
										Div(
											Div(Class("text-sm text-muted-foreground"), g.Text("PKCE Required")),
											Div(
												g.El("template", g.Attr("x-if", "client.requirePkce"),
													badge.Badge("Yes", badge.WithVariant("default")),
												),
												g.El("template", g.Attr("x-if", "!client.requirePkce"),
													badge.Badge("No", badge.WithVariant("secondary")),
												),
											),
										),
										Div(
											Div(Class("text-sm text-muted-foreground"), g.Text("Require Consent")),
											Div(
												g.El("template", g.Attr("x-if", "client.requireConsent"),
													badge.Badge("Yes", badge.WithVariant("default")),
												),
												g.El("template", g.Attr("x-if", "!client.requireConsent"),
													badge.Badge("No", badge.WithVariant("secondary")),
												),
											),
										),
										Div(
											Div(Class("text-sm text-muted-foreground"), g.Text("Trusted Client")),
											Div(
												g.El("template", g.Attr("x-if", "client.trustedClient"),
													badge.Badge("Yes", badge.WithVariant("default")),
												),
												g.El("template", g.Attr("x-if", "!client.trustedClient"),
													badge.Badge("No", badge.WithVariant("secondary")),
												),
											),
										),
									),
								),
							),
						),

						// Tokens/Scopes
						Div(
							g.Attr("x-show", "activeTab === 'tokens'"),
							Div(Class("space-y-4"),
								card.Card(
									card.Header(card.Title("Grant Types")),
									card.Content(
										Div(
											g.El("template", g.Attr("x-if", "client.grantTypes && client.grantTypes.length > 0"),
												Div(Class("flex flex-wrap gap-2"),
													g.El("template", g.Attr("x-for", "grant in client.grantTypes"),
														Span(
															Class("inline-flex items-center rounded-md bg-primary/10 px-3 py-1.5 text-sm font-medium text-primary border border-primary/20"),
															g.Attr("x-text", "grant"),
														),
													),
												),
											),
											g.El("template", g.Attr("x-if", "!client.grantTypes || client.grantTypes.length === 0"),
												P(Class("text-sm text-muted-foreground"), g.Text("No grant types configured")),
											),
										),
									),
								),
								card.Card(
									card.Header(card.Title("Response Types")),
									card.Content(
										Div(
											g.El("template", g.Attr("x-if", "client.responseTypes && client.responseTypes.length > 0"),
												Div(Class("flex flex-wrap gap-2"),
													g.El("template", g.Attr("x-for", "type in client.responseTypes"),
														Span(
															Class("inline-flex items-center rounded-md bg-secondary/10 px-3 py-1.5 text-sm font-medium text-secondary-foreground border border-secondary/20"),
															g.Attr("x-text", "type"),
														),
													),
												),
											),
											g.El("template", g.Attr("x-if", "!client.responseTypes || client.responseTypes.length === 0"),
												P(Class("text-sm text-muted-foreground"), g.Text("No response types configured")),
											),
										),
									),
								),
								card.Card(
									card.Header(card.Title("Allowed Scopes")),
									card.Content(
										Div(
											g.El("template", g.Attr("x-if", "client.allowedScopes && client.allowedScopes.length > 0"),
												Div(Class("flex flex-wrap gap-2"),
													g.El("template", g.Attr("x-for", "scope in client.allowedScopes"),
														Span(
															Class("inline-flex items-center rounded-md bg-blue-50 dark:bg-blue-950 px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 border border-blue-200 dark:border-blue-800"),
															g.Attr("x-text", "scope"),
														),
													),
												),
											),
											g.El("template", g.Attr("x-if", "!client.allowedScopes || client.allowedScopes.length === 0"),
												P(Class("text-sm text-muted-foreground"), g.Text("No scopes configured")),
											),
										),
									),
								),
							),
						),

						// Stats
						Div(
							g.Attr("x-show", "activeTab === 'stats'"),
							Div(Class("grid gap-6 md:grid-cols-3"),
								card.Card(
									card.Content(
										Div(Class("text-center py-4"),
											Div(Class("text-3xl font-bold"), g.Attr("x-text", "stats ? stats.totalTokens : 0")),
											Div(Class("text-sm text-muted-foreground mt-1"), g.Text("Total Tokens")),
										),
									),
								),
								card.Card(
									card.Content(
										Div(Class("text-center py-4"),
											Div(Class("text-3xl font-bold text-green-600"), g.Attr("x-text", "stats ? stats.activeTokens : 0")),
											Div(Class("text-sm text-muted-foreground mt-1"), g.Text("Active Tokens")),
										),
									),
								),
								card.Card(
									card.Content(
										Div(Class("text-center py-4"),
											Div(Class("text-3xl font-bold text-blue-600"), g.Attr("x-text", "stats ? stats.totalUsers : 0")),
											Div(Class("text-sm text-muted-foreground mt-1"), g.Text("Total Users")),
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
