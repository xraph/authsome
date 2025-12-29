package magiclink

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/authflow"
	"github.com/xraph/authsome/core/contexts"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc            *Service
	rl             *rl.Service
	authInst       core.Authsome
	authCompletion *authflow.CompletionService
}

func NewHandler(s *Service, rls *rl.Service, authInst core.Authsome, authCompletion *authflow.CompletionService) *Handler {
	return &Handler{
		svc:            s,
		rl:             rls,
		authInst:       authInst,
		authCompletion: authCompletion,
	}
}

// Request types
type SendRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// Response types
type ErrorResponse = responses.ErrorResponse
type VerifyResponse = responses.VerifyResponse

type SendResponse struct {
	Status string `json:"status" example:"sent"`
	DevURL string `json:"dev_url,omitempty" example:"http://localhost:3000/magic-link/verify?token=abc123"`
}

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) Send(c forge.Context) error {
	var req SendRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app context (required for Send)
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest), "APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	// Rate limiting
	if h.rl != nil {
		key := "magiclink:send:" + req.Email
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/magic-link/send")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errs.New("RATE_LIMIT_ERROR", "Rate limit check failed", http.StatusInternalServerError).WithError(err))
		}
		if !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests, please try again later", http.StatusTooManyRequests))
		}
	}

	// Extract IP and UA
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()

	// Call service with explicit appID
	url, err := h.svc.Send(c.Request().Context(), appID, req.Email, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_MAGIC_LINK_FAILED", "Failed to send magic link", http.StatusBadRequest)
	}

	// Return structured response
	response := SendResponse{Status: "sent"}
	if url != "" {
		response.DevURL = url
	}
	return c.JSON(http.StatusOK, response)
}

func (h *Handler) Verify(c forge.Context) error {
	q := c.Request().URL.Query()
	token := q.Get("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Token parameter is required", http.StatusBadRequest))
	}
	remember := q.Get("remember") == "true"

	// Get app and environment context (required for session creation)
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

	// Extract IP and UA
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()

	// Call service to verify token and get user info
	result, err := h.svc.Verify(c.Request().Context(), appID, envID, orgIDPtr, token, remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_MAGIC_LINK_FAILED", "Failed to verify magic link", http.StatusBadRequest)
	}

	// Use centralized authentication completion service for signup/signin
	if h.authCompletion != nil {
		var authResp *responses.AuthResponse
		var authErr error

		if result.IsNewUser {
			// New user signup - CompleteSignUpOrSignIn will create user and add membership
			// Generate a random password for magic link users
			pwd, pwdErr := crypto.GenerateToken(16)
			if pwdErr != nil {
				return handleError(c, pwdErr, "PASSWORD_GENERATION_FAILED", "Failed to generate password", http.StatusInternalServerError)
			}
			
			// Extract name from email
			name := result.Email
			if at := strings.Index(result.Email, "@"); at > 0 {
				name = result.Email[:at]
			}

			authResp, authErr = h.authCompletion.CompleteSignUpOrSignIn(&authflow.CompleteSignUpOrSignInRequest{
				Email:        result.Email,
				Password:     pwd,
				Name:         name,
				User:         nil,
				IsNewUser:    true,
				RememberMe:   remember,
				IPAddress:    ip,
				UserAgent:    ua,
				Context:      c.Request().Context(),
				ForgeContext: c,
				AuthMethod:   "magiclink",
				AuthProvider: "",
			})
		} else {
			// Existing user signin - CompleteSignUpOrSignIn will create session and check membership
			authResp, authErr = h.authCompletion.CompleteSignUpOrSignIn(&authflow.CompleteSignUpOrSignInRequest{
				Email:        result.Email,
				Password:     "",
				Name:         result.User.Name,
				User:         result.User,
				IsNewUser:    false,
				RememberMe:   remember,
				IPAddress:    ip,
				UserAgent:    ua,
				Context:      c.Request().Context(),
				ForgeContext: c,
				AuthMethod:   "magiclink",
				AuthProvider: "",
			})
		}

		if authErr != nil {
			return handleError(c, authErr, "AUTH_COMPLETION_FAILED", "Failed to complete authentication", http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, VerifyResponse{
			User:    authResp.User,
			Session: authResp.Session,
			Token:   authResp.Token,
		})
	}

	// Fallback if completion service not available (should not happen)
	return handleError(c, errs.New("AUTH_COMPLETION_NOT_AVAILABLE", "Authentication completion service not available", http.StatusInternalServerError), "AUTH_COMPLETION_NOT_AVAILABLE", "Authentication completion service not available", http.StatusInternalServerError)
}
