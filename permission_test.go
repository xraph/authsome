package authsome_test

// Tests for HasPermission after the warden namespace-support update (566f0e1).
//
// That commit:
//   - Added NamespacePath to all warden Mongo models (roles, permissions, assignments)
//   - Renamed ParentID → ParentSlug on the role model
//   - Made GetRoleBySlug and GetPermissionByName require an explicit namespacePath arg
//   - Changed evaluateRBAC to filter assignments by namespace_path IN [""]
//
// These tests cover the three failure modes that were observed in production:
//   1. Role lookup broken: GetRoleBySlug called without namespace → compile error (now fixed)
//   2. Permission inheritance broken: parent_slug missing on existing roles → no perms
//   3. Assignment namespace filter: assignments without namespace_path field → 403

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// ──────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────

// signUpOnPlatform creates a user on the platform app and returns their ID.
func signUpOnPlatform(t *testing.T, eng *authsome.Engine, email string) id.UserID {
	t.Helper()
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil(), "platform app must be bootstrapped before sign-up")

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss1",
		FirstName: "Test",
	})
	require.NoError(t, err)
	return u.ID
}

// hasRole returns true if the user has the given slug in their role list.
func hasRole(t *testing.T, eng *authsome.Engine, userID id.UserID, slug string) bool {
	t.Helper()
	roles, err := eng.ListUserRoles(context.Background(), userID)
	require.NoError(t, err)
	for _, r := range roles {
		if r.Slug == slug {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────
// GetRoleBySlug: namespace fallback
// ──────────────────────────────────────────────────

// TestGetRoleBySlug_FindsPlatformNamespaceRoles verifies that GetRoleBySlug
// can find roles seeded by the warden DSL into the "platform" namespace
// even though the caller passes only the app ID (no explicit namespace).
// The WardenStore.GetRoleBySlug implementation must try "" then "platform".
func TestGetRoleBySlug_FindsPlatformNamespaceRoles(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	for _, slug := range []string{
		rbac.PlatformUserSlug,
		rbac.PlatformAdminSlug,
		rbac.PlatformOwnerSlug,
	} {
		role, err := eng.GetRoleBySlug(ctx, appID, slug)
		require.NoError(t, err, "GetRoleBySlug(%q) should succeed", slug)
		assert.NotNil(t, role)
		assert.Equal(t, slug, role.Slug)
	}
}

// ──────────────────────────────────────────────────
// HasPermission: no forge scope (background context)
// ──────────────────────────────────────────────────

// TestHasPermission_PlatformOwner_NoScope checks that a platform-owner can
// manage apps even when the context has no forge scope (e.g., a background
// job or CLI call). ensureWardenScope falls back to platform app ID.
func TestHasPermission_PlatformOwner_NoScope(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	userID := signUpOnPlatform(t, eng, "owner@example.com")

	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug),
		"first user should be promoted to platform-owner")

	allowed, err := eng.HasPermission(context.Background(), userID, "manage", "app")
	require.NoError(t, err)
	assert.True(t, allowed, "platform-owner must have app:manage permission")
}

// TestHasPermission_PlatformOwner_ReadUser verifies a permission that comes
// from the grandparent (platform-user) via two-hop inheritance.
// platform-owner → platform-admin → platform-user { user:read }
func TestHasPermission_PlatformOwner_ReadUser(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	userID := signUpOnPlatform(t, eng, "owner2@example.com")

	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug))

	allowed, err := eng.HasPermission(context.Background(), userID, "read", "user")
	require.NoError(t, err)
	assert.True(t, allowed, "platform-owner must inherit user:read from platform-user grandparent")
}

// TestHasPermission_PlatformOwner_ManageSession verifies session:* permission
// inherited from platform-admin.
func TestHasPermission_PlatformOwner_ManageSession(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	userID := signUpOnPlatform(t, eng, "owner3@example.com")

	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug))

	allowed, err := eng.HasPermission(context.Background(), userID, "delete", "session")
	require.NoError(t, err)
	assert.True(t, allowed, "platform-owner must inherit session:delete from platform-admin")
}

// TestHasPermission_PlatformUser_CannotManageApp verifies that a plain
// platform-user does NOT receive app:manage (it is admin-level only).
func TestHasPermission_PlatformUser_CannotManageApp(t *testing.T) {
	// Use count=0 so only InitialOwners path fires; second user stays as
	// platform-user only.
	eng, _ := newBootstrapEngine(t,
		authsome.WithInitialOwnerCount(0),
		authsome.WithInitialOwners("owner@example.com"),
	)
	ownerID := signUpOnPlatform(t, eng, "owner@example.com")
	require.True(t, hasRole(t, eng, ownerID, rbac.PlatformOwnerSlug))

	regularID := signUpOnPlatform(t, eng, "regular@example.com")
	require.False(t, hasRole(t, eng, regularID, rbac.PlatformOwnerSlug),
		"regular user should not have platform-owner role")

	allowed, err := eng.HasPermission(context.Background(), regularID, "manage", "app")
	require.NoError(t, err)
	assert.False(t, allowed, "platform-user must NOT have app:manage permission")
}

// ──────────────────────────────────────────────────
// HasPermission: with forge app scope (simulates middleware)
// ──────────────────────────────────────────────────

// TestHasPermission_PlatformOwner_WithForgeAppScope mirrors the real middleware
// path: forge.WithScope sets AppScope(platformAppID) then HasPermission is called.
// This is the exact context that /v1/admin/apps/:appID/client-config uses.
func TestHasPermission_PlatformOwner_WithForgeAppScope(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	userID := signUpOnPlatform(t, eng, "admin@example.com")

	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug))

	// Inject the forge app scope exactly as middleware/auth.go does.
	ctx := forge.WithScope(context.Background(),
		forge.NewAppScope(eng.PlatformAppID().String()),
	)

	allowed, err := eng.HasPermission(ctx, userID, "manage", "app")
	require.NoError(t, err)
	assert.True(t, allowed, "platform-owner must have app:manage when forge app scope is set")
}

// TestHasPermission_PlatformOwner_WithForgeOrgScope verifies permission checks
// also work when the user is acting inside an org (org-scoped sessions).
func TestHasPermission_PlatformOwner_WithForgeOrgScope(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	userID := signUpOnPlatform(t, eng, "orgowner@example.com")

	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug))

	orgID := id.NewOrgID()
	ctx := forge.WithScope(context.Background(),
		forge.NewOrgScope(eng.PlatformAppID().String(), orgID.String()),
	)

	// Platform-owner check should still pass; the warden scope falls back to
	// orgID as tenant, but ensureWardenScope should map this correctly.
	allowed, err := eng.HasPermission(ctx, userID, "manage", "app")
	require.NoError(t, err)
	assert.True(t, allowed, "platform-owner must have app:manage under org scope")
}

// ──────────────────────────────────────────────────
// HasPermission: unknown user
// ──────────────────────────────────────────────────

// TestHasPermission_UnknownUser_Denied checks that a user ID that has no role
// assignments returns false without an error.
func TestHasPermission_UnknownUser_Denied(t *testing.T) {
	eng, _ := newBootstrapEngine(t)

	phantomID := id.NewUserID()
	allowed, err := eng.HasPermission(context.Background(), phantomID, "manage", "app")
	require.NoError(t, err)
	assert.False(t, allowed, "user with no roles must be denied")
}

// ──────────────────────────────────────────────────
// HasPermission: engine without warden
// ──────────────────────────────────────────────────

// TestHasPermission_NoWarden_ReturnsError verifies that NewEngine returns an
// error when no warden engine is provided. Warden is required for RBAC.
func TestHasPermission_NoWarden_ReturnsError(t *testing.T) {
	s := memory.New()
	_, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithDisableMigrate(),
		authsome.WithAppID("aapp_01jf0000000000000000000000"),
	)
	require.Error(t, err, "NewEngine without warden must return an error")
	assert.Contains(t, err.Error(), "warden")
}

// ──────────────────────────────────────────────────
// HasPermission: multiple platform owners
// ──────────────────────────────────────────────────

// TestHasPermission_AllFirstNUsersHavePermission exercises the InitialOwnerCount
// path: all three first users should pass the app:manage check, the fourth should not.
func TestHasPermission_AllFirstNUsersHavePermission(t *testing.T) {
	eng, _ := newBootstrapEngine(t) // default count=3
	ctx := context.Background()

	users := make([]id.UserID, 4)
	for i := 0; i < 4; i++ {
		email := "user" + string(rune('1'+i)) + "@example.com"
		users[i] = signUpOnPlatform(t, eng, email)
	}

	// Users 0–2 (first 3) must have app:manage.
	for i := 0; i < 3; i++ {
		allowed, err := eng.HasPermission(ctx, users[i], "manage", "app")
		require.NoError(t, err)
		assert.True(t, allowed, "user[%d] (one of first 3) must have app:manage", i)
	}

	// User 3 (fourth) must NOT have app:manage.
	allowed, err := eng.HasPermission(ctx, users[3], "manage", "app")
	require.NoError(t, err)
	assert.False(t, allowed, "4th user must NOT have app:manage")
}

// ──────────────────────────────────────────────────
// Role inheritance integrity
// ──────────────────────────────────────────────────

// TestRoleInheritance_PlatformOwner_InheritsAllAdminPerms verifies that the
// full parent chain platform-owner → platform-admin → platform-user is intact
// by checking permissions defined at each level.
func TestRoleInheritance_PlatformOwner_InheritsAllAdminPerms(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	ctx := context.Background()
	userID := signUpOnPlatform(t, eng, "inherit@example.com")
	require.True(t, hasRole(t, eng, userID, rbac.PlatformOwnerSlug))

	// From platform-admin (app:*, role:*, org:*, ...).
	adminPerms := []struct{ action, resource string }{
		{"manage", "app"},
		{"read", "role"},
		{"create", "org"},
	}
	for _, p := range adminPerms {
		ok, err := eng.HasPermission(ctx, userID, p.action, p.resource)
		require.NoError(t, err)
		assert.True(t, ok, "platform-owner should inherit %s:%s from platform-admin", p.action, p.resource)
	}

	// From platform-user (session:read, device:read, ...).
	userPerms := []struct{ action, resource string }{
		{"read", "session"},
		{"read", "device"},
		{"read", "environment"},
	}
	for _, p := range userPerms {
		ok, err := eng.HasPermission(ctx, userID, p.action, p.resource)
		require.NoError(t, err)
		assert.True(t, ok, "platform-owner should inherit %s:%s from platform-user", p.action, p.resource)
	}
}

// ──────────────────────────────────────────────────
// Regression: warden in-memory store namespace filter
// ──────────────────────────────────────────────────

// TestHasPermission_WardenMemoryStore_NamespaceFilter is a targeted regression
// test for the warden update's namespace filtering. The in-memory store's
// nsMatcher must allow assignments stored with NamespacePath="" when the
// evaluateRBAC query uses AncestorNamespaces("") = [""].
func TestHasPermission_WardenMemoryStore_NamespaceFilter(t *testing.T) {
	// Build a fresh engine using a NEW in-memory warden store to ensure this
	// test is not polluted by any prior state.
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithBootstrap(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	secutil.RelaxAuthDefaults(t, eng)

	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "nstest@example.com",
		Password:  "SecureP@ss1",
		FirstName: "NS",
	})
	require.NoError(t, err)

	require.True(t, hasRole(t, eng, u.ID, rbac.PlatformOwnerSlug),
		"first user must be promoted to platform-owner")

	// This is the exact check that was failing in production.
	allowed, err := eng.HasPermission(ctx, u.ID, "manage", "app")
	require.NoError(t, err)
	assert.True(t, allowed,
		"HasPermission must pass after warden namespace update: "+
			"assignment NamespacePath=\"\" must match filter [\"\"]; "+
			"parent chain platform-owner→platform-admin must be intact")
}
