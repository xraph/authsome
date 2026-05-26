// handlers_roles.go: Phase C.4 — Roles & RBAC dashboard.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type RoleSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type RoleDetail struct {
	RoleSummary
	AppID       string             `json:"appId,omitempty"`
	EnvID       string             `json:"envId,omitempty"`
	ParentID    string             `json:"parentId,omitempty"`
	Permissions []PermissionRecord `json:"permissions,omitempty"`
	UpdatedAt   string             `json:"updatedAt"`
}

type PermissionRecord struct {
	ID       string `json:"id"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type RoleListResponse struct {
	Roles []RoleSummary `json:"roles"`
}
type GetRoleInput struct {
	ID string `json:"id"`
}
type CreateRoleInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}
type UpdateRoleInput struct {
	ID          string  `json:"id"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}
type DeleteRoleInput struct {
	ID string `json:"id"`
}
type AssignRoleInput struct {
	UserID string `json:"userId"`
	RoleID string `json:"roleId"`
}
type UnassignRoleInput struct {
	UserID string `json:"userId"`
	RoleID string `json:"roleId"`
}

func rolesListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (RoleListResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (RoleListResponse, error) {
		if deps.Engine == nil {
			return RoleListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		list, err := deps.Engine.ListRoles(ctx, defaultAppID(deps.Engine))
		if err != nil {
			return RoleListResponse{}, mapEngineError(err)
		}
		out := RoleListResponse{Roles: make([]RoleSummary, 0, len(list))}
		for _, r := range list {
			out.Roles = append(out.Roles, projectRoleSummary(r))
		}
		return out, nil
	}
}

func rolesDetailHandler(deps Deps) func(ctx context.Context, in GetRoleInput, _ contract.Principal) (RoleDetail, error) {
	return func(ctx context.Context, in GetRoleInput, _ contract.Principal) (RoleDetail, error) {
		if deps.Engine == nil {
			return RoleDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		rid, err := parseRoleID(in.ID)
		if err != nil {
			return RoleDetail{}, err
		}
		r, err := deps.Engine.GetRole(ctx, rid)
		if err != nil {
			return RoleDetail{}, mapEngineError(err)
		}
		perms, _ := deps.Engine.ListRolePermissions(ctx, rid) //nolint:errcheck // partial detail is acceptable
		d := RoleDetail{
			RoleSummary: projectRoleSummary(r),
			AppID:       r.AppID,
			EnvID:       r.EnvID,
			ParentID:    r.ParentID,
			UpdatedAt:   r.UpdatedAt.UTC().Format(time.RFC3339),
		}
		for _, p := range perms {
			d.Permissions = append(d.Permissions, PermissionRecord{ID: p.ID, Action: p.Action, Resource: p.Resource})
		}
		return d, nil
	}
}

func rolesCreateHandler(deps Deps) func(ctx context.Context, in CreateRoleInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CreateRoleInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		name := strings.TrimSpace(in.Name)
		slug := strings.TrimSpace(in.Slug)
		if name == "" || slug == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "name and slug are required"}
		}
		r := &rbac.Role{
			Name: name, Slug: slug, Description: in.Description,
			AppID: defaultAppID(deps.Engine).String(),
		}
		if err := deps.Engine.CreateRole(ctx, r); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: r.ID}, nil
	}
}

func rolesUpdateHandler(deps Deps) func(ctx context.Context, in UpdateRoleInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateRoleInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		rid, err := parseRoleID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		current, err := deps.Engine.GetRole(ctx, rid)
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		if in.Name != nil {
			current.Name = *in.Name
		}
		if in.Description != nil {
			current.Description = *in.Description
		}
		if err := deps.Engine.UpdateRole(ctx, current); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: rid.String()}, nil
	}
}

func rolesDeleteHandler(deps Deps) func(ctx context.Context, in DeleteRoleInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteRoleInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		rid, err := parseRoleID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.DeleteRole(ctx, rid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: rid.String()}, nil
	}
}

func rolesAssignHandler(deps Deps) func(ctx context.Context, in AssignRoleInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in AssignRoleInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		uid, err := parseUserID(in.UserID)
		if err != nil {
			return AckResponse{}, err
		}
		rid, err := parseRoleID(in.RoleID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.AssignUserRole(ctx, &rbac.UserRole{UserID: uid.String(), RoleID: rid.String()}); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: rid.String()}, nil
	}
}

func rolesUnassignHandler(deps Deps) func(ctx context.Context, in UnassignRoleInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UnassignRoleInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		uid, err := parseUserID(in.UserID)
		if err != nil {
			return AckResponse{}, err
		}
		rid, err := parseRoleID(in.RoleID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.UnassignUserRole(ctx, uid, rid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: rid.String()}, nil
	}
}

func projectRoleSummary(r *rbac.Role) RoleSummary {
	if r == nil {
		return RoleSummary{}
	}
	return RoleSummary{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Description: r.Description,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func parseRoleID(s string) (id.RoleID, error) {
	if strings.TrimSpace(s) == "" {
		return id.RoleID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	rid, err := id.ParseRoleID(s)
	if err != nil {
		return id.RoleID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid role id: " + err.Error()}
	}
	return rid, nil
}
