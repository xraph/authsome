package pages

import (
	"fmt"

	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/notification/builder"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// EmailBuilderPage renders the visual email template builder.
func EmailBuilderPage(currentApp *app.App, basePath string, templateID string, document *builder.Document) g.Node {
	appID := currentApp.ID.String()

	// Build URLs for builder operations
	previewURL := fmt.Sprintf("%s/app/%s/notifications/builder/preview", basePath, appID)
	saveURL := fmt.Sprintf("%s/app/%s/notifications/builder/save", basePath, appID)
	backURL := fmt.Sprintf("%s/app/%s/notifications/templates", basePath, appID)

	// Create builder UI instance
	var builderUI *builder.BuilderUI
	if templateID != "" {
		builderUI = builder.NewBuilderUIWithAutosave(document, previewURL, saveURL, backURL, templateID)
	} else {
		builderUI = builder.NewBuilderUI(document, previewURL, saveURL)
	}

	// Wrap builder in ForgeUI container
	return primitives.Container(
		Div(
			Class("email-builder-wrapper"),

			// Render the complete builder interface
			builderUI.Render(),

			// Additional bridge integration
			Script(g.Raw(fmt.Sprintf(`
				// Bridge integration for email builder
				window.emailBuilderBridge = {
					appId: '%s',
					templateId: '%s',
					basePath: '%s',
					
					async saveTemplate(name, templateKey, subject, builderJSON) {
						try {
							const input = {
								appId: this.appId,
								name: name,
								templateKey: templateKey,
								subject: subject,
								builderJson: builderJSON
							};
							
							if (this.templateId) {
								input.templateId = this.templateId;
							}
							
							const result = await $bridge.call('notification.saveBuilderTemplate', input);
							return { success: true, message: result.message };
						} catch (err) {
							console.error('Failed to save builder template:', err);
							return { success: false, error: err.message };
						}
					},
					
					goBack() {
						window.location.href = this.basePath + '/app/' + this.appId + '/notifications/templates';
					}
				};
			`, appID, templateID, basePath))),
		),
	)
}
