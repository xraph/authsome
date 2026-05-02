package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	forgecontrib "github.com/xraph/forge/extensions/dashboard/contributor"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// stubPermChecker returns a fixed allow/deny answer for one (action, resource,
// userID) tuple. Lets us exercise canDeleteOrg's permission-check fallback
// without booting warden.
type stubPermChecker struct {
	allow      bool
	wantAction string
	wantRes    string
	wantUser   id.UserID
	calls      int
}

func (s *stubPermChecker) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	s.calls++
	if action != s.wantAction || resource != s.wantRes || userID != s.wantUser {
		return false, nil
	}
	return s.allow, nil
}

// orgTestSetup builds a fresh engine + org plugin + buffered chronicle and
// seeds an organization owned by `owner`. Returns everything tests need.
type orgTestSetup struct {
	eng    *authsome.Engine
	plugin *orgplugin.Plugin
	ch     *secutil.BufferedChronicle
	org    *organization.Organization
	owner  id.UserID
}

func newOrgTestSetup(t *testing.T) *orgTestSetup {
	t.Helper()
	secutil.InitTestNonceSigner(t)

	p := orgplugin.New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))

	ch := secutil.NewBufferedChronicle()
	secutil.AttachChronicle(t, eng, ch)

	owner := id.NewUserID()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Acme",
		Slug:      "acme",
		CreatedBy: owner,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, p.CreateOrganization(context.Background(), o))

	return &orgTestSetup{eng: eng, plugin: p, ch: ch, org: o, owner: owner}
}

// renderDelete invokes DashboardRenderPage("/organizations/detail") with a
// POST-style action=delete payload. Returns the actionError surfaced by the
// handler (extracted via a side channel: re-render on failure path returns
// the org-detail page; on success it returns the org-list page).
func (s *orgTestSetup) submitDelete(t *testing.T, ctx context.Context, nonce string) (returnedListPage bool) {
	t.Helper()
	params := forgecontrib.Params{
		PathParams:  map[string]string{"id": s.org.ID.String()},
		QueryParams: map[string]string{},
		FormData: map[string]string{
			"action": "delete",
			"nonce":  nonce,
		},
	}
	comp, err := s.plugin.DashboardRenderPage(ctx, "/organizations/detail", params)
	require.NoError(t, err)
	require.NotNil(t, comp)
	// On the success path renderOrgDetail returns renderOrgList output. We
	// can't easily distinguish by templ type, so callers verify the org's
	// store presence instead.
	_, exists := s.orgExists(t)
	return !exists
}

func (s *orgTestSetup) orgExists(t *testing.T) (*organization.Organization, bool) {
	t.Helper()
	o, err := s.plugin.GetOrganization(context.Background(), s.org.ID)
	if err != nil {
		// Not-found is a clean signal here.
		var notFound interface{ Error() string }
		if errors.As(err, &notFound) {
			return nil, false
		}
		return nil, false
	}
	return o, o != nil
}

func ctxAs(actor id.UserID, sessionID id.SessionID) context.Context {
	ctx := context.Background()
	ctx = middleware.WithUserID(ctx, actor)
	ctx = middleware.WithSessionID(ctx, sessionID)
	return ctx
}

// ─── tests ──────────────────────────────────────────────────────────────────

func TestOrgDelete_RequiresOwnerOrAdmin(t *testing.T) {
	s := newOrgTestSetup(t)

	// Stranger: non-owner, no permission checker installed.
	stranger := id.NewUserID()
	strangerSess := id.NewSessionID()
	nonce := dashboard.GenerateScopedNonce(strangerSess.String(), "org.delete")
	require.NotEmpty(t, nonce)

	s.submitDelete(t, ctxAs(stranger, strangerSess), nonce)

	if _, ok := s.orgExists(t); !ok {
		t.Fatalf("stranger delete should have been refused; org is gone")
	}
	secutil.AssertNoAuditEvent(t, s.ch, "org.delete")

	// Owner submits with a fresh nonce on their own session.
	ownerSess := id.NewSessionID()
	ownerNonce := dashboard.GenerateScopedNonce(ownerSess.String(), "org.delete")
	s.submitDelete(t, ctxAs(s.owner, ownerSess), ownerNonce)

	if _, ok := s.orgExists(t); ok {
		t.Fatalf("owner delete should have removed the org")
	}
}

// spyPermChecker records the (action, resource) pair it was called with and
// returns a fixed allow. Used to assert canonical RBAC arg shapes.
type spyPermChecker struct {
	allow    bool
	gotUser  id.UserID
	gotAct   string
	gotRes   string
	calls    int
}

func (s *spyPermChecker) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	s.calls++
	s.gotUser = userID
	s.gotAct = action
	s.gotRes = resource
	return s.allow, nil
}

// TestOrgDelete_PermissionCheckUsesResourceType pins the codebase RBAC
// convention: HasPermission's third arg is the resource TYPE ("org"), not
// the instance ID. See middleware/rbac.go and rbac/warden_store.go.
func TestOrgDelete_PermissionCheckUsesResourceType(t *testing.T) {
	s := newOrgTestSetup(t)

	admin := id.NewUserID()
	spy := &spyPermChecker{allow: true}
	s.plugin.SetPermCheckerForTest(spy)

	sess := id.NewSessionID()
	nonce := dashboard.GenerateScopedNonce(sess.String(), "org.delete")
	s.submitDelete(t, ctxAs(admin, sess), nonce)

	require.Equal(t, 1, spy.calls, "permission checker must be consulted exactly once")
	assert.Equal(t, admin, spy.gotUser)
	assert.Equal(t, "org.delete", spy.gotAct)
	assert.Equal(t, "org", spy.gotRes,
		"resource arg must be the TYPE %q, not the instance ID; codebase convention is rbac/warden_store.go forwards this as warden.Resource.Type",
		"org")

	if _, ok := s.orgExists(t); ok {
		t.Fatalf("admin with allow=true should have deleted org")
	}
}

func TestOrgDelete_AdminWithPermissionCanDelete(t *testing.T) {
	s := newOrgTestSetup(t)

	admin := id.NewUserID()
	s.plugin.SetPermCheckerForTest(&stubPermChecker{
		allow:      true,
		wantAction: "org.delete",
		// Resource is the TYPE per the codebase RBAC convention — see
		// rbac/warden_store.go (forwarded as warden.Resource.Type).
		wantRes:  "org",
		wantUser: admin,
	})

	sess := id.NewSessionID()
	nonce := dashboard.GenerateScopedNonce(sess.String(), "org.delete")
	s.submitDelete(t, ctxAs(admin, sess), nonce)

	if _, ok := s.orgExists(t); ok {
		t.Fatalf("admin with org.delete permission should have deleted org")
	}
}

func TestOrgDelete_AuditEventOnSuccess(t *testing.T) {
	s := newOrgTestSetup(t)

	sess := id.NewSessionID()
	nonce := dashboard.GenerateScopedNonce(sess.String(), "org.delete")
	s.submitDelete(t, ctxAs(s.owner, sess), nonce)

	secutil.AssertAuditEvent(t, s.ch, "org.delete", func(ev *bridge.AuditEvent) {
		assert.Equal(t, bridge.SeverityCritical, ev.Severity)
		assert.Equal(t, bridge.OutcomeSuccess, ev.Outcome)
		assert.Equal(t, s.owner.String(), ev.ActorID)
		assert.Equal(t, s.org.ID.String(), ev.ResourceID)
		assert.Equal(t, s.org.Slug, ev.Metadata["slug"])
		assert.Equal(t, s.org.AppID.String(), ev.Metadata["app_id"])
	})
}

func TestOrgDelete_NoAuditEventOnRejection(t *testing.T) {
	s := newOrgTestSetup(t)

	stranger := id.NewUserID()
	sess := id.NewSessionID()
	nonce := dashboard.GenerateScopedNonce(sess.String(), "org.delete")
	s.submitDelete(t, ctxAs(stranger, sess), nonce)

	secutil.AssertNoAuditEvent(t, s.ch, "org.delete")
	if _, ok := s.orgExists(t); !ok {
		t.Fatalf("rejected delete must not remove the org")
	}
}

func TestOrgDelete_ScopedNonceRequired(t *testing.T) {
	s := newOrgTestSetup(t)

	sess := id.NewSessionID()
	// Mint nonce for the WRONG scope.
	wrongScope := dashboard.GenerateScopedNonce(sess.String(), "org.create")
	s.submitDelete(t, ctxAs(s.owner, sess), wrongScope)
	if _, ok := s.orgExists(t); !ok {
		t.Fatalf("wrong-scope nonce must not consume; org should remain")
	}
	secutil.AssertNoAuditEvent(t, s.ch, "org.delete")

	// Mint nonce for a DIFFERENT session.
	otherSess := id.NewSessionID()
	otherNonce := dashboard.GenerateScopedNonce(otherSess.String(), "org.delete")
	s.submitDelete(t, ctxAs(s.owner, sess), otherNonce)
	if _, ok := s.orgExists(t); !ok {
		t.Fatalf("cross-session nonce must not consume; org should remain")
	}
	secutil.AssertNoAuditEvent(t, s.ch, "org.delete")
}

func TestAttack_CSRF_StolenScopedNonce_OrgDelete(t *testing.T) {
	s := newOrgTestSetup(t)

	// Attacker steals nonce minted under user-A's session.
	victimSess := id.NewSessionID()
	stolen := dashboard.GenerateScopedNonce(victimSess.String(), "org.delete")
	require.NotEmpty(t, stolen)

	// Attacker submits with their own session context.
	attacker := id.NewUserID()
	attackerSess := id.NewSessionID()
	s.submitDelete(t, ctxAs(attacker, attackerSess), stolen)

	if _, ok := s.orgExists(t); !ok {
		t.Fatalf("CSRF replay across sessions must be rejected; org is gone")
	}
	secutil.AssertNoAuditEvent(t, s.ch, "org.delete")
}
