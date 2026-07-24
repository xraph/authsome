package api

import (
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// ──────────────────────────────────────────────────
// Session route registration
// ──────────────────────────────────────────────────

func (a *API) registerSessionRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("sessions"))

	if err := g.GET("/sessions", a.handleListSessions,
		forge.WithSummary("List sessions"),
		forge.WithDescription("Returns all active sessions for the authenticated user."),
		forge.WithOperationID("listSessions"),
		forge.WithResponseSchema(http.StatusOK, "Session list", SessionListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/sessions/:sessionId", a.handleRevokeSession,
		forge.WithSummary("Revoke session"),
		forge.WithDescription("Revokes a specific session by ID."),
		forge.WithOperationID("revokeSession"),
		forge.WithResponseSchema(http.StatusOK, "Session revoked", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Session handlers
// ──────────────────────────────────────────────────

func (a *API) handleListSessions(ctx forge.Context, _ *ListSessionsRequest) (*SessionListResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	sessions, err := a.engine.ListSessions(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &SessionListResponse{Sessions: safeSessionSlice(sessions)}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleRevokeSession(ctx forge.Context, _ *RevokeSessionRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	rawID := ctx.Param("sessionId")
	sessID, err := id.ParseSessionID(rawID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid session id: %v", err))
	}

	// Ownership check: a caller may only revoke their own sessions. A
	// missing session and a session owned by another user both return 404 so
	// the endpoint never discloses the existence of another user's session.
	sess, err := a.engine.Store().GetSession(ctx.Context(), sessID)
	if err != nil || sess.UserID != userID {
		return nil, forge.NotFound("session not found")
	}

	if err := a.engine.RevokeSession(ctx.Context(), sessID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "revoked"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func safeSessionSlice(sessions []*session.Session) []*session.Session {
	if sessions == nil {
		return []*session.Session{}
	}
	return sessions
}
