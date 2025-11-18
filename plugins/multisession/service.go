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
