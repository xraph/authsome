package authsome_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// newBootstrapEngine creates an engine with bootstrap enabled so that the
// platform app and roles are created, platformAppID is set, and sign-ups
// can trigger promoteFirstUserToOwner.
func newBootstrapEngine(t *testing.T, bootstrapOpts ...authsome.BootstrapOption) (*authsome.Engine, *memory.Store) {
	t.Helper()
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithBootstrap(bootstrapOpts...),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	secutil.RelaxAuthDefaults(t, eng)
	return eng, s
}

// ──────────────────────────────────────────────────
// WithInitialOwners option building
// ──────────────────────────────────────────────────

func TestWithInitialOwners_AppendsBehavior(t *testing.T) {
	cfg := authsome.DefaultBootstrapConfig()
	authsome.WithInitialOwners("alice@example.com", "bob@example.com")(cfg)
	authsome.WithInitialOwners("carol@example.com")(cfg)

	require.Len(t, cfg.InitialOwners, 3)
	assert.Equal(t, "alice@example.com", cfg.InitialOwners[0])
	assert.Equal(t, "bob@example.com", cfg.InitialOwners[1])
	assert.Equal(t, "carol@example.com", cfg.InitialOwners[2])
}

func TestWithInitialOwners_EmptyCallIsNoop(t *testing.T) {
	cfg := authsome.DefaultBootstrapConfig()
	authsome.WithInitialOwners()(cfg)
	assert.Empty(t, cfg.InitialOwners)
}

// ──────────────────────────────────────────────────
// promoteFirstUserToOwner — first-user logic preserved
// ──────────────────────────────────────────────────

func TestBootstrap_FirstUserBecomesOwner(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil(), "platform app should be bootstrapped")

	// Sign up the first user.
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "first@example.com",
		Password:  "SecureP@ss1",
		FirstName: "First",
	})
	require.NoError(t, err)

	roles, err := eng.ListUserRoles(ctx, u.ID)
	require.NoError(t, err)

	hasPlatformOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			hasPlatformOwner = true
			break
		}
	}
	assert.True(t, hasPlatformOwner, "first user should be promoted to platform-owner")
}

// ──────────────────────────────────────────────────
// promoteFirstUserToOwner — InitialOwners path
// ──────────────────────────────────────────────────

func TestBootstrap_InitialOwnerPromotedEvenIfNotFirst(t *testing.T) {
	const ownerEmail = "owner@example.com"

	eng, _ := newBootstrapEngine(t, authsome.WithInitialOwners(ownerEmail))
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil(), "platform app should be bootstrapped")

	// Sign up a regular user first (becomes first user, gets owner too).
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "regular@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Regular",
	})
	require.NoError(t, err)

	// Now sign up the pre-configured owner (second user).
	owner, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     ownerEmail,
		Password:  "SecureP@ss1",
		FirstName: "Owner",
	})
	require.NoError(t, err)

	roles, err := eng.ListUserRoles(ctx, owner.ID)
	require.NoError(t, err)

	hasPlatformOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			hasPlatformOwner = true
			break
		}
	}
	assert.True(t, hasPlatformOwner, "pre-configured initial owner should receive platform-owner role")
}

func TestBootstrap_InitialOwner_CaseInsensitive(t *testing.T) {
	// Owner registered with uppercase in config, signs up with lowercase.
	eng, _ := newBootstrapEngine(t, authsome.WithInitialOwners("Owner@Example.COM"))
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	// First user (different email) gets owner by first-user rule.
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "other@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Other",
	})
	require.NoError(t, err)

	// The configured owner signs up with a different case.
	owner, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "owner@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Owner",
	})
	require.NoError(t, err)

	roles, err := eng.ListUserRoles(ctx, owner.ID)
	require.NoError(t, err)

	hasPlatformOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			hasPlatformOwner = true
			break
		}
	}
	assert.True(t, hasPlatformOwner, "InitialOwners match should be case-insensitive")
}

func TestBootstrap_NonInitialOwner_NotPromoted(t *testing.T) {
	// Pin count to 1 so only the very first user is auto-promoted; the second
	// user (not in InitialOwners) must NOT receive platform-owner.
	eng, _ := newBootstrapEngine(t,
		authsome.WithInitialOwners("owner@example.com"),
		authsome.WithInitialOwnerCount(1),
	)
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	// First user — gets owner by first-user rule.
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "first@example.com",
		Password:  "SecureP@ss1",
		FirstName: "First",
	})
	require.NoError(t, err)

	// Second user, not in InitialOwners.
	regular, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "notowner@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Regular",
	})
	require.NoError(t, err)

	roles, err := eng.ListUserRoles(ctx, regular.ID)
	require.NoError(t, err)

	for _, r := range roles {
		assert.NotEqual(t, rbac.PlatformOwnerSlug, r.Slug,
			"non-listed user should NOT receive platform-owner role")
	}
}

func TestBootstrap_InitialOwnerCount_PromotesFirstN(t *testing.T) {
	// Default count is 3 — first 3 users should all receive platform-owner.
	eng, _ := newBootstrapEngine(t)
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	emails := []string{"u1@example.com", "u2@example.com", "u3@example.com", "u4@example.com"}
	var users []id.UserID
	for i, email := range emails {
		u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
			AppID:     appID,
			Email:     email,
			Password:  "SecureP@ss1",
			FirstName: fmt.Sprintf("User%d", i+1),
		})
		require.NoError(t, err)
		users = append(users, u.ID)
	}

	// Users 1–3 should have platform-owner.
	for i := 0; i < 3; i++ {
		roles, err := eng.ListUserRoles(ctx, users[i])
		require.NoError(t, err)
		hasOwner := false
		for _, r := range roles {
			if r.Slug == rbac.PlatformOwnerSlug {
				hasOwner = true
				break
			}
		}
		assert.True(t, hasOwner, "user %d (index %d) should have platform-owner", i+1, i)
	}

	// User 4 (index 3) should NOT have platform-owner.
	roles, err := eng.ListUserRoles(ctx, users[3])
	require.NoError(t, err)
	for _, r := range roles {
		assert.NotEqual(t, rbac.PlatformOwnerSlug, r.Slug,
			"4th user should not receive platform-owner when count=3")
	}
}

func TestBootstrap_InitialOwnerCount_Zero_DisablesCountPromotion(t *testing.T) {
	// Count=0 disables the count-based path entirely; only InitialOwners emails work.
	eng, _ := newBootstrapEngine(t,
		authsome.WithInitialOwnerCount(0),
		authsome.WithInitialOwners("special@example.com"),
	)
	ctx := context.Background()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil())

	// First user — NOT in InitialOwners, count path disabled.
	first, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID: appID, Email: "first@example.com", Password: "SecureP@ss1", FirstName: "First",
	})
	require.NoError(t, err)
	roles, err := eng.ListUserRoles(ctx, first.ID)
	require.NoError(t, err)
	for _, r := range roles {
		assert.NotEqual(t, rbac.PlatformOwnerSlug, r.Slug, "first user should not be promoted when count=0")
	}

	// Special user — IS in InitialOwners, should be promoted.
	special, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID: appID, Email: "special@example.com", Password: "SecureP@ss1", FirstName: "Special",
	})
	require.NoError(t, err)
	roles, err = eng.ListUserRoles(ctx, special.ID)
	require.NoError(t, err)
	hasOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			hasOwner = true
			break
		}
	}
	assert.True(t, hasOwner, "InitialOwners email should still be promoted when count=0")
}
