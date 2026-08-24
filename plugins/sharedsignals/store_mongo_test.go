package sharedsignals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// The mongo store needs a live server, so the conformance suite does not run
// it by default. What we can check without one is that the doc converters
// preserve every field, which is where mongo backends usually drift from the
// SQL ones.

func TestMongoDocs_InboundStreamRoundTrip(t *testing.T) {
	verified := time.Now().UTC().Truncate(time.Millisecond)
	in := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Name: "okta", Issuer: "https://org.okta.com",
		Audience: "https://authsome.example/ssf", JWKSURI: "https://org.okta.com/keys",
		PushPathHash: "hp", PushTokenHash: "ht",
		AllowedEventTypes:     []string{"a"},
		AllowedSubjectFormats: []string{"iss_sub"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"a": "signal"},
		EnforcementMode:       EnforcementObserve, Status: StatusPaused,
		MaxActionsPerHour: 42, PendingVerifyState: "st",
		LastVerifiedAt: &verified, CreatedAt: verified, UpdatedAt: verified,
	}

	got, err := docToInboundStream(inboundStreamToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, in.ID, got.ID)
	assert.Equal(t, in.AllowedSubjectFormats, got.AllowedSubjectFormats)
	assert.Equal(t, in.ActionOverrides, got.ActionOverrides)
	assert.Equal(t, 42, got.MaxActionsPerHour)
	assert.Equal(t, StatusPaused, got.Status)
	require.NotNil(t, got.LastVerifiedAt)
}

func TestMongoDocs_SubjectLinkRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://i", Subject: "u1", UserID: id.NewUserID(),
		Source: SourceSocial, CreatedAt: now, LastSeenAt: now,
	}
	got, err := docToSubjectLink(subjectLinkToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, in.UserID, got.UserID)
	assert.Equal(t, SourceSocial, got.Source)
}

func TestMongoDocs_ReceivedEventRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		Outcome: OutcomeUnresolved, ReceivedAt: now,
	}
	got, err := docToReceivedEvent(receivedEventToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, "j", got.JTI)
	assert.True(t, got.ResolvedUserID.IsNil())
}

func TestMongoDocs_SignalRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &Signal{
		ID: id.NewSSFSignalID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		UserID: id.NewUserID(), StreamID: id.NewSSFStreamID(),
		EventType: "e", Severity: 75, Reason: "why",
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	got, err := docToSignal(signalToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, 75, got.Severity)
	assert.Equal(t, in.StreamID, got.StreamID)
}
