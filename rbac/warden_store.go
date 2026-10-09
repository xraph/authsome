package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/forge"
	"github.com/xraph/warden"
	wardenassign "github.com/xraph/warden/assignment"
	wardenid "github.com/xraph/warden/id"
	wardenperm "github.com/xraph/warden/permission"
	wardenrole "github.com/xraph/warden/role"
)

// convertToWardenRoleID converts an authsome role ID string (prefix "arol") to a
// warden role ID (prefix "role"). If the string already has the warden prefix, it
// is parsed directly. This handles the prefix mismatch between authsome and warden.
func convertToWardenRoleID(s string) (wardenid.RoleID, error) {
	if wid, err := wardenid.ParseRoleID(s); err == nil {
		return wid, nil
	}
	// Extract suffix and reconstruct with warden role prefix.
	parts := strings.SplitN(s, "_", 2)
	if len(parts) != 2 {
		return wardenid.Nil, fmt.Errorf("rbac: invalid role id format %q", s)
	}
	return wardenid.ParseRoleID(string(wardenid.PrefixRole) + "_" + parts[1])
}

// convertToWardenPermissionID converts an authsome permission ID string (prefix "aprm")
// to a warden permission ID (prefix "perm").
func convertToWardenPermissionID(s string) (wardenid.PermissionID, error) {
	if wid, err := wardenid.ParsePermissionID(s); err == nil {
		return wid, nil
	}
	parts := strings.SplitN(s, "_", 2)
	if len(parts) != 2 {
		return wardenid.Nil, fmt.Errorf("rbac: invalid permission id format %q", s)
	}
	return wardenid.ParsePermissionID(string(wardenid.PrefixPermission) + "_" + parts[1])
}

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

func (s *WardenStore) GetRole(ctx context.Context, appID, roleID string) (*Role, error) {
	wid, err := convertToWardenRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, appID, wid)
	if err != nil {
		return nil, mapWardenError(err)
	}
	return FromWardenRole(wr), nil
}

func (s *WardenStore) GetRoleBySlug(ctx context.Context, appID, slug string) (*Role, error) {
	// Warden now requires an explicit namespace path. Authsome roles can live
	// in the root namespace (""), in the "app" namespace (the per-app
	// user/admin/owner roles seeded from app.warden) or in the "platform"
	// namespace (DSL-seeded platform roles). Try root first for backward
	// compat, then the two seeded namespaces, so DSL-created roles are always
	// resolvable without the caller needing to know the namespace.
	for _, ns := range []string{"", "app", "platform"} {
		wr, err := s.engine.Store().GetRoleBySlug(ctx, appID, ns, slug)
		if err == nil {
			return FromWardenRole(wr), nil
		}
	}
	return nil, ErrRoleNotFound
}

// UpdateRole writes the fields authsome edits, name and description, onto
// the stored warden role and keeps everything else. Warden's store replaces
// the whole row, and a Role has no room for the namespace, parent, default
// flag, member cap or metadata, so writing a converted Role would wipe them.
// Slug stays too, since child roles find their parent by it.
//
// A system role is refused, as warden's own write paths refuse it. On
// success r is refreshed from what was written.
func (s *WardenStore) UpdateRole(ctx context.Context, r *Role) error {
	wid, err := convertToWardenRoleID(r.ID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", r.ID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, r.AppID, wid)
	if err != nil {
		return mapWardenError(err)
	}
	// A role is changed only from the app that owns it.
	if wr.TenantID != r.AppID {
		return ErrRoleNotFound
	}
	if wr.IsSystem {
		return ErrSystemRoleImmutable
	}

	wr.Name = r.Name
	wr.Description = r.Description
	wr.UpdatedAt = time.Now().UTC()
	if err := s.engine.Store().UpdateRole(ctx, wr); err != nil {
		return mapWardenError(err)
	}
	*r = *FromWardenRole(wr)
	return nil
}

func (s *WardenStore) DeleteRole(ctx context.Context, appID, roleID string) error {
	wid, err := convertToWardenRoleID(roleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	if err := s.engine.Store().DeleteRole(ctx, appID, wid); err != nil {
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

func (s *WardenStore) AddPermission(ctx context.Context, appID string, p *Permission) error {
	// The role must belong to appID; its TenantID scopes the permission.
	roleID, err := convertToWardenRoleID(p.RoleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", p.RoleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, appID, roleID)
	if err != nil {
		return mapWardenError(err)
	}

	// Create the warden permission entity.
	wp := ToWardenPermission(p, wr.TenantID)
	if err := s.engine.Store().CreatePermission(ctx, wp); err != nil {
		// Permission may already exist (duplicate name+tenant). Look it up so we
		// can still attach it to this role — AttachPermission is idempotent.
		existing, findErr := s.engine.Store().GetPermissionByName(ctx, wp.TenantID, wp.NamespacePath, wp.Name)
		if findErr != nil || existing == nil {
			return mapWardenError(err) // Return the original CreatePermission error.
		}
		wp = existing
	}

	// Attach the permission to the role (idempotent — safe to call even if
	// the link already exists). Warden's role-permission junction now uses
	// natural keys (NamespacePath + Name) instead of permission IDs.
	ref := wardenperm.Ref{NamespacePath: wp.NamespacePath, Name: wp.Name}
	if err := s.engine.Store().AttachPermission(ctx, wr.TenantID, roleID, ref); err != nil {
		return mapWardenError(err)
	}

	// Copy generated ID back to the DTO.
	p.ID = wp.ID.String()
	return nil
}

func (s *WardenStore) RemovePermission(ctx context.Context, appID, permID string) error {
	wid, err := convertToWardenPermissionID(permID)
	if err != nil {
		return fmt.Errorf("rbac: invalid permission id %q: %w", permID, err)
	}
	// Deleting the permission entity in warden also detaches it from any roles.
	if err := s.engine.Store().DeletePermission(ctx, appID, wid); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) ListRolePermissions(ctx context.Context, appID, roleID string) ([]*Permission, error) {
	wRoleID, err := convertToWardenRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}

	// Warden's ListRolePermissions returns full Permission records now.
	wperms, err := s.engine.Store().ListRolePermissions(ctx, appID, wRoleID)
	if err != nil {
		return nil, mapWardenError(err)
	}

	perms := make([]*Permission, 0, len(wperms))
	for _, wp := range wperms {
		perms = append(perms, FromWardenPermission(wp, roleID))
	}
	return perms, nil
}

// ──────────────────────────────────────────────────
// Role assignment
// ──────────────────────────────────────────────────

func (s *WardenStore) AssignUserRole(ctx context.Context, appID string, ur *UserRole) error {
	// The role must belong to appID; its TenantID scopes the assignment.
	roleID, err := convertToWardenRoleID(ur.RoleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", ur.RoleID, err)
	}
	wr, err := s.engine.Store().GetRole(ctx, appID, roleID)
	if err != nil {
		return mapWardenError(err)
	}

	wa := ToWardenAssignment(ur, wr.TenantID)
	if err := s.engine.Store().CreateAssignment(ctx, wa); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) UnassignUserRole(ctx context.Context, appID, userID, roleID string) error {
	wRoleID, err := convertToWardenRoleID(roleID)
	if err != nil {
		return fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}

	// Find the matching assignment by filtering on subject + role.
	assignments, err := s.engine.Store().ListAssignments(ctx, &wardenassign.ListFilter{
		TenantID:    appID,
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
	if err := s.engine.Store().DeleteAssignment(ctx, appID, assignments[0].ID); err != nil {
		return mapWardenError(err)
	}
	return nil
}

func (s *WardenStore) ListUserRoles(ctx context.Context, userID string) ([]*Role, error) {
	// Try to resolve app scope from forge context.
	tenantID := resolveTenantFromContext(ctx)
	return s.listUserRolesWithTenant(ctx, tenantID, userID)
}

func (s *WardenStore) ListUserRolesForApp(ctx context.Context, appID, userID string) ([]*Role, error) {
	return s.listUserRolesWithTenant(ctx, appID, userID)
}

func (s *WardenStore) listUserRolesWithTenant(ctx context.Context, tenantID, userID string) ([]*Role, error) {
	// Pass nil namespacePaths → returns roles across every namespace in the
	// tenant, which matches the previous "all roles for this subject"
	// semantics.
	roleIDs, err := s.engine.Store().ListRolesForSubject(ctx, tenantID, nil, "user", userID)
	if err != nil {
		return nil, mapWardenError(err)
	}

	ids := make([]string, 0, len(roleIDs))
	for _, rid := range roleIDs {
		ids = append(ids, rid.String())
	}
	return s.GetRoles(ctx, tenantID, ids)
}

// GetRoles implements Store with one warden round trip. Ids that do not
// parse, no longer exist, or belong to another app are left out.
func (s *WardenStore) GetRoles(ctx context.Context, appID string, roleIDs []string) ([]*Role, error) {
	wids := make([]wardenid.RoleID, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if wid, err := convertToWardenRoleID(rid); err == nil {
			wids = append(wids, wid)
		}
	}
	if len(wids) == 0 {
		return []*Role{}, nil
	}
	wrs, err := s.engine.Store().GetRoles(ctx, appID, wids)
	if err != nil {
		return nil, mapWardenError(err)
	}
	// Warden leaves the order undefined; callers get the order they asked in.
	byID := make(map[wardenid.RoleID]*wardenrole.Role, len(wrs))
	for _, wr := range wrs {
		byID[wr.ID] = wr
	}
	roles := make([]*Role, 0, len(wrs))
	for _, wid := range wids {
		if wr, ok := byID[wid]; ok {
			roles = append(roles, FromWardenRole(wr))
			delete(byID, wid)
		}
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

func (s *WardenStore) GetRoleChildren(ctx context.Context, appID, roleID string) ([]*Role, error) {
	wid, err := convertToWardenRoleID(roleID)
	if err != nil {
		return nil, fmt.Errorf("rbac: invalid role id %q: %w", roleID, err)
	}
	// ListChildRoles is now keyed by (tenantID, parentSlug). Look the
	// parent role up first to translate the ID we hold into those keys.
	parent, err := s.engine.Store().GetRole(ctx, appID, wid)
	if err != nil {
		return nil, mapWardenError(err)
	}
	children, err := s.engine.Store().ListChildRoles(ctx, parent.TenantID, parent.Slug)
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

func (s *WardenStore) HasPermission(ctx context.Context, userID, action, resource string) (bool, error) {
	// TODO(service-accounts): when service account RBAC is added, this method needs to
	// accept a principal kind parameter and route service account permission checks
	// to a warden.SubjectServiceAccount (or similar) subject kind rather than
	// warden.SubjectUser. For now, service accounts use API-key-level scopes only.
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
	if errors.Is(err, warden.ErrSystemRoleImmutable) {
		return ErrSystemRoleImmutable
	}
	return err
}
