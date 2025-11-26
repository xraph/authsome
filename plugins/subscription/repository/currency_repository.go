package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// CurrencyRepository defines the interface for currency operations
type CurrencyRepository interface {
	// Currency operations
	ListCurrencies(ctx context.Context) ([]*core.SupportedCurrency, error)
	GetCurrency(ctx context.Context, code string) (*core.SupportedCurrency, error)
	CreateCurrency(ctx context.Context, currency *core.SupportedCurrency) error
	UpdateCurrency(ctx context.Context, currency *core.SupportedCurrency) error
	
	// Exchange rate operations
	CreateExchangeRate(ctx context.Context, rate *core.ExchangeRate) error
	GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*core.ExchangeRate, error)
	GetExchangeRateAt(ctx context.Context, fromCurrency, toCurrency string, at time.Time) (*core.ExchangeRate, error)
	ListExchangeRates(ctx context.Context, appID xid.ID) ([]*core.ExchangeRate, error)
	UpdateExchangeRate(ctx context.Context, rate *core.ExchangeRate) error
	DeleteExchangeRate(ctx context.Context, id xid.ID) error
}

// currencyRepository implements CurrencyRepository using Bun
type currencyRepository struct {
	db *bun.DB
}

// NewCurrencyRepository creates a new currency repository
func NewCurrencyRepository(db *bun.DB) CurrencyRepository {
	return &currencyRepository{db: db}
}

// ListCurrencies returns all supported currencies
func (r *currencyRepository) ListCurrencies(ctx context.Context) ([]*core.SupportedCurrency, error) {
	var currencies []schema.SubscriptionCurrency
	err := r.db.NewSelect().
		Model(&currencies).
		Where("is_active = ?", true).
		Order("is_default DESC", "code ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.SupportedCurrency, len(currencies))
	for i, c := range currencies {
		result[i] = schemaToCurrency(&c)
	}
	return result, nil
}

// GetCurrency returns a currency by code
func (r *currencyRepository) GetCurrency(ctx context.Context, code string) (*core.SupportedCurrency, error) {
	var currency schema.SubscriptionCurrency
	err := r.db.NewSelect().
		Model(&currency).
		Where("code = ?", code).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToCurrency(&currency), nil
}

// CreateCurrency creates a new currency
func (r *currencyRepository) CreateCurrency(ctx context.Context, currency *core.SupportedCurrency) error {
	model := currencyToSchema(currency)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// UpdateCurrency updates a currency
func (r *currencyRepository) UpdateCurrency(ctx context.Context, currency *core.SupportedCurrency) error {
	model := currencyToSchema(currency)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// CreateExchangeRate creates a new exchange rate
func (r *currencyRepository) CreateExchangeRate(ctx context.Context, rate *core.ExchangeRate) error {
	model := exchangeRateToSchema(rate)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetExchangeRate returns the current exchange rate
func (r *currencyRepository) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*core.ExchangeRate, error) {
	var rate schema.SubscriptionExchangeRate
	err := r.db.NewSelect().
		Model(&rate).
		Where("from_currency = ?", fromCurrency).
		Where("to_currency = ?", toCurrency).
		Where("valid_from <= ?", time.Now()).
		Where("(valid_until IS NULL OR valid_until > ?)", time.Now()).
		Order("valid_from DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToExchangeRate(&rate), nil
}

// GetExchangeRateAt returns the exchange rate at a specific time
func (r *currencyRepository) GetExchangeRateAt(ctx context.Context, fromCurrency, toCurrency string, at time.Time) (*core.ExchangeRate, error) {
	var rate schema.SubscriptionExchangeRate
	err := r.db.NewSelect().
		Model(&rate).
		Where("from_currency = ?", fromCurrency).
		Where("to_currency = ?", toCurrency).
		Where("valid_from <= ?", at).
		Where("(valid_until IS NULL OR valid_until > ?)", at).
		Order("valid_from DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToExchangeRate(&rate), nil
}

// ListExchangeRates returns all exchange rates for an app
func (r *currencyRepository) ListExchangeRates(ctx context.Context, appID xid.ID) ([]*core.ExchangeRate, error) {
	var rates []schema.SubscriptionExchangeRate
	err := r.db.NewSelect().
		Model(&rates).
		Where("app_id = ?", appID).
		Where("(valid_until IS NULL OR valid_until > ?)", time.Now()).
		Order("from_currency ASC", "to_currency ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.ExchangeRate, len(rates))
	for i, r := range rates {
		result[i] = schemaToExchangeRate(&r)
	}
	return result, nil
}

// UpdateExchangeRate updates an exchange rate
func (r *currencyRepository) UpdateExchangeRate(ctx context.Context, rate *core.ExchangeRate) error {
	model := exchangeRateToSchema(rate)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteExchangeRate deletes an exchange rate
func (r *currencyRepository) DeleteExchangeRate(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionExchangeRate)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// Helper functions

func schemaToCurrency(s *schema.SubscriptionCurrency) *core.SupportedCurrency {
	return &core.SupportedCurrency{
		ID:            s.ID,
		Code:          s.Code,
		Name:          s.Name,
		Symbol:        s.Symbol,
		DecimalPlaces: s.DecimalPlaces,
		IsDefault:     s.IsDefault,
		IsActive:      s.IsActive,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func currencyToSchema(c *core.SupportedCurrency) *schema.SubscriptionCurrency {
	return &schema.SubscriptionCurrency{
		ID:            c.ID,
		Code:          c.Code,
		Name:          c.Name,
		Symbol:        c.Symbol,
		DecimalPlaces: c.DecimalPlaces,
		IsDefault:     c.IsDefault,
		IsActive:      c.IsActive,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func schemaToExchangeRate(s *schema.SubscriptionExchangeRate) *core.ExchangeRate {
	return &core.ExchangeRate{
		ID:           s.ID,
		AppID:        s.AppID,
		FromCurrency: s.FromCurrency,
		ToCurrency:   s.ToCurrency,
		Rate:         s.Rate,
		ValidFrom:    s.ValidFrom,
		ValidUntil:   s.ValidUntil,
		Source:       s.Source,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func exchangeRateToSchema(r *core.ExchangeRate) *schema.SubscriptionExchangeRate {
	return &schema.SubscriptionExchangeRate{
		ID:           r.ID,
		AppID:        r.AppID,
		FromCurrency: r.FromCurrency,
		ToCurrency:   r.ToCurrency,
		Rate:         r.Rate,
		ValidFrom:    r.ValidFrom,
		ValidUntil:   r.ValidUntil,
		Source:       r.Source,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

