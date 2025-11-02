package idverification

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for identity verification
type Handler struct {
	service *Service
}

// NewHandler creates a new identity verification handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateVerificationSession creates a new verification session
// POST /verification/sessions
func (h *Handler) CreateVerificationSession(c forge.Context) error {
	var req struct {
		Provider       string                 `json:"provider"`
		RequiredChecks []string               `json:"requiredChecks"`
		SuccessURL     string                 `json:"successUrl"`
		CancelURL      string                 `json:"cancelUrl"`
		Config         map[string]interface{} `json:"config"`
		Metadata       map[string]interface{} `json:"metadata"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request body",
		})
	}
	
	// Get user from context (set by auth middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "unauthorized",
		})
	}
	
	orgID := c.Get("organization_id")
	if orgID == nil {
		orgID = "default"
	}
	
	session, err := h.service.CreateVerificationSession(c.Context(), &CreateSessionRequest{
		UserID:         userID.(string),
		OrganizationID: orgID.(string),
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
	
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"session": session,
	})
}

// GetVerificationSession retrieves a verification session
// GET /verification/sessions/:id
func (h *Handler) GetVerificationSession(c forge.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "session ID is required",
		})
	}
	
	session, err := h.service.GetVerificationSession(c.Context(), sessionID)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"session": session,
	})
}

// GetVerification retrieves a verification by ID
// GET /verification/:id
func (h *Handler) GetVerification(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "verification ID is required",
		})
	}
	
	verification, err := h.service.GetVerification(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"verification": verification,
	})
}

// GetUserVerifications retrieves all verifications for the current user
// GET /verification/me
func (h *Handler) GetUserVerifications(c forge.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "unauthorized",
		})
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
	
	verifications, err := h.service.GetUserVerifications(c.Context(), userID.(string), limit, offset)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"verifications": verifications,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetUserVerificationStatus retrieves the verification status for the current user
// GET /verification/me/status
func (h *Handler) GetUserVerificationStatus(c forge.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "unauthorized",
		})
	}
	
	status, err := h.service.GetUserVerificationStatus(c.Context(), userID.(string))
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": status,
	})
}

// RequestReverification requests re-verification for the current user
// POST /verification/me/reverify
func (h *Handler) RequestReverification(c forge.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "unauthorized",
		})
	}
	
	orgID := c.Get("organization_id")
	if orgID == nil {
		orgID = "default"
	}
	
	var req struct {
		Reason string `json:"reason"`
	}
	
	_ = c.BindJSON(&req)
	
	if err := h.service.RequestReverification(c.Context(), userID.(string), orgID.(string), req.Reason); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "reverification requested",
	})
}

// HandleWebhook handles provider webhook callbacks
// POST /verification/webhook/:provider
func (h *Handler) HandleWebhook(c forge.Context) error {
	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "provider is required",
		})
	}
	
	// Read body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "failed to read request body",
		})
	}
	
	// Get signature from header
	signature := c.Request().Header.Get("X-Signature")
	if signature == "" {
		// Try provider-specific headers
		switch provider {
		case "stripe_identity":
			signature = c.Request().Header.Get("Stripe-Signature")
		case "onfido":
			signature = c.Request().Header.Get("X-SHA2-Signature")
		}
	}
	
	// Process webhook
	if err := h.processWebhook(c, provider, signature, body); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "webhook processing failed",
		})
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"received": true,
	})
}

// Admin endpoints

// AdminBlockUser blocks a user from verification (admin only)
// POST /verification/admin/users/:userId/block
func (h *Handler) AdminBlockUser(c forge.Context) error {
	if !h.isAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "forbidden",
		})
	}
	
	userID := c.Param("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "user ID is required",
		})
	}
	
	orgID := c.Get("organization_id")
	if orgID == nil {
		orgID = "default"
	}
	
	var req struct {
		Reason string `json:"reason"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		req.Reason = "administrative action"
	}
	
	if err := h.service.BlockUser(c.Context(), userID, orgID.(string), req.Reason); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "user blocked",
	})
}

// AdminUnblockUser unblocks a user (admin only)
// POST /verification/admin/users/:userId/unblock
func (h *Handler) AdminUnblockUser(c forge.Context) error {
	if !h.isAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "forbidden",
		})
	}
	
	userID := c.Param("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "user ID is required",
		})
	}
	
	orgID := c.Get("organization_id")
	if orgID == nil {
		orgID = "default"
	}
	
	if err := h.service.UnblockUser(c.Context(), userID, orgID.(string)); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "user unblocked",
	})
}

// AdminGetUserVerificationStatus retrieves verification status for any user (admin only)
// GET /verification/admin/users/:userId/status
func (h *Handler) AdminGetUserVerificationStatus(c forge.Context) error {
	if !h.isAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "forbidden",
		})
	}
	
	userID := c.Param("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "user ID is required",
		})
	}
	
	status, err := h.service.GetUserVerificationStatus(c.Context(), userID)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": status,
	})
}

// AdminGetUserVerifications retrieves all verifications for any user (admin only)
// GET /verification/admin/users/:userId/verifications
func (h *Handler) AdminGetUserVerifications(c forge.Context) error {
	if !h.isAdmin(c) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "forbidden",
		})
	}
	
	userID := c.Param("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "user ID is required",
		})
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
	
	verifications, err := h.service.GetUserVerifications(c.Context(), userID, limit, offset)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"verifications": verifications,
		"limit":         limit,
		"offset":        offset,
	})
}

// Helper methods

func (h *Handler) processWebhook(ctx forge.Context, provider, signature string, payload []byte) error {
	// Get the provider instance
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
	
	// Process based on event type
	switch webhook.EventType {
	case "check.completed", "verification.completed", "identity.verification_session.verified":
		// Get the verification record
		var verification *schema.IdentityVerification
		
		if webhook.CheckID != "" {
			verification, err = h.service.repo.GetVerificationByProviderCheckID(ctx.Context(), webhook.CheckID)
		} else if webhook.SessionID != "" {
			// Get session and find related verification
			session, err := h.service.repo.GetSessionByID(ctx.Context(), webhook.SessionID)
			if err == nil && session != nil {
				// Get latest verification for user
				verification, err = h.service.repo.GetLatestVerificationByUser(ctx.Context(), session.UserID)
			}
		}
		
		if err != nil || verification == nil {
			// Create new verification if not found
			// This might be a standalone check initiated through provider dashboard
			return nil
		}
		
		// Process the result
		result := convertWebhookToResult(webhook)
		return h.service.ProcessVerificationResult(ctx.Context(), verification.ID, result)
		
	case "check.failed", "verification.failed", "identity.verification_session.requires_input":
		// Handle failure cases
		// Similar logic to completed case
		
	default:
		// Log unknown event type
		return nil
	}
	
	return nil
}

func (h *Handler) isAdmin(c forge.Context) bool {
	role := c.Get("role")
	if role == nil {
		return false
	}
	
	return role == "admin" || role == "super_admin"
}

func handleError(c forge.Context, err error) error {
	switch err {
	case ErrVerificationNotFound, ErrSessionNotFound, ErrDocumentNotFound:
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": err.Error(),
		})
		
	case ErrVerificationBlocked, ErrMaxAttemptsReached, ErrRateLimitExceeded:
		return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
			"error": err.Error(),
		})
		
	case ErrHighRiskDetected, ErrSanctionsListMatch, ErrPEPDetected, ErrAgeBelowMinimum:
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": err.Error(),
		})
		
	case ErrVerificationExpired, ErrSessionExpired, ErrDocumentExpired:
		return c.JSON(http.StatusGone, map[string]interface{}{
			"error": err.Error(),
		})
		
	case ErrDocumentNotSupported, ErrCountryNotSupported:
		return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error": err.Error(),
		})
		
	default:
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "internal server error",
		})
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
		Status:           webhook.Result.Status,
		IsVerified:       webhook.Result.Result == "clear",
		RiskScore:        webhook.Result.RiskScore,
		ConfidenceScore:  webhook.Result.ConfidenceScore,
		ProviderData:     webhook.RawPayload,
		FirstName:        webhook.Result.FirstName,
		LastName:         webhook.Result.LastName,
		DateOfBirth:      webhook.Result.DateOfBirth,
		DocumentNumber:   webhook.Result.DocumentNumber,
		DocumentCountry:  webhook.Result.DocumentCountry,
		Nationality:      webhook.Result.Nationality,
		Gender:           webhook.Result.Gender,
		IsOnSanctionsList: webhook.Result.IsOnSanctionsList,
		IsPEP:            webhook.Result.IsPEP,
		LivenessScore:    webhook.Result.LivenessScore,
		IsLive:           webhook.Result.IsLive,
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

