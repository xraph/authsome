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

// TaxRepository defines the interface for tax operations
type TaxRepository interface {
	// Tax rate operations
	CreateTaxRate(ctx context.Context, rate *core.TaxRate) error
	GetTaxRate(ctx context.Context, id xid.ID) (*core.TaxRate, error)
	GetTaxRateByLocation(ctx context.Context, appID xid.ID, country, state string) (*core.TaxRate, error)
	ListTaxRates(ctx context.Context, appID xid.ID, activeOnly bool) ([]*core.TaxRate, error)
	UpdateTaxRate(ctx context.Context, rate *core.TaxRate) error
	DeleteTaxRate(ctx context.Context, id xid.ID) error

	// Tax exemption operations
	CreateTaxExemption(ctx context.Context, exemption *core.TaxExemption) error
	GetTaxExemption(ctx context.Context, id xid.ID) (*core.TaxExemption, error)
	GetTaxExemptionByOrg(ctx context.Context, orgID xid.ID, country string) (*core.TaxExemption, error)
	ListTaxExemptions(ctx context.Context, orgID xid.ID) ([]*core.TaxExemption, error)
	UpdateTaxExemption(ctx context.Context, exemption *core.TaxExemption) error
	DeleteTaxExemption(ctx context.Context, id xid.ID) error

	// Customer tax ID operations
	CreateCustomerTaxID(ctx context.Context, taxID *core.CustomerTaxID) error
	GetCustomerTaxID(ctx context.Context, id xid.ID) (*core.CustomerTaxID, error)
	ListCustomerTaxIDs(ctx context.Context, orgID xid.ID) ([]*core.CustomerTaxID, error)
	UpdateCustomerTaxID(ctx context.Context, taxID *core.CustomerTaxID) error
	DeleteCustomerTaxID(ctx context.Context, id xid.ID) error
}

// taxRepository implements TaxRepository using Bun
type taxRepository struct {
	db *bun.DB
}

// NewTaxRepository creates a new tax repository
func NewTaxRepository(db *bun.DB) TaxRepository {
	return &taxRepository{db: db}
}

// CreateTaxRate creates a new tax rate
func (r *taxRepository) CreateTaxRate(ctx context.Context, rate *core.TaxRate) error {
	model := taxRateToSchema(rate)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetTaxRate returns a tax rate by ID
func (r *taxRepository) GetTaxRate(ctx context.Context, id xid.ID) (*core.TaxRate, error) {
	var rate schema.SubscriptionTaxRate
	err := r.db.NewSelect().
		Model(&rate).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToTaxRate(&rate), nil
}

// GetTaxRateByLocation returns the tax rate for a location
func (r *taxRepository) GetTaxRateByLocation(ctx context.Context, appID xid.ID, country, state string) (*core.TaxRate, error) {
	var rate schema.SubscriptionTaxRate
	query := r.db.NewSelect().
		Model(&rate).
		Where("app_id = ?", appID).
		Where("country = ?", country).
		Where("is_active = ?", true).
		Where("valid_from <= ?", time.Now()).
		Where("(valid_until IS NULL OR valid_until > ?)", time.Now())

	if state != "" {
		query = query.Where("(state = ? OR state = '')", state)
	}

	err := query.Order("state DESC", "is_default DESC").Limit(1).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToTaxRate(&rate), nil
}

// ListTaxRates returns all tax rates for an app
func (r *taxRepository) ListTaxRates(ctx context.Context, appID xid.ID, activeOnly bool) ([]*core.TaxRate, error) {
	var rates []schema.SubscriptionTaxRate
	query := r.db.NewSelect().
		Model(&rates).
		Where("app_id = ?", appID)

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	err := query.Order("country ASC", "state ASC", "is_default DESC").Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.TaxRate, len(rates))
	for i, r := range rates {
		result[i] = schemaToTaxRate(&r)
	}
	return result, nil
}

// UpdateTaxRate updates a tax rate
func (r *taxRepository) UpdateTaxRate(ctx context.Context, rate *core.TaxRate) error {
	model := taxRateToSchema(rate)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteTaxRate deletes a tax rate
func (r *taxRepository) DeleteTaxRate(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionTaxRate)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CreateTaxExemption creates a new tax exemption
func (r *taxRepository) CreateTaxExemption(ctx context.Context, exemption *core.TaxExemption) error {
	model := taxExemptionToSchema(exemption)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetTaxExemption returns a tax exemption by ID
func (r *taxRepository) GetTaxExemption(ctx context.Context, id xid.ID) (*core.TaxExemption, error) {
	var exemption schema.SubscriptionTaxExemption
	err := r.db.NewSelect().
		Model(&exemption).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToTaxExemption(&exemption), nil
}

// GetTaxExemptionByOrg returns the tax exemption for an organization
func (r *taxRepository) GetTaxExemptionByOrg(ctx context.Context, orgID xid.ID, country string) (*core.TaxExemption, error) {
	var exemption schema.SubscriptionTaxExemption
	err := r.db.NewSelect().
		Model(&exemption).
		Where("organization_id = ?", orgID).
		Where("country = ?", country).
		Where("is_active = ?", true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToTaxExemption(&exemption), nil
}

// ListTaxExemptions returns all tax exemptions for an organization
func (r *taxRepository) ListTaxExemptions(ctx context.Context, orgID xid.ID) ([]*core.TaxExemption, error) {
	var exemptions []schema.SubscriptionTaxExemption
	err := r.db.NewSelect().
		Model(&exemptions).
		Where("organization_id = ?", orgID).
		Order("country ASC", "created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.TaxExemption, len(exemptions))
	for i, e := range exemptions {
		result[i] = schemaToTaxExemption(&e)
	}
	return result, nil
}

// UpdateTaxExemption updates a tax exemption
func (r *taxRepository) UpdateTaxExemption(ctx context.Context, exemption *core.TaxExemption) error {
	model := taxExemptionToSchema(exemption)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteTaxExemption deletes a tax exemption
func (r *taxRepository) DeleteTaxExemption(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionTaxExemption)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CreateCustomerTaxID creates a new customer tax ID
func (r *taxRepository) CreateCustomerTaxID(ctx context.Context, taxID *core.CustomerTaxID) error {
	model := customerTaxIDToSchema(taxID)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetCustomerTaxID returns a customer tax ID by ID
func (r *taxRepository) GetCustomerTaxID(ctx context.Context, id xid.ID) (*core.CustomerTaxID, error) {
	var taxID schema.SubscriptionCustomerTaxID
	err := r.db.NewSelect().
		Model(&taxID).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToCustomerTaxID(&taxID), nil
}

// ListCustomerTaxIDs returns all tax IDs for an organization
func (r *taxRepository) ListCustomerTaxIDs(ctx context.Context, orgID xid.ID) ([]*core.CustomerTaxID, error) {
	var taxIDs []schema.SubscriptionCustomerTaxID
	err := r.db.NewSelect().
		Model(&taxIDs).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.CustomerTaxID, len(taxIDs))
	for i, t := range taxIDs {
		result[i] = schemaToCustomerTaxID(&t)
	}
	return result, nil
}

// UpdateCustomerTaxID updates a customer tax ID
func (r *taxRepository) UpdateCustomerTaxID(ctx context.Context, taxID *core.CustomerTaxID) error {
	model := customerTaxIDToSchema(taxID)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteCustomerTaxID deletes a customer tax ID
func (r *taxRepository) DeleteCustomerTaxID(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionCustomerTaxID)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// Helper functions

func schemaToTaxRate(s *schema.SubscriptionTaxRate) *core.TaxRate {
	return &core.TaxRate{
		ID:                s.ID,
		AppID:             s.AppID,
		Name:              s.Name,
		Description:       s.Description,
		Type:              core.TaxType(s.Type),
		Percentage:        s.Percentage,
		Country:           s.Country,
		State:             s.State,
		City:              s.City,
		PostalCode:        s.PostalCode,
		Behavior:          core.TaxBehavior(s.Behavior),
		IsDefault:         s.IsDefault,
		IsActive:          s.IsActive,
		ProviderTaxRateID: s.ProviderTaxRateID,
		ValidFrom:         s.ValidFrom,
		ValidUntil:        s.ValidUntil,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

func taxRateToSchema(r *core.TaxRate) *schema.SubscriptionTaxRate {
	return &schema.SubscriptionTaxRate{
		ID:                r.ID,
		AppID:             r.AppID,
		Name:              r.Name,
		Description:       r.Description,
		Type:              string(r.Type),
		Percentage:        r.Percentage,
		Country:           r.Country,
		State:             r.State,
		City:              r.City,
		PostalCode:        r.PostalCode,
		Behavior:          string(r.Behavior),
		IsDefault:         r.IsDefault,
		IsActive:          r.IsActive,
		ProviderTaxRateID: r.ProviderTaxRateID,
		ValidFrom:         r.ValidFrom,
		ValidUntil:        r.ValidUntil,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func schemaToTaxExemption(s *schema.SubscriptionTaxExemption) *core.TaxExemption {
	return &core.TaxExemption{
		ID:             s.ID,
		AppID:          s.AppID,
		OrganizationID: s.OrganizationID,
		Type:           s.Type,
		Certificate:    s.Certificate,
		Country:        s.Country,
		State:          s.State,
		VerifiedAt:     s.VerifiedAt,
		ExpiresAt:      s.ExpiresAt,
		IsActive:       s.IsActive,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func taxExemptionToSchema(e *core.TaxExemption) *schema.SubscriptionTaxExemption {
	return &schema.SubscriptionTaxExemption{
		ID:             e.ID,
		AppID:          e.AppID,
		OrganizationID: e.OrganizationID,
		Type:           e.Type,
		Certificate:    e.Certificate,
		Country:        e.Country,
		State:          e.State,
		VerifiedAt:     e.VerifiedAt,
		ExpiresAt:      e.ExpiresAt,
		IsActive:       e.IsActive,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func schemaToCustomerTaxID(s *schema.SubscriptionCustomerTaxID) *core.CustomerTaxID {
	return &core.CustomerTaxID{
		ID:             s.ID,
		AppID:          s.AppID,
		OrganizationID: s.OrganizationID,
		Type:           core.TaxIDType(s.Type),
		Value:          s.Value,
		Country:        s.Country,
		VerifiedAt:     s.VerifiedAt,
		IsValid:        s.IsValid,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func customerTaxIDToSchema(t *core.CustomerTaxID) *schema.SubscriptionCustomerTaxID {
	return &schema.SubscriptionCustomerTaxID{
		ID:             t.ID,
		AppID:          t.AppID,
		OrganizationID: t.OrganizationID,
		Type:           string(t.Type),
		Value:          t.Value,
		Country:        t.Country,
		VerifiedAt:     t.VerifiedAt,
		IsValid:        t.IsValid,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

