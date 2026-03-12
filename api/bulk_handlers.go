package api

import (
	"net/http"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Bulk route registration
// ──────────────────────────────────────────────────

func (a *API) registerBulkRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base+"/admin/bulk",
		forge.WithGroupTags("admin", "bulk"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequirePermission(a.engine, "manage", "user"),
		),
	)

	if err := g.POST("/users/import", a.handleBulkImportUsers,
		forge.WithSummary("Bulk import users (admin)"),
		forge.WithDescription("Creates multiple users in a single request. Duplicate emails are skipped. Requires admin role."),
		forge.WithOperationID("adminBulkImportUsers"),
		forge.WithRequestSchema(BulkImportUsersRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Import result", authsome.BulkImportResult{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/sessions", a.handleBulkRevokeSessions,
		forge.WithSummary("Bulk revoke sessions (admin)"),
		forge.WithDescription("Revokes all sessions for a specific user. Requires admin role."),
		forge.WithOperationID("adminBulkRevokeSessions"),
		forge.WithResponseSchema(http.StatusOK, "Revocation result", BulkRevokeSessionsResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Bulk handlers
// ──────────────────────────────────────────────────

func (a *API) handleBulkImportUsers(ctx forge.Context, req *BulkImportUsersRequest) (*authsome.BulkImportResult, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if len(req.Users) == 0 {
		return nil, forge.BadRequest("users array is required and must not be empty")
	}
	if len(req.Users) > 1000 {
		return nil, forge.BadRequest("maximum 1000 users per import")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	// Convert request users to domain users
	users := make([]*user.User, len(req.Users))
	for i, u := range req.Users {
		users[i] = &user.User{
			AppID:        appID,
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			Metadata:     u.Metadata,
		}
	}

	result, err := a.engine.AdminBulkImportUsers(ctx.Context(), adminID, users)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, result)
}

func (a *API) handleBulkRevokeSessions(ctx forge.Context, req *BulkRevokeSessionsRequest) (*BulkRevokeSessionsResponse, error) {
	adminID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.UserID == "" {
		return nil, forge.BadRequest("user_id is required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	count, err := a.engine.AdminBulkRevokeSessions(ctx.Context(), adminID, userID)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &BulkRevokeSessionsResponse{
		Revoked: count,
		UserID:  req.UserID,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Request / response types
// ──────────────────────────────────────────────────

// BulkImportUsersRequest binds the body for POST /admin/bulk/users/import.
type BulkImportUsersRequest struct {
	AppID string           `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Users []BulkImportUser `json:"users" description:"Array of users to import (max 1000)"`
}

// BulkImportUser represents a single user in a bulk import request.
type BulkImportUser struct {
	Email        string        `json:"email" description:"User email address"`
	FirstName    string        `json:"first_name,omitempty" description:"First/given name"`
	LastName     string        `json:"last_name,omitempty" description:"Last/family name"`
	Username     string        `json:"username,omitempty" description:"Unique username"`
	PasswordHash string        `json:"password_hash" description:"Pre-hashed password (bcrypt or argon2id)"`
	Metadata     user.Metadata `json:"metadata,omitempty" description:"Arbitrary user metadata"`
}

// BulkRevokeSessionsRequest binds query params for DELETE /admin/bulk/sessions.
type BulkRevokeSessionsRequest struct {
	UserID string `query:"user_id" description:"User ID whose sessions to revoke"`
}

// BulkRevokeSessionsResponse returns the result of a bulk session revocation.
type BulkRevokeSessionsResponse struct {
	Revoked int    `json:"revoked" description:"Number of sessions revoked"`
	UserID  string `json:"user_id" description:"User whose sessions were revoked"`
}
