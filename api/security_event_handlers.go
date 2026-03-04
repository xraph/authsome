package api

import (
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/securityevent"
)

// ──────────────────────────────────────────────────
// Security event route registration
// ──────────────────────────────────────────────────

func (a *API) registerSecurityEventRoutes(router forge.Router) error {
	// Only register if a security event store is configured.
	if a.engine.SecurityEvents() == nil {
		return nil
	}

	base := a.engine.Config().BasePath
	g := router.Group(base+"/admin/security-events",
		forge.WithGroupTags("admin", "security-events"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequireAnyRole(a.engine, "admin", "super_admin"),
		),
	)

	return g.GET("", a.handleListSecurityEvents,
		forge.WithSummary("List security events (admin)"),
		forge.WithDescription("Returns a paginated list of security events. Supports filtering by user, action, and time range. Requires admin role."),
		forge.WithOperationID("adminListSecurityEvents"),
		forge.WithResponseSchema(http.StatusOK, "Security events", SecurityEventListResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Security event handlers
// ──────────────────────────────────────────────────

func (a *API) handleListSecurityEvents(ctx forge.Context, req *ListSecurityEventsRequest) (*SecurityEventListResponse, error) {
	q := &securityevent.Query{
		Action: req.Action,
		Cursor: req.Cursor,
		Limit:  req.Limit,
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	if req.AppID != "" {
		appID, err := id.ParseAppID(req.AppID)
		if err != nil {
			return nil, forge.BadRequest("invalid app_id")
		}
		q.AppID = appID
	}

	if req.UserID != "" {
		userID, err := id.ParseUserID(req.UserID)
		if err != nil {
			return nil, forge.BadRequest("invalid user_id")
		}
		q.UserID = userID
	}

	if req.Since != "" {
		t, err := time.Parse(time.RFC3339, req.Since)
		if err != nil {
			return nil, forge.BadRequest("invalid since: must be RFC3339 format")
		}
		q.Since = t
	}

	if req.Until != "" {
		t, err := time.Parse(time.RFC3339, req.Until)
		if err != nil {
			return nil, forge.BadRequest("invalid until: must be RFC3339 format")
		}
		q.Until = t
	}

	events, cursor, err := a.engine.SecurityEvents().QuerySecurityEvents(ctx.Context(), q)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &SecurityEventListResponse{
		Events:     events,
		NextCursor: cursor,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Request / response types
// ──────────────────────────────────────────────────

// ListSecurityEventsRequest binds query params for GET /admin/security-events.
type ListSecurityEventsRequest struct {
	AppID  string `query:"app_id" description:"Filter by application ID"`
	UserID string `query:"user_id" description:"Filter by user ID"`
	Action string `query:"action" description:"Filter by action (e.g. auth.signin, auth.account_locked)"`
	Since  string `query:"since" description:"Filter events after this time (RFC3339)"`
	Until  string `query:"until" description:"Filter events before this time (RFC3339)"`
	Cursor string `query:"cursor" description:"Pagination cursor"`
	Limit  int    `query:"limit" description:"Maximum number of results (default 50, max 200)"`
}

// SecurityEventListResponse wraps a paginated list of security events.
type SecurityEventListResponse struct {
	Events     []*securityevent.Event `json:"events" description:"List of security events"`
	NextCursor string                 `json:"next_cursor,omitempty" description:"Pagination cursor for next page"`
}
