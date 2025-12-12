package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	subhooks "github.com/xraph/authsome/plugins/subscription/internal/hooks"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// SubscriptionService handles subscription business logic
type SubscriptionService struct {
	repo         repository.SubscriptionRepository
	planRepo     repository.PlanRepository
	customerRepo repository.CustomerRepository
	customerSvc  *CustomerService
	addOnRepo    repository.AddOnRepository
	provider     providers.PaymentProvider
	eventRepo    repository.EventRepository
	hookRegistry *subhooks.SubscriptionHookRegistry
	config       core.Config
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	repo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
	customerRepo repository.CustomerRepository,
	customerSvc *CustomerService,
	addOnRepo repository.AddOnRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
	hookRegistry *subhooks.SubscriptionHookRegistry,
	config core.Config,
) *SubscriptionService {
	return &SubscriptionService{
		repo:         repo,
		planRepo:     planRepo,
		customerRepo: customerRepo,
		customerSvc:  customerSvc,
		addOnRepo:    addOnRepo,
		provider:     provider,
		eventRepo:    eventRepo,
		hookRegistry: hookRegistry,
		config:       config,
	}
}

// Create creates a new subscription
func (s *SubscriptionService) Create(ctx context.Context, req *core.CreateSubscriptionRequest) (*core.Subscription, error) {
	// Execute before hooks
	if err := s.hookRegistry.ExecuteBeforeSubscriptionCreate(ctx, req.OrganizationID, req.PlanID); err != nil {
		return nil, err
	}

	// Check for existing active subscription
	existing, _ := s.repo.FindByOrganizationID(ctx, req.OrganizationID)
	if existing != nil && (existing.Status == "active" || existing.Status == "trialing") {
		return nil, suberrors.ErrSubscriptionAlreadyExists
	}

	// Get plan
	plan, err := s.planRepo.FindByID(ctx, req.PlanID)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}

	if !plan.IsActive {
		return nil, suberrors.ErrPlanNotActive
	}

	// Create subscription
	now := time.Now()
	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	sub := &schema.Subscription{
		ID:                 xid.New(),
		OrganizationID:     req.OrganizationID,
		PlanID:             req.PlanID,
		Status:             string(core.StatusIncomplete),
		Quantity:           quantity,
		CurrentPeriodStart: now,
	}

	// Calculate period end based on billing interval
	switch plan.BillingInterval {
	case "monthly":
		sub.CurrentPeriodEnd = now.AddDate(0, 1, 0)
	case "yearly":
		sub.CurrentPeriodEnd = now.AddDate(1, 0, 0)
	default:
		sub.CurrentPeriodEnd = now.AddDate(0, 1, 0) // Default to monthly
	}

	sub.CreatedAt = now
	sub.UpdatedAt = now

	if req.Metadata != nil {
		sub.Metadata = req.Metadata
	} else {
		sub.Metadata = make(map[string]interface{})
	}

	// Handle trial
	trialDays := plan.TrialDays
	if req.StartTrial && trialDays > 0 {
		// Use custom trial days if specified
		if req.TrialDays > 0 {
			trialDays = req.TrialDays
		}
		trialEnd := now.AddDate(0, 0, trialDays)
		sub.TrialStart = &now
		sub.TrialEnd = &trialEnd
		sub.Status = string(core.StatusTrialing)
	}

	// Get or create customer for provider subscription (lazy creation)
	customer, err := s.customerSvc.GetOrCreate(ctx, req.OrganizationID, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get or create customer: %w", err)
	}

	// Store customer ID
	sub.ProviderCustomerID = customer.ProviderCustomerID

	// Create subscription in payment provider (Stripe/Paddle/etc)
	if s.provider != nil && plan.ProviderPriceID != "" {
		// Calculate trial days for provider
		providerTrialDays := 0
		if req.StartTrial {
			providerTrialDays = plan.TrialDays
			if req.TrialDays > 0 {
				providerTrialDays = req.TrialDays
			}
		}

		// Prepare metadata for provider
		providerMetadata := map[string]interface{}{
			"authsome":        "true",
			"subscription_id": sub.ID.String(),
			"organization_id": req.OrganizationID.String(),
			"plan_id":         req.PlanID.String(),
		}
		// Merge custom metadata
		for k, v := range req.Metadata {
			providerMetadata[k] = v
		}

		providerSubID, err := s.provider.CreateSubscription(
			ctx,
			customer.ProviderCustomerID,
			plan.ProviderPriceID,
			quantity,
			providerTrialDays,
			providerMetadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription in provider: %w", err)
		}

		sub.ProviderSubID = providerSubID
		// Update status based on trial
		if providerTrialDays > 0 {
			sub.Status = string(core.StatusTrialing)
		} else {
			sub.Status = string(core.StatusActive)
		}
	}

	// Create in database
	if err := s.repo.Create(ctx, sub); err != nil {
		// If database creation fails, we should ideally cancel the provider subscription
		// For now, log and return error
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Convert to domain model
	coreSub := s.schemaToCoreSub(sub)
	coreSub.Plan = s.schemaToPlan(plan)

	// Execute after hooks
	if err := s.hookRegistry.ExecuteAfterSubscriptionCreate(ctx, coreSub); err != nil {
		// Log but don't fail
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, string(core.EventSubscriptionCreated), map[string]interface{}{
		"planId":   req.PlanID.String(),
		"quantity": quantity,
		"trial":    req.StartTrial,
	})

	return coreSub, nil
}

// Update updates a subscription
func (s *SubscriptionService) Update(ctx context.Context, id xid.ID, req *core.UpdateSubscriptionRequest) (*core.Subscription, error) {
	// Execute before hooks
	if err := s.hookRegistry.ExecuteBeforeSubscriptionUpdate(ctx, id, req); err != nil {
		return nil, err
	}

	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	oldPlanID := sub.PlanID

	// Update fields
	if req.PlanID != nil {
		// Validate new plan
		plan, err := s.planRepo.FindByID(ctx, *req.PlanID)
		if err != nil {
			return nil, suberrors.ErrPlanNotFound
		}
		if !plan.IsActive {
			return nil, suberrors.ErrPlanNotActive
		}
		sub.PlanID = *req.PlanID
	}

	if req.Quantity != nil {
		sub.Quantity = *req.Quantity
	}

	if req.Metadata != nil {
		sub.Metadata = req.Metadata
	}

	sub.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Reload with relations
	sub, _ = s.repo.FindByID(ctx, id)
	coreSub := s.schemaToCoreSub(sub)

	// Execute after hooks
	if err := s.hookRegistry.ExecuteAfterSubscriptionUpdate(ctx, coreSub); err != nil {
		// Log but don't fail
	}

	// If plan changed, execute plan change hooks
	if req.PlanID != nil && oldPlanID != *req.PlanID {
		s.hookRegistry.ExecuteAfterPlanChange(ctx, id, oldPlanID, *req.PlanID)
	}

	return coreSub, nil
}

// Cancel cancels a subscription
func (s *SubscriptionService) Cancel(ctx context.Context, id xid.ID, req *core.CancelSubscriptionRequest) error {
	// Execute before hooks
	if err := s.hookRegistry.ExecuteBeforeSubscriptionCancel(ctx, id, req.Immediate); err != nil {
		return err
	}

	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	if sub.Status == string(core.StatusCanceled) {
		return suberrors.ErrSubscriptionCanceled
	}

	now := time.Now()
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if req.Immediate {
		sub.EndedAt = &now
		sub.Status = string(core.StatusCanceled)
	} else {
		// Cancel at period end
		sub.CancelAt = &sub.CurrentPeriodEnd
	}

	// Store cancellation reason
	if sub.Metadata == nil {
		sub.Metadata = make(map[string]interface{})
	}
	sub.Metadata["cancellation_reason"] = req.Reason

	if err := s.repo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Execute after hooks
	s.hookRegistry.ExecuteAfterSubscriptionCancel(ctx, id)

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, string(core.EventSubscriptionCanceled), map[string]interface{}{
		"immediate": req.Immediate,
		"reason":    req.Reason,
	})

	return nil
}

// Pause pauses a subscription
func (s *SubscriptionService) Pause(ctx context.Context, id xid.ID, req *core.PauseSubscriptionRequest) error {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	if sub.Status != string(core.StatusActive) {
		return fmt.Errorf("subscription must be active to pause")
	}

	now := time.Now()
	sub.PausedAt = &now
	sub.ResumeAt = req.ResumeAt
	sub.Status = string(core.StatusPaused)
	sub.UpdatedAt = now

	if sub.Metadata == nil {
		sub.Metadata = make(map[string]interface{})
	}
	sub.Metadata["pause_reason"] = req.Reason

	if err := s.repo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to pause subscription: %w", err)
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, string(core.EventSubscriptionPaused), map[string]interface{}{
		"reason":   req.Reason,
		"resumeAt": req.ResumeAt,
	})

	return nil
}

// Resume resumes a paused subscription
func (s *SubscriptionService) Resume(ctx context.Context, id xid.ID) error {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	if sub.Status != string(core.StatusPaused) {
		return fmt.Errorf("subscription is not paused")
	}

	now := time.Now()
	sub.PausedAt = nil
	sub.ResumeAt = nil
	sub.Status = string(core.StatusActive)
	sub.UpdatedAt = now

	if err := s.repo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to resume subscription: %w", err)
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, string(core.EventSubscriptionResumed), nil)

	return nil
}

// GetByID retrieves a subscription by ID
func (s *SubscriptionService) GetByID(ctx context.Context, id xid.ID) (*core.Subscription, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}
	return s.schemaToCoreSub(sub), nil
}

// GetByOrganizationID retrieves the active subscription for an organization
func (s *SubscriptionService) GetByOrganizationID(ctx context.Context, orgID xid.ID) (*core.Subscription, error) {
	sub, err := s.repo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}
	return s.schemaToCoreSub(sub), nil
}

// List retrieves subscriptions with filtering
func (s *SubscriptionService) List(ctx context.Context, appID, orgID, planID *xid.ID, status string, page, pageSize int) ([]*core.Subscription, int, error) {
	filter := &repository.SubscriptionFilter{
		AppID:          appID,
		OrganizationID: orgID,
		PlanID:         planID,
		Status:         status,
		Page:           page,
		PageSize:       pageSize,
	}

	subs, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	result := make([]*core.Subscription, len(subs))
	for i, sub := range subs {
		result[i] = s.schemaToCoreSub(sub)
	}

	return result, count, nil
}

// ChangePlan changes the subscription plan
func (s *SubscriptionService) ChangePlan(ctx context.Context, id, newPlanID xid.ID) (*core.Subscription, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	oldPlanID := sub.PlanID

	// Execute before hooks
	if err := s.hookRegistry.ExecuteBeforePlanChange(ctx, id, oldPlanID, newPlanID); err != nil {
		return nil, err
	}

	// Get new plan
	plan, err := s.planRepo.FindByID(ctx, newPlanID)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}

	if !plan.IsActive {
		return nil, suberrors.ErrPlanNotActive
	}

	// Update plan
	sub.PlanID = newPlanID
	sub.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to change plan: %w", err)
	}

	// Reload
	sub, _ = s.repo.FindByID(ctx, id)
	coreSub := s.schemaToCoreSub(sub)

	// Execute after hooks
	s.hookRegistry.ExecuteAfterPlanChange(ctx, id, oldPlanID, newPlanID)

	return coreSub, nil
}

// UpdateQuantity updates the subscription quantity
func (s *SubscriptionService) UpdateQuantity(ctx context.Context, id xid.ID, quantity int) (*core.Subscription, error) {
	if quantity <= 0 {
		return nil, suberrors.ErrInvalidQuantity
	}

	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	sub.Quantity = quantity
	sub.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update quantity: %w", err)
	}

	sub, _ = s.repo.FindByID(ctx, id)
	return s.schemaToCoreSub(sub), nil
}

// AttachAddOn attaches an add-on to a subscription
func (s *SubscriptionService) AttachAddOn(ctx context.Context, subID, addOnID xid.ID, quantity int) error {
	// Get subscription
	sub, err := s.repo.FindByID(ctx, subID)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	// Get add-on
	addOn, err := s.addOnRepo.FindByID(ctx, addOnID)
	if err != nil {
		return fmt.Errorf("add-on not found")
	}

	// Check existing
	items, _ := s.repo.GetAddOnItems(ctx, subID)
	for _, item := range items {
		if item.AddOnID == addOnID {
			return suberrors.ErrAddOnAlreadyAttached
		}
	}

	// Create in provider first (if subscription is synced)
	var providerItemID string
	if sub.ProviderSubID != "" && addOn.ProviderPriceID != "" && s.provider != nil {
		providerItemID, err = s.provider.AddSubscriptionItem(
			ctx,
			sub.ProviderSubID,
			addOn.ProviderPriceID,
			quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to add item to provider: %w", err)
		}
	}

	// Create local record
	item := &schema.SubscriptionAddOnItem{
		ID:                xid.New(),
		SubscriptionID:    subID,
		AddOnID:           addOnID,
		Quantity:          quantity,
		ProviderSubItemID: providerItemID,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateAddOnItem(ctx, item); err != nil {
		// Rollback provider change if local save fails
		if providerItemID != "" && s.provider != nil {
			s.provider.RemoveSubscriptionItem(ctx, sub.ProviderSubID, providerItemID)
		}
		return fmt.Errorf("failed to create add-on item: %w", err)
	}

	// Record event
	s.recordEvent(ctx, subID, sub.OrganizationID, "addon.attached", map[string]interface{}{
		"addOnId":  addOnID.String(),
		"quantity": quantity,
	})

	return nil
}

// DetachAddOn detaches an add-on from a subscription
func (s *SubscriptionService) DetachAddOn(ctx context.Context, subID, addOnID xid.ID) error {
	// Get the subscription
	sub, err := s.repo.FindByID(ctx, subID)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	// Get the add-on items
	items, err := s.repo.GetAddOnItems(ctx, subID)
	if err != nil {
		return fmt.Errorf("failed to get add-on items: %w", err)
	}

	var item *schema.SubscriptionAddOnItem
	for _, i := range items {
		if i.AddOnID == addOnID {
			item = i
			break
		}
	}

	if item == nil {
		return fmt.Errorf("add-on not attached to subscription")
	}

	// Remove from provider if synced
	if item.ProviderSubItemID != "" && sub.ProviderSubID != "" && s.provider != nil {
		if err := s.provider.RemoveSubscriptionItem(ctx, sub.ProviderSubID, item.ProviderSubItemID); err != nil {
			// Log but don't fail - we'll clean up local anyway
			// In production, you might want to retry or queue this
		}
	}

	// Remove local record
	if err := s.repo.DeleteAddOnItem(ctx, subID, addOnID); err != nil {
		return fmt.Errorf("failed to delete add-on item: %w", err)
	}

	// Record event
	s.recordEvent(ctx, subID, sub.OrganizationID, "addon.detached", map[string]interface{}{
		"addOnId": addOnID.String(),
	})

	return nil
}

// SyncFromProvider syncs subscription data from the provider
func (s *SubscriptionService) SyncFromProvider(ctx context.Context, providerSubID string) (*core.Subscription, error) {
	sub, err := s.repo.FindByProviderID(ctx, providerSubID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Fetch from provider
	if s.provider == nil {
		return nil, fmt.Errorf("provider not available")
	}

	providerSub, err := s.provider.GetSubscription(ctx, providerSubID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription from provider: %w", err)
	}

	// Update local data with provider data
	sub.Status = providerSub.Status
	sub.Quantity = providerSub.Quantity
	sub.CurrentPeriodStart = time.Unix(providerSub.CurrentPeriodStart, 0)
	sub.CurrentPeriodEnd = time.Unix(providerSub.CurrentPeriodEnd, 0)

	if providerSub.TrialStart != nil {
		trialStart := time.Unix(*providerSub.TrialStart, 0)
		sub.TrialStart = &trialStart
	}
	if providerSub.TrialEnd != nil {
		trialEnd := time.Unix(*providerSub.TrialEnd, 0)
		sub.TrialEnd = &trialEnd
	}
	if providerSub.CancelAt != nil {
		cancelAt := time.Unix(*providerSub.CancelAt, 0)
		sub.CancelAt = &cancelAt
	}
	if providerSub.CanceledAt != nil {
		canceledAt := time.Unix(*providerSub.CanceledAt, 0)
		sub.CanceledAt = &canceledAt
	}
	if providerSub.EndedAt != nil {
		endedAt := time.Unix(*providerSub.EndedAt, 0)
		sub.EndedAt = &endedAt
	}

	sub.UpdatedAt = time.Now()

	// Update database
	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, "subscription.synced_from_provider", map[string]interface{}{
		"providerSubId": providerSubID,
		"status":        providerSub.Status,
	})

	return s.schemaToCoreSub(sub), nil
}

// SyncFromProviderByID syncs subscription data from provider using local subscription ID
func (s *SubscriptionService) SyncFromProviderByID(ctx context.Context, id xid.ID) (*core.Subscription, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	if sub.ProviderSubID == "" {
		return nil, fmt.Errorf("subscription not synced to provider yet")
	}

	return s.SyncFromProvider(ctx, sub.ProviderSubID)
}

// SyncToProvider syncs a subscription to the payment provider
func (s *SubscriptionService) SyncToProvider(ctx context.Context, id xid.ID) error {
	// Get subscription
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	// Get or create customer
	customer, err := s.customerSvc.GetOrCreate(ctx, sub.OrganizationID, "", "")
	if err != nil {
		return fmt.Errorf("failed to get or create customer: %w", err)
	}

	// Get plan
	plan, err := s.planRepo.FindByID(ctx, sub.PlanID)
	if err != nil {
		return fmt.Errorf("plan not found")
	}

	// Check if plan is synced
	if plan.ProviderPriceID == "" {
		return fmt.Errorf("plan not synced to provider - sync plan first")
	}

	// If already synced, return
	if sub.ProviderSubID != "" {
		return nil // Already synced
	}

	// Calculate trial days for provider
	providerTrialDays := 0
	if sub.TrialEnd != nil && sub.TrialStart != nil {
		trialDuration := sub.TrialEnd.Sub(*sub.TrialStart)
		providerTrialDays = int(trialDuration.Hours() / 24)
	}

	// Prepare metadata
	metadata := map[string]interface{}{
		"authsome":        "true",
		"subscription_id": sub.ID.String(),
		"organization_id": sub.OrganizationID.String(),
		"plan_id":         sub.PlanID.String(),
	}
	if sub.Metadata != nil {
		for k, v := range sub.Metadata {
			metadata[k] = v
		}
	}

	// Create in provider
	providerSubID, err := s.provider.CreateSubscription(
		ctx,
		customer.ProviderCustomerID,
		plan.ProviderPriceID,
		sub.Quantity,
		providerTrialDays,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription in provider: %w", err)
	}

	// Update local record with provider ID
	sub.ProviderSubID = providerSubID
	sub.ProviderCustomerID = customer.ProviderCustomerID

	// Update status based on trial
	if providerTrialDays > 0 {
		sub.Status = string(core.StatusTrialing)
	} else {
		sub.Status = string(core.StatusActive)
	}

	if err := s.repo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, "subscription.synced", map[string]interface{}{
		"providerSubId": providerSubID,
	})

	return nil
}

// Helper methods

func (s *SubscriptionService) recordEvent(ctx context.Context, subID, orgID xid.ID, eventType string, data map[string]interface{}) {
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

func (s *SubscriptionService) schemaToCoreSub(sub *schema.Subscription) *core.Subscription {
	coreSub := &core.Subscription{
		ID:                 sub.ID,
		OrganizationID:     sub.OrganizationID,
		PlanID:             sub.PlanID,
		Status:             core.SubscriptionStatus(sub.Status),
		Quantity:           sub.Quantity,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		TrialStart:         sub.TrialStart,
		TrialEnd:           sub.TrialEnd,
		CancelAt:           sub.CancelAt,
		CanceledAt:         sub.CanceledAt,
		EndedAt:            sub.EndedAt,
		PausedAt:           sub.PausedAt,
		ResumeAt:           sub.ResumeAt,
		ProviderSubID:      sub.ProviderSubID,
		ProviderCustomerID: sub.ProviderCustomerID,
		Metadata:           sub.Metadata,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
	}

	if sub.Plan != nil {
		coreSub.Plan = s.schemaToPlan(sub.Plan)
	}

	return coreSub
}

func (s *SubscriptionService) schemaToPlan(plan *schema.SubscriptionPlan) *core.Plan {
	return &core.Plan{
		ID:              plan.ID,
		AppID:           plan.AppID,
		Name:            plan.Name,
		Slug:            plan.Slug,
		Description:     plan.Description,
		BillingPattern:  core.BillingPattern(plan.BillingPattern),
		BillingInterval: core.BillingInterval(plan.BillingInterval),
		BasePrice:       plan.BasePrice,
		Currency:        plan.Currency,
		TrialDays:       plan.TrialDays,
		IsActive:        plan.IsActive,
		IsPublic:        plan.IsPublic,
	}
}
