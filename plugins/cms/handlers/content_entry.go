package handlers

import (
	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/query"
	"github.com/xraph/authsome/plugins/cms/service"
)

// ContentEntryHandler handles content entry HTTP requests
type ContentEntryHandler struct {
	entryService       *service.ContentEntryService
	contentTypeService *service.ContentTypeService
}

// NewContentEntryHandler creates a new content entry handler
func NewContentEntryHandler(
	entrySvc *service.ContentEntryService,
	ctSvc *service.ContentTypeService,
) *ContentEntryHandler {
	return &ContentEntryHandler{
		entryService:       entrySvc,
		contentTypeService: ctSvc,
	}
}

// =============================================================================
// Content Entry Endpoints
// =============================================================================

// ListEntries lists entries for a content type
// GET /cms/:type
func (h *ContentEntryHandler) ListEntries(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	// Get the content type
	contentType, err := h.contentTypeService.GetBySlug(c.Request().Context(), typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse URL query parameters
	urlParser := query.NewURLParser()
	q, err := urlParser.Parse(c.Request().URL.Query())
	if err != nil {
		return handleError(c, err)
	}

	// Convert to ListEntriesQuery
	listQuery := &core.ListEntriesQuery{
		Status:    q.Status,
		Search:    q.Search,
		SortBy:    "",
		SortOrder: "",
		Page:      q.Page,
		PageSize:  q.PageSize,
		Select:    q.Select,
	}

	if len(q.Sort) > 0 {
		listQuery.SortBy = q.Sort[0].Field
		if q.Sort[0].Descending {
			listQuery.SortOrder = "desc"
		} else {
			listQuery.SortOrder = "asc"
		}
	}

	// Convert filters
	if q.Filters != nil {
		listQuery.Filters = make(map[string]any)
		for _, cond := range q.Filters.Conditions {
			listQuery.Filters[cond.Field] = cond.Value
		}
	}

	for _, pop := range q.Populate {
		listQuery.Populate = append(listQuery.Populate, pop.Path)
	}

	result, err := h.entryService.List(c.Request().Context(), contentTypeID, listQuery)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// CreateEntry creates a new content entry
// POST /cms/:type
func (h *ContentEntryHandler) CreateEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	// Get the content type
	contentType, err := h.contentTypeService.GetBySlug(c.Request().Context(), typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse request
	var req core.CreateEntryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.entryService.Create(c.Request().Context(), contentTypeID, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, result)
}

// GetEntry retrieves a content entry by ID
// GET /cms/:type/:id
func (h *ContentEntryHandler) GetEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.GetByID(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// UpdateEntry updates a content entry
// PUT /cms/:type/:id
func (h *ContentEntryHandler) UpdateEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse request
	var req core.UpdateEntryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.entryService.Update(c.Request().Context(), id, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// DeleteEntry deletes a content entry
// DELETE /cms/:type/:id
func (h *ContentEntryHandler) DeleteEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	if err := h.entryService.Delete(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// =============================================================================
// Status Operations
// =============================================================================

// PublishEntry publishes a content entry
// POST /cms/:type/:id/publish
func (h *ContentEntryHandler) PublishEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse optional request body
	var req core.PublishEntryRequest
	_ = c.BindJSON(&req) // Ignore error, body is optional

	result, err := h.entryService.Publish(c.Request().Context(), id, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// UnpublishEntry unpublishes a content entry
// POST /cms/:type/:id/unpublish
func (h *ContentEntryHandler) UnpublishEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.Unpublish(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// ArchiveEntry archives a content entry
// POST /cms/:type/:id/archive
func (h *ContentEntryHandler) ArchiveEntry(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.Archive(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// =============================================================================
// Advanced Query Endpoint
// =============================================================================

// QueryEntries performs an advanced query on entries
// POST /cms/:type/query
func (h *ContentEntryHandler) QueryEntries(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	// Get the content type
	contentType, err := h.contentTypeService.GetBySlug(c.Request().Context(), typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse JSON body
	jsonParser := query.NewJSONParser()
	body, err := readBody(c)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "failed to read request body"})
	}

	q, err := jsonParser.Parse(body)
	if err != nil {
		return handleError(c, err)
	}

	// Convert to ListEntriesQuery for now
	// TODO: Use query executor directly for full query support
	listQuery := &core.ListEntriesQuery{
		Status:   q.Status,
		Search:   q.Search,
		Page:     q.Page,
		PageSize: q.PageSize,
		Select:   q.Select,
	}

	if len(q.Sort) > 0 {
		listQuery.SortBy = q.Sort[0].Field
		if q.Sort[0].Descending {
			listQuery.SortOrder = "desc"
		} else {
			listQuery.SortOrder = "asc"
		}
	}

	// Convert filters
	if q.Filters != nil {
		listQuery.Filters = make(map[string]any)
		for _, cond := range q.Filters.Conditions {
			listQuery.Filters[cond.Field] = cond.Value
		}
	}

	for _, pop := range q.Populate {
		listQuery.Populate = append(listQuery.Populate, pop.Path)
	}

	result, err := h.entryService.List(c.Request().Context(), contentTypeID, listQuery)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkRequest represents a bulk operation request
type BulkRequest struct {
	IDs []string `json:"ids"`
}

// BulkPublish publishes multiple entries
// POST /cms/:type/bulk/publish
func (h *ContentEntryHandler) BulkPublish(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	var req BulkRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}
		ids[i] = id
	}

	if err := h.entryService.BulkPublish(c.Request().Context(), ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"message": "entries published",
		"count":   len(ids),
	})
}

// BulkUnpublish unpublishes multiple entries
// POST /cms/:type/bulk/unpublish
func (h *ContentEntryHandler) BulkUnpublish(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	var req BulkRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}
		ids[i] = id
	}

	if err := h.entryService.BulkUnpublish(c.Request().Context(), ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"message": "entries unpublished",
		"count":   len(ids),
	})
}

// BulkDelete deletes multiple entries
// POST /cms/:type/bulk/delete
func (h *ContentEntryHandler) BulkDelete(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	var req BulkRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}
		ids[i] = id
	}

	if err := h.entryService.BulkDelete(c.Request().Context(), ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"message": "entries deleted",
		"count":   len(ids),
	})
}

// =============================================================================
// Stats Endpoint
// =============================================================================

// GetEntryStats returns statistics for entries
// GET /cms/:type/stats
func (h *ContentEntryHandler) GetEntryStats(c forge.Context) error {
	typeSlug := c.Param("type")
	if typeSlug == "" {
		return c.JSON(400, map[string]string{"error": "type is required"})
	}

	// Get the content type
	contentType, err := h.contentTypeService.GetBySlug(c.Request().Context(), typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	stats, err := h.entryService.GetStats(c.Request().Context(), contentTypeID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, stats)
}

