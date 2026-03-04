package golang_test

import (
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
