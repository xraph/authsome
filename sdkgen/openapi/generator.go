package openapi

// Deprecated: This hardcoded generator is superseded by the specgen tool
// (sdkgen/cmd/specgen/) which dynamically generates specs from actual route
// metadata registered on a Forge router. Kept as fallback for runtime use
// when an OpenAPI-enabled router is not available. Use `make dump-spec` to
// produce a spec from the real routes.

// GeneratorConfig holds configuration for OpenAPI spec generation.
type GeneratorConfig struct {
	// Title is the API title (default: "AuthSome API").
	Title string

	// Description is the API description.
	Description string

	// Version is the API version (default: "0.5.0").
	Version string

	// BasePath is the base path for all endpoints (default: "/v1/auth").
	BasePath string

	// ServerURL is the server URL for the spec.
	ServerURL string

	// EnabledPlugins lists which plugins are enabled (affects which paths are included).
	EnabledPlugins []string
}

// Generator produces an OpenAPI 3.1 spec from AuthSome engine metadata.
type Generator struct {
	config GeneratorConfig
}

// NewGenerator creates a new OpenAPI spec generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	if cfg.Title == "" {
		cfg.Title = "AuthSome API"
	}
	if cfg.Version == "" {
		cfg.Version = "0.5.0"
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "/v1/auth"
	}
	return &Generator{config: cfg}
}

// Generate produces a complete OpenAPI 3.1 specification.
func (g *Generator) Generate() *Spec {
	spec := &Spec{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       g.config.Title,
			Description: g.config.Description,
			Version:     g.config.Version,
		},
		Paths:      make(map[string]*PathItem),
		Components: g.buildComponents(),
		Tags:       g.buildTags(),
	}

	if g.config.ServerURL != "" {
		spec.Servers = []Server{{URL: g.config.ServerURL}}
	}

	// Core auth endpoints
	g.addAuthPaths(spec)

	// Password management endpoints
	g.addPasswordPaths(spec)

	// User endpoints
	g.addUserPaths(spec)

	// Session endpoints
	g.addSessionPaths(spec)

	// Device endpoints
	g.addDevicePaths(spec)

	// Plugin endpoints
	enabledPlugins := make(map[string]bool)
	for _, p := range g.config.EnabledPlugins {
		enabledPlugins[p] = true
	}

	if enabledPlugins["organization"] {
		g.addOrgPaths(spec)
	}
	if enabledPlugins["social"] {
		g.addSocialPaths(spec)
	}
	if enabledPlugins["magiclink"] {
		g.addMagicLinkPaths(spec)
	}
	if enabledPlugins["mfa"] {
		g.addMFAPaths(spec)
	}
	if enabledPlugins["apikey"] {
		g.addAPIKeyPaths(spec)
	}
	if enabledPlugins["sso"] {
		g.addSSOPaths(spec)
	}

	// Admin endpoints
	g.addAdminPaths(spec)

	// GDPR endpoints
	g.addGDPRPaths(spec)

	// Well-known endpoints
	g.addWellKnownPaths(spec)

	return spec
}

// ──────────────────────────────────────────────────
// Path builders
// ──────────────────────────────────────────────────

func (g *Generator) addAuthPaths(spec *Spec) {
	spec.Paths["/v1/signup"] = &PathItem{
		Post: &Operation{
			Summary:     "Sign up a new user",
			OperationID: "signUp",
			Tags:        []string{"Authentication"},
			Security:    []SecurityRequirement{{}}, // No auth required
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"email", "password"},
				Properties: map[string]*Schema{
					"email":    {Type: "string", Format: "email"},
					"password": {Type: "string", Format: "password"},
					"name":     {Type: "string"},
					"username": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"201": jsonResponse("Successful signup", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
				"409": jsonResponse("Email already taken", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/signin"] = &PathItem{
		Post: &Operation{
			Summary:     "Sign in with credentials",
			OperationID: "signIn",
			Tags:        []string{"Authentication"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"password"},
				Properties: map[string]*Schema{
					"email":    {Type: "string", Format: "email"},
					"username": {Type: "string"},
					"password": {Type: "string", Format: "password"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Successful sign-in", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"401": jsonResponse("Invalid credentials", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/signout"] = &PathItem{
		Post: &Operation{
			Summary:     "Sign out (revoke session)",
			OperationID: "signOut",
			Tags:        []string{"Authentication"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Signed out", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/refresh"] = &PathItem{
		Post: &Operation{
			Summary:     "Refresh session tokens",
			OperationID: "refreshSession",
			Tags:        []string{"Authentication"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"refresh_token"},
				Properties: map[string]*Schema{
					"refresh_token": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Refreshed tokens", &Schema{Ref: "#/components/schemas/TokenResponse"}),
				"400": jsonResponse("Invalid refresh token", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/health"] = &PathItem{
		Get: &Operation{
			Summary:     "Health check",
			OperationID: "healthCheck",
			Tags:        []string{"System"},
			Security:    []SecurityRequirement{{}},
			Responses: map[string]*Response{
				"200": jsonResponse("Healthy", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"status": {Type: "string"},
						"error":  {Type: "string"},
					},
				}),
			},
		},
	}
}

func (g *Generator) addPasswordPaths(spec *Spec) {
	spec.Paths["/v1/forgot-password"] = &PathItem{
		Post: &Operation{
			Summary:     "Request password reset",
			Description: "Sends a password reset email. Always returns success to prevent email enumeration.",
			OperationID: "forgotPassword",
			Tags:        []string{"Password"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"email"},
				Properties: map[string]*Schema{
					"email": {Type: "string", Format: "email"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Reset email sent", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/reset-password"] = &PathItem{
		Post: &Operation{
			Summary:     "Reset password with token",
			Description: "Resets the user password using a token from the forgot-password email.",
			OperationID: "resetPassword",
			Tags:        []string{"Password"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"token", "new_password"},
				Properties: map[string]*Schema{
					"token":        {Type: "string", Description: "Reset token from email"},
					"new_password": {Type: "string", Format: "password", Description: "New password"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Password reset", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid or expired token", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/change-password"] = &PathItem{
		Post: &Operation{
			Summary:     "Change password",
			Description: "Changes the authenticated user's password.",
			OperationID: "changePassword",
			Tags:        []string{"Password"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"current_password", "new_password"},
				Properties: map[string]*Schema{
					"current_password": {Type: "string", Format: "password", Description: "Current password"},
					"new_password":     {Type: "string", Format: "password", Description: "New password"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Password changed", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Weak password", &Schema{Ref: "#/components/schemas/Error"}),
				"401": jsonResponse("Invalid current password", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/verify-email"] = &PathItem{
		Post: &Operation{
			Summary:     "Verify email address",
			Description: "Verifies the user's email address using a verification token.",
			OperationID: "verifyEmail",
			Tags:        []string{"Password"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"token"},
				Properties: map[string]*Schema{
					"token": {Type: "string", Description: "Email verification token"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Email verified", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid or expired token", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/resend-verification"] = &PathItem{
		Post: &Operation{
			Summary:     "Resend email verification",
			Description: "Resends the email verification link to the authenticated user.",
			OperationID: "resendVerification",
			Tags:        []string{"Password"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Verification email sent", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addUserPaths(spec *Spec) {
	spec.Paths["/v1/me"] = &PathItem{
		Get: &Operation{
			Summary:     "Get current user",
			OperationID: "getMe",
			Tags:        []string{"User"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Current user", &Schema{Ref: "#/components/schemas/User"}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Patch: &Operation{
			Summary:     "Update current user profile",
			OperationID: "updateMe",
			Tags:        []string{"User"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name":     {Type: "string"},
					"username": {Type: "string"},
					"image":    {Type: "string", Format: "uri"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Updated user", &Schema{Ref: "#/components/schemas/User"}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addSessionPaths(spec *Spec) {
	spec.Paths["/v1/sessions"] = &PathItem{
		Get: &Operation{
			Summary:     "List active sessions",
			OperationID: "listSessions",
			Tags:        []string{"Sessions"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Session list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"sessions": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Session"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/sessions/{id}"] = &PathItem{
		Delete: &Operation{
			Summary:     "Revoke a session",
			OperationID: "revokeSession",
			Tags:        []string{"Sessions"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Session revoked", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid session ID", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addOrgPaths(spec *Spec) {
	spec.Paths["/v1/orgs"] = &PathItem{
		Post: &Operation{
			Summary:     "Create organization",
			Description: "Creates a new organization.",
			OperationID: "createOrg",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"name", "slug"},
				Properties: map[string]*Schema{
					"name": {Type: "string", Description: "Organization name"},
					"slug": {Type: "string", Description: "URL-safe slug"},
					"logo": {Type: "string", Format: "uri", Description: "Logo URL"},
				},
			}),
			Responses: map[string]*Response{
				"201": jsonResponse("Organization created", &Schema{Ref: "#/components/schemas/Organization"}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
				"409": jsonResponse("Slug already taken", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Get: &Operation{
			Summary:     "List user organizations",
			Description: "Lists all organizations the authenticated user belongs to.",
			OperationID: "listOrgs",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Organization list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"organizations": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Organization"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/orgs/{orgId}"] = &PathItem{
		Get: &Operation{
			Summary:     "Get organization",
			OperationID: "getOrg",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Organization details", &Schema{Ref: "#/components/schemas/Organization"}),
				"404": jsonResponse("Organization not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Patch: &Operation{
			Summary:     "Update organization",
			OperationID: "updateOrg",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": {Type: "string"},
					"logo": {Type: "string", Format: "uri"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Organization updated", &Schema{Ref: "#/components/schemas/Organization"}),
				"404": jsonResponse("Organization not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Delete: &Operation{
			Summary:     "Delete organization",
			OperationID: "deleteOrg",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Organization deleted", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("Organization not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	// Members
	spec.Paths["/v1/orgs/{orgId}/members"] = &PathItem{
		Get: &Operation{
			Summary:     "List organization members",
			OperationID: "listMembers",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Member list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"members": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Member"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Post: &Operation{
			Summary:     "Add organization member",
			OperationID: "addMember",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"user_id", "role"},
				Properties: map[string]*Schema{
					"user_id": {Type: "string", Description: "User ID to add"},
					"role":    {Type: "string", Description: "Member role (owner, admin, member)"},
				},
			}),
			Responses: map[string]*Response{
				"201": jsonResponse("Member added", &Schema{Ref: "#/components/schemas/Member"}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
				"409": jsonResponse("User already a member", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/orgs/{orgId}/members/{memberId}"] = &PathItem{
		Delete: &Operation{
			Summary:     "Remove organization member",
			OperationID: "removeMember",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
				{Name: "memberId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Member removed", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("Member not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	// Invitations
	spec.Paths["/v1/orgs/{orgId}/invitations"] = &PathItem{
		Get: &Operation{
			Summary:     "List organization invitations",
			OperationID: "listInvitations",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Invitation list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"invitations": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Invitation"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Post: &Operation{
			Summary:     "Create organization invitation",
			OperationID: "createInvitation",
			Tags:        []string{"Organizations"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "orgId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"email", "role"},
				Properties: map[string]*Schema{
					"email": {Type: "string", Format: "email", Description: "Email to invite"},
					"role":  {Type: "string", Description: "Role for the invitee (admin, member)"},
				},
			}),
			Responses: map[string]*Response{
				"201": jsonResponse("Invitation created", &Schema{Ref: "#/components/schemas/Invitation"}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addDevicePaths(spec *Spec) {
	spec.Paths["/v1/devices"] = &PathItem{
		Get: &Operation{
			Summary:     "List devices",
			Description: "Returns all tracked devices for the authenticated user.",
			OperationID: "listDevices",
			Tags:        []string{"Devices"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Device list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"devices": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Device"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/devices/{deviceId}"] = &PathItem{
		Delete: &Operation{
			Summary:     "Delete device",
			Description: "Removes a tracked device.",
			OperationID: "deleteDevice",
			Tags:        []string{"Devices"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "deviceId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Device deleted", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid device ID", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addAPIKeyPaths(spec *Spec) {
	spec.Paths["/v1/keys"] = &PathItem{
		Post: &Operation{
			Summary:     "Create API key",
			Description: "Creates a new API key. The raw key is returned only once.",
			OperationID: "createAPIKey",
			Tags:        []string{"API Keys"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*Schema{
					"name":       {Type: "string", Description: "Human-readable key name"},
					"scopes":     {Type: "array", Items: &Schema{Type: "string"}, Description: "Permission scopes"},
					"expires_at": {Type: "string", Format: "date-time", Description: "Optional expiration time"},
				},
			}),
			Responses: map[string]*Response{
				"201": jsonResponse("API key created", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"key":        {Ref: "#/components/schemas/APIKey"},
						"raw_key":    {Type: "string", Description: "Raw API key (shown only once)"},
						"key_prefix": {Type: "string", Description: "Key prefix for identification"},
					},
				}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Get: &Operation{
			Summary:     "List API keys",
			Description: "Lists all API keys for the authenticated user. Key hashes are not returned.",
			OperationID: "listAPIKeys",
			Tags:        []string{"API Keys"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("API key list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"keys": {Type: "array", Items: &Schema{Ref: "#/components/schemas/APIKey"}},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/keys/{keyId}"] = &PathItem{
		Delete: &Operation{
			Summary:     "Revoke API key",
			OperationID: "revokeAPIKey",
			Tags:        []string{"API Keys"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "keyId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("API key revoked", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("API key not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addSocialPaths(spec *Spec) {
	spec.Paths["/v1/social/{provider}"] = &PathItem{
		Post: &Operation{
			Summary:     "Start social OAuth flow",
			OperationID: "socialStart",
			Tags:        []string{"Social"},
			Security:    []SecurityRequirement{{}},
			Parameters: []Parameter{
				{Name: "provider", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("OAuth authorization URL", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"auth_url": {Type: "string", Format: "uri"}},
				}),
				"400": jsonResponse("Unsupported provider", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/social/{provider}/callback"] = &PathItem{
		Get: &Operation{
			Summary:     "OAuth callback",
			OperationID: "socialCallback",
			Tags:        []string{"Social"},
			Security:    []SecurityRequirement{{}},
			Parameters: []Parameter{
				{Name: "provider", In: "path", Required: true, Schema: &Schema{Type: "string"}},
				{Name: "code", In: "query", Required: true, Schema: &Schema{Type: "string"}},
				{Name: "state", In: "query", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Successful social sign-in", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"400": jsonResponse("OAuth error", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addMagicLinkPaths(spec *Spec) {
	spec.Paths["/v1/magic-link/send"] = &PathItem{
		Post: &Operation{
			Summary:     "Send magic link email",
			OperationID: "magicLinkSend",
			Tags:        []string{"Magic Link"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"email"},
				Properties: map[string]*Schema{
					"email": {Type: "string", Format: "email"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Magic link sent", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"400": jsonResponse("Invalid request", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/magic-link/verify"] = &PathItem{
		Post: &Operation{
			Summary:     "Verify magic link token",
			OperationID: "magicLinkVerify",
			Tags:        []string{"Magic Link"},
			Security:    []SecurityRequirement{{}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"token"},
				Properties: map[string]*Schema{
					"token": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Successful verification", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"401": jsonResponse("Invalid or expired token", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addMFAPaths(spec *Spec) {
	spec.Paths["/v1/mfa/enroll"] = &PathItem{
		Post: &Operation{
			Summary:     "Enroll in MFA",
			OperationID: "mfaEnroll",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"method": {Type: "string", Enum: []string{"totp", "sms"}},
					"phone":  {Type: "string", Description: "Phone number (required for SMS method)"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("MFA enrollment", &Schema{Ref: "#/components/schemas/MFAEnrollment"}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
				"409": jsonResponse("Already enrolled", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/mfa/verify"] = &PathItem{
		Post: &Operation{
			Summary:     "Verify MFA code",
			OperationID: "mfaVerify",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"code"},
				Properties: map[string]*Schema{
					"code": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Verification result", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"verified": {Type: "boolean"},
						"method":   {Type: "string"},
					},
				}),
				"401": jsonResponse("Invalid code", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/mfa/challenge"] = &PathItem{
		Post: &Operation{
			Summary:     "Complete MFA challenge",
			Description: "Completes an MFA challenge during login. Returns full auth response on success.",
			OperationID: "mfaChallenge",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{}}, // Called during login before full auth
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"enrollment_id", "code"},
				Properties: map[string]*Schema{
					"enrollment_id": {Type: "string", Description: "MFA enrollment ID"},
					"code":          {Type: "string", Description: "MFA code from authenticator"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("MFA challenge passed", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"401": jsonResponse("Invalid code", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/mfa/sms/send"] = &PathItem{
		Post: &Operation{
			Summary:     "Send SMS verification code",
			Description: "Sends an SMS verification code to the phone number associated with the MFA enrollment.",
			OperationID: "mfaSMSSend",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"enrollment_id"},
				Properties: map[string]*Schema{
					"enrollment_id": {Type: "string", Description: "MFA enrollment ID"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("SMS code sent", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/mfa/sms/verify"] = &PathItem{
		Post: &Operation{
			Summary:     "Verify SMS code",
			Description: "Verifies an SMS code for MFA enrollment.",
			OperationID: "mfaSMSVerify",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"enrollment_id", "code"},
				Properties: map[string]*Schema{
					"enrollment_id": {Type: "string", Description: "MFA enrollment ID"},
					"code":          {Type: "string", Description: "SMS verification code"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("SMS verified", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"401": jsonResponse("Invalid code", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/mfa/recovery/verify"] = &PathItem{
		Post: &Operation{
			Summary:     "Verify MFA recovery code",
			Description: "Verifies a recovery code as an alternative to MFA during login.",
			OperationID: "mfaRecoveryVerify",
			Tags:        []string{"MFA"},
			Security:    []SecurityRequirement{{}}, // Called during login
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"code"},
				Properties: map[string]*Schema{
					"code": {Type: "string", Description: "Recovery code"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("Recovery code accepted", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"401": jsonResponse("Invalid recovery code", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addSSOPaths(spec *Spec) {
	spec.Paths["/v1/sso/{provider}/login"] = &PathItem{
		Post: &Operation{
			Summary:     "Start SSO login flow",
			OperationID: "ssoLogin",
			Tags:        []string{"SSO"},
			Security:    []SecurityRequirement{{}},
			Parameters: []Parameter{
				{Name: "provider", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("SSO login URL", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"login_url": {Type: "string", Format: "uri"},
						"state":     {Type: "string"},
					},
				}),
				"400": jsonResponse("Unsupported provider", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/sso/{provider}/callback"] = &PathItem{
		Post: &Operation{
			Summary:     "SSO OIDC callback",
			OperationID: "ssoCallback",
			Tags:        []string{"SSO"},
			Security:    []SecurityRequirement{{}},
			Parameters: []Parameter{
				{Name: "provider", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"code", "state"},
				Properties: map[string]*Schema{
					"code":  {Type: "string"},
					"state": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("SSO authentication result", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"400": jsonResponse("Invalid callback", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/sso/{provider}/acs"] = &PathItem{
		Post: &Operation{
			Summary:     "SSO SAML ACS endpoint",
			OperationID: "ssoACS",
			Tags:        []string{"SSO"},
			Security:    []SecurityRequirement{{}},
			Parameters: []Parameter{
				{Name: "provider", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type:     "object",
				Required: []string{"SAMLResponse"},
				Properties: map[string]*Schema{
					"SAMLResponse": {Type: "string"},
					"RelayState":   {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("SAML authentication result", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"400": jsonResponse("Invalid SAML response", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addAdminPaths(spec *Spec) {
	spec.Paths["/v1/admin/users"] = &PathItem{
		Get: &Operation{
			Summary:     "List all users (admin)",
			OperationID: "adminListUsers",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "limit", In: "query", Schema: &Schema{Type: "integer"}},
				{Name: "offset", In: "query", Schema: &Schema{Type: "integer"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("User list", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"users": {Type: "array", Items: &Schema{Ref: "#/components/schemas/User"}},
						"total": {Type: "integer"},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
				"403": jsonResponse("Forbidden", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/admin/users/{userId}"] = &PathItem{
		Get: &Operation{
			Summary:     "Get user (admin)",
			OperationID: "adminGetUser",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "userId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("User details", &Schema{Ref: "#/components/schemas/User"}),
				"404": jsonResponse("User not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
		Delete: &Operation{
			Summary:     "Delete user (admin)",
			OperationID: "adminDeleteUser",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "userId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("User deleted", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("User not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/admin/users/{userId}/ban"] = &PathItem{
		Post: &Operation{
			Summary:     "Ban user (admin)",
			OperationID: "adminBanUser",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "userId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: jsonBody(&Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"reason": {Type: "string"},
				},
			}),
			Responses: map[string]*Response{
				"200": jsonResponse("User banned", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("User not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/admin/users/{userId}/unban"] = &PathItem{
		Post: &Operation{
			Summary:     "Unban user (admin)",
			OperationID: "adminUnbanUser",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "userId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("User unbanned", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"404": jsonResponse("User not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/admin/impersonate/{userId}"] = &PathItem{
		Post: &Operation{
			Summary:     "Impersonate user (admin)",
			OperationID: "adminImpersonate",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Parameters: []Parameter{
				{Name: "userId", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]*Response{
				"200": jsonResponse("Impersonation session", &Schema{Ref: "#/components/schemas/AuthResponse"}),
				"403": jsonResponse("Forbidden", &Schema{Ref: "#/components/schemas/Error"}),
				"404": jsonResponse("User not found", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}

	spec.Paths["/v1/admin/impersonate/stop"] = &PathItem{
		Post: &Operation{
			Summary:     "Stop impersonation (admin)",
			OperationID: "adminStopImpersonation",
			Tags:        []string{"Admin"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("Impersonation stopped", &Schema{
					Type:       "object",
					Properties: map[string]*Schema{"status": {Type: "string"}},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addGDPRPaths(spec *Spec) {
	spec.Paths["/v1/me"].Delete = &Operation{
		Summary:     "Delete account (self-service)",
		Description: "Soft-deletes the authenticated user's account and anonymizes PII.",
		OperationID: "deleteAccount",
		Tags:        []string{"User"},
		Security:    []SecurityRequirement{{"bearerAuth": {}}},
		Responses: map[string]*Response{
			"200": jsonResponse("Account deleted", &Schema{
				Type:       "object",
				Properties: map[string]*Schema{"status": {Type: "string"}},
			}),
			"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
		},
	}

	spec.Paths["/v1/me/export"] = &PathItem{
		Get: &Operation{
			Summary:     "Export user data (GDPR)",
			Description: "Returns all data associated with the authenticated user.",
			OperationID: "exportUserData",
			Tags:        []string{"User"},
			Security:    []SecurityRequirement{{"bearerAuth": {}}},
			Responses: map[string]*Response{
				"200": jsonResponse("User data export", &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"user":     {Ref: "#/components/schemas/User"},
						"sessions": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Session"}},
						"devices":  {Type: "array", Items: &Schema{Ref: "#/components/schemas/Device"}},
						"extra":    {Type: "object", Description: "Plugin-contributed export data"},
					},
				}),
				"401": jsonResponse("Unauthorized", &Schema{Ref: "#/components/schemas/Error"}),
			},
		},
	}
}

func (g *Generator) addWellKnownPaths(spec *Spec) {
	spec.Paths["/.well-known/authsome/manifest"] = &PathItem{
		Get: &Operation{
			Summary:     "API manifest",
			OperationID: "getManifest",
			Tags:        []string{"System"},
			Security:    []SecurityRequirement{{}},
			Responses: map[string]*Response{
				"200": jsonResponse("API manifest", &Schema{Ref: "#/components/schemas/Manifest"}),
			},
		},
	}

	spec.Paths["/.well-known/authsome/openapi"] = &PathItem{
		Get: &Operation{
			Summary:     "OpenAPI specification",
			OperationID: "getOpenAPISpec",
			Tags:        []string{"System"},
			Security:    []SecurityRequirement{{}},
			Responses: map[string]*Response{
				"200": jsonResponse("OpenAPI 3.1 specification", nil),
			},
		},
	}
}

// ──────────────────────────────────────────────────
// Component builders
// ──────────────────────────────────────────────────

func (g *Generator) buildComponents() *Components {
	enabledPlugins := make(map[string]bool)
	for _, p := range g.config.EnabledPlugins {
		enabledPlugins[p] = true
	}

	schemas := map[string]*Schema{
		"User": {
			Type: "object",
			Properties: map[string]*Schema{
				"id":             {Type: "string"},
				"email":          {Type: "string", Format: "email"},
				"email_verified": {Type: "boolean"},
				"name":           {Type: "string"},
				"username":       {Type: "string"},
				"image":          {Type: "string", Format: "uri"},
				"phone":          {Type: "string"},
				"banned":         {Type: "boolean"},
				"created_at":     {Type: "string", Format: "date-time"},
				"updated_at":     {Type: "string", Format: "date-time"},
			},
			Required: []string{"id", "email"},
		},
		"Session": {
			Type: "object",
			Properties: map[string]*Schema{
				"id":         {Type: "string"},
				"user_id":    {Type: "string"},
				"token":      {Type: "string"},
				"expires_at": {Type: "string", Format: "date-time"},
				"created_at": {Type: "string", Format: "date-time"},
			},
		},
		"Device": {
			Type: "object",
			Properties: map[string]*Schema{
				"id":           {Type: "string"},
				"user_id":      {Type: "string"},
				"name":         {Type: "string"},
				"type":         {Type: "string", Description: "Device type (mobile, desktop, tablet, etc.)"},
				"fingerprint":  {Type: "string"},
				"ip_address":   {Type: "string"},
				"user_agent":   {Type: "string"},
				"last_used_at": {Type: "string", Format: "date-time"},
				"created_at":   {Type: "string", Format: "date-time"},
			},
		},
		"APIKey": {
			Type: "object",
			Properties: map[string]*Schema{
				"id":           {Type: "string"},
				"app_id":       {Type: "string"},
				"name":         {Type: "string"},
				"key_prefix":   {Type: "string", Description: "First 8 characters of the key for identification"},
				"scopes":       {Type: "array", Items: &Schema{Type: "string"}},
				"last_used_at": {Type: "string", Format: "date-time"},
				"expires_at":   {Type: "string", Format: "date-time"},
				"created_at":   {Type: "string", Format: "date-time"},
			},
		},
		"MFAEnrollment": {
			Type: "object",
			Properties: map[string]*Schema{
				"id":          {Type: "string"},
				"method":      {Type: "string", Description: "MFA method (totp or sms)"},
				"secret":      {Type: "string", Description: "Base32-encoded TOTP secret"},
				"otpauth_url": {Type: "string", Format: "uri", Description: "OTPAuth URL for QR code"},
			},
		},
		"AuthResponse": {
			Type: "object",
			Properties: map[string]*Schema{
				"user":          {Ref: "#/components/schemas/User"},
				"session_token": {Type: "string"},
				"refresh_token": {Type: "string"},
			},
			Required: []string{"user", "session_token", "refresh_token"},
		},
		"TokenResponse": {
			Type: "object",
			Properties: map[string]*Schema{
				"session_token": {Type: "string"},
				"refresh_token": {Type: "string"},
				"expires_at":    {Type: "string", Format: "date-time"},
			},
			Required: []string{"session_token", "refresh_token", "expires_at"},
		},
		"Error": {
			Type: "object",
			Properties: map[string]*Schema{
				"error": {Type: "string"},
				"code":  {Type: "integer", Description: "HTTP status code"},
			},
			Required: []string{"error"},
		},
		"Manifest": {
			Type: "object",
			Properties: map[string]*Schema{
				"version":   {Type: "string"},
				"base_path": {Type: "string"},
				"endpoints": {Type: "array", Items: &Schema{Type: "object"}},
				"features":  {Type: "object"},
			},
		},
	}

	// Organization schemas are only included when the organization plugin is enabled.
	if enabledPlugins["organization"] {
		schemas["Organization"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":         {Type: "string"},
				"app_id":     {Type: "string"},
				"name":       {Type: "string"},
				"slug":       {Type: "string"},
				"logo":       {Type: "string", Format: "uri"},
				"metadata":   {Type: "object", Description: "Custom metadata"},
				"created_at": {Type: "string", Format: "date-time"},
				"updated_at": {Type: "string", Format: "date-time"},
			},
		}
		schemas["Member"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":         {Type: "string"},
				"org_id":     {Type: "string"},
				"user_id":    {Type: "string"},
				"role":       {Type: "string", Description: "Member role (owner, admin, member)"},
				"created_at": {Type: "string", Format: "date-time"},
				"updated_at": {Type: "string", Format: "date-time"},
			},
		}
		schemas["Invitation"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id":         {Type: "string"},
				"org_id":     {Type: "string"},
				"email":      {Type: "string", Format: "email"},
				"role":       {Type: "string"},
				"token":      {Type: "string"},
				"status":     {Type: "string", Description: "Invitation status (pending, accepted, expired)"},
				"expires_at": {Type: "string", Format: "date-time"},
				"created_at": {Type: "string", Format: "date-time"},
			},
		}
	}

	return &Components{
		Schemas: schemas,
		SecuritySchemes: map[string]*SecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "token",
				Description:  "Session token from sign-in",
			},
			"apiKeyAuth": {
				Type:        "apiKey",
				In:          "header",
				Name:        "X-API-Key",
				Description: "API key for machine-to-machine authentication",
			},
		},
	}
}

func (g *Generator) buildTags() []Tag {
	tags := []Tag{
		{Name: "Authentication", Description: "Sign up, sign in, sign out, refresh"},
		{Name: "Password", Description: "Password management: forgot, reset, change, verify email"},
		{Name: "User", Description: "User profile operations"},
		{Name: "Sessions", Description: "Session management"},
		{Name: "Devices", Description: "Device tracking and management"},
		{Name: "System", Description: "Health checks and API metadata"},
	}

	enabledPlugins := make(map[string]bool)
	for _, p := range g.config.EnabledPlugins {
		enabledPlugins[p] = true
	}

	if enabledPlugins["organization"] {
		tags = append(tags, Tag{Name: "Organizations", Description: "Organization, member, and invitation management"})
	}
	if enabledPlugins["social"] {
		tags = append(tags, Tag{Name: "Social", Description: "Social OAuth providers"})
	}
	if enabledPlugins["magiclink"] {
		tags = append(tags, Tag{Name: "Magic Link", Description: "Passwordless magic link authentication"})
	}
	if enabledPlugins["mfa"] {
		tags = append(tags, Tag{Name: "MFA", Description: "Multi-factor authentication"})
	}
	if enabledPlugins["apikey"] {
		tags = append(tags, Tag{Name: "API Keys", Description: "API key management for machine-to-machine authentication"})
	}
	if enabledPlugins["sso"] {
		tags = append(tags, Tag{Name: "SSO", Description: "Enterprise SSO via SAML 2.0 and OIDC"})
	}

	tags = append(tags, Tag{Name: "Admin", Description: "Admin operations: user management, impersonation"})

	return tags
}

// ──────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────

func jsonBody(schema *Schema) *RequestBody {
	return &RequestBody{
		Required: true,
		Content: map[string]MediaType{
			"application/json": {Schema: schema},
		},
	}
}

func jsonResponse(description string, schema *Schema) *Response {
	resp := &Response{Description: description}
	if schema != nil {
		resp.Content = map[string]MediaType{
			"application/json": {Schema: schema},
		}
	}
	return resp
}
