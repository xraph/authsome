package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/cms/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// CMS Overview Page
// =============================================================================

// CMSOverviewPage renders the CMS overview page with content types list
func CMSOverviewPage(
	currentApp *app.App,
	basePath string,
	contentTypes []*core.ContentTypeSummaryDTO,
	stats *core.CMSStatsDTO,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Build stats node only if stats is not nil
	var statsNode g.Node
	if stats != nil {
		statsNode = cmsStatsCards(stats)
	}

	return Div(
		Class("space-y-6"),

		// Header
		PageHeader(
			"Content Management",
			"Define content types and manage your content entries",
			PrimaryButton(appBase+"/cms/types/create", "Create Content Type", lucide.Plus(Class("size-4"))),
		),

		// Stats cards
		statsNode,

		// Content Types list
		contentTypesGrid(currentApp, basePath, contentTypes),
	)
}

// cmsStatsCards renders CMS statistics cards
func cmsStatsCards(stats *core.CMSStatsDTO) g.Node {
	return Div(
		Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
		StatCard("Content Types", fmt.Sprintf("%d", stats.TotalContentTypes), lucide.Database(Class("size-5")), "text-violet-600"),
		StatCard("Total Entries", fmt.Sprintf("%d", stats.TotalEntries), lucide.FileText(Class("size-5")), "text-blue-600"),
		StatCard("Published", fmt.Sprintf("%d", stats.EntriesByStatus["published"]), lucide.CircleCheck(Class("size-5")), "text-green-600"),
		StatCard("Drafts", fmt.Sprintf("%d", stats.EntriesByStatus["draft"]), lucide.Pencil(Class("size-5")), "text-yellow-600"),
	)
}

// contentTypesGrid renders the content types as a grid of cards
func contentTypesGrid(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	if len(contentTypes) == 0 {
		return Card(
			EmptyState(
				lucide.Database(Class("size-8 text-slate-400")),
				"No content types yet",
				"Create your first content type to start managing content. Content types define the structure for your entries.",
				"Create Content Type",
				appBase+"/cms/types/create",
			),
		)
	}

	cards := make([]g.Node, len(contentTypes))
	for i, ct := range contentTypes {
		cards[i] = contentTypeCard(appBase, ct)
	}

	return Div(
		Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),
		g.Group(cards),
	)
}

// contentTypeCard renders a single content type card
func contentTypeCard(appBase string, ct *core.ContentTypeSummaryDTO) g.Node {
	return A(
		Href(appBase+"/cms/types/"+ct.Name),
		Class("block"),
		Card(
			Div(
				Class("p-6"),
				// Icon and name
				Div(
					Class("flex items-start gap-4"),
					Div(
						Class("flex-shrink-0 rounded-lg bg-violet-100 p-3 dark:bg-violet-900/30"),
						g.If(ct.Icon != "", func() g.Node {
							return Span(Class("text-2xl"), g.Text(ct.Icon))
						}()),
						g.If(ct.Icon == "", func() g.Node {
							return lucide.Database(Class("size-6 text-violet-600 dark:text-violet-400"))
						}()),
					),
					Div(
						Class("flex-1 min-w-0"),
						H3(
							Class("text-lg font-semibold text-slate-900 dark:text-white truncate"),
							g.Text(ct.Name),
						),
						P(
							Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text(ct.Name),
						),
					),
				),

				// Description
				g.If(ct.Description != "", func() g.Node {
					return P(
						Class("mt-3 text-sm text-slate-600 dark:text-gray-400 line-clamp-2"),
						g.Text(ct.Description),
					)
				}()),

				// Stats
				Div(
					Class("flex items-center gap-4 mt-4 pt-4 border-t border-slate-200 dark:border-gray-800"),
					Div(
						Class("flex items-center gap-1.5 text-sm text-slate-600 dark:text-gray-400"),
						lucide.Layers(Class("size-4")),
						g.Text(fmt.Sprintf("%d fields", ct.FieldCount)),
					),
					Div(
						Class("flex items-center gap-1.5 text-sm text-slate-600 dark:text-gray-400"),
						lucide.FileText(Class("size-4")),
						g.Text(fmt.Sprintf("%d entries", ct.EntryCount)),
					),
				),
			),
		),
	)
}

// =============================================================================
// Content Types List Page (Alternate Table View)
// =============================================================================

// ContentTypesListPage renders the content types as a table
func ContentTypesListPage(
	currentApp *app.App,
	basePath string,
	contentTypes []*core.ContentTypeSummaryDTO,
	page, pageSize, totalItems int,
	searchQuery string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	totalPages := (totalItems + pageSize - 1) / pageSize

	return Div(
		Class("space-y-6"),

		// Header
		PageHeader(
			"Content Types",
			"Manage your content type definitions",
			PrimaryButton(appBase+"/cms/types/create", "Create Content Type", lucide.Plus(Class("size-4"))),
		),

		// Search
		Div(
			Class("flex flex-wrap gap-3"),
			SearchInput("Search content types...", searchQuery, appBase+"/cms/types"),
		),

		// Table
		contentTypesTable(currentApp, basePath, contentTypes, page, totalPages),
	)
}

// contentTypesTable renders content types as a table
func contentTypesTable(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO, page, totalPages int) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	if len(contentTypes) == 0 {
		return Card(
			EmptyState(
				lucide.Database(Class("size-8 text-slate-400")),
				"No content types found",
				"Create your first content type to define your content structure.",
				"Create Content Type",
				appBase+"/cms/types/create",
			),
		)
	}

	rows := make([]g.Node, len(contentTypes))
	for i, ct := range contentTypes {
		rows[i] = contentTypeRow(appBase, ct)
	}

	return Div(
		Card(DataTable(
			[]string{"Name", "Name", "Fields", "Entries", "Updated", "Actions"},
			rows,
		)),
		Pagination(page, totalPages, appBase+"/cms/types"),
	)
}

// contentTypeRow renders a single content type table row
func contentTypeRow(appBase string, ct *core.ContentTypeSummaryDTO) g.Node {
	return TableRow(
		// Name
		TableCell(Div(
			Class("flex items-center gap-3"),
			Div(
				Class("flex-shrink-0 rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30"),
				g.If(ct.Icon != "", func() g.Node {
					return Span(Class("text-lg"), g.Text(ct.Icon))
				}()),
				g.If(ct.Icon == "", func() g.Node {
					return lucide.Database(Class("size-4 text-violet-600 dark:text-violet-400"))
				}()),
			),
			Div(
				Div(Class("font-medium"), g.Text(ct.Name)),
				g.If(ct.Description != "", func() g.Node {
					return Div(
						Class("text-xs text-slate-500 dark:text-gray-500 truncate max-w-xs"),
						g.Text(ct.Description),
					)
				}()),
			),
		)),

		// Slug
		TableCellSecondary(Code(
			Class("text-xs bg-slate-100 dark:bg-gray-800 px-1.5 py-0.5 rounded"),
			g.Text(ct.Name),
		)),

		// Fields
		TableCellSecondary(g.Text(fmt.Sprintf("%d", ct.FieldCount))),

		// Entries
		TableCellSecondary(g.Text(fmt.Sprintf("%d", ct.EntryCount))),

		// Updated
		TableCellSecondary(g.Text(FormatTimeAgo(ct.UpdatedAt))),

		// Actions
		TableCellActions(
			IconButton(appBase+"/cms/types/"+ct.Name+"/entries", lucide.FileText(Class("size-4")), "View Entries", "text-blue-600"),
			IconButton(appBase+"/cms/types/"+ct.Name, lucide.Settings(Class("size-4")), "Edit Type", "text-slate-600"),
		),
	)
}

// =============================================================================
// Create Content Type Page
// =============================================================================

// CreateContentTypePage renders the create content type form
func CreateContentTypePage(
	currentApp *app.App,
	basePath string,
	err string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-6 max-w-2xl"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: "Create Content Type", Href: ""},
		),

		// Header
		PageHeader(
			"Create Content Type",
			"Define a new content type structure",
		),

		// Form
		Card(
			Div(
				Class("p-6"),
				g.If(err != "", func() g.Node {
					return Div(
						Class("mb-4 p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"),
						g.Text(err),
					)
				}()),

				FormEl(
					Method("POST"),
					Action(appBase+"/cms/types/create"),
					Class("space-y-6"),
					g.Attr("x-data", `{
					name: '',
					slug: '',
					slugManuallyEdited: false,
					generateSlug(name) {
						return name
							.toLowerCase()
							.trim()
							.replace(/[^\w\s-]/g, '')
							.replace(/[\s_-]+/g, '-')
							.replace(/^-+|-+$/g, '');
					},
					updateSlug() {
						if (!this.slugManuallyEdited) {
							this.slug = this.generateSlug(this.name);
						}
					}
				}`),

					// Name
					formFieldWithAlpine("name", "Name", "text", "", "e.g., Blog Posts, Products, Events", true, "", "name", "@input", "updateSlug()"),

					// Slug
					formFieldWithAlpine("slug", "Name", "text", "", "e.g., blog-posts, products, events", true, "URL-friendly identifier. Use lowercase letters, numbers, and hyphens.", "slug", "@input", "slugManuallyEdited = true"),

					// Description
					Div(
						Label(
							For("description"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
							g.Text("Description"),
						),
						Textarea(
							ID("description"),
							Name("description"),
							Rows("3"),
							Placeholder("Describe what this content type is for..."),
							Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						),
					),

					// Icon
					formField("icon", "Icon", "text", "", "Optional emoji icon, e.g., 📝, 📦, 📅", false, ""),

					// Submit buttons
					Div(
						Class("flex items-center gap-4 pt-4"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
							g.Text("Create Content Type"),
						),
						A(
							Href(appBase+"/cms"),
							Class("px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
	)
}

// formField creates a form field
func formField(name, label, inputType, value, placeholder string, required bool, help string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
			g.If(required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),
		Input(
			Type(inputType),
			ID(name),
			Name(name),
			Value(value),
			Placeholder(placeholder),
			g.If(required, Required()),
			Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
		),
		g.If(help != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(help),
			)
		}()),
	)
}

// formFieldWithAlpine creates a form field with Alpine.js x-model and event binding
func formFieldWithAlpine(name, label, inputType, value, placeholder string, required bool, help string, xModel string, eventName string, eventHandler string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
			g.If(required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),
		Input(
			Type(inputType),
			ID(name),
			Name(name),
			Value(value),
			Placeholder(placeholder),
			g.If(required, Required()),
			g.Attr("x-model", xModel),
			g.If(eventName != "", g.Attr(eventName, eventHandler)),
			Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
		),
		g.If(help != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(help),
			)
		}()),
	)
}
