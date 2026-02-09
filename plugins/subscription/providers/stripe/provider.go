// Package stripe provides Stripe payment provider implementation.
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/entitlements/feature"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
	"github.com/stripe/stripe-go/v76/setupintent"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/subscriptionitem"
	"github.com/stripe/stripe-go/v76/usagerecord"
	"github.com/stripe/stripe-go/v76/webhook"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers/types"
)

// Provider implements the PaymentProvider interface for Stripe.
type Provider struct {
	secretKey     string
	webhookSecret string
}

// NewStripeProvider creates a new Stripe provider.
func NewStripeProvider(secretKey, webhookSecret string) (*Provider, error) {
	if secretKey == "" {
		return nil, errs.RequiredField("secretKey")
	}

	stripe.Key = secretKey

	return &Provider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "stripe"
}

// CreateCustomer creates a Stripe customer.
func (p *Provider) CreateCustomer(ctx context.Context, email, name string, metadata map[string]any) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	if metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return c.ID, nil
}

// UpdateCustomer updates a Stripe customer.
func (p *Provider) UpdateCustomer(ctx context.Context, customerID, email, name string, metadata map[string]any) error {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	if metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	_, err := customer.Update(customerID, params)

	return err
}

// DeleteCustomer deletes a Stripe customer.
func (p *Provider) DeleteCustomer(ctx context.Context, customerID string) error {
	_, err := customer.Del(customerID, nil)

	return err
}

// SyncPlan syncs a plan to Stripe (creates Product and Price).
func (p *Provider) SyncPlan(ctx context.Context, plan *core.Plan) error {
	// Build metadata for AuthSome plan identification and recovery
	metadata := map[string]string{
		"authsome":         "true", // Marker to identify AuthSome-created products
		"plan_id":          plan.ID.String(),
		"app_id":           plan.AppID.String(),
		"slug":             plan.Slug,
		"billing_pattern":  string(plan.BillingPattern),
		"billing_interval": string(plan.BillingInterval),
	}

	// Create or update product
	var productID string

	if plan.ProviderPlanID != "" {
		// Update existing product
		params := &stripe.ProductParams{
			Name:     stripe.String(plan.Name),
			Active:   stripe.Bool(plan.IsActive),
			Metadata: metadata,
		}
		// Only set description if not empty (Stripe rejects empty strings)
		if plan.Description != "" {
			params.Description = stripe.String(plan.Description)
		}

		_, err := product.Update(plan.ProviderPlanID, params)
		if err != nil {
			return fmt.Errorf("failed to update Stripe product: %w", err)
		}

		productID = plan.ProviderPlanID
	} else {
		// Create new product
		params := &stripe.ProductParams{
			Name:     stripe.String(plan.Name),
			Active:   stripe.Bool(plan.IsActive),
			Metadata: metadata,
		}
		// Only set description if not empty (Stripe rejects empty strings)
		if plan.Description != "" {
			params.Description = stripe.String(plan.Description)
		}

		prod, err := product.New(params)
		if err != nil {
			return fmt.Errorf("failed to create Stripe product: %w", err)
		}

		productID = prod.ID
		plan.ProviderPlanID = productID
	}

	// Create or update price
	if plan.ProviderPriceID == "" {
		priceParams := &stripe.PriceParams{
			Product:    stripe.String(productID),
			Currency:   stripe.String(plan.Currency),
			UnitAmount: stripe.Int64(plan.BasePrice),
			Active:     stripe.Bool(plan.IsActive),
		}

		// Set recurring for subscriptions
		if plan.BillingInterval != core.BillingIntervalOneTime {
			interval := stripe.PriceRecurringIntervalMonth
			if plan.BillingInterval == core.BillingIntervalYearly {
				interval = stripe.PriceRecurringIntervalYear
			}

			priceParams.Recurring = &stripe.PriceRecurringParams{
				Interval: stripe.String(string(interval)),
			}
		}

		pr, err := price.New(priceParams)
		if err != nil {
			return fmt.Errorf("failed to create Stripe price: %w", err)
		}

		plan.ProviderPriceID = pr.ID
	}

	return nil
}

// SyncAddOn syncs an add-on to Stripe.
func (p *Provider) SyncAddOn(ctx context.Context, addon *core.AddOn) error {
	// Build metadata for recovery and identification
	metadata := map[string]string{
		"authsome":         "true",
		"addon_id":         addon.ID.String(),
		"app_id":           addon.AppID.String(),
		"slug":             addon.Slug,
		"billing_pattern":  string(addon.BillingPattern),
		"billing_interval": string(addon.BillingInterval),
	}

	// Create or update product
	var productID string

	if addon.ProviderPriceID != "" {
		// Extract product ID from existing price
		// For Stripe, we need to get the price to find the product
		return nil // Already synced
	}

	// Create new product for add-on
	params := &stripe.ProductParams{
		Name:     stripe.String(addon.Name),
		Active:   stripe.Bool(addon.IsActive),
		Metadata: metadata,
	}
	if addon.Description != "" {
		params.Description = stripe.String(addon.Description)
	}

	prod, err := product.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe product for add-on: %w", err)
	}

	productID = prod.ID

	// Create price
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(productID),
		Currency:   stripe.String(addon.Currency),
		UnitAmount: stripe.Int64(addon.Price),
		Active:     stripe.Bool(addon.IsActive),
	}

	// Set recurring for subscriptions
	if addon.BillingInterval != core.BillingIntervalOneTime {
		interval := stripe.PriceRecurringIntervalMonth
		if addon.BillingInterval == core.BillingIntervalYearly {
			interval = stripe.PriceRecurringIntervalYear
		}

		priceParams.Recurring = &stripe.PriceRecurringParams{
			Interval: stripe.String(string(interval)),
		}
	}

	pr, err := price.New(priceParams)
	if err != nil {
		return fmt.Errorf("failed to create Stripe price for add-on: %w", err)
	}

	// Update add-on with provider IDs
	addon.ProviderPriceID = pr.ID

	return nil
}

// CreateSubscription creates a Stripe subscription.
func (p *Provider) CreateSubscription(ctx context.Context, customerID string, priceID string, quantity int, trialDays int, metadata map[string]any) (string, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(int64(quantity)),
			},
		},
	}

	if trialDays > 0 {
		params.TrialPeriodDays = stripe.Int64(int64(trialDays))
	}

	if metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	sub, err := subscription.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	return sub.ID, nil
}

// UpdateSubscription updates a Stripe subscription.
func (p *Provider) UpdateSubscription(ctx context.Context, subscriptionID string, priceID string, quantity int) error {
	// Get existing subscription to find item ID
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return err
	}

	if len(sub.Items.Data) == 0 {
		return errs.BadRequest("subscription has no items")
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(sub.Items.Data[0].ID),
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(int64(quantity)),
			},
		},
	}

	_, err = subscription.Update(subscriptionID, params)

	return err
}

// CancelSubscription cancels a Stripe subscription.
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	if immediate {
		_, err := subscription.Cancel(subscriptionID, nil)

		return err
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := subscription.Update(subscriptionID, params)

	return err
}

// PauseSubscription pauses a Stripe subscription.
func (p *Provider) PauseSubscription(ctx context.Context, subscriptionID string) error {
	params := &stripe.SubscriptionParams{
		PauseCollection: &stripe.SubscriptionPauseCollectionParams{
			Behavior: stripe.String(string(stripe.SubscriptionPauseCollectionBehaviorVoid)),
		},
	}
	_, err := subscription.Update(subscriptionID, params)

	return err
}

// ResumeSubscription resumes a Stripe subscription.
func (p *Provider) ResumeSubscription(ctx context.Context, subscriptionID string) error {
	params := &stripe.SubscriptionParams{}
	params.AddExtra("pause_collection", "")
	_, err := subscription.Update(subscriptionID, params)

	return err
}

// AddSubscriptionItem adds an item (add-on) to an existing subscription.
func (p *Provider) AddSubscriptionItem(ctx context.Context, subscriptionID string, priceID string, quantity int) (string, error) {
	params := &stripe.SubscriptionItemParams{
		Subscription: stripe.String(subscriptionID),
		Price:        stripe.String(priceID),
		Quantity:     stripe.Int64(int64(quantity)),
	}

	item, err := subscriptionitem.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to add subscription item: %w", err)
	}

	return item.ID, nil
}

// RemoveSubscriptionItem removes an item from a subscription.
func (p *Provider) RemoveSubscriptionItem(ctx context.Context, subscriptionID string, itemID string) error {
	_, err := subscriptionitem.Del(itemID, nil)

	return err
}

// UpdateSubscriptionItem updates the quantity of a subscription item.
func (p *Provider) UpdateSubscriptionItem(ctx context.Context, subscriptionID string, itemID string, quantity int) error {
	params := &stripe.SubscriptionItemParams{
		Quantity: stripe.Int64(int64(quantity)),
	}

	_, err := subscriptionitem.Update(itemID, params)

	return err
}

// Ensure Provider implements types.PaymentProvider.
var _ types.PaymentProvider = (*Provider)(nil)

// GetSubscription retrieves a Stripe subscription.
func (p *Provider) GetSubscription(ctx context.Context, subscriptionID string) (*types.ProviderSubscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, err
	}

	result := &types.ProviderSubscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		Metadata:           make(map[string]any),
	}

	if sub.TrialStart > 0 {
		result.TrialStart = &sub.TrialStart
	}

	if sub.TrialEnd > 0 {
		result.TrialEnd = &sub.TrialEnd
	}

	if sub.CancelAt > 0 {
		result.CancelAt = &sub.CancelAt
	}

	if sub.CanceledAt > 0 {
		result.CanceledAt = &sub.CanceledAt
	}

	if sub.EndedAt > 0 {
		result.EndedAt = &sub.EndedAt
	}

	if len(sub.Items.Data) > 0 {
		result.PriceID = sub.Items.Data[0].Price.ID
		result.Quantity = int(sub.Items.Data[0].Quantity)
	}

	for k, v := range sub.Metadata {
		result.Metadata[k] = v
	}

	return result, nil
}

// CreateCheckoutSession creates a Stripe checkout session.
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutSession, error) {
	mode := stripe.CheckoutSessionModeSubscription

	switch req.Mode {
	case types.CheckoutModePayment:
		mode = stripe.CheckoutSessionModePayment
	case types.CheckoutModeSetup:
		mode = stripe.CheckoutSessionModeSetup
	}

	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(req.CustomerID),
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Mode:       stripe.String(string(mode)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(int64(req.Quantity)),
			},
		},
	}

	if req.AllowPromoCodes {
		params.AllowPromotionCodes = stripe.Bool(true)
	}

	if req.TrialDays > 0 && mode == stripe.CheckoutSessionModeSubscription {
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(int64(req.TrialDays)),
		}
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &types.CheckoutSession{
		ID:            sess.ID,
		URL:           sess.URL,
		CustomerID:    sess.Customer.ID,
		PaymentStatus: string(sess.PaymentStatus),
	}, nil
}

// CreatePortalSession creates a Stripe billing portal session.
func (p *Provider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}

	return sess.URL, nil
}

// CreateSetupIntent creates a Stripe setup intent.
func (p *Provider) CreateSetupIntent(ctx context.Context, customerID string) (*core.SetupIntentResult, error) {
	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
	}

	si, err := setupintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create setup intent: %w", err)
	}

	return &core.SetupIntentResult{
		ClientSecret:  si.ClientSecret,
		SetupIntentID: si.ID,
	}, nil
}

// GetPaymentMethod retrieves a Stripe payment method.
func (p *Provider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*core.PaymentMethod, error) {
	pm, err := paymentmethod.Get(paymentMethodID, nil)
	if err != nil {
		return nil, err
	}

	result := &core.PaymentMethod{
		ProviderMethodID: pm.ID,
		Type:             core.PaymentMethodType(pm.Type),
	}

	if pm.Card != nil {
		result.CardBrand = string(pm.Card.Brand)
		result.CardLast4 = pm.Card.Last4
		result.CardExpMonth = int(pm.Card.ExpMonth)
		result.CardExpYear = int(pm.Card.ExpYear)
		result.CardFunding = string(pm.Card.Funding)
	}

	return result, nil
}

// AttachPaymentMethod attaches a payment method to a customer.
func (p *Provider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	_, err := paymentmethod.Attach(paymentMethodID, params)

	return err
}

// DetachPaymentMethod detaches a payment method.
func (p *Provider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := paymentmethod.Detach(paymentMethodID, nil)

	return err
}

// SetDefaultPaymentMethod sets the default payment method for a customer.
func (p *Provider) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	params := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	}
	_, err := customer.Update(customerID, params)

	return err
}

// ReportUsage reports usage to Stripe.
func (p *Provider) ReportUsage(ctx context.Context, subscriptionItemID string, records []*core.UsageRecord) (string, error) {
	var lastID string

	for _, record := range records {
		params := &stripe.UsageRecordParams{
			SubscriptionItem: stripe.String(subscriptionItemID),
			Quantity:         stripe.Int64(record.Quantity),
			Timestamp:        stripe.Int64(record.Timestamp.Unix()),
		}

		if record.Action == core.UsageActionSet {
			params.Action = stripe.String(string(stripe.UsageRecordActionSet))
		} else {
			params.Action = stripe.String(string(stripe.UsageRecordActionIncrement))
		}

		ur, err := usagerecord.New(params)
		if err != nil {
			return "", err
		}

		lastID = ur.ID
	}

	return lastID, nil
}

// GetInvoice retrieves a Stripe invoice.
func (p *Provider) GetInvoice(ctx context.Context, invoiceID string) (*types.ProviderInvoice, error) {
	inv, err := invoice.Get(invoiceID, nil)
	if err != nil {
		return nil, err
	}

	return &types.ProviderInvoice{
		ID:             inv.ID,
		CustomerID:     inv.Customer.ID,
		SubscriptionID: inv.Subscription.ID,
		Status:         string(inv.Status),
		Currency:       string(inv.Currency),
		AmountDue:      inv.AmountDue,
		AmountPaid:     inv.AmountPaid,
		Total:          inv.Total,
		PeriodStart:    inv.PeriodStart,
		PeriodEnd:      inv.PeriodEnd,
		PDFURL:         inv.InvoicePDF,
		HostedURL:      inv.HostedInvoiceURL,
	}, nil
}

// GetInvoicePDF returns the PDF URL for an invoice.
func (p *Provider) GetInvoicePDF(ctx context.Context, invoiceID string) (string, error) {
	inv, err := invoice.Get(invoiceID, nil)
	if err != nil {
		return "", err
	}

	return inv.InvoicePDF, nil
}

// VoidInvoice voids a Stripe invoice.
func (p *Provider) VoidInvoice(ctx context.Context, invoiceID string) error {
	_, err := invoice.VoidInvoice(invoiceID, nil)

	return err
}

// ListInvoices lists invoices for a customer from Stripe.
func (p *Provider) ListInvoices(ctx context.Context, customerID string, limit int) ([]*types.ProviderInvoice, error) {
	params := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID),
	}
	params.Limit = stripe.Int64(int64(limit))

	var invoices []*types.ProviderInvoice

	iter := invoice.List(params)
	for iter.Next() {
		inv := iter.Invoice()
		invoices = append(invoices, &types.ProviderInvoice{
			ID:             inv.ID,
			CustomerID:     inv.Customer.ID,
			SubscriptionID: inv.Subscription.ID,
			Status:         string(inv.Status),
			Currency:       string(inv.Currency),
			AmountDue:      inv.AmountDue,
			AmountPaid:     inv.AmountPaid,
			Total:          inv.Total,
			Subtotal:       inv.Subtotal,
			Tax:            inv.Tax,
			PeriodStart:    inv.PeriodStart,
			PeriodEnd:      inv.PeriodEnd,
			PDFURL:         inv.InvoicePDF,
			HostedURL:      inv.HostedInvoiceURL,
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	return invoices, nil
}

// ListSubscriptionInvoices lists invoices for a subscription from Stripe.
func (p *Provider) ListSubscriptionInvoices(ctx context.Context, subscriptionID string, limit int) ([]*types.ProviderInvoice, error) {
	params := &stripe.InvoiceListParams{
		Subscription: stripe.String(subscriptionID),
	}
	params.Limit = stripe.Int64(int64(limit))

	var invoices []*types.ProviderInvoice

	iter := invoice.List(params)
	for iter.Next() {
		inv := iter.Invoice()
		invoices = append(invoices, &types.ProviderInvoice{
			ID:             inv.ID,
			CustomerID:     inv.Customer.ID,
			SubscriptionID: inv.Subscription.ID,
			Status:         string(inv.Status),
			Currency:       string(inv.Currency),
			AmountDue:      inv.AmountDue,
			AmountPaid:     inv.AmountPaid,
			Total:          inv.Total,
			Subtotal:       inv.Subtotal,
			Tax:            inv.Tax,
			PeriodStart:    inv.PeriodStart,
			PeriodEnd:      inv.PeriodEnd,
			PDFURL:         inv.InvoicePDF,
			HostedURL:      inv.HostedInvoiceURL,
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list subscription invoices: %w", err)
	}

	return invoices, nil
}

// SyncFeature syncs a feature to Stripe as an Entitlement Feature.
func (p *Provider) SyncFeature(ctx context.Context, coreFeature *core.Feature) (string, error) {
	// Check if feature already exists in Stripe (has ProviderFeatureID)
	if coreFeature.ProviderFeatureID != "" {
		// Update existing feature
		params := &stripe.EntitlementsFeatureParams{}

		// Update name if changed
		if coreFeature.Name != "" {
			params.Name = stripe.String(coreFeature.Name)
		}

		// Add metadata to track AuthSome feature mapping
		params.AddMetadata("authsome_feature_id", coreFeature.ID.String())
		params.AddMetadata("authsome_feature_key", coreFeature.Key)
		params.AddMetadata("authsome_app_id", coreFeature.AppID.String())
		params.AddMetadata("feature_type", string(coreFeature.Type))
		params.AddMetadata("unit", coreFeature.Unit)

		updatedFeature, err := feature.Update(coreFeature.ProviderFeatureID, params)
		if err != nil {
			return "", fmt.Errorf("failed to update Stripe feature: %w", err)
		}

		return updatedFeature.ID, nil
	}

	// Create new feature in Stripe
	params := &stripe.EntitlementsFeatureParams{}
	params.Name = stripe.String(coreFeature.Name)

	// Stripe requires a lookup_key - use our feature key with app prefix for uniqueness
	lookupKey := fmt.Sprintf("%s_%s", coreFeature.AppID.String(), coreFeature.Key)
	params.LookupKey = stripe.String(lookupKey)

	// Add metadata to track AuthSome feature mapping
	params.AddMetadata("authsome_feature_id", coreFeature.ID.String())
	params.AddMetadata("authsome_feature_key", coreFeature.Key)
	params.AddMetadata("authsome_app_id", coreFeature.AppID.String())
	params.AddMetadata("feature_type", string(coreFeature.Type))
	params.AddMetadata("unit", coreFeature.Unit)
	params.AddMetadata("description", coreFeature.Description)

	stripeFeature, err := feature.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe feature: %w", err)
	}

	return stripeFeature.ID, nil
}

// ListProviderFeatures lists features from Stripe for a product.
func (p *Provider) ListProviderFeatures(ctx context.Context, productID string) ([]*types.ProviderFeature, error) {
	var features []*types.ProviderFeature

	params := &stripe.EntitlementsFeatureListParams{}
	params.Limit = stripe.Int64(100)

	// Note: Metadata is included by default in Stripe's feature list response
	// We don't need to expand it (and Stripe doesn't allow expanding it)

	iter := feature.List(params)
	for iter.Next() {
		feat := iter.EntitlementsFeature()

		// Filter by app ID from metadata if specified
		if productID != "" {
			if appID, exists := feat.Metadata["authsome_app_id"]; !exists || appID != productID {
				continue
			}
		}

		metadata := make(map[string]any)
		for k, v := range feat.Metadata {
			metadata[k] = v
		}

		providerFeature := &types.ProviderFeature{
			ID:        feat.ID,
			Name:      feat.Name,
			LookupKey: feat.LookupKey,
			Metadata:  metadata,
		}

		features = append(features, providerFeature)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list Stripe features: %w", err)
	}

	return features, nil
}

// GetProviderFeature gets a specific feature from Stripe.
func (p *Provider) GetProviderFeature(ctx context.Context, featureID string) (*types.ProviderFeature, error) {
	feat, err := feature.Get(featureID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe feature: %w", err)
	}

	metadata := make(map[string]any)
	for k, v := range feat.Metadata {
		metadata[k] = v
	}

	providerFeature := &types.ProviderFeature{
		ID:        feat.ID,
		Name:      feat.Name,
		LookupKey: feat.LookupKey,
		Metadata:  metadata,
	}

	return providerFeature, nil
}

// DeleteProviderFeature deletes a feature from Stripe.
func (p *Provider) DeleteProviderFeature(ctx context.Context, featureID string) error {
	// Note: Stripe Entitlements Features cannot be deleted once created
	// They can only be archived/deactivated
	// We'll return nil here since this is not an error condition
	// The feature will remain in Stripe but won't be synced anymore
	return nil
}

// ListProducts lists all products from Stripe, filtering for AuthSome-created products.
func (p *Provider) ListProducts(ctx context.Context) ([]*types.ProviderProduct, error) {
	var products []*types.ProviderProduct

	params := &stripe.ProductListParams{}
	params.Limit = stripe.Int64(100)

	iter := product.List(params)
	for iter.Next() {
		prod := iter.Product()

		// Convert metadata
		metadata := make(map[string]string)
		maps.Copy(metadata, prod.Metadata)

		products = append(products, &types.ProviderProduct{
			ID:          prod.ID,
			Name:        prod.Name,
			Description: prod.Description,
			Active:      prod.Active,
			Metadata:    metadata,
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list Stripe products: %w", err)
	}

	return products, nil
}

// GetProduct retrieves a single product from Stripe.
func (p *Provider) GetProduct(ctx context.Context, productID string) (*types.ProviderProduct, error) {
	prod, err := product.Get(productID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe product: %w", err)
	}

	// Convert metadata
	metadata := make(map[string]string)
	maps.Copy(metadata, prod.Metadata)

	return &types.ProviderProduct{
		ID:          prod.ID,
		Name:        prod.Name,
		Description: prod.Description,
		Active:      prod.Active,
		Metadata:    metadata,
	}, nil
}

// ListPrices lists all prices for a product from Stripe.
func (p *Provider) ListPrices(ctx context.Context, productID string) ([]*types.ProviderPrice, error) {
	var prices []*types.ProviderPrice

	params := &stripe.PriceListParams{}
	params.Product = stripe.String(productID)
	params.Limit = stripe.Int64(100)

	iter := price.List(params)
	for iter.Next() {
		pr := iter.Price()

		// Convert metadata
		metadata := make(map[string]string)
		maps.Copy(metadata, pr.Metadata)

		providerPrice := &types.ProviderPrice{
			ID:         pr.ID,
			ProductID:  pr.Product.ID,
			Active:     pr.Active,
			Currency:   string(pr.Currency),
			UnitAmount: pr.UnitAmount,
			Metadata:   metadata,
		}

		// Add recurring info if present
		if pr.Recurring != nil {
			providerPrice.Recurring = &types.PriceRecurring{
				Interval:      string(pr.Recurring.Interval),
				IntervalCount: int(pr.Recurring.IntervalCount),
			}
		}

		prices = append(prices, providerPrice)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list Stripe prices: %w", err)
	}

	return prices, nil
}

// HandleWebhook handles a Stripe webhook.
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook: %w", err)
	}

	// Parse the raw JSON into a map
	var data map[string]any
	if err := json.Unmarshal(event.Data.Raw, &data); err != nil {
		data = make(map[string]any)
	}

	return &types.WebhookEvent{
		ID:        event.ID,
		Type:      string(event.Type),
		Data:      data,
		Timestamp: event.Created,
	}, nil
}
