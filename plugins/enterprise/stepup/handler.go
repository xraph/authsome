package stepup

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles step-up HTTP requests.
type Handler struct {
	service *Service
	config  *Config
}

// Response types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

type RequirementsResponse struct {
	Requirements any `json:"requirements"`
	Count        int `json:"count"`
}

type VerificationsResponse struct {
	Verifications any `json:"verifications"`
	Count         int `json:"count"`
}

type StepUpDevicesResponse struct {
	Devices any `json:"devices"`
	Count   int `json:"count"`
}

type ForgetDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// NewHandler creates a new step-up handler.
func NewHandler(service *Service, config *Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// EvaluateRequest is the request for evaluating step-up requirements.
type EvaluateRequest struct {
	Route        string         `json:"route,omitempty"`
	Method       string         `json:"method,omitempty"`
	Amount       float64        `json:"amount,omitempty"`
	Currency     string         `json:"currency,omitempty"`
	ResourceType string         `json:"resource_type,omitempty"`
	Action       string         `json:"action,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Evaluate handles POST /stepup/evaluate.
func (h *Handler) Evaluate(c forge.Context) error {
	var req EvaluateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	// Extract user context
	userID := c.Get("user_id")
	orgID := c.Get("org_id")
	sessionID := c.Get("session_id")

	if userID == nil || userID == "" {
		return c.JSON(401, errs.Unauthorized())
	}

	// Build evaluation context
	evalCtx := &EvaluationContext{
		UserID:       userID.(string),
		OrgID:        getStringOrEmpty(orgID),
		SessionID:    getStringOrEmpty(sessionID),
		Route:        req.Route,
		Method:       req.Method,
		Amount:       req.Amount,
		Currency:     req.Currency,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		IP:           c.Request().RemoteAddr,
		UserAgent:    c.Request().Header.Get("User-Agent"),
		DeviceID:     extractDeviceID(c),
		Metadata:     req.Metadata,
	}

	result, err := h.service.EvaluateRequirement(c.Request().Context(), evalCtx)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to evaluate requirement"))
	}

	return c.JSON(200, result)
}

// Verify handles POST /stepup/verify.
func (h *Handler) Verify(c forge.Context) error {
	var req VerifyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	// Set IP and User Agent from request
	req.IP = c.Request().RemoteAddr
	req.UserAgent = c.Request().Header.Get("User-Agent")

	// Extract device ID if not provided
	if req.DeviceID == "" {
		req.DeviceID = extractDeviceID(c)
	}

	response, err := h.service.VerifyStepUp(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Verification failed"))
	}

	if !response.Success {
		return c.JSON(401, response)
	}

	return c.JSON(200, response)
}

// GetRequirement handles GET /stepup/requirements/:id.
func (h *Handler) GetRequirement(c forge.Context) error {
	requirementID := c.Param("id")
	if requirementID == "" {
		return c.JSON(400, errs.RequiredField("requirement_id"))
	}

	requirement, err := h.service.repo.GetRequirement(c.Request().Context(), requirementID)
	if err != nil {
		return c.JSON(404, errs.NotFound("Requirement not found"))
	}

	// Verify ownership
	userID := c.Get("user_id")
	if userID == nil || requirement.UserID != userID.(string) {
		return c.JSON(403, errs.PermissionDenied("access", "step-up"))
	}

	return c.JSON(200, requirement)
}

// ListPendingRequirements handles GET /stepup/requirements/pending.
func (h *Handler) ListPendingRequirements(c forge.Context) error {
	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	requirements, err := h.service.repo.ListPendingRequirements(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
	)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to list requirements"))
	}

	return c.JSON(200, &RequirementsResponse{Requirements: requirements, Count: len(requirements)})
}

// ListVerifications handles GET /stepup/verifications.
func (h *Handler) ListVerifications(c forge.Context) error {
	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	// Parse pagination
	limit := 20
	offset := 0

	if l := c.Request().URL.Query().Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Request().URL.Query().Get("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	verifications, err := h.service.repo.ListVerifications(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
		limit,
		offset,
	)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to list verifications"))
	}

	return c.JSON(200, map[string]any{
		"verifications": verifications,
		"count":         len(verifications),
		"limit":         limit,
		"offset":        offset,
	})
}

// ListRememberedDevices handles GET /stepup/devices.
func (h *Handler) ListRememberedDevices(c forge.Context) error {
	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	devices, err := h.service.repo.ListRememberedDevices(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
	)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to list remembered devices"))
	}

	return c.JSON(200, &StepUpDevicesResponse{Devices: devices, Count: len(devices)})
}

// ForgetDevice handles DELETE /stepup/devices/:id.
func (h *Handler) ForgetDevice(c forge.Context) error {
	deviceID := c.Param("id")
	if deviceID == "" {
		return c.JSON(400, errs.RequiredField("device_id"))
	}

	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	err := h.service.ForgetDevice(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
		deviceID,
	)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, &ForgetDeviceResponse{Success: true, Message: "Device forgotten successfully"})
}

// CreatePolicy handles POST /stepup/policies.
func (h *Handler) CreatePolicy(c forge.Context) error {
	var policy StepUpPolicy
	if err := json.NewDecoder(c.Request().Body).Decode(&policy); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	// Set org ID from context
	orgID := c.Get("org_id")
	if orgID == nil {
		return c.JSON(401, errs.UnauthorizedWithMessage("Organization context required"))
	}

	policy.OrgID = orgID.(string)

	// Generate ID if not provided
	if policy.ID == "" {
		policy.ID = generateID()
	}

	if err := h.service.repo.CreatePolicy(c.Request().Context(), &policy); err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to create policy"))
	}

	return c.JSON(201, policy)
}

// ListPolicies handles GET /stepup/policies.
func (h *Handler) ListPolicies(c forge.Context) error {
	orgID := c.Get("org_id")
	if orgID == nil {
		return c.JSON(401, errs.UnauthorizedWithMessage("Organization context required"))
	}

	policies, err := h.service.repo.ListPolicies(c.Request().Context(), orgID.(string))
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to list policies"))
	}

	return c.JSON(200, map[string]any{
		"policies": policies,
		"count":    len(policies),
	})
}

// GetPolicy handles GET /stepup/policies/:id.
func (h *Handler) GetPolicy(c forge.Context) error {
	policyID := c.Param("id")
	if policyID == "" {
		return c.JSON(400, errs.RequiredField("policy_id"))
	}

	policy, err := h.service.repo.GetPolicy(c.Request().Context(), policyID)
	if err != nil {
		return c.JSON(404, errs.NotFound("Policy not found"))
	}

	// Verify organization access
	orgID := c.Get("org_id")
	if orgID == nil || policy.OrgID != orgID.(string) {
		return c.JSON(403, errs.PermissionDenied("access", "step-up"))
	}

	return c.JSON(200, policy)
}

// UpdatePolicy handles PUT /stepup/policies/:id.
func (h *Handler) UpdatePolicy(c forge.Context) error {
	policyID := c.Param("id")
	if policyID == "" {
		return c.JSON(400, errs.RequiredField("policy_id"))
	}

	// Get existing policy
	existing, err := h.service.repo.GetPolicy(c.Request().Context(), policyID)
	if err != nil {
		return c.JSON(404, errs.NotFound("Policy not found"))
	}

	// Verify organization access
	orgID := c.Get("org_id")
	if orgID == nil || existing.OrgID != orgID.(string) {
		return c.JSON(403, errs.PermissionDenied("access", "step-up"))
	}

	// Decode updates
	var updates StepUpPolicy
	if err := json.NewDecoder(c.Request().Body).Decode(&updates); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	// Preserve immutable fields
	updates.ID = existing.ID
	updates.OrgID = existing.OrgID
	updates.CreatedAt = existing.CreatedAt

	if err := h.service.repo.UpdatePolicy(c.Request().Context(), &updates); err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to update policy"))
	}

	return c.JSON(200, updates)
}

// DeletePolicy handles DELETE /stepup/policies/:id.
func (h *Handler) DeletePolicy(c forge.Context) error {
	policyID := c.Param("id")
	if policyID == "" {
		return c.JSON(400, errs.RequiredField("policy_id"))
	}

	// Get existing policy
	existing, err := h.service.repo.GetPolicy(c.Request().Context(), policyID)
	if err != nil {
		return c.JSON(404, errs.NotFound("Policy not found"))
	}

	// Verify organization access
	orgID := c.Get("org_id")
	if orgID == nil || existing.OrgID != orgID.(string) {
		return c.JSON(403, errs.PermissionDenied("access", "step-up"))
	}

	if err := h.service.repo.DeletePolicy(c.Request().Context(), policyID); err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to delete policy"))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"message": "Policy deleted successfully",
	})
}

// GetAuditLogs handles GET /stepup/audit.
func (h *Handler) GetAuditLogs(c forge.Context) error {
	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	// Parse pagination
	limit := 50
	offset := 0

	if l := c.Request().URL.Query().Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Request().URL.Query().Get("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	logs, err := h.service.repo.ListAuditLogs(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
		limit,
		offset,
	)
	if err != nil {
		return c.JSON(500, errs.InternalServerErrorWithMessage("Failed to list audit logs"))
	}

	return c.JSON(200, map[string]any{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}

// Status handles GET /stepup/status.
func (h *Handler) Status(c forge.Context) error {
	userID := c.Get("user_id")
	orgID := c.Get("org_id")

	if userID == nil {
		return c.JSON(401, errs.Unauthorized())
	}

	// Build evaluation context for current request
	evalCtx := &EvaluationContext{
		UserID:    userID.(string),
		OrgID:     getStringOrEmpty(orgID),
		Route:     c.Request().URL.Path,
		Method:    c.Request().Method,
		IP:        c.Request().RemoteAddr,
		UserAgent: c.Request().Header.Get("User-Agent"),
		DeviceID:  extractDeviceID(c),
	}

	currentLevel := h.service.determineCurrentLevel(c.Request().Context(), evalCtx)

	// Get pending requirements
	pending, _ := h.service.repo.ListPendingRequirements(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
	)

	// Get remembered devices
	devices, _ := h.service.repo.ListRememberedDevices(
		c.Request().Context(),
		userID.(string),
		getStringOrEmpty(orgID),
	)

	return c.JSON(200, map[string]any{
		"enabled":            h.config.Enabled,
		"current_level":      currentLevel,
		"pending_count":      len(pending),
		"remembered_devices": len(devices),
		"remember_enabled":   h.config.RememberStepUp,
		"risk_based_enabled": h.config.RiskBasedEnabled,
	})
}

// Helper functions

func getStringOrEmpty(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	return ""
}

func extractDeviceID(c forge.Context) string {
	// Check cookie first
	if cookie, err := c.Request().Cookie("device_id"); err == nil {
		return cookie.Value
	}

	// Check header
	if deviceID := c.Request().Header.Get("X-Device-Id"); deviceID != "" {
		return deviceID
	}

	// Check context
	if deviceID := c.Get("device_id"); deviceID != nil {
		if s, ok := deviceID.(string); ok {
			return s
		}
	}

	return ""
}

func parseIntParam(s string) (int, error) {
	var i int

	_, err := fmt.Sscanf(s, "%d", &i)

	return i, err
}

func generateID() string {
	// Simple ID generation - in production use UUID
	return fmt.Sprintf("pol_%d", time.Now().UnixNano())
}
