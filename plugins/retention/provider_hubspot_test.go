package retention

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: every test here drives an httptest.NewServer with BaseURL
// overridden, so nothing touches the network. classifyHTTPError itself
// (the retry-classification table) is tested exhaustively in
// provider_generic_classify_test.go; the 429 and 404 cases here exist to
// prove HubSpot's non-2xx responses actually reach that classifier rather
// than being handled some other way.

func TestHubSpotProviderCapabilities(t *testing.T) {
	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok"})
	require.NoError(t, err)

	caps := p.Capabilities()
	assert.True(t, caps.Has(CapContacts))
	assert.True(t, caps.Has(CapActivities))
	assert.True(t, caps.Has(CapActivityDedupe),
		"LogActivity searches before it recreates, so the bit is earned")
}

func TestHubSpotProviderName(t *testing.T) {
	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot-prod", Token: "tok"})
	require.NoError(t, err)
	assert.Equal(t, "hubspot-prod", p.Name())
}

func TestHubSpotProviderConstructor_RequiresName(t *testing.T) {
	_, err := NewHubSpotProvider(ProviderConfig{Token: "tok"})
	assert.Error(t, err)
}

func TestHubSpotProviderConstructor_RequiresToken(t *testing.T) {
	_, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot"})
	assert.Error(t, err, "an empty token must fail construction rather than send an unauthenticated request")
}

// hubspotFakeServer wires up a minimal fake of the three HubSpot endpoints
// this provider calls, and records what it receives so tests can assert on
// the actual requests, not just the returned ref.
type hubspotFakeServer struct {
	t *testing.T

	searchRequests []map[string]interface{}
	searchReqCount int
	searchStatus   int
	searchBody     string

	createRequests []map[string]interface{}
	createBody     string

	updateRequests []map[string]interface{}
	updatePaths    []string
	updateStatus   int
	updateBody     string
	updateHeader   http.Header

	noteRequests []map[string]interface{}

	// The note-search endpoint backs LogActivity's redelivery guard. Its
	// default is an empty result set, so a test that says nothing about it
	// gets "this note does not exist yet".
	noteSearchRequests []map[string]interface{}
	noteSearchStatus   int
	noteSearchBody     string

	authHeaders []string
}

func newHubSpotFakeServer(t *testing.T) *hubspotFakeServer {
	return &hubspotFakeServer{
		t:            t,
		searchStatus: http.StatusOK,
		searchBody:   `{"results":[]}`,
		createBody:   `{"id":"new-1"}`,
		updateStatus: http.StatusOK,
		updateBody:   `{}`,

		noteSearchStatus: http.StatusOK,
		noteSearchBody:   `{"results":[]}`,
	}
}

func (f *hubspotFakeServer) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.authHeaders = append(f.authHeaders, r.Header.Get("Authorization"))

		var body map[string]interface{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == hubspotContactSearchPath:
			f.searchRequests = append(f.searchRequests, body)
			f.searchReqCount++
			if f.searchStatus == http.StatusTooManyRequests {
				w.Header().Set("Retry-After", "30")
			}
			w.WriteHeader(f.searchStatus)
			_, _ = w.Write([]byte(f.searchBody))

		case r.Method == http.MethodPost && r.URL.Path == hubspotContactsPath:
			f.createRequests = append(f.createRequests, body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(f.createBody))

		case r.Method == http.MethodPatch && len(r.URL.Path) > len(hubspotContactsPath)+1 &&
			r.URL.Path[:len(hubspotContactsPath)] == hubspotContactsPath:
			f.updateRequests = append(f.updateRequests, body)
			f.updatePaths = append(f.updatePaths, r.URL.Path)
			f.updateHeader = r.Header.Clone()
			w.WriteHeader(f.updateStatus)
			_, _ = w.Write([]byte(f.updateBody))

		case r.Method == http.MethodPost && r.URL.Path == hubspotNoteSearchPath:
			f.noteSearchRequests = append(f.noteSearchRequests, body)
			w.WriteHeader(f.noteSearchStatus)
			_, _ = w.Write([]byte(f.noteSearchBody))

		case r.Method == http.MethodPost && r.URL.Path == hubspotNotesPath:
			f.noteRequests = append(f.noteRequests, body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"note-1"}`))

		default:
			f.t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}
}

func TestHubSpotProviderUpsertContact_UnknownEmailCreates(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok_abc", BaseURL: srv.URL})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{Email: "new@example.com", FirstName: "New", LastName: "Person"})
	require.NoError(t, err)

	assert.Equal(t, "new-1", ref.ID)
	assert.Equal(t, "hubspot", ref.Provider)
	assert.Equal(t, "contact", ref.ObjectType)

	require.Len(t, f.searchRequests, 1, "must search by email before deciding to create")
	require.Len(t, f.createRequests, 1)
	assert.Empty(t, f.updateRequests, "an unknown email must not hit the update path")

	createProps, _ := f.createRequests[0]["properties"].(map[string]interface{})
	assert.Equal(t, "new@example.com", createProps["email"])
	assert.Equal(t, "New", createProps["firstname"])
	assert.Equal(t, "Person", createProps["lastname"])

	require.NotEmpty(t, f.authHeaders)
	assert.Equal(t, "Bearer tok_abc", f.authHeaders[0])
}

func TestHubSpotProviderUpsertContact_KnownEmailUpdates(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.searchBody = `{"results":[{"id":"existing-42","properties":{"email":"known@example.com"}}]}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{Email: "known@example.com", FirstName: "Known"})
	require.NoError(t, err)

	assert.Equal(t, "existing-42", ref.ID, "must reuse the existing remote id, not mint a new one")

	assert.Empty(t, f.createRequests, "a known email must not hit the create path and duplicate the contact")
	require.Len(t, f.updateRequests, 1)
	require.Len(t, f.updatePaths, 1)
	assert.Equal(t, hubspotContactsPath+"/existing-42", f.updatePaths[0],
		"the update request must address the existing record's id")

	updateProps, _ := f.updateRequests[0]["properties"].(map[string]interface{})
	assert.Equal(t, "known@example.com", updateProps["email"])
	assert.Equal(t, "Known", updateProps["firstname"])
}

func TestHubSpotProviderUpsertContact_SearchRateLimited(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.searchStatus = http.StatusTooManyRequests
	f.searchBody = `{"message":"rate limited"}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "someone@example.com"})
	require.Error(t, err)

	var pe *ProviderError
	require.True(t, errors.As(err, &pe), "the error must be a ProviderError produced by classifyHTTPError")
	assert.True(t, pe.Retryable)
	assert.Equal(t, 30*time.Second, pe.RetryAfter)
}

func TestHubSpotProviderUpsertContact_UpdateNotFoundDropsRef(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.searchBody = `{"results":[{"id":"gone-99","properties":{"email":"ghost@example.com"}}]}`
	f.updateStatus = http.StatusNotFound
	f.updateBody = `{"message":"contact not found"}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "ghost@example.com"})
	require.Error(t, err)

	var pe *ProviderError
	require.True(t, errors.As(err, &pe))
	assert.True(t, pe.DropRef, "a 404 on the update path means the remote record is gone")
	assert.True(t, pe.Retryable, "the classifier retries a dropped-ref 404 so the recreate can happen")
}

// TestHubSpotProviderUpsertContact_MalformedSearchResponse drives a 2xx
// search response whose body does not look the way the docs describe, in
// each of the shapes that used to be silently treated as "no match found".
// Reading any of these as "not found" would send UpsertContact down the
// create path and mint a duplicate contact for a user who may already have
// one — the opposite of what a genuinely empty `results: []` means. Each
// case must come back as an error, and none may reach the create endpoint.
func TestHubSpotProviderUpsertContact_MalformedSearchResponse(t *testing.T) {
	cases := []struct {
		name       string
		searchBody string
	}{
		{"results field missing entirely", `{}`},
		{"results field present but null", `{"results":null}`},
		{"results field present but not an array", `{"results":"unexpected-string"}`},
		{"results field present but a number", `{"results":42}`},
		{"result present but not an object", `{"results":["not-an-object"]}`},
		{"result present but id is unreadable", `{"results":[{"id":null,"properties":{"email":"x@example.com"}}]}`},
		{"result present but id field is absent", `{"results":[{"properties":{"email":"x@example.com"}}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newHubSpotFakeServer(t)
			f.searchBody = tc.searchBody
			srv := httptest.NewServer(f.handler())
			defer srv.Close()

			p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
			require.NoError(t, err)

			_, err = p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
			require.Error(t, err, "an unreadable search response must not be treated as a clean miss")

			var pe *ProviderError
			require.True(t, errors.As(err, &pe), "the error must be a *ProviderError, not a bare error")
			assert.True(t, pe.Retryable,
				"a malformed response body on a 2xx status is treated as a transient shape surprise, not a permanent one")

			assert.Empty(t, f.createRequests,
				"a response we could not read must never fall through to create and duplicate an existing contact")
			assert.Empty(t, f.updateRequests)
		})
	}
}

// TestHubSpotProviderUpsertContact_UnknownEmailStillCreatesAfterTightening
// re-asserts the genuinely-empty-array case now that malformed shapes are
// rejected: {"results":[]} is the one shape that must still mean "create",
// not an error.
func TestHubSpotProviderUpsertContact_UnknownEmailStillCreatesAfterTightening(t *testing.T) {
	f := newHubSpotFakeServer(t)
	f.searchBody = `{"results":[]}`
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{Email: "brand-new@example.com"})
	require.NoError(t, err, "a genuinely empty results array must still mean \"no contact, go create one\"")
	assert.Equal(t, "new-1", ref.ID)
	require.Len(t, f.createRequests, 1)
}

func TestHubSpotProviderLogActivity_PostsNoteAssociatedToContact(t *testing.T) {
	f := newHubSpotFakeServer(t)
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	p, err := NewHubSpotProvider(ProviderConfig{Name: "hubspot", Token: "tok", BaseURL: srv.URL})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "contact-777"}
	occurred := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	err = p.LogActivity(t.Context(), ref, &Activity{
		Type:       "logged_in",
		OccurredAt: occurred,
		Properties: map[string]string{"ip": "203.0.113.5"},
	})
	require.NoError(t, err)

	require.Len(t, f.noteRequests, 1)
	note := f.noteRequests[0]

	props, _ := note["properties"].(map[string]interface{})
	require.NotNil(t, props)
	assert.Contains(t, props["hs_note_body"], "logged_in")
	assert.Contains(t, props["hs_note_body"], "203.0.113.5")
	assert.Equal(t, occurred.Format(time.RFC3339), props["hs_timestamp"])

	assocs, _ := note["associations"].([]interface{})
	require.Len(t, assocs, 1)
	assoc, _ := assocs[0].(map[string]interface{})
	to, _ := assoc["to"].(map[string]interface{})
	assert.Equal(t, "contact-777", to["id"])

	types, _ := assoc["types"].([]interface{})
	require.Len(t, types, 1)
	typ, _ := types[0].(map[string]interface{})
	assert.Equal(t, "HUBSPOT_DEFINED", typ["associationCategory"])
	assert.Equal(t, float64(202), typ["associationTypeId"])
}

func TestBuildProvidersBuildsAHubSpotProvider(t *testing.T) {
	built, err := buildProviders([]ProviderConfig{{
		Name: "crm", Type: "hubspot", Token: "tok",
	}})
	require.NoError(t, err)
	assert.Contains(t, built, "crm")
}

func TestBuildProvidersHubSpotWithoutTokenFails(t *testing.T) {
	_, err := buildProviders([]ProviderConfig{{Name: "crm", Type: "hubspot"}})
	assert.Error(t, err)
}
