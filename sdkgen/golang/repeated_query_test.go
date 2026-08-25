package golang_test

import (
	"go/format"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/golang"
	"github.com/xraph/authsome/sdkgen/openapi"
)

// repeatedQuerySpec describes an operation carrying an array-typed query
// parameter, which is how RFC 8707 resource indicators reach the SDKs.
//
// A query parameter defaults to style form with explode true, so an array
// means the name is sent once per value. Nothing in the document has to say
// so beyond the array itself.
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

func clientFile(t *testing.T, files []golang.GeneratedFile) string {
	t.Helper()

	for _, f := range files {
		if f.Path == "client.go" {
			return f.Content
		}
	}

	t.Fatal("no client.go in the generated files")

	return ""
}

// An array query parameter has to be sent once per value. url.Values.Set
// replaces, so the loop must use Add, and the value must go in whole rather
// than through fmt.Sprint, which would render a slice as "[a b]".
func TestGenerate_RepeatedQueryParamIsSentOncePerValue(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	content := clientFile(t, files)

	assert.Contains(t, content, `for _, v := range params.Resource {`)
	assert.Contains(t, content, `q.Add("resource", v)`)

	// Set would collapse the repeats back to one value.
	assert.NotContains(t, content, `q.Set("resource"`)
	assert.NotContains(t, content, `fmt.Sprint(params.Resource)`)
}

// The params struct has to hold more than one value for the loop to have
// anything to iterate.
func TestGenerate_RepeatedQueryParamIsASlice(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	content := clientFile(t, files)

	assert.Regexp(t, `Resource\s+\[\]string`, content)
}

// The scalar parameters on the same operation must keep working as they did.
func TestGenerate_ScalarQueryParamIsUntouchedByRepeatSupport(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	content := clientFile(t, files)

	assert.Contains(t, content, `q.Set("client_id", params.ClientID)`)
}

// gofmt only parses, and comparing a slice with != is a type error rather than
// a parse error, so the formatting check cannot see this one. Type-checking the
// render is what proves the emitted loop is real Go.
//
// The whole-SDK equivalent is `go build ./sdk/go/...` after regeneration; this
// keeps the same guarantee inside the unit tests, where it fails in a second
// instead of at the end of a generate run.
func TestGenerate_RepeatedQueryParamOutputParsesAndFormats(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})

	files, err := gen.Generate(repeatedQuerySpec())
	require.NoError(t, err)

	for _, f := range files {
		formatted, formatErr := format.Source([]byte(f.Content))
		require.NoErrorf(t, formatErr, "%s does not parse", f.Path)
		assert.Equalf(t, string(formatted), f.Content, "%s is not gofmt-clean", f.Path)
	}
}
