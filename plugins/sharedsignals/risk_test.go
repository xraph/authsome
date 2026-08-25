package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/riskengine"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

func newRiskFixture(t *testing.T) (*Plugin, id.AppID, id.EnvironmentID, *user.User) {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	authStore := memory.New()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, EnvID: envID,
		Email: "target@corp.com", EmailVerified: true,
	}
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: envID,
		Email: u.Email, Verified: true, IsPrimary: true,
	}))

	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	return p, appID, envID, u
}

func TestEvaluateRisk_NoSignalsScoresZero(t *testing.T) {
	p, appID, _, u := newRiskFixture(t)
	got, err := p.EvaluateRisk(context.Background(), &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, got.Score)
}

func TestEvaluateRisk_FreshSignalScoresFull(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100, Reason: "compromised",
		EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Greater(t, got.Score, 90, "a signal received seconds ago has barely decayed")
	assert.Equal(t, "sharedsignals", got.Source)
	assert.Equal(t, 2.0, got.Weight)
}

// A signal near the end of its life should barely move the score, so an old
// event does not keep challenging a user forever.
func TestEvaluateRisk_SignalDecays(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100,
		EventAt: now.Add(-23 * time.Hour), ExpiresAt: now.Add(time.Hour),
		CreatedAt: now.Add(-23 * time.Hour),
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Less(t, got.Score, 20, "a nearly expired signal must have decayed")
	assert.Greater(t, got.Score, 0)
}

func TestEvaluateRisk_TakesTheHighestSignal(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	for _, sev := range []int{20, 90, 40} {
		require.NoError(t, p.store.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
			EventType: "e", Severity: sev,
			EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
		}))
	}

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Greater(t, got.Score, 80)
}

// A sign-in we cannot attribute to a user must score nothing rather than
// guessing.
func TestEvaluateRisk_UnknownUserScoresZero(t *testing.T) {
	p, appID, _, _ := newRiskFixture(t)
	got, err := p.EvaluateRisk(context.Background(), &riskengine.RiskRequest{
		AppID: appID.String(), Email: "stranger@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Score)
}

func TestPlugin_IsRiskContributor(_ *testing.T) {
	var _ riskengine.RiskContributor = New()
}
