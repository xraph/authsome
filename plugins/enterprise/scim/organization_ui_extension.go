package scim

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/ui"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// OrganizationUIExtension implements ui.OrganizationUIExtension to extend organization pages
type OrganizationUIExtension struct {
	plugin *Plugin
}

// NewOrganizationUIExtension creates a new organization UI extension
func NewOrganizationUIExtension(plugin *Plugin) *OrganizationUIExtension {
	return &OrganizationUIExtension{plugin: plugin}
}

// ExtensionID returns the unique identifier for this extension
func (e *OrganizationUIExtension) ExtensionID() string {
	return "scim"
}

// OrganizationWidgets returns widgets for the organization detail page
func (e *OrganizationUIExtension) OrganizationWidgets() []ui.OrganizationWidget {
	return []ui.OrganizationWidget{
		{
			ID:           "scim-sync-status",
			Title:        "SCIM Sync Status",
			Icon:         lucide.RefreshCw(Class("size-5")),
			Order:        10,
			Size:         1, // 1/3 width
			RequireAdmin: true,
			Renderer:     e.renderSyncStatusWidget,
		},
		{
			ID:           "scim-active-providers",
			Title:        "Active Providers",
			Icon:         lucide.Cloud(Class("size-5")),
			Order:        11,
			Size:         1, // 1/3 width
			RequireAdmin: true,
			Renderer:     e.renderActiveProvidersWidget,
		},
	}
}

// OrganizationTabs returns full-page tabs for organization content
func (e *OrganizationUIExtension) OrganizationTabs() []ui.OrganizationTab {
	return []ui.OrganizationTab{
		{
			ID:           "scim-provisioning",
			Label:        "SCIM Provisioning",
			Icon:         lucide.Users(Class("size-4")),
			Order:        20,
			RequireAdmin: true,
			Path:         "scim",
			Renderer:     e.renderProvisioningTab,
		},
		{
			ID:           "scim-providers",
			Label:        "SCIM Providers",
			Icon:         lucide.Cloud(Class("size-4")),
			Order:        21,
			RequireAdmin: true,
			Path:         "scim-providers",
			Renderer:     e.renderProvidersTab,
		},
		{
			ID:           "scim-monitoring",
			Label:        "SCIM Monitoring",
			Icon:         lucide.Activity(Class("size-4")),
			Order:        22,
			RequireAdmin: true,
			Path:         "scim-monitoring",
			Renderer:     e.renderMonitoringTab,
		},
	}
}

// OrganizationActions returns action buttons for the organization header
func (e *OrganizationUIExtension) OrganizationActions() []ui.OrganizationAction {
	return []ui.OrganizationAction{
		{
			ID:           "trigger-scim-sync",
			Label:        "Sync Now",
			Icon:         lucide.RefreshCw(Class("size-4")),
			Order:        10,
			Style:        "secondary",
			RequireAdmin: true,
			Action:       "triggerSCIMSync()",
		},
	}
}

// OrganizationQuickLinks returns quick access cards
func (e *OrganizationUIExtension) OrganizationQuickLinks() []ui.OrganizationQuickLink {
	return []ui.OrganizationQuickLink{
		{
			ID:           "scim-configuration",
			Title:        "SCIM Configuration",
			Description:  "Configure provisioning settings",
			Icon:         lucide.Settings(Class("size-6 text-indigo-600 dark:text-indigo-400")),
			Order:        50,
			RequireAdmin: true,
			URLBuilder: func(basePath string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/organizations/%s/tabs/scim",
					basePath, appID.String(), orgID.String())
			},
		},
		{
			ID:           "scim-tokens",
			Title:        "SCIM Tokens",
			Description:  "Manage bearer tokens",
			Icon:         lucide.Key(Class("size-6 text-green-600 dark:text-green-400")),
			Order:        51,
			RequireAdmin: true,
			URLBuilder: func(basePath string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/settings/scim-tokens?orgId=%s",
					basePath, appID.String(), orgID.String())
			},
		},
	}
}

// OrganizationSettingsSections returns settings sections for org settings
func (e *OrganizationUIExtension) OrganizationSettingsSections() []ui.OrganizationSettingsSection {
	return []ui.OrganizationSettingsSection{}
}

// Widget renderers

func (e *OrganizationUIExtension) renderSyncStatusWidget(ctx ui.OrgExtensionContext) g.Node {
	// Fetch org-scoped sync status
	stats, err := e.plugin.service.GetSyncStatusForOrg(ctx.Request.Context(), ctx.OrgID)
	if err != nil {
		return e.renderErrorWidget("Failed to load sync status")
	}

	statusIcon := lucide.Check(Class("size-5 text-green-500"))
	statusText := "All systems operational"
	
	if !stats.IsHealthy {
		statusIcon = lucide.Octagon(Class("size-5 text-yellow-500"))
		statusText = stats.Message
	}

	return Div(
		Class("text-center"),
		Div(
			Class("flex items-center justify-center gap-2 mb-2"),
			statusIcon,
			Div(
				Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Textf("%d", stats.ActiveProviders),
			),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Active Providers"),
		),
		Div(
			Class("text-xs text-slate-400 dark:text-gray-500 mt-1"),
			g.Text(statusText),
		),
	)
}

func (e *OrganizationUIExtension) renderActiveProvidersWidget(ctx ui.OrgExtensionContext) g.Node {
	// Fetch org-scoped provider stats
	stats, err := e.plugin.service.GetProviderStatsForOrg(ctx.Request.Context(), ctx.OrgID)
	if err != nil {
		return e.renderErrorWidget("Failed to load provider stats")
	}

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white mb-2"),
			g.Textf("%d", stats.TotalProviders),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Configured Providers"),
		),
		Div(
			Class("text-xs text-green-600 dark:text-green-400 mt-1"),
			g.Textf("%d active", stats.ActiveProviders),
		),
	)
}

func (e *OrganizationUIExtension) renderErrorWidget(message string) g.Node {
	return Div(
		Class("text-center text-red-500"),
		lucide.Octagon(Class("size-8 mx-auto mb-2")),
		Div(Class("text-sm"), g.Text(message)),
	)
}

// Tab renderers

func (e *OrganizationUIExtension) renderProvisioningTab(ctx ui.OrgExtensionContext) g.Node {
	// Fetch org-scoped configuration
	config, err := e.plugin.service.GetConfigForOrg(ctx.Request.Context(), ctx.OrgID)
	if err != nil {
		return e.renderError("Failed to load provisioning configuration", err)
	}

	return Div(
		Class("space-y-6"),
		// Header
		Div(
			Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
			H2(Class("text-2xl font-bold text-slate-900 dark:text-white mb-2"),
				g.Text("SCIM Provisioning Configuration")),
			P(Class("text-slate-600 dark:text-gray-400"),
				g.Text("Configure how users and groups are provisioned from your identity provider")),
		),

		// User Provisioning Section
		e.renderUserProvisioningSection(ctx, config),

		// Group Sync Section
		e.renderGroupSyncSection(ctx, config),

		// Attribute Mapping Section
		e.renderAttributeMappingSection(ctx, config),

		// Security Settings
		e.renderSecuritySettingsSection(ctx, config),
	)
}

func (e *OrganizationUIExtension) renderProvidersTab(ctx ui.OrgExtensionContext) g.Node {
	// Fetch org-scoped providers
	_, err := e.plugin.service.GetProvidersForOrg(ctx.Request.Context(), ctx.OrgID)
	if err != nil {
		return e.renderError("Failed to load providers", err)
	}

	// Convert to SCIMProviderInfo (currently empty, so create empty slice)
	// TODO: Convert providersRaw to []SCIMProviderInfo when providers are implemented
	providers := make([]SCIMProviderInfo, 0)

	return Div(
		Class("space-y-6"),
		// Header with action
		Div(
			Class("flex items-center justify-between bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
			Div(
				H2(Class("text-2xl font-bold text-slate-900 dark:text-white mb-2"),
					g.Text("SCIM Providers")),
				P(Class("text-slate-600 dark:text-gray-400"),
					g.Text("Manage identity provider connections")),
			),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/add?orgId=%s",
					ctx.BasePath, ctx.AppID.String(), ctx.OrgID.String())),
				Class("px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"),
				lucide.Plus(Class("size-4")),
				g.Text("Add Provider"),
			),
		),

		// Providers list
		e.renderProvidersList(ctx, providers),
	)
}

func (e *OrganizationUIExtension) renderMonitoringTab(ctx ui.OrgExtensionContext) g.Node {
	// Fetch org-scoped monitoring data
	_, err := e.plugin.service.GetRecentEventsForOrg(ctx.Request.Context(), ctx.OrgID, 10)
	if err != nil {
		return e.renderError("Failed to load monitoring data", err)
	}

	// Convert to SyncEvent (currently empty, so create empty slice)
	// TODO: Convert eventsRaw to []SyncEvent when events are implemented
	events := make([]SyncEvent, 0)

	stats, _ := e.plugin.service.GetSyncStatsForOrg(ctx.Request.Context(), ctx.OrgID)

	return Div(
		Class("space-y-6"),
		// Stats Grid
		Div(
			Class("grid grid-cols-1 md:grid-cols-3 gap-4"),
			e.renderStatCard("Total Syncs", fmt.Sprintf("%d", stats.TotalSyncs), lucide.RefreshCw(Class("size-6 text-indigo-600"))),
			e.renderStatCard("Success Rate", fmt.Sprintf("%.1f%%", stats.SuccessRate), lucide.Check(Class("size-6 text-green-600"))),
			e.renderStatCard("Failed", fmt.Sprintf("%d", stats.FailedSyncs), lucide.X(Class("size-6 text-red-600"))),
		),

		// Recent Events
		Div(
			Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Recent Sync Events")),
			e.renderEventsList(ctx, events),
		),
	)
}

// Helper renderers

func (e *OrganizationUIExtension) renderUserProvisioningSection(ctx ui.OrgExtensionContext, config *Config) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
			lucide.Users(Class("size-5 inline-block mr-2")),
			g.Text("User Provisioning")),
		Div(
			Class("space-y-4"),
			e.renderConfigToggle("Auto-activate Users", "Automatically activate provisioned users", config.UserProvisioning.AutoActivate),
			e.renderConfigToggle("Send Welcome Email", "Send welcome emails to new users", config.UserProvisioning.SendWelcomeEmail),
			e.renderConfigToggle("Prevent Duplicates", "Block duplicate user creation", config.UserProvisioning.PreventDuplicates),
		),
	)
}

func (e *OrganizationUIExtension) renderGroupSyncSection(ctx ui.OrgExtensionContext, config *Config) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
			lucide.Users(Class("size-5 inline-block mr-2")),
			g.Text("Group Synchronization")),
		Div(
			Class("space-y-4"),
			e.renderConfigToggle("Enable Group Sync", "Synchronize groups from IdP", config.GroupSync.Enabled),
			e.renderConfigToggle("Sync to Teams", "Map SCIM groups to teams", config.GroupSync.SyncToTeams),
			e.renderConfigToggle("Sync to Roles", "Map SCIM groups to roles", config.GroupSync.SyncToRoles),
			e.renderConfigToggle("Create Missing Groups", "Auto-create missing groups", config.GroupSync.CreateMissingGroups),
		),
	)
}

func (e *OrganizationUIExtension) renderAttributeMappingSection(ctx ui.OrgExtensionContext, config *Config) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
			lucide.Settings(Class("size-5 inline-block mr-2")),
			g.Text("Attribute Mapping")),
		P(Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
			g.Text("Custom attribute mappings are managed in the SCIM Configuration settings")),
		A(
			Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-config?orgId=%s",
				ctx.BasePath, ctx.AppID.String(), ctx.OrgID.String())),
			Class("text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 text-sm"),
			g.Text("Configure Attribute Mapping →"),
		),
	)
}

func (e *OrganizationUIExtension) renderSecuritySettingsSection(ctx ui.OrgExtensionContext, config *Config) g.Node {
	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6"),
		H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
			lucide.Shield(Class("size-5 inline-block mr-2")),
			g.Text("Security Settings")),
		Div(
			Class("space-y-4"),
			e.renderConfigToggle("Require HTTPS", "Enforce HTTPS for all SCIM requests", config.Security.RequireHTTPS),
			e.renderConfigToggle("Audit All Operations", "Log all SCIM operations", config.Security.AuditAllOperations),
			e.renderConfigToggle("Mask Sensitive Data", "Hide sensitive data in logs", config.Security.MaskSensitiveData),
		),
	)
}

func (e *OrganizationUIExtension) renderProvidersList(ctx ui.OrgExtensionContext, providers []SCIMProviderInfo) g.Node {
	if len(providers) == 0 {
		return Div(
			Class("bg-white dark:bg-slate-800 rounded-lg shadow p-12 text-center"),
			lucide.Cloud(Class("size-12 mx-auto mb-4 text-slate-400")),
			P(Class("text-slate-600 dark:text-gray-400 mb-4"),
				g.Text("No providers configured yet")),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/add?orgId=%s",
					ctx.BasePath, ctx.AppID.String(), ctx.OrgID.String())),
				Class("text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"),
				g.Text("Add your first provider →"),
			),
		)
	}

	return Div(
		Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
		g.Group(g.Map(providers, func(p SCIMProviderInfo) g.Node {
			return e.renderProviderCard(ctx, p)
		})),
	)
}

func (e *OrganizationUIExtension) renderProviderCard(ctx ui.OrgExtensionContext, provider SCIMProviderInfo) g.Node {
	statusColor := "text-green-600"
	statusText := "Active"
	statusIcon := lucide.Check(Class("size-4"))

	if !provider.Enabled {
		statusColor = "text-gray-500"
		statusText = "Disabled"
		statusIcon = lucide.X(Class("size-4"))
	}

	return Div(
		Class("bg-white dark:bg-slate-800 rounded-lg shadow p-6 hover:shadow-lg transition-shadow"),
		Div(
			Class("flex items-start justify-between mb-4"),
			Div(
				H4(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text(provider.Name)),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text(provider.Type)),
			),
			Div(
				Class(fmt.Sprintf("flex items-center gap-1 text-sm %s", statusColor)),
				statusIcon,
				g.Text(statusText),
			),
		),
		Div(
			Class("space-y-2 text-sm text-slate-600 dark:text-gray-400"),
			Div(g.Textf("Endpoint: %s", provider.Endpoint)),
			Div(g.Textf("Last Sync: %s", provider.LastSyncTime)),
		),
		Div(
			Class("mt-4 pt-4 border-t border-slate-200 dark:border-slate-700 flex gap-2"),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/%s?orgId=%s",
					ctx.BasePath, ctx.AppID.String(), provider.ID, ctx.OrgID.String())),
				Class("px-3 py-1 text-sm text-indigo-600 hover:bg-indigo-50 dark:text-indigo-400 rounded-md transition-colors"),
				g.Text("Configure"),
			),
			Button(
				Type("button"),
				Class("px-3 py-1 text-sm text-green-600 hover:bg-green-50 dark:text-green-400 rounded-md transition-colors"),
				g.Text("Test Connection"),
			),
		),
	)
}

func (e *OrganizationUIExtension) renderEventsList(ctx ui.OrgExtensionContext, events []SyncEvent) g.Node {
	if len(events) == 0 {
		return P(Class("text-slate-600 dark:text-gray-400 text-center py-8"),
			g.Text("No recent sync events"))
	}

	return Div(
		Class("space-y-2"),
		g.Group(g.Map(events, func(event SyncEvent) g.Node {
			return e.renderEventRow(event)
		})),
	)
}

func (e *OrganizationUIExtension) renderEventRow(event SyncEvent) g.Node {
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
					g.Text(event.Operation)),
				Div(Class("text-xs text-slate-600 dark:text-gray-400"),
					g.Text(event.Timestamp)),
			),
		),
		Div(Class("text-sm text-slate-600 dark:text-gray-400"),
			g.Text(event.Details)),
	)
}

func (e *OrganizationUIExtension) renderStatCard(title, value string, icon g.Node) g.Node {
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

func (e *OrganizationUIExtension) renderConfigToggle(label, description string, enabled bool) g.Node {
	return Div(
		Class("flex items-start justify-between py-3 border-b border-slate-200 dark:border-slate-700 last:border-0"),
		Div(
			Div(Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(label)),
			Div(Class("text-xs text-slate-600 dark:text-gray-400 mt-1"),
				g.Text(description)),
		),
		Div(
			Class("flex items-center"),
			g.If(enabled,
				lucide.Check(Class("size-5 text-green-600")),
			),
			g.If(!enabled,
				lucide.X(Class("size-5 text-gray-400")),
			),
		),
	)
}

func (e *OrganizationUIExtension) renderError(message string, err error) g.Node {
	return Div(
		Class("bg-red-50 dark:bg-red-900/20 rounded-lg p-6 text-center"),
		lucide.Octagon(Class("size-12 mx-auto mb-4 text-red-600")),
		H3(Class("text-lg font-semibold text-red-900 dark:text-red-400 mb-2"),
			g.Text(message)),
		P(Class("text-sm text-red-700 dark:text-red-500"),
			g.Text(err.Error())),
	)
}

// Helper types for rendering

type SCIMProviderInfo struct {
	ID           string
	Name         string
	Type         string
	Endpoint     string
	Enabled      bool
	LastSyncTime string
}

type SyncEvent struct {
	Operation string
	Status    string
	Timestamp string
	Details   string
}
