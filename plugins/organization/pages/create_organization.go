package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
)

// CreateOrganizationPage renders the create organization form page.
func CreateOrganizationPage(currentApp *app.App, basePath string, errorMsg string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())

	return Div(
		Class("space-y-6"),

		// Back link
		BackLink(appBase+"/organizations", "Back to Organizations"),

		// Page header
		PageHeader(
			"Create Organization",
			"Create a new organization for your team",
		),

		// Error message
		g.If(errorMsg != "", func() g.Node {
			return Div(
				Class("bg-destructive/10 border border-destructive/20 rounded-lg p-4 mb-6"),
				Div(
					Class("flex items-center gap-2 text-destructive"),
					lucide.TriangleAlert(Class("size-5")),
					Span(g.Text(errorMsg)),
				),
			)
		}()),

		// Form
		Card(
			Class("p-6"),
			Form(
				Method("POST"),
				Action(appBase+"/organizations/create"),
				Class("space-y-6"),

				// Organization Name
				Div(
					Class("space-y-2"),
					Label(
						For("name"),
						Class("text-sm font-medium"),
						g.Text("Organization Name"),
						Span(Class("text-destructive"), g.Text("*")),
					),
					input.Input(
						input.WithType("text"),
						input.WithID("name"),
						input.WithName("name"),
						input.WithPlaceholder("Acme Inc."),
						input.WithAttrs(Required()),
					),
					P(Class("text-xs text-muted-foreground"), g.Text("The display name of your organization")),
				),

				// Slug
				Div(
					Class("space-y-2"),
					Label(
						For("slug"),
						Class("text-sm font-medium"),
						g.Text("Slug"),
						Span(Class("text-destructive"), g.Text("*")),
					),
					input.Input(
						input.WithType("text"),
						input.WithID("slug"),
						input.WithName("slug"),
						input.WithPlaceholder("acme-inc"),
						input.WithAttrs(
							Required(),
							g.Attr("pattern", "[a-z0-9-]+"),
						),
					),
					P(Class("text-xs text-muted-foreground"), g.Text("URL-friendly identifier (lowercase letters, numbers, and hyphens only)")),
				),

				// Logo URL (optional)
				Div(
					Class("space-y-2"),
					Label(
						For("logo"),
						Class("text-sm font-medium"),
						g.Text("Logo URL"),
					),
					input.Input(
						input.WithType("url"),
						input.WithID("logo"),
						input.WithName("logo"),
						input.WithPlaceholder("https://example.com/logo.png"),
					),
					P(Class("text-xs text-muted-foreground"), g.Text("Optional: URL to your organization's logo")),
				),

				// Metadata (optional)
				Div(
					Class("space-y-2"),
					Label(
						For("metadata"),
						Class("text-sm font-medium"),
						g.Text("Metadata (JSON)"),
					),
					Textarea(
						ID("metadata"),
						Name("metadata"),
						Placeholder(`{"key": "value"}`),
						Rows("4"),
						Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
					),
					P(Class("text-xs text-muted-foreground"), g.Text("Optional: Additional metadata in JSON format")),
				),

				// Actions
				Div(
					Class("flex justify-end gap-2 pt-4"),
					button.Button(
						g.Text("Cancel"),
						button.WithVariant("outline"),
						button.WithAttrs(
							Type("button"),
							g.Attr("onclick", fmt.Sprintf("window.location.href='%s/organizations'", appBase)),
						),
					),
					button.Button(
						Div(
							lucide.Plus(Class("size-4")),
							g.Text("Create Organization"),
						),
						button.WithVariant("default"),
						button.WithAttrs(Type("submit")),
					),
				),
			),
		),
	)
}
