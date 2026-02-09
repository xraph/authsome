package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// InvoiceService handles invoice business logic.
type InvoiceService struct {
	repo      repository.InvoiceRepository
	subRepo   repository.SubscriptionRepository
	provider  providers.PaymentProvider
	eventRepo repository.EventRepository
}

// NewInvoiceService creates a new invoice service.
func NewInvoiceService(
	repo repository.InvoiceRepository,
	subRepo repository.SubscriptionRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *InvoiceService {
	return &InvoiceService{
		repo:      repo,
		subRepo:   subRepo,
		provider:  provider,
		eventRepo: eventRepo,
	}
}

// GetByID retrieves an invoice by ID.
func (s *InvoiceService) GetByID(ctx context.Context, id xid.ID) (*core.Invoice, error) {
	invoice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrInvoiceNotFound
	}

	return s.schemaToCoreInvoice(invoice), nil
}

// GetByNumber retrieves an invoice by number.
func (s *InvoiceService) GetByNumber(ctx context.Context, number string) (*core.Invoice, error) {
	invoice, err := s.repo.FindByNumber(ctx, number)
	if err != nil {
		return nil, suberrors.ErrInvoiceNotFound
	}

	return s.schemaToCoreInvoice(invoice), nil
}

// List retrieves invoices with filtering.
func (s *InvoiceService) List(ctx context.Context, orgID, subID *xid.ID, status string, page, pageSize int) ([]*core.Invoice, int, error) {
	filter := &repository.InvoiceFilter{
		OrganizationID: orgID,
		SubscriptionID: subID,
		Status:         status,
		Page:           page,
		PageSize:       pageSize,
	}

	invoices, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}

	result := make([]*core.Invoice, len(invoices))
	for i, inv := range invoices {
		result[i] = s.schemaToCoreInvoice(inv)
	}

	return result, count, nil
}

// GetPDFURL returns the PDF URL for an invoice.
func (s *InvoiceService) GetPDFURL(ctx context.Context, id xid.ID) (string, error) {
	invoice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return "", suberrors.ErrInvoiceNotFound
	}

	if invoice.ProviderPDFURL != "" {
		return invoice.ProviderPDFURL, nil
	}

	// Try to get from provider
	if invoice.ProviderInvoiceID != "" {
		url, err := s.provider.GetInvoicePDF(ctx, invoice.ProviderInvoiceID)
		if err == nil && url != "" {
			// Update cached URL
			invoice.ProviderPDFURL = url
			_ = s.repo.Update(ctx, invoice)

			return url, nil
		}
	}

	return "", errs.New(errs.CodeNotFound, "PDF not available", http.StatusNotFound)
}

// Void voids an invoice.
func (s *InvoiceService) Void(ctx context.Context, id xid.ID) error {
	invoice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrInvoiceNotFound
	}

	if invoice.Status == string(core.InvoiceStatusPaid) {
		return suberrors.ErrInvoiceAlreadyPaid
	}

	if invoice.Status == string(core.InvoiceStatusVoid) {
		return suberrors.ErrInvoiceVoided
	}

	now := time.Now()
	invoice.Status = string(core.InvoiceStatusVoid)
	invoice.VoidedAt = &now
	invoice.UpdatedAt = now

	if err := s.repo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to void invoice: %w", err)
	}

	// Record event
	s.recordEvent(ctx, invoice.SubscriptionID, invoice.OrganizationID, "invoice.voided", map[string]any{
		"invoiceId":     invoice.ID.String(),
		"invoiceNumber": invoice.Number,
	})

	return nil
}

// Create creates a new invoice.
func (s *InvoiceService) Create(ctx context.Context, subID, orgID xid.ID, periodStart, periodEnd time.Time) (*core.Invoice, error) {
	// Get subscription
	sub, err := s.subRepo.FindByID(ctx, subID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Generate invoice number
	number, err := s.repo.GetNextInvoiceNumber(ctx, sub.Plan.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	now := time.Now()
	invoice := &schema.SubscriptionInvoice{
		ID:             xid.New(),
		SubscriptionID: subID,
		OrganizationID: orgID,
		Number:         number,
		Status:         string(core.InvoiceStatusDraft),
		Currency:       sub.Plan.Currency,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		DueDate:        periodStart.AddDate(0, 0, 14), // Net 14
		Metadata:       make(map[string]any),
	}
	invoice.CreatedAt = now
	invoice.UpdatedAt = now

	if err := s.repo.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Record event
	s.recordEvent(ctx, subID, orgID, string(core.EventInvoiceCreated), map[string]any{
		"invoiceId":     invoice.ID.String(),
		"invoiceNumber": invoice.Number,
	})

	return s.schemaToCoreInvoice(invoice), nil
}

// AddItem adds a line item to an invoice.
func (s *InvoiceService) AddItem(ctx context.Context, invoiceID xid.ID, description string, quantity, unitAmount int64) error {
	invoice, err := s.repo.FindByID(ctx, invoiceID)
	if err != nil {
		return suberrors.ErrInvoiceNotFound
	}

	if invoice.Status != string(core.InvoiceStatusDraft) {
		return suberrors.ErrInvoiceNotOpen
	}

	item := &schema.SubscriptionInvoiceItem{
		ID:          xid.New(),
		InvoiceID:   invoiceID,
		Description: description,
		Quantity:    quantity,
		UnitAmount:  unitAmount,
		Amount:      quantity * unitAmount,
		PeriodStart: invoice.PeriodStart,
		PeriodEnd:   invoice.PeriodEnd,
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateItem(ctx, item); err != nil {
		return fmt.Errorf("failed to add invoice item: %w", err)
	}

	// Recalculate totals
	return s.recalculateTotals(ctx, invoiceID)
}

// Finalize finalizes an invoice.
func (s *InvoiceService) Finalize(ctx context.Context, id xid.ID) error {
	invoice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrInvoiceNotFound
	}

	if invoice.Status != string(core.InvoiceStatusDraft) {
		return suberrors.ErrInvoiceNotOpen
	}

	invoice.Status = string(core.InvoiceStatusOpen)
	invoice.UpdatedAt = time.Now()

	return s.repo.Update(ctx, invoice)
}

// MarkPaid marks an invoice as paid.
func (s *InvoiceService) MarkPaid(ctx context.Context, id xid.ID) error {
	invoice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrInvoiceNotFound
	}

	if invoice.Status == string(core.InvoiceStatusPaid) {
		return suberrors.ErrInvoiceAlreadyPaid
	}

	now := time.Now()
	invoice.Status = string(core.InvoiceStatusPaid)
	invoice.PaidAt = &now
	invoice.AmountPaid = invoice.Total
	invoice.AmountDue = 0
	invoice.UpdatedAt = now

	if err := s.repo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to mark invoice paid: %w", err)
	}

	// Record event
	s.recordEvent(ctx, invoice.SubscriptionID, invoice.OrganizationID, string(core.EventInvoicePaid), map[string]any{
		"invoiceId":     invoice.ID.String(),
		"invoiceNumber": invoice.Number,
		"amount":        invoice.Total,
	})

	return nil
}

// SyncFromProvider syncs an invoice from the provider.
func (s *InvoiceService) SyncFromProvider(ctx context.Context, providerInvoiceID string) (*core.Invoice, error) {
	invoice, err := s.repo.FindByProviderID(ctx, providerInvoiceID)
	if err != nil {
		return nil, suberrors.ErrInvoiceNotFound
	}

	// TODO: Fetch from provider and update local data

	return s.schemaToCoreInvoice(invoice), nil
}

// Helper methods

func (s *InvoiceService) recalculateTotals(ctx context.Context, invoiceID xid.ID) error {
	invoice, err := s.repo.FindByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	items, err := s.repo.GetItems(ctx, invoiceID)
	if err != nil {
		return err
	}

	var subtotal int64
	for _, item := range items {
		subtotal += item.Amount
	}

	invoice.Subtotal = subtotal
	invoice.Total = subtotal + invoice.Tax
	invoice.AmountDue = invoice.Total - invoice.AmountPaid
	invoice.UpdatedAt = time.Now()

	return s.repo.Update(ctx, invoice)
}

func (s *InvoiceService) recordEvent(ctx context.Context, subID, orgID xid.ID, eventType string, data map[string]any) {
	event := &schema.SubscriptionEvent{
		ID:             xid.New(),
		SubscriptionID: &subID,
		OrganizationID: orgID,
		EventType:      eventType,
		EventData:      data,
		CreatedAt:      time.Now(),
	}
	s.eventRepo.Create(ctx, event)
}

func (s *InvoiceService) schemaToCoreInvoice(invoice *schema.SubscriptionInvoice) *core.Invoice {
	coreInvoice := &core.Invoice{
		ID:                invoice.ID,
		SubscriptionID:    invoice.SubscriptionID,
		OrganizationID:    invoice.OrganizationID,
		Number:            invoice.Number,
		Status:            core.InvoiceStatus(invoice.Status),
		Currency:          invoice.Currency,
		Subtotal:          invoice.Subtotal,
		Tax:               invoice.Tax,
		Total:             invoice.Total,
		AmountPaid:        invoice.AmountPaid,
		AmountDue:         invoice.AmountDue,
		Description:       invoice.Description,
		PeriodStart:       invoice.PeriodStart,
		PeriodEnd:         invoice.PeriodEnd,
		DueDate:           invoice.DueDate,
		PaidAt:            invoice.PaidAt,
		VoidedAt:          invoice.VoidedAt,
		ProviderInvoiceID: invoice.ProviderInvoiceID,
		ProviderPDFURL:    invoice.ProviderPDFURL,
		HostedInvoiceURL:  invoice.HostedInvoiceURL,
		Metadata:          invoice.Metadata,
		CreatedAt:         invoice.CreatedAt,
		UpdatedAt:         invoice.UpdatedAt,
	}

	// Convert items
	coreInvoice.Items = make([]core.InvoiceItem, len(invoice.Items))
	for i, item := range invoice.Items {
		coreInvoice.Items[i] = core.InvoiceItem{
			ID:             item.ID,
			InvoiceID:      item.InvoiceID,
			Description:    item.Description,
			Quantity:       item.Quantity,
			UnitAmount:     item.UnitAmount,
			Amount:         item.Amount,
			PlanID:         item.PlanID,
			AddOnID:        item.AddOnID,
			PeriodStart:    item.PeriodStart,
			PeriodEnd:      item.PeriodEnd,
			Proration:      item.Proration,
			Metadata:       item.Metadata,
			ProviderItemID: item.ProviderItemID,
			CreatedAt:      item.CreatedAt,
		}
	}

	return coreInvoice
}
