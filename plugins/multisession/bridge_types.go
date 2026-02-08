package multisession

import (
	"time"
)

// =============================================================================
// Bridge Function Input/Output Types
// =============================================================================

// GetSessionsInput is the input for bridgeGetSessions
type GetSessionsInput struct {
	AppID    string `json:"appId"`
	UserID   string `json:"userId,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
	Status   string `json:"status,omitempty"` // "active", "expiring", "expired", "all"
	Device   string `json:"device,omitempty"` // "mobile", "desktop", "tablet", "all"
	Search   string `json:"search,omitempty"` // Search by user ID
}

// GetSessionsResult is the output for bridgeGetSessions
type GetSessionsResult struct {
	Sessions   []SessionDTO       `json:"sessions"`
	Stats      SessionStatsDTO    `json:"stats"`
	Pagination PaginationInfoDTO  `json:"pagination"`
}

// GetSessionInput is the input for bridgeGetSession
type GetSessionInput struct {
	AppID     string `json:"appId"`
	SessionID string `json:"sessionId"`
}

// GetSessionResult is the output for bridgeGetSession
type GetSessionResult struct {
	Session SessionDetailDTO `json:"session"`
}

// RevokeSessionInput is the input for bridgeRevokeSession
type RevokeSessionInput struct {
	AppID     string `json:"appId"`
	SessionID string `json:"sessionId"`
}

// RevokeSessionResult is the output for bridgeRevokeSession
type RevokeSessionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// RevokeAllUserSessionsInput is the input for bridgeRevokeAllUserSessions
type RevokeAllUserSessionsInput struct {
	AppID  string `json:"appId"`
	UserID string `json:"userId"`
}

// RevokeAllUserSessionsResult is the output for bridgeRevokeAllUserSessions
type RevokeAllUserSessionsResult struct {
	Success      bool   `json:"success"`
	RevokedCount int    `json:"revokedCount"`
	Message      string `json:"message,omitempty"`
}

// GetSessionStatsInput is the input for bridgeGetSessionStats
type GetSessionStatsInput struct {
	AppID string `json:"appId"`
}

// GetSessionStatsResult is the output for bridgeGetSessionStats
type GetSessionStatsResult struct {
	Stats SessionStatsDTO `json:"stats"`
}

// GetUserSessionsInput is the input for bridgeGetUserSessions
type GetUserSessionsInput struct {
	AppID    string `json:"appId"`
	UserID   string `json:"userId"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// GetUserSessionsResult is the output for bridgeGetUserSessions
type GetUserSessionsResult struct {
	Sessions    []SessionDTO      `json:"sessions"`
	UserID      string            `json:"userId"`
	TotalCount  int               `json:"totalCount"`
	ActiveCount int               `json:"activeCount"`
	Pagination  PaginationInfoDTO `json:"pagination"`
}

// GetSettingsInput is the input for bridgeGetSettings
type GetSettingsInput struct {
	AppID string `json:"appId"`
}

// GetSettingsResult is the output for bridgeGetSettings
type GetSettingsResult struct {
	Settings SettingsDTO `json:"settings"`
}

// UpdateSettingsInput is the input for bridgeUpdateSettings
type UpdateSettingsInput struct {
	AppID                string `json:"appId"`
	MaxSessionsPerUser   int    `json:"maxSessionsPerUser,omitempty"`
	SessionExpiryHours   int    `json:"sessionExpiryHours,omitempty"`
	EnableDeviceTracking *bool  `json:"enableDeviceTracking,omitempty"`
	AllowCrossPlatform   *bool  `json:"allowCrossPlatform,omitempty"`
}

// UpdateSettingsResult is the output for bridgeUpdateSettings
type UpdateSettingsResult struct {
	Success  bool        `json:"success"`
	Settings SettingsDTO `json:"settings"`
	Message  string      `json:"message,omitempty"`
}

// =============================================================================
// DTO Types
// =============================================================================

// SessionDTO represents a session in list views
type SessionDTO struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	UserEmail    string    `json:"userEmail,omitempty"`
	IPAddress    string    `json:"ipAddress"`
	UserAgent    string    `json:"userAgent"`
	DeviceType   string    `json:"deviceType"`   // mobile, desktop, tablet, bot
	DeviceInfo   string    `json:"deviceInfo"`   // Short device description
	Browser      string    `json:"browser"`
	BrowserVer   string    `json:"browserVersion"`
	OS           string    `json:"os"`
	OSVersion    string    `json:"osVersion"`
	Status       string    `json:"status"`       // active, expiring, expired
	IsActive     bool      `json:"isActive"`
	IsExpiring   bool      `json:"isExpiring"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
	LastUsed     string    `json:"lastUsed"`     // Relative time string
	ExpiresIn    string    `json:"expiresIn"`    // Relative time string
}

// SessionDetailDTO represents detailed session information
type SessionDetailDTO struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	UserEmail       string    `json:"userEmail,omitempty"`
	AppID           string    `json:"appId"`
	OrganizationID  string    `json:"organizationId,omitempty"`
	EnvironmentID   string    `json:"environmentId,omitempty"`
	IPAddress       string    `json:"ipAddress"`
	UserAgent       string    `json:"userAgent"`
	DeviceType      string    `json:"deviceType"`
	DeviceInfo      string    `json:"deviceInfo"`
	Browser         string    `json:"browser"`
	BrowserVer      string    `json:"browserVersion"`
	OS              string    `json:"os"`
	OSVersion       string    `json:"osVersion"`
	Status          string    `json:"status"`
	IsActive        bool      `json:"isActive"`
	IsExpiring      bool      `json:"isExpiring"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
	LastRefreshedAt *time.Time `json:"lastRefreshedAt,omitempty"`
	CreatedAtFmt    string    `json:"createdAtFormatted"`
	UpdatedAtFmt    string    `json:"updatedAtFormatted"`
	ExpiresAtFmt    string    `json:"expiresAtFormatted"`
	LastRefreshedFmt string   `json:"lastRefreshedFormatted,omitempty"`
}

// SessionStatsDTO contains session statistics
type SessionStatsDTO struct {
	TotalSessions int64 `json:"totalSessions"`
	ActiveCount   int   `json:"activeCount"`
	ExpiringCount int   `json:"expiringCount"`
	ExpiredCount  int   `json:"expiredCount"`
	MobileCount   int   `json:"mobileCount"`
	DesktopCount  int   `json:"desktopCount"`
	TabletCount   int   `json:"tabletCount"`
	UniqueUsers   int   `json:"uniqueUsers"`
}

// SettingsDTO represents plugin settings
type SettingsDTO struct {
	MaxSessionsPerUser   int  `json:"maxSessionsPerUser"`
	SessionExpiryHours   int  `json:"sessionExpiryHours"`
	EnableDeviceTracking bool `json:"enableDeviceTracking"`
	AllowCrossPlatform   bool `json:"allowCrossPlatform"`
}

// PaginationInfoDTO contains pagination metadata
type PaginationInfoDTO struct {
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
}
