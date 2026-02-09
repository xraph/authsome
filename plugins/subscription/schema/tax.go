package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SubscriptionTaxRate represents a tax rate in the database.
type SubscriptionTaxRate struct {
	bun.BaseModel `bun:"table:subscription_tax_rates,alias:str"`

	ID                xid.ID     `bun:"id,pk,type:char(20)"`
	AppID             xid.ID     `bun:"app_id,notnull,type:char(20)"`
	Name              string     `bun:"name,notnull"`
	Description       string     `bun:"description"`
	Type              string     `bun:"type,notnull"` // vat, gst, sales_tax, etc.
	Percentage        float64    `bun:"percentage,notnull"`
	Country           string     `bun:"country,notnull"`
	State             string     `bun:"state"`
	City              string     `bun:"city"`
	PostalCode        string     `bun:"postal_code"`
	Behavior          string     `bun:"behavior,notnull,default:'exclusive'"` // exclusive, inclusive
	IsDefault         bool       `bun:"is_default,notnull,default:false"`
	IsActive          bool       `bun:"is_active,notnull,default:true"`
	ProviderTaxRateID string     `bun:"provider_tax_rate_id"`
	ValidFrom         time.Time  `bun:"valid_from,notnull"`
	ValidUntil        *time.Time `bun:"valid_until"`
	CreatedAt         time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt         time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionTaxExemption represents a tax exemption in the database.
type SubscriptionTaxExemption struct {
	bun.BaseModel `bun:"table:subscription_tax_exemptions,alias:ste"`

	ID             xid.ID     `bun:"id,pk,type:char(20)"`
	AppID          xid.ID     `bun:"app_id,notnull,type:char(20)"`
	OrganizationID xid.ID     `bun:"organization_id,notnull,type:char(20)"`
	Type           string     `bun:"type,notnull"` // nonprofit, government, resale
	Certificate    string     `bun:"certificate,notnull"`
	Country        string     `bun:"country,notnull"`
	State          string     `bun:"state"`
	VerifiedAt     *time.Time `bun:"verified_at"`
	ExpiresAt      *time.Time `bun:"expires_at"`
	IsActive       bool       `bun:"is_active,notnull,default:true"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionCustomerTaxID represents a customer's tax ID in the database.
type SubscriptionCustomerTaxID struct {
	bun.BaseModel `bun:"table:subscription_customer_tax_ids,alias:sctid"`

	ID             xid.ID     `bun:"id,pk,type:char(20)"`
	AppID          xid.ID     `bun:"app_id,notnull,type:char(20)"`
	OrganizationID xid.ID     `bun:"organization_id,notnull,type:char(20)"`
	Type           string     `bun:"type,notnull"` // eu_vat, in_gst, etc.
	Value          string     `bun:"value,notnull"`
	Country        string     `bun:"country,notnull"`
	VerifiedAt     *time.Time `bun:"verified_at"`
	IsValid        bool       `bun:"is_valid,notnull,default:false"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}
