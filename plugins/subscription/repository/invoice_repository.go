package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// invoiceRepository implements InvoiceRepository using Bun.
type invoiceRepository struct {
	db *bun.DB
}

// NewInvoiceRepository creates a new invoice repository.
func NewInvoiceRepository(db *bun.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

// Create creates a new invoice.
func (r *invoiceRepository) Create(ctx context.Context, invoice *schema.SubscriptionInvoice) error {
	_, err := r.db.NewInsert().Model(invoice).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	return nil
}

// Update updates an existing invoice.
func (r *invoiceRepository) Update(ctx context.Context, invoice *schema.SubscriptionInvoice) error {
	_, err := r.db.NewUpdate().
		Model(invoice).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	return nil
}

// FindByID retrieves an invoice by ID.
func (r *invoiceRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionInvoice, error) {
	invoice := new(schema.SubscriptionInvoice)

	err := r.db.NewSelect().
		Model(invoice).
		Relation("Items").
		Relation("Subscription").
		Where("si.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find invoice: %w", err)
	}

	return invoice, nil
}

// FindByNumber retrieves an invoice by number.
func (r *invoiceRepository) FindByNumber(ctx context.Context, number string) (*schema.SubscriptionInvoice, error) {
	invoice := new(schema.SubscriptionInvoice)

	err := r.db.NewSelect().
		Model(invoice).
		Relation("Items").
		Relation("Subscription").
		Where("si.number = ?", number).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find invoice by number: %w", err)
	}

	return invoice, nil
}

// FindByProviderID retrieves an invoice by provider invoice ID.
func (r *invoiceRepository) FindByProviderID(ctx context.Context, providerInvoiceID string) (*schema.SubscriptionInvoice, error) {
	invoice := new(schema.SubscriptionInvoice)

	err := r.db.NewSelect().
		Model(invoice).
		Relation("Items").
		Relation("Subscription").
		Where("si.provider_invoice_id = ?", providerInvoiceID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find invoice by provider ID: %w", err)
	}

	return invoice, nil
}

// List retrieves invoices with optional filters.
func (r *invoiceRepository) List(ctx context.Context, filter *InvoiceFilter) ([]*schema.SubscriptionInvoice, int, error) {
	var invoices []*schema.SubscriptionInvoice

	query := r.db.NewSelect().
		Model(&invoices).
		Relation("Subscription").
		Order("si.created_at DESC")

	if filter != nil {
		if filter.OrganizationID != nil {
			query = query.Where("si.organization_id = ?", *filter.OrganizationID)
		}

		if filter.SubscriptionID != nil {
			query = query.Where("si.subscription_id = ?", *filter.SubscriptionID)
		}

		if filter.Status != "" {
			query = query.Where("si.status = ?", filter.Status)
		}

		// Pagination
		pageSize := filter.PageSize
		if pageSize <= 0 {
			pageSize = 20
		}

		page := filter.Page
		if page <= 0 {
			page = 1
		}

		offset := (page - 1) * pageSize
		query = query.Limit(pageSize).Offset(offset)
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}

	return invoices, count, nil
}

// CreateItem creates an invoice line item.
func (r *invoiceRepository) CreateItem(ctx context.Context, item *schema.SubscriptionInvoiceItem) error {
	_, err := r.db.NewInsert().Model(item).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create invoice item: %w", err)
	}

	return nil
}

// GetItems retrieves all items for an invoice.
func (r *invoiceRepository) GetItems(ctx context.Context, invoiceID xid.ID) ([]*schema.SubscriptionInvoiceItem, error) {
	var items []*schema.SubscriptionInvoiceItem

	err := r.db.NewSelect().
		Model(&items).
		Where("sii.invoice_id = ?", invoiceID).
		Order("sii.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice items: %w", err)
	}

	return items, nil
}

// GetNextInvoiceNumber generates the next invoice number.
func (r *invoiceRepository) GetNextInvoiceNumber(ctx context.Context, appID xid.ID) (string, error) {
	// Get current year
	year := time.Now().Year()

	// Count existing invoices for this app/year
	count, err := r.db.NewSelect().
		Model((*schema.SubscriptionInvoice)(nil)).
		Join("JOIN subscriptions AS sub ON sub.id = si.subscription_id").
		Join("JOIN subscription_plans AS plan ON plan.id = sub.plan_id").
		Where("plan.app_id = ?", appID).
		Where("EXTRACT(YEAR FROM si.created_at) = ?", year).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to count invoices: %w", err)
	}

	// Generate number in format: INV-YYYY-NNNN
	number := fmt.Sprintf("INV-%d-%04d", year, count+1)

	return number, nil
}
