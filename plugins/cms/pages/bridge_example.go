package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ContentTypesListDynamic renders a dynamic content types list using bridge functions
// This demonstrates how to use the CMS bridge functions from the frontend
func ContentTypesListDynamic(currentApp *app.App, basePath string) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-2"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: "Content Types", Href: ""},
		),

		// Header
		PageHeader(
			"Content Types",
			"Manage your content types and schemas",
			PrimaryButton(appBase+"/cms/types/create", "Create Type", lucide.Plus(Class("size-4"))),
		),

		// Dynamic content loaded via bridge
		Div(
			// Alpine.js state management
			g.Attr("x-data", `{
				contentTypes: [],
				loading: true,
				error: null,
				
				async loadContentTypes() {
					this.loading = true;
					this.error = null;
					try {
						// Call the CMS bridge function
						const result = await $bridge.call('cms.getContentTypes', {
							appId: '`+currentApp.ID.String()+`'
						});
						this.contentTypes = result.types || [];
					} catch (err) {
						console.error('Failed to load content types:', err);
						this.error = err.message || 'Failed to load content types';
					} finally {
						this.loading = false;
					}
				},
				
				async deleteContentType(typeName) {
					if (!confirm('Are you sure you want to delete this content type?')) {
						return;
					}
					try {
						await $bridge.call('cms.deleteContentType', {
							appId: '`+currentApp.ID.String()+`',
							name: typeName
						});
						await this.loadContentTypes(); // Reload list
					} catch (err) {
						alert('Failed to delete content type: ' + err.message);
					}
				}
			}`),
			g.Attr("x-init", "loadContentTypes()"),

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

			// Content types grid
			Div(
				g.Attr("x-show", "!loading && !error && contentTypes.length > 0"),
				g.Attr("x-cloak", ""),
				Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),

				// Template for each content type
				g.El("template", g.Attr("x-for", "type in contentTypes"), g.Attr("x-bind:key", "type.id"),
					Div(
						Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-6 hover:border-violet-300 dark:hover:border-violet-600 transition-colors"),

						// Type name
						H3(
							Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
							g.Attr("x-text", "type.name"),
						),

						// Type description
						P(
							Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
							g.Attr("x-text", "type.description || 'No description'"),
						),

						// Stats
						Div(
							Class("flex items-center gap-4 mb-4 text-sm text-slate-600 dark:text-gray-400"),
							Div(
								Class("flex items-center gap-1"),
								lucide.FileText(Class("size-4")),
								Span(g.Attr("x-text", "(type.fields?.length || 0) + ' fields'")),
							),
						),

						// Actions
						Div(
							Class("flex items-center gap-2 pt-4 border-t border-slate-200 dark:border-gray-700"),
							A(
								g.Attr("x-bind:href", "`"+appBase+"/cms/types/${type.name}`"),
								Class("text-sm text-violet-600 dark:text-violet-400 hover:underline"),
								g.Text("View details →"),
							),
							Button(
								Type("button"),
								g.Attr("@click", "deleteContentType(type.name)"),
								Class("ml-auto text-sm text-red-600 dark:text-red-400 hover:underline"),
								g.Text("Delete"),
							),
						),
					),
				),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!loading && !error && contentTypes.length === 0"),
				g.Attr("x-cloak", ""),
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
	)
}

// EntriesListDynamic renders a dynamic entries list using bridge functions
func EntriesListDynamic(currentApp *app.App, basePath string, typeName string) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + typeName

	return Div(
		Class("space-y-2"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: typeName, Href: typeBase},
			BreadcrumbItem{Label: "Entries", Href: ""},
		),

		// Header
		PageHeader(
			typeName+" Entries",
			"Manage entries for this content type",
			PrimaryButton(typeBase+"/entries/create", "Create Entry", lucide.Plus(Class("size-4"))),
		),

		// Dynamic content loaded via bridge
		Div(
			// Alpine.js state management
			g.Attr("x-data", `{
				entries: [],
				stats: null,
				loading: true,
				error: null,
				page: 1,
				pageSize: 20,
				totalItems: 0,
				search: '',
				status: '',
				
				get totalPages() {
					return Math.ceil(this.totalItems / this.pageSize);
				},
				
				async loadEntries() {
					this.loading = true;
					this.error = null;
					try {
						// Call the CMS bridge function
						const result = await $bridge.call('cms.getEntries', {
							appId: '`+currentApp.ID.String()+`',
							typeName: '`+typeName+`',
							page: this.page,
							pageSize: this.pageSize,
							search: this.search,
							status: this.status
						});
						this.entries = result.entries || [];
						this.totalItems = result.totalItems || 0;
						this.stats = result.stats;
					} catch (err) {
						console.error('Failed to load entries:', err);
						this.error = err.message || 'Failed to load entries';
					} finally {
						this.loading = false;
					}
				},
				
				async deleteEntry(entryId) {
					if (!confirm('Are you sure you want to delete this entry?')) {
						return;
					}
					try {
						await $bridge.call('cms.deleteEntry', {
							appId: '`+currentApp.ID.String()+`',
							typeName: '`+typeName+`',
							entryId: entryId
						});
						await this.loadEntries(); // Reload list
					} catch (err) {
						alert('Failed to delete entry: ' + err.message);
					}
				},
				
				nextPage() {
					if (this.page < this.totalPages) {
						this.page++;
						this.loadEntries();
					}
				},
				
				prevPage() {
					if (this.page > 1) {
						this.page--;
						this.loadEntries();
					}
				}
			}`),
			g.Attr("x-init", "loadEntries()"),

			// Stats (if available)
			Div(
				g.Attr("x-show", "stats"),
				g.Attr("x-cloak", ""),
				Class("grid grid-cols-2 md:grid-cols-4 gap-4 mb-6"),

				Div(
					Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-4"),
					Div(Class("flex items-center gap-2 mb-2"),
						lucide.FileText(Class("size-5 text-blue-600")),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Total")),
					),
					Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Attr("x-text", "stats?.totalEntries || 0")),
				),

				Div(
					Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-4"),
					Div(Class("flex items-center gap-2 mb-2"),
						lucide.CircleCheck(Class("size-5 text-green-600")),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Published")),
					),
					Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Attr("x-text", "stats?.publishedEntries || 0")),
				),

				Div(
					Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-4"),
					Div(Class("flex items-center gap-2 mb-2"),
						lucide.Pencil(Class("size-5 text-yellow-600")),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Drafts")),
					),
					Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Attr("x-text", "stats?.draftEntries || 0")),
				),

				Div(
					Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg p-4"),
					Div(Class("flex items-center gap-2 mb-2"),
						lucide.Archive(Class("size-5 text-gray-600")),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Archived")),
					),
					Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Attr("x-text", "stats?.archivedEntries || 0")),
				),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600")),
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

			// Entries table
			Div(
				g.Attr("x-show", "!loading && !error && entries.length > 0"),
				g.Attr("x-cloak", ""),
				Class("bg-white dark:bg-gray-800 border border-slate-200 dark:border-gray-700 rounded-lg overflow-hidden"),

				Table(
					Class("w-full"),
					THead(
						Class("bg-slate-50 dark:bg-gray-900 border-b border-slate-200 dark:border-gray-700"),
						Tr(
							Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Entry")),
							Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
							Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Updated")),
							Th(Class("px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
						),
					),
					TBody(
						Class("divide-y divide-slate-200 dark:divide-gray-700"),
						g.El("template", g.Attr("x-for", "entry in entries"), g.Attr("x-bind:key", "entry.id"),
							Tr(
								Class("hover:bg-slate-50 dark:hover:bg-gray-900/50"),
								Td(
									Class("px-6 py-4"),
									Div(Class("text-sm font-medium text-slate-900 dark:text-white"), g.Attr("x-text", "entry.id")),
								),
								Td(
									Class("px-6 py-4"),
									Span(
										Class("px-2 py-1 text-xs rounded-full"),
										g.Attr("x-bind:class", `{
											'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': entry.status === 'published',
											'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400': entry.status === 'draft',
											'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400': entry.status === 'archived'
										}`),
										g.Attr("x-text", "entry.status"),
									),
								),
								Td(
									Class("px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
									g.Attr("x-text", "new Date(entry.updatedAt).toLocaleDateString()"),
								),
								Td(
									Class("px-6 py-4 text-right"),
									Div(
										Class("flex items-center justify-end gap-2"),
										A(
											g.Attr("x-bind:href", "`"+typeBase+"/entries/${entry.id}`"),
											Class("text-sm text-violet-600 dark:text-violet-400 hover:underline"),
											g.Text("View"),
										),
										Button(
											Type("button"),
											g.Attr("@click", "deleteEntry(entry.id)"),
											Class("text-sm text-red-600 dark:text-red-400 hover:underline"),
											g.Text("Delete"),
										),
									),
								),
							),
						),
					),
				),

				// Pagination
				Div(
					Class("px-6 py-4 border-t border-slate-200 dark:border-gray-700 flex items-center justify-between"),
					Div(
						Class("text-sm text-slate-600 dark:text-gray-400"),
						Span(g.Text("Page ")),
						Span(g.Attr("x-text", "page")),
						Span(g.Text(" of ")),
						Span(g.Attr("x-text", "totalPages")),
					),
					Div(
						Class("flex items-center gap-2"),
						Button(
							Type("button"),
							g.Attr("@click", "prevPage()"),
							g.Attr("x-bind:disabled", "page === 1"),
							Class("px-3 py-1 text-sm border border-slate-300 dark:border-gray-700 rounded hover:bg-slate-50 dark:hover:bg-gray-900 disabled:opacity-50 disabled:cursor-not-allowed"),
							g.Text("Previous"),
						),
						Button(
							Type("button"),
							g.Attr("@click", "nextPage()"),
							g.Attr("x-bind:disabled", "page === totalPages"),
							Class("px-3 py-1 text-sm border border-slate-300 dark:border-gray-700 rounded hover:bg-slate-50 dark:hover:bg-gray-900 disabled:opacity-50 disabled:cursor-not-allowed"),
							g.Text("Next"),
						),
					),
				),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!loading && !error && entries.length === 0"),
				g.Attr("x-cloak", ""),
				Class("text-center py-12"),
				Div(
					Class("inline-flex items-center justify-center w-16 h-16 rounded-full bg-slate-100 dark:bg-gray-800 mb-4"),
					lucide.FileText(Class("size-8 text-slate-400")),
				),
				H3(
					Class("text-lg font-medium text-slate-900 dark:text-white mb-2"),
					g.Text("No entries"),
				),
				P(
					Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
					g.Text("Create your first entry to get started"),
				),
				A(
					Href(typeBase+"/entries/create"),
					Class("inline-flex items-center gap-2 px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 transition-colors"),
					lucide.Plus(Class("size-4")),
					g.Text("Create Entry"),
				),
			),
		),
	)
}
