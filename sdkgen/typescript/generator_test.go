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

// formBodySpec mirrors how forge describes the OAuth2 token endpoint: the body
// is application/x-www-form-urlencoded, per RFC 6749, rather than JSON.
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
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(formBodySpec())
	require.NoError(t, err)

	var typesContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "types.ts") {
			typesContent = f.Content
			break
		}
	}
	require.NotEmpty(t, typesContent)

	assert.Contains(t, typesContent, "grant_type: string")
	assert.Contains(t, typesContent, "code?: string")
	assert.NotContains(t, typesContent, "Oauth2TokenRequest = Record<string, unknown>")
}

// RFC 6749 section 4.1.3 requires application/x-www-form-urlencoded at the
// token endpoint, so the generated client has to encode those bodies as a form
// rather than posting JSON at them.
func TestGenerate_FormEncodedBodyIsPostedAsForm(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(formBodySpec())
	require.NoError(t, err)

	var clientContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "client.ts") {
			clientContent = f.Content
			break
		}
	}
	require.NotEmpty(t, clientContent)

	// The helper and the header live in the request plumbing and are emitted
	// whether or not anything uses them, so asserting on those alone would pass
	// with the encoder switched off. What proves the wiring is the call site
	// electing to use it.
	assert.Contains(t, clientContent, "      body,\n      true,\n    );")
	assert.Contains(t, clientContent, "application/x-www-form-urlencoded")
	assert.Contains(t, clientContent, "encodeForm")
}

func TestGenerate_DPoPFile(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var dpopContent string
	for _, f := range files {
		if f.Path == "src/dpop.ts" {
			dpopContent = f.Content
			break
		}
	}
	require.NotEmpty(t, dpopContent, "standalone mode should produce src/dpop.ts")

	// The non-extractable generateKey call is the entire security proposition:
	// signing stays possible, reading the private key never does.
	assert.Contains(t, dpopContent, `generateKey(ALG, false, ["sign", "verify"])`)
	assert.Contains(t, dpopContent, "export interface DPoPKeyStore")
	assert.Contains(t, dpopContent, "export class DPoPSession")
	assert.Contains(t, dpopContent, "export class IndexedDBKeyStore")
	assert.Contains(t, dpopContent, "indexedDB.open")
}

func TestGenerate_ClientWiresDPoP(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	var clientContent, indexContent string
	for _, f := range files {
		switch f.Path {
		case "src/client.ts":
			clientContent = f.Content
		case "src/index.ts":
			indexContent = f.Content
		}
	}
	require.NotEmpty(t, clientContent)
	require.NotEmpty(t, indexContent)

	assert.Contains(t, clientContent, "import { DPoPSession } from './dpop'")
	assert.Contains(t, clientContent, "enableDPoP")
	assert.Contains(t, clientContent, "`DPoP ${token}`")
	assert.Contains(t, clientContent, "headers['DPoP']")
	assert.Contains(t, clientContent, "DPoP-Nonce")
	// Threaded retry flag so the one-shot retry cannot recurse into a loop.
	assert.Contains(t, clientContent, "isRetry")

	// The proof must be minted whenever DPoP is enabled, not only when a
	// token exists -- sign-in/sign-up/token-exchange have no token yet but
	// still need a proof to bind the token about to be issued. Pin the
	// structural marker: `if (this.dpop)` guarding the proof mint, separate
	// from and preceding the `if (token)` guarding the Authorization header.
	dpopGuardIdx := strings.Index(clientContent, "if (this.dpop) {")
	tokenGuardIdx := strings.Index(clientContent, "if (token) {")
	require.NotEqual(t, -1, dpopGuardIdx, "proof-minting guard should exist")
	require.NotEqual(t, -1, tokenGuardIdx, "Authorization guard should exist")
	assert.Less(t, dpopGuardIdx, tokenGuardIdx, "the DPoP proof must be minted independently of, and before, the token-gated Authorization header")

	// RFC 9449 section 8: the nonce challenge can be a 401 (resource server)
	// or a 400 (token endpoint / sign-in and similar binding calls). Both
	// shapes must be checked, and the body must be cloned so the later
	// error-handling read still works.
	assert.Contains(t, clientContent, "WWW-Authenticate")
	assert.Contains(t, clientContent, "use_dpop_nonce")
	assert.Contains(t, clientContent, "response.status === 400")
	assert.Contains(t, clientContent, ".clone()")

	assert.Contains(t, indexContent, "export { DPoPSession, IndexedDBKeyStore } from './dpop'")
}

func TestGenerate_EmbeddedModeHasNoDPoP(t *testing.T) {
	gen := typescript.NewGenerator(typescript.GeneratorConfig{OutputMode: "embedded"})
	files, err := gen.Generate(testSpec())
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "src/dpop.ts", f.Path, "embedded mode should not emit dpop.ts")
		assert.NotContains(t, f.Content, "./dpop", "embedded output should not reference the standalone-only dpop module")
	}
}
