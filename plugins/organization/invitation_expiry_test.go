package organization_test

import (
	"context"
	"encoding/json"
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
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// The invitation endpoint set no expiry, so an invitation link worked
// forever. Invitations now lapse after Config.InvitationTTL, seven days
// unless you say otherwise.

func newOrgHTTPWith(t *testing.T, cfg orgplugin.Config) (*orgplugin.Plugin, http.Handler) {
	t.Helper()
	p := orgplugin.New(cfg)
	_ = secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return p, router.Handler()
}

func inviteOverHTTP(t *testing.T, h http.Handler, orgID id.OrgID, caller id.UserID) time.Time {
	t.Helper()
	rec := httptest.NewRecorder()
	body := postJSON(t, map[string]string{"email": "guest@example.com"})
	h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+orgID.String()+"/invitations", body, caller))
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var inv organization.Invitation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &inv))
	return inv.ExpiresAt
}

func TestCreateInvitation_DefaultsToAWeek(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	got := inviteOverHTTP(t, h, o.ID, owner)
	assert.WithinDuration(t, time.Now().Add(orgplugin.DefaultInvitationTTL), got, time.Minute)
	assert.Equal(t, 7*24*time.Hour, orgplugin.DefaultInvitationTTL)
}

func TestCreateInvitation_HonoursConfiguredTTL(t *testing.T) {
	p, h := newOrgHTTPWith(t, orgplugin.Config{InvitationTTL: 2 * time.Hour})
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	got := inviteOverHTTP(t, h, o.ID, owner)
	assert.WithinDuration(t, time.Now().Add(2*time.Hour), got, time.Minute)
}

func TestCreateInvitation_NegativeTTLNeverExpires(t *testing.T) {
	p, h := newOrgHTTPWith(t, orgplugin.Config{InvitationTTL: -1})
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)

	assert.True(t, inviteOverHTTP(t, h, o.ID, owner).IsZero())
}

func TestAcceptInvitation_ExpiredIsGone(t *testing.T) {
	f := newAcceptFixture(t)
	f.inv.Status = organization.InvitationPending
	late := &organization.Invitation{
		ID:        id.NewInvitationID(),
		OrgID:     f.org.ID,
		Email:     "invitee@example.com",
		Role:      organization.RoleMember,
		Status:    organization.InvitationPending,
		Token:     "tok-expired",
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	require.NoError(t, f.plugin.CreateInvitation(context.Background(), late))

	// Only the invitee is told it has expired.
	rec := f.post(t, "/v1/orgs/invitations/accept", late.Token, f.invitee)
	assert.Equal(t, http.StatusGone, rec.Code, "body=%s", rec.Body.String())

	_, err := f.plugin.AcceptInvitation(context.Background(), late.Token)
	assert.Error(t, err, "an expired invitation stays refused")
}
