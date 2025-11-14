package passkey

import (
	"encoding/json"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) BeginRegister(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	userID, err := xid.FromString(body.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}
	res, err := h.svc.BeginRegistration(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err, "BEGIN_REGISTRATION_FAILED", "Failed to begin passkey registration", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) FinishRegister(c forge.Context) error {
	var body struct {
		UserID       string `json:"user_id"`
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	userID, err := xid.FromString(body.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	if err := h.svc.FinishRegistration(c.Request().Context(), userID, body.CredentialID, ip, ua); err != nil {
		return handleError(c, err, "FINISH_REGISTRATION_FAILED", "Failed to complete passkey registration", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "registered"})
}

func (h *Handler) BeginLogin(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	userID, err := xid.FromString(body.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}
	res, err := h.svc.BeginLogin(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err, "BEGIN_LOGIN_FAILED", "Failed to begin passkey login", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) FinishLogin(c forge.Context) error {
	var body struct {
		UserID   string `json:"user_id"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	userID, err := xid.FromString(body.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	res, err := h.svc.FinishLogin(c.Request().Context(), userID, body.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "FINISH_LOGIN_FAILED", "Failed to complete passkey login", http.StatusUnauthorized)
	}
	return c.JSON(http.StatusOK, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}

func (h *Handler) List(c forge.Context) error {
	userIDStr := c.Request().URL.Query().Get("user_id")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID parameter is required", http.StatusBadRequest))
	}
	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}
	out, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err, "LIST_PASSKEYS_FAILED", "Failed to list passkeys", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, out)
}

func (h *Handler) Delete(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PASSKEY_ID", "Passkey ID is required", http.StatusBadRequest))
	}
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_PASSKEY_ID", "Invalid passkey ID format", http.StatusBadRequest).WithError(err))
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	if err := h.svc.Delete(c.Request().Context(), id, ip, ua); err != nil {
		return handleError(c, err, "DELETE_PASSKEY_FAILED", "Failed to delete passkey", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
