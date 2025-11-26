package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
)

// TaxService handles tax rate and calculation operations
type TaxService struct {
	repo repository.TaxRepository
}

// NewTaxService creates a new tax service
func NewTaxService(repo repository.TaxRepository) *TaxService {
	return &TaxService{repo: repo}
}

// CreateTaxRate creates a new tax rate
func (s *TaxService) CreateTaxRate(ctx context.Context, appID xid.ID, req *core.CreateTaxRateRequest) (*core.TaxRate, error) {
	rate := &core.TaxRate{
		ID:          xid.New(),
		AppID:       appID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Percentage:  req.Percentage,
		Country:     req.Country,
		State:       req.State,
		City:        req.City,
		PostalCode:  req.PostalCode,
		Behavior:    req.Behavior,
		IsDefault:   req.IsDefault,
		IsActive:    true,
		ValidFrom:   req.ValidFrom,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if rate.ValidFrom.IsZero() {
		rate.ValidFrom = time.Now()
	}
	
	if err := s.repo.CreateTaxRate(ctx, rate); err != nil {
		return nil, fmt.Errorf("failed to create tax rate: %w", err)
	}
	return rate, nil
}

// GetTaxRate returns a tax rate by ID
func (s *TaxService) GetTaxRate(ctx context.Context, id xid.ID) (*core.TaxRate, error) {
	return s.repo.GetTaxRate(ctx, id)
}

// GetTaxRateForLocation returns the applicable tax rate for a location
func (s *TaxService) GetTaxRateForLocation(ctx context.Context, appID xid.ID, country, state string) (*core.TaxRate, error) {
	return s.repo.GetTaxRateByLocation(ctx, appID, country, state)
}

// ListTaxRates returns all tax rates for an app
func (s *TaxService) ListTaxRates(ctx context.Context, appID xid.ID, activeOnly bool) ([]*core.TaxRate, error) {
	return s.repo.ListTaxRates(ctx, appID, activeOnly)
}

// UpdateTaxRate updates a tax rate
func (s *TaxService) UpdateTaxRate(ctx context.Context, id xid.ID, req *core.UpdateTaxRateRequest) (*core.TaxRate, error) {
	rate, err := s.repo.GetTaxRate(ctx, id)
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, fmt.Errorf("tax rate not found")
	}
	
	if req.Name != nil {
		rate.Name = *req.Name
	}
	if req.Description != nil {
		rate.Description = *req.Description
	}
	if req.Percentage != nil {
		rate.Percentage = *req.Percentage
	}
	if req.Behavior != nil {
		rate.Behavior = *req.Behavior
	}
	if req.IsDefault != nil {
		rate.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		rate.IsActive = *req.IsActive
	}
	if req.ValidUntil != nil {
		rate.ValidUntil = req.ValidUntil
	}
	
	rate.UpdatedAt = time.Now()
	
	if err := s.repo.UpdateTaxRate(ctx, rate); err != nil {
		return nil, fmt.Errorf("failed to update tax rate: %w", err)
	}
	return rate, nil
}

// DeleteTaxRate deletes a tax rate
func (s *TaxService) DeleteTaxRate(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteTaxRate(ctx, id)
}

// CalculateTax calculates tax for an amount
func (s *TaxService) CalculateTax(ctx context.Context, appID xid.ID, req *core.CalculateTaxRequest) (*core.TaxCalculation, error) {
	// Check for exemption first
	if req.OrganizationID != nil && req.BillingAddress != nil {
		exemption, _ := s.repo.GetTaxExemptionByOrg(ctx, *req.OrganizationID, req.BillingAddress.Country)
		if exemption != nil && exemption.IsActive {
			return &core.TaxCalculation{
				TaxableAmount: req.Amount,
				TaxAmount:     0,
				TotalAmount:   req.Amount,
				Currency:      req.Currency,
				IsExempt:      true,
				ExemptionID:   &exemption.ID,
			}, nil
		}
	}
	
	// Get applicable tax rate
	var rate *core.TaxRate
	var err error
	if req.BillingAddress != nil {
		rate, err = s.repo.GetTaxRateByLocation(ctx, appID, req.BillingAddress.Country, req.BillingAddress.State)
		if err != nil {
			return nil, fmt.Errorf("failed to get tax rate: %w", err)
		}
	}
	
	if rate == nil {
		// No tax rate found - return zero tax
		return &core.TaxCalculation{
			TaxableAmount: req.Amount,
			TaxAmount:     0,
			TotalAmount:   req.Amount,
			Currency:      req.Currency,
			IsExempt:      false,
		}, nil
	}
	
	// Calculate tax
	taxAmount := int64(float64(req.Amount) * rate.Percentage / 100)
	totalAmount := req.Amount
	
	if rate.Behavior == core.TaxBehaviorExclusive {
		totalAmount = req.Amount + taxAmount
	} else {
		// Tax is inclusive - calculate the tax portion
		taxAmount = int64(float64(req.Amount) * (rate.Percentage / (100 + rate.Percentage)))
	}
	
	return &core.TaxCalculation{
		TaxRateID:     rate.ID,
		TaxRateName:   rate.Name,
		TaxType:       rate.Type,
		Percentage:    rate.Percentage,
		TaxableAmount: req.Amount,
		TaxAmount:     taxAmount,
		TotalAmount:   totalAmount,
		Currency:      req.Currency,
		IsExempt:      false,
	}, nil
}

// CreateTaxExemption creates a tax exemption for an organization
func (s *TaxService) CreateTaxExemption(ctx context.Context, appID xid.ID, req *core.CreateTaxExemptionRequest) (*core.TaxExemption, error) {
	exemption := &core.TaxExemption{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: req.OrganizationID,
		Type:           req.Type,
		Certificate:    req.Certificate,
		Country:        req.Country,
		State:          req.State,
		ExpiresAt:      req.ExpiresAt,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	if err := s.repo.CreateTaxExemption(ctx, exemption); err != nil {
		return nil, fmt.Errorf("failed to create tax exemption: %w", err)
	}
	return exemption, nil
}

// ListTaxExemptions lists tax exemptions for an organization
func (s *TaxService) ListTaxExemptions(ctx context.Context, orgID xid.ID) ([]*core.TaxExemption, error) {
	return s.repo.ListTaxExemptions(ctx, orgID)
}

// VerifyTaxExemption marks an exemption as verified
func (s *TaxService) VerifyTaxExemption(ctx context.Context, id xid.ID) error {
	exemption, err := s.repo.GetTaxExemption(ctx, id)
	if err != nil {
		return err
	}
	if exemption == nil {
		return fmt.Errorf("exemption not found")
	}
	
	now := time.Now()
	exemption.VerifiedAt = &now
	exemption.UpdatedAt = now
	
	return s.repo.UpdateTaxExemption(ctx, exemption)
}

// CreateCustomerTaxID creates a tax ID for an organization
func (s *TaxService) CreateCustomerTaxID(ctx context.Context, appID xid.ID, req *core.CreateTaxIDRequest) (*core.CustomerTaxID, error) {
	taxID := &core.CustomerTaxID{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: req.OrganizationID,
		Type:           req.Type,
		Value:          req.Value,
		Country:        req.Country,
		IsValid:        false, // Needs verification
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	if err := s.repo.CreateCustomerTaxID(ctx, taxID); err != nil {
		return nil, fmt.Errorf("failed to create tax ID: %w", err)
	}
	return taxID, nil
}

// ListCustomerTaxIDs lists tax IDs for an organization
func (s *TaxService) ListCustomerTaxIDs(ctx context.Context, orgID xid.ID) ([]*core.CustomerTaxID, error) {
	return s.repo.ListCustomerTaxIDs(ctx, orgID)
}

// ValidateVAT validates a VAT number (stub - would call VIES or similar)
func (s *TaxService) ValidateVAT(ctx context.Context, countryCode, vatNumber string) (*core.VATValidationResult, error) {
	// This is a stub - in production would call VIES API for EU VAT validation
	return &core.VATValidationResult{
		Valid:       true,
		CountryCode: countryCode,
		VATNumber:   vatNumber,
		Message:     "Validation not implemented - stub returns valid",
	}, nil
}

