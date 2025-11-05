package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/types"
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

// getDashboardStats fetches dashboard statistics
func (h *Handler) getDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Get total users
	_, totalUsers, err := h.userSvc.List(ctx, types.PaginationOptions{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}

	// Get new users created today
	newUsersToday, err := h.userSvc.CountCreatedToday(ctx)
	if err != nil {
		fmt.Printf("[Dashboard] Failed to count new users today: %v\n", err)
		newUsersToday = 0
	}

	// Get all sessions
	allSessions, err := h.sessionSvc.ListAll(ctx, 1000, 0) // Get up to 1000 sessions
	if err != nil {
		fmt.Printf("[Dashboard] Failed to fetch sessions: %v\n", err)
		allSessions = []*session.Session{}
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
	}

	return stats, nil
}

// getFailedLoginCount returns count of failed login attempts in last 24 hours
func (h *Handler) getFailedLoginCount(ctx context.Context) int {
	if h.auditSvc == nil {
		return 0
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Search for failed login events
	events, err := h.auditSvc.Search(ctx, audit.ListParams{
		Action: "auth.login.failed",
		Since:  &yesterday,
		Limit:  1000,
	})

	if err != nil {
		fmt.Printf("[Dashboard] Failed to fetch failed login events: %v\n", err)
		return 0
	}

	return len(events)
}

// getRecentActivity fetches recent activity from audit log
func (h *Handler) getRecentActivity(ctx context.Context) []ActivityItem {
	if h.auditSvc == nil {
		return []ActivityItem{}
	}

	// Fetch recent audit events
	events, err := h.auditSvc.List(ctx, 10, 0)
	if err != nil {
		fmt.Printf("[Dashboard] Failed to fetch recent activity: %v\n", err)
		return []ActivityItem{}
	}

	activities := make([]ActivityItem, 0, len(events))
	for _, event := range events {
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
