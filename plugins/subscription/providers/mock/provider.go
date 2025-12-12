// Package mock provides a mock payment provider for testing.
package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers/types"
)

// Provider implements a mock payment provider for testing
type Provider struct {
	mu                 sync.RWMutex
	customers          map[string]*mockCustomer
	subscriptions      map[string]*mockSubscription
	subscriptionItems  map[string]*mockSubscriptionItem
	products           map[string]*mockProduct
	prices             map[string]*mockPrice
	invoices           map[string]*mockInvoice
	paymentMethods     map[string]*mockPaymentMethod
	features           map[string]*mockFeature
}

type mockProduct struct {
	ID          string
	Name        string
	Description string
	Active      bool
	Metadata    map[string]string
}

type mockCustomer struct {
	ID       string
	Email    string
	Name     string
	Metadata map[string]interface{}
}

type mockSubscription struct {
	ID                 string
	CustomerID         string
	PriceID            string
	Status             string
	Quantity           int
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
	TrialStart         *int64
	TrialEnd           *int64
	CancelAt           *int64
	Metadata           map[string]interface{}
}

type mockPrice struct {
	ID        string
	ProductID string
	Amount    int64
	Currency  string
	Interval  string
}

type mockSubscriptionItem struct {
	ID             string
	SubscriptionID string
	PriceID        string
	Quantity       int
	Metadata       map[string]interface{}
}

type mockFeature struct {
	ID        string
	Name      string
	LookupKey string
	Active    bool
	Metadata  map[string]interface{}
}

type mockInvoice struct {
	ID             string
	CustomerID     string
	SubscriptionID string
	Status         string
	Currency       string
	AmountDue      int64
	AmountPaid     int64
	Total          int64
	Subtotal       int64
	Tax            int64
	PeriodStart    int64
	PeriodEnd      int64
	PDFURL         string
	HostedURL      string
}

type mockPaymentMethod struct {
	ID           string
	CustomerID   string
	Type         string
	CardBrand    string
	CardLast4    string
	CardExpMonth int
	CardExpYear  int
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *Provider {
	return &Provider{
		customers:         make(map[string]*mockCustomer),
		subscriptions:     make(map[string]*mockSubscription),
		subscriptionItems: make(map[string]*mockSubscriptionItem),
		products:          make(map[string]*mockProduct),
		prices:            make(map[string]*mockPrice),
		invoices:          make(map[string]*mockInvoice),
		paymentMethods:    make(map[string]*mockPaymentMethod),
		features:          make(map[string]*mockFeature),
	}
}

// Ensure Provider implements types.PaymentProvider
var _ types.PaymentProvider = (*Provider)(nil)

func (p *Provider) Name() string {
	return "mock"
}

func (p *Provider) CreateCustomer(ctx context.Context, email, name string, metadata map[string]interface{}) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := "cus_mock_" + xid.New().String()
	p.customers[id] = &mockCustomer{
		ID:       id,
		Email:    email,
		Name:     name,
		Metadata: metadata,
	}
	return id, nil
}

func (p *Provider) UpdateCustomer(ctx context.Context, customerID, email, name string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.customers[customerID]; ok {
		c.Email = email
		c.Name = name
		c.Metadata = metadata
	}
	return nil
}

func (p *Provider) DeleteCustomer(ctx context.Context, customerID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.customers, customerID)
	return nil
}

func (p *Provider) SyncPlan(ctx context.Context, plan *core.Plan) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	productID := "prod_mock_" + plan.ID.String()
	priceID := "price_mock_" + plan.ID.String()

	// Store the product with metadata
	p.products[productID] = &mockProduct{
		ID:          productID,
		Name:        plan.Name,
		Description: plan.Description,
		Active:      plan.IsActive,
		Metadata: map[string]string{
			"authsome":         "true",
			"plan_id":          plan.ID.String(),
			"app_id":           plan.AppID.String(),
			"slug":             plan.Slug,
			"billing_pattern":  string(plan.BillingPattern),
			"billing_interval": string(plan.BillingInterval),
		},
	}

	p.prices[priceID] = &mockPrice{
		ID:        priceID,
		ProductID: productID,
		Amount:    plan.BasePrice,
		Currency:  plan.Currency,
		Interval:  string(plan.BillingInterval),
	}

	plan.ProviderPlanID = productID
	plan.ProviderPriceID = priceID
	return nil
}

func (p *Provider) SyncAddOn(ctx context.Context, addon *core.AddOn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	productID := "prod_addon_mock_" + addon.ID.String()
	priceID := "price_addon_mock_" + addon.ID.String()

	// Store the product
	p.products[productID] = &mockProduct{
		ID:          productID,
		Name:        addon.Name,
		Description: addon.Description,
		Active:      addon.IsActive,
		Metadata: map[string]string{
			"authsome":         "true",
			"addon_id":         addon.ID.String(),
			"app_id":           addon.AppID.String(),
			"slug":             addon.Slug,
			"billing_pattern":  string(addon.BillingPattern),
			"billing_interval": string(addon.BillingInterval),
		},
	}

	// Store the price
	p.prices[priceID] = &mockPrice{
		ID:        priceID,
		ProductID: productID,
		Amount:    addon.Price,
		Currency:  addon.Currency,
		Interval:  string(addon.BillingInterval),
	}

	addon.ProviderPriceID = priceID
	return nil
}

func (p *Provider) CreateSubscription(ctx context.Context, customerID, priceID string, quantity, trialDays int, metadata map[string]interface{}) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := "sub_mock_" + xid.New().String()
	now := time.Now().Unix()

	sub := &mockSubscription{
		ID:                 id,
		CustomerID:         customerID,
		PriceID:            priceID,
		Status:             "active",
		Quantity:           quantity,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now + 30*24*60*60, // 30 days
		Metadata:           metadata,
	}

	if trialDays > 0 {
		sub.Status = "trialing"
		trialEnd := now + int64(trialDays*24*60*60)
		sub.TrialStart = &now
		sub.TrialEnd = &trialEnd
	}

	p.subscriptions[id] = sub
	return id, nil
}

func (p *Provider) UpdateSubscription(ctx context.Context, subscriptionID, priceID string, quantity int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[subscriptionID]; ok {
		sub.PriceID = priceID
		sub.Quantity = quantity
	}
	return nil
}

func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[subscriptionID]; ok {
		if immediate {
			sub.Status = "canceled"
		} else {
			cancelAt := sub.CurrentPeriodEnd
			sub.CancelAt = &cancelAt
		}
	}
	return nil
}

func (p *Provider) PauseSubscription(ctx context.Context, subscriptionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[subscriptionID]; ok {
		sub.Status = "paused"
	}
	return nil
}

func (p *Provider) ResumeSubscription(ctx context.Context, subscriptionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[subscriptionID]; ok {
		sub.Status = "active"
	}
	return nil
}

func (p *Provider) GetSubscription(ctx context.Context, subscriptionID string) (*types.ProviderSubscription, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sub, ok := p.subscriptions[subscriptionID]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}

	return &types.ProviderSubscription{
		ID:                 sub.ID,
		CustomerID:         sub.CustomerID,
		Status:             sub.Status,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		TrialStart:         sub.TrialStart,
		TrialEnd:           sub.TrialEnd,
		CancelAt:           sub.CancelAt,
		PriceID:            sub.PriceID,
		Quantity:           sub.Quantity,
		Metadata:           sub.Metadata,
	}, nil
}

// AddSubscriptionItem adds an item to a subscription
func (p *Provider) AddSubscriptionItem(ctx context.Context, subscriptionID string, priceID string, quantity int) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.subscriptions[subscriptionID]; !ok {
		return "", fmt.Errorf("subscription not found")
	}

	itemID := "si_mock_" + xid.New().String()
	// In a real implementation, we'd track subscription items
	// For mock, just return the ID
	return itemID, nil
}

// RemoveSubscriptionItem removes an item from a subscription
func (p *Provider) RemoveSubscriptionItem(ctx context.Context, subscriptionID string, itemID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.subscriptions[subscriptionID]; !ok {
		return fmt.Errorf("subscription not found")
	}

	// In mock, just return success
	return nil
}

// UpdateSubscriptionItem updates the quantity of a subscription item
func (p *Provider) UpdateSubscriptionItem(ctx context.Context, subscriptionID string, itemID string, quantity int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.subscriptions[subscriptionID]; !ok {
		return fmt.Errorf("subscription not found")
	}

	// In mock, just return success
	return nil
}

func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutSession, error) {
	id := "cs_mock_" + xid.New().String()
	return &types.CheckoutSession{
		ID:            id,
		URL:           "https://mock.stripe.com/checkout/" + id,
		CustomerID:    req.CustomerID,
		PaymentStatus: "unpaid",
	}, nil
}

func (p *Provider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "https://mock.stripe.com/portal/session", nil
}

func (p *Provider) CreateSetupIntent(ctx context.Context, customerID string) (*core.SetupIntentResult, error) {
	return &core.SetupIntentResult{
		ClientSecret:  "seti_mock_secret",
		SetupIntentID: "seti_mock_" + xid.New().String(),
	}, nil
}

func (p *Provider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*core.PaymentMethod, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pm, ok := p.paymentMethods[paymentMethodID]
	if !ok {
		// Return a mock card
		return &core.PaymentMethod{
			ProviderMethodID: paymentMethodID,
			Type:             core.PaymentMethodCard,
			CardBrand:        "visa",
			CardLast4:        "4242",
			CardExpMonth:     12,
			CardExpYear:      2025,
			CardFunding:      "credit",
		}, nil
	}

	return &core.PaymentMethod{
		ProviderMethodID: pm.ID,
		Type:             core.PaymentMethodType(pm.Type),
		CardBrand:        pm.CardBrand,
		CardLast4:        pm.CardLast4,
		CardExpMonth:     pm.CardExpMonth,
		CardExpYear:      pm.CardExpYear,
	}, nil
}

func (p *Provider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return nil
}

func (p *Provider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return nil
}

func (p *Provider) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return nil
}

func (p *Provider) ReportUsage(ctx context.Context, subscriptionItemID string, records []*core.UsageRecord) (string, error) {
	return "ur_mock_" + xid.New().String(), nil
}

func (p *Provider) GetInvoice(ctx context.Context, invoiceID string) (*types.ProviderInvoice, error) {
	return &types.ProviderInvoice{
		ID:         invoiceID,
		Status:     "paid",
		Currency:   "usd",
		AmountDue:  0,
		AmountPaid: 1000,
		Total:      1000,
	}, nil
}

func (p *Provider) GetInvoicePDF(ctx context.Context, invoiceID string) (string, error) {
	return "https://mock.stripe.com/invoices/" + invoiceID + ".pdf", nil
}

func (p *Provider) VoidInvoice(ctx context.Context, invoiceID string) error {
	return nil
}

// ListInvoices lists invoices for a customer
func (p *Provider) ListInvoices(ctx context.Context, customerID string, limit int) ([]*types.ProviderInvoice, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*types.ProviderInvoice
	count := 0
	for _, inv := range p.invoices {
		if inv.CustomerID == customerID {
			result = append(result, &types.ProviderInvoice{
				ID:             inv.ID,
				CustomerID:     inv.CustomerID,
				SubscriptionID: inv.SubscriptionID,
				Status:         inv.Status,
				Currency:       inv.Currency,
				AmountDue:      inv.AmountDue,
				AmountPaid:     inv.AmountPaid,
				Total:          inv.Total,
				Subtotal:       inv.Subtotal,
				Tax:            inv.Tax,
				PeriodStart:    inv.PeriodStart,
				PeriodEnd:      inv.PeriodEnd,
				PDFURL:         inv.PDFURL,
				HostedURL:      inv.HostedURL,
			})
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

// ListSubscriptionInvoices lists invoices for a subscription
func (p *Provider) ListSubscriptionInvoices(ctx context.Context, subscriptionID string, limit int) ([]*types.ProviderInvoice, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*types.ProviderInvoice
	count := 0
	for _, inv := range p.invoices {
		if inv.SubscriptionID == subscriptionID {
			result = append(result, &types.ProviderInvoice{
				ID:             inv.ID,
				CustomerID:     inv.CustomerID,
				SubscriptionID: inv.SubscriptionID,
				Status:         inv.Status,
				Currency:       inv.Currency,
				AmountDue:      inv.AmountDue,
				AmountPaid:     inv.AmountPaid,
				Total:          inv.Total,
				Subtotal:       inv.Subtotal,
				Tax:            inv.Tax,
				PeriodStart:    inv.PeriodStart,
				PeriodEnd:      inv.PeriodEnd,
				PDFURL:         inv.PDFURL,
				HostedURL:      inv.HostedURL,
			})
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

// SyncFeature syncs a feature to the mock provider
func (p *Provider) SyncFeature(ctx context.Context, feature *core.Feature) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	featureID := feature.ID.String()
	if feature.Metadata != nil {
		if providerID, ok := feature.Metadata["provider_feature_id"].(string); ok && providerID != "" {
			featureID = providerID
		}
	}
	
	// Generate mock provider ID if creating new
	if feature.ProviderFeatureID == "" {
		featureID = "mock_feat_" + feature.ID.String()
	} else {
		featureID = feature.ProviderFeatureID
	}

	p.features[featureID] = &mockFeature{
		ID:        featureID,
		Name:      feature.Name,
		LookupKey: feature.Key,
		Active:    true,
		Metadata:  feature.Metadata,
	}

	return featureID, nil
}

// ListProviderFeatures lists all features from the mock provider
func (p *Provider) ListProviderFeatures(ctx context.Context, productID string) ([]*types.ProviderFeature, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*types.ProviderFeature
	for _, f := range p.features {
		result = append(result, &types.ProviderFeature{
			ID:        f.ID,
			Name:      f.Name,
			LookupKey: f.LookupKey,
			Active:    f.Active,
			Metadata:  f.Metadata,
		})
	}
	return result, nil
}

// GetProviderFeature gets a specific feature from the mock provider
func (p *Provider) GetProviderFeature(ctx context.Context, featureID string) (*types.ProviderFeature, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	f, ok := p.features[featureID]
	if !ok {
		return nil, fmt.Errorf("feature not found")
	}

	return &types.ProviderFeature{
		ID:        f.ID,
		Name:      f.Name,
		LookupKey: f.LookupKey,
		Active:    f.Active,
		Metadata:  f.Metadata,
	}, nil
}

// DeleteProviderFeature deletes a feature from the mock provider
func (p *Provider) DeleteProviderFeature(ctx context.Context, featureID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.features, featureID)
	return nil
}

func (p *Provider) ListProducts(ctx context.Context) ([]*types.ProviderProduct, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var products []*types.ProviderProduct
	for _, prod := range p.products {
		products = append(products, &types.ProviderProduct{
			ID:          prod.ID,
			Name:        prod.Name,
			Description: prod.Description,
			Active:      prod.Active,
			Metadata:    prod.Metadata,
		})
	}
	return products, nil
}

func (p *Provider) GetProduct(ctx context.Context, productID string) (*types.ProviderProduct, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	prod, ok := p.products[productID]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	return &types.ProviderProduct{
		ID:          prod.ID,
		Name:        prod.Name,
		Description: prod.Description,
		Active:      prod.Active,
		Metadata:    prod.Metadata,
	}, nil
}

func (p *Provider) ListPrices(ctx context.Context, productID string) ([]*types.ProviderPrice, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var prices []*types.ProviderPrice
	for _, pr := range p.prices {
		if pr.ProductID == productID {
			providerPrice := &types.ProviderPrice{
				ID:         pr.ID,
				ProductID:  pr.ProductID,
				Active:     true,
				Currency:   pr.Currency,
				UnitAmount: pr.Amount,
			}
			if pr.Interval != "" && pr.Interval != "one_time" {
				providerPrice.Recurring = &types.PriceRecurring{
					Interval:      pr.Interval,
					IntervalCount: 1,
				}
			}
			prices = append(prices, providerPrice)
		}
	}
	return prices, nil
}

func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	return &types.WebhookEvent{
		ID:        "evt_mock_" + xid.New().String(),
		Type:      "test",
		Timestamp: time.Now().Unix(),
	}, nil
}
