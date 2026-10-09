package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wardenid "github.com/xraph/warden/id"
	wardenrole "github.com/xraph/warden/role"

	authsome "github.com/xraph/authsome"
)

// seedChildRole creates a role in the seeded "app" namespace whose parent is
// given by slug, the way warden links roles. It carries no permissions of
// its own, so everything it confers comes from the parent chain.
func seedChildRole(t *testing.T, eng *authsome.Engine, slug, parentSlug string) string {
	t.Helper()
	return seedRoleIn(t, eng, "app", slug, parentSlug)
}

// seedRoleIn is seedChildRole for any namespace. It hands back the new
// role's own id in the authsome form, since a slug lookup searches several
// namespaces and could land on a different role with the same slug.
func seedRoleIn(t *testing.T, eng *authsome.Engine, ns, slug, parentSlug string) string {
	t.Helper()
	r := &wardenrole.Role{
		ID:            wardenid.NewRoleID(),
		TenantID:      testAppIDStr,
		AppID:         testAppIDStr,
		NamespacePath: ns,
		Name:          slug,
		Slug:          slug,
		ParentSlug:    parentSlug,
	}
	require.NoError(t, eng.Warden().Store().CreateRole(context.Background(), r))
	_, suffix, _ := strings.Cut(r.ID.String(), "_")
	return "arol_" + suffix
}

// A role read back from the store carries warden's id form, which the walk
// has to accept, or the ceiling sees no permissions at all, not even the
// role's own.
func TestAssignRole_AdminCannotGrantDirectPermissions(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "direct-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	admin := appAdmin(t, eng, "direct-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "direct-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	rec := sendJSON(t, handler, eng, owner, http.MethodPost, "/v1/roles", `{"name":"Tuner","slug":"tuner"}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	tuner := roleIDBySlug(t, eng, "tuner")
	rec = postJSON(t, handler, eng, owner, "/v1/roles/"+tuner+"/permissions", `{"action":"read","resource":"settings"}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	rec = postJSON(t, handler, eng, admin, "/v1/roles/"+tuner+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "tuner holds settings:read itself; body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "settings")
}

// A child of owner confers settings:*, which an admin does not hold. The
// ceiling has to see that through the parent, not just the child's own
// (empty) permission list.
func TestAssignRole_AdminCannotGrantInheritedPermissions(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "inherit-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "inherit-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	deputy := seedChildRole(t, eng, "deputy", "owner")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+deputy+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "deputy inherits settings:* from owner; body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "settings")
}

// A child of user confers only what an admin already holds, so resolving
// the parent must not turn a legitimate grant into a refusal.
func TestAssignRole_AdminGrantsChildOfHeldRole(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "inherit-ok-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "inherit-ok-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	member := seedChildRole(t, eng, "member", "user")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+member+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "an admin holds everything user confers; body=%s", rec.Body.String())
}

// Two roles that name each other as parent must not hang the walk, and the
// grant is still judged on what the pair confers.
func TestAssignRole_ParentCycleTerminates(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "inherit-cycle-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "inherit-cycle-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	ping := seedChildRole(t, eng, "ping", "pong")
	seedChildRole(t, eng, "pong", "ping")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+ping+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "the cycle confers nothing an admin lacks; body=%s", rec.Body.String())
}

// Warden finds a parent by slug in the child's own namespace. A decoy owner
// in the root namespace, holding nothing, must not stand in for the real
// owner in "app" that deputy inherits from.
func TestAssignRole_ParentResolvesInChildNamespace(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "ns-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "ns-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	seedRoleIn(t, eng, "", "owner", "")
	deputy := seedChildRole(t, eng, "deputy", "owner")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+deputy+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "deputy inherits the app owner's settings:*; body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "settings")
}

// The other direction: a root-namespace role naming owner as its parent
// inherits nothing, because warden never looks in "app" for it. Charging
// the grant with the app owner's permissions would refuse a role that
// confers nothing.
func TestAssignRole_ParentInOtherNamespaceConfersNothing(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "ns-other-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "ns-other-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	stray := seedRoleIn(t, eng, "", "stray", "owner")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+stray+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "warden resolves no parent for stray; body=%s", rec.Body.String())
}
