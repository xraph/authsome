package handlers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	rl "github.com/xraph/authsome/core/ratelimit"
	rbac "github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// OrganizationHandler exposes HTTP endpoints for organization management
type OrganizationHandler struct {
	org         *organization.Service
	rl          *rl.Service
	sess        session.ServiceInterface
	rbac        *rbac.Service
	roles       *repo.UserRoleRepository
	roleRepo    *repo.RoleRepository
	policyRepo  *repo.PolicyRepository
	enforceRBAC bool
}

func NewOrganizationHandler(s *organization.Service, rlsvc *rl.Service, sess session.ServiceInterface, rbacsvc *rbac.Service, roles *repo.UserRoleRepository, roleRepo *repo.RoleRepository, policyRepo *repo.PolicyRepository, enforce bool) *OrganizationHandler {
	return &OrganizationHandler{org: s, rl: rlsvc, sess: sess, rbac: rbacsvc, roles: roles, roleRepo: roleRepo, policyRepo: policyRepo, enforceRBAC: enforce}
}

// CreateOrganization creates a new organization
func (h *OrganizationHandler) CreateOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	if !h.checkRBAC(c, "create", "organization:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	var req organization.CreateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	org, err := h.org.CreateOrganization(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, org)
}

// GetOrganizations supports fetching a single org by id or slug, or listing
func (h *OrganizationHandler) GetOrganizations(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	if !h.checkRBAC(c, "read", "organization:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	q := c.Request().URL.Query()
	idStr := q.Get("id")
	slug := q.Get("slug")
	if idStr != "" {
		id, err := xid.FromString(idStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid id"})
		}
		if !h.checkRBAC(c, "read", "organization:"+id.String(), &id) {
			return c.JSON(403, map[string]string{"error": "forbidden"})
		}
		org, err := h.org.FindOrganizationByID(c.Request().Context(), id)
		if err != nil {
			return c.JSON(404, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, org)
	}
	if slug != "" {
		org, err := h.org.FindOrganizationBySlug(c.Request().Context(), slug)
		if err != nil {
			return c.JSON(404, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, org)
	}
	// List with pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	orgs, err := h.org.ListOrganizations(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total, err := h.org.CountOrganizations(c.Request().Context())
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	page := 1
	pageSize := len(orgs)
	totalPages := 1
	if limit > 0 {
		page = (offset / limit) + 1
		pageSize = limit
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		} else {
			totalPages = 0
		}
	}
	return c.JSON(200, types.PaginatedResult{Data: orgs, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages})
}

// GetOrganizationByID fetches a single organization via path param
func (h *OrganizationHandler) GetOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	if !h.checkRBAC(c, "read", "organization:"+id.String(), &id) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	org, err := h.org.FindOrganizationByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, org)
}

// UpdateOrganization updates an organization
func (h *OrganizationHandler) UpdateOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID   xid.ID                                 `json:"id"`
		Data organization.UpdateOrganizationRequest `json:"data"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "organization:"+body.ID.String(), &body.ID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	org, err := h.org.UpdateOrganization(c.Request().Context(), body.ID, &body.Data)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, org)
}

// UpdateOrganizationByID updates an organization using path param id
func (h *OrganizationHandler) UpdateOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}
	var req organization.UpdateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	if !h.checkRBAC(c, "update", "organization:"+id.String(), &id) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	org, err := h.org.UpdateOrganization(c.Request().Context(), id, &req)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, org)
}

// DeleteOrganization deletes an organization
func (h *OrganizationHandler) DeleteOrganization(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "delete", "organization:"+body.ID.String(), &body.ID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.DeleteOrganization(c.Request().Context(), body.ID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// DeleteOrganizationByID deletes an organization using path param id
func (h *OrganizationHandler) DeleteOrganizationByID(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	if !h.checkRBAC(c, "delete", "organization:"+id.String(), &id) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.DeleteOrganization(c.Request().Context(), id); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// checkRBAC enforces RBAC when enabled; returns true when allowed
func (h *OrganizationHandler) checkRBAC(c forge.Context, action string, resource string, orgID *xid.ID) bool {
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
func (h *OrganizationHandler) CreateMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var m organization.Member
	if err := json.NewDecoder(c.Request().Body).Decode(&m); err != nil || m.OrganizationID.IsNil() || m.UserID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "create", "member:*", &m.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.CreateMember(c.Request().Context(), &m); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, m)
}

// GetMembers lists members or fetches a single member
func (h *OrganizationHandler) GetMembers(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	q := c.Request().URL.Query()
	memberIDStr := q.Get("id")
	if memberIDStr != "" {
		memberID, err := xid.FromString(memberIDStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid id"})
		}
		m, err := h.org.FindMemberByID(c.Request().Context(), memberID)
		if err != nil {
			return c.JSON(404, map[string]string{"error": err.Error()})
		}
		if !h.checkRBAC(c, "read", "member:"+memberID.String(), &m.OrganizationID) {
			return c.JSON(403, map[string]string{"error": "forbidden"})
		}
		return c.JSON(200, m)
	}
	orgIDStr := q.Get("org_id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "org_id required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid org_id"})
	}
	if !h.checkRBAC(c, "read", "member:*", &orgID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	ms, err := h.org.ListMembers(c.Request().Context(), orgID, limit, offset)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total, err := h.org.CountMembers(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	page := 1
	pageSize := len(ms)
	totalPages := 1
	if limit > 0 {
		page = (offset / limit) + 1
		pageSize = limit
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		} else {
			totalPages = 0
		}
	}
	return c.JSON(200, types.PaginatedResult{Data: ms, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages})
}

// UpdateMember updates a member
func (h *OrganizationHandler) UpdateMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var m organization.Member
	if err := json.NewDecoder(c.Request().Body).Decode(&m); err != nil || m.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "member:"+m.ID.String(), &m.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.UpdateMember(c.Request().Context(), &m); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, m)
}

// DeleteMember deletes a member
func (h *OrganizationHandler) DeleteMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// fetch member to determine org for RBAC
	m, err := h.org.FindMemberByID(c.Request().Context(), body.ID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	if !h.checkRBAC(c, "delete", "member:"+body.ID.String(), &m.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.DeleteMember(c.Request().Context(), body.ID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// CreateTeam creates a new team
func (h *OrganizationHandler) CreateTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var t organization.Team
	if err := json.NewDecoder(c.Request().Body).Decode(&t); err != nil || t.OrganizationID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "create", "team:*", &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.CreateTeam(c.Request().Context(), &t); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, t)
}

// GetTeams lists teams in an organization
func (h *OrganizationHandler) GetTeams(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	orgIDStr := c.Request().URL.Query().Get("org_id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "org_id required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid org_id"})
	}
	if !h.checkRBAC(c, "read", "team:*", &orgID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	q := c.Request().URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	ts, err := h.org.ListTeams(c.Request().Context(), orgID, limit, offset)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total, err := h.org.CountTeams(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	page := 1
	pageSize := len(ts)
	totalPages := 1
	if limit > 0 {
		page = (offset / limit) + 1
		pageSize = limit
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		} else {
			totalPages = 0
		}
	}
	return c.JSON(200, types.PaginatedResult{Data: ts, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages})
}

// UpdateTeam updates a team
func (h *OrganizationHandler) UpdateTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var t organization.Team
	if err := json.NewDecoder(c.Request().Body).Decode(&t); err != nil || t.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "team:"+t.ID.String(), &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.UpdateTeam(c.Request().Context(), &t); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, t)
}

// DeleteTeam deletes a team
func (h *OrganizationHandler) DeleteTeam(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// fetch team to determine org for RBAC
	t, err := h.org.FindTeamByID(c.Request().Context(), body.ID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	if !h.checkRBAC(c, "delete", "team:"+body.ID.String(), &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.DeleteTeam(c.Request().Context(), body.ID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// AddTeamMember adds a member to a team
func (h *OrganizationHandler) AddTeamMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var tm organization.TeamMember
	if err := json.NewDecoder(c.Request().Body).Decode(&tm); err != nil || tm.TeamID.IsNil() || tm.MemberID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// fetch team to determine org for RBAC
	t, err := h.org.FindTeamByID(c.Request().Context(), tm.TeamID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	if !h.checkRBAC(c, "update", "team:"+tm.TeamID.String(), &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.AddTeamMember(c.Request().Context(), &tm); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, tm)
}

// RemoveTeamMember removes a member from a team
func (h *OrganizationHandler) RemoveTeamMember(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct{ TeamID, MemberID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.TeamID.IsNil() || body.MemberID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// fetch team to determine org for RBAC
	t, err := h.org.FindTeamByID(c.Request().Context(), body.TeamID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	if !h.checkRBAC(c, "update", "team:"+body.TeamID.String(), &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.RemoveTeamMember(c.Request().Context(), body.TeamID, body.MemberID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "removed"})
}

// GetTeamMembers lists members of a team
func (h *OrganizationHandler) GetTeamMembers(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	teamIDStr := c.Request().URL.Query().Get("team_id")
	if teamIDStr == "" {
		return c.JSON(400, map[string]string{"error": "team_id required"})
	}
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team_id"})
	}
	// fetch team to determine org for RBAC
	t, err := h.org.FindTeamByID(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	if !h.checkRBAC(c, "read", "team:"+teamID.String(), &t.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	q := c.Request().URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	ms, err := h.org.ListTeamMembers(c.Request().Context(), teamID, limit, offset)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total, err := h.org.CountTeamMembers(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	page := 1
	pageSize := len(ms)
	totalPages := 1
	if limit > 0 {
		page = (offset / limit) + 1
		pageSize = limit
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		} else {
			totalPages = 0
		}
	}
	return c.JSON(200, types.PaginatedResult{Data: ms, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages})
}

// CreateInvitation creates an invitation
func (h *OrganizationHandler) CreateInvitation(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var inv organization.Invitation
	if err := json.NewDecoder(c.Request().Body).Decode(&inv); err != nil || inv.OrganizationID.IsNil() || inv.Email == "" {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "create", "invitation:*", &inv.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.org.CreateInvitation(c.Request().Context(), &inv); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, inv)
}

// CreatePolicy creates a new RBAC policy expression
func (h *OrganizationHandler) CreatePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Expression == "" {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "create", "policy:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if h.policyRepo == nil {
		return c.JSON(500, map[string]string{"error": "policy repository not configured"})
	}
	if err := h.policyRepo.Create(c.Request().Context(), body.Expression); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(201, map[string]string{"status": "created"})
}

// GetPolicies lists stored RBAC policy expressions
func (h *OrganizationHandler) GetPolicies(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	if !h.checkRBAC(c, "read", "policy:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if h.policyRepo == nil {
		return c.JSON(500, map[string]string{"error": "policy repository not configured"})
	}
	rows, err := h.policyRepo.List(c.Request().Context())
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
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
func (h *OrganizationHandler) DeletePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID xid.ID `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "delete", "policy:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if h.policyRepo == nil {
		return c.JSON(500, map[string]string{"error": "policy repository not configured"})
	}
	if err := h.policyRepo.Delete(c.Request().Context(), body.ID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// UpdatePolicy updates an existing RBAC policy expression by ID
func (h *OrganizationHandler) UpdatePolicy(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		ID         xid.ID `json:"id"`
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.ID.IsNil() || strings.TrimSpace(body.Expression) == "" {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "policy:*", nil) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if h.policyRepo == nil {
		return c.JSON(500, map[string]string{"error": "policy repository not configured"})
	}
	// Validate expression syntax using RBAC parser
	parser := rbac.NewParser()
	if _, err := parser.Parse(body.Expression); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid expression: " + err.Error()})
	}
	if err := h.policyRepo.Update(c.Request().Context(), body.ID, body.Expression); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	// Reload policies into RBAC service
	if h.rbac != nil {
		_ = h.rbac.LoadPolicies(c.Request().Context(), h.policyRepo)
	}
	return c.JSON(200, map[string]string{"status": "updated"})
}

// CreateRole creates a role, optionally scoped to an organization
func (h *OrganizationHandler) CreateRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct {
		OrganizationID *xid.ID `json:"organization_id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Name == "" {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	var orgPtr *xid.ID
	if body.OrganizationID != nil && !body.OrganizationID.IsNil() {
		orgPtr = body.OrganizationID
	}
	if !h.checkRBAC(c, "create", "role:*", orgPtr) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	var orgIDPtr *xid.ID
	if orgPtr != nil {
		orgIDPtr = orgPtr
	}
	role := &schema.Role{OrganizationID: orgIDPtr, Name: body.Name, Description: body.Description}
	if h.roleRepo == nil {
		return c.JSON(500, map[string]string{"error": "role repository not configured"})
	}
	if err := h.roleRepo.Create(c.Request().Context(), role); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, role)
}

// GetRoles lists roles, optionally filtered by organization
func (h *OrganizationHandler) GetRoles(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	q := c.Request().URL.Query()
	orgIDStr := q.Get("org_id")
	var orgIDPtr *xid.ID
	var orgStrPtr *string
	if orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid org_id"})
		}
		orgIDPtr = &id
		s := id.String()
		orgStrPtr = &s
	}
	if !h.checkRBAC(c, "read", "role:*", orgIDPtr) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if h.roleRepo == nil {
		return c.JSON(500, map[string]string{"error": "role repository not configured"})
	}
	roles, err := h.roleRepo.ListByOrg(c.Request().Context(), orgStrPtr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total := len(roles)
	return c.JSON(200, types.PaginatedResult{Data: roles, Total: total, Page: 1, PageSize: total, TotalPages: 1})
}

// AssignUserRole assigns a role to a user within an organization
func (h *OrganizationHandler) AssignUserRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct{ UserID, RoleID, OrganizationID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.UserID.IsNil() || body.RoleID.IsNil() || body.OrganizationID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "role:*", &body.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.roles.Assign(c.Request().Context(), body.UserID, body.RoleID, body.OrganizationID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, map[string]string{"status": "assigned"})
}

// RemoveUserRole removes a role assignment from a user within an organization
func (h *OrganizationHandler) RemoveUserRole(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	var body struct{ UserID, RoleID, OrganizationID xid.ID }
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.UserID.IsNil() || body.RoleID.IsNil() || body.OrganizationID.IsNil() {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if !h.checkRBAC(c, "update", "role:*", &body.OrganizationID) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	if err := h.roles.Unassign(c.Request().Context(), body.UserID, body.RoleID, body.OrganizationID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "removed"})
}

// GetUserRoles lists roles assigned to a user, optionally filtered by organization
func (h *OrganizationHandler) GetUserRoles(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
		}
	}
	q := c.Request().URL.Query()
	userIDStr := q.Get("user_id")
	if userIDStr == "" {
		return c.JSON(400, map[string]string{"error": "user_id required"})
	}
	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid user_id"})
	}
	var orgPtr *xid.ID
	orgIDStr := q.Get("org_id")
	if orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid org_id"})
		}
		orgPtr = &id
	}
	if !h.checkRBAC(c, "read", "role:*", orgPtr) {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	roles, err := h.roles.ListRolesForUser(c.Request().Context(), userID, orgPtr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	total := len(roles)
	return c.JSON(200, types.PaginatedResult{Data: roles, Total: total, Page: 1, PageSize: total, TotalPages: 1})
}
