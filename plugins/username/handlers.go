package username

import (
	"errors"
	"net"
	"net/http"
	"time"

	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for username auth.
type Handler struct {
	svc   *Service
	rl    *rl.Service
	twofa *repo.TwoFARepository
}

// Request types.
type SignUpRequest struct {
	Username string `example:"johndoe"       json:"username" validate:"required"`
	Password string `example:"SecureP@ss123" json:"password" validate:"required"`
}

type SignInRequest struct {
	Username string `example:"johndoe"       json:"username" validate:"required"`
	Password string `example:"SecureP@ss123" json:"password" validate:"required"`
	Remember bool   `example:"false"         json:"remember"`
}

// Response types.
type SignUpResponse struct {
	Status  string `example:"created"                   json:"status"`
	Message string `example:"User created successfully" json:"message,omitempty"`
}

type SignInResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	Token   string           `example:"session_token_abc123" json:"token"`
}

type TwoFARequiredResponse struct {
	User         *user.User `json:"user"`
	RequireTwoFA bool       `example:"true"               json:"require_twofa"`
	DeviceID     string     `example:"device_fingerprint" json:"device_id"`
}

type AccountLockedResponse struct {
	Code          string    `example:"ACCOUNT_LOCKED"                                       json:"code"`
	Message       string    `example:"Account locked due to too many failed login attempts" json:"message"`
	LockedUntil   time.Time `example:"2025-11-20T12:00:00Z"                                 json:"locked_until"`
	LockedMinutes int       `example:"15"                                                   json:"locked_minutes"`
}

func NewHandler(s *Service, rls *rl.Service, tf *repo.TwoFARepository) *Handler {
	return &Handler{svc: s, rl: rls, twofa: tf}
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Check for account lockout error
	lockoutErr := &AccountLockoutError{}
	if errors.As(err, &lockoutErr) {
		return c.JSON(http.StatusForbidden, &AccountLockedResponse{
			Code:          "ACCOUNT_LOCKED",
			Message:       "Account locked due to too many failed login attempts",
			LockedUntil:   lockoutErr.LockedUntil,
			LockedMinutes: lockoutErr.LockedMinutes,
		})
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// extractIP extracts IP address from request.
func extractIP(c forge.Context) string {
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	return ip
}

// SignUp handles user registration with username and password.
func (h *Handler) SignUp(c forge.Context) error {
	var req SignUpRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Extract IP and user agent
	ip := extractIP(c)
	ua := c.Request().UserAgent()

	// Rate limiting
	if h.rl != nil {
		key := "username:signup:" + ip

		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/username/signup")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
		}

		if !ok {
			return c.JSON(http.StatusTooManyRequests, errs.RateLimitExceeded(15*time.Minute))
		}
	}

	// Process signup
	err := h.svc.SignUpWithUsername(c.Request().Context(), req.Username, req.Password, ip, ua)
	if err != nil {
		return handleError(c, err, "SIGNUP_FAILED", "Failed to create user", http.StatusBadRequest)
	}

	return c.JSON(http.StatusCreated, &SignUpResponse{
		Status:  "created",
		Message: "User created successfully",
	})
}

// SignIn handles user authentication with username and password.
func (h *Handler) SignIn(c forge.Context) error {
	var req SignInRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Extract IP and user agent
	ip := extractIP(c)
	ua := c.Request().UserAgent()

	// Rate limiting
	if h.rl != nil {
		key := "username:signin:" + ip

		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/username/signin")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
		}

		if !ok {
			return c.JSON(http.StatusTooManyRequests, errs.RateLimitExceeded(15*time.Minute))
		}
	}

	// Authenticate user
	authRes, err := h.svc.SignInWithUsername(c.Request().Context(), req.Username, req.Password, req.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "SIGNIN_FAILED", "Failed to sign in", http.StatusUnauthorized)
	}

	// Check 2FA requirement
	if h.twofa != nil {
		// Determine device fingerprint from IP and UA
		fp := ua + "|" + ip

		// Check if user has 2FA enabled
		if sec, _ := h.twofa.GetSecret(c.Request().Context(), authRes.User.ID); sec != nil && sec.Enabled {
			trusted, _ := h.twofa.IsTrustedDevice(c.Request().Context(), authRes.User.ID, fp, time.Now())
			if !trusted {
				return c.JSON(http.StatusOK, &TwoFARequiredResponse{
					User:         authRes.User,
					RequireTwoFA: true,
					DeviceID:     fp,
				})
			}
		}
	}

	return c.JSON(http.StatusOK, &SignInResponse{
		User:    authRes.User,
		Session: authRes.Session,
		Token:   authRes.Token,
	})
}
