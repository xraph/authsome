package handlers

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// PaymentHandlers handles payment method HTTP endpoints
type PaymentHandlers struct {
	paymentSvc  *service.PaymentService
	customerSvc *service.CustomerService
}

// NewPaymentHandlers creates a new PaymentHandlers instance
func NewPaymentHandlers(paymentSvc *service.PaymentService, customerSvc *service.CustomerService) *PaymentHandlers {
	return &PaymentHandlers{
		paymentSvc:  paymentSvc,
		customerSvc: customerSvc,
	}
}

// CreateSetupIntentRequest is the request body for creating a setup intent
type CreateSetupIntentRequest struct {
	OrganizationID xid.ID `json:"organizationId" validate:"required"`
}

// AddPaymentMethodRequest is the request body for adding a payment method
type AddPaymentMethodRequest struct {
	OrganizationID  xid.ID `json:"organizationId" validate:"required"`
	PaymentMethodID string `json:"paymentMethodId" validate:"required"`
	SetAsDefault    bool   `json:"setAsDefault"`
}

// SetDefaultPaymentMethodRequest is the request body for setting default payment method
type SetDefaultPaymentMethodRequest struct {
	OrganizationID xid.ID `json:"organizationId" validate:"required"`
}

// HandleCreateSetupIntent creates a setup intent for adding payment method
func (h *PaymentHandlers) HandleCreateSetupIntent(c forge.Context) error {
	ctx := c.Request().Context()

	var req CreateSetupIntentRequest
	if err := c.BindRequest(&req); err != nil {
		return errs.BadRequest(err.Error())
	}

	// Ensure customer exists for this organization
	customer, err := h.customerSvc.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		// Customer doesn't exist, we need organization details to create one
		return errs.BadRequest("customer not found for organization - create subscription first")
	}

	// Create setup intent
	result, err := h.paymentSvc.CreateSetupIntent(ctx, req.OrganizationID)
	if err != nil {
		return errs.InternalError(fmt.Errorf("failed to create setup intent: %w", err))
	}

	return c.JSON(200, map[string]interface{}{
		"clientSecret":   result.ClientSecret,
		"setupIntentId":  result.SetupIntentID,
		"publishableKey": result.PublishableKey,
		"customerId":     customer.ProviderCustomerID,
	})
}

// HandleAddPaymentMethod attaches a tokenized payment method
func (h *PaymentHandlers) HandleAddPaymentMethod(c forge.Context) error {
	ctx := c.Request().Context()

	var req AddPaymentMethodRequest
	if err := c.BindRequest(&req); err != nil {
		return errs.BadRequest(err.Error())
	}

	// Validate payment method ID format (Stripe format: pm_xxx)
	if len(req.PaymentMethodID) < 3 || req.PaymentMethodID[:3] != "pm_" {
		return errs.BadRequest("invalid payment method ID format")
	}

	// Add payment method
	paymentMethod, err := h.paymentSvc.AddPaymentMethod(ctx, req.OrganizationID, req.PaymentMethodID, req.SetAsDefault)
	if err != nil {
		return errs.InternalError(fmt.Errorf("failed to add payment method: %w", err))
	}

	return c.JSON(201, paymentMethod)
}

// HandleListPaymentMethods lists all payment methods for org
func (h *PaymentHandlers) HandleListPaymentMethods(c forge.Context) error {
	ctx := c.Request().Context()

	// Get organizationId from query param
	orgIDStr := c.Query("organizationId")
	if orgIDStr == "" {
		return errs.BadRequest("organizationId query parameter required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return errs.BadRequest("invalid organizationId format")
	}

	paymentMethods, err := h.paymentSvc.ListPaymentMethods(ctx, orgID)
	if err != nil {
		return errs.InternalError(fmt.Errorf("failed to list payment methods: %w", err))
	}

	return c.JSON(200, map[string]interface{}{
		"paymentMethods": paymentMethods,
		"total":          len(paymentMethods),
	})
}

// HandleGetPaymentMethod gets a single payment method
func (h *PaymentHandlers) HandleGetPaymentMethod(c forge.Context) error {
	ctx := c.Request().Context()

	// Get payment method ID from URL param
	pmIDStr := c.Param("id")
	if pmIDStr == "" {
		return errs.BadRequest("payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return errs.BadRequest("invalid payment method ID format")
	}

	// Get from repository (we need to add this method to PaymentService)
	// For now, list all and filter
	orgIDStr := c.Query("organizationId")
	if orgIDStr == "" {
		return errs.BadRequest("organizationId query parameter required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return errs.BadRequest("invalid organizationId format")
	}

	paymentMethods, err := h.paymentSvc.ListPaymentMethods(ctx, orgID)
	if err != nil {
		return errs.InternalError(fmt.Errorf("failed to get payment method: %w", err))
	}

	// Find the specific payment method
	for _, pm := range paymentMethods {
		if pm.ID == pmID {
			return c.JSON(200, pm)
		}
	}

	return errs.NotFound("payment method not found")
}

// HandleSetDefaultPaymentMethod sets default payment method
func (h *PaymentHandlers) HandleSetDefaultPaymentMethod(c forge.Context) error {
	ctx := c.Request().Context()

	// Get payment method ID from URL param
	pmIDStr := c.Param("id")
	if pmIDStr == "" {
		return errs.BadRequest("payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return errs.BadRequest("invalid payment method ID format")
	}

	var req SetDefaultPaymentMethodRequest
	if err := c.BindRequest(&req); err != nil {
		return errs.BadRequest(err.Error())
	}

	// Set as default
	if err := h.paymentSvc.SetDefaultPaymentMethod(ctx, req.OrganizationID, pmID); err != nil {
		return errs.InternalError(fmt.Errorf("failed to set default payment method: %w", err))
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Payment method set as default",
	})
}

// HandleRemovePaymentMethod removes a payment method
func (h *PaymentHandlers) HandleRemovePaymentMethod(c forge.Context) error {
	ctx := c.Request().Context()

	// Get payment method ID from URL param
	pmIDStr := c.Param("id")
	if pmIDStr == "" {
		return errs.BadRequest("payment method ID required")
	}

	pmID, err := xid.FromString(pmIDStr)
	if err != nil {
		return errs.BadRequest("invalid payment method ID format")
	}

	// Remove payment method
	if err := h.paymentSvc.RemovePaymentMethod(ctx, pmID); err != nil {
		return errs.InternalError(fmt.Errorf("failed to remove payment method: %w", err))
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Payment method removed",
	})
}
