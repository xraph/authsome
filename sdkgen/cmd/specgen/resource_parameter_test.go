package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// committedSpec reads the spec.json that ships with the repository. CI
// regenerates it and fails on any diff, so asserting against the committed
// copy is equivalent to asserting against a fresh dump, without paying for one
// in a unit test.
func committedSpec(t *testing.T) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "spec.json"))
	require.NoError(t, err, "spec.json should be committed alongside the generator")

	var spec map[string]any
	require.NoError(t, json.Unmarshal(raw, &spec))

	return spec
}

// queryParameter finds a query parameter by name on one operation.
func queryParameter(t *testing.T, spec map[string]any, path, method, name string) map[string]any {
	t.Helper()

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok, "spec has no paths")

	item, ok := paths[path].(map[string]any)
	require.True(t, ok, "spec has no path %s", path)

	op, ok := item[method].(map[string]any)
	require.True(t, ok, "spec has no %s on %s", method, path)

	params, _ := op["parameters"].([]any)
	for _, raw := range params {
		param, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if param["name"] == name && param["in"] == "query" {
			return param
		}
	}

	return nil
}

// The RFC 8707 resource indicator is repeatable, and the authorization
// endpoint honours every value it is given. It only reaches a generated client
// if the spec describes it, and the spec only describes parameters that exist
// as fields on the handler's request struct.
//
// The field cannot exist yet. go-utils v1.1.7 taught bindFormParam to fill a
// []string from a repeated parameter, which is why the device endpoint below
// carries one, but bindQueryParam still reads a single value through c.Query.
// A []string query field would therefore bind the first resource and silently
// drop the rest, which is worse than the error the old binder raised, so
// handleAuthorize reads the raw query instead and the parameter stays
// undescribed.
//
// Unskip this once bindQueryParam handles repeated values: add
// `Resource []string` with a query tag to AuthorizeRequest, drop
// resourceParams, and regenerate.
func TestSpec_AuthorizeExposesRepeatableResource(t *testing.T) {
	t.Skip("blocked on go-utils bindQueryParam, which collapses a repeated query parameter to its first value")

	param := queryParameter(t, committedSpec(t), "/v1/oauth/authorize", "get", "resource")

	require.NotNil(t, param, "the authorize endpoint should describe a resource query parameter")

	schema, ok := param["schema"].(map[string]any)
	require.True(t, ok, "resource parameter has no schema")

	assert.Equal(t, "array", schema["type"],
		"resource must be an array so a client can send more than one")

	items, ok := schema["items"].(map[string]any)
	require.True(t, ok, "an array parameter needs an items schema")
	assert.Equal(t, "string", items["type"])

	assert.NotEqual(t, true, param["required"], "resource is optional")
}

// The device authorization endpoint takes the same indicator, in a form body
// rather than a query string.
func TestSpec_DeviceAuthorizeExposesResource(t *testing.T) {
	paths, ok := committedSpec(t)["paths"].(map[string]any)
	require.True(t, ok)

	item, ok := paths["/v1/oauth/device/authorize"].(map[string]any)
	require.True(t, ok, "spec has no device authorize path")

	op, ok := item["post"].(map[string]any)
	require.True(t, ok)

	body, ok := op["requestBody"].(map[string]any)
	require.True(t, ok, "device authorize should carry a request body")

	content, ok := body["content"].(map[string]any)
	require.True(t, ok)

	form, ok := content["application/x-www-form-urlencoded"].(map[string]any)
	require.True(t, ok, "device authorize is form-encoded per RFC 8628")

	schema, ok := form["schema"].(map[string]any)
	require.True(t, ok)

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	resource, ok := props["resource"].(map[string]any)
	require.True(t, ok, "device authorize should describe a resource field")
	assert.Equal(t, "array", resource["type"])
}
