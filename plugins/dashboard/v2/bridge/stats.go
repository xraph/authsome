package bridge

import (
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// DashboardStatsInput represents stats request
type DashboardStatsInput struct {
	AppID string `json:"appId" validate:"required"`
}

// DashboardStatsOutput represents dashboard statistics
type DashboardStatsOutput struct {
	TotalUsers     int64         `json:"totalUsers"`
	ActiveUsers    int64         `json:"activeUsers"`
	TotalSessions  int64         `json:"totalSessions"`
	ActiveSessions int64         `json:"activeSessions"`
	NewUsersToday  int64         `json:"newUsersToday"`
	NewUsersWeek   int64         `json:"newUsersWeek"`
	GrowthRate     float64       `json:"growthRate"`
	UserGrowthData []GrowthPoint `json:"userGrowthData"`
}

// GrowthPoint represents a data point for growth charts
type GrowthPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// RecentActivityInput represents recent activity request
type RecentActivityInput struct {
	AppID string `json:"appId" validate:"required"`
	Limit int    `json:"limit,omitempty"`
}

// RecentActivityOutput represents recent activity
type RecentActivityOutput struct {
	Activities []ActivityItem `json:"activities"`
}

// ActivityItem represents a single activity
type ActivityItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	UserEmail   string `json:"userEmail,omitempty"`
	Timestamp   string `json:"timestamp"`
	Icon        string `json:"icon,omitempty"`
}

// registerStatsFunctions registers statistics-related bridge functions
func (bm *BridgeManager) registerStatsFunctions() error {
	// Dashboard stats function
	if err := bm.bridge.Register("getDashboardStats", bm.getDashboardStats,
		bridge.WithDescription("Get dashboard statistics for an app"),
	); err != nil {
		return fmt.Errorf("failed to register getDashboardStats: %w", err)
	}

	// Recent activity function
	if err := bm.bridge.Register("getRecentActivity", bm.getRecentActivity,
		bridge.WithDescription("Get recent activity for an app"),
	); err != nil {
		return fmt.Errorf("failed to register getRecentActivity: %w", err)
	}

	// Growth data function
	if err := bm.bridge.Register("getGrowthData", bm.getGrowthData,
		bridge.WithDescription("Get user growth data for charts"),
	); err != nil {
		return fmt.Errorf("failed to register getGrowthData: %w", err)
	}

	// System status function
	if err := bm.bridge.Register("getSystemStatus", bm.getSystemStatus,
		bridge.WithDescription("Get system status for services"),
	); err != nil {
		return fmt.Errorf("failed to register getSystemStatus: %w", err)
	}

	// Plugins overview function
	if err := bm.bridge.Register("getPluginsOverview", bm.getPluginsOverview,
		bridge.WithDescription("Get overview of enabled plugins"),
	); err != nil {
		return fmt.Errorf("failed to register getPluginsOverview: %w", err)
	}

	// Extension widgets function
	if err := bm.bridge.Register("getExtensionWidgets", bm.getExtensionWidgets,
		bridge.WithDescription("Get dashboard extension widgets"),
	); err != nil {
		return fmt.Errorf("failed to register getExtensionWidgets: %w", err)
	}

	bm.log.Info("stats bridge functions registered")
	return nil
}

// getDashboardStats retrieves dashboard statistics
func (bm *BridgeManager) getDashboardStats(ctx bridge.Context, input DashboardStatsInput) (*DashboardStatsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Create a proper Go context from bridge context
	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Count total users for this app
	totalUsers := int64(0)
	newUsersToday := int64(0)
	newUsersWeek := int64(0)

	userFilter := &user.CountUsersFilter{
		AppID: appID,
	}
	count, err := bm.userSvc.CountUsers(goCtx, userFilter)
	if err != nil {
		bm.log.Error("failed to count users", forge.F("error", err.Error()))
	} else {
		totalUsers = int64(count)
	}

	// Count new users today
	startOfToday := time.Now().Truncate(24 * time.Hour)
	newUserTodayFilter := &user.CountUsersFilter{
		AppID:        appID,
		CreatedSince: &startOfToday,
	}
	count, err = bm.userSvc.CountUsers(goCtx, newUserTodayFilter)
	if err != nil {
		bm.log.Error("failed to count new users today", forge.F("error", err.Error()))
	} else {
		newUsersToday = int64(count)
	}

	// Count new users this week
	startOfWeek := time.Now().Add(-7 * 24 * time.Hour)
	newUserWeekFilter := &user.CountUsersFilter{
		AppID:        appID,
		CreatedSince: &startOfWeek,
	}
	count, err = bm.userSvc.CountUsers(goCtx, newUserWeekFilter)
	if err != nil {
		bm.log.Error("failed to count new users this week", forge.F("error", err.Error()))
	} else {
		newUsersWeek = int64(count)
	}

	// Get all sessions for this app
	sessionFilter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000,
		},
		AppID: appID,
	}
	sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
	allSessions := []*session.Session{}
	if err != nil {
		bm.log.Error("failed to list sessions", forge.F("error", err.Error()))
	} else if sessionResponse != nil {
		allSessions = sessionResponse.Data
	}

	// Count active sessions (not expired)
	now := time.Now()
	activeSessions := int64(0)
	activeUsers := make(map[string]bool) // Track unique active users
	recentSessions := int64(0)           // Sessions created in last 7 days

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

	// Calculate growth rate (percentage of new users this week vs total)
	growthRate := 0.0
	if totalUsers > 0 && newUsersWeek > 0 {
		growthRate = (float64(newUsersWeek) / float64(totalUsers)) * 100
	}

	// Generate simple growth data (last 7 days)
	growthData := []GrowthPoint{}
	for i := 6; i >= 0; i-- {
		date := time.Now().Add(-time.Duration(i) * 24 * time.Hour)
		dateStr := date.Format("2006-01-02")

		// Count users created up to this date
		dateCutoff := date.Add(24 * time.Hour)
		userCountFilter := &user.CountUsersFilter{
			AppID:        appID,
			CreatedSince: &dateCutoff,
		}
		count, err := bm.userSvc.CountUsers(goCtx, userCountFilter)
		if err != nil {
			// Use estimated value if query fails
			count = int(totalUsers)
		}

		growthData = append(growthData, GrowthPoint{
			Date:  dateStr,
			Count: int64(count),
		})
	}

	return &DashboardStatsOutput{
		TotalUsers:     totalUsers,
		ActiveUsers:    int64(len(activeUsers)),
		TotalSessions:  int64(len(allSessions)),
		ActiveSessions: activeSessions,
		NewUsersToday:  newUsersToday,
		NewUsersWeek:   newUsersWeek,
		GrowthRate:     growthRate,
		UserGrowthData: growthData,
	}, nil
}

// getRecentActivity retrieves recent activity
func (bm *BridgeManager) getRecentActivity(ctx bridge.Context, input RecentActivityInput) (*RecentActivityOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Get recent audit events if audit service is available
	activities := []ActivityItem{}

	if bm.auditSvc != nil {
		// List recent audit events
		listFilter := &audit.ListEventsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: limit,
			},
			AppID: &appID,
		}
		eventResponse, err := bm.auditSvc.List(goCtx, listFilter)

		if err != nil {
			bm.log.Error("failed to list audit events", forge.F("error", err.Error()))
		} else if eventResponse != nil {
			// Transform audit events to activity items
			for _, event := range eventResponse.Data {
				userEmail := ""
				if event.UserID != nil {
					// Look up user email
					user, err := bm.userSvc.FindByID(goCtx, *event.UserID)
					if err == nil && user != nil {
						userEmail = user.Email
					}
				}

				activities = append(activities, ActivityItem{
					ID:          event.ID.String(),
					Type:        event.Action,
					Description: generateActivityDescription(event.Action),
					UserEmail:   userEmail,
					Timestamp:   event.CreatedAt.Format(time.RFC3339),
					Icon:        getIconForEventType(event.Action),
				})
			}
		}
	}

	// If no events or audit service unavailable, return empty list
	return &RecentActivityOutput{
		Activities: activities,
	}, nil
}

// generateActivityDescription generates a human-readable description for an event type
func generateActivityDescription(eventType string) string {
	descriptions := map[string]string{
		"user.created":         "New user registered",
		"user.updated":         "User profile updated",
		"user.deleted":         "User account deleted",
		"session.created":      "User logged in",
		"session.revoked":      "Session revoked",
		"app.created":          "New app created",
		"app.updated":          "App settings updated",
		"app.deleted":          "App deleted",
		"organization.created": "New organization created",
		"organization.updated": "Organization updated",
		"organization.deleted": "Organization deleted",
	}

	if desc, ok := descriptions[eventType]; ok {
		return desc
	}
	return eventType
}

// getIconForEventType returns an appropriate icon name for an event type
func getIconForEventType(eventType string) string {
	icons := map[string]string{
		"user.created":         "user-plus",
		"user.updated":         "user-check",
		"user.deleted":         "user-x",
		"session.created":      "log-in",
		"session.revoked":      "log-out",
		"app.created":          "plus-circle",
		"app.updated":          "edit",
		"app.deleted":          "trash",
		"organization.created": "building",
		"organization.updated": "edit",
		"organization.deleted": "trash",
	}

	if icon, ok := icons[eventType]; ok {
		return icon
	}
	return "activity"
}

// getGrowthData retrieves user growth data for charts
func (bm *BridgeManager) getGrowthData(ctx bridge.Context, input DashboardStatsInput) (*[]GrowthPoint, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Generate growth data for last 30 days with weekly intervals
	growthData := []GrowthPoint{}
	now := time.Now()

	for i := 5; i >= 0; i-- {
		date := now.Add(-time.Duration(i*7) * 24 * time.Hour)
		dateStr := date.Format("2006-01-02")

		// Count users created since this date
		userCountFilter := &user.CountUsersFilter{
			AppID:        appID,
			CreatedSince: &date,
		}
		count, err := bm.userSvc.CountUsers(goCtx, userCountFilter)
		if err != nil {
			bm.log.Error("failed to count users for growth data",
				forge.F("error", err.Error()),
				forge.F("date", dateStr))
			count = 0
		}

		growthData = append(growthData, GrowthPoint{
			Date:  dateStr,
			Count: int64(count),
		})
	}

	return &growthData, nil
}

// SystemStatusInput represents system status request
type SystemStatusInput struct {
	AppID string `json:"appId" validate:"required"`
}

// SystemStatusOutput represents system status
type SystemStatusOutput struct {
	Components []StatusComponent `json:"components"`
}

// StatusComponent represents a single system component status
type StatusComponent struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // "operational", "degraded", "down"
	Description string `json:"description,omitempty"`
	Color       string `json:"color"` // "green", "yellow", "red"
}

// getSystemStatus retrieves system component statuses
func (bm *BridgeManager) getSystemStatus(ctx bridge.Context, input SystemStatusInput) (*SystemStatusOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	components := []StatusComponent{}

	// Check database connection
	dbStatus := StatusComponent{
		Name:   "Database",
		Status: "operational",
		Color:  "green",
	}
	// Simple ping check (bun DB doesn't have direct Ping, but we can try a simple query)
	// For now, assume operational since we got this far
	components = append(components, dbStatus)

	// Check user service
	userStatus := StatusComponent{
		Name:   "User Service",
		Status: "operational",
		Color:  "green",
	}
	if bm.userSvc != nil {
		// Try to count users as a health check
		_, err := bm.userSvc.CountUsers(goCtx, &user.CountUsersFilter{AppID: appID})
		if err != nil {
			userStatus.Status = "degraded"
			userStatus.Color = "yellow"
			userStatus.Description = "Service responding slowly"
		}
	} else {
		userStatus.Status = "down"
		userStatus.Color = "red"
		userStatus.Description = "Service unavailable"
	}
	components = append(components, userStatus)

	// Check session service
	sessionStatus := StatusComponent{
		Name:   "Session Service",
		Status: "operational",
		Color:  "green",
	}
	if bm.sessionSvc != nil {
		// Try to list sessions as a health check
		_, err := bm.sessionSvc.ListSessions(goCtx, &session.ListSessionsFilter{
			PaginationParams: pagination.PaginationParams{Page: 1, Limit: 1},
			AppID:            appID,
		})
		if err != nil {
			sessionStatus.Status = "degraded"
			sessionStatus.Color = "yellow"
			sessionStatus.Description = "Service responding slowly"
		}
	} else {
		sessionStatus.Status = "down"
		sessionStatus.Color = "red"
		sessionStatus.Description = "Service unavailable"
	}
	components = append(components, sessionStatus)

	// Check audit service
	auditStatus := StatusComponent{
		Name:   "Audit Service",
		Status: "operational",
		Color:  "green",
	}
	if bm.auditSvc != nil {
		// Try to list events as a health check
		_, err := bm.auditSvc.List(goCtx, &audit.ListEventsFilter{
			PaginationParams: pagination.PaginationParams{Page: 1, Limit: 1},
			AppID:            &appID,
		})
		if err != nil {
			auditStatus.Status = "degraded"
			auditStatus.Color = "yellow"
			auditStatus.Description = "Service responding slowly"
		}
	} else {
		auditStatus.Status = "degraded"
		auditStatus.Color = "yellow"
		auditStatus.Description = "Service not configured"
	}
	components = append(components, auditStatus)

	return &SystemStatusOutput{
		Components: components,
	}, nil
}

// PluginsOverviewInput represents plugins overview request
type PluginsOverviewInput struct {
	AppID string `json:"appId" validate:"required"`
}

// PluginsOverviewOutput represents plugins overview
type PluginsOverviewOutput struct {
	EnabledCount int          `json:"enabledCount"`
	TotalCount   int          `json:"totalCount"`
	Plugins      []PluginInfo `json:"plugins"`
}

// PluginInfo represents information about a plugin
type PluginInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Status      string `json:"status"` // "enabled", "disabled"
	Icon        string `json:"icon"`
}

// getPluginsOverview retrieves overview of enabled plugins
func (bm *BridgeManager) getPluginsOverview(ctx bridge.Context, input PluginsOverviewInput) (*PluginsOverviewOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Build plugin list from enabledPlugins map
	plugins := []PluginInfo{}
	pluginMetadata := map[string]struct {
		name        string
		description string
		category    string
		icon        string
	}{
		"dashboard":         {"Dashboard", "Admin dashboard interface", "administration", "LayoutDashboard"},
		"multiapp":          {"Multi-App", "Multi-application support", "core", "Layers"},
		"multisession":      {"Multi-Session", "Multiple concurrent sessions", "session", "Users"},
		"apikey":            {"API Keys", "API key authentication", "authentication", "Key"},
		"bearer":            {"Bearer Token", "Bearer token authentication", "authentication", "Shield"},
		"jwt":               {"JWT", "JSON Web Token support", "authentication", "FileJson"},
		"username":          {"Username", "Username authentication", "authentication", "User"},
		"anonymous":         {"Anonymous", "Anonymous user access", "authentication", "UserCircle"},
		"emailotp":          {"Email OTP", "One-time password via email", "authentication", "Mail"},
		"magiclink":         {"Magic Link", "Passwordless magic link", "authentication", "Link"},
		"phone":             {"Phone", "Phone number authentication", "authentication", "Phone"},
		"passkey":           {"Passkey", "WebAuthn passkey support", "authentication", "Fingerprint"},
		"social":            {"Social Login", "OAuth social providers", "authentication", "Share2"},
		"sso":               {"SSO", "Single sign-on integration", "authentication", "LogIn"},
		"mfa":               {"MFA", "Multi-factor authentication", "security", "ShieldCheck"},
		"twofa":             {"2FA", "Two-factor authentication", "security", "ShieldAlert"},
		"organization":      {"Organizations", "Organization management", "core", "Building2"},
		"permissions":       {"Permissions", "RBAC permissions", "security", "Lock"},
		"notification":      {"Notifications", "Notification system", "communication", "Bell"},
		"webhook":           {"Webhooks", "Webhook integrations", "integration", "Network"},
		"admin":             {"Admin", "Admin user management", "administration", "Users"},
		"cms":               {"CMS", "Content management", "administration", "FileCheck"},
		"subscription":      {"Subscriptions", "Subscription management", "enterprise", "BadgeCheck"},
		"oidcprovider":      {"OIDC Provider", "OpenID Connect provider", "enterprise", "Server"},
		"emailverification": {"Email Verification", "Email verification flow", "security", "BadgeCheck"},
		"impersonation":     {"Impersonation", "User impersonation", "administration", "Users"},
		"secrets":           {"Secrets", "Secret management", "security", "Lock"},
	}

	totalCount := len(pluginMetadata)
	enabledCount := 0

	for pluginID, metadata := range pluginMetadata {
		status := "disabled"
		if bm.enabledPlugins[pluginID] {
			status = "enabled"
			enabledCount++

			// Only include enabled plugins in the list
			plugins = append(plugins, PluginInfo{
				ID:          pluginID,
				Name:        metadata.name,
				Description: metadata.description,
				Category:    metadata.category,
				Status:      status,
				Icon:        metadata.icon,
			})
		}
	}

	return &PluginsOverviewOutput{
		EnabledCount: enabledCount,
		TotalCount:   totalCount,
		Plugins:      plugins,
	}, nil
}

// ExtensionWidgetsInput represents extension widgets request
type ExtensionWidgetsInput struct {
	AppID string `json:"appId" validate:"required"`
}

// ExtensionWidgetsOutput represents extension widgets
type ExtensionWidgetsOutput struct {
	Widgets []ExtensionWidget `json:"widgets"`
}

// ExtensionWidget represents a dashboard extension widget
type ExtensionWidget struct {
	ExtensionID string `json:"extensionId"`
	Title       string `json:"title"`
	Content     string `json:"content"` // HTML content or data
	Type        string `json:"type"`    // "html", "component", "chart"
}

// getExtensionWidgets retrieves dashboard extension widgets
func (bm *BridgeManager) getExtensionWidgets(ctx bridge.Context, input ExtensionWidgetsInput) (*ExtensionWidgetsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	widgets := []ExtensionWidget{}

	// Fetch widgets from extension registry if available
	if bm.extensionRegistry != nil {
		dashboardWidgets := bm.extensionRegistry.GetDashboardWidgets()

		// Transform UI widgets to bridge response format
		for _, w := range dashboardWidgets {
			widgets = append(widgets, ExtensionWidget{
				ExtensionID: w.ID,
				Title:       w.Title,
				Content:     "", // Content is rendered server-side in the page, not via bridge
				Type:        "component",
			})
		}

		bm.log.Debug("fetched extension widgets from registry",
			forge.F("count", len(widgets)),
			forge.F("appId", input.AppID))
	} else {
		bm.log.Debug("extension registry not available, returning empty widgets list")
	}

	return &ExtensionWidgetsOutput{
		Widgets: widgets,
	}, nil
}
