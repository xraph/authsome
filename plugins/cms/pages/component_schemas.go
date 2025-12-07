package pages

import (
	"encoding/json"
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/cms/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// Component Schemas List Page
// =============================================================================

// ComponentSchemasPage renders the component schemas list page
func ComponentSchemasPage(
	currentApp *app.App,
	basePath string,
	components []*core.ComponentSchemaSummaryDTO,
	page, pageSize, totalItems int,
	searchQuery string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	totalPages := (totalItems + pageSize - 1) / pageSize

	return Div(
		Class("space-y-6"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: "Component Schemas", Href: ""},
		),

		// Header
		PageHeader(
			"Component Schemas",
			"Reusable schema definitions for nested object fields",
			PrimaryButton(appBase+"/cms/components/create", "Create Component", lucide.Plus(Class("size-4"))),
		),

		// Search
		Div(
			Class("flex flex-wrap gap-3"),
			SearchInput("Search components...", searchQuery, appBase+"/cms/components"),
		),

		// Table
		componentSchemasTable(currentApp, basePath, components, page, totalPages),
	)
}

// componentSchemasTable renders component schemas as a table
func componentSchemasTable(currentApp *app.App, basePath string, components []*core.ComponentSchemaSummaryDTO, page, totalPages int) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	if len(components) == 0 {
		return Card(
			EmptyState(
				lucide.Braces(Class("size-8 text-slate-400")),
				"No component schemas yet",
				"Create your first component schema to define reusable nested structures for your content types.",
				"Create Component",
				appBase+"/cms/components/create",
			),
		)
	}

	rows := make([]g.Node, len(components))
	for i, cs := range components {
		rows[i] = componentSchemaRow(appBase, cs)
	}

	return Div(
		Card(DataTable(
			[]string{"Name", "Name", "Fields", "Usage", "Updated", "Actions"},
			rows,
		)),
		Pagination(page, totalPages, appBase+"/cms/components"),
	)
}

// componentSchemaRow renders a single component schema table row
func componentSchemaRow(appBase string, cs *core.ComponentSchemaSummaryDTO) g.Node {
	return TableRow(
		// Name
		TableCell(Div(
			Class("flex items-center gap-3"),
			Div(
				Class("flex-shrink-0 rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30"),
				g.If(cs.Icon != "", func() g.Node {
					return Span(Class("text-lg"), g.Text(cs.Icon))
				}()),
				g.If(cs.Icon == "", func() g.Node {
					return lucide.Braces(Class("size-4 text-indigo-600 dark:text-indigo-400"))
				}()),
			),
			Div(
				Div(Class("font-medium"), g.Text(cs.Name)),
				g.If(cs.Description != "", func() g.Node {
					return Div(
						Class("text-xs text-slate-500 dark:text-gray-500 truncate max-w-xs"),
						g.Text(cs.Description),
					)
				}()),
			),
		)),

		// Slug
		TableCellSecondary(Code(
			Class("text-xs bg-slate-100 dark:bg-gray-800 px-1.5 py-0.5 rounded"),
			g.Text(cs.Name),
		)),

		// Fields
		TableCellSecondary(g.Text(fmt.Sprintf("%d", cs.FieldCount))),

		// Usage
		TableCell(Div(
			g.If(cs.UsageCount > 0, func() g.Node {
				return Span(
					Class("inline-flex items-center gap-1 text-blue-600 dark:text-blue-400"),
					lucide.Link(Class("size-3")),
					g.Text(fmt.Sprintf("%d fields", cs.UsageCount)),
				)
			}()),
			g.If(cs.UsageCount == 0, func() g.Node {
				return Span(
					Class("text-slate-400 dark:text-gray-600"),
					g.Text("Not used"),
				)
			}()),
		)),

		// Updated
		TableCellSecondary(g.Text(FormatTimeAgo(cs.UpdatedAt))),

		// Actions
		TableCellActions(
			IconButton(appBase+"/cms/components/"+cs.Name, lucide.Pencil(Class("size-4")), "Edit Component", "text-slate-600"),
			g.If(cs.UsageCount == 0, func() g.Node {
				return ConfirmButton(
					appBase+"/cms/components/"+cs.Name+"/delete",
					"POST",
					"",
					"Are you sure you want to delete this component schema?",
					"text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20",
					lucide.Trash2(Class("size-4")),
				)
			}()),
		),
	)
}

// =============================================================================
// Create Component Schema Page
// =============================================================================

// CreateComponentSchemaPage renders the create component schema form
func CreateComponentSchemaPage(
	currentApp *app.App,
	basePath string,
	errorMsg string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-6 max-w-4xl"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: "Component Schemas", Href: appBase + "/cms/components"},
			BreadcrumbItem{Label: "Create", Href: ""},
		),

		// Header
		PageHeader(
			"Create Component Schema",
			"Define a reusable schema for nested object fields",
		),

		// Form
		Card(
			Div(
				Class("p-6"),
				g.If(errorMsg != "", func() g.Node {
					return Div(
						Class("mb-4 p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"),
						g.Text(errorMsg),
					)
				}()),

				FormEl(
					Method("POST"),
					Action(appBase+"/cms/components/create"),
					Class("space-y-6"),
					g.Attr("x-data", componentSchemaFormData()),

					// Basic info section
					Div(
						Class("space-y-4"),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Basic Information")),

						// Name and Slug
						Div(
							Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
							formFieldWithAlpine("name", "Name", "text", "", "e.g., Address, Contact Info", true, "", "name", "@input", "updateSlug()"),
							formFieldWithAlpine("slug", "Name", "text", "", "e.g., address, contact-info", true, "URL-friendly identifier", "slug", "@input", "slugManuallyEdited = true"),
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
								Placeholder("Describe what this component represents..."),
								Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
						),

						// Icon
						formField("icon", "Icon", "text", "", "Optional emoji icon, e.g., 📍, 👤", false, ""),
					),

					// Fields section
					Div(
						Class("space-y-4 pt-6 border-t border-slate-200 dark:border-gray-800"),
						Div(
							Class("flex items-center justify-between"),
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Fields")),
							Button(
								Type("button"),
								g.Attr("@click", "addField()"),
								Class("inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-violet-600 bg-violet-50 rounded-lg hover:bg-violet-100 dark:bg-violet-900/30 dark:text-violet-400 dark:hover:bg-violet-900/50"),
								lucide.Plus(Class("size-4")),
								g.Text("Add Field"),
							),
						),

						// Fields list
						Div(
							g.Attr("x-show", "fields.length > 0"),
							Class("space-y-4"),
							Template(
								g.Attr("x-for", "(field, index) in fields"),
								g.Attr(":key", "index"),
								nestedFieldEditor(),
							),
						),

						// Empty state
						Div(
							g.Attr("x-show", "fields.length === 0"),
							Class("p-8 border-2 border-dashed border-slate-300 rounded-lg text-center dark:border-gray-700"),
							P(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text("No fields defined yet. Click \"Add Field\" to add your first field.")),
						),

						// Hidden input for fields JSON
						Input(
							Type("hidden"),
							Name("fields"),
							g.Attr(":value", "JSON.stringify(fields)"),
						),
					),

					// Submit buttons
					Div(
						Class("flex items-center gap-4 pt-6 border-t border-slate-200 dark:border-gray-800"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
							g.Text("Create Component Schema"),
						),
						A(
							Href(appBase+"/cms/components"),
							Class("px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
	)
}

// =============================================================================
// Edit Component Schema Page
// =============================================================================

// EditComponentSchemaPage renders the edit component schema form
func EditComponentSchemaPage(
	currentApp *app.App,
	basePath string,
	component *core.ComponentSchemaDTO,
	errorMsg string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	// Serialize fields to JSON for Alpine.js
	fieldsJSON, _ := json.Marshal(component.Fields)

	return Div(
		Class("space-y-6 max-w-4xl"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: "Component Schemas", Href: appBase + "/cms/components"},
			BreadcrumbItem{Label: component.Name, Href: ""},
		),

		// Header
		PageHeader(
			"Edit Component Schema",
			fmt.Sprintf("Edit the %s component schema", component.Name),
			g.If(component.UsageCount == 0, func() g.Node {
				return ConfirmButton(
					appBase+"/cms/components/"+component.Name+"/delete",
					"POST",
					"Delete",
					"Are you sure you want to delete this component schema?",
					"bg-red-600 text-white hover:bg-red-700",
					lucide.Trash2(Class("size-4")),
				)
			}()),
		),

		// Usage warning
		g.If(component.UsageCount > 0, func() g.Node {
			return Div(
				Class("p-4 bg-yellow-50 border border-yellow-200 rounded-lg dark:bg-yellow-900/20 dark:border-yellow-800"),
				Div(
					Class("flex items-start gap-3"),
					lucide.TriangleAlert(Class("size-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5")),
					Div(
						P(
							Class("text-sm font-medium text-yellow-800 dark:text-yellow-300"),
							g.Text(fmt.Sprintf("This component is used by %d field(s)", component.UsageCount)),
						),
						P(
							Class("text-sm text-yellow-700 dark:text-yellow-400 mt-1"),
							g.Text("Changes to this component will affect all content types using it."),
						),
					),
				),
			)
		}()),

		// Form
		Card(
			Div(
				Class("p-6"),
				g.If(errorMsg != "", func() g.Node {
					return Div(
						Class("mb-4 p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"),
						g.Text(errorMsg),
					)
				}()),

				FormEl(
					Method("POST"),
					Action(appBase+"/cms/components/"+component.Name),
					Class("space-y-6"),
					g.Attr("x-data", componentSchemaFormDataWithValues(component.Name, component.Name, string(fieldsJSON))),

					// Basic info section
					Div(
						Class("space-y-4"),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Basic Information")),

						// Name and Slug (slug is read-only on edit)
						Div(
							Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
							formField("name", "Name", "text", component.Name, "e.g., Address, Contact Info", true, ""),
							Div(
								Label(
									For("slug"),
									Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
									g.Text("Name"),
								),
								Input(
									Type("text"),
									ID("slug"),
									Name("slug"),
									Value(component.Name),
									Disabled(),
									Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-slate-100 dark:bg-gray-800 dark:border-gray-700 dark:text-gray-400 cursor-not-allowed"),
								),
								P(Class("mt-1 text-xs text-slate-500 dark:text-gray-500"), g.Text("Slug cannot be changed after creation")),
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
								Placeholder("Describe what this component represents..."),
								Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
								g.Text(component.Description),
							),
						),

						// Icon
						formField("icon", "Icon", "text", component.Icon, "Optional emoji icon, e.g., 📍, 👤", false, ""),
					),

					// Fields section
					Div(
						Class("space-y-4 pt-6 border-t border-slate-200 dark:border-gray-800"),
						Div(
							Class("flex items-center justify-between"),
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Fields")),
							Button(
								Type("button"),
								g.Attr("@click", "addField()"),
								Class("inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-violet-600 bg-violet-50 rounded-lg hover:bg-violet-100 dark:bg-violet-900/30 dark:text-violet-400 dark:hover:bg-violet-900/50"),
								lucide.Plus(Class("size-4")),
								g.Text("Add Field"),
							),
						),

						// Fields list
						Div(
							g.Attr("x-show", "fields.length > 0"),
							Class("space-y-4"),
							Template(
								g.Attr("x-for", "(field, index) in fields"),
								g.Attr(":key", "index"),
								nestedFieldEditor(),
							),
						),

						// Empty state
						Div(
							g.Attr("x-show", "fields.length === 0"),
							Class("p-8 border-2 border-dashed border-slate-300 rounded-lg text-center dark:border-gray-700"),
							P(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text("No fields defined yet. Click \"Add Field\" to add your first field.")),
						),

						// Hidden input for fields JSON
						Input(
							Type("hidden"),
							Name("fields"),
							g.Attr(":value", "JSON.stringify(fields)"),
						),
					),

					// Submit buttons
					Div(
						Class("flex items-center gap-4 pt-6 border-t border-slate-200 dark:border-gray-800"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
							g.Text("Save Changes"),
						),
						A(
							Href(appBase+"/cms/components"),
							Class("px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
	)
}

// =============================================================================
// Alpine.js Data and Field Editor
// =============================================================================

// componentSchemaFormData returns the Alpine.js data for the component schema form
func componentSchemaFormData() string {
	return `{
		name: '',
		slug: '',
		slugManuallyEdited: false,
		fields: [],
		
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
		
		addField() {
			this.fields.push({
				name: '',
				slug: '',
				type: 'text',
				required: false,
				description: ''
			});
		},
		
		removeField(index) {
			this.fields.splice(index, 1);
		},
		
		updateFieldSlug(index) {
			const field = this.fields[index];
			if (field.name && !field.slugManuallyEdited) {
				field.slug = this.generateSlug(field.name);
			}
		}
	}`
}

// componentSchemaFormDataWithValues returns Alpine.js data with initial values
func componentSchemaFormDataWithValues(name, slug, fieldsJSON string) string {
	return fmt.Sprintf(`{
		name: '%s',
		slug: '%s',
		slugManuallyEdited: true,
		fields: %s,
		
		generateSlug(name) {
			return name
				.toLowerCase()
				.trim()
				.replace(/[^\w\s-]/g, '')
				.replace(/[\s_-]+/g, '-')
				.replace(/^-+|-+$/g, '');
		},
		
		updateSlug() {},
		
		addField() {
			this.fields.push({
				name: '',
				slug: '',
				type: 'text',
				required: false,
				description: ''
			});
		},
		
		removeField(index) {
			this.fields.splice(index, 1);
		},
		
		updateFieldSlug(index) {
			const field = this.fields[index];
			if (field.name && !field.slugManuallyEdited) {
				field.slug = this.generateSlug(field.name);
			}
		}
	}`, name, slug, fieldsJSON)
}

// nestedFieldEditor renders the field editor for a single nested field
func nestedFieldEditor() g.Node {
	return Div(
		Class("p-4 border border-slate-200 rounded-lg dark:border-gray-700 bg-slate-50 dark:bg-gray-800/50"),

		// Field header with remove button
		Div(
			Class("flex items-center justify-between mb-4"),
			Span(
				Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Field "),
				Span(g.Attr("x-text", "index + 1")),
			),
			Button(
				Type("button"),
				g.Attr("@click", "removeField(index)"),
				Class("p-1 text-red-600 hover:bg-red-50 rounded dark:hover:bg-red-900/30"),
				lucide.X(Class("size-4")),
			),
		),

		// Field form
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 gap-4"),

			// Name
			Div(
				Label(
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
					g.Text("Name"),
					Span(Class("text-red-500 ml-1"), g.Text("*")),
				),
				Input(
					Type("text"),
					g.Attr("x-model", "field.name"),
					g.Attr("@input", "updateFieldSlug(index)"),
					Placeholder("e.g., Street Address"),
					Required(),
					Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				),
			),

			// Slug
			Div(
				Label(
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
					g.Text("Name"),
					Span(Class("text-red-500 ml-1"), g.Text("*")),
				),
				Input(
					Type("text"),
					g.Attr("x-model", "field.slug"),
					g.Attr("@input", "field.slugManuallyEdited = true"),
					Placeholder("e.g., street-address"),
					Required(),
					Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				),
			),

			// Type
			Div(
				Label(
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
					g.Text("Type"),
					Span(Class("text-red-500 ml-1"), g.Text("*")),
				),
				Select(
					g.Attr("x-model", "field.type"),
					Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					Option(Value("text"), g.Text("Text")),
					Option(Value("textarea"), g.Text("Textarea")),
					Option(Value("number"), g.Text("Number")),
					Option(Value("integer"), g.Text("Integer")),
					Option(Value("float"), g.Text("Float")),
					Option(Value("boolean"), g.Text("Boolean")),
					Option(Value("date"), g.Text("Date")),
					Option(Value("datetime"), g.Text("Date & Time")),
					Option(Value("email"), g.Text("Email")),
					Option(Value("url"), g.Text("URL")),
					Option(Value("phone"), g.Text("Phone")),
					Option(Value("color"), g.Text("Color")),
					Option(Value("select"), g.Text("Select")),
				),
			),

			// Required
			Div(
				Class("flex items-center"),
				Label(
					Class("flex items-center gap-2 cursor-pointer"),
					Input(
						Type("checkbox"),
						g.Attr("x-model", "field.required"),
						Class("w-4 h-4 text-violet-600 border-slate-300 rounded focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
					),
					Span(Class("text-sm text-slate-700 dark:text-gray-300"), g.Text("Required")),
				),
			),
		),

		// Description (full width)
		Div(
			Class("mt-4"),
			Label(
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
				g.Text("Description"),
			),
			Input(
				Type("text"),
				g.Attr("x-model", "field.description"),
				Placeholder("Optional description for this field"),
				Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
			),
		),
	)
}

// =============================================================================
// Component Schema Selector (for field creation forms)
// =============================================================================

// ComponentSchemaSelector renders a select dropdown for choosing a component schema
func ComponentSchemaSelector(components []*core.ComponentSchemaSummaryDTO, selectedName string) g.Node {
	options := make([]g.Node, len(components)+1)
	options[0] = Option(Value(""), g.Text("-- Select a component schema --"))

	for i, cs := range components {
		options[i+1] = Option(
			Value(cs.Name),
			g.If(cs.Name == selectedName, Selected()),
			g.Text(fmt.Sprintf("%s (%d fields)", cs.Name, cs.FieldCount)),
		)
	}

	return Select(
		Name("componentRef"),
		ID("componentRef"),
		Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
		g.Group(options),
	)
}

