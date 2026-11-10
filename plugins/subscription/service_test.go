package subscription

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"

	"github.com/xraph/ledger"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/store/memory"
	"github.com/xraph/ledger/subscription"
	"github.com/xraph/ledger/types"
)

// newLedgerService builds a Service over an in-memory ledger with two active
// plans in one app: a monthly "pro" and a yearly "team" with a 7-day trial.
func newLedgerService(t *testing.T) (svc *Service, appID string, pro, team *plan.Plan) {
	t.Helper()
	ctx := context.Background()
	appID = "app_test"
	st := memory.New()
	l := ledger.New(st)

	pro = &plan.Plan{
		Name: "Pro", Slug: "pro", Currency: "usd", Status: plan.StatusActive, AppID: appID,
		Pricing: &plan.Pricing{BaseAmount: types.Money{Amount: 2900, Currency: "usd"}, BillingPeriod: plan.PeriodMonthly},
	}
	team = &plan.Plan{
		Name: "Team", Slug: "team", Currency: "usd", Status: plan.StatusActive, AppID: appID, TrialDays: 7,
		Pricing: &plan.Pricing{BaseAmount: types.Money{Amount: 29000, Currency: "usd"}, BillingPeriod: plan.PeriodYearly},
	}
	require.NoError(t, l.CreatePlan(ctx, pro))
	require.NoError(t, l.CreatePlan(ctx, team))

	svc = &Service{ledger: l, ledgerStore: st}
	return svc, appID, pro, team
}

func subscribeOrFail(t *testing.T, svc *Service, planID ledgerid.PlanID, appID string) *subscription.Subscription {
	t.Helper()
	sub, err := svc.Subscribe(context.Background(), "tenant_a", planID, appID, nil)
	require.NoError(t, err)
	return sub
}

func TestResume_CanceledSubscriptionRefused(t *testing.T) {
	svc, appID, pro, _ := newLedgerService(t)
	ctx := context.Background()
	sub := subscribeOrFail(t, svc, pro.ID, appID)
	require.NoError(t, svc.CancelSubscription(ctx, sub.ID, true))

	_, err := svc.ResumeSubscription(ctx, sub.ID)
	require.Error(t, err)

	got, err := svc.GetSubscription(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, subscription.StatusCanceled, got.Status)
}

func TestChangePlan_CanceledSubscriptionRefused(t *testing.T) {
	svc, appID, pro, team := newLedgerService(t)
	ctx := context.Background()
	sub := subscribeOrFail(t, svc, pro.ID, appID)
	require.NoError(t, svc.CancelSubscription(ctx, sub.ID, true))

	_, err := svc.ChangePlan(ctx, sub.ID, team.ID, nil)
	assert.True(t, errors.Is(err, ledger.ErrSubscriptionCanceled), "got %v", err)
}

func TestChangePlan_ReturnsUpdatedSubscription(t *testing.T) {
	svc, appID, pro, team := newLedgerService(t)
	sub := subscribeOrFail(t, svc, pro.ID, appID)

	got, err := svc.ChangePlan(context.Background(), sub.ID, team.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, team.ID, got.PlanID)
}

func TestSubscribe_UsesPlanTrialAndPeriod(t *testing.T) {
	svc, appID, _, team := newLedgerService(t)
	sub := subscribeOrFail(t, svc, team.ID, appID)

	assert.Equal(t, subscription.StatusTrialing, sub.Status)
	require.NotNil(t, sub.TrialEnd)
	assert.WithinDuration(t, sub.CurrentPeriodStart.AddDate(0, 0, 7), *sub.TrialEnd, time.Second)
	assert.WithinDuration(t, sub.CurrentPeriodStart.AddDate(1, 0, 0), sub.CurrentPeriodEnd, 48*time.Hour)
}

func TestPause_ReturnsPausedSubscription(t *testing.T) {
	svc, appID, pro, _ := newLedgerService(t)
	ctx := context.Background()
	sub := subscribeOrFail(t, svc, pro.ID, appID)

	paused, err := svc.PauseSubscription(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, subscription.StatusPaused, paused.Status)
	assert.NotNil(t, paused.PausedAt)

	resumed, err := svc.ResumeSubscription(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, subscription.StatusActive, resumed.Status)
}

func TestKeep_ClearsScheduledCancel(t *testing.T) {
	svc, appID, pro, _ := newLedgerService(t)
	ctx := context.Background()
	sub := subscribeOrFail(t, svc, pro.ID, appID)
	require.NoError(t, svc.CancelSubscription(ctx, sub.ID, false))

	scheduled, err := svc.GetSubscription(ctx, sub.ID)
	require.NoError(t, err)
	require.NotNil(t, scheduled.CancelAt, "a period-end cancel should schedule cancel_at")

	kept, err := svc.KeepSubscription(ctx, sub.ID)
	require.NoError(t, err)
	assert.Nil(t, kept.CancelAt)
	assert.Equal(t, subscription.StatusActive, kept.Status)

	_, err = svc.KeepSubscription(ctx, sub.ID)
	assert.ErrorIs(t, err, ErrNoScheduledCancel)
}

func TestKeep_CanceledSubscriptionRefused(t *testing.T) {
	svc, appID, pro, _ := newLedgerService(t)
	ctx := context.Background()
	sub := subscribeOrFail(t, svc, pro.ID, appID)
	require.NoError(t, svc.CancelSubscription(ctx, sub.ID, true))

	_, err := svc.KeepSubscription(ctx, sub.ID)
	assert.ErrorIs(t, err, ledger.ErrSubscriptionCanceled)
}

// A lifecycle write the ledger refuses is the caller's mistake, not a server
// fault: changing or resuming an ended subscription answers 409.
func TestIDRoutes_EndedSubscriptionConflict(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true

	subID, err := ledgerid.ParseSubscriptionID(f.subID)
	require.NoError(t, err)
	require.NoError(t, f.plugin.service.CancelSubscription(context.Background(), subID, true))

	rec := f.send(t, http.MethodPost, "/v1/billing/subscriptions/"+f.subID+"/change-plan", operator, "", `{"new_plan_id":"`+f.planID+`"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	rec = f.send(t, http.MethodPost, "/v1/billing/subscriptions/"+f.subID+"/resume", operator, "", "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}
