package bridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// SessionsListInput represents sessions list request.
type SessionsListInput struct {
	AppID       string  `json:"appId"                 validate:"required"`
	UserID      string  `json:"userId,omitempty"`
	Page        int     `json:"page"`
	PageSize    int     `json:"pageSize"`
	Status      *string `json:"status,omitempty"`      // "active", "expired", "all"
	SearchEmail string  `json:"searchEmail,omitempty"` // Email search filter
}

// SessionsListOutput represents sessions list response.
type SessionsListOutput struct {
	Sessions   []SessionItem `json:"sessions"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// SessionItem represents a session in the list.
type SessionItem struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	Device    string `json:"device,omitempty"`
	Location  string `json:"location,omitempty"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
	LastUsed  string `json:"lastUsed,omitempty"`
	IsActive  bool   `json:"isActive"`
}

// RevokeSessionInput represents session revoke request.
type RevokeSessionInput struct {
	SessionID string `json:"sessionId" validate:"required"`
}

// RevokeAllSessionsInput represents revoke all sessions request.
type RevokeAllSessionsInput struct {
	UserID string `json:"userId" validate:"required"`
}

// registerSessionFunctions registers session management bridge functions.
func (bm *BridgeManager) registerSessionFunctions() error {
	// List sessions
	if err := bm.bridge.Register("getSessionsList", bm.getSessionsList,
		bridge.WithDescription("Get list of active sessions"),
	); err != nil {
		return fmt.Errorf("failed to register getSessionsList: %w", err)
	}

	// Revoke session
	if err := bm.bridge.Register("revokeSession", bm.revokeSession,
		bridge.WithDescription("Revoke a specific session"),
	); err != nil {
		return fmt.Errorf("failed to register revokeSession: %w", err)
	}

	// Revoke all user sessions
	if err := bm.bridge.Register("revokeAllSessions", bm.revokeAllSessions,
		bridge.WithDescription("Revoke all sessions for a user"),
	); err != nil {
		return fmt.Errorf("failed to register revokeAllSessions: %w", err)
	}

	bm.log.Info("session bridge functions registered")

	return nil
}

// getSessionsList retrieves list of sessions.
func (bm *BridgeManager) getSessionsList(ctx bridge.Context, input SessionsListInput) (*SessionsListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Set defaults
	if input.Page == 0 {
		input.Page = 1
	}

	if input.PageSize == 0 {
		input.PageSize = 20
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Build filter
	filter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  input.Page,
			Limit: input.PageSize,
		},
		AppID: appID,
	}

	// Add user filter if provided
	if input.UserID != "" {
		userID, err := xid.FromString(input.UserID)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
		}

		filter.UserID = &userID
	}

	// List sessions from service
	response, err := bm.sessionSvc.ListSessions(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list sessions", forge.F("error", err.Error()))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch sessions")
	}

	// Transform sessions to SessionItem DTOs
	now := time.Now()

	sessions := make([]SessionItem, 0, len(response.Data))
	for _, s := range response.Data {
		// Look up user email
		userEmail := ""

		if bm.userSvc != nil {
			user, err := bm.userSvc.FindByID(goCtx, s.UserID)
			if err == nil && user != nil {
				userEmail = user.Email
			}
		}

		// Determine if session is active (not expired)
		isActive := s.ExpiresAt.After(now)

		// Apply status filter
		if input.Status != nil && *input.Status != "" && *input.Status != "all" {
			if *input.Status == "active" && !isActive {
				continue
			}

			if *input.Status == "expired" && isActive {
				continue
			}
		}

		// Apply email search filter
		if input.SearchEmail != "" {
			if userEmail == "" || !containsIgnoreCase(userEmail, input.SearchEmail) {
				continue
			}
		}

		// Parse user agent for device info
		device := parseUserAgent(s.UserAgent)

		sessions = append(sessions, SessionItem{
			ID:        s.ID.String(),
			UserID:    s.UserID.String(),
			UserEmail: userEmail,
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			Device:    device,
			Location:  "", // TODO: IP geolocation if needed
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			ExpiresAt: s.ExpiresAt.Format(time.RFC3339),
			LastUsed:  s.CreatedAt.Format(time.RFC3339), // Use CreatedAt as proxy
			IsActive:  isActive,
		})
	}

	// Calculate totals based on filtered results
	totalFiltered := len(sessions)

	totalPages := 0
	if totalFiltered > 0 && input.PageSize > 0 {
		totalPages = (totalFiltered + input.PageSize - 1) / input.PageSize
	}

	return &SessionsListOutput{
		Sessions:   sessions,
		Total:      totalFiltered,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// containsIgnoreCase checks if s contains substr (case insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			indexIgnoreCase(s, substr) >= 0)
}

// indexIgnoreCase returns the index of substr in s (case insensitive).
func indexIgnoreCase(s, substr string) int {
	sLower := ""
	substrLower := ""

	var sLowerSb212 strings.Builder

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			sLowerSb212.WriteRune(r + 32)
		} else {
			sLowerSb212.WriteRune(r)
		}
	}

	sLower += sLowerSb212.String()

	var substrLowerSb219 strings.Builder

	for _, r := range substr {
		if r >= 'A' && r <= 'Z' {
			substrLowerSb219.WriteRune(r + 32)
		} else {
			substrLowerSb219.WriteRune(r)
		}
	}

	substrLower += substrLowerSb219.String()

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return i
		}
	}

	return -1
}

// parseUserAgent extracts device info from user agent string.
func parseUserAgent(ua string) string {
	// Simple parsing - in production, use a proper UA parser library
	if ua == "" {
		return "Unknown Device"
	}

	// Basic detection
	if len(ua) > 50 {
		return ua[:50] + "..."
	}

	return ua
}

// revokeSession revokes a specific session.
func (bm *BridgeManager) revokeSession(ctx bridge.Context, input RevokeSessionInput) (*GenericSuccessOutput, error) {
	if input.SessionID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "sessionId is required")
	}

	// Parse sessionID
	sessionID, err := xid.FromString(input.SessionID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid sessionId")
	}

	goCtx := bm.buildContext(ctx)

	// Revoke session
	err = bm.sessionSvc.RevokeByID(goCtx, sessionID)
	if err != nil {
		bm.log.Error("failed to revoke session", forge.F("error", err.Error()), forge.F("sessionId", input.SessionID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to revoke session")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "session.revoked", "session:"+input.SessionID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Session revoked successfully",
	}, nil
}

// revokeAllSessions revokes all sessions for a user.
func (bm *BridgeManager) revokeAllSessions(ctx bridge.Context, input RevokeAllSessionsInput) (*GenericSuccessOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}

	// Parse userID
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	goCtx := bm.buildContext(ctx)

	// List all sessions for user and revoke them
	sessionFilter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1000, // Get all sessions
		},
		UserID: &userID,
	}

	sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
	if err != nil {
		bm.log.Error("failed to list user sessions", forge.F("error", err.Error()), forge.F("userId", input.UserID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to revoke sessions")
	}

	// Revoke each session
	revokedCount := 0

	if sessionResponse != nil {
		for _, sess := range sessionResponse.Data {
			err := bm.sessionSvc.RevokeByID(goCtx, sess.ID)
			if err == nil {
				revokedCount++
			}
		}
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		metadata, _ := json.Marshal(map[string]int{"count": revokedCount})
		_ = bm.auditSvc.Log(goCtx, &userID, "sessions.revoked_all", "user:"+input.UserID, "", "", string(metadata))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("All sessions revoked successfully (%d sessions)", revokedCount),
	}, nil
}
