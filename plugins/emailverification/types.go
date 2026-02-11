package emailverification

import (
	"time"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// SendRequest represents request types.
type SendRequest struct {
	Email string `example:"user@example.com" json:"email" validate:"required,email"`
}

type SendResponse struct {
	Status   string `example:"sent"         json:"status"`
	DevToken string `example:"abc123xyz789" json:"devToken,omitempty"`
}

type VerifyRequest struct {
	Token string `example:"abc123xyz789" query:"token" validate:"required"`
}

type VerifyResponse struct {
	Success bool             `example:"true"                 json:"success"`
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session,omitempty"`
	Token   string           `example:"session_token_abc123" json:"token,omitempty"`
}

type ResendRequest struct {
	Email string `example:"user@example.com" json:"email" validate:"required,email"`
}

type ResendResponse struct {
	Status string `example:"sent" json:"status"`
}

type StatusResponse struct {
	EmailVerified   bool       `example:"true"                 json:"emailVerified"`
	EmailVerifiedAt *time.Time `example:"2024-12-13T15:45:00Z" json:"emailVerifiedAt,omitempty"`
}

// ErrorResponse types - use shared responses from core.
//
//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse

// Error definitions.
var (
	ErrTokenNotFound     = errs.New("TOKEN_NOT_FOUND", "Verification token not found or invalid", 404)
	ErrTokenExpired      = errs.New("TOKEN_EXPIRED", "Verification token has expired", 410)
	ErrTokenAlreadyUsed  = errs.New("TOKEN_USED", "Verification token has already been used", 410)
	ErrAlreadyVerified   = errs.New("ALREADY_VERIFIED", "Email address is already verified", 400)
	ErrRateLimitExceeded = errs.New("RATE_LIMIT_EXCEEDED", "Too many verification requests, please try again later", 429)
	ErrUserNotFound      = errs.New("USER_NOT_FOUND", "User not found", 404)
	ErrInvalidEmail      = errs.New("INVALID_EMAIL", "Invalid email address", 400)
)
