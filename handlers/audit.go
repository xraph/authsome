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
//
//	ipAddress, ipAddresses, ipRange, since, until, sortBy, sortOrder
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

// =============================================================================
// AGGREGATION ENDPOINTS
// =============================================================================

// GetAggregations returns all aggregations in one call
// GET /audit/aggregations
func (h *AuditHandler) GetAggregations(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetAllAggregations(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get aggregations"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctActions returns distinct actions with counts
// GET /audit/actions
func (h *AuditHandler) GetDistinctActions(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctActions(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct actions"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctSources returns distinct sources with counts
// GET /audit/sources
func (h *AuditHandler) GetDistinctSources(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctSources(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct sources"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctResources returns distinct resources with counts
// GET /audit/resources
func (h *AuditHandler) GetDistinctResources(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctResources(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct resources"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctUsers returns distinct users with counts
// GET /audit/users
func (h *AuditHandler) GetDistinctUsers(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctUsers(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct users"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctIPs returns distinct IP addresses with counts
// GET /audit/ips
func (h *AuditHandler) GetDistinctIPs(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctIPs(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct IPs"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctApps returns distinct apps with counts
// GET /audit/apps
func (h *AuditHandler) GetDistinctApps(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctApps(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct apps"))
	}

	return c.JSON(http.StatusOK, result)
}

// GetDistinctOrganizations returns distinct organizations with counts
// GET /audit/organizations
func (h *AuditHandler) GetDistinctOrganizations(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, errs.NotImplemented("audit service"))
	}

	filter := h.parseAggregationFilter(c)

	result, err := h.service.GetDistinctOrganizations(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Failed to get distinct organizations"))
	}

	return c.JSON(http.StatusOK, result)
}

// parseAggregationFilter parses query parameters into AggregationFilter
func (h *AuditHandler) parseAggregationFilter(c forge.Context) *audit.AggregationFilter {
	q := c.Request().URL.Query()

	filter := &audit.AggregationFilter{}

	// Parse IDs
	if appID := q.Get("app_id"); appID != "" {
		if id, err := xid.FromString(appID); err == nil {
			filter.AppID = &id
		}
	}

	if orgID := q.Get("organization_id"); orgID != "" {
		if id, err := xid.FromString(orgID); err == nil {
			filter.OrganizationID = &id
		}
	}

	if envID := q.Get("environment_id"); envID != "" {
		if id, err := xid.FromString(envID); err == nil {
			filter.EnvironmentID = &id
		}
	}

	// Parse time range
	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = &t
		}
	}

	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = &t
		}
	}

	// Parse limit
	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	return filter
}
