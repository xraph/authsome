// Package handlers provides HTTP handlers for the subscription plugin.
package handlers

import (
	"encoding/json"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// PublicHandlers handles HTTP requests for public pricing APIs.
type PublicHandlers struct {
	featureSvc *service.FeatureService
	planSvc    *service.PlanService
}

// NewPublicHandlers creates a new public handlers instance.
func NewPublicHandlers(featureSvc *service.FeatureService, planSvc *service.PlanService) *PublicHandlers {
	return &PublicHandlers{
		featureSvc: featureSvc,
		planSvc:    planSvc,
	}
}

// PublicPlan represents a plan in public API format.
type PublicPlan struct {
	ID              string                    `json:"id"`
	Slug            string                    `json:"slug"`
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	BillingPattern  string                    `json:"billingPattern"`
	BillingInterval string                    `json:"billingInterval"`
	BasePrice       int64                     `json:"basePrice"`
	Currency        string                    `json:"currency"`
	TrialDays       int                       `json:"trialDays"`
	DisplayOrder    int                       `json:"displayOrder"`
	Features        []*core.PublicPlanFeature `json:"features,omitempty"`
	Metadata        map[string]any            `json:"metadata,omitempty"`
}

// HandleListPublicPlans handles listing public plans with features.
func (h *PublicHandlers) HandleListPublicPlans(c forge.Context) error {
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	// Get public plans
	plans, _, err := h.planSvc.List(c.Context(), appID, true, true, 1, 100)
	if err != nil {
		return handleError(c, err)
	}

	result := make([]*PublicPlan, 0, len(plans))
	for _, p := range plans {
		publicPlan := &PublicPlan{
			ID:              p.ID.String(),
			Slug:            p.Slug,
			Name:            p.Name,
			Description:     p.Description,
			BillingPattern:  string(p.BillingPattern),
			BillingInterval: string(p.BillingInterval),
			BasePrice:       p.BasePrice,
			Currency:        p.Currency,
			TrialDays:       p.TrialDays,
			DisplayOrder:    p.DisplayOrder,
			Metadata:        filterPublicMetadata(p.Metadata),
		}

		// Get features for this plan
		features, err := h.featureSvc.GetPublicPlanFeatures(c.Context(), p.ID)
		if err == nil {
			publicPlan.Features = features
		}

		result = append(result, publicPlan)
	}

	return c.JSON(200, map[string]any{
		"plans": result,
	})
}

// HandleGetPublicPlan handles getting a single public plan by slug.
func (h *PublicHandlers) HandleGetPublicPlan(c forge.Context) error {
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(400, errorResponse{Error: "invalid_slug", Message: "plan slug required"})
	}

	plan, err := h.planSvc.GetBySlug(c.Context(), appID, slug)
	if err != nil {
		return handleError(c, err)
	}

	// Check if plan is public
	if !plan.IsPublic || !plan.IsActive {
		return c.JSON(404, errorResponse{Error: "not_found", Message: "plan not found"})
	}

	publicPlan := &PublicPlan{
		ID:              plan.ID.String(),
		Slug:            plan.Slug,
		Name:            plan.Name,
		Description:     plan.Description,
		BillingPattern:  string(plan.BillingPattern),
		BillingInterval: string(plan.BillingInterval),
		BasePrice:       plan.BasePrice,
		Currency:        plan.Currency,
		TrialDays:       plan.TrialDays,
		DisplayOrder:    plan.DisplayOrder,
		Metadata:        filterPublicMetadata(plan.Metadata),
	}

	// Get features for this plan
	features, err := h.featureSvc.GetPublicPlanFeatures(c.Context(), plan.ID)
	if err == nil {
		publicPlan.Features = features
	}

	return c.JSON(200, publicPlan)
}

// HandleGetPublicPlanFeatures handles getting features for a public plan.
func (h *PublicHandlers) HandleGetPublicPlanFeatures(c forge.Context) error {
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(400, errorResponse{Error: "invalid_slug", Message: "plan slug required"})
	}

	plan, err := h.planSvc.GetBySlug(c.Context(), appID, slug)
	if err != nil {
		return handleError(c, err)
	}

	// Check if plan is public
	if !plan.IsPublic || !plan.IsActive {
		return c.JSON(404, errorResponse{Error: "not_found", Message: "plan not found"})
	}

	features, err := h.featureSvc.GetPublicPlanFeatures(c.Context(), plan.ID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"features": features,
	})
}

// HandleListPublicFeatures handles listing all public features.
func (h *PublicHandlers) HandleListPublicFeatures(c forge.Context) error {
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	features, err := h.featureSvc.GetPublicFeatures(c.Context(), appID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"features": features,
	})
}

// HandleComparePlans handles comparing features across plans.
func (h *PublicHandlers) HandleComparePlans(c forge.Context) error {
	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	// Parse plan IDs from query
	planIDsStr := c.Query("planIds")
	if planIDsStr == "" {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: "planIds query parameter required"})
	}

	var planIDs []string
	if err := json.Unmarshal([]byte(planIDsStr), &planIDs); err != nil {
		// planIDs comma-separated
		planIDs = splitString(planIDsStr, ",")
	}

	if len(planIDs) == 0 {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: "at least one plan ID required"})
	}

	// Get all public features
	allFeatures, err := h.featureSvc.GetPublicFeatures(c.Context(), appID)
	if err != nil {
		return handleError(c, err)
	}

	// Build comparison matrix
	comparison := make(map[string]map[string]any)
	for _, f := range allFeatures {
		comparison[f.Key] = map[string]any{
			"name":        f.Name,
			"description": f.Description,
			"type":        f.Type,
			"unit":        f.Unit,
			"plans":       make(map[string]any),
		}
	}

	// Get features for each plan
	for _, planIDStr := range planIDs {
		planID, err := xid.FromString(planIDStr)
		if err != nil {
			continue
		}

		features, err := h.featureSvc.GetPublicPlanFeatures(c.Context(), planID)
		if err != nil {
			continue
		}

		for _, f := range features {
			if featureData, ok := comparison[f.Key]; ok {
				plans := featureData["plans"].(map[string]any)
				plans[planIDStr] = map[string]any{
					"value":         f.Value,
					"isHighlighted": f.IsHighlighted,
					"isBlocked":     f.IsBlocked,
				}
			}
		}
	}

	return c.JSON(200, map[string]any{
		"comparison": comparison,
	})
}

// Helper functions

func filterPublicMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	// Filter out private metadata fields (those starting with _)
	result := make(map[string]any)

	for k, v := range metadata {
		if len(k) > 0 && k[0] != '_' {
			result[k] = v
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}

	result := make([]string, 0)
	start := 0

	for i := range len(s) {
		if s[i] == sep[0] {
			if start < i {
				result = append(result, s[start:i])
			}

			start = i + 1
		}
	}

	if start < len(s) {
		result = append(result, s[start:])
	}

	return result
}
