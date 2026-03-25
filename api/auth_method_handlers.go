package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
)

// ──────────────────────────────────────────────────
// Auth method route registration
// ──────────────────────────────────────────────────

// registerAuthMethodRoutes registers the auth method listing/unlinking endpoints.
func (a *API) registerAuthMethodRoutes(router forge.Router) error {
	g := router.Group("/v1/me", forge.WithGroupTags("Auth Methods"))

	if err := g.GET("/auth-methods", a.handleListAuthMethods,
		forge.WithSummary("List linked auth methods"),
		forge.WithDescription("Returns all authentication methods linked to the current user."),
		forge.WithOperationID("listAuthMethods"),
		forge.WithResponseSchema(http.StatusOK, "Auth methods", ListAuthMethodsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/auth-methods/:provider", a.handleUnlinkAuthMethod,
		forge.WithSummary("Unlink an auth method"),
		forge.WithDescription("Removes an authentication method from the current user. Cannot remove the last method."),
		forge.WithOperationID("unlinkAuthMethod"),
		forge.WithResponseSchema(http.StatusOK, "Unlinked", UnlinkAuthMethodResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request / response types
// ──────────────────────────────────────────────────

// ListAuthMethodsRequest is the (empty) request for listing auth methods.
type ListAuthMethodsRequest struct{}

// ListAuthMethodsResponse contains all auth methods linked to the current user.
type ListAuthMethodsResponse struct {
	Methods []*plugin.AuthMethod `json:"methods"`
}

// UnlinkAuthMethodRequest carries the provider name to unlink.
type UnlinkAuthMethodRequest struct {
	Provider string `param:"provider"`
}

// UnlinkAuthMethodResponse confirms the unlink operation.
type UnlinkAuthMethodResponse struct {
	Status string `json:"status"`
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (a *API) handleListAuthMethods(ctx forge.Context, _ *ListAuthMethodsRequest) (*ListAuthMethodsResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	methods, err := a.engine.ListAuthMethods(ctx.Context(), userID)
	if err != nil {
		return nil, forge.InternalError(err)
	}
	if methods == nil {
		methods = []*plugin.AuthMethod{}
	}

	return &ListAuthMethodsResponse{Methods: methods}, nil
}

func (a *API) handleUnlinkAuthMethod(ctx forge.Context, req *UnlinkAuthMethodRequest) (*UnlinkAuthMethodResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Provider == "" {
		return nil, forge.BadRequest("provider is required")
	}

	if err := a.engine.UnlinkAuthMethod(ctx.Context(), userID, req.Provider); err != nil {
		return nil, forge.BadRequest(err.Error())
	}

	return &UnlinkAuthMethodResponse{Status: "unlinked"}, nil
}
