package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// customerRepository implements CustomerRepository using Bun
type customerRepository struct {
	db *bun.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *bun.DB) CustomerRepository {
	return &customerRepository{db: db}
}

// Create creates a new customer
func (r *customerRepository) Create(ctx context.Context, customer *schema.SubscriptionCustomer) error {
	_, err := r.db.NewInsert().Model(customer).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}
	return nil
}

// Update updates an existing customer
func (r *customerRepository) Update(ctx context.Context, customer *schema.SubscriptionCustomer) error {
	_, err := r.db.NewUpdate().
		Model(customer).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}
	return nil
}

// Delete deletes a customer
func (r *customerRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionCustomer)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	return nil
}

// FindByID retrieves a customer by ID
func (r *customerRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionCustomer, error) {
	customer := new(schema.SubscriptionCustomer)
	err := r.db.NewSelect().
		Model(customer).
		Where("sc.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}
	return customer, nil
}

// FindByOrganizationID retrieves a customer by organization ID
func (r *customerRepository) FindByOrganizationID(ctx context.Context, orgID xid.ID) (*schema.SubscriptionCustomer, error) {
	customer := new(schema.SubscriptionCustomer)
	err := r.db.NewSelect().
		Model(customer).
		Where("sc.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find customer by organization: %w", err)
	}
	return customer, nil
}

// FindByProviderID retrieves a customer by provider customer ID
func (r *customerRepository) FindByProviderID(ctx context.Context, providerCustomerID string) (*schema.SubscriptionCustomer, error) {
	customer := new(schema.SubscriptionCustomer)
	err := r.db.NewSelect().
		Model(customer).
		Where("sc.provider_customer_id = ?", providerCustomerID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find customer by provider ID: %w", err)
	}
	return customer, nil
}
