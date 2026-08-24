package oauth2provider

import (
	"net/http"
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
