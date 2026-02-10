package builder

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// BuilderButton returns a button component to launch the builder.
func BuilderButton(basePath string, appID string) g.Node {
	return A(
		Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder", basePath, appID)),
		Class("inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-violet-600 to-indigo-600 text-white rounded-lg hover:from-violet-700 hover:to-indigo-700 transition-all shadow-sm"),
		lucide.Sparkles(Class("size-4")),
		Span(g.Text("Visual Builder")),
	)
}

// TemplatePreviewCard renders a preview card for a template.
func TemplatePreviewCard(template *Document, name, description, category string) g.Node {
	// Render a small preview
	renderer := NewRenderer(template)
	html, _ := renderer.RenderToHTML()

	// Get category color
	categoryColor := getCategoryColor(category)

	return Div(
		Class("group relative bg-white border border-slate-200 rounded-xl overflow-hidden hover:border-violet-400 hover:shadow-lg transition-all duration-200 cursor-pointer dark:bg-slate-800 dark:border-slate-700 dark:hover:border-violet-500"),

		// Preview thumbnail
		Div(
			Class("relative bg-slate-100 border-b border-slate-200 dark:bg-slate-900 dark:border-slate-700"),
			g.Attr("style", "height: 180px; overflow: hidden;"),
			g.El("iframe",
				g.Attr("srcdoc", html),
				g.Attr("style", "width: 200%; height: 360px; transform: scale(0.5); transform-origin: top left; pointer-events: none; border: none;"),
			),
			// Hover overlay - pointer-events-none ensures clicks pass through
			Div(
				Class("absolute inset-0 bg-gradient-to-t from-violet-600/90 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end justify-center pb-4 pointer-events-none"),
				Span(
					Class("inline-flex items-center gap-2 px-4 py-2 bg-white text-violet-700 rounded-lg text-sm font-medium shadow-lg"),
					lucide.Sparkles(Class("size-4")),
					g.Text("Use Template"),
				),
			),
		),

		// Template info
		Div(
			Class("p-4"),
			// Category badge
			Span(
				Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium mb-2"),
				g.Attr("style", fmt.Sprintf("background-color: %s20; color: %s;", categoryColor, categoryColor)),
				g.Text(category),
			),
			H4(Class("font-semibold text-slate-900 dark:text-white"), g.Text(name)),
			P(Class("text-sm text-slate-500 mt-1 dark:text-slate-400"), g.Text(description)),
		),
	)
}

// getCategoryColor returns a color for a category.
func getCategoryColor(category string) string {
	colors := map[string]string{
		"Onboarding":     "#4F46E5", // Indigo
		"Authentication": "#0EA5E9", // Sky
		"Collaboration":  "#10B981", // Emerald
		"Security":       "#EF4444", // Red
		"Transactional":  "#F59E0B", // Amber
		"Marketing":      "#8B5CF6", // Violet
	}
	if color, ok := colors[category]; ok {
		return color
	}

	return "#6B7280" // Gray default
}

// SampleTemplatesGallery renders a gallery of sample templates.
func SampleTemplatesGallery(basePath string, currentApp *app.App) g.Node {
	templates := GetAllTemplateInfo()

	nodes := make([]g.Node, len(templates))
	for i, template := range templates {
		doc, _ := GetSampleTemplate(template.Name)
		t := template // Capture for closure
		nodes[i] = A(
			Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder?sample=%s",
				basePath, currentApp.ID, t.Name)),
			TemplatePreviewCard(doc, t.DisplayName, t.Description, t.Category),
		)
	}

	return Div(
		Class("space-y-6"),
		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H3(
					Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Start from a Template"),
				),
				P(
					Class("text-sm text-slate-500 dark:text-slate-400"),
					g.Text("Choose a professionally designed template and customize it"),
				),
			),
		),
		// Grid
		Div(
			Class("grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5"),
			g.Group(nodes),
		),
	)
}

// SampleTemplatesCompact renders a compact version for sidebars/modals.
func SampleTemplatesCompact(basePath string, currentApp *app.App) g.Node {
	templates := GetAllTemplateInfo()

	nodes := make([]g.Node, len(templates))
	for i, template := range templates {
		t := template // Capture for closure
		categoryColor := getCategoryColor(t.Category)
		nodes[i] = A(
			Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder?sample=%s",
				basePath, currentApp.ID, t.Name)),
			Class("flex items-center gap-3 p-3 rounded-lg border border-slate-200 hover:border-violet-400 hover:bg-violet-50 transition-all dark:border-slate-700 dark:hover:border-violet-500 dark:hover:bg-violet-900/20"),
			// Icon
			Div(
				Class("flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center"),
				g.Attr("style", fmt.Sprintf("background-color: %s20;", categoryColor)),
				getTemplateIcon(t.Name, categoryColor),
			),
			// Info
			Div(
				Class("flex-1 min-w-0"),
				Div(Class("font-medium text-sm text-slate-900 dark:text-white"), g.Text(t.DisplayName)),
				Div(Class("text-xs text-slate-500 truncate dark:text-slate-400"), g.Text(t.Description)),
			),
			// Arrow
			lucide.ChevronRight(Class("size-4 text-slate-400")),
		)
	}

	return Div(
		Class("space-y-2"),
		g.Group(nodes),
	)
}

// getTemplateIcon returns an icon for a template type.
func getTemplateIcon(name, color string) g.Node {
	iconClass := "size-5"
	style := fmt.Sprintf("color: %s;", color)

	switch name {
	case "welcome":
		return lucide.PartyPopper(Class(iconClass), g.Attr("style", style))
	case "otp":
		return lucide.KeyRound(Class(iconClass), g.Attr("style", style))
	case "reset_password":
		return lucide.Lock(Class(iconClass), g.Attr("style", style))
	case "invitation":
		return lucide.UserPlus(Class(iconClass), g.Attr("style", style))
	case "magic_link":
		return lucide.Sparkles(Class(iconClass), g.Attr("style", style))
	case "order_confirmation":
		return lucide.ShoppingBag(Class(iconClass), g.Attr("style", style))
	case "newsletter":
		return lucide.Newspaper(Class(iconClass), g.Attr("style", style))
	case "account_alert":
		return lucide.ShieldAlert(Class(iconClass), g.Attr("style", style))
	default:
		return lucide.Mail(Class(iconClass), g.Attr("style", style))
	}
}

// BuilderIntegrationCard renders an info card about the builder.
func BuilderIntegrationCard() g.Node {
	return Div(
		Class("bg-gradient-to-br from-violet-50 to-indigo-50 border border-violet-200 rounded-xl p-6 dark:from-violet-900/20 dark:to-indigo-900/20 dark:border-violet-800"),

		Div(
			Class("flex items-start gap-4"),

			// Icon
			Div(
				Class("flex-shrink-0 w-12 h-12 bg-gradient-to-br from-violet-600 to-indigo-600 rounded-xl flex items-center justify-center shadow-lg"),
				lucide.Sparkles(Class("size-6 text-white")),
			),

			// Content
			Div(
				Class("flex-1"),
				H3(Class("text-lg font-semibold text-slate-900 mb-2 dark:text-white"), g.Text("Visual Email Builder")),
				P(
					Class("text-sm text-slate-600 mb-4 dark:text-slate-300"),
					g.Text("Design beautiful, responsive email templates visually with our drag-and-drop builder. No HTML knowledge required."),
				),

				// Features list
				Ul(
					Class("grid grid-cols-2 gap-2"),
					featureListItem("Drag & drop blocks"),
					featureListItem("Real-time preview"),
					featureListItem("Mobile responsive"),
					featureListItem("Auto-save"),
					featureListItem("Pre-built templates"),
					featureListItem("HTML export"),
				),
			),
		),
	)
}

// featureListItem renders a feature list item with check icon.
func featureListItem(text string) g.Node {
	return Li(
		Class("flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300"),
		lucide.Check(Class("size-4 text-emerald-500")),
		Span(g.Text(text)),
	)
}

// TemplateSelector renders a template selection component for the create modal.
func TemplateSelector(basePath string, currentApp *app.App) g.Node {
	templates := GetAllTemplateInfo()

	// Group by category
	categories := make(map[string][]TemplateInfo)
	for _, t := range templates {
		categories[t.Category] = append(categories[t.Category], t)
	}

	categoryOrder := []string{"Onboarding", "Authentication", "Collaboration", "Security", "Transactional", "Marketing"}

	var categoryNodes []g.Node

	for _, category := range categoryOrder {
		if tmps, ok := categories[category]; ok {
			categoryColor := getCategoryColor(category)

			var templateNodes []g.Node

			for _, t := range tmps {
				// Capture
				templateNodes = append(templateNodes, Div(
					Class("template-option cursor-pointer p-3 rounded-lg border-2 border-transparent hover:border-violet-400 hover:bg-violet-50 transition-all dark:hover:bg-violet-900/20"),
					g.Attr("@click", fmt.Sprintf("selectTemplate('%s')", t.Name)),
					g.Attr(":class", fmt.Sprintf("{'border-violet-500 bg-violet-50 dark:bg-violet-900/30': selectedTemplate === '%s'}", t.Name)),
					Div(
						Class("flex items-center gap-3"),
						Div(
							Class("w-10 h-10 rounded-lg flex items-center justify-center"),
							g.Attr("style", fmt.Sprintf("background-color: %s20;", categoryColor)),
							getTemplateIcon(t.Name, categoryColor),
						),
						Div(
							Class("flex-1"),
							Div(Class("font-medium text-sm text-slate-900 dark:text-white"), g.Text(t.DisplayName)),
							Div(Class("text-xs text-slate-500 dark:text-slate-400"), g.Text(t.Description)),
						),
						// Selected check
						Div(
							g.Attr("x-show", fmt.Sprintf("selectedTemplate === '%s'", t.Name)),
							g.Attr("x-cloak", ""),
							lucide.CircleCheck(Class("size-5 text-violet-600")),
						),
					),
				))
			}

			categoryNodes = append(categoryNodes, Div(
				Class("space-y-2"),
				// Category header
				Div(
					Class("flex items-center gap-2 py-2"),
					Span(
						Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"),
						g.Attr("style", fmt.Sprintf("background-color: %s20; color: %s;", categoryColor, categoryColor)),
						g.Text(category),
					),
				),
				// Templates
				Div(
					Class("grid gap-2"),
					g.Group(templateNodes),
				),
			))
		}
	}

	return Div(
		Class("space-y-4 max-h-96 overflow-y-auto pr-2"),
		g.Attr("x-data", "{selectedTemplate: ''}"),
		// Blank option
		Div(
			Class("template-option cursor-pointer p-3 rounded-lg border-2 border-transparent hover:border-violet-400 hover:bg-violet-50 transition-all dark:hover:bg-violet-900/20"),
			g.Attr("@click", "selectTemplate('')"),
			g.Attr(":class", "{'border-violet-500 bg-violet-50 dark:bg-violet-900/30': selectedTemplate === ''}"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center dark:bg-slate-700"),
					lucide.Plus(Class("size-5 text-slate-500")),
				),
				Div(
					Class("flex-1"),
					Div(Class("font-medium text-sm text-slate-900 dark:text-white"), g.Text("Blank Template")),
					Div(Class("text-xs text-slate-500 dark:text-slate-400"), g.Text("Start from scratch")),
				),
				Div(
					g.Attr("x-show", "selectedTemplate === ''"),
					g.Attr("x-cloak", ""),
					lucide.CircleCheck(Class("size-5 text-violet-600")),
				),
			),
		),
		// Divider
		Div(Class("border-t border-slate-200 dark:border-slate-700 my-4")),
		// Categories
		g.Group(categoryNodes),
	)
}
