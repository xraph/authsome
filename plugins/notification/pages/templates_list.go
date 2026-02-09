package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TemplatesListPage renders the templates list page.
func TemplatesListPage(currentApp *app.App, basePath string) g.Node {
	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-2xl font-bold"), g.Text("Templates")),
					P(Class("mt-1 text-sm text-muted-foreground"),
						g.Text("Manage email and SMS notification templates")),
				),
				Button(
					Type("button"),
					g.Attr("@click", "showCreateModal = true"),
					Class("inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Create Template"),
				),
			),

			// Main content with Alpine.js
			Div(
				g.Attr("x-data", templatesListData(currentApp.ID.String())),
				g.Attr("x-init", "await loadTemplates()"),
				Class("space-y-6"),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error message
				ErrorMessage("error && !loading"),

				// Success message
				SuccessMessage("successMessage && !loading"),

				// Empty state
				Div(
					g.Attr("x-show", "!loading && !error && templates.length === 0"),
					Class("flex flex-col items-center justify-center py-12 px-4 text-center"),
					card.Card(
						card.Content(
							Class("p-12"),
							lucide.Mail(Class("mx-auto h-12 w-12 text-muted-foreground mb-4")),
							H3(Class("text-lg font-semibold mb-2"), g.Text("No templates yet")),
							P(Class("text-sm text-muted-foreground mb-6"),
								g.Text("Create your first notification template to get started")),
							Button(
								Type("button"),
								g.Attr("@click", "showCreateModal = true"),
								Class("inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"),
								lucide.Plus(Class("h-4 w-4")),
								g.Text("Create Template"),
							),
						),
					),
				),

				// Templates grid
				Div(
					g.Attr("x-show", "!loading && !error && templates.length > 0"),
					Class("grid gap-4 md:grid-cols-2 lg:grid-cols-3"),

					// Template cards
					Template(
						g.Attr("x-for", "template in templates"),
						g.Attr(":key", "template.id"),
						card.Card(
							Class("hover:shadow-md transition-shadow"),
							card.Content(
								Class("p-6"),
								// Template header with icon
								Div(
									Class("flex items-start gap-4 mb-4"),
									Div(
										Class("flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10"),
										lucide.Mail(Class("h-5 w-5 text-primary")),
									),
									Div(
										Class("flex-1 min-w-0"),
										Div(
											Class("flex items-center gap-2 flex-wrap"),
											H3(
												Class("font-semibold text-base truncate"),
												g.Attr("x-text", "template.name"),
											),
											Span(
												g.Attr("x-show", "template.isDefault"),
												Class("inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"),
												g.Text("Default"),
											),
											Span(
												g.Attr("x-show", "!template.active"),
												Class("inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-400"),
												g.Text("Inactive"),
											),
										),
										P(
											Class("text-xs text-muted-foreground mt-1 font-mono truncate"),
											g.Attr("x-text", "template.templateKey"),
										),
									),
								),

								// Template metadata
								Div(
									Class("flex items-center gap-3 text-xs text-muted-foreground mb-4 pb-4 border-b"),
									Div(
										Class("flex items-center gap-1"),
										lucide.FileText(Class("h-3.5 w-3.5")),
										Span(g.Attr("x-text", "template.type.charAt(0).toUpperCase() + template.type.slice(1)")),
									),
									Div(
										Class("flex items-center gap-1"),
										lucide.Globe(Class("h-3.5 w-3.5")),
										Span(g.Attr("x-text", "template.language.toUpperCase()")),
									),
									Div(
										g.Attr("x-show", "template.variables && template.variables.length > 0"),
										Class("flex items-center gap-1"),
										lucide.Code(Class("h-3.5 w-3.5")),
										Span(g.Attr("x-text", "template.variables.length + ' vars'")),
									),
								),
								// Actions
								Div(
									Class("flex items-center gap-2"),
									Button(
										Type("button"),
										g.Attr("@click", "viewTemplate(template)"),
										Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
										lucide.Eye(Class("h-4 w-4")),
									),
									Button(
										Type("button"),
										g.Attr("@click", "editTemplate(template)"),
										Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
										lucide.Pencil(Class("h-4 w-4")),
									),
									Button(
										Type("button"),
										g.Attr("@click", "openBuilder(template)"),
										g.Attr("x-show", "template.type === 'email'"),
										Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
										lucide.Palette(Class("h-4 w-4")),
										g.Text("Builder"),
									),
									Button(
										Type("button"),
										g.Attr("@click", "deleteTemplate(template)"),
										g.Attr("x-show", "!template.isDefault"),
										Class("inline-flex items-center gap-2 rounded-md border border-destructive bg-destructive px-3 py-2 text-sm text-destructive-foreground hover:bg-destructive/90"),
										lucide.Trash2(Class("h-4 w-4")),
									),
								),
							),
						),
					),
				),

				// Empty state
				Div(
					g.Attr("x-show", "!loading && templates.length === 0"),
					card.Card(
						card.Content(
							Class("p-12 text-center"),
							Div(
								Class("mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-muted"),
								lucide.FileText(Class("h-10 w-10 text-muted-foreground")),
							),
							H3(Class("mt-4 text-lg font-semibold"), g.Text("No templates yet")),
							P(Class("mt-2 text-sm text-muted-foreground"),
								g.Text("Create your first notification template to get started")),
							Div(
								Class("mt-6"),
								Button(
									Type("button"),
									g.Attr("@click", "showCreateModal = true"),
									Class("inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"),
									lucide.Plus(Class("h-4 w-4")),
									g.Text("Create Template"),
								),
							),
						),
					),
				),

				// Create/Edit Modal with full form
				TemplateFormModal(currentApp.ID.String()),
			),
		),
	)
}

func templatesListData(appID string) string {
	basePath := "/api/identity/ui"

	return fmt.Sprintf(`{
		templates: [],
		loading: false,
		error: null,
		successMessage: null,
		showCreateModal: false,
		showEditModal: false,
		currentTemplate: null,
		// Form state (shared with TemplateFormModal)
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

		async loadTemplates() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.listTemplates', {
					page: 1,
					limit: 50
				});
				this.templates = result.templates || [];
			} catch (err) {
				console.error('Failed to load templates:', err);
				this.error = err.message || 'Failed to load templates';
			} finally {
				this.loading = false;
			}
		},

		viewTemplate(template) {
			// Navigate to template detail/preview page
			window.location.href = '%s/app/%s/notifications/templates/' + template.id;
		},
		
		openBuilder(template) {
			// Navigate to email builder
			const url = template.id 
				? '%s/app/%s/notifications/builder/' + template.id
				: '%s/app/%s/notifications/builder';
			window.location.href = url;
		},

		editTemplate(template) {
			this.currentTemplate = template;
			// Pre-populate form with template data
			this.form.name = template.name || '';
			this.form.templateKey = template.templateKey || '';
			this.form.type = template.type || 'email';
			this.form.language = template.language || 'en';
			this.form.subject = template.subject || '';
			this.form.body = template.body || '';
			this.form.variables = template.variables || [];
			this.form.active = template.active !== undefined ? template.active : true;
			this.variablesInput = this.form.variables.join(', ');
			this.errors = {};
			this.formError = null;
			this.showEditModal = true;
		},

		async deleteTemplate(template) {
			if (!confirm('Are you sure you want to delete this template?')) {
				return;
			}

			try {
				await $bridge.call('notification.deleteTemplate', {
					templateId: template.id
				});
				this.successMessage = 'Template deleted successfully';
				await this.loadTemplates();
				setTimeout(() => this.successMessage = null, 3000);
			} catch (err) {
				console.error('Failed to delete template:', err);
				this.error = err.message || 'Failed to delete template';
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
	}`, basePath, appID, basePath, appID, basePath, appID)
}
