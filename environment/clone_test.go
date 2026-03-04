package environment

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Test mocks
// ──────────────────────────────────────────────────

type mockEnvStore struct {
	envs map[string]*Environment
}

func newMockEnvStore() *mockEnvStore {
	return &mockEnvStore{envs: make(map[string]*Environment)}
}

func (m *mockEnvStore) CreateEnvironment(_ context.Context, e *Environment) error {
	m.envs[e.ID.String()] = e
	return nil
}

func (m *mockEnvStore) GetEnvironment(_ context.Context, envID id.EnvironmentID) (*Environment, error) {
	e, ok := m.envs[envID.String()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return e, nil
}

func (m *mockEnvStore) GetEnvironmentBySlug(_ context.Context, _ id.AppID, _ string) (*Environment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockEnvStore) GetDefaultEnvironment(_ context.Context, _ id.AppID) (*Environment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockEnvStore) UpdateEnvironment(_ context.Context, _ *Environment) error {
	return nil
}

func (m *mockEnvStore) DeleteEnvironment(_ context.Context, _ id.EnvironmentID) error {
	return nil
}

func (m *mockEnvStore) ListEnvironments(_ context.Context, _ id.AppID) ([]*Environment, error) {
	return nil, nil
}

func (m *mockEnvStore) SetDefaultEnvironment(_ context.Context, _ id.AppID, _ id.EnvironmentID) error {
	return nil
}

type mockCloneSource struct {
	roles       []*RoleForClone
	permissions map[string][]*PermissionForClone
	webhooks    []*WebhookForClone
}

func (m *mockCloneSource) ListRolesForClone(_ context.Context, _ id.AppID, _ id.EnvironmentID) ([]*RoleForClone, error) {
	return m.roles, nil
}

func (m *mockCloneSource) ListPermissionsForClone(_ context.Context, roleID string) ([]*PermissionForClone, error) {
	return m.permissions[roleID], nil
}

func (m *mockCloneSource) ListWebhooksForClone(_ context.Context, _ id.AppID, _ id.EnvironmentID) ([]*WebhookForClone, error) {
	return m.webhooks, nil
}

type mockCloneTarget struct {
	createdRoles       []*RoleForClone
	createdPermissions []*PermissionForClone
	createdWebhooks    []*WebhookForClone
}

func (m *mockCloneTarget) CreateClonedRole(_ context.Context, r *RoleForClone) error {
	m.createdRoles = append(m.createdRoles, r)
	return nil
}

func (m *mockCloneTarget) CreateClonedPermission(_ context.Context, p *PermissionForClone) error {
	m.createdPermissions = append(m.createdPermissions, p)
	return nil
}

func (m *mockCloneTarget) CreateClonedWebhook(_ context.Context, w *WebhookForClone) error {
	m.createdWebhooks = append(m.createdWebhooks, w)
	return nil
}

// ──────────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────────

func testSourceEnv() *Environment {
	return &Environment{
		ID:    id.EnvironmentID(id.NewEnvironmentID()),
		AppID: id.NewAppID(),
		Name:  "Production",
		Slug:  "production",
		Type:  TypeProduction,
		Settings: &Settings{
			RateLimitEnabled: boolPtr(true),
		},
	}
}

// ──────────────────────────────────────────────────
// topologicalSortRoles tests
// ──────────────────────────────────────────────────

func TestTopologicalSortRoles_NoParents(t *testing.T) {
	roles := []*RoleForClone{
		{ID: "r1", Name: "Admin"},
		{ID: "r2", Name: "Editor"},
		{ID: "r3", Name: "Viewer"},
	}
	sorted := topologicalSortRoles(roles)
	assert.Len(t, sorted, 3)
}

func TestTopologicalSortRoles_ParentBeforeChild(t *testing.T) {
	roles := []*RoleForClone{
		{ID: "child", ParentID: "parent", Name: "Child"},
		{ID: "parent", Name: "Parent"},
	}
	sorted := topologicalSortRoles(roles)
	require.Len(t, sorted, 2)

	parentIdx, childIdx := -1, -1
	for i, r := range sorted {
		if r.ID == "parent" {
			parentIdx = i
		}
		if r.ID == "child" {
			childIdx = i
		}
	}
	assert.True(t, parentIdx < childIdx, "parent should come before child")
}

func TestTopologicalSortRoles_ThreeLevels(t *testing.T) {
	roles := []*RoleForClone{
		{ID: "grandchild", ParentID: "child", Name: "Grandchild"},
		{ID: "child", ParentID: "root", Name: "Child"},
		{ID: "root", Name: "Root"},
	}
	sorted := topologicalSortRoles(roles)
	require.Len(t, sorted, 3)

	rootIdx, childIdx, gcIdx := -1, -1, -1
	for i, r := range sorted {
		switch r.ID {
		case "root":
			rootIdx = i
		case "child":
			childIdx = i
		case "grandchild":
			gcIdx = i
		}
	}
	assert.True(t, rootIdx < childIdx, "root before child")
	assert.True(t, childIdx < gcIdx, "child before grandchild")
}

func TestTopologicalSortRoles_Empty(t *testing.T) {
	sorted := topologicalSortRoles(nil)
	assert.Empty(t, sorted)
}

// ──────────────────────────────────────────────────
// Clone operation tests
// ──────────────────────────────────────────────────

func TestClone_BasicEnvironment(t *testing.T) {
	srcEnv := testSourceEnv()
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID: srcEnv.ID,
		Name:        "Staging",
		Slug:        "staging",
		Type:        TypeStaging,
		Description: "Cloned from production",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "Staging", result.Environment.Name)
	assert.Equal(t, "staging", result.Environment.Slug)
	assert.Equal(t, TypeStaging, result.Environment.Type)
	assert.False(t, result.Environment.IsDefault)
	assert.Equal(t, TypeStaging.DefaultColor(), result.Environment.Color)
	assert.Equal(t, srcEnv.ID, result.Environment.ClonedFrom)
	assert.Equal(t, "Cloned from production", result.Environment.Description)
	assert.False(t, result.Environment.CreatedAt.IsZero())
}

func TestClone_WithRolesAndPermissions(t *testing.T) {
	srcEnv := testSourceEnv()
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{
		roles: []*RoleForClone{
			{ID: "old-admin", AppID: srcEnv.AppID.String(), EnvID: srcEnv.ID.String(), Name: "Admin", Slug: "admin"},
			{ID: "old-viewer", AppID: srcEnv.AppID.String(), EnvID: srcEnv.ID.String(), ParentID: "old-admin", Name: "Viewer", Slug: "viewer"},
		},
		permissions: map[string][]*PermissionForClone{
			"old-admin": {
				{ID: "old-perm1", RoleID: "old-admin", Action: "*", Resource: "*"},
			},
			"old-viewer": {
				{ID: "old-perm2", RoleID: "old-viewer", Action: "read", Resource: "document"},
			},
		},
	}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID: srcEnv.ID,
		Name:        "Dev",
		Slug:        "development",
		Type:        TypeDevelopment,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, result.RolesCloned)
	assert.Equal(t, 2, result.PermissionsCloned)
	assert.Len(t, result.RoleIDMap, 2)
	assert.Len(t, result.PermissionIDMap, 2)

	// Verify new IDs are different from old.
	for oldID, newID := range result.RoleIDMap {
		assert.NotEqual(t, oldID, newID)
	}

	// Verify parent remapping in target.
	require.Len(t, target.createdRoles, 2)
	newAdminID := result.RoleIDMap["old-admin"]
	for _, r := range target.createdRoles {
		if r.Name == "Viewer" {
			assert.Equal(t, newAdminID, r.ParentID, "viewer's parent should be remapped to new admin ID")
		}
	}

	// Verify permissions remapped to new role IDs.
	require.Len(t, target.createdPermissions, 2)
	for _, p := range target.createdPermissions {
		assert.NotEqual(t, "old-admin", p.RoleID)
		assert.NotEqual(t, "old-viewer", p.RoleID)
	}
}

func TestClone_WithWebhooks(t *testing.T) {
	srcEnv := testSourceEnv()
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{
		webhooks: []*WebhookForClone{
			{ID: "old-wh1", AppID: srcEnv.AppID.String(), EnvID: srcEnv.ID.String(), URL: "https://prod.example.com/hook", Events: []string{"user.created"}, Active: true},
			{ID: "old-wh2", AppID: srcEnv.AppID.String(), EnvID: srcEnv.ID.String(), URL: "https://prod.example.com/hook2", Events: []string{"user.deleted"}, Active: false},
		},
	}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID: srcEnv.ID,
		Name:        "Staging",
		Slug:        "staging",
		Type:        TypeStaging,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, result.WebhooksCloned)
	assert.Len(t, result.WebhookIDMap, 2)
	require.Len(t, target.createdWebhooks, 2)

	// URLs should be preserved.
	assert.Equal(t, "https://prod.example.com/hook", target.createdWebhooks[0].URL)
	assert.Equal(t, "https://prod.example.com/hook2", target.createdWebhooks[1].URL)
}

func TestClone_WithWebhookURLOverride(t *testing.T) {
	srcEnv := testSourceEnv()
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{
		webhooks: []*WebhookForClone{
			{ID: "old-wh", AppID: srcEnv.AppID.String(), EnvID: srcEnv.ID.String(), URL: "https://prod.example.com/hook", Events: []string{"user.created"}},
		},
	}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID:        srcEnv.ID,
		Name:               "Dev",
		Slug:               "development",
		Type:               TypeDevelopment,
		WebhookURLOverride: "https://staging.example.com/hook",
	})
	require.NoError(t, err)
	require.Len(t, target.createdWebhooks, 1)
	assert.Equal(t, "https://staging.example.com/hook", target.createdWebhooks[0].URL)
	assert.Equal(t, 1, result.WebhooksCloned)
}

func TestClone_WithSettingsOverride(t *testing.T) {
	srcEnv := testSourceEnv()
	srcEnv.Settings = &Settings{RateLimitEnabled: boolPtr(true)}
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID:     srcEnv.ID,
		Name:            "Dev",
		Slug:            "development",
		Type:            TypeDevelopment,
		SettingsOverride: &Settings{RateLimitEnabled: boolPtr(false)},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Environment.Settings)

	// The override should win: type defaults (dev: false) <- source (true) <- override (false).
	assert.False(t, *result.Environment.Settings.RateLimitEnabled)
}

func TestClone_ClonedFromLineage(t *testing.T) {
	srcEnv := testSourceEnv()
	store := newMockEnvStore()
	store.envs[srcEnv.ID.String()] = srcEnv

	source := &mockCloneSource{}
	target := &mockCloneTarget{}

	cloner := NewCloner(store, source, target)
	result, err := cloner.Clone(context.Background(), CloneRequest{
		SourceEnvID: srcEnv.ID,
		Name:        "Staging",
		Slug:        "staging",
		Type:        TypeStaging,
	})
	require.NoError(t, err)
	assert.Equal(t, srcEnv.ID, result.Environment.ClonedFrom)
}
