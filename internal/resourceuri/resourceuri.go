// Package resourceuri holds the RFC 8707 section 2 syntax rule for a resource
// indicator, in one place both the OAuth2 provider and the core settings can
// reach.
//
// The provider validates what a client sends; the core validates what an
// operator configures as session.resource_identifier. Those two values are
// compared against each other at request time, so a rule enforced on one side
// and not the other is worse than no rule: the deployment looks configured and
// matches nothing. The provider plugin already imports the root package, so
// the root cannot import the provider back, which is why the rule lives here
// rather than in either of them.
package resourceuri

import (
	"fmt"
	"net/url"
	"strings"
)

// SyntaxError checks one resource indicator and returns a description of what
// is wrong with it, or an empty string when it is valid.
//
// A string return rather than an error because the callers wrap it in their
// own error shapes: an OAuth2 invalid_target response on one side, a settings
// write rejection on the other.
func SyntaxError(raw string) string {
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
