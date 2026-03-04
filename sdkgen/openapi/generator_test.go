package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/openapi"
)

func TestGenerator_DefaultConfig(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	assert.Equal(t, "3.1.0", spec.OpenAPI)
	assert.Equal(t, "AuthSome API", spec.Info.Title)
	assert.Equal(t, "0.5.0", spec.Info.Version)
}

func TestGenerator_CustomConfig(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		Title:       "My Auth API",
		Description: "Custom auth service",
		Version:     "1.0.0",
		ServerURL:   "https://api.example.com",
	})
	spec := gen.Generate()

	assert.Equal(t, "My Auth API", spec.Info.Title)
	assert.Equal(t, "Custom auth service", spec.Info.Description)
	assert.Equal(t, "1.0.0", spec.Info.Version)
	require.Len(t, spec.Servers, 1)
	assert.Equal(t, "https://api.example.com", spec.Servers[0].URL)
}

func TestGenerator_CoreEndpoints(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	// Core auth endpoints
	assert.NotNil(t, spec.Paths["/v1/auth/signup"])
	assert.NotNil(t, spec.Paths["/v1/auth/signup"].Post)

	assert.NotNil(t, spec.Paths["/v1/auth/signin"])
	assert.NotNil(t, spec.Paths["/v1/auth/signin"].Post)

	assert.NotNil(t, spec.Paths["/v1/auth/signout"])
	assert.NotNil(t, spec.Paths["/v1/auth/signout"].Post)

	assert.NotNil(t, spec.Paths["/v1/auth/refresh"])
	assert.NotNil(t, spec.Paths["/v1/auth/refresh"].Post)

	assert.NotNil(t, spec.Paths["/v1/auth/health"])
	assert.NotNil(t, spec.Paths["/v1/auth/health"].Get)

	// User endpoints
	assert.NotNil(t, spec.Paths["/v1/auth/me"])
	assert.NotNil(t, spec.Paths["/v1/auth/me"].Get)
	assert.NotNil(t, spec.Paths["/v1/auth/me"].Patch)

	// Session endpoints
	assert.NotNil(t, spec.Paths["/v1/auth/sessions"])
	assert.NotNil(t, spec.Paths["/v1/auth/sessions/{id}"])

	// Well-known endpoints
	assert.NotNil(t, spec.Paths["/.well-known/authsome/manifest"])
	assert.NotNil(t, spec.Paths["/.well-known/authsome/openapi"])
}

func TestGenerator_PasswordEndpoints(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/forgot-password"])
	assert.NotNil(t, spec.Paths["/v1/auth/forgot-password"].Post)
	assert.Equal(t, "forgotPassword", spec.Paths["/v1/auth/forgot-password"].Post.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/reset-password"])
	assert.NotNil(t, spec.Paths["/v1/auth/reset-password"].Post)
	assert.Equal(t, "resetPassword", spec.Paths["/v1/auth/reset-password"].Post.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/change-password"])
	assert.NotNil(t, spec.Paths["/v1/auth/change-password"].Post)
	assert.Equal(t, "changePassword", spec.Paths["/v1/auth/change-password"].Post.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/verify-email"])
	assert.NotNil(t, spec.Paths["/v1/auth/verify-email"].Post)
	assert.Equal(t, "verifyEmail", spec.Paths["/v1/auth/verify-email"].Post.OperationID)

	// Change password requires auth
	changePass := spec.Paths["/v1/auth/change-password"].Post
	require.NotEmpty(t, changePass.Security)
	_, hasBearer := changePass.Security[0]["bearerAuth"]
	assert.True(t, hasBearer)
}

func TestGenerator_OrgEndpoints(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization"},
	})
	spec := gen.Generate()

	// Org CRUD
	assert.NotNil(t, spec.Paths["/v1/auth/orgs"])
	assert.NotNil(t, spec.Paths["/v1/auth/orgs"].Post)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs"].Get)
	assert.Equal(t, "createOrg", spec.Paths["/v1/auth/orgs"].Post.OperationID)
	assert.Equal(t, "listOrgs", spec.Paths["/v1/auth/orgs"].Get.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}"])
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}"].Get)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}"].Patch)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}"].Delete)

	// Members
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/members"])
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/members"].Get)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/members"].Post)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/members/{memberId}"])
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/members/{memberId}"].Delete)

	// Invitations
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/invitations"])
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/invitations"].Get)
	assert.NotNil(t, spec.Paths["/v1/auth/orgs/{orgId}/invitations"].Post)
}

func TestGenerator_DeviceEndpoints(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/devices"])
	assert.NotNil(t, spec.Paths["/v1/auth/devices"].Get)
	assert.Equal(t, "listDevices", spec.Paths["/v1/auth/devices"].Get.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/devices/{deviceId}"])
	assert.NotNil(t, spec.Paths["/v1/auth/devices/{deviceId}"].Delete)
	assert.Equal(t, "deleteDevice", spec.Paths["/v1/auth/devices/{deviceId}"].Delete.OperationID)
}

func TestGenerator_APIKeyEndpoints(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"apikey"},
	})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/keys"])
	assert.NotNil(t, spec.Paths["/v1/auth/keys"].Post)
	assert.NotNil(t, spec.Paths["/v1/auth/keys"].Get)
	assert.Equal(t, "createAPIKey", spec.Paths["/v1/auth/keys"].Post.OperationID)
	assert.Equal(t, "listAPIKeys", spec.Paths["/v1/auth/keys"].Get.OperationID)

	assert.NotNil(t, spec.Paths["/v1/auth/keys/{keyId}"])
	assert.NotNil(t, spec.Paths["/v1/auth/keys/{keyId}"].Delete)
	assert.Equal(t, "revokeAPIKey", spec.Paths["/v1/auth/keys/{keyId}"].Delete.OperationID)
}

func TestGenerator_APIKeyEndpoints_NotIncludedByDefault(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	// Without apikey plugin enabled, API key paths should not be present
	assert.Nil(t, spec.Paths["/v1/auth/keys"])
	assert.Nil(t, spec.Paths["/v1/auth/keys/{keyId}"])
}

func TestGenerator_NoPluginsExcludesPluginPaths(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	// Without plugins enabled, plugin paths should not be present
	assert.Nil(t, spec.Paths["/v1/auth/orgs"])
	assert.Nil(t, spec.Paths["/v1/auth/social/{provider}"])
	assert.Nil(t, spec.Paths["/v1/auth/magic-link/send"])
	assert.Nil(t, spec.Paths["/v1/auth/mfa/enroll"])
	assert.Nil(t, spec.Paths["/v1/auth/keys"])
}

func TestGenerator_WithSocialPlugin(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"social"},
	})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/social/{provider}"])
	assert.NotNil(t, spec.Paths["/v1/auth/social/{provider}/callback"])
}

func TestGenerator_WithMagicLinkPlugin(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"magiclink"},
	})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/magic-link/send"])
	assert.NotNil(t, spec.Paths["/v1/auth/magic-link/verify"])
}

func TestGenerator_WithMFAPlugin(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"mfa"},
	})
	spec := gen.Generate()

	assert.NotNil(t, spec.Paths["/v1/auth/mfa/enroll"])
	assert.NotNil(t, spec.Paths["/v1/auth/mfa/verify"])
	assert.NotNil(t, spec.Paths["/v1/auth/mfa/challenge"])
}

func TestGenerator_AllPlugins(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa", "apikey"},
	})
	spec := gen.Generate()

	// Core (5) + password (4) + user (1, combined) + sessions (2) + devices (2)
	// + orgs (5) + social (2) + magiclink (2) + mfa (3) + apikey (2) + well-known (2) = 30 paths
	assert.True(t, len(spec.Paths) >= 25, "expected at least 25 paths, got %d", len(spec.Paths))
}

func TestGenerator_Components(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization"},
	})
	spec := gen.Generate()

	require.NotNil(t, spec.Components)

	// Core schemas
	assert.NotNil(t, spec.Components.Schemas["User"])
	assert.NotNil(t, spec.Components.Schemas["Session"])
	assert.NotNil(t, spec.Components.Schemas["AuthResponse"])
	assert.NotNil(t, spec.Components.Schemas["TokenResponse"])
	assert.NotNil(t, spec.Components.Schemas["Error"])
	assert.NotNil(t, spec.Components.Schemas["Manifest"])

	// Organization schemas (enabled via plugin)
	assert.NotNil(t, spec.Components.Schemas["Organization"])
	assert.NotNil(t, spec.Components.Schemas["Member"])
	assert.NotNil(t, spec.Components.Schemas["Invitation"])
	assert.NotNil(t, spec.Components.Schemas["Device"])
	assert.NotNil(t, spec.Components.Schemas["APIKey"])

	// Security schemes
	assert.NotNil(t, spec.Components.SecuritySchemes["bearerAuth"])
	assert.Equal(t, "http", spec.Components.SecuritySchemes["bearerAuth"].Type)
	assert.Equal(t, "bearer", spec.Components.SecuritySchemes["bearerAuth"].Scheme)

	assert.NotNil(t, spec.Components.SecuritySchemes["apiKeyAuth"])
	assert.Equal(t, "apiKey", spec.Components.SecuritySchemes["apiKeyAuth"].Type)
	assert.Equal(t, "header", spec.Components.SecuritySchemes["apiKeyAuth"].In)
}

func TestGenerator_Tags(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa", "apikey"},
	})
	spec := gen.Generate()

	tagNames := make([]string, len(spec.Tags))
	for i, tag := range spec.Tags {
		tagNames[i] = tag.Name
	}

	// Core tags (always present)
	assert.Contains(t, tagNames, "Authentication")
	assert.Contains(t, tagNames, "Password")
	assert.Contains(t, tagNames, "User")
	assert.Contains(t, tagNames, "Sessions")
	assert.Contains(t, tagNames, "Devices")

	// Organization tag (plugin-enabled)
	assert.Contains(t, tagNames, "Organizations")
	assert.Contains(t, tagNames, "System")

	// Plugin tags
	assert.Contains(t, tagNames, "Social")
	assert.Contains(t, tagNames, "Magic Link")
	assert.Contains(t, tagNames, "MFA")
	assert.Contains(t, tagNames, "API Keys")
}

func TestGenerator_OperationIDs(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa", "apikey"},
	})
	spec := gen.Generate()

	// Collect all operation IDs
	opIDs := make(map[string]bool)
	for _, path := range spec.Paths {
		for _, op := range []*openapi.Operation{path.Get, path.Post, path.Put, path.Patch, path.Delete} {
			if op != nil && op.OperationID != "" {
				assert.False(t, opIDs[op.OperationID], "duplicate operationId: %s", op.OperationID)
				opIDs[op.OperationID] = true
			}
		}
	}

	// Core operations
	assert.True(t, opIDs["signUp"])
	assert.True(t, opIDs["signIn"])
	assert.True(t, opIDs["signOut"])
	assert.True(t, opIDs["refreshSession"])
	assert.True(t, opIDs["getMe"])
	assert.True(t, opIDs["updateMe"])

	// Password operations
	assert.True(t, opIDs["forgotPassword"])
	assert.True(t, opIDs["resetPassword"])
	assert.True(t, opIDs["changePassword"])
	assert.True(t, opIDs["verifyEmail"])

	// Org operations
	assert.True(t, opIDs["createOrg"])
	assert.True(t, opIDs["listOrgs"])
	assert.True(t, opIDs["getOrg"])
	assert.True(t, opIDs["updateOrg"])
	assert.True(t, opIDs["deleteOrg"])
	assert.True(t, opIDs["listMembers"])
	assert.True(t, opIDs["addMember"])
	assert.True(t, opIDs["removeMember"])
	assert.True(t, opIDs["listInvitations"])
	assert.True(t, opIDs["createInvitation"])

	// Device operations
	assert.True(t, opIDs["listDevices"])
	assert.True(t, opIDs["deleteDevice"])

	// API Key operations
	assert.True(t, opIDs["createAPIKey"])
	assert.True(t, opIDs["listAPIKeys"])
	assert.True(t, opIDs["revokeAPIKey"])
}

func TestGenerator_JSONSerialization(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		Title:          "Test API",
		EnabledPlugins: []string{"organization", "social", "magiclink", "mfa", "apikey"},
	})
	spec := gen.Generate()

	// Verify it serializes to valid JSON
	data, err := json.Marshal(spec)
	require.NoError(t, err)
	assert.True(t, len(data) > 100)

	// Verify it deserializes back
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "3.1.0", parsed["openapi"])
}

func TestGenerator_SignUpEndpointDetails(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	signup := spec.Paths["/v1/auth/signup"].Post
	require.NotNil(t, signup)

	assert.Equal(t, "signUp", signup.OperationID)
	assert.Contains(t, signup.Tags, "Authentication")
	require.NotNil(t, signup.RequestBody)
	assert.True(t, signup.RequestBody.Required)

	// Check responses
	assert.NotNil(t, signup.Responses["201"])
	assert.NotNil(t, signup.Responses["400"])
	assert.NotNil(t, signup.Responses["409"])
}

func TestGenerator_OrganizationSchema(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		EnabledPlugins: []string{"organization"},
	})
	spec := gen.Generate()

	org := spec.Components.Schemas["Organization"]
	require.NotNil(t, org)
	assert.Equal(t, "object", org.Type)
	assert.NotNil(t, org.Properties["id"])
	assert.NotNil(t, org.Properties["name"])
	assert.NotNil(t, org.Properties["slug"])
	assert.NotNil(t, org.Properties["logo"])
	assert.NotNil(t, org.Properties["created_at"])
}

func TestGenerator_DeviceSchema(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	device := spec.Components.Schemas["Device"]
	require.NotNil(t, device)
	assert.Equal(t, "object", device.Type)
	assert.NotNil(t, device.Properties["id"])
	assert.NotNil(t, device.Properties["user_id"])
	assert.NotNil(t, device.Properties["fingerprint"])
	assert.NotNil(t, device.Properties["ip_address"])
}

func TestGenerator_APIKeySchema(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	apiKey := spec.Components.Schemas["APIKey"]
	require.NotNil(t, apiKey)
	assert.Equal(t, "object", apiKey.Type)
	assert.NotNil(t, apiKey.Properties["id"])
	assert.NotNil(t, apiKey.Properties["name"])
	assert.NotNil(t, apiKey.Properties["key_prefix"])
	assert.NotNil(t, apiKey.Properties["scopes"])
	assert.NotNil(t, apiKey.Properties["expires_at"])
}

func TestGenerator_ErrorSchemaIncludesCode(t *testing.T) {
	gen := openapi.NewGenerator(openapi.GeneratorConfig{})
	spec := gen.Generate()

	errSchema := spec.Components.Schemas["Error"]
	require.NotNil(t, errSchema)
	assert.NotNil(t, errSchema.Properties["error"])
	assert.NotNil(t, errSchema.Properties["code"])
	assert.Equal(t, "integer", errSchema.Properties["code"].Type)
}
