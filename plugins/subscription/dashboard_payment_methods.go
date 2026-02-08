package subscription

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ServePaymentMethodsPage renders the payment methods management page
func (e *DashboardExtension) ServePaymentMethodsPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Get payment methods for this organization
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(reqCtx, currentApp.ID)
	if err != nil {
		paymentMethods = []*core.PaymentMethod{}
	}

	content := Div(
		Class("space-y-6"),

		// Header with back button
		Div(
			Class("flex items-center justify-between"),
			Div(
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/billing"),
					Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white mb-4"),
					lucide.ArrowLeft(Class("size-4")),
					g.Text("Back to Billing"),
				),
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Payment Methods")),
				P(Class("text-slate-600 dark:text-gray-400 mt-2"), g.Text("Manage your payment methods for subscription billing")),
			),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/billing/payment-methods/add"),
				Class("inline-flex items-center gap-2 px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 transition-colors"),
				lucide.Plus(Class("size-4")),
				g.Text("Add Payment Method"),
			),
		),

		// Payment methods list
		g.If(len(paymentMethods) == 0,
			renderEmptyPaymentMethodsState(basePath, currentApp.ID.String()),
		),

		g.If(len(paymentMethods) > 0,
			Div(
				Class("space-y-4"),
				ID("payment-methods-list"),
				g.Group(g.Map(paymentMethods, func(pm *core.PaymentMethod) g.Node {
					return renderPaymentMethodCard(pm, basePath, currentApp.ID.String())
				})),
			),
		),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeAddPaymentMethodPage renders the add payment method page with Stripe Elements
func (e *DashboardExtension) ServeAddPaymentMethodPage(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.baseUIPath

	// Create setup intent
	setupIntent, err := e.plugin.paymentSvc.CreateSetupIntent(reqCtx, currentApp.ID)
	if err != nil {
		return nil, errs.InternalServerError("Failed to create setup intent: "+err.Error(), err)
	}

	// Get Stripe publishable key from config
	publishableKey := e.plugin.config.StripeConfig.PublishableKey
	if publishableKey == "" {
		return nil, errs.InternalServerError("Stripe not configured", nil)
	}

	content := Div(
		Class("space-y-6 max-w-2xl"),

		// Header with back button
		A(
			Href(basePath+"/app/"+currentApp.ID.String()+"/billing/payment-methods"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Payment Methods"),
		),

		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Add Payment Method")),
		P(Class("text-slate-600 dark:text-gray-400"), g.Text("Add a new payment method for your subscription billing")),

		// Info banner
		Div(
			Class("rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950"),
			Div(
				Class("flex gap-3"),
				lucide.Info(Class("size-5 text-blue-600 dark:text-blue-400 flex-shrink-0")),
				Div(
					Class("text-sm text-blue-900 dark:text-blue-100"),
					P(Class("font-medium"), g.Text("Secure Payment Processing")),
					P(Class("mt-1"), g.Text("Your payment information is processed securely by Stripe. We never see or store your full card details.")),
				),
			),
		),

		// Stripe Elements form
		renderStripeElementsForm(setupIntent.ClientSecret, publishableKey, basePath, currentApp.ID.String()),
	)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderEmptyPaymentMethodsState renders the empty state for payment methods
func renderEmptyPaymentMethodsState(basePath string, appID string) g.Node {
	return Div(
		Class("rounded-lg border-2 border-dashed border-slate-200 bg-slate-50 p-12 text-center dark:border-gray-700 dark:bg-gray-900"),
		Div(
			Class("mx-auto flex max-w-md flex-col items-center"),
			lucide.CreditCard(Class("size-12 text-slate-400 dark:text-gray-600 mb-4")),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text("No Payment Methods")),
			P(Class("text-sm text-slate-600 dark:text-gray-400 mb-6"),
				g.Text("Add a payment method to enable automatic billing for your subscriptions."),
			),
			A(
				Href(basePath+"/app/"+appID+"/billing/payment-methods/add"),
				Class("inline-flex items-center gap-2 px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 transition-colors"),
				lucide.Plus(Class("size-4")),
				g.Text("Add Payment Method"),
			),
		),
	)
}

// renderPaymentMethodCard renders a payment method card component
func renderPaymentMethodCard(pm *core.PaymentMethod, basePath string, appID string) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Class("flex items-center gap-4"),
				// Card icon
				Div(
					Class("flex h-12 w-12 items-center justify-center rounded-lg bg-slate-100 dark:bg-gray-800"),
					g.If(pm.IsCard(),
						lucide.CreditCard(Class("size-6 text-slate-600 dark:text-gray-400")),
					),
					g.If(pm.IsBankAccount(),
						lucide.Building2(Class("size-6 text-slate-600 dark:text-gray-400")),
					),
				),
				// Card details
				Div(
					Div(
						Class("flex items-center gap-2"),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
							g.Text(pm.DisplayName()),
						),
						g.If(pm.IsDefault,
							Span(
								Class("inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"),
								g.Text("Default"),
							),
						),
						g.If(pm.IsExpired(),
							Span(
								Class("inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-200"),
								g.Text("Expired"),
							),
						),
						g.If(pm.WillExpireSoon(30) && !pm.IsExpired(),
							Span(
								Class("inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"),
								g.Text("Expiring Soon"),
							),
						),
					),
					P(Class("text-sm text-slate-600 dark:text-gray-400 mt-1"),
						g.If(pm.IsCard(),
							g.Textf("Expires %02d/%d", pm.CardExpMonth, pm.CardExpYear),
						),
						g.If(pm.IsBankAccount() && pm.BankAccountType != "",
							g.Textf("%s account", pm.BankAccountType),
						),
					),
				),
			),
			// Actions
			Div(
				Class("flex items-center gap-2"),
				g.If(!pm.IsDefault,
					Button(
						Type("button"),
						g.Attr("hx-post", basePath+"/app/"+appID+"/billing/payment-methods/set-default/"+pm.ID.String()),
						g.Attr("hx-target", "#payment-methods-list"),
						g.Attr("hx-swap", "outerHTML"),
						Class("px-3 py-1.5 text-sm font-medium text-violet-600 hover:text-violet-700 hover:bg-violet-50 rounded-md transition-colors dark:text-violet-400 dark:hover:bg-violet-950"),
						g.Text("Set as Default"),
					),
				),
				g.If(!pm.IsDefault,
					Button(
						Type("button"),
						g.Attr("hx-delete", basePath+"/app/"+appID+"/billing/payment-methods/"+pm.ID.String()),
						g.Attr("hx-confirm", "Are you sure you want to remove this payment method?"),
						g.Attr("hx-target", "#payment-methods-list"),
						g.Attr("hx-swap", "outerHTML"),
						Class("p-1.5 text-red-600 hover:text-red-700 hover:bg-red-50 rounded-md transition-colors dark:text-red-400 dark:hover:bg-red-950"),
						lucide.Trash2(Class("size-4")),
					),
				),
			),
		),
	)
}

// renderStripeElementsForm renders the Stripe Elements payment form
func renderStripeElementsForm(clientSecret, publishableKey, basePath, appID string) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),

		// Form
		Form(
			ID("payment-form"),
			Class("space-y-6"),

			// Stripe Elements container
			Div(
				ID("payment-element"),
				Class("mb-4"),
			),

			// Set as default checkbox
			Div(
				Class("flex items-center gap-2"),
				Input(
					Type("checkbox"),
					ID("set-default"),
					Name("setDefault"),
					Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
				),
				Label(
					For("set-default"),
					Class("text-sm text-slate-700 dark:text-gray-300"),
					g.Text("Set as default payment method"),
				),
			),

			// Error message container
			Div(
				ID("error-message"),
				Class("hidden text-sm text-red-600 dark:text-red-400"),
			),

			// Submit button
			Button(
				Type("submit"),
				ID("submit-button"),
				Class("w-full px-4 py-3 bg-violet-600 text-white font-medium rounded-lg hover:bg-violet-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"),
				g.Text("Add Payment Method"),
			),

			// Loading spinner (hidden by default)
			Div(
				ID("loading-spinner"),
				Class("hidden text-center text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Processing..."),
			),
		),

		// Stripe script and integration
		Script(Src("https://js.stripe.com/v3/")),
		Script(
			g.Raw(fmt.Sprintf(`
const stripe = Stripe('%s');
const elements = stripe.elements({
    clientSecret: '%s',
    appearance: {
        theme: 'stripe',
        variables: {
            colorPrimary: '#7c3aed',
            colorBackground: document.documentElement.classList.contains('dark') ? '#111827' : '#ffffff',
            colorText: document.documentElement.classList.contains('dark') ? '#f3f4f6' : '#1f2937',
            colorDanger: '#dc2626',
            fontFamily: 'system-ui, sans-serif',
            borderRadius: '0.5rem',
        }
    }
});

const paymentElement = elements.create('payment', {
    layout: 'tabs',
});

paymentElement.mount('#payment-element');

const form = document.getElementById('payment-form');
const submitButton = document.getElementById('submit-button');
const errorMessage = document.getElementById('error-message');
const loadingSpinner = document.getElementById('loading-spinner');

form.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    // Disable submit button and show loading
    submitButton.disabled = true;
    submitButton.classList.add('opacity-50');
    loadingSpinner.classList.remove('hidden');
    errorMessage.classList.add('hidden');
    errorMessage.textContent = '';
    
    const {error, setupIntent} = await stripe.confirmSetup({
        elements,
        confirmParams: {
            return_url: window.location.origin + '%s/app/%s/billing/payment-methods',
        },
        redirect: 'if_required'
    });
    
    if (error) {
        errorMessage.textContent = error.message;
        errorMessage.classList.remove('hidden');
        submitButton.disabled = false;
        submitButton.classList.remove('opacity-50');
        loadingSpinner.classList.add('hidden');
    } else if (setupIntent && setupIntent.status === 'succeeded') {
        // Attach payment method to organization
        try {
            const response = await fetch('%s/subscription/payment-methods', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json'
                },
                body: JSON.stringify({
                    organizationId: '%s',
                    paymentMethodId: setupIntent.payment_method,
                    setAsDefault: document.getElementById('set-default').checked
                })
            });
            
            if (response.ok) {
                window.location.href = '%s/app/%s/billing/payment-methods';
            } else {
                const data = await response.json();
                errorMessage.textContent = data.message || 'Failed to add payment method';
                errorMessage.classList.remove('hidden');
                submitButton.disabled = false;
                submitButton.classList.remove('opacity-50');
                loadingSpinner.classList.add('hidden');
            }
        } catch (err) {
            errorMessage.textContent = 'Network error: ' + err.message;
            errorMessage.classList.remove('hidden');
            submitButton.disabled = false;
            submitButton.classList.remove('opacity-50');
            loadingSpinner.classList.add('hidden');
        }
    }
});
`, publishableKey, clientSecret, basePath, appID, basePath, appID, basePath, appID)),
		),
	)
}
