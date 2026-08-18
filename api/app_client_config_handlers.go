package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
)

// ──────────────────────────────────────────────────
// Admin app client config route registration
// ──────────────────────────────────────────────────

func (a *API) registerAppClientConfigRoutes(router forge.Router) error {
	g := router.Group("/v1/admin",
		forge.WithGroupTags("admin", "app-client-config"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequirePermission(a.engine, "manage", "app"),
		),
		// Declared so the requirement the middleware above enforces reaches
		// the OpenAPI document and the generated clients. Declaring does not
		// enforce: the middleware is still what refuses the request.
		forge.WithGroupAuth("session", "session-cookie"),
		forge.WithGroupAllPermissions(rbac.PermissionString("manage", "app")),
	)

	if err := g.GET("/apps/:appId/client-config", a.handleGetAppClientConfig,
		forge.WithSummary("Get per-app client config overrides"),
		forge.WithDescription("Returns the per-app client configuration overrides. Nil fields inherit from plugin defaults."),
		forge.WithOperationID("getAppClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "App client config", appclientconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PUT("/apps/:appId/client-config", a.handleSetAppClientConfig,
		forge.WithSummary("Set per-app client config overrides"),
		forge.WithDescription("Creates or updates per-app client configuration overrides. Nil fields inherit from plugin defaults."),
		forge.WithOperationID("setAppClientConfig"),
		forge.WithRequestSchema(SetAppClientConfigRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated app client config", appclientconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/apps/:appId/client-config", a.handleDeleteAppClientConfig,
		forge.WithSummary("Delete per-app client config overrides"),
		forge.WithDescription("Removes per-app client configuration overrides, reverting to plugin defaults."),
		forge.WithOperationID("deleteAppClientConfig"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Admin app client config handlers
// ──────────────────────────────────────────────────

func (a *API) handleGetAppClientConfig(ctx forge.Context, req *GetAppClientConfigRequest) (*appclientconfig.Config, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	cfg, err := a.engine.Store().GetAppClientConfig(ctx.Context(), appID)
	if err != nil {
		if errors.Is(err, appclientconfig.ErrNotFound) {
			return nil, forge.NotFound("no client config found for this app")
		}
		return nil, mapError(err)
	}

	return cfg, nil
}

func (a *API) handleSetAppClientConfig(ctx forge.Context, req *SetAppClientConfigRequest) (*appclientconfig.Config, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Try to load existing config to preserve the ID and creation timestamp.
	existing, err := a.engine.Store().GetAppClientConfig(ctx.Context(), appID)
	if err != nil && !errors.Is(err, appclientconfig.ErrNotFound) {
		return nil, mapError(err)
	}

	cfg := &appclientconfig.Config{
		AppID:                    appID,
		SignupEnabled:            req.SignupEnabled,
		PasswordEnabled:          req.PasswordEnabled,
		PasskeyEnabled:           req.PasskeyEnabled,
		MagicLinkEnabled:         req.MagicLinkEnabled,
		MFAEnabled:               req.MFAEnabled,
		MFARequired:              req.MFARequired,
		SSOEnabled:               req.SSOEnabled,
		SocialEnabled:            req.SocialEnabled,
		WaitlistEnabled:          req.WaitlistEnabled,
		RequireEmailVerification: req.RequireEmailVerification,
		SocialProviders:          req.SocialProviders,
		MFAMethods:               req.MFAMethods,
		AppName:                  req.AppName,
		LogoURL:                  req.LogoURL,
		UpdatedAt:                now,
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

	return cfg, nil
}

func (a *API) handleDeleteAppClientConfig(ctx forge.Context, req *DeleteAppClientConfigRequest) (*StatusResponse, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
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
