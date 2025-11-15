package dashboard

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
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
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
)

// Handler handles dashboard HTTP requests
type Handler struct {
	assets         embed.FS
	userSvc        user.ServiceInterface
	sessionSvc     session.ServiceInterface
	auditSvc       *audit.Service
	rbacSvc        *rbac.Service
	apikeyService  *apikey.Service
	appService     app.Service
	orgService     *organization.Service
	envService     environment.EnvironmentService
	db             *bun.DB
	isMultiApp     bool
	basePath       string
	enabledPlugins map[string]bool
	hookRegistry   *hooks.HookRegistry // For executing lifecycle hooks
}

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
	}

	return h
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
		fmt.Printf("[Dashboard] Warning: Could not extract environment: %v\n", err)
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

// ServeAppsList serves the dashboard index page - shows app cards or redirects to default app
func (h *Handler) ServeAppsList(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	ctx := c.Request().Context()

	// Check if multiapp mode is enabled
	if !h.isMultiApp {
		// Standalone mode - redirect to default app
		return h.redirectToDefaultApp(c, ctx)
	}

	// Multiapp mode - show app cards
	return h.renderAppCards(c, ctx, currentUser)
}

// redirectToDefaultApp finds the default app and redirects to it
func (h *Handler) redirectToDefaultApp(c forge.Context, ctx context.Context) error {
	// Get the platform/default app
	platformApp, err := h.appService.GetPlatformApp(ctx)
	if err != nil {
		return h.renderError(c, "Failed to find default app", err)
	}

	// Redirect to the app's dashboard
	redirectURL := h.basePath + "/dashboard/app/" + platformApp.ID.String() + "/"
	return c.Redirect(http.StatusFound, redirectURL)
}

// renderAppCards renders the app cards for multiapp mode
func (h *Handler) renderAppCards(c forge.Context, ctx context.Context, currentUser *user.User) error {
	// Get user's memberships to find their apps
	memberships, err := h.appService.GetUserMemberships(ctx, currentUser.ID)
	if err != nil {
		return h.renderError(c, "Failed to load your apps", err)
	}

	// Deduplicate apps by app ID
	seenApps := make(map[xid.ID]bool)
	appCards := make([]*pages.AppCardData, 0)

	for _, membership := range memberships {
		// Skip if we've already processed this app
		if seenApps[membership.AppID] {
			continue
		}
		seenApps[membership.AppID] = true

		// Get the app details
		appEntity, err := h.appService.FindAppByID(ctx, membership.AppID)
		if err != nil {
			// Skip apps we can't load
			continue
		}

		// Get member count for this app
		memberCount, _ := h.appService.CountMembers(ctx, appEntity.ID)

		// Get the user's actual RBAC role for this app from user_roles table
		role := h.getUserRoleForApp(ctx, currentUser.ID, appEntity.ID)

		appCards = append(appCards, &pages.AppCardData{
			App:         appEntity,
			Role:        role,
			MemberCount: memberCount,
		})
	}

	// Check if user can create apps (for now, always show if multiapp enabled)
	canCreateApps := h.isMultiApp

	pageData := components.PageData{
		Title:           "Your Apps",
		User:            currentUser,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "apps",
		BasePath:        h.basePath,
		EnabledPlugins:  h.enabledPlugins,
		ShowAppSwitcher: false, // Don't show switcher on app list page
	}

	appsListData := pages.AppsListPageData{
		Apps:              appCards,
		BasePath:          h.basePath,
		CanCreateApps:     canCreateApps,
		ShowCreateAppCard: canCreateApps,
	}

	content := pages.AppsListPage(appsListData)
	return h.renderWithLayout(c, pageData, content)
}

// PageData represents common data for all pages
type PageData struct {
	Title          string
	User           *user.User
	CSRFToken      string
	ActivePage     string
	BasePath       string
	Data           interface{}
	Error          string
	Success        string
	Year           int
	EnabledPlugins map[string]bool
}

// ServeDashboard serves the main dashboard page
func (h *Handler) ServeDashboard(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{} // Fallback to empty if failed
	}

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	// Get dashboard stats (now app-scoped via context)
	stats, err := h.getDashboardStats(ctx)
	if err != nil {
		return h.renderError(c, "Failed to load dashboard statistics", err)
	}

	pageData := components.PageData{
		Title:              "Dashboard",
		ActivePage:         "dashboard",
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Convert to pages.DashboardStats
	pageStats := &pages.DashboardStats{
		TotalUsers:     stats.TotalUsers,
		ActiveUsers:    stats.ActiveUsers,
		NewUsersToday:  stats.NewUsersToday,
		TotalSessions:  stats.TotalSessions,
		ActiveSessions: stats.ActiveSessions,
		FailedLogins:   stats.FailedLogins,
		UserGrowth:     stats.UserGrowth,
		SessionGrowth:  stats.SessionGrowth,
		RecentActivity: convertActivityItems(stats.RecentActivity),
		SystemStatus:   convertStatusItems(stats.SystemStatus),
		Plugins:        convertPluginItems(stats.Plugins),
	}

	content := pages.DashboardPage(pageStats, h.basePath, currentApp.ID.String())
	return h.renderWithLayout(c, pageData, content)
}

// Helper converters for stats
func convertActivityItems(items []ActivityItem) []pages.ActivityItem {
	result := make([]pages.ActivityItem, len(items))
	for i, item := range items {
		result[i] = pages.ActivityItem{
			Title:       item.Title,
			Description: item.Description,
			Time:        item.Time,
			Type:        item.Type,
		}
	}
	return result
}

func convertStatusItems(items []StatusItem) []pages.StatusItem {
	result := make([]pages.StatusItem, len(items))
	for i, item := range items {
		result[i] = pages.StatusItem{
			Name:   item.Name,
			Status: item.Status,
			Color:  item.Color,
		}
	}
	return result
}

func convertPluginItems(items []PluginItem) []pages.PluginItem {
	result := make([]pages.PluginItem, len(items))
	for i, item := range items {
		result[i] = pages.PluginItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Category:    item.Category,
			Status:      item.Status,
			Icon:        item.Icon,
		}
	}
	return result
}

// ServeUsers serves the users list page
func (h *Handler) ServeUsers(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, currentUser.ID)
	if err != nil {
		userApps = []*app.App{} // Fallback to empty if failed
	}

	// Get pagination parameters
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		fmt.Sscanf(pageParam, "%d", &page)
	}

	pageSize := 20
	if sizeParam := c.Query("size"); sizeParam != "" {
		fmt.Sscanf(sizeParam, "%d", &pageSize)
	}

	// Parse search query
	query := c.Query("q")

	// Build filter for listing users
	filter := &user.ListUsersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: pageSize,
		},
		AppID: currentApp.ID,
	}

	// Add search filter if query provided
	if query != "" {
		filter.Search = &query
	}

	// Fetch users
	response, err := h.userSvc.ListUsers(ctx, filter)
	if err != nil {
		return h.renderError(c, "Failed to load users", err)
	}

	users := response.Data
	total := int(response.Pagination.Total)
	totalPages := response.Pagination.TotalPages

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	pageData := components.PageData{
		Title:              "Users",
		User:               currentUser,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "users",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	usersData := pages.UsersData{
		Users:      users,
		Query:      query,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
	}

	content := pages.UsersPage(usersData, h.basePath, currentApp.ID.String())
	return h.renderWithLayout(c, pageData, content)
}

// ServeUserDetail serves a single user detail page
func (h *Handler) ServeUserDetail(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get user ID from URL
	userIDStr := c.Param("id")
	if userIDStr == "" {
		return h.renderError(c, "Invalid user ID", nil)
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return h.renderError(c, "Invalid user ID format", err)
	}

	// Get user details (app-scoped via context)
	targetUser, err := h.userSvc.FindByID(ctx, userID)
	if err != nil {
		return h.renderError(c, "User not found", err)
	}

	// Get active sessions for this user (limit to 10 for detail view)
	sessionFilter := &session.ListSessionsFilter{
		AppID:  currentApp.ID,
		UserID: &userID,
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 10,
		},
	}
	sessionResponse, err := h.sessionSvc.ListSessions(ctx, sessionFilter)
	allSessions := []*session.Session{}
	if err == nil && sessionResponse != nil {
		allSessions = sessionResponse.Data
	}

	// Convert sessions to page data format
	sessionData := make([]pages.SessionData, 0, len(allSessions))
	for _, s := range allSessions {
		sessionData = append(sessionData, pages.SessionData{
			ID:        s.ID.String(),
			UserID:    s.UserID.String(),
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		})
	}

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	pageData := components.PageData{
		Title:              fmt.Sprintf("User: %s", targetUser.Email),
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "users",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	detailData := pages.UserDetailPageData{
		User: pages.UserDetailData{
			ID:            targetUser.ID.String(),
			Email:         targetUser.Email,
			Name:          targetUser.Name,
			Username:      targetUser.Username,
			EmailVerified: targetUser.EmailVerified,
			CreatedAt:     targetUser.CreatedAt,
			UpdatedAt:     targetUser.UpdatedAt,
		},
		Sessions:  sessionData,
		BasePath:  h.basePath,
		CSRFToken: h.getCSRFToken(c),
	}

	content := pages.UserDetailPage(detailData)
	return h.renderWithLayout(c, pageData, content)
}

// ServeUserEdit serves the user edit page
func (h *Handler) ServeUserEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get user ID from path
	userID := c.Param("id")
	id, err := xid.FromString(userID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Fetch user details
	targetUser, err := h.userSvc.FindByID(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	pageData := components.PageData{
		Title:           fmt.Sprintf("Edit User: %s", targetUser.Email),
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "users",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	editData := pages.UserEditPageData{
		User: pages.UserEditData{
			UserID:        targetUser.ID.String(),
			Name:          targetUser.Name,
			Email:         targetUser.Email,
			Username:      targetUser.Username,
			EmailVerified: targetUser.EmailVerified,
		},
		BasePath:  h.basePath,
		CSRFToken: h.getCSRFToken(c),
	}

	content := pages.UserEditPage(editData)
	return h.renderWithLayout(c, pageData, content)
}

// HandleUserEdit processes the user edit form
func (h *Handler) HandleUserEdit(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, _, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get user ID from path
	userID := c.Param("id")
	id, err := xid.FromString(userID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Fetch user details
	targetUser, err := h.userSvc.FindByID(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	// Update user fields
	name := c.Request().FormValue("name")
	email := c.Request().FormValue("email")
	username := c.Request().FormValue("username")
	emailVerified := c.Request().FormValue("email_verified") == "true"

	if name == "" || email == "" {
		return c.String(http.StatusBadRequest, "Name and email are required")
	}

	// Update user
	updateReq := &user.UpdateUserRequest{
		Name:          &name,
		Email:         &email,
		EmailVerified: &emailVerified,
	}

	if username != "" {
		updateReq.Username = &username
	}

	updatedUser, err := h.userSvc.Update(ctx, targetUser, updateReq)
	if err != nil {
		fmt.Printf("[Dashboard] Failed to update user: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to update user")
	}

	fmt.Printf("[Dashboard] User %s updated by admin %s\n", updatedUser.ID, currentUser.ID)

	// Redirect back to user detail page with success message
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/users/"+userID+"?success=updated")
}

// HandleUserDelete deletes a user
func (h *Handler) HandleUserDelete(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, _, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get user ID from path
	userID := c.Param("id")
	id, err := xid.FromString(userID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Prevent self-deletion
	if id.String() == currentUser.ID.String() {
		return c.String(http.StatusBadRequest, "Cannot delete your own account")
	}

	// Delete user
	if err := h.userSvc.Delete(ctx, id); err != nil {
		fmt.Printf("[Dashboard] Failed to delete user: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to delete user")
	}

	fmt.Printf("[Dashboard] User %s deleted by admin %s\n", userID, currentUser.ID)

	// Redirect to users list with success message
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/users?success=deleted")
}

// ServeSessions serves the sessions list page
func (h *Handler) ServeSessions(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, currentUser.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Parse search query
	query := c.Query("q")

	// Fetch all active sessions for current app
	sessionFilter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000,
		},
		AppID: currentApp.ID,
	}
	sessionResponse, err := h.sessionSvc.ListSessions(ctx, sessionFilter)
	allSessions := []*session.Session{}
	if err == nil && sessionResponse != nil {
		allSessions = sessionResponse.Data
	} else if err != nil {
		fmt.Printf("[Dashboard] Failed to list sessions: %v\n", err)
	}

	// Filter sessions if search query provided
	var sessions []*session.Session
	if query != "" {
		queryLower := strings.ToLower(query)
		for _, sess := range allSessions {
			// Search by User ID or IP Address
			if strings.Contains(strings.ToLower(sess.UserID.String()), queryLower) ||
				strings.Contains(strings.ToLower(sess.IPAddress), queryLower) {
				sessions = append(sessions, sess)
			}
		}
	} else {
		sessions = allSessions
	}

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	pageData := components.PageData{
		Title:              "Sessions",
		User:               currentUser,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "sessions",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Convert sessions to page data format
	sessionData := make([]pages.SessionData, len(sessions))
	for i, s := range sessions {
		sessionData[i] = pages.SessionData{
			ID:        s.ID.String(),
			UserID:    s.UserID.String(),
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		}
	}

	// Calculate statistics from all sessions (not filtered)
	avgDuration, sessionsToday, sessionsThisWeek := h.calculateSessionStatistics(allSessions)

	sessionsPageData := pages.SessionsPageData{
		Sessions:         sessionData,
		Query:            query,
		BasePath:         h.basePath,
		CSRFToken:        h.getCSRFToken(c),
		AvgDuration:      avgDuration,
		SessionsToday:    sessionsToday,
		SessionsThisWeek: sessionsThisWeek,
	}

	content := pages.SessionsPage(sessionsPageData)
	return h.renderWithLayout(c, pageData, content)
}

// HandleRevokeSession revokes a single session
func (h *Handler) HandleRevokeSession(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, _, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get session ID from path
	sessionID := c.Param("id")
	id, err := xid.FromString(sessionID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid session ID")
	}

	// Revoke session
	if err := h.sessionSvc.RevokeByID(ctx, id); err != nil {
		fmt.Printf("[Dashboard] Failed to revoke session: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to revoke session")
	}

	fmt.Printf("[Dashboard] Session %s revoked by admin %s\n", sessionID, currentUser.ID)

	// Redirect back to sessions list with success message
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/sessions?success=revoked")
}

// HandleEnvironmentSwitch switches the current environment
func (h *Handler) HandleEnvironmentSwitch(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get target environment ID from form
	envIDStr := c.FormValue("env_id")
	if envIDStr == "" {
		return c.String(http.StatusBadRequest, "Environment ID is required")
	}

	envID, err := xid.FromString(envIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment ID")
	}

	// Verify environment exists and belongs to current app
	env, err := h.envService.GetEnvironment(ctx, envID)
	if err != nil {
		return c.String(http.StatusNotFound, "Environment not found")
	}

	if env.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Environment does not belong to current app")
	}

	// Set environment cookie
	h.setEnvironmentCookie(c, envID)

	// Get referrer or redirect to dashboard
	referer := c.Request().Header.Get("Referer")
	if referer == "" {
		referer = h.basePath + "/dashboard/app/" + currentApp.ID.String() + "/"
	}

	return c.Redirect(http.StatusFound, referer)
}

// ServeEnvironments renders the environments list page
func (h *Handler) ServeEnvironments(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		return h.renderError(c, "Failed to load apps", err)
	}

	// Get all environments for current app
	environments, err := h.getUserEnvironments(ctx, currentApp.ID)
	if err != nil {
		return h.renderError(c, "Failed to load environments", err)
	}

	// Get current environment (from context)
	currentEnv, _ := h.getEnvironmentFromCookie(c, currentApp.ID)
	if currentEnv == nil {
		// Try to get default
		currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, currentApp.ID)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:              "Environments - Dashboard",
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "environments",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Prepare environments data
	envsData := pages.EnvironmentsData{
		Environments: environments,
		Pagination:   nil, // TODO: Add pagination if needed
	}

	// Render page
	content := pages.EnvironmentsPage(envsData, h.basePath, appIDStr)
	return h.renderWithLayout(c, pageData, content)
}

// ServeEnvironmentDetail renders the environment detail page
func (h *Handler) ServeEnvironmentDetail(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get environment ID from URL
	envIDStr := c.Param("envId")
	envID, err := xid.FromString(envIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment ID")
	}

	// Get the environment
	env, err := h.envService.GetEnvironment(ctx, envID)
	if err != nil {
		return h.renderError(c, "Environment not found", err)
	}

	// Verify environment belongs to current app
	if env.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Environment does not belong to current app")
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		return h.renderError(c, "Failed to load apps", err)
	}

	// Get all environments for current app
	environments, err := h.getUserEnvironments(ctx, currentApp.ID)
	if err != nil {
		return h.renderError(c, "Failed to load environments", err)
	}

	// Get current environment (from context)
	currentEnv, _ := h.getEnvironmentFromCookie(c, currentApp.ID)
	if currentEnv == nil {
		// Try to get default
		currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, currentApp.ID)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:              env.Name + " - Environments - Dashboard",
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "environments",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Prepare environment detail data
	envData := pages.EnvironmentDetailData{
		Environment: env,
	}

	// Render page
	content := pages.EnvironmentDetailPage(envData, h.basePath, appIDStr)
	return h.renderWithLayout(c, pageData, content)
}

// ServeEnvironmentCreate renders the create environment page
func (h *Handler) ServeEnvironmentCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		return h.renderError(c, "Failed to load apps", err)
	}

	// Get all environments for current app
	environments, err := h.getUserEnvironments(ctx, currentApp.ID)
	if err != nil {
		return h.renderError(c, "Failed to load environments", err)
	}

	// Get current environment (from context)
	currentEnv, _ := h.getEnvironmentFromCookie(c, currentApp.ID)
	if currentEnv == nil {
		// Try to get default
		currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, currentApp.ID)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:              "Create Environment - Dashboard",
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "environments",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Render page
	content := pages.EnvironmentCreatePage(h.basePath, appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleEnvironmentCreate processes environment creation
func (h *Handler) HandleEnvironmentCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Parse form
	name := c.FormValue("name")
	slug := c.FormValue("slug")
	envType := c.FormValue("type")
	// isDefault := c.FormValue("is_default") == "true"

	// Validate required fields
	if name == "" || slug == "" || envType == "" {
		return c.String(http.StatusBadRequest, "Name, slug, and type are required")
	}

	// Create environment request
	req := &environment.CreateEnvironmentRequest{
		AppID: currentApp.ID,
		Name:  name,
		Slug:  slug,
		Type:  envType,
		// IsDefault: isDefault,
	}

	// Create environment
	env, err := h.envService.CreateEnvironment(ctx, req)
	if err != nil {
		return h.renderError(c, "Failed to create environment", err)
	}

	// Redirect to environment detail
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/app/"+appIDStr+"/environments/"+env.ID.String())
}

// ServeEnvironmentEdit renders the edit environment page
func (h *Handler) ServeEnvironmentEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get environment ID from URL
	envIDStr := c.Param("envId")
	envID, err := xid.FromString(envIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment ID")
	}

	// Get the environment
	env, err := h.envService.GetEnvironment(ctx, envID)
	if err != nil {
		return h.renderError(c, "Environment not found", err)
	}

	// Verify environment belongs to current app
	if env.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Environment does not belong to current app")
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		return h.renderError(c, "Failed to load apps", err)
	}

	// Get all environments for current app
	environments, err := h.getUserEnvironments(ctx, currentApp.ID)
	if err != nil {
		return h.renderError(c, "Failed to load environments", err)
	}

	// Get current environment (from context)
	currentEnv, _ := h.getEnvironmentFromCookie(c, currentApp.ID)
	if currentEnv == nil {
		// Try to get default
		currentEnv, _ = h.envService.GetDefaultEnvironment(ctx, currentApp.ID)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:              "Edit " + env.Name + " - Dashboard",
		User:               user,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "environments",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	// Prepare environment edit data
	envData := pages.EnvironmentEditData{
		Environment: env,
	}

	// Render page
	content := pages.EnvironmentEditPage(envData, h.basePath, appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleEnvironmentEdit processes environment update
func (h *Handler) HandleEnvironmentEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get environment ID from URL
	envIDStr := c.Param("envId")
	envID, err := xid.FromString(envIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment ID")
	}

	// Get the environment
	env, err := h.envService.GetEnvironment(ctx, envID)
	if err != nil {
		return h.renderError(c, "Environment not found", err)
	}

	// Verify environment belongs to current app
	if env.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Environment does not belong to current app")
	}

	// Parse form
	name := c.FormValue("name")
	slug := c.FormValue("slug")
	envType := c.FormValue("type")

	// Validate required fields
	if name == "" || slug == "" || envType == "" {
		return c.String(http.StatusBadRequest, "Name, slug, and type are required")
	}

	// Update environment request
	req := &environment.UpdateEnvironmentRequest{
		Name: &name,
		// Slug: slug,
		Type: &envType,
	}

	// Update environment
	updatedEnv, err := h.envService.UpdateEnvironment(ctx, envID, req)
	if err != nil {
		return h.renderError(c, "Failed to update environment", err)
	}

	// Redirect to environment detail
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/app/"+appIDStr+"/environments/"+updatedEnv.ID.String())
}

// HandleEnvironmentDelete processes environment deletion
func (h *Handler) HandleEnvironmentDelete(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get environment ID from URL
	envIDStr := c.Param("envId")
	envID, err := xid.FromString(envIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment ID")
	}

	// Get the environment
	env, err := h.envService.GetEnvironment(ctx, envID)
	if err != nil {
		return h.renderError(c, "Environment not found", err)
	}

	// Verify environment belongs to current app
	if env.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Environment does not belong to current app")
	}

	// Check if it's the default environment
	if env.IsDefault {
		return c.String(http.StatusForbidden, "Cannot delete the default environment")
	}

	// Delete environment
	if err := h.envService.DeleteEnvironment(ctx, envID); err != nil {
		return h.renderError(c, "Failed to delete environment", err)
	}

	// Clear cookie if this was the selected environment
	currentEnvCookie, _ := h.getEnvironmentFromCookie(c, currentApp.ID)
	if currentEnvCookie != nil && currentEnvCookie.ID == envID {
		h.clearEnvironmentCookie(c)
	}

	// Redirect to environments list
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/app/"+appIDStr+"/environments")
}

// Serve404 serves the 404 page
func (h *Handler) Serve404(c forge.Context) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusNotFound)
	page := pages.NotFound(h.basePath)
	return h.render(c, page)
}

// ServeSettings serves the settings page
func (h *Handler) ServeSettings(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, currentUser.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get active tab from query parameter
	activeTab := c.Query("tab")
	if activeTab == "" {
		activeTab = "general"
	}

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

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

	// Populate settings data
	// TODO: Load these from actual configuration

	// Debug: Log enabled plugins
	fmt.Printf("[Dashboard] Settings page - Enabled plugins: %v\n", h.enabledPlugins)
	fmt.Printf("[Dashboard] Settings page - API Key plugin enabled: %v\n", h.enabledPlugins["apikey"])

	// Prepare API Keys data
	apiKeysData := pages.APIKeysTabPageData{
		APIKeys:       []pages.APIKeyData{}, // TODO: Fetch from h.apikeyService
		Organizations: []pages.OrganizationOption{},
		IsSaaSMode:    h.isMultiApp,
		CanCreateKeys: true,
		CSRFToken:     h.getCSRFToken(c),
	}

	// If SaaS mode, fetch user's organizations
	if h.isMultiApp && h.orgService != nil {
		// TODO: Fetch user's organizations and populate apiKeysData.Organizations
	}

	settingsData := pages.SettingsPageData{
		ActiveTab: activeTab,
		General: pages.GeneralSettings{
			DashboardName:            "AuthSome Dashboard",
			SessionDuration:          24,
			MaxLoginAttempts:         5,
			RequireEmailVerification: true,
		},
		APIKeys:               apiKeysData,
		Webhooks:              []pages.Webhook{},              // TODO: Load from webhook service
		NotificationTemplates: []pages.NotificationTemplate{}, // TODO: Load from notification service
		SocialProviders:       []pages.SocialProvider{},       // TODO: Load from social auth service
		ImpersonationLogs:     []pages.ImpersonationLog{},     // TODO: Load from impersonation service
		MFAMethods:            []pages.MFAMethod{},            // TODO: Load from MFA service
		IsSaaSMode:            h.isMultiApp,
		BasePath:              h.basePath,
		CSRFToken:             h.getCSRFToken(c),
		EnabledPlugins:        h.enabledPlugins,
	}

	content := pages.SettingsPage(settingsData)
	return h.renderWithLayout(c, pageData, content)
}

// ServePlugins serves the plugins management page
func (h *Handler) ServePlugins(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get all user apps for the switcher
	userApps, err := h.getUserApps(ctx, currentUser.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get filter parameters
	filterStatus := c.Query("status")
	if filterStatus == "" {
		filterStatus = "all"
	}

	filterCategory := c.Query("category")

	// Get plugin information from stats (reuse the getPluginInfo function)
	statsPlugins := h.getPluginInfo()

	// Convert to pages.PluginItem
	plugins := make([]pages.PluginItem, len(statsPlugins))
	for i, p := range statsPlugins {
		plugins[i] = pages.PluginItem{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Category:    p.Category,
			Status:      p.Status,
			Icon:        p.Icon,
		}
	}

	// Get environment data for PageData
	currentEnv, environments := h.getEnvironmentData(c, ctx, currentApp.ID)

	pageData := components.PageData{
		Title:              "Plugins",
		User:               currentUser,
		CSRFToken:          h.getCSRFToken(c),
		ActivePage:         "plugins",
		BasePath:           h.basePath,
		IsMultiApp:         h.isMultiApp,
		CurrentApp:         currentApp,
		UserApps:           userApps,
		ShowAppSwitcher:    len(userApps) > 0,
		CurrentEnvironment: currentEnv,
		UserEnvironments:   environments,
		ShowEnvSwitcher:    len(environments) > 0,
	}

	pluginsPageData := pages.PluginsPageData{
		Plugins:        plugins,
		FilterStatus:   filterStatus,
		FilterCategory: filterCategory,
		BasePath:       h.basePath,
		CSRFToken:      h.getCSRFToken(c),
	}

	content := pages.PluginsPage(pluginsPageData)
	return h.renderWithLayout(c, pageData, content)
}

// HandleRevokeUserSessions revokes all sessions for a specific user
func (h *Handler) HandleRevokeUserSessions(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	// Get user ID from form
	userIDStr := c.Request().FormValue("user_id")
	if userIDStr == "" {
		return c.String(http.StatusBadRequest, "User ID is required")
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Get all sessions for this user in current app
	sessionFilter := &session.ListSessionsFilter{
		AppID:  currentApp.ID,
		UserID: &userID,
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000,
		},
	}
	sessionResponse, err := h.sessionSvc.ListSessions(ctx, sessionFilter)
	sessions := []*session.Session{}
	if err == nil && sessionResponse != nil {
		sessions = sessionResponse.Data
	} else if err != nil {
		fmt.Printf("[Dashboard] Failed to list user sessions: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to list user sessions")
	}

	// Revoke each session
	revokedCount := 0
	for _, sess := range sessions {
		if err := h.sessionSvc.RevokeByID(ctx, sess.ID); err != nil {
			fmt.Printf("[Dashboard] Failed to revoke session %s: %v\n", sess.ID, err)
			continue
		}
		revokedCount++
	}

	fmt.Printf("[Dashboard] %d sessions revoked for user %s by admin %s\n", revokedCount, userIDStr, currentUser.ID)

	// Redirect back with success message
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/sessions?success=revoked_all&count="+fmt.Sprintf("%d", revokedCount))
}

// ServeLogin serves the login page
func (h *Handler) ServeLogin(c forge.Context) error {
	fmt.Printf("[Dashboard] ServeLogin called for path: %s\n", c.Request().URL.Path)

	// Check if already authenticated (check session cookie directly since no auth middleware)
	if user := h.checkExistingSession(c); user != nil {
		fmt.Printf("[Dashboard] User already authenticated: %s, redirecting to dashboard\n", user.Email)
		// Already logged in, redirect to dashboard
		redirect := c.Query("redirect")
		if redirect == "" {
			redirect = h.basePath + "/dashboard/"
		}
		return c.Redirect(http.StatusFound, redirect)
	}

	fmt.Printf("[Dashboard] No valid session, showing login page\n")

	redirect := c.Query("redirect")
	errorParam := c.Query("error")

	// Check if this is the first user (show signup prominently)
	isFirstUser, _ := h.isFirstUser(c.Request().Context())

	// Map error codes to user-friendly messages
	var errorMessage string
	switch errorParam {
	case "admin_required":
		errorMessage = "Admin access required to view dashboard"
	case "invalid_session":
		errorMessage = "Your session is invalid. Please log in again"
	case "insufficient_permissions":
		errorMessage = "You don't have permission to access the dashboard"
	case "session_expired":
		errorMessage = "Your session has expired. Please log in again"
	}

	loginData := pages.LoginPageData{
		Title:     "Login",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     errorMessage,
		Data: pages.LoginData{
			Redirect:    redirect,
			ShowSignup:  true,
			IsFirstUser: isFirstUser,
		},
	}

	page := pages.Login(loginData)
	return h.render(c, page)
}

// HandleLogin processes the login form
func (h *Handler) HandleLogin(c forge.Context) error {
	// Check if already authenticated
	if user := h.checkExistingSession(c); user != nil {
		fmt.Printf("[Dashboard] User already authenticated during login attempt: %s, redirecting\n", user.Email)
		redirect := c.Request().FormValue("redirect")
		if redirect == "" {
			redirect = h.basePath + "/dashboard/"
		}
		return c.Redirect(http.StatusFound, redirect)
	}

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return h.renderLoginError(c, "Invalid form data", c.Query("redirect"))
	}

	email := c.Request().FormValue("email")
	password := c.Request().FormValue("password")
	redirect := c.Request().FormValue("redirect")
	csrfToken := c.Request().FormValue("csrf_token")

	// Note: CSRF validation is handled by the CSRF middleware
	// This check is redundant but kept for extra safety
	if csrfToken == "" {
		return h.renderLoginError(c, "Invalid CSRF token", redirect)
	}

	// Validate credentials
	if email == "" || password == "" {
		return h.renderLoginError(c, "Email and password are required", redirect)
	}

	// Get platform app to use as context
	ctx := c.Request().Context()
	platformApp, err := h.appService.GetPlatformApp(ctx)
	if err != nil {
		fmt.Printf("[Dashboard] Login error: Failed to get platform app: %v\n", err)
		return h.renderLoginError(c, "System configuration error. Please contact administrator.", redirect)
	}

	ctx = contexts.SetAppID(ctx, platformApp.ID)

	// Find user by email in platform app context
	user, err := h.userSvc.FindByAppAndEmail(ctx, platformApp.ID, email)
	fmt.Printf("[Dashboard] Login: Email: %s, Platform App: %s\n", email, platformApp.ID.String())
	fmt.Printf("[Dashboard] Login: User: %+v\n", user)
	fmt.Printf("[Dashboard] Login: Error: %+v\n", err)

	if err != nil || user == nil {
		fmt.Printf("[Dashboard] Login error: Failed to find user: %v\n", err)
		return h.renderLoginError(c, "Invalid email or password", redirect)
	}

	fmt.Printf("[Dashboard] Login: Found user %s (ID: %s), checking password...\n", user.Email, user.ID)
	fmt.Printf("[Dashboard] Login: Password hash length: %d, hash preview: %s...\n", len(user.PasswordHash), func() string {
		if len(user.PasswordHash) > 20 {
			return user.PasswordHash[:20]
		}
		return user.PasswordHash
	}())

	fmt.Printf("[Dashboard] Login: Password: %s\n", password)
	fmt.Printf("[Dashboard] Login: Password hash: %s\n", user.PasswordHash)

	// Verify password
	passwordValid := crypto.CheckPassword(password, user.PasswordHash)
	fmt.Printf("[Dashboard] Login: Password check result: %v\n", passwordValid)
	if !passwordValid {
		fmt.Printf("[Dashboard] Login error: Password verification failed for user %s\n", user.Email)
		return h.renderLoginError(c, "Invalid email or password", redirect)
	}

	fmt.Printf("[Dashboard] Login: Password verified successfully for user %s\n", user.Email)

	// Note: Role checking is handled by the RequireAdmin middleware
	// The middleware will check if the user has the required permissions
	// using the fast PermissionChecker after successful authentication

	// App membership verification happens in extractAndInjectAppID for app-scoped routes

	// Create session
	sess, err := h.sessionSvc.Create(c.Request().Context(), &session.CreateSessionRequest{
		UserID:    user.ID,
		IPAddress: c.Request().RemoteAddr,
		UserAgent: c.Request().UserAgent(),
		Remember:  false,
		AppID:     platformApp.ID,
	})
	if err != nil {
		return h.renderLoginError(c, "Failed to create session", redirect)
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil, // Auto-detect HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds()),
	}
	http.SetCookie(c.Response(), cookie)

	// Redirect to dashboard or specified redirect URL
	if redirect == "" {
		redirect = h.basePath + "/dashboard/"
	}
	return c.Redirect(http.StatusFound, redirect)
}

// renderLoginError renders the login page with an error message
func (h *Handler) renderLoginError(c forge.Context, message string, redirect string) error {
	loginData := pages.LoginPageData{
		Title:     "Login",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     message,
		Data: pages.LoginData{
			Redirect:   redirect,
			ShowSignup: true,
		},
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	page := pages.Login(loginData)
	return h.render(c, page)
}

// ServeSignup serves the signup page
func (h *Handler) ServeSignup(c forge.Context) error {
	// Check if already authenticated (check session cookie directly since no auth middleware)
	if user := h.checkExistingSession(c); user != nil {
		fmt.Printf("[Dashboard] User already authenticated: %s, redirecting to dashboard\n", user.Email)
		// Already logged in, redirect to dashboard
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/")
	}

	redirect := c.Query("redirect")

	// Check if this is the first user
	isFirstUser, err := h.isFirstUser(c.Request().Context())
	if err != nil {
		return h.renderError(c, "Failed to check system status", err)
	}

	signupData := pages.SignupPageData{
		Title:     "Sign Up",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Data: pages.SignupData{
			Redirect:    redirect,
			IsFirstUser: isFirstUser,
		},
	}

	page := pages.Signup(signupData)
	return h.render(c, page)
}

// HandleSignup processes the signup form
func (h *Handler) HandleSignup(c forge.Context) error {
	// Check if already authenticated
	if user := h.checkExistingSession(c); user != nil {
		fmt.Printf("[Dashboard] User already authenticated during signup attempt: %s, redirecting\n", user.Email)
		redirect := c.Request().FormValue("redirect")
		if redirect == "" {
			redirect = h.basePath + "/dashboard/"
		}
		return c.Redirect(http.StatusFound, redirect)
	}

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		fmt.Printf("[Dashboard] Signup form parse error: %v\n", err)
		return h.renderSignupError(c, "Invalid form data", c.Query("redirect"))
	}

	name := c.Request().FormValue("name")
	email := c.Request().FormValue("email")
	password := c.Request().FormValue("password")
	confirmPassword := c.Request().FormValue("password_confirm")
	redirect := c.Request().FormValue("redirect")
	csrfToken := c.Request().FormValue("csrf_token")

	fmt.Printf("[Dashboard] Signup attempt for email: %s\n", email)

	// Validate CSRF token
	if csrfToken == "" {
		fmt.Printf("[Dashboard] Signup error: Missing CSRF token\n")
		return h.renderSignupError(c, "Invalid CSRF token", redirect)
	}

	// Validate inputs
	if name == "" || email == "" || password == "" {
		fmt.Printf("[Dashboard] Signup error: Missing required fields\n")
		return h.renderSignupError(c, "All fields are required", redirect)
	}

	if password != confirmPassword {
		fmt.Printf("[Dashboard] Signup error: Passwords don't match\n")
		return h.renderSignupError(c, "Passwords do not match", redirect)
	}

	if len(password) < 8 {
		fmt.Printf("[Dashboard] Signup error: Password too short\n")
		return h.renderSignupError(c, "Password must be at least 8 characters", redirect)
	}

	// Get platform app to use as context for user creation
	ctx := c.Request().Context()
	platformApp, err := h.appService.GetPlatformApp(ctx)
	if err != nil {
		fmt.Printf("[Dashboard] Signup error: Failed to get platform app: %v\n", err)
		return h.renderSignupError(c, "System configuration error. Please contact administrator.", redirect)
	}

	ctx = contexts.SetAppID(ctx, platformApp.ID)

	// Create user in platform app context
	fmt.Printf("[Dashboard] Signup: Creating user with email: %s in platform app: %s\n", email, platformApp.ID.String())
	newUser, err := h.userSvc.Create(ctx, &user.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
		AppID:    platformApp.ID,
	})
	if err != nil {
		fmt.Printf("[Dashboard] Signup error: Failed to create user: %v\n", err)
		return h.renderSignupError(c, fmt.Sprintf("Failed to create account: %v", err), redirect)
	}

	// Add user as member of platform app
	// Note: CreateMember will automatically detect if this is the first user
	// and promote them to owner/superadmin as needed
	_, err = h.appService.CreateMember(ctx, &app.Member{
		ID:       xid.New(),
		AppID:    platformApp.ID,
		UserID:   newUser.ID,
		Role:     app.MemberRoleMember, // Will be auto-promoted to owner if first user
		Status:   app.MemberStatusActive,
		JoinedAt: time.Now(),
	})
	if err != nil {
		fmt.Printf("[Dashboard] Signup warning: Failed to add user to platform app: %v\n", err)
		// Continue anyway - user is created, just not added to app yet
	}

	fmt.Printf("[Dashboard] User created successfully: %s (%s)\n", newUser.Email, newUser.ID.String())
	fmt.Printf("[Dashboard] Signup: Password hash stored - length: %d, preview: %s...\n", len(newUser.PasswordHash), func() string {
		if len(newUser.PasswordHash) > 20 {
			return newUser.PasswordHash[:20]
		}
		return newUser.PasswordHash
	}())

	// Test password verification immediately after creation
	testPasswordCheck := crypto.CheckPassword(password, newUser.PasswordHash)
	fmt.Printf("[Dashboard] Signup: Immediate password verification test: %v\n", testPasswordCheck)
	if !testPasswordCheck {
		fmt.Printf("[Dashboard] ERROR: Password verification failed immediately after creation! This indicates a hashing issue.\n")
	}

	// Create session for the new user
	sess, err := h.sessionSvc.Create(c.Request().Context(), &session.CreateSessionRequest{
		UserID:    newUser.ID,
		IPAddress: c.Request().RemoteAddr,
		UserAgent: c.Request().UserAgent(),
		Remember:  false,
		AppID:     platformApp.ID,
	})
	if err != nil {
		fmt.Printf("[Dashboard] Signup error: Failed to create session: %v\n", err)
		return h.renderSignupError(c, "Account created but failed to log you in. Please try logging in.", redirect)
	}

	fmt.Printf("[Dashboard] Session created for user: %s\n", newUser.Email)

	// Set session cookie
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds()),
	}
	http.SetCookie(c.Response(), cookie)

	fmt.Printf("[Dashboard] Session cookie set for user: %s\n", newUser.Email)

	// Redirect to dashboard or specified redirect URL
	if redirect == "" {
		redirect = h.basePath + "/dashboard/"
	}

	fmt.Printf("[Dashboard] Redirecting user to: %s\n", redirect)
	return c.Redirect(http.StatusFound, redirect)
}

// renderSignupError renders the signup page with an error message
func (h *Handler) renderSignupError(c forge.Context, message string, redirect string) error {
	isFirstUser, _ := h.isFirstUser(c.Request().Context())

	signupData := pages.SignupPageData{
		Title:     "Sign Up",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     message,
		Data: pages.SignupData{
			Redirect:    redirect,
			IsFirstUser: isFirstUser,
		},
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	page := pages.Signup(signupData)
	return h.render(c, page)
}

// HandleLogout processes the logout request
func (h *Handler) HandleLogout(c forge.Context) error {
	fmt.Printf("[Dashboard] Logout requested\n")

	// Get session token from cookie
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err == nil && cookie != nil && cookie.Value != "" {
		// Revoke the session
		sess, err := h.sessionSvc.FindByToken(c.Request().Context(), cookie.Value)
		if err == nil && sess != nil {
			if err := h.sessionSvc.RevokeByID(c.Request().Context(), sess.ID); err != nil {
				fmt.Printf("[Dashboard] Failed to revoke session: %v\n", err)
			} else {
				fmt.Printf("[Dashboard] Session revoked: %s\n", sess.ID)
			}
		}
	}

	// Clear session cookie
	http.SetCookie(c.Response(), &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	})

	fmt.Printf("[Dashboard] Session cookie cleared, redirecting to login\n")

	// Redirect to login page
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
}

// isFirstUser checks if there are any users in the system
// This is a global check that bypasses organization context for the first system user
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
// TODO: Implement proper CSRF token generation and validation
func (h *Handler) generateCSRFToken() string {
	return xid.New().String()
}

// ServeStatic serves static assets (CSS, JS, images)
func (h *Handler) ServeStatic(c forge.Context) error {
	// Get the wildcard path from the route parameter
	// The route is registered as "/dashboard/static/*" so we get everything after /static/
	path := c.Param("*")

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid path"})
	}

	// Read file from embedded assets
	fullPath := filepath.Join("static", path)
	fmt.Println("fullPath", fullPath, "path", path)
	content, err := fs.ReadFile(h.assets, fullPath)
	if err != nil {
		fmt.Println("error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	// Set content type
	contentType := getContentType(filepath.Ext(path))
	c.SetHeader("Content-Type", contentType)
	c.SetHeader("Cache-Control", "public, max-age=31536000") // 1 year cache

	return c.String(http.StatusOK, string(content))
}

// Helper methods

// render renders a template with the given data
// render renders a gomponent node
func (h *Handler) render(c forge.Context, node g.Node) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return node.Render(c.Response())
}

// renderWithLayout renders content within the base layout
func (h *Handler) renderWithLayout(c forge.Context, pageData components.PageData, content g.Node) error {
	pageData.Year = time.Now().Year()
	pageData.EnabledPlugins = h.enabledPlugins
	page := components.BaseLayout(pageData, content)
	return h.render(c, page)
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

// calculateSessionStatistics computes statistics from session data
func (h *Handler) calculateSessionStatistics(sessions []*session.Session) (avgDuration string, sessionsToday int, sessionsThisWeek int) {
	if len(sessions) == 0 {
		return "N/A", 0, 0
	}

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfWeek := startOfToday.AddDate(0, 0, -7)

	var totalDuration time.Duration
	activeSessions := 0

	for _, sess := range sessions {
		// Count sessions created today
		if sess.CreatedAt.After(startOfToday) {
			sessionsToday++
		}

		// Count sessions created this week
		if sess.CreatedAt.After(startOfWeek) {
			sessionsThisWeek++
		}

		// Calculate average duration for non-expired sessions
		if now.Before(sess.ExpiresAt) {
			duration := now.Sub(sess.CreatedAt)
			totalDuration += duration
			activeSessions++
		}
	}

	// Calculate average duration
	if activeSessions > 0 {
		avgDurationVal := totalDuration / time.Duration(activeSessions)

		// Format duration in a human-readable way
		hours := int(avgDurationVal.Hours())
		minutes := int(avgDurationVal.Minutes()) % 60

		if hours > 0 {
			if minutes > 0 {
				avgDuration = fmt.Sprintf("%dh %dm", hours, minutes)
			} else {
				avgDuration = fmt.Sprintf("%dh", hours)
			}
		} else {
			avgDuration = fmt.Sprintf("%dm", minutes)
		}
	} else {
		avgDuration = "N/A"
	}

	return avgDuration, sessionsToday, sessionsThisWeek
}

// checkExistingSession checks if there's a valid session without middleware
// Returns user if authenticated, nil otherwise
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

	// Get user information
	user, err := h.userSvc.FindByID(c.Request().Context(), sess.UserID)
	if err != nil || user == nil {
		return nil
	}

	return user
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

// ==============================================================================
// Organization Management Handlers (User-Created Organizations)
// ==============================================================================
// These handlers manage user-created Organizations (Clerk-style workspaces)
// within an app, NOT platform-level Apps (which are managed by multiapp plugin).

// ServeOrganizations renders the organizations list page
func (h *Handler) ServeOrganizations(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get page number from query params
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	pageSize := 20

	// List organizations for current app
	filter := &organization.ListOrganizationsFilter{
		AppID: currentApp.ID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: pageSize,
		},
	}

	response, err := h.orgService.ListOrganizations(ctx, filter)
	if err != nil {
		return h.renderError(c, "Failed to load organizations", err)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           "Organizations - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "organizations",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Prepare organizations data
	orgsData := pages.OrganizationsData{
		Organizations: response.Data,
		Page:          response.Pagination.CurrentPage,
		TotalPages:    response.Pagination.TotalPages,
		Total:         int(response.Pagination.Total),
	}

	// Render page
	content := pages.OrganizationsPage(orgsData, appIDStr)
	return h.renderWithLayout(c, pageData, content)
}

// ServeOrganizationDetail renders the organization detail page
func (h *Handler) ServeOrganizationDetail(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get organization ID from URL
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.String(http.StatusBadRequest, "Organization ID is required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization ID")
	}

	// Get organization
	org, err := h.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return c.String(http.StatusNotFound, "Organization not found")
	}

	// Verify organization belongs to current app
	if org.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Organization does not belong to this app")
	}

	// Get member count by listing members
	memberFilter := &organization.ListMembersFilter{
		OrganizationID: orgID,
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1,
		},
	}
	memberResponse, err := h.orgService.ListMembers(ctx, memberFilter)
	memberCount := 0
	if err == nil && memberResponse.Pagination != nil {
		memberCount = int(memberResponse.Pagination.Total)
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           org.Name + " - Organizations - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "organizations",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Prepare organization detail data
	orgData := pages.OrganizationDetailData{
		Organization: org,
		MemberCount:  memberCount,
	}

	// Render page
	content := pages.OrganizationDetailPage(orgData, appIDStr)
	return h.renderWithLayout(c, pageData, content)
}

// ServeOrganizationCreate renders the organization creation form
func (h *Handler) ServeOrganizationCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           "Create Organization - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "organizations",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Render page
	content := pages.OrganizationCreatePage(appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleOrganizationCreate processes organization creation
func (h *Handler) HandleOrganizationCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Parse form
	name := c.FormValue("name")
	slug := c.FormValue("slug")

	if name == "" || slug == "" {
		return c.String(http.StatusBadRequest, "Name and slug are required")
	}

	// Create organization request
	req := &organization.CreateOrganizationRequest{
		Name: name,
		Slug: slug,
	}

	// TODO: Get environment ID from context or use default
	environmentID := xid.NilID()

	// Create organization
	org, err := h.orgService.CreateOrganization(ctx, req, user.ID, currentApp.ID, environmentID)
	if err != nil {
		return h.renderError(c, "Failed to create organization", err)
	}

	// Redirect to organization detail page
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/organizations/%s", h.basePath, appIDStr, org.ID.String()))
}

// ServeOrganizationEdit renders the organization edit form
func (h *Handler) ServeOrganizationEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get organization ID from URL
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.String(http.StatusBadRequest, "Organization ID is required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization ID")
	}

	// Get organization
	org, err := h.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return c.String(http.StatusNotFound, "Organization not found")
	}

	// Verify organization belongs to current app
	if org.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Organization does not belong to this app")
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           "Edit " + org.Name + " - Organizations - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "organizations",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Prepare organization edit data
	orgData := pages.OrganizationEditData{
		Organization: org,
	}

	// Render page
	content := pages.OrganizationEditPage(orgData, appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleOrganizationEdit processes organization update
func (h *Handler) HandleOrganizationEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get organization ID from URL
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.String(http.StatusBadRequest, "Organization ID is required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization ID")
	}

	// Get organization
	org, err := h.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return c.String(http.StatusNotFound, "Organization not found")
	}

	// Verify organization belongs to current app
	if org.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Organization does not belong to this app")
	}

	// Parse form
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Update organization request
	req := &organization.UpdateOrganizationRequest{
		Name: &name,
	}

	// Update organization
	_, err = h.orgService.UpdateOrganization(ctx, orgID, req)
	if err != nil {
		return h.renderError(c, "Failed to update organization", err)
	}

	// Redirect to organization detail page
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/organizations/%s", h.basePath, appIDStr, orgID.String()))
}

// HandleOrganizationDelete processes organization deletion
func (h *Handler) HandleOrganizationDelete(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject app ID from URL
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get organization ID from URL
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.String(http.StatusBadRequest, "Organization ID is required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization ID")
	}

	// Get organization
	org, err := h.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return c.String(http.StatusNotFound, "Organization not found")
	}

	// Verify organization belongs to current app
	if org.AppID != currentApp.ID {
		return c.String(http.StatusForbidden, "Organization does not belong to this app")
	}

	// Delete organization
	err = h.orgService.DeleteOrganization(ctx, orgID, user.ID)
	if err != nil {
		return h.renderError(c, "Failed to delete organization", err)
	}

	// Redirect to organizations list
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/organizations", h.basePath, appIDStr))
}

// ==============================================================================
// App Management Handlers (Platform Apps Management)
// ==============================================================================
// These handlers manage platform-level Apps (multi-tenancy), NOT user-created
// Organizations (workspaces). Create permission is based on multiapp plugin.

// // ServeAppsManagement renders the apps management list page (admin only)
// func (h *Handler) ServeAppsManagement(c forge.Context) error {
// 	user := h.getUserFromContext(c)
// 	if user == nil {
// 		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
// 	}
//
// 	// Extract and inject app ID from URL (for navigation context)
// 	ctx, currentApp, err := h.extractAndInjectAppID(c)
// 	if err != nil {
// 		return h.renderError(c, "Failed to load app context", err)
// 	}
//
// 	appIDStr := currentApp.ID.String()
//
// 	// Get user apps for switcher
// 	userApps, err := h.getUserApps(ctx, user.ID)
// 	if err != nil {
// 		userApps = []*app.App{}
// 	}
//
// 	// Get page number from query params
// 	page := 1
// 	if pageStr := c.Query("page"); pageStr != "" {
// 		fmt.Sscanf(pageStr, "%d", &page)
// 		if page < 1 {
// 			page = 1
// 		}
// 	}
//
// 	pageSize := 20
//
// 	// List all apps (admin can see all)
// 	appsFilter := &app.ListAppsFilter{
// 		PaginationParams: pagination.PaginationParams{
// 			Page:  page,
// 			Limit: pageSize,
// 		},
// 	}
// 	appsResponse, err := h.appService.ListApps(ctx, appsFilter)
// 	if err != nil {
// 		return h.renderError(c, "Failed to load apps", err)
// 	}
//
// 	pagedApps := appsResponse.Data
// 	total := int(appsResponse.Pagination.Total)
// 	totalPages := appsResponse.Pagination.TotalPages
//
// 	// Check if multiapp plugin is enabled (determines if user can create apps)
// 	canCreateApps := h.enabledPlugins["multiapp"]
//
// 	// Prepare page data
// 	pageData := components.PageData{
// 		Title:           "Apps Management - Dashboard",
// 		User:            user,
// 		CSRFToken:       h.getCSRFToken(c),
// 		ActivePage:      "apps-management",
// 		BasePath:        h.basePath,
// 		IsSaaSMode:      h.isMultiApp,
// 		CurrentApp:      currentApp,
// 		UserApps:        userApps,
// 		ShowAppSwitcher: len(userApps) > 0,
// 	}
//
// 	// Prepare apps management data
// 	appsData := pages.AppsManagementData{
// 		Apps:          pagedApps,
// 		Page:          page,
// 		TotalPages:    totalPages,
// 		Total:         total,
// 		CanCreateApps: canCreateApps,
// 	}
//
// 	// Render page
// 	content := pages.AppsManagementPage(appsData, appIDStr)
// 	return h.renderWithLayout(c, pageData, content)
// }

// ServeAppMgmtDetail renders the app management detail page
func (h *Handler) ServeAppMgmtDetail(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject current app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get target app ID from URL
	targetAppIDStr := c.Param("targetAppId")
	if targetAppIDStr == "" {
		return c.String(http.StatusBadRequest, "App ID is required")
	}

	targetAppID, err := xid.FromString(targetAppIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app ID")
	}

	// Get target app
	targetApp, err := h.appService.FindAppByID(ctx, targetAppID)
	if err != nil {
		return c.String(http.StatusNotFound, "App not found")
	}

	// Get member count
	memberCount, err := h.appService.CountMembers(ctx, targetAppID)
	if err != nil {
		memberCount = 0
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           targetApp.Name + " - Apps Management - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "apps-management",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Prepare app detail data
	appData := pages.AppManagementDetailData{
		App:         targetApp,
		MemberCount: memberCount,
	}

	// Render page
	content := pages.AppManagementDetailPage(appData, appIDStr)
	return h.renderWithLayout(c, pageData, content)
}

// ServeAppMgmtCreate renders the app creation form
func (h *Handler) ServeAppMgmtCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Check if multiapp plugin is enabled
	if !h.enabledPlugins["multiapp"] {
		return c.String(http.StatusForbidden, "App creation is only available when multiapp plugin is enabled")
	}

	// Extract and inject app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           "Create App - Apps Management - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "apps-management",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Render page
	content := pages.AppManagementCreatePage(appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleAppMgmtCreate processes app creation
func (h *Handler) HandleAppMgmtCreate(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Check if multiapp plugin is enabled
	if !h.enabledPlugins["multiapp"] {
		return c.String(http.StatusForbidden, "App creation is only available when multiapp plugin is enabled")
	}

	// Extract and inject app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Parse form
	name := c.FormValue("name")
	slug := c.FormValue("slug")

	if name == "" || slug == "" {
		return c.String(http.StatusBadRequest, "Name and slug are required")
	}

	// Create app request
	req := &app.CreateAppRequest{
		Name: name,
		Slug: slug,
	}

	// Create app
	newApp, err := h.appService.CreateApp(ctx, req)
	if err != nil {
		return h.renderError(c, "Failed to create app", err)
	}

	// Redirect to app detail page
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/apps-management/%s", h.basePath, appIDStr, newApp.ID.String()))
}

// ServeAppMgmtEdit renders the app edit form
func (h *Handler) ServeAppMgmtEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject current app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get user apps for switcher
	userApps, err := h.getUserApps(ctx, user.ID)
	if err != nil {
		userApps = []*app.App{}
	}

	// Get target app ID from URL
	targetAppIDStr := c.Param("targetAppId")
	if targetAppIDStr == "" {
		return c.String(http.StatusBadRequest, "App ID is required")
	}

	targetAppID, err := xid.FromString(targetAppIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app ID")
	}

	// Get target app
	targetApp, err := h.appService.FindAppByID(ctx, targetAppID)
	if err != nil {
		return c.String(http.StatusNotFound, "App not found")
	}

	// Prepare page data
	pageData := components.PageData{
		Title:           "Edit " + targetApp.Name + " - Apps Management - Dashboard",
		User:            user,
		CSRFToken:       h.getCSRFToken(c),
		ActivePage:      "apps-management",
		BasePath:        h.basePath,
		IsMultiApp:      h.isMultiApp,
		CurrentApp:      currentApp,
		UserApps:        userApps,
		ShowAppSwitcher: len(userApps) > 0,
	}

	// Prepare app edit data
	appData := pages.AppManagementEditData{
		App: targetApp,
	}

	// Render page
	content := pages.AppManagementEditPage(appData, appIDStr, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleAppMgmtEdit processes app update
func (h *Handler) HandleAppMgmtEdit(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject current app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get target app ID from URL
	targetAppIDStr := c.Param("targetAppId")
	if targetAppIDStr == "" {
		return c.String(http.StatusBadRequest, "App ID is required")
	}

	targetAppID, err := xid.FromString(targetAppIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app ID")
	}

	// Get target app
	targetApp, err := h.appService.FindAppByID(ctx, targetAppID)
	if err != nil {
		return c.String(http.StatusNotFound, "App not found")
	}

	// Parse form
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Update app request
	req := &app.UpdateAppRequest{
		Name: &name,
	}

	// Update app
	_, err = h.appService.UpdateApp(ctx, targetAppID, req)
	if err != nil {
		return h.renderError(c, "Failed to update app", err)
	}

	// Redirect to app detail page
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/apps-management/%s", h.basePath, appIDStr, targetApp.ID.String()))
}

// HandleAppMgmtDelete processes app deletion
func (h *Handler) HandleAppMgmtDelete(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Extract and inject current app ID from URL (for navigation context)
	ctx, currentApp, err := h.extractAndInjectAppID(c)
	if err != nil {
		return h.renderError(c, "Failed to load app context", err)
	}

	appIDStr := currentApp.ID.String()

	// Get target app ID from URL
	targetAppIDStr := c.Param("targetAppId")
	if targetAppIDStr == "" {
		return c.String(http.StatusBadRequest, "App ID is required")
	}

	targetAppID, err := xid.FromString(targetAppIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app ID")
	}

	// Get target app
	targetApp, err := h.appService.FindAppByID(ctx, targetAppID)
	if err != nil {
		return c.String(http.StatusNotFound, "App not found")
	}

	// Prevent deleting platform app
	if targetApp.IsPlatform {
		return c.String(http.StatusForbidden, "Cannot delete platform app")
	}

	// Delete app
	err = h.appService.DeleteApp(ctx, targetAppID)
	if err != nil {
		return h.renderError(c, "Failed to delete app", err)
	}

	// Redirect to apps management list
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/apps-management", h.basePath, appIDStr))
}
