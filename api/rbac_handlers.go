package api

import (
	"fmt"
	"net/http"

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

func (a *API) handleCreateRole(ctx forge.Context, req *CreateRoleRequest) (*rbac.Role, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.Slug == "" {
		return nil, forge.BadRequest("slug is required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
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
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusCreated, r)
}

func (a *API) handleListRoles(ctx forge.Context, req *ListRolesRequest) (*RoleListResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	roles, err := a.engine.ListRoles(ctx.Context(), appID)
	if err != nil {
		return nil, mapError(err)
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

	r, err := a.engine.GetRole(ctx.Context(), roleID)
	if err != nil {
		return nil, mapError(err)
	}

	return r, nil
}

func (a *API) handleUpdateRole(ctx forge.Context, req *UpdateRoleRequest) (*rbac.Role, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	r, err := a.engine.GetRole(ctx.Context(), roleID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.Name != nil {
		r.Name = *req.Name
	}
	if req.Description != nil {
		r.Description = *req.Description
	}
	if req.ParentID != nil {
		r.ParentID = *req.ParentID
	}

	if err := a.engine.UpdateRole(ctx.Context(), r); err != nil {
		return nil, mapError(err)
	}

	return r, nil
}

func (a *API) handleDeleteRole(ctx forge.Context, _ *DeleteRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if err := a.engine.DeleteRole(ctx.Context(), roleID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAddPermission(ctx forge.Context, req *AddPermissionRequest) (*rbac.Permission, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	if req.Action == "" {
		return nil, forge.BadRequest("action is required")
	}
	if req.Resource == "" {
		return nil, forge.BadRequest("resource is required")
	}

	perm := &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   roleID.String(),
		Action:   req.Action,
		Resource: req.Resource,
	}

	if err := a.engine.AddPermission(ctx.Context(), perm); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusCreated, perm)
}

func (a *API) handleListRolePermissions(ctx forge.Context, _ *ListRolePermissionsRequest) (*PermissionListResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	perms, err := a.engine.ListRolePermissions(ctx.Context(), roleID)
	if err != nil {
		return nil, mapError(err)
	}

	if perms == nil {
		perms = []*rbac.Permission{}
	}
	resp := &PermissionListResponse{Permissions: perms}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleRemovePermission(ctx forge.Context, _ *RemovePermissionRequest) (*StatusResponse, error) {
	permID, err := id.ParsePermissionID(ctx.Param("permissionId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid permission id: %v", err))
	}

	if err := a.engine.RemovePermission(ctx.Context(), permID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "removed"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleAssignRole(ctx forge.Context, req *AssignRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user_id: %v", err))
	}

	ur := &rbac.UserRole{
		UserID: userID.String(),
		RoleID: roleID.String(),
		OrgID:  req.OrgID,
	}

	if err := a.engine.AssignUserRole(ctx.Context(), ur); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "assigned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleUnassignRole(ctx forge.Context, req *UnassignRoleRequest) (*StatusResponse, error) {
	roleID, err := id.ParseRoleID(ctx.Param("roleId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid role id: %v", err))
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user_id: %v", err))
	}

	if err := a.engine.UnassignUserRole(ctx.Context(), userID, roleID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "unassigned"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleListUserRoles(ctx forge.Context, _ *ListUserRolesRequest) (*UserRoleListResponse, error) {
	userID, err := id.ParseUserID(ctx.Param("userId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user id: %v", err))
	}

	roles, err := a.engine.ListUserRoles(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	if roles == nil {
		roles = []*rbac.Role{}
	}
	resp := &UserRoleListResponse{Roles: roles}
	return nil, ctx.JSON(http.StatusOK, resp)
}
