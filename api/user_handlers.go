package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// User route registration
// ──────────────────────────────────────────────────

func (a *API) registerUserRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("user"),
		// Every route in this group authenticates by session; the manifest in
		// api.go has said so by hand since before routes could declare it.
		forge.WithGroupAuth("session", "session-cookie"))

	if err := g.GET("/me", a.handleGetMe,
		forge.WithSummary("Get current user"),
		forge.WithDescription("Returns the currently authenticated user's profile and the roles their session carries."),
		forge.WithOperationID("getMe"),
		forge.WithResponseSchema(http.StatusOK, "User profile", MeResponse{}),
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

	if err := g.POST("/me/switch-org", a.handleSwitchOrg,
		forge.WithSummary("Switch active organization"),
		forge.WithDescription("Sets the active organization on the caller's session. The user must be a member of the target org. An empty org_id clears the active org."),
		forge.WithOperationID("switchOrg"),
		forge.WithRequestSchema(SwitchOrgRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated session", SwitchOrgResponse{}),
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

// MeResponse is the authenticated user plus the roles their session carries.
//
// user.User is embedded rather than nested, so every field this endpoint
// already returned keeps its place at the top level of the JSON and existing
// consumers see no change. Roles is the addition.
//
// A generated client needs the principal before its capability surface can
// answer anything: canCall reads from whatever was last handed to
// setPrincipal, and until this endpoint returned roles, every caller had to
// assemble them from the admin role-listing endpoints by hand.
type MeResponse struct {
	*user.User

	// Roles are the slugs stamped onto the session at sign-in, which is what
	// authorization is decided against for this request. Deliberately the
	// session's roles rather than a fresh lookup: a fresh list would show a
	// role the current session cannot actually exercise, and a client would
	// enable a control the server then refuses.
	Roles []string `json:"roles"`
}

func (a *API) handleGetMe(ctx forge.Context, _ *GetMeRequest) (*MeResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	u, err := a.engine.GetMe(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &MeResponse{User: u}
	if sess, ok := middleware.SessionFrom(ctx.Context()); ok && sess != nil {
		resp.Roles = sess.Roles
	}

	return resp, nil
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

	return u, nil
}

func (a *API) handleSwitchOrg(ctx forge.Context, req *SwitchOrgRequest) (*SwitchOrgResponse, error) {
	sessionID, ok := middleware.SessionIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	var newOrgID id.OrgID
	if req.OrgID != "" {
		parsed, err := id.ParseOrgID(req.OrgID)
		if err != nil {
			return nil, forge.BadRequest("invalid org_id")
		}
		newOrgID = parsed
	}

	updated, err := a.engine.SwitchActiveOrg(ctx.Context(), sessionID, newOrgID)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &SwitchOrgResponse{
		SessionID: updated.ID.String(),
	}
	if !updated.OrgID.IsNil() {
		resp.OrgID = updated.OrgID.String()
	}
	return resp, nil
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

func (a *API) handleExportData(ctx forge.Context, _ *ExportDataRequest) (*map[string]any, error) { //nolint:gocritic // Forge requires pointer return type for handler detection
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
