package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// PaymentService handles payment method management
type PaymentService struct {
	repo         repository.PaymentMethodRepository
	customerRepo repository.CustomerRepository
	provider     providers.PaymentProvider
	eventRepo    repository.EventRepository
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	repo repository.PaymentMethodRepository,
	customerRepo repository.CustomerRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *PaymentService {
	return &PaymentService{
		repo:         repo,
		customerRepo: customerRepo,
		provider:     provider,
		eventRepo:    eventRepo,
	}
}

// CreateSetupIntent creates a setup intent for adding a payment method
func (s *PaymentService) CreateSetupIntent(ctx context.Context, orgID xid.ID) (*core.SetupIntentResult, error) {
	// Get or create customer
	customer, err := s.customerRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("customer not found - create customer first")
	}

	result, err := s.provider.CreateSetupIntent(ctx, customer.ProviderCustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to create setup intent: %w", err)
	}

	return result, nil
}

// AddPaymentMethod adds a payment method from provider
func (s *PaymentService) AddPaymentMethod(ctx context.Context, orgID xid.ID, providerMethodID string, setDefault bool) (*core.PaymentMethod, error) {
	customer, err := s.customerRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("customer not found")
	}

	// Get payment method details from provider
	pm, err := s.provider.GetPaymentMethod(ctx, providerMethodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}

	// Create schema
	now := time.Now()
	method := &schema.SubscriptionPaymentMethod{
		ID:                 xid.New(),
		OrganizationID:     orgID,
		Type:               string(pm.Type),
		IsDefault:          setDefault,
		ProviderMethodID:   providerMethodID,
		ProviderCustomerID: customer.ProviderCustomerID,
		CardBrand:          pm.CardBrand,
		CardLast4:          pm.CardLast4,
		CardExpMonth:       pm.CardExpMonth,
		CardExpYear:        pm.CardExpYear,
		CardFunding:        pm.CardFunding,
		Metadata:           make(map[string]interface{}),
	}
	method.CreatedAt = now
	method.UpdatedAt = now

	// Clear existing default if setting new default
	if setDefault {
		s.repo.ClearDefault(ctx, orgID)
	}

	if err := s.repo.Create(ctx, method); err != nil {
		return nil, fmt.Errorf("failed to save payment method: %w", err)
	}

	return s.schemaToCorePayment(method), nil
}

// RemovePaymentMethod removes a payment method
func (s *PaymentService) RemovePaymentMethod(ctx context.Context, id xid.ID) error {
	pm, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrPaymentMethodNotFound
	}

	if pm.IsDefault {
		return suberrors.ErrDefaultPaymentMethodDelete
	}

	// Detach from provider
	if err := s.provider.DetachPaymentMethod(ctx, pm.ProviderMethodID); err != nil {
		// Log but continue
	}

	return s.repo.Delete(ctx, id)
}

// SetDefaultPaymentMethod sets a payment method as default
func (s *PaymentService) SetDefaultPaymentMethod(ctx context.Context, orgID, paymentMethodID xid.ID) error {
	return s.repo.SetDefault(ctx, orgID, paymentMethodID)
}

// ListPaymentMethods lists all payment methods for an organization
func (s *PaymentService) ListPaymentMethods(ctx context.Context, orgID xid.ID) ([]*core.PaymentMethod, error) {
	methods, err := s.repo.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	result := make([]*core.PaymentMethod, len(methods))
	for i, m := range methods {
		result[i] = s.schemaToCorePayment(m)
	}

	return result, nil
}

// GetDefaultPaymentMethod gets the default payment method
func (s *PaymentService) GetDefaultPaymentMethod(ctx context.Context, orgID xid.ID) (*core.PaymentMethod, error) {
	pm, err := s.repo.GetDefault(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrPaymentMethodNotFound
	}
	return s.schemaToCorePayment(pm), nil
}

func (s *PaymentService) schemaToCorePayment(pm *schema.SubscriptionPaymentMethod) *core.PaymentMethod {
	return &core.PaymentMethod{
		ID:                 pm.ID,
		OrganizationID:     pm.OrganizationID,
		Type:               core.PaymentMethodType(pm.Type),
		IsDefault:          pm.IsDefault,
		ProviderMethodID:   pm.ProviderMethodID,
		ProviderCustomerID: pm.ProviderCustomerID,
		CardBrand:          pm.CardBrand,
		CardLast4:          pm.CardLast4,
		CardExpMonth:       pm.CardExpMonth,
		CardExpYear:        pm.CardExpYear,
		CardFunding:        pm.CardFunding,
		Metadata:           pm.Metadata,
		CreatedAt:          pm.CreatedAt,
		UpdatedAt:          pm.UpdatedAt,
	}
}

