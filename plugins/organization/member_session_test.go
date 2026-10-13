package organization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
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
