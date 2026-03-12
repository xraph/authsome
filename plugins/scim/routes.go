package scim

import (
	"fmt"

	"github.com/xraph/forge"
)

// RegisterRoutes registers SCIM 2.0 API routes on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("scim: expected forge.Router, got %T", r)
	}

	prefix := p.config.BasePath

	// SCIM discovery endpoints.
	scim := router.Group(prefix, forge.WithGroupTags("SCIM 2.0"))

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

	return scim.DELETE("/Groups/:groupId", p.handleDeleteGroup,
		forge.WithSummary("Delete SCIM Group"),
		forge.WithOperationID("scimDeleteGroup"),
	)
}
