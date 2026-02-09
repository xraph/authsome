package ui

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// RoleFormData contains data for rendering the role form.
type RoleFormData struct {
	Role            *schema.Role
	Permissions     []*schema.Permission
	SelectedPermIDs map[xid.ID]bool
	IsTemplate      bool
	CanSetOwnerRole bool
	Errors          map[string]string
	ActionURL       string
	CancelURL       string
}

// RoleForm renders a form for creating/editing a role.
func RoleForm(data RoleFormData) g.Node {
	roleID := ""
	roleName := ""
	roleDescription := ""
	isOwnerRole := false

	if data.Role != nil {
		roleID = data.Role.ID.String()
		roleName = data.Role.Name
		roleDescription = data.Role.Description
		isOwnerRole = data.Role.IsOwnerRole
	}

	return Div(
		Class("role-form space-y-6"),

		// Hidden field for role ID (if editing)
		g.If(roleID != "", Input(Type("hidden"), Name("roleID"), Value(roleID))),

		// Name field
		Div(
			Class("form-group"),
			Label(
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
				g.Attr("for", "roleName"),
				g.Text("Role Name"),
				Span(Class("text-red-500"), g.Text(" *")),
			),
			Input(
				Type("text"),
				ID("roleName"),
				Name("name"),
				Value(roleName),
				Required(),
				Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm"),
				Placeholder("e.g., Administrator, Editor, Viewer"),
			),
			g.If(data.Errors["name"] != "",
				P(Class("mt-1 text-sm text-red-600"), g.Text(data.Errors["name"])),
			),
		),

		// Description field
		Div(
			Class("form-group"),
			Label(
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
				g.Attr("for", "roleDescription"),
				g.Text("Description"),
			),
			Textarea(
				ID("roleDescription"),
				Name("description"),
				Rows("3"),
				Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm"),
				Placeholder("Describe the purpose and responsibilities of this role"),
				g.Text(roleDescription),
			),
			g.If(data.Errors["description"] != "",
				P(Class("mt-1 text-sm text-red-600"), g.Text(data.Errors["description"])),
			),
		),

		// Owner role checkbox (only for templates)
		g.If(data.CanSetOwnerRole,
			Div(
				Class("form-group"),
				Div(
					Class("flex items-center"),
					Input(
						Type("checkbox"),
						ID("isOwnerRole"),
						Name("isOwnerRole"),
						Value("true"),
						g.If(isOwnerRole, Checked()),
						Class("h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
					),
					Label(
						Class("ml-2 block text-sm text-gray-700 dark:text-gray-300"),
						g.Attr("for", "isOwnerRole"),
						g.Text("Default Owner Role"),
					),
				),
				P(
					Class("mt-1 text-xs text-gray-500 dark:text-gray-400"),
					g.Text("This role will be automatically assigned to organization creators"),
				),
			),
		),

		// Permissions selector
		Div(
			Class("form-group"),
			Label(
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3"),
				g.Text("Permissions"),
				Span(Class("text-red-500"), g.Text(" *")),
			),
			PermissionSelector(PermissionSelectorData{
				Permissions:     data.Permissions,
				SelectedPermIDs: data.SelectedPermIDs,
			}),
			g.If(data.Errors["permissions"] != "",
				P(Class("mt-2 text-sm text-red-600"), g.Text(data.Errors["permissions"])),
			),
		),

		// Action buttons
		Div(
			Class("flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700"),
			A(
				Href(data.CancelURL),
				Class("px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"),
				g.Text("Cancel"),
			),
			Button(
				Type("submit"),
				Class("px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"),
				g.Text("Save Role"),
			),
		),
	)
}

// PermissionSelectorData contains data for the permission selector.
type PermissionSelectorData struct {
	Permissions     []*schema.Permission
	SelectedPermIDs map[xid.ID]bool
}

// PermissionSelector renders a permission multi-select component.
func PermissionSelector(data PermissionSelectorData) g.Node {
	// Group permissions by category
	categories := make(map[string][]*schema.Permission)

	for _, perm := range data.Permissions {
		category := perm.Category
		if category == "" {
			category = "General"
		}

		categories[category] = append(categories[category], perm)
	}

	hasPermissions := len(data.Permissions) > 0

	return Div(
		Class("permission-selector space-y-4"),

		// Header with search and add button
		Div(
			Class("flex items-center gap-3"),
			// Search box
			Div(
				Class("relative flex-1"),
				Input(
					Type("text"),
					ID("permissionSearch"),
					Placeholder("Search permissions..."),
					Class("block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white dark:bg-gray-700 dark:border-gray-600 placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"),
					g.Attr("onkeyup", "filterPermissions(this.value)"),
				),
				Div(
					Class("absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"),
					lucide.Search(Class("h-5 w-5 text-gray-400")),
				),
			),
			// Add permission button
			Button(
				Type("button"),
				Class("inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"),
				g.Attr("onclick", "document.getElementById('addPermissionModal').classList.remove('hidden')"),
				lucide.Plus(Class("h-4 w-4 mr-1")),
				g.Text("Add"),
			),
		),

		// Empty state or permission categories
		g.If(!hasPermissions,
			Div(
				Class("border border-dashed border-gray-300 dark:border-gray-600 rounded-md p-8 text-center"),
				lucide.Shield(Class("mx-auto h-12 w-12 text-gray-400")),
				H3(Class("mt-2 text-sm font-medium text-gray-900 dark:text-gray-100"), g.Text("No permissions available")),
				P(Class("mt-1 text-sm text-gray-500 dark:text-gray-400"), g.Text("Create custom permissions to assign to this role")),
				Button(
					Type("button"),
					Class("mt-4 inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"),
					g.Attr("onclick", "document.getElementById('addPermissionModal').classList.remove('hidden')"),
					lucide.Plus(Class("h-4 w-4 mr-2")),
					g.Text("Create First Permission"),
				),
			),
		),

		g.If(hasPermissions,
			Div(
				Class("permission-categories max-h-96 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-md"),
				g.Group(renderPermissionCategories(categories, data.SelectedPermIDs)),
			),
		),

		// Selected count
		Div(
			Class("text-sm text-gray-600 dark:text-gray-400"),
			g.Textf("%d permissions selected", len(data.SelectedPermIDs)),
		),

		// Add permission modal
		renderAddPermissionModal(),
	)
}

// renderPermissionCategories renders grouped permission checkboxes.
func renderPermissionCategories(categories map[string][]*schema.Permission, selectedIDs map[xid.ID]bool) []g.Node {
	nodes := make([]g.Node, 0, len(categories))

	// Sort categories for consistent display
	categoryNames := []string{"users", "organizations", "settings", "dashboard", "roles", "permissions", "sessions", "apikeys", "audit_logs", "General"}

	for _, catName := range categoryNames {
		perms, exists := categories[catName]
		if !exists || len(perms) == 0 {
			continue
		}

		nodes = append(nodes, renderPermissionCategory(catName, perms, selectedIDs))
	}

	return nodes
}

// renderPermissionCategory renders a single category section.
func renderPermissionCategory(category string, perms []*schema.Permission, selectedIDs map[xid.ID]bool) g.Node {
	return Details(
		Class("permission-category border-b border-gray-200 dark:border-gray-700 last:border-b-0"),
		g.Attr("open", ""), // Open by default

		Summary(
			Class("px-4 py-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 flex items-center justify-between"),
			Div(
				Class("flex items-center space-x-2"),
				lucide.FolderOpen(Class("h-4 w-4 text-gray-500")),
				Span(Class("text-sm font-medium text-gray-900 dark:text-gray-100"), g.Text(category)),
				Span(
					Class("text-xs text-gray-500 dark:text-gray-400 ml-2"),
					g.Textf("(%d)", len(perms)),
				),
			),
		),

		Div(
			Class("px-4 py-2 space-y-2 bg-gray-50 dark:bg-gray-900"),
			g.Group(renderPermissionCheckboxes(perms, selectedIDs)),
		),
	)
}

// renderPermissionCheckboxes renders checkboxes for permissions.
func renderPermissionCheckboxes(perms []*schema.Permission, selectedIDs map[xid.ID]bool) []g.Node {
	nodes := make([]g.Node, 0, len(perms))

	for _, perm := range perms {
		isSelected := selectedIDs[perm.ID]

		nodes = append(nodes, Div(
			Class("flex items-start"),
			Div(
				Class("flex items-center h-5"),
				Input(
					Type("checkbox"),
					Name("permissionIDs[]"),
					Value(perm.ID.String()),
					ID("perm-"+perm.ID.String()),
					g.If(isSelected, Checked()),
					Class("h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
				),
			),
			Div(
				Class("ml-3 text-sm"),
				Label(
					g.Attr("for", "perm-"+perm.ID.String()),
					Class("font-medium text-gray-700 dark:text-gray-300 cursor-pointer"),
					g.Text(perm.Name),
				),
				g.If(perm.Description != "",
					P(Class("text-gray-500 dark:text-gray-400 text-xs mt-0.5"), g.Text(perm.Description)),
				),
				g.If(perm.IsCustom,
					Span(
						Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200 ml-2"),
						g.Text("Custom"),
					),
				),
			),
		))
	}

	return nodes
}

// RoleListTableData contains data for the role list table.
type RoleListTableData struct {
	Roles       []*schema.Role
	IsTemplate  bool
	BasePath    string
	OnEdit      func(roleID xid.ID) string
	OnDelete    func(roleID xid.ID) string
	OnClone     func(roleID xid.ID) string
	ShowActions bool
}

// RoleListTable renders a table of roles with actions.
func RoleListTable(data RoleListTableData) g.Node {
	return Div(
		Class("role-list-table overflow-hidden shadow ring-1 ring-black ring-opacity-5 rounded-lg"),
		Table(
			Class("min-w-full divide-y divide-gray-300 dark:divide-gray-700"),

			// Header
			THead(
				Class("bg-gray-50 dark:bg-gray-800"),
				Tr(
					Th(
						Scope("col"),
						Class("py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-gray-100 sm:pl-6"),
						g.Text("Role"),
					),
					Th(
						Scope("col"),
						Class("px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-gray-100"),
						g.Text("Description"),
					),
					Th(
						Scope("col"),
						Class("px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-gray-100"),
						g.Text("Status"),
					),
					g.If(data.ShowActions,
						Th(
							Scope("col"),
							Class("relative py-3.5 pl-3 pr-4 sm:pr-6"),
							Span(Class("sr-only"), g.Text("Actions")),
						),
					),
				),
			),

			// Body
			TBody(
				Class("divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-900"),
				g.Group(renderRoleRows(data)),
			),
		),
	)
}

// renderRoleRows renders table rows for roles.
func renderRoleRows(data RoleListTableData) []g.Node {
	if len(data.Roles) == 0 {
		return []g.Node{
			Tr(
				Td(
					g.Attr("colspan", "4"),
					Class("px-6 py-8 text-center text-sm text-gray-500 dark:text-gray-400"),
					Div(
						lucide.Inbox(Class("mx-auto h-12 w-12 text-gray-400")),
						P(Class("mt-2"), g.Text("No roles found")),
					),
				),
			),
		}
	}

	nodes := make([]g.Node, 0, len(data.Roles))

	for _, role := range data.Roles {
		nodes = append(nodes, renderRoleRow(role, data))
	}

	return nodes
}

// renderRoleRow renders a single role row.
func renderRoleRow(role *schema.Role, data RoleListTableData) g.Node {
	return Tr(
		Class("hover:bg-gray-50 dark:hover:bg-gray-800"),

		// Name column
		Td(
			Class("whitespace-nowrap py-4 pl-4 pr-3 text-sm sm:pl-6"),
			Div(
				Class("flex items-center space-x-2"),
				Div(
					Class("flex-shrink-0"),
					Div(
						Class("h-10 w-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center"),
						lucide.Shield(Class("h-5 w-5 text-blue-600 dark:text-blue-400")),
					),
				),
				Div(
					Div(
						Class("font-medium text-gray-900 dark:text-gray-100"),
						g.Text(role.Name),
					),
					g.If(role.IsOwnerRole,
						Span(
							Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"),
							lucide.Crown(Class("h-3 w-3 mr-1")),
							g.Text("Owner"),
						),
					),
				),
			),
		),

		// Description column
		Td(
			Class("px-3 py-4 text-sm text-gray-500 dark:text-gray-400"),
			g.Text(role.Description),
		),

		// Status column
		Td(
			Class("whitespace-nowrap px-3 py-4 text-sm"),
			g.If(role.IsTemplate,
				Span(
					Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"),
					g.Text("Template"),
				),
			),
			g.If(!role.IsTemplate && role.OrganizationID != nil,
				Span(
					Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"),
					g.Text("Org-Specific"),
				),
			),
		),

		// Actions column
		g.If(data.ShowActions,
			Td(
				Class("relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"),
				Div(
					Class("flex items-center justify-end space-x-2"),

					// Edit button
					g.If(data.OnEdit != nil,
						A(
							Href(data.OnEdit(role.ID)),
							Class("text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"),
							lucide.Pencil(Class("h-4 w-4")),
						),
					),

					// Clone button (for templates)
					g.If(role.IsTemplate && data.OnClone != nil,
						A(
							Href(data.OnClone(role.ID)),
							Class("text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-300"),
							Title("Clone template"),
							lucide.Copy(Class("h-4 w-4")),
						),
					),

					// Delete button (not for templates or owner roles)
					g.If(!role.IsTemplate && !role.IsOwnerRole && data.OnDelete != nil,
						A(
							Href(data.OnDelete(role.ID)),
							Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
							g.Attr("onclick", fmt.Sprintf("return confirm('Are you sure you want to delete the role %s?')", role.Name)),
							lucide.Trash(Class("h-4 w-4")),
						),
					),
				),
			),
		),
	)
}

// RoleManagementInterfaceData contains data for the full role management interface.
type RoleManagementInterfaceData struct {
	Title         string
	Description   string
	Roles         []*schema.Role
	IsTemplate    bool
	BasePath      string
	CreateRoleURL string
	AppID         xid.ID
	OrgID         *xid.ID
	ShowActions   bool
}

// RoleManagementInterface renders the complete role management UI.
func RoleManagementInterface(data RoleManagementInterfaceData) g.Node {
	return Div(
		Class("role-management-interface space-y-6"),

		// Header
		Div(
			Class("md:flex md:items-center md:justify-between"),
			Div(
				Class("flex-1 min-w-0"),
				H2(
					Class("text-2xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:text-3xl sm:truncate"),
					g.Text(data.Title),
				),
				g.If(data.Description != "",
					P(Class("mt-1 text-sm text-gray-500 dark:text-gray-400"), g.Text(data.Description)),
				),
			),
			Div(
				Class("mt-4 flex md:mt-0 md:ml-4"),
				A(
					Href(data.CreateRoleURL),
					Class("inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"),
					lucide.Plus(Class("h-4 w-4 mr-2")),
					g.Text("Create Role"),
				),
			),
		),

		// Role list
		RoleListTable(RoleListTableData{
			Roles:      data.Roles,
			IsTemplate: data.IsTemplate,
			BasePath:   data.BasePath,
			OnEdit: func(roleID xid.ID) string {
				if data.OrgID != nil {
					return fmt.Sprintf("%s/organizations/%s/roles/%s/edit", data.BasePath, data.OrgID.String(), roleID.String())
				}

				return fmt.Sprintf("%s/settings/roles/%s/edit", data.BasePath, roleID.String())
			},
			OnDelete: func(roleID xid.ID) string {
				if data.OrgID != nil {
					return fmt.Sprintf("%s/organizations/%s/roles/%s/delete", data.BasePath, data.OrgID.String(), roleID.String())
				}

				return fmt.Sprintf("%s/settings/roles/%s/delete", data.BasePath, roleID.String())
			},
			OnClone: func(roleID xid.ID) string {
				return fmt.Sprintf("%s/settings/roles/%s/clone", data.BasePath, roleID.String())
			},
			ShowActions: data.ShowActions,
		}),
	)
}

// renderAddPermissionModal renders a modal for creating custom permissions.
func renderAddPermissionModal() g.Node {
	return Div(
		ID("addPermissionModal"),
		Class("fixed inset-0 z-50 hidden items-center justify-center bg-black bg-opacity-50"),
		g.Attr("onclick", "if(event.target === this) this.classList.add('hidden')"),
		Div(
			Class("bg-white dark:bg-gray-900 rounded-lg p-6 max-w-md w-full mx-4 shadow-xl"),

			// Header
			Div(
				Class("flex items-center justify-between mb-4"),
				H3(
					Class("text-lg font-semibold text-gray-900 dark:text-white"),
					g.Text("Create Custom Permission"),
				),
				Button(
					Type("button"),
					Class("text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"),
					g.Attr("onclick", "document.getElementById('addPermissionModal').classList.add('hidden')"),
					lucide.X(Class("h-5 w-5")),
				),
			),

			// Form
			Form(
				ID("addPermissionForm"),
				Class("space-y-4"),
				g.Attr("onsubmit", "return addCustomPermission(event)"),

				// Permission Name
				Div(
					Label(
						Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
						g.Attr("for", "newPermName"),
						g.Text("Permission Name"),
						Span(Class("text-red-500"), g.Text(" *")),
					),
					Input(
						Type("text"),
						ID("newPermName"),
						Name("name"),
						Required(),
						Placeholder("e.g., users:create, content:delete"),
						Class("block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm"),
					),
					P(
						Class("mt-1 text-xs text-gray-500 dark:text-gray-400"),
						g.Text("Use format: resource:action (e.g., posts:edit)"),
					),
				),

				// Description
				Div(
					Label(
						Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
						g.Attr("for", "newPermDesc"),
						g.Text("Description"),
					),
					Textarea(
						ID("newPermDesc"),
						Name("description"),
						Rows("2"),
						Placeholder("Describe what this permission allows..."),
						Class("block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm"),
					),
				),

				// Category
				Div(
					Label(
						Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
						g.Attr("for", "newPermCategory"),
						g.Text("Category"),
					),
					Select(
						ID("newPermCategory"),
						Name("category"),
						Class("block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm"),
						Option(Value("users"), g.Text("Users")),
						Option(Value("organizations"), g.Text("Organizations")),
						Option(Value("settings"), g.Text("Settings")),
						Option(Value("dashboard"), g.Text("Dashboard")),
						Option(Value("roles"), g.Text("Roles")),
						Option(Value("permissions"), g.Text("Permissions")),
						Option(Value("sessions"), g.Text("Sessions")),
						Option(Value("apikeys"), g.Text("API Keys")),
						Option(Value("audit_logs"), g.Text("Audit Logs")),
						Option(Value("General"), Selected(), g.Text("General")),
					),
				),

				// Actions
				Div(
					Class("flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700"),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"),
						g.Attr("onclick", "document.getElementById('addPermissionModal').classList.add('hidden')"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"),
						g.Text("Create Permission"),
					),
				),
			),
		),

		// JavaScript for handling the form
		g.Raw(`
<script>
function filterPermissions(searchTerm) {
	const categories = document.querySelectorAll('.permission-category');
	searchTerm = searchTerm.toLowerCase();
	
	categories.forEach(category => {
		const checkboxes = category.querySelectorAll('input[type="checkbox"]');
		let visibleCount = 0;
		
		checkboxes.forEach(checkbox => {
			const label = checkbox.closest('.flex').querySelector('label');
			const text = label ? label.textContent.toLowerCase() : '';
			const visible = text.includes(searchTerm);
			
			checkbox.closest('.flex').style.display = visible ? '' : 'none';
			if (visible) visibleCount++;
		});
		
		// Hide category if no visible checkboxes
		category.style.display = visibleCount > 0 ? '' : 'none';
	});
}

async function addCustomPermission(event) {
	event.preventDefault();
	
	const form = event.target;
	const formData = new FormData(form);
	const name = formData.get('name');
	const description = formData.get('description');
	const category = formData.get('category');
	
	try {
		// Submit the form to create the permission
		const response = await fetch(window.location.pathname + '/add-permission', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({ name, description, category })
		});
		
		if (response.ok) {
			// Reload the page to show the new permission
			window.location.reload();
		} else {
			const error = await response.text();
			alert('Failed to create permission: ' + error);
		}
	} catch (error) {
		console.error('Error creating permission:', error);
		alert('Failed to create permission. Please try again.');
	}
	
	return false;
}
</script>
		`),
	)
}
