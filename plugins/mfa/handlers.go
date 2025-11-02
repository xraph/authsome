package mfa

import (
	"encoding/json"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/forge"
)

// Handler provides HTTP endpoints for MFA operations
type Handler struct {
	service *Service
}

// NewHandler creates a new MFA handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ==================== Factor Management ====================

// EnrollFactor handles POST /mfa/factors/enroll
func (h *Handler) EnrollFactor(c forge.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	var req FactorEnrollmentRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	
	resp, err := h.service.EnrollFactor(c.Request().Context(), userID, &req)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, resp)
}

// ListFactors handles GET /mfa/factors
func (h *Handler) ListFactors(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	// Check for active_only query param
	activeOnly := c.Request().URL.Query().Get("active_only") == "true"
	
	factors, err := h.service.ListFactors(c.Request().Context(), userID, activeOnly)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]interface{}{
		"factors": factors,
		"count":   len(factors),
	})
}

// GetFactor handles GET /mfa/factors/:id
func (h *Handler) GetFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	factorIDStr := c.Param("id")
	factorID, err := xid.FromString(factorIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid factor ID"})
	}
	
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}
	
	// Verify factor belongs to user
	if factor.UserID != userID {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	
	return c.JSON(200, factor)
}

// UpdateFactor handles PUT /mfa/factors/:id
func (h *Handler) UpdateFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	factorIDStr := c.Param("id")
	factorID, err := xid.FromString(factorIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid factor ID"})
	}
	
	// First verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}
	if factor.UserID != userID {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	
	var updates map[string]interface{}
	if err := json.NewDecoder(c.Request().Body).Decode(&updates); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	
	if err := h.service.UpdateFactor(c.Request().Context(), factorID, updates); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]string{"message": "factor updated"})
}

// DeleteFactor handles DELETE /mfa/factors/:id
func (h *Handler) DeleteFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	factorIDStr := c.Param("id")
	factorID, err := xid.FromString(factorIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid factor ID"})
	}
	
	// Verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}
	if factor.UserID != userID {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	
	if err := h.service.DeleteFactor(c.Request().Context(), factorID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]string{"message": "factor deleted"})
}

// VerifyFactor handles POST /mfa/factors/:id/verify
func (h *Handler) VerifyFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	factorIDStr := c.Param("id")
	factorID, err := xid.FromString(factorIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid factor ID"})
	}
	
	// Verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "factor not found"})
	}
	if factor.UserID != userID {
		return c.JSON(403, map[string]string{"error": "forbidden"})
	}
	
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	
	if err := h.service.VerifyEnrollment(c.Request().Context(), factorID, req.Code); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]string{"message": "factor verified"})
}

// ==================== Challenge & Verification ====================

// InitiateChallenge handles POST /mfa/challenge
func (h *Handler) InitiateChallenge(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	var req ChallengeRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		// Use empty request if no body provided
		req = ChallengeRequest{}
	}
	
	// Override userID from context
	req.UserID = userID
	
	// Add IP and user agent to metadata
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	req.Metadata["ip_address"] = c.Request().RemoteAddr
	req.Metadata["user_agent"] = c.Request().UserAgent()
	
	resp, err := h.service.InitiateChallenge(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, resp)
}

// VerifyChallenge handles POST /mfa/verify
func (h *Handler) VerifyChallenge(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	var req VerificationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	
	// Verify challenge belongs to user (done in service)
	resp, err := h.service.VerifyChallenge(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	_ = userID // Verify in service layer
	
	return c.JSON(200, resp)
}

// GetChallengeStatus handles GET /mfa/challenge/:id
func (h *Handler) GetChallengeStatus(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	_ = userID // TODO: Implement challenge status lookup
	
	return c.JSON(501, map[string]string{"error": "not implemented"})
}

// ==================== Trusted Devices ====================

// TrustDevice handles POST /mfa/devices/trust
func (h *Handler) TrustDevice(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	var deviceInfo DeviceInfo
	if err := json.NewDecoder(c.Request().Body).Decode(&deviceInfo); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	
	if err := h.service.TrustDevice(c.Request().Context(), userID, &deviceInfo); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]string{"message": "device trusted"})
}

// ListTrustedDevices handles GET /mfa/devices
func (h *Handler) ListTrustedDevices(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	devices, err := h.service.ListTrustedDevices(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// RevokeTrustedDevice handles DELETE /mfa/devices/:id
func (h *Handler) RevokeTrustedDevice(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	deviceIDStr := c.Param("id")
	deviceID, err := xid.FromString(deviceIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid device ID"})
	}
	
	// TODO: Verify device belongs to user
	_ = userID
	
	if err := h.service.RevokeTrustedDevice(c.Request().Context(), deviceID); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, map[string]string{"message": "device revoked"})
}

// ==================== Status & Info ====================

// GetStatus handles GET /mfa/status
func (h *Handler) GetStatus(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	
	deviceID := c.Request().URL.Query().Get("device_id")
	
	status, err := h.service.GetMFAStatus(c.Request().Context(), userID, deviceID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(200, status)
}

// GetPolicy handles GET /mfa/policy
func (h *Handler) GetPolicy(c forge.Context) error {
	// TODO: Get organization ID from context
	// For now, return config-based policy
	
	return c.JSON(200, map[string]interface{}{
		"enabled":               h.service.config.Enabled,
		"required_factor_count": h.service.config.RequiredFactorCount,
		"allowed_factor_types":  h.service.config.AllowedFactorTypes,
		"grace_period_days":     h.service.config.GracePeriodDays,
		"adaptive_mfa_enabled":  h.service.config.AdaptiveMFA.Enabled,
	})
}

// ==================== Admin Endpoints ====================

// UpdatePolicy handles PUT /mfa/policy (admin only)
func (h *Handler) UpdatePolicy(c forge.Context) error {
	// TODO: Check admin permissions
	return c.JSON(501, map[string]string{"error": "not implemented"})
}

// ResetUserMFA handles POST /mfa/users/:id/reset (admin only)
func (h *Handler) ResetUserMFA(c forge.Context) error {
	// TODO: Check admin permissions
	// TODO: Implement MFA reset
	return c.JSON(501, map[string]string{"error": "not implemented"})
}

// ==================== Helper Functions ====================

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(c forge.Context) (xid.ID, error) {
	// Try to get from context value
	if userID, ok := c.Get("user_id").(xid.ID); ok {
		return userID, nil
	}
	
	// Try to get from string
	if userIDStr, ok := c.Get("user_id").(string); ok {
		return xid.FromString(userIDStr)
	}
	
	return xid.ID{}, fmt.Errorf("user_id not found in context")
}

