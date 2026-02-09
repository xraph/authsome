package core

import (
	"time"

	"github.com/rs/xid"
)

// TaxType represents the type of tax.
type TaxType string

const (
	TaxTypeVAT         TaxType = "vat"         // Value Added Tax (EU, UK)
	TaxTypeGST         TaxType = "gst"         // Goods and Services Tax (AU, NZ, IN)
	TaxTypeSalesTax    TaxType = "sales_tax"   // Sales Tax (US)
	TaxTypeHST         TaxType = "hst"         // Harmonized Sales Tax (Canada)
	TaxTypePST         TaxType = "pst"         // Provincial Sales Tax (Canada)
	TaxTypeConsumption TaxType = "consumption" // Consumption Tax (JP)
	TaxTypeCustom      TaxType = "custom"      // Custom tax type
)

// TaxBehavior defines how tax is applied to prices.
type TaxBehavior string

const (
	TaxBehaviorExclusive TaxBehavior = "exclusive" // Tax added on top of price
	TaxBehaviorInclusive TaxBehavior = "inclusive" // Tax included in price
)

// TaxRate represents a tax rate configuration.
type TaxRate struct {
	ID          xid.ID      `json:"id"`
	AppID       xid.ID      `json:"appId"`
	Name        string      `json:"name"`        // Display name (e.g., "US Sales Tax", "UK VAT")
	Description string      `json:"description"` // Optional description
	Type        TaxType     `json:"type"`        // Type of tax
	Percentage  float64     `json:"percentage"`  // Tax rate as percentage (e.g., 20.0 for 20%)
	Country     string      `json:"country"`     // ISO 3166-1 alpha-2 country code
	State       string      `json:"state"`       // State/province code (for US, CA)
	City        string      `json:"city"`        // City (for local taxes)
	PostalCode  string      `json:"postalCode"`  // Postal/ZIP code pattern
	Behavior    TaxBehavior `json:"behavior"`    // Exclusive or inclusive
	IsDefault   bool        `json:"isDefault"`   // Default rate for the region
	IsActive    bool        `json:"isActive"`    // Is this rate active

	// Provider integration
	ProviderTaxRateID string `json:"providerTaxRateId"` // Tax rate ID in payment provider

	// Timestamps
	ValidFrom  time.Time  `json:"validFrom"`  // When rate becomes valid
	ValidUntil *time.Time `json:"validUntil"` // When rate expires
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// TaxExemption represents a tax exemption for an organization.
type TaxExemption struct {
	ID             xid.ID     `json:"id"`
	AppID          xid.ID     `json:"appId"`
	OrganizationID xid.ID     `json:"organizationId"`
	Type           string     `json:"type"`        // Exemption type (nonprofit, government, resale)
	Certificate    string     `json:"certificate"` // Exemption certificate number
	Country        string     `json:"country"`     // Country where exemption applies
	State          string     `json:"state"`       // State where exemption applies
	VerifiedAt     *time.Time `json:"verifiedAt"`  // When exemption was verified
	ExpiresAt      *time.Time `json:"expiresAt"`   // When exemption expires
	IsActive       bool       `json:"isActive"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// TaxCalculation represents a calculated tax amount.
type TaxCalculation struct {
	TaxRateID     xid.ID  `json:"taxRateId"`
	TaxRateName   string  `json:"taxRateName"`
	TaxType       TaxType `json:"taxType"`
	Percentage    float64 `json:"percentage"`
	TaxableAmount int64   `json:"taxableAmount"` // Amount before tax
	TaxAmount     int64   `json:"taxAmount"`     // Calculated tax amount
	TotalAmount   int64   `json:"totalAmount"`   // Amount after tax
	Currency      string  `json:"currency"`
	IsExempt      bool    `json:"isExempt"`
	ExemptionID   *xid.ID `json:"exemptionId"`
}

// TaxSummary represents a summary of taxes on an invoice.
type TaxSummary struct {
	Calculations []TaxCalculation `json:"calculations"`
	TotalTax     int64            `json:"totalTax"`
	Currency     string           `json:"currency"`
}

// TaxBillingAddress represents a customer's billing address for tax purposes
// Note: Uses BillingAddress from payment.go for actual address storage
// This alias is for documentation purposes.
type TaxBillingAddress = BillingAddress

// TaxIDType represents the type of tax ID.
type TaxIDType string

const (
	TaxIDTypeVAT    TaxIDType = "eu_vat"
	TaxIDTypeGSTIN  TaxIDType = "in_gst"
	TaxIDTypeABN    TaxIDType = "au_abn"
	TaxIDTypeGST_NZ TaxIDType = "nz_gst"
	TaxIDTypeEIN    TaxIDType = "us_ein"
	TaxIDTypeGST_CA TaxIDType = "ca_gst"
	TaxIDTypeBN     TaxIDType = "ca_bn"
	TaxIDTypeCNPJ   TaxIDType = "br_cnpj"
	TaxIDTypeCPF    TaxIDType = "br_cpf"
	TaxIDTypeCustom TaxIDType = "custom"
)

// CustomerTaxID represents a customer's tax identification.
type CustomerTaxID struct {
	ID             xid.ID     `json:"id"`
	AppID          xid.ID     `json:"appId"`
	OrganizationID xid.ID     `json:"organizationId"`
	Type           TaxIDType  `json:"type"`
	Value          string     `json:"value"`
	Country        string     `json:"country"`
	VerifiedAt     *time.Time `json:"verifiedAt"`
	IsValid        bool       `json:"isValid"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// CreateTaxRateRequest is used to create a new tax rate.
type CreateTaxRateRequest struct {
	Name        string      `json:"name"        validate:"required"`
	Description string      `json:"description"`
	Type        TaxType     `json:"type"        validate:"required"`
	Percentage  float64     `json:"percentage"  validate:"required,gte=0,lte=100"`
	Country     string      `json:"country"     validate:"required,len=2"`
	State       string      `json:"state"`
	City        string      `json:"city"`
	PostalCode  string      `json:"postalCode"`
	Behavior    TaxBehavior `json:"behavior"    validate:"required"`
	IsDefault   bool        `json:"isDefault"`
	ValidFrom   time.Time   `json:"validFrom"`
}

// UpdateTaxRateRequest is used to update a tax rate.
type UpdateTaxRateRequest struct {
	Name        *string      `json:"name"`
	Description *string      `json:"description"`
	Percentage  *float64     `json:"percentage"`
	Behavior    *TaxBehavior `json:"behavior"`
	IsDefault   *bool        `json:"isDefault"`
	IsActive    *bool        `json:"isActive"`
	ValidUntil  *time.Time   `json:"validUntil"`
}

// CalculateTaxRequest is used to calculate tax for an amount.
type CalculateTaxRequest struct {
	Amount         int64           `json:"amount"         validate:"required"`
	Currency       string          `json:"currency"       validate:"required,len=3"`
	BillingAddress *BillingAddress `json:"billingAddress"`
	OrganizationID *xid.ID         `json:"organizationId"`
	ProductType    string          `json:"productType"` // For product-specific tax rules
}

// CreateTaxExemptionRequest is used to create a tax exemption.
type CreateTaxExemptionRequest struct {
	OrganizationID xid.ID     `json:"organizationId" validate:"required"`
	Type           string     `json:"type"           validate:"required"`
	Certificate    string     `json:"certificate"    validate:"required"`
	Country        string     `json:"country"        validate:"required,len=2"`
	State          string     `json:"state"`
	ExpiresAt      *time.Time `json:"expiresAt"`
}

// CreateTaxIDRequest is used to create a customer tax ID.
type CreateTaxIDRequest struct {
	OrganizationID xid.ID    `json:"organizationId" validate:"required"`
	Type           TaxIDType `json:"type"           validate:"required"`
	Value          string    `json:"value"          validate:"required"`
	Country        string    `json:"country"        validate:"required,len=2"`
}

// VATValidationResult represents the result of VAT validation.
type VATValidationResult struct {
	Valid       bool   `json:"valid"`
	CountryCode string `json:"countryCode"`
	VATNumber   string `json:"vatNumber"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Message     string `json:"message"`
}
