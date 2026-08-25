package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestMemoryStore_InboundStreamRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID := id.NewAppID()

	in := &InboundStream{
		ID:                    id.NewSSFStreamID(),
		AppID:                 appID,
		EnvID:                 id.NewEnvironmentID(),
		Name:                  "okta-prod",
		Issuer:                "https://org.okta.com",
		Audience:              "https://authsome.example/ssf",
		JWKSURI:               "https://org.okta.com/oauth2/v1/keys",
		PushPathHash:          "hash-abc",
		PushTokenHash:         "hash-tok",
		AllowedEventTypes:     []string{"a", "b"},
		AllowedSubjectFormats: []string{"iss_sub"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"x": "signal"},
		EnforcementMode:       EnforcementEnforce,
		Status:                StatusEnabled,
		MaxActionsPerHour:     100,
	}
	require.NoError(t, s.CreateInboundStream(ctx, in))

	got, err := s.GetInboundStream(ctx, in.ID)
	require.NoError(t, err)
	assert.Equal(t, "okta-prod", got.Name)
	assert.Equal(t, []string{"iss_sub"}, got.AllowedSubjectFormats)
	assert.Equal(t, map[string]string{"x": "signal"}, got.ActionOverrides)

	byHash, err := s.GetInboundStreamByPushPathHash(ctx, "hash-abc")
	require.NoError(t, err)
	assert.Equal(t, in.ID, byHash.ID)

	_, err = s.GetInboundStreamByPushPathHash(ctx, "nope")
	require.ErrorIs(t, err, ErrNotFound)

	got.Status = StatusPaused
	require.NoError(t, s.UpdateInboundStream(ctx, got))
	after, err := s.GetInboundStream(ctx, in.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status)

	list, err := s.ListInboundStreams(ctx, appID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, s.DeleteInboundStream(ctx, in.ID))
	_, err = s.GetInboundStream(ctx, in.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStore_SubjectLinkUpsert(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	link := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: userID, Source: "sso",
	}
	require.NoError(t, s.UpsertSubjectLink(ctx, link))

	got, err := s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)

	// Upsert on the same tuple replaces rather than duplicating.
	other := id.NewUserID()
	require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: other, Source: "sso",
	}))
	got, err = s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, other, got.UserID)

	// A different app must not see it.
	_, err = s.GetSubjectLink(ctx, id.NewAppID(), envID, "https://i", "u1")
	require.ErrorIs(t, err, ErrNotFound)
}

// The unique (stream_id, jti) constraint is the replay guard, so the second
// insert of a jti must be distinguishable from any other failure.
func TestMemoryStore_ReceivedEventDedupe(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	streamID := id.NewSSFStreamID()

	first := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	}
	require.NoError(t, s.InsertReceivedEvent(ctx, first))

	err := s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	})
	require.ErrorIs(t, err, ErrDuplicateJTI)

	// The same jti on a different stream is a different event.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	}))

	first.Outcome = OutcomeApplied
	first.ActionTaken = "revoked_all"
	require.NoError(t, s.UpdateReceivedEvent(ctx, first))
}

func TestMemoryStore_CountEventsSince(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	streamID := id.NewSSFStreamID()
	now := time.Now()

	for i := 0; i < 3; i++ {
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID,
			JTI: "recent-" + string(rune('a'+i)), EventType: "e",
			Outcome: OutcomeApplied, ActionTaken: "revoked_all", ReceivedAt: now,
		}))
	}
	// Outside the window, so it does not count.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "old", EventType: "e",
		Outcome: OutcomeApplied, ActionTaken: "revoked_all",
		ReceivedAt: now.Add(-2 * time.Hour),
	}))
	// Inside the window and took no action, so it DOES count: the breaker
	// bounds what an authentic transmitter can make us record, not just
	// what it can make us revoke.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "noop", EventType: "e",
		Outcome: OutcomeIgnored, ReceivedAt: now,
	}))

	count, err := s.CountEventsSince(ctx, streamID, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestMemoryStore_StreamTimestampIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	streamID := id.NewSSFStreamID()
	verified := time.Now()

	stream := &InboundStream{
		ID:              streamID,
		AppID:           id.NewAppID(),
		EnvID:           id.NewEnvironmentID(),
		Name:            "test",
		Issuer:          "https://issuer",
		Audience:        "https://aud",
		JWKSURI:         "https://keys",
		LastVerifiedAt:  &verified,
		Status:          StatusEnabled,
		EnforcementMode: EnforcementEnforce,
	}
	require.NoError(t, s.CreateInboundStream(ctx, stream))

	// Capture the expected value as an independent copy before any mutation
	want := verified

	// Fetch twice
	first, err := s.GetInboundStream(ctx, streamID)
	require.NoError(t, err)
	require.NotNil(t, first.LastVerifiedAt)

	second, err := s.GetInboundStream(ctx, streamID)
	require.NoError(t, err)
	require.NotNil(t, second.LastVerifiedAt)

	// Mutate the pointee on the first result
	*first.LastVerifiedAt = want.Add(time.Hour)

	// A second clone must not see a mutation made through the first
	assert.True(t, second.LastVerifiedAt.Equal(want),
		"a second clone must not see a mutation made through the first")

	// The stored row must not see a mutation made through a clone
	third, err := s.GetInboundStream(ctx, streamID)
	require.NoError(t, err)
	assert.True(t, third.LastVerifiedAt.Equal(want),
		"the stored row must not see a mutation made through a clone")
}

func TestMemoryStore_SignalsExpire(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, userID := id.NewAppID(), id.NewUserID()
	envID := id.NewEnvironmentID()
	now := time.Now()

	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: userID,
		EventType: "live", Severity: 90,
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))
	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: userID,
		EventType: "expired", Severity: 90,
		EventAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	}))

	got, err := s.ListActiveSignals(ctx, appID, envID, userID, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "live", got[0].EventType)

	// Another user's signals are not ours.
	got, err = s.ListActiveSignals(ctx, appID, envID, id.NewUserID(), now)
	require.NoError(t, err)
	assert.Empty(t, got)

	// Signals in a different environment are not ours.
	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: id.NewEnvironmentID(), UserID: userID,
		EventType: "other-env", Severity: 90,
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))
	got, err = s.ListActiveSignals(ctx, appID, envID, userID, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "live", got[0].EventType)
}
