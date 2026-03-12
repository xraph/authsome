// Command specgen dumps the AuthSome OpenAPI spec by booting the engine with
// all known plugins, registering their routes on a Forge router, and serializing
// the dynamically-generated spec to JSON. This replaces the hardcoded spec
// generator and ensures the SDK always matches the actual API surface.
//
// Usage:
//
//	go run ./sdkgen/cmd/specgen --out=spec.json --title="AuthSome API" --version=0.5.0
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/mfa"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/sso"
	"github.com/xraph/authsome/store/memory"
)

// publicOperations lists operationIDs that do NOT require authentication.
// All other operations default to bearerAuth security.
var publicOperations = map[string]bool{
	// Core auth — unauthenticated by design
	"signUp":        true,
	"signIn":        true,
	"refreshTokens": true,

	// Password recovery flow
	"forgotPassword": true,
	"resetPassword":  true,
	"verifyEmail":    true,

	// Well-known / health
	"getManifest": true,
	"getOpenAPI":  true,
	"getHealth":   true,

	// Magic link (unauthenticated send + verify)
	"sendMagicLink":   true,
	"verifyMagicLink": true,

	// Social OAuth (redirect-based, no token yet)
	"startOAuth":    true,
	"oauthCallback": true,

	// SSO / SAML
	"startSSOLogin": true,
	"ssoACS":        true,
	"ssoCallback":   true,

	// Passkey login (no token yet)
	"passkeyLoginBegin":  true,
	"passkeyLoginFinish": true,

	// MFA challenge/verify — called during login before full auth
	"challengeMFA":      true,
	"verifyMFA":         true,
	"verifyMFARecovery": true,
}

func main() {
	out := flag.String("out", "spec.json", "Output file path for the generated OpenAPI spec")
	title := flag.String("title", "AuthSome API", "API title in the spec info block")
	version := flag.String("version", "0.5.0", "API version in the spec info block")
	flag.Parse()

	if err := run(*out, *title, *version); err != nil {
		fmt.Fprintf(os.Stderr, "specgen: %v\n", err)
		os.Exit(1)
	}
}

func run(outPath, title, version string) error {
	logger := log.NewNoopLogger()
	store := memory.New()

	// Build the engine with every known plugin so their routes are registered.
	// We don't call engine.Start — we only need route metadata, not runtime state.
	engine, err := authsome.NewEngine(
		authsome.WithStore(store),
		authsome.WithLogger(logger),
		authsome.WithDisableMigrate(),

		// Core
		authsome.WithPlugin(password.New()),

		// Social OAuth (zero-config — no providers, but routes are still registered)
		authsome.WithPlugin(social.New(social.Config{})),

		// MFA (TOTP, SMS, recovery codes)
		authsome.WithPlugin(mfa.New(mfa.Config{})),

		// API keys
		authsome.WithPlugin(apikey.New()),

		// Magic link (zero-config — no mailer, but routes are still registered)
		authsome.WithPlugin(magiclink.New(magiclink.Config{})),

		// SSO / SAML (zero-config — no providers)
		authsome.WithPlugin(sso.New(sso.Config{})),

		// Passkeys / WebAuthn
		authsome.WithPlugin(passkey.New(passkey.Config{})),

		// Organizations (opt-in)
		authsome.WithPlugin(orgplugin.New()),
	)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}

	// Create a Forge router with OpenAPI generation enabled.
	router := forge.NewRouter(forge.WithOpenAPI(forge.OpenAPIConfig{
		Title:       title,
		Version:     version,
		Description: "Authentication API powered by AuthSome",
		Security: map[string]forge.SecurityScheme{
			"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
		},
	}))

	// Register core API routes (auth, password, user, session, org, device, etc.).
	apiHandler := api.New(engine)
	if routeErr := apiHandler.RegisterRoutes(router); routeErr != nil {
		return fmt.Errorf("register API routes: %w", routeErr)
	}

	// Register plugin routes (each plugin that implements RouteProvider).
	for _, rp := range engine.Plugins().RouteProviders() {
		if pluginRouteErr := rp.RegisterRoutes(router); pluginRouteErr != nil {
			return fmt.Errorf("register plugin routes (%T): %w", rp, pluginRouteErr)
		}
	}

	// Start the router so the OpenAPI generator processes all registered routes.
	if startErr := router.Start(context.Background()); startErr != nil {
		return fmt.Errorf("start router: %w", startErr)
	}

	// Extract the dynamically-generated OpenAPI spec.
	spec := router.OpenAPISpec()
	if spec == nil {
		return fmt.Errorf("router returned nil OpenAPI spec (is WithOpenAPI configured?)")
	}

	// Marshal the Forge spec to JSON, then unmarshal to a generic map so we
	// can post-process (add security requirements, clean up GET request bodies).
	rawJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	var specMap map[string]any
	if err := json.Unmarshal(rawJSON, &specMap); err != nil {
		return fmt.Errorf("unmarshal spec: %w", err)
	}

	// Post-process: add security requirements and clean up.
	postProcess(specMap)

	data, err := json.MarshalIndent(specMap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil { //nolint:gosec // G306: file permissions appropriate for generated spec
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "OpenAPI spec written to %s (%d bytes)\n", outPath, len(data))
	return nil
}

// postProcess adds security requirements to operations, removes request
// bodies from GET operations, and fixes schema types that the Forge OpenAPI
// generator couldn't resolve (Go interface/any fields → unknown).
func postProcess(spec map[string]any) {
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return
	}

	bearerSecurity := []any{
		map[string]any{"bearerAuth": []any{}},
	}

	for _, pathItem := range paths {
		methods, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}

		for method, opAny := range methods {
			op, ok := opAny.(map[string]any)
			if !ok {
				continue
			}

			operationID, _ := op["operationId"].(string)

			// Add security requirements to non-public operations.
			if operationID != "" && !publicOperations[operationID] {
				if _, hasSecurity := op["security"]; !hasSecurity {
					op["security"] = bearerSecurity
				}
			}

			// Remove request bodies from GET operations (not valid in OpenAPI 3.1).
			if method == "get" {
				delete(op, "requestBody")
			}
		}
	}

	// Fix unresolved schema types. The Forge OpenAPI generator emits fields
	// without type/ref when the Go struct uses interface{}/any. We patch
	// these to the correct $ref or type here.
	patchSchemaTypes(spec)
}

// schemaRef returns an OpenAPI $ref to a component schema.
func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

// arrayOfRef returns an OpenAPI array schema referencing a component schema.
func arrayOfRef(name string) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": schemaRef(name),
	}
}

// patchSchemaTypes fixes fields that the Forge OpenAPI generator couldn't
// resolve because the Go types use interface{} / any.
func patchSchemaTypes(spec map[string]any) {
	components, _ := spec["components"].(map[string]any)
	if components == nil {
		return
	}
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		return
	}

	// Map of schema → field → corrected type spec.
	// Values are map[string]any representing the OpenAPI field schema.
	patches := map[string]map[string]any{
		// User fields in auth responses
		"AuthResponse":     {"user": schemaRef("User")},
		"CallbackResponse": {"user": schemaRef("User"), "expires_at": map[string]any{"type": "string"}},
		"VerifyResponse":   {"user": schemaRef("User"), "expires_at": map[string]any{"type": "string"}},

		// List response arrays
		"AdminUserListResponse":  {"users": arrayOfRef("User")},
		"DeviceListResponse":     {"devices": arrayOfRef("Device")},
		"InvitationListResponse": {"invitations": arrayOfRef("Invitation")},
		"MemberListResponse":     {"members": arrayOfRef("Member")},
		"OrgListResponse":        {"organizations": arrayOfRef("Organization")},
		"PermissionListResponse": {"permissions": arrayOfRef("Permission")},
		"RoleListResponse":       {"roles": arrayOfRef("Role")},
		"SessionListResponse":    {"sessions": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}},
		"TeamListResponse":       {"teams": arrayOfRef("Team")},
		"UserRoleListResponse":   {"roles": arrayOfRef("Role")},
		"WebhookListResponse":    {"webhooks": arrayOfRef("Webhook")},

		// WebAuthn opaque options
		"LoginBeginResponse":    {"options": map[string]any{"type": "object"}},
		"RegisterBeginResponse": {"options": map[string]any{"type": "object"}},
	}

	for schemaName, fieldPatches := range patches {
		schemaAny, ok := schemas[schemaName]
		if !ok {
			continue
		}
		schema, ok := schemaAny.(map[string]any)
		if !ok {
			continue
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			continue
		}
		for fieldName, fieldTypeAny := range fieldPatches {
			if _, exists := props[fieldName]; exists {
				props[fieldName] = fieldTypeAny
			}
		}
	}
}
