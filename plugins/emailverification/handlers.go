package emailverification

import (
	"encoding/json"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles email verification HTTP endpoints
type Handler struct {
	svc      *Service
	authInst core.Authsome
}

// NewHandler creates a new email verification handler
func NewHandler(svc *Service, authInst core.Authsome) *Handler {
	return &Handler{
		svc:      svc,
		authInst: authInst,
	}
}

// handleError returns structured error response
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// Send handles manual verification email sending
// POST /email-verification/send
func (h *Handler) Send(c forge.Context) error {
	var req SendRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest),
			"APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	// Find user by email
	user, err := h.svc.users.FindByEmail(c.Request().Context(), req.Email)
	if err != nil || user == nil {
		return handleError(c, ErrUserNotFound, "USER_NOT_FOUND", "User not found", http.StatusNotFound)
	}

	// Check if already verified
	if user.EmailVerified {
		return c.JSON(http.StatusBadRequest, ErrAlreadyVerified)
	}

	// Send verification email
	devToken, err := h.svc.SendVerification(c.Request().Context(), appID, user.ID, req.Email)
	if err != nil {
		return handleError(c, err, "SEND_FAILED", "Failed to send verification email", http.StatusInternalServerError)
	}

	response := &SendResponse{Status: "sent"}
	if devToken != "" {
		response.DevToken = devToken
	}

	return c.JSON(http.StatusOK, response)
}

// Verify handles email verification via token
// GET /email-verification/verify?token=xyz
func (h *Handler) Verify(c forge.Context) error {
	token := c.Query("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("TOKEN_REQUIRED", "Verification token is required", http.StatusBadRequest))
	}

	// Get app context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest),
			"APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	// Get IP and user agent for session creation
	ip := c.Request().RemoteAddr
	ua := c.Request().Header.Get("User-Agent")

	// Verify token
	response, err := h.svc.VerifyToken(c.Request().Context(), appID, token, true, ip, ua)
	if err != nil {
		if err == ErrTokenNotFound {
			return c.JSON(http.StatusNotFound, err)
		} else if err == ErrTokenExpired || err == ErrTokenAlreadyUsed {
			return c.JSON(http.StatusGone, err)
		} else if err == ErrAlreadyVerified {
			return c.JSON(http.StatusBadRequest, err)
		}
		return handleError(c, err, "VERIFY_FAILED", "Failed to verify email", http.StatusInternalServerError)
	}

	// Set session cookie if auto-login is enabled and session was created
	if response.Session != nil && response.Token != "" {
		// Set cookie header
		cookieStr := "session_token=" + response.Token + "; Path=/; HttpOnly; SameSite=Lax"
		c.SetHeader("Set-Cookie", cookieStr)
	}

	return c.JSON(http.StatusOK, response)
}

// Resend handles resending verification email
// POST /email-verification/resend
func (h *Handler) Resend(c forge.Context) error {
	var req ResendRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest),
			"APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	// Resend verification
	err := h.svc.ResendVerification(c.Request().Context(), appID, req.Email)
	if err != nil {
		if err == ErrUserNotFound {
			return c.JSON(http.StatusNotFound, err)
		} else if err == ErrAlreadyVerified {
			return c.JSON(http.StatusBadRequest, err)
		} else if err == ErrRateLimitExceeded {
			return c.JSON(http.StatusTooManyRequests, err)
		}
		return handleError(c, err, "RESEND_FAILED", "Failed to resend verification email", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &ResendResponse{Status: "sent"})
}

// Status handles checking verification status for current user
// GET /email-verification/status (requires authentication)
func (h *Handler) Status(c forge.Context) error {
	// Extract user ID from context (set by auth middleware)
	userID := c.Get("userID")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid, ok := userID.(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusInternalServerError))
	}

	parsedUserID, err := xid.FromString(uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusInternalServerError))
	}

	// Get verification status
	status, err := h.svc.GetStatus(c.Request().Context(), parsedUserID)
	if err != nil {
		return handleError(c, err, "STATUS_FAILED", "Failed to get verification status", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, status)
}
