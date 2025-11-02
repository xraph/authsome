package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// Handler handles dashboard HTTP requests
type Handler struct {
	templates      *template.Template
	assets         embed.FS
	userSvc        *user.Service
	sessionSvc     *session.Service
	auditSvc       *audit.Service
	rbacSvc        *rbac.Service
	basePath       string
	enabledPlugins map[string]bool
}

// NewHandler creates a new dashboard handler
func NewHandler(
	templates *template.Template,
	assets embed.FS,
	userSvc *user.Service,
	sessionSvc *session.Service,
	auditSvc *audit.Service,
	rbacSvc *rbac.Service,
	basePath string,
	enabledPlugins map[string]bool,
) *Handler {
	return &Handler{
		templates:      templates,
		assets:         assets,
		userSvc:        userSvc,
		sessionSvc:     sessionSvc,
		auditSvc:       auditSvc,
		rbacSvc:        rbacSvc,
		basePath:       basePath,
		enabledPlugins: enabledPlugins,
	}
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

	data := PageData{
		Title:      "Dashboard",
		ActivePage: "dashboard",
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		BasePath:   h.basePath,
		Data:       stats,
	}

	return h.render(c, "dashboard.html", data)
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

	data := PageData{
		Title:      "Users",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "users",
		BasePath:   h.basePath,
		Data: map[string]interface{}{
			"Users":      users,
			"Total":      total,
			"Page":       page,
			"PageSize":   pageSize,
			"TotalPages": totalPages,
			"Query":      query,
		},
	}

	return h.render(c, "users.html", data)
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

	data := PageData{
		Title:      fmt.Sprintf("User: %s", targetUser.Email),
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "user_detail",
		BasePath:   h.basePath,
		Data:       targetUser,
	}

	return h.render(c, "user_detail.html", data)
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

	editData := map[string]interface{}{
		"UserID":        targetUser.ID.String(),
		"Name":          targetUser.Name,
		"Email":         targetUser.Email,
		"Username":      targetUser.Username,
		"EmailVerified": targetUser.EmailVerified,
	}

	data := PageData{
		Title:      fmt.Sprintf("Edit User: %s", targetUser.Email),
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "user_edit",
		BasePath:   h.basePath,
		Data:       editData,
	}

	return h.render(c, "user_edit.html", data)
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

	data := PageData{
		Title:      "Sessions",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "sessions",
		BasePath:   h.basePath,
		Data: map[string]interface{}{
			"Sessions": sessions,
			"Query":    query,
		},
	}

	return h.render(c, "sessions.html", data)
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
	data := PageData{
		Title:      "Page Not Found",
		BasePath:   h.basePath,
		ActivePage: "",
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusNotFound)
	return h.templates.ExecuteTemplate(c.Response(), "404.html", data)
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

	// Mock data for API keys and webhooks
	// TODO: Replace with actual data from services
	apiKeys := []map[string]interface{}{}
	webhooks := []map[string]interface{}{}

	data := PageData{
		Title:      "Settings",
		User:       currentUser,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "settings",
		BasePath:   h.basePath,
		Data: map[string]interface{}{
			"ActiveTab": activeTab,
			"APIKeys":   apiKeys,
			"Webhooks":  webhooks,
		},
	}

	return h.render(c, "settings.html", data)
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
	// Check if already authenticated
	user := h.getUserFromContext(c)
	if user != nil {
		// Already logged in, redirect to dashboard
		redirect := c.Query("redirect")
		if redirect == "" {
			redirect = h.basePath + "/dashboard/"
		}
		return c.Redirect(http.StatusFound, redirect)
	}

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

	data := PageData{
		Title:     "Login",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     errorMessage,
		Data: map[string]interface{}{
			"Redirect":    redirect,
			"ShowSignup":  true,
			"IsFirstUser": isFirstUser,
		},
	}

	return h.render(c, "login.html", data)
}

// HandleLogin processes the login form
func (h *Handler) HandleLogin(c forge.Context) error {
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
	user, err := h.userSvc.FindByEmail(c.Request().Context(), email)
	if err != nil || user == nil {
		return h.renderLoginError(c, "Invalid email or password", redirect)
	}

	// Verify password
	if !crypto.CheckPassword(password, user.PasswordHash) {
		return h.renderLoginError(c, "Invalid email or password", redirect)
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
	data := PageData{
		Title:     "Login",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     message,
		Data: map[string]interface{}{
			"Redirect": redirect,
		},
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Response(), "login.html", data)
}

// ServeSignup serves the signup page
func (h *Handler) ServeSignup(c forge.Context) error {
	// Check if already authenticated
	user := h.getUserFromContext(c)
	if user != nil {
		// Already logged in, redirect to dashboard
		return c.Redirect(http.StatusFound, h.basePath+"/dashboard/")
	}

	redirect := c.Query("redirect")

	// Check if this is the first user
	isFirstUser, err := h.isFirstUser(c.Request().Context())
	if err != nil {
		return h.renderError(c, "Failed to check system status", err)
	}

	data := PageData{
		Title:     "Sign Up",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Data: map[string]interface{}{
			"Redirect":    redirect,
			"IsFirstUser": isFirstUser,
		},
	}

	return h.render(c, "signup.html", data)
}

// HandleSignup processes the signup form
func (h *Handler) HandleSignup(c forge.Context) error {
	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		fmt.Printf("[Dashboard] Signup form parse error: %v\n", err)
		return h.renderSignupError(c, "Invalid form data", c.Query("redirect"))
	}

	name := c.Request().FormValue("name")
	email := c.Request().FormValue("email")
	password := c.Request().FormValue("password")
	confirmPassword := c.Request().FormValue("confirm_password")
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

	// If this is the first user, assign admin/owner role
	if isFirstUser {
		// Assign admin role to first user using RBAC service
		// Create a policy that grants full dashboard access to this user
		adminPolicy := &rbac.Policy{
			Subject:  newUser.ID.String(),
			Actions:  []string{"*"}, // All actions
			Resource: "dashboard",
		}
		h.rbacSvc.AddPolicy(adminPolicy)

		// Also grant system owner access
		ownerPolicy := &rbac.Policy{
			Subject:  newUser.ID.String(),
			Actions:  []string{"*"}, // All actions
			Resource: "system",
		}
		h.rbacSvc.AddPolicy(ownerPolicy)

		fmt.Printf("[Dashboard] ✨ First user created (system owner): %s (%s) - Admin roles assigned\n", newUser.Email, newUser.ID.String())
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

	data := PageData{
		Title:     "Sign Up",
		CSRFToken: h.generateCSRFToken(),
		BasePath:  h.basePath,
		Error:     message,
		Data: map[string]interface{}{
			"Redirect":    redirect,
			"IsFirstUser": isFirstUser,
		},
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Response(), "signup.html", data)
}

// isFirstUser checks if there are any users in the system
func (h *Handler) isFirstUser(ctx context.Context) (bool, error) {
	// List users with limit 1 to check if any exist
	users, total, err := h.userSvc.List(ctx, types.PaginationOptions{
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
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
func (h *Handler) render(c forge.Context, templateName string, data PageData) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")

	// Always set the current year
	if data.Year == 0 {
		data.Year = time.Now().Year()
	}

	// Always pass enabled plugins to templates
	if data.EnabledPlugins == nil {
		data.EnabledPlugins = h.enabledPlugins
	}

	err := h.templates.ExecuteTemplate(c.Response(), templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	return nil
}

// renderError renders an error page
func (h *Handler) renderError(c forge.Context, message string, err error) error {
	user := h.getUserFromContext(c)

	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	data := PageData{
		Title:      "Error",
		User:       user,
		CSRFToken:  h.getCSRFToken(c),
		ActivePage: "",
		BasePath:   h.basePath,
		Error:      errorMsg,
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Response(), "error.html", data)
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
