package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
)

// sendInOrg makes a request as userID with activeOrg as the session's active
// organization, the way a session that switched org carries it. A nil
// activeOrg leaves the session without one.
func (f *billingFixture) sendInOrg(t *testing.T, method, path string, userID id.UserID, activeOrg *id.OrgID, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	ctx := middleware.WithUserID(r.Context(), userID)
	if activeOrg != nil {
		ctx = middleware.WithOrgID(ctx, *activeOrg)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, r.WithContext(ctx))
	return rec
}

// addMember adds userID to orgID with the given role.
func (f *billingFixture) addMember(t *testing.T, orgID id.OrgID, userID id.UserID, role organization.MemberRole) {
	t.Helper()
	require.NoError(t, f.authStore.CreateMember(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: orgID, UserID: userID, Role: role,
	}))
}

func TestActiveSubscription_DefaultsToActiveOrg(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	user := id.NewUserID()
	f.addOrg(t, org, user)

	rec := f.sendInOrg(t, http.MethodGet, "/v1/billing/subscriptions/active", user, &org, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body SubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, org.String(), body.TenantID)
}

func TestActiveSubscription_NoActiveOrg400(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	user := id.NewUserID()
	f.addOrg(t, org, user)

	rec := f.sendInOrg(t, http.MethodGet, "/v1/billing/subscriptions/active", user, nil, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "no active organization")
}

func TestActiveSubscription_UserModeDefaultsToCaller(t *testing.T) {
	user := id.NewUserID()
	f := newBillingFixture(t, "user", user.String())

	rec := f.sendInOrg(t, http.MethodGet, "/v1/billing/subscriptions/active", user, nil, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), user.String())
}

func TestEntitlementAndUsage_DefaultTenantAndApp(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	user := id.NewUserID()
	f.addOrg(t, org, user)

	for _, path := range []string{"/v1/billing/entitlements/sso", "/v1/billing/usage", "/v1/billing/invoices", "/v1/billing/subscriptions"} {
		rec := f.sendInOrg(t, http.MethodGet, path, user, &org, "")
		assert.Equal(t, http.StatusOK, rec.Code, "%s: %s", path, rec.Body.String())
	}
}

func TestActiveOrgOfAnotherOrg_404(t *testing.T) {
	victim := id.NewOrgID()
	f := newBillingFixture(t, "organization", victim.String())
	f.addOrg(t, victim, id.NewUserID())
	outsider := id.NewUserID()

	// A session claiming an org it doesn't belong to reads nothing of it.
	rec := f.sendInOrg(t, http.MethodGet, "/v1/billing/subscriptions/active", outsider, &victim, "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestOperatorListWithoutTenant_StillListsEveryTenant(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	other := id.NewOrgID()

	rec := f.sendInOrg(t, http.MethodGet, "/v1/billing/subscriptions?app_id="+f.appID, operator, &other, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body ListSubscriptionsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Total, "an operator's tenant-less list stays app-wide")
}

func TestCanManage_OwnerYesMemberNo(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	owner, member := id.NewUserID(), id.NewUserID()
	f.addMember(t, org, owner, organization.RoleOwner)
	f.addMember(t, org, member, organization.RoleMember)
	ctx := context.Background()

	ok, err := f.plugin.canManageTenant(ctx, owner, org.String())
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = f.plugin.canManageTenant(ctx, member, org.String())
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = f.plugin.canManageTenant(ctx, id.NewUserID(), org.String())
	require.NoError(t, err)
	assert.False(t, ok, "a non-member never manages")
}

func TestCanManage_OrgOverrideGrantsAdmin(t *testing.T) {
	org, otherOrg := id.NewOrgID(), id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	admin, otherAdmin := id.NewUserID(), id.NewUserID()
	f.addMember(t, org, admin, organization.RoleAdmin)
	f.addMember(t, otherOrg, otherAdmin, organization.RoleAdmin)
	f.settings[SettingBillingRoles.Def.Key] = map[string]any{org.String(): []string{"Owner", " admin "}}
	ctx := context.Background()

	ok, err := f.plugin.canManageTenant(ctx, admin, org.String())
	require.NoError(t, err)
	assert.True(t, ok, "the org's own override grants admin")

	ok, err = f.plugin.canManageTenant(ctx, otherAdmin, otherOrg.String())
	require.NoError(t, err)
	assert.False(t, ok, "another org keeps the owner-only default")
}

func TestCanManage_UserModeAndOperator(t *testing.T) {
	user := id.NewUserID()
	f := newBillingFixture(t, "user", user.String())
	ctx := context.Background()

	ok, err := f.plugin.canManageTenant(ctx, user, user.String())
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = f.plugin.canManageTenant(ctx, id.NewUserID(), user.String())
	require.NoError(t, err)
	assert.False(t, ok)

	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	ok, err = f.plugin.canManageTenant(ctx, operator, user.String())
	require.NoError(t, err)
	assert.True(t, ok)
}
