package oidcprovider

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/plugins/oidcprovider/bridge"
	"github.com/xraph/authsome/plugins/oidcprovider/pages"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the OIDC provider plugin
type DashboardExtension struct {
	clientRepo     *repository.OAuthClientRepository
	tokenRepo      *repository.OAuthTokenRepository
	consentRepo    *repository.OAuthConsentRepository
	deviceCodeRepo *repository.DeviceCodeRepository
	service        *Service
	logger         forge.Logger
	bridgeManager  *bridge.BridgeManager
	pagesManager   *pages.PagesManager
}

// NewDashboardExtension creates a new dashboard extension
func NewDashboardExtension(
	clientRepo *repository.OAuthClientRepository,
	tokenRepo *repository.OAuthTokenRepository,
	consentRepo *repository.OAuthConsentRepository,
	deviceCodeRepo *repository.DeviceCodeRepository,
	service *Service,
	logger forge.Logger,
) *DashboardExtension {
	ext := &DashboardExtension{
		clientRepo:     clientRepo,
		tokenRepo:      tokenRepo,
		consentRepo:    consentRepo,
		deviceCodeRepo: deviceCodeRepo,
		service:        service,
		logger:         logger,
	}

	// Initialize bridge manager
	ext.bridgeManager = bridge.NewBridgeManager(
		clientRepo,
		tokenRepo,
		consentRepo,
		deviceCodeRepo,
		service,
		logger,
	)

	// Initialize pages manager
	ext.pagesManager = pages.NewPagesManager(ext.bridgeManager, logger)

	return ext
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "oidcprovider"
}

// SetBaseUIPath sets the base UI path for navigation links
func (e *DashboardExtension) SetBaseUIPath(baseUIPath string) {
	if e.pagesManager != nil {
		e.pagesManager.SetBaseUIPath(baseUIPath)
	}
}

// NavigationItems returns navigation items for OAuth & OIDC section
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "oauth-oidc",
			Label:    "OAuth & OIDC",
			Icon:     lucide.Shield(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    55, // After API Keys (50), before others
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/oauth/clients"
				}
				return basePath + "/app/" + currentApp.ID.String() + "/oauth/clients"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "oauth-clients" ||
					activePage == "oauth-client-detail" ||
					activePage == "oauth-client-create" ||
					activePage == "oauth-device-flow" ||
					activePage == "oauth-settings"
			},
			RequiresPlugin: "oidcprovider",
		},
	}
}

// Routes returns dashboard routes for OIDC provider pages
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// OAuth Clients List
		{
			Method:       "GET",
			Path:         "/oauth/clients",
			Handler:      e.pagesManager.ClientsListPage,
			Name:         "oidc.dashboard.clients.list",
			Summary:      "OAuth Clients",
			Description:  "View and manage OAuth2/OIDC clients",
			Tags:         []string{"Dashboard", "OIDC", "OAuth"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Client
		{
			Method:       "GET",
			Path:         "/oauth/clients/new",
			Handler:      e.pagesManager.ClientCreatePage,
			Name:         "oidc.dashboard.clients.create",
			Summary:      "Create OAuth Client",
			Description:  "Create a new OAuth2/OIDC client",
			Tags:         []string{"Dashboard", "OIDC", "OAuth"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Client Detail
		{
			Method:       "GET",
			Path:         "/oauth/clients/:clientId",
			Handler:      e.pagesManager.ClientDetailPage,
			Name:         "oidc.dashboard.clients.detail",
			Summary:      "OAuth Client Details",
			Description:  "View OAuth2/OIDC client details",
			Tags:         []string{"Dashboard", "OIDC", "OAuth"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Device Flow Monitor
		{
			Method:       "GET",
			Path:         "/oauth/device-flow",
			Handler:      e.pagesManager.DeviceFlowMonitorPage,
			Name:         "oidc.dashboard.device-flow",
			Summary:      "Device Flow Monitor",
			Description:  "Monitor active device authorization flows",
			Tags:         []string{"Dashboard", "OIDC", "Device Flow"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// OIDC Settings (legacy path)
		{
			Method:       "GET",
			Path:         "/oauth/settings",
			Handler:      e.pagesManager.SettingsPage,
			Name:         "oidc.dashboard.settings",
			Summary:      "OIDC Settings",
			Description:  "Configure OIDC provider settings",
			Tags:         []string{"Dashboard", "OIDC", "Settings"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// OIDC Settings (settings page path - matches SettingsPages() Path)
		{
			Method:       "GET",
			Path:         "/settings/oauth",
			Handler:      e.pagesManager.SettingsPage,
			Name:         "oidc.settings.oauth",
			Summary:      "OAuth & OIDC Settings",
			Description:  "Configure OAuth2/OIDC provider settings",
			Tags:         []string{"Dashboard", "Settings", "OAuth", "OIDC"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, returning empty)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{}
}

// SettingsPages returns full settings pages
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "oauth-oidc",
			Label:         "OAuth & OIDC",
			Description:   "Manage OAuth2/OIDC clients and settings",
			Icon:          lucide.Shield(Class("size-4")),
			Category:      "security",
			Order:         20, // After API Keys
			Path:          "oauth",
			RequirePlugin: "oidcprovider",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets for the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "oauth-stats",
			Title: "OAuth & OIDC",
			Icon:  lucide.Shield(Class("size-5")),
			Order: 60,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return Div(
					Class("space-y-3"),
					// Active clients stat
					Div(
						Class("flex items-center justify-between"),
						Span(Class("text-sm text-gray-600 dark:text-gray-400"), g.Text("OAuth Clients")),
						Span(
							Class("text-lg font-semibold text-gray-900 dark:text-gray-100"),
							g.Attr("x-data", `{
								count: 0,
								async load() {
									try {
										const result = await $bridge.call('oidcprovider.getStats', {
											appId: '`+currentApp.ID.String()+`'
										});
										this.count = result.data.clientCount || 0;
									} catch (err) {
										console.error('Failed to load OAuth stats:', err);
									}
								}
							}`),
							g.Attr("x-init", "load()"),
							g.Attr("x-text", "count"),
						),
					),
					// Active tokens stat
					Div(
						Class("flex items-center justify-between"),
						Span(Class("text-sm text-gray-600 dark:text-gray-400"), g.Text("Active Tokens")),
						Span(
							Class("text-lg font-semibold text-gray-900 dark:text-gray-100"),
							g.Attr("x-data", `{
								count: 0,
								async load() {
									try {
										const result = await $bridge.call('oidcprovider.getStats', {
											appId: '`+currentApp.ID.String()+`'
										});
										this.count = result.data.activeTokens || 0;
									} catch (err) {
										console.error('Failed to load OAuth stats:', err);
									}
								}
							}`),
							g.Attr("x-init", "load()"),
							g.Attr("x-text", "count"),
						),
					),
					// View all link
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/oauth/clients"),
						Class("text-sm text-primary hover:text-primary-dark transition-colors"),
						g.Text("View all clients →"),
					),
				)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for OIDC provider
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		// Client management
		{
			Name:        "getClients",
			Handler:     e.bridgeManager.GetClients,
			Description: "List OAuth2/OIDC clients with pagination and search",
		},
		{
			Name:        "getClient",
			Handler:     e.bridgeManager.GetClient,
			Description: "Get OAuth2/OIDC client details",
		},
		{
			Name:        "createClient",
			Handler:     e.bridgeManager.CreateClient,
			Description: "Create a new OAuth2/OIDC client",
		},
		{
			Name:        "updateClient",
			Handler:     e.bridgeManager.UpdateClient,
			Description: "Update OAuth2/OIDC client",
		},
		{
			Name:        "deleteClient",
			Handler:     e.bridgeManager.DeleteClient,
			Description: "Delete OAuth2/OIDC client and revoke all tokens",
		},
		{
			Name:        "regenerateSecret",
			Handler:     e.bridgeManager.RegenerateSecret,
			Description: "Regenerate client secret",
		},
		{
			Name:        "getClientStats",
			Handler:     e.bridgeManager.GetClientStats,
			Description: "Get client usage statistics",
		},

		// Device flow
		{
			Name:        "getDeviceCodes",
			Handler:     e.bridgeManager.GetDeviceCodes,
			Description: "List device authorization codes",
		},
		{
			Name:        "revokeDeviceCode",
			Handler:     e.bridgeManager.RevokeDeviceCode,
			Description: "Revoke a device authorization code",
		},
		{
			Name:        "cleanupExpiredDeviceCodes",
			Handler:     e.bridgeManager.CleanupExpiredDeviceCodes,
			Description: "Trigger cleanup of expired device codes",
		},

		// Settings
		{
			Name:        "getSettings",
			Handler:     e.bridgeManager.GetSettings,
			Description: "Get OIDC provider configuration",
		},
		{
			Name:        "updateTokenSettings",
			Handler:     e.bridgeManager.UpdateTokenSettings,
			Description: "Update token lifetime settings",
		},
		{
			Name:        "updateDeviceFlowSettings",
			Handler:     e.bridgeManager.UpdateDeviceFlowSettings,
			Description: "Update device flow configuration",
		},
		{
			Name:        "rotateKeys",
			Handler:     e.bridgeManager.RotateKeys,
			Description: "Trigger JWT key rotation",
		},

		// Stats
		{
			Name:        "getStats",
			Handler:     e.bridgeManager.GetStats,
			Description: "Get overall OAuth/OIDC statistics",
		},
	}
}
