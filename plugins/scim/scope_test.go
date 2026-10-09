package scim

import (
	"bytes"
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
	"github.com/xraph/authsome/user"
)

// A SCIM bearer token is scoped to one app, and optionally to one
// organization inside it. Every row the handlers load by id must sit inside
// that scope, and a row outside it must be indistinguishable from a missing
// one, so an IdP token for one tenant cannot read, rename, ban or delete
// anything in another.

type scopeFixture struct {
	p     *Plugin
	eng   *authsome.Engine
	h     http.Handler
	appID id.AppID
}

func newScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()
	p := New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	appID, err := id.ParseAppID(scimLifecycleTestAppID)
	require.NoError(t, err)
	return &scopeFixture{p: p, eng: eng, h: router.Handler(), appID: appID}
}

// token mints a bearer token for a config on the fixture app, org-scoped
// when orgID is set.
func (f *scopeFixture) token(t *testing.T, orgID id.OrgID) string {
	t.Helper()
	cfg := &SCIMConfig{
		Name: "idp", AppID: f.appID, OrgID: orgID,
		Enabled: true, AutoCreate: true, AutoSuspend: true, GroupSync: !orgID.IsNil(),
	}
	require.NoError(t, f.p.service.CreateConfig(context.Background(), cfg))
	tok, _, err := f.p.service.GenerateToken(context.Background(), cfg.ID, "t", nil)
	require.NoError(t, err)
	return tok
}

func (f *scopeFixture) user(t *testing.T, appID id.AppID, email string) *user.User {
	t.Helper()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, Email: email, FirstName: "Given",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, f.eng.Store().CreateUserWithPrimaryEmail(context.Background(), u, user.NewPrimaryEmail(u, "test")))
	return u
}

func (f *scopeFixture) org(t *testing.T, slug string) id.OrgID {
	t.Helper()
	o := &organization.Organization{
		ID: id.NewOrgID(), AppID: f.appID, Name: slug, Slug: slug,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, f.eng.Store().CreateOrganization(context.Background(), o))
	return o.ID
}

func (f *scopeFixture) member(t *testing.T, orgID id.OrgID, uid id.UserID) {
	t.Helper()
	require.NoError(t, f.eng.Store().CreateMember(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: orgID, UserID: uid, Role: organization.RoleMember,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
}

func (f *scopeFixture) team(t *testing.T, orgID id.OrgID, name string) *organization.Team {
	t.Helper()
	tm := &organization.Team{
		ID: id.NewTeamID(), OrgID: orgID, Name: name, Slug: name,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, f.eng.Store().CreateTeam(context.Background(), tm))
	return tm
}

func (f *scopeFixture) do(t *testing.T, tok, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw := []byte(nil)
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, f.p.config.BasePath+path, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func (f *scopeFixture) banned(t *testing.T, uid id.UserID) bool {
	t.Helper()
	u, err := f.eng.Store().GetUser(context.Background(), uid)
	require.NoError(t, err)
	return u.Banned
}

func deactivate() PatchOp {
	return PatchOp{Schemas: []string{SchemaPatchOp}, Operations: []Operation{{Op: "replace", Path: "active", Value: false}}}
}

func TestSCIM_UserInAnotherApp_IsInvisible(t *testing.T) {
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	mine := f.user(t, f.appID, "mine@example.com")
	theirs := f.user(t, id.NewAppID(), "theirs@example.com")

	assert.Equal(t, http.StatusOK, f.do(t, tok, http.MethodGet, "/Users/"+mine.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodGet, "/Users/"+theirs.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodPatch, "/Users/"+theirs.ID.String(), deactivate()).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodDelete, "/Users/"+theirs.ID.String(), nil).Code)
	assert.False(t, f.banned(t, theirs.ID), "another app's user must be untouched")
}

func TestSCIM_OrgScoped_NonMemberIsInvisible(t *testing.T) {
	f := newScopeFixture(t)
	orgID := f.org(t, "acme")
	tok := f.token(t, orgID)
	member := f.user(t, f.appID, "member@example.com")
	f.member(t, orgID, member.ID)
	outsider := f.user(t, f.appID, "outsider@example.com")

	assert.Equal(t, http.StatusOK, f.do(t, tok, http.MethodGet, "/Users/"+member.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodGet, "/Users/"+outsider.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodDelete, "/Users/"+outsider.ID.String(), nil).Code)
	assert.False(t, f.banned(t, outsider.ID))

	// A POST whose email resolves to an app user outside the org must not
	// silently adopt that account.
	rec := f.do(t, tok, http.MethodPost, "/Users", UserResource{
		Schemas: []string{SchemaUser}, UserName: outsider.Email,
		Emails: []Email{{Value: outsider.Email, Primary: true}}, Active: true,
	})
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

func TestSCIM_ReplaceUser_BodyEmailOfAnotherUser_IsRefused(t *testing.T) {
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	target := f.user(t, f.appID, "target@example.com")
	victim := f.user(t, f.appID, "victim@example.com")

	rec := f.do(t, tok, http.MethodPut, "/Users/"+target.ID.String(), UserResource{
		Schemas: []string{SchemaUser}, UserName: victim.Email,
		Name:   Name{GivenName: "Renamed"},
		Emails: []Email{{Value: victim.Email, Primary: true}}, Active: false,
	})
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
	assert.False(t, f.banned(t, target.ID), "the path target must be untouched")
	assert.False(t, f.banned(t, victim.ID), "the email owner must be untouched")

	// PUT with the target's own address applies.
	rec = f.do(t, tok, http.MethodPut, "/Users/"+target.ID.String(), UserResource{
		Schemas: []string{SchemaUser}, UserName: target.Email,
		Name:   Name{GivenName: "Renamed"},
		Emails: []Email{{Value: target.Email, Primary: true}}, Active: false,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.True(t, f.banned(t, target.ID))
}

func TestSCIM_GroupInAnotherOrg_IsInvisible(t *testing.T) {
	f := newScopeFixture(t)
	mineOrg := f.org(t, "mine")
	otherOrg := f.org(t, "other")
	tok := f.token(t, mineOrg)
	mine := f.team(t, mineOrg, "eng")
	theirs := f.team(t, otherOrg, "ops")

	rename := PatchOp{Schemas: []string{SchemaPatchOp}, Operations: []Operation{{Op: "replace", Path: "displayName", Value: "pwned"}}}
	assert.Equal(t, http.StatusOK, f.do(t, tok, http.MethodGet, "/Groups/"+mine.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodGet, "/Groups/"+theirs.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodPatch, "/Groups/"+theirs.ID.String(), rename).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodPut, "/Groups/"+theirs.ID.String(),
		GroupResource{Schemas: []string{SchemaGroup}, DisplayName: "pwned"}).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodDelete, "/Groups/"+theirs.ID.String(), nil).Code)

	got, err := f.eng.Store().GetTeam(context.Background(), theirs.ID)
	require.NoError(t, err, "the other org's team must survive")
	assert.Equal(t, "ops", got.Name)
}

func TestSCIM_AppScopedToken_SeesNoGroups(t *testing.T) {
	// Groups only exist inside an organization; an app-level token has no
	// org and therefore no group it may touch.
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	tm := f.team(t, f.org(t, "acme"), "eng")
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodGet, "/Groups/"+tm.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, tok, http.MethodDelete, "/Groups/"+tm.ID.String(), nil).Code)
}
