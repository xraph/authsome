package builder

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BuilderButton returns a button component to launch the builder
func BuilderButton(basePath string, appID string) g.Node {
	return A(
		Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder", basePath, appID)),
		Class("inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-violet-600 to-indigo-600 text-white rounded-lg hover:from-violet-700 hover:to-indigo-700 transition-all shadow-sm"),
		lucide.Sparkles(Class("size-4")),
		Span(g.Text("Visual Builder")),
	)
}

// TemplatePreviewCard renders a preview card for a template
func TemplatePreviewCard(template *Document, name, description string) g.Node {
	// Render a small preview
	renderer := NewRenderer(template)
	html, _ := renderer.RenderToHTML()

	return Div(
		Class("border border-gray-200 rounded-lg overflow-hidden hover:border-blue-500 transition-colors cursor-pointer dark:border-gray-700"),

		// Preview thumbnail
		Div(
			Class("bg-gray-50 p-4 border-b border-gray-200 dark:bg-gray-800 dark:border-gray-700"),
			g.Attr("style", "height: 200px; overflow: hidden;"),
			g.El("iframe",
				g.Attr("srcdoc", html),
				g.Attr("style", "width: 100%; height: 400px; transform: scale(0.5); transform-origin: top left; pointer-events: none;"),
			),
		),

		// Template info
		Div(
			Class("p-4"),
			H4(Class("font-medium text-gray-900 dark:text-white"), g.Text(name)),
			P(Class("text-sm text-gray-600 mt-1 dark:text-gray-400"), g.Text(description)),
		),
	)
}

// SampleTemplatesGallery renders a gallery of sample templates
func SampleTemplatesGallery(basePath string, currentApp *app.App) g.Node {
	templates := []struct {
		Name        string
		DisplayName string
		Description string
	}{
		{"welcome", "Welcome Email", "Greet new users warmly"},
		{"otp", "OTP Verification", "Send one-time passwords"},
		{"reset_password", "Password Reset", "Secure password reset link"},
		{"invitation", "Invitation", "Invite users to organizations"},
		{"notification", "Notification", "General purpose alerts"},
	}

	nodes := make([]g.Node, len(templates))
	for i, template := range templates {
		doc, _ := GetSampleTemplate(template.Name)
		nodes[i] = A(
			Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder?sample=%s",
				basePath, currentApp.ID, template.Name)),
			TemplatePreviewCard(doc, template.DisplayName, template.Description),
		)
	}

	return Div(
		Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
		g.Group(nodes),
	)
}

// BuilderIntegrationCard renders an info card about the builder
func BuilderIntegrationCard() g.Node {
	return Div(
		Class("bg-gradient-to-br from-blue-50 to-indigo-50 border border-blue-200 rounded-xl p-6 dark:from-blue-900/20 dark:to-indigo-900/20 dark:border-blue-800"),

		Div(
			Class("flex items-start gap-4"),

			// Icon
			Div(
				Class("flex-shrink-0 w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center"),
				lucide.Sparkles(Class("size-6 text-white")),
			),

			// Content
			Div(
				Class("flex-1"),
				H3(Class("text-lg font-semibold text-gray-900 mb-2 dark:text-white"), g.Text("Email Template Builder")),
				P(
					Class("text-sm text-gray-700 mb-4 dark:text-gray-300"),
					g.Text("Design beautiful, responsive email templates visually with our drag-and-drop builder. No HTML knowledge required."),
				),

				// Features list
				Ul(
					Class("space-y-2 mb-4"),
					featureListItem("Drag & drop block editor"),
					featureListItem("Real-time preview"),
					featureListItem("Mobile responsive"),
					featureListItem("Sample templates included"),
				),
			),
		),
	)
}

// featureListItem renders a feature list item with check icon
func featureListItem(text string) g.Node {
	return Li(
		Class("flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"),
		lucide.Check(Class("size-4 text-green-600 dark:text-green-400")),
		Span(g.Text(text)),
	)
}
