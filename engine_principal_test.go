package authsome_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/warden"
	wardenassign "github.com/xraph/warden/assignment"
	wardenid "github.com/xraph/warden/id"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/serviceaccount"
)

// Delegation narrows and never widens. Each row is one of the four
// combinations of (subject allowed, actor allowed), and only the case where
// both allow may pass.
func TestCanIntersectsSubjectAndActor(t *testing.T) {
	for _, tc := range []struct {
		name         string
		subjectAllow bool
		actorAllow   bool
		wantAllowed  bool
	}{
		{"both allow", true, true, true},
		{"actor denied", true, false, false},
		{"subject denied", false, true, false},
		{"both denied", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, userID, agentRef := setupCanFixture(t, tc.subjectAllow, tc.actorAllow)

			got, err := e.Can(context.Background(),
				principal.UserRef(userID),
				principal.Chain{agentRef},
				"read", "document")
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllowed, got)
		})
	}
}

// With no chain the behaviour must be byte-identical to a plain subject check,
// which is what every existing caller relies on.
func TestCanWithEmptyChainIsASingleCheck(t *testing.T) {
	e, userID, _ := setupCanFixture(t, true, false)

	got, err := e.Can(context.Background(), principal.UserRef(userID), nil, "read", "document")
	require.NoError(t, err)
	assert.True(t, got, "an empty chain must not consult any actor")
}

// HasPermission must keep answering exactly as it did, since it is on the
// plugin.PermissionChecker contract.
func TestHasPermissionDelegatesToCan(t *testing.T) {
	e, userID, _ := setupCanFixture(t, true, false)

	got, err := e.HasPermission(context.Background(), userID, "read", "document")
	require.NoError(t, err)
	assert.True(t, got)
}

// A multi-hop chain checks every hop. An ephemeral child acting through a
// parent that has lost the permission must be denied, or revoking the parent
// would leave its children running.
func TestCanChecksEveryHop(t *testing.T) {
	e, userID, childRef, parentRef := setupMultiHopFixture(t,
		true /*subject*/, true /*child*/, false /*parent*/)

	got, err := e.Can(context.Background(),
		principal.UserRef(userID),
		principal.Chain{childRef, parentRef},
		"read", "document")
	require.NoError(t, err)
	assert.False(t, got, "a denied parent must deny the child acting through it")
}

// A child may not out-scope its parent. Otherwise minting a child is a
// privilege escalation with an extra step.
func TestMintChildRefusesScopesTheParentLacks(t *testing.T) {
	e, _, parent := setupParentFixture(t, []string{"repo:read"})

	_, _, _, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", []string{"repo:write"}, time.Hour)
	assert.Error(t, err)
}

// A child may not outlive its parent, or revoking the parent leaves its
// children running.
func TestMintChildCapsTTLByParentExpiry(t *testing.T) {
	e, _, parent := setupParentFixtureExpiring(t, 5*time.Minute)

	child, _, _, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", nil, 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, child.ExpiresAt)
	assert.False(t, child.ExpiresAt.After(*parent.ExpiresAt))
}

func TestMintChildRecordsTheParent(t *testing.T) {
	e, appID, parent := setupParentFixture(t, nil)

	child, key, secret, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", nil, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, parent.ID.String(), child.ParentID.String())
	assert.Equal(t, appID.String(), child.AppID.String())
	assert.NotEmpty(t, secret, "the secret is returned once and never stored")
	assert.Equal(t, child.ID.String(), key.ServiceAccountID.String())
}

func TestReapRemovesExpiredChildrenOnly(t *testing.T) {
	e, appID, parent := setupParentFixture(t, nil)
	ctx := context.Background()

	lapsed, _, _, err := e.MintChildPrincipal(ctx, parent.ID, "lapsed", nil, -time.Hour)
	require.NoError(t, err)

	n, err := e.ReapExpiredPrincipals(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = e.GetServiceAccount(ctx, lapsed.ID)
	assert.Error(t, err, "the lapsed child must be gone")
	_, err = e.GetServiceAccount(ctx, parent.ID)
	assert.NoError(t, err, "the durable parent must survive the reaper")
}

// A durable principal that happens to carry its own expiry must not be
// swept: an operator who time-limited a registration they can see in the
// dashboard did not consent to it being deleted out from under them. Only
// rows with ParentID set (minted children) are the reaper's business.
func TestReapLeavesExpiredNonChildrenAlone(t *testing.T) {
	e, s := newTestEngine(t)
	appID := testTenantID(t, e)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	durable := &serviceaccount.ServiceAccount{
		ID:        id.NewServiceAccountID(),
		AppID:     appID,
		Name:      "durable-but-expired",
		Kind:      principal.KindWorkload,
		Active:    true,
		ExpiresAt: &past,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateServiceAccount(ctx, durable))

	n, err := e.ReapExpiredPrincipals(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "a durable principal with its own expiry is not a child and must not be reaped")

	_, err = e.GetServiceAccount(ctx, durable.ID)
	assert.NoError(t, err, "the durable expired principal must survive the reaper")
}

// One level only. An ephemeral child must not itself be able to mint a
// grandchild: a tree of ephemeral principals is a revocation problem nobody
// can reason about.
func TestMintChildRefusesFromAnEphemeralParent(t *testing.T) {
	e, _, parent := setupParentFixture(t, nil)
	ctx := context.Background()

	child, _, _, err := e.MintChildPrincipal(ctx, parent.ID, "child", nil, time.Hour)
	require.NoError(t, err)

	_, _, _, err = e.MintChildPrincipal(ctx, child.ID, "grandchild", nil, time.Hour)
	assert.Error(t, err, "an ephemeral principal must not be able to mint children of its own")
}

// ──────────────────────────────────────────────────
// fixtures
// ──────────────────────────────────────────────────

// setupParentFixture builds an engine with a durable, non-expiring parent
// service account carrying the given scopes.
func setupParentFixture(t *testing.T, scopes []string) (*authsome.Engine, id.AppID, *serviceaccount.ServiceAccount) {
	t.Helper()
	eng, s := newTestEngine(t)
	appID := testTenantID(t, eng)

	svcID := id.NewServiceAccountID()
	now := time.Now()
	parent := &serviceaccount.ServiceAccount{
		ID:        svcID,
		AppID:     appID,
		Name:      "parent-" + svcID.String(),
		Kind:      principal.KindAgent,
		Scopes:    scopes,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateServiceAccount(context.Background(), parent))
	return eng, appID, parent
}

// setupParentFixtureExpiring is setupParentFixture but the parent itself
// carries an expiry, ttl out from now, so a minted child can be proven
// capped by the PARENT's expiry rather than by its own requested TTL.
func setupParentFixtureExpiring(t *testing.T, ttl time.Duration) (*authsome.Engine, id.AppID, *serviceaccount.ServiceAccount) {
	t.Helper()
	eng, s := newTestEngine(t)
	appID := testTenantID(t, eng)

	svcID := id.NewServiceAccountID()
	now := time.Now()
	expires := now.Add(ttl)
	parent := &serviceaccount.ServiceAccount{
		ID:        svcID,
		AppID:     appID,
		Name:      "parent-" + svcID.String(),
		Kind:      principal.KindAgent,
		Active:    true,
		ExpiresAt: &expires,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateServiceAccount(context.Background(), parent))
	return eng, appID, parent
}

// canFixtureSeq keeps every role slug created across these fixtures unique,
// since Warden enforces (tenant, namespace, slug) uniqueness and t.Name()
// alone is not distinct enough once a fixture grants more than one ref.
var canFixtureSeq atomic.Int64

// wardenSubjectKindForTest mirrors engine_principal.go's wardenSubjectKind: a
// human user maps to warden.SubjectUser, everything else (agent, workload,
// service) collapses onto warden.SubjectServiceAcct. Duplicated here rather
// than exported, since the mapping is an internal implementation detail of
// Can and these tests only need to construct fixtures matching it.
func wardenSubjectKindForTest(k principal.Kind) warden.SubjectKind {
	if k == principal.KindUser || k == "" {
		return warden.SubjectUser
	}
	return warden.SubjectServiceAcct
}

// newActorRef mints a principal ref of the given kind with a fresh, unique ID.
func newActorRef(kind principal.Kind) principal.Ref {
	return principal.Ref{Kind: kind, ID: id.New(id.PrefixServiceAccount).String()}
}

// testTenantID returns the app ID Can's checks will actually resolve to for
// an engine built by newTestEngine.
//
// newTestEngine wires WithAppID but never a bootstrap config, so
// Engine.PlatformAppID stays nil: ensureWardenScope's platform-app branch
// never fires and it falls through to e.config.AppID instead. Fixtures must
// grant roles under that same app ID or they grant them in a tenant Can never
// looks at.
func testTenantID(t *testing.T, eng *authsome.Engine) id.AppID {
	t.Helper()
	appID, err := id.ParseAppID(eng.Config().AppID)
	require.NoError(t, err)
	return appID
}

// grantOrDeny sets up ref's standing in Warden for the "read document"
// permission this suite checks.
//
// Every ref gets a role and a real assignment in Warden, whether or not it is
// meant to pass: the role always carries a decoy permission unrelated to
// "read document", and only carries "read document" itself when allow is
// true. That way a "denied" party is denied because Warden evaluated its
// roles and found no matching grant, not because Warden had nothing on file
// for it at all. A fixture that left the denied side of the table completely
// unconfigured would pass even if Can never called Warden.
func grantOrDeny(t *testing.T, eng *authsome.Engine, appID id.AppID, ref principal.Ref, allow bool) {
	t.Helper()
	ctx := context.Background()

	seq := canFixtureSeq.Add(1)
	role := &rbac.Role{
		ID:    id.NewRoleID().String(),
		AppID: appID.String(),
		Name:  fmt.Sprintf("can-fixture-role-%d", seq),
		Slug:  fmt.Sprintf("can-fixture-role-%d", seq),
	}
	require.NoError(t, eng.CreateRole(ctx, role))

	// Decoy permission: always present, so the role (and the assignment
	// binding it to ref) is real regardless of allow.
	require.NoError(t, eng.AddPermission(ctx, &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   role.ID,
		Action:   "noop",
		Resource: "decoy",
	}))

	if allow {
		require.NoError(t, eng.AddPermission(ctx, &rbac.Permission{
			ID:       id.NewPermissionID().String(),
			RoleID:   role.ID,
			Action:   "read",
			Resource: "document",
		}))
	}

	// CreateRole rewrites role.ID in place to Warden's own "role_..." format,
	// so it parses directly here rather than through authsome's "arol_..."
	// conversion helper (which is unexported outside the rbac package).
	wRoleID, err := wardenid.ParseRoleID(role.ID)
	require.NoError(t, err)

	require.NoError(t, eng.Warden().Store().CreateAssignment(ctx, &wardenassign.Assignment{
		ID:          wardenid.NewAssignmentID(),
		TenantID:    appID.String(),
		AppID:       appID.String(),
		RoleID:      wRoleID,
		SubjectKind: string(wardenSubjectKindForTest(ref.Kind)),
		SubjectID:   ref.ID,
		CreatedAt:   time.Now(),
	}))
}

// setupCanFixture builds an engine with a user subject and a single agent
// actor, granting or withholding "read document" on each per the arguments.
func setupCanFixture(t *testing.T, subjectAllow, actorAllow bool) (*authsome.Engine, id.UserID, principal.Ref) {
	t.Helper()
	eng, _ := newTestEngine(t)
	appID := testTenantID(t, eng)

	userID := id.NewUserID()
	agentRef := newActorRef(principal.KindAgent)

	grantOrDeny(t, eng, appID, principal.UserRef(userID), subjectAllow)
	grantOrDeny(t, eng, appID, agentRef, actorAllow)

	return eng, userID, agentRef
}

// setupMultiHopFixture builds an engine with a user subject and a two-hop
// actor chain: an ephemeral child acting through a registered parent.
func setupMultiHopFixture(t *testing.T, subjectAllow, childAllow, parentAllow bool) (
	eng *authsome.Engine, userID id.UserID, childRef, parentRef principal.Ref,
) {
	t.Helper()
	eng, _ = newTestEngine(t)
	appID := testTenantID(t, eng)

	userID = id.NewUserID()
	childRef = newActorRef(principal.KindAgent)
	parentRef = newActorRef(principal.KindWorkload)

	grantOrDeny(t, eng, appID, principal.UserRef(userID), subjectAllow)
	grantOrDeny(t, eng, appID, childRef, childAllow)
	grantOrDeny(t, eng, appID, parentRef, parentAllow)

	return eng, userID, childRef, parentRef
}
