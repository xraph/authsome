package core

import (
	"time"

	"github.com/rs/xid"
)

// SupportedCurrency represents a currency supported by the system
type SupportedCurrency struct {
	ID            xid.ID    `json:"id"`
	Code          string    `json:"code"`          // ISO 4217 code (USD, EUR, GBP, etc.)
	Name          string    `json:"name"`          // Display name
	Symbol        string    `json:"symbol"`        // Currency symbol ($, €, £)
	DecimalPlaces int       `json:"decimalPlaces"` // Number of decimal places (2 for most, 0 for JPY)
	IsDefault     bool      `json:"isDefault"`     // Is this the default currency
	IsActive      bool      `json:"isActive"`      // Is this currency currently active
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ExchangeRate represents a currency exchange rate
type ExchangeRate struct {
	ID           xid.ID     `json:"id"`
	AppID        xid.ID     `json:"appId"`
	FromCurrency string     `json:"fromCurrency"` // Source currency code
	ToCurrency   string     `json:"toCurrency"`   // Target currency code
	Rate         float64    `json:"rate"`         // Exchange rate (multiply by this to convert)
	ValidFrom    time.Time  `json:"validFrom"`    // When this rate becomes valid
	ValidUntil   *time.Time `json:"validUntil"`   // When this rate expires (nil = current)
	Source       string     `json:"source"`       // Source of the rate (manual, api, etc.)
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// CurrencyAmount represents an amount in a specific currency
type CurrencyAmount struct {
	Amount       int64  `json:"amount"`       // Amount in smallest unit (cents, pence, etc.)
	Currency     string `json:"currency"`     // Currency code
	DisplayValue string `json:"displayValue"` // Formatted display value
}

// MultiCurrencyPrice represents a price in multiple currencies
type MultiCurrencyPrice struct {
	BaseCurrency string                    `json:"baseCurrency"`
	BaseAmount   int64                     `json:"baseAmount"`
	Prices       map[string]CurrencyAmount `json:"prices"` // Currency code -> amount
}

// CreateExchangeRateRequest is used to create a new exchange rate
type CreateExchangeRateRequest struct {
	FromCurrency string    `json:"fromCurrency" validate:"required,len=3"`
	ToCurrency   string    `json:"toCurrency" validate:"required,len=3"`
	Rate         float64   `json:"rate" validate:"required,gt=0"`
	ValidFrom    time.Time `json:"validFrom"`
	Source       string    `json:"source"`
}

// UpdateExchangeRateRequest is used to update an exchange rate
type UpdateExchangeRateRequest struct {
	Rate       *float64   `json:"rate"`
	ValidUntil *time.Time `json:"validUntil"`
}

// ConvertCurrencyRequest is used to convert between currencies
type ConvertCurrencyRequest struct {
	Amount       int64  `json:"amount" validate:"required"`
	FromCurrency string `json:"fromCurrency" validate:"required,len=3"`
	ToCurrency   string `json:"toCurrency" validate:"required,len=3"`
}

// ConvertCurrencyResponse contains the conversion result
type ConvertCurrencyResponse struct {
	OriginalAmount    int64     `json:"originalAmount"`
	OriginalCurrency  string    `json:"originalCurrency"`
	ConvertedAmount   int64     `json:"convertedAmount"`
	ConvertedCurrency string    `json:"convertedCurrency"`
	ExchangeRate      float64   `json:"exchangeRate"`
	ConvertedAt       time.Time `json:"convertedAt"`
}

// Common currency codes
const (
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
	CurrencyGBP = "GBP"
	CurrencyJPY = "JPY"
	CurrencyCAD = "CAD"
	CurrencyAUD = "AUD"
	CurrencyCHF = "CHF"
	CurrencyCNY = "CNY"
	CurrencyINR = "INR"
	CurrencyBRL = "BRL"
)

// DefaultCurrencies returns the default supported currencies
func DefaultCurrencies() []SupportedCurrency {
	return []SupportedCurrency{
		{Code: CurrencyUSD, Name: "US Dollar", Symbol: "$", DecimalPlaces: 2, IsDefault: true, IsActive: true},
		{Code: CurrencyEUR, Name: "Euro", Symbol: "€", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyGBP, Name: "British Pound", Symbol: "£", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyJPY, Name: "Japanese Yen", Symbol: "¥", DecimalPlaces: 0, IsActive: true},
		{Code: CurrencyCAD, Name: "Canadian Dollar", Symbol: "CA$", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyAUD, Name: "Australian Dollar", Symbol: "A$", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyCHF, Name: "Swiss Franc", Symbol: "CHF", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyCNY, Name: "Chinese Yuan", Symbol: "¥", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyINR, Name: "Indian Rupee", Symbol: "₹", DecimalPlaces: 2, IsActive: true},
		{Code: CurrencyBRL, Name: "Brazilian Real", Symbol: "R$", DecimalPlaces: 2, IsActive: true},
	}
}

// FormatAmount formats an amount for display
func FormatAmount(amount int64, currency string) string {
	currencies := DefaultCurrencies()
	for _, c := range currencies {
		if c.Code == currency {
			if c.DecimalPlaces == 0 {
				return c.Symbol + formatIntWithCommas(amount)
			}
			return c.Symbol + formatDecimal(amount, c.DecimalPlaces)
		}
	}
	// Default to 2 decimal places
	return currency + " " + formatDecimal(amount, 2)
}

func formatDecimal(amount int64, decimals int) string {
	if decimals == 0 {
		return formatIntWithCommas(amount)
	}
	divisor := int64(1)
	for i := 0; i < decimals; i++ {
		divisor *= 10
	}
	whole := amount / divisor
	frac := amount % divisor

	// Simple formatting without using fmt to avoid import
	result := formatIntWithCommas(whole) + "."
	fracStr := ""
	for i := 0; i < decimals; i++ {
		fracStr = string(rune('0'+frac%10)) + fracStr
		frac /= 10
	}
	return result + fracStr
}

func formatIntWithCommas(n int64) string {
	if n < 0 {
		return "-" + formatIntWithCommas(-n)
	}
	if n < 1000 {
		return intToString(n)
	}
	return formatIntWithCommas(n/1000) + "," + padLeft(intToString(n%1000), 3, '0')
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func padLeft(s string, length int, pad rune) string {
	for len(s) < length {
		s = string(pad) + s
	}
	return s
}
