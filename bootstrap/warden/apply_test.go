package wardenseed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	"github.com/xraph/authsome/id"
)

func newTestEngine(t *testing.T) *warden.Engine {
	t.Helper()
	eng, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	return eng
}

func mustAppID(t *testing.T) id.AppID {
	t.Helper()
	aid, err := id.ParseAppID(testAppID)
	require.NoError(t, err)
	return aid
}

// TestApplyForApp_Platform creates roles + permissions for the platform app
// (shared + platform programs) on a fresh engine and verifies the second
// apply is a complete no-op (idempotency).
func TestApplyForApp_Platform(t *testing.T) {
	ctx := context.Background()
	eng := newTestEngine(t)
	appID := mustAppID(t)

	res, err := ApplyForApp(ctx, eng, appID, true, ApplyOptions{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.Created, "first apply should create entities")

	// Idempotency: second apply mutates nothing.
	res2, err := ApplyForApp(ctx, eng, appID, true, ApplyOptions{})
	require.NoError(t, err)
	require.Empty(t, res2.Created, "second apply should not create anything")
	require.Empty(t, res2.Updated, "second apply should not update anything")
	require.Greater(t, res2.NoOps, 0, "second apply should report no-ops")
}

// TestApplyForApp_NonPlatform skips the platform program when isPlatform=false.
func TestApplyForApp_NonPlatform(t *testing.T) {
	ctx := context.Background()
	eng := newTestEngine(t)
	appID := mustAppID(t)

	res, err := ApplyForApp(ctx, eng, appID, false, ApplyOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, res.Created)

	// Platform-only roles must NOT exist for a non-platform app.
	platformOnlySlugs := []string{"platform-user", "platform-admin", "platform-owner"}
	for _, slug := range platformOnlySlugs {
		role, err := eng.Store().GetRoleBySlug(ctx, appID.String(), slug)
		require.Error(t, err, "platform-only role %q must not be created on a non-platform app", slug)
		require.Nil(t, role)
	}

	// App-scoped roles must exist.
	for _, slug := range []string{"user", "admin", "owner"} {
		role, err := eng.Store().GetRoleBySlug(ctx, appID.String(), slug)
		require.NoError(t, err, "expected app-scoped role %q to exist", slug)
		require.NotNil(t, role)
	}
}

// TestApplyForApp_DryRun confirms DryRun reports the diff but writes nothing.
func TestApplyForApp_DryRun(t *testing.T) {
	ctx := context.Background()
	eng := newTestEngine(t)
	appID := mustAppID(t)

	res, err := ApplyForApp(ctx, eng, appID, true, ApplyOptions{DryRun: true})
	require.NoError(t, err)
	require.NotEmpty(t, res.Created)

	// Engine store should remain empty.
	role, err := eng.Store().GetRoleBySlug(ctx, appID.String(), "user")
	require.Error(t, err, "DryRun must not write any roles")
	require.Nil(t, role)
}

// TestApplyForApp_RejectsNilEngine surfaces a clean error when no engine is provided.
func TestApplyForApp_RejectsNilEngine(t *testing.T) {
	_, err := ApplyForApp(context.Background(), nil, mustAppID(t), true, ApplyOptions{})
	require.Error(t, err)
}

// TestApplyForApp_NamespacePathStamping verifies the new namespace nesting
// feature lands roles at the right NamespacePath in the store.
func TestApplyForApp_NamespacePathStamping(t *testing.T) {
	ctx := context.Background()
	eng := newTestEngine(t)
	appID := mustAppID(t)

	_, err := ApplyForApp(ctx, eng, appID, true, ApplyOptions{})
	require.NoError(t, err)

	cases := []struct {
		slug   string
		nsPath string
	}{
		{"user", "app"},
		{"admin", "app"},
		{"owner", "app"},
		{"platform-user", "platform"},
		{"platform-admin", "platform"},
		{"platform-owner", "platform"},
	}
	for _, tc := range cases {
		role, err := eng.Store().GetRoleBySlug(ctx, appID.String(), tc.slug)
		require.NoError(t, err, "role %q should exist", tc.slug)
		require.Equal(t, tc.nsPath, role.NamespacePath, "role %q should be in namespace %q", tc.slug, tc.nsPath)
	}
}
