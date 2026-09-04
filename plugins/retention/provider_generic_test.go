package retention

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// NOTE: the tests in this file exercise transport, auth and field-mapping
// through 2xx responses. The non-2xx paths, and classifyHTTPError's retry
// policy itself, are covered in provider_generic_classify_test.go, which
// drives real non-2xx responses through the same post path now that
// NewGenericProvider builds unconditionally.

func TestGenericProviderCapabilities_NoActivityURL(t *testing.T) {
	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		Type:       "generic",
		ContactURL: "http://example.invalid/contacts",
	})
	require.NoError(t, err)

	caps := p.Capabilities()
	assert.True(t, caps.Has(CapContacts))
	assert.False(t, caps.Has(CapActivities),
		"a provider with no activity_url must not advertise CapActivities")
}

func TestGenericProviderCapabilities_WithActivityURL(t *testing.T) {
	p, err := NewGenericProvider(ProviderConfig{
		Name:        "acme",
		Type:        "generic",
		ContactURL:  "http://example.invalid/contacts",
		ActivityURL: "http://example.invalid/activities",
	})
	require.NoError(t, err)

	caps := p.Capabilities()
	assert.True(t, caps.Has(CapContacts))
	assert.True(t, caps.Has(CapActivities))
}

func TestGenericProviderConstructor_RequiresContactURL(t *testing.T) {
	_, err := NewGenericProvider(ProviderConfig{Name: "acme", Type: "generic"})
	assert.Error(t, err, "a provider with no contact_url at all cannot upsert contacts, so construction must fail fast")
}

func TestGenericProviderConstructor_RequiresName(t *testing.T) {
	_, err := NewGenericProvider(ProviderConfig{Type: "generic", ContactURL: "http://example.invalid/contacts"})
	assert.Error(t, err)
}

func TestGenericProviderName(t *testing.T) {
	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme-crm",
		ContactURL: "http://example.invalid/contacts",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme-crm", p.Name())
}

func TestGenericProviderUpsertContact_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"remote-501"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		ContactURL: srv.URL,
	})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{
		UserID: id.NewUserID(),
		AppID:  id.NewAppID(),
		Email:  "someone@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "remote-501", ref.ID)
	assert.Equal(t, "acme", ref.Provider)
	assert.False(t, ref.IsZero())
}

func TestGenericProviderUpsertContact_DefaultBearerAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		ContactURL: srv.URL,
		Token:      "tok_abc123",
	})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok_abc123", gotAuth)
}

func TestGenericProviderUpsertContact_CustomHeaderAuth(t *testing.T) {
	var gotHeaderValue, gotAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaderValue = r.Header.Get("X-Api-Key")
		gotAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		ContactURL: srv.URL,
		AuthType:   "header",
		AuthHeader: "X-Api-Key",
		Token:      "sekrit",
	})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.NoError(t, err)
	assert.Equal(t, "sekrit", gotHeaderValue)
	assert.Empty(t, gotAuthHeader, "header auth must not also set Authorization")
}

func TestGenericProviderUpsertContact_FieldMapRenamesOutgoingFields(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		ContactURL: srv.URL,
		FieldMap: map[string]string{
			"email":      "EmailAddress",
			"first_name": "FirstName",
		},
	})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{
		Email:     "x@example.com",
		FirstName: "Xena",
		LastName:  "Warrior",
	})
	require.NoError(t, err)

	assert.Equal(t, "x@example.com", gotBody["EmailAddress"])
	assert.Equal(t, "Xena", gotBody["FirstName"])
	assert.Equal(t, "Warrior", gotBody["last_name"], "an unmapped field keeps its canonical name")
	assert.NotContains(t, gotBody, "email", "a renamed field must not also appear under its canonical name")
}

func TestGenericProviderUpsertContact_RemoteIDDefaultsToID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"default-id-1","other":"ignored"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{Name: "acme", ContactURL: srv.URL})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.NoError(t, err)
	assert.Equal(t, "default-id-1", ref.ID)
}

func TestGenericProviderUpsertContact_RemoteIDFromConfiguredPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"contact_id":"nested-777"}}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:       "acme",
		ContactURL: srv.URL,
		FieldMap: map[string]string{
			"remote_id": "result.contact_id",
		},
	})
	require.NoError(t, err)

	ref, err := p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.NoError(t, err)
	assert.Equal(t, "nested-777", ref.ID)
}

func TestGenericProviderLogActivity_Success(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:        "acme",
		ContactURL:  "http://example.invalid/contacts",
		ActivityURL: srv.URL,
	})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "acme", ObjectType: "contact", ID: "remote-501"}
	err = p.LogActivity(t.Context(), ref, &Activity{
		Type:       "login",
		OccurredAt: time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	assert.Equal(t, "login", gotBody["type"])
	assert.Equal(t, "remote-501", gotBody["contact_id"])
}

func TestGenericProviderLogActivity_FieldMapRenames(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{
		Name:        "acme",
		ContactURL:  "http://example.invalid/contacts",
		ActivityURL: srv.URL,
		FieldMap: map[string]string{
			"type":       "EventType",
			"contact_id": "ContactRef",
		},
	})
	require.NoError(t, err)

	ref := RemoteRef{Provider: "acme", ObjectType: "contact", ID: "remote-501"}
	err = p.LogActivity(t.Context(), ref, &Activity{Type: "signup", OccurredAt: time.Now()})
	require.NoError(t, err)

	assert.Equal(t, "signup", gotBody["EventType"])
	assert.Equal(t, "remote-501", gotBody["ContactRef"])
	assert.NotContains(t, gotBody, "type")
	assert.NotContains(t, gotBody, "contact_id")
}

func TestBuildProvidersBuildsAGenericProviderNowThatThePolicyIsDecided(t *testing.T) {
	built, err := buildProviders([]ProviderConfig{{
		Name: "crm", Type: "generic", ContactURL: "http://example.invalid/contacts",
	}})
	require.NoError(t, err, "the retry-classification policy is decided, so OnInit must be able to build this provider")
	assert.Contains(t, built, "crm")
}
