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

// ProvidersListPage renders the SCIM providers list page.
func ProvidersListPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			Div(
				H1(Class("text-2xl font-bold tracking-tight"), g.Text("Identity Providers")),
				P(Class("text-muted-foreground"), g.Text("Configure SCIM identity provider integrations")),
			),
			A(
				Href(appBase+"/scim/providers/add"),
				button.Button(
					Div(Class("flex items-center gap-2"), lucide.Plus(Class("size-4")), g.Text("Add Provider")),
					button.WithVariant("default"),
				),
			),
		),

		// Alpine.js container
		Div(
			g.Attr("x-data", providersListData(appID)),
			g.Attr("x-init", "loadProviders()"),

			// Filters
			card.Card(
				card.Content(
					Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
					Div(
						Class("flex items-center gap-2"),
						input.Input(
							input.WithType("text"),
							input.WithPlaceholder("Search providers..."),
							input.WithAttrs(
								g.Attr("x-model", "filters.search"),
								g.Attr("@input.debounce.300ms", "loadProviders()"),
								Class("w-64"),
							),
						),
					),
					Div(
						Class("flex items-center gap-2"),
						Select(
							Class("px-3 py-2 border rounded-md text-sm bg-background"),
							g.Attr("x-model", "filters.status"),
							g.Attr("@change", "loadProviders()"),
							Option(Value(""), g.Text("All Status")),
							Option(Value("active"), g.Text("Active")),
							Option(Value("inactive"), g.Text("Inactive")),
							Option(Value("error"), g.Text("Error")),
						),
						button.Button(
							lucide.RefreshCw(Class("size-4")),
							button.WithVariant("outline"),
							button.WithSize("icon"),
							button.WithAttrs(g.Attr("@click", "loadProviders()")),
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
					g.Attr("x-show", "providers.length === 0"),
					card.Card(
						card.Content(
							Class("text-center py-12"),
							lucide.Server(Class("size-16 mx-auto text-muted-foreground mb-4")),
							H3(Class("text-lg font-semibold mb-2"), g.Text("No providers configured")),
							P(Class("text-muted-foreground mb-4"), g.Text("Get started by adding your first identity provider")),
							A(
								Href(appBase+"/scim/providers/add"),
								button.Button(
									Div(Class("flex items-center gap-2"), lucide.Plus(Class("size-4")), g.Text("Add Provider")),
									button.WithVariant("default"),
								),
							),
						),
					),
				),

				// Providers grid
				Div(
					g.Attr("x-show", "providers.length > 0"),
					Class("grid gap-4 md:grid-cols-2 lg:grid-cols-3"),
					g.El("template",
						g.Attr("x-for", "provider in providers"),
						g.Attr(":key", "provider.id"),
						providerCard(appBase),
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
						g.Text(" providers"),
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
		),
	)
}

// providerCard renders a single provider card.
func providerCard(appBase string) g.Node {
	return card.Card(
		Class("hover:shadow-md transition-shadow"),
		card.Header(
			Class("pb-2"),
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("p-2 rounded-lg bg-primary/10"),
						lucide.Server(Class("size-5 text-primary")),
					),
					Div(
						Div(
							Class("font-semibold"),
							g.Attr("x-text", "provider.name"),
						),
						Div(
							Class("text-sm text-muted-foreground"),
							g.Attr("x-text", "provider.type"),
						),
					),
				),
				Span(
					Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium"),
					g.Attr(":class", "provider.status === 'active' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : provider.status === 'error' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'"),
					g.Attr("x-text", "provider.status"),
				),
			),
		),
		card.Content(
			Class("pt-0 space-y-3"),
			// Stats
			Div(
				Class("grid grid-cols-2 gap-4 text-sm"),
				Div(
					Div(Class("text-muted-foreground"), g.Text("Users")),
					Div(Class("font-semibold"), g.Attr("x-text", "provider.userCount")),
				),
				Div(
					Div(Class("text-muted-foreground"), g.Text("Groups")),
					Div(Class("font-semibold"), g.Attr("x-text", "provider.groupCount")),
				),
			),
			// Last sync
			Div(
				Class("text-sm"),
				Span(Class("text-muted-foreground"), g.Text("Last sync: ")),
				Span(g.Attr("x-text", "provider.lastSync ? formatRelativeTime(provider.lastSync) : 'Never'")),
			),
		),
		card.Footer(
			Class("flex items-center justify-between pt-4 border-t"),
			Div(
				Class("flex items-center gap-1"),
				button.Button(
					lucide.RefreshCw(Class("size-4"), g.Attr(":class", "syncing === provider.id ? 'animate-spin' : ''")),
					button.WithVariant("ghost"),
					button.WithSize("icon"),
					button.WithAttrs(
						g.Attr("@click", "triggerSync(provider.id)"),
						g.Attr(":disabled", "syncing === provider.id"),
					),
				),
				button.Button(
					lucide.Zap(Class("size-4")),
					button.WithVariant("ghost"),
					button.WithSize("icon"),
					button.WithAttrs(g.Attr("@click", "testConnection(provider.id)")),
				),
			),
			A(
				g.Attr(":href", fmt.Sprintf("'%s/scim/providers/' + provider.id", appBase)),
				button.Button(
					Div(Class("flex items-center gap-1"), g.Text("View Details"), lucide.ChevronRight(Class("size-4"))),
					button.WithVariant("outline"),
				),
			),
		),
	)
}

// providersListData returns the Alpine.js data for the providers list.
func providersListData(appID string) string {
	return fmt.Sprintf(`{
		providers: [],
		pagination: {
			page: 1,
			pageSize: 12,
			total: 0,
			totalPages: 0
		},
		filters: {
			search: '',
			status: ''
		},
		loading: true,
		error: null,
		syncing: null,
		
		async loadProviders() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('scim.getProviders', {
					appId: '%s',
					page: this.pagination.page,
					pageSize: this.pagination.pageSize,
					search: this.filters.search,
					status: this.filters.status
				});
				this.providers = result.providers || [];
				this.pagination.total = result.total || 0;
				this.pagination.totalPages = result.totalPages || 0;
			} catch (err) {
				console.error('Failed to load providers:', err);
				this.error = err.message || 'Failed to load providers';
			} finally {
				this.loading = false;
			}
		},
		
		goToPage(page) {
			if (page >= 1 && page <= this.pagination.totalPages) {
				this.pagination.page = page;
				this.loadProviders();
			}
		},
		
		async triggerSync(providerId) {
			this.syncing = providerId;
			try {
				await $bridge.call('scim.triggerSync', {
					appId: '%s',
					providerId: providerId,
					fullSync: false
				});
				alert('Synchronization started');
				await this.loadProviders();
			} catch (err) {
				alert('Failed to trigger sync: ' + (err.message || 'Unknown error'));
			} finally {
				this.syncing = null;
			}
		},
		
		async testConnection(providerId) {
			try {
				const result = await $bridge.call('scim.testConnection', {
					appId: '%s',
					providerId: providerId
				});
				if (result.success) {
					alert('Connection successful! Response time: ' + result.responseTime + 'ms');
				} else {
					alert('Connection failed: ' + result.message);
				}
			} catch (err) {
				alert('Failed to test connection: ' + (err.message || 'Unknown error'));
			}
		},
		
		formatRelativeTime(timestamp) {
			if (!timestamp) return 'Never';
			const date = new Date(timestamp);
			const now = new Date();
			const diff = Math.floor((now - date) / 1000);
			
			if (diff < 60) return 'just now';
			if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
			if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
			return Math.floor(diff / 86400) + 'd ago';
		}
	}`, appID, appID, appID)
}

// AddProviderPage renders the add provider form.
func AddProviderPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center gap-4"),
			A(
				Href(appBase+"/scim/providers"),
				Class("p-2 rounded-lg hover:bg-accent"),
				lucide.ArrowLeft(Class("size-5")),
			),
			Div(
				H1(Class("text-2xl font-bold tracking-tight"), g.Text("Add Identity Provider")),
				P(Class("text-muted-foreground"), g.Text("Configure a new SCIM identity provider")),
			),
		),

		// Form
		Div(
			g.Attr("x-data", addProviderData(appID, appBase)),
			Class("max-w-2xl"),

			card.Card(
				card.Content(
					Class("space-y-6"),

					// Provider type selection
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Provider Type")),
						Div(
							Class("grid grid-cols-2 md:grid-cols-4 gap-3"),
							providerTypeOption("okta", "Okta", lucide.Shield(Class("size-6"))),
							providerTypeOption("azure", "Azure AD", lucide.Cloud(Class("size-6"))),
							providerTypeOption("onelogin", "OneLogin", lucide.Users(Class("size-6"))),
							providerTypeOption("custom", "Custom", lucide.Settings(Class("size-6"))),
						),
					),

					// Name
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Provider Name")),
						input.Input(
							input.WithType("text"),
							input.WithPlaceholder("e.g., Corporate Okta"),
							input.WithAttrs(g.Attr("x-model", "form.name")),
						),
					),

					// Endpoint URL
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("SCIM Endpoint URL")),
						input.Input(
							input.WithType("url"),
							input.WithPlaceholder("https://your-idp.com/scim/v2"),
							input.WithAttrs(g.Attr("x-model", "form.endpointUrl")),
						),
						P(Class("text-xs text-muted-foreground mt-1"), g.Text("The base URL for SCIM API requests")),
					),

					// Auth method
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Authentication Method")),
						Select(
							Class("w-full px-3 py-2 border rounded-md bg-background"),
							g.Attr("x-model", "form.authMethod"),
							Option(Value("bearer"), g.Text("Bearer Token")),
							Option(Value("oauth2"), g.Text("OAuth 2.0")),
							Option(Value("basic"), g.Text("Basic Auth")),
						),
					),

					// Sync options
					Div(
						Class("space-y-3"),
						Label(Class("text-sm font-medium block"), g.Text("Sync Options")),
						Label(
							Class("flex items-center gap-2 cursor-pointer"),
							Input(
								Type("checkbox"),
								g.Attr("x-model", "form.enableUserSync"),
								Class("rounded"),
							),
							Span(Class("text-sm"), g.Text("Enable user synchronization")),
						),
						Label(
							Class("flex items-center gap-2 cursor-pointer"),
							Input(
								Type("checkbox"),
								g.Attr("x-model", "form.enableGroupSync"),
								Class("rounded"),
							),
							Span(Class("text-sm"), g.Text("Enable group synchronization")),
						),
					),
				),
				card.Footer(
					Class("flex items-center justify-between"),
					A(
						Href(appBase+"/scim/providers"),
						button.Button(
							g.Text("Cancel"),
							button.WithVariant("outline"),
						),
					),
					button.Button(
						Div(
							Class("flex items-center gap-2"),
							lucide.Plus(Class("size-4")),
							Span(g.Attr("x-text", "submitting ? 'Creating...' : 'Create Provider'")),
						),
						button.WithVariant("default"),
						button.WithAttrs(
							g.Attr("@click", "createProvider()"),
							g.Attr(":disabled", "submitting"),
						),
					),
				),
			),

			// Token display modal
			g.El("template",
				g.Attr("x-if", "createdToken"),
				Div(
					Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
					card.Card(
						Class("w-full max-w-lg mx-4"),
						card.Header(
							card.Title("Provider Created Successfully"),
							card.Description("Save this token now. It will only be shown once."),
						),
						card.Content(
							Class("space-y-4"),
							Div(
								Class("p-4 bg-muted rounded-lg font-mono text-sm break-all"),
								g.Attr("x-text", "createdToken"),
							),
							button.Button(
								Div(Class("flex items-center gap-2"), lucide.Copy(Class("size-4")), g.Text("Copy Token")),
								button.WithVariant("outline"),
								button.WithAttrs(
									Class("w-full"),
									g.Attr("@click", "copyToken()"),
								),
							),
						),
						card.Footer(
							A(
								Href(appBase+"/scim/providers"),
								button.Button(
									g.Text("Done"),
									button.WithVariant("default"),
									button.WithAttrs(Class("w-full")),
								),
							),
						),
					),
				),
			),
		),
	)
}

// providerTypeOption creates a provider type selection option.
func providerTypeOption(value, label string, icon g.Node) g.Node {
	return Label(
		Class("flex flex-col items-center gap-2 p-4 border rounded-lg cursor-pointer transition-colors"),
		g.Attr(":class", fmt.Sprintf("form.type === '%s' ? 'border-primary bg-primary/5' : 'hover:bg-accent'", value)),
		Input(
			Type("radio"),
			Name("providerType"),
			Value(value),
			g.Attr("x-model", "form.type"),
			Class("sr-only"),
		),
		g.Group([]g.Node{icon}),
		Span(Class("text-sm font-medium"), g.Text(label)),
	)
}

// addProviderData returns the Alpine.js data for the add provider form.
func addProviderData(appID, appBase string) string {
	return fmt.Sprintf(`{
		form: {
			type: 'okta',
			name: '',
			endpointUrl: '',
			authMethod: 'bearer',
			enableUserSync: true,
			enableGroupSync: true
		},
		submitting: false,
		createdToken: null,
		
		async createProvider() {
			if (!this.form.name || !this.form.endpointUrl) {
				alert('Please fill in all required fields');
				return;
			}
			
			this.submitting = true;
			try {
				const result = await $bridge.call('scim.createProvider', {
					appId: '%s',
					name: this.form.name,
					type: this.form.type,
					endpointUrl: this.form.endpointUrl,
					authMethod: this.form.authMethod,
					enableUserSync: this.form.enableUserSync,
					enableGroupSync: this.form.enableGroupSync
				});
				this.createdToken = result.token;
			} catch (err) {
				alert('Failed to create provider: ' + (err.message || 'Unknown error'));
			} finally {
				this.submitting = false;
			}
		},
		
		copyToken() {
			navigator.clipboard.writeText(this.createdToken);
			alert('Token copied to clipboard');
		}
	}`, appID)
}
