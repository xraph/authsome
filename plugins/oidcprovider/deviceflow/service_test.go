package deviceflow

import (
	"testing"
	"time"

	"github.com/xraph/authsome/schema"
)

func TestDeviceCode_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-10 * time.Minute),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &schema.DeviceCode{
				ExpiresAt: tt.expiresAt,
			}

			if got := dc.IsExpired(); got != tt.want {
				t.Errorf("DeviceCode.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceCode_IsPending(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "pending and not expired",
			status:    schema.DeviceCodeStatusPending,
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      true,
		},
		{
			name:      "pending but expired",
			status:    schema.DeviceCodeStatusPending,
			expiresAt: time.Now().Add(-10 * time.Minute),
			want:      false,
		},
		{
			name:      "authorized",
			status:    schema.DeviceCodeStatusAuthorized,
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      false,
		},
		{
			name:      "denied",
			status:    schema.DeviceCodeStatusDenied,
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &schema.DeviceCode{
				Status:    tt.status,
				ExpiresAt: tt.expiresAt,
			}

			if got := dc.IsPending(); got != tt.want {
				t.Errorf("DeviceCode.IsPending() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceCode_ShouldSlowDown(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		lastPolledAt *time.Time
		interval     int
		want         bool
	}{
		{
			name:         "no previous poll",
			lastPolledAt: nil,
			interval:     5,
			want:         false,
		},
		{
			name: "polling too fast",
			lastPolledAt: func() *time.Time {
				t := now.Add(-2 * time.Second)

				return &t
			}(),
			interval: 5,
			want:     true,
		},
		{
			name: "polling at correct interval",
			lastPolledAt: func() *time.Time {
				t := now.Add(-6 * time.Second)

				return &t
			}(),
			interval: 5,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &schema.DeviceCode{
				LastPolledAt: tt.lastPolledAt,
				Interval:     tt.interval,
			}

			if got := dc.ShouldSlowDown(); got != tt.want {
				t.Errorf("DeviceCode.ShouldSlowDown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("DefaultConfig().Enabled = false, want true")
	}

	if config.DeviceCodeExpiry != 10*time.Minute {
		t.Errorf("DefaultConfig().DeviceCodeExpiry = %v, want 10m", config.DeviceCodeExpiry)
	}

	if config.UserCodeLength != 8 {
		t.Errorf("DefaultConfig().UserCodeLength = %d, want 8", config.UserCodeLength)
	}

	if config.UserCodeFormat != "XXXX-XXXX" {
		t.Errorf("DefaultConfig().UserCodeFormat = %s, want XXXX-XXXX", config.UserCodeFormat)
	}

	if config.PollingInterval != 5 {
		t.Errorf("DefaultConfig().PollingInterval = %d, want 5", config.PollingInterval)
	}

	if config.VerificationURI != "/device" {
		t.Errorf("DefaultConfig().VerificationURI = %s, want /device", config.VerificationURI)
	}
}
