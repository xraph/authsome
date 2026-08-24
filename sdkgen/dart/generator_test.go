package dart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/dart"
	"github.com/xraph/authsome/sdkgen/openapi"
)

// formRefBodySpec describes a request body the way forge describes the OAuth2
// routes, as application/x-www-form-urlencoded rather than JSON, and points it
// at a component so the generator has a name it can use.
func formRefBodySpec() *openapi.Spec {
	return &openapi.Spec{
		OpenAPI: "3.0.3",
		Info:    openapi.Info{Title: "Test API", Version: "1"},
		Paths: map[string]*openapi.PathItem{
			"/v1/oauth/token": {
				Post: &openapi.Operation{
					OperationID: "oauth2Token",
					Summary:     "Token",
					RequestBody: &openapi.RequestBody{
						Required: true,
						Content: map[string]openapi.MediaType{
							"application/x-www-form-urlencoded": {
								Schema: &openapi.Schema{Ref: "#/components/schemas/TokenRequest"},
							},
						},
					},
					Responses: map[string]*openapi.Response{},
				},
			},
		},
		Components: &openapi.Components{
			Schemas: map[string]*openapi.Schema{
				"TokenRequest": {
					Type:     "object",
					Required: []string{"grant_type"},
					Properties: map[string]*openapi.Schema{
						"grant_type": {Type: "string"},
					},
				},
			},
		},
	}
}

// A form-encoded body is still a body. Typing it as an untyped map because the
// content type is not JSON throws away a name the spec handed us.
func TestGenerate_FormEncodedBodyKeepsItsType(t *testing.T) {
	gen := dart.NewGenerator(dart.GeneratorConfig{})
	files, err := gen.Generate(formRefBodySpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.dart") {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	assert.Contains(t, clientContent, "required TokenRequest body")
	assert.NotContains(t, clientContent, "required Map<String, dynamic> body")
}
