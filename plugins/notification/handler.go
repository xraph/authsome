package notification

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles notification HTTP requests.
type Handler struct {
	service      *notification.Service
	templateSvc  *TemplateService
	providerSvc  *notification.ProviderService
	versionSvc   *notification.VersionService
	abTestSvc    *notification.ABTestService
	analyticsSvc *notification.AnalyticsService
	config       Config
}

// ErrorResponse types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

type TemplatesResponse struct {
	Templates any `json:"templates"`
	Count     int `json:"count"`
}

type ChannelsResponse struct {
	Channels any `json:"channels"`
	Count    int `json:"count"`
}

type NotificationsResponse struct {
	Notifications any `json:"notifications"`
	Count         int `json:"count"`
}

// NewHandler creates a new notification handler.
func NewHandler(service *notification.Service, templateSvc *TemplateService, config Config) *Handler {
	// Initialize sub-services
	repo := service.GetRepository()

	return &Handler{
		service:      service,
		templateSvc:  templateSvc,
		providerSvc:  notification.NewProviderService(repo),
		versionSvc:   notification.NewVersionService(repo),
		abTestSvc:    notification.NewABTestService(repo),
		analyticsSvc: notification.NewAnalyticsService(repo),
		config:       config,
	}
}

// =============================================================================
// TEMPLATE HANDLERS
// =============================================================================

// CreateTemplate creates a new notification template.
func (h *Handler) CreateTemplate(c forge.Context) error {
	// Try to extract app ID from query parameter first (for dashboard requests)
	appIDStr := c.Query("app_id")

	var (
		appID xid.ID
		err   error
	)

	if appIDStr != "" {
		appID, err = xid.FromString(appIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &MessageResponse{Message: "invalid app_id parameter"})
		}
	} else {
		// Fall back to context (for API requests)
		appID, err = contexts.RequireAppID(c.Context())
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.Unauthorized())
		}
	}

	var req notification.CreateTemplateRequest

	// Support both JSON and form data
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data" {
		// Bind form data
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	} else {
		// Bind JSON data
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	}

	// Set app ID from query parameter or context
	req.AppID = appID

	template, err := h.service.CreateTemplate(c.Context(), &req)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Check for redirect parameter (for form submissions)
	if redirect := c.Query("redirect"); redirect != "" {
		return c.Redirect(http.StatusSeeOther, redirect)
	}

	return c.JSON(http.StatusCreated, template)
}

// GetTemplate retrieves a template by ID.
func (h *Handler) GetTemplate(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	template, err := h.service.GetTemplate(c.Context(), id)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, template)
}

// ListTemplates lists all templates with pagination.
func (h *Handler) ListTemplates(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	// Parse pagination parameters
	page := 1

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// notifType filter parameters
	var notifType *notification.NotificationType

	if typeStr := c.Query("type"); typeStr != "" {
		t := notification.NotificationType(typeStr)
		notifType = &t
	}

	var language *string
	if lang := c.Query("language"); lang != "" {
		language = &lang
	}

	var active *bool

	if activeStr := c.Query("active"); activeStr != "" {
		if a, err := strconv.ParseBool(activeStr); err == nil {
			active = &a
		}
	}

	filter := &notification.ListTemplatesFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		AppID:    appID,
		Type:     notifType,
		Language: language,
		Active:   active,
	}

	response, err := h.service.ListTemplates(c.Context(), filter)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateTemplate updates a template.
func (h *Handler) UpdateTemplate(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	var req notification.UpdateTemplateRequest

	// Support both JSON and form data
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data" {
		// Bind form data
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	} else {
		// Bind JSON data
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	}

	if err := h.service.UpdateTemplate(c.Context(), id, &req); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Check for redirect parameter (for form submissions)
	if redirect := c.Query("redirect"); redirect != "" {
		return c.Redirect(http.StatusSeeOther, redirect)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "template updated successfully"})
}

// DeleteTemplate deletes a template.
func (h *Handler) DeleteTemplate(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	if err := h.service.DeleteTemplate(c.Context(), id); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Check for redirect parameter (for form submissions)
	if redirect := c.Query("redirect"); redirect != "" {
		return c.Redirect(http.StatusSeeOther, redirect)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "template deleted successfully"})
}

// ResetTemplate resets a template to default values.
func (h *Handler) ResetTemplate(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	if err := h.service.ResetTemplate(c.Context(), id); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "template reset to default successfully"})
}

// ResetAllTemplates resets all templates for an app to defaults.
func (h *Handler) ResetAllTemplates(c forge.Context) error {
	// Try to extract app ID from query parameter first (for dashboard requests)
	appIDStr := c.Query("app_id")

	var (
		appID xid.ID
		err   error
	)

	if appIDStr != "" {
		appID, err = xid.FromString(appIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &MessageResponse{Message: "invalid app_id parameter"})
		}
	} else {
		// Fall back to context (for API requests)
		appID, err = contexts.RequireAppID(c.Context())
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.Unauthorized())
		}
	}

	if err := h.service.ResetAllTemplates(c.Context(), appID); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, &MessageResponse{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "all templates reset to defaults successfully"})
}

// GetTemplateDefaults returns default template metadata.
func (h *Handler) GetTemplateDefaults(c forge.Context) error {
	// Get default template metadata
	defaults := notification.GetDefaultTemplateMetadata()

	return c.JSON(http.StatusOK, map[string]any{
		"templates": defaults,
		"count":     len(defaults),
	})
}

// PreviewTemplate renders a template with provided variables.
func (h *Handler) PreviewTemplate(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	var req struct {
		Variables map[string]any `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	template, err := h.service.GetTemplate(c.Context(), id)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Create template engine for rendering
	engine := NewTemplateEngine()

	// Render body
	body, err := engine.Render(template.Body, req.Variables)
	if err != nil {
		renderErr := notification.TemplateRenderFailed(err)

		return c.JSON(renderErr.HTTPStatus, renderErr)
	}

	// Render subject if present
	subject := ""
	if template.Subject != "" {
		subject, err = engine.Render(template.Subject, req.Variables)
		if err != nil {
			renderErr := notification.TemplateRenderFailed(err)

			return c.JSON(renderErr.HTTPStatus, renderErr)
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"subject": subject,
		"body":    body,
	})
}

// RenderTemplate renders a template string with variables (no template ID required).
func (h *Handler) RenderTemplate(c forge.Context) error {
	var req struct {
		Template  string         `json:"template"`
		Variables map[string]any `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	engine := NewTemplateEngine()

	rendered, err := engine.Render(req.Template, req.Variables)
	if err != nil {
		renderErr := notification.TemplateRenderFailed(err)

		return c.JSON(renderErr.HTTPStatus, renderErr)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rendered": rendered,
	})
}

// =============================================================================
// NOTIFICATION HANDLERS
// =============================================================================

// SendNotification sends a notification.
func (h *Handler) SendNotification(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var req notification.SendRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	// Set app ID from context
	req.AppID = appID

	notif, err := h.service.Send(c.Context(), &req)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, notif)
}

// GetNotification retrieves a notification by ID.
func (h *Handler) GetNotification(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid notification ID format"))
	}

	notif, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, notif)
}

// ListNotifications lists all notifications with pagination.
func (h *Handler) ListNotifications(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	// Parse pagination parameters
	page := 1

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// notifType filter parameters
	var notifType *notification.NotificationType

	if typeStr := c.Query("type"); typeStr != "" {
		t := notification.NotificationType(typeStr)
		notifType = &t
	}

	var status *notification.NotificationStatus

	if statusStr := c.Query("status"); statusStr != "" {
		s := notification.NotificationStatus(statusStr)
		status = &s
	}

	var recipient *string
	if rec := c.Query("recipient"); rec != "" {
		recipient = &rec
	}

	filter := &notification.ListNotificationsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		AppID:     appID,
		Type:      notifType,
		Status:    status,
		Recipient: recipient,
	}

	response, err := h.service.ListNotifications(c.Context(), filter)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}

// ResendNotification resends a notification.
func (h *Handler) ResendNotification(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid notification ID format"))
	}

	// Get original notification
	originalNotif, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Send new notification with same details
	newNotif, err := h.service.Send(c.Context(), &notification.SendRequest{
		AppID:     appID,
		Type:      originalNotif.Type,
		Recipient: originalNotif.Recipient,
		Subject:   originalNotif.Subject,
		Body:      originalNotif.Body,
		Metadata:  originalNotif.Metadata,
	})
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, newNotif)
}

// HandleWebhook handles provider webhook callbacks.
func (h *Handler) HandleWebhook(c forge.Context) error {
	providerID := c.Param("provider")
	if providerID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider ID required"))
	}

	// Parse webhook payload based on provider
	// payload is a placeholder - actual implementation would depend on provider specs
	var payload map[string]any
	if err := c.BindJSON(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid webhook payload"))
	}

	// Process webhook based on provider
	// For now, just acknowledge receipt
	return c.JSON(http.StatusOK, &StatusResponse{Status: "processed"})
}

// =============================================================================
// PROVIDER HANDLERS
// =============================================================================

// CreateProvider creates a new notification provider.
func (h *Handler) CreateProvider(c forge.Context) error {
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var req struct {
		OrganizationID *string        `form:"organizationId" json:"organizationId,omitempty"`
		ProviderType   string         `form:"providerType"   json:"providerType"` // "email" or "sms"
		ProviderName   string         `form:"providerName"   json:"providerName"` // "smtp", "sendgrid", "twilio", etc.
		Config         map[string]any `form:"config"         json:"config"`
		IsDefault      bool           `form:"isDefault"      json:"isDefault"`
	}

	// Support both JSON and form data
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data" {
		// Bind form data
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	} else {
		// Bind JSON data
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
		}
	}

	var orgID *xid.ID

	if req.OrganizationID != nil {
		id, err := xid.FromString(*req.OrganizationID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid organization ID"))
		}

		orgID = &id
	}

	provider, err := h.providerSvc.CreateProvider(c.Context(), appID, orgID, req.ProviderType, req.ProviderName, req.Config, req.IsDefault)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// Check for redirect parameter (for form submissions)
	if redirect := c.Query("redirect"); redirect != "" {
		return c.Redirect(http.StatusSeeOther, redirect)
	}

	return c.JSON(http.StatusCreated, provider)
}

// GetProvider retrieves a provider by ID.
func (h *Handler) GetProvider(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid provider ID"))
	}

	provider, err := h.providerSvc.GetProvider(c.Context(), id)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, provider)
}

// ListProviders lists all providers for an app/org.
func (h *Handler) ListProviders(c forge.Context) error {
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var orgID *xid.ID

	if orgIDStr := c.Query("organizationId"); orgIDStr != "" {
		id, err := xid.FromString(orgIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid organization ID"))
		}

		orgID = &id
	}

	providers, err := h.providerSvc.ListProviders(c.Context(), appID, orgID)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"providers": providers,
		"count":     len(providers),
	})
}

// UpdateProvider updates a provider's configuration.
func (h *Handler) UpdateProvider(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid provider ID"))
	}

	var req struct {
		Config    map[string]any `json:"config"`
		IsActive  bool           `json:"isActive"`
		IsDefault bool           `json:"isDefault"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	if err := h.providerSvc.UpdateProvider(c.Context(), id, req.Config, req.IsActive, req.IsDefault); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "provider updated successfully"})
}

// DeleteProvider deletes a provider.
func (h *Handler) DeleteProvider(c forge.Context) error {
	idStr := c.Param("id")

	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid provider ID"))
	}

	if err := h.providerSvc.DeleteProvider(c.Context(), id); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "provider deleted successfully"})
}

// =============================================================================
// TEMPLATE VERSIONING HANDLERS
// =============================================================================

// CreateTemplateVersion creates a new version for a template.
func (h *Handler) CreateTemplateVersion(c forge.Context) error {
	idStr := c.Param("id")

	templateID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	var req struct {
		Changes string `json:"changes"` // Description of what changed
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	// Get current template
	template, err := h.service.GetTemplate(c.Context(), templateID)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	// changedBy current user ID if available (for changedBy)
	var changedBy *xid.ID
	// TODO: Extract user ID from context

	version, err := h.versionSvc.CreateVersion(c.Context(), template.ToSchema(), changedBy, req.Changes)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusCreated, version)
}

// GetTemplateVersion retrieves a specific template version.
func (h *Handler) GetTemplateVersion(c forge.Context) error {
	idStr := c.Param("versionId")

	versionID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid version ID"))
	}

	version, err := h.versionSvc.GetVersion(c.Context(), versionID)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, version)
}

// ListTemplateVersions lists all versions for a template.
func (h *Handler) ListTemplateVersions(c forge.Context) error {
	idStr := c.Param("id")

	templateID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	versions, err := h.versionSvc.ListVersions(c.Context(), templateID)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"versions": versions,
		"count":    len(versions),
	})
}

// RestoreTemplateVersion restores a template to a previous version.
func (h *Handler) RestoreTemplateVersion(c forge.Context) error {
	templateIDStr := c.Param("id")

	templateID, err := xid.FromString(templateIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	versionIDStr := c.Param("versionId")

	versionID, err := xid.FromString(versionIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid version ID"))
	}

	// restoredBy current user ID if available (for restoredBy)
	var restoredBy *xid.ID
	// TODO: Extract user ID from context

	if err := h.versionSvc.RestoreVersion(c.Context(), templateID, versionID, restoredBy); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "template restored successfully"})
}

// =============================================================================
// A/B TESTING HANDLERS
// =============================================================================

// CreateABTestVariant creates a new A/B test variant.
func (h *Handler) CreateABTestVariant(c forge.Context) error {
	parentIDStr := c.Param("id")

	parentTemplateID, err := xid.FromString(parentIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	var req struct {
		Name    string `json:"name"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Weight  int    `json:"weight"` // 0-100
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	variant, err := h.abTestSvc.CreateVariant(c.Context(), parentTemplateID, req.Name, req.Weight, req.Subject, req.Body)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusCreated, variant)
}

// GetABTestResults retrieves A/B test results for a test group.
func (h *Handler) GetABTestResults(c forge.Context) error {
	abTestGroup := c.Query("abTestGroup")
	if abTestGroup == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("abTestGroup required"))
	}

	results, err := h.abTestSvc.GetABTestResults(c.Context(), abTestGroup)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, results)
}

// DeclareABTestWinner declares a winner for an A/B test.
func (h *Handler) DeclareABTestWinner(c forge.Context) error {
	var req struct {
		WinnerID    string `json:"winnerId"`
		ABTestGroup string `json:"abTestGroup"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	winnerID, err := xid.FromString(req.WinnerID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid winner ID"))
	}

	if err := h.abTestSvc.DeclareWinner(c.Context(), winnerID, req.ABTestGroup); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "winner declared successfully"})
}

// =============================================================================
// ANALYTICS HANDLERS
// =============================================================================

// TrackNotificationEvent tracks an analytics event.
func (h *Handler) TrackNotificationEvent(c forge.Context) error {
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var req struct {
		OrganizationID *string        `json:"organizationId,omitempty"`
		NotificationID string         `json:"notificationId"`
		TemplateID     string         `json:"templateId"`
		Event          string         `json:"event"` // "sent", "delivered", "opened", "clicked", "converted"
		EventData      map[string]any `json:"eventData,omitempty"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	notificationID, err := xid.FromString(req.NotificationID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid notification ID"))
	}

	templateID, err := xid.FromString(req.TemplateID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	var orgID *xid.ID

	if req.OrganizationID != nil {
		id, err := xid.FromString(*req.OrganizationID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid organization ID"))
		}

		orgID = &id
	}

	if err := h.analyticsSvc.TrackEvent(c.Context(), notificationID, templateID, appID, orgID, req.Event, req.EventData); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "event tracked successfully"})
}

// GetTemplateAnalytics retrieves analytics for a template.
func (h *Handler) GetTemplateAnalytics(c forge.Context) error {
	idStr := c.Param("id")

	templateID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID"))
	}

	// TODO: Parse date range from query params
	end := time.Now()
	start := end.AddDate(0, 0, -30)

	report, err := h.analyticsSvc.GetTemplateAnalytics(c.Context(), templateID, start, end)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, report)
}

// GetAppAnalytics retrieves analytics for an app.
func (h *Handler) GetAppAnalytics(c forge.Context) error {
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	// TODO: Parse date range from query params
	end := time.Now()
	start := end.AddDate(0, 0, -30)

	report, err := h.analyticsSvc.GetAppAnalytics(c.Context(), appID, start, end)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, report)
}

// GetOrgAnalytics retrieves analytics for an organization.
func (h *Handler) GetOrgAnalytics(c forge.Context) error {
	orgIDStr := c.Query("organizationId")
	if orgIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization ID required"))
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid organization ID"))
	}

	// TODO: Parse date range from query params
	end := time.Now()
	start := end.AddDate(0, 0, -30)

	report, err := h.analyticsSvc.GetOrgAnalytics(c.Context(), orgID, start, end)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, report)
}
