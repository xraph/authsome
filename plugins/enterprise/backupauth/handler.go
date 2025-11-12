package backupauth

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/interfaces"
	"github.com/xraph/forge"
)

// Handler provides HTTP handlers for backup authentication
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ===== Recovery Session Handlers =====

// StartRecovery handles POST /recovery/start
func (h *Handler) StartRecovery(c forge.Context) error {
	var req StartRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.StartRecovery(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// ContinueRecovery handles POST /recovery/continue
func (h *Handler) ContinueRecovery(c forge.Context) error {
	var req ContinueRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.ContinueRecovery(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// CompleteRecovery handles POST /recovery/complete
func (h *Handler) CompleteRecovery(c forge.Context) error {
	var req CompleteRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.CompleteRecovery(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// CancelRecovery handles POST /recovery/cancel
func (h *Handler) CancelRecovery(c forge.Context) error {
	var req CancelRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	if err := h.service.CancelRecovery(c.Request().Context(), &req); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, SuccessResponse{Success: true, Message: "Recovery cancelled"})
}

// ===== Recovery Codes Handlers =====

// GenerateRecoveryCodes handles POST /recovery-codes/generate
func (h *Handler) GenerateRecoveryCodes(c forge.Context) error {
	userID := h.getUserIDFromContext(c)
	appID, userOrgID := h.getAppAndOrgFromContext(c)

	if userID == "" {
		return c.JSON(401, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_user_id", Message: err.Error()})
	}

	var req GenerateRecoveryCodesRequest
	if err := c.BindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = GenerateRecoveryCodesRequest{}
	}

	resp, err := h.service.GenerateRecoveryCodes(c.Request().Context(), uid, appID, userOrgID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// VerifyRecoveryCode handles POST /recovery-codes/verify
func (h *Handler) VerifyRecoveryCode(c forge.Context) error {
	var req VerifyRecoveryCodeRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.VerifyRecoveryCode(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// ===== Security Questions Handlers =====

// SetupSecurityQuestions handles POST /security-questions/setup
func (h *Handler) SetupSecurityQuestions(c forge.Context) error {
	userID := h.getUserIDFromContext(c)
	appID, userOrgID := h.getAppAndOrgFromContext(c)

	if userID == "" {
		return c.JSON(401, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_user_id", Message: err.Error()})
	}

	var req SetupSecurityQuestionsRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.SetupSecurityQuestions(c.Request().Context(), uid, appID, userOrgID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// GetSecurityQuestions handles POST /security-questions/get
func (h *Handler) GetSecurityQuestions(c forge.Context) error {
	var req GetSecurityQuestionsRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.GetSecurityQuestions(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// VerifySecurityAnswers handles POST /security-questions/verify
func (h *Handler) VerifySecurityAnswers(c forge.Context) error {
	var req VerifySecurityAnswersRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.VerifySecurityAnswers(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// ===== Trusted Contacts Handlers =====

// AddTrustedContact handles POST /trusted-contacts/add
func (h *Handler) AddTrustedContact(c forge.Context) error {
	userID := h.getUserIDFromContext(c)
	appID, userOrgID := h.getAppAndOrgFromContext(c)

	if userID == "" {
		return c.JSON(401, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_user_id", Message: err.Error()})
	}

	var req AddTrustedContactRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.AddTrustedContact(c.Request().Context(), uid, appID, userOrgID, &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(201, resp)
}

// ListTrustedContacts handles GET /trusted-contacts
func (h *Handler) ListTrustedContacts(c forge.Context) error {
	userID := h.getUserIDFromContext(c)
	appID, userOrgID := h.getAppAndOrgFromContext(c)

	if userID == "" {
		return c.JSON(401, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_user_id", Message: err.Error()})
	}

	resp, err := h.service.ListTrustedContacts(c.Request().Context(), uid, appID, userOrgID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// VerifyTrustedContact handles POST /trusted-contacts/verify
func (h *Handler) VerifyTrustedContact(c forge.Context) error {
	var req VerifyTrustedContactRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.VerifyTrustedContact(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// RequestTrustedContactVerification handles POST /trusted-contacts/request-verification
func (h *Handler) RequestTrustedContactVerification(c forge.Context) error {
	var req RequestTrustedContactVerificationRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	resp, err := h.service.RequestTrustedContactVerification(c.Request().Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, resp)
}

// RemoveTrustedContact handles DELETE /trusted-contacts/:id
func (h *Handler) RemoveTrustedContact(c forge.Context) error {
	userID := h.getUserIDFromContext(c)
	appID, userOrgID := h.getAppAndOrgFromContext(c)

	if userID == "" {
		return c.JSON(401, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_user_id", Message: err.Error()})
	}

	contactIDStr := c.Param("id")
	contactID, err := xid.FromString(contactIDStr)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_contact_id", Message: err.Error()})
	}

	req := &RemoveTrustedContactRequest{ContactID: contactID}
	if err := h.service.RemoveTrustedContact(c.Request().Context(), uid, appID, userOrgID, req); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(200, SuccessResponse{Success: true, Message: "Trusted contact removed"})
}

// ===== Email/SMS Verification Handlers =====

// SendVerificationCode handles POST /verification/send
func (h *Handler) SendVerificationCode(c forge.Context) error {
	var req SendVerificationCodeRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement email/SMS sending
	resp := &SendVerificationCodeResponse{
		Sent:         true,
		MaskedTarget: "***@example.com",
		Message:      "Verification code sent",
	}

	return c.JSON(200, resp)
}

// VerifyCode handles POST /verification/verify
func (h *Handler) VerifyCode(c forge.Context) error {
	var req VerifyCodeRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement code verification
	resp := &VerifyCodeResponse{
		Valid:   true,
		Message: "Code verified successfully",
	}

	return c.JSON(200, resp)
}

// ===== Video Verification Handlers =====

// ScheduleVideoSession handles POST /video/schedule
func (h *Handler) ScheduleVideoSession(c forge.Context) error {
	var req ScheduleVideoSessionRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement video session scheduling
	resp := &ScheduleVideoSessionResponse{
		VideoSessionID: xid.New(),
		ScheduledAt:    req.ScheduledAt,
		Message:        "Video session scheduled",
	}

	return c.JSON(200, resp)
}

// StartVideoSession handles POST /video/start
func (h *Handler) StartVideoSession(c forge.Context) error {
	var req StartVideoSessionRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement video session start
	resp := &StartVideoSessionResponse{
		VideoSessionID: req.VideoSessionID,
		SessionURL:     "https://example.com/video",
		Message:        "Video session started",
	}

	return c.JSON(200, resp)
}

// CompleteVideoSession handles POST /video/complete (admin)
func (h *Handler) CompleteVideoSession(c forge.Context) error {
	var req CompleteVideoSessionRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement video session completion
	resp := &CompleteVideoSessionResponse{
		VideoSessionID: req.VideoSessionID,
		Result:         req.VerificationResult,
		Message:        "Video session completed",
	}

	return c.JSON(200, resp)
}

// ===== Document Verification Handlers =====

// UploadDocument handles POST /documents/upload
func (h *Handler) UploadDocument(c forge.Context) error {
	var req UploadDocumentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement document upload and verification
	resp := &UploadDocumentResponse{
		DocumentID: xid.New(),
		Status:     "pending",
		Message:    "Document uploaded successfully",
	}

	return c.JSON(200, resp)
}

// GetDocumentVerification handles GET /documents/:id
func (h *Handler) GetDocumentVerification(c forge.Context) error {
	documentIDStr := c.Param("id")
	documentID, err := xid.FromString(documentIDStr)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_document_id", Message: err.Error()})
	}

	req := &GetDocumentVerificationRequest{DocumentID: documentID}
	_ = req // TODO: Implement

	resp := &GetDocumentVerificationResponse{
		DocumentID: documentID,
		Status:     "pending",
		Message:    "Document verification in progress",
	}

	return c.JSON(200, resp)
}

// ReviewDocument handles POST /documents/:id/review (admin)
func (h *Handler) ReviewDocument(c forge.Context) error {
	documentIDStr := c.Param("id")
	documentID, err := xid.FromString(documentIDStr)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_document_id", Message: err.Error()})
	}

	var req ReviewDocumentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}
	req.DocumentID = documentID

	// TODO: Implement document review

	return c.JSON(200, SuccessResponse{Success: true, Message: "Document reviewed"})
}

// ===== Admin Handlers =====

// ListRecoverySessions handles GET /admin/sessions (admin)
func (h *Handler) ListRecoverySessions(c forge.Context) error {
	var req ListRecoverySessionsRequest
	// Parse query parameters
	req.Page = 1
	req.PageSize = 20

	// TODO: Implement session listing

	resp := &ListRecoverySessionsResponse{
		Sessions:   []RecoverySessionInfo{},
		TotalCount: 0,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	return c.JSON(200, resp)
}

// ApproveRecovery handles POST /admin/sessions/:id/approve (admin)
func (h *Handler) ApproveRecovery(c forge.Context) error {
	sessionIDStr := c.Param("id")
	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_session_id", Message: err.Error()})
	}

	var req ApproveRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		// Use defaults if no body
		req = ApproveRecoveryRequest{}
	}
	req.SessionID = sessionID

	// TODO: Implement session approval

	resp := &ApproveRecoveryResponse{
		SessionID: sessionID,
		Approved:  true,
		Message:   "Recovery session approved",
	}

	return c.JSON(200, resp)
}

// RejectRecovery handles POST /admin/sessions/:id/reject (admin)
func (h *Handler) RejectRecovery(c forge.Context) error {
	sessionIDStr := c.Param("id")
	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_session_id", Message: err.Error()})
	}

	var req RejectRecoveryRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}
	req.SessionID = sessionID

	// TODO: Implement session rejection

	resp := &RejectRecoveryResponse{
		SessionID: sessionID,
		Rejected:  true,
		Reason:    req.Reason,
		Message:   "Recovery session rejected",
	}

	return c.JSON(200, resp)
}

// GetRecoveryStats handles GET /admin/stats (admin)
func (h *Handler) GetRecoveryStats(c forge.Context) error {
	// TODO: Parse time range from query params

	resp := &GetRecoveryStatsResponse{
		TotalAttempts:        0,
		SuccessfulRecoveries: 0,
		FailedRecoveries:     0,
		PendingRecoveries:    0,
		SuccessRate:          0.0,
		MethodStats:          make(map[RecoveryMethod]int),
	}

	return c.JSON(200, resp)
}

// ===== Configuration Handlers =====

// GetRecoveryConfig handles GET /admin/config (admin)
func (h *Handler) GetRecoveryConfig(c forge.Context) error {
	// TODO: Implement config retrieval

	resp := &GetRecoveryConfigResponse{
		EnabledMethods:       []RecoveryMethod{},
		RequireMultipleSteps: true,
		MinimumStepsRequired: 2,
	}

	return c.JSON(200, resp)
}

// UpdateRecoveryConfig handles PUT /admin/config (admin)
func (h *Handler) UpdateRecoveryConfig(c forge.Context) error {
	var req UpdateRecoveryConfigRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
	}

	// TODO: Implement config update

	return c.JSON(200, SuccessResponse{Success: true, Message: "Configuration updated"})
}

// ===== Health Check =====

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c forge.Context) error {
	resp := &HealthCheckResponse{
		Healthy:        true,
		Version:        "1.0.0",
		EnabledMethods: []RecoveryMethod{RecoveryMethodCodes, RecoveryMethodEmail},
		Message:        "Backup authentication plugin is healthy",
	}

	return c.JSON(200, resp)
}

// ===== Helper Methods =====

func (h *Handler) getUserIDFromContext(c forge.Context) string {
	if userID := c.Get("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

func (h *Handler) getAppAndOrgFromContext(c forge.Context) (xid.ID, *xid.ID) {
	appID := interfaces.GetAppID(c.Context())
	orgID := interfaces.GetOrganizationID(c.Context())
	// Convert to pointer, returning nil if it's NilID
	if orgID == xid.NilID() {
		return appID, nil
	}
	return appID, &orgID
}

func (h *Handler) handleError(c forge.Context, err error) error {
	// Map errors to HTTP status codes
	statusCode := 500
	errorCode := "internal_error"

	switch err {
	case ErrRecoverySessionNotFound:
		statusCode = 404
		errorCode = "session_not_found"
	case ErrRecoverySessionExpired:
		statusCode = 410
		errorCode = "session_expired"
	case ErrRecoverySessionInProgress:
		statusCode = 409
		errorCode = "session_in_progress"
	case ErrRecoveryMethodNotEnabled:
		statusCode = 400
		errorCode = "method_not_enabled"
	case ErrRateLimitExceeded:
		statusCode = 429
		errorCode = "rate_limit_exceeded"
	case ErrUnauthorized:
		statusCode = 401
		errorCode = "unauthorized"
	case ErrPermissionDenied:
		statusCode = 403
		errorCode = "permission_denied"
	case ErrInvalidInput:
		statusCode = 400
		errorCode = "invalid_input"
	}

	return c.JSON(statusCode, ErrorResponse{
		Error:   errorCode,
		Message: err.Error(),
		Code:    fmt.Sprintf("BACKUP_AUTH_%s", errorCode),
	})
}
