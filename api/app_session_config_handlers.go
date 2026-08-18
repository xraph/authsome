package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
)

// ──────────────────────────────────────────────────
// App session config route registration
// ──────────────────────────────────────────────────

func (a *API) registerAppSessionConfigRoutes(router forge.Router) error {
	g := router.Group("/v1/admin",
		forge.WithGroupTags("admin", "app-session-config"),
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

	if err := g.GET("/apps/:appId/session-config", a.handleGetAppSessionConfig,
		forge.WithSummary("Get per-app session config"),
		forge.WithDescription("Returns the per-app session configuration overrides for the specified app."),
		forge.WithOperationID("getAppSessionConfig"),
		forge.WithResponseSchema(http.StatusOK, "App session config", appsessionconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PUT("/apps/:appId/session-config", a.handleSetAppSessionConfig,
		forge.WithSummary("Set per-app session config"),
		forge.WithDescription("Creates or updates the per-app session configuration overrides. Nil fields inherit from global/environment config."),
		forge.WithOperationID("setAppSessionConfig"),
		forge.WithRequestSchema(SetAppSessionConfigRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated app session config", appsessionconfig.Config{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/apps/:appId/session-config", a.handleDeleteAppSessionConfig,
		forge.WithSummary("Delete per-app session config"),
		forge.WithDescription("Removes the per-app session configuration overrides, reverting to global/environment defaults."),
		forge.WithOperationID("deleteAppSessionConfig"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// App session config handlers
// ──────────────────────────────────────────────────

func (a *API) handleGetAppSessionConfig(ctx forge.Context, req *GetAppSessionConfigRequest) (*appsessionconfig.Config, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	cfg, err := a.engine.Store().GetAppSessionConfig(ctx.Context(), appID)
	if err != nil {
		if errors.Is(err, appsessionconfig.ErrNotFound) {
			return nil, forge.NotFound("no session config found for this app")
		}
		return nil, mapError(err)
	}

	return cfg, nil
}

func (a *API) handleSetAppSessionConfig(ctx forge.Context, req *SetAppSessionConfigRequest) (*appsessionconfig.Config, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Try to load existing config to preserve the ID and creation timestamp.
	existing, err := a.engine.Store().GetAppSessionConfig(ctx.Context(), appID)
	if err != nil && !errors.Is(err, appsessionconfig.ErrNotFound) {
		return nil, mapError(err)
	}

	cfg := &appsessionconfig.Config{
		AppID:                  appID,
		TokenTTLSeconds:        req.TokenTTLSeconds,
		RefreshTokenTTLSeconds: req.RefreshTokenTTLSeconds,
		MaxActiveSessions:      req.MaxActiveSessions,
		RotateRefreshToken:     req.RotateRefreshToken,
		BindToIP:               req.BindToIP,
		BindToDevice:           req.BindToDevice,
		TokenFormat:            req.TokenFormat,
		UpdatedAt:              now,
	}

	if existing != nil {
		cfg.ID = existing.ID
		cfg.CreatedAt = existing.CreatedAt
	} else {
		cfg.ID = id.NewAppSessionConfigID()
		cfg.CreatedAt = now
	}

	if err := a.engine.Store().SetAppSessionConfig(ctx.Context(), cfg); err != nil {
		return nil, mapError(err)
	}

	return cfg, nil
}

func (a *API) handleDeleteAppSessionConfig(ctx forge.Context, req *DeleteAppSessionConfigRequest) (*StatusResponse, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if err := a.engine.Store().DeleteAppSessionConfig(ctx.Context(), appID); err != nil {
		if errors.Is(err, appsessionconfig.ErrNotFound) {
			return nil, forge.NotFound("no session config found for this app")
		}
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
