package organization_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
)

// Removing a member left their sessions naming the org as active, so a
// session from before the removal kept acting in an org the user had left.
// Removal now clears that org from the member's sessions, and only that org.

func TestRemoveMember_ClearsActiveOrgFromTheirSessions(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()
	st := s.eng.Store()

	leaver := id.NewUserID()
	m := &organization.Member{ID: id.NewMemberID(), OrgID: s.org.ID, UserID: leaver, Role: organization.RoleMember}
	require.NoError(t, s.plugin.AddMember(ctx, m))

	newSession := func(userID id.UserID, orgID id.OrgID, token string) id.SessionID {
		sess := &session.Session{ID: id.NewSessionID(), AppID: s.org.AppID, UserID: userID, OrgID: orgID, Token: token}
		require.NoError(t, st.CreateSession(ctx, sess))
		return sess.ID
	}
	otherOrg := id.NewOrgID()
	inOrg := newSession(leaver, s.org.ID, "tok-in-org")
	elsewhere := newSession(leaver, otherOrg, "tok-elsewhere")
	ownerSess := newSession(s.owner, s.org.ID, "tok-owner")

	require.NoError(t, s.plugin.RemoveMember(ctx, m.ID))

	got, err := st.GetSession(ctx, inOrg)
	require.NoError(t, err, "the session stays signed in")
	assert.True(t, got.OrgID.IsNil(), "the org the member left must no longer be active, got %s", got.OrgID)

	got, err = st.GetSession(ctx, elsewhere)
	require.NoError(t, err)
	assert.Equal(t, otherOrg.String(), got.OrgID.String(), "a session active in another org is untouched")

	got, err = st.GetSession(ctx, ownerSess)
	require.NoError(t, err)
	assert.Equal(t, s.org.ID.String(), got.OrgID.String(), "other members' sessions are untouched")
}

func TestRemoveMemberHTTP_ClearsActiveOrg(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()
	h := orgHandler(t, s)

	leaver := id.NewUserID()
	m := &organization.Member{ID: id.NewMemberID(), OrgID: s.org.ID, UserID: leaver, Role: organization.RoleMember}
	require.NoError(t, s.plugin.AddMember(ctx, m))
	sess := &session.Session{ID: id.NewSessionID(), AppID: s.org.AppID, UserID: leaver, OrgID: s.org.ID, Token: "tok-http"}
	require.NoError(t, s.eng.Store().CreateSession(ctx, sess))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+s.org.ID.String()+"/members/"+m.ID.String(), nil, s.owner))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	got, err := s.eng.Store().GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.True(t, got.OrgID.IsNil())
}

// orgHandler mounts the routes of an already-built setup's plugin.
func orgHandler(t *testing.T, s *txTestSetup) http.Handler {
	t.Helper()
	router := forge.NewRouter()
	require.NoError(t, s.plugin.RegisterRoutes(router))
	return router.Handler()
}

// Deleting an org left every member's sessions naming it as active, the same
// gap RemoveMember had. Deletion now clears it from all of its members'
// sessions, and leaves sessions active in other orgs alone.

func TestDeleteOrganization_ClearsActiveOrgFromMembersSessions(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()
	st := s.eng.Store()

	member := id.NewUserID()
	m := &organization.Member{ID: id.NewMemberID(), OrgID: s.org.ID, UserID: member, Role: organization.RoleMember}
	require.NoError(t, s.plugin.AddMember(ctx, m))

	newSession := func(userID id.UserID, orgID id.OrgID, token string) id.SessionID {
		sess := &session.Session{ID: id.NewSessionID(), AppID: s.org.AppID, UserID: userID, OrgID: orgID, Token: token}
		require.NoError(t, st.CreateSession(ctx, sess))
		return sess.ID
	}
	otherOrg := id.NewOrgID()
	ownerIn := newSession(s.owner, s.org.ID, "tok-owner-in")
	memberIn := newSession(member, s.org.ID, "tok-member-in")
	memberElsewhere := newSession(member, otherOrg, "tok-member-elsewhere")
	outsider := id.NewUserID()
	outsiderElsewhere := newSession(outsider, otherOrg, "tok-outsider")

	require.NoError(t, s.plugin.DeleteOrganization(ctx, s.org.ID))

	for name, sid := range map[string]id.SessionID{"owner": ownerIn, "member": memberIn} {
		got, err := st.GetSession(ctx, sid)
		require.NoError(t, err, "%s: the session stays signed in", name)
		assert.True(t, got.OrgID.IsNil(), "%s: the deleted org must no longer be active, got %s", name, got.OrgID)
	}
	for name, sid := range map[string]id.SessionID{"member": memberElsewhere, "outsider": outsiderElsewhere} {
		got, err := st.GetSession(ctx, sid)
		require.NoError(t, err)
		assert.Equal(t, otherOrg.String(), got.OrgID.String(), "%s: a session active in another org is untouched", name)
	}
}

func TestDeleteOrganization_SessionClearFailureIsNotAnError(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()
	st := s.eng.Store()

	sess := &session.Session{ID: id.NewSessionID(), AppID: s.org.AppID, UserID: s.owner, OrgID: s.org.ID, Token: "tok-fault"}
	require.NoError(t, st.CreateSession(ctx, sess))
	secutil.InjectStoreFault(t, s.eng, "UpdateSession", errors.New("synthetic update-session failure"))

	require.NoError(t, s.plugin.DeleteOrganization(ctx, s.org.ID),
		"the org is already gone, so a failed session clear must not fail the delete")

	_, err := st.GetOrganization(ctx, s.org.ID)
	assert.Error(t, err, "the org is deleted")

	got, err := st.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, s.org.ID.String(), got.OrgID.String(), "the injected fault did hit the session clear")
}

func TestDeleteOrganizationHTTP_ClearsActiveOrg(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()
	h := orgHandler(t, s)

	sess := &session.Session{ID: id.NewSessionID(), AppID: s.org.AppID, UserID: s.owner, OrgID: s.org.ID, Token: "tok-http-delete"}
	require.NoError(t, s.eng.Store().CreateSession(ctx, sess))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, orgReq("DELETE", "/v1/orgs/"+s.org.ID.String(), nil, s.owner))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	got, err := s.eng.Store().GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.True(t, got.OrgID.IsNil())
}
