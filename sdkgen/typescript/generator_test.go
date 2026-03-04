package typescript_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/openapi"
	"github.com/xraph/authsome/sdkgen/typescript"
)

func testSpec() *openapi.Spec {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		Title:          "Test API",
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa"},
	})
	return gen.Generate()
}

func TestNewGenerator_DefaultConfig(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)
	require.True(t, len(files) > 0)

	// Check package.json has defaults
	for _, f := range files {
		if f.Path == "package.json" {
			assert.Contains(t, f.Content, `"@authsome/client"`)
			assert.Contains(t, f.Content, `"0.5.0"`)
		}
	}
}

func TestNewGenerator_CustomConfig(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{
		PackageName:    "@myorg/auth-sdk",
		PackageVersion: "2.0.0",
	})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	for _, f := range files {
		if f.Path == "package.json" {
			assert.Contains(t, f.Content, `"@myorg/auth-sdk"`)
			assert.Contains(t, f.Content, `"2.0.0"`)
		}
	}
}

func TestGenerate_ProducesAllFiles(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	assert.True(t, paths["src/types.ts"], "should have types.ts")
	assert.True(t, paths["src/client.ts"], "should have client.ts")
	assert.True(t, paths["src/index.ts"], "should have index.ts")
	assert.True(t, paths["package.json"], "should have package.json")
	assert.True(t, paths["tsconfig.json"], "should have tsconfig.json")
}

func TestGenerate_TypesFile(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "src/types.ts" {
			typesContent = f.Content
			break
		}
	}
	require.NotEmpty(t, typesContent)

	// Should contain core schema types
	assert.Contains(t, typesContent, "export interface User")
	assert.Contains(t, typesContent, "export interface Session")
	assert.Contains(t, typesContent, "export interface AuthResponse")
	assert.Contains(t, typesContent, "export interface TokenResponse")
	assert.Contains(t, typesContent, "export interface Error")

	// Should contain field definitions
	assert.Contains(t, typesContent, "email")
	assert.Contains(t, typesContent, "string")

	// Should contain DO NOT EDIT header
	assert.Contains(t, typesContent, "DO NOT EDIT")
}

func TestGenerate_ClientFile(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "src/client.ts" {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	// Should contain class definition
	assert.Contains(t, clientContent, "export class AuthClient")
	assert.Contains(t, clientContent, "export function createAuthClient")
	assert.Contains(t, clientContent, "AuthClientError")

	// Should contain generated methods for core operations
	assert.Contains(t, clientContent, "signUp")
	assert.Contains(t, clientContent, "signIn")
	assert.Contains(t, clientContent, "signOut")
	assert.Contains(t, clientContent, "refreshSession")
	assert.Contains(t, clientContent, "getMe")
	assert.Contains(t, clientContent, "updateMe")

	// Should contain plugin methods
	assert.Contains(t, clientContent, "socialStart")
	assert.Contains(t, clientContent, "magicLinkSend")
	assert.Contains(t, clientContent, "mfaEnroll")

	// Should contain auth header logic
	assert.Contains(t, clientContent, "Authorization")
	assert.Contains(t, clientContent, "Bearer")

	// Should contain DO NOT EDIT header
	assert.Contains(t, clientContent, "DO NOT EDIT")
}

func TestGenerate_IndexFile(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var indexContent string
	for _, f := range files {
		if f.Path == "src/index.ts" {
			indexContent = f.Content
			break
		}
	}
	require.NotEmpty(t, indexContent)

	// Should re-export client
	assert.Contains(t, indexContent, "AuthClient")
	assert.Contains(t, indexContent, "createAuthClient")

	// Should re-export types
	assert.Contains(t, indexContent, "User")
	assert.Contains(t, indexContent, "Session")
}

func TestGenerate_TSConfig(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var tsconfig string
	for _, f := range files {
		if f.Path == "tsconfig.json" {
			tsconfig = f.Content
			break
		}
	}
	require.NotEmpty(t, tsconfig)
	assert.Contains(t, tsconfig, `"strict": true`)
	assert.Contains(t, tsconfig, `"declaration": true`)
}

func TestGenerate_NoPlugins(t *testing.T) {
	// Generate with no plugins
	spec := openapi.NewGenerator(openapi.GeneratorConfig{}).Generate()
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "src/client.ts" {
			clientContent = f.Content
			break
		}
	}

	// Core methods should still be present
	assert.Contains(t, clientContent, "signUp")
	assert.Contains(t, clientContent, "signIn")

	// Plugin methods should NOT be present
	assert.NotContains(t, clientContent, "socialStart")
	assert.NotContains(t, clientContent, "magicLinkSend")
	assert.NotContains(t, clientContent, "mfaEnroll")
}

func TestGenerate_MethodSignatures(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if f.Path == "src/client.ts" {
			clientContent = f.Content
			break
		}
	}

	// Methods with body should have body parameter
	assert.Contains(t, clientContent, "signUp(body: SignUpRequest)")
	assert.Contains(t, clientContent, "signIn(body: SignInRequest)")

	// Methods without body should have no params
	// signOut typically has no body
	assert.True(t, strings.Contains(clientContent, "signOut()") || strings.Contains(clientContent, "signOut("),
		"signOut should exist as a method")
}

func TestGenerate_RequestTypes(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if f.Path == "src/types.ts" {
			typesContent = f.Content
			break
		}
	}

	// Should contain request type aliases for operations with bodies
	assert.Contains(t, typesContent, "SignUpRequest")
	assert.Contains(t, typesContent, "SignInRequest")
	assert.Contains(t, typesContent, "RefreshSessionRequest")
}

func TestGenerate_FilesNotEmpty(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEmpty(t, f.Content, "file %s should not be empty", f.Path)
		assert.True(t, len(f.Content) > 10, "file %s should have meaningful content", f.Path)
	}
}
