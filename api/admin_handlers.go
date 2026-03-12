package api

import (
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Admin route registration
// ──────────────────────────────────────────────────

func (a *API) registerAdminRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base+"/admin",
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

	return g.POST("/impersonate/stop", a.handleAdminStopImpersonation,
		forge.WithSummary("Stop impersonation (admin)"),
		forge.WithDescription("Terminates the current impersonation session. Can be called by the impersonated session itself."),
		forge.WithOperationID("adminStopImpersonation"),
		forge.WithResponseSchema(http.StatusOK, "Stopped", StatusResponse{}),
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
