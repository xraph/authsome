package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
)

// tenantFixture is the billing fixture plus an owner and a plain member of
// the seeded org (which has an active "pro" subscription) and a second org,
// "fresh", with an owner and no subscription.
type tenantFixture struct {
	*billingFixture
	org, fresh          id.OrgID
	owner, member       id.UserID
	freshOwner          id.UserID
	team, archived, add *plan.Plan
}

func newTenantFixture(t *testing.T) *tenantFixture {
	t.Helper()
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	tf := &tenantFixture{
		billingFixture: f, org: org, fresh: id.NewOrgID(),
		owner: id.NewUserID(), member: id.NewUserID(), freshOwner: id.NewUserID(),
	}
	f.addMember(t, org, tf.owner, organization.RoleOwner)
	f.addMember(t, org, tf.member, organization.RoleMember)
	f.addMember(t, tf.fresh, tf.freshOwner, organization.RoleOwner)

	ctx := context.Background()
	tf.team = &plan.Plan{
		Name: "Team", Slug: "team", Currency: "usd", Status: plan.StatusActive, AppID: f.appID,
		Features: []plan.Feature{
			{ID: ledgerid.NewFeatureID(), Key: "seats", Name: "Seats", Type: plan.FeatureSeat, Limit: 10},
			{ID: ledgerid.NewFeatureID(), Key: "sso", Name: "SSO", Type: plan.FeatureBoolean, Limit: 1},
		},
	}
	tf.archived = &plan.Plan{Name: "Legacy", Slug: "legacy", Currency: "usd", Status: plan.StatusArchived, AppID: f.appID}
	tf.add = &plan.Plan{Name: "Extra", Slug: "extra", Currency: "usd", Status: plan.StatusActive, AppID: f.appID, Metadata: map[string]string{"addon": "true"}}
	for _, pl := range []*plan.Plan{tf.team, tf.archived, tf.add} {
		require.NoError(t, f.plugin.service.CreatePlan(ctx, pl))
	}
	return tf
}

func decodeMutation(t *testing.T, rec *httptest.ResponseRecorder) SubscriptionMutationResponse {
	t.Helper()
	var out SubscriptionMutationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func TestCurrentSubscription_NullWhenNone(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"subscription":null,"can_manage":true}`, rec.Body.String())
}

func TestCurrentSubscription_CanManageFlag(t *testing.T) {
	tf := newTenantFixture(t)

	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.owner, &tf.org, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Subscription)
	assert.Equal(t, "pro", body.Subscription.Plan.Slug)
	assert.True(t, body.CanManage)

	rec = tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.member, &tf.org, "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.False(t, body.CanManage, "a member reads the plan but can't change it")

	tf.settings[SettingSelfServiceUpgrade.Def.Key] = map[string]any{"": false}
	rec = tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.owner, &tf.org, "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.False(t, body.CanManage, "nobody self-serves while self-service is off")
}

func TestSubscribe_BySlug_OwnerGranted(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team","seats":4}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	require.NotNil(t, out.Subscription)
	assert.Empty(t, out.CheckoutURL)
	assert.Equal(t, "team", out.Subscription.Plan.Slug)
	assert.Equal(t, tf.fresh.String(), out.Subscription.TenantID)
	require.NotNil(t, out.Subscription.Seats)
	assert.Equal(t, int64(4), *out.Subscription.Seats.Limit)
	assert.Equal(t, int64(1), out.Subscription.Seats.Used)
}

func TestSubscribe_ByPlanID(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"`+tf.team.ID.String()+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "team", decodeMutation(t, rec).Subscription.Plan.Slug)
}

func TestSubscribe_MemberForbidden(t *testing.T) {
	tf := newTenantFixture(t)
	member := id.NewUserID()
	tf.addMember(t, tf.fresh, member, organization.RoleMember)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", member, &tf.fresh, `{"plan":"team"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestSubscribe_SelfServiceOff403(t *testing.T) {
	tf := newTenantFixture(t)
	tf.settings[SettingSelfServiceUpgrade.Def.Key] = map[string]any{"": false}
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "turned off")
}

func TestSubscribe_Conflict409(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}

func TestSubscribe_ArchivedAndAddonPlans400(t *testing.T) {
	tf := newTenantFixture(t)
	for _, slug := range []string{"legacy", "extra"} {
		rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"`+slug+`"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%s: %s", slug, rec.Body.String())
	}
}

func TestSubscribe_UnknownOrOtherAppPlan404(t *testing.T) {
	tf := newTenantFixture(t)
	other := &plan.Plan{Name: "Elsewhere", Slug: "elsewhere", Currency: "usd", Status: plan.StatusActive, AppID: id.NewAppID().String()}
	require.NoError(t, tf.plugin.service.CreatePlan(context.Background(), other))

	for _, ref := range []string{"nope", other.ID.String()} {
		rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"`+ref+`"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s: %s", ref, rec.Body.String())
	}
}

func TestUpdate_ChangesPlanReturnsView(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	assert.Equal(t, "team", out.Subscription.Plan.Slug)
	assert.Equal(t, tf.subID, out.Subscription.ID)
}

func TestUpdate_SeatsOnly(t *testing.T) {
	tf := newTenantFixture(t)
	require.Equal(t, http.StatusOK, tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`).Code)

	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"seats":7}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, int64(7), *decodeMutation(t, rec).Subscription.Seats.Limit)
}

func TestUpdate_SeatsWithoutSeatFeature400(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"seats":3}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestUpdate_NothingToChange400(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestUpdate_NoSubscription404(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestCancel_PeriodEndSetsCancelAt(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/cancel", tf.owner, &tf.org, `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	assert.NotNil(t, out.Subscription.CancelAt)
	assert.Nil(t, out.Subscription.CanceledAt)
	assert.Equal(t, "active", out.Subscription.Status)
}

func TestCancel_Now(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/cancel", tf.owner, &tf.org, `{"when":"now"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "canceled", decodeMutation(t, rec).Subscription.Status)
}

func TestCancel_BadWhen400(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/cancel", tf.owner, &tf.org, `{"when":"tomorrow"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResume_UndoesScheduledCancel(t *testing.T) {
	tf := newTenantFixture(t)
	require.Equal(t, http.StatusOK, tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/cancel", tf.owner, &tf.org, `{}`).Code)

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/resume", tf.owner, &tf.org, `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Nil(t, decodeMutation(t, rec).Subscription.CancelAt)

	rec = tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/resume", tf.owner, &tf.org, `{}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "nothing left to take back")
}

func TestEntitlements_AllKeys(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/entitlements", tf.member, &tf.org, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body EntitlementsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Contains(t, body.Entitlements, "sso")
	assert.True(t, body.Entitlements["sso"].Allowed)

	rec = tf.sendInOrg(t, http.MethodGet, "/v1/billing/entitlements", tf.freshOwner, &tf.fresh, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entitlements":{}}`, rec.Body.String())
}

func TestTenantWrites_CannotTargetOtherOrg(t *testing.T) {
	tf := newTenantFixture(t)
	// The fresh org's owner names the seeded org in the body.
	for _, w := range [][2]string{
		{http.MethodPatch, "/v1/billing/subscription"},
		{http.MethodPost, "/v1/billing/subscription/cancel"},
		{http.MethodPost, "/v1/billing/subscription/resume"},
	} {
		rec := tf.sendInOrg(t, w[0], w[1], tf.freshOwner, &tf.fresh, `{"plan":"team","tenant_id":"`+tf.org.String()+`"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s: %s", w[0], w[1], rec.Body.String())
	}

	sub, err := tf.plugin.service.GetSubscription(context.Background(), fixtureSub(t, tf.billingFixture))
	require.NoError(t, err)
	assert.Equal(t, "active", string(sub.Status))
	assert.Nil(t, sub.CancelAt)
}
