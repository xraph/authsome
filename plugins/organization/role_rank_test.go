package organization_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// The membership and invitation writes required only org admin and then took
// the role from the request body. So an admin could invite an alias -- or add
// an existing account -- as owner, and remove the owner. Granting a role, and
// removing someone, must not reach above the caller's own rank.

func postJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestCreateInvitation_AdminCannotInviteOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	for _, role := range []string{"owner", "Owner", " OWNER "} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/invitations",
			postJSON(t, map[string]string{"email": "alias@example.com", "role": role}), admin))
		assert.Equal(t, http.StatusForbidden, rec.Code, "role %q: body=%s", role, rec.Body.String())
	}
}

func TestCreateInvitation_AdminMayInviteAdminAndBelow(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	// Free-form roles some apps use (e.g. "viewer") still work; they outrank no one.
	for i, role := range []string{"admin", "member", "viewer", ""} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/invitations",
			postJSON(t, map[string]string{"email": "p" + string(rune('a'+i)) + "@example.com", "role": role}), admin))
		assert.Equal(t, http.StatusCreated, rec.Code, "role %q: body=%s", role, rec.Body.String())
	}
}

func TestCreateInvitation_OwnerMayInviteOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/invitations",
		postJSON(t, map[string]string{"email": "co-owner@example.com", "role": "owner"}), owner))
	assert.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

func TestAddMember_AdminCannotAddOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	alias := id.NewUserID()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/members",
		postJSON(t, map[string]string{"user_id": alias.String(), "role": "Owner"}), admin))
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

func TestAddMember_AdminMayAddAdmin(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	newcomer := id.NewUserID()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+o.ID.String()+"/members",
		postJSON(t, map[string]string{"user_id": newcomer.String(), "role": "admin"}), admin))
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, organization.RoleAdmin, memberRole(t, p, o.ID, newcomer))
}

func TestRemoveMember_AdminCannotRemoveOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	ownerMember := memberRecord(t, p, o.ID, owner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+o.ID.String()+"/members/"+ownerMember.ID.String(), nil, admin))
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, organization.RoleOwner, memberRole(t, p, o.ID, owner), "the owner must still be a member")
}

func TestRemoveMember_AdminMayRemoveMemberAndAdmin(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	for _, role := range []organization.MemberRole{organization.RoleMember, organization.RoleAdmin} {
		target := addMember(t, p, o.ID, id.NewUserID(), role)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+o.ID.String()+"/members/"+target.ID.String(), nil, admin))
		assert.Equal(t, http.StatusOK, rec.Code, "removing a %s: body=%s", role, rec.Body.String())
	}
}

func TestRemoveMember_OwnerMayRemoveOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	coOwner := addMember(t, p, o.ID, id.NewUserID(), organization.RoleOwner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+o.ID.String()+"/members/"+coOwner.ID.String(), nil, owner))
	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// memberRecord returns userID's membership record in orgID.
func memberRecord(t *testing.T, p *orgplugin.Plugin, orgID id.OrgID, userID id.UserID) *organization.Member {
	t.Helper()
	members, err := p.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	for _, m := range members {
		if m.UserID.String() == userID.String() {
			return m
		}
	}
	t.Fatalf("user %s is not a member of org %s", userID, orgID)
	return nil
}
