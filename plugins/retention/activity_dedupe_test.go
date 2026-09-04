package retention

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"
)

// Delivery is at-least-once. A contact upsert survives that because the
// contact ref turns a repeat into an update; an activity has nothing playing
// that role, so a redelivery would put the same login in front of the CS
// team twice. These pin the guard that stops it, and the exact places it
// still does not reach.

const testExternalID = "0f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"

func newActivity(externalID string, redelivery bool) *Activity {
	return &Activity{
		Type:       "logged_in",
		OccurredAt: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC),
		Properties: map[string]string{"activity_type": "logged_in"},
		ExternalID: externalID,
		Redelivery: redelivery,
	}
}

// ── HubSpot ──────────────────────────────────────────────────────

func TestHubSpotLogActivity_FirstDeliveryDoesNotSearch(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity(testExternalID, false)))

	assert.Empty(t, f.noteSearchRequests,
		"a first delivery has nothing to collide with, and search is the rate-limited endpoint")
	require.Len(t, f.noteRequests, 1)
}

func TestHubSpotLogActivity_NoteBodyCarriesTheExternalID(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity(testExternalID, false)))

	require.Len(t, f.noteRequests, 1)
	props, _ := f.noteRequests[0]["properties"].(map[string]interface{})
	body, _ := props["hs_note_body"].(string)

	assert.Contains(t, body, hubspotExternalIDMarker(testExternalID),
		"without the marker in the body there is nothing for the search to find")
	assert.True(t, strings.HasPrefix(body, "logged_in"),
		"the part a person reads still comes first")

	// A custom property would be tidier and is not available: it has to
	// exist in the portal first, and HubSpot 400s a create that names an
	// undefined one.
	assert.Len(t, props, 2, "only the two built-in note properties are sent")
	assert.NotEmpty(t, props["hs_timestamp"])
}

func TestHubSpotLogActivity_RedeliveryFindsTheExistingNoteAndStops(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.noteSearchBody = `{"results":[{"id":"note-9","properties":{"hs_note_body":"logged_in\n` +
		hubspotExternalIDMarker(testExternalID) + `"}}]}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity(testExternalID, true)))

	require.Len(t, f.noteSearchRequests, 1)
	assert.Empty(t, f.noteRequests, "the note is already there; creating again is the duplicate")

	// The filter has to be the documented shape or the search silently
	// matches nothing and every redelivery duplicates.
	groups, _ := f.noteSearchRequests[0]["filterGroups"].([]interface{})
	require.Len(t, groups, 1)
	group, _ := groups[0].(map[string]interface{})
	filters, _ := group["filters"].([]interface{})
	require.Len(t, filters, 1)
	filter, _ := filters[0].(map[string]interface{})
	assert.Equal(t, "hs_note_body", filter["propertyName"])
	assert.Equal(t, "CONTAINS_TOKEN", filter["operator"])
	assert.Equal(t, testExternalID, filter["value"])
}

func TestHubSpotLogActivity_RedeliveryCreatesWhenNoNoteExists(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity(testExternalID, true)))

	require.Len(t, f.noteSearchRequests, 1)
	require.Len(t, f.noteRequests, 1,
		"the previous attempt died before writing anything; this activity still has to land")
}

// CONTAINS_TOKEN matches tokens, not substrings, so a hit is a candidate and
// not a proof. A result that comes back without the exact marker is not this
// activity and must not suppress the create.
func TestHubSpotLogActivity_RedeliveryIgnoresANonMatchingNeighbour(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.noteSearchBody = `{"results":[{"id":"note-8","properties":{"hs_note_body":"logged_in\n` +
		hubspotExternalIDMarker("some-other-activity") + `"}}]}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity(testExternalID, true)))

	require.Len(t, f.noteRequests, 1, "somebody else's note is not this activity")
}

// A search we cannot read is not "no such note". Treating it as one sends
// LogActivity straight to the create, which is the duplicate this guard is
// for. Every shape surprise stops the delivery instead.
func TestHubSpotLogActivity_UnreadableSearchNeverCreates(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no results field", `{}`},
		{"null results", `{"results":null}`},
		{"results is not an array", `{"results":{"id":"note-1"}}`},
		{"result is not an object", `{"results":["note-1"]}`},
		{"result has no properties", `{"results":[{"id":"note-1"}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newHubSpotFakeServer(t)
			f.noteSearchBody = tc.body
			srv := httptest.NewServer(f.handler())
			defer srv.Close()

			p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
			require.NoError(t, err)

			ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
			err = p.LogActivity(t.Context(), ref, newActivity(testExternalID, true))

			require.Error(t, err)
			assert.Empty(t, f.noteRequests, "a shape we cannot read must never be read as not-found")

			retryable, _ := Retryable(err)
			assert.True(t, retryable,
				"a surprise on a 2xx affects every job on this endpoint, not this one activity")
		})
	}
}

// A failed search is the same argument: we do not know, so we do not write.
func TestHubSpotLogActivity_FailedSearchDoesNotFallThroughToCreate(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.noteSearchStatus = http.StatusServiceUnavailable
	f.noteSearchBody = `{"status":"error"}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	err = p.LogActivity(t.Context(), ref, newActivity(testExternalID, true))

	require.Error(t, err)
	assert.Empty(t, f.noteRequests)
	retryable, _ := Retryable(err)
	assert.True(t, retryable, "a 503 reaches classifyHTTPError like any other")
}

// Nothing in the hooks produces an empty key, but a job carrying one has no
// external id to match on. Matching every note that also has none would be
// worse than creating.
func TestHubSpotLogActivity_NoExternalIDSkipsTheGuard(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "42"}
	require.NoError(t, p.LogActivity(t.Context(), ref, newActivity("", true)))

	assert.Empty(t, f.noteSearchRequests)
	require.Len(t, f.noteRequests, 1)
	props, _ := f.noteRequests[0]["properties"].(map[string]interface{})
	body, _ := props["hs_note_body"].(string)
	assert.NotContains(t, body, hubspotExternalIDLabel, "no id, no marker line")
}

// ── Generic ──────────────────────────────────────────────────────

func TestGenericProviderSendsExternalIDButPromisesNothing(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name: "crm", ContactURL: srv.URL + "/contacts", ActivityURL: srv.URL + "/activities",
	})
	require.NoError(t, err)

	require.NoError(t, p.LogActivity(t.Context(),
		RemoteRef{Provider: "crm", ObjectType: "contact", ID: "7"},
		newActivity(testExternalID, true)))

	assert.Equal(t, testExternalID, got["external_id"],
		"a CRM that can upsert on a field of its own should be given the chance")

	assert.False(t, p.Capabilities().Has(CapActivityDedupe),
		"sending the field is not knowing the far end honours it")
}

func TestGenericProviderOmitsAnEmptyExternalID(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name: "crm", ContactURL: srv.URL + "/contacts", ActivityURL: srv.URL + "/activities",
	})
	require.NoError(t, err)

	require.NoError(t, p.LogActivity(t.Context(),
		RemoteRef{Provider: "crm", ObjectType: "contact", ID: "7"}, newActivity("", false)))

	_, present := got["external_id"]
	assert.False(t, present, "an empty id would be a field mapping onto nothing")
}

func TestGenericProviderMapsTheExternalIDField(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name: "crm", ContactURL: srv.URL + "/contacts", ActivityURL: srv.URL + "/activities",
		FieldMap: map[string]string{"external_id": "dedupe_key"},
	})
	require.NoError(t, err)

	require.NoError(t, p.LogActivity(t.Context(),
		RemoteRef{Provider: "crm", ObjectType: "contact", ID: "7"},
		newActivity(testExternalID, true)))

	assert.Equal(t, testExternalID, got["dedupe_key"],
		"external_id goes through FieldMap like every other field")
	_, unmapped := got["external_id"]
	assert.False(t, unmapped)
}

// ── Worker ───────────────────────────────────────────────────────

func TestWorkerPassesTheIdempotencyKeyToTheProvider(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	w := newTestWorker(t, s, p)

	enqueued(t, s, KindActivityLog, "key-abc")
	w.runOnce(ctx)

	require.Len(t, p.activity, 1)
	assert.Equal(t, "key-abc", p.activity[0].ExternalID,
		"the job's own idempotency key is the activity's external id")
	assert.False(t, p.activity[0].Redelivery, "a fresh pending claim is not a redelivery")
}

// The case Risk B is actually about. MarkDone fails after the provider call
// succeeded, so the row stays in_flight with Attempts untouched: nothing got
// to record a failure. Only the reclaim after the lease expires knows.
func TestWorkerMarksAReclaimedJobAsRedelivery(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	w := newTestWorker(t, s, p)

	enqueued(t, s, KindActivityLog, "key-reclaim")

	// First delivery: claimed, sent, and then the store swallows MarkDone.
	first, err := s.ClaimDue(ctx, 10, time.Minute, time.Now())
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.False(t, first[0].Redelivered(), "the first time out is not a redelivery")

	// Lease expires and the worker picks it up again.
	again, err := s.ClaimDue(ctx, 10, time.Minute, time.Now().Add(2*time.Minute))
	require.NoError(t, err)
	require.Len(t, again, 1)
	assert.Equal(t, 0, again[0].Attempts,
		"nothing recorded a failure, so the attempt count is blind to this")
	assert.True(t, again[0].Reclaimed)
	assert.True(t, again[0].Redelivered())

	w.deliver(ctx, again[0])
	require.Len(t, p.activity, 1)
	assert.True(t, p.activity[0].Redelivery,
		"the provider has to be told, or it creates a second note")
	assert.Equal(t, "key-reclaim", p.activity[0].ExternalID)
}

func TestWorkerTreatsAFailedAttemptAsRedeliveryToo(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	w := newTestWorker(t, s, p)

	j := enqueued(t, s, KindActivityLog, "key-attempted")
	require.NoError(t, s.MarkRetry(ctx, j.ID, time.Now().Add(-time.Second), "503"))

	w.runOnce(ctx)

	require.Len(t, p.activity, 1)
	assert.True(t, p.activity[0].Redelivery,
		"an error is no proof the CRM did nothing; a read that failed after a successful "+
			"write looks exactly like a write that never landed")
}

// A provider that cannot deduplicate does not get to be quiet about it.
func TestWorkerWarnsWhenRedeliveringToAProviderWithoutDedupe(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}

	rec := &recordingLogger{Logger: log.NewNoopLogger()}
	w := newTestWorker(t, s, p)
	w.deps.Logger = rec

	j := enqueued(t, s, KindActivityLog, "key-warn")
	require.NoError(t, s.MarkRetry(ctx, j.ID, time.Now().Add(-time.Second), "503"))
	w.runOnce(ctx)

	require.Len(t, p.activity, 1, "the activity still goes; the warning is not a refusal")
	assert.True(t, rec.sawWarning("may end up with a duplicate"),
		"the gap has to be legible to whoever finds three logins on one contact")
}

func TestWorkerStaysQuietForAProviderThatCanDedupe(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities | CapActivityDedupe}

	rec := &recordingLogger{Logger: log.NewNoopLogger()}
	w := newTestWorker(t, s, p)
	w.deps.Logger = rec

	j := enqueued(t, s, KindActivityLog, "key-quiet")
	require.NoError(t, s.MarkRetry(ctx, j.ID, time.Now().Add(-time.Second), "503"))
	w.runOnce(ctx)

	assert.False(t, rec.sawWarning("may end up with a duplicate"))
}

// recordingLogger captures Warn messages so a test can assert the worker
// said something, without asserting on the whole logging surface.
type recordingLogger struct {
	log.Logger
	warnings []string
}

func (l *recordingLogger) Warn(msg string, _ ...log.Field) {
	l.warnings = append(l.warnings, msg)
}

func (l *recordingLogger) sawWarning(substr string) bool {
	for _, w := range l.warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
