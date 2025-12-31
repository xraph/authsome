package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// AuditHandler exposes endpoints to query audit events
type AuditHandler struct {
	service *audit.Service
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service *audit.Service) *AuditHandler {
	return &AuditHandler{service: service}
}

// ListEvents returns audit events with pagination and optional filters
// Query params: q, environmentId, userId, userIds, action, actions, actionPattern, resource, resources, resourcePattern,
//               ipAddress, ipAddresses, ipRange, since, until, sortBy, sortOrder
func (h *AuditHandler) ListEvents(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	q := c.Request().URL.Query()

	// Parse pagination parameters
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit for performance
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &audit.ListEventsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
	}

	// ========== Full-Text Search ==========
	if searchQuery := q.Get("q"); searchQuery != "" {
		filter.SearchQuery = &searchQuery
	}

	if searchFields := q.Get("search_fields"); searchFields != "" {
		filter.SearchFields = strings.Split(searchFields, ",")
	}

	// ========== Exact Match Filters ==========
	if envIDStr := q.Get("environmentId"); envIDStr != "" {
		if envID, err := xid.FromString(envIDStr); err == nil {
			filter.EnvironmentID = &envID
		}
	}

	if uidStr := q.Get("userId"); uidStr != "" {
		if uid, err := xid.FromString(uidStr); err == nil {
			filter.UserID = &uid
		}
	}

	if action := q.Get("action"); action != "" {
		filter.Action = &action
	}

	if resource := q.Get("resource"); resource != "" {
		filter.Resource = &resource
	}

	if ipAddress := q.Get("ipAddress"); ipAddress != "" {
		filter.IPAddress = &ipAddress
	}

	// ========== Multiple Value Filters ==========
	if userIDsStr := q.Get("userIds"); userIDsStr != "" {
		userIDStrs := strings.Split(userIDsStr, ",")
		filter.UserIDs = make([]xid.ID, 0, len(userIDStrs))
		for _, idStr := range userIDStrs {
			if id, err := xid.FromString(strings.TrimSpace(idStr)); err == nil {
				filter.UserIDs = append(filter.UserIDs, id)
			}
		}
	}

	if actionsStr := q.Get("actions"); actionsStr != "" {
		filter.Actions = strings.Split(actionsStr, ",")
		// Trim whitespace
		for i := range filter.Actions {
			filter.Actions[i] = strings.TrimSpace(filter.Actions[i])
		}
	}

	if resourcesStr := q.Get("resources"); resourcesStr != "" {
		filter.Resources = strings.Split(resourcesStr, ",")
		// Trim whitespace
		for i := range filter.Resources {
			filter.Resources[i] = strings.TrimSpace(filter.Resources[i])
		}
	}

	if ipAddressesStr := q.Get("ipAddresses"); ipAddressesStr != "" {
		filter.IPAddresses = strings.Split(ipAddressesStr, ",")
		// Trim whitespace
		for i := range filter.IPAddresses {
			filter.IPAddresses[i] = strings.TrimSpace(filter.IPAddresses[i])
		}
	}

	// ========== Pattern Matching ==========
	if actionPattern := q.Get("actionPattern"); actionPattern != "" {
		filter.ActionPattern = &actionPattern
	}

	if resourcePattern := q.Get("resourcePattern"); resourcePattern != "" {
		filter.ResourcePattern = &resourcePattern
	}

	// ========== IP Range Filtering ==========
	if ipRange := q.Get("ipRange"); ipRange != "" {
		filter.IPRange = &ipRange
	}

	// ========== Metadata Filtering ==========
	// Support simple metadata filters via query params: metadata.key=value
	metadataFilters := make([]audit.MetadataFilter, 0)
	for key, values := range q {
		if strings.HasPrefix(key, "metadata.") {
			metadataKey := strings.TrimPrefix(key, "metadata.")
			if len(values) > 0 && values[0] != "" {
				metadataFilters = append(metadataFilters, audit.MetadataFilter{
					Key:      metadataKey,
					Value:    values[0],
					Operator: "contains",
				})
			}
		}
	}
	if len(metadataFilters) > 0 {
		filter.MetadataFilters = metadataFilters
	}

	// ========== Time Range Filters ==========
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			filter.Since = &t
		}
	}

	if untilStr := q.Get("until"); untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			filter.Until = &t
		}
	}

	// ========== Sorting Parameters ==========
	if sortBy := q.Get("sortBy"); sortBy != "" {
		filter.SortBy = &sortBy
	}

	if sortOrder := q.Get("sortOrder"); sortOrder != "" {
		filter.SortOrder = &sortOrder
	}

	// Call service
	resp, err := h.service.List(c.Request().Context(), filter)
	if err != nil {
		// Handle structured errors
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to list audit events"))
	}

	// Set pagination headers for backward compatibility
	if resp.Pagination != nil {
		c.SetHeader("X-Total-Count", strconv.FormatInt(resp.Pagination.Total, 10))
		c.SetHeader("X-Page", strconv.Itoa(resp.Pagination.CurrentPage))
		c.SetHeader("X-Total-Pages", strconv.Itoa(resp.Pagination.TotalPages))
	}

	// Return paginated response
	return c.JSON(http.StatusOK, resp)
}

// SearchEvents performs full-text search on audit events
// Query params: q (required), fields, fuzzy, environmentId, userId, action, since, until, limit, offset
func (h *AuditHandler) SearchEvents(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	q := c.Request().URL.Query()

	// Parse search query (required)
	searchQuery := q.Get("q")
	if searchQuery == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("search query 'q' is required"))
	}

	// Build search query
	query := &audit.SearchQuery{
		Query: searchQuery,
	}

	// Parse fields to search
	if fieldsStr := q.Get("fields"); fieldsStr != "" {
		query.Fields = strings.Split(fieldsStr, ",")
		for i := range query.Fields {
			query.Fields[i] = strings.TrimSpace(query.Fields[i])
		}
	}

	// Parse fuzzy match
	if fuzzyStr := q.Get("fuzzy"); fuzzyStr == "true" || fuzzyStr == "1" {
		query.FuzzyMatch = true
	}

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}
	query.Limit = limit

	offset, _ := strconv.Atoi(q.Get("offset"))
	query.Offset = offset

	// Parse filters
	if envIDStr := q.Get("environmentId"); envIDStr != "" {
		if envID, err := xid.FromString(envIDStr); err == nil {
			query.EnvironmentID = &envID
		}
	}

	if uidStr := q.Get("userId"); uidStr != "" {
		if uid, err := xid.FromString(uidStr); err == nil {
			query.UserID = &uid
		}
	}

	if action := q.Get("action"); action != "" {
		query.Action = action
	}

	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			query.Since = &t
		}
	}

	if untilStr := q.Get("until"); untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			query.Until = &t
		}
	}

	// Execute search
	resp, err := h.service.Search(c.Request().Context(), query)
	if err != nil {
		// Handle structured errors
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to search audit events"))
	}

	// Set pagination headers
	if resp.Pagination != nil {
		c.SetHeader("X-Total-Count", strconv.FormatInt(resp.Pagination.Total, 10))
		c.SetHeader("X-Page", strconv.Itoa(resp.Pagination.CurrentPage))
		c.SetHeader("X-Total-Pages", strconv.Itoa(resp.Pagination.TotalPages))
		c.SetHeader("X-Search-Time-Ms", strconv.FormatInt(resp.TookMs, 10))
	}

	return c.JSON(http.StatusOK, resp)
}
