package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Public client config route registration
// ──────────────────────────────────────────────────

func (a *API) registerClientConfigRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base, forge.WithGroupTags("Client Config"))

	return g.GET("/client-config", a.handleGetClientConfig,
		forge.WithSummary("Get client configuration"),
		forge.WithDescription("Returns the merged client-facing configuration for an app. "+
			"If a publishable key is provided via the 'key' query param, the config is resolved "+
			"for that app. Otherwise, the platform app config is returned. No authentication required."),
		forge.WithOperationID("getClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "Client configuration", map[string]any{}),
		forge.WithErrorResponses(),
	)
}

func (a *API) handleGetClientConfig(ctx forge.Context, req *GetClientConfigRequest) (*map[string]any, error) {
	var appID id.AppID

	if req.Key != "" {
		// Resolve app by publishable key.
		app, err := a.engine.Store().GetAppByPublishableKey(ctx.Context(), req.Key)
		if err != nil {
			return nil, forge.NotFound("invalid publishable key")
		}
		appID = app.ID
	} else {
		// Fall back to platform app.
		appID = a.engine.PlatformAppID()
		if appID.IsNil() {
			return nil, forge.NotFound("no platform app configured")
		}
	}

	config := a.engine.ClientConfig(ctx.Context(), appID)
	return nil, ctx.JSON(http.StatusOK, config)
}
