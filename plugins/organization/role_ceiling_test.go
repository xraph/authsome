package organization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// An org admin may add admins and members, never owners; owners cannot be
// removed by admins; the last owner stays. Free-form roles are covered in
// role_rank_test.go.

func TestOrgAddMember_AdminCannotGrantOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	body := []byte(`{"user_id":"` + id.NewUserID().String() + `","role":"owner"}`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq(http.MethodPost, "/v1/orgs/"+o.ID.String()+"/members", body, admin))

	assert.Equal(t, http.StatusForbidden, rec.Code, "an admin must not mint an owner; body=%s", rec.Body.String())
}

func TestOrgInvitation_AdminCannotInviteOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)

	body := []byte(`{"email":"new-owner@example.com","role":"owner"}`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq(http.MethodPost, "/v1/orgs/"+o.ID.String()+"/invitations", body, admin))

	assert.Equal(t, http.StatusForbidden, rec.Code, "an admin must not invite an owner; body=%s", rec.Body.String())
}

func TestOrgRemoveMember_AdminCannotRemoveOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	ownerMemberID := memberIDFor(t, p, o.ID, owner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq(http.MethodDelete, "/v1/orgs/"+o.ID.String()+"/members/"+ownerMemberID.String(), nil, admin))

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, organization.RoleOwner, memberRole(t, p, o.ID, owner))
}

func TestOrgRemoveMember_LastOwnerStays(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	ownerMemberID := memberIDFor(t, p, o.ID, owner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq(http.MethodDelete, "/v1/orgs/"+o.ID.String()+"/members/"+ownerMemberID.String(), nil, owner))

	assert.Equal(t, http.StatusConflict, rec.Code, "the last owner must not be removable; body=%s", rec.Body.String())
}

func TestOrgRemoveMember_OwnerRemovesOtherOwner(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	second := id.NewUserID()
	m := addMember(t, p, o.ID, second, organization.RoleOwner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq(http.MethodDelete, "/v1/orgs/"+o.ID.String()+"/members/"+m.ID.String(), nil, owner))

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// memberIDFor returns the membership id of userID within orgID.
func memberIDFor(t *testing.T, p *orgplugin.Plugin, orgID id.OrgID, userID id.UserID) id.MemberID {
	t.Helper()
	members, err := p.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	for _, m := range members {
		if m.UserID.String() == userID.String() {
			return m.ID
		}
	}
	t.Fatalf("user %s is not a member of org %s", userID, orgID)
	return id.MemberID{}
}
