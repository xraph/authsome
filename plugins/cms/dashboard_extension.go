package cms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/pages"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the CMS plugin
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
}

// NewDashboardExtension creates a new dashboard extension
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "cms"
}

// NavigationItems returns navigation items for the dashboard
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "cms",
			Label:    "Content",
			Icon:     lucide.Database(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    35, // After Users (20), before Secrets (60)
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/dashboard/cms"
				}
				return basePath + "/dashboard/app/" + currentApp.ID.String() + "/cms"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "cms" ||
					activePage == "cms-types" ||
					activePage == "cms-type-detail" ||
					activePage == "cms-type-create" ||
					activePage == "cms-entries" ||
					activePage == "cms-entry-detail" ||
					activePage == "cms-entry-create" ||
					activePage == "cms-entry-edit" ||
					activePage == "cms-components"
			},
			RequiresPlugin: "cms",
		},
	}
}

// Routes returns dashboard routes
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// CMS Settings Page (in Settings section)
		{
			Method:       "GET",
			Path:         "/settings/cms",
			Handler:      e.ServeCMSSettings,
			Name:         "cms.dashboard.settings",
			Summary:      "CMS Settings",
			Description:  "Configure CMS settings and content types",
			Tags:         []string{"Dashboard", "Settings", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// CMS Overview
		{
			Method:       "GET",
			Path:         "/cms",
			Handler:      e.ServeCMSOverview,
			Name:         "cms.dashboard.overview",
			Summary:      "CMS Overview",
			Description:  "View CMS overview with content type list",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Types list
		{
			Method:       "GET",
			Path:         "/cms/types",
			Handler:      e.ServeContentTypesList,
			Name:         "cms.dashboard.types.list",
			Summary:      "Content Types",
			Description:  "List all content types",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Content Type page
		{
			Method:       "GET",
			Path:         "/cms/types/create",
			Handler:      e.ServeCreateContentType,
			Name:         "cms.dashboard.types.create",
			Summary:      "Create Content Type",
			Description:  "Create a new content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Content Type action
		{
			Method:       "POST",
			Path:         "/cms/types/create",
			Handler:      e.HandleCreateContentType,
			Name:         "cms.dashboard.types.create.submit",
			Summary:      "Submit Create Content Type",
			Description:  "Process content type creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Type detail
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName",
			Handler:      e.ServeContentTypeDetail,
			Name:         "cms.dashboard.types.detail",
			Summary:      "Content Type Detail",
			Description:  "View content type details and fields",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Content Type action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/delete",
			Handler:      e.HandleDeleteContentType,
			Name:         "cms.dashboard.types.delete",
			Summary:      "Delete Content Type",
			Description:  "Delete a content type and all its fields",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Add Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields",
			Handler:      e.HandleAddField,
			Name:         "cms.dashboard.fields.create",
			Summary:      "Add Field",
			Description:  "Add a new field to a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields/:fieldName/update",
			Handler:      e.HandleUpdateField,
			Name:         "cms.dashboard.fields.update",
			Summary:      "Update Field",
			Description:  "Update a field in a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields/:fieldName/delete",
			Handler:      e.HandleDeleteField,
			Name:         "cms.dashboard.fields.delete",
			Summary:      "Delete Field",
			Description:  "Delete a field from a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Display Settings
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/settings/display",
			Handler:      e.HandleUpdateDisplaySettings,
			Name:         "cms.dashboard.settings.display",
			Summary:      "Update Display Settings",
			Description:  "Update content type display settings (title, description, preview fields)",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Feature Settings
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/settings/features",
			Handler:      e.HandleUpdateFeatureSettings,
			Name:         "cms.dashboard.settings.features",
			Summary:      "Update Feature Settings",
			Description:  "Update content type feature settings (revisions, drafts, etc.)",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Entries list
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries",
			Handler:      e.ServeEntriesList,
			Name:         "cms.dashboard.entries.list",
			Summary:      "Content Entries",
			Description:  "List entries for a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Entry page
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/create",
			Handler:      e.ServeCreateEntry,
			Name:         "cms.dashboard.entries.create",
			Summary:      "Create Entry",
			Description:  "Create a new content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/create",
			Handler:      e.HandleCreateEntry,
			Name:         "cms.dashboard.entries.create.submit",
			Summary:      "Submit Create Entry",
			Description:  "Process entry creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Entry detail
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/:entryId",
			Handler:      e.ServeEntryDetail,
			Name:         "cms.dashboard.entries.detail",
			Summary:      "Entry Detail",
			Description:  "View entry details",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit Entry page
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/:entryId/edit",
			Handler:      e.ServeEditEntry,
			Name:         "cms.dashboard.entries.edit",
			Summary:      "Edit Entry",
			Description:  "Edit a content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/:entryId/update",
			Handler:      e.HandleUpdateEntry,
			Name:         "cms.dashboard.entries.update",
			Summary:      "Update Entry",
			Description:  "Process entry update form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/:entryId/delete",
			Handler:      e.HandleDeleteEntry,
			Name:         "cms.dashboard.entries.delete",
			Summary:      "Delete Entry",
			Description:  "Delete a content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Component Schemas list
		{
			Method:       "GET",
			Path:         "/cms/components",
			Handler:      e.ServeComponentSchemasList,
			Name:         "cms.dashboard.components.list",
			Summary:      "Component Schemas",
			Description:  "List all component schemas",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Component Schema page
		{
			Method:       "GET",
			Path:         "/cms/components/create",
			Handler:      e.ServeCreateComponentSchema,
			Name:         "cms.dashboard.components.create",
			Summary:      "Create Component Schema",
			Description:  "Create a new component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/create",
			Handler:      e.HandleCreateComponentSchema,
			Name:         "cms.dashboard.components.create.submit",
			Summary:      "Submit Create Component Schema",
			Description:  "Process component schema creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Component Schema detail/edit
		{
			Method:       "GET",
			Path:         "/cms/components/:componentSlug",
			Handler:      e.ServeComponentSchemaDetail,
			Name:         "cms.dashboard.components.detail",
			Summary:      "Component Schema Detail",
			Description:  "View/edit component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/:componentSlug",
			Handler:      e.HandleUpdateComponentSchema,
			Name:         "cms.dashboard.components.update",
			Summary:      "Update Component Schema",
			Description:  "Process component schema update form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/:componentSlug/delete",
			Handler:      e.HandleDeleteComponentSchema,
			Name:         "cms.dashboard.components.delete",
			Summary:      "Delete Component Schema",
			Description:  "Delete a component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "cms-settings",
			Label:         "Content Management",
			Description:   "Configure CMS settings and content types",
			Icon:          lucide.Database(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         30,
			Path:          "cms",
			RequirePlugin: "cms",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns dashboard widgets
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "cms-stats",
			Title: "Content",
			Icon:  lucide.Database(Class("size-5")),
			Order: 35,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderCMSWidget(currentApp)
			},
		},
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

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

	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path
func (e *DashboardExtension) getBasePath() string {
	if e.registry != nil && e.registry.GetHandler() != nil {
		return e.registry.GetHandler().GetBasePath()
	}
	return ""
}

// injectContext injects app and environment IDs into context
func (e *DashboardExtension) injectContext(c forge.Context) context.Context {
	ctx := c.Request().Context()

	// Get app ID from URL
	var appID xid.ID
	if appIDStr := c.Param("appId"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			appID = id
			ctx = contexts.SetAppID(ctx, appID)
		}
	}

	// Get environment ID from header or context
	if envIDStr := c.Request().Header.Get("X-Environment-ID"); envIDStr != "" {
		if envID, err := xid.FromString(envIDStr); err == nil {
			ctx = contexts.SetEnvironmentID(ctx, envID)
		}
	}

	// Try to get from existing context
	if envID, ok := contexts.GetEnvironmentID(c.Request().Context()); ok {
		ctx = contexts.SetEnvironmentID(ctx, envID)
	}

	// If no environment ID yet, try to get default environment for the app
	if _, ok := contexts.GetEnvironmentID(ctx); !ok && !appID.IsNil() {
		if envSvc := e.plugin.authInst.GetServiceRegistry().EnvironmentService(); envSvc != nil {
			if defaultEnv, err := envSvc.GetDefaultEnvironment(ctx, appID); err == nil && defaultEnv != nil {
				ctx = contexts.SetEnvironmentID(ctx, defaultEnv.ID)
			}
		}
	}

	return ctx
}

// =============================================================================
// Widget Renderer
// =============================================================================

func (e *DashboardExtension) renderCMSWidget(currentApp *app.App) g.Node {
	ctx := context.Background()
	if currentApp != nil {
		ctx = contexts.SetAppID(ctx, currentApp.ID)
	}

	// Get stats
	stats, err := e.plugin.contentTypeSvc.GetStats(ctx)
	if err != nil {
		stats = &core.CMSStatsDTO{
			TotalContentTypes: 0,
			TotalEntries:      0,
		}
	}

	return Div(
		Class("text-center"),
		Div(
			Class("grid grid-cols-2 gap-4"),
			Div(
				Div(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(fmt.Sprintf("%d", stats.TotalContentTypes)),
				),
				Div(
					Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text("Content Types"),
				),
			),
			Div(
				Div(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(fmt.Sprintf("%d", stats.TotalEntries)),
				),
				Div(
					Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text("Total Entries"),
				),
			),
		),
	)
}

// =============================================================================
// CMS Settings Handler
// =============================================================================

func (e *DashboardExtension) ServeCMSSettings(c forge.Context) error {
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

	ctx := e.injectContext(c)

	// Get content types for stats
	result, err := e.plugin.contentTypeSvc.List(ctx, &core.ListContentTypesQuery{
		PageSize: 100,
	})
	if err != nil {
		result = &core.ListContentTypesResponse{ContentTypes: []*core.ContentTypeSummaryDTO{}}
	}

	// Get stats
	stats, _ := e.plugin.contentTypeSvc.GetStats(ctx)

	basePath := handler.GetBasePath()
	content := e.renderCMSSettingsContent(currentApp, basePath, result.ContentTypes, stats)

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "cms-settings", content)
}

// renderCMSSettingsContent renders the CMS settings page content
func (e *DashboardExtension) renderCMSSettingsContent(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO, stats *core.CMSStatsDTO) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Build stats display
	var totalTypes, totalEntries int
	if stats != nil {
		totalTypes = stats.TotalContentTypes
		totalEntries = stats.TotalEntries
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Content Management")),
			P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Configure CMS settings and manage your content types")),
		),

		// Stats overview
		Div(
			Class("grid gap-4 md:grid-cols-2"),

			// Content Types card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
						lucide.Database(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
					),
					Div(
						H3(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(fmt.Sprintf("%d", totalTypes))),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Content Types")),
					),
				),
			),

			// Entries card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20"),
						lucide.FileText(Class("h-5 w-5 text-blue-600 dark:text-blue-400")),
					),
					Div(
						H3(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(fmt.Sprintf("%d", totalEntries))),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Total Entries")),
					),
				),
			),
		),

		// Quick actions
		Div(
			Class("grid gap-4 md:grid-cols-3"),

			// Manage Content Types
			A(
				Href(appBase+"/cms/types"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
						lucide.Layers(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Content Types")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Define and manage your content schemas")),
			),

			// Create Content Type
			A(
				Href(appBase+"/cms/types/create"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20"),
						lucide.Plus(Class("h-5 w-5 text-green-600 dark:text-green-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("New Content Type")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Create a new content type schema")),
			),

			// CMS Overview
			A(
				Href(appBase+"/cms"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-900/20"),
						lucide.LayoutDashboard(Class("h-5 w-5 text-slate-600 dark:text-slate-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("CMS Overview")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("View the full CMS dashboard")),
			),
		),

		// Recent content types
		g.If(len(contentTypes) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-base font-semibold text-slate-900 dark:text-white"),
						g.Text("Recent Content Types")),
				),
				Div(
					Class("divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(g.Map(contentTypes, func(ct *core.ContentTypeSummaryDTO) g.Node {
						return A(
							Href(appBase+"/cms/types/"+ct.Name),
							Class("flex items-center justify-between px-6 py-4 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("flex h-8 w-8 items-center justify-center rounded bg-slate-100 dark:bg-gray-800"),
									lucide.FileCode(Class("h-4 w-4 text-slate-600 dark:text-gray-400")),
								),
								Div(
									H4(Class("text-sm font-medium text-slate-900 dark:text-white"),
										g.Text(ct.Name)),
									P(Class("text-xs text-slate-500 dark:text-gray-400"),
										g.Textf("%d entries", ct.EntryCount)),
								),
							),
							lucide.ChevronRight(Class("h-4 w-4 text-slate-400")),
						)
					})),
				),
			),
		),
	)
}

// =============================================================================
// CMS Overview Handler
// =============================================================================

func (e *DashboardExtension) ServeCMSOverview(c forge.Context) error {
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

	ctx := e.injectContext(c)

	// Get content types
	result, err := e.plugin.contentTypeSvc.List(ctx, &core.ListContentTypesQuery{
		PageSize: 100,
	})
	if err != nil {
		result = &core.ListContentTypesResponse{ContentTypes: []*core.ContentTypeSummaryDTO{}}
	}

	// Get stats
	stats, _ := e.plugin.contentTypeSvc.GetStats(ctx)

	basePath := handler.GetBasePath()
	content := pages.CMSOverviewPage(currentApp, basePath, result.ContentTypes, stats)

	pageData := components.PageData{
		Title:      "Content Management",
		User:       currentUser,
		ActivePage: "cms",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// =============================================================================
// Content Types Handlers
// =============================================================================

func (e *DashboardExtension) ServeContentTypesList(c forge.Context) error {
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

	ctx := e.injectContext(c)

	searchQuery := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	// Get content types
	result, err := e.plugin.contentTypeSvc.List(ctx, &core.ListContentTypesQuery{
		Search:   searchQuery,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListContentTypesResponse{ContentTypes: []*core.ContentTypeSummaryDTO{}}
	}

	basePath := handler.GetBasePath()
	content := pages.ContentTypesListPage(currentApp, basePath, result.ContentTypes, page, pageSize, result.TotalItems, searchQuery)

	pageData := components.PageData{
		Title:      "Content Types",
		User:       currentUser,
		ActivePage: "cms-types",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) ServeCreateContentType(c forge.Context) error {
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

	basePath := handler.GetBasePath()
	errMsg := c.Query("error")

	content := pages.CreateContentTypePage(currentApp, basePath, errMsg)

	pageData := components.PageData{
		Title:      "Create Content Type",
		User:       currentUser,
		ActivePage: "cms-type-create",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) HandleCreateContentType(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Parse form
	req := &core.CreateContentTypeRequest{
		Title:       c.FormValue("name"),
		Name:        c.FormValue("slug"),
		Description: c.FormValue("description"),
		Icon:        c.FormValue("icon"),
	}

	// Create content type
	result, err := e.plugin.contentTypeSvc.Create(ctx, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/create?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+result.Name)
}

func (e *DashboardExtension) ServeContentTypeDetail(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Get stats
	contentTypeID, _ := xid.FromString(contentType.ID)
	stats, _ := e.plugin.entrySvc.GetStats(ctx, contentTypeID)

	// Get environment ID from context (set by injectContext)
	var envIDStr string
	if envID, ok := contexts.GetEnvironmentID(ctx); ok {
		envIDStr = envID.String()
	}

	// Get all content types for relation field dropdown
	allContentTypes := []*core.ContentTypeSummaryDTO{}
	ctResult, _ := e.plugin.contentTypeSvc.List(ctx, &core.ListContentTypesQuery{PageSize: 100})
	if ctResult != nil {
		allContentTypes = ctResult.ContentTypes
	}

	// Get all component schemas for nested field dropdowns
	allComponentSchemas := []*core.ComponentSchemaSummaryDTO{}
	csResult, _ := e.plugin.componentSchemaSvc.List(ctx, &core.ListComponentSchemasQuery{PageSize: 100})
	if csResult != nil {
		allComponentSchemas = csResult.Components
	}

	content := pages.ContentTypeDetailPage(currentApp, basePath, contentType, stats, envIDStr, allContentTypes, allComponentSchemas)

	pageData := components.PageData{
		Title:      contentType.Name,
		User:       currentUser,
		ActivePage: "cms-type-detail",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// HandleAddField handles adding a new field to a content type
func (e *DashboardExtension) HandleAddField(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values
	req := &core.CreateFieldRequest{
		Title:       c.FormValue("name"),
		Name:        c.FormValue("slug"),
		Type:        c.FormValue("type"),
		Description: c.FormValue("description"),
		Required:    c.FormValue("required") == "true",
		Unique:      c.FormValue("unique") == "true",
		Indexed:     c.FormValue("indexed") == "true",
		Localized:   c.FormValue("localized") == "true",
		Options:     e.parseFieldOptions(c),
	}

	// Create the field
	_, err = e.plugin.fieldSvc.Create(ctx, contentTypeID, req)
	if err != nil {
		// Redirect back with error
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+err.Error())
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName)
}

// HandleUpdateField handles updating a field in a content type
func (e *DashboardExtension) HandleUpdateField(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	fieldName := c.Param("fieldName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values for update
	req := &core.UpdateFieldRequest{
		Title:       c.FormValue("name"),
		Description: c.FormValue("description"),
		Options:     e.parseFieldOptions(c),
	}

	// Parse boolean fields
	if c.FormValue("required") != "" {
		v := c.FormValue("required") == "true"
		req.Required = &v
	}
	if c.FormValue("unique") != "" {
		v := c.FormValue("unique") == "true"
		req.Unique = &v
	}
	if c.FormValue("indexed") != "" {
		v := c.FormValue("indexed") == "true"
		req.Indexed = &v
	}
	if c.FormValue("localized") != "" {
		v := c.FormValue("localized") == "true"
		req.Localized = &v
	}

	// Update the field
	_, err = e.plugin.fieldSvc.UpdateByName(ctx, contentTypeID, fieldName, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+err.Error())
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName)
}

// HandleDeleteField handles deleting a field from a content type
func (e *DashboardExtension) HandleDeleteField(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	fieldName := c.Param("fieldName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Delete the field
	err = e.plugin.fieldSvc.DeleteByName(ctx, contentTypeID, fieldName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+err.Error())
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName)
}

// HandleUpdateDisplaySettings handles updating content type display settings
func (e *DashboardExtension) HandleUpdateDisplaySettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values
	titleField := c.FormValue("titleField")
	descriptionField := c.FormValue("descriptionField")
	previewField := c.FormValue("previewField")

	// Get current settings
	currentSettings := contentType.Settings

	// Update display fields
	req := &core.UpdateContentTypeRequest{
		Settings: &core.ContentTypeSettingsDTO{
			TitleField:         titleField,
			DescriptionField:   descriptionField,
			PreviewField:       previewField,
			EnableRevisions:    currentSettings.EnableRevisions,
			EnableDrafts:       currentSettings.EnableDrafts,
			EnableSoftDelete:   currentSettings.EnableSoftDelete,
			EnableSearch:       currentSettings.EnableSearch,
			EnableScheduling:   currentSettings.EnableScheduling,
			DefaultPermissions: currentSettings.DefaultPermissions,
			MaxEntries:         currentSettings.MaxEntries,
		},
	}

	// Update content type
	_, err = e.plugin.contentTypeSvc.Update(ctx, contentTypeID, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()))
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?success=Display+settings+saved")
}

// HandleUpdateFeatureSettings handles updating content type feature settings
func (e *DashboardExtension) HandleUpdateFeatureSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values (checkboxes are only sent if checked)
	enableRevisions := c.FormValue("enableRevisions") == "on"
	enableDrafts := c.FormValue("enableDrafts") == "on"
	enableSoftDelete := c.FormValue("enableSoftDelete") == "on"
	enableSearch := c.FormValue("enableSearch") == "on"
	enableScheduling := c.FormValue("enableScheduling") == "on"

	// Get current settings
	currentSettings := contentType.Settings

	// Update feature settings
	req := &core.UpdateContentTypeRequest{
		Settings: &core.ContentTypeSettingsDTO{
			TitleField:         currentSettings.TitleField,
			DescriptionField:   currentSettings.DescriptionField,
			PreviewField:       currentSettings.PreviewField,
			EnableRevisions:    enableRevisions,
			EnableDrafts:       enableDrafts,
			EnableSoftDelete:   enableSoftDelete,
			EnableSearch:       enableSearch,
			EnableScheduling:   enableScheduling,
			DefaultPermissions: currentSettings.DefaultPermissions,
			MaxEntries:         currentSettings.MaxEntries,
		},
	}

	// Update content type
	_, err = e.plugin.contentTypeSvc.Update(ctx, contentTypeID, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()))
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?success=Feature+settings+saved")
}

// HandleDeleteContentType handles deleting a content type
func (e *DashboardExtension) HandleDeleteContentType(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type to get its ID
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Check if there are entries - if so, don't allow delete
	entries, _ := e.plugin.entrySvc.List(ctx, contentTypeID, &core.ListEntriesQuery{PageSize: 1})
	if entries != nil && entries.TotalItems > 0 {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error=Cannot+delete+content+type+with+existing+entries.+Delete+all+entries+first.")
	}

	// Delete the content type (this also deletes all fields)
	err = e.plugin.contentTypeSvc.Delete(ctx, contentTypeID)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"?error="+err.Error())
	}

	// Redirect back to content types list
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?success=Content+type+deleted+successfully")
}

// =============================================================================
// Content Entries Handlers
// =============================================================================

func (e *DashboardExtension) ServeEntriesList(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	searchQuery := c.Query("search")
	statusFilter := c.Query("status")
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Get entries
	contentTypeID, _ := xid.FromString(contentType.ID)
	result, err := e.plugin.entrySvc.List(ctx, contentTypeID, &core.ListEntriesQuery{
		Search:   searchQuery,
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListEntriesResponse{Entries: []*core.ContentEntryDTO{}}
	}

	// Get stats
	stats, _ := e.plugin.entrySvc.GetStats(ctx, contentTypeID)

	content := pages.EntriesListPage(currentApp, basePath, contentType, result.Entries, stats, page, pageSize, result.TotalItems, searchQuery, statusFilter)

	pageData := components.PageData{
		Title:      contentType.Name + " Entries",
		User:       currentUser,
		ActivePage: "cms-entries",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) ServeCreateEntry(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	errMsg := c.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Resolve component references for object/array fields
	e.resolveComponentReferences(ctx, contentType)

	content := pages.CreateEntryPage(currentApp, basePath, contentType, errMsg)

	pageData := components.PageData{
		Title:      "Create " + contentType.Name,
		User:       currentUser,
		ActivePage: "cms-entry-create",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) HandleCreateEntry(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form data into map
	data := make(map[string]any)
	for _, field := range contentType.Fields {
		value := c.FormValue("data[" + field.Name + "]")

		// Handle boolean fields specially (checkboxes send "true" string or nothing)
		if field.Type == "boolean" {
			data[field.Name] = value == "true" || value == "on" || value == "1"
			continue
		}

		if value != "" {
			// Parse JSON for object, array, oneOf, and json field types
			if field.Type == "object" || field.Type == "array" || field.Type == "oneOf" || field.Type == "json" {
				var parsedValue any
				if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
					data[field.Name] = parsedValue
				} else {
					// If parsing fails, store as-is (validation will catch it)
					data[field.Name] = value
				}
			} else if field.Type == "number" || field.Type == "integer" || field.Type == "float" {
				// Parse number fields to preserve numeric types
				if field.Type == "integer" {
					if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
						data[field.Name] = intVal
					} else {
						data[field.Name] = value
					}
				} else {
					if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
						data[field.Name] = floatVal
					} else {
						data[field.Name] = value
					}
				}
			} else {
				data[field.Name] = value
			}
		}
	}

	// Create entry
	req := &core.CreateEntryRequest{
		Data:   data,
		Status: "draft",
	}

	result, err := e.plugin.entrySvc.Create(ctx, contentTypeID, req)
	if err != nil {
		// Format validation errors with field details if available
		errorMsg := err.Error()

		// Try to extract validation details from the error
		if cmsErr, ok := err.(*errs.AuthsomeError); ok {
			if cmsErr.Code == core.ErrCodeEntryValidationFailed {
				if details, ok := cmsErr.Details.(map[string]string); ok && len(details) > 0 {
					errorMsg = "Validation failed:\n"
					for field, msg := range details {
						errorMsg += fmt.Sprintf("• %s: %s\n", field, msg)
					}
				}
			}
		}

		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries/create?error="+url.QueryEscape(errorMsg))
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries/"+result.ID)
}

func (e *DashboardExtension) ServeEntryDetail(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	entryIDStr := c.Param("entryId")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Get entry
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid entry ID")
	}

	entry, err := e.plugin.entrySvc.GetByID(ctx, entryID)
	if err != nil {
		return c.String(http.StatusNotFound, "Entry not found")
	}

	// Get revisions
	var revisionDTOs []*core.ContentRevisionDTO
	if e.plugin.revisionSvc != nil {
		revisions, _ := e.plugin.revisionSvc.List(ctx, entryID, &core.ListRevisionsQuery{PageSize: 5})
		if revisions != nil && revisions.Items != nil {
			revisionDTOs = make([]*core.ContentRevisionDTO, len(revisions.Items))
			for i, rev := range revisions.Items {
				revisionDTOs[i] = &core.ContentRevisionDTO{
					ID:           rev.ID,
					EntryID:      rev.EntryID,
					Version:      rev.Version,
					Data:         rev.Data,
					ChangeReason: rev.Reason,
					ChangedBy:    rev.ChangedBy,
					CreatedAt:    rev.CreatedAt,
				}
			}
		}
	}

	content := pages.EntryDetailPage(currentApp, basePath, contentType, entry, revisionDTOs)

	pageData := components.PageData{
		Title:      "Entry Details",
		User:       currentUser,
		ActivePage: "cms-entry-detail",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) ServeEditEntry(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	entryIDStr := c.Param("entryId")
	errMsg := c.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Resolve component references for object/array fields
	e.resolveComponentReferences(ctx, contentType)

	// Get entry
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid entry ID")
	}

	entry, err := e.plugin.entrySvc.GetByID(ctx, entryID)
	if err != nil {
		return c.String(http.StatusNotFound, "Entry not found")
	}

	content := pages.EditEntryPage(currentApp, basePath, contentType, entry, errMsg)

	pageData := components.PageData{
		Title:      "Edit Entry",
		User:       currentUser,
		ActivePage: "cms-entry-edit",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) HandleUpdateEntry(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	entryIDStr := c.Param("entryId")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(ctx, typeName)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Get entry ID
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid entry ID")
	}

	// Parse form data into map
	data := make(map[string]any)
	for _, field := range contentType.Fields {
		value := c.FormValue("data[" + field.Name + "]")

		// Handle boolean fields specially (checkboxes send "true" string or nothing)
		if field.Type == "boolean" {
			data[field.Name] = value == "true" || value == "on" || value == "1"
			continue
		}

		if value != "" {
			// Parse JSON for object, array, oneOf, and json field types
			if field.Type == "object" || field.Type == "array" || field.Type == "oneOf" || field.Type == "json" {
				var parsedValue any
				if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
					data[field.Name] = parsedValue
				} else {
					// If parsing fails, store as-is (validation will catch it)
					data[field.Name] = value
				}
			} else if field.Type == "number" || field.Type == "integer" || field.Type == "float" {
				// Parse number fields to preserve numeric types
				if field.Type == "integer" {
					if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
						data[field.Name] = intVal
					} else {
						data[field.Name] = value
					}
				} else {
					if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
						data[field.Name] = floatVal
					} else {
						data[field.Name] = value
					}
				}
			} else {
				data[field.Name] = value
			}
		}
	}

	// Update entry
	status := c.FormValue("status")
	req := &core.UpdateEntryRequest{
		Data:   data,
		Status: status,
	}

	_, err = e.plugin.entrySvc.Update(ctx, entryID, req)
	if err != nil {
		// Format validation errors with field details if available
		errorMsg := err.Error()

		// Try to extract validation details from the error
		if cmsErr, ok := err.(*errs.AuthsomeError); ok {
			if cmsErr.Code == core.ErrCodeEntryValidationFailed {
				if details, ok := cmsErr.Details.(map[string]string); ok && len(details) > 0 {
					errorMsg = "Validation failed:\n"
					for field, msg := range details {
						errorMsg += fmt.Sprintf("• %s: %s\n", field, msg)
					}
				}
			}
		}

		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries/"+entryIDStr+"/edit?error="+url.QueryEscape(errorMsg))
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries/"+entryIDStr)
}

// HandleDeleteEntry handles deleting a content entry
func (e *DashboardExtension) HandleDeleteEntry(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	typeName := c.Param("typeName")
	entryIDStr := c.Param("entryId")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get entry ID
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries?error=Invalid+entry+ID")
	}

	// Delete the entry
	err = e.plugin.entrySvc.Delete(ctx, entryID)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries?error="+url.QueryEscape(err.Error()))
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeName+"/entries?success=Entry+deleted+successfully")
}

// =============================================================================
// Component Schema Handlers
// =============================================================================

func (e *DashboardExtension) ServeComponentSchemasList(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()

	searchQuery := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	// Get component schemas
	result, err := e.plugin.componentSchemaSvc.List(ctx, &core.ListComponentSchemasQuery{
		Search:   searchQuery,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListComponentSchemasResponse{Components: []*core.ComponentSchemaSummaryDTO{}}
	}

	content := pages.ComponentSchemasPage(currentApp, basePath, result.Components, page, pageSize, result.TotalItems, searchQuery)

	pageData := components.PageData{
		Title:      "Component Schemas",
		User:       currentUser,
		ActivePage: "cms-components",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) ServeCreateComponentSchema(c forge.Context) error {
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

	basePath := handler.GetBasePath()
	errMsg := c.Query("error")

	content := pages.CreateComponentSchemaPage(currentApp, basePath, errMsg)

	pageData := components.PageData{
		Title:      "Create Component Schema",
		User:       currentUser,
		ActivePage: "cms-components",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) HandleCreateComponentSchema(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Parse nested fields from JSON
	var fields []core.NestedFieldDefDTO
	fieldsJSON := c.FormValue("fields")
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/create?error=Invalid+fields+format")
		}
	}

	// Create request
	req := &core.CreateComponentSchemaRequest{
		Title:       c.FormValue("name"),
		Name:        c.FormValue("slug"),
		Description: c.FormValue("description"),
		Icon:        c.FormValue("icon"),
		Fields:      fields,
	}

	// Create component schema
	result, err := e.plugin.componentSchemaSvc.Create(ctx, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/create?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/"+result.Name)
}

func (e *DashboardExtension) ServeComponentSchemaDetail(c forge.Context) error {
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

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	componentSlug := c.Param("componentSlug")
	errMsg := c.Query("error")

	// Get component schema
	component, err := e.plugin.componentSchemaSvc.GetByName(ctx, componentSlug)
	if err != nil {
		return c.String(http.StatusNotFound, "Component schema not found")
	}

	content := pages.EditComponentSchemaPage(currentApp, basePath, component, errMsg)

	pageData := components.PageData{
		Title:      "Edit " + component.Name,
		User:       currentUser,
		ActivePage: "cms-components",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

func (e *DashboardExtension) HandleUpdateComponentSchema(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	componentSlug := c.Param("componentSlug")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get existing component to get its ID
	component, err := e.plugin.componentSchemaSvc.GetByName(ctx, componentSlug)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/components?error=Component+not+found")
	}

	componentID, _ := xid.FromString(component.ID)

	// Parse nested fields from JSON
	var fields []core.NestedFieldDefDTO
	fieldsJSON := c.FormValue("fields")
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/"+componentSlug+"?error=Invalid+fields+format")
		}
	}

	// Create update request
	req := &core.UpdateComponentSchemaRequest{
		Title:       c.FormValue("name"),
		Description: c.FormValue("description"),
		Icon:        c.FormValue("icon"),
		Fields:      fields,
	}

	// Update component schema
	_, err = e.plugin.componentSchemaSvc.Update(ctx, componentID, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/"+componentSlug+"?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/"+componentSlug+"?success=Component+updated")
}

func (e *DashboardExtension) HandleDeleteComponentSchema(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)
	basePath := handler.GetBasePath()
	componentSlug := c.Param("componentSlug")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get existing component to get its ID
	component, err := e.plugin.componentSchemaSvc.GetByName(ctx, componentSlug)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/components?error=Component+not+found")
	}

	componentID, _ := xid.FromString(component.ID)

	// Delete component schema
	err = e.plugin.componentSchemaSvc.Delete(ctx, componentID)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/components/"+componentSlug+"?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/components?success=Component+deleted")
}

// =============================================================================
// Helper Functions
// =============================================================================

// resolveComponentReferences resolves ComponentRef to NestedFields for object/array fields
func (e *DashboardExtension) resolveComponentReferences(ctx context.Context, contentType *core.ContentTypeDTO) {
	if contentType == nil || len(contentType.Fields) == 0 {
		return
	}

	// Fields is []*ContentFieldDTO, so contentType.Fields[i] is already a pointer
	for i := range contentType.Fields {
		e.resolveFieldComponentRef(ctx, contentType.Fields[i])
	}
}

// resolveFieldComponentRef resolves a single field's ComponentRef recursively
func (e *DashboardExtension) resolveFieldComponentRef(ctx context.Context, field *core.ContentFieldDTO) {
	if field == nil {
		return
	}

	// Handle object and array types
	if field.Type == "object" || field.Type == "array" {
		// If ComponentRef is set and NestedFields is empty, resolve the reference
		if field.Options.ComponentRef != "" && len(field.Options.NestedFields) == 0 {
			componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, field.Options.ComponentRef)
			if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
				field.Options.NestedFields = componentSchema.Fields
			}
		}

		// Recursively resolve nested fields that might have their own ComponentRef
		for i := range field.Options.NestedFields {
			nestedField := &field.Options.NestedFields[i]
			if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
				e.resolveNestedFieldComponentRef(ctx, nestedField)
			}
		}
		return
	}

	// Handle oneOf type - resolve ComponentRef for each schema option
	if field.Type == "oneOf" && len(field.Options.Schemas) > 0 {
		for key, schemaOpt := range field.Options.Schemas {
			modified := false

			// If this schema option has a ComponentRef, resolve it to NestedFields
			if schemaOpt.ComponentRef != "" && len(schemaOpt.NestedFields) == 0 {
				componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, schemaOpt.ComponentRef)
				if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
					schemaOpt.NestedFields = componentSchema.Fields
					modified = true
				}
			}

			// Recursively resolve nested fields within the schema option
			for i := range schemaOpt.NestedFields {
				nestedField := &schemaOpt.NestedFields[i]
				if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
					e.resolveNestedFieldComponentRef(ctx, nestedField)
					modified = true
				}
			}

			// Write back to map if any changes were made
			if modified {
				field.Options.Schemas[key] = schemaOpt
			}
		}
	}
}

// resolveNestedFieldComponentRef resolves ComponentRef for nested fields recursively
func (e *DashboardExtension) resolveNestedFieldComponentRef(ctx context.Context, field *core.NestedFieldDefDTO) {
	if field == nil || field.Options == nil {
		return
	}

	// Handle object and array types
	if field.Type == "object" || field.Type == "array" {
		// If ComponentRef is set and NestedFields is empty, resolve the reference
		if field.Options.ComponentRef != "" && len(field.Options.NestedFields) == 0 {
			componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, field.Options.ComponentRef)
			if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
				field.Options.NestedFields = componentSchema.Fields
			}
		}

		// Recursively resolve nested fields that might have their own ComponentRef
		for i := range field.Options.NestedFields {
			nestedField := &field.Options.NestedFields[i]
			if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
				e.resolveNestedFieldComponentRef(ctx, nestedField)
			}
		}
		return
	}

	// Handle oneOf type in nested fields
	if field.Type == "oneOf" && len(field.Options.Schemas) > 0 {
		for key, schemaOpt := range field.Options.Schemas {
			modified := false

			if schemaOpt.ComponentRef != "" && len(schemaOpt.NestedFields) == 0 {
				componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, schemaOpt.ComponentRef)
				if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
					schemaOpt.NestedFields = componentSchema.Fields
					modified = true
				}
			}

			// Recursively resolve nested fields within the schema option
			for i := range schemaOpt.NestedFields {
				nestedField := &schemaOpt.NestedFields[i]
				if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
					e.resolveNestedFieldComponentRef(ctx, nestedField)
					modified = true
				}
			}

			// Write back to map if any changes were made
			if modified {
				field.Options.Schemas[key] = schemaOpt
			}
		}
	}
}
