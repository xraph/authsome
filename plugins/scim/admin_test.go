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
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
)

// The SCIM admin routes are the dashboard surface: connect an identity
// provider, mint the token it authenticates with, rotate it, revoke it.
//
// They are deliberately NOT the SCIM 2.0 group. That group authenticates
// with a provisioning bearer token, and an IdP holding one has no business
// minting more, so these authenticate as an organization owner or admin
// instead. Most of what follows pins that boundary.

type adminFixture struct {
	p     *Plugin
	eng   *authsome.Engine
	h     http.Handler
	appID id.AppID
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	p := New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	appID, err := id.ParseAppID(scimLifecycleTestAppID)
	require.NoError(t, err)
	return &adminFixture{p: p, eng: eng, h: router.Handler(), appID: appID}
}

// org creates an organization with one member at the given role.
func (f *adminFixture) org(t *testing.T, role organization.MemberRole) (id.OrgID, id.UserID) {
	t.Helper()
	now := time.Now()
	o := &organization.Organization{
		ID: id.NewOrgID(), AppID: f.appID, Name: "Acme", Slug: "acme-" + id.NewOrgID().String(),
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, f.eng.Store().CreateOrganization(context.Background(), o))

	userID := id.NewUserID()
	require.NoError(t, f.eng.Store().CreateMember(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: o.ID, UserID: userID, Role: role,
		CreatedAt: now, UpdatedAt: now,
	}))
	return o.ID, userID
}

// config stores a SCIM configuration for an organization.
func (f *adminFixture) config(t *testing.T, orgID id.OrgID) *SCIMConfig {
	t.Helper()
	cfg := &SCIMConfig{
		ID: id.NewSCIMConfigID(), Name: "Okta", AppID: f.appID, OrgID: orgID,
		Enabled: true, AutoCreate: true, AutoSuspend: true,
	}
	require.NoError(t, f.p.service.CreateConfig(context.Background(), cfg))
	return cfg
}

func (f *adminFixture) do(t *testing.T, method, path string, body any, as id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	}
	if !as.IsNil() {
		req = req.WithContext(middleware.WithUserID(req.Context(), as))
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

// ── Authorization ────────────────────────────────────────

func TestSCIMAdmin_OwnerAndAdminMayList(t *testing.T) {
	for _, role := range []organization.MemberRole{organization.RoleOwner, organization.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			f := newAdminFixture(t)
			orgID, userID := f.org(t, role)
			f.config(t, orgID)

			rec := f.do(t, "GET", "/v1/orgs/"+orgID.String()+"/scim/configs", nil, userID)
			require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

			var got ConfigListResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Len(t, got.Configs, 1)
		})
	}
}

// Connecting an identity provider decides who gets an account, so it sits
// with the roles that already decide membership.
func TestSCIMAdmin_PlainMemberIsRefused(t *testing.T) {
	f := newAdminFixture(t)
	orgID, userID := f.org(t, organization.RoleMember)

	rec := f.do(t, "GET", "/v1/orgs/"+orgID.String()+"/scim/configs", nil, userID)
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

func TestSCIMAdmin_UnauthenticatedIsRefused(t *testing.T) {
	f := newAdminFixture(t)
	orgID, _ := f.org(t, organization.RoleOwner)

	rec := f.do(t, "GET", "/v1/orgs/"+orgID.String()+"/scim/configs", nil, id.UserID{})
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}

// A non-member gets 404, not 403, so probing organization IDs tells an
// outsider nothing about which ones exist.
func TestSCIMAdmin_NonMemberCannotTellTheOrgExists(t *testing.T) {
	f := newAdminFixture(t)
	orgID, _ := f.org(t, organization.RoleOwner)
	stranger := id.NewUserID()

	existing := f.do(t, "GET", "/v1/orgs/"+orgID.String()+"/scim/configs", nil, stranger)
	invented := f.do(t, "GET", "/v1/orgs/"+id.NewOrgID().String()+"/scim/configs", nil, stranger)

	assert.Equal(t, http.StatusNotFound, existing.Code)
	assert.Equal(t, invented.Code, existing.Code, "an org that exists and one that does not must answer the same")
}

// An admin of one organization is nobody in another.
func TestSCIMAdmin_CannotReachAnotherOrgsConfig(t *testing.T) {
	f := newAdminFixture(t)
	mineID, mineOwner := f.org(t, organization.RoleOwner)
	theirsID, _ := f.org(t, organization.RoleOwner)
	_ = mineID
	theirCfg := f.config(t, theirsID)

	rec := f.do(t, "GET", "/v1/scim/configs/"+theirCfg.ID.String(), nil, mineOwner)
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

// An app-level config has no organization to check a role against, so it is
// not reachable here at all. Managing one stays an operator job.
func TestSCIMAdmin_AppLevelConfigIsNotReachable(t *testing.T) {
	f := newAdminFixture(t)
	_, owner := f.org(t, organization.RoleOwner)

	appCfg := &SCIMConfig{
		ID: id.NewSCIMConfigID(), Name: "app-wide", AppID: f.appID, Enabled: true,
	}
	require.NoError(t, f.p.service.CreateConfig(context.Background(), appCfg))

	rec := f.do(t, "GET", "/v1/scim/configs/"+appCfg.ID.String(), nil, owner)
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

// ── Configurations ───────────────────────────────────────

func TestSCIMAdmin_CreateDefaultsToConnected(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)

	rec := f.do(t, "POST", "/v1/orgs/"+orgID.String()+"/scim/configs",
		map[string]any{"name": "Okta"}, owner)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var got SCIMConfig
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	// A connection that quietly does nothing is worse than no connection.
	assert.True(t, got.Enabled)
	assert.True(t, got.AutoCreate)
	assert.True(t, got.AutoSuspend)
	assert.False(t, got.GroupSync)
	assert.Equal(t, orgID.String(), got.OrgID.String())
}

func TestSCIMAdmin_CreateHonoursExplicitFalse(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)

	rec := f.do(t, "POST", "/v1/orgs/"+orgID.String()+"/scim/configs",
		map[string]any{"name": "Okta", "auto_create": false, "group_sync": true}, owner)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var got SCIMConfig
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.False(t, got.AutoCreate, "an explicit false is not the same as leaving it out")
	assert.True(t, got.GroupSync)
}

func TestSCIMAdmin_CreateRejectsAnEmptyName(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)

	rec := f.do(t, "POST", "/v1/orgs/"+orgID.String()+"/scim/configs",
		map[string]any{"name": "   "}, owner)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

// PATCH changes what you send and leaves the rest alone.
func TestSCIMAdmin_UpdateIsPartial(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "PATCH", "/v1/scim/configs/"+cfg.ID.String(),
		map[string]any{"enabled": false}, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var got SCIMConfig
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.False(t, got.Enabled)
	assert.Equal(t, "Okta", got.Name, "a field left out of a PATCH keeps its value")
	assert.True(t, got.AutoCreate)
}

// Every optional field is its own branch, and a PATCH that sets all of them
// is the only way to know none of them was wired to the wrong struct member.
func TestSCIMAdmin_UpdateCanChangeEverything(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "PATCH", "/v1/scim/configs/"+cfg.ID.String(), map[string]any{
		"name":         "Entra",
		"enabled":      false,
		"auto_create":  false,
		"auto_suspend": false,
		"group_sync":   true,
		"default_role": "admin",
		"metadata":     map[string]string{"env": "prod"},
	}, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var got SCIMConfig
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "Entra", got.Name)
	assert.False(t, got.Enabled)
	assert.False(t, got.AutoCreate)
	assert.False(t, got.AutoSuspend)
	assert.True(t, got.GroupSync)
	assert.Equal(t, "admin", got.DefaultRole)
	assert.Equal(t, map[string]string{"env": "prod"}, got.Metadata)
}

func TestSCIMAdmin_UpdateRejectsAnEmptyName(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "PATCH", "/v1/scim/configs/"+cfg.ID.String(),
		map[string]any{"name": "  "}, owner)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

// A config with no tokens answers with an empty list, not null, so a client
// can iterate it without a nil check.
func TestSCIMAdmin_ListTokensIsEmptyNotNull(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "GET", "/v1/scim/configs/"+cfg.ID.String()+"/tokens", nil, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "[]")

	var got TokenListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotNil(t, got.Tokens)
	assert.Empty(t, got.Tokens)
}

// Same for configurations: an org that has connected nothing yet.
func TestSCIMAdmin_ListConfigsIsEmptyNotNull(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)

	rec := f.do(t, "GET", "/v1/orgs/"+orgID.String()+"/scim/configs", nil, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var got ConfigListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotNil(t, got.Configs)
	assert.Empty(t, got.Configs)
}

func TestSCIMAdmin_CreateTokenRejectsAnEmptyName(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "POST", "/v1/scim/configs/"+cfg.ID.String()+"/tokens",
		map[string]any{"name": ""}, owner)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

// Rotating or revoking a token that does not exist is a 404, not a 500.
func TestSCIMAdmin_UnknownTokenIsNotFound(t *testing.T) {
	f := newAdminFixture(t)
	_, owner := f.org(t, organization.RoleOwner)
	ghost := id.NewSCIMTokenID().String()

	assert.Equal(t, http.StatusNotFound,
		f.do(t, "DELETE", "/v1/scim/tokens/"+ghost, nil, owner).Code)
	assert.Equal(t, http.StatusNotFound,
		f.do(t, "POST", "/v1/scim/tokens/"+ghost+"/rotate", map[string]any{}, owner).Code)
}

func TestSCIMAdmin_DeleteRemovesIt(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "DELETE", "/v1/scim/configs/"+cfg.ID.String(), nil, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	after := f.do(t, "GET", "/v1/scim/configs/"+cfg.ID.String(), nil, owner)
	assert.Equal(t, http.StatusNotFound, after.Code)
}

func TestSCIMAdmin_MalformedIDsAreBadRequests(t *testing.T) {
	f := newAdminFixture(t)
	_, owner := f.org(t, organization.RoleOwner)

	assert.Equal(t, http.StatusBadRequest,
		f.do(t, "GET", "/v1/orgs/not-an-org/scim/configs", nil, owner).Code)
	assert.Equal(t, http.StatusBadRequest,
		f.do(t, "GET", "/v1/scim/configs/not-a-config", nil, owner).Code)
	assert.Equal(t, http.StatusBadRequest,
		f.do(t, "DELETE", "/v1/scim/tokens/not-a-token", nil, owner).Code)
}

// ── Tokens ───────────────────────────────────────────────

func TestSCIMAdmin_CreateTokenReturnsThePlaintextOnce(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "POST", "/v1/scim/configs/"+cfg.ID.String()+"/tokens",
		map[string]any{"name": "Okta production"}, owner)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var created TokenCreatedResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Plaintext)
	assert.NotEmpty(t, created.BaseURL, "the IdP needs the URL as well as the token")

	// Listing afterwards shows the record and never the value again.
	list := f.do(t, "GET", "/v1/scim/configs/"+cfg.ID.String()+"/tokens", nil, owner)
	require.Equal(t, http.StatusOK, list.Code)
	assert.NotContains(t, list.Body.String(), created.Plaintext)

	var tokens TokenListResponse
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &tokens))
	require.Len(t, tokens.Tokens, 1)
	assert.Equal(t, "Okta production", tokens.Tokens[0].Name)
}

func TestSCIMAdmin_CreateTokenRejectsAPastExpiry(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	rec := f.do(t, "POST", "/v1/scim/configs/"+cfg.ID.String()+"/tokens",
		map[string]any{"name": "stale", "expires_at": time.Now().Add(-time.Hour)}, owner)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

func TestSCIMAdmin_RotateMintsADifferentToken(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)

	first, tok, err := f.p.service.GenerateToken(context.Background(), cfg.ID, "first", nil)
	require.NoError(t, err)

	rec := f.do(t, "POST", "/v1/scim/tokens/"+tok.ID.String()+"/rotate",
		map[string]any{"name": "second", "grace_seconds": 60}, owner)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var rotated TokenCreatedResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rotated))
	assert.NotEqual(t, first, rotated.Plaintext)
}

func TestSCIMAdmin_RotateRejectsANegativeGrace(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)
	_, tok, err := f.p.service.GenerateToken(context.Background(), cfg.ID, "t", nil)
	require.NoError(t, err)

	rec := f.do(t, "POST", "/v1/scim/tokens/"+tok.ID.String()+"/rotate",
		map[string]any{"grace_seconds": -1}, owner)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

func TestSCIMAdmin_RevokeToken(t *testing.T) {
	f := newAdminFixture(t)
	orgID, owner := f.org(t, organization.RoleOwner)
	cfg := f.config(t, orgID)
	_, tok, err := f.p.service.GenerateToken(context.Background(), cfg.ID, "t", nil)
	require.NoError(t, err)

	rec := f.do(t, "DELETE", "/v1/scim/tokens/"+tok.ID.String(), nil, owner)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var status StatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &status))
	assert.Equal(t, "revoked", status.Status)
}

// A token id alone must not be enough. Without the ownership check on the
// configuration it belongs to, anyone could rotate or revoke anyone's token.
func TestSCIMAdmin_CannotTouchAnotherOrgsToken(t *testing.T) {
	f := newAdminFixture(t)
	_, mineOwner := f.org(t, organization.RoleOwner)
	theirsID, _ := f.org(t, organization.RoleOwner)
	theirCfg := f.config(t, theirsID)
	_, theirTok, err := f.p.service.GenerateToken(context.Background(), theirCfg.ID, "theirs", nil)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound,
		f.do(t, "DELETE", "/v1/scim/tokens/"+theirTok.ID.String(), nil, mineOwner).Code)
	assert.Equal(t, http.StatusNotFound,
		f.do(t, "POST", "/v1/scim/tokens/"+theirTok.ID.String()+"/rotate", map[string]any{}, mineOwner).Code)
}

func TestBoolOr(t *testing.T) {
	yes, no := true, false
	assert.True(t, boolOr(nil, true), "nil means take the default")
	assert.False(t, boolOr(nil, false))
	assert.True(t, boolOr(&yes, false), "a pointer wins over the default")
	assert.False(t, boolOr(&no, true))
}
