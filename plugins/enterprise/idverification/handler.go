package idverification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/base"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for identity verification.
type Handler struct {
	service *Service
}

// ErrorResponse types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

// NewHandler creates a new identity verification handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateVerificationSession creates a new verification session
// CreateVerificationSession /verification/sessions.
func (h *Handler) CreateVerificationSession(c forge.Context) error {
	var req struct {
		Provider       string         `json:"provider"`
		RequiredChecks []string       `json:"requiredChecks"`
		SuccessURL     string         `json:"successUrl"`
		CancelURL      string         `json:"cancelUrl"`
		Config         map[string]any `json:"config"`
		Metadata       map[string]any `json:"metadata"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	ctx := c.Request().Context()

	// Get context values using contexts package
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	envID, _ := contexts.GetEnvironmentID(ctx)

	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}

	session, err := h.service.CreateVerificationSession(ctx, &CreateSessionRequest{
		AppID:          appID,
		EnvironmentID:  envIDPtr,
		OrganizationID: orgID,
		UserID:         userID,
		Provider:       req.Provider,
		RequiredChecks: req.RequiredChecks,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		Config:         req.Config,
		Metadata:       req.Metadata,
		IPAddress:      c.Request().RemoteAddr,
		UserAgent:      c.Request().Header.Get("User-Agent"),
	})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, &VerificationSessionResponse{
		Session: base.FromSchemaIdentityVerificationSession(session),
	})
}

// GetVerificationSession retrieves a verification session
// GetVerificationSession /verification/sessions/:id.
func (h *Handler) GetVerificationSession(c forge.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("session_id"))
	}

	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	session, err := h.service.GetVerificationSession(ctx, appID, sessionID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &VerificationSessionResponse{
		Session: base.FromSchemaIdentityVerificationSession(session),
	})
}

// GetVerification retrieves a verification by ID
// GetVerification /verification/:id.
func (h *Handler) GetVerification(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("verification_id"))
	}

	ctx := c.Request().Context()

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	verification, err := h.service.GetVerification(ctx, appID, id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &VerificationResponse{
		Verification: base.FromSchemaIdentityVerification(verification),
	})
}

// GetUserVerifications retrieves all verifications for the current user
// GetUserVerifications /verification/me.
func (h *Handler) GetUserVerifications(c forge.Context) error {
	ctx := c.Request().Context()

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	verifications, err := h.service.GetUserVerifications(ctx, appID, userID, limit, offset)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &VerificationListResponse{
		Verifications: base.FromSchemaIdentityVerifications(verifications),
		Limit:         limit,
		Offset:        offset,
	})
}

// GetUserVerificationStatus retrieves the verification status for the current user
// GetUserVerificationStatus /verification/me/status.
func (h *Handler) GetUserVerificationStatus(c forge.Context) error {
	ctx := c.Request().Context()

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	status, err := h.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &UserVerificationStatusResponse{
		Status: base.FromSchemaUserVerificationStatus(status),
	})
}

// RequestReverification requests re-verification for the current user
// RequestReverification /verification/me/reverify.
func (h *Handler) RequestReverification(c forge.Context) error {
	ctx := c.Request().Context()

	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	var req struct {
		Reason string `json:"reason"`
	}

	_ = c.BindJSON(&req)

	if err := h.service.RequestReverification(ctx, appID, orgID, userID, req.Reason); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "reverification requested"})
}

// HandleWebhook handles provider webhook callbacks
// HandleWebhook /verification/webhook/:provider.
func (h *Handler) HandleWebhook(c forge.Context) error {
	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("provider"))
	}

	// Read body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to read request body"))
	}

	// Get signature from header
	signature := c.Request().Header.Get("X-Signature")
	if signature == "" {
		// Try provider-specific headers
		switch provider {
		case "stripe_identity":
			signature = c.Request().Header.Get("Stripe-Signature")
		case "onfido":
			signature = c.Request().Header.Get("X-Sha2-Signature")
		}
	}

	// Process webhook
	if err := h.processWebhook(c, provider, signature, body); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("webhook processing failed"))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"received": true,
	})
}

// Admin endpoints

// AdminBlockUser blocks a user from verification (admin only)
// AdminBlockUser /verification/admin/users/:userId/block.
func (h *Handler) AdminBlockUser(c forge.Context) error {
	ctx := c.Request().Context()

	if !h.isAdmin(ctx) {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("admin", "verification"))
	}

	userIDStr := c.Param("userId")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("user_id"))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid user ID"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.BindJSON(&req); err != nil {
		req.Reason = "administrative action"
	}

	if err := h.service.BlockUser(ctx, appID, orgID, userID, req.Reason); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "user blocked"})
}

// AdminUnblockUser unblocks a user (admin only)
// AdminUnblockUser /verification/admin/users/:userId/unblock.
func (h *Handler) AdminUnblockUser(c forge.Context) error {
	ctx := c.Request().Context()

	if !h.isAdmin(ctx) {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("admin", "verification"))
	}

	userIDStr := c.Param("userId")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("user_id"))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid user ID"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	if err := h.service.UnblockUser(ctx, appID, orgID, userID); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "user unblocked"})
}

// AdminGetUserVerificationStatus retrieves verification status for any user (admin only)
// AdminGetUserVerificationStatus /verification/admin/users/:userId/status.
func (h *Handler) AdminGetUserVerificationStatus(c forge.Context) error {
	ctx := c.Request().Context()

	if !h.isAdmin(ctx) {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("admin", "verification"))
	}

	userIDStr := c.Param("userId")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("user_id"))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid user ID"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	orgID, ok := contexts.GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("organization context required"))
	}

	status, err := h.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &UserVerificationStatusResponse{
		Status: base.FromSchemaUserVerificationStatus(status),
	})
}

// AdminGetUserVerifications retrieves all verifications for any user (admin only)
// AdminGetUserVerifications /verification/admin/users/:userId/verifications.
func (h *Handler) AdminGetUserVerifications(c forge.Context) error {
	ctx := c.Request().Context()

	if !h.isAdmin(ctx) {
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("admin", "verification"))
	}

	userIDStr := c.Param("userId")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("user_id"))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid user ID"))
	}

	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	verifications, err := h.service.GetUserVerifications(ctx, appID, userID, limit, offset)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, &VerificationListResponse{
		Verifications: base.FromSchemaIdentityVerifications(verifications),
		Limit:         limit,
		Offset:        offset,
	})
}

// Helper methods

func (h *Handler) processWebhook(c forge.Context, provider, signature string, payload []byte) error {
	ctx := c.Request().Context()
	// p the provider instance
	var p Provider

	for _, prov := range h.service.providers {
		if prov.GetProviderName() == provider {
			p = prov

			break
		}
	}

	if p == nil {
		return ErrUnsupportedProvider
	}

	// Verify webhook signature
	valid, err := p.VerifyWebhook(signature, string(payload))
	if err != nil {
		return fmt.Errorf("webhook verification failed: %w", err)
	}

	if !valid {
		return ErrProviderWebhookInvalid
	}

	// Parse webhook
	webhook, err := p.ParseWebhook(payload)
	if err != nil {
		return fmt.Errorf("webhook parsing failed: %w", err)
	}

	// Get app ID from context or extract from webhook data
	// For webhooks, we may need to extract app ID from webhook payload or use a default
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		// For webhooks, we might need to extract from provider data
		// For now, log and skip if no app context
		return errs.BadRequest("app context required for webhook processing")
	}

	// Process based on event type
	switch webhook.EventType {
	case "check.completed", "verification.completed", "identity.verification_session.verified":
		// verification the verification record
		var verification *schema.IdentityVerification

		if webhook.CheckID != "" {
			verification, err = h.service.repo.GetVerificationByProviderCheckID(ctx, appID, webhook.CheckID)
		} else if webhook.SessionID != "" {
			// Get session and find related verification
			session, err := h.service.repo.GetSessionByID(ctx, appID, webhook.SessionID)
			if err == nil && session != nil {
				// Get latest verification for user
				userID, _ := xid.FromString(session.UserID)
				verification, err = h.service.repo.GetLatestVerificationByUser(ctx, appID, userID)
			}
		}

		if err != nil || verification == nil {
			// Create new verification if not found
			// This might be a standalone check initiated through provider dashboard
			return nil
		}

		// Process the result
		result := convertWebhookToResult(webhook)

		return h.service.ProcessVerificationResult(ctx, appID, verification.ID, result)

	case "check.failed", "verification.failed", "identity.verification_session.requires_input":
		// Handle failure cases
		// Similar logic to completed case

	default:
		// Log unknown event type
		return nil
	}

	return nil
}

func (h *Handler) isAdmin(ctx context.Context) bool {
	// Get auth context
	authCtx, ok := contexts.GetAuthContext(ctx)
	if !ok || authCtx == nil {
		return false
	}

	// Check if can perform admin operations (must have admin scope)
	return authCtx.CanPerformAdminOp()
}

func handleError(c forge.Context, err error) error {
	switch {
	case errors.Is(err, ErrVerificationNotFound), errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrDocumentNotFound):
		return c.JSON(http.StatusNotFound, errs.NotFound(err.Error()))
	case errors.Is(err, ErrVerificationBlocked), errors.Is(err, ErrMaxAttemptsReached), errors.Is(err, ErrRateLimitExceeded):
		return c.JSON(http.StatusTooManyRequests, errs.RateLimitExceeded(0))
	case errors.Is(err, ErrHighRiskDetected), errors.Is(err, ErrSanctionsListMatch), errors.Is(err, ErrPEPDetected), errors.Is(err, ErrAgeBelowMinimum):
		return c.JSON(http.StatusForbidden, errs.PermissionDenied("", err.Error()))
	case errors.Is(err, ErrVerificationExpired), errors.Is(err, ErrSessionExpired), errors.Is(err, ErrDocumentExpired):
		return c.JSON(http.StatusGone, errs.BadRequest(err.Error()))
	case errors.Is(err, ErrDocumentNotSupported), errors.Is(err, ErrCountryNotSupported):
		return c.JSON(http.StatusUnprocessableEntity, errs.BadRequest(err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("internal server error"))
	}
}

func convertWebhookToResult(webhook *WebhookPayload) *VerificationResult {
	if webhook.Result == nil {
		return &VerificationResult{
			Status:       webhook.Status,
			ProviderData: webhook.RawPayload,
		}
	}

	result := &VerificationResult{
		Status:            webhook.Result.Status,
		IsVerified:        webhook.Result.Result == "clear",
		RiskScore:         webhook.Result.RiskScore,
		ConfidenceScore:   webhook.Result.ConfidenceScore,
		ProviderData:      webhook.RawPayload,
		FirstName:         webhook.Result.FirstName,
		LastName:          webhook.Result.LastName,
		DateOfBirth:       webhook.Result.DateOfBirth,
		DocumentNumber:    webhook.Result.DocumentNumber,
		DocumentCountry:   webhook.Result.DocumentCountry,
		Nationality:       webhook.Result.Nationality,
		Gender:            webhook.Result.Gender,
		IsOnSanctionsList: webhook.Result.IsOnSanctionsList,
		IsPEP:             webhook.Result.IsPEP,
		LivenessScore:     webhook.Result.LivenessScore,
		IsLive:            webhook.Result.IsLive,
	}

	// Determine risk level based on score
	if result.RiskScore >= 70 {
		result.RiskLevel = "high"
	} else if result.RiskScore >= 40 {
		result.RiskLevel = "medium"
	} else {
		result.RiskLevel = "low"
	}

	// Build rejection reasons
	if !result.IsVerified {
		for _, subResult := range webhook.Result.SubResults {
			if subResult.Result != "clear" && subResult.Reason != "" {
				result.RejectionReasons = append(result.RejectionReasons, subResult.Reason)
			}
		}
	}

	return result
}
