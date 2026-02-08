package organization

import (
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/xraph/authsome/core/ui"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// MockOrganizationUIExtension is a mock implementation for testing
type MockOrganizationUIExtension struct {
	id      string
	widgets []ui.OrganizationWidget
	tabs    []ui.OrganizationTab
	actions []ui.OrganizationAction
	links   []ui.OrganizationQuickLink
}

func (m *MockOrganizationUIExtension) ExtensionID() string {
	return m.id
}

func (m *MockOrganizationUIExtension) OrganizationWidgets() []ui.OrganizationWidget {
	return m.widgets
}

func (m *MockOrganizationUIExtension) OrganizationTabs() []ui.OrganizationTab {
	return m.tabs
}

func (m *MockOrganizationUIExtension) OrganizationActions() []ui.OrganizationAction {
	return m.actions
}

func (m *MockOrganizationUIExtension) OrganizationQuickLinks() []ui.OrganizationQuickLink {
	return m.links
}

func (m *MockOrganizationUIExtension) OrganizationSettingsSections() []ui.OrganizationSettingsSection {
	return []ui.OrganizationSettingsSection{}
}

func TestNewOrganizationUIRegistry(t *testing.T) {
	registry := NewOrganizationUIRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.extensions)
	assert.Len(t, registry.extensions, 0)
}

func TestOrganizationUIRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		extension ui.OrganizationUIExtension
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful registration",
			extension: &MockOrganizationUIExtension{
				id: "test-ext",
			},
			wantErr: false,
		},
		{
			name:      "nil extension",
			extension: nil,
			wantErr:   true,
			errMsg:    "cannot register nil extension",
		},
		{
			name: "empty extension ID",
			extension: &MockOrganizationUIExtension{
				id: "",
			},
			wantErr: true,
			errMsg:  "extension ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewOrganizationUIRegistry()
			err := registry.Register(tt.extension)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrganizationUIRegistry_Register_Duplicate(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext1 := &MockOrganizationUIExtension{id: "test"}
	err := registry.Register(ext1)
	assert.NoError(t, err)

	// Try to register again with same ID
	ext2 := &MockOrganizationUIExtension{id: "test"}
	err = registry.Register(ext2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestOrganizationUIRegistry_GetWidgets(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	// Register extension with widgets
	ext := &MockOrganizationUIExtension{
		id: "test",
		widgets: []ui.OrganizationWidget{
			{
				ID:           "widget1",
				Title:        "Widget 1",
				Order:        10,
				RequireAdmin: false,
				Renderer:     func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
			{
				ID:           "widget2",
				Title:        "Widget 2",
				Order:        5,
				RequireAdmin: true,
				Renderer:     func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
		},
	}
	err := registry.Register(ext)
	assert.NoError(t, err)

	t.Run("admin user sees all widgets", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: true,
		}

		widgets := registry.GetWidgets(ctx)
		assert.Len(t, widgets, 2)
		// Should be sorted by order (5, 10)
		assert.Equal(t, "widget2", widgets[0].ID)
		assert.Equal(t, "widget1", widgets[1].ID)
	})

	t.Run("non-admin user sees only non-admin widgets", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: false,
		}

		widgets := registry.GetWidgets(ctx)
		assert.Len(t, widgets, 1)
		assert.Equal(t, "widget1", widgets[0].ID)
	})
}

func TestOrganizationUIRegistry_GetTabs(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext := &MockOrganizationUIExtension{
		id: "test",
		tabs: []ui.OrganizationTab{
			{
				ID:           "tab1",
				Label:        "Tab 1",
				Order:        20,
				RequireAdmin: false,
				Path:         "tab1",
				Renderer:     func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
			{
				ID:           "tab2",
				Label:        "Tab 2",
				Order:        10,
				RequireAdmin: true,
				Path:         "tab2",
				Renderer:     func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
		},
	}
	err := registry.Register(ext)
	assert.NoError(t, err)

	t.Run("admin user sees all tabs", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: true,
		}

		tabs := registry.GetTabs(ctx)
		assert.Len(t, tabs, 2)
		// Should be sorted by order (10, 20)
		assert.Equal(t, "tab2", tabs[0].ID)
		assert.Equal(t, "tab1", tabs[1].ID)
	})

	t.Run("non-admin user sees only non-admin tabs", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: false,
		}

		tabs := registry.GetTabs(ctx)
		assert.Len(t, tabs, 1)
		assert.Equal(t, "tab1", tabs[0].ID)
	})
}

func TestOrganizationUIRegistry_GetTabByPath(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext := &MockOrganizationUIExtension{
		id: "test",
		tabs: []ui.OrganizationTab{
			{
				ID:       "tab1",
				Label:    "Tab 1",
				Path:     "tab1",
				Renderer: func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
			{
				ID:           "tab2",
				Label:        "Tab 2",
				Path:         "tab2",
				RequireAdmin: true,
				Renderer:     func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
		},
	}
	err := registry.Register(ext)
	assert.NoError(t, err)

	ctx := ui.OrgExtensionContext{
		OrgID:   xid.New(),
		AppID:   xid.New(),
		IsAdmin: true,
	}

	t.Run("find existing tab", func(t *testing.T) {
		tab := registry.GetTabByPath(ctx, "tab1")
		assert.NotNil(t, tab)
		assert.Equal(t, "tab1", tab.ID)
	})

	t.Run("tab not found", func(t *testing.T) {
		tab := registry.GetTabByPath(ctx, "nonexistent")
		assert.Nil(t, tab)
	})

	t.Run("admin-only tab not visible to non-admin", func(t *testing.T) {
		nonAdminCtx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: false,
		}

		tab := registry.GetTabByPath(nonAdminCtx, "tab2")
		assert.Nil(t, tab)
	})
}

func TestOrganizationUIRegistry_GetActions(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext := &MockOrganizationUIExtension{
		id: "test",
		actions: []ui.OrganizationAction{
			{
				ID:           "action1",
				Label:        "Action 1",
				Order:        10,
				RequireAdmin: false,
			},
			{
				ID:           "action2",
				Label:        "Action 2",
				Order:        5,
				RequireAdmin: true,
			},
		},
	}
	err := registry.Register(ext)
	assert.NoError(t, err)

	t.Run("admin user sees all actions", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: true,
		}

		actions := registry.GetActions(ctx)
		assert.Len(t, actions, 2)
		// Should be sorted by order (5, 10)
		assert.Equal(t, "action2", actions[0].ID)
		assert.Equal(t, "action1", actions[1].ID)
	})

	t.Run("non-admin user sees only non-admin actions", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: false,
		}

		actions := registry.GetActions(ctx)
		assert.Len(t, actions, 1)
		assert.Equal(t, "action1", actions[0].ID)
	})
}

func TestOrganizationUIRegistry_GetQuickLinks(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext := &MockOrganizationUIExtension{
		id: "test",
		links: []ui.OrganizationQuickLink{
			{
				ID:           "link1",
				Title:        "Link 1",
				Order:        10,
				RequireAdmin: false,
				URLBuilder: func(basePath string, orgID, appID xid.ID) string {
					return "/link1"
				},
			},
			{
				ID:           "link2",
				Title:        "Link 2",
				Order:        5,
				RequireAdmin: true,
				URLBuilder: func(basePath string, orgID, appID xid.ID) string {
					return "/link2"
				},
			},
		},
	}
	err := registry.Register(ext)
	assert.NoError(t, err)

	t.Run("admin user sees all links", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: true,
		}

		links := registry.GetQuickLinks(ctx)
		assert.Len(t, links, 2)
		// Should be sorted by order (5, 10)
		assert.Equal(t, "link2", links[0].ID)
		assert.Equal(t, "link1", links[1].ID)
	})

	t.Run("non-admin user sees only non-admin links", func(t *testing.T) {
		ctx := ui.OrgExtensionContext{
			OrgID:   xid.New(),
			AppID:   xid.New(),
			IsAdmin: false,
		}

		links := registry.GetQuickLinks(ctx)
		assert.Len(t, links, 1)
		assert.Equal(t, "link1", links[0].ID)
	})
}

func TestOrganizationUIRegistry_ListExtensions(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext1 := &MockOrganizationUIExtension{id: "ext1"}
	ext2 := &MockOrganizationUIExtension{id: "ext2"}
	ext3 := &MockOrganizationUIExtension{id: "ext3"}

	registry.Register(ext1)
	registry.Register(ext2)
	registry.Register(ext3)

	ids := registry.ListExtensions()
	assert.Len(t, ids, 3)
	// Should be sorted alphabetically
	assert.Equal(t, []string{"ext1", "ext2", "ext3"}, ids)
}

func TestOrganizationUIRegistry_HasExtension(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	ext := &MockOrganizationUIExtension{id: "test"}
	registry.Register(ext)

	assert.True(t, registry.HasExtension("test"))
	assert.False(t, registry.HasExtension("nonexistent"))
}

func TestOrganizationUIRegistry_DuplicateIDsAcrossTypes(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	// Register two extensions with items that have duplicate IDs
	ext1 := &MockOrganizationUIExtension{
		id: "ext1",
		widgets: []ui.OrganizationWidget{
			{
				ID:       "duplicate",
				Title:    "Widget 1",
				Order:    10,
				Renderer: func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
		},
	}

	ext2 := &MockOrganizationUIExtension{
		id: "ext2",
		widgets: []ui.OrganizationWidget{
			{
				ID:       "duplicate",
				Title:    "Widget 2",
				Order:    5,
				Renderer: func(ctx ui.OrgExtensionContext) g.Node { return Div() },
			},
		},
	}

	registry.Register(ext1)
	registry.Register(ext2)

	ctx := ui.OrgExtensionContext{
		OrgID:   xid.New(),
		AppID:   xid.New(),
		IsAdmin: true,
	}

	// Should only return one widget (first registered)
	widgets := registry.GetWidgets(ctx)
	assert.Len(t, widgets, 1)
	assert.Equal(t, "Widget 1", widgets[0].Title)
}

func TestOrganizationUIRegistry_ThreadSafety(t *testing.T) {
	registry := NewOrganizationUIRegistry()

	// Test concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			ext := &MockOrganizationUIExtension{
				id: string(rune('a' + id)),
			}
			registry.Register(ext)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have all extensions registered
	ids := registry.ListExtensions()
	assert.Len(t, ids, 10)
}

func mockContext() ui.OrgExtensionContext {
	return ui.OrgExtensionContext{
		OrgID:    xid.New(),
		AppID:    xid.New(),
		BasePath: "/api",
		Request:  &http.Request{},
		GetOrg: func() (interface{}, error) {
			return nil, nil
		},
		IsAdmin: true,
	}
}

