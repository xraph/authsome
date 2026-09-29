package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
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
func newBootstrapEngine(t *testing.T, bootstrapOpts ...authsome.BootstrapOption) (*authsome.Engine, *memory.Store) { //nolint:unparam // store return retained for future tests
	t.Helper()
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	eng, err := authsome.NewEngine(authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(testEngineConfig()),
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
// promoteVerifiedOwner — ownership is claimed on verification
// ──────────────────────────────────────────────────

// signUpUnverified creates a platform user and leaves the email unproven.
func signUpUnverified(t *testing.T, eng *authsome.Engine, email string) id.UserID {
	t.Helper()
	appID := eng.PlatformAppID()
	require.False(t, appID.IsNil(), "platform app should be bootstrapped")
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss1",
		FirstName: "User",
	})
	require.NoError(t, err)
	return u.ID
}

// signUpVerified creates a platform user and proves the email through the
// real OTP path, which is where ownership is decided.
func signUpVerified(t *testing.T, eng *authsome.Engine, email string) id.UserID {
	t.Helper()
	uid := signUpUnverified(t, eng, email)
	secutil.VerifyEmail(t, eng, uid)
	return uid
}

func isPlatformOwner(t *testing.T, eng *authsome.Engine, uid id.UserID) bool {
	t.Helper()
	roles, err := eng.ListUserRoles(context.Background(), uid)
	require.NoError(t, err)
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			return true
		}
	}
	return false
}

func TestBootstrap_FirstVerifiedUserBecomesOwner(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	uid := signUpVerified(t, eng, "first@example.com")
	assert.True(t, isPlatformOwner(t, eng, uid), "the first verified user claims the owner slot")
}

func TestBootstrap_UnverifiedSignUpIsNotOwner(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	uid := signUpUnverified(t, eng, "unverified@example.com")
	assert.False(t, isPlatformOwner(t, eng, uid), "an unproven email must never hold ownership")

	// Verifying later claims the still-open slot.
	secutil.VerifyEmail(t, eng, uid)
	assert.True(t, isPlatformOwner(t, eng, uid), "verification claims the open slot")
}

func TestBootstrap_SecondVerifiedUserIsNotOwner(t *testing.T) {
	eng, _ := newBootstrapEngine(t) // default: one slot
	first := signUpVerified(t, eng, "first@example.com")
	second := signUpVerified(t, eng, "second@example.com")
	assert.True(t, isPlatformOwner(t, eng, first))
	assert.False(t, isPlatformOwner(t, eng, second), "the default leaves a single owner slot")
}

func TestBootstrap_UnverifiedUserDoesNotConsumeSlot(t *testing.T) {
	// An earlier, still-unverified sign-up must not block a later user who
	// does prove their address.
	eng, _ := newBootstrapEngine(t)
	squatter := signUpUnverified(t, eng, "squatter@example.com")
	proven := signUpVerified(t, eng, "proven@example.com")
	assert.False(t, isPlatformOwner(t, eng, squatter))
	assert.True(t, isPlatformOwner(t, eng, proven), "the slot goes to the first proven email, not the first row")
}

func TestBootstrap_InitialOwnerPromotedOnlyOnceVerified(t *testing.T) {
	const ownerEmail = "owner@example.com"
	eng, _ := newBootstrapEngine(t, authsome.WithInitialOwners(ownerEmail))

	// The slot goes to someone else first.
	_ = signUpVerified(t, eng, "regular@example.com")

	owner := signUpUnverified(t, eng, ownerEmail)
	assert.False(t, isPlatformOwner(t, eng, owner), "a listed address is not trusted until verified")

	secutil.VerifyEmail(t, eng, owner)
	assert.True(t, isPlatformOwner(t, eng, owner), "a listed address is promoted on verification even with no slot open")
}

func TestBootstrap_InitialOwner_CaseInsensitive(t *testing.T) {
	eng, _ := newBootstrapEngine(t, authsome.WithInitialOwners("Owner@Example.COM"))
	_ = signUpVerified(t, eng, "other@example.com")
	owner := signUpVerified(t, eng, "owner@example.com")
	assert.True(t, isPlatformOwner(t, eng, owner), "InitialOwners match should be case-insensitive")
}

func TestBootstrap_NonInitialOwner_NotPromoted(t *testing.T) {
	eng, _ := newBootstrapEngine(t,
		authsome.WithInitialOwners("owner@example.com"),
		authsome.WithInitialOwnerCount(1),
	)
	_ = signUpVerified(t, eng, "first@example.com")
	regular := signUpVerified(t, eng, "notowner@example.com")
	assert.False(t, isPlatformOwner(t, eng, regular), "non-listed user should NOT receive platform-owner role")
}

func TestBootstrap_InitialOwnerCount_PromotesFirstN(t *testing.T) {
	eng, _ := newBootstrapEngine(t, authsome.WithInitialOwnerCount(3))

	emails := []string{"u1@example.com", "u2@example.com", "u3@example.com", "u4@example.com"}
	users := make([]id.UserID, 0, len(emails))
	for _, email := range emails {
		users = append(users, signUpVerified(t, eng, email))
	}
	for i := range 3 {
		assert.True(t, isPlatformOwner(t, eng, users[i]), "user %d should have platform-owner", i+1)
	}
	assert.False(t, isPlatformOwner(t, eng, users[3]), "4th user should not receive platform-owner when count=3")
}

func TestBootstrap_InitialOwnerCount_Zero_DisablesCountPromotion(t *testing.T) {
	eng, _ := newBootstrapEngine(t,
		authsome.WithInitialOwnerCount(0),
		authsome.WithInitialOwners("special@example.com"),
	)
	first := signUpVerified(t, eng, "first@example.com")
	assert.False(t, isPlatformOwner(t, eng, first), "first user should not be promoted when count=0")

	special := signUpVerified(t, eng, "special@example.com")
	assert.True(t, isPlatformOwner(t, eng, special), "InitialOwners email should still be promoted when count=0")
}

func TestBootstrap_OwnerPromotionIsAudited(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	var seen []string
	eng.Hooks().On("owner-test", func(_ context.Context, ev *hook.Event) error {
		if ev.Action == hook.ActionRoleAssign && ev.Metadata["role"] == rbac.PlatformOwnerSlug {
			seen = append(seen, ev.Severity)
		}
		return nil
	})
	_ = signUpVerified(t, eng, "first@example.com")
	require.Len(t, seen, 1, "claiming ownership must leave one trail entry")
	assert.Equal(t, hook.SeverityCritical, seen[0])
}
