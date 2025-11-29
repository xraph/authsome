package cms

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
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
					activePage == "cms-entry-edit"
			},
			RequiresPlugin: "cms",
		},
	}
}

// Routes returns dashboard routes
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
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
			Path:         "/cms/types/:typeSlug",
			Handler:      e.ServeContentTypeDetail,
			Name:         "cms.dashboard.types.detail",
			Summary:      "Content Type Detail",
			Description:  "View content type details and fields",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Add Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeSlug/fields",
			Handler:      e.HandleAddField,
			Name:         "cms.dashboard.fields.create",
			Summary:      "Add Field",
			Description:  "Add a new field to a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeSlug/fields/:fieldSlug/delete",
			Handler:      e.HandleDeleteField,
			Name:         "cms.dashboard.fields.delete",
			Summary:      "Delete Field",
			Description:  "Delete a field from a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Entries list
		{
			Method:       "GET",
			Path:         "/cms/types/:typeSlug/entries",
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
			Path:         "/cms/types/:typeSlug/entries/create",
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
			Path:         "/cms/types/:typeSlug/entries/create",
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
			Path:         "/cms/types/:typeSlug/entries/:entryId",
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
			Path:         "/cms/types/:typeSlug/entries/:entryId/edit",
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
			Path:         "/cms/types/:typeSlug/entries/:entryId/update",
			Handler:      e.HandleUpdateEntry,
			Name:         "cms.dashboard.entries.update",
			Summary:      "Update Entry",
			Description:  "Process entry update form",
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
	if appIDStr := c.Param("appId"); appIDStr != "" {
		if appID, err := xid.FromString(appIDStr); err == nil {
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
		Name:        c.FormValue("name"),
		Slug:        c.FormValue("slug"),
		Description: c.FormValue("description"),
		Icon:        c.FormValue("icon"),
	}

	// Create content type
	result, err := e.plugin.contentTypeSvc.Create(ctx, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/create?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+result.Slug)
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
	typeSlug := c.Param("typeSlug")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	// Get stats
	contentTypeID, _ := xid.FromString(contentType.ID)
	stats, _ := e.plugin.entrySvc.GetStats(ctx, contentTypeID)

	content := pages.ContentTypeDetailPage(currentApp, basePath, contentType, stats)

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
	typeSlug := c.Param("typeSlug")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values
	req := &core.CreateFieldRequest{
		Name:        c.FormValue("name"),
		Slug:        c.FormValue("slug"),
		Type:        c.FormValue("type"),
		Description: c.FormValue("description"),
		Required:    c.FormValue("required") == "true",
		Unique:      c.FormValue("unique") == "true",
		Indexed:     c.FormValue("indexed") == "true",
		Localized:   c.FormValue("localized") == "true",
	}

	// Create the field
	_, err = e.plugin.fieldSvc.Create(ctx, contentTypeID, req)
	if err != nil {
		// Redirect back with error
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"?error="+err.Error())
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug)
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
	typeSlug := c.Param("typeSlug")
	fieldSlug := c.Param("fieldSlug")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types?error=Content+type+not+found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Delete the field
	err = e.plugin.fieldSvc.DeleteBySlug(ctx, contentTypeID, fieldSlug)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"?error="+err.Error())
	}

	// Redirect back to content type detail
	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug)
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
	typeSlug := c.Param("typeSlug")
	searchQuery := c.Query("search")
	statusFilter := c.Query("status")
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
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
	typeSlug := c.Param("typeSlug")
	errMsg := c.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

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
	typeSlug := c.Param("typeSlug")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
	if err != nil {
		return c.String(http.StatusNotFound, "Content type not found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form data into map
	data := make(map[string]any)
	for _, field := range contentType.Fields {
		value := c.FormValue("data[" + field.Slug + "]")
		if value != "" {
			data[field.Slug] = value
		}
	}

	// Create entry
	req := &core.CreateEntryRequest{
		Data:   data,
		Status: "draft",
	}

	result, err := e.plugin.entrySvc.Create(ctx, contentTypeID, req)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"/entries/create?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"/entries/"+result.ID)
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
	typeSlug := c.Param("typeSlug")
	entryIDStr := c.Param("entryId")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
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
	typeSlug := c.Param("typeSlug")
	entryIDStr := c.Param("entryId")
	errMsg := c.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
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
	typeSlug := c.Param("typeSlug")
	entryIDStr := c.Param("entryId")
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetBySlug(ctx, typeSlug)
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
		value := c.FormValue("data[" + field.Slug + "]")
		if value != "" {
			data[field.Slug] = value
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
		return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"/entries/"+entryIDStr+"/edit?error="+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, appBase+"/cms/types/"+typeSlug+"/entries/"+entryIDStr)
}
