package api

import (
	"net/http"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
)

// ──────────────────────────────────────────────────
// Public client config route registration
// ──────────────────────────────────────────────────

func (a *API) registerClientConfigRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("Client Config"))

	return g.GET("/client-config", a.handleGetClientConfig,
		forge.WithSummary("Get client configuration"),
		forge.WithDescription("Returns the merged client-facing configuration for an app "+
			"resolved by the publishable key provided via the 'key' query param. No authentication required."),
		forge.WithOperationID("getClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "Client configuration", authsome.ClientConfigResponse{}),
		forge.WithErrorResponses(),
	)
}

func (a *API) handleGetClientConfig(ctx forge.Context, req *GetClientConfigRequest) (*authsome.ClientConfigResponse, error) {
	if req.Key == "" {
		return nil, forge.BadRequest("publishable key is required")
	}

	app, err := a.engine.ResolveAppByPublicKey(ctx.Context(), req.Key)
	if err != nil {
		return nil, forge.NotFound("invalid publishable key")
	}

	config := a.engine.ClientConfig(ctx.Context(), app.ID)
	return config, nil
}
