package oauth2provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

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
// Both resolveResources and client registration enforce this same rule.
// Sharing the check here means the wording, and the rule itself, cannot drift
// between the two call sites.
func resourceURISyntaxError(raw string) string {
	if raw == "" {
		return "resource must not be empty"
	}

	// RFC 8707 section 2: the value MUST be an absolute URI and MUST NOT
	// carry a fragment. A fragment never reaches a server, so two values
	// differing only after the # would name the same resource while
	// comparing as different audiences.
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() {
		return fmt.Sprintf("resource %q is not an absolute URI", raw)
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Sprintf("resource %q must not include a fragment", raw)
	}

	return ""
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
