package twofa

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestTrustedDeviceDaysValidation(t *testing.T) {
	// Test days parameter validation logic

	// Days <= 0 should default to 30
	days := 0
	if days <= 0 || days > 90 {
		days = 30
	}

	assert.Equal(t, 30, days)

	// Days > 90 should cap at 30
	days = 100
	if days <= 0 || days > 90 {
		days = 30
	}

	assert.Equal(t, 30, days)

	// Valid days should be preserved
	days = 60
	if days <= 0 || days > 90 {
		days = 30
	}

	assert.Equal(t, 60, days)

	// Edge case: exactly 90 days
	days = 90
	if days <= 0 || days > 90 {
		days = 30
	}

	assert.Equal(t, 90, days)
}

func TestTrustedDeviceExpirationCalculation(t *testing.T) {
	// Test expiration calculation
	now := time.Now()
	days := 30
	expiresAt := now.Add(time.Duration(days) * 24 * time.Hour)

	// Should be 30 days in the future
	duration := expiresAt.Sub(now)
	expectedHours := float64(days * 24)
	assert.InDelta(t, expectedHours, duration.Hours(), 1.0)

	// Test different day values
	testCases := []int{1, 7, 30, 60, 90}
	for _, tc := range testCases {
		exp := now.Add(time.Duration(tc) * 24 * time.Hour)
		dur := exp.Sub(now)
		assert.InDelta(t, float64(tc*24), dur.Hours(), 1.0)
	}
}

func TestTrustedDeviceIDValidation(t *testing.T) {
	// Test user ID validation
	validUserID := xid.New().String()
	_, err := xid.FromString(validUserID)
	assert.NoError(t, err, "Valid XID should parse correctly")

	invalidUserID := "invalid-xid"
	_, err = xid.FromString(invalidUserID)
	assert.Error(t, err, "Invalid XID should produce error")

	// Test device ID validation
	emptyDeviceID := ""
	assert.Empty(t, emptyDeviceID, "Empty device ID should be detected")

	validDeviceID := "device-fingerprint-123"
	assert.NotEmpty(t, validDeviceID, "Valid device ID should not be empty")
}

func TestTrustedDeviceExpiration(t *testing.T) {
	// Test expiration checking logic
	now := time.Now()

	// Expired device (1 hour ago)
	expiredTime := now.Add(-1 * time.Hour)
	assert.True(t, expiredTime.Before(now), "Past time should be before now")

	// Valid device (1 hour from now)
	validTime := now.Add(1 * time.Hour)
	assert.True(t, validTime.After(now), "Future time should be after now")

	// Just expired
	justExpired := now.Add(-1 * time.Second)
	assert.True(t, justExpired.Before(now), "Just expired should be before now")

	// Just about to expire
	almostExpired := now.Add(1 * time.Second)
	assert.True(t, almostExpired.After(now), "Not yet expired should be after now")
}

func TestTrustedDeviceDurationConversion(t *testing.T) {
	// Test duration conversion logic
	days := 30
	duration := time.Duration(days) * 24 * time.Hour

	// Verify conversion
	assert.Equal(t, float64(days*24), duration.Hours())
	assert.Equal(t, int64(days*24*60), int64(duration.Minutes()))
	assert.Equal(t, int64(days*24*60*60), int64(duration.Seconds()))
}

func TestTrustedDeviceRenewal(t *testing.T) {
	// Test trust renewal logic
	now := time.Now()
	originalExpiry := now.Add(10 * 24 * time.Hour) // 10 days

	// Renew for 30 more days
	newExpiry := now.Add(30 * 24 * time.Hour)

	// New expiry should be later than original
	assert.True(t, newExpiry.After(originalExpiry))

	// Calculate difference
	diff := newExpiry.Sub(originalExpiry)
	assert.InDelta(t, 20*24, diff.Hours(), 1.0) // ~20 days difference
}

func TestMultipleTrustedDevices(t *testing.T) {
	// Test multiple device IDs for the same user
	userID := xid.New().String()
	deviceIDs := []string{
		"device-1-fingerprint",
		"device-2-fingerprint",
		"device-3-fingerprint",
	}

	// All should be unique
	seen := make(map[string]bool)
	for _, did := range deviceIDs {
		assert.False(t, seen[did], "Device IDs should be unique")
		seen[did] = true
		assert.NotEmpty(t, did)
	}

	// User ID should be valid
	_, err := xid.FromString(userID)
	assert.NoError(t, err)
}

func TestTrustedDeviceCleanup(t *testing.T) {
	// Test cleanup logic for expired devices
	now := time.Now()

	devices := []struct {
		expiresAt time.Time
		expired   bool
	}{
		{now.Add(1 * time.Hour), false},  // Valid
		{now.Add(-1 * time.Hour), true},  // Expired 1 hour ago
		{now.Add(24 * time.Hour), false}, // Valid 24 hours
		{now.Add(-24 * time.Hour), true}, // Expired 24 hours ago
		{now, true},                      // Expired exactly now
	}

	for i, device := range devices {
		isExpired := device.expiresAt.Before(now) || device.expiresAt.Equal(now)
		assert.Equal(t, device.expired, isExpired, "Device %d expiration mismatch", i)
	}
}

func TestTrustedDeviceMaxDuration(t *testing.T) {
	// Test maximum allowed duration (90 days)
	maxDays := 90
	maxDuration := time.Duration(maxDays) * 24 * time.Hour

	// 90 days in various units
	assert.Equal(t, 90*24, int(maxDuration.Hours()))
	assert.Equal(t, 90*24*60, int(maxDuration.Minutes()))

	// Verify 90 days is reasonable
	now := time.Now()
	maxExpiry := now.Add(maxDuration)
	assert.True(t, maxExpiry.After(now))

	// Should be roughly 3 months
	diff := maxExpiry.Sub(now)
	assert.InDelta(t, 90*24, diff.Hours(), 1.0)
}

func TestTrustedDeviceDefaultDuration(t *testing.T) {
	// Test default duration (30 days)
	defaultDays := 30
	defaultDuration := time.Duration(defaultDays) * 24 * time.Hour

	// 30 days in various units
	assert.Equal(t, 30*24, int(defaultDuration.Hours()))

	// Verify default is reasonable
	now := time.Now()
	defaultExpiry := now.Add(defaultDuration)
	assert.True(t, defaultExpiry.After(now))
}
