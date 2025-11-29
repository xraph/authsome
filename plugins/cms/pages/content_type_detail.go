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
// Content Type Detail Page
// =============================================================================

// ContentTypeDetailPage renders the content type detail/edit page
func ContentTypeDetailPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	stats *core.ContentTypeStatsDTO,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Slug
	apiBase := basePath + "/cms/" + contentType.Slug

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
		Class("space-y-6"),
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
							g.Text(contentType.Slug),
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
				fieldsSection(appBase, contentType),
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
				playgroundSection(apiBase, contentType),
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

// fieldsSection renders the fields management section with DaisyUI drawer
func fieldsSection(appBase string, contentType *core.ContentTypeDTO) g.Node {
	typeBase := appBase + "/cms/types/" + contentType.Slug
	drawerID := "add-field-drawer"

	return Div(
		Class("drawer drawer-end"),

		// Hidden checkbox that controls drawer state
		Input(
			ID(drawerID),
			Type("checkbox"),
			Class("drawer-toggle"),
		),

		// Main content (drawer-content)
		Div(
			Class("drawer-content"),
			CardWithHeader(
				"Fields",
				[]g.Node{
					// Trigger label for drawer
					Label(
						g.Attr("for", drawerID),
						Class("inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 cursor-pointer dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"),
						lucide.Plus(Class("size-4")),
						g.Text("Add Field"),
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
						Label(
							g.Attr("for", drawerID),
							Class("mt-4 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 cursor-pointer"),
							lucide.Plus(Class("size-4")),
							g.Text("Add Field"),
						),
					)
				}()),

				g.If(len(contentType.Fields) > 0, func() g.Node {
					return fieldsTable(typeBase, contentType.Fields, drawerID)
				}()),
			),
		),

		// Drawer sidebar
		Div(
			Class("drawer-side z-50"),
			// Overlay that closes drawer when clicked
			Label(
				g.Attr("for", drawerID),
				g.Attr("aria-label", "close sidebar"),
				Class("drawer-overlay"),
			),
			// Drawer content
			Div(
				Class("min-h-full w-96 bg-white dark:bg-neutral-900 flex flex-col"),

				// Header
				Div(
					Class("flex justify-between items-center py-4 px-6 border-b border-slate-200 dark:border-neutral-700"),
					H3(
						Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Add Field"),
					),
					Label(
						g.Attr("for", drawerID),
						Class("size-8 inline-flex justify-center items-center rounded-full bg-slate-100 text-slate-600 hover:bg-slate-200 cursor-pointer dark:bg-neutral-800 dark:text-neutral-400 dark:hover:bg-neutral-700"),
						lucide.X(Class("size-4")),
					),
				),

				// Form body
				Div(
					Class("flex-1 overflow-y-auto p-6"),
					addFieldForm(typeBase, contentType),
				),

				// Footer
				Div(
					Class("flex justify-end items-center gap-3 py-4 px-6 border-t border-slate-200 dark:border-neutral-700"),
					Label(
						g.Attr("for", drawerID),
						Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 cursor-pointer dark:bg-neutral-800 dark:text-neutral-300 dark:border-neutral-700 dark:hover:bg-neutral-700"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						g.Attr("form", "add-field-form"),
						Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700"),
						lucide.Plus(Class("size-4 mr-1")),
						g.Text("Add Field"),
					),
				),
			),
		),
	)
}

// addFieldForm renders the form for adding a new field
func addFieldForm(typeBase string, contentType *core.ContentTypeDTO) g.Node {
	fieldTypes := core.GetAllFieldTypes()

	return FormEl(
		ID("add-field-form"),
		Method("POST"),
		Action(typeBase+"/fields"),
		Class("space-y-5"),

		// Name field
		Div(
			Label(
				For("field-name"),
				Class("block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1.5"),
				g.Text("Name"),
				Span(Class("text-red-500 ml-1"), g.Text("*")),
			),
			Input(
				Type("text"),
				ID("field-name"),
				Name("name"),
				Required(),
				Placeholder("e.g., Title, Content, Author"),
				Class("w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent dark:bg-neutral-800 dark:border-neutral-700 dark:text-white"),
			),
			P(
				Class("mt-1 text-xs text-slate-500 dark:text-neutral-500"),
				g.Text("Human-readable field name"),
			),
		),

		// Slug field
		Div(
			Label(
				For("field-slug"),
				Class("block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1.5"),
				g.Text("Slug"),
			),
			Input(
				Type("text"),
				ID("field-slug"),
				Name("slug"),
				Placeholder("Auto-generated from name"),
				Class("w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent dark:bg-neutral-800 dark:border-neutral-700 dark:text-white"),
			),
			P(
				Class("mt-1 text-xs text-slate-500 dark:text-neutral-500"),
				g.Text("Machine-readable identifier (optional)"),
			),
		),

		// Type field
		Div(
			Label(
				For("field-type"),
				Class("block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1.5"),
				g.Text("Type"),
				Span(Class("text-red-500 ml-1"), g.Text("*")),
			),
			Select(
				ID("field-type"),
				Name("type"),
				Required(),
				Class("w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent dark:bg-neutral-800 dark:border-neutral-700 dark:text-white"),
				Option(Value(""), g.Text("Select a field type...")),
				g.Group(func() []g.Node {
					opts := make([]g.Node, len(fieldTypes))
					for i, ft := range fieldTypes {
						opts[i] = Option(
							Value(ft.Type.String()),
							g.Text(fmt.Sprintf("%s - %s", ft.Name, ft.Description)),
						)
					}
					return opts
				}()),
			),
		),

		// Description field
		Div(
			Label(
				For("field-description"),
				Class("block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1.5"),
				g.Text("Description"),
			),
			Textarea(
				ID("field-description"),
				Name("description"),
				Rows("2"),
				Placeholder("Optional description for this field..."),
				Class("w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent dark:bg-neutral-800 dark:border-neutral-700 dark:text-white"),
			),
		),

		// Options section
		Div(
			Class("pt-3"),
			P(
				Class("text-sm font-medium text-slate-700 dark:text-neutral-300 mb-3"),
				g.Text("Field Options"),
			),
			Div(
				Class("space-y-3"),
				drawerCheckbox("required", "Required", "This field must have a value"),
				drawerCheckbox("unique", "Unique", "Values must be unique across entries"),
				drawerCheckbox("indexed", "Indexed", "Enable fast searching on this field"),
				drawerCheckbox("localized", "Localized", "Support multiple language versions"),
			),
		),
	)
}

// drawerCheckbox creates a checkbox for the drawer form
func drawerCheckbox(name, label, description string) g.Node {
	return Label(
		Class("flex items-start gap-3 cursor-pointer"),
		Input(
			Type("checkbox"),
			Name(name),
			Value("true"),
			Class("mt-0.5 h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-neutral-600 dark:bg-neutral-800"),
		),
		Div(
			Span(
				Class("block text-sm font-medium text-slate-700 dark:text-neutral-300"),
				g.Text(label),
			),
			Span(
				Class("block text-xs text-slate-500 dark:text-neutral-500"),
				g.Text(description),
			),
		),
	)
}

// fieldsTable renders the fields as a table
func fieldsTable(typeBase string, fields []*core.ContentFieldDTO, _ string) g.Node {
	rows := make([]g.Node, len(fields))
	for i, field := range fields {
		rows[i] = fieldRow(typeBase, field)
	}

	return DataTable(
		[]string{"Field", "Type", "Properties", "Actions"},
		rows,
	)
}

// fieldRow renders a single field row
func fieldRow(typeBase string, field *core.ContentFieldDTO) g.Node {
	modalID := "delete-field-" + field.Slug

	return g.Group([]g.Node{
		TableRow(
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
						g.Text(field.Slug),
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

			// Actions - using simple form with confirmation
			TableCell(Div(
				Class("flex items-center justify-end gap-1"),
				// Delete button opens modal
				Label(
					g.Attr("for", modalID),
					Class("inline-flex items-center justify-center size-8 rounded-lg text-red-600 hover:bg-red-50 cursor-pointer dark:hover:bg-red-900/20"),
					g.Attr("title", "Delete Field"),
					lucide.Trash2(Class("size-4")),
				),
			)),
		),

		// Delete confirmation modal (DaisyUI style)
		deleteFieldModal(modalID, typeBase, field),
	})
}

// deleteFieldModal renders a DaisyUI-style confirmation modal for deleting a field
func deleteFieldModal(modalID, typeBase string, field *core.ContentFieldDTO) g.Node {
	return g.Group([]g.Node{
		// Hidden checkbox that controls modal
		Input(
			Type("checkbox"),
			ID(modalID),
			Class("modal-toggle"),
		),

		// Modal container
		Div(
			Class("modal modal-bottom sm:modal-middle"),
			g.Attr("role", "dialog"),

			Div(
				Class("modal-box bg-white dark:bg-neutral-800"),

				// Header with close button
				Div(
					Class("flex justify-between items-center mb-4"),
					H3(
						Class("text-lg font-bold text-slate-900 dark:text-white"),
						g.Text("Delete Field"),
					),
					Label(
						g.Attr("for", modalID),
						Class("btn btn-sm btn-circle btn-ghost"),
						lucide.X(Class("size-4")),
					),
				),

				// Body
				Div(
					Class("text-center py-4"),
					Div(
						Class("mx-auto flex items-center justify-center size-12 rounded-full bg-red-100 dark:bg-red-900/20 mb-4"),
						lucide.TriangleAlert(Class("size-6 text-red-600 dark:text-red-400")),
					),
					P(
						Class("text-slate-700 dark:text-neutral-200"),
						g.Text("Are you sure you want to delete "),
						Strong(g.Text(field.Name)),
						g.Text("?"),
					),
					P(
						Class("mt-2 text-sm text-slate-500 dark:text-neutral-500"),
						g.Text("This action cannot be undone."),
					),
				),

				// Actions
				Div(
					Class("modal-action"),
					Label(
						g.Attr("for", modalID),
						Class("btn btn-ghost"),
						g.Text("Cancel"),
					),
					FormEl(
						Method("POST"),
						Action(typeBase+"/fields/"+field.Slug+"/delete"),
						Button(
							Type("submit"),
							Class("btn btn-error"),
							lucide.Trash2(Class("size-4")),
							g.Text("Delete"),
						),
					),
				),
			),

			// Backdrop - click to close
			Label(
				Class("modal-backdrop"),
				g.Attr("for", modalID),
			),
		),
	})
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

// settingsSection renders the content type settings management section
func settingsSection(typeBase string, contentType *core.ContentTypeDTO) g.Node {
	settings := contentType.Settings

	return Div(
		Class("space-y-6 mt-6"),

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
							Class("label"),
							Span(Class("label-text"), g.Text("Name")),
							Span(Class("label-text-alt text-red-500"), g.Text("*")),
						),
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Value(contentType.Name),
							Required(),
							Class("input input-bordered w-full"),
						),
					),
					// Slug (read-only)
					Div(
						Label(
							For("slug"),
							Class("label"),
							Span(Class("label-text"), g.Text("Slug")),
							Span(Class("label-text-alt text-slate-500"), g.Text("(read-only)")),
						),
						Input(
							Type("text"),
							ID("slug"),
							Name("slug"),
							Value(contentType.Slug),
							Disabled(),
							Class("input input-bordered w-full bg-slate-50 dark:bg-gray-900"),
						),
					),
				),

				// Description
				Div(
					Label(
						For("description"),
						Class("label"),
						Span(Class("label-text"), g.Text("Description")),
					),
					Textarea(
						ID("description"),
						Name("description"),
						Rows("3"),
						Placeholder("A brief description of this content type..."),
						Class("textarea textarea-bordered w-full"),
						g.Text(contentType.Description),
					),
				),

				// Icon
				Div(
					Label(
						For("icon"),
						Class("label"),
						Span(Class("label-text"), g.Text("Icon (emoji)")),
					),
					Input(
						Type("text"),
						ID("icon"),
						Name("icon"),
						Value(contentType.Icon),
						Placeholder("📄"),
						Class("input input-bordered w-full"),
					),
					P(Class("text-xs text-gray-500 mt-1"), g.Text("Use an emoji to visually identify this content type")),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("btn btn-primary"),
						lucide.Save(Class("size-4")),
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

				// Title and Description fields
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
					// Title Field
					Div(
						Label(
							For("titleField"),
							Class("label"),
							Span(Class("label-text"), g.Text("Title Field")),
						),
						Select(
							ID("titleField"),
							Name("titleField"),
							Class("select select-bordered w-full"),
							Option(Value(""), g.Text("Select a field...")),
							g.Group(fieldSelectOptions(contentType.Fields, settings.TitleField, []string{"text", "string"})),
						),
						P(Class("text-xs text-gray-500 mt-1"), g.Text("Field used as the entry title in lists")),
					),
					// Description Field
					Div(
						Label(
							For("descriptionField"),
							Class("label"),
							Span(Class("label-text"), g.Text("Description Field")),
						),
						Select(
							ID("descriptionField"),
							Name("descriptionField"),
							Class("select select-bordered w-full"),
							Option(Value(""), g.Text("Select a field...")),
							g.Group(fieldSelectOptions(contentType.Fields, settings.DescriptionField, []string{"text", "string", "richtext", "markdown"})),
						),
						P(Class("text-xs text-gray-500 mt-1"), g.Text("Field used as the entry description")),
					),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("btn btn-primary"),
						lucide.Save(Class("size-4")),
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
						Class("btn btn-primary"),
						lucide.Save(Class("size-4")),
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
						Class("label"),
						Span(Class("label-text"), g.Text("Maximum Entries")),
					),
					Input(
						Type("number"),
						ID("maxEntries"),
						Name("maxEntries"),
						Value(fmt.Sprintf("%d", settings.MaxEntries)),
						g.Attr("min", "0"),
						Placeholder("0 = unlimited"),
						Class("input input-bordered w-full max-w-xs"),
					),
					P(Class("text-xs text-gray-500 mt-1"), g.Text("Set to 0 for unlimited entries")),
				),

				// Submit
				Div(
					Class("pt-4"),
					Button(
						Type("submit"),
						Class("btn btn-primary"),
						lucide.Save(Class("size-4")),
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
					Button(
						Type("button"),
						g.Attr("onclick", "document.getElementById('delete-type-modal').checked = true"),
						Class("btn btn-error btn-sm"),
						lucide.Trash2(Class("size-4")),
						g.Text("Delete"),
					),
				),
			),
		),

		// Delete confirmation modal
		deleteContentTypeModal(typeBase, contentType),
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
		isSelected := field.Slug == selectedValue
		option := Option(
			Value(field.Slug),
			g.If(isSelected, Selected()),
			g.Text(field.Name),
		)
		options = append(options, option)
	}
	return options
}

// featureToggle renders a feature toggle checkbox
func featureToggle(name, label, description string, checked bool) g.Node {
	return Div(
		Class("form-control"),
		Label(
			Class("label cursor-pointer justify-start gap-4"),
			Input(
				Type("checkbox"),
				Name(name),
				Value("true"),
				g.If(checked, Checked()),
				Class("checkbox checkbox-primary"),
			),
			Div(
				Span(Class("label-text font-medium"), g.Text(label)),
				P(Class("text-xs text-gray-500 dark:text-gray-400"), g.Text(description)),
			),
		),
	)
}

// deleteContentTypeModal renders the delete confirmation modal
func deleteContentTypeModal(typeBase string, contentType *core.ContentTypeDTO) g.Node {
	return g.Group([]g.Node{
		Input(Type("checkbox"), ID("delete-type-modal"), Class("modal-toggle")),
		Div(
			Class("modal modal-bottom sm:modal-middle"),
			Div(
				Class("modal-box"),

				// Header
				Div(
					Class("flex items-center justify-between pb-4 border-b border-slate-200 dark:border-gray-700"),
					H3(
						Class("font-bold text-lg text-red-600 dark:text-red-400"),
						g.Text("Delete Content Type"),
					),
					Label(
						g.Attr("for", "delete-type-modal"),
						Class("btn btn-sm btn-circle btn-ghost"),
						lucide.X(Class("size-4")),
					),
				),

				// Body
				Div(
					Class("py-6"),
					Div(
						Class("text-center"),
						Div(
							Class("mx-auto flex items-center justify-center size-16 rounded-full bg-red-100 dark:bg-red-900/20 mb-4"),
							lucide.TriangleAlert(Class("size-8 text-red-600 dark:text-red-400")),
						),
						P(
							Class("text-lg font-medium text-slate-900 dark:text-white"),
							g.Text("Delete \""),
							g.Text(contentType.Name),
							g.Text("\"?"),
						),
						P(
							Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
							g.Text("This will permanently delete all entries, fields, and revisions."),
						),
					),
				),

				// Actions
				Div(
					Class("modal-action"),
					Label(
						g.Attr("for", "delete-type-modal"),
						Class("btn btn-ghost"),
						g.Text("Cancel"),
					),
					FormEl(
						Method("POST"),
						Action(typeBase+"/delete"),
						Button(
							Type("submit"),
							Class("btn btn-error"),
							lucide.Trash2(Class("size-4")),
							g.Text("Delete Forever"),
						),
					),
				),
			),
			Label(Class("modal-backdrop"), g.Attr("for", "delete-type-modal")),
		),
	})
}

// =============================================================================
// API Section
// =============================================================================

// apiSection renders the API documentation section
func apiSection(apiBase string, contentType *core.ContentTypeDTO) g.Node {
	return Div(
		Class("space-y-6 mt-6"),

		// Quick Reference Card
		CardWithHeader("API Endpoints", []g.Node{
			Span(Class("badge badge-primary badge-outline"), g.Text("REST API")),
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
			Span(Class("badge badge-secondary badge-outline"), g.Text("GET requests")),
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
						filterOpExample("eq", "Equals", "filter[status]=eq.published"),
						filterOpExample("ne", "Not equals", "filter[status]=ne.draft"),
						filterOpExample("gt", "Greater than", "filter[price]=gt.100"),
						filterOpExample("gte", "Greater or equal", "filter[price]=gte.100"),
						filterOpExample("lt", "Less than", "filter[price]=lt.50"),
						filterOpExample("lte", "Less or equal", "filter[price]=lte.50"),
						filterOpExample("like", "Contains (case-sensitive)", "filter[title]=like.hello"),
						filterOpExample("ilike", "Contains (case-insensitive)", "filter[title]=ilike.hello"),
						filterOpExample("in", "In list", "filter[status]=in.(draft,published)"),
						filterOpExample("nin", "Not in list", "filter[status]=nin.(archived)"),
						filterOpExample("null", "Is null", "filter[deletedAt]=null.true"),
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
			Span(Class("badge badge-accent badge-outline"), g.Text(fmt.Sprintf("%d fields", len(contentType.Fields)))),
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
	methodColor := "badge-info"
	switch method {
	case "GET":
		methodColor = "badge-success"
	case "POST":
		methodColor = "badge-primary"
	case "PUT":
		methodColor = "badge-warning"
	case "DELETE":
		methodColor = "badge-error"
	}

	return Tr(
		Td(Span(Class("badge "+methodColor+" badge-sm font-mono"), g.Text(method))),
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
				Code(Class("text-xs font-mono font-medium"), g.Text(field.Slug)),
			),
		),
		Td(Span(Class("badge badge-ghost badge-sm"), g.Text(field.Type))),
		Td(
			g.If(field.Required, func() g.Node {
				return Span(Class("badge badge-error badge-xs"), g.Text("required"))
			}()),
			g.If(!field.Required, func() g.Node {
				return Span(Class("badge badge-ghost badge-xs"), g.Text("optional"))
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
		sampleData += fmt.Sprintf(`    "%s": `, field.Slug)
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
					Span(Class("badge badge-primary badge-sm"), g.Text("POST")),
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
					Span(Class("badge badge-success badge-sm"), g.Text("GET")),
					g.Text("List with Filters"),
				),
				Pre(
					Class("bg-slate-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto"),
					Code(
						Class("text-sm text-emerald-400 font-mono whitespace-pre"),
						g.Text(fmt.Sprintf(`curl "%s?filter[status]=eq.published&sort=-createdAt&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"`, apiBase)),
					),
				),
			),

			// Advanced Query Example
			Div(
				H4(Class("font-medium text-sm text-slate-900 dark:text-white mb-2 flex items-center gap-2"),
					Span(Class("badge badge-primary badge-sm"), g.Text("POST")),
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
        { "status": { "$eq": "published" } },
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
func playgroundSection(apiBase string, contentType *core.ContentTypeDTO) g.Node {
	// Default query example
	defaultQuery := fmt.Sprintf(`{
  "filter": {
    "status": { "$eq": "published" }
  },
  "sort": ["-createdAt"],
  "page": 1,
  "pageSize": 10
}`)

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
			
			async executeQuery() {
				this.loading = true;
				this.error = null;
				this.response = null;
				
				try {
					let url = this.endpoint;
					let options = {
						method: this.method,
						headers: {
							'Content-Type': 'application/json'
						}
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
		}`, "`"+defaultQuery+"`", apiBase)),

		// Query Builder Card
		CardWithHeader("Query Builder", []g.Node{
			// Method selector
			Div(
				Class("flex items-center gap-2"),
				Select(
					g.Attr("x-model", "method"),
					Class("select select-bordered select-sm"),
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
						Class("btn btn-primary"),
						g.Attr(":class", "loading ? 'loading' : ''"),
						lucide.Play(Class("size-4"), g.Attr("x-show", "!loading")),
						Span(g.Attr("x-show", "loading"), g.Text("Executing...")),
						Span(g.Attr("x-show", "!loading"), g.Text("Execute Query")),
					),
					// Quick actions
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							filter: { status: { "$eq": "published" } },
							sort: ["-createdAt"],
							page: 1,
							pageSize: 10
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("btn btn-ghost btn-sm"),
						g.Text("Published Entries"),
					),
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							filter: { status: { "$eq": "draft" } },
							sort: ["-updatedAt"],
							page: 1,
							pageSize: 10
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("btn btn-ghost btn-sm"),
						g.Text("Drafts"),
					),
					Button(
						Type("button"),
						g.Attr("@click", `query = JSON.stringify({
							sort: ["-createdAt"],
							page: 1,
							pageSize: 50
						}, null, 2); if(window.monacoEditor) window.monacoEditor.setValue(query);`),
						Class("btn btn-ghost btn-sm"),
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
					Class("join"),
					Button(
						Type("button"),
						g.Attr("@click", "viewMode = 'table'"),
						g.Attr(":class", "viewMode === 'table' ? 'btn-active' : ''"),
						Class("btn btn-sm join-item"),
						lucide.Table(Class("size-4")),
						g.Text("Table"),
					),
					Button(
						Type("button"),
						g.Attr("@click", "viewMode = 'json'"),
						g.Attr(":class", "viewMode === 'json' ? 'btn-active' : ''"),
						Class("btn btn-sm join-item"),
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
						Class("overflow-x-auto"),

						// Summary stats
						Div(
							Class("mb-4 flex items-center gap-4 text-sm text-slate-600 dark:text-gray-400"),
							Span(
								g.Attr("x-show", "response?.total !== undefined"),
								g.Attr("x-text", "'Total: ' + (response?.total || 0) + ' entries'"),
							),
							Span(
								g.Attr("x-show", "response?.page !== undefined"),
								g.Attr("x-text", "'Page: ' + (response?.page || 1)"),
							),
						),

						// Table
						Table(
							Class("table table-zebra w-full"),
							THead(
								Tr(
									Th(g.Text("ID")),
									Th(g.Text("Status")),
									g.El("template",
										g.Attr("x-for", "field in getFields()"),
										g.El("th",
											g.Attr("x-text", "field"),
										),
									),
									Th(g.Text("Created")),
								),
							),
							TBody(
								g.El("template",
									g.Attr("x-for", "entry in getEntries()"),
									g.Attr(":key", "entry.id"),
									Tr(
										Td(
											Code(Class("text-xs font-mono"), g.Attr("x-text", "entry.id?.substring(0,8) + '...'")),
										),
										Td(
											Span(
												Class("badge badge-sm"),
												g.Attr(":class", `{
													'badge-success': entry.status === 'published',
													'badge-warning': entry.status === 'draft',
													'badge-ghost': entry.status === 'archived'
												}`),
												g.Attr("x-text", "entry.status"),
											),
										),
										g.El("template",
											g.Attr("x-for", "field in getFields()"),
											Td(
												Class("max-w-xs truncate text-sm"),
												g.Attr("x-text", "typeof (entry.data?.[field] || entry[field]) === 'object' ? JSON.stringify(entry.data?.[field] || entry[field]) : (entry.data?.[field] || entry[field] || '-')"),
											),
										),
										Td(
											Class("text-xs text-slate-500"),
											g.Attr("x-text", "entry.createdAt ? new Date(entry.createdAt).toLocaleDateString() : '-'"),
										),
									),
								),
							),
						),

						// Empty state
						Div(
							g.Attr("x-show", "getEntries().length === 0"),
							Class("text-center py-8 text-slate-500 dark:text-gray-400"),
							lucide.Inbox(Class("mx-auto size-12 mb-4 opacity-50")),
							P(g.Text("No entries found")),
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
							Code(Class("font-mono text-xs"), g.Text(field.Slug)),
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
	err string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Slug

	// Get all field types
	fieldTypes := core.GetAllFieldTypes()

	return Div(
		Class("space-y-6 max-w-2xl"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: typeBase},
			BreadcrumbItem{Label: "Add Field", Href: ""},
		),

		// Header
		PageHeader(
			"Add Field",
			fmt.Sprintf("Add a new field to %s", contentType.Name),
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
					Action(typeBase+"/fields/create"),
					Class("space-y-6"),

					// Name
					formField("name", "Name", "text", "", "e.g., Title, Content, Author", true, "Human-readable field name"),

					// Slug
					formField("slug", "Slug", "text", "", "e.g., title, content, author", true, "Machine-readable identifier. Use lowercase letters, numbers, and underscores."),

					// Type
					Div(
						Label(
							For("type"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
							g.Text("Type"),
							Span(Class("text-red-500 ml-1"), g.Text("*")),
						),
						Select(
							ID("type"),
							Name("type"),
							Required(),
							Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							Option(Value(""), g.Text("Select a field type...")),
							g.Group(func() []g.Node {
								opts := make([]g.Node, len(fieldTypes))
								for i, ft := range fieldTypes {
									opts[i] = Option(
										Value(ft.Type.String()),
										g.Text(fmt.Sprintf("%s - %s", ft.Name, ft.Description)),
									)
								}
								return opts
							}()),
						),
					),

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
							Rows("2"),
							Placeholder("Optional description for this field..."),
							Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						),
					),

					// Options
					Div(
						Class("grid grid-cols-2 gap-4"),
						checkboxField("required", "Required", "This field must have a value"),
						checkboxField("unique", "Unique", "Values must be unique across entries"),
						checkboxField("indexed", "Indexed", "Enable fast searching on this field"),
						checkboxField("localized", "Localized", "Support multiple language versions"),
					),

					// Submit buttons
					Div(
						Class("flex items-center gap-4 pt-4"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
							g.Text("Add Field"),
						),
						A(
							Href(typeBase),
							Class("px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
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
