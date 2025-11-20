package anonymous

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse

// Request types
type SignInRequest struct {
	// Empty for now, could add options later
}

type LinkRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=8" example:"password123"`
	Name     string `json:"name" example:"John Doe"`
}

// Plugin-specific responses
type SignInResponse struct {
	Token   string      `json:"token" example:"session_token_abc123"`
	Session interface{} `json:"session"`
	User    interface{} `json:"user"`
}

type LinkResponse struct {
	User    interface{} `json:"user"`
	Message string      `json:"message"`
}

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// SignIn creates a guest user and session
func (h *Handler) SignIn(c forge.Context) error {
	// Get app and environment context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest), "APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	envID, ok := contexts.GetEnvironmentID(c.Request().Context())
	if !ok || envID.IsNil() {
		return handleError(c, errs.New("ENVIRONMENT_CONTEXT_REQUIRED", "Environment context required", http.StatusBadRequest), "ENVIRONMENT_CONTEXT_REQUIRED", "Environment context required", http.StatusBadRequest)
	}

	// Get optional organization context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Optional body (for future extensions)
	var req SignInRequest
	_ = json.NewDecoder(c.Request().Body).Decode(&req)

	// Create guest session
	sess, err := h.svc.SignInGuest(c.Request().Context(), appID, envID, orgIDPtr, c.Request().RemoteAddr, c.Request().UserAgent())
	if err != nil {
		return handleError(c, err, "SIGNIN_FAILED", "Failed to create anonymous session", http.StatusInternalServerError)
	}

	// Get user for response
	user, _ := h.svc.GetUserByID(c.Request().Context(), sess.UserID)

	return c.JSON(http.StatusOK, SignInResponse{
		Token:   sess.Token,
		Session: sess,
		User:    user,
	})
}

// Link upgrades an anonymous session to a real account
func (h *Handler) Link(c forge.Context) error {
	var req LinkRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Read session token from cookie or header
	token := ""
	if ck, err := c.Request().Cookie("session_token"); err == nil && ck != nil {
		token = ck.Value
	}
	if token == "" {
		// Try Authorization header
		if authHeader := c.Request().Header.Get("Authorization"); authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("SESSION_REQUIRED", "Session token required", http.StatusUnauthorized))
	}

	u, err := h.svc.LinkGuest(c.Request().Context(), token, req.Email, req.Password, req.Name)
	if err != nil {
		return handleError(c, err, "LINK_FAILED", "Failed to link anonymous account", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, LinkResponse{
		User:    u,
		Message: fmt.Sprintf("Successfully linked account to %s", u.Email),
	})
}
