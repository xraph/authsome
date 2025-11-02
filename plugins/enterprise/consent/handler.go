package consent

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/xraph/forge"
)

// Handler handles HTTP requests for consent management
type Handler struct {
	service *Service
}

// NewHandler creates a new consent handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Helper function to extract string from context
func getString(c forge.Context, key string) string {
	val := c.Get(key)
	if val == nil {
		return ""
	}
	if strVal, ok := val.(string); ok {
		return strVal
	}
	return ""
}

// CreateConsent handles POST /consent/records
func (h *Handler) CreateConsent(c forge.Context) error {
	var req CreateConsentRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	if userID == "" || orgID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	consent, err := h.service.CreateConsent(c.Request().Context(), orgID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, consent)
}

// GetConsent handles GET /consent/records/:id
func (h *Handler) GetConsent(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	consent, err := h.service.GetConsent(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, consent)
}

// ListConsentsByUser handles GET /consent/records/user
func (h *Handler) ListConsentsByUser(c forge.Context) error {
	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	if userID == "" || orgID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	consents, err := h.service.ListConsentsByUser(c.Request().Context(), userID, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, consents)
}

// UpdateConsent handles PATCH /consent/records/:id
func (h *Handler) UpdateConsent(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	var req UpdateConsentRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	consent, err := h.service.UpdateConsent(c.Request().Context(), id, orgID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, consent)
}

// RevokeConsent handles POST /consent/records/:id/revoke
func (h *Handler) RevokeConsent(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	var req UpdateConsentRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	granted := false
	req.Granted = &granted
	
	_, err := h.service.UpdateConsent(c.Request().Context(), id, orgID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Consent revoked successfully",
	})
}

// CreateConsentPolicy handles POST /consent/policies
func (h *Handler) CreateConsentPolicy(c forge.Context) error {
	var req CreatePolicyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	orgID := getString(c, "orgID")
	userID := getString(c, "userID")

	policy, err := h.service.CreatePolicy(c.Request().Context(), orgID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, policy)
}

// GetConsentPolicy handles GET /consent/policies/:id
func (h *Handler) GetConsentPolicy(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	policy, err := h.service.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, policy)
}

// RecordCookieConsent handles POST /consent/cookies
func (h *Handler) RecordCookieConsent(c forge.Context) error {
	var req CookieConsentRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	consent, err := h.service.RecordCookieConsent(c.Request().Context(), orgID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, consent)
}

// GetCookieConsent handles GET /consent/cookies
func (h *Handler) GetCookieConsent(c forge.Context) error {
	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	consent, err := h.service.GetCookieConsent(c.Request().Context(), userID, orgID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, consent)
}

// RequestDataExport handles POST /consent/data-exports
func (h *Handler) RequestDataExport(c forge.Context) error {
	var req DataExportRequestInput
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	exportReq, err := h.service.RequestDataExport(c.Request().Context(), userID, orgID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusAccepted, exportReq)
}

// GetDataExport handles GET /consent/data-exports/:id
func (h *Handler) GetDataExport(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	userID := getString(c, "userID")
	exportReq, err := h.service.repo.GetExportRequest(c.Request().Context(), id)
	if err != nil || exportReq.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Export request not found",
		})
	}

	return c.JSON(http.StatusOK, exportReq)
}

// DownloadDataExport handles GET /consent/data-exports/:id/download
func (h *Handler) DownloadDataExport(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	userID := getString(c, "userID")
	exportReq, err := h.service.repo.GetExportRequest(c.Request().Context(), id)
	if err != nil || exportReq.UserID != userID || exportReq.Status != string(StatusCompleted) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Export not ready or not found",
		})
	}

	// Return file download (forge's File method equivalent)
	// In production, use proper file serving with signed URLs
	return c.JSON(http.StatusOK, map[string]string{
		"downloadUrl": exportReq.ExportURL,
		"message":     "Export ready for download",
	})
}

// RequestDataDeletion handles POST /consent/data-deletions
func (h *Handler) RequestDataDeletion(c forge.Context) error {
	var req DataDeletionRequestInput
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	userID := getString(c, "userID")
	orgID := getString(c, "orgID")

	deletionReq, err := h.service.RequestDataDeletion(c.Request().Context(), userID, orgID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusAccepted, deletionReq)
}

// GetDataDeletion handles GET /consent/data-deletions/:id
func (h *Handler) GetDataDeletion(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	deletionReq, err := h.service.repo.GetDeletionRequest(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, deletionReq)
}

// ApproveDeletionRequest handles POST /consent/data-deletions/:id/approve (Admin only)
func (h *Handler) ApproveDeletionRequest(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ID parameter is required",
		})
	}

	approverID := getString(c, "userID")
	orgID := getString(c, "orgID")

	err := h.service.ApproveDeletionRequest(c.Request().Context(), id, orgID, approverID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Deletion request approved",
	})
}

// GetPrivacySettings handles GET /consent/privacy-settings
func (h *Handler) GetPrivacySettings(c forge.Context) error {
	orgID := getString(c, "orgID")

	settings, err := h.service.GetPrivacySettings(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, settings)
}

// UpdatePrivacySettings handles PATCH /consent/privacy-settings (Admin only)
func (h *Handler) UpdatePrivacySettings(c forge.Context) error {
	var req PrivacySettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	orgID := getString(c, "orgID")
	updatedBy := getString(c, "userID")

	settings, err := h.service.UpdatePrivacySettings(c.Request().Context(), orgID, updatedBy, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, settings)
}

// GetConsentAuditLogs handles GET /consent/audit-logs
func (h *Handler) GetConsentAuditLogs(c forge.Context) error {
	// Get audit logs - simplified for now
	// TODO: Implement proper repository method for user-filtered audit logs
	// userID := getString(c, "userID")
	// orgID := getString(c, "orgID")
	// limitStr := c.Query("limit")
	logs := []ConsentAuditLog{}

	return c.JSON(http.StatusOK, logs)
}

// GenerateConsentReport handles GET /consent/reports
func (h *Handler) GenerateConsentReport(c forge.Context) error {
	orgID := getString(c, "orgID")

	// Parse date range from query params (simplified)
	startDate := time.Now().AddDate(0, -1, 0) // Last month
	endDate := time.Now()

	report, err := h.service.GenerateConsentReport(c.Request().Context(), orgID, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, report)
}

// ErrorResponse is a generic error response
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

