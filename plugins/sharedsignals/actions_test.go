package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

type recordingRevoker struct {
	revoked []id.SessionID
	err     error
}

func (r *recordingRevoker) RevokeSession(_ context.Context, sessionID id.SessionID) error {
	if r.err != nil {
		return r.err
	}
	r.revoked = append(r.revoked, sessionID)
	return nil
}

type actionFixture struct {
	plugin   *Plugin
	stream   *InboundStream
	revoker  *recordingRevoker
	userID   id.UserID
	sessions []id.SessionID
}

func newActionFixture(t *testing.T) actionFixture {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	var sessions []id.SessionID
	for i := 0; i < 3; i++ {
		s := &session.Session{
			ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
			Token: "tok-" + string(rune('a'+i)), ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, authStore.CreateSession(ctx, s))
		sessions = append(sessions, s.ID)
	}

	rev := &recordingRevoker{}
	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Status: StatusEnabled,
		EnforcementMode: EnforcementEnforce, MaxActionsPerHour: 100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	return actionFixture{plugin: p, stream: stream, revoker: rev,
		userID: userID, sessions: sessions}
}

func TestActionFor_Defaults(t *testing.T) {
	p := New()
	s := &InboundStream{}
	cases := []struct {
		ev   caep.Event
		want string
	}{
		{caep.Event{Type: caep.EventSessionRevoked}, ActionRevokeAll},
		{caep.Event{Type: caep.EventTokenClaimsChange}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "revoke"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "delete"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "create"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}, ActionSignal},
		{caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}, ActionSignal},
		{caep.Event{Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}, ActionSignal},
		{caep.Event{Type: caep.EventVerification}, ActionNone},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, p.actionFor(s, tc.ev), "event %s", tc.ev.Type)
	}
}

func TestActionFor_StreamOverrideWins(t *testing.T) {
	p := New()
	s := &InboundStream{ActionOverrides: map[string]string{
		caep.EventSessionRevoked: ActionSignal,
	}}
	assert.Equal(t, ActionSignal, p.actionFor(s, caep.Event{Type: caep.EventSessionRevoked}))
}

func TestSeverityFor(t *testing.T) {
	p := New()
	assert.Equal(t, 100, p.severityFor(caep.Event{Type: caep.EventSessionRevoked}))
	assert.Equal(t, 90, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "revoke"}))
	assert.Equal(t, 20, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "create"}))
	assert.Equal(t, 70, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}))
	assert.Equal(t, 60, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "compliant"}))
	assert.Equal(t, 80, p.severityFor(caep.Event{
		Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}))
}

func TestApplyEvent_SessionRevokedRevokesEverySession(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeAll, action)
	assert.Len(t, f.revoker.revoked, 3)
}

// A session member in a complex subject narrows the blast radius to one
// session instead of signing the user out everywhere.
func TestApplyEvent_TargetedSessionRevoke(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, SessionID: f.sessions[1], Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeSession, action)
	require.Len(t, f.revoker.revoked, 1)
	assert.Equal(t, f.sessions[1], f.revoker.revoked[0])
}

// Observe mode must record the signal and skip the revocation, so an operator
// can watch a new stream before trusting it.
func TestApplyEvent_ObserveModeDoesNotRevoke(t *testing.T) {
	f := newActionFixture(t)
	f.stream.EnforcementMode = EnforcementObserve

	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionLog, action)
	assert.Empty(t, f.revoker.revoked)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1, "observe mode still records the signal")
}

func TestApplyEvent_AlwaysWritesASignal(t *testing.T) {
	f := newActionFixture(t)
	_, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.Equal(t, 60, signals[0].Severity)
	assert.Empty(t, f.revoker.revoked)
}

func TestApplyEvent_VerificationTakesNoUserAction(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventVerification, State: "abc"},
		Resolution{Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, "", action)
	assert.Empty(t, f.revoker.revoked)
}

func TestCircuitBreaker_TripsAndPausesStream(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t)
	f.stream.MaxActionsPerHour = 2
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, f.plugin.store.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: f.stream.ID,
			JTI: "prior-" + string(rune('a'+i)), EventType: caep.EventSessionRevoked,
			Outcome: OutcomeApplied, ActionTaken: ActionRevokeAll, ReceivedAt: now,
		}))
	}

	ok, err := f.plugin.checkCircuitBreaker(ctx, f.stream)
	require.NoError(t, err)
	assert.False(t, ok, "the breaker must trip at the limit")

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status,
		"a tripped breaker pauses the stream so it stops acting until a human looks")
}

func TestCircuitBreaker_AllowsUnderLimit(t *testing.T) {
	f := newActionFixture(t)
	ok, err := f.plugin.checkCircuitBreaker(context.Background(), f.stream)
	require.NoError(t, err)
	assert.True(t, ok)
}
