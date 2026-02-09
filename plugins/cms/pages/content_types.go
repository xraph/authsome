package pages

import (
	"fmt"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/cms/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// CMS Overview Page
// =============================================================================

// CMSOverviewDynamic renders a dynamic CMS overview using bridge functions for client-side data fetching.
func CMSOverviewDynamic(currentApp *app.App, basePath string) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-2"),

		// Header
		PageHeader(
			"Content Management",
			"Define content types and manage your content entries",
			PrimaryButton(appBase+"/cms/types/create", "Create Content Type", lucide.Plus(Class("size-4"))),
		),

		// Dynamic content loaded via bridge
		Div(
			// Alpine.js state management
			g.Attr("x-data", `{
				contentTypes: [],
				stats: null,
				loading: true,
				error: null,
				
				async loadData() {
					this.loading = true;
					this.error = null;
					try {
						// Load content types
						const typesResult = await $bridge.call('cms.getContentTypes', {
							appId: '`+currentApp.ID.String()+`'
						});
						this.contentTypes = typesResult.contentTypes || [];
						
						// Load stats
						const statsResult = await $bridge.call('cms.getContentTypeStats', {
							appId: '`+currentApp.ID.String()+`'
						});
						this.stats = statsResult;
					} catch (err) {
						console.error('Failed to load CMS data:', err);
						this.error = err.message || 'Failed to load CMS data';
					} finally {
						this.loading = false;
					}
				}
			}`),
			g.Attr("x-init", "loadData()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(
					Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"),
				),
			),

			// Error state
			Div(
				g.Attr("x-show", "error"),
				g.Attr("x-cloak", ""),
				Class("bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"),
				Div(
					Class("flex items-center gap-2 text-red-700 dark:text-red-400"),
					lucide.TriangleAlert(Class("size-5")),
					Span(g.Attr("x-text", "error")),
				),
			),

			// Content - Stats and Content Types
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Stats cards
				Div(
					g.Attr("x-show", "stats"),
					Class("grid grid-cols-2 md:grid-cols-4 gap-4"),

					// Content Types stat
					Div(
						Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6"),
						Div(
							Class("flex items-center gap-3 mb-2"),
							Div(
								Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
								lucide.Database(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
							),
							Div(
								Div(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "stats?.totalContentTypes || 0"),
								),
								P(Class("text-sm text-slate-600 dark:text-gray-400"),
									g.Text("Content Types")),
							),
						),
					),

					// Total Entries stat
					Div(
						Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6"),
						Div(
							Class("flex items-center gap-3 mb-2"),
							Div(
								Class("flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20"),
								lucide.FileText(Class("h-5 w-5 text-blue-600 dark:text-blue-400")),
							),
							Div(
								Div(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "stats?.totalEntries || 0"),
								),
								P(Class("text-sm text-slate-600 dark:text-gray-400"),
									g.Text("Total Entries")),
							),
						),
					),

					// Published stat
					Div(
						Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6"),
						Div(
							Class("flex items-center gap-3 mb-2"),
							Div(
								Class("flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20"),
								lucide.CircleCheck(Class("h-5 w-5 text-green-600 dark:text-green-400")),
							),
							Div(
								Div(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "stats?.entriesByStatus?.published || 0"),
								),
								P(Class("text-sm text-slate-600 dark:text-gray-400"),
									g.Text("Published")),
							),
						),
					),

					// Drafts stat
					Div(
						Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6"),
						Div(
							Class("flex items-center gap-3 mb-2"),
							Div(
								Class("flex h-10 w-10 items-center justify-center rounded-lg bg-yellow-100 dark:bg-yellow-900/20"),
								lucide.Pencil(Class("h-5 w-5 text-yellow-600 dark:text-yellow-400")),
							),
							Div(
								Div(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "stats?.entriesByStatus?.draft || 0"),
								),
								P(Class("text-sm text-slate-600 dark:text-gray-400"),
									g.Text("Drafts")),
							),
						),
					),
				),

				// Content Types Grid
				Div(
					g.Attr("x-show", "contentTypes.length > 0"),
					Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),

					// Template for each content type
					g.El("template", g.Attr("x-for", "type in contentTypes"), g.Attr("x-bind:key", "type.id"),
						Div(
							Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6 hover:border-violet-300 dark:hover:border-violet-600 transition-colors"),

							// Icon and title
							Div(
								Class("flex items-start justify-between mb-3"),
								Div(
									Class("flex items-center gap-3"),
									Div(
										Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
										lucide.Database(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
									),
									H3(
										Class("text-lg font-semibold text-slate-900 dark:text-white"),
										g.Attr("x-text", "type.name"),
									),
								),
								Span(
									Class("px-2 py-1 text-xs rounded-full bg-slate-100 dark:bg-gray-700 text-slate-600 dark:text-gray-400"),
									g.Attr("x-text", "(type.entryCount || 0) + ' entries'"),
								),
							),

							// Description
							P(
								Class("text-sm text-slate-600 dark:text-gray-400 mb-4 line-clamp-2"),
								g.Attr("x-text", "type.description || 'No description'"),
							),

							// Actions
							Div(
								Class("flex items-center gap-2 pt-4 border-t border-slate-200 dark:border-gray-700"),
								A(
									g.Attr("x-bind:href", "`"+appBase+"/cms/types/${type.name}`"),
									Class("text-sm text-violet-600 dark:text-violet-400 hover:underline font-medium"),
									g.Text("Manage →"),
								),
								A(
									g.Attr("x-bind:href", "`"+appBase+"/cms/types/${type.name}/entries`"),
									Class("ml-auto text-sm text-slate-600 dark:text-gray-400 hover:underline"),
									g.Text("View entries"),
								),
							),
						),
					),
				),

				// Empty state
				Div(
					g.Attr("x-show", "contentTypes.length === 0"),
					Class("text-center py-12"),
					Div(
						Class("inline-flex items-center justify-center w-16 h-16 rounded-full bg-slate-100 dark:bg-gray-800 mb-4"),
						lucide.Database(Class("size-8 text-slate-400")),
					),
					H3(
						Class("text-lg font-medium text-slate-900 dark:text-white mb-2"),
						g.Text("No content types"),
					),
					P(
						Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
						g.Text("Create your first content type to get started"),
					),
					A(
						Href(appBase+"/cms/types/create"),
						Class("inline-flex items-center gap-2 px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 transition-colors"),
						lucide.Plus(Class("size-4")),
						g.Text("Create Content Type"),
					),
				),
			),
		),
	)
}

// CMSOverviewPage renders the CMS overview page with content types list (SSR version).
func CMSOverviewPage(
	currentApp *app.App,
	basePath string,
	contentTypes []*core.ContentTypeSummaryDTO,
	stats *core.CMSStatsDTO,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

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

// cmsStatsCards renders CMS statistics cards.
func cmsStatsCards(stats *core.CMSStatsDTO) g.Node {
	return Div(
		Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
		StatCard("Content Types", strconv.Itoa(stats.TotalContentTypes), lucide.Database(Class("size-5")), "text-violet-600"),
		StatCard("Total Entries", strconv.Itoa(stats.TotalEntries), lucide.FileText(Class("size-5")), "text-blue-600"),
		StatCard("Published", strconv.Itoa(stats.EntriesByStatus["published"]), lucide.CircleCheck(Class("size-5")), "text-green-600"),
		StatCard("Drafts", strconv.Itoa(stats.EntriesByStatus["draft"]), lucide.Pencil(Class("size-5")), "text-yellow-600"),
	)
}

// contentTypesGrid renders the content types as a grid of cards.
func contentTypesGrid(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

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

// contentTypeCard renders a single content type card.
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

// ContentTypesListPage renders the content types as a table.
func ContentTypesListPage(
	currentApp *app.App,
	basePath string,
	contentTypes []*core.ContentTypeSummaryDTO,
	page, pageSize, totalItems int,
	searchQuery string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
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

// contentTypesTable renders content types as a table.
func contentTypesTable(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO, page, totalPages int) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

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

// contentTypeRow renders a single content type table row.
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
		TableCellSecondary(g.Text(strconv.Itoa(ct.FieldCount))),

		// Entries
		TableCellSecondary(g.Text(strconv.Itoa(ct.EntryCount))),

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

// CreateContentTypePage renders the create content type form.
func CreateContentTypePage(
	currentApp *app.App,
	basePath string,
	err string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

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

				Div(
					g.Attr("x-data", `{
					name: '',
					slug: '',
					description: '',
					icon: '',
					slugManuallyEdited: false,
					loading: false,
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
					},
					async createContentType() {
						if (!this.name || !this.slug) {
							this.$root.notification.error('Name and slug are required');
							return;
						}
						
						this.loading = true;
						try {
							const result = await $bridge.call('cms.createContentType', {
								appId: '`+currentApp.ID.String()+`',
								title: this.name,
								name: this.slug,
								description: this.description,
								icon: this.icon
							});
							this.$root.notification.success('Content type created!');
							// Use the slug we submitted since bridge returns the full DTO
							window.location.href = '`+appBase+`/cms/types/' + this.slug;
						} catch (err) {
							const errorMsg = err.error?.message || err.message || 'Failed to create content type';
							this.$root.notification.error(errorMsg);
							this.loading = false;
						}
					}
				}`),
					FormEl(
						g.Attr("@submit.prevent", "createContentType()"),
						Class("space-y-6"),

						// Name
						formFieldWithAlpine("name", "Name", "text", "", "e.g., Blog Posts, Products, Events", true, "", "name", "@input", "updateSlug()"),

						// Slug
						formFieldWithAlpine("slug", "Slug", "text", "", "e.g., blog-posts, products, events", true, "URL-friendly identifier. Use lowercase letters, numbers, and hyphens.", "slug", "@input", "slugManuallyEdited = true"),

						// Description
						Div(
							Label(
								For("description"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
								g.Text("Description"),
							),
							Textarea(
								ID("description"),
								g.Attr("x-model", "description"),
								Rows("3"),
								Placeholder("Describe what this content type is for..."),
								Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
						),

						// Icon
						Div(
							Label(
								For("icon"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
								g.Text("Icon"),
								Span(
									Class("text-xs text-slate-500 dark:text-gray-500 ml-2"),
									g.Text("(Optional)"),
								),
							),
							Input(
								Type("text"),
								ID("icon"),
								g.Attr("x-model", "icon"),
								Placeholder("Optional emoji icon, e.g., 📝, 📦, 📅"),
								Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
						),

						// Submit buttons
						Div(
							Class("flex items-center gap-4 pt-4"),
							Button(
								Type("submit"),
								g.Attr(":disabled", "loading"),
								g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''"),
								Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
								Span(
									g.Attr("x-show", "!loading"),
									g.Text("Create Content Type"),
								),
								Span(
									g.Attr("x-show", "loading"),
									g.Attr("x-cloak", ""),
									Class("flex items-center gap-2"),
									g.El("svg",
										Class("animate-spin h-4 w-4"),
										g.Attr("xmlns", "http://www.w3.org/2000/svg"),
										g.Attr("fill", "none"),
										g.Attr("viewBox", "0 0 24 24"),
										g.El("circle",
											Class("opacity-25"),
											g.Attr("cx", "12"),
											g.Attr("cy", "12"),
											g.Attr("r", "10"),
											g.Attr("stroke", "currentColor"),
											g.Attr("stroke-width", "4"),
										),
										g.El("path",
											Class("opacity-75"),
											g.Attr("fill", "currentColor"),
											g.Attr("d", "M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"),
										),
									),
									g.Text("Creating..."),
								),
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
		),
	)
}

// formField creates a form field.
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

// formFieldWithAlpine creates a form field with Alpine.js x-model and event binding.
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
