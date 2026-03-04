package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// User route registration
// ──────────────────────────────────────────────────

func (a *API) registerUserRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base, forge.WithGroupTags("user"))

	if err := g.GET("/me", a.handleGetMe,
		forge.WithSummary("Get current user"),
		forge.WithDescription("Returns the currently authenticated user's profile."),
		forge.WithOperationID("getMe"),
		forge.WithResponseSchema(http.StatusOK, "User profile", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/me", a.handleUpdateMe,
		forge.WithSummary("Update current user"),
		forge.WithDescription("Updates the authenticated user's profile fields."),
		forge.WithOperationID("updateMe"),
		forge.WithRequestSchema(UpdateMeRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated user", user.User{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/me", a.handleDeleteAccount,
		forge.WithSummary("Delete account"),
		forge.WithDescription("Permanently deletes the authenticated user's account and all associated data (GDPR right to erasure). This action is irreversible."),
		forge.WithOperationID("deleteAccount"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.GET("/me/export", a.handleExportData,
		forge.WithSummary("Export user data"),
		forge.WithDescription("Returns all data associated with the authenticated user for GDPR data portability."),
		forge.WithOperationID("exportUserData"),
		forge.WithResponseSchema(http.StatusOK, "User data export", map[string]any{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// User handlers
// ──────────────────────────────────────────────────

func (a *API) handleGetMe(ctx forge.Context, _ *GetMeRequest) (*user.User, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	u, err := a.engine.GetMe(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	return u, ctx.JSON(http.StatusOK, u)
}

func (a *API) handleUpdateMe(ctx forge.Context, req *UpdateMeRequest) (*user.User, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	u, err := a.engine.GetMe(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	if req.Image != nil {
		u.Image = *req.Image
	}
	if req.Username != nil {
		u.Username = *req.Username
	}

	if err := a.engine.UpdateMe(ctx.Context(), u); err != nil {
		return nil, mapError(err)
	}

	return u, ctx.JSON(http.StatusOK, u)
}

func (a *API) handleDeleteAccount(ctx forge.Context, _ *DeleteAccountRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if err := a.engine.DeleteAccount(ctx.Context(), userID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "account deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleExportData(ctx forge.Context, _ *ExportDataRequest) (*map[string]any, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	export, err := a.engine.ExportUserData(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusOK, export)
}
