package device

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Device represents a user device (DTO).
type Device struct {
	ID          xid.ID    `json:"id"`
	UserID      xid.ID    `json:"userId"`
	Fingerprint string    `json:"fingerprint"`
	UserAgent   string    `json:"userAgent"`
	IPAddress   string    `json:"ipAddress"`
	LastActive  time.Time `json:"lastActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ToSchema converts the Device DTO to schema.Device.
func (d *Device) ToSchema() *schema.Device {
	return &schema.Device{
		ID:          d.ID,
		UserID:      d.UserID,
		Fingerprint: d.Fingerprint,
		UserAgent:   d.UserAgent,
		IPAddress:   d.IPAddress,
		LastActive:  d.LastActive,
		AuditableModel: schema.AuditableModel{
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
	}
}

// FromSchemaDevice converts a schema.Device to Device DTO.
func FromSchemaDevice(s *schema.Device) *Device {
	if s == nil {
		return nil
	}

	return &Device{
		ID:          s.ID,
		UserID:      s.UserID,
		Fingerprint: s.Fingerprint,
		UserAgent:   s.UserAgent,
		IPAddress:   s.IPAddress,
		LastActive:  s.LastActive,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// FromSchemaDevices converts multiple schema.Device to Device DTOs.
func FromSchemaDevices(devices []*schema.Device) []*Device {
	result := make([]*Device, len(devices))
	for i, device := range devices {
		result[i] = FromSchemaDevice(device)
	}

	return result
}

// ListDevicesResponse is a type alias for the paginated response.
type ListDevicesResponse = pagination.PageResponse[*Device]
