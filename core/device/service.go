package device

import (
	"context"
	"github.com/rs/xid"
	"time"
)

// Service manages user devices
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// TrackDevice creates or updates a device record
func (s *Service) TrackDevice(ctx context.Context, userID xid.ID, fingerprint, userAgent, ip string) (*Device, error) {
	now := time.Now().UTC()
	existing, _ := s.repo.FindByFingerprint(ctx, userID, fingerprint)
	if existing != nil {
		existing.UserAgent = userAgent
		existing.IPAddress = ip
		existing.LastActive = now
		existing.UpdatedAt = now
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}
	d := &Device{
		ID:          xid.New(),
		UserID:      userID,
		Fingerprint: fingerprint,
		UserAgent:   userAgent,
		IPAddress:   ip,
		LastActive:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// ListDevices returns devices for a user
func (s *Service) ListDevices(ctx context.Context, userID xid.ID, limit, offset int) ([]*Device, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

// RevokeDevice deletes a device record for a user by fingerprint
func (s *Service) RevokeDevice(ctx context.Context, userID xid.ID, fingerprint string) error {
	return s.repo.DeleteByFingerprint(ctx, userID, fingerprint)
}

// (duplicate removed)
