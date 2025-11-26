package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
)

// CurrencyService handles currency and exchange rate operations
type CurrencyService struct {
	repo repository.CurrencyRepository
}

// NewCurrencyService creates a new currency service
func NewCurrencyService(repo repository.CurrencyRepository) *CurrencyService {
	return &CurrencyService{repo: repo}
}

// ListCurrencies returns all supported currencies
func (s *CurrencyService) ListCurrencies(ctx context.Context) ([]*core.SupportedCurrency, error) {
	return s.repo.ListCurrencies(ctx)
}

// GetCurrency returns a currency by code
func (s *CurrencyService) GetCurrency(ctx context.Context, code string) (*core.SupportedCurrency, error) {
	return s.repo.GetCurrency(ctx, code)
}

// CreateCurrency creates a new supported currency
func (s *CurrencyService) CreateCurrency(ctx context.Context, currency *core.SupportedCurrency) error {
	if currency.ID.IsNil() {
		currency.ID = xid.New()
	}
	currency.CreatedAt = time.Now()
	currency.UpdatedAt = time.Now()
	return s.repo.CreateCurrency(ctx, currency)
}

// SetDefaultCurrency sets a currency as the default
func (s *CurrencyService) SetDefaultCurrency(ctx context.Context, code string) error {
	currency, err := s.repo.GetCurrency(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return fmt.Errorf("currency not found: %s", code)
	}
	
	currency.IsDefault = true
	return s.repo.UpdateCurrency(ctx, currency)
}

// CreateExchangeRate creates a new exchange rate
func (s *CurrencyService) CreateExchangeRate(ctx context.Context, req *core.CreateExchangeRateRequest, appID xid.ID) (*core.ExchangeRate, error) {
	rate := &core.ExchangeRate{
		ID:           xid.New(),
		AppID:        appID,
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         req.Rate,
		ValidFrom:    req.ValidFrom,
		Source:       req.Source,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	if rate.ValidFrom.IsZero() {
		rate.ValidFrom = time.Now()
	}
	if rate.Source == "" {
		rate.Source = "manual"
	}
	
	if err := s.repo.CreateExchangeRate(ctx, rate); err != nil {
		return nil, err
	}
	return rate, nil
}

// GetExchangeRate returns the current exchange rate between two currencies
func (s *CurrencyService) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*core.ExchangeRate, error) {
	if fromCurrency == toCurrency {
		return &core.ExchangeRate{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			Rate:         1.0,
		}, nil
	}
	return s.repo.GetExchangeRate(ctx, fromCurrency, toCurrency)
}

// ListExchangeRates returns all exchange rates for an app
func (s *CurrencyService) ListExchangeRates(ctx context.Context, appID xid.ID) ([]*core.ExchangeRate, error) {
	return s.repo.ListExchangeRates(ctx, appID)
}

// Convert converts an amount from one currency to another
func (s *CurrencyService) Convert(ctx context.Context, req *core.ConvertCurrencyRequest) (*core.ConvertCurrencyResponse, error) {
	if req.FromCurrency == req.ToCurrency {
		return &core.ConvertCurrencyResponse{
			OriginalAmount:    req.Amount,
			OriginalCurrency:  req.FromCurrency,
			ConvertedAmount:   req.Amount,
			ConvertedCurrency: req.ToCurrency,
			ExchangeRate:      1.0,
			ConvertedAt:       time.Now(),
		}, nil
	}
	
	rate, err := s.repo.GetExchangeRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, fmt.Errorf("exchange rate not found for %s to %s", req.FromCurrency, req.ToCurrency)
	}
	
	convertedAmount := int64(float64(req.Amount) * rate.Rate)
	
	return &core.ConvertCurrencyResponse{
		OriginalAmount:    req.Amount,
		OriginalCurrency:  req.FromCurrency,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: req.ToCurrency,
		ExchangeRate:      rate.Rate,
		ConvertedAt:       time.Now(),
	}, nil
}

// FormatAmount formats an amount for display
func (s *CurrencyService) FormatAmount(amount int64, currency string) string {
	return core.FormatAmount(amount, currency)
}

