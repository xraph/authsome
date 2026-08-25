package oauth2provider

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/resourceuri"
)

// resourceQuery documents the RFC 8707 resource indicator on the authorization
// endpoint. It is never bound to and never populated.
//
// Nothing reads this struct at request time; resourceParams below does that off
// the raw request. It exists because a parameter the handler reads by hand is a
// parameter the generated document knows nothing about, and an undocumented
// parameter is one no SDK can send. forge.WithQuerySchema reflects over the tags
// for the document only -- its metadata key is read by the OpenAPI generator and
// by nothing else -- so declaring the shape here cannot put a []string in front
// of a binder that could not cope with one.
//
// The type is what carries the meaning. A query parameter defaults to style form
// with explode true, so an array says "send it again per value", which is what
// RFC 8707 asks for and what resourceParams already honours.
//
// Only /authorize needs this. It is a GET, so the query string is the only place
// the value can go. The two POST endpoints carry it as a body field instead, for
// the reason on TokenRequest.Resource.
//
// This replaced forge.WithParameter, which as of v1.9.11 wrote route metadata
// that the OpenAPI generator never read: it compiled, ran, returned no error,
// and put nothing in the document. That has since been fixed upstream, but the
// struct stays. WithParameter has no type argument and infers one from the
// example, and a field that says []string beats an example a later edit could
// quietly change to a bare string.
type resourceQuery struct {
	Resource []string `query:"resource" description:"RFC 8707 resource indicator. Repeatable. Absolute URI, no fragment." optional:"true"`
}

// resourceParams extracts the repeatable RFC 8707 resource parameter.
//
// This cannot go through the struct binder. go-utils' bindQueryParam reads a
// single value via url.Values.Get, and setFieldValue has no reflect.Slice
// case, so a []string field tagged query:"resource" does not merely lose the
// second value, it fails the whole request with "unsupported field type".
// Reading the raw request is the only way to honour a parameter the RFC
// defines as repeatable.
//
// Query values come first so a GET works without touching the body. For a POST
// the form is parsed too, because RFC 6749 sends token-endpoint parameters as
// application/x-www-form-urlencoded. ParseForm merges the query string into
// r.Form, so only r.PostForm is read here to avoid returning each query value
// twice.
func resourceParams(r *http.Request) []string {
	if r == nil {
		return nil
	}

	var out []string
	out = append(out, r.URL.Query()["resource"]...)

	if r.Method == http.MethodPost {
		// A parse failure means an unreadable body, which the handler will
		// reject on its own terms. There is nothing to add here.
		_ = r.ParseForm() //nolint:errcheck // handler rejects a malformed body
		out = append(out, r.PostForm["resource"]...)
	}

	return out
}

// resourceURISyntaxError checks a single RFC 8707 resource indicator against
// the syntax rule shared by request-time resolution and admin registration:
// the value must be an absolute URI and must not carry a fragment. It returns
// an empty string when the value is syntactically valid, and a description of
// the violation otherwise.
//
// resolveResources, client registration, the dashboard create form and the
// core's session.resource_identifier setting all enforce this same rule. It
// lives in internal/resourceuri because the last of those is in the root
// package, which this plugin imports, so the dependency cannot run the other
// way. Sharing it means the rule and its wording cannot drift between the
// value a client may ask for and the value a deployment answers to.
func resourceURISyntaxError(raw string) string {
	return resourceuri.SyntaxError(raw)
}

// resolveResources validates the requested resource indicators against the
// client's allowlist and returns the audience to grant.
//
// This mirrors resolveScopes, with one deliberate difference. An empty scope
// request yields the client's whole registered set, because a scope names
// something the client already holds. An empty resource request yields
// nothing, because widening a token to every resource a client may target is
// the opposite of what RFC 8707 is for.
//
// A client with an empty allowlist may target nothing. That is the state every
// client registered before this existed is in, and it is why the deny is safe:
// such a client has never sent a resource parameter, so it never reaches the
// rejection.
func resolveResources(client *OAuth2Client, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	allowed := make(map[string]struct{}, len(client.Resources))
	for _, r := range client.Resources {
		allowed[r] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))

	for _, raw := range requested {
		if msg := resourceURISyntaxError(raw); msg != "" {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target", msg)
		}

		if _, ok := allowed[raw]; !ok {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q is not registered for this client", raw))
		}

		if _, dup := seen[raw]; dup {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}

	return out, nil
}

// tokenRequestResources collects the resource indicators from a token
// request.
//
// A form-encoded request populates resourceParams and leaves the JSON field
// empty; a JSON request does the reverse, so in practice only one of the two
// ever carries values. If both do, the raw request wins and the two sets are
// never concatenated.
func tokenRequestResources(r *http.Request, req *TokenRequest) []string {
	if raw := resourceParams(r); len(raw) > 0 {
		return raw
	}
	return req.Resource
}

// narrowResources restricts an already-granted audience to the subset the
// token request asked for (RFC 8707 section 2.2).
//
// A token request may narrow what the user authorized but must never widen
// it. An empty request inherits the whole granted set, which is what a
// client that only sends `resource` at the authorization endpoint does.
func narrowResources(granted, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return granted, nil
	}

	grantedSet := make(map[string]struct{}, len(granted))
	for _, g := range granted {
		grantedSet[g] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, r := range requested {
		if _, ok := grantedSet[r]; !ok {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q was not granted by this authorization", r))
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out, nil
}
