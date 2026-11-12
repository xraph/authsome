package organization_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// Resend and revoke act on an invitation that already exists. Resend rotates
// the token and restarts the clock, revoke retires the row but keeps it so the
// audit trail has something to point at.

func invitationURL(orgID id.OrgID, invID id.InvitationID, suffix string) string {
	return "/v1/orgs/" + orgID.String() + "/invitations/" + invID.String() + suffix
}

func resendInvitation(t *testing.T, h http.Handler, orgID id.OrgID, invID id.InvitationID, caller id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("POST", invitationURL(orgID, invID, "/resend"), nil, caller))
	return rec
}

func revokeInvitation(t *testing.T, h http.Handler, orgID id.OrgID, invID id.InvitationID, caller id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", invitationURL(orgID, invID, ""), nil, caller))
	return rec
}

// sentLongAgo seeds an invitation that was sent two minutes ago, outside the
// resend limit.
func sentLongAgo(i *organization.Invitation) {
	i.CreatedAt = time.Now().Add(-2 * time.Minute)
}

func TestResendInvitation_rotatesTokenAndExtendsExpiry(t *testing.T) {
	p, h := newOrgHTTPWith(t, orgplugin.Config{InvitationTTL: 48 * time.Hour})
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-old", func(i *organization.Invitation) {
		sentLongAgo(i)
		i.ExpiresAt = time.Now().Add(time.Hour)
	})

	rec := resendInvitation(t, h, o.ID, inv.ID, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var got organization.Invitation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, inv.ID.String(), got.ID.String())
	assert.Equal(t, organization.InvitationPending, got.Status)
	assert.WithinDuration(t, time.Now().Add(48*time.Hour), got.ExpiresAt, time.Minute)
	assert.NotContains(t, rec.Body.String(), "tok-old")

	// The link that went out first stops working, and says nothing about why.
	assert.Equal(t, http.StatusNotFound, getPreview(t, h, "tok-old").Code)
	unknown := getPreview(t, h, "tok-never-existed")
	old := getPreview(t, h, "tok-old")
	assert.Equal(t, unknown.Code, old.Code)
	assert.Equal(t, unknown.Body.String(), old.Body.String())
}

func TestResendInvitation_newTokenWorksAndOldOneDoesNot(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-first", sentLongAgo)

	renewed, err := p.ResendInvitation(context.Background(), o.ID, inv.ID)
	require.NoError(t, err)
	require.NotEmpty(t, renewed.Token)
	assert.NotEqual(t, "tok-first", renewed.Token)

	assert.Equal(t, http.StatusNotFound, getPreview(t, h, "tok-first").Code)
	assert.Equal(t, http.StatusOK, getPreview(t, h, renewed.Token).Code)
}

func TestResendInvitation_fromExpiredRevivesIt(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-lapsed", func(i *organization.Invitation) {
		sentLongAgo(i)
		i.Status = organization.InvitationExpired
		i.ExpiresAt = time.Now().Add(-time.Hour)
	})

	rec := resendInvitation(t, h, o.ID, inv.ID, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got organization.Invitation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, organization.InvitationPending, got.Status)
	assert.True(t, got.ExpiresAt.After(time.Now()))
}

func TestResendInvitation_acceptedIs409(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	for _, status := range []organization.InvitationStatus{
		organization.InvitationAccepted, organization.InvitationDeclined, organization.InvitationRevoked,
	} {
		t.Run(string(status), func(t *testing.T) {
			inv := seedInvitation(t, p, o.ID, owner, "tok-"+string(status), func(i *organization.Invitation) {
				sentLongAgo(i)
				i.Status = status
			})
			rec := resendInvitation(t, h, o.ID, inv.ID, owner)
			assert.Equal(t, http.StatusConflict, rec.Code, "body=%s", rec.Body.String())
		})
	}
}

func TestResendInvitation_withinAMinuteIs429(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	// Just created, so it was just sent.
	inv := seedInvitation(t, p, o.ID, owner, "tok-fresh", nil)

	rec := resendInvitation(t, h, o.ID, inv.ID, owner)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "body=%s", rec.Body.String())

	// A resend that goes through starts a new minute.
	old := seedInvitation(t, p, o.ID, owner, "tok-aged", sentLongAgo)
	require.Equal(t, http.StatusOK, resendInvitation(t, h, o.ID, old.ID, owner).Code)
	assert.Equal(t, http.StatusTooManyRequests, resendInvitation(t, h, o.ID, old.ID, owner).Code)
}

func TestResendInvitation_needsAdminAndTheRightOrg(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	ownerOfOther := id.NewUserID()
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)
	now := time.Now()
	other := &organization.Organization{
		ID: id.NewOrgID(), AppID: appID, Name: "Beta", Slug: "beta",
		CreatedBy: ownerOfOther, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, p.CreateOrganization(context.Background(), other))
	inv := seedInvitation(t, p, o.ID, owner, "tok-guarded", sentLongAgo)

	assert.Equal(t, http.StatusUnauthorized, resendInvitation(t, h, o.ID, inv.ID, id.UserID{}).Code)
	assert.Equal(t, http.StatusNotFound, resendInvitation(t, h, o.ID, inv.ID, id.NewUserID()).Code)

	member := id.NewUserID()
	addMember(t, p, o.ID, member, organization.RoleMember)
	assert.Equal(t, http.StatusForbidden, resendInvitation(t, h, o.ID, inv.ID, member).Code)

	// An invitation that belongs to another org is not found through this one.
	assert.Equal(t, http.StatusNotFound, resendInvitation(t, h, other.ID, inv.ID, ownerOfOther).Code)
}

func TestResendInvitation_adminCannotResendAnOwnerInvite(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	admin := id.NewUserID()
	addMember(t, p, o.ID, admin, organization.RoleAdmin)
	inv := seedInvitation(t, p, o.ID, owner, "tok-owner-seat", func(i *organization.Invitation) {
		sentLongAgo(i)
		i.Role = organization.RoleOwner
	})

	assert.Equal(t, http.StatusForbidden, resendInvitation(t, h, o.ID, inv.ID, admin).Code)
	assert.Equal(t, http.StatusForbidden, revokeInvitation(t, h, o.ID, inv.ID, admin).Code)
}

func TestRevokeInvitation_tokenStopsWorking(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-revoke", nil)

	rec := revokeInvitation(t, h, o.ID, inv.ID, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	// The row stays, marked revoked, so the audit has something to point at.
	all, err := p.ListInvitations(context.Background(), o.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, organization.InvitationRevoked, all[0].Status)

	_, err = p.AcceptInvitation(context.Background(), "tok-revoke")
	require.Error(t, err)

	// Revoking twice is a conflict, not a second success.
	assert.Equal(t, http.StatusConflict, revokeInvitation(t, h, o.ID, inv.ID, owner).Code)
}

func TestRevokeInvitation_fromExpiredAllowedFromAcceptedNot(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	expired := seedInvitation(t, p, o.ID, owner, "tok-exp", func(i *organization.Invitation) {
		i.Status = organization.InvitationExpired
	})
	assert.Equal(t, http.StatusOK, revokeInvitation(t, h, o.ID, expired.ID, owner).Code)

	for _, status := range []organization.InvitationStatus{organization.InvitationAccepted, organization.InvitationDeclined} {
		inv := seedInvitation(t, p, o.ID, owner, "tok-"+string(status), func(i *organization.Invitation) {
			i.Status = status
		})
		assert.Equal(t, http.StatusConflict, revokeInvitation(t, h, o.ID, inv.ID, owner).Code, string(status))
	}
}

func TestRevokeInvitation_previewIndistinguishableFromUnknown(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-gone", nil)
	require.Equal(t, http.StatusOK, getPreview(t, h, "tok-gone").Code)

	require.Equal(t, http.StatusOK, revokeInvitation(t, h, o.ID, inv.ID, owner).Code)

	revoked := getPreview(t, h, "tok-gone")
	unknown := getPreview(t, h, "tok-never-existed")
	assert.Equal(t, http.StatusNotFound, revoked.Code)
	assert.Equal(t, unknown.Code, revoked.Code)
	assert.Equal(t, unknown.Body.String(), revoked.Body.String())
}

func TestRevokeInvitation_firesHook(t *testing.T) {
	p := orgplugin.New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	var mu sync.Mutex
	var seen []*hook.Event
	eng.Hooks().On("test-invitation-events", func(_ context.Context, e *hook.Event) error {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, e)
		return nil
	})

	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	toResend := seedInvitation(t, p, o.ID, owner, "tok-hook-1", sentLongAgo)
	toRevoke := seedInvitation(t, p, o.ID, owner, "tok-hook-2", nil)

	require.Equal(t, http.StatusOK, resendInvitation(t, h, o.ID, toResend.ID, owner).Code)
	require.Equal(t, http.StatusOK, revokeInvitation(t, h, o.ID, toRevoke.ID, owner).Code)

	byAction := map[string]*hook.Event{}
	mu.Lock()
	for _, e := range seen {
		byAction[e.Action] = e
	}
	mu.Unlock()

	resent := byAction[hook.ActionInvitationResend]
	require.NotNil(t, resent, "invitation.resent hook did not fire")
	assert.Equal(t, toResend.ID.String(), resent.ResourceID)
	assert.Equal(t, o.ID.String(), resent.Tenant)
	assert.Equal(t, owner.String(), resent.ActorID)

	revoked := byAction[hook.ActionInvitationRevoke]
	require.NotNil(t, revoked, "invitation.revoked hook did not fire")
	assert.Equal(t, toRevoke.ID.String(), revoked.ResourceID)
	assert.Equal(t, o.ID.String(), revoked.Tenant)
	assert.Equal(t, owner.String(), revoked.ActorID)
}
