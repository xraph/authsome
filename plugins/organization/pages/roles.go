package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/card"
)

// RolesPage renders the organization roles management page
func RolesPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Back link
		BackLink(baseURL, "Back to Organization"),

		// Page header
		PageHeader(
			"Roles & Permissions",
			"Manage organization roles and their permissions",
		),

		// Roles info
		Div(
			Class("grid gap-6 md:grid-cols-3"),

			// Owner role
			card.Card(
				Class("border-primary/50"),
				card.Header(
					Div(
						Class("flex items-center gap-2"),
						lucide.Crown(Class("size-5 text-primary")),
						card.Title("Owner"),
					),
				),
				card.Content(
					Div(
						Class("space-y-3 text-sm"),
						P(Class("text-muted-foreground"), g.Text("The creator of the organization with full control.")),
						Div(
							Class("space-y-2"),
							permissionItem("Full organization management", true),
							permissionItem("Delete organization", true),
							permissionItem("Manage all members and teams", true),
							permissionItem("Invite and remove members", true),
							permissionItem("Create and manage teams", true),
							permissionItem("Update organization settings", true),
						),
						Div(
							Class("pt-2 border-t border-border"),
							P(Class("text-xs text-muted-foreground"), g.Text("Cannot be changed or removed")),
						),
					),
				),
			),

			// Admin role
			card.Card(
				card.Header(
					Div(
						Class("flex items-center gap-2"),
						lucide.ShieldCheck(Class("size-5 text-secondary")),
						card.Title("Admin"),
					),
				),
				card.Content(
					Div(
						Class("space-y-3 text-sm"),
						P(Class("text-muted-foreground"), g.Text("Administrators with elevated permissions.")),
						Div(
							Class("space-y-2"),
							permissionItem("View organization", true),
							permissionItem("Invite and remove members", true),
							permissionItem("Create and manage teams", true),
							permissionItem("Update organization settings", true),
							permissionItem("Delete organization", false),
							permissionItem("Change owner role", false),
						),
					),
				),
			),

			// Member role
			card.Card(
				card.Header(
					Div(
						Class("flex items-center gap-2"),
						lucide.User(Class("size-5 text-muted-foreground")),
						card.Title("Member"),
					),
				),
				card.Content(
					Div(
						Class("space-y-3 text-sm"),
						P(Class("text-muted-foreground"), g.Text("Regular members with view access.")),
						Div(
							Class("space-y-2"),
							permissionItem("View organization", true),
							permissionItem("View members and teams", true),
							permissionItem("Create teams", true),
							permissionItem("Invite members", false),
							permissionItem("Remove members", false),
							permissionItem("Manage teams", false),
						),
					),
				),
			),
		),

		// Additional info
		card.Card(
			Class("bg-muted/50"),
			card.Content(
				Class("p-6"),
				Div(
					Class("flex items-start gap-4"),
					Div(
						Class("rounded-full bg-primary/10 p-3"),
						lucide.Info(Class("size-6 text-primary")),
					),
					Div(
						Class("flex-1"),
						H3(Class("font-semibold mb-2"), g.Text("Role Management")),
						Div(
							Class("space-y-2 text-sm text-muted-foreground"),
							P(g.Text("• The organization owner is automatically assigned when creating an organization.")),
							P(g.Text("• Admins can be assigned by the owner or other admins.")),
							P(g.Text("• Member permissions can be extended through custom integrations.")),
							P(g.Text("• For custom role templates, visit the Settings page.")),
						),
						Div(
							Class("mt-4"),
							A(
								Href(appBase+"/settings/roles"),
								Class("inline-flex items-center gap-2 text-sm text-primary hover:underline"),
								lucide.Settings(Class("size-4")),
								g.Text("Manage Role Templates"),
								lucide.ArrowRight(Class("size-3")),
							),
						),
					),
				),
			),
		),
	)
}

// permissionItem renders a single permission item with checkmark or x
func permissionItem(text string, granted bool) g.Node {
	var icon g.Node
	var textClass string

	if granted {
		icon = lucide.Check(Class("size-4 text-emerald-600 dark:text-emerald-400"))
		textClass = "text-foreground"
	} else {
		icon = lucide.X(Class("size-4 text-muted-foreground"))
		textClass = "text-muted-foreground"
	}

	return Div(
		Class("flex items-center gap-2"),
		icon,
		Span(Class(textClass), g.Text(text)),
	)
}
