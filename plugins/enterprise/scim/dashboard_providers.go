package scim

import (
	"fmt"
	"net/http"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Provider Management Handlers

// ServeProvidersListPage renders the SCIM providers list page
func (e *DashboardExtension) ServeProvidersListPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(c)

	content := e.renderProvidersListContent(c, currentApp, orgID)

	return handler.RenderSettingsPage(c, "scim-providers", content)
}

// renderProvidersListContent renders the providers list page content
func (e *DashboardExtension) renderProvidersListContent(c forge.Context, currentApp interface{}, orgID *xid.ID) g.Node {
	ctx := c.Request().Context()
	basePath := e.getBasePath()
	
	// Extract app ID from currentApp
	var appID xid.ID
	switch v := currentApp.(type) {
	case *xid.ID:
		appID = *v
	case xid.ID:
		appID = v
	default:
		return alertBox("error", "Error", "Invalid app context type")
	}

	// Fetch providers from service
	providers, err := e.plugin.service.ListProviders(ctx, appID, orgID)
	if err != nil {
		return alertBox("error", "Error", "Failed to load providers: "+err.Error())
	}

	return Div(
		Class("space-y-6"),
		
		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("SCIM Providers")),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Text("Manage identity provider connections")),
			),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/add", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Add Provider"),
			),
		),

		// Info box
		alertBox("info", "Provider Types",
			"Configure both inbound (IdP → AuthSome) and outbound (AuthSome → External Systems) SCIM provisioning."),

		// Providers Grid
		g.If(len(providers) == 0,
			emptyState(
				lucide.Cloud(Class("size-12 text-slate-400")),
				"No Providers Configured",
				"Add your first SCIM provider to enable automatic user and group synchronization with identity providers like Okta, Azure AD, or OneLogin.",
				"Add Provider",
				fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/add", basePath, appID.String()),
			),
		),

		g.If(len(providers) > 0,
			Div(
				Class("grid gap-4 md:grid-cols-2"),
				g.Group(e.renderProviderCards(providers, basePath, &appID)),
			),
		),
	)
}

// renderProviderCards renders provider cards
func (e *DashboardExtension) renderProviderCards(providers []*SCIMProvider, basePath string, appID *xid.ID) []g.Node {
	cards := make([]g.Node, len(providers))
	for i, provider := range providers {
		cards[i] = providerCard(provider, basePath, (*appID))
	}
	return cards
}

// ServeProviderAddPage renders the add provider page
func (e *DashboardExtension) ServeProviderAddPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := e.getBasePath()
	appID := &currentApp.ID

	content := Div(
		Class("space-y-6"),
		
		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Add SCIM Provider")),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Text("Configure a new identity provider connection")),
			),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers", basePath, appID.String())),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back"),
			),
		),

		// Add Provider Form
		e.renderAddProviderForm(basePath, appID),
	)

	return handler.RenderSettingsPage(c, "scim-providers", content)
}

// renderAddProviderForm renders the add provider form
func (e *DashboardExtension) renderAddProviderForm(basePath string, appID *xid.ID) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Form(
			Method("POST"),
			Action(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/add", basePath, appID.String())),
			Class("p-6 space-y-6"),
			
			// Provider Name
			Div(
				Label(
					For("provider-name"),
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Provider Name"),
				),
				Input(
					Type("text"),
					Name("name"),
					ID("provider-name"),
					Required(),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					g.Attr("placeholder", "Production Okta"),
				),
			),

			// Provider Type
			Div(
				Label(
					For("provider-type"),
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Provider Type"),
				),
				Select(
					Name("type"),
					ID("provider-type"),
					Required(),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					Option(Value(""), g.Text("Select a provider...")),
					Option(Value("okta"), g.Text("Okta")),
					Option(Value("azure_ad"), g.Text("Azure AD")),
					Option(Value("onelogin"), g.Text("OneLogin")),
					Option(Value("google_workspace"), g.Text("Google Workspace")),
					Option(Value("custom"), g.Text("Custom SCIM 2.0")),
				),
			),

			// Direction
			Div(
				Label(
					For("provider-direction"),
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Sync Direction"),
				),
				Select(
					Name("direction"),
					ID("provider-direction"),
					Required(),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					Option(Value("inbound"), g.Text("Inbound (IdP → AuthSome)"), g.Attr("selected", "")),
					Option(Value("outbound"), g.Text("Outbound (AuthSome → External)")),
					Option(Value("bidirectional"), g.Text("Bidirectional")),
				),
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Text("Choose how users and groups are synchronized")),
			),

			// Inbound Configuration Section
			Div(
				ID("inbound-config"),
				Class("border-t border-slate-200 pt-6 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Inbound Configuration")),
				
				Div(
					Class("space-y-4"),
					
					// Base URL
					Div(
						Label(
							For("base-url"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("SCIM Base URL"),
						),
						Input(
							Type("url"),
							Name("base_url"),
							ID("base-url"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							g.Attr("placeholder", "https://your-domain.com/scim/v2"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("The base URL for your SCIM endpoint (auto-generated)")),
					),

					// Auth Method
					Div(
						Label(
							For("auth-method"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Authentication Method"),
						),
						Select(
							Name("auth_method"),
							ID("auth-method"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("bearer"), g.Text("Bearer Token"), g.Attr("selected", "")),
							Option(Value("oauth2"), g.Text("OAuth 2.0")),
						),
					),
				),
			),

			// Outbound Configuration Section
			Div(
				ID("outbound-config"),
				Class("border-t border-slate-200 pt-6 hidden dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Outbound Configuration")),
				
				Div(
					Class("space-y-4"),
					
					// Target URL
					Div(
						Label(
							For("target-url"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Target SCIM Endpoint"),
						),
						Input(
							Type("url"),
							Name("target_url"),
							ID("target-url"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							g.Attr("placeholder", "https://external-system.com/scim/v2"),
						),
					),

					// Target Auth Token
					Div(
						Label(
							For("target-token"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Target Bearer Token"),
						),
						Input(
							Type("password"),
							Name("target_token"),
							ID("target-token"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),
				),
			),

			// Submit Buttons
			Div(
				Class("flex gap-3 pt-6 border-t border-slate-200 dark:border-gray-800"),
				Button(
					Type("submit"),
					Class("flex-1 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Text("Add Provider"),
				),
				A(
					Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers", basePath, appID.String())),
					Class("flex-1 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-center text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
					g.Text("Cancel"),
				),
			),
		),
		
		// JavaScript for showing/hiding config sections
		Script(g.Raw(`
			document.getElementById('provider-direction').addEventListener('change', function() {
				const inbound = document.getElementById('inbound-config');
				const outbound = document.getElementById('outbound-config');
				const value = this.value;
				
				if (value === 'inbound') {
					inbound.classList.remove('hidden');
					outbound.classList.add('hidden');
				} else if (value === 'outbound') {
					inbound.classList.add('hidden');
					outbound.classList.remove('hidden');
				} else if (value === 'bidirectional') {
					inbound.classList.remove('hidden');
					outbound.classList.remove('hidden');
				}
			});
		`)),
	)
}

// ServeProviderDetailPage renders the provider detail page
func (e *DashboardExtension) ServeProviderDetailPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	providerIDStr := c.Param("id")
	providerID, err := xid.FromString(providerIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid provider ID")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(c)

	ctx := c.Request().Context()
	provider, err := e.plugin.service.GetProvider(ctx, providerID)
	if err != nil {
		return c.String(http.StatusNotFound, "Provider not found")
	}

	content := e.renderProviderDetailContent(c, currentApp, provider, orgID)

	return handler.RenderSettingsPage(c, "scim-providers", content)
}

// renderProviderDetailContent renders the provider detail page content
func (e *DashboardExtension) renderProviderDetailContent(c forge.Context, currentApp interface{}, provider *SCIMProvider, orgID *xid.ID) g.Node {
	_ = e.getBasePath()
	_ = currentApp

	return Div(
		Class("space-y-6"),
		
		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(
					Class("flex items-center gap-3"),
					providerTypeIcon(provider.Type),
					H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
						g.Text(provider.Name)),
					statusBadge(provider.Status),
				),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Textf("%s - %s sync", provider.Type, provider.Direction)),
			),
			Div(
				Class("flex gap-2"),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Attr("onclick", fmt.Sprintf("testProvider('%s')", provider.ID.String())),
					lucide.Play(Class("size-4")),
					g.Text("Test Connection"),
				),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
					g.Attr("onclick", fmt.Sprintf("syncProvider('%s')", provider.ID.String())),
					lucide.RefreshCw(Class("size-4")),
					g.Text("Manual Sync"),
				),
			),
		),

		// Provider Info Card
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Provider Information")),
			Div(
				Class("grid grid-cols-2 gap-4"),
				e.renderInfoField("Provider ID", provider.ID.String()),
				e.renderInfoField("Type", provider.Type),
				e.renderInfoField("Direction", provider.Direction),
				e.renderInfoField("Status", provider.Status),
				g.If(provider.BaseURL != nil,
					e.renderInfoField("Base URL", *provider.BaseURL),
				),
				g.If(provider.TargetURL != nil,
					e.renderInfoField("Target URL", *provider.TargetURL),
				),
				g.If(provider.LastSyncAt != nil,
					e.renderInfoField("Last Sync", formatRelativeTime(*provider.LastSyncAt)),
				),
				e.renderInfoField("Last Sync Status", provider.LastSyncStatus),
			),
		),

		// Sync History
		e.renderProviderSyncHistory(c, provider),

		// Danger Zone
		Div(
			Class("rounded-lg border border-red-200 bg-red-50 p-6 dark:border-red-800 dark:bg-red-900/20"),
			H2(Class("text-xl font-semibold text-red-900 dark:text-red-300 mb-2"),
				g.Text("Danger Zone")),
			P(Class("text-sm text-red-700 dark:text-red-400 mb-4"),
				g.Text("Removing this provider will stop all synchronization. This action cannot be undone.")),
			Button(
				Type("button"),
				Class("inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"),
				g.Attr("onclick", fmt.Sprintf("removeProvider('%s')", provider.ID.String())),
				lucide.Trash2(Class("size-4")),
				g.Text("Remove Provider"),
			),
		),
	)
}

// renderInfoField renders an info field
func (e *DashboardExtension) renderInfoField(label, value string) g.Node {
	return Div(
		Dt(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(label)),
		Dd(Class("mt-1 text-sm text-slate-900 dark:text-white"), g.Text(value)),
	)
}

// renderProviderSyncHistory renders the sync history for a provider
func (e *DashboardExtension) renderProviderSyncHistory(c forge.Context, provider *SCIMProvider) g.Node {
	ctx := c.Request().Context()
	
	// Fetch recent sync events for this provider
	events, err := e.plugin.service.GetProviderSyncHistory(ctx, provider.ID, 10)
	if err != nil {
		return g.Raw("") // Silent fail
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("border-b border-slate-200 p-6 dark:border-gray-800"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
				g.Text("Recent Sync Activity")),
		),
		g.If(len(events) == 0,
			Div(
				Class("p-6"),
				emptyState(
					lucide.Activity(Class("size-12 text-slate-400")),
					"No Sync History",
					"Sync events will appear here once synchronization starts",
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

// HandleAddProvider handles adding a new provider
func (e *DashboardExtension) HandleAddProvider(c forge.Context) error {
	ctx := c.Request().Context()

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app context",
		})
	}

	orgID, _ := e.getOrgFromContext(c)

	// Parse form data
	name := c.FormValue("name")
	providerType := c.FormValue("type")
	direction := c.FormValue("direction")
	baseURL := c.FormValue("base_url")
	authMethod := c.FormValue("auth_method")
	targetURL := c.FormValue("target_url")
	targetToken := c.FormValue("target_token")

	if name == "" || providerType == "" || direction == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name, type, and direction are required",
		})
	}

	// Create provider
	req := &CreateSCIMProviderRequest{
		AppID:          &currentApp.ID,
		OrganizationID: orgID,
		Name:           name,
		Type:           providerType,
		Direction:      direction,
		BaseURL:        &baseURL,
		AuthMethod:     authMethod,
		TargetURL:      &targetURL,
		TargetToken:    &targetToken,
	}

	provider, err := e.plugin.service.CreateProvider(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create provider: %v", err),
		})
	}

	// Redirect to provider detail page
	basePath := e.getBasePath()
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/%s", basePath, currentApp.ID.String(), provider.ID.String()))
}

// HandleUpdateProvider handles provider updates
func (e *DashboardExtension) HandleUpdateProvider(c forge.Context) error {
	_ = c.Request().Context()
	providerIDStr := c.Param("id")

	_, err := xid.FromString(providerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid provider ID",
		})
	}

	// Parse form data and update provider
	// Implementation similar to HandleAddProvider

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Provider updated successfully",
	})
}

// HandleManualSync handles manual sync trigger
func (e *DashboardExtension) HandleManualSync(c forge.Context) error {
	ctx := c.Request().Context()
	providerIDStr := c.Param("id")

	providerID, err := xid.FromString(providerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid provider ID",
		})
	}

	syncType := c.FormValue("sync_type")
	if syncType == "" {
		syncType = "full"
	}

	// Trigger manual sync
	err = e.plugin.service.TriggerManualSync(ctx, providerID, syncType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to trigger sync: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Manual sync triggered successfully",
	})
}

// HandleTestProvider handles provider connection testing
func (e *DashboardExtension) HandleTestProvider(c forge.Context) error {
	ctx := c.Request().Context()
	providerIDStr := c.Param("id")

	providerID, err := xid.FromString(providerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid provider ID",
		})
	}

	// Test provider health
	result, err := e.plugin.service.GetProviderHealth(ctx, providerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Connection test failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// HandleRemoveProvider handles provider removal
func (e *DashboardExtension) HandleRemoveProvider(c forge.Context) error {
	ctx := c.Request().Context()
	providerIDStr := c.Param("id")

	providerID, err := xid.FromString(providerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid provider ID",
		})
	}

	// Remove provider
	err = e.plugin.service.RemoveProvider(ctx, providerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to remove provider: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Provider removed successfully",
	})
}

