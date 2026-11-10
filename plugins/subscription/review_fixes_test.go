package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lsub "github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// setFixtureStatus rewrites the seeded subscription's status in the ledger
// store, the way the lifecycle clock would.
func (tf *tenantFixture) setFixtureStatus(t *testing.T, status lsub.Status) {
	t.Helper()
	ctx := context.Background()
	sub, err := tf.plugin.service.GetSubscription(ctx, fixtureSub(t, tf.billingFixture))
	require.NoError(t, err)
	sub.Status = status
	require.NoError(t, tf.plugin.ledgerStore.UpdateSubscription(ctx, sub))
}

func (tf *tenantFixture) liveSubscriptions(t *testing.T, tenantID string) int {
	t.Helper()
	subs, err := tf.plugin.service.ListSubscriptions(context.Background(), tenantID, tf.appID, lsub.ListOpts{})
	require.NoError(t, err)
	n := 0
	for _, s := range subs {
		if s.Status != lsub.StatusCanceled && s.Status != lsub.StatusExpired {
			n++
		}
	}
	return n
}

func TestCurrent_PastDueTenantIsStillSubscribed(t *testing.T) {
	tf := newTenantFixture(t)
	tf.setFixtureStatus(t, lsub.StatusPastDue)

	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.owner, &tf.org, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	require.NotNil(t, cur.Subscription, "a tenant in dunning still has a subscription")
	assert.Equal(t, "past_due", cur.Subscription.Status)

	rec = tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "no second subscription next to a past_due one")

	rec = tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/cancel", tf.owner, &tf.org, `{}`)
	assert.Equal(t, http.StatusOK, rec.Code, "a past_due tenant can still cancel: %s", rec.Body.String())
	assert.Equal(t, 1, tf.liveSubscriptions(t, tf.org.String()))
}

func TestCurrent_PausedTenant(t *testing.T) {
	tf := newTenantFixture(t)
	_, err := tf.plugin.service.PauseSubscription(context.Background(), fixtureSub(t, tf.billingFixture))
	require.NoError(t, err)

	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription", tf.owner, &tf.org, "")
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	require.NotNil(t, cur.Subscription)
	assert.Equal(t, "paused", cur.Subscription.Status)
	assert.NotNil(t, cur.Subscription.PausedAt)

	rec = tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "a pause can't be dodged by subscribing again")

	rec = tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team"}`)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "paused")
}

func TestKeep_DueCancellationRefused(t *testing.T) {
	tf := newTenantFixture(t)
	ctx := context.Background()
	sub, err := tf.plugin.service.GetSubscription(ctx, fixtureSub(t, tf.billingFixture))
	require.NoError(t, err)
	past := time.Now().Add(-time.Minute)
	sub.CancelAt = &past
	require.NoError(t, tf.plugin.ledgerStore.UpdateSubscription(ctx, sub))

	rec := tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription/resume", tf.owner, &tf.org, `{}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}

func TestUpdate_SeatedPlanToPlanWithoutSeats(t *testing.T) {
	tf := newTenantFixture(t)
	rec := tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"team","seats":7}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = tf.sendInOrg(t, http.MethodPatch, "/v1/billing/subscription", tf.owner, &tf.org, `{"plan":"pro"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	out := decodeMutation(t, rec)
	assert.Equal(t, "pro", out.Subscription.Plan.Slug)
	assert.Nil(t, out.Subscription.Seats)
}

func TestSubscribe_ConcurrentRequestsMakeOneSubscription(t *testing.T) {
	tf := newTenantFixture(t)
	const n = 8
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			codes[i] = tf.sendInOrg(t, http.MethodPost, "/v1/billing/subscription", tf.freshOwner, &tf.fresh, `{"plan":"team"}`).Code
		})
	}
	wg.Wait()

	ok := 0
	for _, c := range codes {
		if c == http.StatusOK {
			ok++
		} else {
			assert.Equal(t, http.StatusConflict, c)
		}
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, 1, tf.liveSubscriptions(t, tf.fresh.String()))
}

func TestTenantReads_UseTheRequestApp(t *testing.T) {
	tf := newTenantFixture(t)
	other := id.NewAppID().String()

	// An app_id on the query can't point a read at a different app than the
	// writes act on: both use the request's app.
	rec := tf.sendInOrg(t, http.MethodGet, "/v1/billing/subscription?app_id="+other, tf.owner, &tf.org, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	require.NotNil(t, cur.Subscription)
	assert.Equal(t, tf.appID, cur.Subscription.AppID)
}

// sendDelegated makes a request as an agent acting for userID.
func (tf *tenantFixture) sendDelegated(t *testing.T, method, path string, userID id.UserID, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	ctx := middleware.WithUserID(r.Context(), userID)
	ctx = middleware.WithOrgID(ctx, tf.org)
	ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), UserID: userID, AgentID: id.NewAgentID()})
	rec := httptest.NewRecorder()
	tf.handler.ServeHTTP(rec, r.WithContext(ctx))
	return rec
}

func TestTenantWrites_DelegatedSessionRefused(t *testing.T) {
	tf := newTenantFixture(t)
	for _, w := range [][3]string{
		{http.MethodPatch, "/v1/billing/subscription", `{"plan":"team"}`},
		{http.MethodPost, "/v1/billing/subscription/cancel", `{}`},
		{http.MethodPost, "/v1/billing/subscription/resume", `{}`},
		{http.MethodPost, "/v1/billing/portal", `{}`},
	} {
		rec := tf.sendDelegated(t, w[0], w[1], tf.owner, w[2])
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s", w[0], w[1])
	}

	rec := tf.sendDelegated(t, http.MethodGet, "/v1/billing/subscription", tf.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var cur CurrentSubscriptionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cur))
	assert.False(t, cur.CanManage, "an agent acting for an owner can't change billing")
}
