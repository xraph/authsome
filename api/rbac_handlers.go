package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
)

// ──────────────────────────────────────────────────
// RBAC route registration
// ──────────────────────────────────────────────────

func (a *API) registerRBACRoutes(router forge.Router) error {
	g := router.Group("/v1",
		forge.WithGroupTags("RBAC"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
		),
	)

	// Role CRUD
	if err := g.POST("/roles", a.handleCreateRole,
		forge.WithSummary("Create role"),
		forge.WithDescription("Creates a new RBAC role."),
		forge.WithOperationID("authsomeCreateRole"),
		forge.WithRequestSchema(CreateRoleRequest{}),
		forge.WithCreatedResponse(rbac.Role{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "create", "role")),
	); err != nil {
		return err
	}

	if err := g.GET("/roles", a.handleListRoles,
		forge.WithSummary("List roles"),
		forge.WithDescription("Returns all RBAC roles for an app."),
		forge.WithOperationID("authsomeListRoles"),
		forge.WithResponseSchema(http.StatusOK, "Role list", RoleListResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "read", "role")),
	); err != nil {
		return err
	}

	if err := g.GET("/roles/:roleId", a.handleGetRole,
		forge.WithSummary("Get role"),
		forge.WithDescription("Returns a role by ID."),
		forge.WithOperationID("authsomeGetRole"),
		forge.WithRequestSchema(GetRoleRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Role details", rbac.Role{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "read", "role")),
	); err != nil {
		return err
	}

	if err := g.PATCH("/roles/:roleId", a.handleUpdateRole,
		forge.WithSummary("Update role"),
		forge.WithDescription("Updates a role's name or description."),
		forge.WithOperationID("authsomeUpdateRole"),
		forge.WithRequestSchema(UpdateRoleRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated role", rbac.Role{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "update", "role")),
	); err != nil {
		return err
	}

	if err := g.DELETE("/roles/:roleId", a.handleDeleteRole,
		forge.WithSummary("Delete role"),
		forge.WithDescription("Deletes a role and its permissions and assignments."),
		forge.WithOperationID("authsomeDeleteRole"),
		forge.WithRequestSchema(DeleteRoleRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "delete", "role")),
	); err != nil {
		return err
	}

	// Permission management
	if err := g.POST("/roles/:roleId/permissions", a.handleAddPermission,
		forge.WithSummary("Add permission to role"),
		forge.WithDescription("Adds a permission (action + resource) to a role."),
		forge.WithOperationID("authsomeAddPermission"),
		forge.WithRequestSchema(AddPermissionRequest{}),
		forge.WithCreatedResponse(rbac.Permission{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "create", "permission")),
	); err != nil {
		return err
	}

	if err := g.GET("/roles/:roleId/permissions", a.handleListRolePermissions,
		forge.WithSummary("List role permissions"),
		forge.WithDescription("Returns all permissions for a role."),
		forge.WithOperationID("authsomeListRolePermissions"),
		forge.WithRequestSchema(ListRolePermissionsRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Permission list", PermissionListResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "read", "permission")),
	); err != nil {
		return err
	}

	if err := g.DELETE("/roles/:roleId/permissions/:permissionId", a.handleRemovePermission,
		forge.WithSummary("Remove permission from role"),
		forge.WithDescription("Removes a permission from a role."),
		forge.WithOperationID("authsomeRemovePermission"),
		forge.WithRequestSchema(RemovePermissionRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Removed", StatusResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "delete", "permission")),
	); err != nil {
		return err
	}

	// Role assignment
	if err := g.POST("/roles/:roleId/assign", a.handleAssignRole,
		forge.WithSummary("Assign role to user"),
		forge.WithDescription("Assigns a role to a user, optionally scoped to an organization."),
		forge.WithOperationID("authsomeAssignRole"),
		forge.WithRequestSchema(AssignRoleRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Assigned", StatusResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "assign", "role")),
	); err != nil {
		return err
	}

	if err := g.POST("/roles/:roleId/unassign", a.handleUnassignRole,
		forge.WithSummary("Unassign role from user"),
		forge.WithDescription("Removes a role assignment from a user."),
		forge.WithOperationID("authsomeUnassignRole"),
		forge.WithRequestSchema(UnassignRoleRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Unassigned", StatusResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "unassign", "role")),
	); err != nil {
		return err
	}

	// User roles
	return g.GET("/users/:userId/roles", a.handleListUserRoles,
		forge.WithSummary("List user roles"),
		forge.WithDescription("Returns all roles assigned to a user."),
		forge.WithOperationID("authsomeListUserRoles"),
		forge.WithRequestSchema(ListUserRolesRequest{}),
		forge.WithResponseSchema(http.StatusOK, "User role list", UserRoleListResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RequirePermission(a.engine, "read", "role")),
	)
}

// ──────────────────────────────────────────────────
// RBAC handlers
// ──────────────────────────────────────────────────

// roleInCallerApp loads a role and verifies it belongs to the caller's tenant
// app. A missing role and a role owned by another app both return 404, so a
// caller can neither read nor infer the existence of another tenant's roles.
func (a *API) roleInCallerApp(ctx forge.Context, roleID id.RoleID) (*rbac.Role, error) {
	appID, ok := a.callerAppID(ctx)
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	r, err := a.engine.GetRole(ctx.Context(), roleID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	if r.AppID != appID.String() {
		return nil, forge.NotFound("role not found")
	}
	return r, nil
}

// ownerSlugs are the roles that confer ownership. Only a caller holding one
// may grant one, and only such a caller may change their own roles.
var ownerSlugs = map[string]bool{"owner": true, rbac.PlatformOwnerSlug: true}

// callerHoldsOwnership reports whether the caller holds an owner role in
// their app or is a platform owner.
func (a *API) callerHoldsOwnership(ctx forge.Context, callerID id.UserID) (bool, error) {
	roles, err := a.engine.ListUserRoles(ctx.Context(), callerID)
	if err != nil {
		return false, mapErrorCtx(ctx, err)
	}
	for _, r := range roles {
		if ownerSlugs[r.Slug] {
			return true, nil
		}
	}
	return false, nil
}

// requireCallerHolds refuses unless the caller holds every listed
// permission. It is what keeps a grant from exceeding the granter.
func (a *API) requireCallerHolds(ctx forge.Context, perms []*rbac.Permission) error {
	callerID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return forge.Unauthorized("authentication required")
	}
	for _, p := range perms {
		held, err := a.engine.HasPermission(ctx.Context(), callerID, p.Action, p.Resource)
		if err != nil {
			return mapErrorCtx(ctx, err)
		}
		if !held {
			return forge.Forbidden(fmt.Sprintf("cannot grant %s:%s: the caller does not hold it", p.Resource, p.Action))
		}
	}
	return nil
}

// effectivePermissions collects a role's own permissions plus every
// ancestor's, since assignment confers all of them.
func (a *API) effectivePermissions(ctx forge.Context, role *rbac.Role) ([]*rbac.Permission, error) {
	appID, err := id.ParseAppID(role.AppID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	var out []*rbac.Permission
	seen := map[string]bool{}
	current := role
	for current != nil && !seen[current.ID] {
		seen[current.ID] = true
		rid, err := storedRoleID(current.ID)
		if err != nil {
			return nil, mapErrorCtx(ctx, err)
		}
		perms, err := a.engine.ListRolePermissions(ctx.Context(), rid)
		if err != nil {
			return nil, mapErrorCtx(ctx, err)
		}
		out = append(out, perms...)
		if current.ParentID == "" {
			break
		}
		// ParentID holds warden's parent slug, not an id. Warden skips a
		// parent it cannot find, so a dangling slug confers nothing more.
		parent, err := a.engine.GetRoleBySlug(ctx.Context(), appID, current.ParentID)
		if errors.Is(err, rbac.ErrRoleNotFound) {
			break
		}
		if err != nil {
			return nil, mapErrorCtx(ctx, err)
		}
		current = parent
	}
	return out, nil
}

// storedRoleID turns a role id as the store hands it back, in warden's
// "role" form, into the authsome form the engine takes. The two share the
// suffix.
func storedRoleID(s string) (id.RoleID, error) {
	if rid, err := id.ParseRoleID(s); err == nil {
		return rid, nil
	}
	_, suffix, ok := strings.Cut(s, "_")
	if !ok {
		return id.Nil, fmt.Errorf("invalid role id %q", s)
	}
	return id.ParseRoleID(string(id.PrefixRole) + "_" + suffix)
}

// requireAssignable applies the grant ceilings: the caller must hold every
// permission the role confers, ownership may only be granted by an owner,
// and a caller may not change their own roles unless they are an owner.
func (a *API) requireAssignable(ctx forge.Context, role *rbac.Role, target id.UserID) error {
	callerID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return forge.Unauthorized("authentication required")
	}
	isOwner, err := a.callerHoldsOwnership(ctx, callerID)
	if err != nil {
		return err
	}
	if ownerSlugs[role.Slug] && !isOwner {
		return forge.Forbidden("only an owner may grant ownership")
	}
	if target.String() == callerID.String() && !isOwner {
		return forge.Forbidden("a caller may not change their own roles")
	}
	perms, err := a.effectivePermissions(ctx, role)
	if err != nil {
		return err
	}
	return a.requireCallerHolds(ctx, perms)
}

func (a *API) handleCreateRole(ctx forge.Context, req *CreateRoleRequest) (*rbac.Role, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.Slug == "" {
		return nil, forge.BadRequest("slug is required")
	}

	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	// A parent role, if given, must live in the caller's app too — otherwise a
	// new role could inherit another tenant's permissions.
	if req.ParentID != "" {
		if err := a.assertParentRoleInApp(ctx, req.ParentID); err != nil {
			return nil, err
		}
	}

	r := &rbac.Role{
		ID:          id.NewRoleID().String(),
		AppID:       appID.String(),
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := a.engine.CreateRole(ctx.Context(), r); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, r)
}

// assertParentRoleInApp verifies a parent role id refers to a role in the
// caller's app, preventing cross-tenant permission inheritance via re-parenting.
func (a *API) assertParentRoleInApp(ctx forge.Context, parentID string) error {
	pid, err := id.ParseRoleID(parentID)
	if err != nil {
		return forge.BadRequest(fmt.Sprintf("invalid parent_id: %v", err))
	}
	if _, err := a.roleInCallerApp(ctx, pid); err != nil {
		return err
	}
	return nil
}

func (a *API) handleListRoles(ctx forge.Context, req *ListRolesRequest) (*RoleListResponse, error) {
	appID, err := a.scopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	roles, err := a.engine.ListRoles(ctx.Context(), appID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	if roles == nil {
		roles = []*rbac.Role{}
	}
	resp := &RoleListResponse{Roles: roles}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleGetRole(ctx forge.Context, _ *GetRoleRequest) (*rbac.Role, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	r, err := a.roleInCallerApp(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (a *API) handleUpdateRole(ctx forge.Context, req *UpdateRoleRequest) (*rbac.Role, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	r, err := a.roleInCallerApp(ctx, roleID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		r.Name = *req.Name
	}
	if req.Description != nil {
		r.Description = *req.Description
	}
	// The store writes only name and description, so a new parent would be
	// dropped without a word. Refuse it instead. Echoing back the parent the
	// role already has (the slug a GET returns) is not a change.
	if req.ParentID != nil && *req.ParentID != r.ParentID {
		return nil, forge.BadRequest("parent_id cannot be changed here: re-parenting a role is not supported by this endpoint")
	}

	if err := a.engine.UpdateRole(ctx.Context(), r); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return r, nil
}

func (a *API) handleDeleteRole(ctx forge.Context, _ *DeleteRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if _, err = a.roleInCallerApp(ctx, roleID); err != nil {
		return nil, err
	}

	if err := a.engine.DeleteRole(ctx.Context(), roleID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAddPermission(ctx forge.Context, req *AddPermissionRequest) (*rbac.Permission, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if _, err = a.roleInCallerApp(ctx, roleID); err != nil {
		return nil, err
	}

	if req.Action == "" {
		return nil, forge.BadRequest("action is required")
	}
	if req.Resource == "" {
		return nil, forge.BadRequest("resource is required")
	}

	// A caller may only hand out what they hold. Without this, an admin
	// whose role grants permission:* could add settings:manage to their own
	// role and reach every tenant's configuration.
	if err := a.requireCallerHolds(ctx, []*rbac.Permission{{Action: req.Action, Resource: req.Resource}}); err != nil {
		return nil, err
	}

	perm := &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   roleID.String(),
		Action:   req.Action,
		Resource: req.Resource,
	}

	if err := a.engine.AddPermission(ctx.Context(), perm); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, perm)
}

func (a *API) handleListRolePermissions(ctx forge.Context, _ *ListRolePermissionsRequest) (*PermissionListResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if _, err = a.roleInCallerApp(ctx, roleID); err != nil {
		return nil, err
	}

	perms, err := a.engine.ListRolePermissions(ctx.Context(), roleID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	if perms == nil {
		perms = []*rbac.Permission{}
	}
	resp := &PermissionListResponse{Permissions: perms}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleRemovePermission(ctx forge.Context, _ *RemovePermissionRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}
	permID, err := id.ParsePermissionID(ctx.Param("permissionId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid permission id: %v", err))
	}

	if _, err = a.roleInCallerApp(ctx, roleID); err != nil {
		return nil, err
	}
	// The permission must actually belong to this role, so a caller cannot
	// detach a permission from a role they don't own by pairing it with one
	// they do.
	perms, err := a.engine.ListRolePermissions(ctx.Context(), roleID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	found := false
	for _, p := range perms {
		if p.ID == permID.String() {
			found = true
			break
		}
	}
	if !found {
		return nil, forge.NotFound("permission not found")
	}

	if err := a.engine.RemovePermission(ctx.Context(), permID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "removed"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAssignRole(ctx forge.Context, req *AssignRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	role, err := a.roleInCallerApp(ctx, roleID)
	if err != nil {
		return nil, err
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user_id: %v", err))
	}
	if _, err := a.userInCallerApp(ctx, userID); err != nil {
		return nil, err
	}
	if err := a.requireAssignable(ctx, role, userID); err != nil {
		return nil, err
	}

	ur := &rbac.UserRole{
		UserID: userID.String(),
		RoleID: roleID.String(),
		OrgID:  req.OrgID,
	}

	if err := a.engine.AssignUserRole(ctx.Context(), ur); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "assigned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleUnassignRole(ctx forge.Context, req *UnassignRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if _, err = a.roleInCallerApp(ctx, roleID); err != nil {
		return nil, err
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user_id: %v", err))
	}

	if err := a.engine.UnassignUserRole(ctx.Context(), userID, roleID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "unassigned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleListUserRoles(ctx forge.Context, req *ListUserRolesRequest) (*UserRoleListResponse, error) {
	userID, err := id.ParseUserID(ctx.Param("userId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user id: %v", err))
	}

	// Always scope to the caller's app so one tenant cannot enumerate a user's
	// role assignments in other tenants.
	appReq := ""
	if req != nil {
		appReq = req.AppID
	}
	appID, err := a.scopedAppID(ctx, appReq)
	if err != nil {
		return nil, err
	}
	roles, err := a.engine.ListUserRolesInApp(ctx.Context(), appID, userID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	if roles == nil {
		roles = []*rbac.Role{}
	}
	resp := &UserRoleListResponse{Roles: roles}
	return nil, ctx.JSON(http.StatusOK, resp)
}
