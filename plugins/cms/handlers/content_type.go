// Package handlers provides HTTP handlers for the CMS plugin.
package handlers

import (
	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/service"
)

// ContentTypeHandler handles content type HTTP requests.
type ContentTypeHandler struct {
	service      *service.ContentTypeService
	fieldService *service.ContentFieldService
}

// NewContentTypeHandler creates a new content type handler.
func NewContentTypeHandler(
	svc *service.ContentTypeService,
	fieldSvc *service.ContentFieldService,
) *ContentTypeHandler {
	return &ContentTypeHandler{
		service:      svc,
		fieldService: fieldSvc,
	}
}

// =============================================================================
// Content Type Endpoints
// =============================================================================

// ListContentTypes lists all content types
// GET /cms/types.
func (h *ContentTypeHandler) ListContentTypes(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req ListContentTypesRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Apply defaults
	if req.Page < 1 {
		req.Page = 1
	}

	if req.PageSize < 1 {
		req.PageSize = 20
	}

	query := core.ListContentTypesQuery{
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	result, err := h.service.List(ctx, &query)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// CreateContentType creates a new content type
// POST /cms/types.
func (h *ContentTypeHandler) CreateContentType(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req core.CreateContentTypeRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.service.Create(ctx, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, result)
}

// GetContentType retrieves a content type by slug
// GET /cms/types/:slug.
func (h *ContentTypeHandler) GetContentType(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req GetContentTypeRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	result, err := h.service.GetByName(ctx, req.Slug)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// UpdateContentType updates a content type
// PUT /cms/types/:slug.
func (h *ContentTypeHandler) UpdateContentType(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var pathReq UpdateContentTypeRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, pathReq.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse update request
	var req core.UpdateContentTypeRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	result, err := h.service.Update(ctx, id, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// DeleteContentType deletes a content type
// DELETE /cms/types/:slug.
func (h *ContentTypeHandler) DeleteContentType(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req DeleteContentTypeRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, req.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	if err := h.service.Delete(ctx, id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// =============================================================================
// Content Field Endpoints
// =============================================================================

// ListFields lists all fields for a content type
// GET /cms/types/:slug/fields.
func (h *ContentTypeHandler) ListFields(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req GetContentTypeRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, req.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	fields, err := h.fieldService.List(ctx, id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"fields": fields,
	})
}

// AddField adds a new field to a content type
// POST /cms/types/:slug/fields.
func (h *ContentTypeHandler) AddField(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var pathReq AddFieldRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	slug := pathReq.Slug

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse request
	var req core.CreateFieldRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.fieldService.Create(ctx, id, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, result)
}

// GetField retrieves a field by slug
// GET /cms/types/:slug/fields/:fieldSlug.
func (h *ContentTypeHandler) GetField(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req UpdateFieldRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, req.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	field, err := h.fieldService.GetByName(ctx, id, req.FieldID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, field)
}

// UpdateField updates a field
// PUT /cms/types/:slug/fields/:fieldSlug.
func (h *ContentTypeHandler) UpdateField(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var pathReq UpdateFieldRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, pathReq.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Get the field
	field, err := h.fieldService.GetByName(ctx, contentTypeID, pathReq.FieldID)
	if err != nil {
		return handleError(c, err)
	}

	// Parse field ID
	fieldID, err := xid.FromString(field.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid field ID"})
	}

	// Parse request
	var req core.UpdateFieldRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.fieldService.Update(ctx, fieldID, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// DeleteField deletes a field
// DELETE /cms/types/:slug/fields/:fieldSlug.
func (h *ContentTypeHandler) DeleteField(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var req DeleteFieldRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, req.Slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Get the field
	field, err := h.fieldService.GetByName(ctx, contentTypeID, req.FieldID)
	if err != nil {
		return handleError(c, err)
	}

	// Parse field ID
	fieldID, err := xid.FromString(field.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid field ID"})
	}

	if err := h.fieldService.Delete(ctx, fieldID); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// ReorderFields reorders fields in a content type
// POST /cms/types/:slug/fields/reorder.
func (h *ContentTypeHandler) ReorderFields(c forge.Context) error {
	ctx := getContextWithHeaders(c)

	var pathReq ReorderFieldsRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	slug := pathReq.Slug

	// Get the content type first
	contentType, err := h.service.GetByName(ctx, slug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	id, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse request
	var req core.ReorderFieldsRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.fieldService.Reorder(ctx, id, &req); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]string{"message": "fields reordered"})
}

// GetFieldTypes returns all available field types
// GET /cms/field-types.
func (h *ContentTypeHandler) GetFieldTypes(c forge.Context) error {
	return c.JSON(200, map[string]any{
		"fieldTypes": core.GetAllFieldTypes(),
	})
}
