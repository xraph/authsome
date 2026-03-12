package dashboard

import (
	"context"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
)

// NewManifest builds a contributor.Manifest for the authsome dashboard.
// It starts with the base nav items, widgets, and settings, then merges
// any additional contributions from plugins implementing Plugin.
// The engine is used to create context-aware switcher components.
func NewManifest(engine *authsome.Engine, plugins []plugin.Plugin) *contributor.Manifest {
	m := &contributor.Manifest{
		Name:        "authsome",
		DisplayName: "Authsome",
		Icon:        "shield-check",
		Version:     "1.0.0",
		Layout:      "extension",
		ShowSidebar: boolPtr(true),
		TopbarConfig: &contributor.TopbarConfig{
			Title:       "Authsome",
			LogoIcon:    "shield-check",
			AccentColor: "#8b5cf6",
			ShowSearch:  true,
			Actions: []contributor.TopbarAction{
				{Label: "API Docs", Icon: "file-text", Href: "/docs", Variant: "ghost"},
			},
		},
		Nav:      baseNav(),
		Widgets:  baseWidgets(),
		Settings: baseSettings(),
		Capabilities: []string{
			"searchable",
		},
		SidebarHeaderContent: appSwitcherFromContext(engine),
		TopbarExtraContent:   envSwitcherFromContext(engine),
		AuthPages: []contributor.AuthPageDef{
			{
				Type:     "login",
				Path:     "/login",
				Title:    "Sign In",
				Icon:     "shield-check",
				Priority: 0,
			},
			{
				Type:     "register",
				Path:     "/register",
				Title:    "Sign Up",
				Icon:     "user-plus",
				Priority: 1,
			},
			{
				Type:     "logout",
				Path:     "/logout",
				Title:    "Sign Out",
				Icon:     "log-out",
				Priority: 2,
			},
		},
	}

	// Merge plugin-contributed nav items and widgets.
	for _, p := range plugins {
		// PageContributor provides nav items for pages with route params.
		if dpc, ok := p.(PageContributor); ok {
			m.Nav = append(m.Nav, dpc.DashboardNavItems()...)
		}

		dp, ok := p.(Plugin)
		if !ok {
			continue
		}

		for _, pp := range dp.DashboardPages() {
			m.Nav = append(m.Nav, contributor.NavItem{
				Label:    pp.Label,
				Path:     pp.Route,
				Icon:     pp.Icon,
				Group:    "Authsome",
				Priority: 10,
			})
		}

		for _, pw := range dp.DashboardWidgets(context.Background()) {
			m.Widgets = append(m.Widgets, contributor.WidgetDescriptor{
				ID:         pw.ID,
				Title:      pw.Title,
				Size:       pw.Size,
				RefreshSec: pw.RefreshSec,
				Group:      "Authsome",
			})
		}
	}

	return m
}

// baseNav returns the core navigation items for the authsome dashboard.
func baseNav() []contributor.NavItem {
	return []contributor.NavItem{
		// Authsome
		{Label: "Overview", Path: "/", Icon: "layout-dashboard", Group: "Authsome", Priority: 0},

		// User Management
		{Label: "Users", Path: "/users", Icon: "users", Group: "User Management", Priority: 0},
		{Label: "Sessions", Path: "/sessions", Icon: "key-round", Group: "User Management", Priority: 1},
		{Label: "Devices", Path: "/devices", Icon: "monitor-smartphone", Group: "User Management", Priority: 2},

		// Access Control
		{Label: "Roles", Path: "/roles", Icon: "shield", Group: "Access Control", Priority: 0},

		// Configuration
		{Label: "Applications", Path: "/apps", Icon: "building-2", Group: "Configuration", Priority: -1},
		{Label: "Settings", Path: "/settings", Icon: "settings", Group: "Configuration", Priority: 0},
		{Label: "Environments", Path: "/environments", Icon: "globe", Group: "Configuration", Priority: 1},
		{Label: "Signup Forms", Path: "/signup-forms", Icon: "file-edit", Group: "Configuration", Priority: 2},

		// Developer
		{Label: "Credentials", Path: "/credentials", Icon: "key", Group: "Developer", Priority: 0},
		{Label: "Webhooks", Path: "/webhooks", Icon: "webhook", Group: "Developer", Priority: 1},
		{Label: "Plugins", Path: "/plugins", Icon: "puzzle", Group: "Developer", Priority: 2},
	}
}

// baseWidgets returns the core widget descriptors for the authsome dashboard.
func baseWidgets() []contributor.WidgetDescriptor {
	return []contributor.WidgetDescriptor{
		{
			ID:          "authsome-stats",
			Title:       "Auth Stats",
			Description: "User counts",
			Size:        "md",
			RefreshSec:  60,
			Group:       "Authsome",
		},
		{
			ID:          "authsome-recent-signups",
			Title:       "Recent Signups",
			Description: "Latest user registrations",
			Size:        "md",
			RefreshSec:  30,
			Group:       "Authsome",
		},
		{
			ID:          "authsome-activity",
			Title:       "Auth Activity",
			Description: "Recent authentication events",
			Size:        "lg",
			RefreshSec:  15,
			Group:       "Authsome",
		},
	}
}

func boolPtr(b bool) *bool { return &b }

// baseSettings returns the core settings descriptors for the authsome dashboard.
func baseSettings() []contributor.SettingsDescriptor {
	return []contributor.SettingsDescriptor{
		{
			ID:          "authsome-config",
			Title:       "Authentication Settings",
			Description: "Configure authentication behavior",
			Group:       "Authsome",
			Icon:        "shield-check",
		},
	}
}
