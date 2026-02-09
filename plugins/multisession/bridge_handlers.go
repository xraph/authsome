package multisession

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Bridge Function Implementations
// =============================================================================

// bridgeGetSessions handles the getSessions bridge call.
func (e *DashboardExtension) bridgeGetSessions(ctx bridge.Context, input GetSessionsInput) (*GetSessionsResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse app ID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	// Set defaults
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	// Build filter
	filter := &session.ListSessionsFilter{
		AppID: appID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: pageSize,
		},
	}

	// Add user filter if provided
	if input.UserID != "" {
		userID, err := xid.FromString(input.UserID)
		if err != nil {
			return nil, errs.BadRequest("invalid userId")
		}

		filter.UserID = &userID
	}

	// Fetch sessions
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(goCtx, filter)
	if err != nil {
		e.plugin.logger.Error("failed to list sessions", forge.F("error", err.Error()))

		return nil, errs.InternalServerError("failed to fetch sessions", err)
	}

	// Calculate stats and filter sessions
	now := time.Now()
	soonThreshold := 24 * time.Hour
	userMap := make(map[string]bool)

	stats := SessionStatsDTO{}
	sessions := make([]SessionDTO, 0, len(sessionsResp.Data))

	for _, sess := range sessionsResp.Data {
		device := ParseUserAgent(sess.UserAgent)
		isActive := sess.ExpiresAt.After(now)
		isExpiring := isActive && sess.ExpiresAt.Sub(now) < soonThreshold

		// Calculate stats
		if !sess.UserID.IsNil() {
			userMap[sess.UserID.String()] = true
		}

		if isActive {
			stats.ActiveCount++
			if isExpiring {
				stats.ExpiringCount++
			}
		} else {
			stats.ExpiredCount++
		}

		switch {
		case device.IsMobile:
			stats.MobileCount++
		case device.IsTablet:
			stats.TabletCount++
		default:
			stats.DesktopCount++
		}

		// Apply filters
		if !e.matchesFilters(sess, device, input.Status, input.Device, input.Search, now, soonThreshold) {
			continue
		}

		// Build DTO
		status := "active"
		if !isActive {
			status = "expired"
		} else if isExpiring {
			status = "expiring"
		}

		sessions = append(sessions, SessionDTO{
			ID:         sess.ID.String(),
			UserID:     sess.UserID.String(),
			IPAddress:  sess.IPAddress,
			UserAgent:  sess.UserAgent,
			DeviceType: device.DeviceType,
			DeviceInfo: device.ShortDeviceInfo(),
			Browser:    device.Browser,
			BrowserVer: device.BrowserVer,
			OS:         device.OS,
			OSVersion:  device.OSVersion,
			Status:     status,
			IsActive:   isActive,
			IsExpiring: isExpiring,
			CreatedAt:  sess.CreatedAt,
			ExpiresAt:  sess.ExpiresAt,
			LastUsed:   FormatRelativeTime(sess.CreatedAt),
			ExpiresIn:  FormatExpiresIn(sess.ExpiresAt),
		})
	}

	stats.TotalSessions = sessionsResp.Pagination.Total
	stats.UniqueUsers = len(userMap)

	// Build pagination
	totalPages := int((sessionsResp.Pagination.Total + int64(pageSize) - 1) / int64(pageSize))

	return &GetSessionsResult{
		Sessions: sessions,
		Stats:    stats,
		Pagination: PaginationInfoDTO{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  sessionsResp.Pagination.Total,
			TotalPages:  totalPages,
		},
	}, nil
}

// matchesFilters checks if a session matches the filter criteria.
func (e *DashboardExtension) matchesFilters(sess *session.Session, device *DeviceInfo, statusFilter, deviceFilter, search string, now time.Time, soonThreshold time.Duration) bool {
	isActive := sess.ExpiresAt.After(now)
	isExpiring := isActive && sess.ExpiresAt.Sub(now) < soonThreshold

	// Status filter
	if statusFilter != "" && statusFilter != "all" {
		switch statusFilter {
		case "active":
			if !isActive {
				return false
			}
		case "expiring":
			if !isExpiring {
				return false
			}
		case "expired":
			if isActive {
				return false
			}
		}
	}

	// Device filter
	if deviceFilter != "" && deviceFilter != "all" {
		switch deviceFilter {
		case "mobile":
			if !device.IsMobile {
				return false
			}
		case "desktop":
			if !device.IsDesktop {
				return false
			}
		case "tablet":
			if !device.IsTablet {
				return false
			}
		}
	}

	// Search filter (by user ID)
	if search != "" {
		userIDStr := sess.UserID.String()
		if len(search) > len(userIDStr) {
			return false
		}
		// Simple contains check
		found := false

		for i := 0; i <= len(userIDStr)-len(search); i++ {
			if userIDStr[i:i+len(search)] == search {
				found = true

				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// bridgeGetSession handles the getSession bridge call.
func (e *DashboardExtension) bridgeGetSession(ctx bridge.Context, input GetSessionInput) (*GetSessionResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse session ID
	sessionID, err := xid.FromString(input.SessionID)
	if err != nil {
		return nil, errs.BadRequest("invalid sessionId")
	}

	// Fetch session
	sess, err := e.plugin.service.sessionSvc.FindByID(goCtx, sessionID)
	if err != nil {
		return nil, errs.NotFound("session not found")
	}

	// Parse device info
	device := ParseUserAgent(sess.UserAgent)
	now := time.Now()
	isActive := sess.ExpiresAt.After(now)
	isExpiring := isActive && sess.ExpiresAt.Sub(now) < 24*time.Hour

	status := "active"
	if !isActive {
		status = "expired"
	} else if isExpiring {
		status = "expiring"
	}

	detail := SessionDetailDTO{
		ID:           sess.ID.String(),
		UserID:       sess.UserID.String(),
		AppID:        sess.AppID.String(),
		IPAddress:    sess.IPAddress,
		UserAgent:    sess.UserAgent,
		DeviceType:   device.DeviceType,
		DeviceInfo:   device.FormatDeviceInfo(),
		Browser:      device.Browser,
		BrowserVer:   device.BrowserVer,
		OS:           device.OS,
		OSVersion:    device.OSVersion,
		Status:       status,
		IsActive:     isActive,
		IsExpiring:   isExpiring,
		CreatedAt:    sess.CreatedAt,
		UpdatedAt:    sess.UpdatedAt,
		ExpiresAt:    sess.ExpiresAt,
		CreatedAtFmt: sess.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"),
		UpdatedAtFmt: sess.UpdatedAt.Format("Jan 2, 2006 at 3:04 PM"),
		ExpiresAtFmt: sess.ExpiresAt.Format("Jan 2, 2006 at 3:04 PM"),
	}

	// Optional IDs
	if sess.OrganizationID != nil {
		detail.OrganizationID = sess.OrganizationID.String()
	}

	if sess.EnvironmentID != nil {
		detail.EnvironmentID = sess.EnvironmentID.String()
	}

	if sess.LastRefreshedAt != nil {
		detail.LastRefreshedAt = sess.LastRefreshedAt
		detail.LastRefreshedFmt = sess.LastRefreshedAt.Format("Jan 2, 2006 at 3:04 PM")
	}

	return &GetSessionResult{
		Session: detail,
	}, nil
}

// bridgeRevokeSession handles the revokeSession bridge call.
func (e *DashboardExtension) bridgeRevokeSession(ctx bridge.Context, input RevokeSessionInput) (*RevokeSessionResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse session ID
	sessionID, err := xid.FromString(input.SessionID)
	if err != nil {
		return nil, errs.BadRequest("invalid sessionId")
	}

	// Revoke session
	if err := e.plugin.service.sessionSvc.RevokeByID(goCtx, sessionID); err != nil {
		e.plugin.logger.Error("failed to revoke session",
			forge.F("error", err.Error()),
			forge.F("sessionId", input.SessionID))

		return nil, errs.InternalServerError("failed to revoke session", err)
	}

	return &RevokeSessionResult{
		Success: true,
		Message: "Session revoked successfully",
	}, nil
}

// bridgeRevokeAllUserSessions handles the revokeAllUserSessions bridge call.
func (e *DashboardExtension) bridgeRevokeAllUserSessions(ctx bridge.Context, input RevokeAllUserSessionsInput) (*RevokeAllUserSessionsResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, errs.BadRequest("invalid userId")
	}

	// Get all user sessions
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(goCtx, &session.ListSessionsFilter{
		UserID: &userID,
		AppID:  appID,
		PaginationParams: pagination.PaginationParams{
			Limit: 1000,
		},
	})
	if err != nil {
		return nil, errs.InternalServerError("failed to fetch user sessions", err)
	}

	// Revoke each session
	revokedCount := 0

	if sessionsResp != nil {
		for _, sess := range sessionsResp.Data {
			if err := e.plugin.service.sessionSvc.RevokeByID(goCtx, sess.ID); err == nil {
				revokedCount++
			}
		}
	}

	return &RevokeAllUserSessionsResult{
		Success:      true,
		RevokedCount: revokedCount,
		Message:      fmt.Sprintf("Revoked %d sessions", revokedCount),
	}, nil
}

// bridgeGetUserSessions handles the getUserSessions bridge call.
func (e *DashboardExtension) bridgeGetUserSessions(ctx bridge.Context, input GetUserSessionsInput) (*GetUserSessionsResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, errs.BadRequest("invalid userId")
	}

	// Set defaults
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}

	// Fetch user sessions
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(goCtx, &session.ListSessionsFilter{
		UserID: &userID,
		AppID:  appID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: pageSize,
		},
	})
	if err != nil {
		return nil, errs.InternalServerError("failed to fetch user sessions", err)
	}

	now := time.Now()
	soonThreshold := 24 * time.Hour
	activeCount := 0
	sessions := make([]SessionDTO, 0, len(sessionsResp.Data))

	for _, sess := range sessionsResp.Data {
		device := ParseUserAgent(sess.UserAgent)
		isActive := sess.ExpiresAt.After(now)
		isExpiring := isActive && sess.ExpiresAt.Sub(now) < soonThreshold

		if isActive {
			activeCount++
		}

		status := "active"
		if !isActive {
			status = "expired"
		} else if isExpiring {
			status = "expiring"
		}

		sessions = append(sessions, SessionDTO{
			ID:         sess.ID.String(),
			UserID:     sess.UserID.String(),
			IPAddress:  sess.IPAddress,
			UserAgent:  sess.UserAgent,
			DeviceType: device.DeviceType,
			DeviceInfo: device.ShortDeviceInfo(),
			Browser:    device.Browser,
			BrowserVer: device.BrowserVer,
			OS:         device.OS,
			OSVersion:  device.OSVersion,
			Status:     status,
			IsActive:   isActive,
			IsExpiring: isExpiring,
			CreatedAt:  sess.CreatedAt,
			ExpiresAt:  sess.ExpiresAt,
			LastUsed:   FormatRelativeTime(sess.CreatedAt),
			ExpiresIn:  FormatExpiresIn(sess.ExpiresAt),
		})
	}

	totalPages := int((sessionsResp.Pagination.Total + int64(pageSize) - 1) / int64(pageSize))

	return &GetUserSessionsResult{
		Sessions:    sessions,
		UserID:      input.UserID,
		TotalCount:  len(sessions),
		ActiveCount: activeCount,
		Pagination: PaginationInfoDTO{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  sessionsResp.Pagination.Total,
			TotalPages:  totalPages,
		},
	}, nil
}

// bridgeGetSessionStats handles the getSessionStats bridge call.
func (e *DashboardExtension) bridgeGetSessionStats(ctx bridge.Context, input GetSessionStatsInput) (*GetSessionStatsResult, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse app ID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	// Fetch all sessions for stats
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(goCtx, &session.ListSessionsFilter{
		AppID: appID,
		PaginationParams: pagination.PaginationParams{
			Limit: 1000,
		},
	})
	if err != nil {
		return nil, errs.InternalServerError("failed to fetch sessions", err)
	}

	now := time.Now()
	soonThreshold := 24 * time.Hour
	userMap := make(map[string]bool)

	stats := SessionStatsDTO{
		TotalSessions: sessionsResp.Pagination.Total,
	}

	for _, sess := range sessionsResp.Data {
		device := ParseUserAgent(sess.UserAgent)
		isActive := sess.ExpiresAt.After(now)
		isExpiring := isActive && sess.ExpiresAt.Sub(now) < soonThreshold

		if !sess.UserID.IsNil() {
			userMap[sess.UserID.String()] = true
		}

		if isActive {
			stats.ActiveCount++
			if isExpiring {
				stats.ExpiringCount++
			}
		} else {
			stats.ExpiredCount++
		}

		switch {
		case device.IsMobile:
			stats.MobileCount++
		case device.IsTablet:
			stats.TabletCount++
		default:
			stats.DesktopCount++
		}
	}

	stats.UniqueUsers = len(userMap)

	return &GetSessionStatsResult{
		Stats: stats,
	}, nil
}

// bridgeGetSettings handles the getSettings bridge call.
func (e *DashboardExtension) bridgeGetSettings(ctx bridge.Context, input GetSettingsInput) (*GetSettingsResult, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return &GetSettingsResult{
		Settings: SettingsDTO{
			MaxSessionsPerUser:   e.plugin.config.MaxSessionsPerUser,
			SessionExpiryHours:   e.plugin.config.SessionExpiryHours,
			EnableDeviceTracking: e.plugin.config.EnableDeviceTracking,
			AllowCrossPlatform:   e.plugin.config.AllowCrossPlatform,
		},
	}, nil
}

// bridgeUpdateSettings handles the updateSettings bridge call.
func (e *DashboardExtension) bridgeUpdateSettings(ctx bridge.Context, input UpdateSettingsInput) (*UpdateSettingsResult, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Update config values
	if input.MaxSessionsPerUser > 0 && input.MaxSessionsPerUser <= 100 {
		e.plugin.config.MaxSessionsPerUser = input.MaxSessionsPerUser
	}

	if input.SessionExpiryHours > 0 && input.SessionExpiryHours <= 8760 {
		e.plugin.config.SessionExpiryHours = input.SessionExpiryHours
	}

	if input.EnableDeviceTracking != nil {
		e.plugin.config.EnableDeviceTracking = *input.EnableDeviceTracking
	}

	if input.AllowCrossPlatform != nil {
		e.plugin.config.AllowCrossPlatform = *input.AllowCrossPlatform
	}

	return &UpdateSettingsResult{
		Success: true,
		Settings: SettingsDTO{
			MaxSessionsPerUser:   e.plugin.config.MaxSessionsPerUser,
			SessionExpiryHours:   e.plugin.config.SessionExpiryHours,
			EnableDeviceTracking: e.plugin.config.EnableDeviceTracking,
			AllowCrossPlatform:   e.plugin.config.AllowCrossPlatform,
		},
		Message: "Settings updated successfully",
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildContextFromBridge retrieves the Go context from the HTTP request.
// The context has already been enriched by the dashboard v2 BridgeContextMiddleware.
func (e *DashboardExtension) buildContextFromBridge(bridgeCtx bridge.Context, appID string) (context.Context, error) {
	// Get the already-enriched context from the HTTP request
	var goCtx context.Context

	req := bridgeCtx.Request()

	if req != nil {
		goCtx = req.Context()
	} else {
		goCtx = bridgeCtx.Context()
	}

	// Parse the requested app ID
	requestedAppID, err := xid.FromString(appID)
	if err != nil {
		e.plugin.logger.Error("[MultisessionBridge] Invalid app ID", forge.F("appID", appID), forge.F("error", err))

		return nil, errs.BadRequest("invalid appId")
	}

	// Verify that user is authenticated
	userID, hasUserID := contexts.GetUserID(goCtx)
	if !hasUserID || userID == xid.NilID() {
		e.plugin.logger.Error("[MultisessionBridge] Unauthorized - no user ID in context")

		return nil, errs.Unauthorized()
	}

	// Override app ID if different from session
	existingAppID, _ := contexts.GetAppID(goCtx)
	if existingAppID != requestedAppID {
		goCtx = contexts.SetAppID(goCtx, requestedAppID)
	}

	return goCtx, nil
}

// getBridgeFunctions returns the bridge functions for registration.
func (e *DashboardExtension) getBridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		{
			Name:        "getSessions",
			Handler:     e.bridgeGetSessions,
			Description: "List sessions with filters and pagination",
		},
		{
			Name:        "getSession",
			Handler:     e.bridgeGetSession,
			Description: "Get detailed session information",
		},
		{
			Name:        "revokeSession",
			Handler:     e.bridgeRevokeSession,
			Description: "Revoke a specific session",
		},
		{
			Name:        "revokeAllUserSessions",
			Handler:     e.bridgeRevokeAllUserSessions,
			Description: "Revoke all sessions for a user",
		},
		{
			Name:        "getUserSessions",
			Handler:     e.bridgeGetUserSessions,
			Description: "Get all sessions for a specific user",
		},
		{
			Name:        "getSessionStats",
			Handler:     e.bridgeGetSessionStats,
			Description: "Get session statistics for dashboard",
		},
		{
			Name:        "getSettings",
			Handler:     e.bridgeGetSettings,
			Description: "Get multisession plugin settings",
		},
		{
			Name:        "updateSettings",
			Handler:     e.bridgeUpdateSettings,
			Description: "Update multisession plugin settings",
		},
	}
}
