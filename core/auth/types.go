package auth

import "github.com/xraph/authsome/core/responses"

// SignUpRequest represents a signup request
type SignUpRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	Name       string `json:"name" validate:"required"`
	RememberMe bool   `json:"rememberMe,omitempty"`
	IPAddress  string `json:"ipAddress,omitempty"`
	UserAgent  string `json:"userAgent,omitempty"`
}

// SignInRequest represents a signin request
type SignInRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	RememberMe bool   `json:"rememberMe,omitempty"`
	// Optional alternative naming per docs
	IPAddress string `json:"ipAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// SignOutRequest represents a signout request
type SignOutRequest struct {
	Token string `json:"token" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse = responses.AuthResponse
