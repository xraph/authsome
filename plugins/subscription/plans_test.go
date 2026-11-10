package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/types"

	"github.com/xraph/authsome/id"
)

// planSlugs lists plans as userID; extra is appended to the query string.
func (tf *tenantFixture) planSlugs(t *testing.T, extra string, userID id.UserID) (int, []string) {
	t.Helper()
	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/plans?app_id="+tf.appID+extra, userID, &tf.org, "")
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	var body ListPlansResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	slugs := make([]string, 0, len(body.Plans))
	for _, pl := range body.Plans {
		slugs = append(slugs, pl.Slug)
	}
	assert.Equal(t, len(slugs), body.Total)
	return rec.Code, slugs
}

func TestListPlans_DefaultHidesArchivedAndAddons(t *testing.T) {
	tf := newTenantFixture(t)
	code, slugs := tf.planSlugs(t, "", tf.member)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{"pro", "team"}, slugs)
}

func TestListPlans_SortedByPrice(t *testing.T) {
	tf := newTenantFixture(t)
	ctx := context.Background()
	for _, p := range []struct {
		slug   string
		amount int64
	}{{"business", 4900}, {"starter", 900}} {
		require.NoError(t, tf.plugin.service.CreatePlan(ctx, &plan.Plan{
			Name: p.slug, Slug: p.slug, Currency: "usd", Status: plan.StatusActive, AppID: tf.appID,
			Pricing: &plan.Pricing{BaseAmount: types.Money{Amount: p.amount, Currency: "usd"}, BillingPeriod: plan.PeriodMonthly},
		}))
	}

	_, slugs := tf.planSlugs(t, "", tf.member)
	assert.Equal(t, []string{"pro", "team", "starter", "business"}, slugs, "unpriced plans first, then cheapest first")
}

func TestListPlans_IncludeAddons(t *testing.T) {
	tf := newTenantFixture(t)
	_, slugs := tf.planSlugs(t, "&include_addons=true", tf.member)
	assert.ElementsMatch(t, []string{"pro", "team", "extra"}, slugs)
}

func TestListPlans_StatusAllNeedsManage(t *testing.T) {
	tf := newTenantFixture(t)
	code, _ := tf.planSlugs(t, "&status=all", tf.member)
	assert.Equal(t, http.StatusForbidden, code)
	code, _ = tf.planSlugs(t, "&status=archived", tf.member)
	assert.Equal(t, http.StatusForbidden, code)

	operator := id.NewUserID()
	tf.engine.managers[operator.String()] = true
	code, slugs := tf.planSlugs(t, "&status=all&include_addons=true", operator)
	require.Equal(t, http.StatusOK, code)
	assert.ElementsMatch(t, []string{"pro", "team", "legacy", "extra"}, slugs)

	_, slugs = tf.planSlugs(t, "&status=archived", operator)
	assert.Equal(t, []string{"legacy"}, slugs)
}
