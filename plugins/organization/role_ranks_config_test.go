package organization_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// A product with its own role names (wakflo: builder, operator, viewer,
// guest) ranked every one of them 0, below member, so none could read the
// member list. Config.RoleRanks gives them a rank; names it does not list
// stay at 0 and the built-in three never move.
func TestOrgRoleRank_customRolesFromConfig(t *testing.T) {
	p, _ := newOrgHTTPWith(t, orgplugin.Config{RoleRanks: map[string]int{
		"builder": 1, "Operator": 1, " viewer ": 1, "guest": 0,
		"owner": 0, "admin": 0, "member": 0, // built-ins are not overridable
	}})
	for role, want := range map[organization.MemberRole]int{
		"owner": 3, "admin": 2, "member": 1,
		"builder": 1, "BUILDER": 1, "operator": 1, "viewer": 1, "guest": 0,
		"stranger": 0, "": 0,
	} {
		assert.Equal(t, want, p.RoleRankForTest(role), "role %q", role)
	}

	d, _ := newOrgHTTPWith(t, orgplugin.Config{})
	assert.Equal(t, 0, d.RoleRankForTest("builder"), "unconfigured custom roles keep rank 0")
}

func listMembersStatus(t *testing.T, cfg orgplugin.Config, role organization.MemberRole) int {
	t.Helper()
	p, h := newOrgHTTPWith(t, cfg)
	o := seedOrg(t, p, id.NewUserID())
	caller := id.NewUserID()
	addMember(t, p, o.ID, caller, role)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/"+o.ID.String()+"/members", nil, caller))
	return rec.Code
}

func TestListMembers_customRoleRankedAsMemberCanList(t *testing.T) {
	cfg := orgplugin.Config{RoleRanks: map[string]int{"builder": 1, "guest": 0}}
	assert.Equal(t, http.StatusOK, listMembersStatus(t, cfg, "builder"))
	assert.Equal(t, http.StatusForbidden, listMembersStatus(t, cfg, "guest"))
	// Not configured: refused, as before the option existed.
	assert.Equal(t, http.StatusForbidden, listMembersStatus(t, orgplugin.Config{}, "builder"))
	assert.Equal(t, http.StatusForbidden, listMembersStatus(t, cfg, "stranger"))
}

// A role ranked as member passes member checks only. Every write that needs
// admin or owner still refuses it.
func TestRoleRanks_memberRankedRoleStillCannotAdminister(t *testing.T) {
	p, h := newOrgHTTPWith(t, orgplugin.Config{RoleRanks: map[string]int{"builder": 1}})
	o := seedOrg(t, p, id.NewUserID())
	builder := id.NewUserID()
	addMember(t, p, o.ID, builder, "builder")
	target := addMember(t, p, o.ID, id.NewUserID(), organization.RoleMember)

	base := "/v1/orgs/" + o.ID.String()
	for _, c := range []struct {
		method, path string
		body         []byte
	}{
		{"POST", base + "/invitations", postJSON(t, map[string]string{"email": "x@example.com"})},
		{"POST", base + "/members", postJSON(t, map[string]string{"user_id": id.NewUserID().String(), "role": "member"})},
		{"PATCH", base + "/members/" + target.ID.String(), postJSON(t, map[string]string{"role": "admin"})},
		{"DELETE", base + "/members/" + target.ID.String(), nil},
		{"DELETE", base, nil},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, orgReq(c.method, c.path, c.body, builder))
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", c.method, c.path, rec.Body.String())
	}
}
