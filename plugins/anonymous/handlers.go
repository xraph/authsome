package anonymous

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/helpers"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc      *Service
	authInst core.Authsome
}

func NewHandler(s *Service, authInst core.Authsome) *Handler {
	return &Handler{svc: s, authInst: authInst}
}

// ErrorResponse is the error response type.
type ErrorResponse = responses.ErrorResponse

// SignInRequest is the request type for anonymous sign-in.
type SignInRequest struct {
	// Empty for now, could add options later
}

type LinkRequest struct {
	Email    string `example:"user@example.com" json:"email"    validate:"required,email"`
	Password string `example:"password123"      json:"password" validate:"required,min=8"`
	Name     string `example:"John Doe"         json:"name"`
}

// SignInResponse is the response type for anonymous sign-in.
type SignInResponse struct {
	Token   string `example:"session_token_abc123" json:"token"`
	Session any    `json:"session"`
	User    any    `json:"user"`
}

type LinkResponse struct {
	User    any    `json:"user"`
	Message string `json:"message"`
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// SignIn creates a guest user and session.
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

	_ = c.BindRequest(&req)

	// Create guest session
	sess, err := h.svc.SignInGuest(c.Request().Context(), appID, envID, orgIDPtr, c.Request().RemoteAddr, c.Request().UserAgent())
	if err != nil {
		return handleError(c, err, "SIGNIN_FAILED", "Failed to create anonymous session", http.StatusInternalServerError)
	}

	// Get user for response
	user, _ := h.svc.GetUserByID(c.Request().Context(), sess.UserID)

	// Set session cookie if enabled
	if h.authInst != nil {
		_ = helpers.SetSessionCookieFromAuth(c, h.authInst, sess.Token, sess.ExpiresAt)
	}

	return c.JSON(http.StatusOK, SignInResponse{
		Token:   sess.Token,
		Session: sess,
		User:    user,
	})
}

// Link upgrades an anonymous session to a real account.
func (h *Handler) Link(c forge.Context) error {
	var req LinkRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
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
		Message: "Successfully linked account to " + u.Email,
	})
}
