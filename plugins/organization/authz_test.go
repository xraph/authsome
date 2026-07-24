package organization_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

const orgTestAppID = "aapp_01jf0000000000000000000000"

// newOrgHTTP builds an initialized org plugin and mounts its routes on a fresh
// router, returning the plugin (for seeding) and the HTTP handler.
func newOrgHTTP(t *testing.T) (*orgplugin.Plugin, http.Handler) {
	t.Helper()
	p := orgplugin.New()
	_ = secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return p, router.Handler()
}

// seedOrg creates an organization owned by ownerID (CreateOrganization also
// adds ownerID as an owner member).
func seedOrg(t *testing.T, p *orgplugin.Plugin, ownerID id.UserID) *organization.Organization {
	t.Helper()
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)
	now := time.Now()
	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Acme",
		Slug:      "acme",
		CreatedBy: ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, p.CreateOrganization(context.Background(), o))
	return o
}

// memberRole returns the role of userID within orgID, failing if not a member.
func memberRole(t *testing.T, p *orgplugin.Plugin, orgID id.OrgID, userID id.UserID) organization.MemberRole {
	t.Helper()
	members, err := p.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	for _, m := range members {
		if m.UserID.String() == userID.String() {
			return m.Role
		}
	}
	t.Fatalf("user %s is not a member of org %s", userID, orgID)
	return ""
}

// addMember adds userID to orgID with the given role and returns the member.
func addMember(t *testing.T, p *orgplugin.Plugin, orgID id.OrgID, userID id.UserID, role organization.MemberRole) *organization.Member {
	t.Helper()
	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
	}
	require.NoError(t, p.AddMember(context.Background(), m))
	return m
}

// orgReq builds a request, optionally authenticated as userID (pass the zero
// UserID for an unauthenticated request).
func orgReq(method, target string, body []byte, userID id.UserID) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequestWithContext(context.Background(), method, target, nil)
	}
	if !userID.IsNil() {
		r = r.WithContext(middleware.WithUserID(r.Context(), userID))
	}
	return r
}

// ──────────────────────────────────────────────────
// GET /orgs/:orgId — read requires membership
// ──────────────────────────────────────────────────

func TestOrgGet_RequiresAuth(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/"+o.ID.String(), nil, id.UserID{}))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrgGet_RejectsNonMember(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	stranger := id.NewUserID()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/"+o.ID.String(), nil, stranger))

	assert.Equal(t, http.StatusNotFound, rec.Code, "a non-member must not see the org")
}

func TestOrgGet_MemberSucceeds(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	member := id.NewUserID()
	addMember(t, p, o.ID, member, organization.RoleMember)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/"+o.ID.String(), nil, member))

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// PATCH /orgs/:orgId — update requires admin
// ──────────────────────────────────────────────────

func TestOrgUpdate_RequiresAuth(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", "/v1/orgs/"+o.ID.String(), []byte(`{"name":"Hacked"}`), id.UserID{}))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrgUpdate_RejectsMember(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	member := id.NewUserID()
	addMember(t, p, o.ID, member, organization.RoleMember)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", "/v1/orgs/"+o.ID.String(), []byte(`{"name":"Hacked"}`), member))

	assert.Equal(t, http.StatusForbidden, rec.Code)

	got, err := p.GetOrganization(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", got.Name, "a plain member must not rename the org")
}

func TestOrgUpdate_AdminSucceeds(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", "/v1/orgs/"+o.ID.String(), []byte(`{"name":"Renamed"}`), admin))

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// DELETE /orgs/:orgId — delete requires owner
// ──────────────────────────────────────────────────

func TestOrgDelete_RejectsAdmin(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+o.ID.String(), nil, admin))

	assert.Equal(t, http.StatusForbidden, rec.Code, "only an owner may delete an org")

	_, err := p.GetOrganization(context.Background(), o.ID)
	assert.NoError(t, err, "org must survive an admin's delete attempt")
}

func TestOrgDelete_OwnerSucceeds(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+o.ID.String(), nil, owner))

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// Members — add requires admin; role change requires owner; no self-escalation
// ──────────────────────────────────────────────────

func TestListMembers_RequiresAuth(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/"+o.ID.String()+"/members", nil, id.UserID{}))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAddMember_RejectsMember(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	member := id.NewUserID()
	addMember(t, p, o.ID, member, organization.RoleMember)

	body := []byte(`{"user_id":"` + id.NewUserID().String() + `","role":"member"}`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/members", body, member))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAddMember_AdminSucceeds(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	body := []byte(`{"user_id":"` + id.NewUserID().String() + `","role":"member"}`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/members", body, admin))

	assert.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

// TestUpdateMember_SelfEscalation_Rejected is the headline case: a plain member
// must not be able to promote themselves (or anyone) to owner.
func TestUpdateMember_SelfEscalation_Rejected(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	attacker := id.NewUserID()
	m := addMember(t, p, o.ID, attacker, organization.RoleMember)

	body := []byte(`{"role":"owner"}`)
	target := "/v1/orgs/" + o.ID.String() + "/members/" + m.ID.String()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", target, body, attacker))

	assert.Equal(t, http.StatusForbidden, rec.Code, "a member must not escalate their own role")

	assert.Equal(t, organization.RoleMember, memberRole(t, p, o.ID, attacker), "attacker's role must be unchanged")
}

func TestUpdateMember_RejectsAdmin(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	victim := id.NewUserID()
	vm := addMember(t, p, o.ID, victim, organization.RoleMember)

	body := []byte(`{"role":"admin"}`)
	target := "/v1/orgs/" + o.ID.String() + "/members/" + vm.ID.String()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", target, body, admin))

	assert.Equal(t, http.StatusForbidden, rec.Code, "only an owner may change member roles")
}

func TestUpdateMember_OwnerSucceeds(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	victim := id.NewUserID()
	vm := addMember(t, p, o.ID, victim, organization.RoleMember)

	body := []byte(`{"role":"admin"}`)
	target := "/v1/orgs/" + o.ID.String() + "/members/" + vm.ID.String()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", target, body, owner))

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// Teams — create requires admin
// ──────────────────────────────────────────────────

func TestCreateTeam_RejectsMember(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	member := id.NewUserID()
	addMember(t, p, o.ID, member, organization.RoleMember)

	body := []byte(`{"name":"Eng","slug":"eng"}`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/teams", body, member))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestUpdateMember_CrossOrg_Rejected verifies an owner of one org cannot mutate
// a member record that belongs to a different org by supplying a foreign
// memberId under their own org's path.
func TestUpdateMember_CrossOrg_Rejected(t *testing.T) {
	p, h := newOrgHTTP(t)

	ownerA := id.NewUserID()
	orgA := seedOrg(t, p, ownerA)

	ownerB := id.NewUserID()
	// Second org for the same app needs a distinct slug.
	appID, _ := id.ParseAppID(orgTestAppID)
	now := time.Now()
	orgB := &organization.Organization{
		ID: id.NewOrgID(), AppID: appID, Name: "Beta", Slug: "beta",
		CreatedBy: ownerB, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, p.CreateOrganization(context.Background(), orgB))
	victim := id.NewUserID()
	vm := addMember(t, p, orgB.ID, victim, organization.RoleMember)

	// ownerA targets orgB's member via orgA's path.
	body := []byte(`{"role":"owner"}`)
	target := "/v1/orgs/" + orgA.ID.String() + "/members/" + vm.ID.String()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("PATCH", target, body, ownerA))

	assert.Equal(t, http.StatusNotFound, rec.Code, "a member from another org must not be reachable")

	assert.Equal(t, organization.RoleMember, memberRole(t, p, orgB.ID, victim), "victim's role must be unchanged")
}
