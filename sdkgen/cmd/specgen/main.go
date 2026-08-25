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
	"sort"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/consent"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/oauth2provider"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/sso"
	"github.com/xraph/authsome/plugins/subscription"
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
	// Resend the verification link to a possibly-unauthenticated user
	// who can't sign in until verified. Endpoint always returns 200
	// regardless of whether the email is registered, preserving the
	// enumeration-resistance contract.
	"resendEmailVerification": true,

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

	// Phone OTP (unauthenticated start + verify)
	"phoneAuthStart":  true,
	"phoneAuthVerify": true,

	// OAuth2 Provider (token endpoint is public per RFC 6749)
	"oauth2Token":           true,
	"oauth2Authorize":       true,
	"oauth2DeviceAuthorize": true,

	// Token introspection (RFC 7662 — public by design, validates tokens)
	"introspectToken": true,

	// SCIM discovery endpoints (public per RFC 7644)
	"scimGetServiceProviderConfig": true,
	"scimGetSchemas":               true,
	"scimGetResourceTypes":         true,
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
	// We call engine.Start so plugins receive OnInit (needed for route prefixes).
	// Migrations are disabled via WithDisableMigrate().
	wardenEng, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	if err != nil {
		return fmt.Errorf("create warden engine: %w", err)
	}

	engine, err := authsome.NewEngine(
		authsome.WithStore(store),
		authsome.WithLogger(logger),
		authsome.WithWarden(wardenEng),
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

		// Organizations
		authsome.WithPlugin(orgplugin.New(orgplugin.Config{})),

		// SCIM (System for Cross-domain Identity Management)
		authsome.WithPlugin(scim.New()),

		// Subscription / Billing
		authsome.WithPlugin(subscription.New()),

		// Phone OTP
		authsome.WithPlugin(phone.New()),

		// Consent management
		authsome.WithPlugin(consent.New()),

		// OAuth2 / OIDC Provider
		authsome.WithPlugin(oauth2provider.New()),
	)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}

	// Start the engine so plugins receive OnInit (needed for route prefix setup).
	// Migrations are disabled, so this only initializes plugins and strategies.
	if startErr := engine.Start(context.Background()); startErr != nil {
		return fmt.Errorf("start engine: %w", startErr)
	}

	// Create a Forge router with OpenAPI generation enabled.
	router := forge.NewRouter(forge.WithOpenAPI(forge.OpenAPIConfig{
		Title:       title,
		Version:     version,
		Description: "Authentication API powered by AuthSome",
		// Every name a route can reach for via WithGroupAuth has to be declared
		// here. Routes were already asking for "session" and "session-cookie",
		// but nothing declared them, so the spec carried $refs to components
		// that did not exist and the SDK generators had no scheme to inspect —
		// which is how 87 operations ended up with no way to pass a token.
		Security: map[string]forge.SecurityScheme{
			"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			// Reads Authorization: Bearer first, then falls back to the cookie
			// below. See extractToken in extension/auth_pages.go.
			"session":        {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			"session-cookie": {Type: "apiKey", In: "cookie", Name: "auth_token"},
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
	if unmarshalErr := json.Unmarshal(rawJSON, &specMap); unmarshalErr != nil {
		return fmt.Errorf("unmarshal spec: %w", unmarshalErr)
	}

	// Post-process: add security requirements, clean up paths, fix schemas.
	// Order matters: the body-suffix names are normalized first, so the schema
	// patches below are keyed on the names that actually reach the document.
	normalizeRequestBodyNames(specMap)
	normalizeFormContentTypes(specMap)

	if patchErr := postProcess(specMap); patchErr != nil {
		return patchErr
	}

	cleanPaths(specMap)

	data, err := json.MarshalIndent(specMap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	// #nosec G306 -- file written for the process's own data.
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "OpenAPI spec written to %s (%d bytes)\n", outPath, len(data))
	return nil
}

// normalizeRequestBodyNames renames a component called XRequestBody to
// XRequest and moves every reference with it.
//
// The suffix is not ours. Forge appends "Body" when a request body is one field
// of a larger request struct, which describes how the Go type is laid out and
// says nothing a client author cares about: SignUpRequestBody is the type you
// pass to signUp, and SignUpRequest is what it should be called. Before forge
// v1.9.10 these components had no suffix, so leaving it in place would rename
// fifty-six types across the Go, TypeScript and Dart SDKs and break every
// caller of them for no gain.
//
// A name is only taken if nothing already holds it. If both XRequest and
// XRequestBody are in the document they are two different types, and the one
// that named itself keeps its name.
func normalizeRequestBodyNames(spec map[string]any) {
	components, _ := spec["components"].(map[string]any) //nolint:errcheck // type assertion
	if components == nil {
		return
	}

	schemas, _ := components["schemas"].(map[string]any) //nolint:errcheck // type assertion
	if schemas == nil {
		return
	}

	const suffix = "RequestBody"

	renames := make(map[string]string)

	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		target := strings.TrimSuffix(name, "Body")
		if _, taken := schemas[target]; taken {
			continue
		}

		// Reserve it, so two components cannot both normalize onto one name.
		schemas[target] = schemas[name]

		delete(schemas, name)

		renames[name] = target
	}

	if len(renames) == 0 {
		return
	}

	rewriteRefs(spec, renames)
}

// rewriteRefs walks the whole document and repoints every $ref named in
// renames. Refs are plain strings sitting anywhere in the tree, so the walk has
// to cover all of it rather than the handful of places bodies usually appear.
// normalizeFormContentTypes rewrites form request bodies from
// multipart/form-data to application/x-www-form-urlencoded.
//
// Forge describes every form-tagged struct as multipart, which is the wrong
// half of the form family for the routes that actually have one. The OAuth2
// endpoints are the whole population here, and RFC 6749 §4.1.3 requires
// urlencoded; a conformant client will never send multipart to a token
// endpoint. Left alone, the published document tells every consumer, ours and
// anyone else's, to speak an encoding the endpoint does not want.
//
// A body carrying binary is a real upload and keeps its multipart, since
// describing a file as a urlencoded scalar would be a worse lie than the one
// being fixed. A body that already offers urlencoded is left as it is.
func normalizeFormContentTypes(spec map[string]any) {
	const (
		multipart  = "multipart/form-data"
		urlencoded = "application/x-www-form-urlencoded"
	)

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return
	}

	for _, pathItem := range paths {
		methods, isObject := pathItem.(map[string]any)
		if !isObject {
			continue
		}

		for _, op := range methods {
			operation, isOp := op.(map[string]any)
			if !isOp {
				continue
			}

			body, hasBody := operation["requestBody"].(map[string]any)
			if !hasBody {
				continue
			}

			content, hasContent := body["content"].(map[string]any)
			if !hasContent {
				continue
			}

			media, isMultipart := content[multipart]
			if !isMultipart {
				continue
			}

			if _, alreadyURLEncoded := content[urlencoded]; alreadyURLEncoded {
				continue
			}

			if schemaCarriesBinary(media) {
				continue
			}

			content[urlencoded] = media
			delete(content, multipart)
		}
	}
}

// schemaCarriesBinary reports whether a media type describes any binary
// property, which is what separates a file upload from a set of form scalars.
func schemaCarriesBinary(media any) bool {
	entry, ok := media.(map[string]any)
	if !ok {
		return false
	}

	schema, hasSchema := entry["schema"].(map[string]any)
	if !hasSchema {
		return false
	}

	props, hasProps := schema["properties"].(map[string]any)
	if !hasProps {
		return false
	}

	for _, prop := range props {
		field, isObject := prop.(map[string]any)
		if !isObject {
			continue
		}

		if field["format"] == "binary" {
			return true
		}

		// An array of files is still a file upload.
		if items, isArray := field["items"].(map[string]any); isArray {
			if items["format"] == "binary" {
				return true
			}
		}
	}

	return false
}

func rewriteRefs(node any, renames map[string]string) {
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			if name, found := strings.CutPrefix(ref, componentPrefix); found {
				if target, renamed := renames[name]; renamed {
					v["$ref"] = componentPrefix + target
				}
			}
		}

		for _, child := range v {
			rewriteRefs(child, renames)
		}
	case []any:
		for _, child := range v {
			rewriteRefs(child, renames)
		}
	}
}

// postProcess adds security requirements to operations, removes request
// bodies from GET operations, and fixes schema types that the Forge OpenAPI
// generator couldn't resolve (Go interface/any fields → unknown).
func postProcess(spec map[string]any) error {
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return nil
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

			operationID, _ := op["operationId"].(string) //nolint:errcheck // type assertion, fallback to zero

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
	return patchSchemaTypes(spec)
}

// componentPrefix is the $ref prefix every component schema reference carries.
const componentPrefix = "#/components/schemas/"

// schemaRef returns an OpenAPI $ref to a component schema.
func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": componentPrefix + name}
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
func patchSchemaTypes(spec map[string]any) error {
	components, _ := spec["components"].(map[string]any) //nolint:errcheck // type assertion
	if components == nil {
		return nil
	}
	schemas, _ := components["schemas"].(map[string]any) //nolint:errcheck // type assertion
	if schemas == nil {
		return nil
	}

	// Map of schema → field → corrected type spec.
	// Values are map[string]any representing the OpenAPI field schema.
	patches := map[string]map[string]any{
		// User fields in auth responses
		// The three callback/verify responses carry the same two any-typed
		// fields. They used to share the component names CallbackResponse and
		// VerifyResponse, which meant one entry each here; forge now keeps the
		// magiclink, social and SSO versions apart, so each gets its own.
		"AuthResponse":            {"user": schemaRef("User")},
		"SocialCallbackResponse":  {"user": schemaRef("User"), "expires_at": map[string]any{"type": "string"}},
		"SsoCallbackResponse":     {"user": schemaRef("User"), "expires_at": map[string]any{"type": "string"}},
		"MagiclinkVerifyResponse": {"user": schemaRef("User"), "expires_at": map[string]any{"type": "string"}},

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

	// A patch keyed on a name no longer in the document is not a no-op, it is
	// a silent hole: the field it was meant to type stays untyped and the SDKs
	// generate an opaque blob for it. That is exactly what a forge upgrade did
	// here, renaming VerifyResponse and CallbackResponse out from under two
	// entries. Fail instead, so the next rename is a build error you can fix in
	// one line rather than a regression nobody notices until a client hits it.
	missing := make([]string, 0)

	for schemaName, fieldPatches := range patches {
		schemaAny, ok := schemas[schemaName]
		if !ok {
			missing = append(missing, schemaName)

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

	if len(missing) > 0 {
		sort.Strings(missing)

		return fmt.Errorf(
			"schema patches target components the spec no longer has: %s"+
				" (a dependency probably renamed them; update the patches map in this file)",
			strings.Join(missing, ", "))
	}

	return nil
}

// cleanPaths normalizes path keys: replaces double slashes with single,
// removes trailing slashes, and ensures leading slash.
func cleanPaths(spec map[string]any) {
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return
	}

	cleaned := make(map[string]any, len(paths))
	for path, value := range paths {
		// Replace double slashes
		for strings.Contains(path, "//") {
			path = strings.ReplaceAll(path, "//", "/")
		}
		// Remove trailing slash (except root)
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			path = strings.TrimRight(path, "/")
		}
		cleaned[path] = value
	}
	spec["paths"] = cleaned
}
