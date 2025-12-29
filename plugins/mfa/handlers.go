package mfa

import (
	"net"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler provides HTTP endpoints for MFA operations
type Handler struct {
	service *Service
}

// Response types - use shared responses from core
type MessageResponse = responses.MessageResponse

type FactorsResponse struct {
	Factors interface{} `json:"factors"`
	Count   int         `json:"count"`
}

type DevicesResponse struct {
	Devices interface{} `json:"devices"`
	Count   int         `json:"count"`
}

type MFAConfigResponse struct {
	Enabled             bool     `json:"enabled"`
	RequiredFactorCount int      `json:"required_factor_count"`
	AllowedFactorTypes  []string `json:"allowed_factor_types"`
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
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req EnrollFactorRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	resp, err := h.service.EnrollFactor(c.Request().Context(), userID, &FactorEnrollmentRequest{
		Type:     req.Type,
		Priority: req.Priority,
		Name:     req.Name,
		Metadata: req.Metadata,
	})
	if err != nil {
		return handleError(c, err, "ENROLL_FACTOR_FAILED", "Failed to enroll factor", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// ListFactors handles GET /mfa/factors
func (h *Handler) ListFactors(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req ListFactorsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	factors, err := h.service.ListFactors(c.Request().Context(), userID, req.ActiveOnly)
	if err != nil {
		return handleError(c, err, "LIST_FACTORS_FAILED", "Failed to list factors", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &FactorsResponse{Factors: factors, Count: len(factors)})
}

// GetFactor handles GET /mfa/factors/:id
func (h *Handler) GetFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req GetFactorRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	factorID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid factor ID"))
	}

	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("MFA factor not found"))
	}

	// Verify factor belongs to user
	if factor.UserID != userID {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("access", "factor"))
	}

	return c.JSON(http.StatusOK, factor)
}

// UpdateFactor handles PUT /mfa/factors/:id
func (h *Handler) UpdateFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req UpdateFactorRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	factorID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid factor ID"))
	}

	// First verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("MFA factor not found"))
	}
	if factor.UserID != userID {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("access", "factor"))
	}

	// Convert UpdateFactorRequest to map for service layer
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	if err := h.service.UpdateFactor(c.Request().Context(), factorID, updates); err != nil {
		return handleError(c, err, "UPDATE_FACTOR_FAILED", "Failed to update factor", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "factor updated"})
}

// DeleteFactor handles DELETE /mfa/factors/:id
func (h *Handler) DeleteFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req DeleteFactorRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	factorID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid factor ID"))
	}

	// Verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("MFA factor not found"))
	}
	if factor.UserID != userID {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("access", "factor"))
	}

	if err := h.service.DeleteFactor(c.Request().Context(), factorID); err != nil {
		return handleError(c, err, "DELETE_FACTOR_FAILED", "Failed to delete factor", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "factor deleted"})
}

// VerifyFactor handles POST /mfa/factors/:id/verify
func (h *Handler) VerifyFactor(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req VerifyEnrolledFactorRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	factorID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid factor ID"))
	}

	// Verify factor belongs to user
	factor, err := h.service.GetFactor(c.Request().Context(), factorID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("MFA factor not found"))
	}
	if factor.UserID != userID {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("access", "factor"))
	}

	if err := h.service.VerifyEnrollment(c.Request().Context(), factorID, req.Code); err != nil {
		return handleError(c, err, "VERIFY_ENROLLMENT_FAILED", "Failed to verify factor enrollment", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "factor verified"})
}

// ==================== Challenge & Verification ====================

// InitiateChallenge handles POST /mfa/challenge
func (h *Handler) InitiateChallenge(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var reqDTO InitiateChallengeRequest
	if err := c.BindRequest(&reqDTO); err != nil {
		// Use empty request if no body provided
		reqDTO = InitiateChallengeRequest{}
	}

	// Create service request
	req := ChallengeRequest{
		UserID:      userID,
		FactorTypes: reqDTO.FactorTypes,
		Context:     reqDTO.Context,
		Metadata:    reqDTO.Metadata,
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}

	// Add IP and user agent to metadata
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	req.Metadata["ip_address"] = ip
	req.Metadata["user_agent"] = c.Request().UserAgent()

	resp, err := h.service.InitiateChallenge(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "INITIATE_CHALLENGE_FAILED", "Failed to initiate MFA challenge", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// VerifyChallenge handles POST /mfa/verify
func (h *Handler) VerifyChallenge(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var reqDTO VerifyChallengeRequest
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Create service request
	req := VerificationRequest{
		ChallengeID:    reqDTO.ChallengeID,
		FactorID:       reqDTO.FactorID,
		Code:           reqDTO.Code,
		Data:           reqDTO.Data,
		RememberDevice: reqDTO.RememberDevice,
		DeviceInfo:     reqDTO.DeviceInfo,
	}

	// Verify challenge belongs to user (done in service)
	resp, err := h.service.VerifyChallenge(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "VERIFY_CHALLENGE_FAILED", "Failed to verify MFA challenge", http.StatusBadRequest)
	}

	_ = userID // Verify in service layer

	return c.JSON(http.StatusOK, resp)
}

// GetChallengeStatus handles GET /mfa/challenge/:id
func (h *Handler) GetChallengeStatus(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req GetChallengeStatusRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	sessionID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid session ID"))
	}

	status, err := h.service.GetChallengeStatus(c.Request().Context(), sessionID, userID)
	if err != nil {
		return handleError(c, err, "GET_CHALLENGE_STATUS_FAILED", "Failed to get challenge status", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, status)
}

// ==================== Trusted Devices ====================

// TrustDevice handles POST /mfa/devices/trust
func (h *Handler) TrustDevice(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req TrustDeviceRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	deviceInfo := &DeviceInfo{
		DeviceID: req.DeviceID,
		Name:     req.Name,
		Metadata: req.Metadata,
	}

	if err := h.service.TrustDevice(c.Request().Context(), userID, deviceInfo); err != nil {
		return handleError(c, err, "TRUST_DEVICE_FAILED", "Failed to trust device", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "device trusted"})
}

// ListTrustedDevices handles GET /mfa/devices
func (h *Handler) ListTrustedDevices(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	devices, err := h.service.ListTrustedDevices(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err, "LIST_DEVICES_FAILED", "Failed to list trusted devices", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &DevicesResponse{Devices: devices, Count: len(devices)})
}

// RevokeTrustedDevice handles DELETE /mfa/devices/:id
func (h *Handler) RevokeTrustedDevice(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req RevokeTrustedDeviceRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	deviceID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid device ID"))
	}

	// TODO: Verify device belongs to user
	_ = userID

	if err := h.service.RevokeTrustedDevice(c.Request().Context(), deviceID); err != nil {
		return handleError(c, err, "REVOKE_DEVICE_FAILED", "Failed to revoke trusted device", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "device revoked"})
}

// ==================== Status & Info ====================

// GetStatus handles GET /mfa/status
func (h *Handler) GetStatus(c forge.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	deviceID := c.Request().URL.Query().Get("device_id")

	status, err := h.service.GetMFAStatus(c.Request().Context(), userID, deviceID)
	if err != nil {
		return handleError(c, err, "GET_STATUS_FAILED", "Failed to get MFA status", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, status)
}

// GetPolicy handles GET /mfa/policy
func (h *Handler) GetPolicy(c forge.Context) error {
	// TODO: Get organization ID from context
	// For now, return config-based policy

	// Convert []FactorType to []string
	allowedTypes := make([]string, len(h.service.config.AllowedFactorTypes))
	for i, ft := range h.service.config.AllowedFactorTypes {
		allowedTypes[i] = string(ft)
	}

	return c.JSON(http.StatusOK, &MFAConfigResponse{
		Enabled:             h.service.config.Enabled,
		RequiredFactorCount: h.service.config.RequiredFactorCount,
		AllowedFactorTypes:  allowedTypes,
	})
}

// ==================== Admin Endpoints ====================

// AdminPolicyRequest represents a request to update MFA policy
type AdminPolicyRequest struct {
	RequiredFactors int      `json:"requiredFactors"` // Number of factors required
	AllowedTypes    []string `json:"allowedTypes"`    // e.g., ["totp", "sms", "email", "webauthn", "backup"]
	GracePeriod     int      `json:"gracePeriod"`     // Grace period in seconds for new users
	Enabled         bool     `json:"enabled"`         // Enable/disable MFA requirement
}

// AdminBypassRequest represents a request to grant temporary MFA bypass
type AdminBypassRequest struct {
	UserID   xid.ID `json:"userId"`
	Duration int    `json:"duration"` // Bypass duration in seconds
	Reason   string `json:"reason"`   // Reason for bypass
}

// AdminGetPolicy handles GET /mfa/admin/policy
// Gets the current MFA policy for an app
func (h *Handler) AdminGetPolicy(c forge.Context) error {
	_ = c.Request().Context() // ctx for future use

	// Get app context
	appID := getUserAppID(c)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest))
	}

	// TODO: Check admin permission via RBAC
	// userID, err := getUserIDFromContext(c)
	// if err != nil {
	//     return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	// }
	// if !h.rbacService.HasPermission(ctx, userID, "mfa:admin") {
	//     return c.JSON(http.StatusForbidden, errs.PermissionDenied("mfa:admin", "policy"))
	// }

	// TODO: Load policy from database for this app
	// For now, return default policy
	policy := map[string]interface{}{
		"appId":           appID.String(),
		"requiredFactors": 1,
		"allowedTypes":    []string{"totp", "sms", "email", "webauthn", "backup"},
		"gracePeriod":     86400, // 24 hours
		"enabled":         true,
	}

	return c.JSON(http.StatusOK, policy)
}

// AdminUpdatePolicy handles PUT /mfa/admin/policy
// Updates the MFA policy for an app (admin only)
func (h *Handler) AdminUpdatePolicy(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest))
	}

	// Get org context (optional)
	orgID, _ := contexts.GetOrganizationID(ctx)
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Get admin user ID
	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// TODO: Check admin permission via RBAC
	// if !h.rbacService.HasPermission(ctx, adminID, "mfa:admin") {
	//     return c.JSON(http.StatusForbidden, errs.PermissionDenied("mfa:admin", "policy"))
	// }

	var req AdminPolicyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Validate policy
	if req.RequiredFactors < 0 || req.RequiredFactors > 3 {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("requiredFactors must be between 0 and 3"))
	}

	// Update policy via service
	policy, err := h.service.UpdatePolicy(ctx, appID, orgIDPtr, adminID, &req)
	if err != nil {
		return handleError(c, err, "UPDATE_POLICY_FAILED", "Failed to update MFA policy", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, policy)
}

// AdminGrantBypass handles POST /mfa/admin/bypass
// Grants temporary MFA bypass for a user (admin only)
func (h *Handler) AdminGrantBypass(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest))
	}

	// Get admin user ID
	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// TODO: Check admin permission via RBAC
	// if !h.rbacService.HasPermission(ctx, adminID, "mfa:admin") {
	//     return c.JSON(http.StatusForbidden, errs.PermissionDenied("mfa:admin", "bypass"))
	// }

	var req AdminBypassRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Validate request
	if req.UserID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("userId"))
	}

	if req.Duration <= 0 || req.Duration > 86400*7 { // Max 7 days
		return c.JSON(http.StatusBadRequest, errs.BadRequest("duration must be between 1 second and 7 days"))
	}

	if req.Reason == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("reason"))
	}

	// Grant bypass via service
	bypass, err := h.service.GrantBypass(ctx, appID, req.UserID, adminID, req.Duration, req.Reason)
	if err != nil {
		return handleError(c, err, "GRANT_BYPASS_FAILED", "Failed to grant MFA bypass", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, bypass)
}

// AdminResetUserMFA handles POST /mfa/admin/users/:id/reset
// Resets all MFA factors for a user (admin only)
func (h *Handler) AdminResetUserMFA(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest))
	}

	var req ResetUserMFARequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	userID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	// Get admin user ID
	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// TODO: Check admin permission via RBAC
	// if !h.rbacService.HasPermission(ctx, adminID, "mfa:admin") {
	//     return c.JSON(http.StatusForbidden, errs.PermissionDenied("mfa:admin", "reset"))
	// }

	// Reset MFA via service
	if err := h.service.ResetUserMFA(ctx, appID, userID, adminID); err != nil {
		return handleError(c, err, "RESET_MFA_FAILED", "Failed to reset user MFA", http.StatusInternalServerError)
	}

	response := map[string]interface{}{
		"message": "MFA reset successfully",
		"userId":  userID.String(),
		"appId":   appID.String(),
	}

	return c.JSON(http.StatusOK, response)
}

// getUserAppID extracts app ID from request context
func getUserAppID(c forge.Context) xid.ID {
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return xid.NilID()
	}
	return appID
}

// ==================== Helper Functions ====================

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(c forge.Context) (xid.ID, error) {
	userID, err := contexts.RequireUserID(c.Request().Context())
	if err != nil {
		return xid.NilID(), err
	}
	return userID, nil
}
