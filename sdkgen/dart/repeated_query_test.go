package dart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/dart"
	"github.com/xraph/authsome/sdkgen/openapi"
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

func repeatedDartClient(t *testing.T) string {
	t.Helper()

	gen := dart.NewGenerator(dart.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.dart") {
			return f.Content
		}
	}

	t.Fatal("no client.dart in the generated files")

	return ""
}

// A Map<String, String> cannot hold the same key twice, so a repeatable
// parameter needs a structure that can. Every parameter goes through the same
// list of pairs rather than only the repeatable ones, so there is one way the
// query string gets built instead of two that have to agree.
func TestGenerate_QueryParamsAreBuiltFromPairs(t *testing.T) {
	content := repeatedDartClient(t)

	assert.Contains(t, content, "final queryPairs = <MapEntry<String, String>>[];")
	assert.NotContains(t, content, "final queryParams = <String, String>{};")
}

// The repeatable parameter contributes one pair per value.
//
// The loop is braced rather than written on one line because the Flutter
// analyzer runs curly_braces_in_flow_control_structures over the embedded copy
// of this client, and a one-line for trips it.
func TestGenerate_RepeatedQueryParamAddsOnePairPerValue(t *testing.T) {
	content := repeatedDartClient(t)

	assert.Contains(t, content, strings.Join([]string{
		"    for (final element in resource ?? const []) {",
		"      queryPairs.add(MapEntry('resource', element.toString()));",
		"    }",
	}, "\n"))
}

// The scalar parameters on the same operation must still contribute exactly
// one pair each, required ones unconditionally and optional ones when set.
func TestGenerate_ScalarQueryParamsStillContributeOnePair(t *testing.T) {
	content := repeatedDartClient(t)

	assert.Contains(t, content,
		"queryPairs.add(MapEntry('client_id', clientId.toString()));")
}

// The query string is assembled off the pairs, so repeats survive to the wire.
func TestGenerate_QueryStringIsAssembledFromPairs(t *testing.T) {
	content := repeatedDartClient(t)

	assert.Contains(t, content, "queryPairs.isNotEmpty")
	assert.Contains(t, content,
		"queryPairs.map((e) => '${e.key}=${Uri.encodeComponent(e.value)}').join('&')")
	// The old form read .entries off a map; a list has no such member.
	assert.NotContains(t, content, "queryPairs.entries.map")
}
