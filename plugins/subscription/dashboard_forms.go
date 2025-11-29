package subscription

import (
	"fmt"
	"net/http"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServePlanCreatePage renders the plan creation form
func (e *DashboardExtension) ServePlanCreatePage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	content := Div(
		Class("space-y-6"),

		// Back button and header
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Plans"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Create Plan")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Create a new subscription plan for your customers")),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/create"),
				Class("space-y-6"),

				// Plan name
				Div(
					Label(For("name"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Plan Name")),
					Input(
						Type("text"), Name("name"), ID("name"), Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Pro Plan"),
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
					Class("space-y-4"),
					// Is Active
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_active"), ID("is_active"), Value("true"), Checked(),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_active"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Active - Plan is available for new subscriptions")),
					),
					// Is Public
					Div(
						Class("flex items-center"),
						Input(Type("checkbox"), Name("is_public"), ID("is_public"), Value("true"), Checked(),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500")),
						Label(For("is_public"), Class("ml-2 text-sm text-slate-700 dark:text-gray-300"), g.Text("Public - Plan is visible on pricing pages")),
					),
				),

				// Submit buttons
				Div(
					Class("flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-gray-800"),
					A(
						Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans"),
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
		),
	)

	pageData := components.PageData{
		Title:      "Create Plan",
		User:       currentUser,
		ActivePage: "plans",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// ServePlanEditPage renders the plan edit form
func (e *DashboardExtension) ServePlanEditPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()

	planID, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid plan ID")
	}

	plan, err := e.plugin.planSvc.GetByID(ctx, planID)
	if err != nil {
		return c.String(http.StatusNotFound, "Plan not found")
	}

	content := Div(
		Class("space-y-6"),

		// Back button and header
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Plan"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Edit Plan")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Update plan details and pricing")),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()+"/update"),
				Class("space-y-6"),

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

				// Billing pattern
				Div(
					Label(For("billing_pattern"), Class("block text-sm font-medium text-slate-700 dark:text-gray-300"), g.Text("Billing Pattern")),
					Select(
						Name("billing_pattern"), ID("billing_pattern"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("flat"), g.If(plan.BillingPattern == "flat", g.Attr("selected", "")), g.Text("Flat Rate")),
						Option(Value("per_seat"), g.If(plan.BillingPattern == "per_seat", g.Attr("selected", "")), g.Text("Per Seat")),
						Option(Value("tiered"), g.If(plan.BillingPattern == "tiered", g.Attr("selected", "")), g.Text("Tiered")),
						Option(Value("usage"), g.If(plan.BillingPattern == "usage", g.Attr("selected", "")), g.Text("Usage-based")),
						Option(Value("hybrid"), g.If(plan.BillingPattern == "hybrid", g.Attr("selected", "")), g.Text("Hybrid")),
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
					Class("space-y-4"),
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

				// Submit buttons
				Div(
					Class("flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-gray-800"),
					A(
						Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/plans/"+plan.ID.String()),
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
		),
	)

	pageData := components.PageData{
		Title:      "Edit Plan: " + plan.Name,
		User:       currentUser,
		ActivePage: "plans",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeAddOnCreatePage renders the add-on creation form
func (e *DashboardExtension) ServeAddOnCreatePage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	content := Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/addons"),
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
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/addons/create"),
				Class("space-y-6"),

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
						Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/addons"),
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

	pageData := components.PageData{
		Title:      "Create Add-on",
		User:       currentUser,
		ActivePage: "addons",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeCouponCreatePage renders the coupon creation form
func (e *DashboardExtension) ServeCouponCreatePage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	content := Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/coupons"),
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
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/coupons/create"),
				Class("space-y-6"),

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
						Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/billing/coupons"),
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

	pageData := components.PageData{
		Title:      "Create Coupon",
		User:       currentUser,
		ActivePage: "coupons",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	return handler.RenderWithLayout(c, pageData, content)
}

// Helper for int64 to string
func itoa(i int64) string {
	return fmt.Sprintf("%d", i)
}
