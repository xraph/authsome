package typescript_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/openapi"
	"github.com/xraph/authsome/sdkgen/typescript"
)

// repeatedQuerySpec describes an operation carrying an array-typed query
// parameter, which is how RFC 8707 resource indicators reach the SDKs.
func repeatedQuerySpec() *openapi.Spec {
	return &openapi.Spec{
		OpenAPI: "3.0.3",
		Info:    openapi.Info{Title: "Test API", Version: "1"},
		Paths: map[string]*openapi.PathItem{
			"/v1/oauth/authorize": {
				Get: &openapi.Operation{
					OperationID: "oauth2Authorize",
					Summary:     "Authorize",
					Parameters: []openapi.Parameter{
						{
							Name:     "client_id",
							In:       "query",
							Required: true,
							Schema:   &openapi.Schema{Type: "string"},
						},
						{
							Name:        "resource",
							In:          "query",
							Description: "RFC 8707 resource indicator. Repeatable.",
							Schema: &openapi.Schema{
								Type:  "array",
								Items: &openapi.Schema{Type: "string"},
							},
						},
					},
					Responses: map[string]*openapi.Response{},
				},
			},
		},
	}
}

func repeatedClientSource(t *testing.T) string {
	t.Helper()

	gen := typescript.NewGenerator(typescript.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	for _, f := range files {
		if f.Path == "src/client.ts" {
			return f.Content
		}
	}

	t.Fatal("no src/client.ts in the generated files")

	return ""
}

// URLSearchParams.set replaces the previous value, so a repeatable parameter
// has to go through append once per element. String() on the array itself is
// the trap: JavaScript renders ["a","b"] as "a,b", which reaches the server as
// one resource whose value contains a comma.
func TestGenerate_RepeatedQueryParamIsAppendedPerValue(t *testing.T) {
	content := repeatedClientSource(t)

	assert.Contains(t, content, `for (const v of resource) params.append('resource', String(v));`)
	assert.NotContains(t, content, `params.set('resource'`)
}

// The signature has to accept more than one value for the loop to have
// anything to iterate.
func TestGenerate_RepeatedQueryParamIsTypedAsAnArray(t *testing.T) {
	content := repeatedClientSource(t)

	assert.Contains(t, content, "resource?: string[]")
}

// The scalar parameters on the same operation must keep working as they did.
func TestGenerate_ScalarQueryParamKeepsUsingSet(t *testing.T) {
	content := repeatedClientSource(t)

	assert.Contains(t, content, `params.set('client_id', String(client_id));`)
}
