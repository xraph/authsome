package pages

import (
	"fmt"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/checkbox"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/components/textarea"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// Content Type Detail Page
// =============================================================================

// ContentTypeDetailPage renders the content type detail/edit page
func ContentTypeDetailPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	stats *core.ContentTypeStatsDTO,
	environmentID string,
	allContentTypes []*core.ContentTypeSummaryDTO,
	allComponentSchemas []*core.ComponentSchemaSummaryDTO,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Name
	apiBase := strings.ReplaceAll(basePath, "/ui", "") + "/cms/" + contentType.Name

	// Build icon node
	var iconNode g.Node
	if contentType.Icon != "" {
		iconNode = Span(Class("text-3xl"), g.Text(contentType.Icon))
	} else {
		iconNode = lucide.Database(Class("size-8 text-violet-600 dark:text-violet-400"))
	}

	// Build description node
	var descNode g.Node
	if contentType.Description != "" {
		descNode = P(
			Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
			g.Text(contentType.Description),
		)
	}

	// Build stats node only if stats is not nil
	var statsNode g.Node
	if stats != nil {
		statsNode = contentTypeStats(stats)
	}

	return Div(
		Class("space-y-2"),
		g.Attr("x-data", `{ activeTab: 'fields' }`),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: ""},
		),

		// Header with actions
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"),
			Div(
				Class("flex items-start gap-4"),
				// Icon
				Div(
					Class("flex-shrink-0 rounded-lg bg-violet-100 p-4 dark:bg-violet-900/30"),
					iconNode,
				),
				// Title and description
				Div(
					H1(
						Class("text-2xl font-bold text-slate-900 dark:text-white"),
						g.Text(contentType.Name),
					),
					P(
						Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
						Code(
							Class("text-xs bg-slate-100 dark:bg-gray-800 px-1.5 py-0.5 rounded"),
							g.Text(contentType.Name),
						),
					),
					descNode,
				),
			),
			// Actions
			Div(
				Class("flex items-center gap-2"),
				PrimaryButton(typeBase+"/entries", "View Entries", lucide.FileText(Class("size-4"))),
				PrimaryButton(typeBase+"/entries/create", "Create Entry", lucide.Plus(Class("size-4"))),
			),
		),

		// Stats
		statsNode,

		// Tabs Navigation
		Div(
			Class("border-b border-slate-200 dark:border-gray-800"),
			Nav(
				Class("flex gap-6"),
				alpineTabButton("fields", "Fields", lucide.Layers(Class("size-4"))),
				alpineTabButton("settings", "Settings", lucide.Settings(Class("size-4"))),
				alpineTabButton("api", "API", lucide.Code(Class("size-4"))),
				alpineTabButton("playground", "Playground", lucide.Play(Class("size-4"))),
			),
		),

		// Tab Content
		Div(
			// Fields Tab
			Div(
				g.Attr("x-show", "activeTab === 'fields'"),
				g.Attr("x-cloak", ""),
				fieldsSection(appBase, contentType, allContentTypes, allComponentSchemas),
			),

			// Settings Tab
			Div(
				g.Attr("x-show", "activeTab === 'settings'"),
				g.Attr("x-cloak", ""),
				settingsSection(typeBase, contentType),
			),

			// API Tab
			Div(
				g.Attr("x-show", "activeTab === 'api'"),
				g.Attr("x-cloak", ""),
				apiSection(apiBase, contentType),
			),

			// Playground Tab
			Div(
				g.Attr("x-show", "activeTab === 'playground'"),
				g.Attr("x-cloak", ""),
				playgroundSection(apiBase, contentType, currentApp.ID.String(), environmentID),
			),
		),
	)
}

// alpineTabButton renders a tab button with Alpine.js state
func alpineTabButton(id, label string, icon g.Node) g.Node {
	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf("activeTab = '%s'", id)),
		g.Attr(":class", fmt.Sprintf("activeTab === '%s' ? 'border-violet-600 text-violet-600 dark:border-violet-400 dark:text-violet-400' : 'border-transparent text-slate-600 hover:text-slate-900 hover:border-slate-300 dark:text-gray-400 dark:hover:text-white'", id)),
		Class("pb-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2"),
		icon,
		g.Text(label),
	)
}

// contentTypeStats renders statistics for a content type
func contentTypeStats(stats *core.ContentTypeStatsDTO) g.Node {
	return Div(
		Class("grid grid-cols-2 md:grid-cols-5 gap-4"),
		StatCard("Total Entries", fmt.Sprintf("%d", stats.TotalEntries), lucide.FileText(Class("size-5")), "text-blue-600"),
		StatCard("Published", fmt.Sprintf("%d", stats.PublishedEntries), lucide.CircleCheck(Class("size-5")), "text-green-600"),
		StatCard("Drafts", fmt.Sprintf("%d", stats.DraftEntries), lucide.Pencil(Class("size-5")), "text-yellow-600"),
		StatCard("Archived", fmt.Sprintf("%d", stats.ArchivedEntries), lucide.Archive(Class("size-5")), "text-gray-600"),
	)
}

// tabButton renders a tab button
func tabButton(label string, active bool) g.Node {
	activeClass := "border-violet-600 text-violet-600 dark:border-violet-400 dark:text-violet-400"
	inactiveClass := "border-transparent text-slate-600 hover:text-slate-900 hover:border-slate-300 dark:text-gray-400 dark:hover:text-white dark:hover:border-gray-600"

	class := inactiveClass
	if active {
		class = activeClass
	}

	return Button(
		Type("button"),
		Class("pb-3 text-sm font-medium border-b-2 transition-colors "+class),
		g.Text(label),
	)
}

// fieldsSection renders the fields management section with Preline/Alpine drawer
func fieldsSection(appBase string, contentType *core.ContentTypeDTO, allContentTypes []*core.ContentTypeSummaryDTO, allComponentSchemas []*core.ComponentSchemaSummaryDTO) g.Node {
	typeBase := appBase + "/cms/types/" + contentType.Name

	return Div(
		g.Attr("x-data", "{ drawerOpen: false }"),
		g.Attr("@close-drawer.window", "drawerOpen = false"),
		Class("relative"),

		// Main content
		CardWithHeader(
			"Fields",
			[]g.Node{
				// Trigger button for drawer
				button.Button(
					g.Group([]g.Node{
						lucide.Plus(Class("size-4")),
						g.Text("Add Field"),
					}),
					button.WithVariant(forgeui.VariantOutline),
					button.WithAttrs(
						Type("button"),
						g.Attr("@click", "drawerOpen = true"),
					),
				),
			},
			g.If(len(contentType.Fields) == 0, func() g.Node {
				return Div(
					Class("text-center py-12"),
					lucide.Layers(Class("mx-auto size-12 text-slate-300 dark:text-gray-600")),
					H3(
						Class("mt-4 text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("No fields defined"),
					),
					P(
						Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Add fields to define the structure of your content entries."),
					),
					button.Button(
						g.Group([]g.Node{
							lucide.Plus(Class("size-4")),
							g.Text("Add Field"),
						}),
						button.WithVariant(forgeui.VariantDefault),
						button.WithClass("mt-4"),
						button.WithAttrs(
							Type("button"),
							g.Attr("@click", "drawerOpen = true"),
						),
					),
				)
			}()),

			g.If(len(contentType.Fields) > 0, func() g.Node {
				return fieldsTable(typeBase, contentType.Fields, contentType.AppID, contentType.Name)
			}()),
		),

		// Sheet/Drawer - slides from right
		g.El("div",
			g.Attr("x-show", "drawerOpen"),
			g.Attr("x-cloak", ""),
			Class("fixed inset-0 z-50"),
			g.Attr("role", "dialog"),
			g.Attr("aria-modal", "true"),

			// Backdrop overlay
			g.El("div",
				g.Attr("x-show", "drawerOpen"),
				g.Attr("x-transition:enter", "transition-opacity ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0"),
				g.Attr("x-transition:enter-end", "opacity-100"),
				g.Attr("x-transition:leave", "transition-opacity ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100"),
				g.Attr("x-transition:leave-end", "opacity-0"),
				g.Attr("@click", "drawerOpen = false"),
				Class("fixed inset-0 bg-black/50 backdrop-blur-sm"),
			),

			// Sheet panel - using grid for reliable layout
			g.El("div",
				g.Attr("x-show", "drawerOpen"),
				g.Attr("x-transition:enter", "transform transition ease-out duration-300"),
				g.Attr("x-transition:enter-start", "translate-x-full"),
				g.Attr("x-transition:enter-end", "translate-x-0"),
				g.Attr("x-transition:leave", "transform transition ease-in duration-200"),
				g.Attr("x-transition:leave-start", "translate-x-0"),
				g.Attr("x-transition:leave-end", "translate-x-full"),
				Class("fixed top-0 right-0 bottom-0 w-full max-w-md bg-background border-l border-border shadow-xl flex flex-col"),

				// Header
				g.El("div",
					Class("flex items-center justify-between px-6 py-4 border-b border-border shrink-0"),
					g.El("div",
						g.El("h2",
							Class("text-lg font-semibold text-foreground"),
							g.Text("Add Field"),
						),
						g.El("p",
							Class("text-sm text-muted-foreground mt-0.5"),
							g.Text("Configure a new field for this content type"),
						),
					),
					button.Button(
						lucide.X(Class("h-4 w-4")),
						button.WithVariant(forgeui.VariantGhost),
						button.WithSize(forgeui.SizeIcon),
						button.WithClass("rounded-full hover:bg-accent"),
						button.WithAttrs(
							Type("button"),
							g.Attr("@click", "drawerOpen = false"),
						),
					),
				),

				// Scrollable body - buttons are now inside the form
				g.El("div",
					Class("flex-1 overflow-y-auto p-6"),
					addFieldForm(typeBase, contentType, allContentTypes, allComponentSchemas),
				),
			),
		),
	)
}

// addFieldForm renders the form for adding a new field using Preline-style components with Alpine.js
func addFieldForm(typeBase string, contentType *core.ContentTypeDTO, allContentTypes []*core.ContentTypeSummaryDTO, allComponentSchemas []*core.ComponentSchemaSummaryDTO) g.Node {
	fieldTypesByCategory := core.GetFieldTypesByCategory()

	// Build content type options for relations (exclude current type)
	var contentTypeOptions []string
	for _, ct := range allContentTypes {
		if ct.ID != contentType.ID {
			contentTypeOptions = append(contentTypeOptions, fmt.Sprintf(`<option value="%s">%s</option>`, ct.Name, ct.Name))
		}
	}
	contentTypeOptionsHTML := strings.Join(contentTypeOptions, "\n")

	// Build component schema options for nested fields
	var componentSchemaOptions []string
	for _, cs := range allComponentSchemas {
		componentSchemaOptions = append(componentSchemaOptions, fmt.Sprintf(`<option value="%s">%s</option>`, cs.Name, cs.Name))
	}
	componentSchemaOptionsHTML := strings.Join(componentSchemaOptions, "\n")

	return FormEl(
		ID("add-field-form"),
		g.Attr("@submit.prevent", "submitAddField($event)"),
		Class("space-y-5"),
		g.Attr("x-data", drawerFieldFormAlpineData(contentType.AppID, contentType.Name)),

		// Name field with auto-slug
		Div(
			Label(
				Class("block mb-2 text-sm font-medium"),
				g.Text("Name"),
				Span(Class("text-red-500 ms-1"), g.Text("*")),
			),
			input.Input(
				input.WithType("text"),
				input.WithID("field-name"),
				input.WithName("name"),
				input.WithPlaceholder("e.g., Title, Content, Author"),
				input.WithAttrs(
					Required(),
					g.Attr("x-model", "name"),
					g.Attr("@input", "updateSlug()"),
				),
			),
		),

		// Slug field
		Div(
			Label(
				Class("block mb-2 text-sm font-medium"),
				g.Text("Slug"),
				Span(Class("text-red-500 ms-1"), g.Text("*")),
			),
			input.Input(
				input.WithType("text"),
				input.WithID("field-slug"),
				input.WithName("slug"),
				input.WithPlaceholder("e.g., title, content, author"),
				input.WithClass("font-mono"),
				input.WithAttrs(
					Required(),
					g.Attr("x-model", "slug"),
					g.Attr("@input", "slugEdited = true"),
				),
			),
		),

		// Type field with categories
		Div(
			Label(
				Class("block mb-2 text-sm font-medium"),
				g.Text("Type"),
				Span(Class("text-red-500 ms-1"), g.Text("*")),
			),
			Select(
				ID("field-type"),
				Name("type"),
				Required(),
				g.Attr("x-model", "fieldType"),
				Class("py-2.5 px-3 pe-9 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 disabled:opacity-50 disabled:pointer-events-none dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-400 dark:placeholder-neutral-500 dark:focus:ring-neutral-600"),
				Option(Value(""), g.Text("Select a field type...")),
				g.Group(func() []g.Node {
					categories := []string{"text", "number", "date", "selection", "relation", "media", "nested", "advanced"}
					categoryNames := map[string]string{
						"text": "📝 Text", "number": "🔢 Number", "date": "📅 Date & Time",
						"selection": "☑️ Selection", "relation": "🔗 Relations", "media": "🖼️ Media",
						"nested": "📦 Nested", "advanced": "⚙️ Advanced",
					}
					var groups []g.Node
					for _, cat := range categories {
						types, ok := fieldTypesByCategory[cat]
						if !ok || len(types) == 0 {
							continue
						}
						options := make([]g.Node, len(types))
						for i, ft := range types {
							options[i] = Option(Value(ft.Type.String()), g.Text(ft.Name))
						}
						groups = append(groups, OptGroup(g.Attr("label", categoryNames[cat]), g.Group(options)))
					}
					return groups
				}()),
			),
		),

		// Description field
		Div(
			Label(
				Class("block mb-2 text-sm font-medium"),
				g.Text("Description"),
			),
			textarea.Textarea(
				textarea.WithID("field-description"),
				textarea.WithName("description"),
				textarea.WithPlaceholder("Help text for editors..."),
				textarea.WithClass("resize-none"),
				textarea.WithAttrs(Rows("2")),
			),
		),

		// Type-specific options section
		Div(
			g.Attr("x-show", "fieldType !== ''"),
			g.Attr("x-transition", ""),
			Class("border-t border-gray-200 dark:border-neutral-700 pt-4 space-y-4"),

			P(Class("text-sm font-medium"), g.Text("Field Configuration")),

			// Text type options
			Div(
				g.Attr("x-show", "isTextType()"),
				Class("space-y-3"),
				Div(
					Class("grid grid-cols-2 gap-3"),
					Div(
						Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Min Length")),
						input.Input(
							input.WithType("number"),
							input.WithName("options.minLength"),
							input.WithPlaceholder("0"),
							input.WithAttrs(g.Attr("min", "0")),
						),
					),
					Div(
						Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Max Length")),
						input.Input(
							input.WithType("number"),
							input.WithName("options.maxLength"),
							input.WithPlaceholder("255"),
							input.WithAttrs(g.Attr("min", "0")),
						),
					),
				),
				Div(
					Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Regex Pattern")),
					input.Input(
						input.WithType("text"),
						input.WithName("options.regex"),
						input.WithPlaceholder("e.g., ^[A-Za-z]+$"),
						input.WithClass("font-mono"),
					),
				),
			),

			// Number type options
			Div(
				g.Attr("x-show", "isNumberType()"),
				Class("space-y-3"),
				Div(
					Class("grid grid-cols-3 gap-3"),
					Div(
						Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Min")),
						input.Input(
							input.WithType("number"),
							input.WithName("options.min"),
							input.WithPlaceholder("0"),
						),
					),
					Div(
						Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Max")),
						input.Input(
							input.WithType("number"),
							input.WithName("options.max"),
							input.WithPlaceholder("100"),
						),
					),
					Div(
						Label(Class("block mb-1 text-xs font-medium text-muted-foreground"), g.Text("Step")),
						input.Input(
							input.WithType("number"),
							input.WithName("options.step"),
							input.WithPlaceholder("1"),
							input.WithAttrs(g.Attr("min", "0"), g.Attr("step", "any")),
						),
					),
				),
			),

			// Selection/Enum type options
			Div(
				g.Attr("x-show", "isSelectionType()"),
				Class("space-y-3"),
				Div(
					Class("flex items-center justify-between"),
					Label(Class("text-xs font-medium text-muted-foreground"), g.Text("Options")),
					Span(Class("text-xs text-gray-400"), g.Attr("x-text", "enumOptions.length + ' option(s)'")),
				),
				Div(
					Class("space-y-2 max-h-48 overflow-y-auto"),
					g.El("template",
						g.Attr("x-for", "(opt, idx) in enumOptions"),
						g.Attr(":key", "idx"),
						Div(
							Class("flex gap-2 items-center"),
							input.Input(
								input.WithType("text"),
								input.WithPlaceholder("Label"),
								input.WithClass("flex-1"),
								input.WithAttrs(
									g.Attr("x-model", "opt.label"),
									g.Attr(":name", "'options.enum[' + idx + '].label'"),
								),
							),
							input.Input(
								input.WithType("text"),
								input.WithPlaceholder("Value"),
								input.WithClass("flex-1 font-mono"),
								input.WithAttrs(
									g.Attr("x-model", "opt.value"),
									g.Attr(":name", "'options.enum[' + idx + '].value'"),
									g.Attr("@input", "if (!opt.value && opt.label) opt.value = opt.label.toLowerCase().replace(/\\s+/g, '_')"),
								),
							),
							button.Button(
								lucide.X(Class("size-4")),
								button.WithVariant(forgeui.VariantGhost),
								button.WithSize(forgeui.SizeIcon),
								button.WithClass("text-gray-400 hover:text-red-500"),
								button.WithAttrs(
									Type("button"),
									g.Attr("@click", "removeOption(idx)"),
									g.Attr(":disabled", "enumOptions.length <= 1"),
								),
							),
						),
					),
				),
				button.Button(
					g.Group([]g.Node{
						lucide.Plus(Class("size-4")),
						g.Text("Add Option"),
					}),
					button.WithVariant(forgeui.VariantOutline),
					button.WithClass("w-full border-dashed hover:border-primary hover:text-primary"),
					button.WithAttrs(
						Type("button"),
						g.Attr("@click", "addOption()"),
					),
				),
			),

			// Relation, Media, and other type-specific options (keeping as raw HTML for complex structures)
			g.Raw(fmt.Sprintf(`
			<!-- Relation type options -->
			<div x-show="fieldType === 'relation'" class="space-y-3">
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Related Content Type</label>
					<select name="options.relatedType" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<option value="">Select content type...</option>
						%s
					</select>
				</div>
				<div>
					<label class="block mb-2 text-xs font-medium text-gray-600 dark:text-neutral-400">Relation Type</label>
					<div class="grid grid-cols-2 gap-2">
						<label class="flex flex-col items-center p-3 border-2 rounded-lg cursor-pointer transition-all" :class="relationType === 'oneToOne' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700'">
							<input type="radio" name="options.relationType" value="oneToOne" x-model="relationType" class="sr-only">
							<span class="text-sm font-medium">One to One</span>
							<span class="text-xs text-gray-500">1 → 1</span>
						</label>
						<label class="flex flex-col items-center p-3 border-2 rounded-lg cursor-pointer transition-all" :class="relationType === 'oneToMany' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700'">
							<input type="radio" name="options.relationType" value="oneToMany" x-model="relationType" class="sr-only">
							<span class="text-sm font-medium">One to Many</span>
							<span class="text-xs text-gray-500">1 → N</span>
						</label>
						<label class="flex flex-col items-center p-3 border-2 rounded-lg cursor-pointer transition-all" :class="relationType === 'manyToOne' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700'">
							<input type="radio" name="options.relationType" value="manyToOne" x-model="relationType" class="sr-only">
							<span class="text-sm font-medium">Many to One</span>
							<span class="text-xs text-gray-500">N → 1</span>
						</label>
						<label class="flex flex-col items-center p-3 border-2 rounded-lg cursor-pointer transition-all" :class="relationType === 'manyToMany' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700'">
							<input type="radio" name="options.relationType" value="manyToMany" x-model="relationType" class="sr-only">
							<span class="text-sm font-medium">Many to Many</span>
							<span class="text-xs text-gray-500">N → N</span>
						</label>
					</div>
				</div>
			</div>
			
			<!-- Media type options -->
			<div x-show="fieldType === 'media'" class="space-y-3">
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Allowed Media</label>
					<select name="options.mediaType" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<option value="any">Any file type</option>
						<option value="image">Images only</option>
						<option value="video">Videos only</option>
						<option value="audio">Audio only</option>
						<option value="document">Documents only</option>
					</select>
				</div>
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Max File Size (MB)</label>
					<input type="number" name="options.maxFileSize" min="1" max="100" placeholder="10" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
				</div>
			</div>
			
			<!-- Slug type options -->
			<div x-show="fieldType === 'slug'" class="space-y-3">
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Generate From Field</label>
					<input type="text" name="options.sourceField" placeholder="e.g., title" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
					<p class="mt-1 text-xs text-gray-500">Auto-generate slug from another field</p>
				</div>
			</div>
			
			<!-- Boolean type options -->
			<div x-show="fieldType === 'boolean'" class="space-y-3">
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Default Value</label>
					<select name="options.defaultBool" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<option value="">No default</option>
						<option value="true">True</option>
						<option value="false">False</option>
					</select>
				</div>
			</div>
			
			<!-- Nested type options (object/array) -->
			<div x-show="isNestedType()" class="space-y-3">
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Schema Source</label>
					<select name="schemaSource" x-model="schemaSource" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<option value="component">Use Component Schema</option>
						<option value="inline">Define Inline</option>
					</select>
				</div>
				<div x-show="schemaSource === 'component'">
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Component Schema</label>
					<select name="options.componentRef" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<option value="">Select a component schema...</option>
						%s
					</select>
					<p class="mt-1 text-xs text-gray-500">Reusable schema defined in Component Schemas</p>
				</div>
				<div x-show="schemaSource === 'inline'" class="text-xs text-amber-600 dark:text-amber-400">
					<p>Inline field definitions can be added after creating the field, or use a Component Schema for reusable schemas.</p>
				</div>
				<div x-show="fieldType === 'array'" class="grid grid-cols-2 gap-3 mt-3">
					<div>
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Min Items</label>
						<input type="number" name="options.minItems" min="0" placeholder="0" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
					</div>
					<div>
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Max Items</label>
						<input type="number" name="options.maxItems" min="0" placeholder="No limit" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
					</div>
				</div>
				<div class="flex items-center gap-3 mt-2">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="options.collapsible" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Collapsible in form</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="options.defaultExpanded" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Expanded by default</span>
					</label>
				</div>
			</div>
			
			<!-- OneOf type options -->
			<div x-show="fieldType === 'oneOf'" class="space-y-3">
				<div class="bg-violet-50 dark:bg-violet-900/20 p-3 rounded-lg border border-violet-200 dark:border-violet-800">
					<p class="text-xs font-medium text-violet-700 dark:text-violet-300 mb-1">Discriminated Union</p>
					<p class="text-xs text-violet-600 dark:text-violet-400">The schema displayed depends on the value of another field (discriminator).</p>
				</div>
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Discriminator Field</label>
					<input type="text" name="options.discriminatorField" placeholder="e.g., auth-type, config-type" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm font-mono dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
					<p class="mt-1 text-xs text-gray-500">Slug of another field (typically a select) whose value determines which schema to show</p>
				</div>
				<div>
					<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Schema Mappings</label>
					<div class="space-y-2 max-h-64 overflow-y-auto">
						<template x-for="(schema, idx) in oneOfSchemas" :key="idx">
							<div class="p-3 bg-gray-50 dark:bg-neutral-900 rounded-lg border border-gray-200 dark:border-neutral-700">
								<div class="flex gap-2 items-start">
									<div class="flex-1 space-y-2">
										<input type="text" x-model="schema.key" :name="'options.schemas[' + idx + '].key'" placeholder="Discriminator value (e.g., oauth2)" class="w-full py-2 px-3 border-gray-200 rounded-lg text-sm font-mono dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
										<input type="text" x-model="schema.label" :name="'options.schemas[' + idx + '].label'" placeholder="Display label (e.g., OAuth2 Configuration)" class="w-full py-2 px-3 border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
										<select x-model="schema.componentRef" :name="'options.schemas[' + idx + '].componentRef'" class="w-full py-2 px-3 border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
											<option value="">Select component schema...</option>
											%s
										</select>
									</div>
									<button type="button" @click="removeOneOfSchema(idx)" :disabled="oneOfSchemas.length <= 1" class="p-2 text-gray-400 hover:text-red-500 disabled:opacity-30">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
									</button>
								</div>
							</div>
						</template>
					</div>
					<button type="button" @click="addOneOfSchema()" class="mt-2 w-full py-2 border-2 border-dashed border-gray-300 dark:border-neutral-600 rounded-lg text-sm text-gray-500 hover:border-violet-400 hover:text-violet-600 flex items-center justify-center gap-2">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
						Add Schema Mapping
					</button>
				</div>
				<label class="flex items-center gap-2 cursor-pointer mt-2">
					<input type="checkbox" name="options.clearOnDiscriminatorChange" value="true" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
					<span class="text-xs text-gray-600 dark:text-neutral-400">Clear data when discriminator changes</span>
				</label>
				<div class="flex items-center gap-3">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="options.collapsible" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Collapsible</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="options.defaultExpanded" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Expanded by default</span>
					</label>
				</div>
			</div>
			
			<!-- Conditional Visibility options (for all field types) -->
			<div x-show="fieldType !== ''" class="border-t border-gray-200 dark:border-neutral-700 pt-4 mt-4 space-y-3">
				<div class="flex items-center justify-between">
					<p class="text-sm font-medium text-gray-800 dark:text-neutral-200">Conditional Visibility</p>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" x-model="hasCondition" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Enable</span>
					</label>
				</div>
				<div x-show="hasCondition" x-transition class="space-y-3">
					<div>
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Condition Type</label>
						<select x-model="conditionType" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
							<option value="showWhen">Show When</option>
							<option value="hideWhen">Hide When</option>
						</select>
					</div>
					<div>
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Watch Field</label>
						<input type="text" :name="conditionType + '.field'" x-model="conditionField" placeholder="Field name to watch (e.g., status)" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm font-mono dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
					</div>
					<div>
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Operator</label>
						<select :name="conditionType + '.operator'" x-model="conditionOperator" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
							<option value="eq">Equals (eq)</option>
							<option value="ne">Not Equals (ne)</option>
							<option value="in">In List (in)</option>
							<option value="notIn">Not In List (notIn)</option>
							<option value="exists">Exists (has value)</option>
							<option value="notExists">Not Exists (empty)</option>
						</select>
					</div>
					<div x-show="!['exists', 'notExists'].includes(conditionOperator)">
						<label class="block mb-1 text-xs font-medium text-gray-600 dark:text-neutral-400">Value</label>
						<input type="text" :name="conditionType + '.value'" x-model="conditionValue" placeholder="Value to compare" class="py-2 px-3 block w-full border-gray-200 rounded-lg text-sm dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-300">
						<p class="mt-1 text-xs text-gray-500">For 'in' or 'notIn', enter comma-separated values</p>
					</div>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="options.clearWhenHidden" value="true" class="w-4 h-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700">
						<span class="text-xs text-gray-600 dark:text-neutral-400">Clear field value when hidden</span>
					</label>
				</div>
			</div>
		</div>`, contentTypeOptionsHTML, componentSchemaOptionsHTML, componentSchemaOptionsHTML)),
		),

		// Validation options section - collapsible
		Div(
			Class("mt-6 pt-6 border-t border-border"),
			g.Attr("x-data", "{ optionsOpen: false }"),
			// Collapsible header
			Button(
				Type("button"),
				g.Attr("@click", "optionsOpen = !optionsOpen"),
				Class("w-full flex items-center justify-between py-2 text-left"),
				Div(
					Class("flex items-center gap-2"),
					lucide.Settings2(Class("h-4 w-4 text-muted-foreground")),
					Span(
						Class("text-sm font-semibold text-foreground"),
						g.Text("Validation & Options"),
					),
				),
				g.El("svg",
					Class("h-4 w-4 text-muted-foreground transition-transform duration-200"),
					g.Attr(":class", "optionsOpen ? 'rotate-180' : ''"),
					g.Attr("xmlns", "http://www.w3.org/2000/svg"),
					g.Attr("viewBox", "0 0 24 24"),
					g.Attr("fill", "none"),
					g.Attr("stroke", "currentColor"),
					g.Attr("stroke-width", "2"),
					g.El("path", g.Attr("d", "m6 9 6 6 6-6")),
				),
			),
			// Collapsible content
			Div(
				g.Attr("x-show", "optionsOpen"),
				g.Attr("x-collapse", ""),
				Class("space-y-1 mt-3"),
				forgeUICheckboxOption("required", "Required", "This field must have a value", lucide.Asterisk(Class("size-3.5 text-muted-foreground"))),
				forgeUICheckboxOption("unique", "Unique", "Values must be unique across entries", lucide.Fingerprint(Class("size-3.5 text-muted-foreground"))),
				forgeUICheckboxOption("indexed", "Indexed", "Enable fast searching on this field", lucide.Search(Class("size-3.5 text-muted-foreground"))),
				forgeUICheckboxOption("localized", "Localized", "Support multiple language versions", lucide.Globe(Class("size-3.5 text-muted-foreground"))),
			),
		),

		// Form action buttons - inside form for reliable rendering
		Div(
			Class("mt-8 pt-6 border-t border-border sticky bottom-0 bg-background pb-4"),
			Div(
				Class("flex gap-3"),
				Button(
					Type("button"),
					g.Attr("@click", "$dispatch('close-drawer')"),
					Class("flex-1 inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("flex-1 inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground shadow hover:bg-primary/90 h-10 px-4 py-2"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Add Field"),
				),
			),
		),
	)
}

// drawerFieldFormAlpineData returns the Alpine.js data for the drawer field form
func drawerFieldFormAlpineData(appID, typeName string) string {
	return fmt.Sprintf(`{
		name: '',
		slug: '',
		fieldType: '',
		relationType: 'oneToMany',
		schemaSource: 'component',
		slugEdited: false,
		enumOptions: [{label: '', value: ''}],
		oneOfSchemas: [{key: '', label: '', componentRef: ''}],
		loading: false,
		
		// Conditional visibility
		hasCondition: false,
		conditionType: 'showWhen',
		conditionField: '',
		conditionOperator: 'eq',
		conditionValue: '',
		
		updateSlug() {
			if (!this.slugEdited) {
				this.slug = this.name.toLowerCase().trim()
					.replace(/[^\w\s-]/g, '')
					.replace(/[\s_-]+/g, '_')
					.replace(/^_+|_+$/g, '');
			}
		},
		addOption() {
			this.enumOptions.push({label: '', value: ''});
		},
		removeOption(idx) {
			if (this.enumOptions.length > 1) {
				this.enumOptions.splice(idx, 1);
			}
		},
		addOneOfSchema() {
			this.oneOfSchemas.push({key: '', label: '', componentRef: ''});
		},
		removeOneOfSchema(idx) {
			if (this.oneOfSchemas.length > 1) {
				this.oneOfSchemas.splice(idx, 1);
			}
		},
		isTextType() {
			return ['text', 'textarea', 'richText', 'markdown', 'email', 'url', 'phone', 'password'].includes(this.fieldType);
		},
		isNumberType() {
			return ['number', 'integer', 'float', 'decimal', 'bigInteger'].includes(this.fieldType);
		},
		isSelectionType() {
			return ['select', 'multiSelect', 'enumeration'].includes(this.fieldType);
		},
		isNestedType() {
			return ['object', 'array'].includes(this.fieldType);
		},
		async submitAddField(event) {
			const formData = new FormData(event.target);
			const data = Object.fromEntries(formData.entries());
			
			// Numeric option keys that need to be converted from string to number
			const numericOptionKeys = ['minLength', 'maxLength', 'min', 'max', 'step', 'precision', 'decimalPlaces', 'minItems', 'maxItems'];
			
			// Build options object from form data
			const options = {};
			for (const [key, value] of formData.entries()) {
				if (key.startsWith('options.')) {
					const optKey = key.replace('options.', '');
					// Skip empty values
					if (value === '' || value === null || value === undefined) {
						continue;
					}
					// Convert numeric options to numbers
					if (numericOptionKeys.includes(optKey)) {
						const numVal = parseFloat(value);
						if (!isNaN(numVal)) {
							options[optKey] = numVal;
						}
					} else {
						options[optKey] = value;
					}
				}
			}
			
			// Handle enum options if present
			if (data.fieldType === 'select' || data.fieldType === 'multiSelect' || data.fieldType === 'enumeration') {
				options.enum = this.enumOptions;
			}
			
			this.loading = true;
			try {
				await $bridge.call('cms.addField', {
					appId: '%s',
					typeName: '%s',
					title: data.name,
					name: data.slug,
					type: data.type,
					description: data.description || '',
					required: formData.has('required'),
					unique: formData.has('unique'),
					indexed: formData.has('indexed'),
					localized: formData.has('localized'),
					options: options
				});
				// Dispatch toast event for success notification
				window.dispatchEvent(new CustomEvent('toast', { detail: { type: 'success', message: 'Field added successfully' } }));
				window.location.reload();
			} catch (err) {
				const errorMsg = err.error?.message || err.message || 'Failed to add field';
				// Dispatch toast event for error notification
				window.dispatchEvent(new CustomEvent('toast', { detail: { type: 'error', message: errorMsg } }));
				this.loading = false;
			}
		}
	}`, appID, typeName)
}

// prelineCheckbox creates a checkbox with label and description using ForgeUI
func prelineCheckbox(name, labelText, description string) g.Node {
	checkboxID := "field-option-" + name
	return Div(
		Class("flex gap-3"),
		checkbox.Checkbox(
			checkbox.WithID(checkboxID),
			checkbox.WithName(name),
			checkbox.WithValue("true"),
		),
		Div(
			Class("flex-1"),
			Label(
				Class("cursor-pointer"),
				g.Attr("for", checkboxID),
				Div(
					Class("text-sm font-medium"),
					g.Text(labelText),
				),
				Div(
					Class("text-xs text-muted-foreground"),
					g.Text(description),
				),
			),
		),
	)
}

// forgeUICheckboxOption creates a styled checkbox option with icon
func forgeUICheckboxOption(name, labelText, description string, iconNode g.Node) g.Node {
	checkboxID := "field-option-" + name
	return Label(
		g.Attr("for", checkboxID),
		Class("flex items-start gap-3 p-3 rounded-lg border border-transparent hover:bg-accent/50 cursor-pointer transition-colors has-[:checked]:bg-accent has-[:checked]:border-border"),
		Div(
			Class("pt-0.5"),
			checkbox.Checkbox(
				checkbox.WithID(checkboxID),
				checkbox.WithName(name),
				checkbox.WithValue("true"),
			),
		),
		Div(
			Class("flex-1 min-w-0"),
			Div(
				Class("flex items-center gap-2"),
				iconNode,
				Span(
					Class("text-sm font-medium text-foreground"),
					g.Text(labelText),
				),
			),
			P(
				Class("text-xs text-muted-foreground mt-0.5"),
				g.Text(description),
			),
		),
	)
}

// fieldsTable renders the fields as a table
func fieldsTable(typeBase string, fields []*core.ContentFieldDTO, appID string, typeName string) g.Node {
	rows := make([]g.Node, len(fields))
	for i, field := range fields {
		rows[i] = fieldRow(typeBase, field, appID, typeName)
	}

	return DataTable(
		[]string{"Field", "Type", "Properties", "Actions"},
		rows,
	)
}

// fieldRow renders a single field row
func fieldRow(typeBase string, field *core.ContentFieldDTO, appID string, typeName string) g.Node {
	return TableRow(
		// Field name and slug
		TableCell(Div(
			Class("flex items-center gap-3"),
			Div(
				Class("flex-shrink-0 rounded bg-slate-100 p-1.5 dark:bg-gray-800"),
				fieldTypeIcon(field.Type),
			),
			Div(
				Div(Class("font-medium"), g.Text(field.Name)),
				Code(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text(field.Name),
				),
			),
		)),

		// Type
		TableCellSecondary(Badge(field.Type, "bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-300")),

		// Properties
		TableCell(Div(
			Class("flex flex-wrap gap-1"),
			g.If(field.Required, func() g.Node {
				return Badge("Required", "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400")
			}()),
			g.If(field.Unique, func() g.Node {
				return Badge("Unique", "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400")
			}()),
			g.If(field.Indexed, func() g.Node {
				return Badge("Indexed", "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400")
			}()),
			g.If(field.Localized, func() g.Node {
				return Badge("Localized", "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400")
			}()),
		)),

		// Actions - Delete button with inline modal
		TableCell(Div(
			Class("flex items-center justify-end gap-1"),
			// Delete modal component (now contains both button and modal)
			deleteFieldModal(appID, typeName, field),
		)),
	)
}

// deleteFieldModal renders a ForgeUI confirmDialog-based delete button for a field
func deleteFieldModal(appID string, typeName string, field *core.ContentFieldDTO) g.Node {
	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf(`
			$root.confirmDialog.confirm({
				title: 'Delete Field',
				message: 'Are you sure you want to delete "%s"? This will delete all data in this field. This action cannot be undone.',
				confirmText: 'Delete Field',
				cancelText: 'Cancel',
				onConfirm: async () => {
					try {
						await $bridge.call('cms.deleteField', {
							appId: '%s',
							typeName: '%s',
							fieldName: '%s'
						});
						$root.notification.success('Field deleted');
						window.location.reload();
					} catch (err) {
						const errorMsg = err.error?.message || err.message || 'Failed to delete field';
						$root.notification.error(errorMsg);
					}
				}
			})
		`, field.Title, appID, typeName, field.Name)),
		Class("inline-flex items-center justify-center size-8 rounded-lg text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"),
		g.Attr("title", "Delete Field"),
		lucide.Trash2(Class("size-4")),
	)
}

// fieldTypeIcon returns an icon for a field type
func fieldTypeIcon(fieldType string) g.Node {
	switch fieldType {
	case "text", "string":
		return lucide.Type(Class("size-4 text-slate-600 dark:text-gray-400"))
	case "number", "integer", "float":
		return lucide.Hash(Class("size-4 text-blue-600 dark:text-blue-400"))
	case "boolean":
		return lucide.ToggleLeft(Class("size-4 text-green-600 dark:text-green-400"))
	case "date", "datetime":
		return lucide.Calendar(Class("size-4 text-orange-600 dark:text-orange-400"))
	case "richtext", "markdown":
		return lucide.FileText(Class("size-4 text-violet-600 dark:text-violet-400"))
	case "media", "file", "image":
		return lucide.Image(Class("size-4 text-pink-600 dark:text-pink-400"))
	case "relation":
		return lucide.Link(Class("size-4 text-cyan-600 dark:text-cyan-400"))
	case "select", "enum":
		return lucide.List(Class("size-4 text-yellow-600 dark:text-yellow-400"))
	case "json":
		return lucide.Braces(Class("size-4 text-indigo-600 dark:text-indigo-400"))
	case "email":
		return lucide.Mail(Class("size-4 text-red-600 dark:text-red-400"))
	case "url":
		return lucide.Globe(Class("size-4 text-teal-600 dark:text-teal-400"))
	case "password":
		return lucide.Key(Class("size-4 text-gray-600 dark:text-gray-400"))
	default:
		return lucide.Circle(Class("size-4 text-slate-400 dark:text-gray-600"))
	}
}

// =============================================================================
// Settings Section
// =============================================================================

// settingsSection renders the content type settings management section using Preline-style components
func settingsSection(typeBase string, contentType *core.ContentTypeDTO) g.Node {
	settings := contentType.Settings

	return Div(
		Class("space-y-2 mt-6"),

		// General Settings Card
		CardWithHeader("General Settings", nil,
			FormEl(
				Method("POST"),
				Action(typeBase+"/settings"),
				Class("space-y-4"),

				// Name and Slug
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
					// Name
					Div(
						Label(
							For("name"),
							Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
							g.Text("Name"),
							Span(Class("text-red-500 ms-1"), g.Text("*")),
						),
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Value(contentType.Name),
							Required(),
							Class("py-2 px-3 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
						),
					),
					// Slug (read-only)
					Div(
						Label(
							For("slug"),
							Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
							g.Text("Name"),
							Span(Class("text-gray-500 ms-1 text-xs"), g.Text("(read-only)")),
						),
						Input(
							Type("text"),
							ID("slug"),
							Name("slug"),
							Value(contentType.Name),
							Disabled(),
							Class("py-2 px-3 block w-full border-gray-200 rounded-lg text-sm bg-gray-50 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-500 cursor-not-allowed"),
						),
					),
				),

				// Description
				Div(
					Label(
						For("description"),
						Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
						g.Text("Description"),
					),
					Textarea(
						ID("description"),
						Name("description"),
						Rows("3"),
						Placeholder("A brief description of this content type..."),
						Class("py-2 px-3 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400 dark:placeholder-neutral-500"),
						g.Text(contentType.Description),
					),
				),

				// Icon
				Div(
					Label(
						For("icon"),
						Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
						g.Text("Icon (emoji)"),
					),
					Input(
						Type("text"),
						ID("icon"),
						Name("icon"),
						Value(contentType.Icon),
						Placeholder("📄"),
						Class("py-2 px-3 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
					),
					P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"), g.Text("Use an emoji to visually identify this content type")),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700"),
						lucide.Save(Class("shrink-0 size-4")),
						g.Text("Save Changes"),
					),
				),
			),
		),

		// Display Settings Card
		CardWithHeader("Display Settings", nil,
			FormEl(
				Method("POST"),
				Action(typeBase+"/settings/display"),
				Class("space-y-4"),

				// Title, Description and Preview fields
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),
					// Title Field
					Div(
						Label(
							For("titleField"),
							Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
							g.Text("Title Field"),
						),
						Select(
							ID("titleField"),
							Name("titleField"),
							Class("py-2 px-3 pe-9 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
							Option(Value(""), g.Text("Select a field...")),
							g.Group(fieldSelectOptions(contentType.Fields, settings.TitleField, []string{"text", "string"})),
						),
						P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"), g.Text("Field used as the entry title in lists")),
					),
					// Description Field
					Div(
						Label(
							For("descriptionField"),
							Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
							g.Text("Description Field"),
						),
						Select(
							ID("descriptionField"),
							Name("descriptionField"),
							Class("py-2 px-3 pe-9 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
							Option(Value(""), g.Text("Select a field...")),
							g.Group(fieldSelectOptions(contentType.Fields, settings.DescriptionField, []string{"text", "string", "richtext", "markdown"})),
						),
						P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"), g.Text("Field used as the entry description")),
					),
					// Preview Field
					Div(
						Label(
							For("previewField"),
							Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
							g.Text("Preview Field"),
						),
						Select(
							ID("previewField"),
							Name("previewField"),
							Class("py-2 px-3 pe-9 block w-full border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
							Option(Value(""), g.Text("Select a field...")),
							g.Group(fieldSelectOptions(contentType.Fields, settings.PreviewField, []string{"text", "string", "number", "integer"})),
						),
						P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"), g.Text("Field shown below title for easy identification (e.g., SKU, code)")),
					),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700"),
						lucide.Save(Class("shrink-0 size-4")),
						g.Text("Save Display Settings"),
					),
				),
			),
		),

		// Features Card
		CardWithHeader("Features", nil,
			FormEl(
				Method("POST"),
				Action(typeBase+"/settings/features"),
				Class("space-y-4"),

				// Feature toggles grid
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
					featureToggle("enableRevisions", "Enable Revisions", "Track changes and allow rollback to previous versions", settings.EnableRevisions),
					featureToggle("enableDrafts", "Enable Drafts", "Allow entries to be saved as drafts before publishing", settings.EnableDrafts),
					featureToggle("enableSoftDelete", "Soft Delete", "Move deleted entries to trash instead of permanent deletion", settings.EnableSoftDelete),
					featureToggle("enableSearch", "Full-text Search", "Enable full-text search on entry content", settings.EnableSearch),
					featureToggle("enableScheduling", "Scheduled Publishing", "Allow scheduling entries for future publication", settings.EnableScheduling),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700"),
						lucide.Save(Class("shrink-0 size-4")),
						g.Text("Save Features"),
					),
				),
			),
		),

		// Limits Card
		CardWithHeader("Limits", nil,
			FormEl(
				Method("POST"),
				Action(typeBase+"/settings/limits"),
				Class("space-y-4"),

				// Max entries
				Div(
					Label(
						For("maxEntries"),
						Class("block mb-2 text-sm font-medium text-gray-800 dark:text-neutral-200"),
						g.Text("Maximum Entries"),
					),
					Input(
						Type("number"),
						ID("maxEntries"),
						Name("maxEntries"),
						Value(fmt.Sprintf("%d", settings.MaxEntries)),
						g.Attr("min", "0"),
						Placeholder("0 = unlimited"),
						Class("py-2 px-3 block w-full max-w-xs border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-400"),
					),
					P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"), g.Text("Set to 0 for unlimited entries")),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700"),
						lucide.Save(Class("shrink-0 size-4")),
						g.Text("Save Limits"),
					),
				),
			),
		),

		// Danger Zone Card
		CardWithHeader("Danger Zone", nil,
			Div(
				Class("border border-red-200 dark:border-red-800 rounded-lg p-4 bg-red-50 dark:bg-red-900/10"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						H4(
							Class("text-sm font-medium text-red-900 dark:text-red-300"),
							g.Text("Delete Content Type"),
						),
						P(
							Class("text-xs text-red-700 dark:text-red-400 mt-1"),
							g.Text("Permanently delete this content type and all its entries. This action cannot be undone."),
						),
					),
					// Delete confirmation modal (includes button)
					deleteContentTypeModal(typeBase, contentType),
				),
			),
		),
	)
}

// fieldSelectOptions generates options for field selection dropdowns
func fieldSelectOptions(fields []*core.ContentFieldDTO, selectedValue string, allowedTypes []string) []g.Node {
	var options []g.Node
	for _, field := range fields {
		// Filter by allowed types if specified
		if len(allowedTypes) > 0 {
			allowed := false
			for _, t := range allowedTypes {
				if field.Type == t {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		isSelected := field.Name == selectedValue
		option := Option(
			Value(field.Name),
			g.If(isSelected, Selected()),
			g.Text(field.Name),
		)
		options = append(options, option)
	}
	return options
}

// featureToggle renders a Preline-style feature toggle checkbox
func featureToggle(name, label, description string, checked bool) g.Node {
	checkboxID := "feature-" + name
	return Div(
		Class("flex"),
		Input(
			Type("checkbox"),
			ID(checkboxID),
			Name(name),
			Value("true"),
			g.If(checked, Checked()),
			Class("shrink-0 mt-0.5 border-gray-200 rounded text-violet-600 focus:ring-violet-500 dark:bg-neutral-800 dark:border-neutral-700 dark:checked:bg-violet-500 dark:checked:border-violet-500 dark:focus:ring-offset-neutral-800"),
		),
		Label(
			For(checkboxID),
			Class("ms-3 cursor-pointer"),
			Span(Class("block text-sm font-medium text-gray-800 dark:text-neutral-200"), g.Text(label)),
			P(Class("text-xs text-gray-500 dark:text-neutral-500"), g.Text(description)),
		),
	)
}

// deleteContentTypeModal renders a ForgeUI confirmDialog-based delete button
func deleteContentTypeModal(typeBase string, contentType *core.ContentTypeDTO) g.Node {
	// Get the redirect path by removing /types/{typeName} from typeBase
	redirectPath := typeBase[:len(typeBase)-len("/types/"+contentType.Name)]

	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf(`
			$root.confirmDialog.confirm({
				title: 'Delete Content Type',
				message: 'This will permanently delete "%s" and all its entries, fields, and revisions. This action cannot be undone.',
				confirmText: 'Delete Forever',
				cancelText: 'Cancel',
				onConfirm: async () => {
				try {
					await $bridge.call('cms.deleteContentType', {
						appId: '%s',
						name: '%s'
					});
					$root.notification.success('Content type deleted');
					window.location.href = '%s';
				} catch (err) {
					const errorMsg = err.error?.message || err.message || 'Failed to delete content type';
					$root.notification.error(errorMsg);
				}
				}
			})
		`, contentType.Title, contentType.AppID, contentType.Name, redirectPath)),
		Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-red-600 text-white hover:bg-red-700 focus:outline-none focus:bg-red-700 transition-colors"),
		lucide.Trash2(Class("shrink-0 size-4")),
		g.Text("Delete"),
	)
}

// =============================================================================
// API Section
// =============================================================================

// apiSection renders the API documentation section
func apiSection(apiBase string, contentType *core.ContentTypeDTO) g.Node {
	return Div(
		Class("space-y-2 mt-6"),

		// Quick Reference Card
		CardWithHeader("API Endpoints", []g.Node{
			Span(Class("py-1 px-2.5 inline-flex items-center gap-x-1 text-xs font-medium bg-violet-100 text-violet-800 rounded-full dark:bg-violet-900 dark:text-violet-300"), g.Text("REST API")),
		},
			Div(
				Class("space-y-4"),

				// Base URL
				Div(
					Class("bg-slate-100 dark:bg-gray-800 rounded-lg p-4"),
					P(Class("text-xs font-medium text-slate-500 dark:text-gray-400 mb-2"), g.Text("Base URL")),
					Code(
						Class("text-sm text-violet-600 dark:text-violet-400 font-mono"),
						g.Text(apiBase),
					),
				),

				// Endpoints Table
				Div(
					Class("overflow-x-auto"),
					Table(
						Class("table table-zebra w-full"),
						THead(
							Tr(
								Th(g.Text("Method")),
								Th(g.Text("Endpoint")),
								Th(g.Text("Description")),
							),
						),
						TBody(
							apiEndpointRow("GET", "/", "List all entries (with pagination, filters, sorting)"),
							apiEndpointRow("POST", "/", "Create a new entry"),
							apiEndpointRow("GET", "/:id", "Get a single entry by ID"),
							apiEndpointRow("PUT", "/:id", "Update an entry"),
							apiEndpointRow("DELETE", "/:id", "Delete an entry"),
							apiEndpointRow("POST", "/:id/publish", "Publish a draft entry"),
							apiEndpointRow("POST", "/:id/unpublish", "Unpublish an entry"),
							apiEndpointRow("POST", "/query", "Advanced query with JSON body"),
							apiEndpointRow("GET", "/:id/revisions", "List entry revisions"),
							apiEndpointRow("POST", "/:id/rollback/:version", "Rollback to a specific version"),
						),
					),
				),
			),
		),

		// Query Parameters Card
		CardWithHeader("Query Parameters", []g.Node{
			Span(Class("py-1 px-2.5 inline-flex items-center gap-x-1 text-xs font-medium bg-gray-100 text-gray-800 rounded-full dark:bg-neutral-700 dark:text-neutral-300"), g.Text("GET requests")),
		},
			Div(
				Class("space-y-4"),

				// Pagination
				Div(
					H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2"), g.Text("Pagination")),
					Code(Class("text-xs bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded block mb-2 font-mono"),
						g.Text("?page=1&pageSize=25")),
					P(Class("text-xs text-slate-600 dark:text-gray-400"), g.Text("Default page size: 25, Max: 100")),
				),

				Div(Class("divider")),

				// Filtering
				Div(
					H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2"), g.Text("Filtering")),
					Code(Class("text-xs bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded block mb-2 font-mono"),
						g.Text("?filter[fieldName]=op.value")),
					Div(
						Class("mt-2 space-y-1"),
						filterOpExample("eq", "Equals", "filter[_meta.status]=eq.published"),
						filterOpExample("ne", "Not equals", "filter[_meta.status]=ne.draft"),
						filterOpExample("gt", "Greater than", "filter[price]=gt.100"),
						filterOpExample("gte", "Greater or equal", "filter[price]=gte.100"),
						filterOpExample("lt", "Less than", "filter[price]=lt.50"),
						filterOpExample("lte", "Less or equal", "filter[price]=lte.50"),
						filterOpExample("like", "Contains (case-sensitive)", "filter[title]=like.hello"),
						filterOpExample("ilike", "Contains (case-insensitive)", "filter[title]=ilike.hello"),
						filterOpExample("in", "In list", "filter[_meta.status]=in.(draft,published)"),
						filterOpExample("nin", "Not in list", "filter[_meta.status]=nin.(archived)"),
						filterOpExample("null", "Is null", "filter[_meta.deletedAt]=null.true"),
					),
				),

				Div(Class("divider")),

				// Sorting
				Div(
					H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2"), g.Text("Sorting")),
					Code(Class("text-xs bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded block mb-2 font-mono"),
						g.Text("?sort=-createdAt,title")),
					P(Class("text-xs text-slate-600 dark:text-gray-400"), g.Text("Prefix with - for descending order. Comma-separate multiple fields.")),
				),

				Div(Class("divider")),

				// Field Selection
				Div(
					H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2"), g.Text("Field Selection")),
					Code(Class("text-xs bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded block mb-2 font-mono"),
						g.Text("?fields=id,title,status")),
					P(Class("text-xs text-slate-600 dark:text-gray-400"), g.Text("Select specific fields to return. Reduces response size.")),
				),

				Div(Class("divider")),

				// Population
				Div(
					H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2"), g.Text("Population (Relations)")),
					Code(Class("text-xs bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded block mb-2 font-mono"),
						g.Text("?populate=author,category")),
					P(Class("text-xs text-slate-600 dark:text-gray-400"), g.Text("Eager-load related entries. Comma-separate multiple relations.")),
				),
			),
		),

		// Schema Card
		CardWithHeader("Entry Schema", []g.Node{
			Span(Class("py-1 px-2.5 inline-flex items-center gap-x-1 text-xs font-medium bg-cyan-100 text-cyan-800 rounded-full dark:bg-cyan-900 dark:text-cyan-300"), g.Text(fmt.Sprintf("%d fields", len(contentType.Fields)))),
		},
			Div(
				Class("overflow-x-auto"),
				Table(
					Class("table table-zebra w-full"),
					THead(
						Tr(
							Th(g.Text("Field")),
							Th(g.Text("Type")),
							Th(g.Text("Required")),
							Th(g.Text("Description")),
						),
					),
					TBody(
						g.Group(func() []g.Node {
							rows := make([]g.Node, len(contentType.Fields))
							for i, field := range contentType.Fields {
								rows[i] = schemaFieldRow(field)
							}
							return rows
						}()),
					),
				),
			),
		),

		// Example Request/Response Card
		apiExamplesCard(apiBase, contentType),
	)
}

// apiEndpointRow renders a row in the API endpoints table
func apiEndpointRow(method, endpoint, description string) g.Node {
	methodColor := "bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-400"
	switch method {
	case "GET":
		methodColor = "bg-teal-100 text-teal-800 dark:bg-teal-900/50 dark:text-teal-400"
	case "POST":
		methodColor = "bg-violet-100 text-violet-800 dark:bg-violet-900/50 dark:text-violet-400"
	case "PUT":
		methodColor = "bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-400"
	case "DELETE":
		methodColor = "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-400"
	}

	return Tr(
		Td(Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-mono font-medium rounded "+methodColor), g.Text(method))),
		Td(Code(Class("text-xs font-mono"), g.Text(endpoint))),
		Td(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text(description)),
	)
}

// filterOpExample renders a filter operator example
func filterOpExample(op, name, example string) g.Node {
	return Div(
		Class("flex items-center gap-2 text-xs"),
		Code(Class("bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300 px-1.5 py-0.5 rounded font-mono"), g.Text(op)),
		Span(Class("text-slate-600 dark:text-gray-400"), g.Text(name+":")),
		Code(Class("text-slate-500 dark:text-gray-500 font-mono"), g.Text(example)),
	)
}

// schemaFieldRow renders a field row in the schema table
func schemaFieldRow(field *core.ContentFieldDTO) g.Node {
	return Tr(
		Td(
			Div(
				Class("flex items-center gap-2"),
				fieldTypeIcon(field.Type),
				Code(Class("text-xs font-mono font-medium"), g.Text(field.Name)),
			),
		),
		Td(Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-gray-100 text-gray-600 rounded dark:bg-neutral-700 dark:text-neutral-400"), g.Text(field.Type))),
		Td(
			g.If(field.Required, func() g.Node {
				return Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-red-100 text-red-800 rounded dark:bg-red-900/50 dark:text-red-400"), g.Text("required"))
			}()),
			g.If(!field.Required, func() g.Node {
				return Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-gray-100 text-gray-600 rounded dark:bg-neutral-700 dark:text-neutral-400"), g.Text("optional"))
			}()),
		),
		Td(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text(field.Description)),
	)
}

// apiExamplesCard renders example API requests
func apiExamplesCard(apiBase string, contentType *core.ContentTypeDTO) g.Node {
	// Generate sample entry data
	sampleData := "{\n"
	for i, field := range contentType.Fields {
		if i > 0 {
			sampleData += ",\n"
		}
		sampleData += fmt.Sprintf(`    "%s": `, field.Name)
		switch field.Type {
		case "text", "string", "email", "url", "richtext", "markdown":
			sampleData += `"sample value"`
		case "number", "integer", "float":
			sampleData += "123"
		case "boolean":
			sampleData += "true"
		case "date", "datetime":
			sampleData += `"2025-01-01T00:00:00Z"`
		case "relation":
			sampleData += `"related-entry-id"`
		case "json":
			sampleData += `{}`
		case "select", "enum":
			sampleData += `"option1"`
		default:
			sampleData += `"value"`
		}
	}
	sampleData += "\n}"

	return CardWithHeader("Example Requests", nil,
		Div(
			Class("space-y-4"),

			// Create Entry Example
			Div(
				H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2 flex items-center gap-2"),
					Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-violet-100 text-violet-800 rounded dark:bg-violet-900 dark:text-violet-300"), g.Text("POST")),
					g.Text("Create Entry"),
				),
				Pre(
					Class("bg-slate-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto"),
					Code(
						Class("text-sm text-emerald-400 font-mono whitespace-pre"),
						g.Text(fmt.Sprintf(`curl -X POST %s \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '%s'`, apiBase, sampleData)),
					),
				),
			),

			// List with Filters Example
			Div(
				H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2 flex items-center gap-2"),
					Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-teal-100 text-teal-800 rounded dark:bg-teal-900 dark:text-teal-300"), g.Text("GET")),
					g.Text("List with Filters"),
				),
				Pre(
					Class("bg-slate-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto"),
					Code(
						Class("text-sm text-emerald-400 font-mono whitespace-pre"),
						g.Text(fmt.Sprintf(`curl "%s?filter[_meta.status]=eq.published&sort=-createdAt&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"`, apiBase)),
					),
				),
			),

			// Advanced Query Example
			Div(
				H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2 flex items-center gap-2"),
					Span(Class("py-0.5 px-2 inline-flex items-center text-xs font-medium bg-violet-100 text-violet-800 rounded dark:bg-violet-900 dark:text-violet-300"), g.Text("POST")),
					g.Text("Advanced Query"),
				),
				Pre(
					Class("bg-slate-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto"),
					Code(
						Class("text-sm text-emerald-400 font-mono whitespace-pre"),
						g.Text(fmt.Sprintf(`curl -X POST %s/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "filter": {
      "$and": [
        { "_meta.status": { "$eq": "published" } },
        { "$or": [
          { "featured": { "$eq": true } },
          { "views": { "$gte": 1000 } }
        ]}
      ]
    },
    "sort": ["-createdAt"],
    "page": 1,
    "pageSize": 25
  }'`, apiBase)),
					),
				),
			),
		),
	)
}

// =============================================================================
// Playground Section
// =============================================================================

// playgroundSection renders the API playground with Monaco Editor
func playgroundSection(apiBase string, contentType *core.ContentTypeDTO, appID string, environmentID string) g.Node {
	// Default query example
	defaultQuery := `{
  "filter": {
    "_meta.status": { "$eq": "published" }
  },
  "sort": ["-createdAt"],
  "page": 1,
  "pageSize": 10
}`

	// Pass app ID and environment ID directly from server-side
	return Div(
		Class("space-y-6 mt-6"),
		g.Attr("x-data", fmt.Sprintf(`{
			query: %s,
			response: null,
			loading: false,
			error: null,
			viewMode: 'table',
			endpoint: '%s',
			method: 'GET',
			appId: '%s',
			envId: '%s',
			
			async executeQuery() {
				this.loading = true;
				this.error = null;
				this.response = null;
				
				try {
					let url = this.endpoint;
					
					let headers = {
							'Content-Type': 'application/json'
					};
					
					// Add app context headers
					if (this.appId) {
						headers['X-App-ID'] = this.appId;
					}
					if (this.envId) {
						headers['X-Environment-ID'] = this.envId;
					}
					
					let options = {
						method: this.method,
						headers: headers
					};
					
					if (this.method === 'POST') {
						url = this.endpoint + '/query';
						options.body = this.query;
					} else if (this.method === 'GET') {
						// Parse query params from the query JSON
						try {
							const queryObj = JSON.parse(this.query);
							const params = new URLSearchParams();
							
							// Handle filters
							if (queryObj.filter) {
								for (const [field, ops] of Object.entries(queryObj.filter)) {
									if (typeof ops === 'object') {
										for (const [op, val] of Object.entries(ops)) {
											const opMap = { '$eq': 'eq', '$ne': 'ne', '$gt': 'gt', '$gte': 'gte', '$lt': 'lt', '$lte': 'lte', '$like': 'like', '$ilike': 'ilike' };
											params.append('filter[' + field + ']', (opMap[op] || op.replace('$','')) + '.' + val);
										}
									}
								}
							}
							
							// Handle pagination
							if (queryObj.page) params.append('page', queryObj.page);
							if (queryObj.pageSize) params.append('pageSize', queryObj.pageSize);
							
							// Handle sorting
							if (queryObj.sort && Array.isArray(queryObj.sort)) {
								params.append('sort', queryObj.sort.join(','));
							}
							
							const queryString = params.toString();
							if (queryString) url += '?' + queryString;
						} catch(e) {
							// If query isn't valid JSON for GET, just use the endpoint
						}
					}
					
					const res = await fetch(url, options);
					const data = await res.json();
					
					if (!res.ok) {
						throw new Error(data.message || data.error || 'Request failed');
					}
					
					this.response = data;
				} catch (e) {
					this.error = e.message;
				} finally {
					this.loading = false;
				}
			},
			
			formatJSON(obj) {
				return JSON.stringify(obj, null, 2);
			},
			
			getEntries() {
				if (!this.response) return [];
				return this.response.entries || this.response.items || this.response.data || (Array.isArray(this.response) ? this.response : []);
			},
			
			getFields() {
				const entries = this.getEntries();
				if (entries.length === 0) return [];
				const fields = new Set();
				entries.forEach(entry => {
					Object.keys(entry.data || entry).forEach(k => fields.add(k));
				});
				return Array.from(fields).slice(0, 10); // Limit to 10 fields
			}
		}`, "`"+defaultQuery+"`", apiBase, appID, environmentID)),

		// Query Builder Card
		CardWithHeader("Query Builder", []g.Node{
			// Method selector
			Div(
				Class("flex items-center gap-2"),
				Select(
					g.Attr("x-model", "method"),
					Class("py-1.5 px-3 border border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-400"),
					Option(Value("GET"), g.Text("GET")),
					Option(Value("POST"), g.Text("POST")),
				),
				Code(
					Class("text-xs font-mono bg-slate-100 dark:bg-gray-800 px-2 py-1 rounded"),
					g.Attr("x-text", "method === 'POST' ? endpoint + '/query' : endpoint"),
				),
			),
		},
			Div(
				Class("space-y-4"),

				// Monaco Editor container
				Div(
					ID("monaco-editor-container"),
					Class("w-full border border-slate-200 dark:border-gray-700 rounded-lg overflow-hidden"),
					monacoEditor(defaultQuery),
				),

				// Execute button
				Div(
					Class("flex items-center gap-4"),
					Button(
						Type("button"),
						g.Attr("@click", "executeQuery()"),
						g.Attr(":disabled", "loading"),
						Class("py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none dark:focus:ring-offset-neutral-800"),
						lucide.Play(Class("size-4"), g.Attr("x-show", "!loading")),
						Span(g.Attr("x-show", "loading"), g.Text("Executing...")),
						Span(g.Attr("x-show", "!loading"), g.Text("Execute Query")),
					),
					// Quick actions
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							filter: { "_meta.status": { "$eq": "published" } },
							sort: ["-createdAt"],
							page: 1,
							pageSize: 10
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("py-1.5 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg text-gray-700 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-300 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:ring-neutral-700"),
						g.Text("Published Entries"),
					),
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							filter: { "_meta.status": { "$eq": "draft" } },
							sort: ["-updatedAt"],
							page: 1,
							pageSize: 10
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("py-1.5 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg text-gray-700 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-300 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:ring-neutral-700"),
						g.Text("Drafts"),
					),
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							sort: ["-createdAt"],
							page: 1,
							pageSize: 50
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("py-1.5 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg text-gray-700 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-300 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:ring-neutral-700"),
						g.Text("All Entries"),
					),
				),
			),
		),

		// Results Card
		Div(
			g.Attr("x-show", "response || error"),
			g.Attr("x-cloak", ""),
			CardWithHeader("Results", []g.Node{
				// View mode toggle
				Div(
					Class("inline-flex rounded-lg shadow-sm"),
					Button(
						Type("button"),
						g.Attr("@click", "viewMode = 'table'"),
						g.Attr(":class", "viewMode === 'table' ? 'bg-violet-100 text-violet-700 dark:bg-violet-900 dark:text-violet-300' : 'bg-white text-gray-700 dark:bg-neutral-800 dark:text-neutral-300'"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 -ms-px first:rounded-s-lg first:ms-0 last:rounded-e-lg text-sm font-medium border border-gray-200 hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-violet-500 dark:border-neutral-700 dark:hover:bg-neutral-700"),
						lucide.Table(Class("size-4")),
						g.Text("Table"),
					),
					Button(
						Type("button"),
						g.Attr("@click", "viewMode = 'json'"),
						g.Attr(":class", "viewMode === 'json' ? 'bg-violet-100 text-violet-700 dark:bg-violet-900 dark:text-violet-300' : 'bg-white text-gray-700 dark:bg-neutral-800 dark:text-neutral-300'"),
						Class("py-2 px-3 inline-flex items-center gap-x-2 -ms-px first:rounded-s-lg first:ms-0 last:rounded-e-lg text-sm font-medium border border-gray-200 hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-violet-500 dark:border-neutral-700 dark:hover:bg-neutral-700"),
						lucide.Braces(Class("size-4")),
						g.Text("JSON"),
					),
				),
			},
				Div(
					// Error state
					Div(
						g.Attr("x-show", "error"),
						Class("p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg"),
						Div(
							Class("flex items-center gap-2 text-red-600 dark:text-red-400"),
							lucide.CircleX(Class("size-5")),
							Span(Class("font-medium"), g.Text("Error")),
						),
						P(
							Class("mt-2 text-sm text-red-700 dark:text-red-300"),
							g.Attr("x-text", "error"),
						),
					),

					// Success state - Table view
					Div(
						g.Attr("x-show", "response && viewMode === 'table'"),

						// Summary stats
						Div(
							Class("py-3 px-4 flex items-center gap-4 text-sm text-gray-600 dark:text-neutral-400 border-b border-gray-200 dark:border-neutral-700"),
							Span(
								g.Attr("x-show", "response?.total !== undefined"),
								Class("inline-flex items-center gap-1.5"),
								lucide.Database(Class("size-4")),
								Span(g.Attr("x-text", "'Total: ' + (response?.total || 0) + ' entries'")),
							),
							Span(
								g.Attr("x-show", "response?.page !== undefined"),
								Class("inline-flex items-center gap-1.5"),
								lucide.FileText(Class("size-4")),
								Span(g.Attr("x-text", "'Page: ' + (response?.page || 1)")),
							),
						),

						// Table with Preline styling
						Div(
							Class("overflow-x-auto"),
							Table(
								Class("min-w-full divide-y divide-gray-200 dark:divide-neutral-700"),
								THead(
									Class("bg-gray-50 dark:bg-neutral-800"),
									Tr(
										Th(
											Class("px-6 py-3 text-start text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-neutral-400"),
											g.Text("ID"),
										),
										Th(
											Class("px-6 py-3 text-start text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-neutral-400"),
											g.Text("Status"),
										),
										g.El("template",
											g.Attr("x-for", "field in getFields()"),
											g.El("th",
												g.Attr("class", "px-6 py-3 text-start text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-neutral-400"),
												g.Attr("x-text", "field"),
											),
										),
										Th(
											Class("px-6 py-3 text-start text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-neutral-400"),
											g.Text("Created"),
										),
									),
								),
								TBody(
									Class("divide-y divide-gray-200 dark:divide-neutral-700"),
									g.El("template",
										g.Attr("x-for", "(entry, index) in getEntries()"),
										g.Attr(":key", "entry.id"),
										g.El("tr",
											g.Attr(":class", "index % 2 === 0 ? 'bg-white dark:bg-neutral-900' : 'bg-gray-50 dark:bg-neutral-800'"),
											g.Attr("class", "hover:bg-gray-100 dark:hover:bg-neutral-700 transition-colors"),
											g.El("td",
												g.Attr("class", "px-6 py-4 whitespace-nowrap text-sm"),
												g.El("code",
													g.Attr("class", "text-xs font-mono text-gray-900 dark:text-neutral-200 bg-gray-100 dark:bg-neutral-700 px-1.5 py-0.5 rounded"),
													g.Attr("x-text", "entry.id?.substring(0,8) + '...'"),
												),
											),
											g.El("td",
												g.Attr("class", "px-6 py-4 whitespace-nowrap"),
												g.El("span",
													g.Attr("class", "py-1 px-2 inline-flex items-center text-xs font-medium rounded-full"),
													g.Attr(":class", `{
														'bg-teal-100 text-teal-800 dark:bg-teal-900/50 dark:text-teal-400': entry.status === 'published',
														'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-400': entry.status === 'draft',
														'bg-gray-100 text-gray-600 dark:bg-neutral-700 dark:text-neutral-400': entry.status === 'archived'
												}`),
													g.Attr("x-text", "entry.status"),
												),
											),
											g.El("template",
												g.Attr("x-for", "field in getFields()"),
												g.El("td",
													g.Attr("class", "px-6 py-4 whitespace-nowrap text-sm text-gray-800 dark:text-neutral-200 max-w-xs truncate"),
													g.Attr("x-text", "typeof (entry.data?.[field] || entry[field]) === 'object' ? JSON.stringify(entry.data?.[field] || entry[field]) : (entry.data?.[field] || entry[field] || '-')"),
												),
											),
											g.El("td",
												g.Attr("class", "px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-neutral-400"),
												g.Attr("x-text", "entry.createdAt ? new Date(entry.createdAt).toLocaleDateString() : '-'"),
											),
										),
									),
								),
							),
						),

						// Empty state
						Div(
							g.Attr("x-show", "getEntries().length === 0"),
							Class("flex flex-col items-center justify-center py-12 text-center"),
							Div(
								Class("rounded-full bg-gray-100 dark:bg-neutral-800 p-3 mb-4"),
								lucide.Inbox(Class("size-8 text-gray-400 dark:text-neutral-500")),
							),
							P(Class("text-sm font-medium text-gray-900 dark:text-neutral-200"), g.Text("No entries found")),
							P(Class("text-sm text-gray-500 dark:text-neutral-400 mt-1"), g.Text("Try adjusting your query filters")),
						),
					),

					// Success state - JSON view
					Div(
						g.Attr("x-show", "response && viewMode === 'json'"),
						Pre(
							Class("bg-slate-900 dark:bg-gray-950 rounded-lg p-4 overflow-auto max-h-96"),
							Code(
								Class("text-sm text-emerald-400 font-mono whitespace-pre"),
								g.Attr("x-text", "formatJSON(response)"),
							),
						),
					),
				),
			),
		),

		// Field Reference Card
		CardWithHeader("Field Reference", nil,
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-2"),
				g.Group(func() []g.Node {
					nodes := make([]g.Node, len(contentType.Fields))
					for i, field := range contentType.Fields {
						nodes[i] = Div(
							Class("flex items-center gap-2 px-3 py-2 bg-slate-50 dark:bg-gray-800 rounded text-sm"),
							fieldTypeIcon(field.Type),
							Code(Class("font-mono text-xs"), g.Text(field.Name)),
							Span(Class("text-slate-400 text-xs"), g.Text("("+field.Type+")")),
						)
					}
					return nodes
				}()),
			),
		),
	)
}

// monacoEditor renders the Monaco Editor component
func monacoEditor(defaultValue string) g.Node {
	// Build the x-data attribute
	xData := fmt.Sprintf(`{
		monacoContent: %s,
		monacoLanguage: 'json',
		monacoLoader: true,
		monacoId: $id('monaco-editor'),
		
		monacoEditorAddLoaderScriptToHead() {
			if (document.querySelector('script[src*="monaco-editor"]')) return;
			const script = document.createElement('script');
			script.src = 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.45.0/min/vs/loader.min.js';
			document.head.appendChild(script);
		}
	}`, "`"+defaultValue+"`")

	// Build x-init - need to avoid backticks in Go raw strings
	xInit := `
		monacoEditorAddLoaderScriptToHead();
		
		const checkLoader = setInterval(() => {
			if (typeof require !== 'undefined' && typeof require.config === 'function') {
				require.config({ 
					paths: { 'vs': 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.45.0/min/vs' }
				});
				
				const workerCode = "self.MonacoEnvironment = { baseUrl: 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.45.0/min' }; importScripts('https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.45.0/min/vs/base/worker/workerMain.min.js');";
				const proxy = URL.createObjectURL(new Blob([workerCode], { type: 'text/javascript' }));
				
				window.MonacoEnvironment = { getWorkerUrl: () => proxy };
				
				require(['vs/editor/editor.main'], function() {
					const editor = monaco.editor.create($refs.monacoEl, {
						value: monacoContent,
						language: monacoLanguage,
						theme: document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs',
						minimap: { enabled: false },
						fontSize: 14,
						lineNumbers: 'on',
						scrollBeyondLastLine: false,
						automaticLayout: true,
						tabSize: 2,
						formatOnPaste: true,
						formatOnType: true,
					});
					
					window.monacoEditor = editor;
					
					editor.onDidChangeModelContent(() => {
						query = editor.getValue();
					});
					
					// Update theme on dark mode toggle
					const observer = new MutationObserver(() => {
						monaco.editor.setTheme(document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs');
					});
					observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
					
					monacoLoader = false;
				});
				
				clearInterval(checkLoader);
			}
		}, 100);
	`

	return Div(
		g.Attr("x-data", xData),
		g.Attr("x-init", xInit),
		g.Attr(":id", "monacoId"),
		Class("relative"),

		// Loading state
		Div(
			g.Attr("x-show", "monacoLoader"),
			Class("absolute inset-0 z-20 flex items-center justify-center bg-slate-100 dark:bg-gray-800"),
			Div(
				Class("flex items-center gap-2 text-slate-500 dark:text-gray-400"),
				Span(Class("loading loading-spinner loading-sm")),
				Span(g.Text("Loading editor...")),
			),
		),

		// Editor element
		Div(
			g.Attr("x-ref", "monacoEl"),
			Class("w-full h-64"),
		),
	)
}

// =============================================================================
// Add Field Page
// =============================================================================

// AddFieldPage renders the add field form
func AddFieldPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	allContentTypes []*core.ContentTypeSummaryDTO,
	err string,
) g.Node {
	return fieldFormPage(currentApp, basePath, contentType, allContentTypes, nil, err, false)
}

// EditFieldPage renders the edit field form
func EditFieldPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	allContentTypes []*core.ContentTypeSummaryDTO,
	field *core.ContentFieldDTO,
	err string,
) g.Node {
	return fieldFormPage(currentApp, basePath, contentType, allContentTypes, field, err, true)
}

// fieldFormPage renders the add/edit field form
func fieldFormPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	allContentTypes []*core.ContentTypeSummaryDTO,
	field *core.ContentFieldDTO,
	err string,
	isEdit bool,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Name

	// Get all field types
	fieldTypesByCategory := core.GetFieldTypesByCategory()

	// Determine action URL and title
	actionURL := typeBase + "/fields/create"
	pageTitle := "Add Field"
	pageDesc := fmt.Sprintf("Configure a new field for %s", contentType.Name)
	submitText := "Add Field"
	if isEdit && field != nil {
		actionURL = typeBase + "/fields/" + field.Name + "/update"
		pageTitle = "Edit Field"
		pageDesc = fmt.Sprintf("Update %s field configuration", field.Name)
		submitText = "Save Changes"
	}

	// Build content types options for relations
	contentTypeOptions := make([]selectOption, 0, len(allContentTypes)+1)
	contentTypeOptions = append(contentTypeOptions, selectOption{Value: "", Label: "Select content type..."})
	for _, ct := range allContentTypes {
		if ct.ID != contentType.ID {
			contentTypeOptions = append(contentTypeOptions, selectOption{Value: ct.Name, Label: ct.Name})
		}
	}

	return Div(
		Class("space-y-2 max-w-4xl mx-auto"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: typeBase},
			BreadcrumbItem{Label: pageTitle, Href: ""},
		),

		// Header with icon
		Div(
			Class("flex items-center gap-4"),
			Div(
				Class("p-3 bg-violet-100 dark:bg-violet-900/30 rounded-xl"),
				g.If(isEdit, lucide.Pencil(Class("size-6 text-violet-600 dark:text-violet-400"))),
				g.If(!isEdit, lucide.Plus(Class("size-6 text-violet-600 dark:text-violet-400"))),
			),
			Div(
				H1(Class("text-2xl font-semibold text-gray-900 dark:text-white"), g.Text(pageTitle)),
				P(Class("text-sm text-gray-500 dark:text-gray-400"), g.Text(pageDesc)),
			),
		),

		// Error message
		g.If(err != "", func() g.Node {
			return Div(
				Class("p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl dark:bg-red-900/20 dark:border-red-800 dark:text-red-400 flex items-center gap-3"),
				lucide.CircleAlert(Class("size-5 flex-shrink-0")),
				g.Text(err),
			)
		}()),

		// Form
		FormEl(
			Method("POST"),
			Action(actionURL),
			g.Attr("x-data", fieldBuilderAlpineData(field)),

			// Main content grid
			Div(
				Class("grid grid-cols-1 lg:grid-cols-3 gap-6"),

				// Left column - Main settings
				Div(
					Class("lg:col-span-2 space-y-6"),

					// Basic Info Card
					Div(
						Class("bg-white dark:bg-neutral-900 rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden"),
						Div(
							Class("px-5 py-4 border-b border-gray-200 dark:border-neutral-700 bg-gray-50 dark:bg-neutral-800"),
							H3(Class("text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"),
								lucide.Settings2(Class("size-4 text-gray-500")),
								g.Text("Basic Information"),
							),
						),
						Div(
							Class("p-5 space-y-5"),

							// Name & Slug row
							Div(
								Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
								fieldInput("name", "Field Name", "text", "e.g., Title, Author", true, "name", "@input", "updateSlug()"),
								fieldInput("slug", "API Identifier", "text", "e.g., title, author", true, "slug", "@input", "slugManuallyEdited = true"),
							),

							// Type selector with icons
							Div(
								Label(
									For("type"),
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
									g.Text("Field Type"),
									Span(Class("text-red-500 ml-1"), g.Text("*")),
								),
								Select(
									ID("type"),
									Name("type"),
									Required(),
									g.Attr("x-model", "type"),
									Class("block w-full px-4 py-3 text-sm border border-gray-200 rounded-xl bg-white dark:bg-neutral-800 dark:border-neutral-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"),
									Option(Value(""), g.Text("Select a field type...")),
									g.Group(func() []g.Node {
										categories := []string{"text", "number", "date", "selection", "relation", "media", "nested", "advanced"}
										categoryNames := map[string]string{
											"text": "📝 Text Fields", "number": "🔢 Number Fields", "date": "📅 Date & Time",
											"selection": "☑️ Selection Fields", "relation": "🔗 Relations", "media": "🖼️ Media",
											"nested": "📦 Nested Fields", "advanced": "⚙️ Advanced",
										}
										var groups []g.Node
										for _, cat := range categories {
											types, ok := fieldTypesByCategory[cat]
											if !ok || len(types) == 0 {
												continue
											}
											options := make([]g.Node, len(types))
											for i, ft := range types {
												options[i] = Option(Value(ft.Type.String()), g.Text(ft.Name))
											}
											groups = append(groups, OptGroup(g.Attr("label", categoryNames[cat]), g.Group(options)))
										}
										return groups
									}()),
								),
								// Type description
								Div(
									g.Attr("x-show", "type !== ''"),
									g.Attr("x-transition", ""),
									Class("mt-2 text-xs text-gray-500 dark:text-gray-400"),
									g.Attr("x-text", "getTypeDescription()"),
								),
							),

							// Description
							Div(
								Label(For("description"), Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Description")),
								Textarea(
									ID("description"), Name("description"), Rows("2"),
									Placeholder("Help text for content editors..."),
									Class("block w-full px-4 py-3 text-sm border border-gray-200 rounded-xl bg-white dark:bg-neutral-800 dark:border-neutral-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all resize-none"),
								),
							),
						),
					),

					// Type-specific configuration card
					Div(
						g.Attr("x-show", "type !== ''"),
						g.Attr("x-transition:enter", "transition ease-out duration-200"),
						g.Attr("x-transition:enter-start", "opacity-0 translate-y-2"),
						g.Attr("x-transition:enter-end", "opacity-100 translate-y-0"),
						Class("bg-white dark:bg-neutral-900 rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden"),
						Div(
							Class("px-5 py-4 border-b border-gray-200 dark:border-neutral-700 bg-gray-50 dark:bg-neutral-800"),
							H3(Class("text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"),
								lucide.SlidersHorizontal(Class("size-4 text-gray-500")),
								g.Text("Field Configuration"),
							),
						),
						Div(
							Class("p-5"),

							// Text type options
							Div(
								g.Attr("x-show", "isTextType()"),
								Class("space-y-5"),
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									fieldInput("options.default", "Default Value", "text", "Enter default text", false, "", "", ""),
									fieldInput("options.regex", "Validation Pattern", "text", "e.g., ^[A-Za-z]+$", false, "", "", ""),
								),
								Div(
									Class("grid grid-cols-2 gap-4"),
									numberInputPreline("options.minLength", "Min Length", 0, 10000, 0),
									numberInputPreline("options.maxLength", "Max Length", 0, 10000, 0),
								),
								// Rich text specific
								Div(
									g.Attr("x-show", "type === 'richText' || type === 'markdown'"),
									Class("grid grid-cols-2 gap-4 pt-4 border-t border-gray-100 dark:border-neutral-800"),
									numberInputPreline("options.maxWords", "Max Words", 0, 100000, 0),
									switchField("options.allowHtml", "Allow Raw HTML"),
								),
							),

							// Number type options
							Div(
								g.Attr("x-show", "isNumberType()"),
								Class("space-y-5"),
								Div(
									Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
									numberInputPreline("options.min", "Minimum", -1000000, 1000000, 0),
									numberInputPreline("options.max", "Maximum", -1000000, 1000000, 0),
									numberInputPreline("options.step", "Step", 0, 1000, 1),
									fieldInput("options.defaultNumber", "Default", "number", "0", false, "", "", ""),
								),
								Div(
									g.Attr("x-show", "type === 'decimal'"),
									Class("grid grid-cols-2 gap-4 pt-4 border-t border-gray-100 dark:border-neutral-800"),
									numberInputPreline("options.precision", "Precision", 0, 20, 2),
									numberInputPreline("options.scale", "Scale", 0, 10, 2),
								),
							),

							// Date type options
							Div(
								g.Attr("x-show", "isDateType()"),
								Class("space-y-5"),
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									fieldInput("options.minDate", "Minimum Date", "date", "", false, "", "", ""),
									fieldInput("options.maxDate", "Maximum Date", "date", "", false, "", "", ""),
								),
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									selectField("options.dateFormat", "Display Format", []selectOption{
										{Value: "", Label: "Default"},
										{Value: "YYYY-MM-DD", Label: "2024-01-15"},
										{Value: "DD/MM/YYYY", Label: "15/01/2024"},
										{Value: "MM/DD/YYYY", Label: "01/15/2024"},
										{Value: "MMMM D, YYYY", Label: "January 15, 2024"},
									}),
									switchField("options.includeTime", "Include Time"),
								),
							),

							// Selection type options (enum builder) - using raw HTML for Alpine template
							Div(
								g.Attr("x-show", "isSelectionType()"),
								Class("space-y-4"),
								Div(
									Class("flex items-center justify-between"),
									Label(Class("text-sm font-medium text-gray-700 dark:text-gray-300"), g.Text("Options")),
									Span(Class("text-xs text-gray-400"), g.Attr("x-text", "enumOptions.length + ' option(s)'")),
								),
								// Options list with Alpine template
								g.Raw(`<div class="space-y-2 max-h-64 overflow-y-auto pr-2">
									<template x-for="(option, index) in enumOptions" :key="index">
										<div class="group flex gap-2 items-center p-3 bg-gray-50 dark:bg-neutral-800 rounded-lg border border-gray-200 dark:border-neutral-700 hover:border-violet-300 dark:hover:border-violet-700 transition-colors">
											<div class="text-gray-400 cursor-move">
												<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="12" r="1"/><circle cx="9" cy="5" r="1"/><circle cx="9" cy="19" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="5" r="1"/><circle cx="15" cy="19" r="1"/></svg>
											</div>
											<div class="flex-1">
												<input type="text" x-model="option.label" :name="'options.enum[' + index + '].label'" placeholder="Display label" class="w-full px-3 py-2 text-sm bg-white dark:bg-neutral-900 border border-gray-200 dark:border-neutral-600 rounded-lg focus:ring-2 focus:ring-violet-500 focus:border-transparent">
											</div>
											<div class="flex-1">
												<input type="text" x-model="option.value" :name="'options.enum[' + index + '].value'" placeholder="API value" @input="if (!option.value && option.label) option.value = option.label.toLowerCase().replace(/\s+/g, '_')" class="w-full px-3 py-2 text-sm bg-white dark:bg-neutral-900 border border-gray-200 dark:border-neutral-600 rounded-lg focus:ring-2 focus:ring-violet-500 focus:border-transparent font-mono">
											</div>
											<button type="button" @click="removeOption(index)" :disabled="enumOptions.length <= 1" class="p-2 text-gray-400 hover:text-red-500 disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
												<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
											</button>
										</div>
									</template>
								</div>`),
								// Add option button
								Button(
									Type("button"),
									g.Attr("@click", "addOption()"),
									Class("w-full py-2.5 border-2 border-dashed border-gray-300 dark:border-neutral-600 rounded-lg text-sm font-medium text-gray-500 dark:text-gray-400 hover:border-violet-400 hover:text-violet-600 dark:hover:border-violet-500 dark:hover:text-violet-400 transition-colors flex items-center justify-center gap-2"),
									lucide.Plus(Class("size-4")),
									g.Text("Add Option"),
								),
								// Default value for selection
								Div(
									Class("pt-4 border-t border-gray-100 dark:border-neutral-800"),
									fieldInput("options.defaultEnum", "Default Value", "text", "Enter default option value", false, "", "", ""),
								),
							),

							// Relation type options with builder
							Div(
								g.Attr("x-show", "isRelationType()"),
								Class("space-y-5"),
								// Related type selector
								selectField("options.relatedType", "Related Content Type", contentTypeOptions),
								// Relation type visual selector
								Div(
									Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3"), g.Text("Relation Type")),
									g.Raw(`<div class="grid grid-cols-2 gap-3">
										<label class="relative flex flex-col items-center p-4 border-2 rounded-xl cursor-pointer transition-all" :class="relationType === 'oneToOne' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700 hover:border-gray-300'">
											<input type="radio" name="options.relationType" value="oneToOne" x-model="relationType" class="sr-only">
											<div class="flex items-center gap-2 mb-2">
												<div class="w-3 h-3 bg-violet-500 rounded-full"></div>
												<div class="w-8 h-0.5 bg-gray-300"></div>
												<div class="w-3 h-3 bg-violet-500 rounded-full"></div>
											</div>
											<span class="text-sm font-medium text-gray-900 dark:text-white">One to One</span>
											<span class="text-xs text-gray-500">1 → 1</span>
										</label>
										<label class="relative flex flex-col items-center p-4 border-2 rounded-xl cursor-pointer transition-all" :class="relationType === 'oneToMany' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700 hover:border-gray-300'">
											<input type="radio" name="options.relationType" value="oneToMany" x-model="relationType" class="sr-only">
											<div class="flex items-center gap-2 mb-2">
												<div class="w-3 h-3 bg-violet-500 rounded-full"></div>
												<div class="w-8 h-0.5 bg-gray-300"></div>
												<div class="flex flex-col gap-0.5">
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
												</div>
											</div>
											<span class="text-sm font-medium text-gray-900 dark:text-white">One to Many</span>
											<span class="text-xs text-gray-500">1 → N</span>
										</label>
										<label class="relative flex flex-col items-center p-4 border-2 rounded-xl cursor-pointer transition-all" :class="relationType === 'manyToOne' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700 hover:border-gray-300'">
											<input type="radio" name="options.relationType" value="manyToOne" x-model="relationType" class="sr-only">
											<div class="flex items-center gap-2 mb-2">
												<div class="flex flex-col gap-0.5">
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
												</div>
												<div class="w-8 h-0.5 bg-gray-300"></div>
												<div class="w-3 h-3 bg-violet-500 rounded-full"></div>
											</div>
											<span class="text-sm font-medium text-gray-900 dark:text-white">Many to One</span>
											<span class="text-xs text-gray-500">N → 1</span>
										</label>
										<label class="relative flex flex-col items-center p-4 border-2 rounded-xl cursor-pointer transition-all" :class="relationType === 'manyToMany' ? 'border-violet-500 bg-violet-50 dark:bg-violet-900/20' : 'border-gray-200 dark:border-neutral-700 hover:border-gray-300'">
											<input type="radio" name="options.relationType" value="manyToMany" x-model="relationType" class="sr-only">
											<div class="flex items-center gap-2 mb-2">
												<div class="flex flex-col gap-0.5">
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
												</div>
												<div class="w-8 h-0.5 bg-gray-300"></div>
												<div class="flex flex-col gap-0.5">
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
													<div class="w-2 h-2 bg-violet-500 rounded-full"></div>
												</div>
											</div>
											<span class="text-sm font-medium text-gray-900 dark:text-white">Many to Many</span>
											<span class="text-xs text-gray-500">N → N</span>
										</label>
									</div>`),
								),
								// Additional relation options
								Div(
									Class("grid grid-cols-2 gap-4 pt-4 border-t border-gray-100 dark:border-neutral-800"),
									selectField("options.onDelete", "On Delete", []selectOption{
										{Value: "restrict", Label: "Restrict (prevent delete)"},
										{Value: "cascade", Label: "Cascade (delete related)"},
										{Value: "setNull", Label: "Set Null (clear reference)"},
									}),
									fieldInput("options.inverseField", "Inverse Field Name", "text", "e.g., author_posts", false, "", "", ""),
								),
								// Relation display options
								Div(
									Class("grid grid-cols-2 gap-4"),
									fieldInput("options.displayField", "Display Field", "text", "e.g., title, name", false, "", "", ""),
									switchField("options.allowCreate", "Allow Creating New"),
								),
							),

							// Media type options
							Div(
								g.Attr("x-show", "isMediaType()"),
								Class("space-y-5"),
								selectField("options.mediaType", "Allowed Media", []selectOption{
									{Value: "any", Label: "Any file type"},
									{Value: "image", Label: "Images only"},
									{Value: "video", Label: "Videos only"},
									{Value: "audio", Label: "Audio only"},
									{Value: "document", Label: "Documents only"},
								}),
								Div(
									Class("grid grid-cols-2 gap-4"),
									numberInputPreline("options.maxFileSize", "Max Size (MB)", 1, 1000, 10),
									numberInputPreline("options.maxFiles", "Max Files", 1, 100, 1),
								),
								fieldInput("options.allowedMimeTypes", "Allowed MIME Types", "text", "e.g., image/png, image/jpeg", false, "", "", ""),
							),

							// Slug type options
							Div(
								g.Attr("x-show", "type === 'slug'"),
								Class("space-y-4"),
								fieldInput("options.sourceField", "Generate From", "text", "e.g., title", false, "", "", ""),
								P(Class("text-xs text-gray-500"), g.Text("Auto-generate slug from another field")),
							),

							// Boolean type options
							Div(
								g.Attr("x-show", "type === 'boolean'"),
								Class("space-y-4"),
								selectField("options.defaultBool", "Default Value", []selectOption{
									{Value: "", Label: "No default"},
									{Value: "true", Label: "True"},
									{Value: "false", Label: "False"},
								}),
								Div(
									Class("grid grid-cols-2 gap-4"),
									fieldInput("options.trueLabel", "True Label", "text", "Yes", false, "", "", ""),
									fieldInput("options.falseLabel", "False Label", "text", "No", false, "", "", ""),
								),
							),

							// JSON type options
							Div(
								g.Attr("x-show", "type === 'json'"),
								Class("space-y-4"),
								Div(
									Label(For("options.schema"), Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("JSON Schema (optional)")),
									Textarea(
										ID("options.schema"), Name("options.schema"), Rows("4"),
										Placeholder(`{"type": "object", "properties": {...}}`),
										Class("block w-full px-4 py-3 text-sm font-mono border border-gray-200 rounded-xl bg-white dark:bg-neutral-800 dark:border-neutral-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all resize-none"),
									),
								),
							),

							// Color type options
							Div(
								g.Attr("x-show", "type === 'color'"),
								Class("space-y-4"),
								selectField("options.colorFormat", "Color Format", []selectOption{
									{Value: "hex", Label: "HEX (#ffffff)"},
									{Value: "rgb", Label: "RGB (255, 255, 255)"},
									{Value: "rgba", Label: "RGBA (255, 255, 255, 1)"},
									{Value: "hsl", Label: "HSL (0, 0%, 100%)"},
								}),
								fieldInput("options.defaultColor", "Default Color", "color", "", false, "", "", ""),
							),
						),
					),
				),

				// Right column - Settings sidebar
				Div(
					Class("space-y-6"),

					// Validation card
					Div(
						Class("bg-white dark:bg-neutral-900 rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden"),
						Div(
							Class("px-5 py-4 border-b border-gray-200 dark:border-neutral-700 bg-gray-50 dark:bg-neutral-800"),
							H3(Class("text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"),
								lucide.Shield(Class("size-4 text-gray-500")),
								g.Text("Validation"),
							),
						),
						Div(
							Class("p-5 space-y-4"),
							toggleOption("required", "Required", "Field must have a value", "text-red-500"),
							toggleOption("unique", "Unique", "Values must be unique", "text-amber-500"),
						),
					),

					// Indexing card
					Div(
						Class("bg-white dark:bg-neutral-900 rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden"),
						Div(
							Class("px-5 py-4 border-b border-gray-200 dark:border-neutral-700 bg-gray-50 dark:bg-neutral-800"),
							H3(Class("text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"),
								lucide.Search(Class("size-4 text-gray-500")),
								g.Text("Search & Index"),
							),
						),
						Div(
							Class("p-5 space-y-4"),
							toggleOption("indexed", "Database Index", "Faster queries on this field", "text-blue-500"),
							toggleOption("searchable", "Full-text Search", "Include in search results", "text-green-500"),
						),
					),

					// Advanced card
					Div(
						Class("bg-white dark:bg-neutral-900 rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden"),
						Div(
							Class("px-5 py-4 border-b border-gray-200 dark:border-neutral-700 bg-gray-50 dark:bg-neutral-800"),
							H3(Class("text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"),
								lucide.Settings(Class("size-4 text-gray-500")),
								g.Text("Advanced"),
							),
						),
						Div(
							Class("p-5 space-y-4"),
							toggleOption("localized", "Localization", "Multiple language versions", "text-purple-500"),
							toggleOption("hidden", "Hidden", "Hide from API responses", "text-gray-500"),
							toggleOption("readOnly", "Read Only", "Cannot be modified via API", "text-orange-500"),
						),
					),
				),
			),

			// Submit footer
			Div(
				Class("flex items-center justify-between pt-6 mt-6 border-t border-gray-200 dark:border-neutral-700"),
				A(
					Href(typeBase),
					Class("px-4 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white transition-colors"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("px-6 py-2.5 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 focus:ring-4 focus:ring-violet-200 dark:focus:ring-violet-800 transition-all flex items-center gap-2"),
					g.If(isEdit, lucide.Check(Class("size-4"))),
					g.If(!isEdit, lucide.Plus(Class("size-4"))),
					g.Text(submitText),
				),
			),
		),
	)
}

// fieldBuilderAlpineData returns the Alpine.js data for the field builder
func fieldBuilderAlpineData(field *core.ContentFieldDTO) string {
	// Default values
	name := ""
	slug := ""
	fieldType := ""
	relationType := "oneToMany"
	enumOptionsJSON := "[{label: '', value: ''}]"
	slugManuallyEdited := "false"

	// Populate from existing field if editing
	if field != nil {
		name = field.Name
		slug = field.Name
		fieldType = field.Type
		slugManuallyEdited = "true"

		// Extract enum options if present
		if len(field.Options.Choices) > 0 {
			var opts []string
			for _, c := range field.Options.Choices {
				opts = append(opts, fmt.Sprintf("{label: '%s', value: '%s'}", c.Label, c.Value))
			}
			if len(opts) > 0 {
				enumOptionsJSON = "[" + opts[0]
				for i := 1; i < len(opts); i++ {
					enumOptionsJSON += ", " + opts[i]
				}
				enumOptionsJSON += "]"
			}
		}

		// Extract relation type if present
		if field.Options.RelationType != "" {
			relationType = field.Options.RelationType
		}
	}

	return fmt.Sprintf(`{
		name: '%s',
		slug: '%s',
		type: '%s',
		relationType: '%s',
		slugManuallyEdited: %s,
		enumOptions: %s,
		
		generateSlug(name) {
			return name.toLowerCase().trim()
				.replace(/[^\w\s-]/g, '')
				.replace(/[\s_-]+/g, '_')
				.replace(/^_+|_+$/g, '');
		},
		updateSlug() {
			if (!this.slugManuallyEdited) {
				this.slug = this.generateSlug(this.name);
			}
		},
		addOption() {
			this.enumOptions.push({label: '', value: ''});
		},
		removeOption(index) {
			if (this.enumOptions.length > 1) {
				this.enumOptions.splice(index, 1);
			}
		},
		getTypeDescription() {
			const descriptions = {
				'text': 'Short text up to 255 characters',
				'textarea': 'Multi-line text without formatting',
				'richText': 'Formatted text with HTML support',
				'markdown': 'Text with Markdown formatting',
				'number': 'Any numeric value (integer or decimal)',
				'integer': 'Whole numbers only',
				'float': 'Decimal numbers',
				'decimal': 'Precise decimal with fixed precision',
				'bigInteger': 'Very large whole numbers',
				'boolean': 'True/false toggle',
				'date': 'Date without time',
				'datetime': 'Date with time',
				'time': 'Time only',
				'email': 'Email address with validation',
				'url': 'Web URL with validation',
				'phone': 'Phone number',
				'slug': 'URL-friendly identifier',
				'uuid': 'Unique identifier (UUID v4)',
				'color': 'Color picker',
				'password': 'Encrypted password field',
				'json': 'Arbitrary JSON data',
				'select': 'Single choice from options',
				'multiSelect': 'Multiple choices from options',
				'enumeration': 'Predefined set of values',
				'relation': 'Reference to another content type',
				'media': 'File or image upload'
			};
			return descriptions[this.type] || '';
		},
		isTextType() {
			return ['text', 'textarea', 'richText', 'markdown', 'email', 'url', 'phone', 'slug', 'password'].includes(this.type);
		},
		isNumberType() {
			return ['number', 'integer', 'float', 'decimal', 'bigInteger'].includes(this.type);
		},
		isDateType() {
			return ['date', 'datetime', 'time'].includes(this.type);
		},
		isSelectionType() {
			return ['select', 'multiSelect', 'enumeration'].includes(this.type);
		},
		isRelationType() {
			return ['relation'].includes(this.type);
		},
		isMediaType() {
			return ['media'].includes(this.type);
		}
	}`, name, slug, fieldType, relationType, slugManuallyEdited, enumOptionsJSON)
}

// fieldInput creates a styled input field with optional Alpine bindings
func fieldInput(name, label, inputType, placeholder string, required bool, xModel, eventName, eventHandler string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
			g.Text(label),
			g.If(required, Span(Class("text-red-500 ml-1"), g.Text("*"))),
		),
		Input(
			Type(inputType),
			ID(name),
			Name(name),
			Placeholder(placeholder),
			g.If(required, Required()),
			g.If(xModel != "", g.Attr("x-model", xModel)),
			g.If(eventName != "", g.Attr(eventName, eventHandler)),
			Class("block w-full px-4 py-2.5 text-sm border border-gray-200 rounded-xl bg-white dark:bg-neutral-800 dark:border-neutral-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"),
		),
	)
}

// numberInputPreline creates a Preline-styled number input with increment/decrement
func numberInputPreline(name, label string, min, max, defaultVal int) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
			g.Text(label),
		),
		Div(
			Class("py-2 px-3 bg-white border border-gray-200 rounded-xl dark:bg-neutral-800 dark:border-neutral-700"),
			g.Attr("data-hs-input-number", fmt.Sprintf(`{"min": %d, "max": %d}`, min, max)),
			Div(
				Class("flex items-center gap-x-1.5"),
				Button(
					Type("button"),
					Class("size-7 inline-flex justify-center items-center text-sm font-medium rounded-lg border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 disabled:opacity-50 dark:bg-neutral-800 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-700 transition-all"),
					g.Attr("data-hs-input-number-decrement", ""),
					g.Attr("aria-label", "Decrease"),
					lucide.Minus(Class("size-3.5")),
				),
				Input(
					Type("text"),
					ID(name),
					Name(name),
					Class("p-0 w-12 bg-transparent border-0 text-gray-800 text-center text-sm focus:ring-0 dark:text-white [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"),
					Value(fmt.Sprintf("%d", defaultVal)),
					g.Attr("data-hs-input-number-input", ""),
				),
				Button(
					Type("button"),
					Class("size-7 inline-flex justify-center items-center text-sm font-medium rounded-lg border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 disabled:opacity-50 dark:bg-neutral-800 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-700 transition-all"),
					g.Attr("data-hs-input-number-increment", ""),
					g.Attr("aria-label", "Increase"),
					lucide.Plus(Class("size-3.5")),
				),
			),
		),
	)
}

// selectOption represents an option for select fields
type selectOption struct {
	Value string
	Label string
}

// selectField creates a styled select dropdown
func selectField(name, label string, options []selectOption) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
			g.Text(label),
		),
		Select(
			ID(name),
			Name(name),
			Class("block w-full px-4 py-2.5 text-sm border border-gray-200 rounded-xl bg-white dark:bg-neutral-800 dark:border-neutral-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"),
			g.Group(func() []g.Node {
				opts := make([]g.Node, len(options))
				for i, opt := range options {
					opts[i] = Option(Value(opt.Value), g.Text(opt.Label))
				}
				return opts
			}()),
		),
	)
}

// switchField creates a toggle switch
func switchField(name, label string) g.Node {
	return Div(
		Class("flex items-center justify-between"),
		Label(For(name), Class("text-sm text-gray-700 dark:text-gray-300"), g.Text(label)),
		Div(
			Class("relative"),
			Input(
				Type("checkbox"),
				ID(name),
				Name(name),
				Value("true"),
				Class("sr-only peer"),
			),
			Label(
				For(name),
				Class("relative w-11 h-6 bg-gray-200 peer-focus:ring-4 peer-focus:ring-violet-100 dark:peer-focus:ring-violet-800 rounded-full peer dark:bg-neutral-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-neutral-600 peer-checked:bg-violet-600 cursor-pointer"),
			),
		),
	)
}

// toggleOption creates a toggle option for the sidebar
func toggleOption(name, label, description, iconColor string) g.Node {
	return Div(
		Class("flex items-start gap-3"),
		Div(
			Class("flex items-center h-5"),
			Input(
				Type("checkbox"),
				ID(name),
				Name(name),
				Value("true"),
				Class("w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-neutral-800 focus:ring-2 dark:bg-neutral-700 dark:border-neutral-600"),
			),
		),
		Label(
			For(name),
			Class("flex-1 cursor-pointer"),
			Div(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text(label)),
			Div(Class("text-xs text-gray-500 dark:text-gray-400"), g.Text(description)),
		),
	)
}

// inputNumber creates a numeric input with increment/decrement buttons (Preline style)
func inputNumber(name, label string, min, max, step int, help string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
		),
		Div(
			Class("py-2 px-3 inline-block bg-white border border-gray-200 rounded-lg dark:bg-neutral-900 dark:border-neutral-700"),
			g.Attr("data-hs-input-number", ""),
			Div(
				Class("flex items-center gap-x-1.5"),
				Button(
					Type("button"),
					Class("size-6 inline-flex justify-center items-center gap-x-2 text-sm font-medium rounded-md border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 disabled:opacity-50 disabled:pointer-events-none dark:bg-neutral-900 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-800"),
					g.Attr("data-hs-input-number-decrement", ""),
					lucide.Minus(Class("size-3.5")),
				),
				Input(
					Type("text"),
					Class("p-0 w-full bg-transparent border-0 text-gray-800 text-center focus:ring-0 dark:text-white"),
					Name(name),
					Value("0"),
					g.Attr("data-hs-input-number-input", ""),
				),
				Button(
					Type("button"),
					Class("size-6 inline-flex justify-center items-center gap-x-2 text-sm font-medium rounded-md border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 disabled:opacity-50 disabled:pointer-events-none dark:bg-neutral-900 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-800"),
					g.Attr("data-hs-input-number-increment", ""),
					lucide.Plus(Class("size-3.5")),
				),
			),
		),
		g.If(help != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(help),
			)
		}()),
	)
}

// checkboxField creates a checkbox field
func checkboxField(name, label, description string) g.Node {
	return Div(
		Class("flex items-start gap-3"),
		Input(
			Type("checkbox"),
			ID(name),
			Name(name),
			Value("true"),
			Class("mt-1 h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
		),
		Label(
			For(name),
			Class("text-sm"),
			Div(
				Class("font-medium text-slate-700 dark:text-gray-300"),
				g.Text(label),
			),
			Div(
				Class("text-xs text-slate-500 dark:text-gray-500"),
				g.Text(description),
			),
		),
	)
}
