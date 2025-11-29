package builder

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BuilderUI renders the visual email builder interface
type BuilderUI struct {
	document   *Document
	previewURL string
	saveURL    string
}

// NewBuilderUI creates a new builder UI instance
func NewBuilderUI(doc *Document, previewURL, saveURL string) *BuilderUI {
	return &BuilderUI{
		document:   doc,
		previewURL: previewURL,
		saveURL:    saveURL,
	}
}

// Render renders the complete builder interface
func (b *BuilderUI) Render() g.Node {
	docJSON, _ := b.document.ToJSON()

	return Div(
		Class("email-builder-container"),
		g.Attr("x-data", fmt.Sprintf(`emailBuilder(%s)`, docJSON)),

		// Prism.js for syntax highlighting (CDN)
		Link(Rel("stylesheet"), Href("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism.min.css")),
		Script(Src("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js")),
		Script(Src("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-markup.min.js")),
		Script(Src("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-json.min.js")),

		// Builder styles
		StyleEl(g.Raw(builderCSS)),

		// Builder JavaScript
		Script(g.Raw(builderJS)),

		// Top Toolbar
		b.renderToolbar(),

		// Main builder layout
		Div(
			Class("builder-layout"),

			// Left sidebar - Navigation / Templates
			b.renderLeftSidebar(),

			// Center - Canvas
			b.renderCanvas(),

			// Right sidebar - Styles/Inspect
			b.renderRightSidebar(),
		),

		// Floating Block Picker
		b.renderFloatingBlockPicker(),
	)
}

// renderToolbar renders the top toolbar
func (b *BuilderUI) renderToolbar() g.Node {
	return Div(
		Class("builder-toolbar"),
		Div(
			Class("toolbar-left"),
			H1(Class("toolbar-title"), g.Text("EmailBuilder.js")),
		),
		Div(
			Class("toolbar-center"),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr(":class", "{'active': view === 'design'}"),
				g.Attr("@click", "view = 'design'"),
				lucide.Pencil(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr(":class", "{'active': view === 'preview'}"),
				g.Attr("@click", "view = 'preview'"),
				lucide.Eye(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr(":class", "{'active': view === 'code'}"),
				g.Attr("@click", "view = 'code'"),
				lucide.Code(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr(":class", "{'active': view === 'json'}"),
				g.Attr("@click", "view = 'json'"),
				lucide.Braces(Class("size-4")),
			),
		),
		Div(
			Class("toolbar-actions"),
			// Desktop/Mobile toggle
			Div(
				Class("view-toggle"),
				Button(
					Class("toolbar-icon-btn"),
					g.Attr(":class", "{'active': deviceView === 'desktop'}"),
					g.Attr("@click", "deviceView = 'desktop'"),
					lucide.Monitor(Class("size-4")),
				),
				Button(
					Class("toolbar-icon-btn"),
					g.Attr(":class", "{'active': deviceView === 'mobile'}"),
					g.Attr("@click", "deviceView = 'mobile'"),
					lucide.Smartphone(Class("size-4")),
				),
			),
			// Actions
			Button(
				Class("toolbar-icon-btn"),
				g.Attr("@click", "undo()"),
				g.Attr(":disabled", "!canUndo"),
				lucide.Undo(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr("@click", "redo()"),
				g.Attr(":disabled", "!canRedo"),
				lucide.Redo(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr("@click", "downloadHTML()"),
				lucide.Download(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				g.Attr("@click", "uploadJSON()"),
				lucide.Upload(Class("size-4")),
			),
			Button(
				Class("toolbar-icon-btn"),
				lucide.Share(Class("size-4")),
			),
		),
	)
}

// renderLeftSidebar renders the template navigation sidebar
func (b *BuilderUI) renderLeftSidebar() g.Node {
	return Div(
		Class("builder-sidebar builder-sidebar-left"),
		Div(
			Class("sidebar-content"),
			H3(Class("sidebar-section-title"), g.Text("EMPTY")),
			Div(
				Class("sidebar-nav-item"),
				g.Attr("@click", "loadTemplate('empty')"),
				g.Text("Empty"),
			),
			H3(Class("sidebar-section-title"), g.Text("SAMPLE TEMPLATES")),
			b.renderTemplateNavItem("welcome", "Welcome email"),
			b.renderTemplateNavItem("otp", "One-time passcode (OTP)"),
			b.renderTemplateNavItem("reset_password", "Reset password"),
			b.renderTemplateNavItem("ecommerce", "E-commerce receipt"),
			b.renderTemplateNavItem("subscription", "Subscription receipt"),
			b.renderTemplateNavItem("reservation", "Reservation reminder"),
			b.renderTemplateNavItem("metrics", "Post metrics"),
			b.renderTemplateNavItem("inquiry", "Respond to inquiry"),

			Div(
				Class("sidebar-footer"),
				A(Href("#"), Class("sidebar-link"), g.Text("Learn more")),
				A(Href("#"), Class("sidebar-link"), g.Text("View on GitHub")),
			),
		),
	)
}

func (b *BuilderUI) renderTemplateNavItem(id, label string) g.Node {
	return Div(
		Class("sidebar-nav-item"),
		g.Attr("@click", fmt.Sprintf("loadTemplate('%s')", id)),
		g.Text(label),
	)
}

// renderCanvas renders the main editing area
func (b *BuilderUI) renderCanvas() g.Node {
	return Div(
		Class("builder-canvas"),
		g.Attr(":class", "{'mobile-view': deviceView === 'mobile'}"),
		g.Attr(":style", "getBackdropStyle()"),

		// Canvas Area
		Div(
			Class("canvas-area"),

			// Design View
			Div(
				g.Attr("x-show", "view === 'design'"),
				Class("canvas-design-wrapper"),
				g.Attr(":class", "{'mobile-width': deviceView === 'mobile'}"),
				Div(
					Class("canvas-document"),
					g.Attr("@click.self", "selectBlock('root')"),
					g.Attr(":style", "getCanvasStyle()"),

					// Render blocks recursively
					Template(
						g.Attr("x-for", "(block, index) in blocks"),
						g.Attr(":key", "block.id"),
						b.renderBlockItem(),
					),

					// Plus button at the end of document
					Div(
						Class("add-block-placeholder"),
						g.Attr("@click.stop", "openBlockPicker(null)"),
						Div(Class("plus-icon"), lucide.Plus(Class("size-4 text-white"))),
					),
				),
			),

			// Preview View (rendered HTML)
			Div(
				g.Attr("x-show", "view === 'preview'"),
				Class("canvas-preview-wrapper"),
				g.Attr(":class", "{'mobile-width': deviceView === 'mobile'}"),
				Div(
					Class("canvas-preview"),
					g.Attr("x-html", "getRenderedHTML()"),
				),
			),

			// Code View (HTML output with syntax highlighting)
			Div(
				g.Attr("x-show", "view === 'code'"),
				Class("canvas-code-wrapper"),
				g.Attr("x-effect", "if(view === 'code') $nextTick(() => highlightCode())"),
				Pre(
					Class("code-editor-prism"),
					Code(
						Class("language-html"),
						g.Attr("x-html", "escapeAndHighlightHTML(getRenderedHTML())"),
					),
				),
			),

			// JSON View (with syntax highlighting)
			Div(
				g.Attr("x-show", "view === 'json'"),
				Class("canvas-code-wrapper"),
				g.Attr("x-effect", "if(view === 'json') $nextTick(() => highlightCode())"),
				Pre(
					Class("code-editor-prism"),
					Code(
						Class("language-json"),
						g.Attr("x-html", "escapeAndHighlightJSON(documentJSON)"),
					),
				),
			),
		),
	)
}

// renderBlockItem renders a single block in the canvas
func (b *BuilderUI) renderBlockItem() g.Node {
	return Div(
		Class("canvas-block"),
		g.Attr(":class", "{'selected': selectedBlock === block.id}"),
		g.Attr("@click.stop", "selectBlock(block.id)"),
		g.Attr("@mouseover.stop", "hoverBlock = block.id"),
		g.Attr("@mouseleave", "hoverBlock = null"),

		// Block Actions (Left side floating)
		Div(
			Class("block-actions-sidebar"),
			g.Attr("x-show", "selectedBlock === block.id"),
			Button(
				Class("block-action-btn"),
				g.Attr("@click.stop", "moveBlock(block.id, 'up')"),
				lucide.ArrowUp(Class("size-4")),
			),
			Button(
				Class("block-action-btn"),
				g.Attr("@click.stop", "moveBlock(block.id, 'down')"),
				lucide.ArrowDown(Class("size-4")),
			),
			Button(
				Class("block-action-btn delete-btn"),
				g.Attr("@click.stop", "deleteBlock(block.id)"),
				lucide.Trash2(Class("size-4")),
			),
		),

		// Block Content
		Div(
			Class("block-content"),
			g.Attr("x-html", "renderBlockPreview(block)"),
		),

		// Add button (inside/after block)
		Div(
			Class("block-add-trigger"),
			g.Attr("x-show", "selectedBlock === block.id"),
			g.Attr("@click.stop", "openBlockPicker(block.id)"),
			Div(Class("plus-icon-small"), lucide.Plus(Class("size-3 text-white"))),
		),
	)
}

// renderRightSidebar renders the Styles/Inspect panel
func (b *BuilderUI) renderRightSidebar() g.Node {
	return Div(
		Class("builder-sidebar builder-sidebar-right"),
		g.Attr("x-data", "{ activeTab: 'styles' }"),
		g.Attr("x-effect", "if (selectedBlock && selectedBlock !== 'root') activeTab = 'inspect'"),

		// Tabs
		Div(
			Class("sidebar-tabs"),
			Div(
				Class("sidebar-tab"),
				g.Attr(":class", "{'active': activeTab === 'styles'}"),
				g.Attr("@click", "activeTab = 'styles'; selectBlock('root')"),
				g.Text("Styles"),
			),
			Div(
				Class("sidebar-tab"),
				g.Attr(":class", "{'active': activeTab === 'inspect'}"),
				g.Attr("@click", "if(selectedBlock && selectedBlock !== 'root') activeTab = 'inspect'"),
				g.Text("Inspect"),
			),
		),

		// STYLES TAB (Global Settings)
		Div(
			Class("sidebar-content"),
			g.Attr("x-show", "activeTab === 'styles'"),
			b.renderGlobalSettings(),
		),

		// INSPECT TAB (Block Specific)
		Div(
			Class("sidebar-content"),
			g.Attr("x-show", "activeTab === 'inspect'"),

			// Selected Block Content
			Div(
				g.Attr("x-show", "selectedBlock && selectedBlock !== 'root'"),
				H4(
					Class("properties-header"),
					Span(g.Attr("x-text", "(selectedBlockType || 'BLOCK') + ' BLOCK'")),
				),
				b.renderBlockInspector(),
			),

			// Empty State
			Div(
				Class("empty-properties"),
				g.Attr("x-show", "!selectedBlock || selectedBlock === 'root'"),
				P(g.Text("Select a block to inspect properties")),
			),
		),
	)
}

// renderGlobalSettings renders global canvas settings
func (b *BuilderUI) renderGlobalSettings() g.Node {
	return Div(
		Class("properties-form"),

		Div(Class("properties-section-title"), g.Text("GLOBAL")),

		// Backdrop Color
		b.renderColorPickerWithPlus("Backdrop color", "document.blocks[document.root].data.backdropColor", "#F5F5F5"),

		// Canvas Color
		b.renderColorPickerWithPlus("Canvas color", "document.blocks[document.root].data.canvasColor", "#FFFFFF"),

		// Canvas Border Color
		b.renderColorPickerWithPlus("Canvas border color", "document.blocks[document.root].data.borderColor", ""),

		// Canvas Border Radius
		b.renderSliderInput("Canvas border radius", "document.blocks[document.root].data.borderRadius", 0, 20, "px"),

		// Font Family
		Div(
			Class("property-group"),
			Label(Class("property-label"), g.Text("Font family")),
			Select(
				Class("select-input"),
				g.Attr("x-model", "document.blocks[document.root].data.fontFamily"),
				Option(Value("MODERN_SANS"), g.Text("Modern sans")),
				Option(Value("BOOK_SANS"), g.Text("Book sans")),
				Option(Value("ORGANIC_SANS"), g.Text("Organic sans")),
				Option(Value("GEOMETRIC_SANS"), g.Text("Geometric sans")),
				Option(Value("HEAVY_SANS"), g.Text("Heavy sans")),
				Option(Value("ROUNDED_SANS"), g.Text("Rounded sans")),
				Option(Value("MODERN_SERIF"), g.Text("Modern serif")),
				Option(Value("BOOK_SERIF"), g.Text("Book serif")),
				Option(Value("MONOSPACE"), g.Text("Monospace")),
			),
		),

		// Text Color
		b.renderColorPickerWithPlus("Text color", "document.blocks[document.root].data.textColor", "#242424"),
	)
}

// renderBlockInspector renders the inspector for the selected block
func (b *BuilderUI) renderBlockInspector() g.Node {
	return Div(
		Class("properties-form"),

		// Content Field (Text/Heading/Button)
		Div(
			Class("property-group"),
			g.Attr("x-show", "['Text', 'Heading', 'Button'].includes(selectedBlockType)"),
			Label(Class("property-label"), g.Text("Content")),
			Textarea(
				Class("text-input"),
				g.Attr("x-model", "selectedBlockData.props.text"),
				Rows("3"),
			),
		),

		// Heading Level (Heading only)
		Div(
			Class("property-group"),
			g.Attr("x-show", "selectedBlockType === 'Heading'"),
			Label(Class("property-label"), g.Text("Level")),
			Div(
				Class("toggle-group"),
				b.renderToggleOption("H1", "selectedBlockData.props.level", "h1"),
				b.renderToggleOption("H2", "selectedBlockData.props.level", "h2"),
				b.renderToggleOption("H3", "selectedBlockData.props.level", "h3"),
			),
		),

		// URL (Button/Image)
		Div(
			Class("property-group"),
			g.Attr("x-show", "['Button', 'Image'].includes(selectedBlockType)"),
			Label(Class("property-label"), g.Text("Url")),
			Input(
				Type("text"),
				Class("text-input"),
				g.Attr("x-model", "selectedBlockData.props.url"),
				Placeholder("https://"),
			),
		),

		// Width (Button)
		Div(
			Class("property-group"),
			g.Attr("x-show", "selectedBlockType === 'Button'"),
			Label(Class("property-label"), g.Text("Width")),
			Div(
				Class("toggle-group"),
				b.renderToggleOption("Full", "selectedBlockData.props.fullWidth", "true"),
				b.renderToggleOption("Auto", "selectedBlockData.props.fullWidth", "false"),
			),
		),

		// Size (Button)
		Div(
			Class("property-group"),
			g.Attr("x-show", "selectedBlockType === 'Button'"),
			Label(Class("property-label"), g.Text("Size")),
			Div(
				Class("toggle-group"),
				b.renderToggleOption("Xs", "selectedBlockData.props.size", "xs"),
				b.renderToggleOption("Sm", "selectedBlockData.props.size", "sm"),
				b.renderToggleOption("Md", "selectedBlockData.props.size", "md"),
				b.renderToggleOption("Lg", "selectedBlockData.props.size", "lg"),
			),
		),

		// Style (Button)
		Div(
			Class("property-group"),
			g.Attr("x-show", "selectedBlockType === 'Button'"),
			Label(Class("property-label"), g.Text("Style")),
			Div(
				Class("toggle-group"),
				b.renderToggleOption("Rectangle", "selectedBlockData.props.buttonStyle", "rectangle"),
				b.renderToggleOption("Rounded", "selectedBlockData.props.buttonStyle", "rounded"),
				b.renderToggleOption("Pill", "selectedBlockData.props.buttonStyle", "pill"),
			),
		),

		// Text Color
		Div(
			g.Attr("x-show", "['Text', 'Heading', 'Button'].includes(selectedBlockType)"),
			b.renderColorPickerWithPlus("Text color", "selectedBlockData.style.color", ""),
		),

		// Button Color
		Div(
			g.Attr("x-show", "selectedBlockType === 'Button'"),
			b.renderColorPickerWithPlus("Button color", "selectedBlockData.props.buttonColor", "#999999"),
		),

		// Background Color
		b.renderColorPickerWithPlus("Background color", "selectedBlockData.style.backgroundColor", ""),

		// Font Settings
		Div(
			g.Attr("x-show", "['Text', 'Heading', 'Button'].includes(selectedBlockType)"),
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Font family")),
				Select(
					Class("select-input"),
					g.Attr("x-model", "selectedBlockData.style.fontFamily"),
					Option(Value(""), g.Text("Match email settings")),
					Option(Value("MODERN_SANS"), g.Text("Modern sans")),
					Option(Value("BOOK_SANS"), g.Text("Book sans")),
					Option(Value("MONOSPACE"), g.Text("Monospace")),
				),
			),
		),

		// Font Size
		Div(
			g.Attr("x-show", "['Text', 'Heading', 'Button'].includes(selectedBlockType)"),
			b.renderSliderInput("Font size", "selectedBlockData.style.fontSize", 10, 72, "px"),
		),

		// Font Weight
		Div(
			Class("property-group"),
			g.Attr("x-show", "['Text', 'Heading', 'Button'].includes(selectedBlockType)"),
			Label(Class("property-label"), g.Text("Font weight")),
			Div(
				Class("toggle-group"),
				b.renderToggleOption("Regular", "selectedBlockData.style.fontWeight", "normal"),
				b.renderToggleOption("Bold", "selectedBlockData.style.fontWeight", "bold"),
			),
		),

		// Alignment
		Div(
			Class("property-group"),
			Label(Class("property-label"), g.Text("Alignment")),
			Div(
				Class("toggle-group"),
				b.renderIconToggleOption(lucide.AlignLeft(Class("size-4")), "selectedBlockData.style.textAlign", "left"),
				b.renderIconToggleOption(lucide.AlignCenter(Class("size-4")), "selectedBlockData.style.textAlign", "center"),
				b.renderIconToggleOption(lucide.AlignRight(Class("size-4")), "selectedBlockData.style.textAlign", "right"),
			),
		),

		// Columns Block Properties
		Div(
			g.Attr("x-show", "selectedBlockType === 'Columns'"),

			// Number of columns
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Number of columns")),
				Div(
					Class("toggle-group"),
					Div(
						Class("toggle-option"),
						g.Attr(":class", "{'active': selectedBlockData?.props?.columnsCount == 2}"),
						g.Attr("@click", "updateColumnsCount(selectedBlock, 2)"),
						g.Text("2"),
					),
					Div(
						Class("toggle-option"),
						g.Attr(":class", "{'active': selectedBlockData?.props?.columnsCount == 3}"),
						g.Attr("@click", "updateColumnsCount(selectedBlock, 3)"),
						g.Text("3"),
					),
				),
			),

			// Column widths (simplified)
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Column widths")),
				Div(
					Class("columns-width-inputs"),
					Div(
						Class("column-width-input"),
						Label(g.Text("Column 1")),
						Input(Type("text"), Class("text-input small"), Placeholder("auto"), Value("auto")),
						Span(g.Text("px")),
					),
					Div(
						Class("column-width-input"),
						Label(g.Text("Column 2")),
						Input(Type("text"), Class("text-input small"), Placeholder("auto"), Value("auto")),
						Span(g.Text("px")),
					),
					Div(
						Class("column-width-input"),
						g.Attr("x-show", "selectedBlockData.props.columnsCount == 3"),
						Label(g.Text("Column 3")),
						Input(Type("text"), Class("text-input small"), Placeholder("auto"), Value("auto")),
						Span(g.Text("px")),
					),
				),
			),

			// Columns gap
			b.renderSliderInput("Columns gap", "selectedBlockData.props.columnsGap", 0, 48, "px"),

			// Vertical alignment
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Alignment")),
				Div(
					Class("toggle-group"),
					b.renderIconToggleOption(lucide.AlignStartVertical(Class("size-4")), "selectedBlockData.props.verticalAlign", "top"),
					b.renderIconToggleOption(lucide.AlignCenterVertical(Class("size-4")), "selectedBlockData.props.verticalAlign", "middle"),
					b.renderIconToggleOption(lucide.AlignEndVertical(Class("size-4")), "selectedBlockData.props.verticalAlign", "bottom"),
				),
			),
		),

		// Spacer height
		Div(
			g.Attr("x-show", "selectedBlockType === 'Spacer'"),
			b.renderSliderInput("Height", "selectedBlockData.props.height", 8, 96, "px"),
		),

		// Divider properties
		Div(
			g.Attr("x-show", "selectedBlockType === 'Divider'"),
			b.renderColorPickerWithPlus("Line color", "selectedBlockData.props.lineColor", "#e0e0e0"),
			b.renderSliderInput("Line height", "selectedBlockData.props.lineHeight", 1, 8, "px"),
		),

		// Image properties
		Div(
			g.Attr("x-show", "selectedBlockType === 'Image'"),
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Alt text")),
				Input(Type("text"), Class("text-input"), g.Attr("x-model", "selectedBlockData.props.alt"), Placeholder("Image description")),
			),
			Div(
				Class("property-group"),
				Label(Class("property-label"), g.Text("Content alignment")),
				Div(
					Class("toggle-group"),
					b.renderToggleOption("Left", "selectedBlockData.props.contentAlignment", "left"),
					b.renderToggleOption("Center", "selectedBlockData.props.contentAlignment", "center"),
					b.renderToggleOption("Right", "selectedBlockData.props.contentAlignment", "right"),
				),
			),
		),

		// Background Color (common)
		b.renderColorPickerWithPlus("Background color", "selectedBlockData.style.backgroundColor", ""),

		// Padding (common)
		Div(
			Class("property-group"),
			Label(Class("property-label"), g.Text("Padding")),
			b.renderSliderInput("Top", "selectedBlockData.style.padding.top", 0, 100, "px"),
			b.renderSliderInput("Bottom", "selectedBlockData.style.padding.bottom", 0, 100, "px"),
			b.renderSliderInput("Left", "selectedBlockData.style.padding.left", 0, 100, "px"),
			b.renderSliderInput("Right", "selectedBlockData.style.padding.right", 0, 100, "px"),
		),
	)
}

// Helper components

func (b *BuilderUI) renderColorPickerWithPlus(label, model, defaultColor string) g.Node {
	return Div(
		Class("property-group"),
		Label(Class("property-label"), g.Text(label)),
		Div(
			Class("color-picker-box"),
			g.Attr(":class", fmt.Sprintf("{'has-color': %s && %s !== ''}", model, model)),
			g.Attr(":style", fmt.Sprintf("(%s && %s !== '') ? 'background-color: ' + %s : ''", model, model, model)),

			// Plus icon when no color
			Div(
				Class("color-plus-icon"),
				g.Attr("x-show", fmt.Sprintf("!%s || %s === ''", model, model)),
				lucide.Plus(Class("size-4")),
			),

			// Hidden color input
			Input(
				Type("color"),
				Class("color-input-hidden"),
				g.Attr("x-model", model),
				g.If(defaultColor != "", Value(defaultColor)),
			),
		),
	)
}

func (b *BuilderUI) renderSliderInput(label, model string, min, max int, unit string) g.Node {
	return Div(
		Class("slider-input-group"),
		Div(Class("slider-icon"),
			g.If(label == "Top", lucide.AlignVerticalJustifyStart(Class("size-4"))),
			g.If(label == "Bottom", lucide.AlignVerticalJustifyEnd(Class("size-4"))),
			g.If(label == "Left", lucide.AlignHorizontalJustifyStart(Class("size-4"))),
			g.If(label == "Right", lucide.AlignHorizontalJustifyEnd(Class("size-4"))),
			g.If(label == "Font size", lucide.Type(Class("size-4"))),
			g.If(label == "Canvas border radius", lucide.Maximize(Class("size-4"))),
		),
		Input(
			Type("range"),
			Class("slider-range"),
			Min(fmt.Sprintf("%d", min)), Max(fmt.Sprintf("%d", max)),
			g.Attr("x-model", model),
		),
		Span(Class("slider-value"), g.Attr("x-text", fmt.Sprintf("(%s || 0) + '%s'", model, unit))),
	)
}

func (b *BuilderUI) renderToggleOption(label, model, value string) g.Node {
	return Div(
		Class("toggle-option"),
		g.Attr(":class", fmt.Sprintf("{'active': %s == '%s' || String(%s) === '%s'}", model, value, model, value)),
		g.Attr("@click", fmt.Sprintf("%s = '%s'", model, value)),
		g.Text(label),
	)
}

func (b *BuilderUI) renderIconToggleOption(icon g.Node, model, value string) g.Node {
	return Div(
		Class("toggle-option icon-option"),
		g.Attr(":class", fmt.Sprintf("{'active': %s == '%s'}", model, value)),
		g.Attr("@click", fmt.Sprintf("%s = '%s'", model, value)),
		icon,
	)
}

// blockItemData holds data for rendering block items
type blockItemData struct {
	Type BlockType
	Icon g.Node
	Name string
	Desc string
}

// renderFloatingBlockPicker renders the floating menu for adding blocks
func (b *BuilderUI) renderFloatingBlockPicker() g.Node {
	blocks := []blockItemData{
		{BlockTypeHeading, lucide.Heading(Class("size-5")), "Heading", ""},
		{BlockTypeText, lucide.Type(Class("size-5")), "Text", ""},
		{BlockTypeButton, lucide.MousePointerClick(Class("size-5")), "Button", ""},
		{BlockTypeImage, lucide.Image(Class("size-5")), "Image", ""},
		{BlockTypeAvatar, lucide.User(Class("size-5")), "Avatar", ""},
		{BlockTypeDivider, lucide.Minus(Class("size-5")), "Divider", ""},
		{BlockTypeSpacer, lucide.Space(Class("size-5")), "Spacer", ""},
		{BlockTypeHTML, lucide.Code(Class("size-5")), "Html", ""},
		{BlockTypeColumns, lucide.Columns2(Class("size-5")), "Columns", ""},
		{BlockTypeContainer, lucide.Box(Class("size-5")), "Container", ""},
	}

	return Div(
		Class("floating-block-picker"),
		g.Attr("x-show", "showPicker"),
		g.Attr("x-transition", ""),
		g.Attr("@click.outside", "showPicker = false"),
		g.Attr(":style", "'top: ' + pickerTop + 'px; left: ' + pickerLeft + 'px'"),

		Div(
			Class("picker-grid"),
			g.Group(g.Map(blocks, func(block blockItemData) g.Node {
				return Div(
					Class("picker-item"),
					g.Attr("@click", fmt.Sprintf("addBlock('%s')", block.Type)),
					Div(Class("picker-icon"), block.Icon),
					Span(Class("picker-label"), g.Text(block.Name)),
				)
			})),
		),
	)
}

const builderCSS = `
	/* Reset & Layout */
	.email-builder-container {
		height: 100vh;
		display: flex;
		flex-direction: column;
		background: #ffffff;
		font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
		color: #242424;
	}

	.builder-toolbar {
		height: 48px;
		border-bottom: 1px solid #e0e0e0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 16px;
		background: #fff;
		z-index: 10;
	}

	.toolbar-title {
		font-size: 14px;
		font-weight: 600;
		margin: 0;
	}

	.builder-layout {
		flex: 1;
		display: grid;
		grid-template-columns: 220px 1fr 300px;
		overflow: hidden;
	}

	/* Toolbar */
	.toolbar-center, .toolbar-actions, .view-toggle {
		display: flex;
		gap: 4px;
		align-items: center;
	}

	.view-toggle {
		background: #f5f5f5;
		padding: 2px;
		border-radius: 6px;
		margin-right: 8px;
	}

	.toolbar-icon-btn {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		background: transparent;
		border-radius: 6px;
		cursor: pointer;
		color: #666;
	}

	.toolbar-icon-btn:hover { background: #f0f0f0; color: #333; }
	.toolbar-icon-btn.active { background: #fff; box-shadow: 0 1px 2px rgba(0,0,0,0.08); color: #000; }
	.toolbar-icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }

	/* Sidebars */
	.builder-sidebar {
		background: #fff;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
	}

	.builder-sidebar-left { border-right: 1px solid #e0e0e0; }
	.builder-sidebar-right { border-left: 1px solid #e0e0e0; }

	.sidebar-content { padding: 16px; }

	.sidebar-section-title {
		font-size: 11px;
		font-weight: 600;
		color: #999;
		text-transform: uppercase;
		margin: 16px 0 8px;
		letter-spacing: 0.5px;
	}
	.sidebar-section-title:first-child { margin-top: 0; }

	.sidebar-nav-item {
		padding: 8px 0;
		cursor: pointer;
		font-size: 14px;
		color: #333;
	}
	.sidebar-nav-item:hover { color: #000; }

	.sidebar-footer { margin-top: auto; padding: 16px; display: flex; flex-direction: column; gap: 8px; border-top: 1px solid #e0e0e0; }
	.sidebar-link { font-size: 13px; color: #666; text-decoration: none; }
	.sidebar-link:hover { color: #333; }

	/* Right Sidebar (Properties) */
	.sidebar-tabs {
		display: flex;
		border-bottom: 1px solid #e0e0e0;
	}
	.sidebar-tab {
		flex: 1;
		text-align: center;
		padding: 12px 16px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		color: #666;
		transition: all 0.2s;
	}
	.sidebar-tab:hover { color: #333; }
	.sidebar-tab.active { border-bottom-color: #000; color: #000; }

	.properties-header {
		font-size: 11px;
		font-weight: 700;
		color: #999;
		text-transform: uppercase;
		margin-bottom: 20px;
		letter-spacing: 0.5px;
	}
	.properties-section-title {
		font-size: 11px;
		font-weight: 700;
		color: #999;
		text-transform: uppercase;
		margin-bottom: 16px;
		letter-spacing: 0.5px;
	}

	.property-group { margin-bottom: 20px; }
	.property-label { display: block; font-size: 13px; color: #555; margin-bottom: 8px; }

	.empty-properties { padding: 40px 20px; text-align: center; color: #999; font-size: 13px; }

	/* Inputs */
	.text-input {
		width: 100%;
		padding: 8px 12px;
		border: 1px solid #e0e0e0;
		border-radius: 4px;
		font-size: 14px;
		transition: all 0.2s;
		background: #fff;
		resize: vertical;
	}
	.text-input:focus { border-color: #2196f3; outline: none; }

	.select-input {
		width: 100%;
		padding: 8px 12px;
		border: 1px solid #e0e0e0;
		border-radius: 4px;
		font-size: 14px;
		background-color: #fff;
		cursor: pointer;
	}

	/* Color Picker Box (email-builder style) */
	.color-picker-box {
		width: 36px;
		height: 36px;
		border-radius: 4px;
		border: 1px solid #e0e0e0;
		cursor: pointer;
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #fff;
		transition: all 0.2s;
	}
	.color-picker-box:hover { border-color: #ccc; }
	.color-picker-box.has-color .color-plus-icon { display: none; }
	.color-plus-icon { color: #999; }
	.color-input-hidden { 
		position: absolute;
		width: 100%;
		height: 100%;
		opacity: 0;
		cursor: pointer;
	}

	/* Slider */
	.slider-input-group { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
	.slider-icon { color: #999; width: 20px; display: flex; justify-content: center; flex-shrink: 0; }
	.slider-range { 
		flex: 1;
		-webkit-appearance: none;
		height: 4px;
		background: #e0e0e0;
		border-radius: 2px;
	}
	.slider-range::-webkit-slider-thumb {
		-webkit-appearance: none;
		width: 14px;
		height: 14px;
		background: #2196f3;
		border-radius: 50%;
		cursor: pointer;
	}
	.slider-value { font-size: 12px; color: #666; width: 45px; text-align: right; flex-shrink: 0; }

	/* Toggle Group */
	.toggle-group {
		display: flex;
		background: #f5f5f5;
		padding: 3px;
		border-radius: 6px;
		gap: 2px;
	}
	.toggle-option {
		flex: 1;
		text-align: center;
		padding: 6px 8px;
		font-size: 12px;
		font-weight: 500;
		color: #666;
		border-radius: 4px;
		cursor: pointer;
		transition: all 0.15s;
	}
	.toggle-option:hover { color: #333; }
	.toggle-option.active { background: #fff; color: #000; box-shadow: 0 1px 2px rgba(0,0,0,0.05); }
	.icon-option { display: flex; align-items: center; justify-content: center; padding: 6px; }

	/* Canvas */
	.builder-canvas {
		display: flex;
		align-items: flex-start;
		justify-content: center;
		overflow: auto;
		padding: 32px;
		transition: background-color 0.3s;
		/* Backdrop color applied via inline style */
	}

	.canvas-area {
		width: 100%;
		display: flex;
		justify-content: center;
		background: transparent;
	}

	.canvas-design-wrapper, .canvas-preview-wrapper {
		width: 600px;
		min-height: 600px;
		transition: width 0.3s ease;
		background: transparent;
	}

	.canvas-design-wrapper.mobile-width, .canvas-preview-wrapper.mobile-width {
		width: 360px;
	}

	.canvas-document, .canvas-preview {
		min-height: 100%;
		position: relative;
		box-shadow: 0 2px 8px rgba(0,0,0,0.08);
		/* Canvas color is applied via inline style */
	}

	.canvas-code-wrapper {
		width: 100%;
		max-width: 100%;
		padding: 0 24px;
	}

	.code-editor-prism {
		width: 100%;
		min-height: 500px;
		max-height: calc(100vh - 150px);
		padding: 20px !important;
		margin: 0 !important;
		border-radius: 8px;
		font-family: 'JetBrains Mono', 'Fira Code', 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
		font-size: 13px;
		line-height: 1.6;
		background: #1e1e1e !important;
		overflow: auto;
		tab-size: 2;
	}

	.code-editor-prism code {
		font-family: inherit;
		background: transparent !important;
		text-shadow: none !important;
	}

	/* Dark theme overrides for Prism */
	.code-editor-prism,
	.code-editor-prism code[class*="language-"] {
		color: #d4d4d4 !important;
		background: #1e1e1e !important;
	}

	.code-editor-prism .token.tag { color: #569cd6; }
	.code-editor-prism .token.attr-name { color: #9cdcfe; }
	.code-editor-prism .token.attr-value { color: #ce9178; }
	.code-editor-prism .token.punctuation { color: #808080; }
	.code-editor-prism .token.string { color: #ce9178; }
	.code-editor-prism .token.property { color: #9cdcfe; }
	.code-editor-prism .token.number { color: #b5cea8; }
	.code-editor-prism .token.boolean { color: #569cd6; }
	.code-editor-prism .token.null { color: #569cd6; }
	.code-editor-prism .token.comment { color: #6a9955; font-style: italic; }
	.code-editor-prism .token.doctype { color: #608b4e; }
	.code-editor-prism .token.prolog { color: #608b4e; }

	.canvas-block {
		position: relative;
		min-height: 32px;
		border: 2px solid transparent;
		transition: all 0.15s;
	}

	.canvas-block:hover { border-color: #e0e0e0; }
	.canvas-block.selected { border-color: #2196f3; }

	/* Block Actions Sidebar */
	.block-actions-sidebar {
		position: absolute;
		left: -44px;
		top: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
		background: #fff;
		padding: 4px;
		border-radius: 8px;
		box-shadow: 0 2px 8px rgba(0,0,0,0.1);
	}

	.block-action-btn {
		width: 28px;
		height: 28px;
		border-radius: 4px;
		border: none;
		background: transparent;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		color: #666;
	}
	.block-action-btn:hover { background: #f0f0f0; color: #333; }
	.delete-btn:hover { color: #f44336; background: #ffebee; }

	/* Add Trigger */
	.block-add-trigger {
		position: absolute;
		bottom: -12px;
		left: 50%;
		transform: translateX(-50%);
		width: 24px;
		height: 24px;
		background: #2196f3;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		z-index: 5;
		opacity: 0;
		transition: opacity 0.2s;
	}
	
	.canvas-block.selected .block-add-trigger { opacity: 1; }

	.add-block-placeholder {
		margin: 16px;
		height: 40px;
		border: 2px dashed #ddd;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		background: transparent;
		transition: all 0.2s;
	}
	.add-block-placeholder:hover { border-color: #2196f3; background: rgba(33,150,243,0.05); }
	.plus-icon { background: #2196f3; border-radius: 50%; padding: 4px; color: #fff; }

	/* Floating Picker */
	.floating-block-picker {
		position: fixed;
		background: #fff;
		border-radius: 8px;
		box-shadow: 0 4px 20px rgba(0,0,0,0.15);
		padding: 16px;
		width: 320px;
		z-index: 100;
		border: 1px solid #e0e0e0;
	}

	.picker-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 8px;
	}

	.picker-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		cursor: pointer;
		padding: 8px 4px;
		border-radius: 6px;
	}
	.picker-item:hover { background: #f5f5f5; }

	.picker-icon {
		width: 36px;
		height: 36px;
		background: #fff;
		border: 1px solid #e0e0e0;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #333;
	}
	.picker-label { font-size: 11px; color: #666; }

	/* Columns */
	.columns-width-inputs { display: flex; flex-direction: column; gap: 8px; }
	.column-width-input { display: flex; align-items: center; gap: 8px; }
	.column-width-input label { font-size: 12px; color: #666; width: 60px; }
	.column-width-input .text-input.small { width: 60px; padding: 4px 8px; font-size: 12px; }
	.column-width-input span { font-size: 12px; color: #999; }

	/* Column add button in preview */
	.column-add-btn { transition: transform 0.2s, background 0.2s; }
	.column-add-btn:hover { transform: scale(1.1); background: #1976d2 !important; }
`

const builderJS = `
	function emailBuilder(initialDoc) {
		return {
			document: initialDoc,
			view: 'design',
			deviceView: 'desktop',
			selectedBlock: null,
			hoverBlock: null,
			showPicker: false,
			pickerTop: 0,
			pickerLeft: 0,
			insertTargetId: null,
			history: [],
			historyIndex: -1,
			
			get blocks() {
				const root = this.document.blocks[this.document.root];
				const childrenIds = root?.data?.childrenIds || [];
				return childrenIds.map(id => ({
					id,
					type: this.document.blocks[id]?.type,
					data: this.document.blocks[id]?.data
				}));
			},
			
			get selectedBlockData() {
				if (!this.selectedBlock || this.selectedBlock === 'root') return null;
				return this.document.blocks[this.selectedBlock]?.data;
			},
			
			get selectedBlockType() {
				if (!this.selectedBlock || this.selectedBlock === 'root') return null;
				return this.document.blocks[this.selectedBlock]?.type;
			},
			
			get documentJSON() {
				return JSON.stringify(this.document, null, 2);
			},
			
			get canUndo() { return this.historyIndex > 0; },
			get canRedo() { return this.historyIndex < this.history.length - 1; },

			// Watch for column count changes
			updateColumnsCount(blockId, newCount) {
				const block = this.document.blocks[blockId];
				if (!block || block.type !== 'Columns') return;
				
				const currentCount = block.data.childrenIds?.length || 0;
				newCount = parseInt(newCount);
				
				if (newCount > currentCount) {
					// Add columns
					for (let i = currentCount; i < newCount; i++) {
						const colId = 'col-' + Date.now() + '-' + i;
						this.document.blocks[colId] = { 
							type: 'Column', 
							data: { style: { backgroundColor: '' }, props: {}, childrenIds: [] } 
						};
						block.data.childrenIds.push(colId);
					}
				} else if (newCount < currentCount) {
					// Remove columns (from the end)
					const removed = block.data.childrenIds.splice(newCount);
					for (const colId of removed) {
						// Also remove children of removed columns
						const col = this.document.blocks[colId];
						if (col?.data?.childrenIds) {
							for (const childId of col.data.childrenIds) {
								delete this.document.blocks[childId];
							}
						}
						delete this.document.blocks[colId];
					}
				}
				
				block.data.props.columnsCount = newCount;
				this.saveHistory();
			},
			
			init() {
				this.saveHistory();
				// Listen for column add button clicks
				document.addEventListener('click', (e) => {
					if (e.target.classList.contains('column-add-btn')) {
						const colId = e.target.dataset.columnId;
						const colIndex = parseInt(e.target.dataset.columnIndex);
						const parentId = e.target.dataset.parentId;
						this.openColumnBlockPicker(parentId, colId, colIndex);
					}
				});
			},

			openColumnBlockPicker(parentId, columnId, columnIndex) {
				this.insertTargetColumn = { parentId, columnId, columnIndex };
				this.showPicker = true;
				this.pickerTop = window.innerHeight / 2 - 100;
				this.pickerLeft = window.innerWidth / 2 - 160;
			},

			getBackdropStyle() {
				const root = this.document.blocks[this.document.root]?.data || {};
				return 'background-color: ' + (root.backdropColor || '#F5F5F5') + ';';
			},

			getCanvasStyle() {
				const root = this.document.blocks[this.document.root]?.data || {};
				let style = 'background-color: ' + (root.canvasColor || '#FFFFFF') + ';';
				style += 'color: ' + (root.textColor || '#242424') + ';';
				style += 'font-family: ' + this.getFontFamily(root.fontFamily) + ';';
				if (root.borderRadius) style += 'border-radius: ' + root.borderRadius + 'px;';
				if (root.borderColor) style += 'border: 1px solid ' + root.borderColor + ';';
				return style;
			},

			getFontFamily(key) {
				const fonts = {
					'MODERN_SANS': '"Helvetica Neue", "Arial Nova", "Nimbus Sans", Arial, sans-serif',
					'BOOK_SANS': 'Optima, Candara, "Noto Sans", source-sans-pro, sans-serif',
					'ORGANIC_SANS': 'Seravek, "Gill Sans Nova", Ubuntu, Calibri, "DejaVu Sans", sans-serif',
					'GEOMETRIC_SANS': 'Avenir, Montserrat, Corbel, "URW Gothic", source-sans-pro, sans-serif',
					'HEAVY_SANS': 'Bahnschrift, "DIN Alternate", "Franklin Gothic Medium", sans-serif',
					'ROUNDED_SANS': 'ui-rounded, "Hiragino Maru Gothic ProN", Quicksand, Comfortaa, sans-serif',
					'MODERN_SERIF': 'Charter, "Bitstream Charter", "Sitka Text", Cambria, serif',
					'BOOK_SERIF': '"Iowan Old Style", "Palatino Linotype", "URW Palladio L", serif',
					'MONOSPACE': '"Nimbus Mono PS", "Courier New", monospace'
				};
				return fonts[key] || fonts['MODERN_SANS'];
			},
			
			selectBlock(blockId) {
				this.selectedBlock = blockId;
				this.showPicker = false;
			},
			
			openBlockPicker(targetId) {
				this.insertTargetId = targetId;
				this.showPicker = true;
				this.pickerTop = window.innerHeight / 2 - 100;
				this.pickerLeft = window.innerWidth / 2 - 160;
			},
			
			addBlock(blockType) {
				const blockId = 'block-' + Date.now();
				const defaultData = this.getDefaultBlockData(blockType);
				
				this.document.blocks[blockId] = { type: blockType, data: defaultData };
				
				// Check if adding to a column
				if (this.insertTargetColumn) {
					const { parentId, columnId, columnIndex } = this.insertTargetColumn;
					let targetColId = columnId;
					
					// If no column exists yet, create one
					if (!targetColId) {
						const parentBlock = this.document.blocks[parentId];
						if (parentBlock && parentBlock.data.childrenIds) {
							targetColId = parentBlock.data.childrenIds[columnIndex];
						}
					}
					
					if (targetColId && this.document.blocks[targetColId]) {
						const col = this.document.blocks[targetColId];
						col.data.childrenIds = col.data.childrenIds || [];
						col.data.childrenIds.push(blockId);
					}
					
					this.insertTargetColumn = null;
				} else {
					// Add to root
					const root = this.document.blocks[this.document.root];
					const children = root.data.childrenIds || [];
					
					if (this.insertTargetId) {
						const index = children.indexOf(this.insertTargetId);
						children.splice(index + 1, 0, blockId);
					} else {
						children.push(blockId);
					}
					root.data.childrenIds = children;
				}
				
				this.showPicker = false;
				this.selectedBlock = blockId;
				this.saveHistory();
			},
			
			deleteBlock(blockId) {
				const root = this.document.blocks[this.document.root];
				root.data.childrenIds = root.data.childrenIds.filter(id => id !== blockId);
				delete this.document.blocks[blockId];
				this.selectedBlock = null;
				this.saveHistory();
			},
			
			moveBlock(blockId, direction) {
				const root = this.document.blocks[this.document.root];
				const children = root.data.childrenIds;
				const index = children.indexOf(blockId);
				
				if (direction === 'up' && index > 0) {
					[children[index], children[index - 1]] = [children[index - 1], children[index]];
				} else if (direction === 'down' && index < children.length - 1) {
					[children[index], children[index + 1]] = [children[index + 1], children[index]];
				}
				this.saveHistory();
			},
			
			renderBlockPreview(block) {
				const data = block.data || {};
				const props = data.props || {};
				const style = this.styleToString(data.style);
				
				if (block.type === 'Text') {
					return '<div style="' + style + '">' + (props.text || 'Text block') + '</div>';
				}
				if (block.type === 'Heading') {
					const level = props.level || 'h2';
					return '<' + level + ' style="margin:0;font-weight:bold;' + style + '">' + (props.text || 'Heading') + '</' + level + '>';
				}
				if (block.type === 'Button') {
					const btnStyle = props.buttonStyle || 'rounded';
					const radius = btnStyle === 'pill' ? '24px' : (btnStyle === 'rounded' ? '4px' : '0');
					const bgColor = props.buttonColor || '#999999';
					return '<div style="' + style + '"><a href="' + (props.url || '#') + '" style="display:inline-block;padding:12px 20px;background:' + bgColor + ';color:#fff;text-decoration:none;border-radius:' + radius + ';font-weight:bold;">' + (props.text || 'Button') + '</a></div>';
				}
				if (block.type === 'Image') {
					const imgUrl = props.url || '';
					if (imgUrl) {
						return '<div style="text-align:' + (props.contentAlignment || 'center') + ';' + style + '"><img src="' + imgUrl + '" alt="' + (props.alt || '') + '" style="max-width:100%;height:auto;" /></div>';
					}
					return '<div style="text-align:center;padding:20px;background:#f0f0f0;color:#999;' + style + '">Image Block</div>';
				}
				if (block.type === 'Divider') {
					const lineColor = props.lineColor || '#e0e0e0';
					const lineHeight = props.lineHeight || 1;
					return '<hr style="border:none;border-top:' + lineHeight + 'px solid ' + lineColor + ';margin:16px 0;" />';
				}
				if (block.type === 'Spacer') {
					return '<div style="height:' + (props.height || 20) + 'px;"></div>';
				}
				if (block.type === 'Columns') {
					return this.renderColumnsPreview(block);
				}
				if (block.type === 'Container') {
					return this.renderContainerPreview(block);
				}
				
				return '<div style="padding:16px;border:1px dashed #ccc;text-align:center;color:#999;">' + block.type + '</div>';
			},

			renderColumnsPreview(block) {
				const data = block.data || {};
				const props = data.props || {};
				const style = data.style || {};
				const columnsCount = props.columnsCount || 2;
				const columnsGap = props.columnsGap || 16;
				const columnIds = data.childrenIds || [];
				const bgColor = style.backgroundColor || 'transparent';
				const padding = style.padding || { top: 16, bottom: 16, left: 24, right: 24 };
				
				let html = '<div style="display:flex;gap:' + columnsGap + 'px;background:' + bgColor + ';padding:' + padding.top + 'px ' + padding.right + 'px ' + padding.bottom + 'px ' + padding.left + 'px;">';
				
				for (let i = 0; i < columnsCount; i++) {
					const colId = columnIds[i];
					const colBlock = colId ? this.document.blocks[colId] : null;
					const colData = colBlock?.data || {};
					const colBg = colData.style?.backgroundColor || '#f8f8f8';
					const colChildrenIds = colData.childrenIds || [];
					
					html += '<div style="flex:1;min-height:60px;background:' + colBg + ';border-radius:4px;position:relative;display:flex;flex-direction:column;align-items:center;justify-content:center;">';
					
					if (colChildrenIds.length > 0) {
						// Render children
						for (const childId of colChildrenIds) {
							const childBlock = this.document.blocks[childId];
							if (childBlock) {
								html += this.renderBlockPreview({ id: childId, type: childBlock.type, data: childBlock.data });
							}
						}
					} else {
						// Empty column - show add button
						html += '<div class="column-add-btn" data-column-id="' + (colId || '') + '" data-column-index="' + i + '" data-parent-id="' + block.id + '" style="width:28px;height:28px;background:#2196f3;border-radius:50%;display:flex;align-items:center;justify-content:center;cursor:pointer;color:#fff;font-size:18px;">+</div>';
					}
					
					html += '</div>';
				}
				
				html += '</div>';
				return html;
			},

			renderContainerPreview(block) {
				const data = block.data || {};
				const style = data.style || {};
				const bgColor = style.backgroundColor || 'transparent';
				const padding = style.padding || { top: 16, bottom: 16, left: 24, right: 24 };
				const childrenIds = data.childrenIds || [];
				
				let html = '<div style="background:' + bgColor + ';padding:' + padding.top + 'px ' + padding.right + 'px ' + padding.bottom + 'px ' + padding.left + 'px;">';
				
				if (childrenIds.length > 0) {
					for (const childId of childrenIds) {
						const childBlock = this.document.blocks[childId];
						if (childBlock) {
							html += this.renderBlockPreview({ id: childId, type: childBlock.type, data: childBlock.data });
						}
					}
				} else {
					html += '<div style="min-height:40px;border:2px dashed #ddd;border-radius:4px;display:flex;align-items:center;justify-content:center;color:#999;">Drop blocks here</div>';
				}
				
				html += '</div>';
				return html;
			},

			getRenderedHTML() {
				// Generate full HTML email output
				const root = this.document.blocks[this.document.root]?.data || {};
				const fontFamily = this.getFontFamily(root.fontFamily);
				
				let html = '<!doctype html>\\n<html>\\n<body>\\n';
				html += '<div style="background-color:' + (root.backdropColor || '#F5F5F5') + ';color:' + (root.textColor || '#262626') + ';font-family:' + fontFamily + ';font-size:16px;font-weight:400;">\\n';
				html += '<table align="center" width="100%" style="margin:0 auto;max-width:600px;background-color:' + (root.canvasColor || '#FFFFFF') + ';border-radius:' + (root.borderRadius || 0) + 'px" role="presentation" cellspacing="0" cellpadding="0" border="0">\\n';
				html += '<tbody><tr style="width:100%"><td>\\n';
				
				for (const block of this.blocks) {
					html += this.renderBlockToHTML(block) + '\\n';
				}
				
				html += '</td></tr></tbody></table>\\n</div>\\n</body>\\n</html>';
				return html;
			},

			renderBlockToHTML(block) {
				const data = block.data || {};
				const props = data.props || {};
				const style = data.style || {};
				const padding = style.padding || { top: 16, bottom: 16, left: 24, right: 24 };
				const paddingStr = padding.top + 'px ' + padding.right + 'px ' + padding.bottom + 'px ' + padding.left + 'px';
				
				if (block.type === 'Heading') {
					const level = props.level || 'h2';
					const fontSize = level === 'h1' ? 32 : (level === 'h2' ? 24 : 20);
					return '<' + level + ' style="font-weight:bold;margin:0;font-size:' + fontSize + 'px;padding:' + paddingStr + '">' + (props.text || '') + '</' + level + '>';
				}
				if (block.type === 'Text') {
					return '<div style="padding:' + paddingStr + '">' + (props.text || '') + '</div>';
				}
				if (block.type === 'Button') {
					const btnStyle = props.buttonStyle || 'rounded';
					const radius = btnStyle === 'pill' ? '24px' : (btnStyle === 'rounded' ? '4px' : '0');
					return '<div style="padding:' + paddingStr + '"><a href="' + (props.url || '#') + '" style="color:#FFFFFF;font-size:16px;font-weight:bold;background-color:' + (props.buttonColor || '#999999') + ';display:inline-block;padding:12px 20px;text-decoration:none;border-radius:' + radius + '" target="_blank">' + (props.text || 'Button') + '</a></div>';
				}
				if (block.type === 'Divider') {
					return '<hr style="border:none;border-top:1px solid #e0e0e0;margin:16px 24px;" />';
				}
				if (block.type === 'Spacer') {
					return '<div style="height:' + (props.height || 20) + 'px;"></div>';
				}
				return '';
			},

			styleToString(style) {
				if (!style) return '';
				let s = '';
				if (style.color) s += 'color:' + style.color + ';';
				if (style.backgroundColor) s += 'background-color:' + style.backgroundColor + ';';
				if (style.fontSize) s += 'font-size:' + style.fontSize + 'px;';
				if (style.fontWeight) s += 'font-weight:' + style.fontWeight + ';';
				if (style.textAlign) s += 'text-align:' + style.textAlign + ';';
				if (style.padding) {
					s += 'padding:' + (style.padding.top||16) + 'px ' + (style.padding.right||24) + 'px ' + (style.padding.bottom||16) + 'px ' + (style.padding.left||24) + 'px;';
				}
				return s;
			},
			
			getDefaultBlockData(blockType) {
				const defaults = {
					style: { 
						padding: { top: 16, bottom: 16, left: 24, right: 24 },
						textAlign: 'left'
					},
					props: {},
					childrenIds: []
				};
				
				if (blockType === 'Heading') {
					defaults.props = { text: 'Hello friend', level: 'h2' };
				} else if (blockType === 'Text') {
					defaults.props = { text: 'This is a text block.' };
				} else if (blockType === 'Button') {
					defaults.props = { text: 'Button', url: 'https://www.usewaypoint.com', buttonColor: '#999999', buttonStyle: 'rounded', size: 'md', fullWidth: 'false' };
				} else if (blockType === 'Spacer') {
					defaults.props = { height: 20 };
				} else if (blockType === 'Divider') {
					defaults.props = { lineColor: '#e0e0e0', lineHeight: 1 };
				} else if (blockType === 'Image') {
					defaults.props = { url: '', alt: '', contentAlignment: 'center' };
				} else if (blockType === 'Columns') {
					defaults.props = { columnsCount: 2, columnsGap: 16 };
					// Create column children
					const col1Id = 'col-' + Date.now() + '-1';
					const col2Id = 'col-' + Date.now() + '-2';
					this.document.blocks[col1Id] = { type: 'Column', data: { style: { backgroundColor: '' }, props: {}, childrenIds: [] } };
					this.document.blocks[col2Id] = { type: 'Column', data: { style: { backgroundColor: '' }, props: {}, childrenIds: [] } };
					defaults.childrenIds = [col1Id, col2Id];
				} else if (blockType === 'Container') {
					defaults.props = {};
				} else {
					defaults.props = { text: 'New ' + blockType };
				}
				
				return defaults;
			},
			
			saveHistory() {
				this.history = this.history.slice(0, this.historyIndex + 1);
				this.history.push(JSON.parse(JSON.stringify(this.document)));
				this.historyIndex++;
			},
			undo() {
				if (this.canUndo) {
					this.historyIndex--;
					this.document = JSON.parse(JSON.stringify(this.history[this.historyIndex]));
				}
			},
			redo() {
				if (this.canRedo) {
					this.historyIndex++;
					this.document = JSON.parse(JSON.stringify(this.history[this.historyIndex]));
				}
			},
			downloadHTML() {
				const html = this.getRenderedHTML();
				const blob = new Blob([html], { type: 'text/html' });
				const url = URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = 'email-template.html';
				a.click();
				URL.revokeObjectURL(url);
			},
			uploadJSON() {
				const input = document.createElement('input');
				input.type = 'file';
				input.accept = '.json';
				input.onchange = (e) => {
					const file = e.target.files[0];
					const reader = new FileReader();
					reader.onload = (event) => {
						try {
							this.document = JSON.parse(event.target.result);
							this.saveHistory();
						} catch (err) {
							alert('Invalid JSON file');
						}
					};
					reader.readAsText(file);
				};
				input.click();
			},
			save() {
				const json = JSON.stringify(this.document);
				console.log('Saving:', json);
				alert('Template saved! Check console for JSON output.');
			},
			loadTemplate(id) {
				console.log('Load template:', id);
				if (id === 'empty') {
					this.document = {
						root: 'root',
						blocks: {
							root: {
								type: 'EmailLayout',
								data: {
									backdropColor: '#F5F5F5',
									canvasColor: '#FFFFFF',
									textColor: '#242424',
									fontFamily: 'MODERN_SANS',
									childrenIds: []
								}
							}
						}
					};
					this.selectedBlock = null;
					this.saveHistory();
				}
			},

			// Syntax highlighting helpers
			escapeHtml(text) {
				const div = document.createElement('div');
				div.textContent = text;
				return div.innerHTML;
			},

			formatHTML(html) {
				// Pretty print HTML with proper indentation
				let formatted = '';
				let indent = 0;
				const tab = '  ';
				
				// Replace escaped newlines with actual newlines
				html = html.replace(/\\n/g, '\n');
				
				// Split by tags
				const tokens = html.split(/(<[^>]+>)/g).filter(t => t.trim());
				
				tokens.forEach(token => {
					if (token.match(/^<\/\w/)) {
						// Closing tag
						indent = Math.max(0, indent - 1);
						formatted += tab.repeat(indent) + token + '\n';
					} else if (token.match(/^<\w[^>]*[^\/]>$/)) {
						// Opening tag (not self-closing)
						formatted += tab.repeat(indent) + token + '\n';
						if (!token.match(/^<(br|hr|img|input|link|meta)/i)) {
							indent++;
						}
					} else if (token.match(/^<\w[^>]*\/>$/)) {
						// Self-closing tag
						formatted += tab.repeat(indent) + token + '\n';
					} else if (token.trim()) {
						// Text content
						formatted += tab.repeat(indent) + token.trim() + '\n';
					}
				});
				
				return formatted.trim();
			},

			escapeAndHighlightHTML(html) {
				if (typeof Prism === 'undefined') {
					return this.escapeHtml(this.formatHTML(html));
				}
				const formatted = this.formatHTML(html);
				return Prism.highlight(formatted, Prism.languages.html, 'html');
			},

			escapeAndHighlightJSON(json) {
				if (typeof Prism === 'undefined') {
					return this.escapeHtml(json);
				}
				return Prism.highlight(json, Prism.languages.json, 'json');
			},

			highlightCode() {
				// Re-highlight if Prism is available
				if (typeof Prism !== 'undefined') {
					Prism.highlightAll();
				}
			}
		}
	}
`
