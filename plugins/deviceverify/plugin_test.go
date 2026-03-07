package deviceverify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "deviceverify", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New()

	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)

	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)

	_, ok = p.(plugin.AfterSignIn)
	assert.True(t, ok)
}

func TestNewDevice_Notifies(t *testing.T) {
	s := memory.New()
	p := New(Config{NotifyOnNewDevice: true, ChallengeTTL: 10 * time.Minute})
	p.store = s
	p.logger = log.NewNoopLogger()

	ctx := context.Background()
	devID := id.NewDeviceID()
	userID := id.NewUserID()
	appID, _ := id.ParseAppID("aapp_01jf0000000000000000000000")

	// Create a device that was just created (within 30 seconds).
	dev := &device.Device{
		ID:          devID,
		UserID:      userID,
		AppID:       appID,
		Name:        "Test Device",
		Type:        "desktop",
		Browser:     "Chrome",
		OS:          "macOS",
		Fingerprint: "fp123",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := s.CreateDevice(ctx, dev)
	require.NoError(t, err)

	u := &user.User{ID: userID, AppID: appID, Email: "test@example.com", FirstName: "Test"}
	sess := &session.Session{DeviceID: devID, IPAddress: "1.2.3.4"}

	// Should not error - notification attempted but herald is nil (no-op).
	err = p.OnAfterSignIn(ctx, u, sess)
	assert.NoError(t, err)
}

func TestExistingDevice_NoAction(t *testing.T) {
	s := memory.New()
	p := New(Config{NotifyOnNewDevice: true})
	p.store = s
	p.logger = log.NewNoopLogger()

	ctx := context.Background()
	devID := id.NewDeviceID()
	userID := id.NewUserID()
	appID, _ := id.ParseAppID("aapp_01jf0000000000000000000000")

	// Create a device that was created more than 30 seconds ago.
	dev := &device.Device{
		ID:          devID,
		UserID:      userID,
		AppID:       appID,
		Fingerprint: "fp456",
		CreatedAt:   time.Now().Add(-5 * time.Minute),
		UpdatedAt:   time.Now(),
	}
	err := s.CreateDevice(ctx, dev)
	require.NoError(t, err)

	u := &user.User{ID: userID, AppID: appID, Email: "test@example.com"}
	sess := &session.Session{DeviceID: devID}

	err = p.OnAfterSignIn(ctx, u, sess)
	assert.NoError(t, err)
}

func TestNoDeviceID_NoAction(t *testing.T) {
	s := memory.New()
	p := New()
	p.store = s
	p.logger = log.NewNoopLogger()

	u := &user.User{ID: id.NewUserID()}
	sess := &session.Session{}

	err := p.OnAfterSignIn(context.Background(), u, sess)
	assert.NoError(t, err)
}

func TestNoStore_NoOp(t *testing.T) {
	p := New()
	p.logger = log.NewNoopLogger()

	u := &user.User{ID: id.NewUserID()}
	devID := id.NewDeviceID()
	sess := &session.Session{DeviceID: devID}

	err := p.OnAfterSignIn(context.Background(), u, sess)
	assert.NoError(t, err)
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	require.NoError(t, err)
	assert.Len(t, token, 64) // 32 bytes = 64 hex chars

	// Generate another token - should be different.
	token2, err := GenerateToken()
	require.NoError(t, err)
	assert.NotEqual(t, token, token2)
}
