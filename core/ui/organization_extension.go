package ui

import (
	"net/http"

	"github.com/rs/xid"
	g "maragu.dev/gomponents"
)

// OrganizationUIExtension is the interface that plugins implement to extend organization pages
// This allows plugins to add widgets, tabs, actions, and quick links to organization-scoped pages
type OrganizationUIExtension interface {
	// ExtensionID returns a unique identifier for this extension
	ExtensionID() string

	// OrganizationWidgets returns widget cards to display on the organization detail page
	// These appear as stats cards alongside the default org stats
	OrganizationWidgets() []OrganizationWidget

	// OrganizationTabs returns full-page tabs for organization content
	// Each tab gets its own route and can display complete pages
	OrganizationTabs() []OrganizationTab

	// OrganizationActions returns action buttons for the organization header
	// These appear as buttons in the organization detail page header
	OrganizationActions() []OrganizationAction

	// OrganizationQuickLinks returns quick access cards
	// These appear alongside default quick links (Members, Teams, Roles, Invitations)
	OrganizationQuickLinks() []OrganizationQuickLink

	// OrganizationSettingsSections returns settings sections for org settings pages
	// These can be used to add custom settings to organization configuration
	OrganizationSettingsSections() []OrganizationSettingsSection
}

// OrganizationWidget represents a stats card/widget on the organization detail page
type OrganizationWidget struct {
	// ID is a unique identifier for this widget
	ID string
	// Title is the widget title
	Title string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// Size determines the widget size in grid columns (1-3)
	// 1 = takes 1/3 of row, 2 = takes 2/3 of row, 3 = full width
	Size int
	// RequireAdmin indicates if admin privileges are required to view this widget
	RequireAdmin bool
	// Renderer renders the widget content
	// The function receives context with org ID and can fetch its own data
	Renderer func(ctx OrgExtensionContext) g.Node
}

// OrganizationTab represents a full-page tab for organization content
type OrganizationTab struct {
	// ID is a unique identifier for this tab
	ID string
	// Label is the display text in the tab navigation
	Label string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Order determines display order in tab bar (lower = first)
	Order int
	// RequireAdmin indicates if admin privileges are required to view this tab
	RequireAdmin bool
	// Path is the URL path segment for this tab (e.g., "scim", "billing")
	// Will be accessible at: /dashboard/app/:appId/organizations/:orgId/tabs/:path
	Path string
	// Renderer renders the full tab content
	// The function receives context with org ID and can fetch its own data
	Renderer func(ctx OrgExtensionContext) g.Node
}

// OrganizationAction represents an action button in the organization header
type OrganizationAction struct {
	// ID is a unique identifier for this action
	ID string
	// Label is the button text
	Label string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// Style determines button styling: "primary", "secondary", "danger"
	Style string
	// RequireAdmin indicates if admin privileges are required to see this action
	RequireAdmin bool
	// Action is the onclick handler or htmx attribute
	// Examples:
	//   - "htmx.ajax('POST', '/api/scim/sync', {target: '#status'})"
	//   - "document.getElementById('modal').showModal()"
	Action string
}

// OrganizationQuickLink represents a quick access card on the organization detail page
type OrganizationQuickLink struct {
	// ID is a unique identifier for this link
	ID string
	// Title is the card title
	Title string
	// Description is the card description
	Description string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// URLBuilder builds the URL for this link
	// Receives basePath, orgID, and appID to construct the full URL
	URLBuilder func(basePath string, orgID xid.ID, appID xid.ID) string
	// RequireAdmin indicates if admin privileges are required to see this link
	RequireAdmin bool
}

// OrganizationSettingsSection represents a settings section for organization configuration
type OrganizationSettingsSection struct {
	// ID is a unique identifier for this section
	ID string
	// Title is the section title
	Title string
	// Description is the section description
	Description string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// RequireAdmin indicates if admin privileges are required to see this section
	RequireAdmin bool
	// Renderer renders the section content
	Renderer func(ctx OrgExtensionContext) g.Node
}

// OrgExtensionContext provides context information to extension renderers
// Extensions receive minimal information and fetch their own data as needed
type OrgExtensionContext struct {
	// OrgID is the organization identifier
	OrgID xid.ID
	// AppID is the application identifier
	AppID xid.ID
	// BasePath is the dashboard base path
	BasePath string
	// Request is the HTTP request for accessing additional context
	Request *http.Request
	// GetOrg is a helper function to lazy-load the organization object if needed
	// Most extensions should fetch their own org-scoped data instead
	GetOrg func() (interface{}, error)
	// IsAdmin indicates if the current user has admin privileges in this org
	IsAdmin bool
}
