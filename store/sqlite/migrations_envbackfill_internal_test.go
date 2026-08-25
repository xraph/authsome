//go:build integration

package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
)

// applyAllMigrations brings a fresh executor up to the current schema.
func applyAllMigrations(ctx context.Context, t *testing.T, exec migrate.Executor) {
	t.Helper()
	for _, m := range Migrations.Migrations() {
		require.NoErrorf(t, m.Up(ctx, exec), "migration %s (%s)", m.Version, m.Name)
	}
}

// seedLegacyApp writes the shape migration 7 used to leave behind: an app with
// no environment at all, and a user whose env_id is the empty string that
// migration's ALTER defaulted it to.
func seedLegacyApp(ctx context.Context, t *testing.T, exec migrate.Executor) (appID, userID string) {
	t.Helper()
	appID = id.NewAppID().String()
	userID = id.NewUserID().String()

	_, err := exec.Exec(ctx,
		`INSERT INTO authsome_apps (id, name, slug) VALUES (?, 'Legacy', ?)`,
		appID, "legacy-"+appID[len(appID)-8:])
	require.NoError(t, err)

	_, err = exec.Exec(ctx,
		`INSERT INTO authsome_users (id, app_id, env_id, email) VALUES (?, ?, '', ?)`,
		userID, appID, "legacy@test.com")
	require.NoError(t, err)

	return appID, userID
}

func envIDOfUser(ctx context.Context, t *testing.T, exec migrate.Executor, userID string) string {
	t.Helper()
	rows, err := exec.Query(ctx, `SELECT env_id FROM authsome_users WHERE id = ?`, userID)
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next(), "user %s not found", userID)
	var envID string
	require.NoError(t, rows.Scan(&envID))
	return envID
}

func defaultEnvCount(ctx context.Context, t *testing.T, exec migrate.Executor, appID string) int {
	t.Helper()
	rows, err := exec.Query(ctx,
		`SELECT COUNT(*) FROM authsome_environments WHERE app_id = ? AND is_default = TRUE`, appID)
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	var n int
	require.NoError(t, rows.Scan(&n))
	return n
}

// A row left behind by migration 7 belongs to no environment. Once it is
// stamped, the env-scoped user lookups can find it again; until then a
// sign-in that resolves a real environment misses the account entirely.
func TestBackfillDefaultEnvironments_StampsLegacyRows(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)
	applyAllMigrations(ctx, t, exec)

	appID, userID := seedLegacyApp(ctx, t, exec)
	require.Equal(t, "", envIDOfUser(ctx, t, exec, userID), "precondition: legacy row has no env")
	require.Equal(t, 0, defaultEnvCount(ctx, t, exec, appID), "precondition: app has no default env")

	require.NoError(t, backfillDefaultEnvironments(ctx, exec))

	assert.Equal(t, 1, defaultEnvCount(ctx, t, exec, appID),
		"the app must gain exactly one default environment")
	got := envIDOfUser(ctx, t, exec, userID)
	assert.NotEmpty(t, got, "the legacy user must be stamped with an environment")
	_, parseErr := id.ParseEnvironmentID(got)
	assert.NoError(t, parseErr, "stamped value must be an environment id, got %q", got)
}

// Migrations get re-run: against a partially migrated database, by the
// self-heal path, or by an operator. A second pass must not mint a second
// default environment or move a user that already has one.
func TestBackfillDefaultEnvironments_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)
	applyAllMigrations(ctx, t, exec)

	appID, userID := seedLegacyApp(ctx, t, exec)
	require.NoError(t, backfillDefaultEnvironments(ctx, exec))
	first := envIDOfUser(ctx, t, exec, userID)

	require.NoError(t, backfillDefaultEnvironments(ctx, exec))

	assert.Equal(t, 1, defaultEnvCount(ctx, t, exec, appID),
		"a second pass must not create another default environment")
	assert.Equal(t, first, envIDOfUser(ctx, t, exec, userID),
		"a second pass must not move a user that already has an environment")
}

// An app that already has a default environment keeps it. Re-pointing users at
// a freshly minted environment would silently move accounts between
// environments, which is the very thing the env scoping exists to prevent.
func TestBackfillDefaultEnvironments_ReusesExistingDefault(t *testing.T) {
	ctx := context.Background()
	exec := openExec(t)
	applyAllMigrations(ctx, t, exec)

	appID, userID := seedLegacyApp(ctx, t, exec)
	existing := id.NewEnvironmentID().String()
	_, err := exec.Exec(ctx,
		`INSERT INTO authsome_environments (id, app_id, name, slug, type, is_default)
		 VALUES (?, ?, 'Production', 'production', 'production', TRUE)`, existing, appID)
	require.NoError(t, err)

	require.NoError(t, backfillDefaultEnvironments(ctx, exec))

	assert.Equal(t, 1, defaultEnvCount(ctx, t, exec, appID))
	assert.Equal(t, existing, envIDOfUser(ctx, t, exec, userID),
		"the app's existing default environment must be reused, not replaced")
}
