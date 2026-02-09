package twofa

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// MarkTrustedDevice marks a device as trusted for a specified number of days.
func (s *Service) MarkTrustedDevice(ctx context.Context, userID, deviceID string, days int) error {
	if days <= 0 || days > 90 {
		days = 30 // Default to 30 days, max 90 days
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if deviceID == "" {
		return errs.New(errs.CodeInvalidInput, "device ID is required", http.StatusBadRequest)
	}

	// Check if device is already trusted
	var existing schema.TrustedDevice

	err = s.repo.DB().NewSelect().
		Model(&existing).
		Where("user_id = ? AND device_id = ?", uid, deviceID).
		Scan(ctx)

	expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	if err != nil {
		// Device not found, create new trust record
		trustedDevice := &schema.TrustedDevice{
			ID:        xid.New(),
			UserID:    uid,
			DeviceID:  deviceID,
			ExpiresAt: expiresAt,
		}
		trustedDevice.CreatedBy = uid
		trustedDevice.UpdatedBy = uid

		_, err = s.repo.DB().NewInsert().Model(trustedDevice).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to mark device as trusted: %w", err)
		}
	} else {
		// Update existing trust record
		existing.ExpiresAt = expiresAt
		existing.UpdatedBy = uid
		existing.UpdatedAt = time.Now().UTC()

		_, err = s.repo.DB().NewUpdate().
			Model(&existing).
			Column("expires_at", "updated_at", "updated_by").
			Where("id = ?", existing.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update trusted device: %w", err)
		}
	}

	return nil
}

// IsTrustedDevice checks if a device is currently trusted (not expired).
func (s *Service) IsTrustedDevice(ctx context.Context, userID, deviceID string) bool {
	uid, err := xid.FromString(userID)
	if err != nil {
		return false
	}

	if deviceID == "" {
		return false
	}

	var trustedDevice schema.TrustedDevice

	err = s.repo.DB().NewSelect().
		Model(&trustedDevice).
		Where("user_id = ? AND device_id = ? AND expires_at > ?", uid, deviceID, time.Now()).
		Scan(ctx)

	return err == nil // Trusted if found and not expired
}

// RemoveTrustedDevice removes trust for a specific device.
func (s *Service) RemoveTrustedDevice(ctx context.Context, userID, deviceID string) error {
	uid, err := xid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = s.repo.DB().NewDelete().
		Model((*schema.TrustedDevice)(nil)).
		Where("user_id = ? AND device_id = ?", uid, deviceID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove trusted device: %w", err)
	}

	return nil
}

// ListTrustedDevices returns all trusted devices for a user.
func (s *Service) ListTrustedDevices(ctx context.Context, userID string) ([]schema.TrustedDevice, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var devices []schema.TrustedDevice

	err = s.repo.DB().NewSelect().
		Model(&devices).
		Where("user_id = ? AND expires_at > ?", uid, time.Now()).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list trusted devices: %w", err)
	}

	return devices, nil
}

// CleanupExpiredDevices removes expired trusted device records.
func (s *Service) CleanupExpiredDevices(ctx context.Context) error {
	_, err := s.repo.DB().NewDelete().
		Model((*schema.TrustedDevice)(nil)).
		Where("expires_at <= ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired devices: %w", err)
	}

	return nil
}
