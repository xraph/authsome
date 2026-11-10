package scim

import (
	"net/http"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
)

// RegisterRoutes registers SCIM 2.0 API routes on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	prefix := p.config.BasePath

	// One budget for the whole SCIM surface, keyed by client address: an
	// IdP provisions in bursts, a guessed bearer token does not get to
	// try unlimited times.
	groupOpts := []forge.GroupOption{forge.WithGroupTags("SCIM 2.0")}
	if mw := authsome.PluginRateLimitMiddleware(p.engine, func(c authsome.RateLimitConfig) int { return c.SCIMLimit }); mw != nil {
		groupOpts = append(groupOpts, forge.WithGroupMiddleware(mw))
	}
	scim := router.Group(prefix, groupOpts...)

	if err := scim.GET("/ServiceProviderConfig", p.handleServiceProviderConfig,
		forge.WithSummary("SCIM Service Provider Configuration"),
		forge.WithOperationID("scimServiceProviderConfig"),
	); err != nil {
		return err
	}

	if err := scim.GET("/Schemas", p.handleSchemas,
		forge.WithSummary("SCIM Schemas"),
		forge.WithOperationID("scimSchemas"),
	); err != nil {
		return err
	}

	if err := scim.GET("/ResourceTypes", p.handleResourceTypes,
		forge.WithSummary("SCIM Resource Types"),
		forge.WithOperationID("scimResourceTypes"),
	); err != nil {
		return err
	}

	// User endpoints (scoped by config via bearer token).
	if err := scim.GET("/Users", p.handleListUsers,
		forge.WithSummary("List SCIM Users"),
		forge.WithOperationID("scimListUsers"),
	); err != nil {
		return err
	}

	if err := scim.GET("/Users/:userId", p.handleGetUser,
		forge.WithSummary("Get SCIM User"),
		forge.WithOperationID("scimGetUser"),
	); err != nil {
		return err
	}

	if err := scim.POST("/Users", p.handleCreateUser,
		forge.WithSummary("Create SCIM User"),
		forge.WithOperationID("scimCreateUser"),
	); err != nil {
		return err
	}

	if err := scim.PUT("/Users/:userId", p.handleReplaceUser,
		forge.WithSummary("Replace SCIM User"),
		forge.WithOperationID("scimReplaceUser"),
	); err != nil {
		return err
	}

	if err := scim.PATCH("/Users/:userId", p.handlePatchUser,
		forge.WithSummary("Patch SCIM User"),
		forge.WithOperationID("scimPatchUser"),
	); err != nil {
		return err
	}

	if err := scim.DELETE("/Users/:userId", p.handleDeleteUser,
		forge.WithSummary("Delete SCIM User"),
		forge.WithOperationID("scimDeleteUser"),
	); err != nil {
		return err
	}

	// Group endpoints.
	if err := scim.GET("/Groups", p.handleListGroups,
		forge.WithSummary("List SCIM Groups"),
		forge.WithOperationID("scimListGroups"),
	); err != nil {
		return err
	}

	if err := scim.GET("/Groups/:groupId", p.handleGetGroup,
		forge.WithSummary("Get SCIM Group"),
		forge.WithOperationID("scimGetGroup"),
	); err != nil {
		return err
	}

	if err := scim.POST("/Groups", p.handleCreateGroup,
		forge.WithSummary("Create SCIM Group"),
		forge.WithOperationID("scimCreateGroup"),
	); err != nil {
		return err
	}

	if err := scim.PUT("/Groups/:groupId", p.handleReplaceGroup,
		forge.WithSummary("Replace SCIM Group"),
		forge.WithOperationID("scimReplaceGroup"),
	); err != nil {
		return err
	}

	if err := scim.PATCH("/Groups/:groupId", p.handlePatchGroup,
		forge.WithSummary("Patch SCIM Group"),
		forge.WithOperationID("scimPatchGroup"),
	); err != nil {
		return err
	}

	if err := scim.DELETE("/Groups/:groupId", p.handleDeleteGroup,
		forge.WithSummary("Delete SCIM Group"),
		forge.WithOperationID("scimDeleteGroup"),
	); err != nil {
		return err
	}

	return p.registerAdminRoutes(router)
}

// registerAdminRoutes mounts the dashboard surface: creating a SCIM
// configuration, minting a token for an identity provider, rotating it,
// revoking it.
//
// These live under /v1 rather than the SCIM base path on purpose. The SCIM
// group authenticates with a provisioning bearer token, and an identity
// provider holding one has no business creating more. These authenticate as
// a signed-in organization owner or admin instead.
func (p *Plugin) registerAdminRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("SCIM Admin"))

	if err := g.GET("/orgs/:orgId/scim/configs", p.handleListConfigs,
		forge.WithSummary("List SCIM configurations"),
		forge.WithDescription("Lists an organization's SCIM configurations. Requires an organization owner or admin."),
		forge.WithOperationID("listSCIMConfigs"),
		forge.WithResponseSchema(http.StatusOK, "SCIM configurations", ConfigListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/:orgId/scim/configs", p.handleCreateConfig,
		forge.WithSummary("Create SCIM configuration"),
		forge.WithDescription("Creates a SCIM configuration for an organization, which an identity provider then provisions into."),
		forge.WithOperationID("createSCIMConfig"),
		forge.WithRequestSchema(CreateSCIMConfigRequest{}),
		forge.WithCreatedResponse(SCIMConfig{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/scim/configs/:configId", p.handleGetConfig,
		forge.WithSummary("Get SCIM configuration"),
		forge.WithOperationID("getSCIMConfig"),
		forge.WithResponseSchema(http.StatusOK, "SCIM configuration", SCIMConfig{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/scim/configs/:configId", p.handleUpdateConfig,
		forge.WithSummary("Update SCIM configuration"),
		forge.WithDescription("Changes only the fields present in the body."),
		forge.WithOperationID("updateSCIMConfig"),
		forge.WithRequestSchema(UpdateSCIMConfigRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated configuration", SCIMConfig{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/scim/configs/:configId", p.handleDeleteConfig,
		forge.WithSummary("Delete SCIM configuration"),
		forge.WithDescription("Removes the configuration and stops accepting provisioning for it."),
		forge.WithOperationID("deleteSCIMConfig"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/scim/configs/:configId/tokens", p.handleListTokens,
		forge.WithSummary("List SCIM tokens"),
		forge.WithDescription("Lists a configuration's tokens. The token values themselves are never returned."),
		forge.WithOperationID("listSCIMTokens"),
		forge.WithResponseSchema(http.StatusOK, "Tokens", TokenListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/scim/configs/:configId/tokens", p.handleCreateToken,
		forge.WithSummary("Create SCIM token"),
		forge.WithDescription("Mints a bearer token for an identity provider. The plaintext is returned once and never again."),
		forge.WithOperationID("createSCIMToken"),
		forge.WithRequestSchema(CreateSCIMTokenRequest{}),
		forge.WithCreatedResponse(TokenCreatedResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/scim/tokens/:tokenId/rotate", p.handleRotateToken,
		forge.WithSummary("Rotate SCIM token"),
		forge.WithDescription("Mints a replacement and optionally keeps the old token working for a grace period, so an identity provider can be updated without a gap in provisioning."),
		forge.WithOperationID("rotateSCIMToken"),
		forge.WithRequestSchema(RotateSCIMTokenRequest{}),
		forge.WithCreatedResponse(TokenCreatedResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/scim/tokens/:tokenId", p.handleRevokeToken,
		forge.WithSummary("Revoke SCIM token"),
		forge.WithOperationID("revokeSCIMToken"),
		forge.WithResponseSchema(http.StatusOK, "Revoked", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}
