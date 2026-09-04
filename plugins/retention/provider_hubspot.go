package retention

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Endpoint paths confirmed against HubSpot's current developer
// documentation (developers.hubspot.com) while implementing this provider,
// on 2026-09-03:
//
//   - Base host: https://api.hubapi.com
//     (developers.hubspot.com/docs/apps/developer-platform/build-apps/authentication/oauth/oauth-quickstart-guide)
//   - Auth: "Authorization: Bearer <private-app-token>", no other header
//     shape documented for private apps.
//     (developers.hubspot.com/docs/apps/developer-platform/build-apps/authentication/oauth/oauth-quickstart-guide,
//     developers.hubspot.com/docs/cms/reference/serverless-functions/serverless-functions-in-projects)
//   - HubSpot's REST APIs moved to date-based versioning; the docs say "for
//     new integrations, always use the latest date version"
//     (developers.hubspot.com/docs/api/overview,
//     developers.hubspot.com/docs/api-reference/latest/overview). The
//     current latest version is 2026-03, replacing the older /crm/v3/
//     paths.
//   - Search by email: POST /crm/objects/2026-03/contacts/search with a
//     filterGroups body.
//     (developers.hubspot.com/docs/api-reference/latest/crm/objects/contacts/search/search-contacts)
//   - Create contact: POST /crm/objects/2026-03/contacts
//     (developers.hubspot.com/docs/api-reference/latest/crm/objects/contacts/guide)
//   - Update contact by id: PATCH /crm/objects/2026-03/contacts/{contactId}
//     (developers.hubspot.com/docs/api-reference/latest/crm/objects/custom-objects/update-custom-object,
//     the same pattern documented for contacts)
//   - Log an activity: POST /crm/objects/2026-03/notes, associated to the
//     contact via the HubSpot-defined note-to-contact association type id
//     202.
//     (developers.hubspot.com/docs/api-reference/latest/crm/activities/notes/guide
//     shows the note-to-contact association example using type id 202;
//     confirmed independently via the HubSpot community and the live
//     association-types endpoint, which reports {"id": "202", "name":
//     "note_to_contact"}.)
const (
	hubspotDefaultBaseURL = "https://api.hubapi.com"

	hubspotContactsPath      = "/crm/objects/2026-03/contacts"
	hubspotContactSearchPath = hubspotContactsPath + "/search"
	hubspotNotesPath         = "/crm/objects/2026-03/notes"
	hubspotNoteToContactAssn = 202
)

// HubSpotProvider is the reference vendor Provider: a real CRM's contacts
// and notes (engagements) APIs mapped onto the two Provider methods.
type HubSpotProvider struct {
	name    string
	baseURL string
	token   string

	client *http.Client
}

// NewHubSpotProvider builds a Provider talking to HubSpot's CRM API. Token
// is required: an empty token would otherwise silently send every request
// unauthenticated, and HubSpot's contacts and notes endpoints both require
// scoped auth, so that would only surface as a 401 on the first delivery
// instead of at startup.
func NewHubSpotProvider(cfg ProviderConfig) (*HubSpotProvider, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("retention: hubspot provider requires a name")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("retention: hubspot provider %q requires a token", cfg.Name)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = hubspotDefaultBaseURL
	}

	return &HubSpotProvider{
		name:    cfg.Name,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   cfg.Token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name implements Provider.
func (h *HubSpotProvider) Name() string { return h.name }

// Capabilities implements Provider. HubSpot's contacts and notes APIs cover
// both, unconditionally, unlike the generic provider where CapActivities
// depends on config.
func (h *HubSpotProvider) Capabilities() Capability {
	return CapContacts | CapActivities
}

// UpsertContact implements Provider. It searches for an existing contact by
// email first: HubSpot's create endpoint does not upsert on email for
// standard contacts, so creating unconditionally would produce a duplicate
// contact on every repeat signup/login for the same user.
func (h *HubSpotProvider) UpsertContact(ctx context.Context, c *Contact) (RemoteRef, error) {
	remoteID, err := h.findContactByEmail(ctx, c.Email)
	if err != nil {
		return RemoteRef{}, err
	}

	props := hubspotContactProperties(c)

	if remoteID != "" {
		path := hubspotContactsPath + "/" + remoteID
		if _, err := h.request(ctx, http.MethodPatch, path, map[string]interface{}{"properties": props}); err != nil {
			return RemoteRef{}, err
		}
		return RemoteRef{Provider: h.name, ObjectType: "contact", ID: remoteID}, nil
	}

	out, err := h.request(ctx, http.MethodPost, hubspotContactsPath, map[string]interface{}{"properties": props})
	if err != nil {
		return RemoteRef{}, err
	}
	newID, ok := lookupJSONPath(out, "id")
	if !ok {
		return RemoteRef{}, &ProviderError{Err: fmt.Errorf("retention: hubspot provider %q: create response has no id field", h.name)}
	}
	return RemoteRef{Provider: h.name, ObjectType: "contact", ID: newID}, nil
}

// LogActivity implements Provider. It records the activity as a HubSpot
// note (the Notes/engagements API), associated to the contact via the
// HubSpot-defined note-to-contact association type.
func (h *HubSpotProvider) LogActivity(ctx context.Context, ref RemoteRef, a *Activity) error {
	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"hs_timestamp": a.OccurredAt.Format(time.RFC3339),
			"hs_note_body": hubspotNoteBody(a),
		},
		"associations": []map[string]interface{}{
			{
				"to": map[string]interface{}{"id": ref.ID},
				"types": []map[string]interface{}{
					{
						"associationCategory": "HUBSPOT_DEFINED",
						"associationTypeId":   hubspotNoteToContactAssn,
					},
				},
			},
		},
	}
	_, err := h.request(ctx, http.MethodPost, hubspotNotesPath, body)
	return err
}

// findContactByEmail runs the search-by-email call and returns the matching
// contact's id, or "" if there is no match. A search that runs cleanly but
// finds nothing is not an error: it is the normal "this is a new contact"
// path.
func (h *HubSpotProvider) findContactByEmail(ctx context.Context, email string) (string, error) {
	body := map[string]interface{}{
		"filterGroups": []map[string]interface{}{
			{
				"filters": []map[string]interface{}{
					{"propertyName": "email", "operator": "EQ", "value": email},
				},
			},
		},
		"properties": []string{"email"},
		"limit":      1,
	}

	out, err := h.request(ctx, http.MethodPost, hubspotContactSearchPath, body)
	if err != nil {
		return "", err
	}

	results, ok := out["results"].([]interface{})
	if !ok || len(results) == 0 {
		return "", nil
	}
	first, ok := results[0].(map[string]interface{})
	if !ok {
		return "", nil
	}
	id, ok := stringifyJSONLeaf(first["id"])
	if !ok {
		return "", nil
	}
	return id, nil
}

// request marshals payload (nil for none) as the request body, sends it
// with bearer auth, and decodes the JSON response on 2xx. Every transport
// failure and every non-2xx status is handed to classifyHTTPError, so
// HubSpot inherits the package's retry/dead-letter policy rather than
// restating it.
func (h *HubSpotProvider) request(ctx context.Context, method, path string, payload interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, &ProviderError{Err: fmt.Errorf("retention: hubspot provider %q: marshal request: %w", h.name, err)}
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, h.baseURL+path, body)
	if err != nil {
		return nil, &ProviderError{Err: fmt.Errorf("retention: hubspot provider %q: build request: %w", h.name, err)}
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// resp is nil here by net/http contract: a non-nil error from Do
		// always means resp is nil (dial failure, timeout, context
		// cancellation, ...).
		return nil, classifyHTTPError(nil, nil, err)
	}
	defer resp.Body.Close()

	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, classifyHTTPError(resp, nil, readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, classifyHTTPError(resp, respBytes, nil)
	}

	if len(respBytes) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, &ProviderError{Err: fmt.Errorf("retention: hubspot provider %q: decode response: %w", h.name, err)}
	}
	return out, nil
}

// hubspotContactProperties maps a Contact onto HubSpot's standard contact
// properties. Only the three built-in properties are sent: an arbitrary
// Traits key has no guarantee of existing as a custom property in the
// target portal, and HubSpot rejects a create/update that references an
// undefined property with a 400. A caller wanting Traits synced to HubSpot
// custom properties needs the generic provider's FieldMap instead, pointed
// at this same portal.
func hubspotContactProperties(c *Contact) map[string]interface{} {
	props := map[string]interface{}{"email": c.Email}
	if c.FirstName != "" {
		props["firstname"] = c.FirstName
	}
	if c.LastName != "" {
		props["lastname"] = c.LastName
	}
	return props
}

// hubspotNoteBody renders an Activity as the note's plain-text body: the
// activity type on the first line, then its properties one per line in a
// stable (sorted) order so the same activity always produces the same note
// text.
func hubspotNoteBody(a *Activity) string {
	var b strings.Builder
	b.WriteString(a.Type)

	if len(a.Properties) > 0 {
		keys := make([]string, 0, len(a.Properties))
		for k := range a.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "\n%s: %s", k, a.Properties[k])
		}
	}

	return b.String()
}
