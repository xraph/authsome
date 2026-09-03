package retention

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GenericProvider is the config-driven REST provider: it lets a CRM with no
// dedicated Go implementation work anyway, as long as it accepts a JSON POST
// and answers with a JSON body carrying the created/updated record's id.
type GenericProvider struct {
	name        string
	contactURL  string
	activityURL string
	authType    string
	authHeader  string
	token       string
	fieldMap    map[string]string

	client *http.Client
}

// NewGenericProvider builds a Provider from a ProviderConfig. ContactURL is
// required: Capabilities always advertises CapContacts, so a provider that
// could never actually reach a contacts endpoint must fail at construction
// rather than fail every delivery later.
func NewGenericProvider(cfg ProviderConfig) (*GenericProvider, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("retention: generic provider requires a name")
	}
	if cfg.ContactURL == "" {
		return nil, fmt.Errorf("retention: generic provider %q requires contact_url", cfg.Name)
	}

	return &GenericProvider{
		name:        cfg.Name,
		contactURL:  cfg.ContactURL,
		activityURL: cfg.ActivityURL,
		authType:    cfg.AuthType,
		authHeader:  cfg.AuthHeader,
		token:       cfg.Token,
		fieldMap:    cfg.FieldMap,
		client:      &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name implements Provider.
func (g *GenericProvider) Name() string { return g.name }

// Capabilities implements Provider. CapActivities is advertised only when
// ActivityURL is configured, so a CRM with no activity endpoint never gets
// handed an activity delivery it has no way to honour: the worker treats an
// unadvertised capability as a deliberate skip, not a failed send.
func (g *GenericProvider) Capabilities() Capability {
	caps := CapContacts
	if g.activityURL != "" {
		caps |= CapActivities
	}
	return caps
}

// UpsertContact implements Provider. It POSTs the contact, field-mapped, to
// ContactURL, and reads the remote id out of the JSON response using the
// FieldMap["remote_id"] path (default "id").
func (g *GenericProvider) UpsertContact(ctx context.Context, c *Contact) (RemoteRef, error) {
	body := g.mapFields(contactFields(c))

	respBody, err := g.post(ctx, g.contactURL, body)
	if err != nil {
		return RemoteRef{}, err
	}

	idPath := g.fieldMap["remote_id"]
	if idPath == "" {
		idPath = "id"
	}
	remoteID, ok := lookupJSONPath(respBody, idPath)
	if !ok {
		return RemoteRef{}, &ProviderError{
			Err: fmt.Errorf("retention: generic provider %q: response has no %q field", g.name, idPath),
		}
	}

	return RemoteRef{Provider: g.name, ObjectType: "contact", ID: remoteID}, nil
}

// LogActivity implements Provider. It POSTs the activity, field-mapped, to
// ActivityURL, with the contact's remote id attached under the canonical
// field "contact_id" (itself subject to FieldMap, like every other field).
func (g *GenericProvider) LogActivity(ctx context.Context, ref RemoteRef, a *Activity) error {
	if g.activityURL == "" {
		// Capabilities() never advertised CapActivities, so a caller that
		// respects it should never reach this. Guard anyway: silently
		// posting to an empty URL would fail confusingly far from the cause.
		return &ProviderError{Err: fmt.Errorf("retention: generic provider %q has no activity_url", g.name)}
	}

	fields := activityFields(a)
	fields["contact_id"] = ref.ID

	_, err := g.post(ctx, g.activityURL, g.mapFields(fields))
	return err
}

// post marshals fields as JSON, sends it with the configured auth header,
// and returns the decoded JSON response body on 2xx. Any transport failure
// or non-2xx status is handed to classifyHTTPError.
func (g *GenericProvider) post(ctx context.Context, url string, fields map[string]interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(fields)
	if err != nil {
		return nil, &ProviderError{Err: fmt.Errorf("retention: generic provider %q: marshal request: %w", g.name, err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Err: fmt.Errorf("retention: generic provider %q: build request: %w", g.name, err)}
	}
	req.Header.Set("Content-Type", "application/json")
	g.setAuth(req)

	resp, err := g.client.Do(req)
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
		return nil, &ProviderError{Err: fmt.Errorf("retention: generic provider %q: decode response: %w", g.name, err)}
	}
	return out, nil
}

// setAuth applies the configured auth to the outgoing request. AuthType
// "header" sends the token under AuthHeader (default "Authorization"
// verbatim, no "Bearer " prefix); anything else, including the empty
// string, sends it as an "Authorization: Bearer <token>" header.
func (g *GenericProvider) setAuth(req *http.Request) {
	if g.token == "" {
		return
	}
	if g.authType == "header" {
		header := g.authHeader
		if header == "" {
			header = "Authorization"
		}
		req.Header.Set(header, g.token)
		return
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
}

// mapFields renames the keys of a canonical field map through FieldMap. A
// key with no entry keeps its canonical name. FieldMap's "remote_id" entry
// is never consulted here: it names a path into the *response*, not an
// outgoing field, and contactFields/activityFields never produce a
// "remote_id" key for it to collide with.
func (g *GenericProvider) mapFields(canonical map[string]interface{}) map[string]interface{} {
	if len(g.fieldMap) == 0 {
		return canonical
	}
	out := make(map[string]interface{}, len(canonical))
	for k, v := range canonical {
		if mapped, ok := g.fieldMap[k]; ok && mapped != "" {
			out[mapped] = v
			continue
		}
		out[k] = v
	}
	return out
}

// contactFields is the canonical (pre-FieldMap) JSON shape of a Contact.
func contactFields(c *Contact) map[string]interface{} {
	out := map[string]interface{}{
		"user_id": c.UserID.String(),
		"app_id":  c.AppID.String(),
		"email":   c.Email,
	}
	if c.FirstName != "" {
		out["first_name"] = c.FirstName
	}
	if c.LastName != "" {
		out["last_name"] = c.LastName
	}
	if len(c.Traits) > 0 {
		out["traits"] = c.Traits
	}
	return out
}

// activityFields is the canonical (pre-FieldMap) JSON shape of an Activity.
// contact_id is added by the caller, since it comes from the RemoteRef, not
// the Activity itself.
func activityFields(a *Activity) map[string]interface{} {
	out := map[string]interface{}{
		"type":        a.Type,
		"occurred_at": a.OccurredAt.Format(time.RFC3339),
	}
	if len(a.Properties) > 0 {
		out["properties"] = a.Properties
	}
	return out
}

// lookupJSONPath walks a dot-separated path (e.g. "id", or "result.id") into
// a decoded JSON object and stringifies whatever it finds.
//
// It only descends through nested JSON objects. It does not support array
// indexing or wildcards, so a path segment that lands on a JSON array, or on
// anything but an object, partway through the walk is reported as not found
// rather than guessed at. A leaf value must be a string, number, or bool;
// null and object/array leaves are also reported as not found. This covers
// "id" and "data.id" shapes, which is what the brief calls for; a CRM whose
// id comes back nested inside an array is out of scope for this path
// language.
func lookupJSONPath(data map[string]interface{}, path string) (string, bool) {
	segments := strings.Split(path, ".")
	var cur interface{} = data
	for i, seg := range segments {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", false
		}
		val, exists := m[seg]
		if !exists {
			return "", false
		}
		if i == len(segments)-1 {
			return stringifyJSONLeaf(val)
		}
		cur = val
	}
	return "", false
}

// stringifyJSONLeaf converts a decoded JSON scalar to a string. encoding/json
// decodes all JSON numbers as float64, so that is the only numeric case.
func stringifyJSONLeaf(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, t != ""
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return "", false
	}
}

// classifyHTTPError maps a CRM's HTTP response onto a ProviderError so the
// worker can decide retry vs dead-letter. resp may be nil when the request
// never completed (dial failure, timeout).
func classifyHTTPError(resp *http.Response, body []byte, err error) *ProviderError {
	panic("classifyHTTPError: policy not yet decided, see Task 9 Step 3")
}
