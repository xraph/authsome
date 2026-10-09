package oauth2test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

// testDeleteExpiredAuthCodes proves the batched sweep removes codes that
// expired before the cutoff, no more per call than asked, and leaves live
// codes alone.
func testDeleteExpiredAuthCodes(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))
	cutoff := now().Add(-time.Hour)

	expired := make([]*oauth2provider.AuthorizationCode, 3)
	for i := range expired {
		expired[i] = newAuthCode(f, c.ClientID)
		expired[i].ExpiresAt = cutoff.Add(-time.Minute)
		require.NoError(t, f.Store.CreateAuthCode(ctx, expired[i]))
	}
	live := newAuthCode(f, c.ClientID)
	require.NoError(t, f.Store.CreateAuthCode(ctx, live))

	n, err := f.Store.DeleteExpiredAuthCodes(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n, "a batch of two removes two")
	n, err = f.Store.DeleteExpiredAuthCodes(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "the next batch removes what is left")
	n, err = f.Store.DeleteExpiredAuthCodes(ctx, cutoff, 0)
	require.NoError(t, err)
	assert.Zero(t, n, "nothing expired before the cutoff remains")

	for _, code := range expired {
		_, err = f.Store.GetAuthCode(ctx, code.Code)
		// The memory store says ErrCodeNotFound; the SQL and mongo stores map
		// a missing row to ErrClientNotFound, as the other code cases accept.
		assert.Error(t, err, "an expired code must be gone")
	}
	_, err = f.Store.GetAuthCode(ctx, live.Code)
	assert.NoError(t, err, "a live code survives the sweep")
}

// testDeleteExpiredDeviceCodesBefore is the device-code twin of the auth
// code sweep.
func testDeleteExpiredDeviceCodesBefore(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))
	cutoff := now().Add(-time.Hour)

	expired := make([]*oauth2provider.DeviceCode, 3)
	for i := range expired {
		expired[i] = newDeviceCode(f, c.ClientID)
		expired[i].ExpiresAt = cutoff.Add(-time.Minute)
		require.NoError(t, f.Store.CreateDeviceCode(ctx, expired[i]))
	}
	live := newDeviceCode(f, c.ClientID)
	require.NoError(t, f.Store.CreateDeviceCode(ctx, live))

	n, err := f.Store.DeleteExpiredDeviceCodesBefore(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
	n, err = f.Store.DeleteExpiredDeviceCodesBefore(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	n, err = f.Store.DeleteExpiredDeviceCodesBefore(ctx, cutoff, 0)
	require.NoError(t, err)
	assert.Zero(t, n)

	for _, dc := range expired {
		_, err = f.Store.GetDeviceCodeByDeviceCode(ctx, dc.DeviceCode)
		assert.Error(t, err, "an expired device code must be gone")
	}
	_, err = f.Store.GetDeviceCodeByDeviceCode(ctx, live.DeviceCode)
	assert.NoError(t, err, "a live device code survives the sweep")
}
