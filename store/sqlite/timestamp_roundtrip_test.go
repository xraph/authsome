//go:build integration

package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// refTime is a fixed, whole-second UTC instant used across the round-trip
// tests. Whole seconds avoid sub-second precision loss in the RFC3339
// serialization the sqlite driver uses, so the read-back instant should be
// exactly equal to what was written.
var refTime = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

// assertSameInstant fails if got is not the same instant as want (compared
// regardless of monotonic clock or location).
func assertSameInstant(t *testing.T, want, got time.Time, field string) {
	t.Helper()
	assert.Falsef(t, got.IsZero(), "%s round-tripped to the zero time", field)
	assert.WithinDurationf(t, want, got, time.Second, "%s did not round-trip", field)
}

// Each test below creates a row with non-zero timestamps through the store's
// public API, then reads it back. Before the TEXT→TIMESTAMP migration the
// read fails with "unsupported Scan, storing driver.Value type string into
// type *time.Time"; after it, the timestamps survive the round-trip.

func TestTimestampRoundTrip_Apps(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	a := &app.App{
		ID:        id.NewAppID(),
		Name:      "Acme",
		Slug:      "acme",
		CreatedAt: refTime,
		UpdatedAt: refTime,
	}
	require.NoError(t, s.CreateApp(ctx, a))

	got, err := s.GetApp(ctx, a.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
}

func TestTimestampRoundTrip_Users(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	ban := refTime.Add(48 * time.Hour)
	u := &user.User{
		ID:         id.NewUserID(),
		AppID:      id.NewAppID(),
		EnvID:      id.NewEnvironmentID(),
		Email:      "user@example.com",
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Banned:     true,
		BanReason:  "testing",
		BanExpires: &ban,
		CreatedAt:  refTime,
		UpdatedAt:  refTime,
	}
	require.NoError(t, s.CreateUser(ctx, u))

	got, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
	require.NotNil(t, got.BanExpires, "nullable ban_expires lost")
	assertSameInstant(t, ban, *got.BanExpires, "ban_expires")
}

func TestTimestampRoundTrip_Sessions(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	exp := refTime.Add(time.Hour)
	rexp := refTime.Add(30 * 24 * time.Hour)
	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 id.NewAppID(),
		EnvID:                 id.NewEnvironmentID(),
		UserID:                id.NewUserID(),
		Token:                 "tok_roundtrip",
		RefreshToken:          "rtk_roundtrip",
		ExpiresAt:             exp,
		RefreshTokenExpiresAt: rexp,
		CreatedAt:             refTime,
		UpdatedAt:             refTime,
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	got, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assertSameInstant(t, exp, got.ExpiresAt, "expires_at")
	assertSameInstant(t, rexp, got.RefreshTokenExpiresAt, "refresh_token_expires_at")
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
}

func TestTimestampRoundTrip_Organizations(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     id.NewAppID(),
		EnvID:     id.NewEnvironmentID(),
		Name:      "Widgets Inc",
		Slug:      "widgets",
		CreatedBy: id.NewUserID(),
		CreatedAt: refTime,
		UpdatedAt: refTime,
	}
	require.NoError(t, s.CreateOrganization(ctx, o))

	got, err := s.GetOrganization(ctx, o.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
}

func TestTimestampRoundTrip_APIKeys(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	exp := refTime.Add(90 * 24 * time.Hour)
	used := refTime.Add(time.Hour)
	k := &apikey.APIKey{
		ID:         id.NewAPIKeyID(),
		AppID:      id.NewAppID(),
		EnvID:      id.NewEnvironmentID(),
		UserID:     id.NewUserID(),
		Name:       "ci-key",
		KeyHash:    "hash_roundtrip",
		KeyPrefix:  "ak_live_rt",
		ExpiresAt:  &exp,
		LastUsedAt: &used,
		CreatedAt:  refTime,
		UpdatedAt:  refTime,
	}
	require.NoError(t, s.CreateAPIKey(ctx, k))

	got, err := s.GetAPIKey(ctx, k.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
	require.NotNil(t, got.ExpiresAt, "nullable expires_at lost")
	assertSameInstant(t, exp, *got.ExpiresAt, "expires_at")
	require.NotNil(t, got.LastUsedAt, "nullable last_used_at lost")
	assertSameInstant(t, used, *got.LastUsedAt, "last_used_at")
}

func TestTimestampRoundTrip_Devices(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	seen := refTime.Add(5 * time.Minute)
	d := &device.Device{
		ID:          id.NewDeviceID(),
		UserID:      id.NewUserID(),
		AppID:       id.NewAppID(),
		EnvID:       id.NewEnvironmentID(),
		Name:        "Pixel",
		Fingerprint: "fp_roundtrip",
		LastSeenAt:  seen,
		CreatedAt:   refTime,
		UpdatedAt:   refTime,
	}
	require.NoError(t, s.CreateDevice(ctx, d))

	got, err := s.GetDevice(ctx, d.ID)
	require.NoError(t, err)
	assertSameInstant(t, seen, got.LastSeenAt, "last_seen_at")
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
}

func TestTimestampRoundTrip_Verifications(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	exp := refTime.Add(15 * time.Minute)
	v := &account.Verification{
		ID:        id.NewVerificationID(),
		AppID:     id.NewAppID(),
		EnvID:     id.NewEnvironmentID(),
		UserID:    id.NewUserID(),
		Token:     "vtok_roundtrip",
		Type:      account.VerificationType("email_verification"),
		ExpiresAt: exp,
		CreatedAt: refTime,
	}
	require.NoError(t, s.CreateVerification(ctx, v))

	got, err := s.GetVerification(ctx, v.Token)
	require.NoError(t, err)
	assertSameInstant(t, exp, got.ExpiresAt, "expires_at")
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
}

func TestTimestampRoundTrip_Notifications(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sent := refTime.Add(time.Minute)
	n := &notification.Notification{
		ID:        id.NewNotificationID(),
		AppID:     id.NewAppID(),
		EnvID:     id.NewEnvironmentID(),
		UserID:    id.NewUserID(),
		Type:      "welcome",
		Channel:   notification.ChannelEmail,
		Subject:   "Hi",
		Body:      "Welcome",
		Sent:      true,
		SentAt:    &sent,
		CreatedAt: refTime,
	}
	require.NoError(t, s.CreateNotification(ctx, n))

	got, err := s.GetNotification(ctx, n.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	require.NotNil(t, got.SentAt, "nullable sent_at lost")
	assertSameInstant(t, sent, *got.SentAt, "sent_at")
}

func TestTimestampRoundTrip_Members(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	m := &organization.Member{
		ID:        id.NewMemberID(),
		OrgID:     id.NewOrgID(),
		UserID:    id.NewUserID(),
		Role:      organization.MemberRole("member"),
		CreatedAt: refTime,
		UpdatedAt: refTime,
	}
	require.NoError(t, s.CreateMember(ctx, m))

	got, err := s.GetMember(ctx, m.ID)
	require.NoError(t, err)
	assertSameInstant(t, refTime, got.CreatedAt, "created_at")
	assertSameInstant(t, refTime, got.UpdatedAt, "updated_at")
}
