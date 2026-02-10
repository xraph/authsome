package pages

import (
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/secrets/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// CreateSecretPage renders the create secret form.
func CreateSecretPage(
	currentApp *app.App,
	basePath string,
	prefill *core.CreateSecretRequest,
	errorMsg string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-2"),

		// Header
		Div(
			Class("space-y-4"),
			// Breadcrumb
			Nav(
				Class("flex items-center gap-2 text-sm"),
				A(
					Href(appBase+"/secrets"),
					Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
					g.Text("Secrets"),
				),
				lucide.ChevronRight(Class("size-4 text-slate-400")),
				Span(
					Class("text-slate-900 dark:text-white font-medium"),
					g.Text("Create Secret"),
				),
			),

			Div(
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Create New Secret"),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Store a new encrypted secret for your application"),
				),
			),
		),

		// Error message
		g.If(errorMsg != "", func() g.Node {
			return Div(
				Class("rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4"),
				Div(
					Class("flex items-start gap-3"),
					lucide.CircleAlert(Class("size-5 text-red-600 dark:text-red-400 flex-shrink-0")),
					Div(
						H3(Class("text-sm font-medium text-red-800 dark:text-red-200"), g.Text("Error")),
						P(Class("mt-1 text-sm text-red-700 dark:text-red-300"), g.Text(errorMsg)),
					),
				),
			)
		}()),

		// Form
		secretForm(appBase, nil, prefill, false),
	)
}

// EditSecretPage renders the edit secret form.
func EditSecretPage(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	errorMsg string,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-2"),

		// Header
		Div(
			Class("space-y-4"),
			// Breadcrumb
			Nav(
				Class("flex items-center gap-2 text-sm"),
				A(
					Href(appBase+"/secrets"),
					Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
					g.Text("Secrets"),
				),
				lucide.ChevronRight(Class("size-4 text-slate-400")),
				A(
					Href(appBase+"/secrets/"+secret.ID),
					Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
					g.Text(secret.Path),
				),
				lucide.ChevronRight(Class("size-4 text-slate-400")),
				Span(
					Class("text-slate-900 dark:text-white font-medium"),
					g.Text("Edit"),
				),
			),

			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-lg bg-violet-100 dark:bg-violet-900/30 p-2"),
					lucide.Key(Class("size-5 text-violet-600 dark:text-violet-400")),
				),
				Div(
					H1(
						Class("text-2xl font-bold text-slate-900 dark:text-white"),
						g.Text("Edit Secret"),
					),
					P(
						Class("text-sm text-slate-600 dark:text-gray-400 font-mono"),
						g.Text(secret.Path),
					),
				),
			),
		),

		// Error message
		g.If(errorMsg != "", func() g.Node {
			return Div(
				Class("rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4"),
				Div(
					Class("flex items-start gap-3"),
					lucide.CircleAlert(Class("size-5 text-red-600 dark:text-red-400 flex-shrink-0")),
					Div(
						H3(Class("text-sm font-medium text-red-800 dark:text-red-200"), g.Text("Error")),
						P(Class("mt-1 text-sm text-red-700 dark:text-red-300"), g.Text(errorMsg)),
					),
				),
			)
		}()),

		// Form
		secretForm(appBase, secret, nil, true),
	)
}

// secretForm renders the actual form for creating/editing secrets.
func secretForm(appBase string, secret *core.SecretDTO, prefill *core.CreateSecretRequest, isEdit bool) g.Node {
	// Determine initial values
	var (
		path, description, schema, valueType string
		tags                                 []string
	)

	hasExpiry := false

	if isEdit && secret != nil {
		path = secret.Path
		description = secret.Description
		// Note: schema is not returned in SecretDTO, only hasSchema flag
		valueType = secret.ValueType
		tags = secret.Tags
		hasExpiry = secret.HasExpiry
	} else if prefill != nil {
		path = prefill.Path
		description = prefill.Description
		schema = prefill.Schema
		valueType = prefill.ValueType
		tags = prefill.Tags
	}

	if valueType == "" {
		valueType = "plain"
	}

	action := appBase + "/secrets/create"
	if isEdit {
		action = appBase + "/secrets/" + secret.ID + "/update"
	}

	return FormEl(
		Method("POST"),
		Action(action),
		Class("space-y-2"),

		// Hidden method for edit
		g.If(isEdit, func() g.Node {
			return Input(Type("hidden"), Name("_method"), Value("PUT"))
		}()),

		// Main card
		Div(
			Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),

			// Path section
			Div(
				Class("p-6 border-b border-slate-200 dark:border-gray-800"),
				formField(
					"path",
					"Secret Path",
					"Use forward slashes to create hierarchy (e.g., database/postgres/password)",
					Input(
						Type("text"),
						Name("path"),
						ID("path"),
						Value(path),
						Placeholder("database/postgres/password"),
						Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"),
						Required(),
						g.If(isEdit, Disabled()),
					),
					true,
				),
			),

			// Value type and value section
			Div(
				Class("p-6 border-b border-slate-200 dark:border-gray-800 space-y-2"),

				// Value type selector
				Div(
					Label(
						For("valueType"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
						g.Text("Value Type"),
					),
					Div(
						Class("grid grid-cols-2 sm:grid-cols-4 gap-3"),
						valueTypeOption("plain", "Plain Text", "Simple string value", valueType == "plain", lucide.Type(Class("size-5"))),
						valueTypeOption("json", "JSON", "Structured JSON object", valueType == "json", lucide.Braces(Class("size-5"))),
						valueTypeOption("yaml", "YAML", "YAML configuration", valueType == "yaml", lucide.FileCode(Class("size-5"))),
						valueTypeOption("binary", "Binary", "Base64 encoded data", valueType == "binary", lucide.Binary(Class("size-5"))),
					),
				),

				// Value input
				Div(
					Label(
						For("value"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
						g.Text("Value"),
						Span(Class("text-red-500 ml-1"), g.Text("*")),
					),
					Div(
						Class("relative"),
						Textarea(
							Name("value"),
							ID("value"),
							Rows("8"),
							Placeholder(getValuePlaceholder(valueType)),
							Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-slate-900 dark:bg-gray-950 text-green-400 font-mono focus:outline-none focus:ring-2 focus:ring-violet-500"),
							Required(),
						),
						// Format hint
						Div(
							ID("value-hint"),
							Class("absolute bottom-2 right-2 text-xs text-slate-500"),
							g.Text(getFormatHint(valueType)),
						),
					),
					g.If(!isEdit, func() g.Node {
						return P(
							Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("The value will be encrypted before storage"),
						)
					}()),
					g.If(isEdit, func() g.Node {
						return P(
							Class("mt-1 text-xs text-amber-600 dark:text-amber-400"),
							g.Text("Leave empty to keep existing value, or enter a new value to update"),
						)
					}()),
				),
			),

			// Advanced options
			Details(
				Class("border-b border-slate-200 dark:border-gray-800"),
				Summary(
					Class("p-6 cursor-pointer text-sm font-medium text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
					g.Text("Advanced Options"),
				),
				Div(
					Class("px-6 pb-6 space-y-6"),

					// JSON Schema
					formField(
						"schema",
						"JSON Schema (optional)",
						"Provide a JSON Schema to validate the value structure",
						Textarea(
							Name("schema"),
							ID("schema"),
							Rows("4"),
							Placeholder(`{"type": "object", "required": ["host", "port"]}`),
							Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-violet-500"),
							g.Text(schema),
						),
						false,
					),

					// Description
					formField(
						"description",
						"Description (optional)",
						"A brief description of what this secret is for",
						Textarea(
							Name("description"),
							ID("description"),
							Rows("2"),
							Placeholder("PostgreSQL production database password"),
							Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							g.Text(description),
						),
						false,
					),

					// Tags
					formField(
						"tags",
						"Tags (optional)",
						"Comma-separated tags for organization",
						Input(
							Type("text"),
							Name("tags"),
							ID("tags"),
							Value(joinTags(tags)),
							Placeholder("production, database, critical"),
							Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
						),
						false,
					),

					// Expiration
					Div(
						Label(
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
							g.Text("Expiration (optional)"),
						),
						Div(
							Class("flex items-center gap-4"),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("radio"),
									Name("expiry_type"),
									Value("never"),
									Class("text-violet-600 focus:ring-violet-500"),
									g.If(!hasExpiry, Checked()),
								),
								Span(Class("text-sm text-slate-700 dark:text-gray-300"), g.Text("Never")),
							),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("radio"),
									Name("expiry_type"),
									Value("date"),
									Class("text-violet-600 focus:ring-violet-500"),
									g.If(hasExpiry, Checked()),
								),
								Span(Class("text-sm text-slate-700 dark:text-gray-300"), g.Text("Custom date")),
							),
						),
						Div(
							ID("expiry-date-input"),
							Class("mt-3"),
							g.If(!hasExpiry, g.Attr("style", "display: none")),
							Input(
								Type("date"),
								Name("expires_at"),
								ID("expires_at"),
								Class("block w-full max-w-xs px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
						),
					),

					// Change reason (edit only)
					g.If(isEdit, func() g.Node {
						return formField(
							"change_reason",
							"Change Reason",
							"Document why this change is being made",
							Input(
								Type("text"),
								Name("change_reason"),
								ID("change_reason"),
								Placeholder("Updated credentials after rotation"),
								Class("block w-full px-4 py-2.5 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
							),
							false,
						)
					}()),
				),
			),
		),

		// Form actions
		Div(
			Class("flex items-center justify-end gap-3"),
			A(
				Href(appBase+"/secrets"),
				Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
				g.Text("Cancel"),
			),
			Button(
				Type("submit"),
				Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
				g.If(!isEdit, lucide.Plus(Class("size-4"))),
				g.If(isEdit, lucide.Save(Class("size-4"))),
				g.If(!isEdit, g.Text("Create Secret")),
				g.If(isEdit, g.Text("Save Changes")),
			),
		),

		// JavaScript for form interactions
		formScript(),
	)
}

func formField(id, label, hint string, input g.Node, required bool) g.Node {
	return Div(
		Label(
			For(id),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
			g.Text(label),
			g.If(required, Span(Class("text-red-500 ml-1"), g.Text("*"))),
		),
		input,
		g.If(hint != "", func() g.Node {
			return P(
				Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
				g.Text(hint),
			)
		}()),
	)
}

func valueTypeOption(value, label, description string, checked bool, icon g.Node) g.Node {
	baseClass := "relative flex flex-col items-center p-4 border-2 rounded-lg cursor-pointer transition-colors"
	checkedClass := " border-violet-500 bg-violet-50 dark:bg-violet-900/20"
	uncheckedClass := " border-slate-200 dark:border-gray-700 hover:border-violet-300 dark:hover:border-violet-700"

	finalClass := baseClass
	if checked {
		finalClass += checkedClass
	} else {
		finalClass += uncheckedClass
	}

	return Label(
		Class(finalClass),
		Input(
			Type("radio"),
			Name("valueType"),
			Value(value),
			Class("sr-only"),
			g.If(checked, Checked()),
			g.Attr("onchange", "updateValueType(this.value)"),
		),
		Div(
			Class("mb-2 text-violet-600 dark:text-violet-400"),
			icon,
		),
		Span(
			Class("text-sm font-medium text-slate-900 dark:text-white"),
			g.Text(label),
		),
		Span(
			Class("text-xs text-slate-500 dark:text-gray-400 text-center mt-1"),
			g.Text(description),
		),
		g.If(checked, func() g.Node {
			return Div(
				Class("absolute top-2 right-2"),
				lucide.CircleCheck(Class("size-4 text-violet-600")),
			)
		}()),
	)
}

func getValuePlaceholder(valueType string) string {
	switch valueType {
	case "json":
		return `{
  "host": "localhost",
  "port": 5432,
  "database": "myapp"
}`
	case "yaml":
		return `host: localhost
port: 5432
database: myapp`
	case "binary":
		return "SGVsbG8gV29ybGQh (Base64 encoded)"
	default:
		return "Enter your secret value..."
	}
}

func getFormatHint(valueType string) string {
	switch valueType {
	case "json":
		return "JSON format"
	case "yaml":
		return "YAML format"
	case "binary":
		return "Base64 encoded"
	default:
		return "Plain text"
	}
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	result := ""

	var resultSb519 strings.Builder

	for i, tag := range tags {
		if i > 0 {
			resultSb519.WriteString(", ")
		}

		resultSb519.WriteString(tag)
	}

	result += resultSb519.String()

	return result
}

func formScript() g.Node {
	return Script(g.Raw(`
		function updateValueType(type) {
			const valueInput = document.getElementById('value');
			const hint = document.getElementById('value-hint');
			
			const placeholders = {
				'plain': 'Enter your secret value...',
				'json': '{\n  "host": "localhost",\n  "port": 5432\n}',
				'yaml': 'host: localhost\nport: 5432',
				'binary': 'SGVsbG8gV29ybGQh (Base64 encoded)'
			};
			
			const hints = {
				'plain': 'Plain text',
				'json': 'JSON format',
				'yaml': 'YAML format',
				'binary': 'Base64 encoded'
			};
			
			valueInput.placeholder = placeholders[type] || placeholders['plain'];
			hint.textContent = hints[type] || hints['plain'];
			
			// Update radio button styles
			document.querySelectorAll('input[name="valueType"]').forEach(radio => {
				const label = radio.closest('label');
				if (radio.checked) {
					label.classList.remove('border-slate-200', 'dark:border-gray-700');
					label.classList.add('border-violet-500', 'bg-violet-50', 'dark:bg-violet-900/20');
				} else {
					label.classList.add('border-slate-200', 'dark:border-gray-700');
					label.classList.remove('border-violet-500', 'bg-violet-50', 'dark:bg-violet-900/20');
				}
			});
		}
		
		// Expiry type toggle
		document.querySelectorAll('input[name="expiry_type"]').forEach(radio => {
			radio.addEventListener('change', function() {
				const dateInput = document.getElementById('expiry-date-input');
				if (this.value === 'date') {
					dateInput.style.display = 'block';
				} else {
					dateInput.style.display = 'none';
				}
			});
		});
		
		// Form validation
		document.querySelector('form').addEventListener('submit', function(e) {
			const path = document.getElementById('path').value;
			const value = document.getElementById('value').value;
			
			// Validate path format
			const pathRegex = /^[a-zA-Z0-9][a-zA-Z0-9_\-\/]*[a-zA-Z0-9]$/;
			if (!pathRegex.test(path) && path.length > 1) {
				e.preventDefault();
				alert('Invalid path format. Use alphanumeric characters, underscores, hyphens, and forward slashes.');
				return;
			}
			
			// Validate JSON if type is json
			const valueType = document.querySelector('input[name="valueType"]:checked')?.value;
			if (valueType === 'json' && value) {
				try {
					JSON.parse(value);
				} catch (err) {
					e.preventDefault();
					alert('Invalid JSON format: ' + err.message);
					return;
				}
			}
		});
	`))
}

// SecretFormValidation renders client-side validation hints.
func SecretFormValidation() g.Node {
	return Div(
		Class("bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 p-4 mb-6"),
		Div(
			Class("flex items-start gap-3"),
			lucide.Info(Class("size-5 text-blue-600 dark:text-blue-400 flex-shrink-0")),
			Div(
				H3(Class("text-sm font-medium text-blue-800 dark:text-blue-200"), g.Text("Path Format")),
				Ul(
					Class("mt-2 text-sm text-blue-700 dark:text-blue-300 list-disc list-inside space-y-1"),
					Li(g.Text("Use forward slashes (/) to create hierarchy")),
					Li(g.Text("Alphanumeric characters, underscores, and hyphens allowed")),
					Li(g.Text("Must start and end with alphanumeric character")),
					Li(g.Text("Examples: database/password, api/stripe/key")),
				),
			),
		),
	)
}

// SecretImportForm renders a bulk import form.
func SecretImportForm(appBase string) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 p-6"),
		H3(
			Class("text-lg font-medium text-slate-900 dark:text-white mb-4"),
			g.Text("Bulk Import"),
		),
		P(
			Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
			g.Text("Import multiple secrets from a JSON or YAML file"),
		),
		FormEl(
			Method("POST"),
			Action(appBase+"/secrets/import"),
			g.Attr("enctype", "multipart/form-data"),
			Div(
				Class("border-2 border-dashed border-slate-300 dark:border-gray-700 rounded-lg p-8 text-center"),
				lucide.Upload(Class("size-8 text-slate-400 mx-auto mb-3")),
				P(
					Class("text-sm text-slate-600 dark:text-gray-400 mb-2"),
					g.Text("Drag and drop a file, or "),
					Label(
						For("import-file"),
						Class("text-violet-600 hover:text-violet-700 cursor-pointer"),
						g.Text("browse"),
					),
				),
				Input(
					Type("file"),
					Name("file"),
					ID("import-file"),
					Accept(".json,.yaml,.yml"),
					Class("hidden"),
				),
				P(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text("Supports JSON and YAML formats"),
				),
			),
			Div(
				Class("mt-4 flex justify-end"),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
					lucide.Upload(Class("size-4")),
					g.Text("Import"),
				),
			),
		),
	)
}

// Format template for JSON secrets.
const jsonTemplate = `{
  "secrets": [
    {
      "path": "database/postgres/password",
      "value": "your-password",
      "valueType": "plain",
      "description": "PostgreSQL password",
      "tags": ["database", "production"]
    }
  ]
}`

// Format template for YAML secrets.
const yamlTemplate = `secrets:
  - path: database/postgres/password
    value: your-password
    valueType: plain
    description: PostgreSQL password
    tags:
      - database
      - production`

// ImportTemplateDownload renders download links for import templates.
func ImportTemplateDownload() g.Node {
	return Div(
		Class("flex items-center gap-4 mt-4"),
		A(
			Href("data:application/json,"+jsonTemplate),
			Download("secrets-template.json"),
			Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
			g.Text("Download JSON template"),
		),
		A(
			Href("data:text/yaml,"+yamlTemplate),
			Download("secrets-template.yaml"),
			Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
			g.Text("Download YAML template"),
		),
	)
}
