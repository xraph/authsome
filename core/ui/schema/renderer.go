package schema

import (
	"fmt"
	"strings"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// Renderer renders schema elements to ForgeUI components
type Renderer struct {
	// Theme is the current theme (light/dark)
	theme string
	// DataPrefix is the Alpine.js data object prefix (e.g., "settings")
	dataPrefix string
	// ErrorPrefix is the Alpine.js error object prefix (e.g., "errors")
	errorPrefix string
	// OnSave is the function to call when save is triggered
	onSave string
}

// NewRenderer creates a new renderer with default settings
func NewRenderer() *Renderer {
	return &Renderer{
		theme:       "light",
		dataPrefix:  "data",
		errorPrefix: "errors",
		onSave:      "save()",
	}
}

// WithTheme sets the theme
func (r *Renderer) WithTheme(theme string) *Renderer {
	r.theme = theme
	return r
}

// WithDataPrefix sets the Alpine.js data prefix
func (r *Renderer) WithDataPrefix(prefix string) *Renderer {
	r.dataPrefix = prefix
	return r
}

// WithErrorPrefix sets the Alpine.js error prefix
func (r *Renderer) WithErrorPrefix(prefix string) *Renderer {
	r.errorPrefix = prefix
	return r
}

// WithOnSave sets the save function name
func (r *Renderer) WithOnSave(fn string) *Renderer {
	r.onSave = fn
	return r
}

// RenderSchema renders a complete schema as a form
func (r *Renderer) RenderSchema(schema *Schema, options ...RenderOption) g.Node {
	opts := r.applyOptions(options...)

	sections := make([]g.Node, 0, len(schema.Sections))
	for _, section := range schema.Sections {
		sections = append(sections, r.renderSectionWithOpts(section, opts))
	}

	return html.Div(
		html.Class("space-y-6"),
		g.Group(sections),
	)
}

// RenderSection renders a section with all its fields
func (r *Renderer) RenderSection(section *Section, options ...RenderOption) g.Node {
	opts := r.applyOptions(options...)
	return r.renderSectionWithOpts(section, opts)
}

// renderSectionWithOpts renders a section with pre-processed options
func (r *Renderer) renderSectionWithOpts(section *Section, opts renderOptions) g.Node {
	fields := make([]g.Node, 0, len(section.GetSortedFields()))
	for _, field := range section.GetSortedFields() {
		fields = append(fields, r.renderFieldWithOpts(field, opts))
	}

	// Build section header
	var headerContent []g.Node
	if section.Icon != "" {
		headerContent = append(headerContent, r.renderIcon(section.Icon))
	}
	headerContent = append(headerContent, g.Text(section.Title))

	header := card.Header(
		card.Title(section.Title),
		g.If(section.Description != "", card.Description(section.Description)),
	)

	content := card.Content(
		html.Div(
			html.Class("space-y-4"),
			g.Group(fields),
		),
	)

	// Add footer with save button if requested
	var footer g.Node
	if opts.showSaveButton {
		footer = card.Footer(
			button.Button(
				g.Text("Save Changes"),
				button.WithAttrs(g.Attr("@click", r.onSave)),
				button.WithAttrs(g.Attr(":disabled", "saving")),
			),
		)
	}

	if section.Collapsible {
		return r.renderCollapsibleCard(section, header, content, footer, opts)
	}

	if footer != nil {
		return card.Card(header, content, footer)
	}
	return card.Card(header, content)
}

// renderCollapsibleCard renders a collapsible card section
func (r *Renderer) renderCollapsibleCard(section *Section, header, content, footer g.Node, opts renderOptions) g.Node {
	collapseVar := fmt.Sprintf("collapsed_%s", section.ID)
	defaultState := "false"
	if section.DefaultCollapsed {
		defaultState = "true"
	}

	return html.Div(
		g.Attr("x-data", fmt.Sprintf("{%s: %s}", collapseVar, defaultState)),
		card.Card(
			html.Div(
				html.Class("cursor-pointer"),
				g.Attr("@click", fmt.Sprintf("%s = !%s", collapseVar, collapseVar)),
				header,
			),
			html.Div(
				g.Attr("x-show", fmt.Sprintf("!%s", collapseVar)),
				g.Attr("x-collapse", ""),
				content,
				g.If(footer != nil, footer),
			),
		),
	)
}

// RenderField renders a single field based on its type
func (r *Renderer) RenderField(field *Field, options ...RenderOption) g.Node {
	opts := r.applyOptions(options...)
	return r.renderFieldWithOpts(field, opts)
}

// renderFieldWithOpts renders a field with pre-processed options
func (r *Renderer) renderFieldWithOpts(field *Field, opts renderOptions) g.Node {

	if field.Hidden {
		return g.Group(nil)
	}

	// Build model binding
	modelPath := fmt.Sprintf("%s.%s", r.dataPrefix, field.ID)
	errorPath := fmt.Sprintf("%s.%s", r.errorPrefix, field.ID)

	// Determine width class
	widthClass := "w-full"
	switch field.Width {
	case "half":
		widthClass = "w-full md:w-1/2"
	case "third":
		widthClass = "w-full md:w-1/3"
	case "quarter":
		widthClass = "w-full md:w-1/4"
	}

	// Build visibility conditions
	var visibilityAttrs []g.Node
	for _, cond := range field.Conditions {
		if cond.Action == ActionShow {
			visibilityAttrs = append(visibilityAttrs, g.Attr("x-show", r.buildConditionExpression(cond)))
		} else if cond.Action == ActionHide {
			visibilityAttrs = append(visibilityAttrs, g.Attr("x-show", "!("+r.buildConditionExpression(cond)+")"))
		}
	}

	// Render the input based on type
	inputNode := r.renderInput(field, modelPath, opts)

	// Wrap with label and error display
	return html.Div(
		html.Class(widthClass),
		g.Group(visibilityAttrs),
		html.Div(
			html.Class("space-y-2"),
			// Label
			html.Label(
				html.Class("block text-sm font-medium"),
				html.For(field.ID),
				g.Text(field.Label),
				g.If(field.Required, html.Span(html.Class("text-red-500 ml-1"), g.Text("*"))),
			),
			// Input
			inputNode,
			// Description/Help text
			g.If(field.Description != "", html.P(
				html.Class("text-sm text-gray-500 dark:text-gray-400"),
				g.Text(field.Description),
			)),
			// Error message
			html.Div(
				g.Attr("x-show", errorPath),
				html.Class("text-sm text-red-500"),
				g.Attr("x-text", errorPath),
			),
		),
	)
}

// renderInput renders the appropriate input component for a field type
func (r *Renderer) renderInput(field *Field, modelPath string, opts renderOptions) g.Node {
	switch field.Type {
	case FieldTypeText:
		return r.renderTextInput(field, modelPath)
	case FieldTypeNumber:
		return r.renderNumberInput(field, modelPath)
	case FieldTypeBoolean:
		return r.renderBooleanInput(field, modelPath)
	case FieldTypeSelect:
		return r.renderSelectInput(field, modelPath)
	case FieldTypeMultiSelect:
		return r.renderMultiSelectInput(field, modelPath)
	case FieldTypePassword:
		return r.renderPasswordInput(field, modelPath)
	case FieldTypeEmail:
		return r.renderEmailInput(field, modelPath)
	case FieldTypeURL:
		return r.renderURLInput(field, modelPath)
	case FieldTypeTextArea:
		return r.renderTextAreaInput(field, modelPath)
	case FieldTypeJSON:
		return r.renderJSONInput(field, modelPath)
	case FieldTypeSlider:
		return r.renderSliderInput(field, modelPath)
	case FieldTypeTags:
		return r.renderTagsInput(field, modelPath)
	case FieldTypeColor:
		return r.renderColorInput(field, modelPath)
	case FieldTypeDate, FieldTypeDateTime:
		return r.renderDateInput(field, modelPath)
	default:
		return r.renderTextInput(field, modelPath)
	}
}

// renderTextInput renders a text input
func (r *Renderer) renderTextInput(field *Field, modelPath string) g.Node {
	attrs := []input.Option{
		input.WithID(field.ID),
		input.WithPlaceholder(field.Placeholder),
		input.WithAttrs(g.Attr("x-model", modelPath)),
	}

	if field.Disabled || field.ReadOnly {
		attrs = append(attrs, input.WithAttrs(g.Attr("disabled", "")))
	}

	// Add prefix/suffix if present
	if field.Prefix != "" || field.Suffix != "" {
		return r.renderInputWithAddons(field, modelPath, "text")
	}

	return input.Input(attrs...)
}

// renderInputWithAddons renders an input with prefix/suffix
func (r *Renderer) renderInputWithAddons(field *Field, modelPath string, inputType string) g.Node {
	return html.Div(
		html.Class("flex items-center"),
		g.If(field.Prefix != "", html.Span(
			html.Class("inline-flex items-center px-3 text-sm text-gray-500 bg-gray-100 dark:bg-gray-800 border border-r-0 border-gray-300 dark:border-gray-600 rounded-l-md"),
			g.Text(field.Prefix),
		)),
		input.Input(
			input.WithID(field.ID),
			input.WithPlaceholder(field.Placeholder),
			input.WithAttrs(g.Attr("x-model", modelPath)),
			input.WithAttrs(g.Attr("type", inputType)),
			input.WithAttrs(html.Class(r.getInputClass(field))),
		),
		g.If(field.Suffix != "", html.Span(
			html.Class("inline-flex items-center px-3 text-sm text-gray-500 bg-gray-100 dark:bg-gray-800 border border-l-0 border-gray-300 dark:border-gray-600 rounded-r-md"),
			g.Text(field.Suffix),
		)),
	)
}

// renderNumberInput renders a number input
func (r *Renderer) renderNumberInput(field *Field, modelPath string) g.Node {
	attrs := []g.Node{
		html.ID(field.ID),
		html.Name(field.ID),
		html.Type("number"),
		g.Attr("x-model.number", modelPath),
		html.Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
	}

	if field.Min != nil {
		attrs = append(attrs, g.Attr("min", fmt.Sprintf("%v", *field.Min)))
	}
	if field.Max != nil {
		attrs = append(attrs, g.Attr("max", fmt.Sprintf("%v", *field.Max)))
	}
	if field.Step != nil {
		attrs = append(attrs, g.Attr("step", fmt.Sprintf("%v", *field.Step)))
	}
	if field.Placeholder != "" {
		attrs = append(attrs, html.Placeholder(field.Placeholder))
	}
	if field.Disabled || field.ReadOnly {
		attrs = append(attrs, g.Attr("disabled", ""))
	}

	return html.Input(attrs...)
}

// renderBooleanInput renders a toggle/checkbox
func (r *Renderer) renderBooleanInput(field *Field, modelPath string) g.Node {
	return html.Label(
		html.Class("relative inline-flex items-center cursor-pointer"),
		html.Input(
			html.Type("checkbox"),
			html.Class("sr-only peer"),
			g.Attr("x-model", modelPath),
			g.If(field.Disabled, g.Attr("disabled", "")),
		),
		html.Div(
			html.Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"),
		),
		g.If(field.HelpText != "", html.Span(
			html.Class("ml-3 text-sm font-medium text-gray-700 dark:text-gray-300"),
			g.Text(field.HelpText),
		)),
	)
}

// renderSelectInput renders a select dropdown
func (r *Renderer) renderSelectInput(field *Field, modelPath string) g.Node {
	options := make([]g.Node, 0, len(field.Options)+1)

	// Add placeholder option
	if field.Placeholder != "" {
		options = append(options, html.Option(
			html.Value(""),
			html.Disabled(),
			g.Text(field.Placeholder),
		))
	}

	// Group options if any have groups
	hasGroups := false
	groups := make(map[string][]SelectOption)
	var ungrouped []SelectOption

	for _, opt := range field.Options {
		if opt.Group != "" {
			hasGroups = true
			groups[opt.Group] = append(groups[opt.Group], opt)
		} else {
			ungrouped = append(ungrouped, opt)
		}
	}

	if hasGroups {
		// Render ungrouped first
		for _, opt := range ungrouped {
			options = append(options, r.renderOption(opt))
		}
		// Then render groups
		for group, opts := range groups {
			groupOptions := make([]g.Node, len(opts))
			for i, opt := range opts {
				groupOptions[i] = r.renderOption(opt)
			}
			options = append(options, html.OptGroup(
				g.Attr("label", group),
				g.Group(groupOptions),
			))
		}
	} else {
		for _, opt := range field.Options {
			options = append(options, r.renderOption(opt))
		}
	}

	return html.Select(
		html.ID(field.ID),
		html.Name(field.ID),
		html.Class("flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"),
		g.Attr("x-model", modelPath),
		g.If(field.Disabled, g.Attr("disabled", "")),
		g.Group(options),
	)
}

// renderOption renders a select option
func (r *Renderer) renderOption(opt SelectOption) g.Node {
	return html.Option(
		html.Value(fmt.Sprintf("%v", opt.Value)),
		g.If(opt.Disabled, html.Disabled()),
		g.Text(opt.Label),
	)
}

// renderMultiSelectInput renders a multi-select component
func (r *Renderer) renderMultiSelectInput(field *Field, modelPath string) g.Node {
	// Simple multi-select with checkboxes
	checkboxes := make([]g.Node, len(field.Options))
	for i, opt := range field.Options {
		checkboxes[i] = html.Label(
			html.Class("flex items-center space-x-2 cursor-pointer"),
			html.Input(
				html.Type("checkbox"),
				html.Value(fmt.Sprintf("%v", opt.Value)),
				html.Class("rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
				g.Attr("x-model", modelPath),
				g.If(opt.Disabled, g.Attr("disabled", "")),
			),
			html.Span(html.Class("text-sm"), g.Text(opt.Label)),
		)
	}

	return html.Div(
		html.Class("space-y-2"),
		g.Group(checkboxes),
	)
}

// renderPasswordInput renders a password input with toggle visibility
func (r *Renderer) renderPasswordInput(field *Field, modelPath string) g.Node {
	showVar := fmt.Sprintf("show_%s", field.ID)

	return html.Div(
		g.Attr("x-data", fmt.Sprintf("{%s: false}", showVar)),
		html.Class("relative"),
		html.Input(
			html.ID(field.ID),
			html.Name(field.ID),
			g.Attr(":type", fmt.Sprintf("%s ? 'text' : 'password'", showVar)),
			html.Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pr-10 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
			g.Attr("x-model", modelPath),
			g.If(field.Placeholder != "", html.Placeholder(field.Placeholder)),
			g.If(field.Disabled, g.Attr("disabled", "")),
		),
		html.Button(
			html.Type("button"),
			html.Class("absolute inset-y-0 right-0 flex items-center pr-3"),
			g.Attr("@click", fmt.Sprintf("%s = !%s", showVar, showVar)),
			html.Span(
				g.Attr("x-show", fmt.Sprintf("!%s", showVar)),
				icons.Eye(icons.WithSize(16)),
			),
			html.Span(
				g.Attr("x-show", showVar),
				icons.EyeOff(icons.WithSize(16)),
			),
		),
	)
}

// renderEmailInput renders an email input
func (r *Renderer) renderEmailInput(field *Field, modelPath string) g.Node {
	return input.Input(
		input.WithID(field.ID),
		input.WithType("email"),
		input.WithPlaceholder(field.Placeholder),
		input.WithAttrs(g.Attr("x-model", modelPath)),
		input.WithAttrs(g.If(field.Disabled, g.Attr("disabled", ""))),
	)
}

// renderURLInput renders a URL input
func (r *Renderer) renderURLInput(field *Field, modelPath string) g.Node {
	return input.Input(
		input.WithID(field.ID),
		input.WithType("url"),
		input.WithPlaceholder(field.Placeholder),
		input.WithAttrs(g.Attr("x-model", modelPath)),
		input.WithAttrs(g.If(field.Disabled, g.Attr("disabled", ""))),
	)
}

// renderTextAreaInput renders a textarea
func (r *Renderer) renderTextAreaInput(field *Field, modelPath string) g.Node {
	rows := 4
	if field.Metadata != nil {
		if r, ok := field.Metadata["rows"].(int); ok {
			rows = r
		}
	}

	return html.Textarea(
		html.ID(field.ID),
		html.Name(field.ID),
		html.Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
		g.Attr("x-model", modelPath),
		g.Attr("rows", fmt.Sprintf("%d", rows)),
		g.If(field.Placeholder != "", html.Placeholder(field.Placeholder)),
		g.If(field.Disabled, g.Attr("disabled", "")),
		g.If(field.MaxLength != nil, g.Attr("maxlength", fmt.Sprintf("%d", *field.MaxLength))),
	)
}

// renderJSONInput renders a JSON editor textarea
func (r *Renderer) renderJSONInput(field *Field, modelPath string) g.Node {
	return html.Div(
		html.Class("relative"),
		html.Textarea(
			html.ID(field.ID),
			html.Name(field.ID),
			html.Class("flex min-h-[200px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
			g.Attr("x-model", modelPath),
			g.Attr("rows", "10"),
			g.If(field.Placeholder != "", html.Placeholder(field.Placeholder)),
			g.If(field.Disabled, g.Attr("disabled", "")),
		),
		// JSON validation indicator
		html.Div(
			html.Class("absolute top-2 right-2"),
			html.Span(
				g.Attr("x-show", fmt.Sprintf("isValidJSON(%s)", modelPath)),
				html.Class("text-green-500"),
				icons.Check(icons.WithSize(16)),
			),
			html.Span(
				g.Attr("x-show", fmt.Sprintf("!isValidJSON(%s)", modelPath)),
				html.Class("text-red-500"),
				icons.X(icons.WithSize(16)),
			),
		),
	)
}

// renderSliderInput renders a range slider
func (r *Renderer) renderSliderInput(field *Field, modelPath string) g.Node {
	min := 0.0
	max := 100.0
	step := 1.0

	if field.Min != nil {
		min = *field.Min
	}
	if field.Max != nil {
		max = *field.Max
	}
	if field.Step != nil {
		step = *field.Step
	}

	return html.Div(
		html.Class("flex items-center space-x-4"),
		html.Input(
			html.Type("range"),
			html.ID(field.ID),
			html.Name(field.ID),
			html.Class("w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"),
			g.Attr("x-model.number", modelPath),
			g.Attr("min", fmt.Sprintf("%v", min)),
			g.Attr("max", fmt.Sprintf("%v", max)),
			g.Attr("step", fmt.Sprintf("%v", step)),
			g.If(field.Disabled, g.Attr("disabled", "")),
		),
		html.Span(
			html.Class("text-sm font-medium text-gray-700 dark:text-gray-300 min-w-[3rem] text-right"),
			g.Attr("x-text", modelPath),
		),
	)
}

// renderTagsInput renders a tags input
func (r *Renderer) renderTagsInput(field *Field, modelPath string) g.Node {
	inputVar := fmt.Sprintf("tagInput_%s", field.ID)

	return html.Div(
		g.Attr("x-data", fmt.Sprintf("{%s: ''}", inputVar)),
		html.Class("space-y-2"),
		// Tags display
		html.Div(
			html.Class("flex flex-wrap gap-2"),
			g.El("template", g.Attr("x-for", fmt.Sprintf("(tag, index) in %s", modelPath)),
				html.Span(
					html.Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"),
					html.Span(g.Attr("x-text", "tag")),
					html.Button(
						html.Type("button"),
						html.Class("ml-1 inline-flex items-center"),
						g.Attr("@click", fmt.Sprintf("%s.splice(index, 1)", modelPath)),
						icons.X(icons.WithSize(12)),
					),
				),
			),
		),
		// Input for new tags
		html.Input(
			html.Type("text"),
			html.Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
			g.Attr("x-model", inputVar),
			html.Placeholder("Type and press Enter to add"),
			g.Attr("@keydown.enter.prevent", fmt.Sprintf("if (%s.trim()) { %s.push(%s.trim()); %s = '' }", inputVar, modelPath, inputVar, inputVar)),
		),
	)
}

// renderColorInput renders a color picker
func (r *Renderer) renderColorInput(field *Field, modelPath string) g.Node {
	return html.Div(
		html.Class("flex items-center space-x-2"),
		html.Input(
			html.Type("color"),
			html.ID(field.ID),
			html.Name(field.ID),
			html.Class("w-10 h-10 p-0 border-0 rounded cursor-pointer"),
			g.Attr("x-model", modelPath),
			g.If(field.Disabled, g.Attr("disabled", "")),
		),
		input.Input(
			input.WithPlaceholder("#000000"),
			input.WithAttrs(g.Attr("x-model", modelPath)),
			input.WithAttrs(html.Class("w-28")),
		),
	)
}

// renderDateInput renders a date or datetime input
func (r *Renderer) renderDateInput(field *Field, modelPath string) g.Node {
	inputType := "date"
	if field.Type == FieldTypeDateTime {
		inputType = "datetime-local"
	}

	return html.Input(
		html.Type(inputType),
		html.ID(field.ID),
		html.Name(field.ID),
		html.Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
		g.Attr("x-model", modelPath),
		g.If(field.Disabled, g.Attr("disabled", "")),
	)
}

// renderIcon renders an icon by name
func (r *Renderer) renderIcon(iconName string) g.Node {
	// Map common icon names to lucide icons
	switch strings.ToLower(iconName) {
	case "settings", "cog":
		return icons.Settings(icons.WithSize(20))
	case "shield", "security":
		return icons.Shield(icons.WithSize(20))
	case "lock":
		return icons.Lock(icons.WithSize(20))
	case "bell", "notification":
		return icons.Bell(icons.WithSize(20))
	case "user":
		return icons.User(icons.WithSize(20))
	case "mail", "email":
		return icons.Mail(icons.WithSize(20))
	case "globe", "web":
		return icons.Globe(icons.WithSize(20))
	case "key":
		return icons.Key(icons.WithSize(20))
	case "database":
		return icons.Database(icons.WithSize(20))
	case "code":
		return icons.Code(icons.WithSize(20))
	default:
		return icons.Settings(icons.WithSize(20))
	}
}

// buildConditionExpression builds an Alpine.js expression for a condition
func (r *Renderer) buildConditionExpression(cond Condition) string {
	fieldPath := fmt.Sprintf("%s.%s", r.dataPrefix, cond.Field)

	switch cond.Operator {
	case ConditionEquals:
		return fmt.Sprintf("%s === %s", fieldPath, r.formatValue(cond.Value))
	case ConditionNotEquals:
		return fmt.Sprintf("%s !== %s", fieldPath, r.formatValue(cond.Value))
	case ConditionEmpty:
		return fmt.Sprintf("!%s || %s === ''", fieldPath, fieldPath)
	case ConditionNotEmpty:
		return fmt.Sprintf("%s && %s !== ''", fieldPath, fieldPath)
	case ConditionContains:
		return fmt.Sprintf("(%s || '').includes(%s)", fieldPath, r.formatValue(cond.Value))
	case ConditionGreaterThan:
		return fmt.Sprintf("%s > %s", fieldPath, r.formatValue(cond.Value))
	case ConditionLessThan:
		return fmt.Sprintf("%s < %s", fieldPath, r.formatValue(cond.Value))
	default:
		return "true"
	}
}

// formatValue formats a value for use in Alpine.js expressions
func (r *Renderer) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getInputClass returns the appropriate class for an input with addons
func (r *Renderer) getInputClass(field *Field) string {
	classes := "flex h-10 w-full px-3 py-2 text-sm border border-input bg-background ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"

	if field.Prefix != "" && field.Suffix != "" {
		classes += " rounded-none"
	} else if field.Prefix != "" {
		classes += " rounded-r-md rounded-l-none"
	} else if field.Suffix != "" {
		classes += " rounded-l-md rounded-r-none"
	} else {
		classes += " rounded-md"
	}

	return classes
}

// RenderOption is a functional option for rendering
type RenderOption func(*renderOptions)

type renderOptions struct {
	showSaveButton bool
	readOnly       bool
}

func (r *Renderer) applyOptions(opts ...RenderOption) renderOptions {
	options := renderOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// WithSaveButton includes a save button in the rendered form
func WithSaveButton() RenderOption {
	return func(o *renderOptions) {
		o.showSaveButton = true
	}
}

// WithReadOnly makes all fields read-only
func WithReadOnly() RenderOption {
	return func(o *renderOptions) {
		o.readOnly = true
	}
}
