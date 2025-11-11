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
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	mtorg "github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/types"
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
	orgService     *organization.Service
	mtOrgService   *mtorg.Service // Multitenancy organization service
	db             *bun.DB
	isSaaSMode     bool
	basePath       string
	enabledPlugins map[string]bool
	hookRegistry   *hooks.HookRegistry // For executing lifecycle hooks
}

// NewHandler creates a new dashboard handler
func NewHandler(
	assets embed.FS,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	auditSvc *audit.Service,
	rbacSvc *rbac.Service,
	apikeyService *apikey.Service,
	orgService *organization.Service,
	db *bun.DB,
	isSaaSMode bool,
	basePath string,
	enabledPlugins map[string]bool,
	hookRegistry *hooks.HookRegistry,
) *Handler {
	h := &Handler{
		assets:         assets,
		userSvc:        userSvc,
		sessionSvc:     sessionSvc,
		auditSvc:       auditSvc,
		rbacSvc:        rbacSvc,
		apikeyService:  apikeyService,
		orgService:     orgService,
		db:             db,
		isSaaSMode:     isSaaSMode,
		basePath:       basePath,
		enabledPlugins: enabledPlugins,
		hookRegistry:   hookRegistry,
	}

	// Try to get multitenancy organization service if available
	// This will be set by the plugin during initialization
	return h
}

// SetMultitenancyOrgService sets the multitenancy organization service
func (h *Handler) SetMultitenancyOrgService(svc *mtorg.Service) {
	h.mtOrgService = svc
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

	// Get dashboard stats
	stats, err := h.getDashboardStats(c.Request().Context())
	if err != nil {
		return h.renderError(c, "Failed to load dashboard statistics", err)
	}

	pageData := components.PageData{
		Title:      "Dashboard",
		ActivePage: "dashboard",
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
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
	}

	content := pages.DashboardPage(pageStats, h.basePath)
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

// ServeUsers serves the users list page
func (h *Handler) ServeUsers(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
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

	// List or search users
	var users []*user.User
	var total int
	var err error

	if query != "" {
		// Search users
		users, total, err = h.userSvc.Search(c.Request().Context(), query, types.PaginationOptions{
			Page:     page,
			PageSize: pageSize,
		})
	} else {
		// List all users
		users, total, err = h.userSvc.List(c.Request().Context(), types.PaginationOptions{
			Page:     page,
			PageSize: pageSize,
		})
	}

	if err != nil {
		return h.renderError(c, "Failed to load users", err)
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize

	pageData := components.PageData{
		Title:      "Users",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "users",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
	}

	usersData := pages.UsersData{
		Users:      users,
		Query:      query,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
	}

	content := pages.UsersPage(usersData, h.basePath)
	return h.renderWithLayout(c, pageData, content)
}

// ServeUserDetail serves a single user detail page
func (h *Handler) ServeUserDetail(c forge.Context) error {
	user := h.getUserFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
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

	// Get user details
	targetUser, err := h.userSvc.FindByID(c.Request().Context(), userID)
	if err != nil {
		return h.renderError(c, "User not found", err)
	}

	// Get active sessions for this user (limit to 10 for detail view)
	allSessions, err := h.sessionSvc.ListByUser(c.Request().Context(), userID, 10, 0)
	if err != nil {
		// Log error but don't fail the page
		// Just show empty sessions
		allSessions = []*session.Session{}
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

	pageData := components.PageData{
		Title:      fmt.Sprintf("User: %s", targetUser.Email),
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "users",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
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

	// Get user ID from path
	userID := c.Param("id")
	id, err := xid.FromString(userID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Fetch user details
	targetUser, err := h.userSvc.FindByID(c.Request().Context(), id)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	pageData := components.PageData{
		Title:      fmt.Sprintf("Edit User: %s", targetUser.Email),
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "users",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
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

	// Get user ID from path
	userID := c.Param("id")
	id, err := xid.FromString(userID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	// Fetch user details
	targetUser, err := h.userSvc.FindByID(c.Request().Context(), id)
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

	updatedUser, err := h.userSvc.Update(c.Request().Context(), targetUser, updateReq)
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
	if err := h.userSvc.Delete(c.Request().Context(), id); err != nil {
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

	// Parse search query
	query := c.Query("q")

	// Fetch all active sessions
	allSessions, err := h.sessionSvc.ListAll(c.Request().Context(), 1000, 0)
	if err != nil {
		fmt.Printf("[Dashboard] Failed to list sessions: %v\n", err)
		allSessions = []*session.Session{} // Show empty state on error
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

	pageData := components.PageData{
		Title:      "Sessions",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "sessions",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
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

	sessionsPageData := pages.SessionsPageData{
		Sessions:  sessionData,
		Query:     query,
		BasePath:  h.basePath,
		CSRFToken: h.getCSRFToken(c),
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

	// Get session ID from path
	sessionID := c.Param("id")
	id, err := xid.FromString(sessionID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid session ID")
	}

	// Revoke session
	if err := h.sessionSvc.RevokeByID(c.Request().Context(), id); err != nil {
		fmt.Printf("[Dashboard] Failed to revoke session: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to revoke session")
	}

	fmt.Printf("[Dashboard] Session %s revoked by admin %s\n", sessionID, currentUser.ID)

	// Redirect back to sessions list with success message
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/sessions?success=revoked")
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

	// Get active tab from query parameter
	activeTab := c.Query("tab")
	if activeTab == "" {
		activeTab = "general"
	}

	pageData := components.PageData{
		Title:      "Settings",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "settings",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
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
		IsSaaSMode:    h.isSaaSMode,
		CanCreateKeys: true,
		CSRFToken:     h.getCSRFToken(c),
	}

	// If SaaS mode, fetch user's organizations
	if h.isSaaSMode && h.orgService != nil {
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
		IsSaaSMode:            h.isSaaSMode,
		BasePath:              h.basePath,
		CSRFToken:             h.getCSRFToken(c),
		EnabledPlugins:        h.enabledPlugins,
	}

	content := pages.SettingsPage(settingsData)
	return h.renderWithLayout(c, pageData, content)
}

// HandleRevokeUserSessions revokes all sessions for a specific user
func (h *Handler) HandleRevokeUserSessions(c forge.Context) error {
	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
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

	// Get all sessions for this user
	sessions, err := h.sessionSvc.ListByUser(c.Request().Context(), userID, 1000, 0)
	if err != nil {
		fmt.Printf("[Dashboard] Failed to list user sessions: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to list user sessions")
	}

	// Revoke each session
	revokedCount := 0
	for _, sess := range sessions {
		if err := h.sessionSvc.RevokeByID(c.Request().Context(), sess.ID); err != nil {
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

	// Find user by email
	// Platform users (first user) can login without organization context
	// The multitenancy decorator will handle this automatically
	ctx := c.Request().Context()
	user, err := h.userSvc.FindByEmail(ctx, email)
	fmt.Printf("[Dashboard] Login: Email: %s\n", email)
	fmt.Printf("[Dashboard] Login: User: %+v\n", user)
	fmt.Printf("[Dashboard] Login: Error: %+v\n", err)

	// If we get "organization context required" error, we're in multi-tenant mode
	// Try with the first organization from database
	if err != nil && err.Error() == "organization context required" && h.isSaaSMode {
		fmt.Printf("[Dashboard] Multi-tenant mode detected, querying for user's organization\n")

		// Query organizations table directly to get the first/platform organization
		// This bypasses the need for the List method
		var orgID string
		err = h.db.NewSelect().
			Table("organizations").
			Column("id").
			Order("created_at ASC").
			Limit(1).
			Scan(ctx, &orgID)

		if err == nil && orgID != "" {
			// Set organization context and retry
			ctx = context.WithValue(ctx, interfaces.OrganizationContextKey, orgID)
			fmt.Printf("[Dashboard] Retrying login with organization context: %s\n", orgID)
			user, err = h.userSvc.FindByEmail(ctx, email)
		}
	}

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

	// Check if user is a member of the platform organization
	// Dashboard access is restricted to platform organization members only
	if h.mtOrgService != nil {
		platformOrg, err := h.mtOrgService.GetOrganizationBySlug(ctx, "platform")
		if err != nil {
			fmt.Printf("[Dashboard] Login error: Could not find platform organization: %v\n", err)
			return h.renderLoginError(c, "Access denied: Platform organization not found", redirect)
		}

		// Check if user is a member of the platform organization
		isMember, err := h.mtOrgService.IsMember(ctx, platformOrg.ID, user.ID)
		if err != nil {
			fmt.Printf("[Dashboard] Login error: Failed to check organization membership: %v\n", err)
			return h.renderLoginError(c, "Access denied: Unable to verify membership", redirect)
		}

		if !isMember {
			fmt.Printf("[Dashboard] Login error: User %s is not a member of platform organization\n", user.Email)
			return h.renderLoginError(c, "Access denied: Only platform users can access this dashboard", redirect)
		}

		fmt.Printf("[Dashboard] Login: User %s verified as platform organization member\n", user.Email)
	}

	// Note: Role checking is now handled by the RequireAdmin middleware
	// The middleware will check if the user has the required permissions
	// using the fast PermissionChecker after successful authentication

	// Create session
	sess, err := h.sessionSvc.Create(c.Request().Context(), &session.CreateSessionRequest{
		UserID:    user.ID,
		IPAddress: c.Request().RemoteAddr,
		UserAgent: c.Request().UserAgent(),
		Remember:  false,
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

	// Check if this is the first user
	isFirstUser, err := h.isFirstUser(c.Request().Context())
	if err != nil {
		fmt.Printf("[Dashboard] Signup error: Failed to check first user status: %v\n", err)
		return h.renderSignupError(c, "Failed to check system status", redirect)
	}

	fmt.Printf("[Dashboard] Is first user: %v\n", isFirstUser)

	// Create user
	fmt.Printf("[Dashboard] Signup: Creating user with email: %s, password length: %d\n", email, len(password))
	newUser, err := h.userSvc.Create(c.Request().Context(), &user.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		fmt.Printf("[Dashboard] Signup error: Failed to create user: %v\n", err)
		return h.renderSignupError(c, fmt.Sprintf("Failed to create account: %v", err), redirect)
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

	// If this is the first user, auto-verify email, assign admin/owner role, and create organization
	if isFirstUser {
		// Auto-verify the first user's email
		emailVerified := true
		updatedUser, err := h.userSvc.Update(c.Request().Context(), newUser, &user.UpdateUserRequest{
			EmailVerified: &emailVerified,
		})
		if err != nil {
			fmt.Printf("[Dashboard] Warning: Failed to auto-verify first user email: %v\n", err)
		} else {
			// Reload user from database to ensure we have the latest data including password hash
			reloadedUser, err := h.userSvc.FindByID(c.Request().Context(), updatedUser.ID)
			if err != nil {
				fmt.Printf("[Dashboard] Warning: Failed to reload user after email verification: %v\n", err)
				newUser = updatedUser // Fallback to updated user
			} else {
				newUser = reloadedUser // Use reloaded user with fresh data
				fmt.Printf("[Dashboard] ✅ First user email auto-verified: %s\n", newUser.Email)
				fmt.Printf("[Dashboard] Reloaded user - password hash length: %d\n", len(newUser.PasswordHash))
			}
		}

		// Execute user creation hooks - this is where the multitenancy plugin creates the platform org
		ctx := c.Request().Context()
		if h.hookRegistry != nil {
			fmt.Printf("[Dashboard] Executing after-user-create hooks for first user: %s\n", newUser.Email)
			if err := h.hookRegistry.ExecuteAfterUserCreate(ctx, newUser); err != nil {
				fmt.Printf("[Dashboard] ERROR: Hook execution failed: %v\n", err)
			} else {
				fmt.Printf("[Dashboard] ✅ Hooks executed successfully\n")
			}
		}

		// Now get the platform organization ID
		var orgID xid.ID
		if h.mtOrgService != nil {
			// SaaS mode: use multitenancy service
			platformOrg, err := h.mtOrgService.GetOrganizationBySlug(ctx, "platform")
			if err != nil {
				fmt.Printf("[Dashboard] Warning: Could not find platform organization after hook execution: %v\n", err)
				// Try to get any organization as fallback
				orgs, err := h.mtOrgService.ListOrganizations(ctx, 1, 0)
				if err == nil && len(orgs) > 0 {
					orgID = orgs[0].ID
					fmt.Printf("[Dashboard] Using first available organization for role assignment: %s\n", orgID.String())
				}
			} else {
				orgID = platformOrg.ID
				fmt.Printf("[Dashboard] ✅ Platform organization found via multitenancy service: %s (ID: %s)\n", platformOrg.Name, orgID.String())
			}
		} else {
			// Standalone mode: query database directly for platform organization
			fmt.Printf("[Dashboard] Standalone mode detected, querying database directly for platform organization...\n")
			var platformOrg schema.Organization
			err := h.db.NewSelect().
				Model(&platformOrg).
				Where("is_platform = ?", true).
				Limit(1).
				Scan(ctx)

			if err != nil {
				fmt.Printf("[Dashboard] ERROR: Failed to find platform organization in database: %v\n", err)
			} else {
				orgID = platformOrg.ID
				fmt.Printf("[Dashboard] ✅ Platform organization found in database: %s (ID: %s)\n", platformOrg.Name, orgID.String())
			}
		}

		// Assign SUPERADMIN role to first user in the database
		// This makes them the platform owner with full system access
		userRoleRepo := repository.NewUserRoleRepository(h.db)
		roleRepo := repository.NewRoleRepository(h.db)

		if orgID.IsNil() {
			fmt.Printf("[Dashboard] ERROR: No organization ID available for role assignment. Cannot assign superadmin role.\n")
			fmt.Printf("[Dashboard] This likely means no platform organization exists in the system.\n")
		} else {
			// Assign superadmin role to user within platform organization context
			if err := EnsureFirstUserIsSuperAdmin(ctx, newUser.ID, orgID, userRoleRepo, roleRepo); err != nil {
				fmt.Printf("[Dashboard] ERROR: Failed to assign superadmin role to first user: %v\n", err)
			} else {
				fmt.Printf("[Dashboard] ✅ Superadmin role assigned to first user: %s (org: %s)\n", newUser.Email, orgID.String())
			}

			// ADD: Ensure user is added to members table for organizational access
			// This is critical for multitenancy decorators to recognize the user
			fmt.Printf("[Dashboard] Adding first user to members table...\n")
			if h.mtOrgService != nil {
				// SaaS mode: use multitenancy service
				_, err := h.mtOrgService.AddMember(ctx, orgID, newUser.ID, "owner")
				if err != nil && err.Error() != "user is already a member of this organization" {
					fmt.Printf("[Dashboard] ERROR: Failed to add first user to members table: %v\n", err)
				} else {
					fmt.Printf("[Dashboard] ✅ First user added to members table as owner\n")
				}
			} else {
				// Standalone mode: insert member record directly via database
				now := time.Now()
				member := &schema.Member{
					ID:             xid.New(),
					OrganizationID: orgID,
					UserID:         newUser.ID,
					Role:           "owner",
					Status:         "active",
					JoinedAt:       now,
				}
				// Set auditable fields on the embedded struct
				member.AuditableModel.ID = member.ID
				member.AuditableModel.CreatedBy = newUser.ID
				member.AuditableModel.UpdatedBy = newUser.ID
				member.AuditableModel.CreatedAt = now
				member.AuditableModel.UpdatedAt = now
				member.AuditableModel.Version = 1

				_, err := h.db.NewInsert().Model(member).Exec(ctx)
				if err != nil {
					fmt.Printf("[Dashboard] ERROR: Failed to insert member record: %v\n", err)
				} else {
					fmt.Printf("[Dashboard] ✅ First user member record created\n")
				}
			}

			// Verify both tables have records
			fmt.Printf("[Dashboard] Verifying first user setup:\n")

			// Check user_roles
			roleCount, err := h.db.NewSelect().Table("user_roles").Where("user_id = ?", newUser.ID).Count(ctx)
			if err != nil {
				fmt.Printf("[Dashboard]   - ERROR checking user_roles: %v\n", err)
			} else {
				fmt.Printf("[Dashboard]   - user_roles records: %d\n", roleCount)
			}

			// Check members
			memberCount, err := h.db.NewSelect().Table("members").Where("user_id = ?", newUser.ID).Count(ctx)
			if err != nil {
				fmt.Printf("[Dashboard]   - ERROR checking members: %v\n", err)
			} else {
				fmt.Printf("[Dashboard]   - members records: %d\n", memberCount)
			}
		}

		// Role-based RBAC policies are set up via SetupDefaultPolicies()
		// No need for user-specific policies - the superadmin role grants all permissions
		fmt.Printf("[Dashboard] ✨ First user created (platform owner): %s (%s) - Email verified ✅ & Superadmin role assigned ✅\n", newUser.Email, newUser.ID.String())
	}

	// Create session for the new user
	sess, err := h.sessionSvc.Create(c.Request().Context(), &session.CreateSessionRequest{
		UserID:    newUser.ID,
		IPAddress: c.Request().RemoteAddr,
		UserAgent: c.Request().UserAgent(),
		Remember:  false,
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
	// In SaaS mode, the user service is decorated with multi-tenant logic
	// which requires organization context. However, for the first user check,
	// we need to check globally if ANY users exist in the system at all.
	//
	// We can detect this by attempting the List call. If it fails with
	// "organization context required", we know we're in multi-tenant mode
	// and no organization exists yet (hence it's the first user).

	users, total, err := h.userSvc.List(ctx, types.PaginationOptions{
		Page:     1,
		PageSize: 1,
	})

	// If we get "organization context required" error, it means:
	// 1. We're in SaaS mode (multi-tenant decorator is active)
	// 2. There's no organization in context (because this is signup)
	// 3. This must be the first user attempting to sign up
	if err != nil {
		if err.Error() == "organization context required" {
			// This is the first user - no organizations exist yet
			return true, nil
		}
		// Some other error
		return false, err
	}

	return total == 0 || len(users) == 0, nil
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
		IsSaaSMode:     h.isSaaSMode,
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

// Organization Management Handlers (only available in SaaS mode)

// ServeOrganizations serves the organizations list page
func (h *Handler) ServeOrganizations(c forge.Context) error {
	// Organization management requires multitenancy service
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
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

	offset := (page - 1) * pageSize

	// List organizations
	orgs, err := h.mtOrgService.ListOrganizations(c.Request().Context(), pageSize, offset)
	if err != nil {
		return h.renderError(c, "Failed to load organizations", err)
	}

	// Get total count (simplified - in production, use a Count method)
	total := len(orgs) // This is approximate, should use proper count
	totalPages := (total + pageSize - 1) / pageSize

	pageData := components.PageData{
		Title:      "Organizations",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "organizations",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
	}

	orgsData := pages.OrganizationsData{
		Organizations: orgs,
		Page:          page,
		TotalPages:    totalPages,
		Total:         total,
	}

	content := pages.OrganizationsPage(orgsData, h.basePath)
	return h.renderWithLayout(c, pageData, content)
}

// ServeOrganizationDetail serves a single organization detail page
func (h *Handler) ServeOrganizationDetail(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return h.renderError(c, "Organization ID is required", fmt.Errorf("missing organization ID"))
	}

	xidOrgID, err := xid.FromString(orgID)
	if err != nil {
		return h.renderError(c, "Invalid organization ID", err)
	}

	// Get organization
	org, err := h.mtOrgService.GetOrganization(c.Request().Context(), xidOrgID)
	if err != nil {
		return h.renderError(c, "Organization not found", err)
	}

	// Get organization members
	members, err := h.mtOrgService.ListMembers(c.Request().Context(), xidOrgID, 100, 0)
	if err != nil {
		fmt.Printf("[Dashboard] Warning: Failed to load members: %v\n", err)
		members = []*mtorg.Member{}
	}

	pageData := components.PageData{
		Title:      "Organization: " + org.Name,
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "organizations",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
	}

	orgDetailData := pages.OrganizationDetailData{
		Organization: org,
		Members:      members,
	}

	content := pages.OrganizationDetailPage(orgDetailData, h.basePath)
	return h.renderWithLayout(c, pageData, content)
}

// ServeOrganizationCreate serves the organization creation form
func (h *Handler) ServeOrganizationCreate(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	pageData := components.PageData{
		Title:      "Create Organization",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "organizations",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
	}

	content := pages.OrganizationCreatePage(h.basePath, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleOrganizationCreate processes the organization creation form
func (h *Handler) HandleOrganizationCreate(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return h.renderError(c, "Invalid form data", err)
	}

	name := c.Request().FormValue("name")
	slug := c.Request().FormValue("slug")
	redirect := c.Request().FormValue("redirect")
	csrfToken := c.Request().FormValue("csrf_token")

	// Validate CSRF token
	if csrfToken == "" {
		return h.renderError(c, "Invalid CSRF token", fmt.Errorf("missing CSRF token"))
	}

	// Validate inputs
	if name == "" || slug == "" {
		return h.renderError(c, "Name and slug are required", fmt.Errorf("missing required fields"))
	}

	// Create organization
	createReq := &mtorg.CreateOrganizationRequest{
		Name: name,
		Slug: slug,
	}

	org, err := h.mtOrgService.CreateOrganization(c.Request().Context(), createReq, currentUser.ID)
	if err != nil {
		return h.renderError(c, fmt.Sprintf("Failed to create organization: %v", err), err)
	}

	fmt.Printf("[Dashboard] Organization created: %s (%s)\n", org.Name, org.ID)

	// Redirect to organization detail page
	if redirect == "" {
		redirect = h.basePath + "/dashboard/organizations/" + org.ID.String()
	}

	return c.Redirect(http.StatusFound, redirect)
}

// ServeOrganizationEdit serves the organization edit form
func (h *Handler) ServeOrganizationEdit(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return h.renderError(c, "Organization ID is required", fmt.Errorf("missing organization ID"))
	}

	xidOrgID, err := xid.FromString(orgID)
	if err != nil {
		return h.renderError(c, "Invalid organization ID", err)
	}

	// Get organization
	org, err := h.mtOrgService.GetOrganization(c.Request().Context(), xidOrgID)
	if err != nil {
		return h.renderError(c, "Organization not found", err)
	}

	pageData := components.PageData{
		Title:      "Edit Organization: " + org.Name,
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "organizations",
		BasePath:   h.basePath,
		IsSaaSMode: h.isSaaSMode,
	}

	orgEditData := pages.OrganizationEditData{
		Organization: org,
	}

	content := pages.OrganizationEditPage(orgEditData, h.basePath, h.getCSRFToken(c))
	return h.renderWithLayout(c, pageData, content)
}

// HandleOrganizationEdit processes the organization edit form
func (h *Handler) HandleOrganizationEdit(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return h.renderError(c, "Organization ID is required", fmt.Errorf("missing organization ID"))
	}

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return h.renderError(c, "Invalid form data", err)
	}

	name := c.Request().FormValue("name")
	redirect := c.Request().FormValue("redirect")
	csrfToken := c.Request().FormValue("csrf_token")

	// Validate CSRF token
	if csrfToken == "" {
		return h.renderError(c, "Invalid CSRF token", fmt.Errorf("missing CSRF token"))
	}

	// Update organization
	updateReq := &mtorg.UpdateOrganizationRequest{}
	if name != "" {
		updateReq.Name = &name
	}

	xidOrgID, err := xid.FromString(orgID)
	if err != nil {
		return h.renderError(c, "Invalid organization ID", err)
	}

	org, err := h.mtOrgService.UpdateOrganization(c.Request().Context(), xidOrgID, updateReq)
	if err != nil {
		return h.renderError(c, fmt.Sprintf("Failed to update organization: %v", err), err)
	}

	fmt.Printf("[Dashboard] Organization updated: %s (%s)\n", org.Name, org.ID)

	// Redirect to organization detail page
	if redirect == "" {
		redirect = h.basePath + "/dashboard/organizations/" + orgID
	}

	return c.Redirect(http.StatusFound, redirect)
}

// HandleOrganizationDelete processes organization deletion
func (h *Handler) HandleOrganizationDelete(c forge.Context) error {
	if h.mtOrgService == nil {
		return h.renderError(c, "Organization management is only available in SaaS mode", fmt.Errorf("multitenancy not enabled"))
	}

	currentUser := h.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/login")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return h.renderError(c, "Organization ID is required", fmt.Errorf("missing organization ID"))
	}

	xidOrgID, err := xid.FromString(orgID)
	if err != nil {
		return h.renderError(c, "Invalid organization ID", err)
	}

	// Delete organization
	err = h.mtOrgService.DeleteOrganization(c.Request().Context(), xidOrgID)
	if err != nil {
		return h.renderError(c, fmt.Sprintf("Failed to delete organization: %v", err), err)
	}

	fmt.Printf("[Dashboard] Organization deleted: %s\n", orgID)

	// Redirect to organizations list
	return c.Redirect(http.StatusFound, h.basePath+"/dashboard/organizations")
}
