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

// classifierPolicyDecided marks that the retry-classification policy in
// classifyHTTPError has been written and recorded in the spec (see
// "Retry classification" in
// docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md). It
// used to gate construction while the policy was still an open decision; now
// that it is decided, the constant just documents that the guard was here
// and why: an unwritten policy that only surfaced at the first CRM error
// would decide, by accident, whether a transient 503 permanently drops a
// customer's sync.
const classifierPolicyDecided = true

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
//
// The policy is decided and recorded in "Retry classification" in
// docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md. One
// rule generates the whole table: a failure that affects every job retries,
// a failure that affects only this job dies now. `dead` is terminal and
// nothing in this codebase re-enqueues it, so dead-lettering a
// whole-integration problem (a revoked token, an outage) destroys the entire
// backlog over something an operator is about to fix, while retrying a bad
// payload only wastes a bounded number of requests.
func classifyHTTPError(resp *http.Response, body []byte, err error) *ProviderError {
	// The request never landed, so nobody decided anything.
	if resp == nil {
		return &ProviderError{Err: err, Retryable: true}
	}

	detail := fmt.Errorf("%s: %s", resp.Status, truncate(body, 512))

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return &ProviderError{Err: detail, Retryable: true, RetryAfter: retryAfter(resp)}

	case http.StatusUnauthorized, http.StatusForbidden:
		// Whole-integration failure: a bad or revoked token fails every job,
		// and a dead row is never re-enqueued. Retry flat and slow so a broken
		// credential does not burn its budget inside the first minute.
		return &ProviderError{Err: detail, Retryable: true, RetryAfter: 2 * time.Minute}

	case http.StatusNotFound:
		// Deleted upstream. Drop the ref so the retry recreates the contact
		// instead of updating a record that is gone.
		return &ProviderError{Err: detail, Retryable: true, DropRef: true}

	case http.StatusRequestTimeout, http.StatusConflict:
		return &ProviderError{Err: detail, Retryable: true}

	case http.StatusNotImplemented:
		// The endpoint does not implement this. Configuration, not weather.
		return &ProviderError{Err: detail}
	}

	if resp.StatusCode >= 500 {
		return &ProviderError{Err: detail, Retryable: true}
	}

	// Every other 4xx is this job's own payload: 400, 422, 413. It will not
	// become valid on the ninth attempt. Anything unrecognised is terminal
	// too, matching the package rule that an unclassified failure does not
	// retry forever.
	return &ProviderError{Err: detail}
}

// retryAfter parses the response's RFC 7231 Retry-After header, in both its
// delta-seconds form ("120") and its HTTP-date form
// ("Wed, 21 Oct 2026 07:28:00 GMT"). It clamps the result to [1s, 30m]: the
// floor stops a "Retry-After: 0" becoming a hot loop, and the ceiling exists
// because a server asking us to wait a day would park the job past the point
// anyone is watching. It returns 0 when the header is absent or unparseable,
// so the caller falls back to its own exponential backoff.
func retryAfter(resp *http.Response) time.Duration {
	const (
		floor = time.Second
		ceil  = 30 * time.Minute
	)

	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}

	var d time.Duration
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		d = time.Duration(secs) * time.Second
	} else if t, err := http.ParseTime(v); err == nil {
		d = time.Until(t)
	} else {
		return 0
	}

	if d < floor {
		return floor
	}
	if d > ceil {
		return ceil
	}
	return d
}

// truncate caps a response body to n bytes for inclusion in the detail
// error, so a large HTML error page does not bloat the last_error column. It
// indicates when truncation happened.
func truncate(body []byte, n int) string {
	if len(body) <= n {
		return string(body)
	}
	return string(body[:n]) + "... (truncated)"
}
