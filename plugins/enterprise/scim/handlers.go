package scim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// Handler handles SCIM HTTP requests
type Handler struct {
	service *Service
	config  *Config
	metrics *Metrics
}

// NewHandler creates a new SCIM handler
func NewHandler(service *Service, config *Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
		metrics: GetMetrics(),
	}
}

// Service Provider Configuration endpoints (RFC 7643 Section 5)

// GetServiceProviderConfig returns the service provider configuration
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
		Meta: &Meta{
			ResourceType: "ServiceProviderConfig",
			Location:     "/scim/v2/ServiceProviderConfig",
		},
	}

	return c.JSON(http.StatusOK, config)
}

// Resource Type endpoints (RFC 7643 Section 6)

// GetResourceTypes returns all supported resource types
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
			Meta: &Meta{
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
			Meta: &Meta{
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

// GetResourceType returns a specific resource type
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
			Meta: &Meta{
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
			Meta: &Meta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/Group",
			},
		}
	default:
		return h.errorResponse(c, http.StatusNotFound, "invalidValue", "Resource type not found")
	}

	return c.JSON(http.StatusOK, resourceType)
}

// Schema endpoints (RFC 7643 Section 7)

// GetSchemas returns all supported schemas
func (h *Handler) GetSchemas(c forge.Context) error {
	// Return simplified schema list
	schemas := []interface{}{
		map[string]interface{}{
			"id":   SchemaCore,
			"name": "User",
		},
		map[string]interface{}{
			"id":   SchemaEnterprise,
			"name": "EnterpriseUser",
		},
		map[string]interface{}{
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

// GetSchema returns a specific schema
func (h *Handler) GetSchema(c forge.Context) error {
	id := c.Param("id")

	// For simplicity, return a basic response
	// In production, implement full schema definitions

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        "Schema",
		"description": "SCIM Schema",
	})
}

// User endpoints (RFC 7644 Section 3)

// CreateUser creates a new user
func (h *Handler) CreateUser(c forge.Context) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordRequestDuration("POST /Users", time.Since(start))
	}()

	var scimUser SCIMUser
	if err := json.NewDecoder(c.Request().Body).Decode(&scimUser); err != nil {
		h.metrics.RecordError("invalid_json")
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	// Get organization ID from context (set by middleware)
	orgID := contexts.GetOrganizationID(c.Request().Context())

	createdUser, err := h.service.CreateUser(c.Request().Context(), &scimUser, orgID)
	if err != nil {
		h.metrics.RecordError("user_creation_failed")
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusCreated, createdUser)
}

// ListUsers lists users with filtering and pagination
func (h *Handler) ListUsers(c forge.Context) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordRequestDuration("GET /Users", time.Since(start))
	}()

	orgID := c.Get("organization_id").(xid.ID)

	// Parse query parameters
	filter := c.Request().URL.Query().Get("filter")
	startIndex := parseIntParam(c.Request().URL.Query().Get("startIndex"), 1)
	count := parseIntParam(c.Request().URL.Query().Get("count"), h.config.Search.DefaultResults)

	// Enforce max results
	if count > h.config.Search.MaxResults {
		count = h.config.Search.MaxResults
	}

	listResponse, err := h.service.ListUsers(c.Request().Context(), orgID, filter, startIndex, count)
	if err != nil {
		h.metrics.RecordError("user_list_failed")
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusOK, listResponse)
}

// GetUser retrieves a specific user
func (h *Handler) GetUser(c forge.Context) error {
	id := c.Param("id")
	orgID := c.Get("organization_id").(xid.ID)

	userID, err := xid.FromString(id)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	scimUser, err := h.service.GetUser(c.Request().Context(), userID, orgID)
	if err != nil {
		return h.errorResponse(c, http.StatusNotFound, "", "User not found")
	}

	return c.JSON(http.StatusOK, scimUser)
}

// ReplaceUser replaces a user (PUT)
func (h *Handler) ReplaceUser(c forge.Context) error {
	id := c.Param("id")
	orgID := c.Get("organization_id").(xid.ID)

	userID, err := xid.FromString(id)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	var scimUser SCIMUser
	if err := json.NewDecoder(c.Request().Body).Decode(&scimUser); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	updatedUser, err := h.service.ReplaceUser(c.Request().Context(), userID, orgID, &scimUser)
	if err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusOK, updatedUser)
}

// UpdateUser updates a user (PATCH)
func (h *Handler) UpdateUser(c forge.Context) error {
	id := c.Param("id")
	orgID := c.Get("organization_id").(xid.ID)

	userID, err := xid.FromString(id)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	var patch PatchOp
	if err := json.NewDecoder(c.Request().Body).Decode(&patch); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	updatedUser, err := h.service.UpdateUser(c.Request().Context(), userID, orgID, &patch)
	if err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes a user
func (h *Handler) DeleteUser(c forge.Context) error {
	id := c.Param("id")
	orgID := c.Get("organization_id").(xid.ID)

	userID, err := xid.FromString(id)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	if err := h.service.DeleteUser(c.Request().Context(), userID, orgID); err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// Group endpoints (RFC 7644 Section 3)

// CreateGroup creates a new group
func (h *Handler) CreateGroup(c forge.Context) error {
	var scimGroup SCIMGroup
	if err := json.NewDecoder(c.Request().Body).Decode(&scimGroup); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	orgID := c.Get("organization_id").(xid.ID)

	createdGroup, err := h.service.CreateGroup(c.Request().Context(), &scimGroup, orgID)
	if err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusCreated, createdGroup)
}

// ListGroups lists groups
func (h *Handler) ListGroups(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)

	// Get query parameters
	filter := c.Query("filter")
	startIndex := 1
	if si := c.Query("startIndex"); si != "" {
		fmt.Sscanf(si, "%d", &startIndex)
	}

	count := 100
	if cnt := c.Query("count"); cnt != "" {
		fmt.Sscanf(cnt, "%d", &count)
	}

	result, err := h.service.ListGroups(c.Request().Context(), orgID, filter, startIndex, count)
	if err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetGroup retrieves a specific group
func (h *Handler) GetGroup(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)
	groupIDStr := c.Param("id")

	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	group, err := h.service.GetGroup(c.Request().Context(), groupID, orgID)
	if err != nil {
		return h.errorResponse(c, http.StatusNotFound, "", err.Error())
	}

	return c.JSON(http.StatusOK, group)
}

// ReplaceGroup replaces a group (PUT)
func (h *Handler) ReplaceGroup(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)
	groupIDStr := c.Param("id")
	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	var scimGroup SCIMGroup
	if err := json.NewDecoder(c.Request().Body).Decode(&scimGroup); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidValue", "Invalid request body")
	}

	updatedGroup, err := h.service.ReplaceGroup(c.Request().Context(), groupID, orgID, &scimGroup)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "", err.Error())
	}

	return c.JSON(http.StatusOK, updatedGroup)
}

// UpdateGroup updates a group (PATCH)
func (h *Handler) UpdateGroup(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)
	groupIDStr := c.Param("id")
	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	var patch PatchOp
	if err := json.NewDecoder(c.Request().Body).Decode(&patch); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidValue", "Invalid request body")
	}

	updatedGroup, err := h.service.UpdateGroup(c.Request().Context(), groupID, orgID, &patch)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "", err.Error())
	}

	return c.JSON(http.StatusOK, updatedGroup)
}

// DeleteGroup deletes a group
func (h *Handler) DeleteGroup(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)
	groupIDStr := c.Param("id")
	groupID, err := xid.FromString(groupIDStr)
	if err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	if err := h.service.DeleteGroup(c.Request().Context(), groupID, orgID); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "", err.Error())
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Bulk operations (RFC 7644 Section 3.7)

// BulkOperation handles bulk operations
func (h *Handler) BulkOperation(c forge.Context) error {
	if !h.config.BulkOperations.Enabled {
		return h.errorResponse(c, http.StatusNotImplemented, "", "Bulk operations are disabled")
	}

	var bulkReq BulkRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&bulkReq); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	// Validate operation count
	if len(bulkReq.Operations) > h.config.BulkOperations.MaxOperations {
		return h.errorResponse(c, http.StatusRequestEntityTooLarge, "tooMany",
			fmt.Sprintf("Maximum %d operations allowed", h.config.BulkOperations.MaxOperations))
	}

	// Process each operation
	results := make([]BulkOperationResult, 0, len(bulkReq.Operations))
	errorCount := 0

	for _, op := range bulkReq.Operations {
		result := h.processBulkOperation(c, &op)
		results = append(results, result)

		if result.Status >= 400 {
			errorCount++
			if bulkReq.FailOnErrors > 0 && errorCount >= bulkReq.FailOnErrors {
				break
			}
		}
	}

	bulkResp := &BulkResponse{
		Schemas:    []string{SchemaBulkResponse},
		Operations: results,
	}

	return c.JSON(http.StatusOK, bulkResp)
}

// Search endpoint (RFC 7644 Section 3.4.3)

// Search handles the /.search endpoint
func (h *Handler) Search(c forge.Context) error {
	var searchReq struct {
		Schemas    []string `json:"schemas"`
		Filter     string   `json:"filter"`
		StartIndex int      `json:"startIndex"`
		Count      int      `json:"count"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&searchReq); err != nil {
		return h.errorResponse(c, http.StatusBadRequest, "invalidSyntax", "Invalid JSON")
	}

	// Default values
	if searchReq.StartIndex == 0 {
		searchReq.StartIndex = 1
	}
	if searchReq.Count == 0 {
		searchReq.Count = h.config.Search.DefaultResults
	}

	orgID := c.Get("organization_id").(xid.ID)

	listResponse, err := h.service.ListUsers(c.Request().Context(), orgID, searchReq.Filter, searchReq.StartIndex, searchReq.Count)
	if err != nil {
		return h.errorResponse(c, http.StatusInternalServerError, "", err.Error())
	}

	return c.JSON(http.StatusOK, listResponse)
}

// Admin endpoints (non-standard, for provisioning management)

// CreateProvisioningToken creates a new provisioning token
func (h *Handler) CreateProvisioningToken(c forge.Context) error {
	var req struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Scopes      []string   `json:"scopes"`
		ExpiresAt   *time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get 3-tier architecture context
	ctx := c.Request().Context()
	appIDVal := contexts.GetAppID(ctx)
	envIDVal := contexts.GetEnvironmentID(ctx)
	orgIDVal := contexts.GetOrganizationID(ctx)

	// If orgID is not set, this is an error
	if orgIDVal.IsNil() {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Organization context required",
		})
	}

	token, provToken, err := h.service.CreateProvisioningToken(
		c.Request().Context(),
		appIDVal,
		envIDVal,
		orgIDVal,
		req.Name,
		req.Description,
		req.Scopes,
		req.ExpiresAt,
	)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"token":   token,
		"id":      provToken.ID.String(),
		"name":    provToken.Name,
		"message": "Store this token securely. It will not be shown again.",
	})
}

// ListProvisioningTokens lists provisioning tokens
func (h *Handler) ListProvisioningTokens(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)

	// Get pagination parameters
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Get 3-tier architecture context
	appID := c.Get("app_id").(xid.ID)
	envID := c.Get("environment_id").(xid.ID)

	tokens, total, err := h.service.ListProvisioningTokens(c.Request().Context(), appID, envID, orgID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Remove sensitive data from response
	safeTokens := make([]map[string]interface{}, 0, len(tokens))
	for _, token := range tokens {
		safeTokens = append(safeTokens, map[string]interface{}{
			"id":           token.ID.String(),
			"name":         token.Name,
			"description":  token.Description,
			"scopes":       token.Scopes,
			"created_at":   token.CreatedAt,
			"updated_at":   token.UpdatedAt,
			"last_used_at": token.LastUsedAt,
			"expires_at":   token.ExpiresAt,
			"revoked_at":   token.RevokedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tokens": safeTokens,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// RevokeProvisioningToken revokes a provisioning token
func (h *Handler) RevokeProvisioningToken(c forge.Context) error {
	tokenID := c.Param("id")

	if err := h.service.RevokeProvisioningToken(c.Request().Context(), tokenID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Token revoked successfully",
	})
}

// GetAttributeMappings gets attribute mappings
func (h *Handler) GetAttributeMappings(c forge.Context) error {
	// Get 3-tier architecture context
	ctx := c.Request().Context()
	appID := contexts.GetAppID(ctx)
	envID := contexts.GetEnvironmentID(ctx)
	orgID := contexts.GetOrganizationID(ctx)

	mappings, err := h.service.GetAttributeMappings(c.Request().Context(), appID, envID, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"mappings": mappings,
	})
}

// UpdateAttributeMappings updates attribute mappings
func (h *Handler) UpdateAttributeMappings(c forge.Context) error {
	// Get 3-tier architecture context
	ctx := c.Request().Context()
	appID := contexts.GetAppID(ctx)
	envID := contexts.GetEnvironmentID(ctx)
	orgID := contexts.GetOrganizationID(ctx)

	var req struct {
		Mappings map[string]string `json:"mappings"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateAttributeMappings(c.Request().Context(), appID, envID, orgID, req.Mappings); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Attribute mappings updated successfully",
	})
}

// GetProvisioningLogs gets provisioning logs
func (h *Handler) GetProvisioningLogs(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)

	// Get pagination and filter parameters
	action := c.Query("action")

	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Get 3-tier architecture context
	appID := c.Get("app_id").(xid.ID)
	envID := c.Get("environment_id").(xid.ID)

	logs, total, err := h.service.GetProvisioningLogs(c.Request().Context(), appID, envID, orgID, action, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetProvisioningStats gets provisioning statistics
func (h *Handler) GetProvisioningStats(c forge.Context) error {
	// Return real-time metrics from the metrics system
	stats := h.metrics.GetStats()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"scim_metrics": stats,
	})
}

// Helper methods

func (h *Handler) errorResponse(c forge.Context, status int, scimType, detail string) error {
	errResp := &ErrorResponse{
		Schemas:  []string{SchemaError},
		Status:   status,
		ScimType: scimType,
		Detail:   detail,
	}
	return c.JSON(status, errResp)
}

func (h *Handler) processBulkOperation(c forge.Context, op *BulkOperation) BulkOperationResult {
	result := BulkOperationResult{
		Method: op.Method,
		BulkID: op.BulkID,
	}

	// TODO: Implement bulk operation processing
	result.Status = http.StatusNotImplemented
	result.Response = map[string]string{"error": "Not implemented"}

	return result
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

func convertToInterfaces(items interface{}) []interface{} {
	switch v := items.(type) {
	case []ResourceType:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	default:
		return []interface{}{}
	}
}
