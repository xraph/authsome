package social

import (
	"fmt"
	"net/http"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the social plugin.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
	configRepo repository.SocialProviderConfigRepository
}

// NewDashboardExtension creates a new dashboard extension.
func NewDashboardExtension(plugin *Plugin, configRepo repository.SocialProviderConfigRepository) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		configRepo: configRepo,
		baseUIPath: "/api/identity/ui",
	}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration).
func (e *DashboardExtension) SetRegistry(registry any) {
	// No longer needed - layout handled by ForgeUI
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "social"
}

// NavigationItems returns the navigation items for the dashboard.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "social",
			Label:    "Social Providers",
			Icon:     lucide.Share2(Class("size-5")),
			Position: ui.NavPositionMain,
			Order:    50,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/app/" + currentApp.ID.String() + "/social"
				}

				return basePath + "/social"
			},
			ActiveChecker: func(currentPage string) bool {
				return currentPage == "social" || currentPage == "social-add" || currentPage == "social-edit"
			},
		},
	}
}

// Routes returns the dashboard routes.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Social Providers List (Main Page)
		{
			Method:       "GET",
			Path:         "/social",
			Handler:      e.ServeProvidersListPage,
			Name:         "social.providers.list",
			Summary:      "Social Providers",
			Description:  "Configure social authentication providers",
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Add Provider Form
		{
			Method:       "GET",
			Path:         "/social/add",
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
			Path:         "/social/create",
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
			Path:         "/social/:id/edit",
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
			Path:         "/social/:id/update",
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
			Path:         "/social/:id/toggle",
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
			Path:         "/social/:id/delete",
			Handler:      e.HandleDeleteProvider,
			Name:         "social.providers.delete",
			Summary:      "Delete Social Provider",
			Description:  "Delete a social authentication provider configuration",
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead).
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages for the plugin.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	// Social providers now have their own main navigation item, not in settings
	return nil
}

// DashboardWidgets returns dashboard widgets.
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

// BridgeFunctions returns bridge functions for the social plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	// No bridge functions for this plugin yet
	return nil
}

// Helper methods

// getUserFromContext extracts the current user from the request context.
func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) *user.User {
	reqCtx := ctx.Request.Context()
	if u, ok := reqCtx.Value("user").(*user.User); ok {
		return u
	}

	return nil
}

// extractAppFromURL extracts the app from the URL parameter.
func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	// First try to extract app from request context (set by middleware)
	reqCtx := ctx.Request.Context()

	appVal := reqCtx.Value(contexts.AppContextKey)
	if appVal != nil {
		if currentApp, ok := appVal.(*app.App); ok {
			return currentApp, nil
		}
	}

	// Fallback: try to get from PageContext (set by ForgeUI router)
	if pageAppVal, exists := ctx.Get("currentApp"); exists && pageAppVal != nil {
		if currentApp, ok := pageAppVal.(*app.App); ok {
			return currentApp, nil
		}
	}

	// Final fallback: parse app ID from URL and create minimal app
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.New(errs.CodeInvalidInput, "app ID is required", http.StatusBadRequest)
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	// Return minimal app with ID - the dashboard handler will enrich it
	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path.
func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// getCurrentEnvironmentID gets the current environment ID from the dashboard handler.
func (e *DashboardExtension) getCurrentEnvironmentID(ctx *router.PageContext, appID xid.ID) (xid.ID, error) {
	// First try to get environment from request context (set by middleware)
	reqCtx := ctx.Request.Context()

	envVal := reqCtx.Value(contexts.EnvironmentContextKey)
	if envVal != nil {
		if currentEnv, ok := envVal.(*environment.Environment); ok {
			return currentEnv.ID, nil
		}
		// Also check if it's directly an xid.ID
		if envID, ok := envVal.(xid.ID); ok {
			return envID, nil
		}
	}

	// Fallback: try to get from PageContext (set by ForgeUI router)
	if pageEnvVal, exists := ctx.Get("currentEnvironment"); exists && pageEnvVal != nil {
		if currentEnv, ok := pageEnvVal.(*environment.Environment); ok {
			return currentEnv.ID, nil
		}
	}

	// If still not found, return NilID - many operations can work without environment
	// The service layer can get the default environment if needed
	return xid.NilID(), nil
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
