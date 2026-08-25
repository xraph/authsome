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

// inlineFormBodySpec describes the body inline, with the properties sitting
// right on the schema, which is how forge writes the four OAuth2 routes.
func inlineFormBodySpec() *openapi.Spec {
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
								Schema: &openapi.Schema{
									Type:     "object",
									Required: []string{"grant_type"},
									Properties: map[string]*openapi.Schema{
										"grant_type": {Type: "string"},
										"code":       {Type: "string"},
									},
								},
							},
						},
					},
					Responses: map[string]*openapi.Response{},
				},
			},
		},
	}
}

func dartFile(t *testing.T, files []dart.GeneratedFile, suffix string) string {
	t.Helper()
	for _, f := range files {
		if strings.HasSuffix(f.Path, suffix) {
			return f.Content
		}
	}
	t.Fatalf("no generated file ending in %q", suffix)
	return ""
}

// An inline body has every field name the endpoint takes. Handing the caller a
// bare map throws all of them away.
func TestGenerate_InlineBodyGetsARequestClass(t *testing.T) {
	gen := dart.NewGenerator(dart.GeneratorConfig{})
	files, err := gen.Generate(inlineFormBodySpec())
	require.NoError(t, err)

	types := dartFile(t, files, "types.dart")
	assert.Contains(t, types, "class Oauth2TokenRequest {")
	assert.Contains(t, types, "final String grantType;")
	assert.Contains(t, types, "final String? code;")
}

// The generated method has to take the class, otherwise the class is dead code
// and the caller still passes a map.
func TestGenerate_InlineBodyMethodTakesTheRequestClass(t *testing.T) {
	gen := dart.NewGenerator(dart.GeneratorConfig{})
	files, err := gen.Generate(inlineFormBodySpec())
	require.NoError(t, err)

	client := dartFile(t, files, "client.dart")
	assert.Contains(t, client, "required Oauth2TokenRequest body")
	assert.Contains(t, client, "body: body.toJson()")
}

// RFC 6749 section 4.1.3 requires application/x-www-form-urlencoded at the
// token endpoint, so the generated client has to encode those bodies as a form
// rather than posting JSON at them.
func TestGenerate_FormEncodedBodyIsPostedAsForm(t *testing.T) {
	gen := dart.NewGenerator(dart.GeneratorConfig{})
	files, err := gen.Generate(inlineFormBodySpec())
	require.NoError(t, err)

	client := dartFile(t, files, "client.dart")
	assert.Contains(t, client, "application/x-www-form-urlencoded")
	assert.Contains(t, client, "form: true")
}
