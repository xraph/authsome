package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/xraph/forge"
	"github.com/xraph/warden"
	wardenassign "github.com/xraph/warden/assignment"
	wardenid "github.com/xraph/warden/id"
	wardenrole "github.com/xraph/warden/role"
)

// WardenStore implements rbac.Store by delegating to a Warden authorization engine.
// Role/permission CRUD is performed via Warden's store. HasPermission uses
// warden.Engine.Check() which evaluates the full RBAC+ReBAC+ABAC stack.
type WardenStore struct {
	engine *warden.Engine
}

// NewWardenStore creates a new WardenStore backed by the given Warden engine.
func NewWardenStore(eng *warden.Engine) *WardenStore {
	return &WardenStore{engine: eng}
}

// Compile-time interface check.
var _ Store = (*WardenStore)(nil)

// ──────────────────────────────────────────────────
// Roles
// ──────────────────────────────────────────────────

func (s *WardenStore) CreateRole(ctx context.Context, r *Role) error {
	wr := ToWardenRole(r)
	if err := s.engine.Store().CreateRole(ctx, wr); err != nil {
		return mapWardenError(err)
	}
	// Copy generated ID back to the DTO.
	r.ID = wr.ID.String()
	return nil
}

func (s *WardenStore) GetRole(ctx context.Context, roleID string) (*Role, error) {
	wid, err := wardenid.ParseRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, wid)
	if err != nil {
		return nil, mapWardenError(err)
	}
	return FromWardenRole(wr), nil
}

func (s *WardenStore) GetRoleBySlug(ctx context.Context, appID string, slug string) (*Role, error) {
	wr, err := s.engine.Store().GetRoleBySlug(ctx, appID, slug)
	if err != nil {
		return nil, mapWardenError(err)
	}
	return FromWardenRole(wr), nil
}

func (s *WardenStore) UpdateRole(ctx context.Context, r *Role) error {
	wr := ToWardenRole(r)
	if err := s.engine.Store().UpdateRole(ctx, wr); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) DeleteRole(ctx context.Context, roleID string) error {
	wid, err := wardenid.ParseRoleID(roleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	if err := s.engine.Store().DeleteRole(ctx, wid); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) ListRoles(ctx context.Context, appID string) ([]*Role, error) {
	wrs, err := s.engine.Store().ListRoles(ctx, &wardenrole.ListFilter{TenantID: appID})
	if err != nil {
		return nil, mapWardenError(err)
	}
	roles := make([]*Role, 0, len(wrs))
	for _, wr := range wrs {
		roles = append(roles, FromWardenRole(wr))
	}
	return roles, nil
}

// ──────────────────────────────────────────────────
// Permissions
// ──────────────────────────────────────────────────

func (s *WardenStore) AddPermission(ctx context.Context, p *Permission) error {
	// Resolve the role to find its TenantID (needed for the warden permission).
	roleID, err := wardenid.ParseRoleID(p.RoleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", p.RoleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, roleID)
	if err != nil {
		return mapWardenError(err)
	}

	// Create the warden permission entity.
	wp := ToWardenPermission(p, wr.TenantID)
	if err := s.engine.Store().CreatePermission(ctx, wp); err != nil {
		// Permission may already exist (duplicate name+tenant). Look it up so we
		// can still attach it to this role — AttachPermission is idempotent.
		existing, findErr := s.engine.Store().GetPermissionByName(ctx, wp.TenantID, wp.Name)
		if findErr != nil || existing == nil {
			return mapWardenError(err) // Return the original CreatePermission error.
		}
		wp = existing
	}

	// Attach the permission to the role (idempotent — safe to call even if
	// the link already exists).
	if err := s.engine.Store().AttachPermission(ctx, roleID, wp.ID); err != nil {
		return mapWardenError(err)
	}

	// Copy generated ID back to the DTO.
	p.ID = wp.ID.String()
	return nil
}

func (s *WardenStore) RemovePermission(ctx context.Context, permID string) error {
	wid, err := wardenid.ParsePermissionID(permID)
	if err != nil {
		return fmt.Errorf("rbac: invalid permission id %q: %w", permID, err)
	}
	// Deleting the permission entity in warden also detaches it from any roles.
	if err := s.engine.Store().DeletePermission(ctx, wid); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) ListRolePermissions(ctx context.Context, roleID string) ([]*Permission, error) {
	wRoleID, err := wardenid.ParseRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}

	// ListRolePermissions returns permission IDs, not full permission objects.
	permIDs, err := s.engine.Store().ListRolePermissions(ctx, wRoleID)
	if err != nil {
		return nil, mapWardenError(err)
	}

	perms := make([]*Permission, 0, len(permIDs))
	for _, pid := range permIDs {
		wp, err := s.engine.Store().GetPermission(ctx, pid)
		if err != nil {
			// Skip permissions that cannot be loaded (deleted concurrently, etc.).
			continue
		}
		perms = append(perms, FromWardenPermission(wp, roleID))
	}
	return perms, nil
}

// ──────────────────────────────────────────────────
// Role assignment
// ──────────────────────────────────────────────────

func (s *WardenStore) AssignUserRole(ctx context.Context, ur *UserRole) error {
	// Resolve the role to find its TenantID for the assignment.
	roleID, err := wardenid.ParseRoleID(ur.RoleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", ur.RoleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, roleID)
	if err != nil {
		return mapWardenError(err)
	}

	wa := ToWardenAssignment(ur, wr.TenantID)
	if err := s.engine.Store().CreateAssignment(ctx, wa); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) UnassignUserRole(ctx context.Context, userID string, roleID string) error {
	wRoleID, err := wardenid.ParseRoleID(roleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}

	// Find the matching assignment by filtering on subject + role.
	assignments, err := s.engine.Store().ListAssignments(ctx, &wardenassign.ListFilter{
		SubjectKind: "user",
		SubjectID:   userID,
		RoleID:      &wRoleID,
	})
	if err != nil {
		return mapWardenError(err)
	}
	if len(assignments) == 0 {
		return ErrRoleNotFound
	}

	// Delete the first matching assignment.
	if err := s.engine.Store().DeleteAssignment(ctx, assignments[0].ID); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) ListUserRoles(ctx context.Context, userID string) ([]*Role, error) {
	// Try to resolve app scope from forge context.
	tenantID := resolveTenantFromContext(ctx)
	return s.listUserRolesWithTenant(ctx, tenantID, userID)
}

func (s *WardenStore) ListUserRolesForApp(ctx context.Context, appID string, userID string) ([]*Role, error) {
	return s.listUserRolesWithTenant(ctx, appID, userID)
}

func (s *WardenStore) listUserRolesWithTenant(ctx context.Context, tenantID string, userID string) ([]*Role, error) {
	roleIDs, err := s.engine.Store().ListRolesForSubject(ctx, tenantID, "user", userID)
	if err != nil {
		return nil, mapWardenError(err)
	}

	roles := make([]*Role, 0, len(roleIDs))
	for _, rid := range roleIDs {
		wr, err := s.engine.Store().GetRole(ctx, rid)
		if err != nil {
			// Skip roles that cannot be loaded (deleted concurrently, etc.).
			continue
		}
		roles = append(roles, FromWardenRole(wr))
	}
	return roles, nil
}

// resolveTenantFromContext extracts the tenant scope from forge context.
func resolveTenantFromContext(ctx context.Context) string {
	scope, ok := forge.ScopeFrom(ctx)
	if ok {
		if appID := scope.AppID(); appID != "" {
			return appID
		}
	}
	return ""
}

// ──────────────────────────────────────────────────
// Hierarchy
// ──────────────────────────────────────────────────

func (s *WardenStore) GetRoleChildren(ctx context.Context, roleID string) ([]*Role, error) {
	wid, err := wardenid.ParseRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	children, err := s.engine.Store().ListChildRoles(ctx, wid)
	if err != nil {
		return nil, mapWardenError(err)
	}
	result := make([]*Role, 0, len(children))
	for _, wr := range children {
		result = append(result, FromWardenRole(wr))
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Permission check
// ──────────────────────────────────────────────────

func (s *WardenStore) HasPermission(ctx context.Context, userID string, action, resource string) (bool, error) {
	result, err := s.engine.Check(ctx, &warden.CheckRequest{
		Subject:  warden.Subject{Kind: warden.SubjectUser, ID: userID},
		Action:   warden.Action{Name: action},
		Resource: warden.Resource{Type: resource},
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// ──────────────────────────────────────────────────
// Error mapping
// ──────────────────────────────────────────────────

// mapWardenError translates warden-level errors to rbac-level errors.
func mapWardenError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, warden.ErrRoleNotFound) {
		return ErrRoleNotFound
	}
	if errors.Is(err, warden.ErrPermissionNotFound) {
		return ErrPermissionNotFound
	}
	if errors.Is(err, warden.ErrDuplicateAssignment) {
		return ErrRoleAlreadyAssigned
	}
	return err
}
