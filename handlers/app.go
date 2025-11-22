package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// AppHandler exposes HTTP endpoints for app (platform tenant) management
type AppHandler struct {
	app         *app.ServiceImpl
	rl          *rl.Service
	sess        session.ServiceInterface
	rbac        *rbac.Service
	roles       *repo.UserRoleRepository
	roleRepo    *repo.RoleRepository
	policyRepo  *repo.PolicyRepository
	enforceRBAC bool
}
func NewAppHandler(s *app.ServiceImpl, rlsvc *rl.Service, sess session.ServiceInterface, rbacsvc *rbac.Service, roles *repo.UserRoleRepository, roleRepo *repo.RoleRepository, policyRepo *repo.PolicyRepository, enforce bool) *AppHandler {
	return &AppHandler{app: s, rl: rlsvc, sess: sess, rbac: rbacsvc, roles: roles, roleRepo: roleRepo, policyRepo: policyRepo, enforceRBAC: enforce}
}

// handleError returns the error in a structured format
// If the error is already an AuthsomeError, return it as-is
// Otherwise, wrap it with the provided code and message
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// CreateOrganization creates a new organization
func (h *AppHandler) CreateOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	if !h.checkRBAC(c, "create", "organization:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	var req app.CreateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	org, err := h.app.CreateApp(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "CREATE_APP_FAILED", "Failed to create app", http.StatusBadRequest)
	}
	return c.JSON(http.StatusCreated, org)
}

// GetOrganizations supports fetching a single org by id or slug, or listing
func (h *AppHandler) GetOrganizations(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	if !h.checkRBAC(c, "read", "organization:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	q := c.Request().URL.Query()
	idStr := q.Get("id")
	slug := q.Get("slug")
	if idStr != "" {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_ID", "Invalid ID format", http.StatusBadRequest))
		}
		if !h.checkRBAC(c, "read", "organization:"+id.String(), &id) {
			return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
		}
		org, err := h.app.FindAppByID(c.Request().Context(), id)
		if err != nil {
			return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
		}
		return c.JSON(200, org)
	}
	if slug != "" {
		org, err := h.app.FindAppBySlug(c.Request().Context(), slug)
		if err != nil {
			return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
		}
		return c.JSON(200, org)
	}
	// List with pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := &app.ListAppsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
	}

	response, err := h.app.ListApps(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}

	return c.JSON(200, response)
}

// GetOrganizationByID fetches a single organization via path param
func (h *AppHandler) GetOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_ID", "ID parameter is required", http.StatusBadRequest))
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_ID", "Invalid ID format", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "read", "organization:"+id.String(), &id) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	org, err := h.app.FindAppByID(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	return c.JSON(200, org)
}

// UpdateOrganization updates an organization
func (h *AppHandler) UpdateOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID   xid.ID               `json:"id"`
		Data app.UpdateAppRequest `json:"data"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "organization:"+body.ID.String(), &body.ID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	org, err := h.app.UpdateApp(c.Request().Context(), body.ID, &body.Data)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, org)
}

// UpdateOrganizationByID updates an organization using path param id
func (h *AppHandler) UpdateOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_ID", "ID parameter is required", http.StatusBadRequest))
	}
	var req app.UpdateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_ID", "Invalid ID format", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "organization:"+id.String(), &id) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	org, err := h.app.UpdateApp(c.Request().Context(), id, &req)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, org)
}

// DeleteOrganization deletes an organization
func (h *AppHandler) DeleteOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "delete", "organization:"+body.ID.String(), &body.ID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.DeleteApp(c.Request().Context(), body.ID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "deleted"})
}

// DeleteOrganizationByID deletes an organization using path param id
func (h *AppHandler) DeleteOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_ID", "ID parameter is required", http.StatusBadRequest))
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_ID", "Invalid ID format", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "delete", "organization:"+id.String(), &id) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.DeleteApp(c.Request().Context(), id); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "deleted"})
}

// checkRBAC enforces RBAC when enabled; returns true when allowed
func (h *AppHandler) checkRBAC(c forge.Context, action string, resource string, orgID *xid.ID) bool {
	if !h.enforceRBAC || h.rbac == nil || h.sess == nil || h.roles == nil {
		return true
	}
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return false
	}
	token := cookie.Value
	sess, err := h.sess.FindByToken(c.Request().Context(), token)
	if err != nil || sess == nil {
		return false
	}
	// List roles for user in org
	var orgPtr *xid.ID
	if orgID != nil && !orgID.IsNil() {
		orgPtr = orgID
	}
	rs, err := h.roles.ListRolesForUser(c.Request().Context(), sess.UserID, orgPtr)
	if err != nil {
		rs = nil
	}
	roleNames := make([]string, 0, len(rs))
	for _, r := range rs {
		roleNames = append(roleNames, r.Name)
	}
	ctx := &rbac.Context{Subject: "user", Action: action, Resource: resource}
	return h.rbac.AllowedWithRoles(ctx, roleNames)
}

// CreateMember adds a new member to an organization
func (h *AppHandler) CreateMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var m app.Member
	if err := json.NewDecoder(c.Request().Body).Decode(&m); err != nil || m.AppID.IsNil() || m.UserID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "create", "member:*", &m.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	created, err := h.app.CreateMember(c.Request().Context(), &m)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, created)
}

// GetMembers lists members or fetches a single member
func (h *AppHandler) GetMembers(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	q := c.Request().URL.Query()
	memberIDStr := q.Get("id")
	if memberIDStr != "" {
		memberID, err := xid.FromString(memberIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_ID", "Invalid ID format", http.StatusBadRequest))
		}
		m, err := h.app.FindMemberByID(c.Request().Context(), memberID)
		if err != nil {
			return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
		}
		if !h.checkRBAC(c, "read", "member:"+memberID.String(), &m.AppID) {
			return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
		}
		return c.JSON(200, m)
	}
	orgIDStr := q.Get("org_id")
	if orgIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_ORG_ID", "Organization ID parameter is required", http.StatusBadRequest))
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_ORG_ID", "Invalid organization ID format", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "read", "member:*", &orgID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := &app.ListMembersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
		AppID: orgID,
	}

	response, err := h.app.ListMembers(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}

	return c.JSON(200, response)
}

// UpdateMember updates a member
func (h *AppHandler) UpdateMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var m app.Member
	if err := json.NewDecoder(c.Request().Body).Decode(&m); err != nil || m.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "member:"+m.ID.String(), &m.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.UpdateMember(c.Request().Context(), &m); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, m)
}

// DeleteMember deletes a member
func (h *AppHandler) DeleteMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// fetch member to determine org for RBAC
	m, err := h.app.FindMemberByID(c.Request().Context(), body.ID)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	if !h.checkRBAC(c, "delete", "member:"+body.ID.String(), &m.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.DeleteMember(c.Request().Context(), body.ID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "deleted"})
}

// CreateTeam creates a new team
func (h *AppHandler) CreateTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var t app.Team
	if err := json.NewDecoder(c.Request().Body).Decode(&t); err != nil || t.AppID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "create", "team:*", &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.CreateTeam(c.Request().Context(), &t); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, t)
}

// GetTeams lists teams in an organization
func (h *AppHandler) GetTeams(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	orgIDStr := c.Request().URL.Query().Get("org_id")
	if orgIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_ORG_ID", "Organization ID parameter is required", http.StatusBadRequest))
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_ORG_ID", "Invalid organization ID format", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "read", "team:*", &orgID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	q := c.Request().URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := &app.ListTeamsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
		AppID: orgID,
	}

	response, err := h.app.ListTeams(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}

	return c.JSON(200, response)
}

// UpdateTeam updates a team
func (h *AppHandler) UpdateTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var t app.Team
	if err := json.NewDecoder(c.Request().Body).Decode(&t); err != nil || t.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "team:"+t.ID.String(), &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.UpdateTeam(c.Request().Context(), &t); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, t)
}

// DeleteTeam deletes a team
func (h *AppHandler) DeleteTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// fetch team to determine org for RBAC
	t, err := h.app.FindTeamByID(c.Request().Context(), body.ID)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	if !h.checkRBAC(c, "delete", "team:"+body.ID.String(), &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.DeleteTeam(c.Request().Context(), body.ID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "deleted"})
}

// AddTeamMember adds a member to a team
func (h *AppHandler) AddTeamMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var tm app.TeamMember
	if err := json.NewDecoder(c.Request().Body).Decode(&tm); err != nil || tm.TeamID.IsNil() || tm.MemberID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// fetch team to determine org for RBAC
	t, err := h.app.FindTeamByID(c.Request().Context(), tm.TeamID)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	if !h.checkRBAC(c, "update", "team:"+tm.TeamID.String(), &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	added, err := h.app.AddTeamMember(c.Request().Context(), &tm)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, added)
}

// RemoveTeamMember removes a member from a team
func (h *AppHandler) RemoveTeamMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct{ TeamID, MemberID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.TeamID.IsNil() || body.MemberID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// fetch team to determine org for RBAC
	t, err := h.app.FindTeamByID(c.Request().Context(), body.TeamID)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	if !h.checkRBAC(c, "update", "team:"+body.TeamID.String(), &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.RemoveTeamMember(c.Request().Context(), body.TeamID, body.MemberID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "removed"})
}

// GetTeamMembers lists members of a team
func (h *AppHandler) GetTeamMembers(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	teamIDStr := c.Request().URL.Query().Get("team_id")
	if teamIDStr == "" {
		return c.JSON(400, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}
	// fetch team to determine org for RBAC
	t, err := h.app.FindTeamByID(c.Request().Context(), teamID)
	if err != nil {
		return handleError(c, err, "NOT_FOUND", "Resource not found", http.StatusNotFound)
	}
	if !h.checkRBAC(c, "read", "team:"+teamID.String(), &t.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	q := c.Request().URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := &app.ListTeamMembersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
		TeamID: teamID,
	}

	response, err := h.app.ListTeamMembers(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}

	return c.JSON(200, response)
}

// CreateInvitation creates an invitation
func (h *AppHandler) CreateInvitation(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var inv app.Invitation
	if err := json.NewDecoder(c.Request().Body).Decode(&inv); err != nil || inv.AppID.IsNil() || inv.Email == "" {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "create", "invitation:*", &inv.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.app.CreateInvitation(c.Request().Context(), &inv); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, inv)
}

// CreatePolicy creates a new RBAC policy expression
func (h *AppHandler) CreatePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Expression == "" {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "create", "policy:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if h.policyRepo == nil {
		return c.JSON(500, errs.New("ERROR", "policy repository not configured", http.StatusBadRequest))
	}
	if err := h.policyRepo.Create(c.Request().Context(), body.Expression); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(201, &StatusResponse{Status: "created"})
}

// GetPolicies lists stored RBAC policy expressions
func (h *AppHandler) GetPolicies(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	if !h.checkRBAC(c, "read", "policy:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if h.policyRepo == nil {
		return c.JSON(500, errs.New("ERROR", "policy repository not configured", http.StatusBadRequest))
	}
	rows, err := h.policyRepo.List(c.Request().Context())
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	total := len(rows)
	// For now, policies list is unpaginated; provide single-page envelope
	return c.JSON(200, types.PaginatedResult{
		Data:       rows,
		Total:      total,
		Page:       1,
		PageSize:   total,
		TotalPages: 1,
	})
}

// DeletePolicy deletes a policy by ID
func (h *AppHandler) DeletePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "delete", "policy:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if h.policyRepo == nil {
		return c.JSON(500, errs.New("ERROR", "policy repository not configured", http.StatusBadRequest))
	}
	if err := h.policyRepo.Delete(c.Request().Context(), body.ID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(200, &StatusResponse{Status: "deleted"})
}

// UpdatePolicy updates an existing RBAC policy expression by ID
func (h *AppHandler) UpdatePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		ID         xid.ID `json:"id"`
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() || strings.TrimSpace(body.Expression) == "" {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "policy:*", nil) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if h.policyRepo == nil {
		return c.JSON(500, errs.New("ERROR", "policy repository not configured", http.StatusBadRequest))
	}
	// Validate expression syntax using RBAC parser
	parser := rbac.NewParser()
	if _, err := parser.Parse(body.Expression); err != nil {
		return c.JSON(400, errs.InvalidPolicy("invalid expression: "+err.Error()))
	}
	if err := h.policyRepo.Update(c.Request().Context(), body.ID, body.Expression); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(200, &StatusResponse{Status: "updated"})
}

// CreateRole creates a role, optionally scoped to an organization
func (h *AppHandler) CreateRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct {
		AppID       *xid.ID `json:"organization_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Name == "" {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	var orgPtr *xid.ID
	if body.AppID != nil && !body.AppID.IsNil() {
		orgPtr = body.AppID
	}
	if !h.checkRBAC(c, "create", "role:*", orgPtr) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	var orgIDPtr *xid.ID
	if orgPtr != nil {
		orgIDPtr = orgPtr
	}
	role := &schema.Role{AppID: orgIDPtr, Name: body.Name, Description: body.Description}
	if h.roleRepo == nil {
		return c.JSON(500, errs.New("ERROR", "role repository not configured", http.StatusBadRequest))
	}
	if err := h.roleRepo.Create(c.Request().Context(), role); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, role)
}

// GetRoles lists roles, optionally filtered by organization
func (h *AppHandler) GetRoles(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	q := c.Request().URL.Query()
	orgIDStr := q.Get("org_id")
	var orgIDPtr *xid.ID
	var orgStrPtr *string
	if orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_ORG_ID", "Invalid organization ID format", http.StatusBadRequest))
		}
		orgIDPtr = &id
		s := id.String()
		orgStrPtr = &s
	}
	if !h.checkRBAC(c, "read", "role:*", orgIDPtr) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if h.roleRepo == nil {
		return c.JSON(500, errs.New("ERROR", "role repository not configured", http.StatusBadRequest))
	}
	roles, err := h.roleRepo.ListByOrg(c.Request().Context(), orgStrPtr)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	total := len(roles)
	return c.JSON(200, types.PaginatedResult{Data: roles, Total: total, Page: 1, PageSize: total, TotalPages: 1})
}

// AssignUserRole assigns a role to a user within an organization
func (h *AppHandler) AssignUserRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct{ UserID, RoleID, AppID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.UserID.IsNil() || body.RoleID.IsNil() || body.AppID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "role:*", &body.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.roles.Assign(c.Request().Context(), body.UserID, body.RoleID, body.AppID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(201, &StatusResponse{Status: "assigned"})
}

// RemoveUserRole removes a role assignment from a user within an organization
func (h *AppHandler) RemoveUserRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	var body struct{ UserID, RoleID, AppID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.UserID.IsNil() || body.RoleID.IsNil() || body.AppID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if !h.checkRBAC(c, "update", "role:*", &body.AppID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	if err := h.roles.Unassign(c.Request().Context(), body.UserID, body.RoleID, body.AppID); err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	return c.JSON(200, &StatusResponse{Status: "removed"})
}

// GetUserRoles lists roles assigned to a user, optionally filtered by organization
func (h *AppHandler) GetUserRoles(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	q := c.Request().URL.Query()
	userIDStr := q.Get("user_id")
	if userIDStr == "" {
		return c.JSON(400, errs.New("MISSING_USER_ID", "User ID parameter is required", http.StatusBadRequest))
	}
	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest))
	}
	var orgPtr *xid.ID
	orgIDStr := q.Get("org_id")
	if orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_ORG_ID", "Invalid organization ID format", http.StatusBadRequest))
		}
		orgPtr = &id
	}
	if !h.checkRBAC(c, "read", "role:*", orgPtr) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}
	roles, err := h.roles.ListRolesForUser(c.Request().Context(), userID, orgPtr)
	if err != nil {
		return handleError(c, err, "BAD_REQUEST", "Bad request", http.StatusBadRequest)
	}
	total := len(roles)
	return c.JSON(200, types.PaginatedResult{Data: roles, Total: total, Page: 1, PageSize: total, TotalPages: 1})
}

// GetAppCookieConfig retrieves the cookie configuration for a specific app
// GET /apps/:appId/cookie-config
func (h *AppHandler) GetAppCookieConfig(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}

	appIDStr := c.Param("appId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	if !h.checkRBAC(c, "read", "organization:"+appID.String(), &appID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}

	cookieConfig, err := h.app.App.GetCookieConfig(c.Request().Context(), appID)
	if err != nil {
		return handleError(c, err, "GET_COOKIE_CONFIG_FAILED", "Failed to get cookie configuration", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, cookieConfig)
}

// UpdateAppCookieConfig updates the cookie configuration for a specific app
// PUT /apps/:appId/cookie-config
func (h *AppHandler) UpdateAppCookieConfig(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}

	appIDStr := c.Param("appId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	if !h.checkRBAC(c, "update", "organization:"+appID.String(), &appID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}

	var cookieConfig session.CookieConfig
	if err := json.NewDecoder(c.Request().Body).Decode(&cookieConfig); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get the app to update its metadata
	existingApp, err := h.app.FindAppByID(c.Request().Context(), appID)
	if err != nil {
		return handleError(c, err, "APP_NOT_FOUND", "App not found", http.StatusNotFound)
	}

	// Update metadata with cookie config
	if existingApp.Metadata == nil {
		existingApp.Metadata = make(map[string]interface{})
	}
	existingApp.Metadata["sessionCookie"] = cookieConfig

	// Update the app
	updateReq := &app.UpdateAppRequest{
		Metadata: existingApp.Metadata,
	}
	updatedApp, err := h.app.UpdateApp(c.Request().Context(), appID, updateReq)
	if err != nil {
		return handleError(c, err, "UPDATE_COOKIE_CONFIG_FAILED", "Failed to update cookie configuration", http.StatusInternalServerError)
	}

	// Return the updated cookie config
	updatedConfig, _ := h.app.App.GetCookieConfig(c.Request().Context(), appID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"app":          updatedApp,
		"cookieConfig": updatedConfig,
	})
}

// DeleteAppCookieConfig removes the cookie configuration override for a specific app
// DELETE /apps/:appId/cookie-config
func (h *AppHandler) DeleteAppCookieConfig(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}

	appIDStr := c.Param("appId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	if !h.checkRBAC(c, "update", "organization:"+appID.String(), &appID) {
		return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Access forbidden", http.StatusForbidden))
	}

	// Get the app to update its metadata
	existingApp, err := h.app.FindAppByID(c.Request().Context(), appID)
	if err != nil {
		return handleError(c, err, "APP_NOT_FOUND", "App not found", http.StatusNotFound)
	}

	// Remove cookie config from metadata
	if existingApp.Metadata != nil {
		delete(existingApp.Metadata, "sessionCookie")
	}

	// Update the app
	updateReq := &app.UpdateAppRequest{
		Metadata: existingApp.Metadata,
	}
	_, err = h.app.UpdateApp(c.Request().Context(), appID, updateReq)
	if err != nil {
		return handleError(c, err, "DELETE_COOKIE_CONFIG_FAILED", "Failed to delete cookie configuration", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "deleted",
		"message": "Cookie configuration removed, using global defaults",
	})
}
