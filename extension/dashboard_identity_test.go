package extension

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/dashboard"
	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
	"github.com/xraph/forge/extensions/dashboard/contract/transport"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/rbac"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// The dashboard identity
//
// CheckAuth used to answer with Subject, DisplayName, Email and AvatarURL and
// nothing else. Forge copies UserInfo.Claims onto every contract principal
// and checks `requires` predicates against UserInfo.Roles and Scopes, so
// with all three empty every app-scoped contributor took its fallback and
// every gated intent was denied to everyone.
// ──────────────────────────────────────────────────

// identityFixture is a bootstrapped engine with a platform app and one tenant
// app, plus a checker over it.
type identityFixture struct {
	eng      *authsome.Engine
	checker  *authChecker
	platform id.AppID
	tenant   id.AppID
}

func newIdentityFixture(t *testing.T, contributors ...string) *identityFixture {
	t.Helper()
	ctx := context.Background()

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	cfg := authsome.DefaultConfig()
	cfg.Password.BcryptCost = bcrypt.MinCost

	eng, err := authsome.NewEngine(authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
		authsome.WithBootstrap(),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })
	secutil.RelaxAuthDefaults(t, eng)

	platform := eng.PlatformAppID()
	require.False(t, platform.IsNil(), "bootstrap should create the platform app")

	tenant := &app.App{Name: "Tenant", Slug: "tenant"}
	require.NoError(t, eng.CreateApp(ctx, tenant))

	return &identityFixture{
		eng: eng,
		checker: &authChecker{
			engine: eng,
			scopes: dashboardScopes{contributors: func() []string { return contributors }},
		},
		platform: platform,
		tenant:   tenant.ID,
	}
}

// signUp creates a user in appID and returns the user's ID and session token.
// The email is never verified, so bootstrap never promotes the user to owner.
func (f *identityFixture) signUp(t *testing.T, appID id.AppID, email string) (id.UserID, string) {
	t.Helper()
	u, sess, err := f.eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss123",
		FirstName: "Op",
	})
	require.NoError(t, err)
	return u.ID, sess.Token
}

// grant assigns the role with slug in appID to the user. Assignment restamps
// the user's live sessions, so the next CheckAuth sees it.
func (f *identityFixture) grant(t *testing.T, appID id.AppID, uid id.UserID, slug string) {
	t.Helper()
	ctx := context.Background()
	role, err := f.eng.GetRoleBySlug(ctx, appID, slug)
	require.NoError(t, err)
	require.NotNil(t, role)
	require.NoError(t, f.eng.AssignUserRole(ctx, appID, &rbac.UserRole{UserID: uid.String(), RoleID: role.ID}))
}

// defaultEnv returns the default environment bootstrap created for appID.
func (f *identityFixture) defaultEnv(t *testing.T, appID id.AppID) id.EnvironmentID {
	t.Helper()
	envs, err := f.eng.ListEnvironments(context.Background(), appID)
	require.NoError(t, err)
	for _, e := range envs {
		if e.IsDefault {
			return e.ID
		}
	}
	require.NotEmpty(t, envs, "bootstrap should create environments for %s", appID)
	return envs[0].ID
}

// check runs CheckAuth for token with the given switcher cookies (name→value).
func (f *identityFixture) check(t *testing.T, token string, cookies map[string]string) *dashauth.UserInfo {
	t.Helper()
	req := dashCheckerRequest(t, token, "")
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	info, err := f.checker.CheckAuth(context.Background(), req)
	require.NoError(t, err)
	return info
}

func TestDashboardIdentity_AppIDComesFromTheSession(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	_, token := f.signUp(t, f.platform, "plain@example.com")

	info := f.check(t, token, nil)

	require.NotNil(t, info)
	assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
	_, hasTenant := info.Claims["tenant_id"]
	assert.False(t, hasTenant, "tenant_id means different things to Chronicle and Warden, so it is never set")
	_, hasOrg := info.Claims[claimOrgID]
	assert.False(t, hasOrg, "a session with no org carries no org_id; a present-but-empty tenant key makes Chronicle refuse")
}

func TestDashboardIdentity_TenantSessionIsScopedToItsApp(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	_, token := f.signUp(t, f.tenant, "tenant-op@example.com")

	info := f.check(t, token, nil)

	require.NotNil(t, info)
	assert.Equal(t, f.tenant.String(), info.Claims[claimAppID])
}

func TestDashboardIdentity_SwitchHonouredForPlatformAdmin(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	uid, token := f.signUp(t, f.platform, "padmin@example.com")
	f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)

	info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()})

	require.NotNil(t, info)
	assert.Equal(t, f.tenant.String(), info.Claims[claimAppID])
	_, hasEnv := info.Claims[claimEnvID]
	assert.False(t, hasEnv, "the session's environment belongs to the platform app and means nothing in the tenant")
}

func TestDashboardIdentity_SwitchHonouredForPlatformOwner(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	uid, token := f.signUp(t, f.platform, "powner@example.com")
	f.grant(t, f.platform, uid, rbac.PlatformOwnerSlug)

	info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()})

	require.NotNil(t, info)
	assert.Equal(t, f.tenant.String(), info.Claims[claimAppID])
}

// TestDashboardIdentity_SwitchNeverWidensScope covers every way the cookie
// can name an app the session is not entitled to. Each one must leave the
// operator exactly where the session put them.
func TestDashboardIdentity_SwitchNeverWidensScope(t *testing.T) {
	t.Parallel()

	t.Run("platform-user is the default sign-up role and grants no switch", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		_, token := f.signUp(t, f.platform, "puser@example.com") // platform-user by default

		info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()})

		require.NotNil(t, info)
		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
	})

	t.Run("an app-level admin administers its own app only", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		uid, token := f.signUp(t, f.platform, "appadmin@example.com")
		f.grant(t, f.platform, uid, rbac.AppAdminSlug)

		info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()})

		require.NotNil(t, info)
		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
	})

	t.Run("a tenant owner cannot switch into the platform app", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		uid, token := f.signUp(t, f.tenant, "towner@example.com")
		f.grant(t, f.tenant, uid, rbac.AppOwnerSlug)

		info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: f.platform.String()})

		require.NotNil(t, info)
		assert.Equal(t, f.tenant.String(), info.Claims[claimAppID])
	})

	t.Run("a platform admin cannot switch into an app that does not exist", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		uid, token := f.signUp(t, f.platform, "padmin-ghost@example.com")
		f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)

		info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: id.NewAppID().String()})

		require.NotNil(t, info)
		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
	})

	t.Run("an unparseable cookie is ignored", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		uid, token := f.signUp(t, f.platform, "padmin-junk@example.com")
		f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)

		info := f.check(t, token, map[string]string{authcontract.AppSwitcherCookie: "not-an-app-id"})

		require.NotNil(t, info)
		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
	})

	t.Run("a revoked platform admin loses the switch on the next request", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t)
		uid, token := f.signUp(t, f.platform, "padmin-revoked@example.com")
		f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)
		cookies := map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()}
		require.Equal(t, f.tenant.String(), f.check(t, token, cookies).Claims[claimAppID])

		role, err := f.eng.GetRoleBySlug(context.Background(), f.platform, rbac.PlatformAdminSlug)
		require.NoError(t, err)
		roleID, err := id.ParseRoleID("arol_" + role.ID[strings.IndexByte(role.ID, '_')+1:])
		require.NoError(t, err)
		require.NoError(t, f.eng.UnassignUserRole(context.Background(), f.platform, uid, roleID))

		assert.Equal(t, f.platform.String(), f.check(t, token, cookies).Claims[claimAppID])
	})
}

func TestDashboardIdentity_EnvironmentFollowsTheResolvedApp(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	uid, token := f.signUp(t, f.platform, "padmin-env@example.com")
	f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)
	tenantEnv := f.defaultEnv(t, f.tenant)
	platformEnv := f.defaultEnv(t, f.platform)

	t.Run("an environment of the switched app is honoured", func(t *testing.T) {
		info := f.check(t, token, map[string]string{
			authcontract.AppSwitcherCookie: f.tenant.String(),
			authcontract.EnvSwitcherCookie: tenantEnv.String(),
		})
		require.NotNil(t, info)
		assert.Equal(t, tenantEnv.String(), info.Claims[claimEnvID])
	})

	t.Run("an environment of another app is ignored", func(t *testing.T) {
		info := f.check(t, token, map[string]string{
			authcontract.AppSwitcherCookie: f.tenant.String(),
			authcontract.EnvSwitcherCookie: platformEnv.String(),
		})
		require.NotNil(t, info)
		_, hasEnv := info.Claims[claimEnvID]
		assert.False(t, hasEnv)
	})

	t.Run("an environment of an app the switch was refused for is ignored", func(t *testing.T) {
		_, plainToken := f.signUp(t, f.platform, "plain-env@example.com")
		info := f.check(t, plainToken, map[string]string{
			authcontract.AppSwitcherCookie: f.tenant.String(),
			authcontract.EnvSwitcherCookie: tenantEnv.String(),
		})
		require.NotNil(t, info)
		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
		assert.NotEqual(t, tenantEnv.String(), info.Claims[claimEnvID])
	})
}

func TestDashboardIdentity_RolesAndScopes(t *testing.T) {
	t.Parallel()

	t.Run("an admin gets <contributor>.admin for every contributor", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t, "chronicle", "authsome")
		uid, token := f.signUp(t, f.platform, "scoped-admin@example.com")
		f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)

		info := f.check(t, token, nil)

		require.NotNil(t, info)
		assert.Contains(t, info.Roles, rbac.PlatformAdminSlug)
		assert.Equal(t, []string{"authsome.admin", "chronicle.admin"}, info.Scopes)
	})

	t.Run("an app owner is an admin of its own app", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t, "chronicle")
		uid, token := f.signUp(t, f.tenant, "tenant-owner@example.com")
		f.grant(t, f.tenant, uid, rbac.AppOwnerSlug)

		info := f.check(t, token, nil)

		require.NotNil(t, info)
		assert.Contains(t, info.Roles, rbac.AppOwnerSlug)
		assert.Equal(t, []string{"chronicle.admin"}, info.Scopes)
		assert.Equal(t, f.tenant.String(), info.Claims[claimAppID], "the scope is bounded by the app claim")
	})

	t.Run("a non-admin gets no admin scopes", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t, "chronicle")
		_, token := f.signUp(t, f.platform, "scoped-user@example.com") // platform-user by default

		info := f.check(t, token, nil)

		require.NotNil(t, info)
		assert.Contains(t, info.Roles, rbac.PlatformUserSlug)
		assert.Empty(t, info.Scopes)
	})

	t.Run("RoleScopes adds explicit grants", func(t *testing.T) {
		t.Parallel()
		f := newIdentityFixture(t, "chronicle")
		f.checker.scopes.roleScopes = map[string][]string{rbac.PlatformUserSlug: {"chronicle.admin", "reports.read"}}
		_, token := f.signUp(t, f.platform, "auditor@example.com") // platform-user by default

		info := f.check(t, token, nil)

		require.NotNil(t, info)
		assert.Equal(t, []string{"chronicle.admin", "reports.read"}, info.Scopes)
	})
}

// TestDashboardIdentity_NoSessionNoIdentity: the claims are only ever built
// on top of a resolved session. A switcher cookie on its own is nothing.
func TestDashboardIdentity_NoSessionNoIdentity(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t, "chronicle")

	info := f.check(t, "not-a-session-token", map[string]string{authcontract.AppSwitcherCookie: f.tenant.String()})

	assert.Nil(t, info)
}

// ──────────────────────────────────────────────────
// Through forge's transport
//
// The point of all of the above is that forge's contract transport evaluates
// an intent's `requires` predicate against this identity. Drive a real
// transport handler, with the checker in front of it the way the dashboard
// mounts it, against a manifest shaped like Chronicle's.
// ──────────────────────────────────────────────────

const gatedManifest = `
schemaVersion: 1
contributor: { name: chronicle, envelope: { supports: [v1], preferred: v1 } }
intents:
  - { name: retention.enforce, kind: command, version: 1, capability: write, requires: { all: ["scope:chronicle.admin"] } }
`

// principalRecorder is a transport.Dispatcher that answers OK and keeps the
// principal it was handed.
type principalRecorder struct {
	got *dashcontract.Principal
}

func (p *principalRecorder) Dispatch(_ context.Context, _ dashcontract.Request, pr dashcontract.Principal) (json.RawMessage, dashcontract.ResponseMeta, error) {
	p.got = &pr
	return json.RawMessage(`{"ok":true}`), dashcontract.ResponseMeta{}, nil
}

func newGatedTransport(t *testing.T, f *identityFixture) (http.Handler, *principalRecorder) {
	t.Helper()

	m, err := loader.Load(strings.NewReader(gatedManifest), "chronicle.yaml")
	require.NoError(t, err)
	reg := dashcontract.NewRegistry()
	require.NoError(t, reg.Register(m))

	// Scopes come from the same registry the transport dispatches against,
	// the way RegisterContractContributor wires it.
	var src contributorSource
	src.set(reg)
	f.checker.scopes.contributors = src.names

	rec := &principalRecorder{}
	h := transport.NewHandler(reg, dashcontract.NewWardenRegistry(), rec, nil)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := f.checker.CheckAuth(r.Context(), r)
		require.NoError(t, err)
		ctx := r.Context()
		if info != nil {
			ctx = dashauth.WithUser(ctx, info)
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	}), rec
}

func enforceRequest(t *testing.T, token string) *http.Request {
	t.Helper()
	body, err := json.Marshal(dashcontract.Request{
		Envelope: "v1", Kind: dashcontract.KindCommand,
		Contributor: "chronicle", Intent: "retention.enforce", IntentVersion: 1,
		CSRF: "csrf", IdempotencyKey: "idem",
	})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/dashboard/v1", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: dashboardCookieName, Value: token})
	return req
}

func TestDashboardIdentity_TransportAllowsTheScopeHolder(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	h, rec := newGatedTransport(t, f)
	uid, token := f.signUp(t, f.platform, "enforcer@example.com")
	f.grant(t, f.platform, uid, rbac.PlatformAdminSlug)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, enforceRequest(t, token))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, rec.got)
	assert.Equal(t, f.platform.String(), rec.got.Claims[claimAppID], "the app claim reaches the handler's principal")
}

func TestDashboardIdentity_TransportDeniesWithoutTheScope(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	h, rec := newGatedTransport(t, f)
	_, token := f.signUp(t, f.platform, "bystander@example.com") // platform-user by default

	w := httptest.NewRecorder()
	h.ServeHTTP(w, enforceRequest(t, token))

	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), string(dashcontract.CodePermissionDenied))
	assert.Nil(t, rec.got, "a denied intent never reaches its handler")
}

// ──────────────────────────────────────────────────
// Client mode
// ──────────────────────────────────────────────────

func TestClientAuthChecker_ClaimsComeFromIntrospection(t *testing.T) {
	t.Parallel()

	appID := id.NewAppID().String()
	envID := id.NewEnvironmentID().String()
	orgID := id.NewOrgID().String()
	userID := id.NewUserID().String()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(authclient.IntrospectResponse{
			Active: true, UserID: userID, AppID: appID, EnvID: envID, OrgID: orgID,
			SessionID: id.NewSessionID().String(),
			User:      &authclient.IntrospectUser{ID: userID, Email: "client@test.com"},
		})
	}))
	t.Cleanup(identity.Close)

	c := newDashClientChecker(t, "unused", "")
	c.client = authclient.NewClient(identity.URL)

	req := dashCheckerRequest(t, "client-token", "")
	// A switcher cookie cannot be verified in client mode, so it is ignored.
	req.AddCookie(&http.Cookie{Name: authcontract.AppSwitcherCookie, Value: id.NewAppID().String()})

	info, err := c.CheckAuth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, map[string]any{claimAppID: appID, claimEnvID: envID, claimOrgID: orgID}, info.Claims)
	assert.Empty(t, info.Roles, "introspection carries no roles")
	assert.Empty(t, info.Scopes)
}

// TestDashboardIdentity_OrgNeverTravelsWithoutApp: an org belongs to one app,
// so org_id is only ever sent next to app_id. Chronicle refuses a tenant
// claim with no app outright.
func TestDashboardIdentity_OrgNeverTravelsWithoutApp(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)
	u := &user.User{ID: id.NewUserID(), Email: "org@example.com"}
	orgID := id.NewOrgID()

	t.Run("a session in an org carries org_id beside app_id", func(t *testing.T) {
		sess := &session.Session{ID: id.NewSessionID(), AppID: f.platform, UserID: u.ID, OrgID: orgID}

		info := f.checker.dashboardIdentity(context.Background(), dashCheckerRequest(t, "unused", ""), sess, u)

		assert.Equal(t, f.platform.String(), info.Claims[claimAppID])
		assert.Equal(t, orgID.String(), info.Claims[claimOrgID])
	})

	t.Run("a session with no app carries no org_id", func(t *testing.T) {
		sess := &session.Session{ID: id.NewSessionID(), UserID: u.ID, OrgID: orgID}

		info := f.checker.dashboardIdentity(context.Background(), dashCheckerRequest(t, "unused", ""), sess, u)

		assert.Empty(t, info.Claims)
	})
}

func TestClientAuthChecker_OrgNeverTravelsWithoutApp(t *testing.T) {
	t.Parallel()

	claims := introspectionClaims(&authclient.IntrospectResponse{
		Active: true, EnvID: id.NewEnvironmentID().String(), OrgID: id.NewOrgID().String(),
	})

	assert.Empty(t, claims)
}

// ──────────────────────────────────────────────────
// Wiring
// ──────────────────────────────────────────────────

// TestRegisterDashboardAuth_InstallsTheIdentityChecker: the checker the
// dashboard gets reads contributor names from the registry
// RegisterContractContributor captured, and role scopes from config.
func TestRegisterDashboardAuth_InstallsTheIdentityChecker(t *testing.T) {
	t.Parallel()
	f := newIdentityFixture(t)

	m, err := loader.Load(strings.NewReader(gatedManifest), "chronicle.yaml")
	require.NoError(t, err)
	reg := dashcontract.NewRegistry()
	require.NoError(t, reg.Register(m))

	e := New()
	e.SetLogger(forge.NewNoopLogger())
	e.engine = f.eng
	e.config.Dashboard.RoleScopes = map[string][]string{"auditor": {"chronicle.admin"}}
	e.config.Dashboard.RequiredRoles = []string{rbac.PlatformAdminSlug}
	e.dashContributors.set(reg)

	dashExt, ok := dashboard.NewExtension().(*dashboard.Extension)
	require.True(t, ok)
	e.RegisterDashboardAuth(dashExt)

	checker, ok := dashExt.AuthChecker().(*authChecker)
	require.True(t, ok, "engine mode installs the engine-backed checker, got %T", dashExt.AuthChecker())
	assert.Same(t, f.eng, checker.engine)
	assert.Equal(t, []string{"chronicle"}, checker.scopes.contributors())
	assert.Equal(t, e.config.Dashboard.RoleScopes, checker.scopes.roleScopes)
	assert.NotNil(t, dashExt.TenantResolver())
}

func TestContributorSource_NoRegistryNoNames(t *testing.T) {
	t.Parallel()
	var src contributorSource
	assert.Nil(t, src.names(), "before RegisterContractContributor runs there is nothing to grant")
}

func TestCookieID_NilRequest(t *testing.T) {
	t.Parallel()
	got, ok := cookieID(nil, authcontract.AppSwitcherCookie, func(s string) (id.ID, error) {
		t.Fatalf("parse must not run without a request, got %q", s)
		return id.ID{}, nil
	})
	assert.False(t, ok)
	assert.True(t, got.IsNil())
}

// TestPlatformAppID_FallsBackToConfiguredAppID: before bootstrap names a
// platform app, the configured AppID stands in for it.
func TestPlatformAppID_FallsBackToConfiguredAppID(t *testing.T) {
	t.Parallel()
	want := id.NewAppID()
	cfg := authsome.DefaultConfig()
	cfg.AppID = want.String()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	eng, err := authsome.NewEngine(
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
	)
	require.NoError(t, err)
	require.True(t, eng.PlatformAppID().IsNil(), "no bootstrap, no platform app")

	assert.Equal(t, want, platformAppID(eng))
}
