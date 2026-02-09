package handlers

import (
	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/query"
	"github.com/xraph/authsome/plugins/cms/service"
)

// ContentEntryHandler handles content entry HTTP requests.
type ContentEntryHandler struct {
	entryService       *service.ContentEntryService
	contentTypeService *service.ContentTypeService
}

// NewContentEntryHandler creates a new content entry handler.
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
// GET /cms/:type.
func (h *ContentEntryHandler) ListEntries(c forge.Context) error {
	var req ListEntriesRequest

	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]any{"error": "invalid request", "details": err.Error()})
	}

	typeSlug := req.TypeSlug

	ctx := getContextWithHeaders(c)

	// Get the content type
	contentType, err := h.contentTypeService.GetByName(ctx, typeSlug)
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
		// Convert camelCase sort fields to snake_case for repository
		sortField := q.Sort[0].Field
		switch sortField {
		case "createdAt":
			sortField = "created_at"
		case "updatedAt":
			sortField = "updated_at"
		case "publishedAt":
			sortField = "published_at"
		case "scheduledAt":
			sortField = "scheduled_at"
		}

		listQuery.SortBy = sortField
		if q.Sort[0].Descending {
			listQuery.SortOrder = "desc"
		} else {
			listQuery.SortOrder = "asc"
		}
	}

	// Convert filters - handle system fields (with _meta prefix) vs JSONB content fields
	if q.Filters != nil {
		listQuery.Filters = make(map[string]any)
		for _, cond := range q.Filters.Conditions {
			// Handle status filter specially - supports both "status" and "_meta.status"
			if cond.Field == "status" || cond.Field == query.MetaPrefix+"status" {
				if val, ok := cond.Value.(string); ok {
					listQuery.Status = val
				}

				continue
			}
			// Skip other system fields from JSONB filters
			if query.IsSystemField(cond.Field) {
				continue
			}
			// Store operator and value together for JSONB fields
			listQuery.Filters[cond.Field] = map[string]any{
				"operator": string(cond.Operator),
				"value":    cond.Value,
			}
		}
	}

	for _, pop := range q.Populate {
		listQuery.Populate = append(listQuery.Populate, pop.Path)
	}

	result, err := h.entryService.List(ctx, contentTypeID, listQuery)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// CreateEntry creates a new content entry
// POST /cms/:type.
func (h *ContentEntryHandler) CreateEntry(c forge.Context) error {
	var pathReq CreateEntryRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := pathReq.TypeSlug

	ctx := getContextWithHeaders(c)

	// Get the content type
	contentType, err := h.contentTypeService.GetByName(ctx, typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	// Parse request body
	var bodyReq core.CreateEntryRequest
	if err := c.BindJSON(&bodyReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.entryService.Create(ctx, contentTypeID, &bodyReq)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, result)
}

// GetEntry retrieves a content entry by ID
// GET /cms/:type/:id.
func (h *ContentEntryHandler) GetEntry(c forge.Context) error {
	var req GetEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.GetByID(ctx, id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// UpdateEntry updates a content entry
// PUT /cms/:type/:id.
func (h *ContentEntryHandler) UpdateEntry(c forge.Context) error {
	var req UpdateEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse request body
	var bodyReq core.UpdateEntryRequest
	if err := c.BindJSON(&bodyReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	result, err := h.entryService.Update(ctx, id, &bodyReq)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// DeleteEntry deletes a content entry
// DELETE /cms/:type/:id.
func (h *ContentEntryHandler) DeleteEntry(c forge.Context) error {
	var req DeleteEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	if err := h.entryService.Delete(ctx, id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// =============================================================================
// Status Operations
// =============================================================================

// PublishEntry publishes a content entry
// POST /cms/:type/:id/publish.
func (h *ContentEntryHandler) PublishEntry(c forge.Context) error {
	var req PublishEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse optional request body
	var bodyReq core.PublishEntryRequest

	_ = c.BindJSON(&bodyReq) // Ignore error, body is optional

	result, err := h.entryService.Publish(ctx, id, &bodyReq)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// UnpublishEntry unpublishes a content entry
// POST /cms/:type/:id/unpublish.
func (h *ContentEntryHandler) UnpublishEntry(c forge.Context) error {
	var req UnpublishEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.Unpublish(ctx, id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// ArchiveEntry archives a content entry
// POST /cms/:type/:id/archive.
func (h *ContentEntryHandler) ArchiveEntry(c forge.Context) error {
	var req ArchiveEntryRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	entryID := req.EntryID
	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	ctx := getContextWithHeaders(c)

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	result, err := h.entryService.Archive(ctx, id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// =============================================================================
// Advanced Query Endpoint
// =============================================================================

// QueryEntries performs an advanced query on entries
// POST /cms/:type/query.
func (h *ContentEntryHandler) QueryEntries(c forge.Context) error {
	var pathReq QueryEntriesRequest
	if err := c.BindRequest(&pathReq); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := pathReq.TypeSlug

	ctx := getContextWithHeaders(c)

	// Get the content type
	contentType, err := h.contentTypeService.GetByName(ctx, typeSlug)
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
		// Convert camelCase sort fields to snake_case for repository
		sortField := q.Sort[0].Field
		switch sortField {
		case "createdAt":
			sortField = "created_at"
		case "updatedAt":
			sortField = "updated_at"
		case "publishedAt":
			sortField = "published_at"
		case "scheduledAt":
			sortField = "scheduled_at"
		}

		listQuery.SortBy = sortField
		if q.Sort[0].Descending {
			listQuery.SortOrder = "desc"
		} else {
			listQuery.SortOrder = "asc"
		}
	}

	// Convert filters - handle system fields (with _meta prefix) vs JSONB content fields
	if q.Filters != nil {
		listQuery.Filters = make(map[string]any)
		for _, cond := range q.Filters.Conditions {
			// Handle status filter specially - supports both "status" and "_meta.status"
			if cond.Field == "status" || cond.Field == query.MetaPrefix+"status" {
				if val, ok := cond.Value.(string); ok {
					listQuery.Status = val
				}

				continue
			}
			// Skip other system fields from JSONB filters
			if query.IsSystemField(cond.Field) {
				continue
			}
			// Store operator and value together
			listQuery.Filters[cond.Field] = map[string]any{
				"operator": string(cond.Operator),
				"value":    cond.Value,
			}
		}
		// Also check nested groups for status filter
		for _, group := range q.Filters.Groups {
			for _, cond := range group.Conditions {
				// Handle status filter - supports both "status" and "_meta.status"
				if cond.Field == "status" || cond.Field == query.MetaPrefix+"status" {
					if val, ok := cond.Value.(string); ok {
						listQuery.Status = val
					}
				} else if !query.IsSystemField(cond.Field) {
					// Only add non-system fields to JSONB filters
					if listQuery.Filters == nil {
						listQuery.Filters = make(map[string]any)
					}

					listQuery.Filters[cond.Field] = map[string]any{
						"operator": string(cond.Operator),
						"value":    cond.Value,
					}
				}
			}
		}
	}

	for _, pop := range q.Populate {
		listQuery.Populate = append(listQuery.Populate, pop.Path)
	}

	result, err := h.entryService.List(ctx, contentTypeID, listQuery)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkRequest represents a bulk operation request.
type BulkRequest struct {
	IDs []string `json:"ids"`
}

// BulkPublish publishes multiple entries
// POST /cms/:type/bulk/publish.
func (h *ContentEntryHandler) BulkPublish(c forge.Context) error {
	var req BulkPublishRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	ctx := getContextWithHeaders(c)

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}

		ids[i] = id
	}

	if err := h.entryService.BulkPublish(ctx, ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"message": "entries published",
		"count":   len(ids),
	})
}

// BulkUnpublish unpublishes multiple entries
// POST /cms/:type/bulk/unpublish.
func (h *ContentEntryHandler) BulkUnpublish(c forge.Context) error {
	var req BulkUnpublishRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	ctx := getContextWithHeaders(c)

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}

		ids[i] = id
	}

	if err := h.entryService.BulkUnpublish(ctx, ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"message": "entries unpublished",
		"count":   len(ids),
	})
}

// BulkDelete deletes multiple entries
// POST /cms/:type/bulk/delete.
func (h *ContentEntryHandler) BulkDelete(c forge.Context) error {
	var req BulkDeleteRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	ctx := getContextWithHeaders(c)

	ids := make([]xid.ID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid entry ID: " + idStr})
		}

		ids[i] = id
	}

	if err := h.entryService.BulkDelete(ctx, ids); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"message": "entries deleted",
		"count":   len(ids),
	})
}

// =============================================================================
// Stats Endpoint
// =============================================================================

// GetEntryStats returns statistics for entries
// GET /cms/:type/stats.
func (h *ContentEntryHandler) GetEntryStats(c forge.Context) error {
	var req GetEntryStatsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	typeSlug := req.TypeSlug

	ctx := getContextWithHeaders(c)

	// Get the content type
	contentType, err := h.contentTypeService.GetByName(ctx, typeSlug)
	if err != nil {
		return handleError(c, err)
	}

	// Parse ID
	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid content type ID"})
	}

	stats, err := h.entryService.GetStats(ctx, contentTypeID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, stats)
}
