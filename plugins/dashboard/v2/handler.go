package dashboard

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
)

// Handler handles dashboard HTTP requests
type Handler struct {
	assets            embed.FS
	userSvc           user.ServiceInterface
	sessionSvc        session.ServiceInterface
	auditSvc          *audit.Service
	rbacSvc           *rbac.Service
	apikeyService     *apikey.Service
	appService        app.Service
	orgService        *organization.Service
	envService        environment.EnvironmentService
	db                *bun.DB
	isMultiApp        bool
	basePath          string
	enabledPlugins    map[string]bool
	hookRegistry      *hooks.HookRegistry // For executing lifecycle hooks
	extensionRegistry *ExtensionRegistry  // For rendering extension navigation items and widgets
	configManager     forge.ConfigManager // For config viewer page
}

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

// NewHandler creates a new dashboard handler
func NewHandler(
	assets embed.FS,
	appService app.Service,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	auditSvc *audit.Service,
	rbacSvc *rbac.Service,
	apikeyService *apikey.Service,
	orgService *organization.Service,
	envService environment.EnvironmentService,
	db *bun.DB,
	isMultiApp bool,
	basePath string,
	enabledPlugins map[string]bool,
	hookRegistry *hooks.HookRegistry,
	configManager forge.ConfigManager,
) *Handler {
	h := &Handler{
		assets:         assets,
		appService:     appService,
		userSvc:        userSvc,
		sessionSvc:     sessionSvc,
		auditSvc:       auditSvc,
		rbacSvc:        rbacSvc,
		apikeyService:  apikeyService,
		orgService:     orgService,
		envService:     envService,
		db:             db,
		isMultiApp:     isMultiApp,
		basePath:       basePath,
		enabledPlugins: enabledPlugins,
		hookRegistry:   hookRegistry,
		configManager:  configManager,
	}

	return h
}

// enrichPageDataWithExtensions adds extension navigation items and widgets to PageData
func (h *Handler) enrichPageDataWithExtensions(pageData *components.PageData) {
	if h.extensionRegistry == nil {
		return
	}

	// Get navigation items for main position
	navItems := h.extensionRegistry.GetNavigationItems(ui.NavPositionMain, h.enabledPlugins)

	// Render navigation items (for header nav)
	if len(navItems) > 0 {
		pageData.ExtensionNavItems = RenderNavigationItems(
			navItems,
			h.basePath,
			pageData.CurrentApp,
			pageData.ActivePage,
		)

		// Also populate raw nav data for sidebar rendering
		pageData.ExtensionNavData = make([]components.ExtensionNavItemData, 0, len(navItems))
		for _, item := range navItems {
			isActive := false
			if item.ActiveChecker != nil {
				isActive = item.ActiveChecker(pageData.ActivePage)
			}
			url := item.URLBuilder(h.basePath, pageData.CurrentApp)

			pageData.ExtensionNavData = append(pageData.ExtensionNavData, components.ExtensionNavItemData{
				Label:    item.Label,
				Icon:     item.Icon,
				URL:      url,
				IsActive: isActive,
			})
		}
	}

	// Get dashboard widgets
	widgets := h.extensionRegistry.GetDashboardWidgets()
	if len(widgets) > 0 {
		widgetNodes := make([]g.Node, 0, len(widgets))
		for _, widget := range widgets {
			if widget.Renderer != nil && pageData.CurrentApp != nil {
				widgetNodes = append(widgetNodes, widget.Renderer(h.basePath, pageData.CurrentApp))
			}
		}
		pageData.ExtensionWidgets = widgetNodes
	}
}

// extractAndInjectAppID extracts appId from URL param and injects it into request context
// Returns the updated context and the app, or an error if invalid/unauthorized
func (h *Handler) extractAndInjectAppID(c forge.Context) (context.Context, *app.App, error) {
	ctx := c.Request().Context()

	// Extract appId from URL param
	appIDStr := c.Param("appId")
	if appIDStr == "" {
		return ctx, nil, fmt.Errorf("app ID is required")
	}

	// Parse appId
	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return ctx, nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	// Validate that the app exists and user has access
	appEntity, err := h.appService.FindAppByID(ctx, appID)
	if err != nil {
		return ctx, nil, fmt.Errorf("app not found: %w", err)
	}

	// Verify user is a member of this app
	user := h.getUserFromContext(c)
	if user != nil {
		isMember, err := h.appService.IsUserMember(ctx, appID, user.ID)
		if err != nil || !isMember {
			return ctx, nil, fmt.Errorf("access denied: you are not a member of this app")
		}
	}

	// Inject app ID into context for downstream services
	ctx = contexts.SetAppID(ctx, appID)

	// Extract and inject environment ID
	ctx, env, err := h.extractAndInjectEnvironmentID(c, ctx, appID)
	if err != nil {
		// If environment extraction fails, log but don't fail the request
		// This allows the dashboard to work even if environments aren't set up yet
	}
	_ = env // Environment is stored in context, no need to return it here

	return ctx, appEntity, nil
}

// extractAndInjectEnvironmentID extracts environment from cookie or gets default, then injects into context
func (h *Handler) extractAndInjectEnvironmentID(c forge.Context, ctx context.Context, appID xid.ID) (context.Context, *environment.Environment, error) {
	// Try to get environment from cookie first
	env, err := h.getEnvironmentFromCookie(c, appID)
	if err != nil {
		// No cookie or invalid cookie, get default environment for app
		env, err = h.envService.GetDefaultEnvironment(ctx, appID)
		if err != nil {
			return ctx, nil, fmt.Errorf("failed to get default environment: %w", err)
		}

		// Set cookie with default environment
		h.setEnvironmentCookie(c, env.ID)
	}

	// Inject environment ID into context
	ctx = contexts.SetEnvironmentID(ctx, env.ID)

	return ctx, env, nil
}

// getUserRoleForApp gets the user's RBAC role for a specific app from the user_roles table
func (h *Handler) getUserRoleForApp(ctx context.Context, userID, appID xid.ID) string {
	// Query user_roles table with role relation to get the role name
	var userRoles []struct {
		UserID   xid.ID `bun:"user_id"`
		AppID    xid.ID `bun:"app_id"`
		RoleID   xid.ID `bun:"role_id"`
		RoleName string `bun:"role__name"`
	}

	err := h.db.NewSelect().
		TableExpr("user_roles").
		ColumnExpr("user_roles.user_id").
		ColumnExpr("user_roles.app_id").
		ColumnExpr("user_roles.role_id").
		ColumnExpr("roles.name AS role__name").
		Join("LEFT JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Where("user_roles.app_id = ?", appID).
		Where("user_roles.deleted_at IS NULL").
		Limit(1).
		Scan(ctx, &userRoles)

	if err != nil || len(userRoles) == 0 {
		// If no role found in user_roles, return "member" as default
		return "member"
	}

	return userRoles[0].RoleName
}

// getUserApps gets all apps the user belongs to (for app switcher)
func (h *Handler) getUserApps(ctx context.Context, userID xid.ID) ([]*app.App, error) {
	// Get user's memberships
	memberships, err := h.appService.GetUserMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get app details for each membership, de-duplicating by app ID
	seenApps := make(map[xid.ID]bool)
	apps := make([]*app.App, 0)
	for _, membership := range memberships {
		if seenApps[membership.AppID] {
			continue // Skip duplicate
		}
		appEntity, err := h.appService.FindAppByID(ctx, membership.AppID)
		if err == nil {
			apps = append(apps, appEntity)
			seenApps[membership.AppID] = true
		}
	}

	return apps, nil
}

// ServeStatic serves static assets (CSS, JS, images)
func (h *Handler) ServeStatic(c forge.Context) error {
	// Get the wildcard path from the route parameter
	// The route is registered as "/ui/static/*" so we get everything after /static/
	path := c.Param("*")

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid path"))
	}

	// Read file from embedded assets
	fullPath := filepath.Join("static", path)
	content, err := fs.ReadFile(h.assets, fullPath)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("asset not found"))
	}

	// Get content type - use Go's mime package first, fallback to our custom function
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = getContentType(ext)
	}

	// Write directly to response writer to ensure correct Content-Type
	w := c.Response()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year cache
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(content)
	return writeErr
}

// Helper methods

// render renders a gomponent node
func (h *Handler) render(c forge.Context, node g.Node) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return node.Render(c.Response())
}

// RenderWithLayout renders content with the dashboard layout (public for extensions)
// This method automatically populates app, environment, and extension data
func (h *Handler) RenderWithLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	ctx := c.Request().Context()

	// Set common fields
	pageData.Year = time.Now().Year()
	pageData.EnabledPlugins = h.enabledPlugins

	// Get user if not already set
	if pageData.User == nil {
		pageData.User = h.getUserFromContext(c)
	}

	// Prepopulate app data if CurrentApp is set
	if pageData.CurrentApp != nil {
		// Get user apps for switcher if not already set
		if pageData.UserApps == nil && pageData.User != nil {
			userApps, _ := h.getUserApps(ctx, pageData.User.ID)
			pageData.UserApps = userApps
			pageData.ShowAppSwitcher = len(userApps) > 0
		}

		// Prepopulate environment data if not already set
		if pageData.CurrentEnvironment == nil {
			currentEnv, _ := h.getEnvironmentFromCookie(c, pageData.CurrentApp.ID)
			if currentEnv == nil {
				// Fall back to default environment
				currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, pageData.CurrentApp.ID)
				if currentEnv != nil {
					h.setEnvironmentCookie(c, currentEnv.ID)
				}
			}
			pageData.CurrentEnvironment = currentEnv
		}

		// Get user environments if not already set
		if pageData.UserEnvironments == nil {
			environments, _ := h.getUserEnvironments(ctx, pageData.CurrentApp.ID)
			pageData.UserEnvironments = environments
			pageData.ShowEnvSwitcher = len(environments) > 0
		}
	}

	// Enrich with extension navigation items and widgets
	h.enrichPageDataWithExtensions(&pageData)

	page := components.BaseSidebarLayout(pageData, content)
	return h.render(c, page)
}

// RenderWithBaseLayout renders content with an empty layout (public for extensions)
func (h *Handler) RenderWithBaseLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	ctx := c.Request().Context()

	// Set common fields
	pageData.Year = time.Now().Year()
	pageData.EnabledPlugins = h.enabledPlugins

	// Get user if not already set
	if pageData.User == nil {
		pageData.User = h.getUserFromContext(c)
	}

	// Prepopulate app data if CurrentApp is set
	if pageData.CurrentApp != nil {
		// Get user apps for switcher if not already set
		if pageData.UserApps == nil && pageData.User != nil {
			userApps, _ := h.getUserApps(ctx, pageData.User.ID)
			pageData.UserApps = userApps
			pageData.ShowAppSwitcher = len(userApps) > 0
		}

		// Prepopulate environment data if not already set
		if pageData.CurrentEnvironment == nil {
			currentEnv, _ := h.getEnvironmentFromCookie(c, pageData.CurrentApp.ID)
			if currentEnv == nil {
				// Fall back to default environment
				currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, pageData.CurrentApp.ID)
				if currentEnv != nil {
					h.setEnvironmentCookie(c, currentEnv.ID)
				}
			}
			pageData.CurrentEnvironment = currentEnv
		}

		// Get user environments if not already set
		if pageData.UserEnvironments == nil {
			environments, _ := h.getUserEnvironments(ctx, pageData.CurrentApp.ID)
			pageData.UserEnvironments = environments
			pageData.ShowEnvSwitcher = len(environments) > 0
		}
	}

	// Enrich with extension navigation items and widgets
	h.enrichPageDataWithExtensions(&pageData)

	page := components.EmptyLayout(pageData, content)
	return h.render(c, page)
}

// RenderWithHeaderLayout renders content with a header layout (public for extensions)
func (h *Handler) RenderWithHeaderLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	ctx := c.Request().Context()

	// Set common fields
	pageData.Year = time.Now().Year()
	pageData.EnabledPlugins = h.enabledPlugins

	// Get user if not already set
	if pageData.User == nil {
		pageData.User = h.getUserFromContext(c)
	}

	// Prepopulate app data if CurrentApp is set
	if pageData.CurrentApp != nil {
		// Get user apps for switcher if not already set
		if pageData.UserApps == nil && pageData.User != nil {
			userApps, _ := h.getUserApps(ctx, pageData.User.ID)
			pageData.UserApps = userApps
			pageData.ShowAppSwitcher = len(userApps) > 0
		}

		// Prepopulate environment data if not already set
		if pageData.CurrentEnvironment == nil {
			currentEnv, _ := h.getEnvironmentFromCookie(c, pageData.CurrentApp.ID)
			if currentEnv == nil {
				// Fall back to default environment
				currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, pageData.CurrentApp.ID)
				if currentEnv != nil {
					h.setEnvironmentCookie(c, currentEnv.ID)
				}
			}
			pageData.CurrentEnvironment = currentEnv
		}

		// Get user environments if not already set
		if pageData.UserEnvironments == nil {
			environments, _ := h.getUserEnvironments(ctx, pageData.CurrentApp.ID)
			pageData.UserEnvironments = environments
			pageData.ShowEnvSwitcher = len(environments) > 0
		}
	}

	// Enrich with extension navigation items and widgets
	h.enrichPageDataWithExtensions(&pageData)

	page := components.BaseLayout(pageData, content)
	return h.render(c, page)
}

// RenderWithSidebarLayout renders content with a sidebar layout (public for extensions)
func (h *Handler) RenderWithSidebarLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	ctx := c.Request().Context()

	// Set common fields
	pageData.Year = time.Now().Year()
	pageData.EnabledPlugins = h.enabledPlugins

	// Get user if not already set
	if pageData.User == nil {
		pageData.User = h.getUserFromContext(c)
	}

	// Prepopulate app data if CurrentApp is set
	if pageData.CurrentApp != nil {
		// Get user apps for switcher if not already set
		if pageData.UserApps == nil && pageData.User != nil {
			userApps, _ := h.getUserApps(ctx, pageData.User.ID)
			pageData.UserApps = userApps
			pageData.ShowAppSwitcher = len(userApps) > 0
		}

		// Prepopulate environment data if not already set
		if pageData.CurrentEnvironment == nil {
			currentEnv, _ := h.getEnvironmentFromCookie(c, pageData.CurrentApp.ID)
			if currentEnv == nil {
				// Fall back to default environment
				currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, pageData.CurrentApp.ID)
				if currentEnv != nil {
					h.setEnvironmentCookie(c, currentEnv.ID)
				}
			}
			pageData.CurrentEnvironment = currentEnv
		}

		// Get user environments if not already set
		if pageData.UserEnvironments == nil {
			environments, _ := h.getUserEnvironments(ctx, pageData.CurrentApp.ID)
			pageData.UserEnvironments = environments
			pageData.ShowEnvSwitcher = len(environments) > 0
		}
	}

	// Enrich with extension navigation items and widgets
	h.enrichPageDataWithExtensions(&pageData)

	page := components.BaseSidebarLayout(pageData, content)
	return h.render(c, page)
}

// renderWithLayout is the internal version (kept for backward compatibility)
func (h *Handler) renderWithLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	return h.RenderWithLayout(c, pageData, content)
}

// renderWithBaseLayout is the internal version (kept for backward compatibility)
func (h *Handler) renderWithBaseLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	return h.RenderWithBaseLayout(c, pageData, content)
}

// renderWithHeaderLayout is the internal version (kept for backward compatibility)
func (h *Handler) renderWithHeaderLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	return h.RenderWithHeaderLayout(c, pageData, content)
}

// renderWithSidebarLayout is the internal version (kept for backward compatibility)
func (h *Handler) renderWithSidebarLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	return h.RenderWithSidebarLayout(c, pageData, content)
}

// renderError renders an error page
func (h *Handler) renderError(c forge.Context, message string, err error) error {
	user := h.getUserFromContext(c)

	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	pageData := components.PageData{
		Title:          "Error",
		User:           user,
		CSRFToken:      h.getCSRFToken(c),
		ActivePage:     "",
		BasePath:       h.basePath,
		IsMultiApp:     h.isMultiApp,
		Error:          errorMsg,
		Year:           time.Now().Year(),
		EnabledPlugins: h.enabledPlugins,
	}

	content := pages.ErrorPage(errorMsg, h.basePath)
	page := components.BaseLayout(pageData, content)
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return h.render(c, page)
}

// Public helper methods for extensions

// GetUserFromContext returns the authenticated user from request context
func (h *Handler) GetUserFromContext(c forge.Context) *user.User {
	return h.getUserFromContext(c)
}

// GetCurrentApp extracts and returns the current app from URL parameter
func (h *Handler) GetCurrentApp(c forge.Context) (*app.App, error) {
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return nil, err
	}
	// Update request context
	*c.Request() = *c.Request().WithContext(ctx)
	return currentApp, nil
}

// GetUserApps returns all apps the user has access to
func (h *Handler) GetUserApps(c forge.Context, userID xid.ID) ([]*app.App, error) {
	ctx := c.Request().Context()
	return h.getUserApps(ctx, userID)
}

// GetCSRFToken returns the CSRF token for the request
func (h *Handler) GetCSRFToken(c forge.Context) string {
	return h.getCSRFToken(c)
}

// GetBasePath returns the dashboard base path
func (h *Handler) GetBasePath() string {
	return h.basePath
}

// GetEnabledPlugins returns map of enabled plugins
func (h *Handler) GetEnabledPlugins() map[string]bool {
	return h.enabledPlugins
}

// RenderSettingsPage renders content within the settings layout
func (h *Handler) RenderSettingsPage(c forge.Context, pageID string, content g.Node) error {
	return h.renderSettingsPage(c, pageID, content)
}

// GetCurrentEnvironment returns the current environment from cookie or default
func (h *Handler) GetCurrentEnvironment(c forge.Context, appID xid.ID) (*environment.Environment, error) {
	ctx := c.Request().Context()

	// Try to get from cookie first
	env, err := h.getEnvironmentFromCookie(c, appID)
	if err != nil {
		// No cookie or invalid cookie, get default environment
		env, err = h.envService.GetDefaultEnvironment(ctx, appID)
		if err != nil {
			return nil, err
		}
		// Set cookie with default environment
		h.setEnvironmentCookie(c, env.ID)
	}

	return env, nil
}

// GetUserEnvironments returns all environments for the given app
func (h *Handler) GetUserEnvironments(c forge.Context, appID xid.ID) ([]*environment.Environment, error) {
	ctx := c.Request().Context()
	return h.getUserEnvironments(ctx, appID)
}

// Private helper methods

// getUserFromContext retrieves the user from the request context
func (h *Handler) getUserFromContext(c forge.Context) *user.User {
	userVal := c.Request().Context().Value("user")
	if userVal == nil {
		return nil
	}

	user, ok := userVal.(*user.User)
	if !ok {
		return nil
	}

	return user
}

// getCSRFToken retrieves the CSRF token from the request context
func (h *Handler) getCSRFToken(c forge.Context) string {
	tokenVal := c.Request().Context().Value("csrf_token")
	if tokenVal == nil {
		return ""
	}

	token, ok := tokenVal.(string)
	if !ok {
		return ""
	}

	return token
}

// getEnvironmentFromCookie retrieves the selected environment ID from cookie
func (h *Handler) getEnvironmentFromCookie(c forge.Context, appID xid.ID) (*environment.Environment, error) {
	cookie, err := c.Request().Cookie(environmentCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil, fmt.Errorf("no environment cookie found")
	}

	envID, err := xid.FromString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID in cookie: %w", err)
	}

	// Fetch the environment and verify it belongs to the app
	env, err := h.envService.GetEnvironment(c.Request().Context(), envID)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Verify environment belongs to the current app
	if env.AppID != appID {
		return nil, fmt.Errorf("environment does not belong to current app")
	}

	return env, nil
}

// setEnvironmentCookie stores the selected environment ID in a cookie
func (h *Handler) setEnvironmentCookie(c forge.Context, envID xid.ID) {
	cookie := &http.Cookie{
		Name:     environmentCookieName,
		Value:    envID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30, // 30 days
	}
	http.SetCookie(c.Response(), cookie)
}

// clearEnvironmentCookie removes the environment cookie
func (h *Handler) clearEnvironmentCookie(c forge.Context) {
	cookie := &http.Cookie{
		Name:     environmentCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	}
	http.SetCookie(c.Response(), cookie)
}

// getUserEnvironments retrieves all environments for the given app
func (h *Handler) getUserEnvironments(ctx context.Context, appID xid.ID) ([]*environment.Environment, error) {
	envs, err := h.envService.ListEnvironments(ctx, &environment.ListEnvironmentsFilter{
		AppID: appID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	return envs.Data, nil
}

// getEnvironmentData retrieves current environment and environment list for PageData
func (h *Handler) getEnvironmentData(c forge.Context, ctx context.Context, appID xid.ID) (currentEnv *environment.Environment, environments []*environment.Environment) {
	// Get all environments for current app
	environments, err := h.getUserEnvironments(ctx, appID)
	if err != nil {
		environments = []*environment.Environment{} // Fallback to empty if failed
	}

	// Get current environment (from context)
	currentEnv, _ = h.getEnvironmentFromCookie(c, appID)
	if currentEnv == nil {
		// Try to get default
		currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, appID)
	}

	return currentEnv, environments
}

// checkExistingSession checks if there's a valid session without middleware
func (h *Handler) checkExistingSession(c forge.Context) *user.User {
	// Extract session token from cookie
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil
	}

	sessionToken := cookie.Value

	// Validate session
	sess, err := h.sessionSvc.FindByToken(c.Request().Context(), sessionToken)
	if err != nil || sess == nil {
		return nil
	}

	// Check if session is expired
	if time.Now().After(sess.ExpiresAt) {
		return nil
	}

	// Set app context from session for user lookup (required for multi-tenancy)
	ctx := c.Request().Context()
	if !sess.AppID.IsNil() {
		ctx = contexts.SetAppID(ctx, sess.AppID)
	}

	// Get user information
	user, err := h.userSvc.FindByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return nil
	}

	return user
}

// isFirstUser checks if there are any users in the system
func (h *Handler) isFirstUser(ctx context.Context) (bool, error) {
	// Check if platform app exists and has any members
	platformApp, err := h.appService.GetPlatformApp(ctx)
	if err != nil {
		// No platform app exists - this is definitely the first user
		return true, nil
	}

	// Count members in the platform app
	count, err := h.appService.CountMembers(ctx, platformApp.ID)
	if err != nil {
		return false, fmt.Errorf("failed to count members: %w", err)
	}

	// If no members exist, this is the first user
	return count == 0, nil
}

// generateCSRFToken generates a simple CSRF token
func (h *Handler) generateCSRFToken() string {
	return xid.New().String()
}

// renderSettingsPage is a helper to render any settings page with the sidebar layout
func (h *Handler) renderSettingsPage(c forge.Context, pageID string, content g.Node) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/login")
	}

	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	userApps, err := h.getUserApps(ctx, currentUser.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	// Build settings navigation from extensions
	navItems := h.buildSettingsNavigation(currentApp)

	layoutData := components.SettingsLayoutData{
		NavItems:    navItems,
		ActivePage:  pageID,
		BasePath:    h.basePath,
		CurrentApp:  currentApp,
		PageContent: content,
	}

	pageData := components.PageData{
		Title:              "Settings",
		User:               currentUser,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "settings",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	settingsPage := components.SettingsLayout(layoutData)
	return h.renderWithLayout(c, pageData, settingsPage)
}

// buildSettingsNavigation builds the settings sidebar navigation
func (h *Handler) buildSettingsNavigation(currentApp *app.App) []components.SettingsNavItem {
	var navItems []components.SettingsNavItem

	// Core settings pages - General
	navItems = append(navItems, components.SettingsNavItem{
		ID:       "general",
		Label:    "General",
		Icon:     nil, // Will use default settings icon in layout
		URL:      components.BuildSettingsURL(h.basePath, currentApp.ID.String(), "general"),
		Category: "general",
	})

	// Add pages from extensions
	if h.extensionRegistry != nil {
		pages := h.extensionRegistry.GetSettingsPages(h.enabledPlugins)
		for _, page := range pages {
			navItems = append(navItems, components.SettingsNavItem{
				ID:            page.ID,
				Label:         page.Label,
				Icon:          page.Icon,
				URL:           components.BuildSettingsURL(h.basePath, currentApp.ID.String(), page.Path),
				Category:      page.Category,
				RequirePlugin: page.RequirePlugin,
			})
		}
	}

	return navItems
}

// getContentType returns the appropriate content type for file extensions
func getContentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
