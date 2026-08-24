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

func TestMemoryStore_CountActionsSince(t *testing.T) {
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
	// Old, and one with no action taken. Neither counts.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "old", EventType: "e",
		Outcome: OutcomeApplied, ActionTaken: "revoked_all",
		ReceivedAt: now.Add(-2 * time.Hour),
	}))
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "noop", EventType: "e",
		Outcome: OutcomeIgnored, ReceivedAt: now,
	}))

	count, err := s.CountActionsSince(ctx, streamID, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMemoryStore_SignalsExpire(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, userID := id.NewAppID(), id.NewUserID()
	now := time.Now()

	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, UserID: userID,
		EventType: "live", Severity: 90,
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))
	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, UserID: userID,
		EventType: "expired", Severity: 90,
		EventAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	}))

	got, err := s.ListActiveSignals(ctx, appID, userID, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "live", got[0].EventType)

	// Another user's signals are not ours.
	got, err = s.ListActiveSignals(ctx, appID, id.NewUserID(), now)
	require.NoError(t, err)
	assert.Empty(t, got)
}
