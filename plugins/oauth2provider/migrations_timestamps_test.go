package oauth2provider

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/user"
)

// Every timestamp here is an ordinary time.Now(). That matters: the bug these
// tests cover only shows up for a time.Time carrying a monotonic reading, which
// is what time.Now() returns and what production code writes. A fixed
// time.Date(…) value would sail through the broken schema and prove nothing.

// newTimestampTestStore migrates a fresh SQLite database and seeds the app and
// user rows the OAuth2 foreign keys require.
func newTimestampTestStore(t *testing.T) (*SqliteStore, id.AppID, id.UserID) {
	t.Helper()
	ctx := context.Background()

	dsn := "file:" + filepath.Join(t.TempDir(), "oauth2-timestamps.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	require.NoError(t, core.Migrate(ctx, SqliteMigrations))

	a := &app.App{ID: id.NewAppID(), Name: "Acme", Slug: "acme"}
	require.NoError(t, core.CreateApp(ctx, a))
	u := &user.User{ID: id.NewUserID(), AppID: a.ID, Email: "user@example.com"}
	require.NoError(t, core.CreateUser(ctx, u))

	return NewSqliteStore(db), a.ID, u.ID
}

func assertSameInstant(t *testing.T, want, got time.Time, field string) {
	t.Helper()
	assert.Falsef(t, got.IsZero(), "%s round-tripped to the zero time", field)
	assert.WithinDurationf(t, want, got, time.Second, "%s did not round-trip", field)
}

func TestTimestampRoundTrip_Clients(t *testing.T) {
	ctx := context.Background()
	s, appID, _ := newTimestampTestStore(t)

	before := time.Now()
	c := &OAuth2Client{
		ID:       id.NewOAuth2ClientID(),
		AppID:    appID,
		Name:     "CLI",
		ClientID: "client_roundtrip",
	}
	require.NoError(t, s.CreateClient(ctx, c))

	got, err := s.GetClient(ctx, "client_roundtrip")
	require.NoError(t, err)
	assertSameInstant(t, before, got.CreatedAt, "created_at")
	assertSameInstant(t, before, got.UpdatedAt, "updated_at")
}

func TestTimestampRoundTrip_AuthCodes(t *testing.T) {
	ctx := context.Background()
	s, appID, userID := newTimestampTestStore(t)

	before := time.Now()
	exp := before.Add(10 * time.Minute)
	code := &AuthorizationCode{
		ID:          id.NewAuthCodeID(),
		Code:        "code_roundtrip",
		ClientID:    "client_roundtrip",
		UserID:      userID,
		AppID:       appID,
		RedirectURI: "https://example.com/cb",
		ExpiresAt:   exp,
	}
	require.NoError(t, s.CreateAuthCode(ctx, code))

	got, err := s.GetAuthCode(ctx, "code_roundtrip")
	require.NoError(t, err)
	assertSameInstant(t, exp, got.ExpiresAt, "expires_at")
	assertSameInstant(t, before, got.CreatedAt, "created_at")
}

func TestTimestampRoundTrip_DeviceCodes(t *testing.T) {
	ctx := context.Background()
	s, appID, _ := newTimestampTestStore(t)

	before := time.Now()
	exp := before.Add(15 * time.Minute)
	dc := &DeviceCode{
		ID:              id.NewDeviceCodeID(),
		DeviceCode:      "device_roundtrip",
		UserCode:        "BCDF-GHJK",
		ClientID:        "client_roundtrip",
		AppID:           appID,
		VerificationURI: "https://example.com/device",
		ExpiresAt:       exp,
		Interval:        5,
		Status:          DeviceCodeStatusPending,
	}
	require.NoError(t, s.CreateDeviceCode(ctx, dc))

	got, err := s.GetDeviceCodeByDeviceCode(ctx, "device_roundtrip")
	require.NoError(t, err)
	assertSameInstant(t, exp, got.ExpiresAt, "expires_at")
	assertSameInstant(t, before, got.CreatedAt, "created_at")

	// last_polled_at is written on every poll, so it has to survive too.
	polled := time.Now()
	got.LastPolledAt = polled
	require.NoError(t, s.UpdateDeviceCode(ctx, got))

	repolled, err := s.GetDeviceCodeByUserCode(ctx, "BCDF-GHJK")
	require.NoError(t, err)
	assertSameInstant(t, polled, repolled.LastPolledAt, "last_polled_at")
}
