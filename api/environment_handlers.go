package api

import (
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Environment route registration
// ──────────────────────────────────────────────────

func (a *API) registerEnvironmentRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	g := router.Group(base, forge.WithGroupTags("environments"))

	if err := g.POST("/environments", a.handleCreateEnvironment,
		forge.WithSummary("Create environment"),
		forge.WithDescription("Creates a new environment for an app."),
		forge.WithOperationID("createEnvironment"),
		forge.WithRequestSchema(CreateEnvironmentRequest{}),
		forge.WithCreatedResponse(environment.Environment{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/environments", a.handleListEnvironments,
		forge.WithSummary("List environments"),
		forge.WithDescription("Returns all environments for an app."),
		forge.WithOperationID("listEnvironments"),
		forge.WithResponseSchema(http.StatusOK, "Environment list", EnvironmentListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/environments/:envId", a.handleGetEnvironment,
		forge.WithSummary("Get environment"),
		forge.WithDescription("Returns details of a specific environment."),
		forge.WithOperationID("getEnvironment"),
		forge.WithResponseSchema(http.StatusOK, "Environment details", environment.Environment{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/environments/:envId", a.handleUpdateEnvironment,
		forge.WithSummary("Update environment"),
		forge.WithDescription("Updates an environment's name, description, or settings."),
		forge.WithOperationID("updateEnvironment"),
		forge.WithRequestSchema(UpdateEnvironmentRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated environment", environment.Environment{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/environments/:envId", a.handleDeleteEnvironment,
		forge.WithSummary("Delete environment"),
		forge.WithDescription("Deletes an environment. Cannot delete the default environment."),
		forge.WithOperationID("deleteEnvironment"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/environments/:envId/clone", a.handleCloneEnvironment,
		forge.WithSummary("Clone environment"),
		forge.WithDescription("Clones an environment's config and structure (roles, permissions, webhooks) into a new environment."),
		forge.WithOperationID("cloneEnvironment"),
		forge.WithRequestSchema(CloneEnvironmentRequest{}),
		forge.WithCreatedResponse(CloneEnvironmentResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/environments/:envId/settings", a.handleGetEnvironmentSettings,
		forge.WithSummary("Get environment settings"),
		forge.WithDescription("Returns the resolved settings for an environment (type defaults merged with overrides)."),
		forge.WithOperationID("getEnvironmentSettings"),
		forge.WithResponseSchema(http.StatusOK, "Resolved settings", EnvironmentSettingsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/environments/:envId/settings", a.handleUpdateEnvironmentSettings,
		forge.WithSummary("Update environment settings"),
		forge.WithDescription("Updates per-environment settings overrides."),
		forge.WithOperationID("updateEnvironmentSettings"),
		forge.WithRequestSchema(environment.Settings{}),
		forge.WithResponseSchema(http.StatusOK, "Updated settings", environment.Environment{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/environments/:envId/set-default", a.handleSetDefaultEnvironment,
		forge.WithSummary("Set default environment"),
		forge.WithDescription("Sets an environment as the default for its app."),
		forge.WithOperationID("setDefaultEnvironment"),
		forge.WithResponseSchema(http.StatusOK, "Default set", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Environment handlers
// ──────────────────────────────────────────────────

func (a *API) handleCreateEnvironment(ctx forge.Context, req *CreateEnvironmentRequest) (*environment.Environment, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.Slug == "" {
		return nil, forge.BadRequest("slug is required")
	}
	if req.Type == "" {
		return nil, forge.BadRequest("type is required")
	}

	envType := environment.Type(req.Type)
	if !envType.IsValid() {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment type: %s (must be development, staging, or production)", req.Type))
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	env := &environment.Environment{
		ID:          id.NewEnvironmentID(),
		AppID:       appID,
		Name:        req.Name,
		Slug:        req.Slug,
		Type:        envType,
		Color:       envType.DefaultColor(),
		Description: req.Description,
	}

	if req.Color != "" {
		env.Color = req.Color
	}

	// Merge type defaults with any explicit settings.
	typeDefaults := environment.DefaultSettingsForType(envType)
	env.Settings = environment.MergeSettings(typeDefaults, req.Settings)

	if err := a.engine.Store().CreateEnvironment(ctx.Context(), env); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusCreated, env)
}

func (a *API) handleListEnvironments(ctx forge.Context, req *ListEnvironmentsRequest) (*EnvironmentListResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	envs, err := a.engine.Store().ListEnvironments(ctx.Context(), appID)
	if err != nil {
		return nil, mapError(err)
	}

	if envs == nil {
		envs = []*environment.Environment{}
	}
	resp := &EnvironmentListResponse{Environments: envs, Total: len(envs)}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleGetEnvironment(ctx forge.Context, _ *GetEnvironmentRequest) (*environment.Environment, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}

	return env, nil
}

func (a *API) handleUpdateEnvironment(ctx forge.Context, req *UpdateEnvironmentRequest) (*environment.Environment, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.Name != nil {
		env.Name = *req.Name
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.Color != nil {
		env.Color = *req.Color
	}

	if err := a.engine.Store().UpdateEnvironment(ctx.Context(), env); err != nil {
		return nil, mapError(err)
	}

	return env, nil
}

func (a *API) handleDeleteEnvironment(ctx forge.Context, _ *DeleteEnvironmentRequest) (*StatusResponse, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	// Check if this is the default environment.
	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}
	if env.IsDefault {
		return nil, forge.BadRequest("cannot delete the default environment")
	}

	if err := a.engine.Store().DeleteEnvironment(ctx.Context(), envID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleCloneEnvironment(ctx forge.Context, req *CloneEnvironmentRequest) (*CloneEnvironmentResponse, error) {
	srcEnvID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.Slug == "" {
		return nil, forge.BadRequest("slug is required")
	}
	if req.Type == "" {
		return nil, forge.BadRequest("type is required")
	}

	envType := environment.Type(req.Type)
	if !envType.IsValid() {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment type: %s", req.Type))
	}

	cloneReq := environment.CloneRequest{
		SourceEnvID:        srcEnvID,
		Name:               req.Name,
		Slug:               req.Slug,
		Type:               envType,
		Description:        req.Description,
		SettingsOverride:   req.Settings,
		WebhookURLOverride: req.WebhookURLOverride,
	}

	result, err := a.engine.CloneEnvironment(ctx.Context(), cloneReq)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &CloneEnvironmentResponse{
		Environment:       result.Environment,
		RolesCloned:       result.RolesCloned,
		PermissionsCloned: result.PermissionsCloned,
		WebhooksCloned:    result.WebhooksCloned,
		RoleIDMap:         result.RoleIDMap,
	}
	return nil, ctx.JSON(http.StatusCreated, resp)
}

func (a *API) handleGetEnvironmentSettings(ctx forge.Context, _ *GetEnvironmentSettingsRequest) (*EnvironmentSettingsResponse, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}

	// Resolve effective settings: type defaults + per-env overrides.
	typeDefaults := environment.DefaultSettingsForType(env.Type)
	effective := environment.MergeSettings(typeDefaults, env.Settings)

	// Also check if settings are available from middleware context.
	if ctxSettings, ok := middleware.EnvironmentSettingsFrom(ctx.Context()); ok {
		effective = ctxSettings
	}

	resp := &EnvironmentSettingsResponse{
		Settings:     effective,
		TypeDefaults: typeDefaults,
		Overrides:    env.Settings,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleUpdateEnvironmentSettings(ctx forge.Context, req *environment.Settings) (*environment.Environment, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}

	// Merge new overrides on top of existing.
	env.Settings = environment.MergeSettings(env.Settings, req)

	if err := a.engine.Store().UpdateEnvironment(ctx.Context(), env); err != nil {
		return nil, mapError(err)
	}

	return env, nil
}

func (a *API) handleSetDefaultEnvironment(ctx forge.Context, _ *SetDefaultEnvironmentRequest) (*StatusResponse, error) {
	envID, err := id.ParseEnvironmentID(ctx.Param("envId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid environment id: %v", err))
	}

	env, err := a.engine.Store().GetEnvironment(ctx.Context(), envID)
	if err != nil {
		return nil, mapError(err)
	}

	if err := a.engine.Store().SetDefaultEnvironment(ctx.Context(), env.AppID, envID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "default environment updated"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
