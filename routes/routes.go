package routes

import (
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/forge"
)

// Register registers auth routes using forge.Router
// authMiddleware is applied to all routes to extract and validate API keys for app identification.
func Register(router forge.Router, basePath string, h *handlers.AuthHandler, authMiddleware forge.Middleware) {
	authGroup := router.Group(basePath)

	// Apply middleware at group level if provided
	if authMiddleware != nil {
		authGroup.Use(authMiddleware)
	}

	// User registration
	authGroup.POST("/signup", h.SignUp,
		forge.WithName("auth.signup"),
		forge.WithSummary("User registration"),
		forge.WithDescription("Register a new user account with email and password"),
		forge.WithRequestSchema(auth.SignUpRequest{}),
		forge.WithResponseSchema(200, "Registration successful", responses.AuthResponse{}),
		forge.WithResponseSchema(400, "Invalid request or registration failed", ErrorResponse{}),
		forge.WithResponseSchema(403, "IP or geo-restriction", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication"),
		forge.WithValidation(true),
	)

	// User authentication
	authGroup.POST("/signin", h.SignIn,
		forge.WithName("auth.signin"),
		forge.WithSummary("User sign in"),
		forge.WithDescription("Authenticate a user with email and password. May require 2FA verification if enabled."),
		forge.WithRequestSchema(auth.SignInRequest{}),
		forge.WithResponseSchema(200, "Sign in successful", responses.AuthResponse{}),
		forge.WithResponseSchema(401, "Invalid credentials", ErrorResponse{}),
		forge.WithResponseSchema(403, "IP or geo-restriction", ErrorResponse{}),
		forge.WithResponseSchema(423, "Account temporarily locked", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication"),
		forge.WithValidation(true),
	)

	// User sign out
	authGroup.POST("/signout", h.SignOut,
		forge.WithName("auth.signout"),
		forge.WithSummary("User sign out"),
		forge.WithDescription("Sign out a user by invalidating their session token"),
		forge.WithRequestSchema(SignOutRequest{}),
		forge.WithResponseSchema(200, "Sign out successful", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid token or sign out failed", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication"),
		forge.WithValidation(true),
	)

	// Refresh session
	authGroup.POST("/refresh", h.RefreshSession,
		forge.WithName("auth.refresh"),
		forge.WithSummary("Refresh session"),
		forge.WithDescription("Refresh an access token using a refresh token (long-lived session pattern)"),
		forge.WithRequestSchema(RefreshSessionRequest{}),
		forge.WithResponseSchema(200, "Session refreshed", RefreshSessionResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid or expired refresh token", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication"),
		forge.WithValidation(true),
	)

	// Get current session
	authGroup.GET("/session", h.GetSession,
		forge.WithName("auth.session"),
		forge.WithSummary("Get current session"),
		forge.WithDescription("Retrieve the current user session and profile information"),
		forge.WithResponseSchema(200, "Session retrieved", SessionResponse{}),
		forge.WithResponseSchema(401, "Not authenticated or invalid session", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Session"),
	)

	// List user devices
	authGroup.GET("/devices", h.ListDevices,
		forge.WithName("auth.devices.list"),
		forge.WithSummary("List user devices"),
		forge.WithDescription("List all devices associated with the authenticated user"),
		forge.WithResponseSchema(200, "Devices retrieved", DevicesResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Devices"),
	)

	// Revoke device
	authGroup.POST("/devices/revoke", h.RevokeDevice,
		forge.WithName("auth.devices.revoke"),
		forge.WithSummary("Revoke a device"),
		forge.WithDescription("Remove a device from the authenticated user's trusted devices"),
		forge.WithRequestSchema(RevokeDeviceRequest{}),
		forge.WithResponseSchema(200, "Device revoked", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Devices"),
		forge.WithValidation(true),
	)

	// Update user profile
	authGroup.POST("/user/update", h.UpdateUser,
		forge.WithName("auth.user.update"),
		forge.WithSummary("Update user profile"),
		forge.WithDescription("Update the authenticated user's profile information (name, image, username)"),
		forge.WithRequestSchema(UpdateUserRequest{}),
		forge.WithResponseSchema(200, "User updated", user.User{}),
		forge.WithResponseSchema(400, "Invalid request or update failed", ErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "User"),
		forge.WithValidation(true),
	)

	// Password reset - Request reset link
	authGroup.POST("/password/reset/request", h.RequestPasswordReset,
		forge.WithName("auth.password.reset.request"),
		forge.WithSummary("Request password reset"),
		forge.WithDescription("Request a password reset link to be sent to the user's email"),
		forge.WithRequestSchema(PasswordResetRequestDTO{}),
		forge.WithResponseSchema(200, "Reset link sent (if email exists)", PasswordResetRequestResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Password"),
		forge.WithValidation(true),
	)

	// Password reset - Confirm with token
	authGroup.POST("/password/reset/confirm", h.ResetPassword,
		forge.WithName("auth.password.reset.confirm"),
		forge.WithSummary("Reset password"),
		forge.WithDescription("Reset password using a valid reset token"),
		forge.WithRequestSchema(PasswordResetConfirmDTO{}),
		forge.WithResponseSchema(200, "Password reset successful", PasswordResetResponse{}),
		forge.WithResponseSchema(400, "Invalid or expired token", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Password"),
		forge.WithValidation(true),
	)

	// Password reset - Validate token
	authGroup.GET("/password/reset/validate", h.ValidateResetToken,
		forge.WithName("auth.password.reset.validate"),
		forge.WithSummary("Validate reset token"),
		forge.WithDescription("Check if a password reset token is valid"),
		forge.WithResponseSchema(200, "Token validation result", TokenValidationResponse{}),
		forge.WithResponseSchema(400, "Missing token parameter", ErrorResponse{}),
		forge.WithTags("Authentication", "Password"),
	)

	// Change password (requires authentication)
	authGroup.POST("/password/change", h.ChangePassword,
		forge.WithName("auth.password.change"),
		forge.WithSummary("Change password"),
		forge.WithDescription("Change the authenticated user's password"),
		forge.WithRequestSchema(ChangePasswordDTO{}),
		forge.WithResponseSchema(200, "Password changed successfully", PasswordChangeResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Incorrect current password or not authenticated", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Password"),
		forge.WithValidation(true),
	)

	// Email change - Request
	authGroup.POST("/email/change/request", h.RequestEmailChange,
		forge.WithName("auth.email.change.request"),
		forge.WithSummary("Request email change"),
		forge.WithDescription("Request to change email address with confirmation"),
		forge.WithRequestSchema(EmailChangeRequestDTO{}),
		forge.WithResponseSchema(200, "Confirmation sent", EmailChangeRequestResponse{}),
		forge.WithResponseSchema(400, "Invalid request or email taken", ErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Authentication", "Email"),
		forge.WithValidation(true),
	)

	// Email change - Confirm
	authGroup.POST("/email/change/confirm", h.ConfirmEmailChange,
		forge.WithName("auth.email.change.confirm"),
		forge.WithSummary("Confirm email change"),
		forge.WithDescription("Confirm email change using a valid token"),
		forge.WithRequestSchema(EmailChangeConfirmDTO{}),
		forge.WithResponseSchema(200, "Email changed successfully", EmailChangeResponse{}),
		forge.WithResponseSchema(400, "Invalid or expired token", ErrorResponse{}),
		forge.WithTags("Authentication", "Email"),
		forge.WithValidation(true),
	)
}

// DTOs for auth routes

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

// StatusResponse represents a status response.
type StatusResponse struct {
	Status string `example:"success" json:"status"`
}

// SignOutRequest represents a sign out request.
type SignOutRequest struct {
	Token string `example:"session_token_here" json:"token" validate:"required"`
}

type RefreshSessionRequest struct {
	RefreshToken string `example:"refresh_token_here" json:"refreshToken" validate:"required"`
}

type RefreshSessionResponse struct {
	Session          any    `json:"session"`
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        string `json:"expiresAt"`
	RefreshExpiresAt string `json:"refreshExpiresAt"`
}

// SessionResponse represents session information.
type SessionResponse struct {
	User    *user.User     `json:"user"`
	Session map[string]any `json:"session"`
}

// DevicesResponse represents a list of devices.
type DevicesResponse []device.Device

// RevokeDeviceRequest represents a device revocation request.
type RevokeDeviceRequest struct {
	Fingerprint string `example:"device_fingerprint_here" json:"fingerprint" validate:"required"`
}

// UpdateUserRequest represents a user update request.
type UpdateUserRequest struct {
	Name            *string `example:"John Doe"                       json:"name,omitempty"`
	Image           *string `example:"https://example.com/avatar.jpg" json:"image,omitempty"`
	Username        *string `example:"johndoe"                        json:"username,omitempty"`
	DisplayUsername *string `example:"John D."                        json:"display_username,omitempty"`
}

// PasswordResetRequestDTO represents a password reset request.
type PasswordResetRequestDTO struct {
	Email string `example:"user@example.com" json:"email" validate:"required,email"`
}

// PasswordResetRequestResponse represents the response to a password reset request.
type PasswordResetRequestResponse struct {
	Message string `example:"If the email exists, a password reset link has been sent" json:"message"`
}

// PasswordResetConfirmDTO represents a password reset confirmation.
type PasswordResetConfirmDTO struct {
	Token       string `example:"reset_token_here"  json:"token"       validate:"required"`
	NewPassword string `example:"NewSecurePass123!" json:"newPassword" validate:"required,min=8"`
}

// PasswordResetResponse represents the response to a password reset confirmation.
type PasswordResetResponse struct {
	Message string `example:"Password has been reset successfully" json:"message"`
}

// TokenValidationResponse represents a token validation response.
type TokenValidationResponse struct {
	Valid bool `example:"true" json:"valid"`
}

// ChangePasswordDTO represents a password change request.
type ChangePasswordDTO struct {
	OldPassword string `example:"CurrentPass123!"   json:"oldPassword" validate:"required"`
	NewPassword string `example:"NewSecurePass123!" json:"newPassword" validate:"required,min=8"`
}

// PasswordChangeResponse represents the response to a password change.
type PasswordChangeResponse struct {
	Message string `example:"Password changed successfully" json:"message"`
}

// EmailChangeRequestDTO represents an email change request.
type EmailChangeRequestDTO struct {
	NewEmail string `example:"newemail@example.com" json:"newEmail" validate:"required,email"`
}

// EmailChangeRequestResponse represents the response to an email change request.
type EmailChangeRequestResponse struct {
	Message string `example:"Email change confirmation sent to your current email address" json:"message"`
}

// EmailChangeConfirmDTO represents an email change confirmation.
type EmailChangeConfirmDTO struct {
	Token string `example:"change_token_here" json:"token" validate:"required"`
}

// EmailChangeResponse represents the response to an email change confirmation.
type EmailChangeResponse struct {
	Message string `example:"Email address has been changed successfully" json:"message"`
}

// RegisterAudit registers audit routes under a base path.
func RegisterAudit(router forge.Router, basePath string, h *handlers.AuditHandler, authMiddleware forge.Middleware) {
	grp := router.Group(basePath)

	// Apply middleware at group level if provided
	if authMiddleware != nil {
		grp.Use(authMiddleware)
	}

	grp.GET("/audit/events", h.ListEvents,
		forge.WithName("audit.events.list"),
		forge.WithSummary("List audit events"),
		forge.WithDescription("Retrieve paginated audit events with optional filters (userId, action, since, until)"),
		forge.WithResponseSchema(200, "Audit events retrieved", AuditEventsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit"),
	)

	// Aggregation endpoints
	grp.GET("/audit/aggregations", h.GetAggregations,
		forge.WithName("audit.aggregations.all"),
		forge.WithSummary("Get all aggregations"),
		forge.WithDescription("Retrieve all distinct values with counts in one call (actions, sources, resources, users, IPs, apps, organizations)"),
		forge.WithResponseSchema(200, "Aggregations retrieved", AggregationsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/actions", h.GetDistinctActions,
		forge.WithName("audit.aggregations.actions"),
		forge.WithSummary("Get distinct actions"),
		forge.WithDescription("Retrieve distinct action values with counts for filter UIs"),
		forge.WithResponseSchema(200, "Distinct actions retrieved", ActionsAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/sources", h.GetDistinctSources,
		forge.WithName("audit.aggregations.sources"),
		forge.WithSummary("Get distinct sources"),
		forge.WithDescription("Retrieve distinct source values with counts"),
		forge.WithResponseSchema(200, "Distinct sources retrieved", SourcesAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/resources", h.GetDistinctResources,
		forge.WithName("audit.aggregations.resources"),
		forge.WithSummary("Get distinct resources"),
		forge.WithDescription("Retrieve distinct resource values with counts"),
		forge.WithResponseSchema(200, "Distinct resources retrieved", ResourcesAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/users", h.GetDistinctUsers,
		forge.WithName("audit.aggregations.users"),
		forge.WithSummary("Get distinct users"),
		forge.WithDescription("Retrieve distinct user values with counts"),
		forge.WithResponseSchema(200, "Distinct users retrieved", UsersAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/ips", h.GetDistinctIPs,
		forge.WithName("audit.aggregations.ips"),
		forge.WithSummary("Get distinct IP addresses"),
		forge.WithDescription("Retrieve distinct IP address values with counts"),
		forge.WithResponseSchema(200, "Distinct IPs retrieved", IPsAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/apps", h.GetDistinctApps,
		forge.WithName("audit.aggregations.apps"),
		forge.WithSummary("Get distinct apps"),
		forge.WithDescription("Retrieve distinct app values with counts"),
		forge.WithResponseSchema(200, "Distinct apps retrieved", AppsAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)

	grp.GET("/audit/organizations", h.GetDistinctOrganizations,
		forge.WithName("audit.aggregations.organizations"),
		forge.WithSummary("Get distinct organizations"),
		forge.WithDescription("Retrieve distinct organization values with counts"),
		forge.WithResponseSchema(200, "Distinct organizations retrieved", OrganizationsAggregationResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithResponseSchema(501, "Audit service not available", ErrorResponse{}),
		forge.WithTags("Audit", "Aggregations"),
	)
}

// AuditEventsResponse represents a paginated list of audit events.
type AuditEventsResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// Aggregation response types.
type AggregationsResponse struct{}
type ActionsAggregationResponse struct{}
type SourcesAggregationResponse struct{}
type ResourcesAggregationResponse struct{}
type UsersAggregationResponse struct{}
type IPsAggregationResponse struct{}
type AppsAggregationResponse struct{}
type OrganizationsAggregationResponse struct{}

// RegisterApp registers app (platform tenant) routes under a base path
// This is used when multitenancy plugin is NOT enabled.
func RegisterApp(router forge.Router, basePath string, h *handlers.AppHandler, authMiddleware forge.Middleware) {
	org := router.Group(basePath)

	// Apply middleware at group level if provided
	if authMiddleware != nil {
		org.Use(authMiddleware)
	}

	// Apps
	org.POST("/", h.CreateApp,
		forge.WithName("apps.create"),
		forge.WithSummary("Create app"),
		forge.WithDescription("Create a new app"),
		forge.WithResponseSchema(200, "App created", AppResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps"),
		forge.WithValidation(true),
	)

	org.GET("/", h.GetApps,
		forge.WithName("apps.list"),
		forge.WithSummary("List apps"),
		forge.WithDescription("List all apps accessible to the user"),
		forge.WithResponseSchema(200, "Organizations retrieved", AppsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps"),
	)

	org.POST("/update", h.UpdateApp,
		forge.WithName("apps.update"),
		forge.WithSummary("Update app"),
		forge.WithDescription("Update app details"),
		forge.WithResponseSchema(200, "App updated", AppResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps"),
		forge.WithValidation(true),
	)

	org.POST("/delete", h.DeleteApp,
		forge.WithName("apps.delete"),
		forge.WithSummary("Delete app"),
		forge.WithDescription("Delete an app"),
		forge.WithResponseSchema(200, "App deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps"),
	)

	// Members
	org.POST("/members", h.CreateMember,
		forge.WithName("apps.members.create"),
		forge.WithSummary("Add app member"),
		forge.WithDescription("Add a new member to the app"),
		forge.WithResponseSchema(200, "Member added", MemberResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Members"),
		forge.WithValidation(true),
	)

	org.GET("/members", h.GetMembers,
		forge.WithName("apps.members.list"),
		forge.WithSummary("List app members"),
		forge.WithDescription("List all members of the app"),
		forge.WithResponseSchema(200, "Members retrieved", MembersResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "Members"),
	)

	org.POST("/members/update", h.UpdateMember,
		forge.WithName("apps.members.update"),
		forge.WithSummary("Update member"),
		forge.WithDescription("Update app member details"),
		forge.WithResponseSchema(200, "Member updated", MemberResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Members"),
		forge.WithValidation(true),
	)

	org.POST("/members/delete", h.DeleteMember,
		forge.WithName("apps.members.delete"),
		forge.WithSummary("Remove member"),
		forge.WithDescription("Remove a member from the app"),
		forge.WithResponseSchema(200, "Member removed", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Members"),
	)

	// Teams
	org.POST("/teams", h.CreateTeam,
		forge.WithName("apps.teams.create"),
		forge.WithSummary("Create team"),
		forge.WithDescription("Create a new team within the app"),
		forge.WithResponseSchema(200, "Team created", TeamResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
		forge.WithValidation(true),
	)

	org.GET("/teams", h.GetTeams,
		forge.WithName("apps.teams.list"),
		forge.WithSummary("List teams"),
		forge.WithDescription("List all teams in the app"),
		forge.WithResponseSchema(200, "Teams retrieved", TeamsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
	)

	org.POST("/teams/update", h.UpdateTeam,
		forge.WithName("apps.teams.update"),
		forge.WithSummary("Update team"),
		forge.WithDescription("Update team details"),
		forge.WithResponseSchema(200, "Team updated", TeamResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
		forge.WithValidation(true),
	)

	org.POST("/teams/delete", h.DeleteTeam,
		forge.WithName("apps.teams.delete"),
		forge.WithSummary("Delete team"),
		forge.WithDescription("Delete a team from the app"),
		forge.WithResponseSchema(200, "Team deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
	)

	// Team members
	org.POST("/team_members", h.AddTeamMember,
		forge.WithName("apps.teams.members.add"),
		forge.WithSummary("Add team member"),
		forge.WithDescription("Add a member to a team"),
		forge.WithResponseSchema(200, "Team member added", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
		forge.WithValidation(true),
	)

	org.GET("/team_members", h.GetTeamMembers,
		forge.WithName("apps.teams.members.list"),
		forge.WithSummary("List team members"),
		forge.WithDescription("List all members of a team"),
		forge.WithResponseSchema(200, "Team members retrieved", TeamMembersResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
	)

	org.POST("/team_members/remove", h.RemoveTeamMember,
		forge.WithName("apps.teams.members.remove"),
		forge.WithSummary("Remove team member"),
		forge.WithDescription("Remove a member from a team"),
		forge.WithResponseSchema(200, "Team member removed", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Teams"),
	)

	// Invitations
	org.POST("/invitations", h.CreateInvitation,
		forge.WithName("apps.invitations.create"),
		forge.WithSummary("Create invitation"),
		forge.WithDescription("Invite a user to join the app"),
		forge.WithResponseSchema(200, "Invitation created", InvitationResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "Invitations"),
		forge.WithValidation(true),
	)

	// App by ID endpoints (registered after specific paths to avoid conflicts)
	org.GET("/{appId}", h.GetAppByID,
		forge.WithName("apps.get"),
		forge.WithSummary("Get app by ID"),
		forge.WithDescription("Retrieve a specific app by ID"),
		forge.WithResponseSchema(200, "App retrieved", AppResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps"),
	)

	org.POST("/{appId}/update", h.UpdateAppByID,
		forge.WithName("apps.update.byid"),
		forge.WithSummary("Update app by ID"),
		forge.WithDescription("Update a specific app by ID"),
		forge.WithResponseSchema(200, "App updated", AppResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps"),
		forge.WithValidation(true),
	)

	org.POST("/{appId}/delete", h.DeleteAppByID,
		forge.WithName("apps.delete.byid"),
		forge.WithSummary("Delete app by ID"),
		forge.WithDescription("Delete a specific app by ID"),
		forge.WithResponseSchema(200, "App deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps"),
	)

	// Cookie configuration endpoints
	org.GET("/{appId}/cookie-config", h.GetAppCookieConfig,
		forge.WithName("apps.cookie_config.get"),
		forge.WithSummary("Get app cookie configuration"),
		forge.WithDescription("Retrieve the cookie configuration for a specific app (merged with global defaults)"),
		forge.WithResponseSchema(200, "Cookie configuration retrieved", CookieConfigResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps", "Configuration"),
	)

	org.PUT("/{appId}/cookie-config", h.UpdateAppCookieConfig,
		forge.WithName("apps.cookie_config.update"),
		forge.WithSummary("Update app cookie configuration"),
		forge.WithDescription("Set or update the cookie configuration for a specific app"),
		forge.WithRequestSchema(CookieConfigRequest{}),
		forge.WithResponseSchema(200, "Cookie configuration updated", CookieConfigUpdateResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps", "Configuration"),
		forge.WithValidation(true),
	)

	org.DELETE("/{appId}/cookie-config", h.DeleteAppCookieConfig,
		forge.WithName("apps.cookie_config.delete"),
		forge.WithSummary("Delete app cookie configuration"),
		forge.WithDescription("Remove app-specific cookie configuration, reverting to global defaults"),
		forge.WithResponseSchema(200, "Cookie configuration deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", ErrorResponse{}),
		forge.WithTags("Apps", "Configuration"),
	)

	// RBAC routes
	RegisterAppRBAC(org, h)
}

// App DTOs (placeholder types - actual implementations should be in handlers or core).
type AppResponse struct{}
type AppsResponse []any
type MemberResponse struct{}
type MembersResponse []any
type TeamResponse struct{}
type TeamsResponse []any
type TeamMembersResponse []any
type InvitationResponse struct{}
type CookieConfigResponse struct{}
type CookieConfigRequest struct{}
type CookieConfigUpdateResponse struct{}

// RegisterAppRBAC registers RBAC-related routes (policies, roles, user roles)
// This is used when multitenancy plugin IS enabled to supplement its routes
// Note: These routes don't apply middleware as they're nested under RegisterApp which already applies it.
func RegisterAppRBAC(router forge.Router, h *handlers.AppHandler) {
	// Policies
	router.POST("/policies", h.CreatePolicy,
		forge.WithName("apps.policies.create"),
		forge.WithSummary("Create policy"),
		forge.WithDescription("Create a new RBAC policy for the app"),
		forge.WithResponseSchema(200, "Policy created", PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Policies"),
		forge.WithValidation(true),
	)

	router.GET("/policies", h.GetPolicies,
		forge.WithName("apps.policies.list"),
		forge.WithSummary("List policies"),
		forge.WithDescription("List all RBAC policies for the app"),
		forge.WithResponseSchema(200, "Policies retrieved", PoliciesResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Policies"),
	)

	router.POST("/policies/delete", h.DeletePolicy,
		forge.WithName("apps.policies.delete"),
		forge.WithSummary("Delete policy"),
		forge.WithDescription("Delete an RBAC policy"),
		forge.WithResponseSchema(200, "Policy deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Policies"),
	)

	router.POST("/policies/update", h.UpdatePolicy,
		forge.WithName("apps.policies.update"),
		forge.WithSummary("Update policy"),
		forge.WithDescription("Update an existing RBAC policy"),
		forge.WithResponseSchema(200, "Policy updated", PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Policies"),
		forge.WithValidation(true),
	)

	// Roles
	router.POST("/roles", h.CreateRole,
		forge.WithName("apps.roles.create"),
		forge.WithSummary("Create role"),
		forge.WithDescription("Create a new RBAC role for the app"),
		forge.WithResponseSchema(200, "Role created", RoleResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Roles"),
		forge.WithValidation(true),
	)

	router.GET("/roles", h.GetRoles,
		forge.WithName("apps.roles.list"),
		forge.WithSummary("List roles"),
		forge.WithDescription("List all RBAC roles for the app"),
		forge.WithResponseSchema(200, "Roles retrieved", RolesResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Roles"),
	)

	// User role assignments
	router.POST("/user_roles/assign", h.AssignUserRole,
		forge.WithName("apps.user_roles.assign"),
		forge.WithSummary("Assign user role"),
		forge.WithDescription("Assign an RBAC role to a user"),
		forge.WithResponseSchema(200, "Role assigned", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Roles"),
		forge.WithValidation(true),
	)

	router.POST("/user_roles/remove", h.RemoveUserRole,
		forge.WithName("apps.user_roles.remove"),
		forge.WithSummary("Remove user role"),
		forge.WithDescription("Remove an RBAC role from a user"),
		forge.WithResponseSchema(200, "Role removed", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Roles"),
	)

	router.GET("/user_roles", h.GetUserRoles,
		forge.WithName("apps.user_roles.list"),
		forge.WithSummary("List user roles"),
		forge.WithDescription("List all roles assigned to users in the app"),
		forge.WithResponseSchema(200, "User roles retrieved", UserRolesResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
		forge.WithTags("Apps", "RBAC", "Roles"),
	)
}

// RBAC DTOs.
type PolicyResponse struct{}
type PoliciesResponse []any
type RoleResponse struct{}
type RolesResponse []any
type UserRolesResponse []any

// RegisterAPIKey is DEPRECATED and removed.
// API key routes are now handled by the apikey plugin.
// Use: auth.RegisterPlugin(apikey.NewPlugin())

// RegisterJWT is DEPRECATED - JWT routes are now handled by the JWT plugin.
// The JWT plugin registers its own routes via plugin.RegisterRoutes().
// Use: auth.RegisterPlugin(jwt.NewPlugin()).
func RegisterJWT(router forge.Router, basePath string, h any) {
	// DEPRECATED: This function is kept for backwards compatibility but does nothing.
	// JWT routes are now registered by the JWT plugin itself.
}

// RegisterWebhook registers webhook routes under a base path.
func RegisterWebhook(router forge.Router, basePath string, h *handlers.WebhookHandler) {
	grp := router.Group(basePath)
	RegisterWebhookRoutes(grp, h)
}

// RegisterNotification is DEPRECATED and removed.
// Notification routes are now handled by the notification plugin.
// Use: auth.RegisterPlugin(notification.NewPlugin())
