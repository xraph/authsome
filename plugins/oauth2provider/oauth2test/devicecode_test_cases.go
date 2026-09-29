package oauth2test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store"
)

func newDeviceCode(f Fixture, clientID string) *oauth2provider.DeviceCode {
	return &oauth2provider.DeviceCode{
		ID:              id.NewDeviceCodeID(),
		DeviceCode:      unique("dc"),
		UserCode:        unique("uc"),
		ClientID:        clientID,
		AppID:           f.AppID,
		Scopes:          []string{"openid"},
		VerificationURI: "https://example.test/device",
		ExpiresAt:       now().Add(10 * time.Minute),
		Interval:        5,
		Status:          oauth2provider.DeviceCodeStatusPending,
		CreatedAt:       now(),
	}
}

func seedDeviceCode(t *testing.T, f Fixture) *oauth2provider.DeviceCode {
	t.Helper()
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))
	dc := newDeviceCode(f, c.ClientID)
	require.NoError(t, f.Store.CreateDeviceCode(ctx, dc))
	return dc
}

// testDeviceCodeRoundTrip checks both lookup paths. RFC 8628 needs them to
// agree: the device polls by device_code while the human types the user_code,
// and both must land on the same record.
func testDeviceCodeRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	dc := seedDeviceCode(t, f)

	byDevice, err := f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, dc.ID, byDevice.ID)
	assert.Empty(t, byDevice.UserCode, "the store keeps only a digest of the user code, so a device-code lookup cannot hand it back")
	assert.Equal(t, dc.ClientID, byDevice.ClientID)
	assert.Equal(t, dc.AppID, byDevice.AppID)
	assert.Equal(t, dc.Scopes, byDevice.Scopes)
	assert.Equal(t, dc.Interval, byDevice.Interval)
	assert.Equal(t, oauth2provider.DeviceCodeStatusPending, byDevice.Status)
	assert.WithinDuration(t, dc.ExpiresAt, byDevice.ExpiresAt, time.Second)

	byUser, err := f.Store.GetDeviceCodeByUserCode(ctx, dc.UserCode)
	require.NoError(t, err)
	assert.Equal(t, dc.ID, byUser.ID, "both lookups must resolve to the same record")
	assert.Empty(t, byUser.DeviceCode, "the polling credential stays hidden from the user-code path")

	_, err = f.Store.GetDeviceCodeByDeviceCode(ctx, unique("absent"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, oauth2provider.ErrDeviceCodeNotFound),
		"unknown device code must return ErrDeviceCodeNotFound, got %v", err)

	_, err = f.Store.GetDeviceCodeByUserCode(ctx, unique("absent"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, oauth2provider.ErrDeviceCodeNotFound),
		"unknown user code must return ErrDeviceCodeNotFound, got %v", err)
}

// testDeviceCodeUpdate covers the approval transition. The user id is written
// only at approval time, so a backend that drops it on update hands out a
// token with no subject.
func testDeviceCodeUpdate(t *testing.T, f Fixture) {
	ctx := context.Background()
	dc := seedDeviceCode(t, f)

	dc.Status = oauth2provider.DeviceCodeStatusAuthorized
	dc.UserID = f.UserID
	dc.LastPolledAt = now()
	require.NoError(t, f.Store.UpdateDeviceCode(ctx, dc))

	got, err := f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, oauth2provider.DeviceCodeStatusAuthorized, got.Status)
	assert.Equal(t, f.UserID, got.UserID, "the approving user must persist through update")

	// And the terminal transition, which is what stops a second redemption.
	got.Status = oauth2provider.DeviceCodeStatusConsumed
	require.NoError(t, f.Store.UpdateDeviceCode(ctx, got))
	again, err := f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, oauth2provider.DeviceCodeStatusConsumed, again.Status)
}

// testDeleteExpiredDeviceCodes checks the sweep removes what has expired and
// leaves what has not. A sweep with an inverted comparison deletes live codes,
// which looks like random logout rather than a bug.
func testDeleteExpiredDeviceCodes(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))

	expired := newDeviceCode(f, c.ClientID)
	expired.ExpiresAt = now().Add(-time.Hour)
	require.NoError(t, f.Store.CreateDeviceCode(ctx, expired))

	live := newDeviceCode(f, c.ClientID)
	live.ExpiresAt = now().Add(time.Hour)
	require.NoError(t, f.Store.CreateDeviceCode(ctx, live))

	require.NoError(t, f.Store.DeleteExpiredDeviceCodes(ctx))

	_, err := f.Store.GetDeviceCodeByDeviceCode(ctx, expired.DeviceCode)
	require.Error(t, err, "an expired device code must be swept")
	assert.True(t, errors.Is(err, oauth2provider.ErrDeviceCodeNotFound), "got %v", err)

	_, err = f.Store.GetDeviceCodeByDeviceCode(ctx, live.DeviceCode)
	assert.NoError(t, err, "the sweep must not delete a device code that is still valid")
}

// testDeviceCodeStoredAsHash proves both device codes are kept as digests,
// that each lookup hands back the plaintext it was given, and that the update
// a poll performs never disturbs either digest.
func testDeviceCodeStoredAsHash(t *testing.T, f Fixture) {
	ctx := context.Background()
	dc := seedDeviceCode(t, f)

	_, err := f.Store.GetDeviceCodeByDeviceCode(ctx, store.HashToken(dc.DeviceCode))
	require.ErrorIs(t, err, oauth2provider.ErrDeviceCodeNotFound, "the stored digest must not act as the device code")
	_, err = f.Store.GetDeviceCodeByUserCode(ctx, store.HashToken(dc.UserCode))
	require.ErrorIs(t, err, oauth2provider.ErrDeviceCodeNotFound, "the stored digest must not act as the user code")

	byDevice, err := f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, dc.DeviceCode, byDevice.DeviceCode, "the presented device code rides back")

	// The poll path: look up by device code, record the poll, then the human
	// authorizes by user code. Both lookups must still resolve afterwards.
	byDevice.LastPolledAt = now()
	require.NoError(t, f.Store.UpdateDeviceCode(ctx, byDevice))
	byUser, err := f.Store.GetDeviceCodeByUserCode(ctx, dc.UserCode)
	require.NoError(t, err, "an update after a device-code lookup must not disturb the user-code digest")
	assert.Equal(t, dc.UserCode, byUser.UserCode, "the presented user code rides back")
	byUser.Status = oauth2provider.DeviceCodeStatusAuthorized
	byUser.UserID = f.UserID
	require.NoError(t, f.Store.UpdateDeviceCode(ctx, byUser))
	again, err := f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
	require.NoError(t, err, "an update after a user-code lookup must not disturb the device-code digest")
	assert.Equal(t, oauth2provider.DeviceCodeStatusAuthorized, again.Status)
	assert.Equal(t, f.UserID, again.UserID)
}
