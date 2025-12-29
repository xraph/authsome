package emailverification

import (
	"time"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// Request types
type SendRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type SendResponse struct {
	Status   string `json:"status" example:"sent"`
	DevToken string `json:"devToken,omitempty" example:"abc123xyz789"`
}

type VerifyRequest struct {
	Token string `query:"token" validate:"required" example:"abc123xyz789"`
}

type VerifyResponse struct {
	Success bool             `json:"success" example:"true"`
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session,omitempty"`
	Token   string           `json:"token,omitempty" example:"session_token_abc123"`
}

type ResendRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type ResendResponse struct {
	Status string `json:"status" example:"sent"`
}

type StatusResponse struct {
	EmailVerified   bool       `json:"emailVerified" example:"true"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty" example:"2024-12-13T15:45:00Z"`
}

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse

// Error definitions
var (
	ErrTokenNotFound     = errs.New("TOKEN_NOT_FOUND", "Verification token not found or invalid", 404)
	ErrTokenExpired      = errs.New("TOKEN_EXPIRED", "Verification token has expired", 410)
	ErrTokenAlreadyUsed  = errs.New("TOKEN_USED", "Verification token has already been used", 410)
	ErrAlreadyVerified   = errs.New("ALREADY_VERIFIED", "Email address is already verified", 400)
	ErrRateLimitExceeded = errs.New("RATE_LIMIT_EXCEEDED", "Too many verification requests, please try again later", 429)
	ErrUserNotFound      = errs.New("USER_NOT_FOUND", "User not found", 404)
	ErrInvalidEmail      = errs.New("INVALID_EMAIL", "Invalid email address", 400)
)
