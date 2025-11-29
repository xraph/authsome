package secrets

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/plugins/secrets/core"
	"github.com/xraph/authsome/plugins/secrets/pages"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the secrets plugin
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
	return "secrets"
}

// NavigationItems returns navigation items for the dashboard
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "secrets",
			Label:    "Secrets",
			Icon:     lucide.KeyRound(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    60,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/dashboard/secrets"
				}
				return basePath + "/dashboard/app/" + currentApp.ID.String() + "/secrets"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "secrets" || activePage == "secret-detail" ||
					activePage == "secret-create" || activePage == "secret-edit" ||
					activePage == "secret-history"
			},
			RequiresPlugin: "secrets",
		},
	}
}

// Routes returns dashboard routes
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Secrets list
		{
			Method:       "GET",
			Path:         "/secrets",
			Handler:      e.ServeSecretsListPage,
			Name:         "secrets.dashboard.list",
			Summary:      "Secrets list",
			Description:  "View and manage application secrets",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create secret page
		{
			Method:       "GET",
			Path:         "/secrets/create",
			Handler:      e.ServeCreateSecretPage,
			Name:         "secrets.dashboard.create",
			Summary:      "Create secret",
			Description:  "Create a new secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create secret action
		{
			Method:       "POST",
			Path:         "/secrets/create",
			Handler:      e.HandleCreateSecret,
			Name:         "secrets.dashboard.create.submit",
			Summary:      "Submit create secret",
			Description:  "Process secret creation form",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Secret detail page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId",
			Handler:      e.ServeSecretDetailPage,
			Name:         "secrets.dashboard.detail",
			Summary:      "Secret details",
			Description:  "View secret details",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit secret page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId/edit",
			Handler:      e.ServeEditSecretPage,
			Name:         "secrets.dashboard.edit",
			Summary:      "Edit secret",
			Description:  "Edit an existing secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update secret action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/update",
			Handler:      e.HandleUpdateSecret,
			Name:         "secrets.dashboard.update",
			Summary:      "Update secret",
			Description:  "Process secret update form",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete secret action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/delete",
			Handler:      e.HandleDeleteSecret,
			Name:         "secrets.dashboard.delete",
			Summary:      "Delete secret",
			Description:  "Delete a secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Version history page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId/history",
			Handler:      e.ServeVersionHistoryPage,
			Name:         "secrets.dashboard.history",
			Summary:      "Version history",
			Description:  "View secret version history",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Rollback action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/rollback/:version",
			Handler:      e.HandleRollback,
			Name:         "secrets.dashboard.rollback",
			Summary:      "Rollback secret",
			Description:  "Rollback to a previous version",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Reveal value (AJAX)
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/reveal",
			Handler:      e.HandleRevealValue,
			Name:         "secrets.dashboard.reveal",
			Summary:      "Reveal secret value",
			Description:  "Get decrypted secret value",
			Tags:         []string{"Dashboard", "Secrets"},
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
			ID:            "secrets-settings",
			Label:         "Secrets Manager",
			Description:   "Configure secrets and encryption settings",
			Icon:          lucide.KeyRound(Class("h-5 w-5")),
			Category:      "security",
			Order:         40,
			Path:          "secrets",
			RequirePlugin: "secrets",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns dashboard widgets
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "secrets-count",
			Title: "Secrets",
			Icon:  lucide.KeyRound(Class("size-5")),
			Order: 50,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderSecretsWidget(currentApp)
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

	// Get environment ID from cookie or context
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

// parseSecretID parses a secret ID from URL parameter
func (e *DashboardExtension) parseSecretID(c forge.Context) (xid.ID, error) {
	idStr := c.Param("secretId")
	if idStr == "" {
		return xid.NilID(), fmt.Errorf("secret ID is required")
	}
	return xid.FromString(idStr)
}

// =============================================================================
// Widget Renderer
// =============================================================================

func (e *DashboardExtension) renderSecretsWidget(currentApp *app.App) g.Node {
	// Create a background context with app context
	ctx := context.Background()
	if currentApp != nil {
		ctx = contexts.SetAppID(ctx, currentApp.ID)
	}

	// Try to get stats - use default context handling
	stats, err := e.plugin.Service().GetStats(ctx)
	count := 0
	if err == nil && stats != nil {
		count = stats.TotalSecrets
	}

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text(fmt.Sprintf("%d", count)),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Total Secrets"),
		),
	)
}

// =============================================================================
// Common UI Components
// =============================================================================

// statsCard renders a statistics card
func (e *DashboardExtension) statsCard(title, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("mt-1 text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
			),
			Div(
				Class("rounded-full bg-violet-100 p-3 dark:bg-violet-900/30"),
				icon,
			),
		),
	)
}

// statusBadge renders a status badge
func (e *DashboardExtension) statusBadge(status string) g.Node {
	var classes string
	switch status {
	case "active", "success":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	case "expired", "error":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
	case "expiring":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
	default:
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}
	return Span(Class(classes), g.Text(status))
}

// valueTypeBadge renders a value type badge
func (e *DashboardExtension) valueTypeBadge(valueType string) g.Node {
	var classes, icon string
	switch valueType {
	case "json":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
		icon = "{}"
	case "yaml":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
		icon = "---"
	case "binary":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400"
		icon = "01"
	default:
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-gray-700 dark:text-gray-300"
		icon = "Aa"
	}

	return Span(
		Class(classes),
		Span(Class("font-mono"), g.Text(icon)),
		g.Text(valueType),
	)
}

// renderPagination renders pagination controls
func (e *DashboardExtension) renderPagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	items := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Previous"),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 border border-violet-600 rounded-md"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(
				Class("px-2 py-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Next"),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(items),
	)
}

// =============================================================================
// Secrets Sub-navigation
// =============================================================================

func (e *DashboardExtension) renderSecretsNav(currentApp *app.App, basePath, activePage string) g.Node {
	type navItem struct {
		label string
		path  string
		page  string
		icon  g.Node
	}

	items := []navItem{
		{"All Secrets", "/secrets", "secrets", lucide.List(Class("size-4"))},
	}

	navItems := make([]g.Node, 0, len(items))
	for _, item := range items {
		isActive := activePage == item.page
		classes := "inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors "
		if isActive {
			classes += "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
		} else {
			classes += "text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800"
		}

		navItems = append(navItems, A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+item.path),
			Class(classes),
			item.icon,
			g.Text(item.label),
		))
	}

	return Nav(
		Class("flex flex-wrap gap-2 mb-6 p-2 bg-slate-50 dark:bg-gray-800/50 rounded-lg"),
		g.Group(navItems),
	)
}

// =============================================================================
// Page Handlers - Placeholders (to be implemented in pages/ folder)
// =============================================================================

// ServeSecretsListPage serves the secrets list page
func (e *DashboardExtension) ServeSecretsListPage(c forge.Context) error {
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

	// Get query parameters
	query := &core.ListSecretsQuery{
		Prefix:   c.QueryDefault("prefix", ""),
		Search:   c.QueryDefault("search", ""),
		PageSize: 20,
		Page:     1,
	}

	if p := c.QueryDefault("page", ""); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			query.Page = parsed
		}
	}

	// Get secrets
	secrets, pag, err := e.plugin.Service().List(ctx, query)
	if err != nil {
		secrets = []*core.SecretDTO{}
		pag = nil
	}

	basePath := handler.GetBasePath()
	content := e.renderSecretsListContent(currentApp, basePath, secrets, pag, query)

	pageData := components.PageData{
		Title:      "Secrets Manager",
		User:       currentUser,
		ActivePage: "secrets",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeCreateSecretPage serves the create secret page
func (e *DashboardExtension) ServeCreateSecretPage(c forge.Context) error {
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
	content := e.renderCreateSecretForm(currentApp, basePath, nil, "")

	pageData := components.PageData{
		Title:      "Create Secret",
		User:       currentUser,
		ActivePage: "secret-create",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// HandleCreateSecret handles the create secret form submission
func (e *DashboardExtension) HandleCreateSecret(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := e.injectContext(c)

	// Parse form
	req := &core.CreateSecretRequest{
		Path:        c.FormValue("path"),
		Value:       c.FormValue("value"),
		ValueType:   c.FormValue("valueType"),
		Schema:      c.FormValue("schema"),
		Description: c.FormValue("description"),
	}

	if tags := c.FormValue("tags"); tags != "" {
		req.Tags = splitTags(tags)
	}

	// Create secret
	_, err = e.plugin.Service().Create(ctx, req)
	if err != nil {
		basePath := handler.GetBasePath()
		content := e.renderCreateSecretForm(currentApp, basePath, req, err.Error())
		pageData := components.PageData{
			Title:      "Create Secret",
			User:       currentUser,
			ActivePage: "secret-create",
			BasePath:   basePath,
			CurrentApp: currentApp,
			Error:      err.Error(),
		}
		return handler.RenderWithLayout(c, pageData, content)
	}

	// Redirect to list
	return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/app/"+currentApp.ID.String()+"/secrets")
}

// ServeSecretDetailPage serves the secret detail page
func (e *DashboardExtension) ServeSecretDetailPage(c forge.Context) error {
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

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	ctx := e.injectContext(c)

	// Get secret
	secret, err := e.plugin.Service().Get(ctx, secretID)
	if err != nil {
		return c.String(http.StatusNotFound, "Secret not found")
	}

	// Get recent versions
	versions, _, _ := e.plugin.Service().GetVersions(ctx, secretID, 1, 5)

	basePath := handler.GetBasePath()
	content := e.renderSecretDetailContent(currentApp, basePath, secret, versions)

	pageData := components.PageData{
		Title:      "Secret Details",
		User:       currentUser,
		ActivePage: "secret-detail",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeEditSecretPage serves the edit secret page
func (e *DashboardExtension) ServeEditSecretPage(c forge.Context) error {
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

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	ctx := e.injectContext(c)

	secret, err := e.plugin.Service().Get(ctx, secretID)
	if err != nil {
		return c.String(http.StatusNotFound, "Secret not found")
	}

	basePath := handler.GetBasePath()
	content := e.renderEditSecretForm(currentApp, basePath, secret, "")

	pageData := components.PageData{
		Title:      "Edit Secret",
		User:       currentUser,
		ActivePage: "secret-edit",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// HandleUpdateSecret handles the update secret form submission
func (e *DashboardExtension) HandleUpdateSecret(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	ctx := e.injectContext(c)

	// Parse form
	req := &core.UpdateSecretRequest{
		Description:  c.FormValue("description"),
		ChangeReason: c.FormValue("changeReason"),
	}

	if value := c.FormValue("value"); value != "" {
		req.Value = value
	}
	if valueType := c.FormValue("valueType"); valueType != "" {
		req.ValueType = valueType
	}
	if tags := c.FormValue("tags"); tags != "" {
		req.Tags = splitTags(tags)
	}

	// Update secret
	_, err = e.plugin.Service().Update(ctx, secretID, req)
	if err != nil {
		secret, _ := e.plugin.Service().Get(ctx, secretID)
		basePath := handler.GetBasePath()
		content := e.renderEditSecretForm(currentApp, basePath, secret, err.Error())
		pageData := components.PageData{
			Title:      "Edit Secret",
			User:       currentUser,
			ActivePage: "secret-edit",
			BasePath:   basePath,
			CurrentApp: currentApp,
			Error:      err.Error(),
		}
		return handler.RenderWithLayout(c, pageData, content)
	}

	// Redirect to detail
	return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/app/"+currentApp.ID.String()+"/secrets/"+secretID.String())
}

// HandleDeleteSecret handles the delete secret action
func (e *DashboardExtension) HandleDeleteSecret(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	ctx := e.injectContext(c)

	if err := e.plugin.Service().Delete(ctx, secretID); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete secret: "+err.Error())
	}

	return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/app/"+currentApp.ID.String()+"/secrets")
}

// ServeVersionHistoryPage serves the version history page
func (e *DashboardExtension) ServeVersionHistoryPage(c forge.Context) error {
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

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	ctx := e.injectContext(c)

	secret, err := e.plugin.Service().Get(ctx, secretID)
	if err != nil {
		return c.String(http.StatusNotFound, "Secret not found")
	}

	page := 1
	if p := c.QueryDefault("page", ""); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	versions, pag, _ := e.plugin.Service().GetVersions(ctx, secretID, page, 20)

	basePath := handler.GetBasePath()
	content := e.renderVersionHistoryContent(currentApp, basePath, secret, versions, pag)

	pageData := components.PageData{
		Title:      "Version History",
		User:       currentUser,
		ActivePage: "secret-history",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// HandleRollback handles the rollback action
func (e *DashboardExtension) HandleRollback(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid secret ID")
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid version number")
	}

	ctx := e.injectContext(c)

	_, err = e.plugin.Service().Rollback(ctx, secretID, version, "Rollback from dashboard")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Rollback failed: "+err.Error())
	}

	return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/app/"+currentApp.ID.String()+"/secrets/"+secretID.String())
}

// HandleRevealValue handles the reveal value AJAX request
func (e *DashboardExtension) HandleRevealValue(c forge.Context) error {
	secretID, err := e.parseSecretID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid secret ID"})
	}

	ctx := e.injectContext(c)

	value, err := e.plugin.Service().GetValue(ctx, secretID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	secret, _ := e.plugin.Service().Get(ctx, secretID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"value":     value,
		"valueType": secret.ValueType,
	})
}

// =============================================================================
// Page Rendering Methods
// =============================================================================

// =============================================================================
// Page Rendering Methods
// =============================================================================

// renderSecretsListContent renders the secrets list page content
func (e *DashboardExtension) renderSecretsListContent(
	currentApp *app.App,
	basePath string,
	secrets []*core.SecretDTO,
	pag *pagination.Pagination,
	query *core.ListSecretsQuery,
) g.Node {
	return pages.SecretsListPage(currentApp, basePath, secrets, pag, query)
}

// renderCreateSecretForm renders the create secret form
func (e *DashboardExtension) renderCreateSecretForm(
	currentApp *app.App,
	basePath string,
	prefill *core.CreateSecretRequest,
	errorMsg string,
) g.Node {
	return pages.CreateSecretPage(currentApp, basePath, prefill, errorMsg)
}

// renderSecretDetailContent renders the secret detail page content
func (e *DashboardExtension) renderSecretDetailContent(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
) g.Node {
	return pages.SecretDetailPage(currentApp, basePath, secret, versions)
}

// renderEditSecretForm renders the edit secret form
func (e *DashboardExtension) renderEditSecretForm(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	errorMsg string,
) g.Node {
	return pages.EditSecretPage(currentApp, basePath, secret, errorMsg)
}

// renderVersionHistoryContent renders the version history page content
func (e *DashboardExtension) renderVersionHistoryContent(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
	pag *pagination.Pagination,
) g.Node {
	return pages.VersionHistoryPage(currentApp, basePath, secret, versions, pag)
}
