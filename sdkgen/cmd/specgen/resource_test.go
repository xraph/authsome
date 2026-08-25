package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateSpec runs the real generator and returns the document it wrote.
func generateSpec(t *testing.T) map[string]any {
	t.Helper()

	out := filepath.Join(t.TempDir(), "spec.json")
	require.NoError(t, run(out, "AuthSome API", "0.5.0"))

	data, err := os.ReadFile(out) // #nosec G304 -- path is this test's own temp dir.
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(data, &spec))

	return spec
}

// operationByID walks the document for the operation carrying the given ID.
func operationByID(t *testing.T, spec map[string]any, id string) map[string]any {
	t.Helper()

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok, "document should have paths")

	for _, pathItem := range paths {
		methods, isObject := pathItem.(map[string]any)
		if !isObject {
			continue
		}

		for _, opAny := range methods {
			op, isOp := opAny.(map[string]any)
			if isOp && op["operationId"] == id {
				return op
			}
		}
	}

	t.Fatalf("no operation with operationId %q in the generated document", id)

	return nil
}

// findParam returns the named parameter from an operation, or nil.
func findParam(op map[string]any, name string) map[string]any {
	params, _ := op["parameters"].([]any)

	for _, p := range params {
		param, ok := p.(map[string]any)
		if ok && param["name"] == name {
			return param
		}
	}

	return nil
}

// formBodyProperty returns the named property of an operation's urlencoded
// request body, or nil.
func formBodyProperty(t *testing.T, op map[string]any, name string) map[string]any {
	t.Helper()

	body, ok := op["requestBody"].(map[string]any)
	require.True(t, ok, "operation should have a request body")

	content, ok := body["content"].(map[string]any)
	require.True(t, ok, "request body should have content")

	media, ok := content["application/x-www-form-urlencoded"].(map[string]any)
	require.True(t, ok, "OAuth2 bodies are urlencoded per RFC 6749")

	schema, ok := media["schema"].(map[string]any)
	require.True(t, ok, "media type should carry a schema")

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "body schema should have properties")

	prop, _ := props[name].(map[string]any)

	return prop
}

// assertRepeatableString checks a schema describes a repeatable string value.
//
// An array is the whole of it. Query parameters and form fields both default to
// style form with explode true, so an array means the name is sent once per
// value, which is what RFC 8707 asks for and what the struct binder reads back
// on the way in.
func assertRepeatableString(t *testing.T, schema map[string]any) {
	t.Helper()

	require.NotNil(t, schema, "resource should carry a schema")
	assert.Equal(t, "array", schema["type"], "resource is repeatable, so it is an array")

	items, isObject := schema["items"].(map[string]any)
	require.True(t, isObject, "an array schema needs items")
	assert.Equal(t, "string", items["type"])
}

// The authorization endpoint is a GET and has no body, so the query string is
// the only place a resource indicator can go.
func TestGeneratedSpec_AuthorizeTakesResourceAsAQueryParameter(t *testing.T) {
	op := operationByID(t, generateSpec(t), "oauth2Authorize")

	param := findParam(op, "resource")
	require.NotNil(t, param, "oauth2Authorize should declare the resource parameter")

	assert.Equal(t, "query", param["in"])
	assert.NotEqual(t, true, param["required"], "resource is optional")

	schema, _ := param["schema"].(map[string]any)
	assertRepeatableString(t, schema)
}

// The two POST endpoints take their parameters in the urlencoded body per RFC
// 6749, so the resource indicator is a body field on both.
//
// The server also reads the query string on these routes, but documenting both
// placements would buy nothing: a generated client only needs one way to send
// it, and adding a query parameter to an operation that already has a body is
// what changes its method signature and breaks every existing caller.
func TestGeneratedSpec_PostEndpointsTakeResourceInTheBody(t *testing.T) {
	spec := generateSpec(t)

	for _, id := range []string{"oauth2Token", "oauth2DeviceAuthorize"} {
		t.Run(id, func(t *testing.T) {
			op := operationByID(t, spec, id)

			assertRepeatableString(t, formBodyProperty(t, op, "resource"))

			assert.Nil(t, findParam(op, "resource"),
				"the body already carries it; a query parameter here only breaks the signature")
		})
	}
}
