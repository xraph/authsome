package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/forge"
)

// OrganizationHandler handles organization-related HTTP requests
type OrganizationHandler struct {
	orgService *organization.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *organization.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganization handles organization creation requests
func (h *OrganizationHandler) CreateOrganization(c forge.Context) error {
	var req organization.CreateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get creator user ID from context/session
	creatorUserID := "system" // placeholder

	org, err := h.orgService.CreateOrganization(c.Request().Context(), &req, creatorUserID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, org)
}

// GetOrganization handles get organization requests
func (h *OrganizationHandler) GetOrganization(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	org, err := h.orgService.GetOrganization(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "organization not found"})
	}

	return c.JSON(200, org)
}

// UpdateOrganization handles organization update requests
func (h *OrganizationHandler) UpdateOrganization(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	var req organization.UpdateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	org, err := h.orgService.UpdateOrganization(c.Request().Context(), id, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, org)
}

// DeleteOrganization handles organization deletion requests
func (h *OrganizationHandler) DeleteOrganization(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	err := h.orgService.DeleteOrganization(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListOrganizations handles list organizations requests
func (h *OrganizationHandler) ListOrganizations(c forge.Context) error {
	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 10 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	orgs, err := h.orgService.ListOrganizations(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"organizations": orgs,
		"limit":         limit,
		"offset":        offset,
	})
}
