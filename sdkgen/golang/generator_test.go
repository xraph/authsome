package golang_test

import (
	"go/format"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/golang"
	"github.com/xraph/authsome/sdkgen/openapi"
)

func testSpec() *openapi.Spec {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		Title:          "Test API",
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa"},
	})
	return gen.Generate()
}

func TestNewGenerator_DefaultConfig(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)
	require.True(t, len(files) > 0)
}

func TestNewGenerator_CustomConfig(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{
		PackageName: "myauth",
		ModulePath:  "github.com/myorg/myauth",
	})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	for _, f := range files {
		assert.Contains(t, f.Content, "package myauth")
	}
}

func TestGenerate_ProducesAllFiles(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	assert.True(t, paths["types.go"], "should have types.go")
	assert.True(t, paths["client.go"], "should have client.go")
}

func TestGenerate_TypesFile(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "types.go" {
			typesContent = f.Content
			break
		}
	}
	require.NotEmpty(t, typesContent)

	// Should contain schema types
	assert.Contains(t, typesContent, "type User struct")
	assert.Contains(t, typesContent, "type Session struct")
	assert.Contains(t, typesContent, "type AuthResponse struct")
	assert.Contains(t, typesContent, "type TokenResponse struct")
	assert.Contains(t, typesContent, "type Error struct")

	// Should contain JSON tags
	assert.Contains(t, typesContent, "`json:\"email")
	assert.Contains(t, typesContent, "`json:\"name")

	// Should contain DO NOT EDIT header
	assert.Contains(t, typesContent, "DO NOT EDIT")

	// Should contain package name
	assert.Contains(t, typesContent, "package authclient")
}

func TestGenerate_ClientFile(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	// Should contain client struct
	assert.Contains(t, clientContent, "type Client struct")
	assert.Contains(t, clientContent, "func NewClient")

	// Should contain generated methods
	assert.Contains(t, clientContent, "func (c *Client) SignUp")
	assert.Contains(t, clientContent, "func (c *Client) SignIn")
	assert.Contains(t, clientContent, "func (c *Client) SignOut")
	assert.Contains(t, clientContent, "func (c *Client) RefreshSession")
	assert.Contains(t, clientContent, "func (c *Client) GetMe")
	assert.Contains(t, clientContent, "func (c *Client) UpdateMe")

	// Should contain plugin methods
	assert.Contains(t, clientContent, "func (c *Client) SocialStart")
	assert.Contains(t, clientContent, "func (c *Client) MagicLinkSend")
	assert.Contains(t, clientContent, "func (c *Client) MfaEnroll")

	// Should contain auth header
	assert.Contains(t, clientContent, "Authorization")
	assert.Contains(t, clientContent, "Bearer")

	// Should contain package name
	assert.Contains(t, clientContent, "package authclient")

	// Should contain DO NOT EDIT header
	assert.Contains(t, clientContent, "DO NOT EDIT")
}

func TestGenerate_RequestTypes(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "types.go" {
			typesContent = f.Content
			break
		}
	}
	require.NotEmpty(t, typesContent)

	// Should contain request types for operations with bodies
	assert.Contains(t, typesContent, "type SignUpRequest struct")
	assert.Contains(t, typesContent, "type SignInRequest struct")
	assert.Contains(t, typesContent, "type RefreshSessionRequest struct")
}

func TestGenerate_NoPlugins(t *testing.T) {
	spec := openapi.NewGenerator(openapi.GeneratorConfig{}).Generate()
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}

	// Core methods present
	assert.Contains(t, clientContent, "func (c *Client) SignUp")
	assert.Contains(t, clientContent, "func (c *Client) SignIn")

	// Plugin methods absent
	assert.NotContains(t, clientContent, "SocialStart")
	assert.NotContains(t, clientContent, "MagicLinkSend")
	assert.NotContains(t, clientContent, "MfaEnroll")
}

func TestGenerate_ClientError(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}

	assert.Contains(t, clientContent, "type ClientError struct")
	assert.Contains(t, clientContent, "StatusCode")
	assert.Contains(t, clientContent, "func (e *ClientError) Error()")
}

func TestGenerate_ClientOptions(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}

	assert.Contains(t, clientContent, "type Option func(*Client)")
	assert.Contains(t, clientContent, "func WithToken")
	assert.Contains(t, clientContent, "func WithHTTPClient")
	assert.Contains(t, clientContent, "func (c *Client) SetToken")
	assert.Contains(t, clientContent, "func (c *Client) Token()")
}

func TestGenerate_FilesNotEmpty(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEmpty(t, f.Content, "file %s should not be empty", f.Path)
		assert.True(t, len(f.Content) > 50, "file %s should have meaningful content", f.Path)
	}
}

func TestExportedName(t *testing.T) {
	// This tests the exported name conversion indirectly through generated types
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "types.go" {
			typesContent = f.Content
			break
		}
	}

	// "user_id" should become "UserID" (acronym handling)
	// "email" should become "Email"
	assert.Contains(t, typesContent, "Email")
	// "created_at" should become "CreatedAt"
	assert.Contains(t, typesContent, "CreatedAt")
}

func TestGenerate_ContextInMethods(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}

	// All methods should accept context.Context
	assert.Contains(t, clientContent, "ctx context.Context")
	assert.Contains(t, clientContent, `"context"`)
}

func TestGenerate_OutputIsGofmtClean(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	// The generated SDK is committed, so anything the templates emit that
	// gofmt would rewrite shows up as noise in every regeneration diff.
	// Formatting the render is what keeps `gofmt -l sdk/go` quiet.
	for _, f := range files {
		formatted, err := format.Source([]byte(f.Content))
		require.NoErrorf(t, err, "%s does not parse", f.Path)
		assert.Equalf(t, string(formatted), f.Content, "%s is not gofmt-clean", f.Path)
	}
}

// formBodySpec builds a spec whose only operation carries its request body as
// application/x-www-form-urlencoded, the encoding RFC 6749 mandates for the
// OAuth2 token endpoint. Forge describes those routes this way because the Go
// struct behind them is form-tagged, and nothing about that should stop the
// generated request struct from having the fields in it.
func formBodySpec() *openapi.Spec {
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

func TestGenerate_FormEncodedBodyKeepsItsFields(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(formBodySpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "types.go" {
			typesContent = f.Content
			break
		}
	}
	require.NotEmpty(t, typesContent)

	assert.Contains(t, typesContent, "type Oauth2TokenRequest struct")
	// gofmt aligns the field types, so match across the padding.
	assert.Regexp(t, `GrantType\s+string\s+`+"`"+`json:"grant_type"`+"`", typesContent)
	assert.Regexp(t, `Code\s+string\s+`+"`"+`json:"code,omitempty"`+"`", typesContent)
}

// RFC 6749 section 4.1.3 requires the token endpoint to take
// application/x-www-form-urlencoded. Posting JSON to it is what a conformant
// server is entitled to reject.
func TestGenerate_FormEncodedBodyIsPostedAsForm(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(formBodySpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	assert.Contains(t, clientContent, `form.Set("grant_type", req.GrantType)`)
	assert.Contains(t, clientContent, `form.Set("code", req.Code)`)
	assert.Contains(t, clientContent, `"application/x-www-form-urlencoded"`)
	assert.NotContains(t, clientContent, "json.Marshal(req)")
}

// The JSON routes are the overwhelming majority and must keep posting JSON.
func TestGenerate_JSONBodyStillPostedAsJSON(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "client.go" {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	assert.Contains(t, clientContent, "json.Marshal(req)")
}

// danglingRefSpec points a request body at a component that is not in the
// document. The schema resolves to nothing, and a body with no fields on it
// generates an empty struct that marshals "{}" over a body the server wants,
// which is the failure this whole area keeps producing quietly.
func danglingRefSpec() *openapi.Spec {
	return &openapi.Spec{
		OpenAPI: "3.0.3",
		Info:    openapi.Info{Title: "Test API", Version: "1"},
		Paths: map[string]*openapi.PathItem{
			"/v1/thing": {
				Post: &openapi.Operation{
					OperationID: "createThing",
					RequestBody: &openapi.RequestBody{
						Required: true,
						Content: map[string]openapi.MediaType{
							"application/json": {
								Schema: &openapi.Schema{Ref: "#/components/schemas/NoSuchThing"},
							},
						},
					},
					Responses: map[string]*openapi.Response{},
				},
			},
		},
		Components: &openapi.Components{Schemas: map[string]*openapi.Schema{}},
	}
}

func TestGenerate_DanglingRefIsAnError(t *testing.T) {
	gen := golang.NewGenerator(golang.GeneratorConfig{})

	_, err := gen.Generate(danglingRefSpec())

	require.Error(t, err, "a ref with no component behind it must not generate quietly")
	assert.Contains(t, err.Error(), "NoSuchThing")
}

// A ref that is not pointing into components/schemas is somebody else's to
// resolve, and passing it through is not the same mistake as swallowing a
// broken one.
func TestGenerate_NonComponentRefIsNotAnError(t *testing.T) {
	spec := danglingRefSpec()
	spec.Paths["/v1/thing"].Post.RequestBody.Content["application/json"] =
		openapi.MediaType{Schema: &openapi.Schema{Ref: "https://example.com/schema.json#/Thing"}}

	gen := golang.NewGenerator(golang.GeneratorConfig{})

	_, err := gen.Generate(spec)

	require.NoError(t, err)
}
