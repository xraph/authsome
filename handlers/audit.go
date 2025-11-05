package handlers

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
	"net/http"
	"strconv"
	"time"
)

// AuditHandler exposes endpoints to query audit events
type AuditHandler struct {
	aud *audit.Service
}

func NewAuditHandler(a *audit.Service) *AuditHandler { return &AuditHandler{aud: a} }

// ListEvents returns recent audit events with pagination via query params: limit, offset
func (h *AuditHandler) ListEvents(c forge.Context) error {
	if h.aud == nil {
		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "audit service not available"})
	}
	q := c.Request().URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	// Optional filters
	var params audit.ListParams
	params.Limit = limit
	params.Offset = offset
	if uidStr := q.Get("userId"); uidStr != "" {
		if uid, err := xid.FromString(uidStr); err == nil {
			params.UserID = &uid
		}
	}
	if action := q.Get("action"); action != "" {
		params.Action = action
	}
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			params.Since = &t
		}
	}
	if untilStr := q.Get("until"); untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			params.Until = &t
		}
	}
	var (
		events []*audit.Event
		total  int
		err    error
	)
	if params.UserID != nil || params.Action != "" || params.Since != nil || params.Until != nil {
		events, total, err = h.aud.SearchWithTotal(c.Request().Context(), params)
	} else {
		events, total, err = h.aud.ListWithTotal(c.Request().Context(), limit, offset)
	}
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	c.SetHeader("X-Total-Count", strconv.Itoa(total))
	// Build RFC 5988 Link headers for pagination
	// prev: offset - limit (>=0), next: offset + limit (< total)
	links := ""
	// Use existing query and update limit/offset
	q.Set("limit", strconv.Itoa(params.Limit))
	// prev
	if params.Offset > 0 {
		prev := params.Offset - params.Limit
		if prev < 0 {
			prev = 0
		}
		q.Set("offset", strconv.Itoa(prev))
		links = links + "<" + c.Request().URL.Path + "?" + q.Encode() + ">; rel=\"prev\""
	}
	// next
	if params.Offset+params.Limit < total {
		if links != "" {
			links += ", "
		}
		next := params.Offset + params.Limit
		q.Set("offset", strconv.Itoa(next))
		links = links + "<" + c.Request().URL.Path + "?" + q.Encode() + ">; rel=\"next\""
	}
	if links != "" {
		c.SetHeader("Link", links)
	}
	// JSON body pagination envelope
	page := 1
	if params.Limit > 0 {
		page = (params.Offset / params.Limit) + 1
	}
	pageSize := params.Limit
	if pageSize <= 0 {
		pageSize = len(events)
	}
	totalPages := 1
	if pageSize > 0 && total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return c.JSON(200, types.PaginatedResult{
		Data:       events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}
