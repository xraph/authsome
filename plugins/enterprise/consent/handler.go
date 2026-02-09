package consent

import (
	"net/http"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for consent management.
type Handler struct {
	service *Service
}

// Response types - use shared responses from core.
type MessageResponse = responses.MessageResponse

type ConsentsResponse struct {
	Consents any `json:"consents"`
	Count    int `json:"count"`
}

// NewHandler creates a new consent handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateConsent handles POST /consent/records.
func (h *Handler) CreateConsent(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID and userID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req CreateConsentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	consent, err := h.service.CreateConsent(ctx, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusCreated, consent)
}

// GetConsent handles GET /consent/records/:id.
func (h *Handler) GetConsent(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	consent, err := h.service.GetConsent(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound(err.Error()))
	}

	return c.JSON(http.StatusOK, consent)
}

// ListConsentsByUser handles GET /consent/records/user.
func (h *Handler) ListConsentsByUser(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	consents, err := h.service.ListConsentsByUser(ctx, userID.String(), appID.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, consents)
}

// UpdateConsent handles PATCH /consent/records/:id.
func (h *Handler) UpdateConsent(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req UpdateConsentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	consent, err := h.service.UpdateConsent(ctx, id, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, consent)
}

// RevokeConsent handles POST /consent/records/:id/revoke.
func (h *Handler) RevokeConsent(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req UpdateConsentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	granted := false
	req.Granted = &granted

	_, err := h.service.UpdateConsent(ctx, id, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "Consent revoked successfully"})
}

// CreateConsentPolicy handles POST /consent/policies.
func (h *Handler) CreateConsentPolicy(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req CreatePolicyRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	policy, err := h.service.CreatePolicy(ctx, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusCreated, policy)
}

// GetConsentPolicy handles GET /consent/policies/:id.
func (h *Handler) GetConsentPolicy(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	policy, err := h.service.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound(err.Error()))
	}

	return c.JSON(http.StatusOK, policy)
}

// RecordCookieConsent handles POST /consent/cookies.
func (h *Handler) RecordCookieConsent(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req CookieConsentRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	consent, err := h.service.RecordCookieConsent(ctx, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusCreated, consent)
}

// GetCookieConsent handles GET /consent/cookies.
func (h *Handler) GetCookieConsent(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	consent, err := h.service.GetCookieConsent(ctx, userID.String(), appID.String())
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound(err.Error()))
	}

	return c.JSON(http.StatusOK, consent)
}

// RequestDataExport handles POST /consent/data-exports.
func (h *Handler) RequestDataExport(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req DataExportRequestInput
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	exportReq, err := h.service.RequestDataExport(ctx, userID.String(), appID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusAccepted, exportReq)
}

// GetDataExport handles GET /consent/data-exports/:id.
func (h *Handler) GetDataExport(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	exportReq, err := h.service.repo.GetExportRequest(ctx, id)
	if err != nil || exportReq.UserID != userID.String() {
		return c.JSON(http.StatusNotFound, errs.NotFound("Export request not found"))
	}

	return c.JSON(http.StatusOK, exportReq)
}

// DownloadDataExport handles GET /consent/data-exports/:id/download.
func (h *Handler) DownloadDataExport(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	exportReq, err := h.service.repo.GetExportRequest(ctx, id)
	if err != nil || exportReq.UserID != userID.String() || exportReq.Status != string(StatusCompleted) {
		return c.JSON(http.StatusNotFound, errs.NotFound("Export not ready or not found"))
	}

	// Return file download (forge's File method equivalent)
	// In production, use proper file serving with signed URLs
	return c.JSON(http.StatusOK, map[string]string{
		"downloadUrl": exportReq.ExportURL,
		"message":     "Export ready for download",
	})
}

// RequestDataDeletion handles POST /consent/data-deletions.
func (h *Handler) RequestDataDeletion(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req DataDeletionRequestInput
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	deletionReq, err := h.service.RequestDataDeletion(ctx, userID.String(), appID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusAccepted, deletionReq)
}

// GetDataDeletion handles GET /consent/data-deletions/:id.
func (h *Handler) GetDataDeletion(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	deletionReq, err := h.service.repo.GetDeletionRequest(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound(err.Error()))
	}

	return c.JSON(http.StatusOK, deletionReq)
}

// ApproveDeletionRequest handles POST /consent/data-deletions/:id/approve (Admin only).
func (h *Handler) ApproveDeletionRequest(c forge.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("id"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	err := h.service.ApproveDeletionRequest(ctx, id, appID.String(), userID.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "Deletion request approved"})
}

// GetPrivacySettings handles GET /consent/privacy-settings.
func (h *Handler) GetPrivacySettings(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	settings, err := h.service.GetPrivacySettings(ctx, appID.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, settings)
}

// UpdatePrivacySettings handles PATCH /consent/privacy-settings (Admin only).
func (h *Handler) UpdatePrivacySettings(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	var req PrivacySettingsRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	settings, err := h.service.UpdatePrivacySettings(ctx, appID.String(), userID.String(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, settings)
}

// GetConsentAuditLogs handles GET /consent/audit-logs.
func (h *Handler) GetConsentAuditLogs(c forge.Context) error {
	// Get audit logs - simplified for now
	// TODO: Implement proper repository method for user-filtered audit logs
	logs := []ConsentAuditLog{}

	return c.JSON(http.StatusOK, logs)
}

// GenerateConsentReport handles GET /consent/reports.
func (h *Handler) GenerateConsentReport(c forge.Context) error {
	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	// Parse date range from query params (simplified)
	startDate := time.Now().AddDate(0, -1, 0) // Last month
	endDate := time.Now()

	report, err := h.service.GenerateConsentReport(ctx, appID.String(), startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, report)
}

// ErrorResponse is a generic error response.
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}
