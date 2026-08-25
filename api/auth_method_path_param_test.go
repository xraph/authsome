package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/api"
)

// TestUnlinkAuthMethod_BindsProviderFromPath drives DELETE
// /v1/me/auth-methods/:provider through the full Forge pipeline.
//
// Forge binds path segments from the `path` struct tag. When
// UnlinkAuthMethodRequest.Provider carried `param:"provider"` instead, the
// field matched no binding source, stayed empty, and Forge's own required-field
// validation rejected the request with `invalid request: Provider: field is
// required` before the handler ever ran. Every call to this route 400'd,
// whatever provider the caller named.
func TestUnlinkAuthMethod_BindsProviderFromPath(t *testing.T) {
	_, eng := newTestAPI(t)
	handler := newAPIWithRouter(t, eng)

	_, token, _ := signUp(t, eng, "unlink-path-bind@test.com", "SecureP@ss1")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/me/auth-methods/password", nil)
	req = asUser(req, userIDFor(t, eng, token))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	assert.NotContains(t, body, "field is required",
		"Forge rejected the request during binding; the provider path segment never reached the handler. body=%s", body)

	// The bare test engine registers no AuthMethodContributor plugins, so the
	// user has no linked methods and the handler's own last-method guard is the
	// expected outcome. Its presence proves binding succeeded and the handler ran.
	assert.Contains(t, body, "cannot unlink last authentication method",
		"expected the handler's last-method guard, meaning the request bound and reached it. body=%s", body)
}

// TestUnlinkAuthMethod_OpenAPIDeclaresNoRequestBody covers the other half of
// the same defect, the half a passing handler would still hide.
//
// Forge infers the `{provider}` path parameter from the route template, so the
// parameter shows up in the document either way. What the tag decides is where
// the struct field lands. Unrecognised, it falls through to the request body:
// the DELETE operation grows a required application/json body of
// {"Provider": string} that nothing on the server ever reads, and the
// generated SDKs make callers construct it.
func TestUnlinkAuthMethod_OpenAPIDeclaresNoRequestBody(t *testing.T) {
	_, eng := newTestAPI(t)

	router := forge.NewRouter(forge.WithOpenAPI(forge.OpenAPIConfig{
		Title:   "AuthSome",
		Version: "test",
	}))
	a := api.New(eng)
	require.NoError(t, a.RegisterRoutes(router))
	require.NoError(t, router.Start(context.Background()))

	spec := router.OpenAPISpec()
	require.NotNil(t, spec, "router must expose an OpenAPI spec")

	raw, err := json.Marshal(spec)
	require.NoError(t, err)

	var doc struct {
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name string `json:"name"`
				In   string `json:"in"`
			} `json:"parameters"`
			RequestBody json.RawMessage `json:"requestBody"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))

	const route = "/v1/me/auth-methods/{provider}"
	op, ok := doc.Paths[route]["delete"]
	require.True(t, ok, "spec must document DELETE %s", route)

	var pathParams []string
	for _, p := range op.Parameters {
		if p.In == "path" {
			pathParams = append(pathParams, p.Name)
		}
	}
	assert.Equal(t, []string{"provider"}, pathParams,
		"DELETE %s takes exactly one path parameter", route)

	assert.Empty(t, string(op.RequestBody),
		"DELETE %s carries no body; the provider travels in the path. requestBody=%s", route, op.RequestBody)
}
