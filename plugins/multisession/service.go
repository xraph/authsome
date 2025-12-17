package multisession

import (
	"context"
	"errors"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
)

// Service provides multi-session operations
type Service struct {
	sessions   session.Repository
	sessionSvc session.ServiceInterface
	devices    dev.Repository
	auth       *auth.Service
}

func NewService(sr session.Repository, sessionSvc session.ServiceInterface, dr dev.Repository, a *auth.Service, _ interface{}) *Service {
	return &Service{sessions: sr, devices: dr, auth: a, sessionSvc: sessionSvc}
}

// CurrentUserFromToken validates token and returns userID
func (s *Service) CurrentUserFromToken(ctx context.Context, token string) (xid.ID, error) {
	res, err := s.auth.GetSession(ctx, token)
	if err != nil || res == nil || res.Session == nil {
		return xid.ID{}, errors.New("not authenticated")
	}
	return res.User.ID, nil
}

// List returns all sessions for a user
func (s *Service) List(ctx context.Context, userID xid.ID) (*session.ListSessionsResponse, error) {
	// Default pagination: return up to 100 sessions
	return s.sessionSvc.ListSessions(ctx, &session.ListSessionsFilter{
		UserID: &userID,
		PaginationParams: pagination.PaginationParams{
			Limit:  100,
			Offset: 0,
		},
	})
}

// Find returns a specific session by ID ensuring ownership
func (s *Service) Find(ctx context.Context, userID xid.ID, id xid.ID) (*session.Session, error) {
	sess, err := s.sessionSvc.FindByID(ctx, id)
	if err != nil || sess == nil {
		return nil, errors.New("session not found")
	}
	if sess.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	return sess, nil
}

// Delete revokes a session by id ensuring ownership
func (s *Service) Delete(ctx context.Context, userID, id xid.ID) error {
	// Ensure session belongs to user
	sess, err := s.sessionSvc.FindByID(ctx, id)
	if err != nil || sess == nil {
		return errors.New("session not found")
	}
	if sess.UserID != userID {
		return errors.New("unauthorized")
	}
	return s.sessionSvc.RevokeByID(ctx, id)
}

// GetCurrentSessionID extracts the session ID from a session token.
// It validates the token and returns the associated session ID.
// Returns an error if the token is invalid or expired.
func (s *Service) GetCurrentSessionID(ctx context.Context, token string) (xid.ID, error) {
	res, err := s.auth.GetSession(ctx, token)
	if err != nil || res == nil || res.Session == nil {
		return xid.ID{}, errors.New("invalid token")
	}
	return res.Session.ID, nil
}

// GetCurrent returns the current session by ID with ownership verification.
// This is a convenience method that wraps Find to retrieve the active session.
// Returns an error if the session doesn't exist or doesn't belong to the user.
func (s *Service) GetCurrent(ctx context.Context, userID, sessionID xid.ID) (*session.Session, error) {
	return s.Find(ctx, userID, sessionID)
}

// RevokeAll revokes all sessions for a user with optional current session inclusion.
// If includeCurrentSession is false, the current session specified by currentSessionID is preserved.
// Returns the count of successfully revoked sessions and any error encountered.
// Use case: Sign out from all devices, or sign out everywhere except current device.
func (s *Service) RevokeAll(ctx context.Context, userID xid.ID, includeCurrentSession bool, currentSessionID xid.ID) (int, error) {
	// Get all sessions for user
	listResp, err := s.List(ctx, userID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, sess := range listResp.Data {
		// Skip current session if requested
		if !includeCurrentSession && sess.ID == currentSessionID {
			continue
		}

		// Revoke the session
		if err := s.sessionSvc.RevokeByID(ctx, sess.ID); err != nil {
			// Log but continue with other sessions
			continue
		}
		count++
	}

	return count, nil
}

// RevokeAllExceptCurrent revokes all sessions except the current one.
// This is commonly used after password changes or when suspicious activity is detected
// to ensure security while keeping the user logged in on their current device.
// Returns the count of successfully revoked sessions and any error encountered.
func (s *Service) RevokeAllExceptCurrent(ctx context.Context, userID, currentSessionID xid.ID) (int, error) {
	return s.RevokeAll(ctx, userID, false, currentSessionID)
}

// RefreshCurrent extends the current session's expiry time using the sliding session pattern.
// This updates the session's expiration timestamp to prevent automatic logout during active use.
// Returns the updated session with the new expiry time or an error if the refresh fails.
func (s *Service) RefreshCurrent(ctx context.Context, userID, sessionID xid.ID) (*session.Session, error) {
	// Find session and verify ownership
	sess, err := s.Find(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	// Use TouchSession to extend the session
	updatedSess, _, err := s.sessionSvc.TouchSession(ctx, sess)
	if err != nil {
		return nil, err
	}

	return updatedSess, nil
}

// SessionStats holds aggregated session statistics for a user.
// Provides an overview of the user's session landscape including counts,
// unique devices, unique locations (based on IP addresses), and session age range.
type SessionStats struct {
	TotalSessions  int              // Total number of sessions (active + expired)
	ActiveSessions int              // Number of currently active (non-expired) sessions
	DeviceCount    int              // Number of unique devices
	LocationCount  int              // Number of unique IP addresses (proxy for locations)
	OldestSession  *session.Session // Oldest session by creation time
	NewestSession  *session.Session // Newest session by creation time
}

// GetStats returns aggregated session statistics for a user.
// Calculates total and active session counts, unique device and location counts,
// and identifies the oldest and newest sessions. Useful for security dashboards
// and user account management interfaces.
// Returns SessionStats containing all aggregated data or an error if retrieval fails.
func (s *Service) GetStats(ctx context.Context, userID xid.ID) (*SessionStats, error) {
	// Get all sessions for user
	listResp, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := &SessionStats{
		TotalSessions: len(listResp.Data),
	}

	if stats.TotalSessions == 0 {
		return stats, nil
	}

	// Track unique locations (IP addresses) and user agents (proxy for devices)
	ipAddresses := make(map[string]bool)
	userAgents := make(map[string]bool)

	for i, sess := range listResp.Data {
		// Count active sessions (not expired)
		if IsSessionActive(sess.ExpiresAt) {
			stats.ActiveSessions++
		}

		// Track oldest and newest sessions
		if stats.OldestSession == nil || sess.CreatedAt.Before(stats.OldestSession.CreatedAt) {
			stats.OldestSession = listResp.Data[i]
		}
		if stats.NewestSession == nil || sess.CreatedAt.After(stats.NewestSession.CreatedAt) {
			stats.NewestSession = listResp.Data[i]
		}

		// Track unique user agents as proxy for device count
		if sess.UserAgent != "" {
			userAgents[sess.UserAgent] = true
		}

		// Track unique IP addresses as proxy for location
		if sess.IPAddress != "" {
			ipAddresses[sess.IPAddress] = true
		}
	}

	stats.DeviceCount = len(userAgents)
	stats.LocationCount = len(ipAddresses)

	return stats, nil
}
