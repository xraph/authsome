package social

import (
	"context"
	"fmt"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServeProvidersListPage renders the social providers list page
func (e *DashboardExtension) ServeProvidersListPage(ctx *router.PageContext) (g.Node, error) {

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	envID, err := e.getCurrentEnvironmentID(ctx, currentApp.ID)
	if err != nil {
		return nil, errs.BadRequest("Invalid environment context")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()

	// Get configured providers for this environment
	configs, err := e.configRepo.ListByEnvironment(reqCtx, currentApp.ID, envID)
	if err != nil {
		configs = []*schema.SocialProviderConfig{}
	}

	// Build a map of configured providers
	configuredProviders := make(map[string]*schema.SocialProviderConfig)
	for _, cfg := range configs {
		configuredProviders[cfg.ProviderName] = cfg
	}

	// Count enabled providers
	enabledCount := 0
	for _, cfg := range configs {
		if cfg.IsEnabled {
			enabledCount++
		}
	}

	content := Div(
		Class("space-y-6"),

		// Page header
		components.SettingsPageHeader("Social Providers", "Configure OAuth social authentication providers for your application"),

		// Stats cards
		Div(
			Class("grid gap-4 md:grid-cols-3"),
			e.renderStatCard("Configured", fmt.Sprintf("%d", len(configs)), lucide.Settings(Class("size-5 text-blue-500"))),
			e.renderStatCard("Enabled", fmt.Sprintf("%d", enabledCount), lucide.Check(Class("size-5 text-green-500"))),
			e.renderStatCard("Available", fmt.Sprintf("%d", len(schema.SupportedProviders)), lucide.Globe(Class("size-5 text-violet-500"))),
		),

		// Add provider button
		Div(
			Class("flex justify-end"),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/social/add"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 transition-colors"),
				lucide.Plus(Class("size-4")),
				g.Text("Add Provider"),
			),
		),

		// Providers grid
		Div(
			Class("grid gap-4 md:grid-cols-2 lg:grid-cols-3"),
			g.Group(e.renderProviderCards(reqCtx, basePath, currentApp.ID.String(), configuredProviders)),
		),
	)

	return content, nil
}

// ServeProviderAddPage renders the add provider form
func (e *DashboardExtension) ServeProviderAddPage(ctx *router.PageContext) (g.Node, error) {

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	envID, err := e.getCurrentEnvironmentID(ctx, currentApp.ID)
	if err != nil {
		return nil, errs.BadRequest("Invalid environment context")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()

	// Get already configured providers
	existingConfigs, _ := e.configRepo.ListByEnvironment(reqCtx, currentApp.ID, envID)
	configuredProviders := make(map[string]bool)
	for _, cfg := range existingConfigs {
		configuredProviders[cfg.ProviderName] = true
	}

	// Get available (unconfigured) providers
	availableProviders := []string{}
	for _, provider := range schema.SupportedProviders {
		if !configuredProviders[provider] {
			availableProviders = append(availableProviders, provider)
		}
	}

	// Pre-select provider from query param
	selectedProvider := ctx.Query("provider")

	content := Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/social"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Social Providers"),
		),

		// Header
		H1(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text("Add Social Provider")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Configure a new OAuth social authentication provider")),

		// Form
		FormEl(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/social/create"),
			Class("space-y-6"),

			// Provider selection
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("Provider")),

				Div(
					Label(For("provider_name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Select Provider")),
					Select(
						Name("provider_name"),
						ID("provider_name"),
						Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						g.Attr("onchange", "updateDefaultScopes(this.value)"),
						Option(Value(""), g.Text("Choose a provider...")),
						g.Group(func() []g.Node {
							nodes := make([]g.Node, 0, len(availableProviders))
							for _, provider := range availableProviders {
								displayName := schema.GetProviderDisplayName(provider)
								nodes = append(nodes, Option(
									Value(provider),
									g.If(provider == selectedProvider, g.Attr("selected", "")),
									g.Text(displayName),
								))
							}
							return nodes
						}()),
					),
					g.If(len(availableProviders) == 0,
						P(Class("mt-2 text-sm text-amber-600 dark:text-amber-400"),
							g.Text("All providers are already configured for this environment.")),
					),
				),
			),

			// OAuth Credentials
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("OAuth Credentials")),

				Div(
					Class("space-y-4"),

					// Client ID
					Div(
						Label(For("client_id"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Client ID")),
						Input(
							Type("text"),
							Name("client_id"),
							ID("client_id"),
							Required(),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("Enter your OAuth client ID"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("The client ID from your OAuth provider")),
					),

					// Client Secret
					Div(
						Label(For("client_secret"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Client Secret")),
						Input(
							Type("password"),
							Name("client_secret"),
							ID("client_secret"),
							Required(),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("Enter your OAuth client secret"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("The client secret from your OAuth provider (will be encrypted)")),
					),
				),
			),

			// Optional Settings
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("Optional Settings")),

				Div(
					Class("space-y-4"),

					// Custom Redirect URL
					Div(
						Label(For("redirect_url"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Custom Redirect URL (Optional)")),
						Input(
							Type("url"),
							Name("redirect_url"),
							ID("redirect_url"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("https://example.com/api/auth/callback/provider"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("Leave empty to use the default callback URL")),
					),

					// Scopes
					Div(
						Label(For("scopes"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("OAuth Scopes")),
						Input(
							Type("text"),
							Name("scopes"),
							ID("scopes"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("openid email profile"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("Space-separated list of OAuth scopes (leave empty for defaults)")),
					),

					// Display Name
					Div(
						Label(For("display_name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Name (Optional)")),
						Input(
							Type("text"),
							Name("display_name"),
							ID("display_name"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("Custom button label"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("Custom name shown on login button")),
					),

					// Enable immediately
					Div(
						Class("flex items-center"),
						Input(
							Type("checkbox"),
							Name("is_enabled"),
							ID("is_enabled"),
							Value("true"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-600 dark:bg-gray-700"),
						),
						Label(
							For("is_enabled"),
							Class("ml-2 text-sm text-slate-700 dark:text-gray-300"),
							g.Text("Enable provider immediately after creation"),
						),
					),
				),
			),

			// Submit buttons
			Div(
				Class("flex justify-end gap-4"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/social"),
					Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700"),
					g.Text("Add Provider"),
				),
			),
		),

		// JavaScript for updating default scopes based on provider selection
		Script(g.Raw(`
			function updateDefaultScopes(provider) {
				const defaultScopes = {
					'google': 'openid email profile',
					'github': 'user:email read:user',
					'microsoft': 'openid email profile User.Read',
					'apple': 'name email',
					'facebook': 'email public_profile',
					'discord': 'identify email',
					'twitter': 'users.read tweet.read',
					'linkedin': 'openid profile email',
					'spotify': 'user-read-email user-read-private',
					'twitch': 'user:read:email',
					'dropbox': 'account_info.read',
					'gitlab': 'read_user openid email',
					'line': 'profile openid email',
					'reddit': 'identity',
					'slack': 'users:read users:read.email',
					'bitbucket': 'account email',
					'notion': ''
				};
				const scopesInput = document.getElementById('scopes');
				if (scopesInput && defaultScopes[provider]) {
					scopesInput.placeholder = defaultScopes[provider] + ' (default)';
				}
			}
		`)),
	)

	return content, nil
}

// ServeProviderEditPage renders the edit provider form
func (e *DashboardExtension) ServeProviderEditPage(ctx *router.PageContext) (g.Node, error) {

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	configID, err := parseXID(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid provider ID")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()

	// Get the provider config
	config, err := e.configRepo.FindByID(reqCtx, configID)
	if err != nil {
		return nil, errs.NotFound("Provider not found")
	}

	// Get scopes as space-separated string
	scopesStr := strings.Join(config.Scopes, " ")

	content := Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/social"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Social Providers"),
		),

		// Header
		H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text(fmt.Sprintf("Edit %s Provider", config.GetDisplayName()))),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Update OAuth configuration for this provider")),

		// Form
		FormEl(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/social/"+config.ID.String()+"/update"),
			Class("space-y-6"),

			// Provider info (read-only)
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("Provider")),

				Div(
					Class("flex items-center gap-3"),
					e.getProviderIcon(config.ProviderName),
					Div(
						Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(schema.GetProviderDisplayName(config.ProviderName))),
						Div(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(config.ProviderName)),
					),
				),
			),

			// OAuth Credentials
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("OAuth Credentials")),

				Div(
					Class("space-y-4"),

					// Client ID
					Div(
						Label(For("client_id"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Client ID")),
						Input(
							Type("text"),
							Name("client_id"),
							ID("client_id"),
							Value(config.ClientID),
							Required(),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),

					// Client Secret
					Div(
						Label(For("client_secret"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Client Secret")),
						Input(
							Type("password"),
							Name("client_secret"),
							ID("client_secret"),
							Placeholder("••••••••"+config.MaskClientSecret()),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("Leave empty to keep the current secret")),
					),
				),
			),

			// Optional Settings
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),

				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("Optional Settings")),

				Div(
					Class("space-y-4"),

					// Custom Redirect URL
					Div(
						Label(For("redirect_url"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Custom Redirect URL")),
						Input(
							Type("url"),
							Name("redirect_url"),
							ID("redirect_url"),
							Value(config.RedirectURL),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("https://example.com/api/auth/callback/provider"),
						),
					),

					// Scopes
					Div(
						Label(For("scopes"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("OAuth Scopes")),
						Input(
							Type("text"),
							Name("scopes"),
							ID("scopes"),
							Value(scopesStr),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("openid email profile"),
						),
					),

					// Display Name
					Div(
						Label(For("display_name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Name")),
						Input(
							Type("text"),
							Name("display_name"),
							ID("display_name"),
							Value(config.DisplayName),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),

					// Enabled
					Div(
						Class("flex items-center"),
						Input(
							Type("checkbox"),
							Name("is_enabled"),
							ID("is_enabled"),
							Value("true"),
							g.If(config.IsEnabled, Checked()),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-600 dark:bg-gray-700"),
						),
						Label(
							For("is_enabled"),
							Class("ml-2 text-sm text-slate-700 dark:text-gray-300"),
							g.Text("Provider enabled"),
						),
					),
				),
			),

			// Submit buttons
			Div(
				Class("flex justify-between"),
				// Delete button
				Button(
					Type("button"),
					Class("px-4 py-2 text-sm font-medium text-red-600 bg-white border border-red-300 rounded-lg hover:bg-red-50 dark:bg-gray-800 dark:text-red-400 dark:border-red-600 dark:hover:bg-red-900/20"),
					g.Attr("onclick", "if(confirm('Are you sure you want to delete this provider?')) { document.getElementById('delete-form').submit(); }"),
					g.Text("Delete Provider"),
				),
				Div(
					Class("flex gap-4"),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/social"),
						Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700"),
						g.Text("Save Changes"),
					),
				),
			),
		),

		// Hidden delete form
		FormEl(
			ID("delete-form"),
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/social/"+config.ID.String()+"/delete"),
		),
	)

	return content, nil
}

// Helper: render provider cards
func (e *DashboardExtension) renderProviderCards(ctx context.Context, basePath, appID string, configured map[string]*schema.SocialProviderConfig) []g.Node {
	cards := make([]g.Node, 0, len(schema.SupportedProviders))

	for _, provider := range schema.SupportedProviders {
		config, isConfigured := configured[provider]
		cards = append(cards, e.renderProviderCard(basePath, appID, provider, config, isConfigured))
	}

	return cards
}

func (e *DashboardExtension) renderProviderCard(basePath, appID, providerName string, config *schema.SocialProviderConfig, isConfigured bool) g.Node {
	displayName := schema.GetProviderDisplayName(providerName)

	// Determine status
	var statusBadge g.Node
	var cardClasses string
	var actionsNode g.Node

	if isConfigured && config != nil && config.IsEnabled {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"),
			lucide.Check(Class("size-3")),
			g.Text("Enabled"),
		)
		cardClasses = "rounded-lg border-2 border-green-200 bg-white p-4 dark:border-green-800 dark:bg-gray-900 hover:shadow-md transition-shadow"
	} else if isConfigured && config != nil {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"),
			lucide.Pause(Class("size-3")),
			g.Text("Disabled"),
		)
		cardClasses = "rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900 hover:shadow-md transition-shadow"
	} else {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-400"),
			g.Text("Not configured"),
		)
		cardClasses = "rounded-lg border border-dashed border-slate-300 bg-slate-50 p-4 dark:border-gray-700 dark:bg-gray-900/50 hover:border-slate-400 transition-colors"
	}

	// Build actions based on configuration state
	if isConfigured && config != nil {
		configID := config.ID.String()
		var toggleButton g.Node
		if config.IsEnabled {
			toggleButton = Button(
				Type("submit"),
				Class("inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400 dark:hover:bg-amber-900/50"),
				lucide.Pause(Class("size-3")),
				g.Text("Disable"),
			)
		} else {
			toggleButton = Button(
				Type("submit"),
				Class("inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400 dark:hover:bg-green-900/50"),
				lucide.Play(Class("size-3")),
				g.Text("Enable"),
			)
		}

		actionsNode = g.Group([]g.Node{
			// Edit button
			A(
				Href(basePath+"/app/"+appID+"/social/"+configID+"/edit"),
				Class("inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.Pencil(Class("size-3")),
				g.Text("Edit"),
			),
			// Toggle form
			FormEl(
				Method("POST"),
				Action(basePath+"/app/"+appID+"/social/"+configID+"/toggle"),
				Class("inline"),
				toggleButton,
			),
		})
	} else {
		actionsNode = A(
			Href(basePath+"/app/"+appID+"/social/add?provider="+providerName),
			Class("inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-400 dark:hover:bg-violet-900/50"),
			lucide.Plus(Class("size-3")),
			g.Text("Configure"),
		)
	}

	return Div(
		Class(cardClasses),
		Div(
			Class("flex items-start justify-between"),
			Div(
				Class("flex items-center gap-3"),
				e.getProviderIcon(providerName),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(displayName)),
					statusBadge,
				),
			),
		),

		// Actions
		Div(
			Class("mt-4 flex gap-2"),
			actionsNode,
		),
	)
}

func (e *DashboardExtension) renderStatCard(label, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(label)),
				Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
			),
			icon,
		),
	)
}

func (e *DashboardExtension) getProviderIcon(providerName string) g.Node {
	iconClass := "size-8 p-1.5 rounded-lg"

	switch providerName {
	case "google":
		return Div(Class(iconClass+" bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400"), lucide.Chrome(Class("size-full")))
	case "github":
		return Div(Class(iconClass+" bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"), lucide.Github(Class("size-full")))
	case "microsoft":
		return Div(Class(iconClass+" bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"), lucide.Monitor(Class("size-full")))
	case "apple":
		return Div(Class(iconClass+" bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"), lucide.Apple(Class("size-full")))
	case "facebook":
		return Div(Class(iconClass+" bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"), lucide.Facebook(Class("size-full")))
	case "discord":
		return Div(Class(iconClass+" bg-indigo-100 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400"), lucide.MessageCircle(Class("size-full")))
	case "twitter":
		return Div(Class(iconClass+" bg-sky-100 text-sky-500 dark:bg-sky-900/30 dark:text-sky-400"), lucide.Twitter(Class("size-full")))
	case "linkedin":
		return Div(Class(iconClass+" bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"), lucide.Linkedin(Class("size-full")))
	case "spotify":
		return Div(Class(iconClass+" bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400"), lucide.Music(Class("size-full")))
	case "twitch":
		return Div(Class(iconClass+" bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400"), lucide.Twitch(Class("size-full")))
	case "dropbox":
		return Div(Class(iconClass+" bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"), lucide.Droplet(Class("size-full")))
	case "gitlab":
		return Div(Class(iconClass+" bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400"), lucide.GitBranch(Class("size-full")))
	case "line":
		return Div(Class(iconClass+" bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400"), lucide.MessageSquare(Class("size-full")))
	case "reddit":
		return Div(Class(iconClass+" bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400"), lucide.CircleDot(Class("size-full")))
	case "slack":
		return Div(Class(iconClass+" bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400"), lucide.Slack(Class("size-full")))
	case "bitbucket":
		return Div(Class(iconClass+" bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"), lucide.GitMerge(Class("size-full")))
	case "notion":
		return Div(Class(iconClass+" bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"), lucide.FileText(Class("size-full")))
	default:
		return Div(Class(iconClass+" bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-400"), lucide.Globe(Class("size-full")))
	}
}
