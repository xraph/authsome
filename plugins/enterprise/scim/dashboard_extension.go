package scim

import (
	"fmt"
	"net/http"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the SCIM plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
}

// NewDashboardExtension creates a new dashboard extension for SCIM
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "scim"
}

// NavigationItems returns navigation items to register
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "scim-provisioning",
			Label:    "SCIM Provisioning",
			Icon:     lucide.Users(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    60,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/dashboard/scim"
				}
				return basePath + "/dashboard/app/" + currentApp.ID.String() + "/scim"
			},
			ActiveChecker: func(activePage string) bool {
				return strings.HasPrefix(activePage, "scim")
			},
			RequiresPlugin: "scim",
		},
	}
}

// Routes returns routes to register under /dashboard/app/:appId/
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Main SCIM Dashboard
		{
			Method:       "GET",
			Path:         "/scim",
			Handler:      e.ServeSCIMDashboard,
			Name:         "dashboard.scim.overview",
			Summary:      "SCIM provisioning dashboard",
			Description:  "View SCIM provisioning status and overview",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Sync Status Page
		{
			Method:       "GET",
			Path:         "/scim/status",
			Handler:      e.ServeSyncStatusPage,
			Name:         "dashboard.scim.status",
			Summary:      "Real-time sync status",
			Description:  "View real-time SCIM synchronization status",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Token Management Routes
		{
			Method:       "GET",
			Path:         "/settings/scim-tokens",
			Handler:      e.ServeTokensListPage,
			Name:         "dashboard.settings.scim-tokens",
			Summary:      "SCIM tokens management",
			Description:  "Manage SCIM bearer tokens for IdP authentication",
			Tags:         []string{"Dashboard", "Settings", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-tokens/create",
			Handler:      e.HandleCreateToken,
			Name:         "dashboard.settings.scim-tokens.create",
			Summary:      "Create SCIM token",
			Description:  "Create a new SCIM provisioning token",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-tokens/:id/rotate",
			Handler:      e.HandleRotateToken,
			Name:         "dashboard.settings.scim-tokens.rotate",
			Summary:      "Rotate SCIM token",
			Description:  "Rotate an existing SCIM token",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-tokens/:id/revoke",
			Handler:      e.HandleRevokeToken,
			Name:         "dashboard.settings.scim-tokens.revoke",
			Summary:      "Revoke SCIM token",
			Description:  "Revoke a SCIM token",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-tokens/:id/test",
			Handler:      e.HandleTestConnection,
			Name:         "dashboard.settings.scim-tokens.test",
			Summary:      "Test SCIM connection",
			Description:  "Test SCIM endpoint connectivity",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Configuration Routes
		{
			Method:       "GET",
			Path:         "/settings/scim-config",
			Handler:      e.ServeConfigPage,
			Name:         "dashboard.settings.scim-config",
			Summary:      "SCIM configuration",
			Description:  "Configure SCIM provisioning settings",
			Tags:         []string{"Dashboard", "Settings", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-config/user-provisioning",
			Handler:      e.HandleUpdateUserProvisioning,
			Name:         "dashboard.settings.scim-config.user-provisioning",
			Summary:      "Update user provisioning settings",
			Description:  "Update SCIM user provisioning configuration",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-config/group-sync",
			Handler:      e.HandleUpdateGroupSync,
			Name:         "dashboard.settings.scim-config.group-sync",
			Summary:      "Update group sync settings",
			Description:  "Update SCIM group synchronization configuration",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-config/attribute-mapping",
			Handler:      e.HandleUpdateAttributeMapping,
			Name:         "dashboard.settings.scim-config.attribute-mapping",
			Summary:      "Update attribute mapping",
			Description:  "Configure SCIM attribute mappings",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-config/security",
			Handler:      e.HandleUpdateSecurity,
			Name:         "dashboard.settings.scim-config.security",
			Summary:      "Update security settings",
			Description:  "Update SCIM security and rate limit settings",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Provider Management Routes
		{
			Method:       "GET",
			Path:         "/settings/scim-providers",
			Handler:      e.ServeProvidersListPage,
			Name:         "dashboard.settings.scim-providers",
			Summary:      "SCIM providers",
			Description:  "Manage SCIM identity providers",
			Tags:         []string{"Dashboard", "Settings", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/scim-providers/add",
			Handler:      e.ServeProviderAddPage,
			Name:         "dashboard.settings.scim-providers.add",
			Summary:      "Add SCIM provider",
			Description:  "Add a new SCIM provider",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-providers/add",
			Handler:      e.HandleAddProvider,
			Name:         "dashboard.settings.scim-providers.add.submit",
			Summary:      "Submit add provider",
			Description:  "Process add provider form",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/scim-providers/:id",
			Handler:      e.ServeProviderDetailPage,
			Name:         "dashboard.settings.scim-providers.detail",
			Summary:      "Provider details",
			Description:  "View SCIM provider details",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-providers/:id/update",
			Handler:      e.HandleUpdateProvider,
			Name:         "dashboard.settings.scim-providers.update",
			Summary:      "Update provider",
			Description:  "Update SCIM provider settings",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-providers/:id/sync",
			Handler:      e.HandleManualSync,
			Name:         "dashboard.settings.scim-providers.sync",
			Summary:      "Trigger manual sync",
			Description:  "Manually trigger SCIM synchronization",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-providers/:id/test",
			Handler:      e.HandleTestProvider,
			Name:         "dashboard.settings.scim-providers.test",
			Summary:      "Test provider connection",
			Description:  "Test SCIM provider connectivity",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/scim-providers/:id/remove",
			Handler:      e.HandleRemoveProvider,
			Name:         "dashboard.settings.scim-providers.remove",
			Summary:      "Remove provider",
			Description:  "Remove SCIM provider",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Monitoring Routes
		{
			Method:       "GET",
			Path:         "/settings/scim-monitoring",
			Handler:      e.ServeMonitoringPage,
			Name:         "dashboard.settings.scim-monitoring",
			Summary:      "SCIM monitoring",
			Description:  "Monitor SCIM synchronization and events",
			Tags:         []string{"Dashboard", "Settings", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/scim-monitoring/logs",
			Handler:      e.ServeLogsPage,
			Name:         "dashboard.settings.scim-monitoring.logs",
			Summary:      "SCIM event logs",
			Description:  "View SCIM provisioning event logs",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/scim-monitoring/stats",
			Handler:      e.ServeStatsPage,
			Name:         "dashboard.settings.scim-monitoring.stats",
			Summary:      "SCIM statistics",
			Description:  "View SCIM analytics and metrics",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/scim-monitoring/export",
			Handler:      e.HandleExportLogs,
			Name:         "dashboard.settings.scim-monitoring.export",
			Summary:      "Export logs",
			Description:  "Export SCIM logs as CSV/JSON",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{} // Using SettingsPages instead
}

// SettingsPages returns full settings pages for the sidebar layout
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "scim-tokens",
			Label:         "SCIM Tokens",
			Description:   "Manage bearer tokens for IdP authentication",
			Icon:          lucide.Key(Class("h-5 w-5")),
			Category:      "security",
			Order:         30,
			Path:          "scim-tokens",
			RequirePlugin: "scim",
			RequireAdmin:  true,
		},
		{
			ID:            "scim-config",
			Label:         "SCIM Configuration",
			Description:   "Configure user provisioning and group sync",
			Icon:          lucide.Settings(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         20,
			Path:          "scim-config",
			RequirePlugin: "scim",
			RequireAdmin:  true,
		},
		{
			ID:            "scim-providers",
			Label:         "SCIM Providers",
			Description:   "Manage identity provider connections",
			Icon:          lucide.Cloud(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         21,
			Path:          "scim-providers",
			RequirePlugin: "scim",
			RequireAdmin:  true,
		},
		{
			ID:            "scim-monitoring",
			Label:         "SCIM Monitoring",
			Description:   "View sync logs and analytics",
			Icon:          lucide.Activity(Class("h-5 w-5")),
			Category:      "advanced",
			Order:         10,
			Path:          "scim-monitoring",
			RequirePlugin: "scim",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "scim-status",
			Title: "SCIM Status",
			Icon:  lucide.Users(Class("size-5")),
			Order: 50,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderSCIMStatusWidget(basePath, currentApp)
			},
		},
		{
			ID:    "scim-sync-stats",
			Title: "Sync Statistics",
			Icon:  lucide.TrendingUp(Class("size-5")),
			Order: 51,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderSyncStatsWidget(basePath, currentApp)
			},
		},
	}
}

// Helper methods

// getUserFromContext extracts the current user from the request context
func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil
	}
	return handler.GetUserFromContext(c)
}

// extractAppFromURL extracts the app from the URL parameter
func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil, fmt.Errorf("handler not available")
	}
	return handler.GetCurrentApp(c)
}

// getBasePath returns the dashboard base path
func (e *DashboardExtension) getBasePath() string {
	if e.registry != nil && e.registry.GetHandler() != nil {
		return e.registry.GetHandler().GetBasePath()
	}
	return ""
}

// detectMode determines if we're in app or organization mode
func (e *DashboardExtension) detectMode() string {
	// Check if organization service is available (indicates organization mode)
	if e.plugin.orgService != nil {
		// Try to determine if it's organization.Service (org mode) or app.Service (app mode)
		// In app mode, the orgService is actually app.Service
		// This is a simple heuristic - in production you'd have a more reliable way
		return "organization" // For now, assume organization mode if orgService exists
	}
	return "app"
}

// getOrgFromContext tries to extract organization ID from context or URL
func (e *DashboardExtension) getOrgFromContext(c forge.Context) (*xid.ID, error) {
	// Try to get from URL parameter first
	orgIDStr := c.Param("orgId")
	if orgIDStr != "" {
		orgID, err := xid.FromString(orgIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid org ID: %w", err)
		}
		return &orgID, nil
	}

	// Try to get from query parameter
	orgIDStr = c.Query("orgId")
	if orgIDStr != "" {
		orgID, err := xid.FromString(orgIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid org ID: %w", err)
		}
		return &orgID, nil
	}

	// In app mode, return nil (org ID not needed)
	if e.detectMode() == "app" {
		return nil, nil
	}

	return nil, fmt.Errorf("organization ID required but not found")
}

// Main Dashboard Handlers

func (e *DashboardExtension) ServeSCIMDashboard(c forge.Context) error {
	// Get current app and user
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app context",
		})
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Get organization ID if in org mode
	orgID, _ := e.getOrgFromContext(c)

	// Fetch dashboard stats
	var stats *DashboardStats
	if orgID != nil {
		stats, err = e.plugin.service.GetDashboardStats(c.Request().Context(), currentApp.ID, orgID)
	} else {
		stats, err = e.plugin.service.GetDashboardStats(c.Request().Context(), currentApp.ID, nil)
	}
	if err != nil {
		stats = &DashboardStats{
			TotalSyncs:     0,
			SuccessRate:    0,
			FailedSyncs:    0,
			LastSyncTime:   "Never",
			LastSyncStatus: "unknown",
		}
	}

	// Fetch sync status
	var syncStatus *SyncStatus
	if orgID != nil {
		syncStatus, err = e.plugin.service.GetSyncStatus(c.Request().Context(), currentApp.ID, orgID)
	} else {
		syncStatus, err = e.plugin.service.GetSyncStatus(c.Request().Context(), currentApp.ID, nil)
	}
	if err != nil {
		syncStatus = &SyncStatus{
			IsHealthy:       false,
			ActiveProviders: 0,
			Status:          "unknown",
			Message:         "Unable to fetch status",
		}
	}

	// Fetch recent activity
	var recentActivity []*SCIMSyncEvent
	if orgID != nil {
		recentActivity, err = e.plugin.service.GetRecentActivity(c.Request().Context(), currentApp.ID, orgID, 5)
	} else {
		recentActivity, err = e.plugin.service.GetRecentActivity(c.Request().Context(), currentApp.ID, nil, 5)
	}
	if err != nil {
		recentActivity = []*SCIMSyncEvent{}
	}

	// Fetch failed operations
	var failedOps []*SCIMSyncEvent
	if orgID != nil {
		failedOps, err = e.plugin.service.GetFailedEvents(c.Request().Context(), currentApp.ID, orgID, 5)
	} else {
		failedOps, err = e.plugin.service.GetFailedEvents(c.Request().Context(), currentApp.ID, nil, 5)
	}
	if err != nil {
		failedOps = []*SCIMSyncEvent{}
	}

	basePath := e.getBasePath()

	// Render the dashboard with proper layout
	content := e.renderDashboardPage(basePath, currentApp, stats, syncStatus, recentActivity, failedOps)

	// Use dashboard handler's RenderWithLayout for proper sidebar layout
	handler := e.registry.GetHandler()
	if handler == nil {
		// Fallback to direct rendering if handler not available
		c.SetHeader("Content-Type", "text/html; charset=utf-8")
		c.Response().WriteHeader(http.StatusOK)
		return content.Render(c.Response())
	}

	return handler.RenderWithLayout(c, e.buildPageData(c, currentUser, currentApp, "scim", "SCIM Provisioning"), content)
}

func (e *DashboardExtension) ServeSyncStatusPage(c forge.Context) error {
	// Get current app
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app context",
		})
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Get organization ID if in org mode
	orgID, _ := e.getOrgFromContext(c)

	// Fetch sync status
	var syncStatus *SyncStatus
	if orgID != nil {
		syncStatus, err = e.plugin.service.GetSyncStatus(c.Request().Context(), currentApp.ID, orgID)
	} else {
		syncStatus, err = e.plugin.service.GetSyncStatus(c.Request().Context(), currentApp.ID, nil)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch sync status",
		})
	}

	basePath := e.getBasePath()

	content := e.renderSyncStatusPage(basePath, currentApp, syncStatus)

	// Use dashboard handler's RenderWithLayout
	handler := e.registry.GetHandler()
	if handler == nil {
		c.SetHeader("Content-Type", "text/html; charset=utf-8")
		c.Response().WriteHeader(http.StatusOK)
		return content.Render(c.Response())
	}

	return handler.RenderWithLayout(c, e.buildPageData(c, currentUser, currentApp, "scim", "Sync Status"), content)
}

// Dashboard page renderer

func (e *DashboardExtension) renderDashboardPage(basePath string, currentApp *app.App, stats *DashboardStats, syncStatus *SyncStatus, recentActivity, failedOps []*SCIMSyncEvent) g.Node {
	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white mb-2"),
				lucide.Users(Class("size-8 inline-block mr-3")),
				g.Text("SCIM Provisioning")),
			P(Class("text-slate-600 dark:text-gray-400"),
				g.Text("Automated user and group provisioning for enterprise identity providers")),
		),

		// Stats Grid
		Div(
			Class("grid grid-cols-1 md:grid-cols-4 gap-4"),
			e.renderStatCard("Total Syncs", fmt.Sprintf("%d", stats.TotalSyncs), lucide.RefreshCw(Class("size-6 text-indigo-600"))),
			e.renderStatCard("Success Rate", fmt.Sprintf("%.1f%%", stats.SuccessRate), lucide.Check(Class("size-6 text-green-600"))),
			e.renderStatCard("Failed", fmt.Sprintf("%d", stats.FailedSyncs), lucide.X(Class("size-6 text-red-600"))),
			e.renderStatCard("Active Providers", fmt.Sprintf("%d", syncStatus.ActiveProviders), lucide.Cloud(Class("size-6 text-blue-600"))),
		),

		// Status Card
		e.renderStatusCard(syncStatus),

		// Quick Actions
		e.renderQuickActionsCard(basePath, currentApp),

		// Two Column Layout
		Div(
			Class("grid grid-cols-1 lg:grid-cols-2 gap-6"),

			// Recent Activity
			Div(
				Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					lucide.Activity(Class("size-5 inline-block mr-2")),
					g.Text("Recent Activity")),
				g.If(len(recentActivity) == 0,
					P(Class("text-slate-600 dark:text-gray-400 text-center py-8"),
						g.Text("No recent activity")),
				),
				g.If(len(recentActivity) > 0,
					Div(
						Class("space-y-2"),
						g.Group(g.Map(recentActivity, func(event *SCIMSyncEvent) g.Node {
							return e.renderEventRow(event)
						})),
					),
				),
			),

			// Failed Operations
			Div(
				Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					lucide.Octagon(Class("size-5 inline-block mr-2")),
					g.Text("Failed Operations")),
				g.If(len(failedOps) == 0,
					P(Class("text-slate-600 dark:text-gray-400 text-center py-8"),
						g.Text("No failed operations")),
				),
				g.If(len(failedOps) > 0,
					Div(
						Class("space-y-2"),
						g.Group(g.Map(failedOps, func(event *SCIMSyncEvent) g.Node {
							return e.renderEventRow(event)
						})),
					),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderSyncStatusPage(basePath string, currentApp *app.App, syncStatus *SyncStatus) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white mb-4"),
				g.Text("Sync Status")),
			e.renderStatusCard(syncStatus),
		),
	)
}

func (e *DashboardExtension) renderStatCard(title, value string, icon g.Node) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm text-slate-600 dark:text-gray-400 mb-1"),
					g.Text(title)),
				Div(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(value)),
			),
			icon,
		),
	)
}

func (e *DashboardExtension) renderStatusCard(syncStatus *SyncStatus) g.Node {
	statusColor := "bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800"
	statusIcon := lucide.Check(Class("size-6 text-green-600"))
	statusText := "text-green-900 dark:text-green-400"

	if !syncStatus.IsHealthy {
		statusColor = "bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800"
		statusIcon = lucide.X(Class("size-6 text-red-600"))
		statusText = "text-red-900 dark:text-red-400"
	}

	return Div(
		Class(fmt.Sprintf("rounded-lg border p-6 %s", statusColor)),
		Div(
			Class("flex items-center gap-4"),
			statusIcon,
			Div(
				Class("flex-1"),
				H3(Class(fmt.Sprintf("text-lg font-semibold mb-1 %s", statusText)),
					g.Text(syncStatus.Status)),
				P(Class("text-sm opacity-80"),
					g.Text(syncStatus.Message)),
			),
		),
	)
}

func (e *DashboardExtension) renderQuickActionsCard(basePath string, currentApp *app.App) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
			g.Text("Quick Actions")),
		Div(
			Class("grid grid-cols-1 md:grid-cols-3 gap-4"),
			e.renderActionButton(
				fmt.Sprintf("%s/dashboard/app/%s/settings/scim-tokens", basePath, currentApp.ID.String()),
				"Manage Tokens",
				"Configure bearer tokens",
				lucide.Key(Class("size-5")),
			),
			e.renderActionButton(
				fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers", basePath, currentApp.ID.String()),
				"Manage Providers",
				"Configure IdP connections",
				lucide.Cloud(Class("size-5")),
			),
			e.renderActionButton(
				fmt.Sprintf("%s/dashboard/app/%s/settings/scim-monitoring", basePath, currentApp.ID.String()),
				"View Logs",
				"Monitor sync events",
				lucide.Activity(Class("size-5")),
			),
		),
	)
}

func (e *DashboardExtension) renderActionButton(href, title, description string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("block p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:border-indigo-500 dark:hover:border-indigo-400 hover:shadow-md transition-all group"),
		Div(
			Class("flex items-center gap-3 mb-2"),
			Div(Class("text-indigo-600 dark:text-indigo-400 group-hover:scale-110 transition-transform"),
				icon),
			H4(Class("font-semibold text-slate-900 dark:text-white"),
				g.Text(title)),
		),
		P(Class("text-sm text-slate-600 dark:text-gray-400"),
			g.Text(description)),
	)
}

func (e *DashboardExtension) renderEventRow(event *SCIMSyncEvent) g.Node {
	statusColor := "text-green-600"
	statusIcon := lucide.Check(Class("size-4"))

	if event.Status == "failed" {
		statusColor = "text-red-600"
		statusIcon = lucide.X(Class("size-4"))
	}

	return Div(
		Class("flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-700/50 rounded-lg"),
		Div(
			Class("flex items-center gap-3"),
			Div(Class(statusColor), statusIcon),
			Div(
				Div(Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text(event.EventType)),
				Div(Class("text-xs text-slate-600 dark:text-gray-400"),
					g.Text(event.CreatedAt.Format("Jan 02, 15:04"))),
			),
		),
		Div(Class("text-sm text-slate-600 dark:text-gray-400"),
			g.Text(event.Status)),
	)
}

// buildPageData builds common PageData for SCIM pages
func (e *DashboardExtension) buildPageData(c forge.Context, currentUser *user.User, currentApp *app.App, activePage, title string) components.PageData {
	handler := e.registry.GetHandler()
	if handler == nil {
		return components.PageData{
			Title:      title,
			User:       currentUser,
			ActivePage: activePage,
			BasePath:   e.getBasePath(),
			CurrentApp: currentApp,
		}
	}

	// Let the handler populate the page data with all required context
	return components.PageData{
		Title:      title,
		User:       currentUser,
		ActivePage: activePage,
		BasePath:   e.getBasePath(),
		CurrentApp: currentApp,
	}
}

// RenderSCIMStatusWidget renders the SCIM status widget for the dashboard
func (e *DashboardExtension) RenderSCIMStatusWidget(basePath string, currentApp *app.App) g.Node {
	if currentApp == nil {
		return Div(Class("text-gray-500"), g.Text("No app context"))
	}

	// TODO: Fetch real stats from service
	activeTokens := 2
	lastSyncTime := "5 minutes ago"
	syncHealth := "healthy"

	return Div(
		Class("text-center"),
		Div(
			Class("flex items-center justify-center gap-2 mb-2"),
			g.If(syncHealth == "healthy",
				lucide.Check(Class("size-5 text-green-500")),
			),
			g.If(syncHealth != "healthy",
				lucide.X(Class("size-5 text-red-500")),
			),
			Div(
				Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Textf("%d", activeTokens),
			),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Active Tokens"),
		),
		Div(
			Class("text-xs text-slate-400 dark:text-gray-500 mt-1"),
			g.Textf("Last sync: %s", lastSyncTime),
		),
	)
}

// RenderSyncStatsWidget renders the sync statistics widget
func (e *DashboardExtension) RenderSyncStatsWidget(basePath string, currentApp *app.App) g.Node {
	if currentApp == nil {
		return Div(Class("text-gray-500"), g.Text("No app context"))
	}

	// TODO: Fetch real stats from service
	totalSyncs := 1234
	successRate := 98.5

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Textf("%d", totalSyncs),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Total Syncs"),
		),
		Div(
			Class("text-xs text-green-600 dark:text-green-400 mt-1"),
			g.Textf("%.1f%% success rate", successRate),
		),
	)
}
