package auth

import "github.com/xraph/authsome/core/responses"

// SignUpRequest represents a signup request
type SignUpRequest struct {
	Email     string
	Password  string
	Name      string
	Remember  bool
	IPAddress string
	UserAgent string
}

// SignInRequest represents a signin request
type SignInRequest struct {
	Email    string
	Password string
	Remember bool
	// Optional alternative naming per docs
	RememberMe bool
	IPAddress  string
	UserAgent  string
}

// SignOutRequest represents a signout request
type SignOutRequest struct {
	Token string
}

// AuthResponse represents an authentication response
type AuthResponse = responses.AuthResponse
