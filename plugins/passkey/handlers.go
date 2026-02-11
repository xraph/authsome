package passkey

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc *Service
}

// ErrorResponse types - use shared responses from core.
//
//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// BeginRegister initiates passkey registration with WebAuthn challenge.
func (h *Handler) BeginRegister(c forge.Context) error {
	var req BeginRegisterRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	resp, err := h.svc.BeginRegistration(c.Request().Context(), userID, req)
	if err != nil {
		return handleError(c, err, "BEGIN_REGISTRATION_FAILED", "Failed to begin passkey registration", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// FinishRegister completes passkey registration with attestation verification.
func (h *Handler) FinishRegister(c forge.Context) error {
	var req FinishRegisterRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	// Extract IP and user agent
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()

	// Marshal response to JSON bytes
	credentialBytes, err := json.Marshal(req.Response)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid credential response format"))
	}

	resp, err := h.svc.FinishRegistration(c.Request().Context(), userID, credentialBytes, req.Name, ip, ua)
	if err != nil {
		return handleError(c, err, "FINISH_REGISTRATION_FAILED", "Failed to complete passkey registration", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// BeginLogin initiates passkey authentication with WebAuthn challenge.
func (h *Handler) BeginLogin(c forge.Context) error {
	var req BeginLoginRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	// For discoverable credentials, userID is optional
	if req.UserID == "" {
		// Use discoverable login flow
		resp, err := h.svc.BeginDiscoverableLogin(c.Request().Context(), req)
		if err != nil {
			return handleError(c, err, "BEGIN_LOGIN_FAILED", "Failed to begin passkey login", http.StatusBadRequest)
		}

		return c.JSON(http.StatusOK, resp)
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	resp, err := h.svc.BeginLogin(c.Request().Context(), userID, req)
	if err != nil {
		return handleError(c, err, "BEGIN_LOGIN_FAILED", "Failed to begin passkey login", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// FinishLogin completes passkey authentication with signature verification.
func (h *Handler) FinishLogin(c forge.Context) error {
	var req FinishLoginRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	// Extract IP and user agent
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()

	// Marshal response to JSON bytes
	credentialBytes, err := json.Marshal(req.Response)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid credential response format"))
	}

	resp, err := h.svc.FinishLogin(c.Request().Context(), credentialBytes, req.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "FINISH_LOGIN_FAILED", "Failed to complete passkey login", http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, resp)
}

// List retrieves all passkeys for a user.
func (h *Handler) List(c forge.Context) error {
	var req ListPasskeysRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	resp, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err, "LIST_PASSKEYS_FAILED", "Failed to list passkeys", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// Update updates a passkey's metadata (name).
func (h *Handler) Update(c forge.Context) error {
	var req UpdatePasskeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	passkeyID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid passkey ID"))
	}

	resp, err := h.svc.Update(c.Request().Context(), passkeyID, req.Name)
	if err != nil {
		return handleError(c, err, "UPDATE_PASSKEY_FAILED", "Failed to update passkey", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, resp)
}

// Delete removes a passkey.
func (h *Handler) Delete(c forge.Context) error {
	var req DeletePasskeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	passkeyID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid passkey ID"))
	}

	// Extract IP and user agent
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()

	err = h.svc.Delete(c.Request().Context(), passkeyID, ip, ua)
	if err != nil {
		return handleError(c, err, "DELETE_PASSKEY_FAILED", "Failed to delete passkey", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &StatusResponse{Status: "deleted"})
}

// Get retrieves a single passkey by ID.
func (h *Handler) Get(c forge.Context) error {
	var req GetPasskeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	passkeyID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid passkey ID"))
	}

	// Get single passkey - could be implemented in service if needed
	// _ now, return not implemented
	_ = passkeyID

	return c.JSON(http.StatusNotImplemented, errs.NotImplemented("get single passkey"))
}
