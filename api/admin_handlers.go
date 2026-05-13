package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	authsomeapp "github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Admin route registration
// ──────────────────────────────────────────────────

func (a *API) registerAdminRoutes(router forge.Router) error {
	g := router.Group("/v1/admin",
		forge.WithGroupTags("admin"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequirePermission(a.engine, "manage", "user"),
		),
	)

	// Users
	if err := g.GET("/users", a.handleAdminListUsers,
		forge.WithSummary("List users (admin)"),
		forge.WithDescription("Returns a paginated list of all users. Requires admin role."),
		forge.WithOperationID("adminListUsers"),
		forge.WithResponseSchema(http.StatusOK, "User list", AdminUserListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/users/:userId", a.handleAdminGetUser,
		forge.WithSummary("Get user (admin)"),
		forge.WithDescription("Returns a user by ID. Requires admin role."),
		forge.WithOperationID("adminGetUser"),
		forge.WithResponseSchema(http.StatusOK, "User", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/users/:userId/ban", a.handleAdminBanUser,
		forge.WithSummary("Ban user (admin)"),
		forge.WithDescription("Bans a user and revokes all active sessions. Requires admin role."),
		forge.WithOperationID("adminBanUser"),
		forge.WithRequestSchema(AdminBanUserRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Banned", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/users/:userId/unban", a.handleAdminUnbanUser,
		forge.WithSummary("Unban user (admin)"),
		forge.WithDescription("Removes a ban from a user account. Requires admin role."),
		forge.WithOperationID("adminUnbanUser"),
		forge.WithResponseSchema(http.StatusOK, "Unbanned", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/users/:userId", a.handleAdminDeleteUser,
		forge.WithSummary("Delete user (admin)"),
		forge.WithDescription("Permanently deletes a user and all associated data. Requires admin role."),
		forge.WithOperationID("adminDeleteUser"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/users/:userId", a.handleAdminUpdateUser,
		forge.WithSummary("Update user (admin)"),
		forge.WithDescription("Updates a user's profile fields. Requires admin role."),
		forge.WithOperationID("adminUpdateUser"),
		forge.WithRequestSchema(AdminUpdateUserRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated user", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/users/create", a.handleAdminCreateUser,
		forge.WithSummary("Create user (admin)"),
		forge.WithDescription("Creates a new user without going through the signup flow. The user is created with email verified. Requires admin role."),
		forge.WithOperationID("adminCreateUser"),
		forge.WithRequestSchema(AdminCreateUserRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Created user", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/users/copy", a.handleAdminCopyUser,
		forge.WithSummary("Copy user to another app (admin)"),
		forge.WithDescription("Provisions a new user record in target_app_id reusing the source user's email, profile, and stored password hash. The duplicated user can authenticate with their original password. 409 when the target app already has a user with the same email. Requires platform-admin role."),
		forge.WithOperationID("adminCopyUser"),
		forge.WithRequestSchema(AdminCopyUserRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Copied user", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Stats
	if err := g.GET("/stats", a.handleAdminStats,
		forge.WithSummary("Get stats (admin)"),
		forge.WithDescription("Returns basic analytics for the application. Requires admin role."),
		forge.WithOperationID("adminGetStats"),
		forge.WithResponseSchema(http.StatusOK, "Stats", AdminStatsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Impersonation
	if err := g.POST("/impersonate/:userId", a.handleAdminImpersonate,
		forge.WithSummary("Impersonate user (admin)"),
		forge.WithDescription("Creates a short-lived session as the target user. The session carries an audit trail of the impersonating admin. Requires admin role."),
		forge.WithOperationID("adminImpersonate"),
		forge.WithResponseSchema(http.StatusOK, "Impersonation session", AuthResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/impersonate/stop", a.handleAdminStopImpersonation,
		forge.WithSummary("Stop impersonation (admin)"),
		forge.WithDescription("Terminates the current impersonation session. Can be called by the impersonated session itself."),
		forge.WithOperationID("adminStopImpersonation"),
		forge.WithResponseSchema(http.StatusOK, "Stopped", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Apps — multi-App tenancy admin endpoints. Caller needs the
	// same manage:user permission as the rest of the admin group;
	// in practice that's a platform-admin API key minted in App
	// `studio`. Used by twinos studio's saga to spin up a fresh
	// Authsome App per workspace under the App-per-workspace
	// architecture.
	if err := g.POST("/apps", a.handleAdminCreateApp,
		forge.WithSummary("Create application (admin)"),
		forge.WithDescription("Creates a new Authsome application. Bootstraps default environments + roles. Requires platform admin role."),
		forge.WithOperationID("adminCreateApp"),
		forge.WithRequestSchema(AdminCreateAppRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Created application", AdminAppResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/apps/:appId", a.handleAdminDeleteApp,
		forge.WithSummary("Delete application (admin)"),
		forge.WithDescription("Permanently deletes an application and cascades to all child envs / orgs / users / sessions / OAuth clients. Requires platform admin role. Used as the rollback action for App-per-workspace provisioning."),
		forge.WithOperationID("adminDeleteApp"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Platform owner management
	if err := g.POST("/platform/owners", a.handleGrantPlatformOwner,
		forge.WithSummary("Grant platform-owner role (admin)"),
		forge.WithDescription("Grants the platform-owner role to a user identified by user_id or email. Requires platform-owner role."),
		forge.WithOperationID("adminGrantPlatformOwner"),
		forge.WithRequestSchema(AdminGrantPlatformOwnerRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Granted", AdminPlatformOwnerResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/platform/owners/:userID", a.handleRevokePlatformOwner,
		forge.WithSummary("Revoke platform-owner role (admin)"),
		forge.WithDescription("Revokes the platform-owner role from a user. Fails if the target user is the last platform owner."),
		forge.WithOperationID("adminRevokePlatformOwner"),
		forge.WithResponseSchema(http.StatusOK, "Revoked", AdminPlatformOwnerResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Service Accounts
	if err := g.POST("/service-accounts", a.handleAdminCreateServiceAccount,
		forge.WithSummary("Create service account (admin)"),
		forge.WithDescription("Creates a new service account for machine-to-machine authentication. Requires admin role."),
		forge.WithOperationID("adminCreateServiceAccount"),
		forge.WithRequestSchema(AdminCreateServiceAccountRequest{}),
		forge.WithCreatedResponse(AdminServiceAccountResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/service-accounts", a.handleAdminListServiceAccounts,
		forge.WithSummary("List service accounts (admin)"),
		forge.WithDescription("Returns a list of all service accounts for the app. Requires admin role."),
		forge.WithOperationID("adminListServiceAccounts"),
		forge.WithResponseSchema(http.StatusOK, "Service account list", AdminServiceAccountListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/service-accounts/:serviceAccountId", a.handleAdminGetServiceAccount,
		forge.WithSummary("Get service account (admin)"),
		forge.WithDescription("Returns a service account by ID. Requires admin role."),
		forge.WithOperationID("adminGetServiceAccount"),
		forge.WithResponseSchema(http.StatusOK, "Service account", AdminServiceAccountResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/service-accounts/:serviceAccountId", a.handleAdminDeleteServiceAccount,
		forge.WithSummary("Delete service account (admin)"),
		forge.WithDescription("Permanently deletes a service account. Requires admin role."),
		forge.WithOperationID("adminDeleteServiceAccount"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/service-accounts/:serviceAccountId/api-keys", a.handleAdminCreateServiceAccountAPIKey,
		forge.WithSummary("Create service account API key (admin)"),
		forge.WithDescription("Mints an API key for a service account. The plaintext secret is returned only once. Requires admin role."),
		forge.WithOperationID("adminCreateServiceAccountAPIKey"),
		forge.WithRequestSchema(AdminCreateServiceAccountAPIKeyRequest{}),
		forge.WithCreatedResponse(AdminServiceAccountAPIKeyResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Admin handlers
// ──────────────────────────────────────────────────

func (a *API) handleAdminListUsers(ctx forge.Context, req *AdminListUsersRequest) (*AdminUserListResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	list, err := a.engine.AdminListUsers(ctx.Context(), &user.Query{
		AppID:  appID,
		Email:  req.Email,
		Cursor: req.Cursor,
		Limit:  limit,
	})
	if err != nil {
		return nil, mapError(err)
	}

	resp := &AdminUserListResponse{
		Users:      list.Users,
		NextCursor: list.NextCursor,
		Total:      list.Total,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminGetUser(ctx forge.Context, req *AdminGetUserRequest) (*user.User, error) {
	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	u, err := a.engine.AdminGetUser(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	return u, nil
}

func (a *API) handleAdminBanUser(ctx forge.Context, req *AdminBanUserRequest) (*StatusResponse, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	// Prevent self-ban
	if adminID == userID {
		return nil, forge.BadRequest("cannot ban yourself")
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, forge.BadRequest("invalid expires_at: must be RFC3339 format")
		}
		expiresAt = &t
	}

	if err := a.engine.AdminBanUser(ctx.Context(), adminID, userID, req.Reason, expiresAt); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "banned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminUnbanUser(ctx forge.Context, req *AdminUnbanUserRequest) (*StatusResponse, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	if err := a.engine.AdminUnbanUser(ctx.Context(), adminID, userID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "unbanned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminDeleteUser(ctx forge.Context, req *AdminDeleteUserRequest) (*StatusResponse, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	// Prevent self-deletion
	if adminID == userID {
		return nil, forge.BadRequest("cannot delete yourself")
	}

	if err := a.engine.AdminDeleteUser(ctx.Context(), adminID, userID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminStats(ctx forge.Context, req *AdminStatsRequest) (*AdminStatsResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	users, err := a.engine.AdminListUsers(ctx.Context(), &user.Query{
		AppID: appID,
		Limit: 1,
	})
	if err != nil {
		return nil, mapError(err)
	}

	resp := &AdminStatsResponse{
		TotalUsers: users.Total,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminImpersonate(ctx forge.Context, req *AdminImpersonateRequest) (*AuthResponse, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	targetID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	if adminID == targetID {
		return nil, forge.BadRequest("cannot impersonate yourself")
	}

	u, sess, err := a.engine.Impersonate(ctx.Context(), adminID, targetID)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, authResponse(u, sess))
}

func (a *API) handleAdminStopImpersonation(ctx forge.Context, _ *AdminStopImpersonationRequest) (*StatusResponse, error) {
	sessID, ok := middleware.SessionIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if err := a.engine.StopImpersonation(ctx.Context(), sessID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "impersonation stopped"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAdminUpdateUser(ctx forge.Context, req *AdminUpdateUserRequest) (*user.User, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	updates := authsome.AdminUserUpdates{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Username:      req.Username,
		EmailVerified: req.EmailVerified,
	}

	if err2 := a.engine.AdminUpdateUser(ctx.Context(), adminID, userID, updates); err2 != nil {
		return nil, mapError(err2)
	}

	u, err := a.engine.AdminGetUser(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	return u, nil
}

func (a *API) handleAdminCreateUser(ctx forge.Context, req *AdminCreateUserRequest) (*user.User, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	u, err := a.engine.AdminCreateUser(ctx.Context(), adminID, appID, id.EnvironmentID{}, req.Email, req.Password, req.FirstName, req.LastName, req.Username)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, u)
}

// handleAdminCopyUser provisions a duplicate of an existing user under
// a different App, reusing the source's stored password hash so the
// duplicate can sign in with the original password. Used by TwinOS
// studio to lift end-users between workspace Apps without resetting
// their credentials. ErrEmailTaken on the target maps to 409 via
// mapError so callers can treat it as idempotent.
func (a *API) handleAdminCopyUser(ctx forge.Context, req *AdminCopyUserRequest) (*user.User, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if strings.TrimSpace(req.SourceUserID) == "" || strings.TrimSpace(req.TargetAppID) == "" {
		return nil, forge.BadRequest("source_user_id and target_app_id are required")
	}

	sourceUserID, err := id.ParseUserID(req.SourceUserID)
	if err != nil {
		return nil, forge.BadRequest("invalid source_user_id")
	}
	targetAppID, err := id.ParseAppID(req.TargetAppID)
	if err != nil {
		return nil, forge.BadRequest("invalid target_app_id")
	}

	envID := id.EnvironmentID{}
	if strings.TrimSpace(req.EnvID) != "" {
		parsed, err := id.ParseEnvironmentID(req.EnvID)
		if err != nil {
			return nil, forge.BadRequest("invalid env_id")
		}
		envID = parsed
	}

	dup, err := a.engine.AdminCopyUserToApp(ctx.Context(), adminID, sourceUserID, targetAppID, envID)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, dup)
}

// handleAdminCreateApp creates a brand new Authsome App. Used by
// downstream services that need their own App per tenant — e.g.
// TwinOS studio creates one App per workspace under the App-per-
// workspace architecture so per-workspace device tokens / sessions
// / OAuth clients are fully isolated.
func (a *API) handleAdminCreateApp(ctx forge.Context, req *AdminCreateAppRequest) (*AdminAppResponse, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if req.Name == "" || req.Slug == "" {
		return nil, forge.BadRequest("name and slug are required")
	}

	newApp := &authsomeapp.App{
		Name:       req.Name,
		Slug:       req.Slug,
		Logo:       req.Logo,
		IsPlatform: false, // platform App is bootstrapped once at startup
	}
	if err := a.engine.CreateApp(ctx.Context(), newApp); err != nil {
		return nil, mapError(err)
	}

	return &AdminAppResponse{
		ID:             newApp.ID.String(),
		Name:           newApp.Name,
		Slug:           newApp.Slug,
		Logo:           newApp.Logo,
		PublishableKey: newApp.PublishableKey,
		IsPlatform:     newApp.IsPlatform,
	}, nil
}

// handleAdminDeleteApp removes an Authsome App and cascades to all
// child entities (envs, orgs, users, sessions, OAuth clients,
// device tokens). The saga's compensating action when downstream
// workspace-creation steps fail.
func (a *API) handleAdminDeleteApp(ctx forge.Context, req *AdminDeleteAppRequest) (*StatusResponse, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	parsed, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}
	if err := a.engine.DeleteApp(ctx.Context(), parsed); err != nil {
		return nil, mapError(err)
	}
	return &StatusResponse{Status: "ok"}, nil
}

// handleGrantPlatformOwner grants the platform-owner role to a user identified
// by user_id or email. The caller must themselves be a platform owner.
func (a *API) handleGrantPlatformOwner(ctx forge.Context, req *AdminGrantPlatformOwnerRequest) (*AdminPlatformOwnerResponse, error) {
	callerID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	// C1: only platform-owners may call this endpoint.
	roles, err := a.engine.ListUserRoles(ctx.Context(), callerID)
	if err != nil {
		return nil, mapError(err)
	}
	isPlatformOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			isPlatformOwner = true
			break
		}
	}
	if !isPlatformOwner {
		return nil, forge.Forbidden("platform-owner role required")
	}

	if req.UserID == "" && req.Email == "" {
		return nil, forge.BadRequest("user_id or email is required")
	}

	appID := a.engine.PlatformAppID()

	// Resolve target user.
	var targetUserID id.UserID
	if req.UserID != "" {
		parsed, err := id.ParseUserID(req.UserID)
		if err != nil {
			return nil, forge.BadRequest("invalid user_id")
		}
		targetUserID = parsed
	} else {
		// Look up by email within the platform app.
		u, err := a.engine.Store().GetUserByEmail(ctx.Context(), appID, strings.ToLower(strings.TrimSpace(req.Email)))
		if err != nil {
			return nil, mapError(err)
		}
		targetUserID = u.ID
	}

	ownerRole, err := a.engine.GetRoleBySlug(ctx.Context(), appID, rbac.PlatformOwnerSlug)
	if err != nil || ownerRole == nil {
		return nil, forge.InternalError(fmt.Errorf("platform-owner role not found"))
	}

	if err := a.engine.AssignUserRole(ctx.Context(), &rbac.UserRole{
		UserID: targetUserID.String(),
		RoleID: ownerRole.ID,
	}); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, &AdminPlatformOwnerResponse{
		UserID: targetUserID.String(),
		RoleID: ownerRole.ID,
		Status: "granted",
	})
}

// handleRevokePlatformOwner revokes the platform-owner role from a user.
// It rejects the request if the target user is the last platform owner.
func (a *API) handleRevokePlatformOwner(ctx forge.Context, req *AdminRevokePlatformOwnerRequest) (*AdminPlatformOwnerResponse, error) {
	callerID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	// C1: only platform-owners may call this endpoint.
	roles, err := a.engine.ListUserRoles(ctx.Context(), callerID)
	if err != nil {
		return nil, mapError(err)
	}
	isPlatformOwner := false
	for _, r := range roles {
		if r.Slug == rbac.PlatformOwnerSlug {
			isPlatformOwner = true
			break
		}
	}
	if !isPlatformOwner {
		return nil, forge.Forbidden("platform-owner role required")
	}

	targetUserID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	appID := a.engine.PlatformAppID()

	ownerRole, err := a.engine.GetRoleBySlug(ctx.Context(), appID, rbac.PlatformOwnerSlug)
	if err != nil || ownerRole == nil {
		return nil, forge.InternalError(fmt.Errorf("platform-owner role not found"))
	}

	// Guard: do not allow removing the last platform owner.
	// List all users with the platform-owner role for this app.
	owners, err := a.engine.ListUsersWithRole(ctx.Context(), appID, rbac.PlatformOwnerSlug)
	if err != nil {
		return nil, mapError(err)
	}
	// I2: treat a zero-length result as a data-inconsistency error rather
	// than silently skipping the guard.
	if len(owners) == 0 {
		return nil, forge.InternalError(fmt.Errorf("no platform owners found"))
	}
	// Check if removing this user would leave zero owners.
	matchesTarget := false
	for _, ownerID := range owners {
		if ownerID == targetUserID {
			matchesTarget = true
			break
		}
	}
	if matchesTarget && len(owners) <= 1 {
		return nil, forge.BadRequest("cannot revoke platform-owner from the last platform owner")
	}

	roleID, err := id.Parse(ownerRole.ID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid platform-owner role ID"))
	}

	if err := a.engine.UnassignUserRole(ctx.Context(), targetUserID, roleID); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, &AdminPlatformOwnerResponse{
		UserID: targetUserID.String(),
		RoleID: ownerRole.ID,
		Status: "revoked",
	})
}

// ──────────────────────────────────────────────────
// Service Account handlers
// ──────────────────────────────────────────────────

// AdminServiceAccountResponse is the JSON representation of a service account.
type AdminServiceAccountResponse struct {
	ID          string    `json:"id"`
	AppID       string    `json:"app_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Scopes      []string  `json:"scopes,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminServiceAccountListResponse is the response for listing service accounts.
type AdminServiceAccountListResponse struct {
	ServiceAccounts []*AdminServiceAccountResponse `json:"service_accounts"`
	Total           int                            `json:"total"`
	NextCursor      string                         `json:"next_cursor,omitempty"`
}

// AdminServiceAccountAPIKeyResponse is the response when minting a service account API key.
type AdminServiceAccountAPIKeyResponse struct {
	ID              string     `json:"id"`
	Key             string     `json:"key"`
	KeyPrefix       string     `json:"key_prefix"`
	PublicKey       string     `json:"public_key,omitempty"`
	PublicKeyPrefix string     `json:"public_key_prefix,omitempty"`
	Name            string     `json:"name"`
	Scopes          []string   `json:"scopes,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func toServiceAccountResponse(svc *serviceaccount.ServiceAccount) *AdminServiceAccountResponse {
	return &AdminServiceAccountResponse{
		ID:          svc.ID.String(),
		AppID:       svc.AppID.String(),
		Name:        svc.Name,
		Description: svc.Description,
		Scopes:      svc.Scopes,
		Active:      svc.Active,
		CreatedAt:   svc.CreatedAt,
		UpdatedAt:   svc.UpdatedAt,
	}
}

func (a *API) handleAdminCreateServiceAccount(ctx forge.Context, req *AdminCreateServiceAccountRequest) (*AdminServiceAccountResponse, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	svc, err := a.engine.CreateServiceAccount(ctx.Context(), appID, req.Name, req.Description, req.Scopes)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusCreated, toServiceAccountResponse(svc))
}

func (a *API) handleAdminListServiceAccounts(ctx forge.Context, req *AdminListServiceAccountsRequest) (*AdminServiceAccountListResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	list, err := a.engine.ListServiceAccounts(ctx.Context(), appID, limit)
	if err != nil {
		return nil, mapError(err)
	}

	items := make([]*AdminServiceAccountResponse, 0, len(list.ServiceAccounts))
	for _, svc := range list.ServiceAccounts {
		items = append(items, toServiceAccountResponse(svc))
	}

	return nil, ctx.JSON(http.StatusOK, &AdminServiceAccountListResponse{
		ServiceAccounts: items,
		Total:           list.Total,
		NextCursor:      list.NextCursor,
	})
}

func (a *API) handleAdminGetServiceAccount(ctx forge.Context, req *AdminGetServiceAccountRequest) (*AdminServiceAccountResponse, error) {
	svcID, err := id.ParseServiceAccountID(req.ServiceAccountID)
	if err != nil {
		return nil, forge.BadRequest("invalid service_account_id")
	}

	svc, err := a.engine.GetServiceAccount(ctx.Context(), svcID)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, toServiceAccountResponse(svc))
}

func (a *API) handleAdminDeleteServiceAccount(ctx forge.Context, req *AdminDeleteServiceAccountRequest) (*StatusResponse, error) {
	svcID, err := id.ParseServiceAccountID(req.ServiceAccountID)
	if err != nil {
		return nil, forge.BadRequest("invalid service_account_id")
	}

	if err := a.engine.DeleteServiceAccount(ctx.Context(), svcID); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "deleted"})
}

func (a *API) handleAdminCreateServiceAccountAPIKey(ctx forge.Context, req *AdminCreateServiceAccountAPIKeyRequest) (*AdminServiceAccountAPIKeyResponse, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}

	svcID, err := id.ParseServiceAccountID(req.ServiceAccountID)
	if err != nil {
		return nil, forge.BadRequest("invalid service_account_id")
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, parseErr := time.Parse(time.RFC3339, *req.ExpiresAt)
		if parseErr != nil {
			return nil, forge.BadRequest("invalid expires_at: must be RFC3339 format")
		}
		expiresAt = &t
	}

	k, secret, err := a.engine.CreateServiceAccountAPIKey(ctx.Context(), svcID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &AdminServiceAccountAPIKeyResponse{
		ID:              k.ID.String(),
		Key:             secret,
		KeyPrefix:       k.KeyPrefix,
		PublicKey:       k.PublicKey,
		PublicKeyPrefix: k.PublicKeyPrefix,
		Name:            k.Name,
		Scopes:          k.Scopes,
		ExpiresAt:       k.ExpiresAt,
		CreatedAt:       k.CreatedAt,
	}
	return nil, ctx.JSON(http.StatusCreated, resp)
}
