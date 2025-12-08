// Package handlers provides HTTP handlers for the subscription plugin.
package handlers

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// FeatureHandlers handles HTTP requests for feature management
type FeatureHandlers struct {
	featureSvc *service.FeatureService
	usageSvc   *service.FeatureUsageService
}

// NewFeatureHandlers creates a new feature handlers instance
func NewFeatureHandlers(featureSvc *service.FeatureService, usageSvc *service.FeatureUsageService) *FeatureHandlers {
	return &FeatureHandlers{
		featureSvc: featureSvc,
		usageSvc:   usageSvc,
	}
}

// Request/Response types

type createFeatureRequest struct {
	Key          string             `json:"key" validate:"required,min=1,max=100"`
	Name         string             `json:"name" validate:"required,min=1,max=100"`
	Description  string             `json:"description"`
	Type         string             `json:"type" validate:"required"`
	Unit         string             `json:"unit"`
	ResetPeriod  string             `json:"resetPeriod"`
	IsPublic     bool               `json:"isPublic"`
	DisplayOrder int                `json:"displayOrder"`
	Icon         string             `json:"icon"`
	Metadata     map[string]any     `json:"metadata"`
	Tiers        []core.FeatureTier `json:"tiers"`
}

type updateFeatureRequest struct {
	Name         *string            `json:"name"`
	Description  *string            `json:"description"`
	Unit         *string            `json:"unit"`
	ResetPeriod  *string            `json:"resetPeriod"`
	IsPublic     *bool              `json:"isPublic"`
	DisplayOrder *int               `json:"displayOrder"`
	Icon         *string            `json:"icon"`
	Metadata     map[string]any     `json:"metadata"`
	Tiers        []core.FeatureTier `json:"tiers"`
}

type linkFeatureRequest struct {
	FeatureID        string         `json:"featureId" validate:"required"`
	Value            string         `json:"value"`
	IsBlocked        bool           `json:"isBlocked"`
	IsHighlighted    bool           `json:"isHighlighted"`
	OverrideSettings map[string]any `json:"overrideSettings"`
}

type updateLinkRequest struct {
	Value            *string        `json:"value"`
	IsBlocked        *bool          `json:"isBlocked"`
	IsHighlighted    *bool          `json:"isHighlighted"`
	OverrideSettings map[string]any `json:"overrideSettings"`
}

type consumeFeatureRequest struct {
	Quantity       int64          `json:"quantity" validate:"required,min=1"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Reason         string         `json:"reason"`
	Metadata       map[string]any `json:"metadata"`
}

type grantFeatureRequest struct {
	GrantType  string         `json:"grantType" validate:"required"`
	Value      int64          `json:"value" validate:"required,min=1"`
	ExpiresAt  *int64         `json:"expiresAt"` // Unix timestamp
	SourceType string         `json:"sourceType"`
	SourceID   string         `json:"sourceId"`
	Reason     string         `json:"reason"`
	Metadata   map[string]any `json:"metadata"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type successResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Feature CRUD Handlers

// HandleCreateFeature handles feature creation
func (h *FeatureHandlers) HandleCreateFeature(c forge.Context) error {
	var req createFeatureRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	appID := getAppID(c)
	if appID.IsNil() {
		return c.JSON(400, errorResponse{Error: "invalid_app", Message: "app ID required"})
	}

	var resetPeriod core.ResetPeriod
	if req.ResetPeriod != "" {
		resetPeriod = core.ResetPeriod(req.ResetPeriod)
	}

	feature, err := h.featureSvc.Create(c.Context(), appID, &core.CreateFeatureRequest{
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
		Type:         core.FeatureType(req.Type),
		Unit:         req.Unit,
		ResetPeriod:  resetPeriod,
		IsPublic:     req.IsPublic,
		DisplayOrder: req.DisplayOrder,
		Icon:         req.Icon,
		Metadata:     req.Metadata,
		Tiers:        req.Tiers,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, feature)
}

// HandleListFeatures handles listing features
func (h *FeatureHandlers) HandleListFeatures(c forge.Context) error {
	appID := getAppID(c)
	publicOnly := c.Query("public") == "true"
	featureType := c.Query("type")
	page := ctxQueryInt(c, "page", 1)
	pageSize := ctxQueryInt(c, "pageSize", 20)

	features, total, err := h.featureSvc.List(c.Context(), appID, featureType, publicOnly, page, pageSize)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"features": features,
		"total":    total,
		"page":     page,
	})
}

// HandleGetFeature handles getting a single feature
func (h *FeatureHandlers) HandleGetFeature(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	feature, err := h.featureSvc.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, feature)
}

// HandleUpdateFeature handles updating a feature
func (h *FeatureHandlers) HandleUpdateFeature(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	var req updateFeatureRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	var resetPeriod *core.ResetPeriod
	if req.ResetPeriod != nil {
		rp := core.ResetPeriod(*req.ResetPeriod)
		resetPeriod = &rp
	}

	feature, err := h.featureSvc.Update(c.Context(), id, &core.UpdateFeatureRequest{
		Name:         req.Name,
		Description:  req.Description,
		Unit:         req.Unit,
		ResetPeriod:  resetPeriod,
		IsPublic:     req.IsPublic,
		DisplayOrder: req.DisplayOrder,
		Icon:         req.Icon,
		Metadata:     req.Metadata,
		Tiers:        req.Tiers,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, feature)
}

// HandleDeleteFeature handles deleting a feature
func (h *FeatureHandlers) HandleDeleteFeature(c forge.Context) error {
	id, err := xid.FromString(c.Param("id"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	if err := h.featureSvc.Delete(c.Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "feature deleted"})
}

// Plan-Feature Link Handlers

// HandleLinkFeatureToPlan handles linking a feature to a plan
func (h *FeatureHandlers) HandleLinkFeatureToPlan(c forge.Context) error {
	planID, err := xid.FromString(c.Param("planId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	var req linkFeatureRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	featureID, err := xid.FromString(req.FeatureID)
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	link, err := h.featureSvc.LinkToPlan(c.Context(), planID, &core.LinkFeatureRequest{
		FeatureID:        featureID,
		Value:            req.Value,
		IsBlocked:        req.IsBlocked,
		IsHighlighted:    req.IsHighlighted,
		OverrideSettings: req.OverrideSettings,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, link)
}

// HandleGetPlanFeatures handles getting all features for a plan
func (h *FeatureHandlers) HandleGetPlanFeatures(c forge.Context) error {
	planID, err := xid.FromString(c.Param("planId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	links, err := h.featureSvc.GetPlanFeatures(c.Context(), planID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"features": links,
	})
}

// HandleUpdatePlanFeatureLink handles updating a feature-plan link
func (h *FeatureHandlers) HandleUpdatePlanFeatureLink(c forge.Context) error {
	planID, err := xid.FromString(c.Param("planId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	featureID, err := xid.FromString(c.Param("featureId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	var req updateLinkRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	link, err := h.featureSvc.UpdatePlanLink(c.Context(), planID, featureID, &core.UpdateLinkRequest{
		Value:            req.Value,
		IsBlocked:        req.IsBlocked,
		IsHighlighted:    req.IsHighlighted,
		OverrideSettings: req.OverrideSettings,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, link)
}

// HandleUnlinkFeatureFromPlan handles removing a feature from a plan
func (h *FeatureHandlers) HandleUnlinkFeatureFromPlan(c forge.Context) error {
	planID, err := xid.FromString(c.Param("planId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid plan ID"})
	}

	featureID, err := xid.FromString(c.Param("featureId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid feature ID"})
	}

	if err := h.featureSvc.UnlinkFromPlan(c.Context(), planID, featureID); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "feature unlinked from plan"})
}

// Organization Feature Usage Handlers

// HandleGetOrgFeatures handles getting all feature access for an organization
func (h *FeatureHandlers) HandleGetOrgFeatures(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	usages, err := h.usageSvc.GetAllUsage(c.Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"features": usages,
	})
}

// HandleGetFeatureUsage handles getting usage for a specific feature
func (h *FeatureHandlers) HandleGetFeatureUsage(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	featureKey := c.Param("key")
	if featureKey == "" {
		return c.JSON(400, errorResponse{Error: "invalid_key", Message: "feature key required"})
	}

	usage, err := h.usageSvc.GetUsage(c.Context(), orgID, featureKey)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, usage)
}

// HandleCheckFeatureAccess handles checking feature access
func (h *FeatureHandlers) HandleCheckFeatureAccess(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	featureKey := c.Param("key")
	if featureKey == "" {
		return c.JSON(400, errorResponse{Error: "invalid_key", Message: "feature key required"})
	}

	access, err := h.usageSvc.CheckAccess(c.Context(), orgID, featureKey)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, access)
}

// HandleConsumeFeature handles consuming feature quota
func (h *FeatureHandlers) HandleConsumeFeature(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	featureKey := c.Param("key")
	if featureKey == "" {
		return c.JSON(400, errorResponse{Error: "invalid_key", Message: "feature key required"})
	}

	var req consumeFeatureRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	usage, err := h.usageSvc.ConsumeFeature(c.Context(), &core.ConsumeFeatureRequest{
		OrganizationID: orgID,
		FeatureKey:     featureKey,
		Quantity:       req.Quantity,
		IdempotencyKey: req.IdempotencyKey,
		Reason:         req.Reason,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, usage)
}

// HandleGrantFeature handles granting additional feature quota
func (h *FeatureHandlers) HandleGrantFeature(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	featureKey := c.Param("key")
	if featureKey == "" {
		return c.JSON(400, errorResponse{Error: "invalid_key", Message: "feature key required"})
	}

	var req grantFeatureRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_request", Message: err.Error()})
	}

	var sourceID *xid.ID
	if req.SourceID != "" {
		id, err := xid.FromString(req.SourceID)
		if err == nil {
			sourceID = &id
		}
	}

	grant, err := h.usageSvc.GrantFeature(c.Context(), &core.GrantFeatureRequest{
		OrganizationID: orgID,
		FeatureKey:     featureKey,
		GrantType:      core.FeatureGrantType(req.GrantType),
		Value:          req.Value,
		SourceType:     req.SourceType,
		SourceID:       sourceID,
		Reason:         req.Reason,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(201, grant)
}

// HandleListGrants handles listing all grants for an organization
func (h *FeatureHandlers) HandleListGrants(c forge.Context) error {
	orgID, err := xid.FromString(c.Param("orgId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid organization ID"})
	}

	grants, err := h.usageSvc.ListGrants(c.Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]interface{}{
		"grants": grants,
	})
}

// HandleRevokeGrant handles revoking a feature grant
func (h *FeatureHandlers) HandleRevokeGrant(c forge.Context) error {
	grantID, err := xid.FromString(c.Param("grantId"))
	if err != nil {
		return c.JSON(400, errorResponse{Error: "invalid_id", Message: "invalid grant ID"})
	}

	if err := h.usageSvc.RevokeGrant(c.Context(), grantID); err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, successResponse{Success: true, Message: "grant revoked"})
}

// Helper functions

func getAppID(c forge.Context) xid.ID {
	if appID, ok := c.Get("appID").(xid.ID); ok {
		return appID
	}
	if appIDStr := c.Header("X-App-ID"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			return id
		}
	}
	return xid.ID{}
}

func ctxQueryInt(c forge.Context, name string, defaultValue int) int {
	str := c.QueryDefault(name, "")
	if str == "" {
		return defaultValue
	}
	var val int
	if _, err := fmt.Sscanf(str, "%d", &val); err != nil {
		return defaultValue
	}
	return val
}

func handleError(c forge.Context, err error) error {
	if suberrors.IsNotFoundError(err) {
		return c.JSON(404, errorResponse{Error: "not_found", Message: err.Error()})
	}
	if suberrors.IsValidationError(err) {
		return c.JSON(400, errorResponse{Error: "validation_error", Message: err.Error()})
	}
	if suberrors.IsConflictError(err) {
		return c.JSON(409, errorResponse{Error: "conflict", Message: err.Error()})
	}
	if suberrors.IsLimitError(err) {
		return c.JSON(403, errorResponse{Error: "limit_exceeded", Message: err.Error()})
	}
	if suberrors.IsPaymentError(err) {
		return c.JSON(402, errorResponse{Error: "payment_required", Message: err.Error()})
	}

	return c.JSON(500, errorResponse{Error: "internal_error", Message: err.Error()})
}
