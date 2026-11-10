package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/types"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
)

func fixtureSub(t *testing.T, f *billingFixture) ledgerid.SubscriptionID {
	t.Helper()
	subID, err := ledgerid.ParseSubscriptionID(f.subID)
	require.NoError(t, err)
	return subID
}

func TestToSubscriptionView_EmbedsPlanAndRFC3339(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	ctx := context.Background()
	sub, err := f.plugin.service.GetSubscription(ctx, fixtureSub(t, f))
	require.NoError(t, err)

	view, err := f.plugin.toSubscriptionView(ctx, sub)
	require.NoError(t, err)
	assert.Equal(t, "Pro", view.Plan.Name)
	assert.Equal(t, "pro", view.Plan.Slug)
	assert.Equal(t, "active", view.Status)
	_, err = time.Parse(time.RFC3339, view.CurrentPeriodEnd)
	assert.NoError(t, err, "current_period_end should be RFC 3339, got %q", view.CurrentPeriodEnd)
	assert.Nil(t, view.Seats, "a plan without a seat feature has no seats")
}

func TestToSubscriptionView_ScheduledCancel(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	ctx := context.Background()
	require.NoError(t, f.plugin.service.CancelSubscription(ctx, fixtureSub(t, f), false))
	sub, err := f.plugin.service.GetSubscription(ctx, fixtureSub(t, f))
	require.NoError(t, err)

	view, err := f.plugin.toSubscriptionView(ctx, sub)
	require.NoError(t, err)
	require.NotNil(t, view.CancelAt)
	assert.Nil(t, view.CanceledAt)
	_, err = time.Parse(time.RFC3339, *view.CancelAt)
	assert.NoError(t, err)
}

func TestToSubscriptionView_Seats(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	ctx := context.Background()
	f.addOrg(t, org, id.NewUserID())
	for range 2 {
		require.NoError(t, f.authStore.CreateMember(ctx, &organization.Member{
			ID: id.NewMemberID(), OrgID: org, UserID: id.NewUserID(), Role: organization.RoleMember,
		}))
	}

	team := &plan.Plan{
		Name: "Team", Slug: "team", Currency: "usd", Status: plan.StatusActive, AppID: f.appID,
		Features: []plan.Feature{{ID: ledgerid.NewFeatureID(), Key: "seats", Name: "Seats", Type: plan.FeatureSeat, Limit: 10}},
	}
	require.NoError(t, f.plugin.service.CreatePlan(ctx, team))
	moved, err := f.plugin.service.ChangePlan(ctx, fixtureSub(t, f), team.ID, nil)
	require.NoError(t, err)

	view, err := f.plugin.toSubscriptionView(ctx, moved)
	require.NoError(t, err)
	require.NotNil(t, view.Seats)
	assert.Equal(t, "seats", view.Seats.FeatureKey)
	assert.Equal(t, int64(3), view.Seats.Used)
	require.NotNil(t, view.Seats.Limit)
	assert.Equal(t, int64(10), *view.Seats.Limit)

	moved, err = f.plugin.service.ChangePlan(ctx, fixtureSub(t, f), team.ID, map[string]int64{"seats": 5})
	require.NoError(t, err)
	view, err = f.plugin.toSubscriptionView(ctx, moved)
	require.NoError(t, err)
	assert.Equal(t, int64(5), *view.Seats.Limit, "a bought quantity overrides the plan's limit")
}

func TestPlanResponse_UnlimitedFeatureIsNull(t *testing.T) {
	pl := &plan.Plan{
		Name: "Pro", Slug: "pro", Status: plan.StatusActive,
		Features: []plan.Feature{
			{Key: "api_calls", Name: "API calls", Type: plan.FeatureMetered, Limit: -1},
			{Key: "projects", Name: "Projects", Type: plan.FeatureMetered, Limit: 5},
		},
	}
	resp := toPlanResponse(pl)
	require.Len(t, resp.Features, 2)
	assert.Nil(t, resp.Features[0].Limit)
	require.NotNil(t, resp.Features[1].Limit)
	assert.Equal(t, int64(5), *resp.Features[1].Limit)
}

func TestPlanResponse_AmountMinor(t *testing.T) {
	pl := &plan.Plan{
		Name: "Pro", Slug: "pro", Currency: "usd", Status: plan.StatusActive,
		Pricing: &plan.Pricing{BaseAmount: types.Money{Amount: 2900, Currency: "usd"}, BillingPeriod: plan.PeriodMonthly},
	}
	resp := toPlanResponse(pl)
	require.NotNil(t, resp.AmountMinor)
	assert.Equal(t, int64(2900), *resp.AmountMinor)
	assert.Equal(t, "29.00", resp.BaseAmount)
	assert.NotNil(t, resp.Features, "features serialises as [] rather than null")
}

func TestSubscriptionResponse_CancelAtAndPausedAt(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	ctx := context.Background()
	require.NoError(t, f.plugin.service.CancelSubscription(ctx, fixtureSub(t, f), false))
	sub, err := f.plugin.service.GetSubscription(ctx, fixtureSub(t, f))
	require.NoError(t, err)

	resp := toSubResponse(sub)
	require.NotNil(t, resp.CancelAt)
	assert.Nil(t, resp.PausedAt)
}
