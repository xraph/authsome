package organization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// Accepting an invitation added the account that owned the invited email,
// whoever was signed in, and needed no sign-in at all. Anyone holding the
// token (a forwarded email, a shared link) could put the invitee into the org.
// Only the account that owns the invited address may accept now.

type acceptFixture struct {
	eng     *authsome.Engine
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

	f := &acceptFixture{eng: eng, plugin: p, h: router.Handler()}
	f.invitee = signUp("invitee@example.com")
	// Sign-up leaves the address unverified; only a verified address counts.
	require.NoError(t, eng.Store().MarkUserEmailVerified(context.Background(), f.invitee, "invitee@example.com"))
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

func (f *acceptFixture) post(t *testing.T, path, token string, as id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, orgReq("POST", path, postJSON(t, map[string]string{"token": token}), as))
	return rec
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

func TestAcceptInvitation_OtherAccountGetsTheUnknownTokenAnswer(t *testing.T) {
	f := newAcceptFixture(t)

	rec := f.accept(t, f.other)
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
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
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// A token that was already used is the caller's conflict, not a server fault.
func TestAcceptInvitation_alreadyAcceptedIs409(t *testing.T) {
	f := newAcceptFixture(t)

	rec := f.accept(t, f.invitee)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	rec = f.accept(t, f.invitee)
	assert.Equal(t, http.StatusConflict, rec.Code, "body=%s", rec.Body.String())
}

func TestAcceptInvitation_revokedIs409(t *testing.T) {
	f := newAcceptFixture(t)

	_, err := f.plugin.RevokeInvitation(context.Background(), f.org.ID, f.inv.ID)
	require.NoError(t, err)

	rec := f.accept(t, f.invitee)
	assert.Equal(t, http.StatusConflict, rec.Code, "body=%s", rec.Body.String())
}

// Someone who is not the invitee learns nothing about a token: a live, used,
// revoked, declined or expired one answers byte for byte like one that never
// existed, on accept and on decline, and the invitation is left as it was.
func TestInvitationTokens_nonInviteeCannotTellStatesApart(t *testing.T) {
	for _, path := range []string{"/v1/orgs/invitations/accept", "/v1/orgs/invitations/decline"} {
		t.Run(path, func(t *testing.T) {
			f := newAcceptFixture(t)
			ctx := context.Background()
			tokens := map[string]organization.InvitationStatus{}
			byToken := map[string]id.InvitationID{}
			for _, st := range []organization.InvitationStatus{
				organization.InvitationPending, organization.InvitationAccepted, organization.InvitationRevoked,
				organization.InvitationDeclined, organization.InvitationExpired,
			} {
				inv := &organization.Invitation{
					ID: id.NewInvitationID(), OrgID: f.org.ID, Email: "invitee@example.com",
					Role: organization.RoleMember, Status: st, Token: "tok-" + string(st),
				}
				require.NoError(t, f.plugin.CreateInvitation(ctx, inv))
				tokens[inv.Token] = st
				byToken[inv.Token] = inv.ID
			}
			// A pending one whose time has run out must not be marked expired by a stranger.
			late := &organization.Invitation{
				ID: id.NewInvitationID(), OrgID: f.org.ID, Email: "invitee@example.com",
				Role: organization.RoleMember, Status: organization.InvitationPending, Token: "tok-late",
				ExpiresAt: time.Now().Add(-time.Hour),
			}
			require.NoError(t, f.plugin.CreateInvitation(ctx, late))
			tokens[late.Token] = organization.InvitationPending
			byToken[late.Token] = late.ID

			unknown := f.post(t, path, "tok-never-existed", f.other)
			require.Equal(t, http.StatusNotFound, unknown.Code, "body=%s", unknown.Body.String())
			for token, st := range tokens {
				rec := f.post(t, path, token, f.other)
				assert.Equal(t, unknown.Code, rec.Code, "token %s", token)
				assert.Equal(t, unknown.Body.String(), rec.Body.String(), "token %s", token)
				assert.Equal(t, unknown.Header().Get("Content-Type"), rec.Header().Get("Content-Type"), "token %s", token)

				got, err := f.plugin.GetInvitation(ctx, f.org.ID, byToken[token])
				require.NoError(t, err)
				assert.Equal(t, st, got.Status, "a non-invitee changed token %s", token)
			}
		})
	}
}

// Declining needs the signed-in invitee, the same as accepting.
func TestDeclineInvitation_requiresTheInvitee(t *testing.T) {
	f := newAcceptFixture(t)
	path := "/v1/orgs/invitations/decline"

	assert.Equal(t, http.StatusUnauthorized, f.post(t, path, f.inv.Token, id.UserID{}).Code)
	assert.Equal(t, http.StatusNotFound, f.post(t, path, f.inv.Token, f.other).Code)

	rec := f.post(t, path, f.inv.Token, f.invitee)
	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	// Now it is used: the invitee, and only the invitee, sees 409.
	assert.Equal(t, http.StatusConflict, f.post(t, path, f.inv.Token, f.invitee).Code)
}

func TestAcceptInvitation_inviteeSeesTheStateOfTheirOwnToken(t *testing.T) {
	f := newAcceptFixture(t)
	ctx := context.Background()
	late := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: f.org.ID, Email: "invitee@example.com",
		Role: organization.RoleMember, Status: organization.InvitationPending, Token: "tok-late",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, f.plugin.CreateInvitation(ctx, late))
	assert.Equal(t, http.StatusGone, f.post(t, "/v1/orgs/invitations/accept", late.Token, f.invitee).Code)
}

// Only a verified address makes someone the invitee. An unverified secondary
// or an unverified primary matching the invitation's email gets the
// unknown-token 404, and a verified secondary passes. Each answer is the same
// on accept and decline.
func TestInvitationTokens_onlyAVerifiedAddressCounts(t *testing.T) {
	f := newAcceptFixture(t)
	ctx := context.Background()
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)

	invite := func(email, token string) *organization.Invitation {
		inv := &organization.Invitation{
			ID: id.NewInvitationID(), OrgID: f.org.ID, Email: email,
			Role: organization.RoleMember, Status: organization.InvitationPending, Token: token,
		}
		require.NoError(t, f.plugin.CreateInvitation(ctx, inv))
		return inv
	}
	signUp := func(email string) id.UserID {
		u, _, err := f.eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: email, Password: "SecureP@ss123", FirstName: "T"})
		require.NoError(t, err)
		return u.ID
	}
	both := []string{"/v1/orgs/invitations/accept", "/v1/orgs/invitations/decline"}
	unknown := f.post(t, both[0], "tok-never-existed", f.other)
	require.Equal(t, http.StatusNotFound, unknown.Code)

	// Unverified primary: the account exists but never verified the address.
	unverified := signUp("unverified@example.com")
	invite("unverified@example.com", "tok-unverified-primary")
	for _, path := range both {
		rec := f.post(t, path, "tok-unverified-primary", unverified)
		assert.Equal(t, unknown.Code, rec.Code, path)
		assert.Equal(t, unknown.Body.String(), rec.Body.String(), path)
	}

	// Unverified secondary.
	claimant := signUp("claimant@example.com")
	require.NoError(t, f.eng.Store().AddUserEmail(ctx, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: claimant, AppID: appID,
		Email: "victim@example.com", Verified: false, Source: "test",
	}))
	invite("victim@example.com", "tok-unverified-secondary")
	for _, path := range both {
		rec := f.post(t, path, "tok-unverified-secondary", claimant)
		assert.Equal(t, unknown.Code, rec.Code, path)
		assert.Equal(t, unknown.Body.String(), rec.Body.String(), path)
	}

	// Once verified, the secondary is enough.
	require.NoError(t, f.eng.Store().MarkUserEmailVerified(ctx, claimant, "victim@example.com"))
	rec := f.post(t, both[0], "tok-unverified-secondary", claimant)
	assert.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

// An invitation whose organization no longer exists reads as an unknown
// token, not a server error.
func TestInvitationTokens_goneOrgIsTheUnknownTokenAnswer(t *testing.T) {
	f := newAcceptFixture(t)
	ctx := context.Background()
	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: id.NewOrgID(), Email: "invitee@example.com",
		Role: organization.RoleMember, Status: organization.InvitationPending, Token: "tok-gone-org",
	}
	if err := f.plugin.CreateInvitation(ctx, inv); err != nil {
		t.Skipf("the store refuses an invitation with no organization: %v", err)
	}
	unknown := f.post(t, "/v1/orgs/invitations/accept", "tok-never-existed", f.invitee)
	for _, path := range []string{"/v1/orgs/invitations/accept", "/v1/orgs/invitations/decline"} {
		rec := f.post(t, path, inv.Token, f.invitee)
		assert.Equal(t, unknown.Code, rec.Code, path)
		assert.Equal(t, unknown.Body.String(), rec.Body.String(), path)
	}
}
