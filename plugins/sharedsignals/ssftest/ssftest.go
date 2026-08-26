// Package ssftest provides a backend-agnostic conformance suite for the
// sharedsignals.Store interface. Every backend (memory, sqlite, postgres,
// mongo) is expected to pass the same contract, so behavioral drift between
// them is caught here rather than in production.
//
// This store carries more security contract than most: received events are
// the replay guard for inbound SETs, and the audit reads are documented to
// answer ErrNotFound rather than a distinguishable "forbidden" when a caller
// reaches across a tenant boundary. Those are the invariants worth pinning
// on every backend.
package ssftest

import (
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	ssf "github.com/xraph/authsome/plugins/sharedsignals"
)

// Fixture is one backend ready to test, plus the tenants its rows hang off.
type Fixture struct {
	Store ssf.Store
	AppID id.AppID
	EnvID id.EnvironmentID
	// OtherAppID and OtherEnvID are a second tenant, used to prove that the
	// audit reads refuse to cross the boundary.
	OtherAppID id.AppID
	OtherEnvID id.EnvironmentID
	// UserID is a real user, for subject links.
	UserID id.UserID
}

// Factory builds a fresh, empty, migrated fixture for a single test.
type Factory func(t *testing.T) Fixture

// RunConformance runs every contract test against fixtures from newFixture.
func RunConformance(t *testing.T, newFixture Factory, skip ...string) {
	t.Helper()
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}
	cases := []struct {
		name string
		fn   func(t *testing.T, f Fixture)
	}{
		{"StreamCRUD", testStreamCRUD},
		{"StreamNotFound", testStreamNotFound},
		{"StreamLookupByPushPathHash", testStreamLookupByPushPathHash},
		{"ListStreamsIsAppScoped", testListStreamsIsAppScoped},
		{"StreamSliceFieldsRoundTrip", testStreamSliceFieldsRoundTrip},
		{"UpdateStream", testUpdateStream},
		{"DeleteStream", testDeleteStream},
		{"SubjectLinkUpsertIsIdempotent", testSubjectLinkUpsertIsIdempotent},
		{"SubjectLinkLookupIsTenantScoped", testSubjectLinkLookupIsTenantScoped},
		{"DuplicateJTIIsRejected", testDuplicateJTIIsRejected},
		{"SameJTIOnAnotherStreamIsAllowed", testSameJTIOnAnotherStreamIsAllowed},
		{"GetReceivedEventIsAppScoped", testGetReceivedEventIsAppScoped},
		{"ListReceivedEventsRefusesForeignStream", testListReceivedEventsRefusesForeignStream},
		{"ListReceivedEventsRespectsWindow", testListReceivedEventsRespectsWindow},
		{"ListReceivedEventsClampsLimit", testListReceivedEventsClampsLimit},
		{"DeleteReceivedEventFreesTheJTI", testDeleteReceivedEventFreesTheJTI},
		{"CountEventsSince", testCountEventsSince},
		{"ExpiredSignalIsNotActive", testExpiredSignalIsNotActive},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// unique returns a per-call unique fragment, so sub-tests sharing one backend
// never collide on a hash or a JTI.
func unique(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, id.NewSSFStreamID().String())
}

func newStream(appID id.AppID, envID id.EnvironmentID) *ssf.InboundStream {
	return &ssf.InboundStream{
		ID:                    id.NewSSFStreamID(),
		AppID:                 appID,
		EnvID:                 envID,
		Name:                  "Test Stream",
		Issuer:                "https://transmitter.example.test",
		Audience:              "https://receiver.example.test",
		JWKSURI:               "https://transmitter.example.test/jwks",
		PushPathHash:          unique("pathhash"),
		PushTokenHash:         unique("tokenhash"),
		AllowedEventTypes:     []string{"https://schemas.openid.net/secevent/caep/event-type/session-revoked"},
		AllowedSubjectFormats: []string{"iss_sub"},
		EnforcementMode:       "enforce",
		Status:                "active",
		MaxActionsPerHour:     100,
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
}

func newEvent(streamID id.SSFStreamID, jti string) *ssf.ReceivedEvent {
	return &ssf.ReceivedEvent{
		ID:          id.NewSSFEventID(),
		StreamID:    streamID,
		JTI:         jti,
		EventType:   sessionRevoked,
		SubjectJSON: `{"format":"iss_sub","iss":"https://transmitter.example.test","sub":"user-1"}`,
		Outcome:     "processed",
		ActionTaken: "session_revoked",
		ReceivedAt:  now(),
	}
}
