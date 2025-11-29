package social

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the social plugin
type DashboardExtension struct {
	plugin     *Plugin
	registry   *dashboard.ExtensionRegistry
	configRepo repository.SocialProviderConfigRepository
}

// NewDashboardExtension creates a new dashboard extension
func NewDashboardExtension(plugin *Plugin, configRepo repository.SocialProviderConfigRepository) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		configRepo: configRepo,
	}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "social"
}

// NavigationItems returns the navigation items for the dashboard
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	// Social providers are configured in settings, not main nav
	return nil
}

// Routes returns the dashboard routes
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Social Providers List (Settings Page)
		{
			Method:      "GET",
			Path:        "/settings/social",
			Handler:     e.ServeProvidersListPage,
			Name:        "social.providers.list",
			Summary:     "Social Providers",
			Description: "Configure social authentication providers",
			RequireAuth: true,
		},
		// Add Provider Form
		{
			Method:       "GET",
			Path:         "/settings/social/add",
			Handler:      e.ServeProviderAddPage,
			Name:         "social.providers.add",
			Summary:      "Add Social Provider",
			Description:  "Add a new social authentication provider",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Provider
		{
			Method:       "POST",
			Path:         "/settings/social/create",
			Handler:      e.HandleCreateProvider,
			Name:         "social.providers.create",
			Summary:      "Create Social Provider",
			Description:  "Create a new social authentication provider configuration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit Provider Form
		{
			Method:       "GET",
			Path:         "/settings/social/:id/edit",
			Handler:      e.ServeProviderEditPage,
			Name:         "social.providers.edit",
			Summary:      "Edit Social Provider",
			Description:  "Edit social authentication provider configuration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Provider
		{
			Method:       "POST",
			Path:         "/settings/social/:id/update",
			Handler:      e.HandleUpdateProvider,
			Name:         "social.providers.update",
			Summary:      "Update Social Provider",
			Description:  "Update social authentication provider configuration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Toggle Provider
		{
			Method:       "POST",
			Path:         "/settings/social/:id/toggle",
			Handler:      e.HandleToggleProvider,
			Name:         "social.providers.toggle",
			Summary:      "Toggle Social Provider",
			Description:  "Enable or disable a social authentication provider",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Provider
		{
			Method:       "POST",
			Path:         "/settings/social/:id/delete",
			Handler:      e.HandleDeleteProvider,
			Name:         "social.providers.delete",
			Summary:      "Delete Social Provider",
			Description:  "Delete a social authentication provider configuration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages for the plugin
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "social-providers",
			Label:         "Social Providers",
			Description:   "Configure OAuth social authentication providers",
			Icon:          lucide.Share2(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         10,
			Path:          "social",
			RequirePlugin: "social",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns dashboard widgets
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "social-providers-count",
			Title: "Social Providers",
			Icon:  lucide.Share2(Class("size-5")),
			Order: 50,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderProvidersWidget(currentApp)
			},
		},
	}
}

// Helper methods

// getUserFromContext extracts the current user from the request context
func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	ctx := c.Request().Context()
	if u, ok := ctx.Value("user").(*user.User); ok {
		return u
	}
	return nil
}

// extractAppFromURL extracts the app from the URL parameter
func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	appIDStr := c.Param("appId")
	if appIDStr == "" {
		return nil, fmt.Errorf("app ID is required")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	// Return minimal app with ID - the dashboard handler will enrich it
	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path
func (e *DashboardExtension) getBasePath() string {
	if e.registry != nil && e.registry.GetHandler() != nil {
		return e.registry.GetHandler().GetBasePath()
	}
	return ""
}

// getCurrentEnvironmentID gets the current environment ID from the dashboard handler
func (e *DashboardExtension) getCurrentEnvironmentID(c forge.Context, appID xid.ID) (xid.ID, error) {
	if e.registry != nil && e.registry.GetHandler() != nil {
		env, err := e.registry.GetHandler().GetCurrentEnvironment(c, appID)
		if err != nil {
			return xid.NilID(), err
		}
		return env.ID, nil
	}
	return xid.NilID(), fmt.Errorf("no environment available")
}

// Widget renderer

func (e *DashboardExtension) renderProvidersWidget(currentApp *app.App) g.Node {
	if currentApp == nil || e.configRepo == nil {
		return e.renderEmptyWidget("No app context")
	}

	// For the widget, we just show a count without needing environment context
	// This is a simple overview widget
	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			lucide.Share2(Class("size-8 mx-auto text-violet-500")),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400 mt-2"),
			g.Text("Configure in Settings"),
		),
	)
}

func (e *DashboardExtension) renderEmptyWidget(message string) g.Node {
	return Div(
		Class("text-center"),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text(message),
		),
	)
}

