package oauth2provider

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// RFC 7591 section 3.2.2 error codes.
const (
	errInvalidRedirectURI    = "invalid_redirect_uri"
	errInvalidClientMetadata = "invalid_client_metadata"
	errAccessDenied          = "access_denied"
)

// dynamicGrantTypes is everything a dynamically registered client may hold.
// client_credentials is absent deliberately: issueClientToken mints a real
// session with a nil user, so an open endpoint handing it out would give
// anyone who can reach /register a working service token with no consent
// step. The device grant is absent for the same reason.
var dynamicGrantTypes = map[string]struct{}{
	"authorization_code": {},
	"refresh_token":      {},
}

// oauthRegError is an RFC 7591 section 3.2.2 error. Forge serialises the
// value returned by ResponseBody verbatim for any status below 500, so this
// puts {"error": ..., "error_description": ...} on the wire instead of the
// house envelope. MCP clients parse those two fields and nothing else.
type oauthRegError struct {
	status int
	Code   string `json:"error"`
	Desc   string `json:"error_description,omitempty"`
}

func (e *oauthRegError) Error() string {
	if e.Desc == "" {
		return e.Code
	}
	return e.Code + ": " + e.Desc
}

func (e *oauthRegError) StatusCode() int   { return e.status }
func (e *oauthRegError) ResponseBody() any { return e }

func regError(status int, code, desc string) *oauthRegError {
	return &oauthRegError{status: status, Code: code, Desc: desc}
}

// validateRedirectURI reports whether a redirect URI may be registered.
//
// Runtime matching is already exact-string in resolveRedirectURI, so this is
// only about what may be recorded in the first place. Three shapes pass:
// https with a real host; http on the loopback literals, which is how
// desktop and CLI clients receive a code (RFC 8252 section 7.3); and a
// private-use scheme containing a dot, which is the RFC 8252 section 7.1
// reverse-domain convention.
//
// http://localhost by name is refused on purpose. Its resolution depends on
// the client host's DNS and hosts file, so it is not the guaranteed-local
// target the literal IP is, and RFC 8252 recommends the IP for that reason.
func validateRedirectURI(raw string) error {
	if raw == "" {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not be empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			fmt.Sprintf("redirect_uri %q is not a valid URI", raw))
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not contain a fragment")
	}
	if u.User != nil {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not contain userinfo")
	}

	switch u.Scheme {
	case "https":
		if u.Hostname() == "" {
			return regError(http.StatusBadRequest, errInvalidRedirectURI,
				"https redirect_uri requires a host")
		}
		if strings.Contains(u.Hostname(), "*") {
			return regError(http.StatusBadRequest, errInvalidRedirectURI,
				"redirect_uri must not contain a wildcard")
		}
		return nil

	case "http":
		host := u.Hostname()
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			return nil
		}
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"http redirect_uri is only allowed on a loopback address (127.0.0.0/8 or [::1])")

	case "":
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must be absolute")

	default:
		// Private-use scheme. RFC 8252 section 7.1 wants a reverse-domain
		// name the app controls, so require a dot. A bare "myapp:" scheme
		// is trivially squattable by another app on the same device.
		if strings.Contains(u.Scheme, ".") {
			return nil
		}
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			fmt.Sprintf("scheme %q is not allowed; use https, loopback http, or a private-use scheme containing a dot", u.Scheme))
	}
}

// clampGrantTypes limits a registration to the grants a dynamic client may
// hold. An empty request defaults to authorization_code, matching the admin
// path. A request naming a forbidden grant is rejected rather than trimmed:
// the client is asking for a capability, not expressing a preference, and
// silently handing back less would leave it broken in a confusing way.
// Duplicates are removed, first occurrence wins, so a repeated grant_type
// cannot inflate the stored slice.
func clampGrantTypes(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return []string{"authorization_code"}, nil
	}
	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, g := range requested {
		if _, ok := dynamicGrantTypes[g]; !ok {
			return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
				fmt.Sprintf("grant_type %q is not available to dynamically registered clients", g))
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	return out, nil
}

// clampScopes intersects requested scopes with the allowlist, dropping
// anything outside it. RFC 7591 section 2 lets the server substitute, and
// dropping beats erroring because clients tend to ask for a broad set
// optimistically; the response echoes what was actually granted. An empty
// request yields the full allowlist. Duplicates in the request are removed,
// first occurrence wins, so a repeated scope cannot inflate the stored
// slice.
func clampScopes(requested, allowlist []string) []string {
	if len(requested) == 0 {
		return append([]string(nil), allowlist...)
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, s := range allowlist {
		allowed[s] = struct{}{}
	}
	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if _, ok := allowed[s]; !ok {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
