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
// Entries List Page
// =============================================================================

// EntriesListPage renders the entries list page for a content type
func EntriesListPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entries []*core.ContentEntryDTO,
	stats *core.ContentTypeStatsDTO,
	page, pageSize, totalItems int,
	searchQuery, statusFilter string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Slug
	totalPages := (totalItems + pageSize - 1) / pageSize

	// Build stats node only if stats is not nil
	var statsNode g.Node
	if stats != nil {
		statsNode = entriesStats(stats)
	}

	return Div(
		Class("space-y-6"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: typeBase},
			BreadcrumbItem{Label: "Entries", Href: ""},
		),

		// Header
		PageHeader(
			contentType.Name,
			fmt.Sprintf("Manage %s entries", contentType.Name),
			PrimaryButton(typeBase+"/entries/create", "Create Entry", lucide.Plus(Class("size-4"))),
		),

		// Stats
		statsNode,

		// Filters
		entriesFilters(typeBase, searchQuery, statusFilter),

		// Table
		entriesTable(appBase, contentType, entries, page, totalPages),
	)
}

// entriesStats renders entry statistics
func entriesStats(stats *core.ContentTypeStatsDTO) g.Node {
	return Div(
		Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
		StatCard("Total", fmt.Sprintf("%d", stats.TotalEntries), lucide.FileText(Class("size-5")), "text-blue-600"),
		StatCard("Published", fmt.Sprintf("%d", stats.PublishedEntries), lucide.CircleCheck(Class("size-5")), "text-green-600"),
		StatCard("Drafts", fmt.Sprintf("%d", stats.DraftEntries), lucide.Pencil(Class("size-5")), "text-yellow-600"),
		StatCard("Archived", fmt.Sprintf("%d", stats.ArchivedEntries), lucide.Archive(Class("size-5")), "text-gray-600"),
	)
}

// entriesFilters renders search and filter controls
func entriesFilters(typeBase, searchQuery, statusFilter string) g.Node {
	return Div(
		Class("flex flex-wrap gap-3"),

		// Search
		SearchInput("Search entries...", searchQuery, typeBase+"/entries"),

		// Status filter
		Div(
			Class("min-w-[150px]"),
			Select(
				Name("status"),
				Class("block w-full py-2 px-3 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				Option(Value(""), g.If(statusFilter == "", Selected()), g.Text("All Statuses")),
				Option(Value("published"), g.If(statusFilter == "published", Selected()), g.Text("Published")),
				Option(Value("draft"), g.If(statusFilter == "draft", Selected()), g.Text("Draft")),
				Option(Value("scheduled"), g.If(statusFilter == "scheduled", Selected()), g.Text("Scheduled")),
				Option(Value("archived"), g.If(statusFilter == "archived", Selected()), g.Text("Archived")),
			),
		),
	)
}

// entriesTable renders the entries table
func entriesTable(appBase string, contentType *core.ContentTypeDTO, entries []*core.ContentEntryDTO, page, totalPages int) g.Node {
	typeBase := appBase + "/cms/types/" + contentType.Slug

	if len(entries) == 0 {
		return Card(
			EmptyState(
				lucide.FileText(Class("size-8 text-slate-400")),
				"No entries yet",
				fmt.Sprintf("Create your first %s entry to get started.", contentType.Name),
				"Create Entry",
				typeBase+"/entries/create",
			),
		)
	}

	rows := make([]g.Node, len(entries))
	for i, entry := range entries {
		rows[i] = entryRow(typeBase, contentType, entry)
	}

	return Div(
		Card(DataTable(
			[]string{"Entry", "Status", "Version", "Updated", "Actions"},
			rows,
		)),
		Pagination(page, totalPages, typeBase+"/entries"),
	)
}

// entryRow renders a single entry row
func entryRow(typeBase string, contentType *core.ContentTypeDTO, entry *core.ContentEntryDTO) g.Node {
	// Get title field value if specified
	title := entry.ID
	if contentType.Settings.TitleField != "" {
		if val, ok := entry.Data[contentType.Settings.TitleField]; ok {
			if s, ok := val.(string); ok && s != "" {
				title = s
			}
		}
	}

	// Get description field value if specified
	description := ""
	if contentType.Settings.DescriptionField != "" {
		if val, ok := entry.Data[contentType.Settings.DescriptionField]; ok {
			if s, ok := val.(string); ok {
				description = s
				if len(description) > 100 {
					description = description[:100] + "..."
				}
			}
		}
	}

	return TableRow(
		// Entry info
		TableCell(Div(
			Div(Class("font-medium"), g.Text(title)),
			g.If(description != "", func() g.Node {
				return Div(
					Class("text-xs text-slate-500 dark:text-gray-500 truncate max-w-md"),
					g.Text(description),
				)
			}()),
		)),

		// Status
		TableCell(StatusBadge(entry.Status)),

		// Version
		TableCellSecondary(g.Text(fmt.Sprintf("v%d", entry.Version))),

		// Updated
		TableCellSecondary(g.Text(FormatTimeAgo(entry.UpdatedAt))),

		// Actions
		TableCellActions(
			IconButton(typeBase+"/entries/"+entry.ID, lucide.Eye(Class("size-4")), "View", "text-slate-600"),
			IconButton(typeBase+"/entries/"+entry.ID+"/edit", lucide.Pencil(Class("size-4")), "Edit", "text-blue-600"),
			g.If(entry.Status == "draft", func() g.Node {
				return IconButton(typeBase+"/entries/"+entry.ID+"/publish", lucide.Send(Class("size-4")), "Publish", "text-green-600")
			}()),
		),
	)
}

// =============================================================================
// Entry Detail Page
// =============================================================================

// EntryDetailPage renders the entry detail/view page
func EntryDetailPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	revisions []*core.ContentRevisionDTO,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Slug
	entryBase := typeBase + "/entries/" + entry.ID

	// Get title
	title := entry.ID
	if contentType.Settings.TitleField != "" {
		if val, ok := entry.Data[contentType.Settings.TitleField]; ok {
			if s, ok := val.(string); ok && s != "" {
				title = s
			}
		}
	}

	return Div(
		Class("space-y-6"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: typeBase},
			BreadcrumbItem{Label: "Entries", Href: typeBase + "/entries"},
			BreadcrumbItem{Label: title, Href: ""},
		),

		// Header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"),
			Div(
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(title),
				),
				Div(
					Class("flex items-center gap-3 mt-2"),
					StatusBadge(entry.Status),
					Span(
						Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text(fmt.Sprintf("Version %d", entry.Version)),
					),
					Span(
						Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Updated "+FormatTimeAgo(entry.UpdatedAt)),
					),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				SecondaryButton(entryBase+"/edit", "Edit", lucide.Pencil(Class("size-4"))),
				g.If(entry.Status == "draft", func() g.Node {
					return PrimaryButton(entryBase+"/publish", "Publish", lucide.Send(Class("size-4")))
				}()),
				g.If(entry.Status == "published", func() g.Node {
					return SecondaryButton(entryBase+"/unpublish", "Unpublish", lucide.X(Class("size-4")))
				}()),
			),
		),

		// Content
		Div(
			Class("grid grid-cols-1 lg:grid-cols-3 gap-6"),

			// Main content
			Div(
				Class("lg:col-span-2 space-y-6"),
				entryDataCard(contentType, entry),
			),

			// Sidebar
			Div(
				Class("space-y-6"),
				entryMetadataCard(entry),
				g.If(len(revisions) > 0, func() g.Node {
					return entryRevisionsCard(entryBase, revisions)
				}()),
			),
		),
	)
}

// entryDataCard renders the entry data
func entryDataCard(contentType *core.ContentTypeDTO, entry *core.ContentEntryDTO) g.Node {
	// Build field rows
	rows := make([]g.Node, 0)

	for _, field := range contentType.Fields {
		value, exists := entry.Data[field.Slug]
		if !exists {
			value = nil
		}
		rows = append(rows, entryFieldRow(field, value))
	}

	return CardWithHeader("Entry Data", nil, g.Group(rows))
}

// entryFieldRow renders a single field value row
func entryFieldRow(field *core.ContentFieldDTO, value any) g.Node {
	return Div(
		Class("py-3 border-b border-slate-100 dark:border-gray-800 last:border-b-0"),
		Div(
			Class("flex items-start gap-4"),
			// Field info
			Div(
				Class("w-1/3"),
				Div(
					Class("flex items-center gap-2"),
					fieldTypeIcon(field.Type),
					Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text(field.Name)),
				),
				Code(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text(field.Slug),
				),
			),
			// Value
			Div(
				Class("w-2/3"),
				renderFieldValue(field.Type, value),
			),
		),
	)
}

// renderFieldValue renders a field value based on its type
func renderFieldValue(fieldType string, value any) g.Node {
	if value == nil {
		return Span(
			Class("text-sm text-slate-400 dark:text-gray-600 italic"),
			g.Text("(empty)"),
		)
	}

	switch fieldType {
	case "boolean":
		if b, ok := value.(bool); ok {
			if b {
				return Badge("Yes", "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400")
			}
			return Badge("No", "bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-400")
		}
	case "richtext", "markdown":
		if s, ok := value.(string); ok {
			return Div(
				Class("prose prose-sm dark:prose-invert max-w-none"),
				g.Raw(s),
			)
		}
	case "json":
		if m, ok := value.(map[string]any); ok {
			return Pre(
				Class("text-xs bg-slate-100 dark:bg-gray-800 p-2 rounded overflow-x-auto"),
				Code(g.Text(fmt.Sprintf("%v", m))),
			)
		}
	case "media", "image":
		if s, ok := value.(string); ok {
			return Img(
				Src(s),
				Alt("Media"),
				Class("max-w-xs rounded"),
			)
		}
	case "relation":
		if s, ok := value.(string); ok {
			return Badge(s, "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400")
		}
		if arr, ok := value.([]any); ok {
			badges := make([]g.Node, len(arr))
			for i, v := range arr {
				badges[i] = Badge(fmt.Sprintf("%v", v), "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400")
			}
			return Div(Class("flex flex-wrap gap-1"), g.Group(badges))
		}
	}

	// Default: render as text
	return Span(
		Class("text-sm text-slate-900 dark:text-white"),
		g.Text(fmt.Sprintf("%v", value)),
	)
}

// entryMetadataCard renders entry metadata
func entryMetadataCard(entry *core.ContentEntryDTO) g.Node {
	rows := []g.Node{
		metadataRow("ID", Code(Class("text-xs"), g.Text(entry.ID))),
		metadataRow("Status", StatusBadge(entry.Status)),
		metadataRow("Version", g.Text(fmt.Sprintf("%d", entry.Version))),
		metadataRow("Created", g.Text(FormatTime(entry.CreatedAt))),
		metadataRow("Updated", g.Text(FormatTime(entry.UpdatedAt))),
	}

	if entry.PublishedAt != nil {
		rows = append(rows, metadataRow("Published", g.Text(FormatTime(*entry.PublishedAt))))
	}

	return CardWithHeader("Metadata", nil,
		Div(
			Class("space-y-3"),
			g.Group(rows),
		),
	)
}

// metadataRow renders a metadata row
func metadataRow(label string, value g.Node) g.Node {
	return Div(
		Class("flex items-center justify-between py-1"),
		Span(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(label)),
		value,
	)
}

// entryRevisionsCard renders revision history
func entryRevisionsCard(entryBase string, revisions []*core.ContentRevisionDTO) g.Node {
	items := make([]g.Node, 0, 5)
	for i, rev := range revisions {
		if i >= 5 {
			break
		}
		items = append(items, Div(
			Class("flex items-center justify-between py-2 border-b border-slate-100 dark:border-gray-800 last:border-b-0"),
			Div(
				Span(Class("text-sm font-medium"), g.Text(fmt.Sprintf("v%d", rev.Version))),
				g.If(rev.ChangeReason != "", func() g.Node {
					return Span(Class("text-xs text-slate-500 dark:text-gray-500 ml-2"), g.Text(rev.ChangeReason))
				}()),
			),
			Span(Class("text-xs text-slate-400"), g.Text(FormatTimeAgo(rev.CreatedAt))),
		))
	}

	return CardWithHeader(
		"Revisions",
		[]g.Node{
			A(
				Href(entryBase+"/revisions"),
				Class("text-xs text-violet-600 hover:text-violet-700 dark:text-violet-400"),
				g.Text("View all"),
			),
		},
		g.Group(items),
	)
}

// =============================================================================
// Create/Edit Entry Page
// =============================================================================

// CreateEntryPage renders the create entry form
func CreateEntryPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	err string,
) g.Node {
	return entryForm(currentApp, basePath, contentType, nil, err, "Create Entry")
}

// EditEntryPage renders the edit entry form
func EditEntryPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	err string,
) g.Node {
	return entryForm(currentApp, basePath, contentType, entry, err, "Edit Entry")
}

// entryForm renders the entry form (used for both create and edit)
func entryForm(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	err, title string,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Slug

	isEdit := entry != nil
	formAction := typeBase + "/entries/create"
	if isEdit {
		formAction = typeBase + "/entries/" + entry.ID + "/update"
	}

	return Div(
		Class("space-y-6 max-w-3xl"),

		// Breadcrumbs
		Breadcrumbs(
			BreadcrumbItem{Label: "Content", Href: appBase + "/cms"},
			BreadcrumbItem{Label: contentType.Name, Href: typeBase},
			BreadcrumbItem{Label: "Entries", Href: typeBase + "/entries"},
			BreadcrumbItem{Label: title, Href: ""},
		),

		// Header
		PageHeader(title, fmt.Sprintf("%s for %s", title, contentType.Name)),

		// Error message
		g.If(err != "", func() g.Node {
			return Div(
				Class("p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"),
				g.Text(err),
			)
		}()),

		// Form
		Card(
			Div(
				Class("p-6"),
				FormEl(
					Method("POST"),
					Action(formAction),
					Class("space-y-6"),

					// Dynamic fields based on content type
					g.Group(func() []g.Node {
						fields := make([]g.Node, len(contentType.Fields))
						for i, field := range contentType.Fields {
							var value any
							if entry != nil {
								value = entry.Data[field.Slug]
							}
							fields[i] = entryFormField(field, value)
						}
						return fields
					}()),

					// Status (only for edit)
					g.If(isEdit && entry != nil, func() g.Node {
						return Div(
							Label(
								For("status"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
								g.Text("Status"),
							),
							Select(
								ID("status"),
								Name("status"),
								Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
								Option(Value("draft"), g.If(entry != nil && entry.Status == "draft", Selected()), g.Text("Draft")),
								Option(Value("published"), g.If(entry != nil && entry.Status == "published", Selected()), g.Text("Published")),
								Option(Value("archived"), g.If(entry != nil && entry.Status == "archived", Selected()), g.Text("Archived")),
							),
						)
					}()),

					// Submit buttons
					Div(
						Class("flex items-center gap-4 pt-4"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
							g.If(isEdit, g.Text("Save Changes")),
							g.If(!isEdit, g.Text("Create Entry")),
						),
						A(
							Href(typeBase+"/entries"),
							Class("px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
	)
}

// entryFormField renders a form field based on field type
func entryFormField(field *core.ContentFieldDTO, value any) g.Node {
	fieldName := "data[" + field.Slug + "]"
	valueStr := ""
	if value != nil {
		valueStr = fmt.Sprintf("%v", value)
	}

	return Div(
		Label(
			For(fieldName),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(field.Name),
			g.If(field.Required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),

		// Render appropriate input based on field type
		g.Group(func() []g.Node {
			switch field.Type {
			case "boolean":
				return []g.Node{
					Div(
						Class("flex items-center gap-2"),
						Input(
							Type("checkbox"),
							ID(fieldName),
							Name(fieldName),
							Value("true"),
							g.If(value == true, Checked()),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Yes")),
					),
				}
			case "richtext", "markdown", "text":
				return []g.Node{
					Textarea(
						ID(fieldName),
						Name(fieldName),
						Rows("6"),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Text(valueStr),
					),
				}
			case "select":
				opts := []g.Node{Option(Value(""), g.Text("Select..."))}
				if field.Options.Choices != nil {
					for _, choice := range field.Options.Choices {
						opts = append(opts, Option(
							Value(choice.Value),
							g.If(valueStr == choice.Value, Selected()),
							g.Text(choice.Label),
						))
					}
				}
				return []g.Node{
					Select(
						ID(fieldName),
						Name(fieldName),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Group(opts),
					),
				}
			case "number", "integer", "float":
				// Build input attributes
				inputAttrs := []g.Node{
					Type("number"),
					ID(fieldName),
					Name(fieldName),
					Value(valueStr),
					Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				}
				if field.Required {
					inputAttrs = append(inputAttrs, Required())
				}
				if field.Options.Min != nil {
					inputAttrs = append(inputAttrs, g.Attr("min", fmt.Sprintf("%v", *field.Options.Min)))
				}
				if field.Options.Max != nil {
					inputAttrs = append(inputAttrs, g.Attr("max", fmt.Sprintf("%v", *field.Options.Max)))
				}
				if field.Options.Step != nil {
					inputAttrs = append(inputAttrs, g.Attr("step", fmt.Sprintf("%v", *field.Options.Step)))
				}
				return []g.Node{Input(inputAttrs...)}
			case "date", "datetime":
				inputType := "date"
				if field.Type == "datetime" {
					inputType = "datetime-local"
				}
				return []g.Node{
					Input(
						Type(inputType),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "email":
				return []g.Node{
					Input(
						Type("email"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "url":
				return []g.Node{
					Input(
						Type("url"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "object":
				return []g.Node{
					renderObjectField(field, fieldName, value),
				}
			case "array":
				return []g.Node{
					renderArrayField(field, fieldName, value),
				}
			default:
				// Default text input
				return []g.Node{
					Input(
						Type("text"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			}
		}()),

		// Description
		g.If(field.Description != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(field.Description),
			)
		}()),
	)
}

// =============================================================================
// Nested Object/Array Field Rendering
// =============================================================================

// renderObjectField renders a nested object field with sub-fields
func renderObjectField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Get nested fields from options
	nestedFields := field.Options.NestedFields
	if len(nestedFields) == 0 {
		return Div(
			Class("text-sm text-slate-500 dark:text-gray-400 italic"),
			g.Text("No nested fields defined"),
		)
	}

	// Convert value to map
	valueMap := make(map[string]any)
	if value != nil {
		if m, ok := value.(map[string]any); ok {
			valueMap = m
		}
	}

	// Serialize the current value for Alpine.js
	valueJSON, _ := json.Marshal(valueMap)

	isCollapsible := field.Options.Collapsible
	defaultExpanded := field.Options.DefaultExpanded

	return Div(
		Class("border border-slate-200 rounded-lg dark:border-gray-700"),
		g.Attr("x-data", fmt.Sprintf(`{
			expanded: %v,
			data: %s,
			toggleExpanded() { this.expanded = !this.expanded; }
		}`, defaultExpanded || !isCollapsible, string(valueJSON))),

		// Header (collapsible)
		g.If(isCollapsible, func() g.Node {
			return Div(
				Class("flex items-center justify-between px-4 py-3 bg-slate-50 dark:bg-gray-800/50 cursor-pointer rounded-t-lg"),
				g.Attr("@click", "toggleExpanded()"),
				Div(
					Class("flex items-center gap-2"),
					lucide.Braces(Class("size-4 text-indigo-500")),
					Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Object Fields")),
				),
				Div(
					g.Attr("x-show", "expanded"),
					lucide.ChevronUp(Class("size-4 text-slate-400")),
				),
				Div(
					g.Attr("x-show", "!expanded"),
					lucide.ChevronDown(Class("size-4 text-slate-400")),
				),
			)
		}()),

		// Content
		Div(
			g.If(isCollapsible, g.Attr("x-show", "expanded")),
			Class("p-4 space-y-4"),

			// Nested fields
			g.Group(func() []g.Node {
				fields := make([]g.Node, len(nestedFields))
				for i, nf := range nestedFields {
					subFieldName := fieldName + "[" + nf.Slug + "]"
					subValue := valueMap[nf.Slug]
					fields[i] = renderNestedFieldInput(nf, subFieldName, subValue, 1)
				}
				return fields
			}()),

			// Hidden input to store the full JSON
			Input(
				Type("hidden"),
				Name(fieldName + "_json"),
				g.Attr(":value", "JSON.stringify(data)"),
			),
		),
	)
}

// renderArrayField renders an array of objects with add/remove functionality
func renderArrayField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Get nested fields from options
	nestedFields := field.Options.NestedFields
	if len(nestedFields) == 0 {
		return Div(
			Class("text-sm text-slate-500 dark:text-gray-400 italic"),
			g.Text("No nested fields defined"),
		)
	}

	// Convert value to slice
	var items []map[string]any
	if value != nil {
		if arr, ok := value.([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
	}

	// Serialize the current value for Alpine.js
	itemsJSON, _ := json.Marshal(items)
	if len(items) == 0 {
		itemsJSON = []byte("[]")
	}

	// Build empty item template
	emptyItem := make(map[string]any)
	for _, nf := range nestedFields {
		emptyItem[nf.Slug] = ""
	}
	emptyItemJSON, _ := json.Marshal(emptyItem)

	minItems := 0
	maxItems := -1
	if field.Options.MinItems != nil {
		minItems = *field.Options.MinItems
	}
	if field.Options.MaxItems != nil {
		maxItems = *field.Options.MaxItems
	}

	isCollapsible := field.Options.Collapsible
	defaultExpanded := field.Options.DefaultExpanded

	return Div(
		Class("border border-slate-200 rounded-lg dark:border-gray-700"),
		g.Attr("x-data", fmt.Sprintf(`{
			expanded: %v,
			items: %s,
			emptyItem: %s,
			minItems: %d,
			maxItems: %d,
			toggleExpanded() { this.expanded = !this.expanded; },
			addItem() {
				if (this.maxItems === -1 || this.items.length < this.maxItems) {
					this.items.push(JSON.parse(JSON.stringify(this.emptyItem)));
				}
			},
			removeItem(index) {
				if (this.items.length > this.minItems) {
					this.items.splice(index, 1);
				}
			},
			canAdd() {
				return this.maxItems === -1 || this.items.length < this.maxItems;
			},
			canRemove() {
				return this.items.length > this.minItems;
			}
		}`, defaultExpanded || !isCollapsible, string(itemsJSON), string(emptyItemJSON), minItems, maxItems)),

		// Header
		Div(
			Class("flex items-center justify-between px-4 py-3 bg-slate-50 dark:bg-gray-800/50 rounded-t-lg"),
			Div(
				Class("flex items-center gap-2"),
				g.If(isCollapsible, func() g.Node {
					return Div(
						Class("cursor-pointer"),
						g.Attr("@click", "toggleExpanded()"),
						Div(
							g.Attr("x-show", "expanded"),
							lucide.ChevronUp(Class("size-4 text-slate-400")),
						),
						Div(
							g.Attr("x-show", "!expanded"),
							lucide.ChevronDown(Class("size-4 text-slate-400")),
						),
					)
				}()),
				lucide.List(Class("size-4 text-indigo-500")),
				Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Array Items")),
				Span(
					Class("text-xs text-slate-500 dark:text-gray-400 ml-2"),
					g.Attr("x-text", "items.length + ' item(s)'"),
				),
			),
			Button(
				Type("button"),
				g.Attr("@click", "addItem()"),
				g.Attr(":disabled", "!canAdd()"),
				Class("inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-violet-600 bg-violet-50 rounded hover:bg-violet-100 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-violet-900/30 dark:text-violet-400 dark:hover:bg-violet-900/50"),
				lucide.Plus(Class("size-3")),
				g.Text("Add Item"),
			),
		),

		// Content
		Div(
			g.If(isCollapsible, g.Attr("x-show", "expanded")),
			Class("p-4"),

			// Items list
			Div(
				Class("space-y-4"),
				Template(
					g.Attr("x-for", "(item, index) in items"),
					g.Attr(":key", "index"),
					renderArrayItemEditor(nestedFields, fieldName),
				),
			),

			// Empty state
			Div(
				g.Attr("x-show", "items.length === 0"),
				Class("py-8 text-center"),
				P(
					Class("text-sm text-slate-500 dark:text-gray-400"),
					g.Text("No items yet. Click \"Add Item\" to add your first item."),
				),
			),

			// Hidden input to store the full JSON
			Input(
				Type("hidden"),
				Name(fieldName + "_json"),
				g.Attr(":value", "JSON.stringify(items)"),
			),
		),
	)
}

// renderArrayItemEditor renders the editor for a single array item
func renderArrayItemEditor(nestedFields []core.NestedFieldDefDTO, fieldName string) g.Node {
	return Div(
		Class("relative p-4 border border-slate-200 rounded-lg dark:border-gray-700 bg-slate-50/50 dark:bg-gray-800/30"),

		// Item header with remove button
		Div(
			Class("flex items-center justify-between mb-4"),
			Span(
				Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
				g.Text("Item "),
				Span(g.Attr("x-text", "index + 1")),
			),
			Button(
				Type("button"),
				g.Attr("@click", "removeItem(index)"),
				g.Attr(":disabled", "!canRemove()"),
				Class("p-1 text-red-600 hover:bg-red-50 rounded disabled:opacity-50 disabled:cursor-not-allowed dark:hover:bg-red-900/30"),
				lucide.Trash2(Class("size-4")),
			),
		),

		// Item fields
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
			g.Group(func() []g.Node {
				fields := make([]g.Node, len(nestedFields))
				for i, nf := range nestedFields {
					fields[i] = renderArrayItemField(nf, fieldName, i)
				}
				return fields
			}()),
		),
	)
}

// renderArrayItemField renders a single field within an array item
func renderArrayItemField(field core.NestedFieldDefDTO, fieldName string, fieldIndex int) g.Node {
	// Note: This function creates a template that will be used with x-for
	// The actual field name and value binding happens through Alpine.js

	return Div(
		Label(
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(field.Name),
			g.If(field.Required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),

		// Render input based on type
		g.Group(func() []g.Node {
			xModel := fmt.Sprintf("item.%s", field.Slug)

			switch field.Type {
			case "boolean":
				return []g.Node{
					Div(
						Class("flex items-center gap-2"),
						Input(
							Type("checkbox"),
							g.Attr("x-model", xModel),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Yes")),
					),
				}
			case "textarea", "text":
				return []g.Node{
					Textarea(
						Rows("2"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "number", "integer", "float":
				return []g.Node{
					Input(
						Type("number"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "date":
				return []g.Node{
					Input(
						Type("date"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "datetime":
				return []g.Node{
					Input(
						Type("datetime-local"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "email":
				return []g.Node{
					Input(
						Type("email"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "url":
				return []g.Node{
					Input(
						Type("url"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "color":
				return []g.Node{
					Input(
						Type("color"),
						g.Attr("x-model", xModel),
						Class("block w-12 h-10 rounded border border-slate-300 dark:border-gray-700"),
					),
				}
			case "select":
				opts := []g.Node{Option(Value(""), g.Text("Select..."))}
				if field.Options != nil && field.Options.Choices != nil {
					for _, choice := range field.Options.Choices {
						opts = append(opts, Option(Value(choice.Value), g.Text(choice.Label)))
					}
				}
				return []g.Node{
					Select(
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Group(opts),
					),
				}
			default:
				// Default: text input
				return []g.Node{
					Input(
						Type("text"),
						g.Attr("x-model", xModel),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			}
		}()),

		// Description
		g.If(field.Description != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(field.Description),
			)
		}()),
	)
}

// renderNestedFieldInput renders an input for a nested field (used in object fields)
func renderNestedFieldInput(field core.NestedFieldDefDTO, fieldName string, value any, depth int) g.Node {
	valueStr := ""
	if value != nil {
		valueStr = fmt.Sprintf("%v", value)
	}

	return Div(
		Label(
			For(fieldName),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(field.Name),
			g.If(field.Required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),

		// Render input based on type
		g.Group(func() []g.Node {
			switch field.Type {
			case "boolean":
				checked := false
				if b, ok := value.(bool); ok {
					checked = b
				}
				return []g.Node{
					Div(
						Class("flex items-center gap-2"),
						Input(
							Type("checkbox"),
							ID(fieldName),
							Name(fieldName),
							Value("true"),
							g.If(checked, Checked()),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						),
						Span(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Yes")),
					),
				}
			case "textarea", "text":
				return []g.Node{
					Textarea(
						ID(fieldName),
						Name(fieldName),
						Rows("3"),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Text(valueStr),
					),
				}
			case "number", "integer", "float":
				return []g.Node{
					Input(
						Type("number"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "date":
				return []g.Node{
					Input(
						Type("date"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "datetime":
				return []g.Node{
					Input(
						Type("datetime-local"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "email":
				return []g.Node{
					Input(
						Type("email"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "url":
				return []g.Node{
					Input(
						Type("url"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "color":
				return []g.Node{
					Input(
						Type("color"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						Class("block w-12 h-10 rounded border border-slate-300 dark:border-gray-700"),
					),
				}
			case "select":
				opts := []g.Node{Option(Value(""), g.Text("Select..."))}
				if field.Options != nil && field.Options.Choices != nil {
					for _, choice := range field.Options.Choices {
						opts = append(opts, Option(
							Value(choice.Value),
							g.If(valueStr == choice.Value, Selected()),
							g.Text(choice.Label),
						))
					}
				}
				return []g.Node{
					Select(
						ID(fieldName),
						Name(fieldName),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Group(opts),
					),
				}
			case "object":
				// Nested object - recurse with increased depth
				if depth < 3 && field.Options != nil && len(field.Options.NestedFields) > 0 {
					return []g.Node{
						renderNestedObjectFieldInput(field.Options.NestedFields, fieldName, value, depth+1),
					}
				}
				return []g.Node{
					P(Class("text-xs text-slate-400"), g.Text("Max nesting depth reached")),
				}
			default:
				// Default: text input
				return []g.Node{
					Input(
						Type("text"),
						ID(fieldName),
						Name(fieldName),
						Value(valueStr),
						g.If(field.Required, Required()),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			}
		}()),

		// Description
		g.If(field.Description != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(field.Description),
			)
		}()),
	)
}

// renderNestedObjectFieldInput renders inputs for nested object fields within an object
func renderNestedObjectFieldInput(fields []core.NestedFieldDefDTO, baseName string, value any, depth int) g.Node {
	valueMap := make(map[string]any)
	if value != nil {
		if m, ok := value.(map[string]any); ok {
			valueMap = m
		}
	}

	return Div(
		Class("pl-4 border-l-2 border-slate-200 dark:border-gray-700 space-y-4"),
		g.Group(func() []g.Node {
			nodes := make([]g.Node, len(fields))
			for i, f := range fields {
				subFieldName := baseName + "[" + f.Slug + "]"
				subValue := valueMap[f.Slug]
				nodes[i] = renderNestedFieldInput(f, subFieldName, subValue, depth)
			}
			return nodes
		}()),
	)
}

