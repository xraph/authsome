package sharedsignals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// TestInboundStreamModel_RoundTrip asserts every field survives the
// from*/to* round trip. "want" is built independently of "in" (its own
// distinct *time.Time for LastVerifiedAt) so a mutation-or-drop bug in the
// mapper cannot compare equal to itself: got.LastVerifiedAt is the exact
// pointer that flowed in from "in", so comparing "in" against "got" would
// prove nothing about correctness.
func TestInboundStreamModel_RoundTrip(t *testing.T) {
	streamID := id.NewSSFStreamID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	verified := time.Now().UTC().Truncate(time.Second)
	created := time.Now().UTC().Truncate(time.Second).Add(-time.Hour)
	updated := time.Now().UTC().Truncate(time.Second)

	in := &InboundStream{
		ID:                    streamID,
		AppID:                 appID,
		EnvID:                 envID,
		Name:                  "okta",
		Issuer:                "https://org.okta.com",
		Audience:              "https://authsome.example/ssf",
		JWKSURI:               "https://org.okta.com/keys",
		PushPathHash:          "hp",
		PushTokenHash:         "ht",
		AllowedEventTypes:     []string{"a", "b"},
		AllowedSubjectFormats: []string{"iss_sub", "email"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"a": "signal"},
		EnforcementMode:       EnforcementObserve,
		Status:                StatusEnabled,
		MaxActionsPerHour:     50,
		PendingVerifyState:    "st",
		LastVerifiedAt:        &verified,
		CreatedAt:             created,
		UpdatedAt:             updated,
	}

	// want is a fully independent copy: same values, but its own
	// LastVerifiedAt pointer, own slices and own map, so it shares no
	// memory with "in" or with anything the mapper hands back.
	wantVerified := verified
	want := &InboundStream{
		ID:                    streamID,
		AppID:                 appID,
		EnvID:                 envID,
		Name:                  "okta",
		Issuer:                "https://org.okta.com",
		Audience:              "https://authsome.example/ssf",
		JWKSURI:               "https://org.okta.com/keys",
		PushPathHash:          "hp",
		PushTokenHash:         "ht",
		AllowedEventTypes:     []string{"a", "b"},
		AllowedSubjectFormats: []string{"iss_sub", "email"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"a": "signal"},
		EnforcementMode:       EnforcementObserve,
		Status:                StatusEnabled,
		MaxActionsPerHour:     50,
		PendingVerifyState:    "st",
		LastVerifiedAt:        &wantVerified,
		CreatedAt:             created,
		UpdatedAt:             updated,
	}

	got, err := toInboundStream(fromInboundStream(in))
	require.NoError(t, err)

	// Compare the *time.Time pointee separately, then blank it out on both
	// sides for the whole-struct comparison below.
	require.NotNil(t, got.LastVerifiedAt)
	require.NotNil(t, want.LastVerifiedAt)
	assert.True(t, got.LastVerifiedAt.Equal(*want.LastVerifiedAt))
	got.LastVerifiedAt = nil
	want.LastVerifiedAt = nil

	assert.Equal(t, want, got)
}

// A stream created without optional collections must not blow up on decode.
func TestInboundStreamModel_EmptyCollections(t *testing.T) {
	in := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Status: StatusEnabled, EnforcementMode: EnforcementEnforce,
	}
	got, err := toInboundStream(fromInboundStream(in))
	require.NoError(t, err)
	assert.Empty(t, got.AllowedEventTypes)
	assert.Empty(t, got.ActionOverrides)
	assert.Nil(t, got.LastVerifiedAt)
}

// TestSubjectLinkModel_RoundTrip asserts every field survives the round
// trip, including AppID, EnvID and Issuer, three of the four columns in
// this table's own (app_id, env_id, issuer, subject) unique index.
func TestSubjectLinkModel_RoundTrip(t *testing.T) {
	linkID := id.NewSSFLinkID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	created := time.Now().UTC().Truncate(time.Second).Add(-time.Minute)
	lastSeen := time.Now().UTC().Truncate(time.Second)

	in := &SubjectLink{
		ID: linkID, AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: userID, Source: SourceSSO,
		CreatedAt:  created,
		LastSeenAt: lastSeen,
	}
	want := &SubjectLink{
		ID: linkID, AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: userID, Source: SourceSSO,
		CreatedAt:  created,
		LastSeenAt: lastSeen,
	}

	got, err := toSubjectLink(fromSubjectLink(in))
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// SubjectLink.AppID has no default in the DDL (NOT NULL, no DEFAULT), so a
// corrupt value must fail the mapper rather than silently becoming the nil
// ID.
func TestSubjectLinkModel_MalformedAppID(t *testing.T) {
	m := &subjectLinkModel{
		ID:     id.NewSSFLinkID().String(),
		AppID:  "not-a-valid-id",
		EnvID:  id.NewEnvironmentID().String(),
		UserID: id.NewUserID().String(),
	}
	_, err := toSubjectLink(m)
	assert.Error(t, err)
}

// TestReceivedEventModel_RoundTrip asserts every field survives the round
// trip, including StreamID, half of the (stream_id, jti) replay-guard
// tuple that the old test never checked at all.
func TestReceivedEventModel_RoundTrip(t *testing.T) {
	eventID := id.NewSSFEventID()
	streamID := id.NewSSFStreamID()
	userID := id.NewUserID()
	receivedAt := time.Now().UTC().Truncate(time.Second)

	in := &ReceivedEvent{
		ID: eventID, StreamID: streamID, JTI: "j1",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		ResolvedUserID: userID, Outcome: OutcomeApplied,
		ActionTaken: "revoked_all", Error: "", ReceivedAt: receivedAt,
	}
	want := &ReceivedEvent{
		ID: eventID, StreamID: streamID, JTI: "j1",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		ResolvedUserID: userID, Outcome: OutcomeApplied,
		ActionTaken: "revoked_all", Error: "", ReceivedAt: receivedAt,
	}

	got, err := toReceivedEvent(fromReceivedEvent(in))
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// ReceivedEvent.StreamID is the replay-guard column: half of the unique
// (stream_id, jti) index. A corrupt value must fail the mapper rather than
// silently becoming "no stream", which would defeat replay protection.
func TestReceivedEventModel_MalformedStreamID(t *testing.T) {
	m := &receivedEventModel{
		ID:       id.NewSSFEventID().String(),
		StreamID: "not-a-valid-id",
		JTI:      "j1",
	}
	_, err := toReceivedEvent(m)
	assert.Error(t, err)
}

// An unresolved event has no user, and the zero ID must survive the trip
// rather than failing to parse.
func TestReceivedEventModel_NoResolvedUser(t *testing.T) {
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j1",
		EventType: "e", Outcome: OutcomeUnresolved,
		ReceivedAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toReceivedEvent(fromReceivedEvent(in))
	require.NoError(t, err)
	assert.True(t, got.ResolvedUserID.IsNil())
}

// TestSignalModel_RoundTrip asserts every field survives the round trip.
func TestSignalModel_RoundTrip(t *testing.T) {
	signalID := id.NewSSFSignalID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	streamID := id.NewSSFStreamID()
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(time.Hour)

	in := &Signal{
		ID: signalID, AppID: appID, EnvID: envID,
		UserID: userID, StreamID: streamID,
		EventType: "e", Severity: 90, Reason: "why",
		EventAt: now, ExpiresAt: expires, CreatedAt: now,
	}
	want := &Signal{
		ID: signalID, AppID: appID, EnvID: envID,
		UserID: userID, StreamID: streamID,
		EventType: "e", Severity: 90, Reason: "why",
		EventAt: now, ExpiresAt: expires, CreatedAt: now,
	}

	got, err := toSignal(fromSignal(in))
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// Signal.AppID has no default in the DDL (NOT NULL, no DEFAULT), so a
// corrupt value must fail the mapper rather than silently becoming the nil
// ID.
func TestSignalModel_MalformedAppID(t *testing.T) {
	m := &signalModel{
		ID:     id.NewSSFSignalID().String(),
		AppID:  "not-a-valid-id",
		UserID: id.NewUserID().String(),
	}
	_, err := toSignal(m)
	assert.Error(t, err)
}

// Signal.UserID has no default in the DDL either.
func TestSignalModel_MalformedUserID(t *testing.T) {
	m := &signalModel{
		ID:     id.NewSSFSignalID().String(),
		AppID:  id.NewAppID().String(),
		UserID: "not-a-valid-id",
	}
	_, err := toSignal(m)
	assert.Error(t, err)
}

func TestMigrationGroups_Named(t *testing.T) {
	assert.Equal(t, "authsome-sharedsignals", PostgresMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", SqliteMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", MongoMigrations.Name())
}
