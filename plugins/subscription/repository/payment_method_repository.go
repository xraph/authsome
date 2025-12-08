package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// paymentMethodRepository implements PaymentMethodRepository using Bun
type paymentMethodRepository struct {
	db *bun.DB
}

// NewPaymentMethodRepository creates a new payment method repository
func NewPaymentMethodRepository(db *bun.DB) PaymentMethodRepository {
	return &paymentMethodRepository{db: db}
}

// Create creates a new payment method
func (r *paymentMethodRepository) Create(ctx context.Context, pm *schema.SubscriptionPaymentMethod) error {
	_, err := r.db.NewInsert().Model(pm).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create payment method: %w", err)
	}
	return nil
}

// Update updates an existing payment method
func (r *paymentMethodRepository) Update(ctx context.Context, pm *schema.SubscriptionPaymentMethod) error {
	_, err := r.db.NewUpdate().
		Model(pm).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update payment method: %w", err)
	}
	return nil
}

// Delete deletes a payment method
func (r *paymentMethodRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionPaymentMethod)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete payment method: %w", err)
	}
	return nil
}

// FindByID retrieves a payment method by ID
func (r *paymentMethodRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionPaymentMethod, error) {
	pm := new(schema.SubscriptionPaymentMethod)
	err := r.db.NewSelect().
		Model(pm).
		Where("spm.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find payment method: %w", err)
	}
	return pm, nil
}

// FindByProviderID retrieves a payment method by provider method ID
func (r *paymentMethodRepository) FindByProviderID(ctx context.Context, providerMethodID string) (*schema.SubscriptionPaymentMethod, error) {
	pm := new(schema.SubscriptionPaymentMethod)
	err := r.db.NewSelect().
		Model(pm).
		Where("spm.provider_method_id = ?", providerMethodID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find payment method by provider ID: %w", err)
	}
	return pm, nil
}

// ListByOrganization retrieves all payment methods for an organization
func (r *paymentMethodRepository) ListByOrganization(ctx context.Context, orgID xid.ID) ([]*schema.SubscriptionPaymentMethod, error) {
	var methods []*schema.SubscriptionPaymentMethod
	err := r.db.NewSelect().
		Model(&methods).
		Where("spm.organization_id = ?", orgID).
		Order("spm.is_default DESC", "spm.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}
	return methods, nil
}

// GetDefault retrieves the default payment method for an organization
func (r *paymentMethodRepository) GetDefault(ctx context.Context, orgID xid.ID) (*schema.SubscriptionPaymentMethod, error) {
	pm := new(schema.SubscriptionPaymentMethod)
	err := r.db.NewSelect().
		Model(pm).
		Where("spm.organization_id = ?", orgID).
		Where("spm.is_default = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default payment method: %w", err)
	}
	return pm, nil
}

// SetDefault sets a payment method as the default
func (r *paymentMethodRepository) SetDefault(ctx context.Context, orgID, paymentMethodID xid.ID) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing default
	_, err = tx.NewUpdate().
		Model((*schema.SubscriptionPaymentMethod)(nil)).
		Set("is_default = ?", false).
		Where("organization_id = ?", orgID).
		Where("is_default = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear existing default: %w", err)
	}

	// Set new default
	_, err = tx.NewUpdate().
		Model((*schema.SubscriptionPaymentMethod)(nil)).
		Set("is_default = ?", true).
		Where("id = ?", paymentMethodID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set new default: %w", err)
	}

	return tx.Commit()
}

// ClearDefault clears the default flag on all payment methods for an organization
func (r *paymentMethodRepository) ClearDefault(ctx context.Context, orgID xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.SubscriptionPaymentMethod)(nil)).
		Set("is_default = ?", false).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear default payment methods: %w", err)
	}
	return nil
}
