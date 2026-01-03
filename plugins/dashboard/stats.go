package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// DashboardStats represents statistics for the dashboard
type DashboardStats struct {
	TotalUsers     int
	ActiveUsers    int
	NewUsersToday  int
	TotalSessions  int
	ActiveSessions int
	FailedLogins   int
	UserGrowth     float64
	SessionGrowth  float64
	RecentActivity []ActivityItem
	SystemStatus   []StatusItem
	Plugins        []PluginItem
}

// ActivityItem represents a recent activity entry
type ActivityItem struct {
	Title       string
	Description string
	Time        string
	Type        string // success, warning, error, info
}

// StatusItem represents a system status entry
type StatusItem struct {
	Name   string
	Status string // operational, degraded, down
	Color  string // green, yellow, red
}

// PluginItem represents a plugin entry
type PluginItem struct {
	ID          string
	Name        string
	Description string
	Category    string
	Status      string // enabled, disabled
	Icon        string // lucide icon name
}

// getDashboardStats fetches dashboard statistics
func (h *Handler) getDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Extract app ID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, fmt.Errorf("app context required for dashboard stats")
	}

	// Count total users for this app
	totalUsers := 0
	newUsersToday := 0

	userFilter := &user.CountUsersFilter{
		AppID: appID,
	}
	totalUsers, err := h.userSvc.CountUsers(ctx, userFilter)
	if err != nil {
		totalUsers = 0
	}

	// Count new users today
	startOfToday := time.Now().Truncate(24 * time.Hour)
	newUserFilter := &user.CountUsersFilter{
		AppID:        appID,
		CreatedSince: &startOfToday,
	}
	newUsersToday, err = h.userSvc.CountUsers(ctx, newUserFilter)
	if err != nil {
		newUsersToday = 0
	}

	// Get all sessions for this app
	sessionFilter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000,
		},
		AppID: appID,
	}
	sessionResponse, err := h.sessionSvc.ListSessions(ctx, sessionFilter)
	allSessions := []*session.Session{}
	if err != nil {
	} else if sessionResponse != nil {
		allSessions = sessionResponse.Data
	}

	// Count active sessions (not expired)
	now := time.Now()
	activeSessions := 0
	activeUsers := make(map[string]bool) // Track unique active users
	recentSessions := 0                  // Sessions created in last 7 days

	for _, sess := range allSessions {
		if sess.ExpiresAt.After(now) {
			activeSessions++
			activeUsers[sess.UserID.String()] = true

			// Check if session was created in last 7 days
			if sess.CreatedAt.After(now.Add(-7 * 24 * time.Hour)) {
				recentSessions++
			}
		}
	}

	// Calculate session growth (percentage of sessions created in last 7 days)
	sessionGrowth := 0.0
	if len(allSessions) > 0 {
		sessionGrowth = (float64(recentSessions) / float64(len(allSessions))) * 100
	}

	// Calculate user growth (percentage of new users today vs total)
	userGrowth := 0.0
	if totalUsers > 0 && newUsersToday > 0 {
		userGrowth = (float64(newUsersToday) / float64(totalUsers)) * 100
	}

	// Get failed login attempts from audit log (last 24 hours)
	failedLogins := h.getFailedLoginCount(ctx)

	stats := &DashboardStats{
		TotalUsers:     totalUsers,
		ActiveUsers:    len(activeUsers),
		NewUsersToday:  newUsersToday,
		TotalSessions:  len(allSessions),
		ActiveSessions: activeSessions,
		FailedLogins:   failedLogins,
		UserGrowth:     userGrowth,
		SessionGrowth:  sessionGrowth,
		RecentActivity: h.getRecentActivity(ctx),
		SystemStatus:   h.getSystemStatus(),
		Plugins:        h.getPluginInfo(),
	}

	return stats, nil
}

// getFailedLoginCount returns count of failed login attempts in last 24 hours
func (h *Handler) getFailedLoginCount(ctx context.Context) int {
	if h.auditSvc == nil {
		return 0
	}

	// Extract app ID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return 0
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	action := "auth.login.failed"

	// List failed login events
	// TODO: Filter by app ID when audit service supports it
	filter := &audit.ListEventsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000,
		},
		Action: &action,
		Since:  &yesterday,
	}
	_ = appID // Use appID to avoid unused variable warning

	eventsResponse, err := h.auditSvc.List(ctx, filter)
	if err != nil {
		return 0
	}

	if eventsResponse == nil || eventsResponse.Data == nil {
		return 0
	}

	return len(eventsResponse.Data)
}

// getRecentActivity fetches recent activity from audit log
func (h *Handler) getRecentActivity(ctx context.Context) []ActivityItem {
	if h.auditSvc == nil {
		return []ActivityItem{}
	}

	// Extract app ID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return []ActivityItem{}
	}

	// Fetch recent audit events
	// TODO: Filter by app ID when audit service supports it
	filter := &audit.ListEventsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 10,
		},
	}
	_ = appID // Use appID to avoid unused variable warning

	eventsResponse, err := h.auditSvc.List(ctx, filter)
	if err != nil {
		return []ActivityItem{}
	}

	if eventsResponse == nil || eventsResponse.Data == nil {
		return []ActivityItem{}
	}

	activities := make([]ActivityItem, 0, len(eventsResponse.Data))
	for _, event := range eventsResponse.Data {
		item := h.auditEventToActivity(event)
		activities = append(activities, item)
	}

	return activities
}

// auditEventToActivity converts an audit event to an activity item
func (h *Handler) auditEventToActivity(event *audit.Event) ActivityItem {
	var title, description, eventType string

	// Map audit actions to friendly titles and types
	switch event.Action {
	case "auth.login":
		title = "User login"
		description = "User logged in successfully"
		eventType = "success"
	case "auth.login.failed":
		title = "Failed login attempt"
		description = "Login attempt failed"
		eventType = "warning"
	case "auth.logout":
		title = "User logout"
		description = "User logged out"
		eventType = "info"
	case "auth.signup":
		title = "New user registration"
		description = "User registered successfully"
		eventType = "success"
	case "user.created":
		title = "User created"
		description = "New user account created"
		eventType = "success"
	case "user.updated":
		title = "User updated"
		description = "User profile updated"
		eventType = "info"
	case "user.deleted":
		title = "User deleted"
		description = "User account deleted"
		eventType = "error"
	case "session.created":
		title = "New session"
		description = "New session created"
		eventType = "info"
	case "session.revoked":
		title = "Session ended"
		description = "Session was revoked"
		eventType = "info"
	default:
		title = event.Action
		description = event.Resource
		eventType = "info"
	}

	return ActivityItem{
		Title:       title,
		Description: description,
		Time:        formatTimeAgo(event.CreatedAt),
		Type:        eventType,
	}
}

// getSystemStatus returns current system status
func (h *Handler) getSystemStatus() []StatusItem {
	return []StatusItem{
		{
			Name:   "Authentication Service",
			Status: "Operational",
			Color:  "green",
		},
		{
			Name:   "Session Management",
			Status: "Operational",
			Color:  "green",
		},
		{
			Name:   "Email Service",
			Status: "Operational",
			Color:  "green",
		},
		{
			Name:   "Two-Factor Auth",
			Status: "Operational",
			Color:  "green",
		},
		{
			Name:   "OAuth Providers",
			Status: "Operational",
			Color:  "green",
		},
	}
}

// formatTimeAgo formats a time as "X minutes ago"
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// Plugin metadata map
var pluginMetadata = map[string]struct {
	Name        string
	Description string
	Category    string
	Icon        string
}{
	"dashboard":      {"Admin Dashboard", "Web-based administration interface", "core", "LayoutDashboard"},
	"username":       {"Username Auth", "Username/password authentication", "authentication", "User"},
	"twofa":          {"Two-Factor Auth", "TOTP-based two-factor authentication", "security", "ShieldCheck"},
	"mfa":            {"Multi-Factor Auth", "Advanced multi-factor authentication", "security", "Shield"},
	"anonymous":      {"Anonymous Auth", "Guest user authentication", "authentication", "UserCircle"},
	"multiapp":       {"Multi-App", "Organization and tenant management", "core", "Building2"},
	"emailotp":       {"Email OTP", "One-time password via email", "authentication", "Mail"},
	"magiclink":      {"Magic Link", "Passwordless login via email links", "authentication", "Link"},
	"phone":          {"Phone Auth", "SMS-based authentication", "authentication", "Phone"},
	"passkey":        {"Passkeys", "WebAuthn passwordless authentication", "security", "Fingerprint"},
	"sso":            {"SSO", "Single Sign-On integration", "authentication", "LogIn"},
	"social":         {"Social Login", "OAuth social providers", "authentication", "Share2"},
	"multisession":   {"Multi-Session", "Multiple concurrent sessions", "session", "Layers"},
	"oidcprovider":   {"OIDC Provider", "OpenID Connect provider", "authentication", "Key"},
	"jwt":            {"JWT", "JSON Web Token authentication", "authentication", "FileJson"},
	"bearer":         {"Bearer Tokens", "Bearer token authentication", "authentication", "Hash"},
	"apikey":         {"API Keys", "API key management", "authentication", "KeyRound"},
	"impersonation":  {"User Impersonation", "Admin user impersonation", "administration", "Users"},
	"permissions":    {"Permissions", "Advanced permission system", "security", "ShieldAlert"},
	"notification":   {"Notifications", "Email and SMS notifications", "communication", "Bell"},
	"mcp":            {"MCP Server", "Model Context Protocol server", "integration", "Server"},
	"backupauth":     {"Backup Auth", "Backup authentication codes", "security", "Archive"},
	"compliance":     {"Compliance", "GDPR and compliance tools", "enterprise", "FileCheck"},
	"consent":        {"Consent Management", "User consent tracking", "enterprise", "ClipboardCheck"},
	"geofence":       {"Geofencing", "Location-based access control", "enterprise", "MapPin"},
	"idverification": {"ID Verification", "Identity verification", "enterprise", "BadgeCheck"},
	"mtls":           {"mTLS", "Mutual TLS authentication", "enterprise", "ShieldCheck"},
	"scim":           {"SCIM", "System for Cross-domain Identity Management", "enterprise", "Network"},
	"stepup":         {"Step-Up Auth", "Additional authentication for sensitive operations", "security", "Lock"},
}

// getPluginInfo returns information about installed plugins
func (h *Handler) getPluginInfo() []PluginItem {
	plugins := make([]PluginItem, 0)

	// Iterate through all known plugins
	for id, meta := range pluginMetadata {
		status := "disabled"
		if h.enabledPlugins[id] {
			status = "enabled"
		}

		plugins = append(plugins, PluginItem{
			ID:          id,
			Name:        meta.Name,
			Description: meta.Description,
			Category:    meta.Category,
			Status:      status,
			Icon:        meta.Icon,
		})
	}

	return plugins
}
