package subscription

import (
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
)

// HandleSetDefaultPaymentMethod sets a payment method as default via HTMX.
func (e *DashboardExtension) HandleSetDefaultPaymentMethod(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get payment method ID from URL param
	pmIDStr := ctx.Param("id")
	if pmIDStr == "" {
		return nil, errs.BadRequest("Payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid payment method ID")
	}

	basePath := e.baseUIPath

	// Set as default
	if err := e.plugin.paymentSvc.SetDefaultPaymentMethod(reqCtx, currentApp.ID, pmID); err != nil {
		return nil, errs.InternalServerError("Failed to set default payment method", err)
	}

	// Fetch updated payment methods list
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(reqCtx, currentApp.ID)
	if err != nil {
		paymentMethods = []*core.PaymentMethod{}
	}

	// Return updated payment methods list
	html := g.Group(g.Map(paymentMethods, func(pm *core.PaymentMethod) g.Node {
		return renderPaymentMethodCard(pm, basePath, currentApp.ID.String())
	}))

	// Set success toast trigger
	ctx.ResponseWriter.Header().Set("Hx-Trigger", `{"showToast": {"message": "Payment method set as default", "type": "success"}}`)

	return html, nil
}

// HandleRemovePaymentMethod removes a payment method via HTMX.
func (e *DashboardExtension) HandleRemovePaymentMethod(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	// basePath := e.baseUIPath

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get payment method ID from URL param
	pmIDStr := ctx.Param("id")
	if pmIDStr == "" {
		return nil, errs.BadRequest("Payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid payment method ID")
	}

	basePath := e.baseUIPath

	// Remove payment method
	if err := e.plugin.paymentSvc.RemovePaymentMethod(reqCtx, pmID); err != nil {
		// Check if it's trying to delete default payment method
		ctx.ResponseWriter.Header().Set("Hx-Trigger", `{"showToast": {"message": "Cannot remove default payment method", "type": "error"}}`)

		return nil, errs.BadRequest("Cannot remove default payment method")
	}

	// Fetch updated payment methods list
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(reqCtx, currentApp.ID)
	if err != nil {
		paymentMethods = []*core.PaymentMethod{}
	}

	// Return updated payment methods list
	html := g.Group(g.Map(paymentMethods, func(pm *core.PaymentMethod) g.Node {
		return renderPaymentMethodCard(pm, basePath, currentApp.ID.String())
	}))

	// Set success toast trigger
	ctx.ResponseWriter.Header().Set("Hx-Trigger", `{"showToast": {"message": "Payment method removed", "type": "success"}}`)

	return html, nil
}

// renderHTMX is a helper to render gomponents to HTTP response.
func renderHTMX(c *router.PageContext, node g.Node) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter.WriteHeader(http.StatusOK)

	return node.Render(c.ResponseWriter)
}
