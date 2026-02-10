package pages

import (
	"encoding/json"
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// TemplateFormModal renders a complete template form in a modal.
func TemplateFormModal(appID string) g.Node {
	return Div(
		g.Attr("x-show", "showCreateModal || showEditModal"),
		g.Attr("@click.outside", "closeModal()"),
		g.Attr("@keydown.escape.window", "closeModal()"),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"),

		// Modal content
		Div(
			g.Attr("@click.stop", ""),
			Class("w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-lg bg-white shadow-xl dark:bg-gray-900"),

			// Header
			Div(
				Class("sticky top-0 z-10 flex items-center justify-between border-b bg-white px-6 py-4 dark:bg-gray-900"),
				H2(
					Class("text-xl font-semibold"),
					g.Attr("x-text", "showCreateModal ? 'Create Template' : 'Edit Template'"),
				),
				Button(
					Type("button"),
					g.Attr("@click", "closeModal()"),
					Class("text-muted-foreground hover:text-foreground"),
					lucide.X(Class("h-5 w-5")),
				),
			),

			// Form body
			Div(
				Class("p-6 space-y-6"),

				// Error message
				ErrorMessage("formError"),

				// Template name
				Div(
					Class("space-y-2"),
					Label(
						For("template-name"),
						Class("text-sm font-medium"),
						g.Text("Template Name"),
						Span(Class("text-red-500 ml-1"), g.Text("*")),
					),
					Input(
						Type("text"),
						ID("template-name"),
						Placeholder("Welcome Email"),
						g.Attr("x-model", "form.name"),
						g.Attr("@blur", "validateField('name')"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
					),
					Div(
						g.Attr("x-show", "errors.name"),
						Class("text-sm text-red-500"),
						g.Attr("x-text", "errors.name"),
					),
				),

				// Template key with suggestions
				Div(
					Class("space-y-2"),
					Label(
						For("template-key"),
						Class("text-sm font-medium"),
						g.Text("Template Key"),
						Span(Class("text-red-500 ml-1"), g.Text("*")),
					),
					Input(
						Type("text"),
						ID("template-key"),
						Placeholder("auth.welcome"),
						g.Attr("x-model", "form.templateKey"),
						g.Attr("@blur", "validateField('templateKey')"),
						g.Attr("list", "template-key-suggestions"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
					),
					DataList(
						ID("template-key-suggestions"),
						Option(Value("auth.welcome"), g.Text("auth.welcome - Welcome email")),
						Option(Value("auth.verification"), g.Text("auth.verification - Email verification")),
						Option(Value("auth.password_reset"), g.Text("auth.password_reset - Password reset")),
						Option(Value("auth.mfa_code"), g.Text("auth.mfa_code - MFA code")),
						Option(Value("auth.magic_link"), g.Text("auth.magic_link - Magic link")),
						Option(Value("org.invite"), g.Text("org.invite - Organization invitation")),
						Option(Value("org.member_added"), g.Text("org.member_added - Member added")),
						Option(Value("org.member_removed"), g.Text("org.member_removed - Member removed")),
						Option(Value("session.new_device"), g.Text("session.new_device - New device login")),
						Option(Value("session.suspicious"), g.Text("session.suspicious - Suspicious activity")),
						Option(Value("account.email_changed"), g.Text("account.email_changed - Email changed")),
						Option(Value("account.password_changed"), g.Text("account.password_changed - Password changed")),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("Format: category.action (e.g., auth.welcome)")),
					Div(
						g.Attr("x-show", "errors.templateKey"),
						Class("text-sm text-red-500"),
						g.Attr("x-text", "errors.templateKey"),
					),
				),

				// Type selector
				Div(
					Class("space-y-2"),
					Label(
						For("template-type"),
						Class("text-sm font-medium"),
						g.Text("Type"),
						Span(Class("text-red-500 ml-1"), g.Text("*")),
					),
					Select(
						ID("template-type"),
						g.Attr("x-model", "form.type"),
						g.Attr("@change", "validateField('type')"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
						Option(Value("email"), g.Text("Email")),
						Option(Value("sms"), g.Text("SMS")),
						Option(Value("push"), g.Text("Push Notification")),
					),
				),

				// Language code
				Div(
					Class("space-y-2"),
					Label(
						For("template-language"),
						Class("text-sm font-medium"),
						g.Text("Language Code"),
					),
					Input(
						Type("text"),
						ID("template-language"),
						Placeholder("en"),
						g.Attr("x-model", "form.language"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("ISO 639-1 language code (e.g., en, es, fr)")),
				),

				// Subject (conditional - only for email)
				Div(
					g.Attr("x-show", "form.type === 'email'"),
					Class("space-y-2"),
					Label(
						For("template-subject"),
						Class("text-sm font-medium"),
						g.Text("Subject"),
						Span(
							g.Attr("x-show", "form.type === 'email'"),
							Class("text-red-500 ml-1"),
							g.Text("*"),
						),
					),
					Input(
						Type("text"),
						ID("template-subject"),
						Placeholder("Welcome to {{appName}}!"),
						g.Attr("x-model", "form.subject"),
						g.Attr("@blur", "validateField('subject')"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
					),
					Div(
						g.Attr("x-show", "errors.subject"),
						Class("text-sm text-red-500"),
						g.Attr("x-text", "errors.subject"),
					),
				),

				// Body textarea
				Div(
					Class("space-y-2"),
					Label(
						For("template-body"),
						Class("text-sm font-medium"),
						g.Text("Template Body"),
						Span(Class("text-red-500 ml-1"), g.Text("*")),
					),
					Textarea(
						ID("template-body"),
						Placeholder("Hello {{username}},\n\nWelcome to {{appName}}!"),
						g.Attr("x-model", "form.body"),
						g.Attr("@blur", "validateField('body')"),
						Rows("10"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("Use {{variableName}} for dynamic content")),
					Div(
						g.Attr("x-show", "errors.body"),
						Class("text-sm text-red-500"),
						g.Attr("x-text", "errors.body"),
					),
				),

				// Variables (comma-separated input)
				Div(
					Class("space-y-2"),
					Label(
						For("template-variables"),
						Class("text-sm font-medium"),
						g.Text("Available Variables"),
					),
					Input(
						Type("text"),
						ID("template-variables"),
						Placeholder("username, email, appName"),
						g.Attr("x-model", "variablesInput"),
						g.Attr("@input", "updateVariablesArray()"),
						Class("w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
					),
					P(Class("text-xs text-muted-foreground"),
						g.Text("Comma-separated list of variable names")),
					// Display variables as badges
					Div(
						g.Attr("x-show", "form.variables.length > 0"),
						Class("flex flex-wrap gap-2 mt-2"),
						Template(
							g.Attr("x-for", "variable in form.variables"),
							g.Attr(":key", "variable"),
							Span(
								Class("inline-flex items-center gap-1 rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/20 dark:text-blue-400"),
								g.Text("{{"),
								g.Attr("x-text", "variable"),
								g.Text("}}"),
							),
						),
					),
				),

				// Active toggle
				Div(
					Class("flex items-center gap-2"),
					Input(
						Type("checkbox"),
						ID("template-active"),
						g.Attr("x-model", "form.active"),
						Class("h-4 w-4 rounded border-gray-300"),
					),
					Label(
						For("template-active"),
						Class("text-sm font-medium"),
						g.Text("Active"),
					),
					P(Class("text-xs text-muted-foreground ml-6"),
						g.Text("Inactive templates will not be used for sending")),
				),
			),

			// Footer
			Div(
				Class("sticky bottom-0 flex items-center justify-end gap-2 border-t bg-white px-6 py-4 dark:bg-gray-900"),
				Button(
					Type("button"),
					g.Attr("@click", "closeModal()"),
					Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
					g.Text("Cancel"),
				),
				Button(
					Type("button"),
					g.Attr("@click", "await saveTemplate()"),
					g.Attr(":disabled", "saving || !isFormValid()"),
					Class("inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"),
					lucide.Save(Class("h-4 w-4")),
					Span(g.Attr("x-text", "saving ? 'Saving...' : 'Save Template'")),
				),
			),
		),
	)
}

// templateFormData returns the Alpine.js data for the template form.
func templateFormData(appID string, existingTemplate any) string {
	var templateJSON string

	if existingTemplate != nil {
		bytes, _ := json.Marshal(existingTemplate)
		templateJSON = string(bytes)
	} else {
		templateJSON = "null"
	}

	return fmt.Sprintf(`{
		appId: '%s',
		form: {
			name: '',
			templateKey: '',
			type: 'email',
			language: 'en',
			subject: '',
			body: '',
			variables: [],
			active: true
		},
		variablesInput: '',
		errors: {},
		saving: false,
		formError: null,
		
		init() {
			// Load existing template if editing
			const template = %s;
			if (template) {
				this.form.name = template.name || '';
				this.form.templateKey = template.templateKey || '';
				this.form.type = template.type || 'email';
				this.form.language = template.language || 'en';
				this.form.subject = template.subject || '';
				this.form.body = template.body || '';
				this.form.variables = template.variables || [];
				this.form.active = template.active !== undefined ? template.active : true;
				this.variablesInput = this.form.variables.join(', ');
			}
		},
		
		updateVariablesArray() {
			// Convert comma-separated string to array
			this.form.variables = this.variablesInput
				.split(',')
				.map(v => v.trim())
				.filter(v => v.length > 0);
		},
		
		validateField(field) {
			this.errors[field] = null;
			
			switch(field) {
				case 'name':
					if (!this.form.name || this.form.name.trim().length < 3) {
						this.errors.name = 'Name must be at least 3 characters';
					} else if (this.form.name.length > 100) {
						this.errors.name = 'Name must be less than 100 characters';
					}
					break;
					
				case 'templateKey':
					if (!this.form.templateKey || this.form.templateKey.trim().length === 0) {
						this.errors.templateKey = 'Template key is required';
					} else if (!/^[a-z]+\.[a-z_]+$/.test(this.form.templateKey)) {
						this.errors.templateKey = 'Format must be category.action (e.g., auth.welcome)';
					}
					break;
					
				case 'type':
					if (!this.form.type || !['email', 'sms', 'push'].includes(this.form.type)) {
						this.errors.type = 'Valid type is required';
					}
					break;
					
				case 'subject':
					if (this.form.type === 'email' && (!this.form.subject || this.form.subject.trim().length === 0)) {
						this.errors.subject = 'Subject is required for email templates';
					}
					break;
					
				case 'body':
					if (!this.form.body || this.form.body.trim().length < 10) {
						this.errors.body = 'Body must be at least 10 characters';
					}
					break;
			}
		},
		
		isFormValid() {
			// Check required fields
			if (!this.form.name || this.form.name.trim().length < 3) return false;
			if (!this.form.templateKey || this.form.templateKey.trim().length === 0) return false;
			if (!this.form.type) return false;
			if (this.form.type === 'email' && (!this.form.subject || this.form.subject.trim().length === 0)) return false;
			if (!this.form.body || this.form.body.trim().length < 10) return false;
			
			// Check for any existing errors
			return Object.values(this.errors).every(err => !err);
		},
		
		validateAllFields() {
			this.validateField('name');
			this.validateField('templateKey');
			this.validateField('type');
			if (this.form.type === 'email') {
				this.validateField('subject');
			}
			this.validateField('body');
			return this.isFormValid();
		},
		
		closeModal() {
			this.showCreateModal = false;
			this.showEditModal = false;
			this.formError = null;
			this.errors = {};
			this.form = {
				name: '',
				templateKey: '',
				type: 'email',
				language: 'en',
				subject: '',
				body: '',
				variables: [],
				active: true
			};
			this.variablesInput = '';
		},
		
		async saveTemplate() {
			// Validate all fields
			if (!this.validateAllFields()) {
				this.formError = 'Please fix the errors before saving';
				return;
			}
			
			this.saving = true;
			this.formError = null;
			
			try {
				const bridgeFunc = this.showCreateModal 
					? 'notification.createTemplate' 
					: 'notification.updateTemplate';
				
				const input = {
					appId: this.appId,
					name: this.form.name,
					templateKey: this.form.templateKey,
					type: this.form.type,
					language: this.form.language,
					subject: this.form.subject,
					body: this.form.body,
					variables: this.form.variables,
					active: this.form.active
				};
				
				// Add templateId for update
				if (this.showEditModal && this.currentTemplate) {
					input.templateId = this.currentTemplate.id;
				}
				
				const result = await $bridge.call(bridgeFunc, input);
				this.successMessage = result.message || 'Template saved successfully';
				await this.loadTemplates();
				this.closeModal();
				
				// Clear success message after 3 seconds
				setTimeout(() => this.successMessage = null, 3000);
			} catch (err) {
				console.error('Failed to save template:', err);
				this.formError = err.message || 'Failed to save template';
			} finally {
				this.saving = false;
			}
		}
	}`, appID, templateJSON)
}
