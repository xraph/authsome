package core

import (
	"time"

	"github.com/rs/xid"
)

// Invoice represents a billing invoice
type Invoice struct {
	ID                xid.ID        `json:"id"`
	SubscriptionID    xid.ID        `json:"subscriptionId"`    // Related subscription
	OrganizationID    xid.ID        `json:"organizationId"`    // Organization billed
	Number            string        `json:"number"`            // Invoice number (e.g., INV-2024-0001)
	Status            InvoiceStatus `json:"status"`            // Current status
	Currency          string        `json:"currency"`          // ISO 4217 currency code
	Subtotal          int64         `json:"subtotal"`          // Amount before tax in cents
	Tax               int64         `json:"tax"`               // Tax amount in cents
	Total             int64         `json:"total"`             // Total amount in cents
	AmountPaid        int64         `json:"amountPaid"`        // Amount already paid in cents
	AmountDue         int64         `json:"amountDue"`         // Remaining amount due in cents
	Description       string        `json:"description"`       // Invoice description
	PeriodStart       time.Time     `json:"periodStart"`       // Billing period start
	PeriodEnd         time.Time     `json:"periodEnd"`         // Billing period end
	DueDate           time.Time     `json:"dueDate"`           // Payment due date
	PaidAt            *time.Time    `json:"paidAt"`            // When paid
	VoidedAt          *time.Time    `json:"voidedAt"`          // When voided
	ProviderInvoiceID string        `json:"providerInvoiceId"` // Stripe Invoice ID
	ProviderPDFURL    string        `json:"providerPdfUrl"`    // Stripe hosted PDF URL
	HostedInvoiceURL  string        `json:"hostedInvoiceUrl"`  // Stripe hosted invoice page
	Metadata          map[string]any `json:"metadata"`         // Custom metadata
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`

	// Relations
	Items []InvoiceItem `json:"items,omitempty"`
}

// InvoiceItem represents a line item on an invoice
type InvoiceItem struct {
	ID              xid.ID         `json:"id"`
	InvoiceID       xid.ID         `json:"invoiceId"`
	Description     string         `json:"description"`     // Line item description
	Quantity        int64          `json:"quantity"`        // Quantity
	UnitAmount      int64          `json:"unitAmount"`      // Unit price in cents
	Amount          int64          `json:"amount"`          // Total amount in cents
	PlanID          *xid.ID        `json:"planId"`          // Related plan if applicable
	AddOnID         *xid.ID        `json:"addOnId"`         // Related add-on if applicable
	PeriodStart     time.Time      `json:"periodStart"`     // Item period start
	PeriodEnd       time.Time      `json:"periodEnd"`       // Item period end
	Proration       bool           `json:"proration"`       // Is this a proration adjustment
	Metadata        map[string]any `json:"metadata"`        // Custom metadata
	ProviderItemID  string         `json:"providerItemId"`  // Stripe Invoice Item ID
	CreatedAt       time.Time      `json:"createdAt"`
}

// NewInvoice creates a new Invoice with default values
func NewInvoice(subID, orgID xid.ID) *Invoice {
	now := time.Now()
	return &Invoice{
		ID:             xid.New(),
		SubscriptionID: subID,
		OrganizationID: orgID,
		Status:         InvoiceStatusDraft,
		Currency:       DefaultCurrency,
		Items:          []InvoiceItem{},
		Metadata:       make(map[string]any),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsPaid returns true if the invoice has been paid
func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

// IsOpen returns true if the invoice is awaiting payment
func (i *Invoice) IsOpen() bool {
	return i.Status == InvoiceStatusOpen
}

// IsDraft returns true if the invoice is still a draft
func (i *Invoice) IsDraft() bool {
	return i.Status == InvoiceStatusDraft
}

// IsOverdue returns true if the invoice is past due date and not paid
func (i *Invoice) IsOverdue() bool {
	if i.Status != InvoiceStatusOpen {
		return false
	}
	return time.Now().After(i.DueDate)
}

// AddItem adds a line item to the invoice
func (i *Invoice) AddItem(description string, quantity, unitAmount int64) *InvoiceItem {
	item := InvoiceItem{
		ID:          xid.New(),
		InvoiceID:   i.ID,
		Description: description,
		Quantity:    quantity,
		UnitAmount:  unitAmount,
		Amount:      quantity * unitAmount,
		PeriodStart: i.PeriodStart,
		PeriodEnd:   i.PeriodEnd,
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}
	i.Items = append(i.Items, item)
	i.recalculateTotals()
	return &item
}

// recalculateTotals recalculates subtotal, total, and amount due
func (i *Invoice) recalculateTotals() {
	var subtotal int64
	for _, item := range i.Items {
		subtotal += item.Amount
	}
	i.Subtotal = subtotal
	i.Total = subtotal + i.Tax
	i.AmountDue = i.Total - i.AmountPaid
}

// MarkPaid marks the invoice as paid
func (i *Invoice) MarkPaid() {
	now := time.Now()
	i.Status = InvoiceStatusPaid
	i.PaidAt = &now
	i.AmountPaid = i.Total
	i.AmountDue = 0
	i.UpdatedAt = now
}

// Void voids the invoice
func (i *Invoice) Void() {
	now := time.Now()
	i.Status = InvoiceStatusVoid
	i.VoidedAt = &now
	i.UpdatedAt = now
}

// Finalize moves the invoice from draft to open
func (i *Invoice) Finalize() {
	i.Status = InvoiceStatusOpen
	i.UpdatedAt = time.Now()
}

// ListInvoicesFilter defines filters for listing invoices
type ListInvoicesFilter struct {
	OrganizationID *xid.ID       `json:"organizationId"`
	SubscriptionID *xid.ID       `json:"subscriptionId"`
	Status         InvoiceStatus `json:"status"`
	FromDate       *time.Time    `json:"fromDate"`
	ToDate         *time.Time    `json:"toDate"`
	Page           int           `json:"page"`
	PageSize       int           `json:"pageSize"`
}

