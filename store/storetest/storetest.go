// Package storetest provides a backend-agnostic conformance suite for the
// store.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract tests, so behavioral drift between
// implementations is caught automatically rather than shipped.
//
// Run it from each backend's test package:
//
//	func TestConformance(t *testing.T) {
//	    storetest.RunConformance(t, func(t *testing.T) store.Store { ... })
//	}
package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// Factory creates a fresh, empty, migrated store for a single test.
type Factory func(t *testing.T) store.Store

// RunConformance runs every contract test against stores produced by newStore.
func RunConformance(t *testing.T, newStore Factory) {
	t.Helper()
	cases := []struct {
		name string
		fn   func(t *testing.T, s store.Store)
	}{
		{"AppCRUD", testAppCRUD},
		{"DeleteAppCascade", testDeleteAppCascade},
		{"UserEmailIsAppScoped", testUserEmailIsAppScoped},
		{"ListUsersTotalAndFilter", testListUsersTotalAndFilter},
		{"ListUsersEmailMetacharsAreSafe", testListUsersEmailMetacharsAreSafe},
		{"SessionCRUD", testSessionCRUD},
		{"SessionLookupByTokenIsScoped", testSessionLookupByTokenIsScoped},
		{"RotateSessionCAS", testRotateSessionCAS},
		{"RefreshTokenRevocation", testRefreshTokenRevocation},
		{"RefreshTokenReplayIsIdempotent", testRefreshTokenReplayIsIdempotent},
		{"OrgMemberLookupAndCascade", testOrgMemberLookupAndCascade},
		{"ListUserSessionsIsScopedToUser", testListUserSessionsIsScopedToUser},
		{"SessionDPoPJKTRoundTrip", testSessionDPoPJKTRoundTrip},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, newStore(t))
		})
	}
}

// ──────────────────────────────────────────────────
// Fixtures
// ──────────────────────────────────────────────────

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

func suffix(idStr string) string {
	if len(idStr) <= 10 {
		return idStr
	}
	return idStr[len(idStr)-10:]
}

type tenant struct {
	AppID id.AppID
	EnvID id.EnvironmentID
}

// seedTenant creates an app + default environment (the parent rows every
// FK-enforcing backend needs before app-scoped rows can be created).
func seedTenant(t *testing.T, s store.Store) tenant {
	t.Helper()
	ctx := context.Background()
	appID := id.NewAppID()
	sfx := suffix(appID.String())
	require.NoError(t, s.CreateApp(ctx, &app.App{
		ID:             appID,
		Name:           "App " + sfx,
		Slug:           "app-" + sfx,
		PublishableKey: "pk_test_" + sfx,
		CreatedAt:      now(),
		UpdatedAt:      now(),
	}))
	env := &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     appID,
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateEnvironment(ctx, env))
	return tenant{AppID: appID, EnvID: env.ID}
}

func seedUser(t *testing.T, s store.Store, tn tenant, email string) *user.User {
	t.Helper()
	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     tn.AppID,
		EnvID:     tn.EnvID,
		Email:     email,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateUser(context.Background(), u))
	return u
}

func seedSession(t *testing.T, s store.Store, tn tenant, userID id.UserID, token, refresh string) *session.Session {
	t.Helper()
	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                userID,
		Token:                 token,
		RefreshToken:          refresh,
		FamilyID:              id.NewSessionFamilyID(),
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(context.Background(), sess))
	return sess
}

// ──────────────────────────────────────────────────
// App
// ──────────────────────────────────────────────────

func testAppCRUD(t *testing.T, s store.Store) {
	ctx := context.Background()
	appID := id.NewAppID()
	sfx := suffix(appID.String())
	a := &app.App{ID: appID, Name: "Acme", Slug: "acme-" + sfx, PublishableKey: "pk_" + sfx, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateApp(ctx, a))

	got, err := s.GetApp(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", got.Name)

	bySlug, err := s.GetAppBySlug(ctx, "acme-"+sfx)
	require.NoError(t, err)
	assert.Equal(t, appID.String(), bySlug.ID.String())

	byKey, err := s.GetAppByPublishableKey(ctx, "pk_"+sfx)
	require.NoError(t, err)
	assert.Equal(t, appID.String(), byKey.ID.String())

	got.Name = "Renamed"
	got.UpdatedAt = now()
	require.NoError(t, s.UpdateApp(ctx, got))
	reread, err := s.GetApp(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", reread.Name)

	require.NoError(t, s.DeleteApp(ctx, appID))
	_, err = s.GetApp(ctx, appID)
	assert.ErrorIs(t, err, store.ErrNotFound, "deleted app must be not-found")

	// Delete is idempotent across the codebase (matching every other Delete*
	// method): removing an already-absent app is a no-op, not an error.
	assert.NoError(t, s.DeleteApp(ctx, appID), "double delete must be idempotent")
}

func testDeleteAppCascade(t *testing.T, s store.Store) {
	ctx := context.Background()

	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "cascade@test.com")
	sess := seedSession(t, s, tn, u.ID, "tok-cascade", "rtok-cascade")
	org := &organization.Organization{ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Org", Slug: "org", CreatedBy: u.ID, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateOrganization(ctx, org))
	require.NoError(t, s.CreateMember(ctx, &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID, Role: organization.RoleOwner, CreatedAt: now(), UpdatedAt: now()}))

	// Survivor tenant.
	other := seedTenant(t, s)
	otherU := seedUser(t, s, other, "keeper@test.com")

	require.NoError(t, s.DeleteApp(ctx, tn.AppID))

	_, err := s.GetApp(ctx, tn.AppID)
	assert.ErrorIs(t, err, store.ErrNotFound, "app row must be deleted")
	_, err = s.GetUser(ctx, u.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "user must be cascaded")
	_, err = s.GetSession(ctx, sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "session must be cascaded")
	_, err = s.GetOrganization(ctx, org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "organization must be cascaded")
	members, err := s.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "org members must be cascaded")
	envs, err := s.ListEnvironments(ctx, tn.AppID)
	require.NoError(t, err)
	assert.Empty(t, envs, "environments must be cascaded")

	// Survivor untouched.
	_, err = s.GetApp(ctx, other.AppID)
	require.NoError(t, err, "other app must survive")
	_, err = s.GetUser(ctx, otherU.ID)
	require.NoError(t, err, "other app's user must survive")
}

// ──────────────────────────────────────────────────
// Users
// ──────────────────────────────────────────────────

func testUserEmailIsAppScoped(t *testing.T, s store.Store) {
	ctx := context.Background()
	a := seedTenant(t, s)
	b := seedTenant(t, s)

	ua := seedUser(t, s, a, "shared@test.com")
	ub := seedUser(t, s, b, "shared@test.com")
	assert.NotEqual(t, ua.ID.String(), ub.ID.String())

	gotA, err := s.GetUserByEmail(ctx, a.AppID, "shared@test.com")
	require.NoError(t, err)
	assert.Equal(t, ua.ID.String(), gotA.ID.String(), "lookup must return this app's user")

	gotB, err := s.GetUserByEmail(ctx, b.AppID, "shared@test.com")
	require.NoError(t, err)
	assert.Equal(t, ub.ID.String(), gotB.ID.String(), "lookup must be scoped per app")

	// An app that has no such user must not resolve one from a sibling app.
	empty := seedTenant(t, s)
	_, err = s.GetUserByEmail(ctx, empty.AppID, "shared@test.com")
	assert.ErrorIs(t, err, store.ErrNotFound, "email lookup must not cross tenants")
}

func testListUsersTotalAndFilter(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)

	// Empty app: total is zero (load-bearing for the first-run setup guard).
	empty, err := s.ListUsers(ctx, &user.Query{AppID: tn.AppID, Limit: 50})
	require.NoError(t, err)
	assert.Equal(t, 0, empty.Total, "empty app must report Total==0")

	seedUser(t, s, tn, "alice@test.com")
	seedUser(t, s, tn, "bob@test.com")

	all, err := s.ListUsers(ctx, &user.Query{AppID: tn.AppID, Limit: 50})
	require.NoError(t, err)
	assert.Positive(t, all.Total, "populated app must report Total>0")
	assert.Len(t, all.Users, 2)

	filtered, err := s.ListUsers(ctx, &user.Query{AppID: tn.AppID, Email: "alice", Limit: 50})
	require.NoError(t, err)
	require.Len(t, filtered.Users, 1, "email filter must narrow results")
	assert.Equal(t, "alice@test.com", filtered.Users[0].Email)

	// Another app's users must never appear.
	other := seedTenant(t, s)
	seedUser(t, s, other, "carol@test.com")
	scoped, err := s.ListUsers(ctx, &user.Query{AppID: tn.AppID, Limit: 50})
	require.NoError(t, err)
	assert.Len(t, scoped.Users, 2, "list must be scoped to the queried app")
}

func testListUsersEmailMetacharsAreSafe(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	seedUser(t, s, tn, "real@test.com")

	// A value full of regex/LIKE metacharacters must be treated as a literal
	// search, never interpreted (no injection, no error, no crash).
	res, err := s.ListUsers(ctx, &user.Query{AppID: tn.AppID, Email: "(a+)+$.*%_", Limit: 50})
	require.NoError(t, err, "metacharacter search must not error")
	assert.Empty(t, res.Users, "a literal search for a nonexistent value must match nothing")
}

// ──────────────────────────────────────────────────
// Sessions
// ──────────────────────────────────────────────────

func testSessionCRUD(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "sess@test.com")
	sess := seedSession(t, s, tn, u.ID, "tok-1", "rtok-1")

	byID, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), byID.UserID.String())

	byToken, err := s.GetSessionByToken(ctx, "tok-1")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byToken.ID.String())

	byRefresh, err := s.GetSessionByRefreshToken(ctx, "rtok-1")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byRefresh.ID.String())

	require.NoError(t, s.DeleteSession(ctx, sess.ID))
	_, err = s.GetSession(ctx, sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetSessionByToken(ctx, "tok-1")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func testSessionLookupByTokenIsScoped(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "scoped@test.com")
	seedSession(t, s, tn, u.ID, "the-token", "the-refresh")

	_, err := s.GetSessionByToken(ctx, "not-a-token")
	assert.ErrorIs(t, err, store.ErrNotFound, "unknown token must be not-found, not another session")
	_, err = s.GetSessionByRefreshToken(ctx, "not-a-refresh")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func testRotateSessionCAS(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "rotate@test.com")
	sess := seedSession(t, s, tn, u.ID, "old-tok", "old-rtok")

	// Rotate with a stale expected token: must be a no-op (compare-and-swap
	// mismatch). This is the guard against the refresh-token TOCTOU race.
	stale := *sess
	stale.Token = "new-tok-a"
	stale.RefreshToken = "new-rtok-a"
	swapped, err := s.RotateSession(ctx, &stale, "WRONG-old-tok")
	require.NoError(t, err)
	assert.False(t, swapped, "rotation with a mismatched expected token must not swap")
	unchanged, err := s.GetSessionByToken(ctx, "old-tok")
	require.NoError(t, err, "the original token must still be valid after a failed CAS")
	assert.Equal(t, sess.ID.String(), unchanged.ID.String())

	// Rotate with the correct expected token: must swap atomically.
	fresh := *sess
	fresh.Token = "new-tok-b"
	fresh.RefreshToken = "new-rtok-b"
	swapped, err = s.RotateSession(ctx, &fresh, "old-tok")
	require.NoError(t, err)
	assert.True(t, swapped, "rotation with the correct expected token must swap")
	_, err = s.GetSessionByToken(ctx, "old-tok")
	assert.ErrorIs(t, err, store.ErrNotFound, "old token must no longer resolve after rotation")
	rotated, err := s.GetSessionByToken(ctx, "new-tok-b")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), rotated.ID.String())
}

func testListUserSessionsIsScopedToUser(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	ua := seedUser(t, s, tn, "usera@test.com")
	ub := seedUser(t, s, tn, "userb@test.com")
	seedSession(t, s, tn, ua.ID, "a-tok", "a-rtok")
	seedSession(t, s, tn, ua.ID, "a-tok-2", "a-rtok-2")
	seedSession(t, s, tn, ub.ID, "b-tok", "b-rtok")

	aSessions, err := s.ListUserSessions(ctx, ua.ID)
	require.NoError(t, err)
	assert.Len(t, aSessions, 2, "must return only user A's sessions")
	for _, se := range aSessions {
		assert.Equal(t, ua.ID.String(), se.UserID.String())
	}
}

func testSessionDPoPJKTRoundTrip(t *testing.T, s store.Store) {
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "dpop-jkt@example.com")

	const jkt = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"

	bound := seedSession(t, s, tn, u.ID, "tok-dpop-bound", "ref-dpop-bound")
	bound.DPoPJKT = jkt
	require.NoError(t, s.UpdateSession(context.Background(), bound))

	got, err := s.GetSessionByToken(context.Background(), "tok-dpop-bound")
	require.NoError(t, err)
	assert.Equal(t, jkt, got.DPoPJKT)

	// An unbound session must come back as the empty string and never as a
	// driver-specific null. Mongo in particular has a history here.
	unbound := seedSession(t, s, tn, u.ID, "tok-dpop-unbound", "ref-dpop-unbound")
	fetched, err := s.GetSessionByToken(context.Background(), unbound.Token)
	require.NoError(t, err)
	assert.Empty(t, fetched.DPoPJKT)
}

// ──────────────────────────────────────────────────
// Refresh-token revocation / replay
// ──────────────────────────────────────────────────

func testRefreshTokenRevocation(t *testing.T, s store.Store) {
	ctx := context.Background()
	family := id.NewSessionFamilyID()
	hash := "hash-revoked-1"

	revoked, err := s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.False(t, revoked, "a fresh token must not be revoked")

	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, family, session.RevokeReasonRotated))
	revoked, err = s.IsRefreshTokenRevoked(ctx, hash)
	require.NoError(t, err)
	assert.True(t, revoked, "a marked token must be revoked")

	gotFamily, err := s.GetRevokedRefreshTokenFamily(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, family.String(), gotFamily.String())
}

func testRefreshTokenReplayIsIdempotent(t *testing.T, s store.Store) {
	ctx := context.Background()
	family := id.NewSessionFamilyID()
	hash := "hash-replayed-1"
	require.NoError(t, s.MarkRefreshTokenRevoked(ctx, hash, family, session.RevokeReasonRotated))

	first, err := s.MarkRefreshTokenReplayed(ctx, hash)
	require.NoError(t, err)
	assert.True(t, first, "the first replay detection must report firstReplay=true")

	second, err := s.MarkRefreshTokenReplayed(ctx, hash)
	require.NoError(t, err)
	assert.False(t, second, "a subsequent replay must report firstReplay=false (idempotent alert)")
}

// ──────────────────────────────────────────────────
// Organizations
// ──────────────────────────────────────────────────

func testOrgMemberLookupAndCascade(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "orguser@test.com")
	org := &organization.Organization{ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Acme", Slug: "acme", CreatedBy: u.ID, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateOrganization(ctx, org))
	require.NoError(t, s.CreateMember(ctx, &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID, Role: organization.RoleOwner, CreatedAt: now(), UpdatedAt: now()}))

	m, err := s.GetMemberByUserAndOrg(ctx, u.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.RoleOwner, m.Role)

	// A non-member must not resolve.
	_, err = s.GetMemberByUserAndOrg(ctx, id.NewUserID(), org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "non-member lookup must be not-found")

	require.NoError(t, s.DeleteOrganizationCascade(ctx, org.ID))
	_, err = s.GetOrganization(ctx, org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "org must be deleted")
	members, err := s.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "members must be cascaded with the org")
}
