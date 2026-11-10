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

// fakeCheckout records the last checkout it was asked for.
type fakeCheckout struct {
	last CheckoutRequest
	url  string
	err  error
}

func (f *fakeCheckout) Checkout(_ context.Context, req CheckoutRequest) (string, error) {
	f.last = req
	return f.url, f.err
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

func TestSubscribe_PaidPlanReturnsCheckoutURL(t *testing.T) {
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
	assert.Equal(t, "https://app.example/billing", fc.last.ReturnURL)
	assert.Equal(t, "https://app.example/plans", fc.last.CancelURL)
	assert.Equal(t, tf.freshOwner, fc.last.UserID)

	// Nothing is granted until the provider completes the change.
	rec = tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, "")
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	assert.Nil(t, cur.Subscription)
}

func TestUpdate_PaidPlanPassesCurrentSubscription(t *testing.T) {
	tf := newTenantFixture(t)
	tf.addPaidPlan(t)
	fc := &fakeCheckout{url: "https://pay.example/session/2"}
	tf.plugin.SetCheckoutProvider(fc)

	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"business"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "https://pay.example/session/2", decodeMutation(t, rec).CheckoutURL)
	require.NotNil(t, fc.last.Current)
	assert.Equal(t, tf.subID, fc.last.Current.ID.String())
}

func TestSubscribe_FreePlanSkipsCheckout(t *testing.T) {
	tf := newTenantFixture(t)
	fc := &fakeCheckout{url: "https://pay.example/never"}
	tf.plugin.SetCheckoutProvider(fc)

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	assert.Empty(t, out.CheckoutURL)
	require.NotNil(t, out.Subscription)
	assert.Empty(t, fc.last.TenantID, "a free plan never reaches the provider")
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
