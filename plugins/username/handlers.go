package username

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for username auth
type Handler struct {
	svc   *Service
	twofa *repo.TwoFARepository
}

func NewHandler(s *Service, tf *repo.TwoFARepository) *Handler { return &Handler{svc: s, twofa: tf} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) SignUp(c forge.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.Username == "" || body.Password == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_FIELDS", "Username and password are required", http.StatusBadRequest))
	}
	if err := h.svc.SignUpWithUsername(c.Request().Context(), body.Username, body.Password); err != nil {
		return handleError(c, err, "SIGNUP_FAILED", "Failed to create user", http.StatusBadRequest)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
}

func (h *Handler) SignIn(c forge.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	un := strings.ToLower(strings.TrimSpace(body.Username))
	if un == "" || body.Password == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_FIELDS", "Username and password are required", http.StatusBadRequest))
	}
	// Lookup user by username and verify password
	u, err := h.svc.users.FindByUsername(c.Request().Context(), un)
	if err != nil || u == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CREDENTIALS", "Invalid username or password", http.StatusUnauthorized))
	}
	if ok := crypto.CheckPassword(body.Password, u.PasswordHash); !ok {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CREDENTIALS", "Invalid username or password", http.StatusUnauthorized))
	}
	// Determine device fingerprint from IP and UA
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	fp := ua + "|" + ip
	// Check 2FA requirement and trusted device
	if h.twofa != nil {
		if sec, _ := h.twofa.GetSecret(c.Request().Context(), u.ID); sec != nil && sec.Enabled {
			trusted, _ := h.twofa.IsTrustedDevice(c.Request().Context(), u.ID, fp, time.Now())
			if !trusted {
				return c.JSON(http.StatusOK, map[string]interface{}{"user": u, "require_twofa": true, "device_id": fp})
			}
		}
	}
	// Create session via core auth service
	res, err := h.svc.auth.CreateSessionForUser(c.Request().Context(), u, body.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "SESSION_FAILED", "Failed to create session", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"user": res.User, "session": res.Session, "token": res.Token})
}
