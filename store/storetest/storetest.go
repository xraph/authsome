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
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// Factory creates a fresh, empty, migrated store for a single test.
type Factory func(t *testing.T) store.Store

// RunConformance runs every contract test against stores produced by newStore.
//
// skip names cases the backend does not yet implement. Each is still
// registered as a subtest and calls t.Skip from inside it, so a skipped case
// shows up in `go test -v` output rather than silently not existing.
func RunConformance(t *testing.T, newStore Factory, skip ...string) {
	t.Helper()
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}
	cases := []struct {
		name string
		fn   func(t *testing.T, s store.Store)
	}{
		{"AppCRUD", testAppCRUD},
		{"DeleteAppCascade", testDeleteAppCascade},
		{"UserEmailIsAppScoped", testUserEmailIsAppScoped},
		{"UserPhoneLookupIsEnvScoped", testUserPhoneLookupIsEnvScoped},
		{"ListUsersTotalAndFilter", testListUsersTotalAndFilter},
		{"ListUsersEmailMetacharsAreSafe", testListUsersEmailMetacharsAreSafe},
		{"SessionCRUD", testSessionCRUD},
		{"SessionLookupByTokenIsScoped", testSessionLookupByTokenIsScoped},
		{"SessionRolesRoundTrip", testSessionRolesRoundTrip},
		{"SessionAudienceRoundTrip", testSessionAudienceRoundTrip},
		{"SessionPrincipalKindRoundTrip", testSessionPrincipalKindRoundTrip},
		{"ServiceAccountSessionRoundTrip", testServiceAccountSessionRoundTrip},
		{"RotateSessionCAS", testRotateSessionCAS},
		{"RefreshTokenRevocation", testRefreshTokenRevocation},
		{"RefreshTokenReplayIsIdempotent", testRefreshTokenReplayIsIdempotent},
		{"OrgMemberLookupAndCascade", testOrgMemberLookupAndCascade},
		{"ListUserSessionsIsScopedToUser", testListUserSessionsIsScopedToUser},
		{"PrincipalRoundTrip", testPrincipalRoundTrip},
		{"EphemeralPrincipalExpiry", testEphemeralPrincipalExpiry},
		{"DelegationLifecycle", testDelegationLifecycle},
		{"SessionActorChainRoundTrip", testSessionActorChainRoundTrip},
		{"ServiceAccountKindDefaultsToService", testServiceAccountKindDefaultsToService},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if skipSet[tc.name] {
				t.Skip("skipped by caller")
			}
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

// seedSiblingEnv adds another environment to an existing app, so a test can
// exercise the boundary between two environments of the same tenant.
func seedSiblingEnv(t *testing.T, s store.Store, tn tenant) tenant {
	t.Helper()
	env := &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     tn.AppID,
		Name:      "Staging",
		Slug:      "staging-" + suffix(id.NewEnvironmentID().String()),
		Type:      environment.TypeStaging,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateEnvironment(context.Background(), env))
	return tenant{AppID: tn.AppID, EnvID: env.ID}
}

func seedUserWithPhone(t *testing.T, s store.Store, tn tenant, email, phone string) *user.User {
	t.Helper()
	u := &user.User{
		ID:            id.NewUserID(),
		AppID:         tn.AppID,
		EnvID:         tn.EnvID,
		Email:         email,
		Phone:         phone,
		PhoneVerified: true,
		CreatedAt:     now(),
		UpdatedAt:     now(),
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

// testUserPhoneLookupIsEnvScoped proves a phone lookup cannot reach across
// environments of the same app. Two environments of one app may legitimately
// hold different people behind the same number (a staging fixture and a real
// production account), and a caller scoped to one environment must never be
// handed the other's user. That is the hole that would let a staging Shared
// Signals stream revoke a production user's sessions.
func testUserPhoneLookupIsEnvScoped(t *testing.T, s store.Store) {
	ctx := context.Background()
	prod := seedTenant(t, s)
	staging := seedSiblingEnv(t, s, prod)

	const phone = "+15550001111"
	prodUser := seedUserWithPhone(t, s, prod, "prod@test.com", phone)
	stagingUser := seedUserWithPhone(t, s, staging, "staging@test.com", phone)
	require.NotEqual(t, prodUser.ID.String(), stagingUser.ID.String())

	gotProd, err := s.GetUserByPhone(ctx, prod.AppID, prod.EnvID, phone)
	require.NoError(t, err)
	assert.Equal(t, prodUser.ID.String(), gotProd.ID.String(),
		"lookup must return this environment's user")

	gotStaging, err := s.GetUserByPhone(ctx, staging.AppID, staging.EnvID, phone)
	require.NoError(t, err)
	assert.Equal(t, stagingUser.ID.String(), gotStaging.ID.String(),
		"lookup must be scoped per environment, not merely per app")

	// An environment holding no such user must not be handed one from a
	// sibling environment of the same app.
	bare := seedSiblingEnv(t, s, prod)
	_, err = s.GetUserByPhone(ctx, bare.AppID, bare.EnvID, phone)
	assert.ErrorIs(t, err, store.ErrNotFound,
		"phone lookup must not cross environments within an app")

	// A nil environment keeps the historical app-wide behaviour, matching
	// GetUserByAnyEmail, so a caller with no environment in hand still
	// resolves rather than silently finding nothing.
	anyEnv, err := s.GetUserByPhone(ctx, prod.AppID, id.Nil, phone)
	require.NoError(t, err)
	assert.Contains(t, []string{prodUser.ID.String(), stagingUser.ID.String()},
		anyEnv.ID.String(), "a nil env must still match app-wide")

	// The app boundary still holds alongside the new environment boundary.
	other := seedTenant(t, s)
	_, err = s.GetUserByPhone(ctx, other.AppID, other.EnvID, phone)
	assert.ErrorIs(t, err, store.ErrNotFound, "phone lookup must not cross apps")
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

// testSessionRolesRoundTrip proves a session's stamped roles survive
// persistence on every backend.
//
// Roles are resolved once, when the session is issued, and read back on every
// authenticated request to satisfy the role requirements a route declares. A
// backend that drops the field does not fail loudly: the session still
// authenticates, and every route declaring a role quietly refuses the user.
// That is the failure this test exists to catch, and it is the reason the
// empty case below is asserted as carefully as the populated one.
func testSessionRolesRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "roles-"+suffix(tn.AppID.String())+"@example.com")

	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		Token:                 "tok-roles-" + suffix(tn.AppID.String()),
		RefreshToken:          "ref-roles-" + suffix(tn.AppID.String()),
		FamilyID:              id.NewSessionFamilyID(),
		Roles:                 []string{"admin", "billing-viewer"},
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	for _, tc := range []struct {
		name string
		get  func() (*session.Session, error)
	}{
		{"GetSession", func() (*session.Session, error) { return s.GetSession(ctx, sess.ID) }},
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, sess.Token) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t, []string{"admin", "billing-viewer"}, got.Roles,
				"stamped roles did not survive the round trip")
		})
	}

	// A session with no roles must come back with none rather than with an
	// empty-string member: a backend encoding []string as a delimited list
	// gets this wrong, and "" is a role slug no principal can hold, which
	// turns into a permanent denial rather than a visible error.
	bare := seedSession(t, s, tn, u.ID, "tok-noroles-"+suffix(tn.AppID.String()), "ref-noroles-"+suffix(tn.AppID.String()))

	got, err := s.GetSession(ctx, bare.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Roles, "a roleless session came back carrying roles")
}

// testSessionAudienceRoundTrip proves a session's granted audience survives
// persistence on every backend.
//
// The audience is what stops a token issued for one resource server from
// authenticating at another, so a backend that drops the field does not fail
// loudly. It silently returns an unrestricted token, which is the exact
// confused-deputy hole RFC 8707 exists to close. The empty case matters just
// as much: a backend that encodes []string as a delimited list hands back a
// single empty-string member, and "" is an audience nobody can match.
func testSessionAudienceRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "aud-"+suffix(tn.AppID.String())+"@example.com")

	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		Token:                 "tok-aud-" + suffix(tn.AppID.String()),
		RefreshToken:          "ref-aud-" + suffix(tn.AppID.String()),
		FamilyID:              id.NewSessionFamilyID(),
		Audience:              []string{"https://api.example.com", "https://files.example.com"},
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	for _, tc := range []struct {
		name string
		get  func() (*session.Session, error)
	}{
		{"GetSession", func() (*session.Session, error) { return s.GetSession(ctx, sess.ID) }},
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, sess.Token) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t,
				[]string{"https://api.example.com", "https://files.example.com"},
				got.Audience,
				"granted audience did not survive the round trip")
		})
	}

	bare := seedSession(t, s, tn, u.ID,
		"tok-noaud-"+suffix(tn.AppID.String()),
		"ref-noaud-"+suffix(tn.AppID.String()))

	got, err := s.GetSession(ctx, bare.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Audience, "an unaudienced session came back carrying an audience")
}

// testSessionPrincipalKindRoundTrip pins the ordinary-user half of the
// principal contract: a session written with PrincipalKind set must come back
// with it intact, and one written without it must stay empty rather than being
// invented on read. Empty means "user" for backwards compatibility with rows
// predating the field (session.Session), so the store must not normalize.
func testSessionPrincipalKindRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "principal@test.com")

	explicit := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		Token:                 "tok-kind",
		RefreshToken:          "rtok-kind",
		FamilyID:              id.NewSessionFamilyID(),
		PrincipalKind:         principal.KindUser,
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, explicit))

	got, err := s.GetSession(ctx, explicit.ID)
	require.NoError(t, err)
	assert.Equal(t, principal.KindUser, got.PrincipalKind, "PrincipalKind must survive the round trip")
	assert.Equal(t, u.ID.String(), got.UserID.String())
	assert.True(t, got.ServiceAccountID.IsNil(), "a user session must not gain a ServiceAccountID")

	// A legacy row carries no principal kind; the store must return it as
	// written rather than filling in a default.
	legacy := seedSession(t, s, tn, u.ID, "tok-legacy", "rtok-legacy")
	gotLegacy, err := s.GetSession(ctx, legacy.ID)
	require.NoError(t, err)
	assert.Empty(t, gotLegacy.PrincipalKind, "an unset PrincipalKind must stay unset, not be defaulted")
}

// testServiceAccountSessionRoundTrip is the regression test for
// service-account sessions losing their principal identity in the store.
//
// A service-account session carries PrincipalKind="service_account" and a
// ServiceAccountID, and leaves UserID as the zero value — there is no user
// behind it. Everything downstream branches on PrincipalKind (see
// middleware/auth.go), so a store that drops these two fields hands back a
// session that is indistinguishable from an ordinary user session whose UserID
// happens to be zero. That is not a cosmetic loss: it is an authorization
// decision made on the wrong principal.
func testServiceAccountSessionRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)

	svcID := id.NewServiceAccountID()
	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		Token:                 "tok-svc",
		RefreshToken:          "rtok-svc",
		FamilyID:              id.NewSessionFamilyID(),
		PrincipalKind:         principal.KindService,
		ServiceAccountID:      svcID,
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess), "a service-account session must be persistable without a user")

	// Every read path must reconstruct the principal, not just the by-id one:
	// middleware resolves sessions by token, and refresh goes by refresh token.
	for _, tc := range []struct {
		name string
		get  func() (*session.Session, error)
	}{
		{"GetSession", func() (*session.Session, error) { return s.GetSession(ctx, sess.ID) }},
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, "tok-svc") }},
		{"GetSessionByRefreshToken", func() (*session.Session, error) { return s.GetSessionByRefreshToken(ctx, "rtok-svc") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t, principal.KindService, got.PrincipalKind, "PrincipalKind must survive the round trip")
			assert.Equal(t, svcID.String(), got.ServiceAccountID.String(), "ServiceAccountID must survive the round trip")
			assert.True(t, got.UserID.IsNil(), "a service-account session must not acquire a UserID")
		})
	}
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

// ──────────────────────────────────────────────────
// Principals and delegations
// ──────────────────────────────────────────────────

// seedPrincipal creates a non-human principal of the given kind.
func seedPrincipal(t *testing.T, s store.Store, tn tenant, kind principal.Kind, name string) *serviceaccount.ServiceAccount { //nolint:unparam // kind kept parameterised for future non-agent conformance cases
	t.Helper()
	svc := &serviceaccount.ServiceAccount{
		ID:        id.NewServiceAccountID(),
		AppID:     tn.AppID,
		EnvID:     tn.EnvID,
		Kind:      kind,
		Name:      name,
		Scopes:    []string{"repo:read"},
		Active:    true,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateServiceAccount(context.Background(), svc))
	return svc
}

// testPrincipalRoundTrip pins that the kind and owner survive persistence and
// that GetPrincipal resolves a stored row into the same ref it was written
// under. A backend that drops Kind hands back a principal that every
// authorization decision then treats as a plain service account.
func testPrincipalRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	owner := seedUser(t, s, tn, "owner@test.com")

	svc := seedPrincipal(t, s, tn, principal.KindAgent, "agent-one")
	svc.OwnerUserID = owner.ID
	require.NoError(t, s.UpdateServiceAccount(ctx, svc))

	got, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindAgent, ID: svc.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, principal.KindAgent, got.Kind)
	assert.Equal(t, svc.ID.String(), got.ID)
	assert.Equal(t, tn.AppID.String(), got.AppID.String())
	require.NotNil(t, got.Owner, "the owning user must survive the round trip")
	assert.Equal(t, principal.UserRef(owner.ID), *got.Owner)
	assert.True(t, got.IsActive(now()))

	// A user ref resolves too, out of the user table rather than this one.
	gotUser, err := s.GetPrincipal(ctx, principal.UserRef(owner.ID))
	require.NoError(t, err)
	assert.Equal(t, principal.KindUser, gotUser.Kind)
	assert.Equal(t, owner.ID.String(), gotUser.ID)

	// A row written with no kind at all reads as a service account, which is
	// what every row predating the column is.
	legacy := &serviceaccount.ServiceAccount{
		ID: id.NewServiceAccountID(), AppID: tn.AppID, Name: "legacy",
		Active: true, CreatedAt: now(), UpdatedAt: now(),
	}
	require.NoError(t, s.CreateServiceAccount(ctx, legacy))
	gotLegacy, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindService, ID: legacy.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, principal.KindService, gotLegacy.Kind)
}

// testServiceAccountKindDefaultsToService pins that every backend answers
// GetServiceAccount the same way for a row created with no Kind set.
// ToPrincipal's empty-Kind-means-service_account fallback covers a row some
// other tool wrote directly, but the store's own write path must not rely on
// it: Kind carries `json:"kind,omitempty"`, so a handler serializing the
// account would emit the key on a backend that normalizes at write time and
// omit it on one that stores the blank straight through. Without this pinned
// here, the two round trips already tested (ToPrincipal in
// testPrincipalRoundTrip, GetPrincipal's own fallback) can each pass while
// GetServiceAccount itself still disagrees across backends.
func testServiceAccountKindDefaultsToService(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)

	svc := &serviceaccount.ServiceAccount{
		ID:        id.NewServiceAccountID(),
		AppID:     tn.AppID,
		EnvID:     tn.EnvID,
		Name:      "no-kind-set",
		Active:    true,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateServiceAccount(ctx, svc))

	got, err := s.GetServiceAccount(ctx, svc.ID)
	require.NoError(t, err)
	assert.Equal(t, principal.KindService, got.Kind,
		"a service account created with no Kind must read back as service_account on every backend")
}

// testEphemeralPrincipalExpiry pins the JIT-minted child contract: the parent
// link survives, and a lapsed child is excluded from an active-only listing
// rather than silently continuing to authenticate.
func testEphemeralPrincipalExpiry(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	parent := seedPrincipal(t, s, tn, principal.KindAgent, "parent-agent")

	past := now().Add(-time.Hour)
	future := now().Add(time.Hour)

	lapsed := seedPrincipal(t, s, tn, principal.KindAgent, "lapsed-child")
	lapsed.ParentID = parent.ID
	lapsed.ExpiresAt = &past
	require.NoError(t, s.UpdateServiceAccount(ctx, lapsed))

	live := seedPrincipal(t, s, tn, principal.KindAgent, "live-child")
	live.ParentID = parent.ID
	live.ExpiresAt = &future
	require.NoError(t, s.UpdateServiceAccount(ctx, live))

	gotLapsed, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindAgent, ID: lapsed.ID.String()})
	require.NoError(t, err, "an expired principal must still be readable, so callers can report why")
	require.NotNil(t, gotLapsed.Parent)
	assert.Equal(t, parent.ID.String(), gotLapsed.Parent.ID)
	assert.False(t, gotLapsed.IsActive(now()), "an expired principal must not read as active")

	active, err := s.ListPrincipals(ctx, &principal.Query{
		AppID: tn.AppID, Kind: principal.KindAgent, ActiveOnly: true, ActiveAsOf: now(),
	})
	require.NoError(t, err)
	ids := make([]string, 0, len(active))
	for _, p := range active {
		ids = append(ids, p.ID)
	}
	assert.Contains(t, ids, live.ID.String())
	assert.Contains(t, ids, parent.ID.String())
	assert.NotContains(t, ids, lapsed.ID.String(), "an active-only listing must exclude the lapsed child")

	// The exact boundary: a principal whose ExpiresAt equals the ActiveAsOf
	// instant passed to the query is still active. principal.Principal.IsActive
	// defines active as `!at.After(expiresAt)`, which is inclusive at
	// equality, and every backend must agree with that method, not just with
	// each other, or a token minted right up to its ExpiresAt would start
	// failing mid-request depending only on which store answered it.
	//
	// The fixture's ExpiresAt and the query's ActiveAsOf below are the exact
	// same time.Time value, not two separate now() calls: two calls even a
	// few nanoseconds apart would make this test flaky for a reason that has
	// nothing to do with the boundary itself, and now() truncates to the
	// millisecond precisely because the SQL backends round-trip through their
	// own timestamp formats and would not preserve anything finer.
	boundary := now()
	boundaryChild := seedPrincipal(t, s, tn, principal.KindAgent, "boundary-child")
	boundaryChild.ParentID = parent.ID
	boundaryChild.ExpiresAt = &boundary
	require.NoError(t, s.UpdateServiceAccount(ctx, boundaryChild))

	atBoundary, err := s.ListPrincipals(ctx, &principal.Query{
		AppID: tn.AppID, Kind: principal.KindAgent, ActiveOnly: true, ActiveAsOf: boundary,
	})
	require.NoError(t, err)
	atBoundaryIDs := make([]string, 0, len(atBoundary))
	for _, p := range atBoundary {
		atBoundaryIDs = append(atBoundaryIDs, p.ID)
	}
	assert.Contains(t, atBoundaryIDs, boundaryChild.ID.String(),
		"a principal expiring at exactly ActiveAsOf must still be active, matching principal.Principal.IsActive")

	// The Parent filter itself: unexercised above, and easy for a backend to
	// get subtly wrong by comparing only the id half of the ref. ToPrincipal
	// stamps a child's Parent ref with the child's own Kind, so the query
	// below must match both children of `parent` and reject a ref that
	// carries the right id under the wrong kind.
	unrelated := seedPrincipal(t, s, tn, principal.KindAgent, "unrelated-agent")

	byParent, err := s.ListPrincipals(ctx, &principal.Query{
		AppID:  tn.AppID,
		Parent: &principal.Ref{Kind: principal.KindAgent, ID: parent.ID.String()},
	})
	require.NoError(t, err)
	byParentIDs := make([]string, 0, len(byParent))
	for _, p := range byParent {
		byParentIDs = append(byParentIDs, p.ID)
	}
	assert.Contains(t, byParentIDs, live.ID.String(), "the parent filter must return this child")
	assert.Contains(t, byParentIDs, lapsed.ID.String(), "the parent filter is not an active filter, so the lapsed child must still appear")
	assert.NotContains(t, byParentIDs, parent.ID.String(), "the parent itself must not appear as its own child")
	assert.NotContains(t, byParentIDs, unrelated.ID.String(), "a principal with a different parent must not appear")

	wrongKind, err := s.ListPrincipals(ctx, &principal.Query{
		AppID:  tn.AppID,
		Parent: &principal.Ref{Kind: principal.KindWorkload, ID: parent.ID.String()},
	})
	require.NoError(t, err)
	assert.Empty(t, wrongKind, "a parent ref naming the wrong kind must match nothing, even with a matching id")
}

// testDelegationLifecycle pins create, lookup and revoke. The revoke half is
// the one that matters: a grant that stays findable after revocation is a
// credential nobody can take away.
func testDelegationLifecycle(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "delegator@test.com")
	agent := seedPrincipal(t, s, tn, principal.KindAgent, "delegated-agent")

	actor := principal.Ref{Kind: principal.KindAgent, ID: agent.ID.String()}
	subject := principal.UserRef(u.ID)

	d := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     tn.AppID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		Scopes:    []string{"repo:read"},
		GrantedBy: subject,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateDelegation(ctx, d))

	got, err := s.GetDelegation(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, actor, got.Actor)
	assert.Equal(t, subject, got.Subject)
	assert.Equal(t, []string{"repo:read"}, got.Scopes)
	assert.True(t, got.IsActive(now()))

	found, err := s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantDelegation)
	require.NoError(t, err)
	assert.Equal(t, d.ID.String(), found.ID.String())

	// The wrong grant kind must not match. Impersonation and delegation are
	// evaluated differently, so crossing them changes the decision.
	_, err = s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantImpersonation)
	assert.ErrorIs(t, err, principal.ErrNotFound)

	listed, err := s.ListDelegations(ctx, &principal.DelegationQuery{AppID: tn.AppID, Subject: &subject})
	require.NoError(t, err)
	assert.Len(t, listed, 1, "the subject must be able to see what may act for them")

	// A second live grant for the same (app, actor, subject, kind) must be
	// rejected, not silently accepted alongside the first. Uniqueness here is
	// enforced by a partial index on live rows in the SQL backends and by an
	// explicit check in memory; a backend where this silently succeeds would
	// let one actor hold two live grants over the same subject, so revoking
	// the grant an admin can see would leave the other one still working.
	dupe := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     tn.AppID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		GrantedBy: subject,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	err = s.CreateDelegation(ctx, dupe)
	assert.ErrorIs(t, err, store.ErrConflict,
		"a true duplicate live grant must be rejected: memory checks explicitly, and every other backend maps its duplicate-key error to store.ErrConflict the same way")

	require.NoError(t, s.RevokeDelegation(ctx, d.ID, now()))
	_, err = s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantDelegation)
	assert.ErrorIs(t, err, principal.ErrNotFound, "a revoked grant must stop being findable")

	afterRevoke, err := s.GetDelegation(ctx, d.ID)
	require.NoError(t, err, "a revoked grant stays readable for audit")
	assert.NotNil(t, afterRevoke.RevokedAt)

	// Revoking twice is not an error. Revocation is the thing you want to
	// succeed on a retry.
	assert.NoError(t, s.RevokeDelegation(ctx, d.ID, now()))

	// A revoked grant must free the (app, actor, subject, kind) slot rather
	// than continue occupying it. The uniqueness constraint that backs this
	// is a partial index on live rows only, so a naive full-tuple index
	// would reject the fresh grant below and an agent whose grant was
	// revoked could never be re-granted.
	second := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     tn.AppID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		Scopes:    []string{"repo:read"},
		GrantedBy: subject,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateDelegation(ctx, second), "revoking the first grant must free the slot for a fresh one")

	foundSecond, err := s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantDelegation)
	require.NoError(t, err)
	assert.Equal(t, second.ID.String(), foundSecond.ID.String(), "the active grant must now be the fresh one, not the revoked one")

	// The exact boundary: a grant whose ExpiresAt equals the ActiveAsOf
	// instant passed to the query is still active. principal.Delegation.IsActive
	// defines active as `!at.After(expiresAt)`, inclusive at equality, and
	// ListDelegations must agree with that method, not just with the other
	// backends. A different grant kind (impersonation, not delegation) keeps
	// this from colliding with `second`'s live slot on the same actor/subject
	// pair.
	//
	// The fixture's ExpiresAt and the query's ActiveAsOf below are the exact
	// same time.Time value, not two separate now() calls: see the identical
	// note in testEphemeralPrincipalExpiry.
	boundary := now()
	boundaryDelegation := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     tn.AppID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantImpersonation,
		GrantedBy: subject,
		ExpiresAt: &boundary,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateDelegation(ctx, boundaryDelegation))

	atBoundary, err := s.ListDelegations(ctx, &principal.DelegationQuery{
		AppID: tn.AppID, Subject: &subject, ActiveOnly: true, ActiveAsOf: boundary,
	})
	require.NoError(t, err)
	atBoundaryIDs := make([]string, 0, len(atBoundary))
	for _, d := range atBoundary {
		atBoundaryIDs = append(atBoundaryIDs, d.ID.String())
	}
	assert.Contains(t, atBoundaryIDs, boundaryDelegation.ID.String(),
		"a grant expiring at exactly ActiveAsOf must still be active, matching principal.Delegation.IsActive")
}

// testSessionActorChainRoundTrip pins that the chain survives every session
// read path, not only the by-id one. Middleware resolves by token and refresh
// resolves by refresh token, and a chain lost on either of those is an
// authorization decision made against the wrong set of principals.
func testSessionActorChainRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "chained@test.com")

	delID := id.NewDelegationID()
	chain := principal.Chain{
		{Kind: principal.KindAgent, ID: "svc_child"},
		{Kind: principal.KindAgent, ID: "svc_parent"},
	}
	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		PrincipalKind:         principal.KindUser,
		Token:                 "tok-chain",
		RefreshToken:          "rtok-chain",
		FamilyID:              id.NewSessionFamilyID(),
		Actors:                chain,
		ActorGrant:            principal.GrantDelegation,
		DelegationID:          delID,
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	for _, tc := range []struct {
		name string
		get  func() (*session.Session, error)
	}{
		{"GetSession", func() (*session.Session, error) { return s.GetSession(ctx, sess.ID) }},
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, "tok-chain") }},
		{"GetSessionByRefreshToken", func() (*session.Session, error) { return s.GetSessionByRefreshToken(ctx, "rtok-chain") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t, chain, got.Actors, "the actor chain must survive the round trip")
			assert.Equal(t, principal.GrantDelegation, got.ActorGrant)
			assert.Equal(t, delID.String(), got.DelegationID.String())
			assert.True(t, got.ImpersonatedBy().IsNil(), "a delegation must not read back as impersonation")
		})
	}

	// The impersonation shape must round-trip too, through whichever column
	// the backend uses for it.
	admin := seedUser(t, s, tn, "admin@test.com")
	imp := seedSession(t, s, tn, u.ID, "tok-imp", "rtok-imp")
	imp.SetImpersonatedBy(admin.ID)
	require.NoError(t, s.UpdateSession(ctx, imp))

	gotImp, err := s.GetSession(ctx, imp.ID)
	require.NoError(t, err)
	assert.Equal(t, admin.ID.String(), gotImp.ImpersonatedBy().String())
}
