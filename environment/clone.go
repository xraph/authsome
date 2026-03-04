package environment

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
)

// CloneRequest configures an environment clone operation.
type CloneRequest struct {
	// SourceEnvID is the environment to clone from.
	SourceEnvID id.EnvironmentID

	// Name for the new environment (required).
	Name string

	// Slug for the new environment (required).
	Slug string

	// Type for the new environment (required).
	Type Type

	// Description for the new environment (optional).
	Description string

	// SettingsOverride merges on top of the source environment's settings (optional).
	SettingsOverride *Settings

	// WebhookURLOverride replaces webhook URLs in cloned webhooks (optional).
	WebhookURLOverride string
}

// CloneResult holds the output of a clone operation.
type CloneResult struct {
	// Environment is the newly created environment.
	Environment *Environment

	// RoleIDMap maps old role IDs to new role IDs.
	RoleIDMap map[string]string

	// PermissionIDMap maps old permission IDs to new permission IDs.
	PermissionIDMap map[string]string

	// WebhookIDMap maps old webhook IDs to new webhook IDs.
	WebhookIDMap map[string]string

	// RolesCloned is the number of roles cloned.
	RolesCloned int

	// PermissionsCloned is the number of permissions cloned.
	PermissionsCloned int

	// WebhooksCloned is the number of webhooks cloned.
	WebhooksCloned int
}

// RoleForClone is a minimal role representation used during cloning.
// This avoids coupling the environment package to rbac or warden types.
type RoleForClone struct {
	ID          string
	AppID       string
	EnvID       string
	ParentID    string
	Name        string
	Slug        string
	Description string
}

// PermissionForClone is a minimal permission representation used during cloning.
type PermissionForClone struct {
	ID       string
	RoleID   string
	Action   string
	Resource string
}

// WebhookForClone is a minimal webhook representation used during cloning.
type WebhookForClone struct {
	ID     string
	AppID  string
	EnvID  string
	URL    string
	Events []string
	Secret string
	Active bool
}

// CloneSource provides read access to the entities being cloned.
// This interface is implemented by the store or service layer.
type CloneSource interface {
	// ListRolesForClone returns all roles in the source environment.
	ListRolesForClone(ctx context.Context, appID id.AppID, envID id.EnvironmentID) ([]*RoleForClone, error)

	// ListPermissionsForClone returns all permissions for a role.
	ListPermissionsForClone(ctx context.Context, roleID string) ([]*PermissionForClone, error)

	// ListWebhooksForClone returns all webhooks in the source environment.
	ListWebhooksForClone(ctx context.Context, appID id.AppID, envID id.EnvironmentID) ([]*WebhookForClone, error)
}

// CloneTarget provides write access for creating cloned entities.
type CloneTarget interface {
	// CreateClonedRole creates a role in the target environment.
	CreateClonedRole(ctx context.Context, r *RoleForClone) error

	// CreateClonedPermission creates a permission in the target environment.
	CreateClonedPermission(ctx context.Context, p *PermissionForClone) error

	// CreateClonedWebhook creates a webhook in the target environment.
	CreateClonedWebhook(ctx context.Context, w *WebhookForClone) error
}

// Cloner executes environment clone operations.
type Cloner struct {
	envStore Store
	source   CloneSource
	target   CloneTarget
}

// NewCloner creates a new Cloner.
func NewCloner(envStore Store, source CloneSource, target CloneTarget) *Cloner {
	return &Cloner{
		envStore: envStore,
		source:   source,
		target:   target,
	}
}

// Clone creates a new environment by cloning config and structure from a source.
// It clones: roles (with hierarchy), permissions, and webhooks.
// It does NOT clone: users, sessions, organizations, API keys, or user data.
func (c *Cloner) Clone(ctx context.Context, req CloneRequest) (*CloneResult, error) {
	// 1. Load source environment.
	srcEnv, err := c.envStore.GetEnvironment(ctx, req.SourceEnvID)
	if err != nil {
		return nil, fmt.Errorf("environment: clone: load source: %w", err)
	}

	// 2. Build the new environment.
	now := time.Now()
	newEnv := &Environment{
		ID:          id.EnvironmentID(id.NewEnvironmentID()),
		AppID:       srcEnv.AppID,
		Name:        req.Name,
		Slug:        req.Slug,
		Type:        req.Type,
		IsDefault:   false, // cloned environments are never default
		Color:       req.Type.DefaultColor(),
		Description: req.Description,
		ClonedFrom:  srcEnv.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Merge settings: type defaults ← source settings ← request overrides.
	typeDefaults := DefaultSettingsForType(req.Type)
	merged := MergeSettings(typeDefaults, srcEnv.Settings)
	if req.SettingsOverride != nil {
		merged = MergeSettings(merged, req.SettingsOverride)
	}
	newEnv.Settings = merged

	// 3. Create the new environment.
	if err := c.envStore.CreateEnvironment(ctx, newEnv); err != nil {
		return nil, fmt.Errorf("environment: clone: create environment: %w", err)
	}

	result := &CloneResult{
		Environment:     newEnv,
		RoleIDMap:       make(map[string]string),
		PermissionIDMap: make(map[string]string),
		WebhookIDMap:    make(map[string]string),
	}

	// 4. Clone roles in topological order (parents first).
	roles, err := c.source.ListRolesForClone(ctx, srcEnv.AppID, srcEnv.ID)
	if err != nil {
		return nil, fmt.Errorf("environment: clone: list roles: %w", err)
	}

	ordered := topologicalSortRoles(roles)
	for _, role := range ordered {
		newRoleID := id.NewRoleID().String()
		result.RoleIDMap[role.ID] = newRoleID

		newRole := &RoleForClone{
			ID:          newRoleID,
			AppID:       role.AppID,
			EnvID:       newEnv.ID.String(),
			Name:        role.Name,
			Slug:        role.Slug,
			Description: role.Description,
		}

		// Remap parent ID if present.
		if role.ParentID != "" {
			if newParentID, ok := result.RoleIDMap[role.ParentID]; ok {
				newRole.ParentID = newParentID
			}
		}

		if err := c.target.CreateClonedRole(ctx, newRole); err != nil {
			return nil, fmt.Errorf("environment: clone: create role %q: %w", role.Name, err)
		}
		result.RolesCloned++
	}

	// 5. Clone permissions, remapping role IDs.
	for oldRoleID, newRoleID := range result.RoleIDMap {
		perms, err := c.source.ListPermissionsForClone(ctx, oldRoleID)
		if err != nil {
			return nil, fmt.Errorf("environment: clone: list permissions for role %s: %w", oldRoleID, err)
		}
		for _, perm := range perms {
			newPermID := id.NewPermissionID().String()
			result.PermissionIDMap[perm.ID] = newPermID

			newPerm := &PermissionForClone{
				ID:       newPermID,
				RoleID:   newRoleID,
				Action:   perm.Action,
				Resource: perm.Resource,
			}
			if err := c.target.CreateClonedPermission(ctx, newPerm); err != nil {
				return nil, fmt.Errorf("environment: clone: create permission: %w", err)
			}
			result.PermissionsCloned++
		}
	}

	// 6. Clone webhooks.
	webhooks, err := c.source.ListWebhooksForClone(ctx, srcEnv.AppID, srcEnv.ID)
	if err != nil {
		return nil, fmt.Errorf("environment: clone: list webhooks: %w", err)
	}
	for _, wh := range webhooks {
		newWhID := id.NewWebhookID().String()
		result.WebhookIDMap[wh.ID] = newWhID

		url := wh.URL
		if req.WebhookURLOverride != "" {
			url = req.WebhookURLOverride
		}

		newWh := &WebhookForClone{
			ID:     newWhID,
			AppID:  wh.AppID,
			EnvID:  newEnv.ID.String(),
			URL:    url,
			Events: wh.Events,
			Secret: wh.Secret,
			Active: wh.Active,
		}
		if err := c.target.CreateClonedWebhook(ctx, newWh); err != nil {
			return nil, fmt.Errorf("environment: clone: create webhook: %w", err)
		}
		result.WebhooksCloned++
	}

	return result, nil
}

// topologicalSortRoles orders roles so that parents come before children.
func topologicalSortRoles(roles []*RoleForClone) []*RoleForClone {
	byID := make(map[string]*RoleForClone, len(roles))
	for _, r := range roles {
		byID[r.ID] = r
	}

	visited := make(map[string]bool, len(roles))
	var sorted []*RoleForClone

	var visit func(r *RoleForClone)
	visit = func(r *RoleForClone) {
		if visited[r.ID] {
			return
		}
		// Visit parent first.
		if r.ParentID != "" {
			if parent, ok := byID[r.ParentID]; ok {
				visit(parent)
			}
		}
		visited[r.ID] = true
		sorted = append(sorted, r)
	}

	for _, r := range roles {
		visit(r)
	}
	return sorted
}
