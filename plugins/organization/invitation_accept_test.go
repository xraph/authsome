package organization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// Accepting an invitation added the account that owned the invited email,
// whoever was signed in, and needed no sign-in at all. Anyone holding the
// token (a forwarded email, a shared link) could put the invitee into the org.
// Only the account that owns the invited address may accept now.

type acceptFixture struct {
	plugin  *orgplugin.Plugin
	h       http.Handler
	org     *organization.Organization
	invitee id.UserID
	other   id.UserID
	inv     *organization.Invitation
}

func newAcceptFixture(t *testing.T) *acceptFixture {
	t.Helper()
	p := orgplugin.New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))

	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)
	signUp := func(email string) id.UserID {
		u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
			AppID: appID, Email: email, Password: "SecureP@ss123", FirstName: "Test",
		})
		require.NoError(t, err)
		return u.ID
	}

	f := &acceptFixture{plugin: p, h: router.Handler()}
	f.invitee = signUp("invitee@example.com")
	f.other = signUp("someone-else@example.com")
	f.org = seedOrg(t, p, id.NewUserID())
	f.inv = &organization.Invitation{
		ID:     id.NewInvitationID(),
		OrgID:  f.org.ID,
		Email:  "invitee@example.com",
		Role:   organization.RoleMember,
		Status: organization.InvitationPending,
		Token:  "tok-accept",
	}
	require.NoError(t, p.CreateInvitation(context.Background(), f.inv))
	return f
}

func (f *acceptFixture) accept(t *testing.T, as id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/invitations/accept", postJSON(t, map[string]string{"token": f.inv.Token}), as))
	return rec
}

func TestAcceptInvitation_RequiresSignIn(t *testing.T) {
	f := newAcceptFixture(t)
	rec := f.accept(t, id.UserID{})
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}

func TestAcceptInvitation_OtherAccountIsForbidden(t *testing.T) {
	f := newAcceptFixture(t)

	rec := f.accept(t, f.other)
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "invitee@example.com", "the refusal must not reveal who was invited")

	members, err := f.plugin.ListMembers(context.Background(), f.org.ID)
	require.NoError(t, err)
	for _, m := range members {
		assert.NotEqual(t, f.invitee.String(), m.UserID.String(), "the invitee must not have been added")
	}

	// The invitation is still there for the person it was sent to.
	rec = f.accept(t, f.invitee)
	assert.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

func TestAcceptInvitation_InviteeSucceeds(t *testing.T) {
	f := newAcceptFixture(t)

	rec := f.accept(t, f.invitee)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, f.invitee.String(), decodeOne(t, rec)["user_id"])
	assert.Equal(t, organization.RoleMember, memberRole(t, f.plugin, f.org.ID, f.invitee))
}

func TestAcceptInvitationAs_StrangerWithNoAccountForTheEmail(t *testing.T) {
	f := newAcceptFixture(t)
	_, err := f.plugin.AcceptInvitationAs(context.Background(), f.inv.Token, id.NewUserID())
	assert.ErrorIs(t, err, orgplugin.ErrInvitationEmailMismatch)
}
