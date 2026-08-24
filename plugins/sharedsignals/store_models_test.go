package sharedsignals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestInboundStreamModel_RoundTrip(t *testing.T) {
	verified := time.Now().UTC().Truncate(time.Second)
	in := &InboundStream{
		ID:                    id.NewSSFStreamID(),
		AppID:                 id.NewAppID(),
		EnvID:                 id.NewEnvironmentID(),
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
		CreatedAt:             verified,
		UpdatedAt:             verified,
	}

	got, err := toInboundStream(fromInboundStream(in))
	require.NoError(t, err)
	assert.Equal(t, in.ID, got.ID)
	assert.Equal(t, in.AllowedEventTypes, got.AllowedEventTypes)
	assert.Equal(t, in.AllowedSubjectFormats, got.AllowedSubjectFormats)
	assert.Equal(t, in.VerifiedDomains, got.VerifiedDomains)
	assert.Equal(t, in.ActionOverrides, got.ActionOverrides)
	require.NotNil(t, got.LastVerifiedAt)
	assert.True(t, in.LastVerifiedAt.Equal(*got.LastVerifiedAt))
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

func TestSubjectLinkModel_RoundTrip(t *testing.T) {
	in := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://i", Subject: "u1", UserID: id.NewUserID(), Source: SourceSSO,
		CreatedAt:  time.Now().UTC().Truncate(time.Second),
		LastSeenAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toSubjectLink(fromSubjectLink(in))
	require.NoError(t, err)
	assert.Equal(t, in.UserID, got.UserID)
	assert.Equal(t, in.Subject, got.Subject)
}

func TestReceivedEventModel_RoundTrip(t *testing.T) {
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j1",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		ResolvedUserID: id.NewUserID(), Outcome: OutcomeApplied,
		ActionTaken: "revoked_all", ReceivedAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toReceivedEvent(fromReceivedEvent(in))
	require.NoError(t, err)
	assert.Equal(t, in.JTI, got.JTI)
	assert.Equal(t, in.ResolvedUserID, got.ResolvedUserID)
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

func TestSignalModel_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	in := &Signal{
		ID: id.NewSSFSignalID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		UserID: id.NewUserID(), StreamID: id.NewSSFStreamID(),
		EventType: "e", Severity: 90, Reason: "why",
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	got, err := toSignal(fromSignal(in))
	require.NoError(t, err)
	assert.Equal(t, 90, got.Severity)
	assert.Equal(t, in.UserID, got.UserID)
}

func TestMigrationGroups_Named(t *testing.T) {
	assert.Equal(t, "authsome-sharedsignals", PostgresMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", SqliteMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", MongoMigrations.Name())
}
