package handlers

import (
	"net/http"
	"strconv"
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
// Query params: limit, offset, userId, action, resource, ipAddress, since, until, sortBy, sortOrder
func (h *AuditHandler) ListEvents(c forge.Context) error {
	if h.service == nil {
		return c.JSON(http.StatusNotImplemented, &ErrorResponse{Error: "audit service not available",})
	}

	q := c.Request().URL.Query()

	// Parse pagination parameters
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50 // Default limit
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &audit.ListEventsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
	}

	// Parse optional filters
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

	// Parse time range filters
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

	// Parse sorting parameters
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
		return c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: "Failed to list audit events",})
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
