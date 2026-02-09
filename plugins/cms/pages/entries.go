package pages

import (
	"encoding/json"
	"fmt"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/cms/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// Entries List Page
// =============================================================================

// EntriesListPage renders the entries list page for a content type.
func EntriesListPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entries []*core.ContentEntryDTO,
	stats *core.ContentTypeStatsDTO,
	page, pageSize, totalItems int,
	searchQuery, statusFilter string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Name
	totalPages := (totalItems + pageSize - 1) / pageSize

	// Build stats node only if stats is not nil
	var statsNode g.Node
	if stats != nil {
		statsNode = entriesStats(stats)
	}

	return Div(
		Class("space-y-2"),

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

// entriesStats renders entry statistics.
func entriesStats(stats *core.ContentTypeStatsDTO) g.Node {
	return Div(
		Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
		StatCard("Total", strconv.Itoa(stats.TotalEntries), lucide.FileText(Class("size-5")), "text-blue-600"),
		StatCard("Published", strconv.Itoa(stats.PublishedEntries), lucide.CircleCheck(Class("size-5")), "text-green-600"),
		StatCard("Drafts", strconv.Itoa(stats.DraftEntries), lucide.Pencil(Class("size-5")), "text-yellow-600"),
		StatCard("Archived", strconv.Itoa(stats.ArchivedEntries), lucide.Archive(Class("size-5")), "text-gray-600"),
	)
}

// entriesFilters renders search and filter controls.
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

// entriesTable renders the entries table.
func entriesTable(appBase string, contentType *core.ContentTypeDTO, entries []*core.ContentEntryDTO, page, totalPages int) g.Node {
	typeBase := appBase + "/cms/types/" + contentType.Name

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

// entryRow renders a single entry row.
func entryRow(typeBase string, contentType *core.ContentTypeDTO, entry *core.ContentEntryDTO) g.Node {
	// Get title field value - defaults to entry ID
	// Priority: configured titleField > common field names (title, label) > entry ID
	title := entry.ID

	// First, try the configured titleField if set
	if contentType.Settings.TitleField != "" {
		if val, ok := entry.Data[contentType.Settings.TitleField]; ok {
			if s, ok := val.(string); ok && s != "" {
				title = s
			}
		}
	} else {
		// If no titleField configured, try common field names
		for _, fieldName := range []string{"title", "Title", "label", "Label"} {
			if val, ok := entry.Data[fieldName]; ok {
				if s, ok := val.(string); ok && s != "" {
					title = s

					break
				}
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

	// Get preview field value if specified
	preview := ""

	if contentType.Settings.PreviewField != "" {
		if val, ok := entry.Data[contentType.Settings.PreviewField]; ok {
			// Handle different types for preview field
			switch v := val.(type) {
			case string:
				preview = v
			case float64, int, int64:
				preview = fmt.Sprintf("%v", v)
			}

			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
		}
	}

	return TableRow(
		// Entry info
		TableCell(Div(
			Div(Class("font-medium"), g.Text(title)),
			g.If(preview != "", func() g.Node {
				return Div(
					Class("text-xs text-slate-400 dark:text-gray-600 font-mono truncate max-w-md mt-1"),
					g.Text(preview),
				)
			}()),
			g.If(description != "", func() g.Node {
				return Div(
					Class("text-xs text-slate-500 dark:text-gray-500 truncate max-w-md mt-1"),
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
			ConfirmButton(
				typeBase+"/entries/"+entry.ID+"/delete",
				"POST",
				"Delete",
				"Are you sure you want to delete this entry? This action cannot be undone.",
				"text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20",
				lucide.Trash2(Class("size-4")),
			),
		),
	)
}

// =============================================================================
// Entry Detail Page
// =============================================================================

// EntryDetailPage renders the entry detail/view page.
func EntryDetailPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	revisions []*core.ContentRevisionDTO,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Name
	entryBase := typeBase + "/entries/" + entry.ID

	// Get title - defaults to entry ID
	// Priority: configured titleField > common field names (title, label) > entry ID
	title := entry.ID

	// First, try the configured titleField if set
	if contentType.Settings.TitleField != "" {
		if val, ok := entry.Data[contentType.Settings.TitleField]; ok {
			if s, ok := val.(string); ok && s != "" {
				title = s
			}
		}
	} else {
		// If no titleField configured, try common field names
		for _, fieldName := range []string{"title", "Title", "label", "Label"} {
			if val, ok := entry.Data[fieldName]; ok {
				if s, ok := val.(string); ok && s != "" {
					title = s

					break
				}
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
				ConfirmButton(
					entryBase+"/delete",
					"POST",
					"Delete",
					"Are you sure you want to delete this entry? This action cannot be undone.",
					"text-red-600 bg-red-50 hover:bg-red-100 dark:bg-red-900/20 dark:hover:bg-red-900/30",
					lucide.Trash2(Class("size-4")),
				),
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

// entryDataCard renders the entry data.
func entryDataCard(contentType *core.ContentTypeDTO, entry *core.ContentEntryDTO) g.Node {
	// Build field rows
	rows := make([]g.Node, 0)

	for _, field := range contentType.Fields {
		value, exists := entry.Data[field.Name]
		if !exists {
			value = nil
		}

		rows = append(rows, entryFieldRow(field, value))
	}

	return CardWithHeader("Entry Data", nil, g.Group(rows))
}

// entryFieldRow renders a single field value row.
func entryFieldRow(field *core.ContentFieldDTO, value any) g.Node {
	// Use Title if available, fallback to Name for display label
	label := field.Name
	if field.Title != "" {
		label = field.Title
	}

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
					Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text(label)),
				),
				Code(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text(field.Name),
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

// renderFieldValue renders a field value based on its type.
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
	case "json", "object":
		// Try to marshal as JSON for proper display
		if jsonBytes, err := json.MarshalIndent(value, "", "  "); err == nil {
			return Pre(
				Class("text-xs bg-slate-100 dark:bg-gray-800 p-2 rounded overflow-x-auto"),
				Code(g.Text(string(jsonBytes))),
			)
		}
		// Fallback to simple string representation
		return Span(
			Class("text-sm text-slate-900 dark:text-white"),
			g.Text(fmt.Sprintf("%v", value)),
		)
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

// entryMetadataCard renders entry metadata.
func entryMetadataCard(entry *core.ContentEntryDTO) g.Node {
	rows := []g.Node{
		metadataRow("ID", Code(Class("text-xs"), g.Text(entry.ID))),
		metadataRow("Status", StatusBadge(entry.Status)),
		metadataRow("Version", g.Text(strconv.Itoa(entry.Version))),
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

// metadataRow renders a metadata row.
func metadataRow(label string, value g.Node) g.Node {
	return Div(
		Class("flex items-center justify-between py-1"),
		Span(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(label)),
		value,
	)
}

// entryRevisionsCard renders revision history.
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

// CreateEntryPage renders the create entry form.
func CreateEntryPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	err string,
) g.Node {
	return entryForm(currentApp, basePath, contentType, nil, err, "Create Entry")
}

// EditEntryPage renders the edit entry form.
func EditEntryPage(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	err string,
) g.Node {
	return entryForm(currentApp, basePath, contentType, entry, err, "Edit Entry")
}

// entryForm renders the entry form (used for both create and edit).
func entryForm(
	currentApp *app.App,
	basePath string,
	contentType *core.ContentTypeDTO,
	entry *core.ContentEntryDTO,
	err, title string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()
	typeBase := appBase + "/cms/types/" + contentType.Name

	isEdit := entry != nil

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

		// Form with Alpine.js state management
		Card(
			Div(
				Class("p-6"),
				FormEl(
					g.Attr("@submit.prevent", fmt.Sprintf(`async (event) => {
						loading = true;
						const formData = new FormData(event.target);
						const data = {};
						for (const [key, value] of formData.entries()) {
							if (key.startsWith('data[') && key.endsWith(']')) {
								const fieldName = key.slice(5, -1);
								data[fieldName] = value;
							}
						}
						
						try {
							%s
							$root.notification.success('%s');
							window.location.href = '%s';
						} catch (err) {
							const errorMsg = err.error?.message || err.message || 'Failed to save entry';
							$root.notification.error(errorMsg);
							loading = false;
						}
					}`, func() string {
						if isEdit {
							return fmt.Sprintf(`await $bridge.call('cms.updateEntry', {
								appId: '%s',
								typeName: '%s',
								entryId: '%s',
								data: data,
								status: formData.get('status') || 'draft'
							});`, currentApp.ID.String(), contentType.Name, entry.ID)
						}

						return fmt.Sprintf(`const result = await $bridge.call('cms.createEntry', {
							appId: '%s',
							typeName: '%s',
							data: data,
							status: 'draft'
						});`, currentApp.ID.String(), contentType.Name)
					}(), func() string {
						if isEdit {
							return "Entry updated successfully"
						}

						return "Entry created successfully"
					}(), typeBase+"/entries")),
					Class("space-y-6"),
					// Alpine.js form-level state for reactive field management
					g.Attr("x-data", buildFormAlpineData(contentType, entry)),

					// Dynamic fields based on content type (skip hidden fields)
					g.Group(func() []g.Node {
						var fields []g.Node

						for _, field := range contentType.Fields {
							// Skip hidden fields in the form
							if field.Hidden {
								continue
							}

							var value any
							if entry != nil {
								value = entry.Data[field.Name]
							}

							fields = append(fields, entryFormField(field, value))
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
							g.Attr(":disabled", "loading"),
							Class("px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"),
							g.El("span",
								g.Attr("x-show", "!loading"),
								g.If(isEdit, g.Text("Save Changes")),
								g.If(!isEdit, g.Text("Create Entry")),
							),
							g.El("span",
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
								g.If(isEdit, g.Text("Saving...")),
								g.If(!isEdit, g.Text("Creating...")),
							),
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

// buildFormAlpineData builds the Alpine.js data object for the entry form.
func buildFormAlpineData(contentType *core.ContentTypeDTO, entry *core.ContentEntryDTO) string {
	// Build initial state from entry data
	formData := make(map[string]any)
	if entry != nil && entry.Data != nil {
		formData = entry.Data
	}

	// Serialize form data for Alpine
	formDataJSON, _ := json.Marshal(formData)

	return fmt.Sprintf(`{
		formData: %s,
		loading: false,
		init() {
			// Initialize any watchers or setup needed
		}
	}`, string(formDataJSON))
}

// entryFormField renders a form field based on field type.
func entryFormField(field *core.ContentFieldDTO, value any) g.Node {
	fieldName := "data[" + field.Name + "]"

	valueStr := ""
	if value != nil {
		valueStr = fmt.Sprintf("%v", value)
	}

	// Build conditional visibility attributes
	hasConditional := field.Options.ShowWhen != nil || field.Options.HideWhen != nil
	conditionalAttrs := buildConditionalVisibilityAttrs(field)

	// Use Title if available, fallback to Name
	label := field.Name
	if field.Title != "" {
		label = field.Title
	}

	// The actual field content
	fieldContent := Div(
		Label(
			For(fieldName),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
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
			case "richtext", "markdown", "textarea":
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
			case "text":
				// Build text input with optional constraints
				inputAttrs := []g.Node{
					Type("text"),
					ID(fieldName),
					Name(fieldName),
					Value(valueStr),
					Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				}
				if field.Required {
					inputAttrs = append(inputAttrs, Required())
				}

				if field.Options.MinLength > 0 {
					inputAttrs = append(inputAttrs, g.Attr("minlength", strconv.Itoa(field.Options.MinLength)))
				}

				if field.Options.MaxLength > 0 {
					inputAttrs = append(inputAttrs, g.Attr("maxlength", strconv.Itoa(field.Options.MaxLength)))
				}

				if field.Options.Pattern != "" {
					inputAttrs = append(inputAttrs, g.Attr("pattern", field.Options.Pattern))
				}

				return []g.Node{Input(inputAttrs...)}
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
						// Bind to Alpine.js state for reactivity
						g.Attr("x-model", fmt.Sprintf("formData['%s']", field.Name)),
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
			case "slug":
				return []g.Node{
					renderSlugField(field, fieldName, valueStr),
				}
			case "object":
				return []g.Node{
					renderObjectField(field, fieldName, value),
				}
			case "array":
				return []g.Node{
					renderArrayField(field, fieldName, value),
				}
			case "oneOf":
				return []g.Node{
					renderOneOfField(field, fieldName, value),
				}
			case "json":
				return []g.Node{
					renderJsonEditorField(field, fieldName, value),
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

	// Wrap with conditional visibility if needed
	if hasConditional {
		return Div(
			g.Group(conditionalAttrs),
			fieldContent,
		)
	}

	return fieldContent
}

// buildConditionalVisibilityAttrs builds Alpine.js attributes for conditional visibility.
func buildConditionalVisibilityAttrs(field *core.ContentFieldDTO) []g.Node {
	attrs := []g.Node{}

	if field.Options.ShowWhen != nil {
		cond := field.Options.ShowWhen
		attrs = append(attrs,
			g.Attr("x-data", fmt.Sprintf(`{
				conditionField: '%s',
				conditionOp: '%s',
				conditionValue: %s,
				clearWhenHidden: %v,
				fieldName: 'data[%s]',
				
				checkCondition() {
					const fieldInput = document.querySelector('[name="data[' + this.conditionField + ']"]');
					if (!fieldInput) return false;
					const value = fieldInput.type === 'checkbox' ? fieldInput.checked : fieldInput.value;
					return this.evaluateCondition(value);
				},
				
				evaluateCondition(value) {
					switch (this.conditionOp) {
						case 'eq': return value === this.conditionValue;
						case 'ne': return value !== this.conditionValue;
						case 'in': return Array.isArray(this.conditionValue) && this.conditionValue.includes(value);
						case 'notIn': return Array.isArray(this.conditionValue) && !this.conditionValue.includes(value);
						case 'exists': return value !== null && value !== undefined && value !== '';
						case 'notExists': return value === null || value === undefined || value === '';
						default: return true;
					}
				},
				
				visible: false,
				
				init() {
					this.visible = this.checkCondition();
					const fieldInput = document.querySelector('[name="data[' + this.conditionField + ']"]');
					if (fieldInput) {
						fieldInput.addEventListener('change', () => {
							const wasVisible = this.visible;
							this.visible = this.checkCondition();
							if (wasVisible && !this.visible && this.clearWhenHidden) {
								const thisInput = document.querySelector('[name="' + this.fieldName + '"]');
								if (thisInput) thisInput.value = '';
							}
						});
					}
				}
			}`, cond.Field, cond.Operator, conditionValueToJS(cond.Value), field.Options.ClearWhenHidden, field.Name)),
			g.Attr("x-show", "visible"),
		)
	} else if field.Options.HideWhen != nil {
		cond := field.Options.HideWhen
		attrs = append(attrs,
			g.Attr("x-data", fmt.Sprintf(`{
				conditionField: '%s',
				conditionOp: '%s',
				conditionValue: %s,
				clearWhenHidden: %v,
				fieldName: 'data[%s]',
				
				checkCondition() {
					const fieldInput = document.querySelector('[name="data[' + this.conditionField + ']"]');
					if (!fieldInput) return false;
					const value = fieldInput.type === 'checkbox' ? fieldInput.checked : fieldInput.value;
					return this.evaluateCondition(value);
				},
				
				evaluateCondition(value) {
					switch (this.conditionOp) {
						case 'eq': return value === this.conditionValue;
						case 'ne': return value !== this.conditionValue;
						case 'in': return Array.isArray(this.conditionValue) && this.conditionValue.includes(value);
						case 'notIn': return Array.isArray(this.conditionValue) && !this.conditionValue.includes(value);
						case 'exists': return value !== null && value !== undefined && value !== '';
						case 'notExists': return value === null || value === undefined || value === '';
						default: return false;
					}
				},
				
				visible: true,
				
				init() {
					this.visible = !this.checkCondition();
					const fieldInput = document.querySelector('[name="data[' + this.conditionField + ']"]');
					if (fieldInput) {
						fieldInput.addEventListener('change', () => {
							const wasVisible = this.visible;
							this.visible = !this.checkCondition();
							if (wasVisible && !this.visible && this.clearWhenHidden) {
								const thisInput = document.querySelector('[name="' + this.fieldName + '"]');
								if (thisInput) thisInput.value = '';
							}
						});
					}
				}
			}`, cond.Field, cond.Operator, conditionValueToJS(cond.Value), field.Options.ClearWhenHidden, field.Name)),
			g.Attr("x-show", "visible"),
		)
	}

	return attrs
}

// buildConditionExpression builds an Alpine.js expression for a field condition.
func buildConditionExpression(cond *core.FieldConditionDTO) string {
	switch cond.Operator {
	case "eq", "ne", "in", "notIn", "exists", "notExists":
		return "checkCondition()"
	default:
		return "true"
	}
}

// conditionValueToJS converts a condition value to JavaScript representation.
func conditionValueToJS(value any) string {
	if value == nil {
		return "null"
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return "null"
	}

	return string(bytes)
}

// =============================================================================
// Nested Object/Array Field Rendering
// =============================================================================

// renderObjectField renders a nested object field with sub-fields.
func renderObjectField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Get nested fields from options (should be resolved from ComponentRef if applicable)
	nestedFields := field.Options.NestedFields
	if len(nestedFields) == 0 {
		// Show helpful message based on whether ComponentRef was set
		if field.Options.ComponentRef != "" {
			return Div(
				Class("p-3 bg-amber-50 border border-amber-200 rounded-lg dark:bg-amber-900/20 dark:border-amber-800"),
				Div(
					Class("flex items-center gap-2 text-amber-700 dark:text-amber-400"),
					lucide.CircleAlert(Class("size-4")),
					Span(Class("text-sm font-medium"), g.Text("Component schema not found")),
				),
				P(
					Class("mt-1 text-xs text-amber-600 dark:text-amber-500"),
					g.Text(fmt.Sprintf("The component schema '%s' could not be resolved. Please ensure it exists.", field.Options.ComponentRef)),
				),
			)
		}

		return Div(
			Class("p-3 bg-slate-50 border border-slate-200 rounded-lg dark:bg-gray-800/50 dark:border-gray-700"),
			Div(
				Class("flex items-center gap-2 text-slate-600 dark:text-gray-400"),
				lucide.Info(Class("size-4")),
				Span(Class("text-sm"), g.Text("No nested fields defined")),
			),
			P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text("Configure this field with nested fields or a component schema reference."),
			),
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
					subFieldName := fieldName + "[" + nf.Name + "]"
					subValue := valueMap[nf.Name]
					fields[i] = renderNestedFieldInput(nf, subFieldName, subValue, 1)
				}

				return fields
			}()),

			// Hidden input to store the full JSON
			Input(
				Type("hidden"),
				Name(fieldName),
				g.Attr(":value", "JSON.stringify(data)"),
			),
		),
	)
}

// renderArrayField renders an array of objects with add/remove functionality.
func renderArrayField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Get nested fields from options (should be resolved from ComponentRef if applicable)
	nestedFields := field.Options.NestedFields
	if len(nestedFields) == 0 {
		// Show helpful message based on whether ComponentRef was set
		if field.Options.ComponentRef != "" {
			return Div(
				Class("p-3 bg-amber-50 border border-amber-200 rounded-lg dark:bg-amber-900/20 dark:border-amber-800"),
				Div(
					Class("flex items-center gap-2 text-amber-700 dark:text-amber-400"),
					lucide.CircleAlert(Class("size-4")),
					Span(Class("text-sm font-medium"), g.Text("Component schema not found")),
				),
				P(
					Class("mt-1 text-xs text-amber-600 dark:text-amber-500"),
					g.Text(fmt.Sprintf("The component schema '%s' could not be resolved. Please ensure it exists.", field.Options.ComponentRef)),
				),
			)
		}

		return Div(
			Class("p-3 bg-slate-50 border border-slate-200 rounded-lg dark:bg-gray-800/50 dark:border-gray-700"),
			Div(
				Class("flex items-center gap-2 text-slate-600 dark:text-gray-400"),
				lucide.Info(Class("size-4")),
				Span(Class("text-sm"), g.Text("No item schema defined")),
			),
			P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text("Configure this array field with nested fields or a component schema reference for array items."),
			),
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
		emptyItem[nf.Name] = ""
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
				Name(fieldName),
				g.Attr(":value", "JSON.stringify(items)"),
			),
		),
	)
}

// renderOneOfField renders a oneOf (discriminated union) field.
func renderOneOfField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Get discriminator field and schemas from options
	discriminatorField := field.Options.DiscriminatorField
	schemas := field.Options.Schemas
	clearOnChange := field.Options.ClearOnDiscriminatorChange

	if discriminatorField == "" || len(schemas) == 0 {
		return Div(
			Class("p-3 bg-amber-50 border border-amber-200 rounded-lg dark:bg-amber-900/20 dark:border-amber-800"),
			Div(
				Class("flex items-center gap-2 text-amber-700 dark:text-amber-400"),
				lucide.CircleAlert(Class("size-4")),
				Span(Class("text-sm font-medium"), g.Text("OneOf configuration incomplete")),
			),
			P(
				Class("mt-1 text-xs text-amber-600 dark:text-amber-500"),
				g.Text("This field requires a discriminatorField and at least one schema."),
			),
		)
	}

	// Convert value to map
	valueMap := make(map[string]any)

	if value != nil {
		if m, ok := value.(map[string]any); ok {
			valueMap = m
		}
	}

	// Serialize current value
	valueJSON, _ := json.Marshal(valueMap)
	if len(valueMap) == 0 {
		valueJSON = []byte("{}")
	}

	// Build schema options for Alpine.js
	schemasJSON, _ := json.Marshal(schemas)

	isCollapsible := field.Options.Collapsible
	defaultExpanded := field.Options.DefaultExpanded

	return Div(
		Class("border border-slate-200 rounded-lg dark:border-gray-700"),
		g.Attr("x-data", fmt.Sprintf(`{
			expanded: %v,
			discriminatorField: '%s',
			schemas: %s,
			data: %s,
			clearOnChange: %v,
			_discriminatorValue: '',
			
			toggleExpanded() { this.expanded = !this.expanded; },
			
			get discriminatorValue() {
				// Try to get from root formData first, fallback to internal state
				if ($root && $root.formData && $root.formData[this.discriminatorField] !== undefined) {
					return $root.formData[this.discriminatorField];
				}
				return this._discriminatorValue || '';
			},
			
			getActiveSchema() {
				return this.schemas[this.discriminatorValue] || null;
			},
			
			updateDiscriminator() {
				const discInput = document.querySelector('[name="data[' + this.discriminatorField + ']"]');
				if (discInput) {
					this._discriminatorValue = discInput.value || '';
				}
			},
			
			init() {
				// Initial value
				this.updateDiscriminator();
				
				// Watch for changes in the discriminator field
				const discInput = document.querySelector('[name="data[' + this.discriminatorField + ']"]');
				if (discInput) {
					discInput.addEventListener('change', () => {
						const oldValue = this._discriminatorValue;
						this.updateDiscriminator();
						if (this.clearOnChange && this._discriminatorValue !== oldValue) {
							this.data = {};
						}
					});
				}
				
				// Also try Alpine watch if $root exists
				if ($root && $root.formData) {
					this.$watch('$root.formData["%s"]', (newValue, oldValue) => {
						this._discriminatorValue = newValue || '';
						if (this.clearOnChange && newValue !== oldValue && oldValue !== undefined) {
							this.data = {};
						}
					});
				}
			}
		}`, defaultExpanded || !isCollapsible, discriminatorField, string(schemasJSON), string(valueJSON), clearOnChange, discriminatorField)),

		// Header (collapsible)
		g.If(isCollapsible, func() g.Node {
			return Div(
				Class("flex items-center justify-between px-4 py-3 bg-slate-50 dark:bg-gray-800/50 cursor-pointer rounded-t-lg"),
				g.Attr("@click", "toggleExpanded()"),
				Div(
					Class("flex items-center gap-2"),
					lucide.GitMerge(Class("size-4 text-indigo-500")),
					Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Conditional Fields")),
					Span(
						Class("text-xs text-slate-500 dark:text-gray-400 ml-2"),
						g.Text("Based on: "),
						Code(Class("px-1 py-0.5 bg-slate-200 dark:bg-gray-700 rounded"), g.Text(discriminatorField)),
					),
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
			Class("p-4"),

			// Active schema label
			Div(
				g.Attr("x-show", "getActiveSchema()"),
				Class("mb-4 pb-3 border-b border-slate-200 dark:border-gray-700"),
				Div(
					Class("flex items-center gap-2"),
					lucide.Layers(Class("size-4 text-violet-500")),
					Span(
						Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Active Schema: "),
						Span(
							Class("text-violet-600 dark:text-violet-400"),
							g.Attr("x-text", "getActiveSchema()?.label || discriminatorValue"),
						),
					),
				),
			),

			// Render schema fields dynamically based on discriminator value
			g.Group(func() []g.Node {
				nodes := make([]g.Node, 0, len(schemas))
				for schemaKey, schemaOpt := range schemas {
					// Render each schema's fields, shown only when active
					if len(schemaOpt.NestedFields) > 0 {
						nodes = append(nodes, Div(
							g.Attr("x-show", fmt.Sprintf("discriminatorValue === '%s'", schemaKey)),
							Class("space-y-4"),
							g.Group(func() []g.Node {
								fields := make([]g.Node, len(schemaOpt.NestedFields))
								for i, nf := range schemaOpt.NestedFields {
									fields[i] = renderOneOfNestedFieldInput(nf, nf.Name)
								}

								return fields
							}()),
						))
					} else if schemaOpt.ComponentRef != "" {
						// Component reference not resolved - show message
						nodes = append(nodes, Div(
							g.Attr("x-show", fmt.Sprintf("discriminatorValue === '%s'", schemaKey)),
							Class("p-3 bg-amber-50 border border-amber-200 rounded-lg dark:bg-amber-900/20 dark:border-amber-800"),
							Div(
								Class("flex items-center gap-2 text-amber-700 dark:text-amber-400"),
								lucide.CircleAlert(Class("size-4")),
								Span(Class("text-sm font-medium"), g.Text("Component schema not resolved")),
							),
							P(
								Class("mt-1 text-xs text-amber-600 dark:text-amber-500"),
								g.Text(fmt.Sprintf("Component '%s' needs to be resolved server-side.", schemaOpt.ComponentRef)),
							),
						))
					}
				}

				return nodes
			}()),

			// No schema selected message
			Div(
				g.Attr("x-show", "!getActiveSchema()"),
				Class("py-6 text-center"),
				Div(
					Class("flex flex-col items-center gap-2 text-slate-500 dark:text-gray-400"),
					lucide.CircleHelp(Class("size-8 opacity-50")),
					P(Class("text-sm"), g.Text("Select a value for \""+discriminatorField+"\" to see the relevant fields.")),
				),
			),

			// Hidden input to store the full JSON
			Input(
				Type("hidden"),
				Name(fieldName),
				g.Attr(":value", "JSON.stringify(data)"),
			),
		),
	)
}

// renderArrayItemEditor renders the editor for a single array item.
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

// renderArrayItemField renders a single field within an array item.
func renderArrayItemField(field core.NestedFieldDefDTO, fieldName string, fieldIndex int) g.Node {
	// Note: This function creates a template that will be used with x-for
	// The actual field name and value binding happens through Alpine.js

	// Use Title if available, fallback to Name
	label := field.Name
	if field.Title != "" {
		label = field.Title
	}

	return Div(
		Label(
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
			g.If(field.Required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),

		// Render input based on type
		g.Group(func() []g.Node {
			xModel := "item." + field.Name

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

// renderOneOfNestedFieldInput renders a nested field input within a oneOf field using Alpine.js binding.
func renderOneOfNestedFieldInput(field core.NestedFieldDefDTO, fieldName string) g.Node {
	xModel := fmt.Sprintf("data['%s']", fieldName)

	// Use Title if available, fallback to Name
	label := field.Name
	if field.Title != "" {
		label = field.Title
	}

	return Div(
		Class("space-y-1"),
		Label(
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
			g.Text(label),
			g.If(field.Required, func() g.Node {
				return Span(Class("text-red-500 ml-1"), g.Text("*"))
			}()),
		),

		// Render input based on type
		g.Group(func() []g.Node {
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
			case "textarea", "richtext", "markdown":
				return []g.Node{
					Textarea(
						Rows("3"),
						g.Attr("x-model", xModel),
						// Don't add required attribute - validation happens server-side for oneOf fields
						Placeholder(field.Description),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "number", "integer", "float":
				return []g.Node{
					Input(
						Type("number"),
						g.Attr("x-model", xModel),
						// Don't add required attribute - validation happens server-side for oneOf fields
						Placeholder(field.Description),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			case "select":
				opts := []g.Node{Option(Value(""), g.Text("Select..."))}

				if field.Options != nil && len(field.Options.Choices) > 0 {
					for _, choice := range field.Options.Choices {
						opts = append(opts, Option(Value(choice.Value), g.Text(choice.Label)))
					}
				}

				return []g.Node{
					Select(
						g.Attr("x-model", xModel),
						// Don't add required attribute - validation happens server-side for oneOf fields
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Group(opts),
					),
				}
			case "json":
				// JSON editor with Alpine.js validation and formatting
				jsonEditorID := "json-editor-" + fieldName

				return []g.Node{
					Div(
						g.Attr("x-data", fmt.Sprintf(`{
							value: %s || {},
							error: '',
							format() {
								try {
									const parsed = typeof this.value === 'string' ? JSON.parse(this.value) : this.value;
									this.value = JSON.stringify(parsed, null, 2);
									this.error = '';
									%s = parsed;
								} catch (e) {
									this.error = 'Invalid JSON: ' + e.message;
								}
							},
							init() {
								this.value = typeof %s === 'string' ? %s : JSON.stringify(%s || {}, null, 2);
							}
						}`, xModel, xModel, xModel, xModel, xModel)),
						Div(Class("space-y-2"),
							Textarea(
								ID(jsonEditorID),
								Rows("6"),
								g.Attr("x-model", "value"),
								g.Attr("@blur", "format()"),
								Placeholder(`{"key": "value"}`),
								Class("block w-full px-3 py-2 text-sm font-mono border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
							Div(Class("flex gap-2"),
								Button(
									Type("button"),
									g.Attr("@click", "format()"),
									Class("px-3 py-1 text-xs font-medium text-violet-700 bg-violet-50 rounded-md hover:bg-violet-100 dark:bg-violet-900/20 dark:text-violet-400 dark:hover:bg-violet-900/30"),
									g.Text("Format JSON"),
								),
							),
							Div(
								g.Attr("x-show", "error"),
								g.Attr("x-text", "error"),
								Class("text-xs text-red-600 dark:text-red-400"),
							),
						),
					),
				}
			default: // text, email, url, etc.
				inputType := "text"
				switch field.Type {
				case "email":
					inputType = "email"
				case "url":
					inputType = "url"
				case "date":
					inputType = "date"
				}

				return []g.Node{
					Input(
						Type(inputType),
						g.Attr("x-model", xModel),
						// Don't add required attribute - validation happens server-side for oneOf fields
						Placeholder(field.Description),
						Class("block w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
					),
				}
			}
		}()),

		// Description
		g.If(field.Description != "", func() g.Node {
			return P(
				Class("text-xs text-slate-500 dark:text-gray-500 mt-1"),
				g.Text(field.Description),
			)
		}()),
	)
}

// renderNestedFieldInput renders an input for a nested field (used in object fields).
func renderNestedFieldInput(field core.NestedFieldDefDTO, fieldName string, value any, depth int) g.Node {
	valueStr := ""
	if value != nil {
		valueStr = fmt.Sprintf("%v", value)
	}

	// Use Title if available, fallback to Name
	label := field.Name
	if field.Title != "" {
		label = field.Title
	}

	return Div(
		Label(
			For(fieldName),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
			g.Text(label),
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
			case "json":
				// JSON editor for nested fields
				jsonValue := "{}"

				if value != nil {
					if jsonBytes, err := json.Marshal(value); err == nil {
						jsonValue = string(jsonBytes)
					}
				}

				jsonEditorID := "json-editor-" + fieldName

				return []g.Node{
					Div(
						g.Attr("x-data", fmt.Sprintf(`{
							value: %s,
							error: '',
							format() {
								try {
									const parsed = JSON.parse(this.value);
									this.value = JSON.stringify(parsed, null, 2);
									this.error = '';
								} catch (e) {
									this.error = 'Invalid JSON: ' + e.message;
								}
							},
							init() {
								try {
									const parsed = JSON.parse(this.value);
									this.value = JSON.stringify(parsed, null, 2);
								} catch (e) {}
							}
						}`, strconv.Quote(jsonValue))),
						Div(Class("space-y-2"),
							Textarea(
								ID(jsonEditorID),
								Rows("6"),
								g.Attr("x-model", "value"),
								g.Attr("@blur", "format()"),
								Placeholder(`{"key": "value"}`),
								Class("block w-full px-3 py-2 text-sm font-mono border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
							// Hidden input for form submission
							Input(
								Type("hidden"),
								Name(fieldName),
								g.Attr("x-bind:value", "value"),
							),
							Div(Class("flex gap-2"),
								Button(
									Type("button"),
									g.Attr("@click", "format()"),
									Class("px-3 py-1 text-xs font-medium text-violet-700 bg-violet-50 rounded-md hover:bg-violet-100 dark:bg-violet-900/20 dark:text-violet-400 dark:hover:bg-violet-900/30"),
									g.Text("Format JSON"),
								),
							),
							Div(
								g.Attr("x-show", "error"),
								g.Attr("x-text", "error"),
								Class("text-xs text-red-600 dark:text-red-400"),
							),
						),
					),
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

// renderNestedObjectFieldInput renders inputs for nested object fields within an object.
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
				subFieldName := baseName + "[" + f.Name + "]"
				subValue := valueMap[f.Name]
				nodes[i] = renderNestedFieldInput(f, subFieldName, subValue, depth)
			}

			return nodes
		}()),
	)
}

// =============================================================================
// JSON Editor Field (Monaco Editor)
// =============================================================================

// renderJsonEditorField renders a JSON editor for JSON fields.
func renderJsonEditorField(field *core.ContentFieldDTO, fieldName string, value any) g.Node {
	// Convert value to JSON string
	jsonValue := "{}"

	if value != nil {
		switch v := value.(type) {
		case string:
			jsonValue = v
		case map[string]any, []any:
			if bytes, err := json.MarshalIndent(v, "", "  "); err == nil {
				jsonValue = string(bytes)
			}
		default:
			if bytes, err := json.MarshalIndent(v, "", "  "); err == nil {
				jsonValue = string(bytes)
			}
		}
	}

	// Escape JSON for JavaScript string
	escapedJSON := escapeJSONForJS(jsonValue)

	return Div(
		Class("space-y-2"),
		// Monaco Editor using simpler textarea with JSON validation
		Div(
			Class("flex flex-col relative w-full border border-slate-300 dark:border-gray-700 rounded-lg overflow-hidden"),
			g.Attr("x-data", fmt.Sprintf(`{
				content: %s,
				isValid: true,
				errorMessage: '',
				
				validateJson() {
					try {
						JSON.parse(this.content);
						this.isValid = true;
						this.errorMessage = '';
					} catch (e) {
						this.isValid = false;
						this.errorMessage = e.message;
					}
				},
				
				formatJson() {
					if (!this.isValid) return;
					try {
						const parsed = JSON.parse(this.content);
						this.content = JSON.stringify(parsed, null, 2);
					} catch (e) {}
				},
				
				init() {
					this.validateJson();
					this.$watch('content', () => this.validateJson());
				}
			}`, escapedJSON)),

			// Toolbar
			Div(
				Class("flex items-center justify-between px-3 py-2 bg-slate-50 dark:bg-gray-800 border-b border-slate-200 dark:border-gray-700"),
				Div(
					Class("flex items-center gap-2"),
					// Validation status
					Div(
						g.Attr("x-show", "isValid"),
						Class("flex items-center gap-1 text-xs text-green-600 dark:text-green-400"),
						lucide.Check(Class("size-3")),
						Span(g.Text("Valid JSON")),
					),
					Div(
						g.Attr("x-show", "!isValid"),
						Class("flex items-center gap-1 text-xs text-red-600 dark:text-red-400"),
						lucide.CircleAlert(Class("size-3")),
						Span(g.Attr("x-text", "errorMessage.substring(0, 50) || 'Invalid JSON'")),
					),
				),
				// Format button
				Button(
					Type("button"),
					g.Attr("@click", "formatJson()"),
					g.Attr(":disabled", "!isValid"),
					Class("inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-gray-700 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-600"),
					lucide.Braces(Class("size-3")),
					g.Text("Format"),
				),
			),

			// Textarea editor
			Textarea(
				g.Attr("x-model", "content"),
				Name(fieldName),
				ID(fieldName),
				Rows("12"),
				Class("block w-full px-3 py-2 text-sm font-mono border-0 bg-white dark:bg-gray-900 dark:text-white focus:outline-none focus:ring-0"),
				Placeholder("{}"),
				g.If(field.Required, Required()),
			),
		),
	)
}

// escapeJSONForJS escapes a JSON string for safe embedding in JavaScript.
func escapeJSONForJS(jsonStr string) string {
	// Parse and re-marshal to ensure valid JSON
	var parsed any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return `{}`
	}

	// Marshal with indentation for readability
	bytes, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return `{}`
	}

	// Escape for JavaScript string context
	escaped := string(bytes)
	escaped = fmt.Sprintf("`%s`", escaped)

	return escaped
}

// =============================================================================
// Slug Field with Auto-Generation
// =============================================================================

// renderSlugField renders a slug field with optional auto-generation from another field.
func renderSlugField(field *core.ContentFieldDTO, fieldName string, value string) g.Node {
	sourceField := field.Options.SourceField
	hasSourceField := sourceField != ""

	// If no source field, render a simple text input
	if !hasSourceField {
		inputAttrs := []g.Node{
			Type("text"),
			ID(fieldName),
			Name(fieldName),
			Value(value),
			Placeholder("e.g., my-slug"),
			Class("block w-full px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"),
		}
		if field.Required {
			inputAttrs = append(inputAttrs, Required())
		}

		return Input(inputAttrs...)
	}

	// With source field - add Alpine.js auto-generation
	return Div(
		Class("space-y-2"),
		g.Attr("x-data", fmt.Sprintf(`{
			slug: '%s',
			manuallyEdited: %v,
			sourceFieldId: 'data[%s]',
			
			slugify(text) {
				return text
					.toLowerCase()
					.trim()
					.replace(/[^\w\s-]/g, '')
					.replace(/[\s_-]+/g, '-')
					.replace(/^-+|-+$/g, '');
			},
			
			generateSlug() {
				const sourceInput = document.querySelector('[name="' + this.sourceFieldId + '"]');
				if (sourceInput) {
					this.slug = this.slugify(sourceInput.value);
				}
			},
			
			watchSource() {
				const sourceInput = document.querySelector('[name="' + this.sourceFieldId + '"]');
				if (sourceInput) {
					sourceInput.addEventListener('input', (e) => {
						if (!this.manuallyEdited) {
							this.slug = this.slugify(e.target.value);
						}
					});
				}
			},
			
			init() {
				this.$nextTick(() => this.watchSource());
			}
		}`, value, value != "", sourceField)),

		// Input with generate button
		Div(
			Class("flex gap-2"),
			Input(
				Type("text"),
				ID(fieldName),
				Name(fieldName),
				g.Attr("x-model", "slug"),
				g.Attr("@input", "manuallyEdited = true"),
				Placeholder("e.g., my-slug"),
				g.If(field.Required, Required()),
				Class("flex-1 px-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"),
			),
			Button(
				Type("button"),
				g.Attr("@click", "generateSlug(); manuallyEdited = false"),
				Class("inline-flex items-center gap-1 px-3 py-2 text-sm font-medium text-slate-600 bg-slate-100 border border-slate-300 rounded-lg hover:bg-slate-200 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-600"),
				lucide.RefreshCw(Class("size-4")),
				g.Text("Generate"),
			),
		),

		// Hint about source field
		P(
			Class("text-xs text-slate-500 dark:text-gray-400"),
			g.Text("Auto-generates from: "),
			Code(
				Class("px-1 py-0.5 bg-slate-100 dark:bg-gray-700 rounded text-xs"),
				g.Text(sourceField),
			),
		),
	)
}
