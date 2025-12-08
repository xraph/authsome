package handlers

import (
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/service"
)

// RevisionHandler handles content revision HTTP requests
type RevisionHandler struct {
	revisionService    *service.RevisionService
	entryService       *service.ContentEntryService
	contentTypeService *service.ContentTypeService
}

// NewRevisionHandler creates a new revision handler
func NewRevisionHandler(
	revSvc *service.RevisionService,
	entrySvc *service.ContentEntryService,
	ctSvc *service.ContentTypeService,
) *RevisionHandler {
	return &RevisionHandler{
		revisionService:    revSvc,
		entryService:       entrySvc,
		contentTypeService: ctSvc,
	}
}

// ListRevisions lists revisions for an entry
// GET /cms/:type/:id/revisions
func (h *RevisionHandler) ListRevisions(c forge.Context) error {
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

	// Verify entry exists
	_, err = h.entryService.GetByID(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	// Parse query params manually
	query := core.ListRevisionsQuery{
		Page:     parseIntDefault(c.Query("page"), 1),
		PageSize: parseIntDefault(c.Query("pageSize"), 20),
	}

	result, err := h.revisionService.List(c.Request().Context(), id, &query)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// GetRevision retrieves a specific revision
// GET /cms/:type/:id/revisions/:version
func (h *RevisionHandler) GetRevision(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	versionStr := c.Param("version")
	if typeSlug == "" || entryID == "" || versionStr == "" {
		return c.JSON(400, map[string]string{"error": "type, id, and version are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse version
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid version number"})
	}

	result, err := h.revisionService.GetByVersion(c.Request().Context(), id, version)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// RestoreRevision restores an entry to a specific revision
// POST /cms/:type/:id/revisions/:version/restore
func (h *RevisionHandler) RestoreRevision(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	versionStr := c.Param("version")
	if typeSlug == "" || entryID == "" || versionStr == "" {
		return c.JSON(400, map[string]string{"error": "type, id, and version are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse version
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid version number"})
	}

	result, err := h.entryService.Restore(c.Request().Context(), id, version)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}

// CompareRevisions compares two revisions
// GET /cms/:type/:id/revisions/compare?from=:v1&to=:v2
func (h *RevisionHandler) CompareRevisions(c forge.Context) error {
	typeSlug := c.Param("type")
	entryID := c.Param("id")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if typeSlug == "" || entryID == "" {
		return c.JSON(400, map[string]string{"error": "type and id are required"})
	}

	if fromStr == "" || toStr == "" {
		return c.JSON(400, map[string]string{"error": "from and to version query params are required"})
	}

	// Parse entry ID
	id, err := xid.FromString(entryID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid entry ID"})
	}

	// Parse versions
	fromVersion, err := strconv.Atoi(fromStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid 'from' version number"})
	}

	toVersion, err := strconv.Atoi(toStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid 'to' version number"})
	}

	result, err := h.revisionService.Compare(c.Request().Context(), id, fromVersion, toVersion)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, result)
}
