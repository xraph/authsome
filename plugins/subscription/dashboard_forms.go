package subscription

import (
	"encoding/json"
	"fmt"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// ServePlanCreatePage renders the plan creation form.
func (e *DashboardExtension) ServePlanCreatePage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Get all features for this app
	features, _, _ := e.plugin.featureSvc.List(reqCtx, currentApp.ID, "", false, 1, 1000)

	content := Div(
		Class("space-y-2"),

		// Back button and header
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Plans"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Create Plan")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Create a new subscription plan for your customers")),

		// Form
		Form(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/create"),
			ID("plan-form"),

			// Basic Info Card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-6"),

				// Section header
				Div(
					Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
					H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Basic Information")),
					P(Class("text-sm text-slate-500"), g.Text("General plan details and pricing")),
				),

				// Plan name and slug row
				Div(
					Class("grid gap-4 md:grid-cols-2"),
					// Plan name
					Div(
						Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Plan Name")),
						Input(
							Type("text"), Name("name"), ID("name"), Required(),
							g.Attr("oninput", "generateSlug(this.value)"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("Pro Plan"),
						),
					),
					// Slug
					Div(
						Label(For("slug"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Slug")),
						Div(
							Class("mt-1 flex rounded-md shadow-sm"),
							Span(
								Class("inline-flex items-center rounded-l-md border border-r-0 border-slate-300 bg-slate-50 px-3 text-slate-500 text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-400"),
								g.Text("/plans/"),
							),
							Input(
								Type("text"), Name("slug"), ID("slug"), Required(),
								g.Attr("pattern", "[a-z0-9-]+"),
								Class("block w-full rounded-none rounded-r-md border-slate-300 focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								Placeholder("pro-plan"),
							),
						),
						P(Class("mt-1 text-xs text-slate-500"), g.Text("URL-friendly identifier (auto-generated from name)")),
					),
				),

				// Description
				Div(
					Label(For("description"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Description")),
					Textarea(
						Name("description"), ID("description"), Rows("3"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Everything you need to grow your business"),
					),
				),

				// Pricing row
				Div(
					Class("grid gap-4 md:grid-cols-3"),
					// Price
					Div(
						Label(For("price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Base Price (in cents)")),
						Input(
							Type("number"), Name("price"), ID("price"), Required(),
							Min("0"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("2900"),
						),
						P(Class("mt-1 text-xs text-slate-500"), g.Text("e.g., 2900 = $29.00")),
					),
					// Currency
					Div(
						Label(For("currency"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Currency")),
						Select(
							Name("currency"), ID("currency"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("USD"), g.Attr("selected", ""), g.Text("USD - US Dollar")),
							Option(Value("EUR"), g.Text("EUR - Euro")),
							Option(Value("GBP"), g.Text("GBP - British Pound")),
							Option(Value("CAD"), g.Text("CAD - Canadian Dollar")),
							Option(Value("AUD"), g.Text("AUD - Australian Dollar")),
						),
					),
					// Billing Interval
					Div(
						Label(For("billing_interval"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Interval")),
						Select(
							Name("billing_interval"), ID("billing_interval"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("monthly"), g.Attr("selected", ""), g.Text("Monthly")),
							Option(Value("yearly"), g.Text("Yearly")),
							Option(Value("one_time"), g.Text("One-time")),
						),
					),
				),

				// Billing pattern
				Div(
					Label(For("billing_pattern"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Pattern")),
					Select(
						Name("billing_pattern"), ID("billing_pattern"),
						g.Attr("onchange", "toggleBillingPatternFields()"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("flat"), g.Attr("selected", ""), g.Text("Flat Rate - Fixed price per billing period")),
						Option(Value("per_seat"), g.Text("Per Seat - Price per team member")),
						Option(Value("tiered"), g.Text("Tiered - Different prices for usage tiers")),
						Option(Value("usage"), g.Text("Usage-based - Pay for what you use")),
						Option(Value("hybrid"), g.Text("Hybrid - Base price + usage")),
					),
					P(Class("mt-1 text-sm text-slate-500"), g.Text("Choose how customers are billed")),
				),

				// Trial days
				Div(
					Label(For("trial_days"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Trial Days")),
					Input(
						Type("number"), Name("trial_days"), ID("trial_days"),
						Min("0"), Max("90"), Value("14"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
					P(Class("mt-1 text-sm text-slate-500"), g.Text("Number of free trial days (0 for no trial)")),
				),

				// Checkboxes row
				Div(
					Class("flex flex-wrap gap-6"),
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_active"), ID("is_active"), Value("true"), Checked(),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_active"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Active")),
					),
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_public"), ID("is_public"), Value("true"), Checked(),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_public"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Public")),
					),
				),
			),

			// Billing Pattern Specific Fields
			e.renderBillingPatternFields(nil),

			// Features Card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-2"),

				// Section header
				Div(
					Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
					Div(
						Class("flex items-center justify-between"),
						Div(
							H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Plan Features")),
							P(Class("text-sm text-slate-500"), g.Text("Select features included in this plan")),
						),
						g.If(len(features) == 0,
							A(
								Href(basePath+"/app/"+currentApp.ID.String()+"/billing/features/create"),
								Class("text-sm text-violet-600 hover:text-violet-700"),
								g.Text("Create Features"),
							),
						),
					),
				),

				// Features list
				g.If(len(features) == 0,
					Div(
						Class("text-center py-8"),
						lucide.Package(Class("mx-auto size-12 text-slate-300")),
						P(Class("mt-4 text-slate-500"), g.Text("No features defined yet")),
						P(Class("mt-1 text-sm text-slate-400"), g.Text("Create features first to link them to this plan")),
					),
				),
				g.If(len(features) > 0,
					e.renderFeatureSelectionList(features, nil),
				),
			),

			// Submit buttons
			Div(
				Class("flex justify-end gap-3"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans"),
					Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Text("Create Plan"),
				),
			),
		),

		// JavaScript for toggling billing pattern fields and slug generation
		Script(g.Raw(`
			function toggleBillingPatternFields() {
				const pattern = document.getElementById('billing_pattern').value;
				const fields = ['per_seat_fields', 'tiered_fields', 'usage_fields', 'hybrid_fields'];
				
				fields.forEach(f => {
					const el = document.getElementById(f);
					if (el) el.style.display = 'none';
				});
				
				const activeField = document.getElementById(pattern + '_fields');
				if (activeField) activeField.style.display = 'block';
			}
			
			function generateSlug(name) {
				const slugField = document.getElementById('slug');
				if (slugField && !slugField.dataset.manual) {
					const slug = name
						.toLowerCase()
						.trim()
						.replace(/[^\w\s-]/g, '')  // Remove special characters
						.replace(/\s+/g, '-')       // Replace spaces with hyphens
						.replace(/-+/g, '-')        // Replace multiple hyphens with single
						.replace(/^-|-$/g, '');     // Remove leading/trailing hyphens
					slugField.value = slug;
				}
			}
			
			// Mark slug as manually edited if user modifies it
			document.addEventListener('DOMContentLoaded', function() {
				toggleBillingPatternFields();
				
				const slugField = document.getElementById('slug');
				if (slugField) {
					slugField.addEventListener('input', function() {
						this.dataset.manual = 'true';
					});
				}
			});
		`)),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServePlanEditPage renders the plan edit form.
func (e *DashboardExtension) ServePlanEditPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	planID, err := xid.FromString(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid plan ID")
	}

	plan, err := e.plugin.planSvc.GetByID(reqCtx, planID)
	if err != nil {
		return nil, errs.NotFound("Plan not found")
	}

	// Get all features for this app
	features, _, _ := e.plugin.featureSvc.List(reqCtx, currentApp.ID, "", false, 1, 1000)

	// Get features linked to this plan
	linkedFeatures, _ := e.plugin.featureSvc.GetPlanFeatures(reqCtx, planID)

	// Create a map of linked feature IDs to their link configs
	linkedMap := make(map[string]*core.PlanFeatureLink)
	for _, link := range linkedFeatures {
		linkedMap[link.FeatureID.String()] = link
	}

	content := Div(
		Class("space-y-2"),

		// Back button and header
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Plan"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Edit Plan")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Update plan details and pricing")),

		// Form
		Form(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/update"),
			ID("plan-form"),
			Class("space-y-2"),

			// Basic Info Card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-6"),

				// Section header
				Div(
					Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
					H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Basic Information")),
				),

				// Plan name
				Div(
					Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Plan Name")),
					Input(
						Type("text"), Name("name"), ID("name"), Required(),
						Value(plan.Name),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),

				// Description
				Div(
					Label(For("description"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Description")),
					Textarea(
						Name("description"), ID("description"), Rows("3"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						g.Text(plan.Description),
					),
				),

				// Pricing row
				Div(
					Class("grid gap-4 md:grid-cols-3"),
					// Price
					Div(
						Label(For("price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Base Price (in cents)")),
						Input(
							Type("number"), Name("price"), ID("price"), Required(),
							Min("0"),
							Value(itoa(plan.BasePrice)),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),
					// Currency
					Div(
						Label(For("currency"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Currency")),
						Select(
							Name("currency"), ID("currency"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("USD"), g.If(plan.Currency == "USD", g.Attr("selected", "")), g.Text("USD")),
							Option(Value("EUR"), g.If(plan.Currency == "EUR", g.Attr("selected", "")), g.Text("EUR")),
							Option(Value("GBP"), g.If(plan.Currency == "GBP", g.Attr("selected", "")), g.Text("GBP")),
							Option(Value("CAD"), g.If(plan.Currency == "CAD", g.Attr("selected", "")), g.Text("CAD")),
							Option(Value("AUD"), g.If(plan.Currency == "AUD", g.Attr("selected", "")), g.Text("AUD")),
						),
					),
					// Billing Interval
					Div(
						Label(For("billing_interval"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Interval")),
						Select(
							Name("billing_interval"), ID("billing_interval"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("monthly"), g.If(plan.BillingInterval == "monthly", g.Attr("selected", "")), g.Text("Monthly")),
							Option(Value("yearly"), g.If(plan.BillingInterval == "yearly", g.Attr("selected", "")), g.Text("Yearly")),
							Option(Value("one_time"), g.If(plan.BillingInterval == "one_time", g.Attr("selected", "")), g.Text("One-time")),
						),
					),
				),

				// Billing pattern (read-only after creation)
				Div(
					Label(For("billing_pattern"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Pattern")),
					Div(
						Class("mt-1 flex items-center gap-2"),
						Input(
							Type("hidden"), Name("billing_pattern"), Value(string(plan.BillingPattern)),
						),
						Span(
							Class("inline-flex items-center px-3 py-2 rounded-md bg-slate-100 text-slate-700 text-sm dark:bg-gray-700 dark:text-gray-300"),
							g.Text(billingPatternLabel(plan.BillingPattern)),
						),
						Span(
							Class("text-xs text-slate-500 dark:text-gray-400"),
							g.Text("(cannot be changed after creation)"),
						),
					),
				),

				// Trial days
				Div(
					Label(For("trial_days"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Trial Days")),
					Input(
						Type("number"), Name("trial_days"), ID("trial_days"),
						Min("0"), Max("90"), Value(itoa(int64(plan.TrialDays))),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),

				// Checkboxes
				Div(
					Class("flex flex-wrap gap-6"),
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_active"), ID("is_active"), Value("true"),
							g.If(plan.IsActive, Checked()),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_active"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Active")),
					),
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_public"), ID("is_public"), Value("true"),
							g.If(plan.IsPublic, Checked()),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_public"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Public")),
					),
				),
			),

			// Billing Pattern Specific Fields
			e.renderBillingPatternFields(plan),

			// Features Card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-6"),

				// Section header
				Div(
					Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
					Div(
						Class("flex items-center justify-between"),
						Div(
							H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Plan Features")),
							P(Class("text-sm text-slate-500"), g.Text("Configure features included in this plan")),
						),
						A(
							Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/features"),
							Class("text-sm text-violet-600 hover:text-violet-700"),
							g.Text("Advanced Feature Config →"),
						),
					),
				),

				// Features list
				g.If(len(features) == 0,
					Div(
						Class("text-center py-8"),
						lucide.Package(Class("mx-auto size-12 text-slate-300")),
						P(Class("mt-4 text-slate-500"), g.Text("No features defined yet")),
					),
				),
				g.If(len(features) > 0,
					e.renderFeatureSelectionList(features, linkedMap),
				),
			),

			// Submit buttons
			Div(
				Class("flex justify-end gap-3"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
					Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Text("Save Changes"),
				),
			),
		),

		// JavaScript for toggling billing pattern fields
		Script(g.Raw(`
			function toggleBillingPatternFields() {
				const pattern = document.getElementById('billing_pattern').value;
				const fields = ['per_seat_fields', 'tiered_fields', 'usage_fields', 'hybrid_fields'];
				
				fields.forEach(f => {
					const el = document.getElementById(f);
					if (el) el.style.display = 'none';
				});
				
				const activeField = document.getElementById(pattern + '_fields');
				if (activeField) activeField.style.display = 'block';
			}
			
			// Initialize on page load
			document.addEventListener('DOMContentLoaded', toggleBillingPatternFields);
		`)),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAddOnCreatePage renders the add-on creation form.
func (e *DashboardExtension) ServeAddOnCreatePage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	content := Div(
		Class("space-y-2"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Add-ons"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Create Add-on")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Create an additional feature or product")),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/app/"+currentApp.ID.String()+"/billing/addons/create"),
				Class("space-y-2"),

				// Name
				Div(
					Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Add-on Name")),
					Input(
						Type("text"), Name("name"), ID("name"), Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Extra Storage"),
					),
				),

				// Description
				Div(
					Label(For("description"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Description")),
					Textarea(
						Name("description"), ID("description"), Rows("3"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Additional storage capacity for your account"),
					),
				),

				// Pricing
				Div(
					Class("grid gap-4 md:grid-cols-3"),
					Div(
						Label(For("price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Price (in cents)")),
						Input(
							Type("number"), Name("price"), ID("price"), Required(),
							Min("0"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("500"),
						),
					),
					Div(
						Label(For("currency"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Currency")),
						Select(
							Name("currency"), ID("currency"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("USD"), g.Attr("selected", ""), g.Text("USD")),
							Option(Value("EUR"), g.Text("EUR")),
							Option(Value("GBP"), g.Text("GBP")),
						),
					),
					Div(
						Label(For("billing_type"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Type")),
						Select(
							Name("billing_type"), ID("billing_type"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("one_time"), g.Text("One-time")),
							Option(Value("recurring"), g.Attr("selected", ""), g.Text("Recurring")),
							Option(Value("usage"), g.Text("Usage-based")),
						),
					),
				),

				// Feature key (optional)
				Div(
					Label(For("feature_key"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Feature Key (Optional)")),
					Input(
						Type("text"), Name("feature_key"), ID("feature_key"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("extra_storage"),
					),
					P(Class("mt-1 text-sm text-slate-500"), g.Text("Used to identify this add-on in your code")),
				),

				// Active checkbox
				Div(
					Class("flex items-center"),
					Input(Type("checkbox"), Name("is_active"), ID("is_active"), Value("true"), Checked(),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
					Label(For("is_active"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Active - Add-on is available for purchase")),
				),

				// Submit
				Div(
					Class("flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-gray-800"),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/addons"),
						Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Create Add-on"),
					),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeCouponCreatePage renders the coupon creation form.
func (e *DashboardExtension) ServeCouponCreatePage(ctx *router.PageContext) (g.Node, error) {
	// basePath := e.baseUIPath
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	content := Div(
		Class("space-y-2"),

		// Back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Coupons"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Create Coupon")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Create a promotional discount code")),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons/create"),
				Class("space-y-2"),

				// Coupon code
				Div(
					Label(For("code"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Coupon Code")),
					Input(
						Type("text"), Name("code"), ID("code"), Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono uppercase"),
						Placeholder("SAVE20"),
					),
					P(Class("mt-1 text-sm text-slate-500"), g.Text("The code customers will enter to apply the discount")),
				),

				// Name
				Div(
					Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Display Name")),
					Input(
						Type("text"), Name("name"), ID("name"), Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("20% Off First Month"),
					),
				),

				// Discount type and value
				Div(
					Class("grid gap-4 md:grid-cols-2"),
					Div(
						Label(For("discount_type"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Discount Type")),
						Select(
							Name("discount_type"), ID("discount_type"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("percentage"), g.Attr("selected", ""), g.Text("Percentage Off")),
							Option(Value("fixed"), g.Text("Fixed Amount Off")),
						),
					),
					Div(
						Label(For("discount_value"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Discount Value")),
						Input(
							Type("number"), Name("discount_value"), ID("discount_value"), Required(),
							Min("0"), Step("0.01"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Placeholder("20"),
						),
						P(Class("mt-1 text-sm text-slate-500"), g.Text("For percentage: 20 = 20%. For fixed: value in cents.")),
					),
				),

				// Usage limits
				Div(
					Class("grid gap-4 md:grid-cols-2"),
					Div(
						Label(For("max_redemptions"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Max Total Redemptions")),
						Input(
							Type("number"), Name("max_redemptions"), ID("max_redemptions"),
							Min("0"), Value("0"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500"), g.Text("0 = unlimited")),
					),
					Div(
						Label(For("max_per_customer"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Max Per Customer")),
						Input(
							Type("number"), Name("max_per_customer"), ID("max_per_customer"),
							Min("0"), Value("1"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500"), g.Text("0 = unlimited per customer")),
					),
				),

				// Expiration
				Div(
					Label(For("expires_at"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Expiration Date (Optional)")),
					Input(
						Type("date"), Name("expires_at"), ID("expires_at"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
					P(Class("mt-1 text-sm text-slate-500"), g.Text("Leave empty for no expiration")),
				),

				// Active checkbox
				Div(
					Class("flex items-center"),
					Input(Type("checkbox"), Name("is_active"), ID("is_active"), Value("true"), Checked(),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
					Label(For("is_active"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Active - Coupon can be redeemed")),
				),

				// Submit
				Div(
					Class("flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-gray-800"),
					A(
						Href(basePath+"/app/"+currentApp.ID.String()+"/billing/coupons"),
						Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Create Coupon"),
					),
				),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderBillingPatternFields renders fields specific to each billing pattern.
func (e *DashboardExtension) renderBillingPatternFields(plan *core.Plan) g.Node {
	// Default values
	minSeats := 1
	maxSeats := 0 // 0 = unlimited
	seatPrice := int64(0)
	unitPrice := int64(0)
	unitName := "unit"
	hybridBasePrice := int64(0)

	// Extract values from existing plan
	if plan != nil {
		if plan.Metadata != nil {
			if v, ok := plan.Metadata["min_seats"].(float64); ok {
				minSeats = int(v)
			}

			if v, ok := plan.Metadata["max_seats"].(float64); ok {
				maxSeats = int(v)
			}

			if v, ok := plan.Metadata["seat_price"].(float64); ok {
				seatPrice = int64(v)
			}

			if v, ok := plan.Metadata["unit_price"].(float64); ok {
				unitPrice = int64(v)
			}

			if v, ok := plan.Metadata["unit_name"].(string); ok {
				unitName = v
			}

			if v, ok := plan.Metadata["hybrid_base_price"].(float64); ok {
				hybridBasePrice = int64(v)
			}
		}
	}

	// Determine which pattern is selected for initial display
	selectedPattern := "flat"
	if plan != nil {
		selectedPattern = string(plan.BillingPattern)
	}

	return g.Group([]g.Node{
		// Per Seat Fields
		Div(
			ID("per_seat_fields"),
			StyleAttr("display: "+boolToDisplay(selectedPattern == "per_seat")),
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-2"),

			Div(
				Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Per Seat Configuration")),
				P(Class("text-sm text-slate-500"), g.Text("Configure seat-based pricing")),
			),

			Div(
				Class("grid gap-4 md:grid-cols-3"),
				Div(
					Label(For("seat_price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Price per Seat (cents)")),
					Input(
						Type("number"), Name("seat_price"), ID("seat_price"),
						Min("0"), Value(itoa(seatPrice)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("500"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Price for each additional seat")),
				),
				Div(
					Label(For("min_seats"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Minimum Seats")),
					Input(
						Type("number"), Name("min_seats"), ID("min_seats"),
						Min("1"), Value(strconv.Itoa(minSeats)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Minimum number of seats required")),
				),
				Div(
					Label(For("max_seats"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Maximum Seats")),
					Input(
						Type("number"), Name("max_seats"), ID("max_seats"),
						Min("0"), Value(strconv.Itoa(maxSeats)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("0 = unlimited")),
				),
			),
		),

		// Tiered Pricing Fields
		Div(
			ID("tiered_fields"),
			StyleAttr("display: "+boolToDisplay(selectedPattern == "tiered")),
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-2"),

			Div(
				Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Tiered Pricing Configuration")),
				P(Class("text-sm text-slate-500"), g.Text("Define pricing tiers based on usage or quantity")),
			),

			// Tiered pricing info
			Div(
				Class("rounded-lg bg-blue-50 dark:bg-blue-900/20 p-4 border border-blue-200 dark:border-blue-800"),
				Div(
					Class("flex items-start gap-3"),
					lucide.Info(Class("size-5 text-blue-600 mt-0.5")),
					Div(
						P(Class("text-sm text-blue-800 dark:text-blue-200 font-medium"), g.Text("Tiered Pricing")),
						P(Class("text-sm text-blue-600 dark:text-blue-300 mt-1"),
							g.Text("Configure pricing tiers after creating the plan. Each tier defines a range and price.")),
					),
				),
			),

			// Tier unit name
			Div(
				Label(For("tier_unit"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Tier Unit Name")),
				Input(
					Type("text"), Name("tier_unit"), ID("tier_unit"),
					Value(unitName),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					Placeholder("users, API calls, GB"),
				),
				P(Class("mt-1 text-xs text-slate-500"), g.Text("What each tier unit represents")),
			),
		),

		// Usage-based Fields
		Div(
			ID("usage_fields"),
			StyleAttr("display: "+boolToDisplay(selectedPattern == "usage")),
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-6"),

			Div(
				Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Usage-Based Configuration")),
				P(Class("text-sm text-slate-500"), g.Text("Configure metered usage pricing")),
			),

			Div(
				Class("grid gap-4 md:grid-cols-2"),
				Div(
					Label(For("unit_name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Unit Name")),
					Input(
						Type("text"), Name("unit_name"), ID("unit_name"),
						Value(unitName),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("API calls, GB, messages"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("What customers are paying for")),
				),
				Div(
					Label(For("unit_price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Price per Unit (cents)")),
					Input(
						Type("number"), Name("unit_price"), ID("unit_price"),
						Min("0"), Value(itoa(unitPrice)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("1"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Cost for each unit consumed")),
				),
			),

			// Usage aggregation type
			Div(
				Label(For("usage_aggregation"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Usage Aggregation")),
				Select(
					Name("usage_aggregation"), ID("usage_aggregation"),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					Option(Value("sum"), g.Attr("selected", ""), g.Text("Sum - Total usage in period")),
					Option(Value("max"), g.Text("Max - Peak usage in period")),
					Option(Value("last"), g.Text("Last - Most recent value")),
				),
			),
		),

		// Hybrid Fields
		Div(
			ID("hybrid_fields"),
			StyleAttr("display: "+boolToDisplay(selectedPattern == "hybrid")),
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 space-y-6"),

			Div(
				Class("border-b border-slate-200 dark:border-gray-700 pb-4 mb-6"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("Hybrid Pricing Configuration")),
				P(Class("text-sm text-slate-500"), g.Text("Combine base price with usage-based pricing")),
			),

			Div(
				Class("grid gap-4 md:grid-cols-3"),
				Div(
					Label(For("hybrid_base"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Platform Fee (cents)")),
					Input(
						Type("number"), Name("hybrid_base"), ID("hybrid_base"),
						Min("0"), Value(itoa(hybridBasePrice)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("999"),
					),
					P(Class("mt-1 text-xs text-slate-500"), g.Text("Fixed base charge per period")),
				),
				Div(
					Label(For("hybrid_unit_name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Usage Unit")),
					Input(
						Type("text"), Name("hybrid_unit_name"), ID("hybrid_unit_name"),
						Value(unitName),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("transactions"),
					),
				),
				Div(
					Label(For("hybrid_unit_price"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Per-Unit Price (cents)")),
					Input(
						Type("number"), Name("hybrid_unit_price"), ID("hybrid_unit_price"),
						Min("0"), Value(itoa(unitPrice)),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("10"),
					),
				),
			),

			// Included units
			Div(
				Label(For("included_units"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Included Units")),
				Input(
					Type("number"), Name("included_units"), ID("included_units"),
					Min("0"), Value("0"),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
				),
				P(Class("mt-1 text-xs text-slate-500"), g.Text("Units included in base price (0 = charge from first unit)")),
			),
		),
	})
}

// renderFeatureSelectionList renders the feature selection checkboxes.
func (e *DashboardExtension) renderFeatureSelectionList(features []*core.Feature, linkedMap map[string]*core.PlanFeatureLink) g.Node {
	var rows []g.Node

	for _, feature := range features {
		isLinked := false

		var linkValue string

		if linkedMap != nil {
			if link, ok := linkedMap[feature.ID.String()]; ok {
				isLinked = true
				linkValue = link.Value
			}
		}

		featureID := feature.ID.String()

		// Build value input based on feature type
		var valueInput g.Node

		switch feature.Type {
		case core.FeatureTypeBoolean:
			checked := isLinked && (linkValue == "" || linkValue == "true")
			valueInput = Div(
				Class("flex items-center gap-2"),
				Input(
					Type("checkbox"),
					Name("feature_value_"+featureID),
					ID("feature_value_"+featureID),
					Value("true"),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
					g.If(checked, Checked()),
				),
				Label(For("feature_value_"+featureID),
					Class("text-sm text-slate-600"),
					g.Text("Enabled"),
				),
			)
		case core.FeatureTypeLimit, core.FeatureTypeMetered:
			var limitVal string

			if isLinked && linkValue != "" {
				var val float64
				if json.Unmarshal([]byte(linkValue), &val) == nil {
					limitVal = fmt.Sprintf("%.0f", val)
				} else {
					limitVal = linkValue
				}
			}

			valueInput = Div(
				Class("flex items-center gap-2"),
				Input(
					Type("number"),
					Name("feature_value_"+featureID),
					ID("feature_value_"+featureID),
					Min("0"),
					Value(limitVal),
					Placeholder("e.g., 10"),
					Class("w-24 rounded-md border-slate-300 text-sm shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
				),
				g.If(feature.Unit != "",
					Span(Class("text-sm text-slate-500"), g.Text(feature.Unit)),
				),
			)
		case core.FeatureTypeUnlimited:
			valueInput = Span(Class("text-sm text-green-600 font-medium"), g.Text("∞ Unlimited"))
		case core.FeatureTypeTiered:
			valueInput = Input(
				Type("text"),
				Name("feature_value_"+featureID),
				ID("feature_value_"+featureID),
				Value(linkValue),
				Placeholder("tier name"),
				Class("w-32 rounded-md border-slate-300 text-sm shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
			)
		default:
			valueInput = Input(
				Type("text"),
				Name("feature_value_"+featureID),
				ID("feature_value_"+featureID),
				Value(linkValue),
				Class("w-32 rounded-md border-slate-300 text-sm shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
			)
		}

		row := Div(
			Class("flex items-center justify-between py-3 border-b border-slate-100 dark:border-slate-700 last:border-0"),

			// Feature info with checkbox
			Div(
				Class("flex items-center gap-3"),
				Input(
					Type("checkbox"),
					Name("feature_"+featureID),
					ID("feature_"+featureID),
					Value(featureID),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
					g.If(isLinked, Checked()),
				),
				Div(
					Label(
						For("feature_"+featureID),
						Class("text-sm font-medium text-slate-900 dark:text-white cursor-pointer"),
						g.Text(feature.Name),
					),
					P(Class("text-xs text-slate-500"), g.Text(feature.Key)),
				),
			),

			// Value config
			Div(
				Class("flex items-center gap-4"),
				e.featureTypeBadge(string(feature.Type)),
				valueInput,
			),
		)

		rows = append(rows, row)
	}

	return Div(
		Class("divide-y divide-slate-100 dark:divide-slate-700"),
		g.Group(rows),
	)
}

// Helper for bool to display style.
func boolToDisplay(b bool) string {
	if b {
		return "block"
	}

	return "none"
}

// billingPatternLabel returns a human-readable label for billing pattern.
func billingPatternLabel(pattern core.BillingPattern) string {
	labels := map[core.BillingPattern]string{
		core.BillingPatternFlat:    "Flat Rate",
		core.BillingPatternPerSeat: "Per Seat",
		core.BillingPatternTiered:  "Tiered",
		core.BillingPatternUsage:   "Usage-based",
		core.BillingPatternHybrid:  "Hybrid",
	}
	if label, ok := labels[pattern]; ok {
		return label
	}

	return string(pattern)
}

// Helper for int64 to string.
func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
