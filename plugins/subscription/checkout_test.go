package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/types"
)

// fakeCheckout records the last checkout it was asked about. A plan in
// free answers with no URL, so the change is applied straight away.
type fakeCheckout struct {
	last  CheckoutRequest
	calls int
	url   string
	err   error
	free  map[string]bool
}

func (f *fakeCheckout) Checkout(_ context.Context, req CheckoutRequest) (CheckoutResult, error) {
	f.last = req
	f.calls++
	if f.err != nil {
		return CheckoutResult{}, f.err
	}
	if f.free[req.Plan.Slug] {
		return CheckoutResult{}, nil
	}
	return CheckoutResult{URL: f.url}, nil
}

// fakePortal is a checkout provider that also offers a billing portal.
type fakePortal struct {
	fakeCheckout
	tenant, returnURL string
}

func (f *fakePortal) Portal(_ context.Context, tenantID, _ string, returnURL string) (string, error) {
	f.tenant, f.returnURL = tenantID, returnURL
	return "https://pay.example/portal", nil
}

// addPaidPlan adds an active "business" plan priced at 49.00 usd a month.
func (tf *tenantFixture) addPaidPlan(t *testing.T) *plan.Plan {
	t.Helper()
	pl := &plan.Plan{
		Name: "Business", Slug: "business", Currency: "usd", Status: plan.StatusActive, AppID: tf.appID,
		Pricing: &plan.Pricing{BaseAmount: types.Money{Amount: 4900, Currency: "usd"}, BillingPeriod: plan.PeriodMonthly},
	}
	require.NoError(t, tf.plugin.service.CreatePlan(context.Background(), pl))
	return pl
}

func TestSubscribe_CheckoutURLChangesNothing(t *testing.T) {
	tf := newTenantFixture(t)
	paid := tf.addPaidPlan(t)
	fc := &fakeCheckout{url: "https://pay.example/session/1"}
	tf.plugin.SetCheckoutProvider(fc)

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh,
		`{"plan":"business","return_url":"https://app.example/billing","cancel_url":"https://app.example/plans"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"checkout_url":"https://pay.example/session/1"}`, rec.Body.String())

	assert.Equal(t, tf.fresh.String(), fc.last.TenantID)
	assert.Equal(t, tf.appID, fc.last.AppID)
	assert.Equal(t, paid.ID, fc.last.Plan.ID)
	assert.Nil(t, fc.last.Current)
	assert.Nil(t, fc.last.CurrentPlan)
	assert.Equal(t, "https://app.example/billing", fc.last.ReturnURL)
	assert.Equal(t, "https://app.example/plans", fc.last.CancelURL)
	assert.Equal(t, tf.freshOwner, fc.last.UserID)

	// Nothing is granted until the payment completes.
	rec = tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, "")
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	assert.Nil(t, cur.Subscription)
}

func TestUpdate_CheckoutSeesCurrentSubscriptionAndPlan(t *testing.T) {
	tf := newTenantFixture(t)
	tf.addPaidPlan(t)
	fc := &fakeCheckout{url: "https://pay.example/session/2"}
	tf.plugin.SetCheckoutProvider(fc)

	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"business"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "https://pay.example/session/2", decodeMutation(t, rec).CheckoutURL)
	require.NotNil(t, fc.last.Current)
	assert.Equal(t, tf.subID, fc.last.Current.ID.String())
	require.NotNil(t, fc.last.CurrentPlan)
	assert.Equal(t, "pro", fc.last.CurrentPlan.Slug)
}

func TestCheckout_EmptyURLAppliesChange(t *testing.T) {
	tf := newTenantFixture(t)
	fc := &fakeCheckout{url: "https://pay.example/never", free: map[string]bool{"team": true}}
	tf.plugin.SetCheckoutProvider(fc)

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	assert.Empty(t, out.CheckoutURL)
	require.NotNil(t, out.Subscription)
	assert.Equal(t, "team", out.Subscription.Plan.Slug)
	assert.Equal(t, 1, fc.calls, "the provider decides, so it is asked about every change")
}

func TestCheckoutError_502(t *testing.T) {
	tf := newTenantFixture(t)
	tf.addPaidPlan(t)
	tf.plugin.SetCheckoutProvider(&fakeCheckout{err: errors.New("provider: account acct_internal_detail rejected")})

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"business"}`)
	// forge answers every 5xx with its status text, so the provider's own
	// error never reaches the client.
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.NotContains(t, rec.Body.String(), "acct_internal_detail", "the provider's error stays in the logs")
}

func TestCompleteCheckout_SubscribesOnceForRepeatedWebhooks(t *testing.T) {
	tf := newTenantFixture(t)
	paid := tf.addPaidPlan(t)
	req := CheckoutRequest{TenantID: tf.fresh.String(), AppID: tf.appID, Plan: paid, UserID: tf.freshOwner}

	first, err := tf.plugin.CompleteCheckout(context.Background(), req)
	require.NoError(t, err)
	second, err := tf.plugin.CompleteCheckout(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, paid.ID, first.PlanID)
	assert.Equal(t, 1, tf.liveSubscriptions(t, tf.fresh.String()))
}

func TestCompleteCheckout_MovesTheCurrentSubscription(t *testing.T) {
	tf := newTenantFixture(t)
	paid := tf.addPaidPlan(t)

	// The request was made before the tenant had a subscription, but it has
	// one now: that's the one that moves.
	sub, err := tf.plugin.CompleteCheckout(context.Background(), CheckoutRequest{
		TenantID: tf.org.String(), AppID: tf.appID, Plan: paid, UserID: tf.owner,
	})
	require.NoError(t, err)
	assert.Equal(t, tf.subID, sub.ID.String())
	assert.Equal(t, paid.ID, sub.PlanID)
	assert.Equal(t, 1, tf.liveSubscriptions(t, tf.org.String()))
}

func TestPortal_NotConfigured404(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/portal", tf.owner, &tf.org, `{"return_url":"https://app.example"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "billing portal is not configured")

	tf.plugin.SetCheckoutProvider(&fakeCheckout{})
	rec = tf.sendInOrg(t, http.MethodPost, "/v1/billing/portal", tf.owner, &tf.org, `{"return_url":"https://app.example"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, "a checkout provider without a portal still has none")
}

func TestPortal_ReturnsURL(t *testing.T) {
	tf := newTenantFixture(t)
	fp := &fakePortal{}
	tf.plugin.SetCheckoutProvider(fp)

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/portal", tf.owner, &tf.org, `{"return_url":"https://app.example/settings"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"url":"https://pay.example/portal"}`, rec.Body.String())
	assert.Equal(t, tf.org.String(), fp.tenant)
	assert.Equal(t, "https://app.example/settings", fp.returnURL)
}

func TestPortal_MemberForbidden(t *testing.T) {
	tf := newTenantFixture(t)
	tf.plugin.SetCheckoutProvider(&fakePortal{})
	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/portal", tf.member, &tf.org, `{}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
