package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SubscriptionCurrency represents a supported currency in the database.
type SubscriptionCurrency struct {
	bun.BaseModel `bun:"table:subscription_currencies,alias:sc"`

	ID            xid.ID    `bun:"id,pk,type:char(20)"`
	Code          string    `bun:"code,notnull,unique"`
	Name          string    `bun:"name,notnull"`
	Symbol        string    `bun:"symbol,notnull"`
	DecimalPlaces int       `bun:"decimal_places,notnull,default:2"`
	IsDefault     bool      `bun:"is_default,notnull,default:false"`
	IsActive      bool      `bun:"is_active,notnull,default:true"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionExchangeRate represents an exchange rate in the database.
type SubscriptionExchangeRate struct {
	bun.BaseModel `bun:"table:subscription_exchange_rates,alias:ser"`

	ID           xid.ID     `bun:"id,pk,type:char(20)"`
	AppID        xid.ID     `bun:"app_id,notnull,type:char(20)"`
	FromCurrency string     `bun:"from_currency,notnull"`
	ToCurrency   string     `bun:"to_currency,notnull"`
	Rate         float64    `bun:"rate,notnull"`
	ValidFrom    time.Time  `bun:"valid_from,notnull"`
	ValidUntil   *time.Time `bun:"valid_until"`
	Source       string     `bun:"source,notnull,default:'manual'"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}
