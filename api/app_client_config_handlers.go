package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Admin app client config route registration
// ──────────────────────────────────────────────────

func (a *API) registerAppClientConfigRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base+"/admin",
		forge.WithGroupTags("admin", "app-client-config"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequireAnyRole(a.engine, "admin", "super_admin"),
		),
	)

	if err := g.GET("/apps/:appId/client-config", a.handleGetAppClientConfig,
		forge.WithSummary("Get per-app client config overrides"),
		forge.WithDescription("Returns the per-app client configuration overrides. Nil fields inherit from plugin defaults. Requires admin role."),
		forge.WithOperationID("getAppClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "App client config", appclientconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PUT("/apps/:appId/client-config", a.handleSetAppClientConfig,
		forge.WithSummary("Set per-app client config overrides"),
		forge.WithDescription("Creates or updates per-app client configuration overrides. Nil fields inherit from plugin defaults. Requires admin role."),
		forge.WithOperationID("setAppClientConfig"),
		forge.WithRequestSchema(SetAppClientConfigRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated app client config", appclientconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/apps/:appId/client-config", a.handleDeleteAppClientConfig,
		forge.WithSummary("Delete per-app client config overrides"),
		forge.WithDescription("Removes per-app client configuration overrides, reverting to plugin defaults. Requires admin role."),
		forge.WithOperationID("deleteAppClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Admin app client config handlers
// ──────────────────────────────────────────────────

func (a *API) handleGetAppClientConfig(ctx forge.Context, req *GetAppClientConfigRequest) (*appclientconfig.Config, error) {
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	cfg, err := a.engine.Store().GetAppClientConfig(ctx.Context(), appID)
	if err != nil {
		if errors.Is(err, appclientconfig.ErrNotFound) {
			return nil, forge.NotFound("no client config found for this app")
		}
		return nil, mapError(err)
	}

	return cfg, ctx.JSON(http.StatusOK, cfg)
}

func (a *API) handleSetAppClientConfig(ctx forge.Context, req *SetAppClientConfigRequest) (*appclientconfig.Config, error) {
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	now := time.Now().UTC()

	// Try to load existing config to preserve the ID and creation timestamp.
	existing, err := a.engine.Store().GetAppClientConfig(ctx.Context(), appID)
	if err != nil && !errors.Is(err, appclientconfig.ErrNotFound) {
		return nil, mapError(err)
	}

	cfg := &appclientconfig.Config{
		AppID:            appID,
		PasswordEnabled:  req.PasswordEnabled,
		PasskeyEnabled:   req.PasskeyEnabled,
		MagicLinkEnabled: req.MagicLinkEnabled,
		MFAEnabled:       req.MFAEnabled,
		SSOEnabled:       req.SSOEnabled,
		SocialEnabled:    req.SocialEnabled,
		SocialProviders:  req.SocialProviders,
		MFAMethods:       req.MFAMethods,
		AppName:          req.AppName,
		LogoURL:          req.LogoURL,
		UpdatedAt:        now,
	}

	if existing != nil {
		cfg.ID = existing.ID
		cfg.CreatedAt = existing.CreatedAt
	} else {
		cfg.ID = id.NewAppClientConfigID()
		cfg.CreatedAt = now
	}

	if err := a.engine.Store().SetAppClientConfig(ctx.Context(), cfg); err != nil {
		return nil, mapError(err)
	}

	return cfg, ctx.JSON(http.StatusOK, cfg)
}

func (a *API) handleDeleteAppClientConfig(ctx forge.Context, req *DeleteAppClientConfigRequest) (*StatusResponse, error) {
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	if err := a.engine.Store().DeleteAppClientConfig(ctx.Context(), appID); err != nil {
		if errors.Is(err, appclientconfig.ErrNotFound) {
			return nil, forge.NotFound("no client config found for this app")
		}
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
