package api

import "github.com/xraph/authsome/environment"

// ---------------------------------------------------------------------------
// Auth requests
// ---------------------------------------------------------------------------

// SignUpRequest binds the body for POST /signup.
type SignUpRequest struct {
	AppID    string `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Email    string `json:"email" description:"User email address"`
	Password string `json:"password" description:"User password"`
	FirstName string `json:"first_name,omitempty" description:"First/given name"`
	LastName  string `json:"last_name,omitempty" description:"Last/family name"`
	Username  string `json:"username,omitempty" description:"Unique username"`
}

// SignInRequest binds the body for POST /signin.
type SignInRequest struct {
	AppID    string `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Email    string `json:"email,omitempty" description:"User email address"`
	Username string `json:"username,omitempty" description:"Username (alternative to email)"`
	Password string `json:"password" description:"User password"`
}

// SignOutRequest is an empty request for POST /signout (session from context).
type SignOutRequest struct{}

// RefreshRequest binds the body for POST /refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" description:"Refresh token to exchange for new tokens"`
}

// ---------------------------------------------------------------------------
// Password management requests
// ---------------------------------------------------------------------------

// ForgotPasswordRequest binds the body for POST /forgot-password.
type ForgotPasswordRequest struct {
	AppID string `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Email string `json:"email" description:"Email address of the account"`
}

// ResetPasswordRequest binds the body for POST /reset-password.
type ResetPasswordRequest struct {
	Token       string `json:"token" description:"Password reset token"`
	NewPassword string `json:"new_password" description:"New password"`
}

// ChangePasswordRequest binds the body for POST /change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" description:"Current password"`
	NewPassword     string `json:"new_password" description:"New password"`
}

// VerifyEmailRequest binds the body for POST /verify-email.
type VerifyEmailRequest struct {
	Token string `json:"token" description:"Email verification token"`
}

// ---------------------------------------------------------------------------
// User requests
// ---------------------------------------------------------------------------

// GetMeRequest is an empty request for GET /me (user from context).
type GetMeRequest struct{}

// UpdateMeRequest binds the body for PATCH /me.
type UpdateMeRequest struct {
	FirstName *string `json:"first_name,omitempty" description:"First/given name"`
	LastName  *string `json:"last_name,omitempty" description:"Last/family name"`
	Image     *string `json:"image,omitempty" description:"Profile image URL"`
	Username *string `json:"username,omitempty" description:"Unique username"`
}

// ---------------------------------------------------------------------------
// Session requests
// ---------------------------------------------------------------------------

// ListSessionsRequest is an empty request for GET /sessions (user from context).
type ListSessionsRequest struct{}

// RevokeSessionRequest binds the path for DELETE /sessions/:sessionId.
type RevokeSessionRequest struct {
	SessionID string `path:"sessionId" description:"Session identifier"`
}

// ---------------------------------------------------------------------------
// Device requests
// ---------------------------------------------------------------------------

// ListDevicesRequest is an empty request for GET /devices (user from context).
type ListDevicesRequest struct{}

// GetDeviceRequest binds the path for GET /devices/:deviceId.
type GetDeviceRequest struct {
	DeviceID string `path:"deviceId" description:"Device identifier"`
}

// DeleteDeviceRequest binds the path for DELETE /devices/:deviceId.
type DeleteDeviceRequest struct {
	DeviceID string `path:"deviceId" description:"Device identifier"`
}

// TrustDeviceRequest binds the path for PATCH /devices/:deviceId/trust.
type TrustDeviceRequest struct {
	DeviceID string `path:"deviceId" description:"Device identifier"`
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// AuthResponse is the standard response for signup/signin.
type AuthResponse struct {
	User         any    `json:"user" description:"User object"`
	SessionToken string `json:"session_token" description:"JWT session token"`
	RefreshToken string `json:"refresh_token" description:"Refresh token"`
	ExpiresAt    string `json:"expires_at" description:"Token expiration time"`
}

// TokenResponse is the response for token refresh.
type TokenResponse struct {
	SessionToken string `json:"session_token" description:"New session token"`
	RefreshToken string `json:"refresh_token" description:"New refresh token"`
	ExpiresAt    string `json:"expires_at" description:"Token expiration time"`
}

// StatusResponse is a generic status response.
type StatusResponse struct {
	Status string `json:"status" description:"Operation status"`
}

// SessionListResponse wraps a list of sessions.
type SessionListResponse struct {
	Sessions any `json:"sessions" description:"List of sessions"`
}

// DeviceListResponse wraps a list of devices.
type DeviceListResponse struct {
	Devices any `json:"devices" description:"List of devices"`
}

// ---------------------------------------------------------------------------
// Webhook requests
// ---------------------------------------------------------------------------

// CreateWebhookRequest binds the body for POST /webhooks.
type CreateWebhookRequest struct {
	AppID  string   `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	URL    string   `json:"url" description:"Webhook endpoint URL"`
	Events []string `json:"events" description:"Event types to subscribe to"`
}

// ListWebhooksRequest binds query params for GET /webhooks.
type ListWebhooksRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// GetWebhookRequest binds the path for GET /webhooks/:webhookId.
type GetWebhookRequest struct {
	WebhookID string `path:"webhookId" description:"Webhook identifier"`
}

// UpdateWebhookRequest binds path + body for PATCH /webhooks/:webhookId.
type UpdateWebhookRequest struct {
	WebhookID string   `path:"webhookId" description:"Webhook identifier"`
	URL       *string  `json:"url,omitempty" description:"Webhook endpoint URL"`
	Events    []string `json:"events,omitempty" description:"Event types to subscribe to"`
	Active    *bool    `json:"active,omitempty" description:"Whether the webhook is active"`
}

// DeleteWebhookRequest binds the path for DELETE /webhooks/:webhookId.
type DeleteWebhookRequest struct {
	WebhookID string `path:"webhookId" description:"Webhook identifier"`
}

// WebhookListResponse wraps a list of webhooks.
type WebhookListResponse struct {
	Webhooks any `json:"webhooks" description:"List of webhooks"`
}

// HealthResponse is the response for the health check endpoint.
type HealthResponse struct {
	Status string `json:"status" description:"Service health status"`
	Error  string `json:"error,omitempty" description:"Error details if unhealthy"`
}

// ---------------------------------------------------------------------------
// RBAC requests
// ---------------------------------------------------------------------------

// CreateRoleRequest binds the body for POST /roles.
type CreateRoleRequest struct {
	AppID       string `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Name        string `json:"name" description:"Role name"`
	Slug        string `json:"slug" description:"URL-safe role slug"`
	Description string `json:"description,omitempty" description:"Role description"`
	ParentID    string `json:"parent_id,omitempty" description:"Parent role ID for inheritance"`
}

// ListRolesRequest binds query params for GET /roles.
type ListRolesRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// GetRoleRequest binds the path for GET /roles/:roleId.
type GetRoleRequest struct {
	RoleID string `path:"roleId" description:"Role identifier"`
}

// UpdateRoleRequest binds path + body for PATCH /roles/:roleId.
type UpdateRoleRequest struct {
	RoleID      string  `path:"roleId" description:"Role identifier"`
	Name        *string `json:"name,omitempty" description:"Role name"`
	Description *string `json:"description,omitempty" description:"Role description"`
	ParentID    *string `json:"parent_id,omitempty" description:"Parent role ID for inheritance (empty string to clear)"`
}

// DeleteRoleRequest binds the path for DELETE /roles/:roleId.
type DeleteRoleRequest struct {
	RoleID string `path:"roleId" description:"Role identifier"`
}

// AddPermissionRequest binds path + body for POST /roles/:roleId/permissions.
type AddPermissionRequest struct {
	RoleID   string `path:"roleId" description:"Role identifier"`
	Action   string `json:"action" description:"Permission action (e.g. read, write, delete, admin)"`
	Resource string `json:"resource" description:"Permission resource (e.g. user, org, document)"`
}

// RemovePermissionRequest binds the path for DELETE /roles/:roleId/permissions/:permissionId.
type RemovePermissionRequest struct {
	RoleID       string `path:"roleId" description:"Role identifier"`
	PermissionID string `path:"permissionId" description:"Permission identifier"`
}

// ListRolePermissionsRequest binds the path for GET /roles/:roleId/permissions.
type ListRolePermissionsRequest struct {
	RoleID string `path:"roleId" description:"Role identifier"`
}

// AssignRoleRequest binds path + body for POST /roles/:roleId/assign.
type AssignRoleRequest struct {
	RoleID string `path:"roleId" description:"Role identifier"`
	UserID string `json:"user_id" description:"User identifier"`
	OrgID  string `json:"org_id,omitempty" description:"Optional organization scope"`
}

// UnassignRoleRequest binds path + body for POST /roles/:roleId/unassign.
type UnassignRoleRequest struct {
	RoleID string `path:"roleId" description:"Role identifier"`
	UserID string `json:"user_id" description:"User identifier"`
}

// ListUserRolesRequest binds the path for GET /users/:userId/roles.
type ListUserRolesRequest struct {
	UserID string `path:"userId" description:"User identifier"`
}

// RoleListResponse wraps a list of roles.
type RoleListResponse struct {
	Roles any `json:"roles" description:"List of roles"`
}

// PermissionListResponse wraps a list of permissions.
type PermissionListResponse struct {
	Permissions any `json:"permissions" description:"List of permissions"`
}

// UserRoleListResponse wraps a list of roles for a user.
type UserRoleListResponse struct {
	Roles any `json:"roles" description:"List of roles assigned to the user"`
}

// ForgotPasswordResponse is returned for forgot password requests.
// Intentionally sparse to avoid email enumeration.
type ForgotPasswordResponse struct {
	Status string `json:"status" description:"Always 'ok' regardless of email existence"`
}

// ---------------------------------------------------------------------------
// Admin requests
// ---------------------------------------------------------------------------

// AdminListUsersRequest binds query params for GET /admin/users.
type AdminListUsersRequest struct {
	AppID  string `query:"app_id" description:"Application ID"`
	Email  string `query:"email" description:"Filter by email (partial match)"`
	Cursor string `query:"cursor" description:"Pagination cursor"`
	Limit  int    `query:"limit" description:"Maximum number of results (default 20, max 100)"`
}

// AdminGetUserRequest binds the path for GET /admin/users/:userId.
type AdminGetUserRequest struct {
	UserID string `path:"userId" description:"User identifier"`
}

// AdminBanUserRequest binds path + body for POST /admin/users/:userId/ban.
type AdminBanUserRequest struct {
	UserID    string  `path:"userId" description:"User identifier"`
	Reason    string  `json:"reason,omitempty" description:"Ban reason"`
	ExpiresAt *string `json:"expires_at,omitempty" description:"Ban expiration (RFC3339, omit for permanent)"`
}

// AdminUnbanUserRequest binds the path for POST /admin/users/:userId/unban.
type AdminUnbanUserRequest struct {
	UserID string `path:"userId" description:"User identifier"`
}

// AdminDeleteUserRequest binds the path for DELETE /admin/users/:userId.
type AdminDeleteUserRequest struct {
	UserID string `path:"userId" description:"User identifier"`
}

// AdminStatsRequest is an empty request for GET /admin/stats.
type AdminStatsRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// ---------------------------------------------------------------------------
// Admin response types
// ---------------------------------------------------------------------------

// AdminUserListResponse wraps a paginated list of users.
type AdminUserListResponse struct {
	Users      any    `json:"users" description:"List of users"`
	NextCursor string `json:"next_cursor,omitempty" description:"Pagination cursor for next page"`
	Total      int    `json:"total" description:"Total number of matching users"`
}

// AdminStatsResponse returns basic analytics for an app.
type AdminStatsResponse struct {
	TotalUsers int `json:"total_users" description:"Total number of users"`
}

// AdminImpersonateRequest binds the path for POST /admin/impersonate/:userId.
type AdminImpersonateRequest struct {
	UserID string `path:"userId" description:"User identifier to impersonate"`
}

// AdminStopImpersonationRequest is an empty request for POST /admin/impersonate/stop.
type AdminStopImpersonationRequest struct{}

// ---------------------------------------------------------------------------
// GDPR / Account deletion requests
// ---------------------------------------------------------------------------

// DeleteAccountRequest is an empty request for DELETE /me (user from context).
type DeleteAccountRequest struct{}

// ExportDataRequest is an empty request for GET /me/export (user from context).
type ExportDataRequest struct{}

// ---------------------------------------------------------------------------
// Environment requests
// ---------------------------------------------------------------------------

// CreateEnvironmentRequest binds the body for POST /environments.
type CreateEnvironmentRequest struct {
	AppID       string                `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Name        string                `json:"name" description:"Environment name"`
	Slug        string                `json:"slug" description:"URL-safe environment slug"`
	Type        string                `json:"type" description:"Environment type (development, staging, production)"`
	Color       string                `json:"color,omitempty" description:"UI badge color (hex, defaults to type color)"`
	Description string                `json:"description,omitempty" description:"Environment description"`
	Settings    *environment.Settings `json:"settings,omitempty" description:"Per-environment settings overrides"`
}

// ListEnvironmentsRequest binds query params for GET /environments.
type ListEnvironmentsRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// GetEnvironmentRequest binds the path for GET /environments/:envId.
type GetEnvironmentRequest struct {
	EnvID string `path:"envId" description:"Environment identifier"`
}

// UpdateEnvironmentRequest binds path + body for PATCH /environments/:envId.
type UpdateEnvironmentRequest struct {
	EnvID       string  `path:"envId" description:"Environment identifier"`
	Name        *string `json:"name,omitempty" description:"Environment name"`
	Description *string `json:"description,omitempty" description:"Environment description"`
	Color       *string `json:"color,omitempty" description:"UI badge color (hex)"`
}

// DeleteEnvironmentRequest binds the path for DELETE /environments/:envId.
type DeleteEnvironmentRequest struct {
	EnvID string `path:"envId" description:"Environment identifier"`
}

// CloneEnvironmentRequest binds path + body for POST /environments/:envId/clone.
type CloneEnvironmentRequest struct {
	EnvID              string                `path:"envId" description:"Source environment identifier"`
	Name               string                `json:"name" description:"Name for the cloned environment"`
	Slug               string                `json:"slug" description:"URL-safe slug for the cloned environment"`
	Type               string                `json:"type" description:"Environment type (development, staging, production)"`
	Description        string                `json:"description,omitempty" description:"Description for the cloned environment"`
	Settings           *environment.Settings `json:"settings,omitempty" description:"Settings overrides for the cloned environment"`
	WebhookURLOverride string                `json:"webhook_url_override,omitempty" description:"Override URL for cloned webhooks"`
}

// GetEnvironmentSettingsRequest binds the path for GET /environments/:envId/settings.
type GetEnvironmentSettingsRequest struct {
	EnvID string `path:"envId" description:"Environment identifier"`
}

// SetDefaultEnvironmentRequest binds the path for POST /environments/:envId/set-default.
type SetDefaultEnvironmentRequest struct {
	EnvID string `path:"envId" description:"Environment identifier"`
}

// EnvironmentListResponse wraps a list of environments.
type EnvironmentListResponse struct {
	Environments any `json:"environments" description:"List of environments"`
	Total        int `json:"total" description:"Total number of environments"`
}

// EnvironmentSettingsResponse returns resolved settings with breakdown.
type EnvironmentSettingsResponse struct {
	Settings     *environment.Settings `json:"settings" description:"Resolved effective settings"`
	TypeDefaults *environment.Settings `json:"type_defaults" description:"Default settings for the environment type"`
	Overrides    *environment.Settings `json:"overrides" description:"Per-environment overrides (nil fields inherit type defaults)"`
}

// CloneEnvironmentResponse wraps the result of a clone operation.
type CloneEnvironmentResponse struct {
	Environment       any               `json:"environment" description:"Newly created environment"`
	RolesCloned       int               `json:"roles_cloned" description:"Number of roles cloned"`
	PermissionsCloned int               `json:"permissions_cloned" description:"Number of permissions cloned"`
	WebhooksCloned    int               `json:"webhooks_cloned" description:"Number of webhooks cloned"`
	RoleIDMap         map[string]string `json:"role_id_map" description:"Mapping of old role IDs to new role IDs"`
}

// ---------------------------------------------------------------------------
// App session config requests
// ---------------------------------------------------------------------------

// GetAppSessionConfigRequest binds the path for GET /admin/apps/:appId/session-config.
type GetAppSessionConfigRequest struct {
	AppID string `path:"appId" description:"Application identifier"`
}

// SetAppSessionConfigRequest binds the path and body for PUT /admin/apps/:appId/session-config.
type SetAppSessionConfigRequest struct {
	AppID                  string `path:"appId" description:"Application identifier"`
	TokenTTLSeconds        *int   `json:"token_ttl_seconds,omitempty" description:"Token TTL in seconds (nil = inherit)"`
	RefreshTokenTTLSeconds *int   `json:"refresh_token_ttl_seconds,omitempty" description:"Refresh token TTL in seconds (nil = inherit)"`
	MaxActiveSessions      *int   `json:"max_active_sessions,omitempty" description:"Maximum active sessions per user (nil = inherit)"`
	RotateRefreshToken     *bool  `json:"rotate_refresh_token,omitempty" description:"Rotate refresh token on use (nil = inherit)"`
	BindToIP               *bool  `json:"bind_to_ip,omitempty" description:"Bind sessions to IP address (nil = inherit)"`
	BindToDevice           *bool  `json:"bind_to_device,omitempty" description:"Bind sessions to device (nil = inherit)"`
	TokenFormat            string `json:"token_format,omitempty" description:"Token format: opaque or jwt (empty = inherit)"`
}

// DeleteAppSessionConfigRequest binds the path for DELETE /admin/apps/:appId/session-config.
type DeleteAppSessionConfigRequest struct {
	AppID string `path:"appId" description:"Application identifier"`
}
