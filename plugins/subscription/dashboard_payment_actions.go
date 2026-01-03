package subscription

import (
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
)

// HandleSetDefaultPaymentMethod sets a payment method as default via HTMX
func (e *DashboardExtension) HandleSetDefaultPaymentMethod(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Get payment method ID from URL param
	pmIDStr := c.Param("id")
	if pmIDStr == "" {
		return c.String(http.StatusBadRequest, "Payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid payment method ID")
	}

	ctx := c.Request().Context()
	basePath := handler.GetBasePath()

	// Set as default
	if err := e.plugin.paymentSvc.SetDefaultPaymentMethod(ctx, currentApp.ID, pmID); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to set default payment method")
	}

	// Fetch updated payment methods list
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(ctx, currentApp.ID)
	if err != nil {
		paymentMethods = []*core.PaymentMethod{}
	}

	// Return updated payment methods list
	html := g.Group(g.Map(paymentMethods, func(pm *core.PaymentMethod) g.Node {
		return renderPaymentMethodCard(pm, basePath, currentApp.ID.String())
	}))

	// Set success toast trigger
	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Payment method set as default", "type": "success"}}`)

	return renderHTMX(c, html)
}

// HandleRemovePaymentMethod removes a payment method via HTMX
func (e *DashboardExtension) HandleRemovePaymentMethod(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Get payment method ID from URL param
	pmIDStr := c.Param("id")
	if pmIDStr == "" {
		return c.String(http.StatusBadRequest, "Payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid payment method ID")
	}

	ctx := c.Request().Context()
	basePath := handler.GetBasePath()

	// Remove payment method
	if err := e.plugin.paymentSvc.RemovePaymentMethod(ctx, pmID); err != nil {
		// Check if it's trying to delete default payment method
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Cannot remove default payment method", "type": "error"}}`)
		return c.String(http.StatusBadRequest, "Cannot remove default payment method")
	}

	// Fetch updated payment methods list
	paymentMethods, err := e.plugin.paymentSvc.ListPaymentMethods(ctx, currentApp.ID)
	if err != nil {
		paymentMethods = []*core.PaymentMethod{}
	}

	// Return updated payment methods list
	html := g.Group(g.Map(paymentMethods, func(pm *core.PaymentMethod) g.Node {
		return renderPaymentMethodCard(pm, basePath, currentApp.ID.String())
	}))

	// Set success toast trigger
	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Payment method removed", "type": "success"}}`)

	return renderHTMX(c, html)
}

// renderHTMX is a helper to render gomponents to HTTP response
func renderHTMX(c forge.Context, node g.Node) error {
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return node.Render(c.Response())
}

