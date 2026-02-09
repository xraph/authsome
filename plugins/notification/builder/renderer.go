package builder

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Renderer converts email builder documents to HTML.
type Renderer struct {
	document *Document
}

// NewRenderer creates a new renderer for the given document.
func NewRenderer(document *Document) *Renderer {
	return &Renderer{document: document}
}

// Render renders the document to gomponents Node.
func (r *Renderer) Render() (g.Node, error) {
	if err := r.document.Validate(); err != nil {
		return nil, fmt.Errorf("invalid document: %w", err)
	}

	return r.renderBlock(r.document.Root), nil
}

// RenderToHTML renders the document to HTML string.
func (r *Renderer) RenderToHTML() (string, error) {
	node, err := r.Render()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := node.Render(&sb); err != nil {
		return "", fmt.Errorf("failed to render HTML: %w", err)
	}

	return sb.String(), nil
}

// renderBlock renders a single block based on its type.
func (r *Renderer) renderBlock(blockID string) g.Node {
	block, exists := r.document.Blocks[blockID]
	if !exists {
		return g.Text("")
	}

	switch block.Type {
	case BlockTypeEmailLayout:
		return r.renderEmailLayout(block)
	case BlockTypeText:
		return r.renderText(block)
	case BlockTypeHeading:
		return r.renderHeading(block)
	case BlockTypeButton:
		return r.renderButton(block)
	case BlockTypeImage:
		return r.renderImage(block)
	case BlockTypeDivider:
		return r.renderDivider(block)
	case BlockTypeSpacer:
		return r.renderSpacer(block)
	case BlockTypeContainer:
		return r.renderContainer(block)
	case BlockTypeColumns:
		return r.renderColumns(block)
	case BlockTypeColumn:
		return r.renderColumn(block)
	case BlockTypeHTML:
		return r.renderHTML(block)
	case BlockTypeAvatar:
		return r.renderAvatar(block)
	default:
		return g.Text("")
	}
}

// renderEmailLayout renders the root email layout.
func (r *Renderer) renderEmailLayout(block Block) g.Node {
	data := block.Data
	backdropColor := getString(data, "backdropColor", "#F8F8F8")
	canvasColor := getString(data, "canvasColor", "#FFFFFF")
	textColor := getString(data, "textColor", "#242424")
	linkColor := getString(data, "linkColor", "#0066CC")
	fontFamily := getString(data, "fontFamily", "system-ui, sans-serif")
	childrenIDs := getStringArray(data, "childrenIds")

	return Doctype(
		HTML(
			Head(
				Meta(Charset("utf-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				Meta(g.Attr("http-equiv", "X-UA-Compatible"), Content("IE=edge")),
				// Tailwind CSS CDN for utility classes support
				Script(Src("https://cdn.tailwindcss.com")),
				// Custom email styles
				StyleEl(g.Raw(fmt.Sprintf(`
					body { margin: 0; padding: 0; background-color: %s; font-family: %s; color: %s; }
					table { border-collapse: collapse; }
					img { border: 0; outline: none; text-decoration: none; -ms-interpolation-mode: bicubic; }
					a { color: %s; text-decoration: none; }
					.email-container { max-width: 600px; margin: 0 auto; }
				`, backdropColor, fontFamily, textColor, linkColor))),
			),
			Body(
				g.Attr("style", fmt.Sprintf("margin: 0; padding: 20px 0; background-color: %s;", backdropColor)),
				Table(
					g.Attr("role", "presentation"),
					g.Attr("style", "width: 100%; border: 0; cellspacing: 0; cellpadding: 0;"),
					Tr(
						Td(
							g.Attr("align", "center"),
							Div(
								Class("email-container"),
								g.Attr("style", fmt.Sprintf("background-color: %s; border-radius: 8px; overflow: hidden;", canvasColor)),
								g.Group(r.renderChildren(childrenIDs)),
							),
						),
					),
				),
			),
		),
	)
}

// renderText renders a text block.
func (r *Renderer) renderText(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	text := getString(props, "text", "")
	styleStr := r.buildStyle(style)

	return Div(
		g.Attr("style", styleStr),
		g.Raw(text), // Allow HTML in text
	)
}

// renderHeading renders a heading block.
func (r *Renderer) renderHeading(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	text := getString(props, "text", "")
	level := getString(props, "level", "h2")
	styleStr := r.buildStyle(style)

	// Default heading styles
	defaultSizes := map[string]string{
		"h1": "32px", "h2": "28px", "h3": "24px",
		"h4": "20px", "h5": "18px", "h6": "16px",
	}

	if !strings.Contains(styleStr, "font-size") {
		if size, ok := defaultSizes[level]; ok {
			styleStr += "; font-size: " + size
		}
	}

	switch level {
	case "h1":
		return H1(g.Attr("style", styleStr), g.Text(text))
	case "h2":
		return H2(g.Attr("style", styleStr), g.Text(text))
	case "h3":
		return H3(g.Attr("style", styleStr), g.Text(text))
	case "h4":
		return H4(g.Attr("style", styleStr), g.Text(text))
	case "h5":
		return H5(g.Attr("style", styleStr), g.Text(text))
	case "h6":
		return H6(g.Attr("style", styleStr), g.Text(text))
	default:
		return H2(g.Attr("style", styleStr), g.Text(text))
	}
}

// renderButton renders a button block.
func (r *Renderer) renderButton(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	text := getString(props, "text", "Click Here")
	url := getString(props, "url", "#")
	buttonColor := getString(props, "buttonColor", "#0066CC")
	textColor := getString(props, "textColor", "#FFFFFF")
	borderRadius := getInt(props, "borderRadius", 4)
	fullWidth := getBool(props, "fullWidth", false)

	containerStyle := r.buildStyle(style)
	if !strings.Contains(containerStyle, "text-align") {
		containerStyle += "; text-align: center"
	}

	buttonStyle := fmt.Sprintf(
		"display: inline-block; padding: 12px 24px; background-color: %s; color: %s; border-radius: %dpx; text-decoration: none; font-weight: 600;",
		buttonColor, textColor, borderRadius,
	)

	if fullWidth {
		buttonStyle += " width: 100%; text-align: center;"
	}

	return Div(
		g.Attr("style", containerStyle),
		A(
			Href(url),
			g.Attr("style", buttonStyle),
			g.Text(text),
		),
	)
}

// renderImage renders an image block.
func (r *Renderer) renderImage(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	url := getString(props, "url", "")
	alt := getString(props, "alt", "")
	linkURL := getString(props, "linkUrl", "")
	width := getString(props, "width", "100%")
	height := getString(props, "height", "auto")
	alignment := getString(props, "contentAlignment", "center")

	containerStyle := r.buildStyle(style)
	if !strings.Contains(containerStyle, "text-align") {
		containerStyle += "; text-align: " + alignment
	}

	imgStyle := fmt.Sprintf("max-width: %s; height: %s; display: block;", width, height)
	if alignment == "center" {
		imgStyle += " margin: 0 auto;"
	}

	img := Img(
		Src(url),
		Alt(alt),
		g.Attr("style", imgStyle),
	)

	if linkURL != "" {
		return Div(
			g.Attr("style", containerStyle),
			A(Href(linkURL), img),
		)
	}

	return Div(g.Attr("style", containerStyle), img)
}

// renderDivider renders a divider block.
func (r *Renderer) renderDivider(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	lineColor := getString(props, "lineColor", "#E5E5E5")
	lineHeight := getInt(props, "lineHeight", 1)

	containerStyle := r.buildStyle(style)
	hrStyle := fmt.Sprintf("border: 0; border-top: %dpx solid %s; margin: 0;", lineHeight, lineColor)

	return Div(
		g.Attr("style", containerStyle),
		Hr(g.Attr("style", hrStyle)),
	)
}

// renderSpacer renders a spacer block.
func (r *Renderer) renderSpacer(block Block) g.Node {
	data := block.Data
	props := getMap(data, "props")

	height := getInt(props, "height", 20)

	return Div(g.Attr("style", fmt.Sprintf("height: %dpx; line-height: %dpx;", height, height)), g.Text("\u00A0"))
}

// renderContainer renders a container block.
func (r *Renderer) renderContainer(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")
	childrenIDs := getStringArray(data, "childrenIds")

	backgroundColor := getString(props, "backgroundColor", "transparent")
	containerStyle := r.buildStyle(style)

	if backgroundColor != "transparent" && backgroundColor != "" {
		containerStyle += "; background-color: " + backgroundColor
	}

	return Div(
		g.Attr("style", containerStyle),
		g.Group(r.renderChildren(childrenIDs)),
	)
}

// renderColumns renders a columns block.
func (r *Renderer) renderColumns(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	childrenIDs := getStringArray(data, "childrenIds")

	containerStyle := r.buildStyle(style)

	return Table(
		g.Attr("role", "presentation"),
		g.Attr("style", "width: 100%; "+containerStyle),
		Tr(
			g.Group(r.renderChildren(childrenIDs)),
		),
	)
}

// renderColumn renders a column block.
func (r *Renderer) renderColumn(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")
	childrenIDs := getStringArray(data, "childrenIds")

	width := getString(props, "width", "auto")
	cellStyle := r.buildStyle(style)

	if width != "auto" && width != "" {
		cellStyle += "; width: " + width
	}

	return Td(
		g.Attr("style", cellStyle+" vertical-align: top;"),
		g.Group(r.renderChildren(childrenIDs)),
	)
}

// renderHTML renders raw HTML block.
func (r *Renderer) renderHTML(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	html := getString(props, "html", "")
	containerStyle := r.buildStyle(style)

	return Div(
		g.Attr("style", containerStyle),
		g.Raw(html),
	)
}

// renderAvatar renders an avatar block.
func (r *Renderer) renderAvatar(block Block) g.Node {
	data := block.Data
	style := getMap(data, "style")
	props := getMap(data, "props")

	imageURL := getString(props, "imageUrl", "")
	alt := getString(props, "alt", "Avatar")
	size := getInt(props, "size", 64)
	shape := getString(props, "shape", "circle")

	containerStyle := r.buildStyle(style)
	if !strings.Contains(containerStyle, "text-align") {
		containerStyle += "; text-align: center"
	}

	borderRadius := "50%"
	switch shape {
	case "square":
		borderRadius = "0"
	case "rounded":
		borderRadius = "8px"
	}

	imgStyle := fmt.Sprintf(
		"width: %dpx; height: %dpx; border-radius: %s; object-fit: cover;",
		size, size, borderRadius,
	)

	return Div(
		g.Attr("style", containerStyle),
		Img(
			Src(imageURL),
			Alt(alt),
			g.Attr("style", imgStyle),
		),
	)
}

// renderChildren renders multiple child blocks.
func (r *Renderer) renderChildren(childrenIDs []string) []g.Node {
	nodes := make([]g.Node, 0, len(childrenIDs))
	for _, childID := range childrenIDs {
		nodes = append(nodes, r.renderBlock(childID))
	}

	return nodes
}

// buildStyle builds a CSS style string from style map.
func (r *Renderer) buildStyle(styleMap map[string]any) string {
	var styles []string

	if color := getString(styleMap, "color", ""); color != "" {
		styles = append(styles, "color: "+color)
	}

	if bgColor := getString(styleMap, "backgroundColor", ""); bgColor != "" {
		styles = append(styles, "background-color: "+bgColor)
	}

	if fontFamily := getString(styleMap, "fontFamily", ""); fontFamily != "" {
		styles = append(styles, "font-family: "+fontFamily)
	}

	if fontSize := getInt(styleMap, "fontSize", 0); fontSize > 0 {
		styles = append(styles, fmt.Sprintf("font-size: %dpx", fontSize))
	}

	if fontWeight := getString(styleMap, "fontWeight", ""); fontWeight != "" {
		styles = append(styles, "font-weight: "+fontWeight)
	}

	if textAlign := getString(styleMap, "textAlign", ""); textAlign != "" {
		styles = append(styles, "text-align: "+textAlign)
	}

	// Handle padding
	if paddingMap, ok := styleMap["padding"].(map[string]any); ok {
		top := getInt(paddingMap, "top", 0)
		right := getInt(paddingMap, "right", 0)
		bottom := getInt(paddingMap, "bottom", 0)
		left := getInt(paddingMap, "left", 0)
		styles = append(styles, fmt.Sprintf("padding: %dpx %dpx %dpx %dpx", top, right, bottom, left))
	}

	return strings.Join(styles, "; ")
}

// Helper functions to safely extract values from maps

func getString(m map[string]any, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}

	return defaultVal
}

func getInt(m map[string]any, key string, defaultVal int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}

	if v, ok := m[key].(int); ok {
		return v
	}

	return defaultVal
}

func getBool(m map[string]any, key string, defaultVal bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}

	return defaultVal
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}

	return make(map[string]any)
}

func getStringArray(m map[string]any, key string) []string {
	if v, ok := m[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}

		return result
	}

	if v, ok := m[key].([]string); ok {
		return v
	}

	return []string{}
}
