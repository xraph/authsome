package scim

import (
	"fmt"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/enterprise/scim/pages"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the SCIM plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension for SCIM
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui", // Default base UI path
	}
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
					return basePath + "/scim"
				}
				return basePath + "/app/" + currentApp.ID.String() + "/scim"
			},
			ActiveChecker: func(activePage string) bool {
				return strings.HasPrefix(activePage, "scim")
			},
			RequiresPlugin: "scim",
		},
	}
}

// Routes returns routes to register under /dashboard/app/:appId/
// Note: All SCIM routes use /scim/ prefix (not /settings/scim-*) to ensure
// they get the dashboard layout instead of settings layout
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Main SCIM Dashboard (Overview)
		{
			Method:       "GET",
			Path:         "/scim",
			Handler:      e.ServeSCIMOverviewPage,
			Name:         "scim.dashboard.overview",
			Summary:      "SCIM provisioning dashboard",
			Description:  "View SCIM provisioning status and overview",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Providers Management
		{
			Method:       "GET",
			Path:         "/scim/providers",
			Handler:      e.ServeProvidersPage,
			Name:         "scim.dashboard.providers",
			Summary:      "SCIM providers list",
			Description:  "Manage SCIM identity providers",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/scim/providers/add",
			Handler:      e.ServeAddProviderPage,
			Name:         "scim.dashboard.providers.add",
			Summary:      "Add SCIM provider",
			Description:  "Add a new SCIM provider",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/scim/providers/:providerId",
			Handler:      e.ServeProviderDetailPageV2,
			Name:         "scim.dashboard.providers.detail",
			Summary:      "Provider details",
			Description:  "View SCIM provider details",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Tokens Management
		{
			Method:       "GET",
			Path:         "/scim/tokens",
			Handler:      e.ServeTokensPage,
			Name:         "scim.dashboard.tokens",
			Summary:      "SCIM tokens management",
			Description:  "Manage SCIM bearer tokens for IdP authentication",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Logs/Monitoring
		{
			Method:       "GET",
			Path:         "/scim/logs",
			Handler:      e.ServeLogsPageV2,
			Name:         "scim.dashboard.logs",
			Summary:      "SCIM event logs",
			Description:  "View SCIM provisioning event logs",
			Tags:         []string{"Dashboard", "SCIM"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Configuration
		{
			Method:       "GET",
			Path:         "/scim/config",
			Handler:      e.ServeConfigPageV2,
			Name:         "scim.dashboard.config",
			Summary:      "SCIM configuration",
			Description:  "Configure SCIM provisioning settings",
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

// SettingsPages returns settings pages
// Note: SCIM is a main navigation item (not a settings page), so we return nil here
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return nil
}

// =============================================================================
// V2 Page Handlers (using Alpine.js and ForgeUI components)
// =============================================================================

// ServeSCIMOverviewPage serves the SCIM overview page using v2 Alpine.js components
func (e *DashboardExtension) ServeSCIMOverviewPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		// Try to get from page context
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	return pages.SCIMOverviewPage(currentApp, e.getBasePath()), nil
}

// ServeProvidersPage serves the SCIM providers list page
func (e *DashboardExtension) ServeProvidersPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	return pages.ProvidersListPage(currentApp, e.getBasePath()), nil
}

// ServeAddProviderPage serves the add provider form page
func (e *DashboardExtension) ServeAddProviderPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	return pages.AddProviderPage(currentApp, e.getBasePath()), nil
}

// ServeProviderDetailPageV2 serves the provider detail page
func (e *DashboardExtension) ServeProviderDetailPageV2(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	// TODO: Create provider detail page
	// For now, redirect to providers list
	return pages.ProvidersListPage(currentApp, e.getBasePath()), nil
}

// ServeTokensPage serves the SCIM tokens management page
func (e *DashboardExtension) ServeTokensPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	return pages.TokensListPage(currentApp, e.getBasePath()), nil
}

// ServeLogsPageV2 serves the SCIM event logs page
func (e *DashboardExtension) ServeLogsPageV2(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	return pages.LogsPage(currentApp, e.getBasePath()), nil
}

// ServeConfigPageV2 serves the SCIM configuration page
func (e *DashboardExtension) ServeConfigPageV2(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		if appVal, exists := ctx.Get("currentApp"); exists && appVal != nil {
			if a, ok := appVal.(*app.App); ok {
				currentApp = a
			}
		}
		if currentApp == nil {
			return nil, errs.BadRequest("Invalid app context")
		}
	}

	// TODO: Create config page with Alpine.js
	// For now, use placeholder
	appID := currentApp.ID.String()
	appBase := e.getBasePath() + "/app/" + appID

	return Div(
		Class("space-y-6"),
		Div(
			H1(Class("text-2xl font-bold"), g.Text("SCIM Configuration")),
			P(Class("text-muted-foreground"), g.Text("Configure SCIM provisioning settings")),
		),
		Div(
			Class("p-8 text-center text-muted-foreground"),
			g.Text("Configuration page coming soon. Use the "),
			A(Href(appBase+"/scim"), Class("text-primary hover:underline"), g.Text("Overview")),
			g.Text(" page for now."),
		),
	), nil
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

// BridgeFunctions returns bridge functions for the SCIM plugin
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return e.getBridgeFunctions()
}

// Helper methods

// getUserFromContext extracts the current user from the request context
func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) *user.User {
	// Extract user from context directly
	if userVal := ctx.Request.Context().Value("user"); userVal != nil {
		if u, ok := userVal.(*user.User); ok {
			return u
		}
	}
	return nil
}

// extractAppFromURL extracts the app from the URL parameter
func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	// Extract app from context
	if appVal := ctx.Request.Context().Value("app"); appVal != nil {
		if a, ok := appVal.(*app.App); ok {
			return a, nil
		}
	}
	return nil, fmt.Errorf("app not found in context")
}

// getBasePath returns the dashboard base path
func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
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
func (e *DashboardExtension) getOrgFromContext(ctx *router.PageContext) (*xid.ID, error) {
	// Try to get from URL parameter first
	orgIDStr := ctx.Param("orgId")
	if orgIDStr != "" {
		orgID, err := xid.FromString(orgIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid org ID: %w", err)
		}
		return &orgID, nil
	}

	// Try to get from query parameter
	orgIDStr = ctx.Request.URL.Query().Get("orgId")
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

func (e *DashboardExtension) ServeSCIMDashboard(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()

	// Get current app and user
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		return nil, errs.Unauthorized()
	}

	// Get organization ID if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	// Fetch dashboard stats
	var stats *DashboardStats
	if orgID != nil {
		stats, err = e.plugin.service.GetDashboardStats(reqCtx, currentApp.ID, orgID)
	} else {
		stats, err = e.plugin.service.GetDashboardStats(reqCtx, currentApp.ID, nil)
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
		syncStatus, err = e.plugin.service.GetSyncStatus(reqCtx, currentApp.ID, orgID)
	} else {
		syncStatus, err = e.plugin.service.GetSyncStatus(reqCtx, currentApp.ID, nil)
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
		recentActivity, err = e.plugin.service.GetRecentActivity(reqCtx, currentApp.ID, orgID, 5)
	} else {
		recentActivity, err = e.plugin.service.GetRecentActivity(reqCtx, currentApp.ID, nil, 5)
	}
	if err != nil {
		recentActivity = []*SCIMSyncEvent{}
	}

	// Fetch failed operations
	var failedOps []*SCIMSyncEvent
	if orgID != nil {
		failedOps, err = e.plugin.service.GetFailedEvents(reqCtx, currentApp.ID, orgID, 5)
	} else {
		failedOps, err = e.plugin.service.GetFailedEvents(reqCtx, currentApp.ID, nil, 5)
	}
	if err != nil {
		failedOps = []*SCIMSyncEvent{}
	}

	basePath := e.getBasePath()

	// Render the dashboard with proper layout
	content := e.renderDashboardPage(basePath, currentApp, stats, syncStatus, recentActivity, failedOps)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

func (e *DashboardExtension) ServeSyncStatusPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()

	// Get current app
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		return nil, errs.Unauthorized()
	}

	// Get organization ID if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	// Fetch sync status
	var syncStatus *SyncStatus
	if orgID != nil {
		syncStatus, err = e.plugin.service.GetSyncStatus(reqCtx, currentApp.ID, orgID)
	} else {
		syncStatus, err = e.plugin.service.GetSyncStatus(reqCtx, currentApp.ID, nil)
	}
	if err != nil {
		return nil, errs.InternalServerError("Failed to fetch sync status", err)
	}

	basePath := e.getBasePath()

	content := e.renderSyncStatusPage(basePath, currentApp, syncStatus)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
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

// buildPageData is removed - PageData no longer used in ForgeUI v2

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
