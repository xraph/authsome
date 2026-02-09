package core

import (
	"time"

	"github.com/rs/xid"
)

// Subscription represents an organization's subscription to a plan.
type Subscription struct {
	ID                 xid.ID             `json:"id"`
	OrganizationID     xid.ID             `json:"organizationId"`     // Links to Organization
	PlanID             xid.ID             `json:"planId"`             // Current plan
	Status             SubscriptionStatus `json:"status"`             // Current status
	Quantity           int                `json:"quantity"`           // For per-seat billing
	CurrentPeriodStart time.Time          `json:"currentPeriodStart"` // Start of current billing period
	CurrentPeriodEnd   time.Time          `json:"currentPeriodEnd"`   // End of current billing period
	TrialStart         *time.Time         `json:"trialStart"`         // Start of trial
	TrialEnd           *time.Time         `json:"trialEnd"`           // End of trial
	CancelAt           *time.Time         `json:"cancelAt"`           // Scheduled cancellation date
	CanceledAt         *time.Time         `json:"canceledAt"`         // When cancellation was requested
	EndedAt            *time.Time         `json:"endedAt"`            // When subscription actually ended
	PausedAt           *time.Time         `json:"pausedAt"`           // When subscription was paused
	ResumeAt           *time.Time         `json:"resumeAt"`           // Scheduled resume date
	ProviderSubID      string             `json:"providerSubId"`      // Stripe Subscription ID
	ProviderCustomerID string             `json:"providerCustomerId"` // Stripe Customer ID
	Metadata           map[string]any     `json:"metadata"`           // Custom metadata
	CreatedAt          time.Time          `json:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt"`

	// Relations (populated when loaded)
	Plan   *Plan               `json:"plan,omitempty"`
	AddOns []SubscriptionAddOn `json:"addOns,omitempty"`
}

// SubscriptionAddOn represents an add-on attached to a subscription.
type SubscriptionAddOn struct {
	ID                xid.ID    `json:"id"`
	SubscriptionID    xid.ID    `json:"subscriptionId"`
	AddOnID           xid.ID    `json:"addOnId"`
	Quantity          int       `json:"quantity"`
	ProviderSubItemID string    `json:"providerSubItemId"` // Stripe Subscription Item ID
	CreatedAt         time.Time `json:"createdAt"`

	// Relations
	AddOn *AddOn `json:"addOn,omitempty"`
}

// NewSubscription creates a new Subscription with default values.
func NewSubscription(orgID, planID xid.ID) *Subscription {
	now := time.Now()

	return &Subscription{
		ID:                 xid.New(),
		OrganizationID:     orgID,
		PlanID:             planID,
		Status:             StatusIncomplete,
		Quantity:           1,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0), // Default to 1 month
		Metadata:           make(map[string]any),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// IsActive returns true if the subscription is currently usable.
func (s *Subscription) IsActive() bool {
	return s.Status.IsActiveOrTrialing()
}

// IsTrialing returns true if the subscription is in trial.
func (s *Subscription) IsTrialing() bool {
	return s.Status == StatusTrialing
}

// IsCanceled returns true if the subscription has been canceled.
func (s *Subscription) IsCanceled() bool {
	return s.Status == StatusCanceled || s.CanceledAt != nil
}

// IsPastDue returns true if payment has failed.
func (s *Subscription) IsPastDue() bool {
	return s.Status == StatusPastDue
}

// IsPaused returns true if the subscription is paused.
func (s *Subscription) IsPaused() bool {
	return s.Status == StatusPaused
}

// WillCancel returns true if the subscription is scheduled to cancel.
func (s *Subscription) WillCancel() bool {
	return s.CancelAt != nil && !s.CancelAt.IsZero()
}

// DaysUntilTrialEnd returns the number of days until trial ends, or -1 if not trialing.
func (s *Subscription) DaysUntilTrialEnd() int {
	if s.TrialEnd == nil || s.Status != StatusTrialing {
		return -1
	}

	duration := time.Until(*s.TrialEnd)

	return int(duration.Hours() / 24)
}

// DaysUntilRenewal returns the number of days until the next billing date.
func (s *Subscription) DaysUntilRenewal() int {
	if s.CurrentPeriodEnd.IsZero() {
		return -1
	}

	duration := time.Until(s.CurrentPeriodEnd)

	return int(duration.Hours() / 24)
}

// StartTrial starts a trial period.
func (s *Subscription) StartTrial(days int) {
	now := time.Now()
	trialEnd := now.AddDate(0, 0, days)
	s.TrialStart = &now
	s.TrialEnd = &trialEnd
	s.Status = StatusTrialing
	s.UpdatedAt = now
}

// ActivateFromTrial converts trial to active subscription.
func (s *Subscription) ActivateFromTrial() {
	now := time.Now()
	s.Status = StatusActive
	s.CurrentPeriodStart = now
	s.UpdatedAt = now
}

// Cancel marks the subscription for cancellation.
func (s *Subscription) Cancel(immediate bool) {
	now := time.Now()
	s.CanceledAt = &now
	s.UpdatedAt = now

	if immediate {
		s.EndedAt = &now
		s.Status = StatusCanceled
	} else {
		// Cancel at end of period
		s.CancelAt = &s.CurrentPeriodEnd
	}
}

// Pause pauses the subscription.
func (s *Subscription) Pause(resumeAt *time.Time) {
	now := time.Now()
	s.PausedAt = &now
	s.ResumeAt = resumeAt
	s.Status = StatusPaused
	s.UpdatedAt = now
}

// Resume resumes a paused subscription.
func (s *Subscription) Resume() {
	now := time.Now()
	s.PausedAt = nil
	s.ResumeAt = nil
	s.Status = StatusActive
	s.UpdatedAt = now
}

// CreateSubscriptionRequest represents a request to create a subscription.
type CreateSubscriptionRequest struct {
	OrganizationID xid.ID         `json:"organizationId" validate:"required"`
	PlanID         xid.ID         `json:"planId"         validate:"required"`
	Quantity       int            `json:"quantity"       validate:"min=1"`
	StartTrial     bool           `json:"startTrial"`
	TrialDays      int            `json:"trialDays"      validate:"min=0,max=365"`
	Metadata       map[string]any `json:"metadata"`
}

// UpdateSubscriptionRequest represents a request to update a subscription.
type UpdateSubscriptionRequest struct {
	PlanID   *xid.ID        `json:"planId,omitempty"`
	Quantity *int           `json:"quantity,omitempty" validate:"omitempty,min=1"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CancelSubscriptionRequest represents a request to cancel a subscription.
type CancelSubscriptionRequest struct {
	Immediate bool   `json:"immediate"` // If true, cancel immediately; otherwise at period end
	Reason    string `json:"reason"`    // Cancellation reason
}

// PauseSubscriptionRequest represents a request to pause a subscription.
type PauseSubscriptionRequest struct {
	ResumeAt *time.Time `json:"resumeAt"` // Optional date to auto-resume
	Reason   string     `json:"reason"`   // Pause reason
}
