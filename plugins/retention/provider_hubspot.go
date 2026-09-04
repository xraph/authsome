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
//   - Search notes: POST /crm/objects/2026-03/notes/search. Notes are one
//     of the five engagement types the CRM Search API covers, hs_note_body
//     is one of the note's default searchable properties, and
//     CONTAINS_TOKEN is a documented operator on it.
//     (developers.hubspot.com/docs/api-reference/latest/crm/search-the-crm)
const (
	hubspotDefaultBaseURL = "https://api.hubapi.com"

	hubspotContactsPath      = "/crm/objects/2026-03/contacts"
	hubspotContactSearchPath = hubspotContactsPath + "/search"
	hubspotNotesPath         = "/crm/objects/2026-03/notes"
	hubspotNoteSearchPath    = hubspotNotesPath + "/search"
	hubspotNoteToContactAssn = 202

	// hubspotExternalIDLabel prefixes the line that carries an activity's
	// external id in the note body.
	//
	// The body rather than a custom property, and not for want of trying. A
	// custom property would be the tidier home and it is not available to
	// us: it has to exist in the target portal before it can be written, and
	// HubSpot answers a create that names an undefined property with a 400.
	// That is the same constraint that keeps hubspotContactProperties to the
	// three built-ins. hs_note_body is a built-in, it is documented as
	// searchable, and it is present in every portal by definition.
	//
	// The cost is a line of machine text on a note a human may read. It is
	// one line, at the end, after the content.
	hubspotExternalIDLabel = "authsome_external_id"
)

// hubspotExternalIDMarker is the exact text the note body carries and the
// search result is verified against. CONTAINS_TOKEN matches tokens rather
// than substrings, so a hit is a candidate and not a proof; checking the
// body for this string is what turns it into one.
func hubspotExternalIDMarker(externalID string) string {
	return hubspotExternalIDLabel + ": " + externalID
}

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
// contacts and activities unconditionally, unlike the generic provider where
// CapActivities depends on config.
//
// CapActivityDedupe is claimed on the strength of LogActivity's
// search-before-create, with one honest limit: HubSpot's own docs say "it
// may take a few moments for newly created or updated CRM objects to appear
// in search results", so a redelivery that arrives inside that indexing lag
// can still create a second note. The window this exists to close is a
// failed MarkDone, which comes back only after the lease expires (two
// minutes by default), and that is well clear of a few moments. A
// redelivery driven by a failed attempt can come back in five seconds and
// is not.
func (h *HubSpotProvider) Capabilities() Capability {
	return CapContacts | CapActivities | CapActivityDedupe
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
//
// On a redelivery it looks for the note first. Delivery is at-least-once,
// and where a repeated contact upsert is absorbed by the contact ref,
// nothing absorbs a repeated note: the CRM just gets the same login logged
// twice, in front of the CS team who asked for this data in the first
// place.
//
// The search runs only when the worker says this job has been out before.
// A first delivery has nothing to collide with, and HubSpot rate limits the
// search endpoints to five requests per second per account, which is twenty
// times tighter than the ordinary object endpoints. Paying that on every
// login to protect the small fraction that get redelivered would cap the
// common case on the rarest one.
func (h *HubSpotProvider) LogActivity(ctx context.Context, ref RemoteRef, a *Activity) error {
	if a.Redelivery && a.ExternalID != "" {
		found, err := h.findNoteByExternalID(ctx, a.ExternalID)
		if err != nil {
			// Do not fall through to the create. The search failing means
			// we do not know whether the note is there, and creating on
			// "do not know" is exactly the duplicate this guard exists to
			// stop. The job is retried, and classifyHTTPError has already
			// decided whether that is worth doing.
			return err
		}
		if found {
			return nil
		}
	}

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
//
// A response we cannot make sense of is a different thing entirely, and
// must never be read as "not found": out["results"] missing or null, not
// being an array, a result that is not an object, or an id we cannot
// stringify, all mean we do not actually know whether a contact exists.
// Reading any of those as "no match" would send UpsertContact down the
// create path and mint a duplicate contact for a user who may already have
// one, which silently defeats the ref-dedup this whole design depends on
// and which nobody downstream can clean up. Only a genuinely empty
// []interface{} means "no such contact, go create one".
//
// Each of these is built as a *ProviderError directly rather than going
// through classifyHTTPError, because classifyHTTPError only classifies by
// HTTP status and this is a 2xx response whose body shape is the surprise.
// Retryable is set true: a response-shape surprise on an otherwise-successful
// status is more likely to be a transient gateway/proxy body, a brief
// HubSpot-side incident, or an API contract change than something specific
// to this one contact's payload, and classifyHTTPError's own rule --- "a
// failure that affects every job retries, a failure that affects only this
// job dies now" --- reads a shape surprise as affecting every job hitting
// this endpoint, not just this one. Terminal is defensible too if you would
// rather a persistent bad state fail loudly rather than fill the retry
// budget silently; this codebase chose the side that keeps a working
// integration syncing through a blip, matching the same choice
// classifyHTTPError makes for a plain 5xx.
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

	raw, present := out["results"]
	if !present || raw == nil {
		return "", &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: search response has no results field", h.name),
			Retryable: true,
		}
	}
	results, ok := raw.([]interface{})
	if !ok {
		return "", &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: search results is %T, want array", h.name, raw),
			Retryable: true,
		}
	}
	if len(results) == 0 {
		// A clean, well-shaped response that genuinely found nothing: the
		// normal "this is a new contact" case.
		return "", nil
	}

	// HubSpot can return more than one match even on an exact email filter
	// (a portal can loosen email's default uniqueness, or a shared/free-tier
	// account may carry test data). Taking the first is deliberate, not an
	// oversight: every candidate matched the same exact email filter, so
	// none is a better address match than another, and there is nothing else
	// in the response to rank them by. Picking the first keeps this a single
	// round trip instead of an ambiguous merge decision this package has no
	// basis to make on its own.
	first, ok := results[0].(map[string]interface{})
	if !ok {
		return "", &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: search result is %T, want object", h.name, results[0]),
			Retryable: true,
		}
	}
	id, ok := stringifyJSONLeaf(first["id"])
	if !ok {
		return "", &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: search result has an unreadable id", h.name),
			Retryable: true,
		}
	}
	return id, nil
}

// findNoteByExternalID reports whether a note carrying this activity's
// external id already exists. A clean search that matches nothing is not an
// error: it is the normal "this really is a new note" answer.
//
// The filter is CONTAINS_TOKEN on hs_note_body, which is a built-in and a
// documented searchable property for notes. CONTAINS_TOKEN matches whole
// tokens, so a hit is a candidate rather than a proof, and every candidate
// is checked against the exact marker text before it counts. The external
// id is a hex digest, which tokenises as one token, but relying on that
// alone would make a duplicate depend on somebody else's tokeniser.
//
// The response-shape handling mirrors findContactByEmail's, for a reason
// that is stronger here. A surprise we cannot read must never be reported
// as "no such note": that answer sends LogActivity straight to the create
// and mints the duplicate. Retryable is set for the same reason it is set
// there, that a shape surprise on a 2xx affects every job hitting this
// endpoint rather than this one activity.
func (h *HubSpotProvider) findNoteByExternalID(ctx context.Context, externalID string) (bool, error) {
	marker := hubspotExternalIDMarker(externalID)

	body := map[string]interface{}{
		"filterGroups": []map[string]interface{}{
			{
				"filters": []map[string]interface{}{
					{"propertyName": "hs_note_body", "operator": "CONTAINS_TOKEN", "value": externalID},
				},
			},
		},
		"properties": []string{"hs_note_body"},
		// More than one, because CONTAINS_TOKEN can return neighbours the
		// marker check then rejects, and a limit of one would let a single
		// near-miss hide the real note behind it.
		"limit": 10,
	}

	out, err := h.request(ctx, http.MethodPost, hubspotNoteSearchPath, body)
	if err != nil {
		return false, err
	}

	raw, present := out["results"]
	if !present || raw == nil {
		return false, &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: note search response has no results field", h.name),
			Retryable: true,
		}
	}
	results, ok := raw.([]interface{})
	if !ok {
		return false, &ProviderError{
			Err:       fmt.Errorf("retention: hubspot provider %q: note search results is %T, want array", h.name, raw),
			Retryable: true,
		}
	}

	for _, r := range results {
		note, ok := r.(map[string]interface{})
		if !ok {
			return false, &ProviderError{
				Err:       fmt.Errorf("retention: hubspot provider %q: note search result is %T, want object", h.name, r),
				Retryable: true,
			}
		}
		props, ok := note["properties"].(map[string]interface{})
		if !ok {
			return false, &ProviderError{
				Err:       fmt.Errorf("retention: hubspot provider %q: note search result has no properties object", h.name),
				Retryable: true,
			}
		}
		// A body we cannot read is not a shape surprise worth failing on,
		// unlike the cases above. The marker lives in the body, so a
		// result whose body is missing or is not a string cannot be the
		// note we are looking for; it is a neighbour the token filter
		// swept up. Skip it and keep checking the rest.
		if noteBody, ok := props["hs_note_body"].(string); ok && strings.Contains(noteBody, marker) {
			return true, nil
		}
	}

	// Either nothing matched, or everything that matched the token filter
	// turned out not to carry the marker. Both mean this activity has not
	// been written yet.
	return false, nil
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
//
// The external id goes last, on its own line, and it is what
// findNoteByExternalID looks for. Last so the part a person reads comes
// first, and in the same "key: value" shape as everything above it so it
// does not read as a glitch.
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

	if a.ExternalID != "" {
		fmt.Fprintf(&b, "\n%s", hubspotExternalIDMarker(a.ExternalID))
	}

	return b.String()
}
