package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
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
	// This case is about decay, not policy: lift the cap so the score it
	// observes is the decayed severity rather than MaxRiskScore.
	p.config.MaxRiskScore = 100
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

// A confirmed upstream compromise should make the next sign-in step up, not
// bar the door. severityFor deliberately scores session-revoked at 100 so the
// stored signal is honest for forensics, but riskengine's default
// HighThreshold is 85, and a single contributor's score becomes the whole
// composite. Handing 100 straight through therefore decided "block", which
// locked a user out with correct credentials and a correct second factor for
// the first stretch of the signal's life. The score we hand the engine is a
// policy statement, separate from how bad the event was.
func TestEvaluateRisk_SessionRevokedChallengesRatherThanBlocks(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100, Reason: "compromised",
		EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), EnvID: envID.String(), Email: u.Email,
	})
	require.NoError(t, err)

	// riskengine's defaults: >= 85 blocks, >= 60 challenges.
	assert.Less(t, got.Score, 85,
		"a session-revoked signal must not reach the block threshold")
	assert.GreaterOrEqual(t, got.Score, 60,
		"but it must still clear the challenge threshold")
}

// The end-to-end proof, through riskengine's own decision logic rather than
// our arithmetic: OnBeforeSignIn returns an error only when the decision is
// block, so a nil error here means the user can still authenticate and be
// challenged instead of being refused outright.
func TestEvaluateRisk_SessionRevokedDoesNotBlockSignIn(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100, Reason: "compromised",
		EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
	}))

	engine := riskengine.New(p)
	err := engine.OnBeforeSignIn(ctx, &account.SignInRequest{
		AppID: appID, EnvID: envID, Email: u.Email, IPAddress: "203.0.113.10",
	})
	require.NoError(t, err,
		"a confirmed compromise must challenge the next sign-in, not refuse it")
}

// An operator who wants a confirmed compromise to bar the door outright can
// still have it, by raising the cap to the full severity.
func TestEvaluateRisk_MaxRiskScoreIsConfigurable(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	p.config.MaxRiskScore = 100
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100,
		EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), EnvID: envID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got.Score, 85,
		"raising the cap must restore the blocking behaviour")
}
