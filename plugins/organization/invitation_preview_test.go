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

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/user"
)

// GET /v1/orgs/invitations/:inviteToken lets whoever holds an invitation see
// what they are being asked to join before they sign in. It is
// unauthenticated on purpose, so what it does NOT say matters as much as what
// it does.

// seedInvitation stores an invitation with a known plaintext token.
func seedInvitation(
	t *testing.T,
	p *orgplugin.Plugin,
	orgID id.OrgID,
	inviter id.UserID,
	token string,
	mutate func(*organization.Invitation),
) *organization.Invitation {
	t.Helper()
	inv := &organization.Invitation{
		ID:        id.NewInvitationID(),
		OrgID:     orgID,
		Email:     "sofia@acme.com",
		Role:      organization.RoleMember,
		InviterID: inviter,
		Status:    organization.InvitationPending,
		Token:     token,
		ExpiresAt: time.Now().Add(6 * 24 * time.Hour),
	}
	if mutate != nil {
		mutate(inv)
	}
	require.NoError(t, p.CreateInvitation(context.Background(), inv))
	return inv
}

func getPreview(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	// The zero UserID means no session, which is the whole point of the route.
	h.ServeHTTP(rec, orgReq("GET", "/v1/orgs/invitations/"+token, nil, id.UserID{}))
	return rec
}

func TestGetInvitation_ShowsWhatYouAreJoining(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	inv := seedInvitation(t, p, o.ID, owner, "tok-visible", nil)

	rec := getPreview(t, h, "tok-visible")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var got orgplugin.InvitationPreview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	assert.Equal(t, o.ID.String(), got.OrgID)
	assert.Equal(t, "Acme", got.OrgName)
	assert.Equal(t, "acme", got.OrgSlug)
	assert.Equal(t, string(organization.RoleMember), got.Role)
	assert.WithinDuration(t, inv.ExpiresAt, got.ExpiresAt, time.Second)
}

func TestGetInvitation_MasksTheInvitedAddress(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-mask", nil)

	rec := getPreview(t, h, "tok-mask")
	require.Equal(t, http.StatusOK, rec.Code)

	// Enough to recognise your own address, not enough to learn a new one,
	// and the raw address never appears anywhere in the body.
	assert.Equal(t, "so***@acme.com", gjsonField(t, rec.Body.Bytes(), "email_masked"))
	assert.NotContains(t, rec.Body.String(), "sofia@acme.com")
}

func TestGetInvitation_NeverLeaksTheToken(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-secret", nil)

	rec := getPreview(t, h, "tok-secret")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "tok-secret")
}

// Expired, consumed and never-existed all answer the same way, so a list of
// guessed tokens cannot be used to find out which ones are real.
func TestGetInvitation_IndistinguishableFailures(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())

	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-expired", func(i *organization.Invitation) {
		i.ExpiresAt = time.Now().Add(-time.Hour)
	})
	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-accepted", func(i *organization.Invitation) {
		i.Status = organization.InvitationAccepted
	})
	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-declined", func(i *organization.Invitation) {
		i.Status = organization.InvitationDeclined
	})

	for _, token := range []string{"tok-expired", "tok-accepted", "tok-declined", "tok-never-existed"} {
		t.Run(token, func(t *testing.T) {
			rec := getPreview(t, h, token)
			assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
		})
	}
}

func TestMaskEmail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"sofia@acme.com", "so***@acme.com"},
		{"rex.raphael@acme.com", "re***@acme.com"},
		// A one-character local part keeps what little there is rather than
		// panicking on the slice.
		{"a@acme.com", "a***@acme.com"},
		// No @ at all is not an address, so there is nothing safe to show.
		{"not-an-address", "***"},
		{"", "***"},
		// A leading @ leaves no local part to keep.
		{"@acme.com", "***"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, orgplugin.MaskEmail(c.in), "input %q", c.in)
	}
}

func TestInviterName_FallsThroughToSomething(t *testing.T) {
	assert.Equal(t, "Rex Raphael", orgplugin.InviterName(&user.User{FirstName: "Rex", LastName: "Raphael"}))
	assert.Equal(t, "Rex", orgplugin.InviterName(&user.User{FirstName: "Rex"}))
	assert.Equal(t, "rexr", orgplugin.InviterName(&user.User{DisplayUsername: "rexr"}))
	assert.Equal(t, "rex", orgplugin.InviterName(&user.User{Username: "rex"}))
	assert.Equal(t, "rex@acme.com", orgplugin.InviterName(&user.User{Email: "rex@acme.com"}))
	// An inviter whose account was deleted since they sent it.
	assert.Equal(t, "Someone", orgplugin.InviterName(nil))
	assert.Equal(t, "Someone", orgplugin.InviterName(&user.User{}))
}

// An invitation whose inviter is no longer in the store still shows, because
// the seat is valid whether or not the account that offered it still exists.
func TestGetInvitation_SurvivesAMissingInviter(t *testing.T) {
	p, h := newOrgHTTP(t)
	o := seedOrg(t, p, id.NewUserID())
	seedInvitation(t, p, o.ID, id.NewUserID(), "tok-ghost", nil)

	rec := getPreview(t, h, "tok-ghost")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, "Someone", gjsonField(t, rec.Body.Bytes(), "inviter_name"))
}

// gjsonField reads one top-level string field out of a JSON body.
func gjsonField(t *testing.T, body []byte, field string) string {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	s, _ := m[field].(string)
	return s
}
