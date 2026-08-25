package waitlist

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
)

// The timestamps below are ordinary time.Now() values. That matters: the bug
// this test covers only bites a time.Time carrying a monotonic reading, which is
// what time.Now() returns and what the store writes. A fixed time.Date(…) value
// would sail through the broken schema and prove nothing.
func TestTimestampRoundTrip_Entries(t *testing.T) {
	ctx := context.Background()

	dsn := "file:" + filepath.Join(t.TempDir(), "waitlist-timestamps.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	core := sqlitestore.New(db)
	require.NoError(t, core.Migrate(ctx, SqliteMigrations))

	a := &app.App{ID: id.NewAppID(), Name: "Acme", Slug: "acme"}
	require.NoError(t, core.CreateApp(ctx, a))

	s := NewSqliteStore(db)

	before := time.Now()
	e := &WaitlistEntry{
		AppID:  a.ID,
		Email:  "hopeful@example.com",
		Name:   "Ada",
		Status: StatusPending,
	}
	require.NoError(t, s.CreateEntry(ctx, e))

	got, err := s.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, before, got.CreatedAt, time.Second, "created_at did not round-trip")
	assert.WithinDuration(t, before, got.UpdatedAt, time.Second, "updated_at did not round-trip")
}
