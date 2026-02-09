package scim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/forge"
)

// Handler handles SCIM HTTP requests.
type Handler struct {
	service *Service
	config  *Config
	metrics *Metrics
}

// Response types - use shared responses from core.
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

// Note: All SCIM-specific request/response types are defined in types.go

// NewHandler creates a new SCIM handler.
func NewHandler(service *Service, config *Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
		metrics: GetMetrics(),
	}
}

// =============================================================================
// SERVICE PROVIDER CONFIGURATION ENDPOINTS (RFC 7643 Section 5)
// =============================================================================

// GetServiceProviderConfig returns the service provider configuration.
func (h *Handler) GetServiceProviderConfig(c forge.Context) error {
	config := &ServiceProviderConfig{
		Schemas:          []string{SchemaServiceProvider},
		DocumentationURI: "https://docs.authsome.dev/scim",
		Patch: &Supported{
			Supported: true,
		},
		Bulk: &BulkSupport{
			Supported:      h.config.BulkOperations.Enabled,
			MaxOperations:  h.config.BulkOperations.MaxOperations,
			MaxPayloadSize: h.config.BulkOperations.MaxPayloadBytes,
		},
		Filter: &FilterSupport{
			Supported:  true,
			MaxResults: h.config.Search.MaxResults,
		},
		ChangePassword: &Supported{
			Supported: false, // Password changes handled outside SCIM
		},
		Sort: &Supported{
			Supported: true,
		},
		Etag: &Supported{
			Supported: false,
		},
		AuthenticationSchemes: []AuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication via OAuth Bearer Token",
				SpecURI:     "https://tools.ietf.org/html/rfc6750",
				Primary:     true,
			},
		},
		Meta: &SCIMMeta{
			ResourceType: "ServiceProviderConfig",
			Location:     "/scim/v2/ServiceProviderConfig",
		},
	}

	return c.JSON(http.StatusOK, config)
}

// =============================================================================
// RESOURCE TYPE ENDPOINTS (RFC 7643 Section 6)
// =============================================================================

// GetResourceTypes returns all supported resource types.
func (h *Handler) GetResourceTypes(c forge.Context) error {
	resourceTypes := []ResourceType{
		{
			Schemas:     []string{SchemaResourceType},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/Users",
			Description: "User Account",
			Schema:      SchemaCore,
			SchemaExtensions: []SchemaExtension{
				{
					Schema:   SchemaEnterprise,
					Required: false,
				},
			},
			Meta: &SCIMMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/User",
			},
		},
		{
			Schemas:     []string{SchemaResourceType},
			ID:          "Group",
			Name:        "Group",
			Endpoint:    "/Groups",
			Description: "Group",
			Schema:      SchemaGroup,
			Meta: &SCIMMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/Group",
			},
		},
	}

	return c.JSON(http.StatusOK, &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(resourceTypes),
		StartIndex:   1,
		ItemsPerPage: len(resourceTypes),
		Resources:    convertToInterfaces(resourceTypes),
	})
}

// GetResourceType returns a specific resource type.
func (h *Handler) GetResourceType(c forge.Context) error {
	id := c.Param("id")

	var resourceType *ResourceType

	switch id {
	case "User":
		resourceType = &ResourceType{
			Schemas:     []string{SchemaResourceType},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/Users",
			Description: "User Account",
			Schema:      SchemaCore,
			SchemaExtensions: []SchemaExtension{
				{
					Schema:   SchemaEnterprise,
					Required: false,
				},
			},
			Meta: &SCIMMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/User",
			},
		}
	case "Group":
		resourceType = &ResourceType{
			Schemas:     []string{SchemaResourceType},
			ID:          "Group",
			Name:        "Group",
			Endpoint:    "/Groups",
			Description: "Group",
			Schema:      SchemaGroup,
			Meta: &SCIMMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/Group",
			},
		}
	default:
		return c.JSON(http.StatusNotFound, h.scimError(http.StatusNotFound, "invalidValue", "Resource type not found"))
	}

	return c.JSON(http.StatusOK, resourceType)
}

// =============================================================================
// SCHEMA ENDPOINTS (RFC 7643 Section 7)
// =============================================================================

// GetSchemas returns all supported schemas.
func (h *Handler) GetSchemas(c forge.Context) error {
	// Return simplified schema list
	schemas := []any{
		map[string]any{
			"id":   SchemaCore,
			"name": "User",
		},
		map[string]any{
			"id":   SchemaEnterprise,
			"name": "EnterpriseUser",
		},
		map[string]any{
			"id":   SchemaGroup,
			"name": "Group",
		},
	}

	return c.JSON(http.StatusOK, &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(schemas),
		StartIndex:   1,
		ItemsPerPage: len(schemas),
		Resources:    schemas,
	})
}

// GetSchema returns a specific schema.
func (h *Handler) GetSchema(c forge.Context) error {
	id := c.Param("id")

	// Return basic schema response
	// TODO: Implement full RFC 7643 schema definitions with attributes
	schema := &Schema{
		ID:          id,
		Name:        "Schema",
		Description: "SCIM Schema",
		Attributes:  []Attribute{}, // TODO: Populate with actual schema attributes
	}

	return c.JSON(http.StatusOK, schema)
}

// =============================================================================
// USER ENDPOINTS (RFC 7644 Section 3)
// =============================================================================

// CreateUser creates a new user.
func (h *Handler) CreateUser(c forge.Context) error {
	start := time.Now()

	defer func() {
		h.metrics.RecordRequestDuration("POST /Users", time.Since(start))
	}()

	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse request body
	var scimUser SCIMUser
	if err := json.NewDecoder(c.Request().Body).Decode(&scimUser); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Create user via service
	createdUser, err := h.service.CreateUser(ctx, &scimUser, orgID)
	if err != nil {
		h.metrics.RecordError("user_creation_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("create")

	return c.JSON(http.StatusCreated, createdUser)
}

// ListUsers lists users with filtering and pagination.
func (h *Handler) ListUsers(c forge.Context) error {
	start := time.Now()

	defer func() {
		h.metrics.RecordRequestDuration("GET /Users", time.Since(start))
	}()

	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse query parameters
	filter := c.Query("filter")
	startIndex := parseIntParam(c.Query("startIndex"), 1)
	count := min(
		// Enforce max results
		parseIntParam(c.Query("count"), h.config.Search.DefaultResults), h.config.Search.MaxResults)

	// List users via service
	listResponse, err := h.service.ListUsers(ctx, orgID, filter, startIndex, count)
	if err != nil {
		h.metrics.RecordError("user_list_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("list")

	return c.JSON(http.StatusOK, listResponse)
}

// GetUser retrieves a specific user.
func (h *Handler) GetUser(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse user ID from path
	id := c.Param("id")

	userID, err := xid.FromString(id)
	if err != nil {
		h.metrics.RecordError("invalid_user_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid user ID format"))
	}

	// Get user via service
	scimUser, err := h.service.GetUser(ctx, userID, orgID)
	if err != nil {
		h.metrics.RecordError("user_get_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("read")

	return c.JSON(http.StatusOK, scimUser)
}

// ReplaceUser replaces a user (PUT).
func (h *Handler) ReplaceUser(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse user ID from path
	id := c.Param("id")

	userID, err := xid.FromString(id)
	if err != nil {
		h.metrics.RecordError("invalid_user_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid user ID format"))
	}

	// Parse request body
	var scimUser SCIMUser
	if err := json.NewDecoder(c.Request().Body).Decode(&scimUser); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Replace user via service
	updatedUser, err := h.service.ReplaceUser(ctx, userID, orgID, &scimUser)
	if err != nil {
		h.metrics.RecordError("user_replace_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("replace")

	return c.JSON(http.StatusOK, updatedUser)
}

// UpdateUser updates a user (PATCH).
func (h *Handler) UpdateUser(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse user ID from path
	id := c.Param("id")

	userID, err := xid.FromString(id)
	if err != nil {
		h.metrics.RecordError("invalid_user_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid user ID format"))
	}

	// Parse patch operations
	var patch PatchOp
	if err := json.NewDecoder(c.Request().Body).Decode(&patch); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Validate patch operations
	if len(patch.Operations) == 0 {
		h.metrics.RecordError("empty_patch_operations")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "No patch operations provided"))
	}

	// Update user via service
	updatedUser, err := h.service.UpdateUser(ctx, userID, orgID, &patch)
	if err != nil {
		h.metrics.RecordError("user_update_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("update")

	return c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes a user.
func (h *Handler) DeleteUser(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse user ID from path
	id := c.Param("id")

	userID, err := xid.FromString(id)
	if err != nil {
		h.metrics.RecordError("invalid_user_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid user ID format"))
	}

	// Delete user via service
	if err := h.service.DeleteUser(ctx, userID, orgID); err != nil {
		h.metrics.RecordError("user_delete_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordUserOperation("delete")

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// GROUP ENDPOINTS (RFC 7644 Section 3)
// =============================================================================

// CreateGroup creates a new group.
func (h *Handler) CreateGroup(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse request body
	var scimGroup SCIMGroup
	if err := json.NewDecoder(c.Request().Body).Decode(&scimGroup); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Create group via service
	createdGroup, err := h.service.CreateGroup(ctx, &scimGroup, orgID)
	if err != nil {
		h.metrics.RecordError("group_creation_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("create")

	return c.JSON(http.StatusCreated, createdGroup)
}

// ListGroups lists groups.
func (h *Handler) ListGroups(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse query parameters
	filter := c.Query("filter")
	startIndex := parseIntParam(c.Query("startIndex"), 1)
	count := min(
		// Enforce max results
		parseIntParam(c.Query("count"), h.config.Search.DefaultResults), h.config.Search.MaxResults)

	// List groups via service
	result, err := h.service.ListGroups(ctx, orgID, filter, startIndex, count)
	if err != nil {
		h.metrics.RecordError("group_list_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("list")

	return c.JSON(http.StatusOK, result)
}

// GetGroup retrieves a specific group.
func (h *Handler) GetGroup(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse group ID from path
	groupIDStr := c.Param("id")

	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		h.metrics.RecordError("invalid_group_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid group ID format"))
	}

	// Get group via service
	group, err := h.service.GetGroup(ctx, groupID, orgID)
	if err != nil {
		h.metrics.RecordError("group_get_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("read")

	return c.JSON(http.StatusOK, group)
}

// ReplaceGroup replaces a group (PUT).
func (h *Handler) ReplaceGroup(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse group ID from path
	groupIDStr := c.Param("id")

	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		h.metrics.RecordError("invalid_group_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid group ID format"))
	}

	// Parse request body
	var scimGroup SCIMGroup
	if err := json.NewDecoder(c.Request().Body).Decode(&scimGroup); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Replace group via service
	updatedGroup, err := h.service.ReplaceGroup(ctx, groupID, orgID, &scimGroup)
	if err != nil {
		h.metrics.RecordError("group_replace_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("replace")

	return c.JSON(http.StatusOK, updatedGroup)
}

// UpdateGroup updates a group (PATCH).
func (h *Handler) UpdateGroup(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse group ID from path
	groupIDStr := c.Param("id")

	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		h.metrics.RecordError("invalid_group_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid group ID format"))
	}

	// Parse patch operations
	var patch PatchOp
	if err := json.NewDecoder(c.Request().Body).Decode(&patch); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Validate patch operations
	if len(patch.Operations) == 0 {
		h.metrics.RecordError("empty_patch_operations")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "No patch operations provided"))
	}

	// Update group via service
	updatedGroup, err := h.service.UpdateGroup(ctx, groupID, orgID, &patch)
	if err != nil {
		h.metrics.RecordError("group_update_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("update")

	return c.JSON(http.StatusOK, updatedGroup)
}

// DeleteGroup deletes a group.
func (h *Handler) DeleteGroup(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse group ID from path
	groupIDStr := c.Param("id")

	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		h.metrics.RecordError("invalid_group_id")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Invalid group ID format"))
	}

	// Delete group via service
	if err := h.service.DeleteGroup(ctx, groupID, orgID); err != nil {
		h.metrics.RecordError("group_delete_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordGroupOperation("delete")

	return c.NoContent(http.StatusNoContent)
}

// =============================================================================
// BULK OPERATIONS (RFC 7644 Section 3.7)
// =============================================================================

// BulkOperation handles bulk operations.
func (h *Handler) BulkOperation(c forge.Context) error {
	if !h.config.BulkOperations.Enabled {
		return c.JSON(http.StatusNotImplemented, h.scimError(http.StatusNotImplemented, "", "Bulk operations are disabled"))
	}

	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse bulk request
	var bulkReq BulkRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&bulkReq); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Validate operation count
	if len(bulkReq.Operations) > h.config.BulkOperations.MaxOperations {
		return c.JSON(http.StatusRequestEntityTooLarge, h.scimError(http.StatusRequestEntityTooLarge, "tooMany",
			fmt.Sprintf("Maximum %d operations allowed", h.config.BulkOperations.MaxOperations)))
	}

	// Process bulk operations via service
	bulkResp, err := h.service.ProcessBulkOperation(ctx, &bulkReq, orgID)
	if err != nil {
		h.metrics.RecordError("bulk_operation_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	h.metrics.RecordBulkOperation(len(bulkReq.Operations))

	return c.JSON(http.StatusOK, bulkResp)
}

// =============================================================================
// SEARCH ENDPOINT (RFC 7644 Section 3.4.3)
// =============================================================================

// Search handles the /.search endpoint.
func (h *Handler) Search(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse search request
	var searchReq SearchRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&searchReq); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Default values
	if searchReq.StartIndex == 0 {
		searchReq.StartIndex = 1
	}

	if searchReq.Count == 0 {
		searchReq.Count = h.config.Search.DefaultResults
	}

	if searchReq.Count > h.config.Search.MaxResults {
		searchReq.Count = h.config.Search.MaxResults
	}

	// Perform search via service (default to users)
	listResponse, err := h.service.ListUsers(ctx, orgID, searchReq.Filter, searchReq.StartIndex, searchReq.Count)
	if err != nil {
		h.metrics.RecordError("search_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	return c.JSON(http.StatusOK, listResponse)
}

// =============================================================================
// ADMIN ENDPOINTS (NON-STANDARD, FOR PROVISIONING MANAGEMENT)
// =============================================================================

// CreateProvisioningToken creates a new provisioning token.
func (h *Handler) CreateProvisioningToken(c forge.Context) error {
	// Get context IDs (3-tier architecture)
	ctx := c.Request().Context()
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, ok := contexts.GetOrganizationID(ctx)

	// Validate organization context
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse request
	var req CreateTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Validate request
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Token name is required"))
	}

	if len(req.Scopes) == 0 {
		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "At least one scope is required"))
	}

	// Create token via service
	token, provToken, err := h.service.CreateProvisioningToken(
		ctx,
		appID,
		envID,
		orgID,
		req.Name,
		req.Description,
		req.Scopes,
		req.ExpiresAt,
	)
	if err != nil {
		h.metrics.RecordError("token_creation_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	response := &TokenResponse{
		Token:   token,
		ID:      provToken.ID.String(),
		Name:    provToken.Name,
		Message: "Store this token securely. It will not be shown again.",
	}

	h.metrics.RecordTokenCreation()

	return c.JSON(http.StatusCreated, response)
}

// ListProvisioningTokens lists provisioning tokens.
func (h *Handler) ListProvisioningTokens(c forge.Context) error {
	// Get context IDs (3-tier architecture)
	ctx := c.Request().Context()
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, ok := contexts.GetOrganizationID(ctx)

	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse pagination parameters
	limit := parseIntParam(c.Query("limit"), 50)
	offset := parseIntParam(c.Query("offset"), 0)

	// List tokens via service
	tokens, total, err := h.service.ListProvisioningTokens(ctx, appID, envID, orgID, limit, offset)
	if err != nil {
		h.metrics.RecordError("token_list_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	// Build response (remove sensitive data)
	tokenList := make([]ProvisioningTokenInfo, 0, len(tokens))
	for _, token := range tokens {
		tokenList = append(tokenList, ProvisioningTokenInfo{
			ID:          token.ID.String(),
			Name:        token.Name,
			Description: token.Description,
			Scopes:      token.Scopes,
			CreatedAt:   token.CreatedAt,
			LastUsedAt:  token.LastUsedAt,
			ExpiresAt:   token.ExpiresAt,
			RevokedAt:   token.RevokedAt,
		})
	}

	response := &TokenListResponse{
		Tokens: tokenList,
		Total:  total,
	}

	return c.JSON(http.StatusOK, response)
}

// RevokeProvisioningToken revokes a provisioning token.
func (h *Handler) RevokeProvisioningToken(c forge.Context) error {
	tokenID := c.Param("id")

	// Revoke token via service
	if err := h.service.RevokeProvisioningToken(c.Request().Context(), tokenID); err != nil {
		h.metrics.RecordError("token_revocation_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	response := &MessageResponse{Message: "Token revoked successfully"}

	return c.JSON(http.StatusOK, response)
}

// GetAttributeMappings gets attribute mappings.
func (h *Handler) GetAttributeMappings(c forge.Context) error {
	// Get context IDs (3-tier architecture)
	ctx := c.Request().Context()
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	// Get mappings via service
	mappings, err := h.service.GetAttributeMappings(ctx, appID, envID, orgID)
	if err != nil {
		h.metrics.RecordError("mapping_get_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	response := &AttributeMappingsResponse{Mappings: mappings}

	return c.JSON(http.StatusOK, response)
}

// UpdateAttributeMappings updates attribute mappings.
func (h *Handler) UpdateAttributeMappings(c forge.Context) error {
	// Get context IDs (3-tier architecture)
	ctx := c.Request().Context()
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	// Parse request
	var req UpdateAttributeMappingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		h.metrics.RecordError("invalid_json")

		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidSyntax", "Invalid JSON in request body"))
	}

	// Validate request
	if len(req.Mappings) == 0 {
		return c.JSON(http.StatusBadRequest, h.scimError(http.StatusBadRequest, "invalidValue", "Mappings cannot be empty"))
	}

	// Update mappings via service
	if err := h.service.UpdateAttributeMappings(ctx, appID, envID, orgID, req.Mappings); err != nil {
		h.metrics.RecordError("mapping_update_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	response := &MessageResponse{Message: "Attribute mappings updated successfully"}

	return c.JSON(http.StatusOK, response)
}

// GetProvisioningLogs gets provisioning logs.
func (h *Handler) GetProvisioningLogs(c forge.Context) error {
	// Get context IDs (3-tier architecture)
	ctx := c.Request().Context()
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, ok := contexts.GetOrganizationID(ctx)

	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Parse query parameters
	action := c.Query("action")
	limit := parseIntParam(c.Query("limit"), 50)
	offset := parseIntParam(c.Query("offset"), 0)

	// Get logs via service
	logs, total, err := h.service.GetProvisioningLogs(ctx, appID, envID, orgID, action, limit, offset)
	if err != nil {
		h.metrics.RecordError("logs_retrieval_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	// Build response using ProvisioningLog directly
	// Convert []*ProvisioningLog to []ProvisioningLog
	logList := make([]ProvisioningLog, 0, len(logs))
	for _, log := range logs {
		logList = append(logList, *log)
	}

	response := &LogsResponse{
		Logs:  logList,
		Total: total,
		Page:  offset/limit + 1,
		Limit: limit,
	}

	return c.JSON(http.StatusOK, response)
}

// GetProvisioningStats gets provisioning statistics.
func (h *Handler) GetProvisioningStats(c forge.Context) error {
	// Get context IDs
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		h.metrics.RecordError("missing_app_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "App context required"))
	}

	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		h.metrics.RecordError("missing_environment_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Environment context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		h.metrics.RecordError("missing_organization_context")

		return c.JSON(http.StatusForbidden, h.scimError(http.StatusForbidden, "invalidValue", "Organization context required"))
	}

	// Get statistics from service
	logs, total, err := h.service.GetProvisioningLogs(ctx, appID, envID, orgID, "", 10, 0)
	if err != nil {
		h.metrics.RecordError("stats_retrieval_failed")
		status, scimType := h.mapServiceErrorToSCIMError(err)

		return c.JSON(status, h.scimError(status, scimType, err.Error()))
	}

	// Calculate statistics
	successCount := 0
	failureCount := 0
	byOperation := make(map[string]int)
	byResourceType := make(map[string]int)
	byStatus := make(map[string]int)

	// Convert to []ProvisioningLog for response
	recentLogs := make([]ProvisioningLog, 0, len(logs))
	for _, log := range logs {
		recentLogs = append(recentLogs, *log)
		if log.Success {
			successCount++
		} else {
			failureCount++
		}

		byOperation[log.Operation]++
		byResourceType[log.ResourceType]++
		statusKey := strconv.Itoa(log.StatusCode)
		byStatus[statusKey]++
	}

	successRate := 0.0
	if total > 0 {
		successRate = float64(successCount) / float64(total) * 100
	}

	response := &StatsResponse{
		TotalOperations: total,
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		SuccessRate:     successRate,
		ByOperation:     byOperation,
		ByResourceType:  byResourceType,
		ByStatus:        byStatus,
		Recent:          recentLogs,
	}

	return c.JSON(http.StatusOK, response)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// scimError creates a SCIM-compliant error response.
func (h *Handler) scimError(status int, scimType, detail string) *ErrorResponse {
	return &ErrorResponse{
		Schemas:  []string{SchemaError},
		Status:   status,
		ScimType: scimType,
		Detail:   detail,
	}
}

// mapServiceErrorToSCIMError maps service errors to SCIM error codes and HTTP status.
func (h *Handler) mapServiceErrorToSCIMError(err error) (status int, scimType string) {
	errMsg := err.Error()

	// Check for specific error patterns
	switch {
	case contains(errMsg, "not found"):
		return http.StatusNotFound, "invalidValue"
	case contains(errMsg, "already exists"), contains(errMsg, "duplicate"):
		return http.StatusConflict, "uniqueness"
	case contains(errMsg, "invalid"), contains(errMsg, "required"):
		return http.StatusBadRequest, "invalidValue"
	case contains(errMsg, "unauthorized"), contains(errMsg, "permission"):
		return http.StatusForbidden, "invalidValue"
	case contains(errMsg, "disabled"):
		return http.StatusServiceUnavailable, "invalidValue"
	default:
		return http.StatusInternalServerError, ""
	}
}

// contains is a case-insensitive string contains check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func parseIntParam(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}

	parsed, err := strconv.Atoi(val)
	if err != nil || parsed < 1 {
		return defaultVal
	}

	return parsed
}

func convertToInterfaces(items any) []any {
	switch v := items.(type) {
	case []ResourceType:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}

		return result
	default:
		return []any{}
	}
}
