package core

import (
	"time"

	"github.com/rs/xid"
)

// PaymentMethod represents a stored payment method
type PaymentMethod struct {
	ID                  xid.ID            `json:"id"`
	OrganizationID      xid.ID            `json:"organizationId"`
	Type                PaymentMethodType `json:"type"`              // card, bank_account, etc.
	IsDefault           bool              `json:"isDefault"`         // Is this the default payment method
	ProviderMethodID    string            `json:"providerMethodId"`  // Stripe PaymentMethod ID
	ProviderCustomerID  string            `json:"providerCustomerId"` // Stripe Customer ID
	
	// Card-specific fields (when Type == PaymentMethodCard)
	CardBrand           string `json:"cardBrand"`           // visa, mastercard, etc.
	CardLast4           string `json:"cardLast4"`           // Last 4 digits
	CardExpMonth        int    `json:"cardExpMonth"`        // Expiration month
	CardExpYear         int    `json:"cardExpYear"`         // Expiration year
	CardFunding         string `json:"cardFunding"`         // credit, debit, prepaid
	
	// Bank account specific fields (when Type == PaymentMethodBankAccount)
	BankName            string `json:"bankName"`
	BankLast4           string `json:"bankLast4"`
	BankRoutingNumber   string `json:"bankRoutingNumber"` // Masked
	BankAccountType     string `json:"bankAccountType"`   // checking, savings
	
	// Billing details
	BillingName         string `json:"billingName"`
	BillingEmail        string `json:"billingEmail"`
	BillingPhone        string `json:"billingPhone"`
	BillingAddressLine1 string `json:"billingAddressLine1"`
	BillingAddressLine2 string `json:"billingAddressLine2"`
	BillingCity         string `json:"billingCity"`
	BillingState        string `json:"billingState"`
	BillingPostalCode   string `json:"billingPostalCode"`
	BillingCountry      string `json:"billingCountry"`
	
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// NewPaymentMethod creates a new PaymentMethod with default values
func NewPaymentMethod(orgID xid.ID, methodType PaymentMethodType) *PaymentMethod {
	now := time.Now()
	return &PaymentMethod{
		ID:             xid.New(),
		OrganizationID: orgID,
		Type:           methodType,
		IsDefault:      false,
		Metadata:       make(map[string]any),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsCard returns true if this is a card payment method
func (pm *PaymentMethod) IsCard() bool {
	return pm.Type == PaymentMethodCard
}

// IsBankAccount returns true if this is a bank account
func (pm *PaymentMethod) IsBankAccount() bool {
	return pm.Type == PaymentMethodBankAccount
}

// DisplayName returns a user-friendly display name for the payment method
func (pm *PaymentMethod) DisplayName() string {
	switch pm.Type {
	case PaymentMethodCard:
		brand := pm.CardBrand
		if brand == "" {
			brand = "Card"
		}
		return brand + " •••• " + pm.CardLast4
	case PaymentMethodBankAccount:
		name := pm.BankName
		if name == "" {
			name = "Bank"
		}
		return name + " •••• " + pm.BankLast4
	case PaymentMethodSepaDebit:
		return "SEPA •••• " + pm.BankLast4
	default:
		return "Payment Method"
	}
}

// IsExpired returns true if the card is expired
func (pm *PaymentMethod) IsExpired() bool {
	if pm.Type != PaymentMethodCard {
		return false
	}
	now := time.Now()
	// Card is expired if we're past the end of the expiration month
	expirationTime := time.Date(pm.CardExpYear, time.Month(pm.CardExpMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	return now.After(expirationTime)
}

// WillExpireSoon returns true if the card will expire within the given days
func (pm *PaymentMethod) WillExpireSoon(days int) bool {
	if pm.Type != PaymentMethodCard {
		return false
	}
	expirationTime := time.Date(pm.CardExpYear, time.Month(pm.CardExpMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	warningTime := time.Now().AddDate(0, 0, days)
	return warningTime.After(expirationTime) && !pm.IsExpired()
}

// SetupIntentResult represents the result of creating a setup intent
type SetupIntentResult struct {
	ClientSecret     string `json:"clientSecret"`     // For client-side confirmation
	SetupIntentID    string `json:"setupIntentId"`    // Stripe SetupIntent ID
	PublishableKey   string `json:"publishableKey"`   // Stripe publishable key
}

// AddPaymentMethodRequest represents a request to add a payment method
type AddPaymentMethodRequest struct {
	OrganizationID   xid.ID            `json:"organizationId" validate:"required"`
	Type             PaymentMethodType `json:"type" validate:"required"`
	SetAsDefault     bool              `json:"setAsDefault"`
	BillingName      string            `json:"billingName"`
	BillingEmail     string            `json:"billingEmail" validate:"omitempty,email"`
	BillingPhone     string            `json:"billingPhone"`
	BillingAddress   *BillingAddress   `json:"billingAddress"`
}

// BillingAddress represents a billing address
type BillingAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country" validate:"len=2"` // ISO 3166-1 alpha-2
}

// Customer represents the billing customer for an organization
type Customer struct {
	ID                 xid.ID         `json:"id"`
	OrganizationID     xid.ID         `json:"organizationId"`
	ProviderCustomerID string         `json:"providerCustomerId"` // Stripe Customer ID
	Email              string         `json:"email"`
	Name               string         `json:"name"`
	Phone              string         `json:"phone"`
	TaxID              string         `json:"taxId"`              // VAT number, etc.
	TaxExempt          bool           `json:"taxExempt"`
	Currency           string         `json:"currency"`           // Preferred currency
	Balance            int64          `json:"balance"`            // Account balance in cents
	DefaultPaymentID   *xid.ID        `json:"defaultPaymentId"`   // Default payment method
	BillingAddress     *BillingAddress `json:"billingAddress"`
	Metadata           map[string]any `json:"metadata"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

// NewCustomer creates a new Customer with default values
func NewCustomer(orgID xid.ID, email, name string) *Customer {
	now := time.Now()
	return &Customer{
		ID:             xid.New(),
		OrganizationID: orgID,
		Email:          email,
		Name:           name,
		Currency:       DefaultCurrency,
		Balance:        0,
		Metadata:       make(map[string]any),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateCustomerRequest represents a request to create a customer
type CreateCustomerRequest struct {
	OrganizationID xid.ID          `json:"organizationId" validate:"required"`
	Email          string          `json:"email" validate:"required,email"`
	Name           string          `json:"name" validate:"required"`
	Phone          string          `json:"phone"`
	TaxID          string          `json:"taxId"`
	TaxExempt      bool            `json:"taxExempt"`
	BillingAddress *BillingAddress `json:"billingAddress"`
	Metadata       map[string]any  `json:"metadata"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	Email          *string         `json:"email,omitempty" validate:"omitempty,email"`
	Name           *string         `json:"name,omitempty"`
	Phone          *string         `json:"phone,omitempty"`
	TaxID          *string         `json:"taxId,omitempty"`
	TaxExempt      *bool           `json:"taxExempt,omitempty"`
	BillingAddress *BillingAddress `json:"billingAddress,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

