package device

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

func TestDevice_FieldsPopulated(t *testing.T) {
	d := &Device{
		ID:          id.NewDeviceID(),
		UserID:      id.NewUserID(),
		AppID:       id.NewAppID(),
		Name:        "Chrome on Mac",
		Type:        "desktop",
		Browser:     "Chrome",
		OS:          "macOS",
		IPAddress:   "192.168.1.1",
		Fingerprint: "abc123",
		Trusted:     false,
		LastSeenAt:  time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.NotEmpty(t, d.ID.String())
	assert.NotEmpty(t, d.UserID.String())
	assert.NotEmpty(t, d.AppID.String())
	assert.Equal(t, "Chrome on Mac", d.Name)
	assert.Equal(t, "desktop", d.Type)
	assert.Equal(t, "Chrome", d.Browser)
	assert.Equal(t, "macOS", d.OS)
	assert.Equal(t, "192.168.1.1", d.IPAddress)
	assert.Equal(t, "abc123", d.Fingerprint)
	assert.False(t, d.Trusted)
}

func TestDevice_TrustedField(t *testing.T) {
	d := &Device{
		ID:      id.NewDeviceID(),
		Trusted: true,
	}
	assert.True(t, d.Trusted)
}

func TestDevice_ZeroValueFingerprint(t *testing.T) {
	d := &Device{ID: id.NewDeviceID()}
	assert.Empty(t, d.Fingerprint)
}
