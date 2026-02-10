package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// CustomerService handles billing customer management.
type CustomerService struct {
	repo      repository.CustomerRepository
	provider  providers.PaymentProvider
	eventRepo repository.EventRepository
}

// NewCustomerService creates a new customer service.
func NewCustomerService(
	repo repository.CustomerRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *CustomerService {
	return &CustomerService{
		repo:      repo,
		provider:  provider,
		eventRepo: eventRepo,
	}
}

// Create creates a new billing customer.
func (s *CustomerService) Create(ctx context.Context, req *core.CreateCustomerRequest) (*core.Customer, error) {
	// Check for existing
	existing, _ := s.repo.FindByOrganizationID(ctx, req.OrganizationID)
	if existing != nil {
		return nil, suberrors.ErrCustomerAlreadyExists
	}

	// Create in provider first
	providerID, err := s.provider.CreateCustomer(ctx, req.Email, req.Name, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer in provider: %w", err)
	}

	// Create local record
	now := time.Now()
	customer := &schema.SubscriptionCustomer{
		ID:                 xid.New(),
		OrganizationID:     req.OrganizationID,
		ProviderCustomerID: providerID,
		Email:              req.Email,
		Name:               req.Name,
		Phone:              req.Phone,
		TaxID:              req.TaxID,
		TaxExempt:          req.TaxExempt,
		Currency:           core.DefaultCurrency,
		Balance:            0,
		Metadata:           req.Metadata,
	}
	customer.CreatedAt = now
	customer.UpdatedAt = now

	if req.BillingAddress != nil {
		customer.BillingAddressLine1 = req.BillingAddress.Line1
		customer.BillingAddressLine2 = req.BillingAddress.Line2
		customer.BillingCity = req.BillingAddress.City
		customer.BillingState = req.BillingAddress.State
		customer.BillingPostalCode = req.BillingAddress.PostalCode
		customer.BillingCountry = req.BillingAddress.Country
	}

	if req.Metadata == nil {
		customer.Metadata = make(map[string]any)
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return s.schemaToCoreCustomer(customer), nil
}

// Update updates a customer.
func (s *CustomerService) Update(ctx context.Context, id xid.ID, req *core.UpdateCustomerRequest) (*core.Customer, error) {
	customer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrCustomerNotFound
	}

	if req.Email != nil {
		customer.Email = *req.Email
	}

	if req.Name != nil {
		customer.Name = *req.Name
	}

	if req.Phone != nil {
		customer.Phone = *req.Phone
	}

	if req.TaxID != nil {
		customer.TaxID = *req.TaxID
	}

	if req.TaxExempt != nil {
		customer.TaxExempt = *req.TaxExempt
	}

	if req.BillingAddress != nil {
		customer.BillingAddressLine1 = req.BillingAddress.Line1
		customer.BillingAddressLine2 = req.BillingAddress.Line2
		customer.BillingCity = req.BillingAddress.City
		customer.BillingState = req.BillingAddress.State
		customer.BillingPostalCode = req.BillingAddress.PostalCode
		customer.BillingCountry = req.BillingAddress.Country
	}

	if req.Metadata != nil {
		customer.Metadata = req.Metadata
	}

	customer.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	// Sync to provider
	if err := s.provider.UpdateCustomer(ctx, customer.ProviderCustomerID, customer.Email, customer.Name, customer.Metadata); err != nil {
		_ = err // Log but don't fail
	}

	return s.schemaToCoreCustomer(customer), nil
}

// GetByOrganizationID retrieves a customer by organization ID.
func (s *CustomerService) GetByOrganizationID(ctx context.Context, orgID xid.ID) (*core.Customer, error) {
	customer, err := s.repo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrCustomerNotFound
	}

	return s.schemaToCoreCustomer(customer), nil
}

// GetOrCreate gets an existing customer or creates a new one.
func (s *CustomerService) GetOrCreate(ctx context.Context, orgID xid.ID, email, name string) (*core.Customer, error) {
	customer, err := s.repo.FindByOrganizationID(ctx, orgID)
	if err == nil {
		return s.schemaToCoreCustomer(customer), nil
	}

	// Create new customer
	return s.Create(ctx, &core.CreateCustomerRequest{
		OrganizationID: orgID,
		Email:          email,
		Name:           name,
	})
}

// SyncToProvider syncs customer data to the provider.
func (s *CustomerService) SyncToProvider(ctx context.Context, id xid.ID) error {
	customer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrCustomerNotFound
	}

	return s.provider.UpdateCustomer(ctx, customer.ProviderCustomerID, customer.Email, customer.Name, customer.Metadata)
}

func (s *CustomerService) schemaToCoreCustomer(customer *schema.SubscriptionCustomer) *core.Customer {
	c := &core.Customer{
		ID:                 customer.ID,
		OrganizationID:     customer.OrganizationID,
		ProviderCustomerID: customer.ProviderCustomerID,
		Email:              customer.Email,
		Name:               customer.Name,
		Phone:              customer.Phone,
		TaxID:              customer.TaxID,
		TaxExempt:          customer.TaxExempt,
		Currency:           customer.Currency,
		Balance:            customer.Balance,
		DefaultPaymentID:   customer.DefaultPaymentID,
		Metadata:           customer.Metadata,
		CreatedAt:          customer.CreatedAt,
		UpdatedAt:          customer.UpdatedAt,
	}

	if customer.BillingAddressLine1 != "" || customer.BillingCountry != "" {
		c.BillingAddress = &core.BillingAddress{
			Line1:      customer.BillingAddressLine1,
			Line2:      customer.BillingAddressLine2,
			City:       customer.BillingCity,
			State:      customer.BillingState,
			PostalCode: customer.BillingPostalCode,
			Country:    customer.BillingCountry,
		}
	}

	return c
}
