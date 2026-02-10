package notification

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/notification/builder"
	"github.com/xraph/authsome/plugins/notification/pages"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// DashboardExtension provides dashboard UI for notifications.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui",
	}
}

// ExtensionID returns the extension ID.
func (e *DashboardExtension) ExtensionID() string {
	return "notification"
}

// Routes returns routes for the notification plugin.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Main notifications overview
		{
			Method:       "GET",
			Path:         "/notifications",
			Handler:      e.ServeOverview,
			Name:         "dashboard.notifications.overview",
			Summary:      "Notifications overview",
			Description:  "View notification statistics and recent activity",
			Tags:         []string{"Dashboard", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Templates list
		{
			Method:       "GET",
			Path:         "/notifications/templates",
			Handler:      e.ServeTemplatesList,
			Name:         "dashboard.notifications.templates",
			Summary:      "Notification templates",
			Description:  "Manage notification templates",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Notification history
		{
			Method:       "GET",
			Path:         "/notifications/history",
			Handler:      e.ServeHistoryList,
			Name:         "dashboard.notifications.history",
			Summary:      "Notification history",
			Description:  "View sent email and SMS notification logs",
			Tags:         []string{"Dashboard", "Notifications", "History"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings: Auto-send rules
		{
			Method:       "GET",
			Path:         "/settings/notification",
			Handler:      e.ServeSettings,
			Name:         "dashboard.settings.notification",
			Summary:      "Notification settings page",
			Description:  "View and configure notification settings",
			Tags:         []string{"Dashboard", "Settings", "Notification"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings: Providers
		{
			Method:       "GET",
			Path:         "/settings/notification/providers",
			Handler:      e.ServeProviders,
			Name:         "dashboard.settings.notification.providers",
			Summary:      "Provider settings",
			Description:  "Configure email and SMS providers",
			Tags:         []string{"Dashboard", "Settings", "Notification"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Notifications: Analytics (dashboard layout)
		{
			Method:       "GET",
			Path:         "/notifications/analytics",
			Handler:      e.ServeAnalytics,
			Name:         "dashboard.notifications.analytics",
			Summary:      "Notification analytics",
			Description:  "View notification analytics and performance",
			Tags:         []string{"Dashboard", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Email Builder: New template
		{
			Method:       "GET",
			Path:         "/notifications/builder",
			Handler:      e.ServeEmailBuilder,
			Name:         "dashboard.notifications.builder",
			Summary:      "Email template builder",
			Description:  "Visual drag-and-drop email template builder",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Email Builder: Edit existing template
		{
			Method:       "GET",
			Path:         "/notifications/builder/:templateId",
			Handler:      e.ServeEmailBuilderWithTemplate,
			Name:         "dashboard.notifications.builder.edit",
			Summary:      "Edit template in builder",
			Description:  "Edit an existing template in the visual builder",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsPages returns settings pages for the notification plugin.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "notification",
			Label:         "Auto-Send Rules",
			Description:   "Configure automatic notification settings",
			Icon:          lucide.Bell(Class("size-4")),
			Category:      "communication",
			Order:         200,
			Path:          "notification",
			RequirePlugin: "notification",
			RequireAdmin:  true,
		},
		{
			ID:            "notification-providers",
			Label:         "Email & SMS Providers",
			Description:   "Configure notification delivery providers",
			Icon:          lucide.Send(Class("size-4")),
			Category:      "communication",
			Order:         201,
			Path:          "notification/providers",
			RequirePlugin: "notification",
			RequireAdmin:  true,
		},
	}
}

// SettingsSections returns empty settings sections for now.
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{}
}

// NavigationItems returns the main "Notifications" navigation item.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:    "notifications",
			Label: "Notifications",
			Icon: lucide.Mail(
				Class("size-4"),
			),
			Position: ui.NavPositionMain,
			Order:    55, // After organizations
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/app/" + currentApp.ID.String() + "/notifications"
				}

				return basePath + "/dashboard/"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "notifications"
			},
			RequiresPlugin: "notification",
		},
	}
}

// DashboardWidgets returns the notification stats widget.
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "notification-stats",
			Title: "Notifications",
			Icon: lucide.Mail(
				Class("size-5"),
			),
			Order: 30,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for the notification plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return e.getBridgeFunctions()
}

// ServeSettings renders the notification settings page.
func (e *DashboardExtension) ServeSettings(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.SettingsPage(currentApp, basePath), nil
}

// Helper methods

func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.RequiredField("app_id")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	return &app.App{ID: appID}, nil
}

func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// =============================================================================
// Page Handlers
// =============================================================================

// ServeOverview renders the notifications overview page.
func (e *DashboardExtension) ServeOverview(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.OverviewPage(currentApp, basePath), nil
}

// ServeTemplatesList renders the templates list page.
func (e *DashboardExtension) ServeTemplatesList(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.TemplatesListPage(currentApp, basePath), nil
}

// ServeHistoryList renders the notification history list page.
func (e *DashboardExtension) ServeHistoryList(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.HistoryListPage(currentApp, basePath), nil
}

// ServeProviders renders the providers settings page.
func (e *DashboardExtension) ServeProviders(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.ProvidersPage(currentApp, basePath), nil
}

// ServeAnalytics renders the analytics page.
func (e *DashboardExtension) ServeAnalytics(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	return pages.AnalyticsPage(currentApp, basePath), nil
}

// RenderDashboardWidget renders the notification stats widget for the main dashboard.
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	return Div(
		g.Attr("x-data", fmt.Sprintf(`{
			stats: null,
			loading: true,
			async init() {
				try {
					const result = await bridge.call('notification.getOverviewStats', {
						appId: '%s',
						days: 7
					});
					this.stats = result.stats;
				} catch (err) {
					console.error('Failed to load notification stats:', err);
				} finally {
					this.loading = false;
				}
			}
		}`, currentApp.ID.String())),
		Class("flex flex-col gap-2"),

		// Loading state
		Div(
			g.Attr("x-show", "loading"),
			Class("text-sm text-muted-foreground"),
			g.Text("Loading..."),
		),

		// Stats display
		Div(
			g.Attr("x-show", "!loading && stats"),

			// Total sent
			Div(
				Class("flex items-center justify-between text-sm"),
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Total Sent")),
				Span(Class("font-semibold text-slate-900 dark:text-white"), g.Attr("x-text", "stats.totalSent")),
			),

			// Delivery rate
			Div(
				Class("flex items-center justify-between text-sm mt-2"),
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Delivered")),
				Span(Class("font-semibold text-green-600 dark:text-green-400"),
					g.Attr("x-text", "stats.deliveryRate.toFixed(1) + '%'")),
			),

			// Open rate
			Div(
				Class("flex items-center justify-between text-sm mt-2"),
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Opened")),
				Span(Class("font-semibold text-blue-600 dark:text-blue-400"),
					g.Attr("x-text", "stats.totalOpened")),
			),

			// View details link
			Div(
				Class("mt-4 pt-2 border-t border-slate-200 dark:border-gray-700"),
				A(
					Href(fmt.Sprintf("%s/app/%s/notifications", basePath, currentApp.ID.String())),
					Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"),
					g.Text("View all notifications →"),
				),
			),
		),
	)
}

// ServeEmailBuilder renders the email builder for new templates.
func (e *DashboardExtension) ServeEmailBuilder(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	// Create new blank document
	document := builder.NewDocument()

	return pages.EmailBuilderPage(currentApp, basePath, "", document), nil
}

// ServeEmailBuilderWithTemplate renders the email builder with an existing template.
func (e *DashboardExtension) ServeEmailBuilderWithTemplate(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	templateIDStr := ctx.Param("templateId")

	templateID, err := xid.FromString(templateIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid template ID")
	}

	basePath := e.getBasePath()

	// Load template from database
	template, err := e.plugin.service.GetTemplate(ctx.Request.Context(), templateID)
	if err != nil {
		return nil, errs.NotFound("Template not found")
	}

	var document *builder.Document

	// Check if template has builder JSON in metadata
	isVisualBuilder := false

	var builderBlocks string

	if template.Metadata != nil {
		if builderType, ok := template.Metadata["builderType"].(string); ok && builderType == "visual" {
			isVisualBuilder = true
		}

		if blocks, ok := template.Metadata["builderBlocks"].(string); ok {
			builderBlocks = blocks
		}
	}

	if isVisualBuilder && builderBlocks != "" {
		// Load from builder JSON
		document, err = builder.FromJSON(builderBlocks)
		if err != nil {
			e.plugin.logger.Error("failed to parse builder JSON", forge.F("error", err))

			document = builder.NewDocument()
		}
	} else {
		// Create new document with HTML block containing the template body
		document = builder.NewDocument()
		_, _ = document.AddBlock(builder.BlockTypeHTML, map[string]any{
			"html": template.Body,
		}, document.Root)
	}

	return pages.EmailBuilderPage(currentApp, basePath, templateIDStr, document), nil
}
