package device

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Service manages user devices
type Service struct {
	repo         Repository
	hookRegistry interface{} // Hook registry for lifecycle events
}

// NewService creates a new device service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetHookRegistry sets the hook registry for executing lifecycle hooks
func (s *Service) SetHookRegistry(registry interface{}) {
	s.hookRegistry = registry
}

// TrackDevice creates or updates a device record
func (s *Service) TrackDevice(ctx context.Context, appID, userID xid.ID, fingerprint, userAgent, ip string) (*Device, error) {
	// Validate fingerprint
	if fingerprint == "" {
		return nil, InvalidFingerprint()
	}

	now := time.Now().UTC()

	// Try to find existing device
	existingSchema, err := s.repo.FindDeviceByFingerprint(ctx, userID, fingerprint)
	if err == nil && existingSchema != nil {
		// Update existing device
		existingSchema.UserAgent = userAgent
		existingSchema.IPAddress = ip
		existingSchema.LastActive = now
		existingSchema.UpdatedAt = now

		if err := s.repo.UpdateDevice(ctx, existingSchema); err != nil {
			return nil, DeviceUpdateFailed(err)
		}

		return FromSchemaDevice(existingSchema), nil
	}

	// Create new device
	deviceSchema := &schema.Device{
		ID:          xid.New(),
		AppID:       appID,
		UserID:      userID,
		Fingerprint: fingerprint,
		UserAgent:   userAgent,
		IPAddress:   ip,
		LastActive:  now,
		AuditableModel: schema.AuditableModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if err := s.repo.CreateDevice(ctx, deviceSchema); err != nil {
		return nil, DeviceCreationFailed(err)
	}

	device := FromSchemaDevice(deviceSchema)

	// Execute new device detected hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteOnNewDeviceDetected(context.Context, xid.ID, string, string, string) error
		}); ok {
			// Extract device name from user agent (simplified)
			deviceName := userAgent
			if len(deviceName) > 50 {
				deviceName = deviceName[:50] + "..."
			}
			// Location would come from IP geolocation service (not implemented)
			location := "Unknown location"
			_ = registry.ExecuteOnNewDeviceDetected(ctx, userID, deviceName, location, ip)
		}
	}

	return device, nil
}

// ListDevices returns devices for a user with pagination
func (s *Service) ListDevices(ctx context.Context, filter *ListDevicesFilter) (*ListDevicesResponse, error) {
	// Get paginated results from repository (returns schema types)
	pageResp, err := s.repo.ListDevices(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema types to DTOs
	dtoDevices := FromSchemaDevices(pageResp.Data)

	// Return paginated response with DTOs
	return &ListDevicesResponse{
		Data:       dtoDevices,
		Pagination: pageResp.Pagination,
	}, nil
}

// GetDevice retrieves a device by ID
func (s *Service) GetDevice(ctx context.Context, id xid.ID) (*Device, error) {
	deviceSchema, err := s.repo.FindDeviceByID(ctx, id)
	if err != nil {
		return nil, DeviceNotFound()
	}

	return FromSchemaDevice(deviceSchema), nil
}

// GetDeviceByFingerprint retrieves a device by user ID and fingerprint
func (s *Service) GetDeviceByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) (*Device, error) {
	if fingerprint == "" {
		return nil, InvalidFingerprint()
	}

	deviceSchema, err := s.repo.FindDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil {
		return nil, DeviceNotFound()
	}

	return FromSchemaDevice(deviceSchema), nil
}

// RevokeDevice deletes a device record for a user by fingerprint
func (s *Service) RevokeDevice(ctx context.Context, userID xid.ID, fingerprint string) error {
	if fingerprint == "" {
		return InvalidFingerprint()
	}

	if err := s.repo.DeleteDeviceByFingerprint(ctx, userID, fingerprint); err != nil {
		return DeviceDeletionFailed(err)
	}

	return nil
}

// RevokeDeviceByID deletes a device record by ID
func (s *Service) RevokeDeviceByID(ctx context.Context, id xid.ID) error {
	// Get device details before deletion for hook
	deviceSchema, err := s.repo.FindDeviceByID(ctx, id)
	if err != nil {
		return DeviceNotFound()
	}

	userID := deviceSchema.UserID
	deviceName := deviceSchema.UserAgent
	if len(deviceName) > 50 {
		deviceName = deviceName[:50] + "..."
	}

	if err := s.repo.DeleteDevice(ctx, id); err != nil {
		return DeviceDeletionFailed(err)
	}

	// Execute device removed hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteOnDeviceRemoved(context.Context, xid.ID, string) error
		}); ok {
			_ = registry.ExecuteOnDeviceRemoved(ctx, userID, deviceName)
		}
	}

	return nil
}

// CountUserDevices returns the count of devices for a user
func (s *Service) CountUserDevices(ctx context.Context, userID xid.ID) (int, error) {
	return s.repo.CountDevices(ctx, userID)
}
