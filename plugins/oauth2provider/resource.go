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
// the raw query. It exists because a parameter the handler reads by hand is a
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
// Only /authorize needs this, and only because it is the one endpoint still
// reading its values by hand. The two POST endpoints bind theirs through the
// request struct, so forge describes those from the field itself.
//
// This replaced forge.WithParameter, which up to v1.9.11 wrote route metadata
// that the OpenAPI generator never read: it compiled, ran, returned no error,
// and put nothing in the document. Forge v1.9.12 fixed that, and the struct
// still stays. WithParameter takes no type argument and infers one from its
// example, so a field that says []string beats an example a later edit could
// quietly trim to a bare string.
type resourceQuery struct {
	Resource []string `query:"resource" description:"RFC 8707 resource indicator. Repeatable. Absolute URI, no fragment." optional:"true"`
}

// resourceParams reads the repeatable RFC 8707 resource parameter off a query
// string.
//
// Form bodies no longer need this: go-utils v1.1.7 taught bindFormParam to
// fill a []string from every occurrence of a parameter, so the token and
// device endpoints bind theirs through the struct. bindQueryParam did not get
// the same treatment. It still reads a single value through c.Query, and
// setFieldValue's new slice case then splits that one value on commas, so a
// []string query field keeps the first resource and silently discards the
// rest. Reading the query directly is the only way the authorization endpoint
// sees every value it was sent.
//
// The parameter is still described in the document. forge.WithQuerySchema
// takes a schema the request struct does not have to carry, so resourceQuery
// above declares the shape and generated clients can send it. go-utils v1.1.8
// fixes bindQueryParam, and once that lands AuthorizeRequest can carry a real
// query-tagged field and both this function and resourceQuery can go.
func resourceParams(r *http.Request) []string {
	if r == nil {
		return nil
	}

	return r.URL.Query()["resource"]
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
