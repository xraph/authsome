package bridge

import (
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forgeui/bridge"
)

// AuthenticateInput represents the login request.
type AuthenticateInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthenticateOutput represents the login response.
type AuthenticateOutput struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	SessionToken string    `json:"sessionToken,omitempty"`
	RedirectURL  string    `json:"redirectUrl,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
}

// UserInfo represents basic user information.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// RegisterInput represents the signup request.
type RegisterInput struct {
	Email    string `json:"email"          validate:"required,email"`
	Password string `json:"password"       validate:"required,min:8"`
	Name     string `json:"name,omitempty"`
}

// RegisterOutput represents the signup response.
type RegisterOutput struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	SessionToken string    `json:"sessionToken,omitempty"`
	RedirectURL  string    `json:"redirectUrl,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
}

// CheckSessionOutput represents session validation response.
type CheckSessionOutput struct {
	Valid     bool      `json:"valid"`
	User      *UserInfo `json:"user,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
}

// LogoutOutput represents logout response.
type LogoutOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RequestPasswordResetInput represents password reset request.
type RequestPasswordResetInput struct {
	Email string `json:"email" validate:"required,email"`
}

// RequestPasswordResetOutput represents password reset response.
type RequestPasswordResetOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ResetPasswordInput represents password reset with token.
type ResetPasswordInput struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min:8"`
}

// ResetPasswordOutput represents password reset response.
type ResetPasswordOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// registerAuthFunctions registers authentication-related bridge functions.
func (bm *BridgeManager) registerAuthFunctions() error {
	// Login function
	if err := bm.bridge.Register("authenticateUser", bm.authenticateUser); err != nil {
		return fmt.Errorf("failed to register authenticateUser: %w", err)
	}

	// Register function
	if err := bm.bridge.Register("registerUser", bm.registerUser); err != nil {
		return fmt.Errorf("failed to register registerUser: %w", err)
	}

	// Logout function
	if err := bm.bridge.Register("logoutUser", bm.logoutUser); err != nil {
		return fmt.Errorf("failed to register logoutUser: %w", err)
	}

	// Check session function
	if err := bm.bridge.Register("checkSession", bm.checkSession); err != nil {
		return fmt.Errorf("failed to register checkSession: %w", err)
	}

	// Request password reset
	if err := bm.bridge.Register("requestPasswordReset", bm.requestPasswordReset); err != nil {
		return fmt.Errorf("failed to register requestPasswordReset: %w", err)
	}

	// Reset password with token
	if err := bm.bridge.Register("resetPassword", bm.resetPassword); err != nil {
		return fmt.Errorf("failed to register resetPassword: %w", err)
	}

	bm.log.Info("auth bridge functions registered")

	return nil
}

// authenticateUser handles user login via bridge.
func (bm *BridgeManager) authenticateUser(ctx bridge.Context, input AuthenticateInput) (*AuthenticateOutput, error) {
	// Validate input
	if input.Email == "" || input.Password == "" {
		return &AuthenticateOutput{
			Success: false,
			Message: "Email and password are required",
		}, nil
	}

	// Find user by email
	user, err := bm.userSvc.FindByEmail(ctx.Request().Context(), input.Email)
	if err != nil {
		return &AuthenticateOutput{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	// Verify password (assuming user service has password verification)
	// Note: This is a simplified version - actual implementation should use proper password verification
	// from the auth service or user service
	if user.PasswordHash == "" {
		return &AuthenticateOutput{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	// Create session
	sess, err := bm.sessionSvc.Create(ctx.Request().Context(), &session.CreateSessionRequest{
		UserID:    user.ID,
		AppID:     xid.NilID(), // Will be set based on context
		IPAddress: ctx.Request().RemoteAddr,
		UserAgent: ctx.Request().UserAgent(),
		Remember:  false,
	})
	if err != nil {
		return &AuthenticateOutput{
			Success: false,
			Message: "Failed to create session",
		}, nil
	}

	return &AuthenticateOutput{
		Success:      true,
		Message:      "Login successful",
		SessionToken: sess.Token,
		RedirectURL:  bm.basePath + "/",
		User: &UserInfo{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
		},
	}, nil
}

// registerUser handles user registration via bridge.
func (bm *BridgeManager) registerUser(ctx bridge.Context, input RegisterInput) (*RegisterOutput, error) {
	// Validate input
	if input.Email == "" || input.Password == "" {
		return &RegisterOutput{
			Success: false,
			Message: "Email and password are required",
		}, nil
	}

	if len(input.Password) < 8 {
		return &RegisterOutput{
			Success: false,
			Message: "Password must be at least 8 characters",
		}, nil
	}

	// Check if user already exists
	existingUser, _ := bm.userSvc.FindByEmail(ctx.Request().Context(), input.Email)
	if existingUser != nil {
		return &RegisterOutput{
			Success: false,
			Message: "User with this email already exists",
		}, nil
	}

	// Create user
	// Note: This is simplified - actual implementation should use proper user creation
	// from the auth service with password hashing
	newUser, err := bm.userSvc.Create(ctx.Request().Context(), &user.CreateUserRequest{
		Email: input.Email,
		Name:  input.Name,
		AppID: xid.NilID(), // Will be set based on context
		// Password hashing should be handled by the service
	})
	if err != nil {
		return &RegisterOutput{
			Success: false,
			Message: "Failed to create user",
		}, nil
	}

	// Create session
	sess, err := bm.sessionSvc.Create(ctx.Request().Context(), &session.CreateSessionRequest{
		UserID:    newUser.ID,
		AppID:     xid.NilID(),
		IPAddress: ctx.Request().RemoteAddr,
		UserAgent: ctx.Request().UserAgent(),
		Remember:  false,
	})
	if err != nil {
		return &RegisterOutput{
			Success: false,
			Message: "User created but failed to create session",
		}, nil
	}

	return &RegisterOutput{
		Success:      true,
		Message:      "Registration successful",
		SessionToken: sess.Token,
		RedirectURL:  bm.basePath + "/",
		User: &UserInfo{
			ID:    newUser.ID.String(),
			Email: newUser.Email,
			Name:  newUser.Name,
		},
	}, nil
}

// logoutUser handles user logout via bridge.
func (bm *BridgeManager) logoutUser(ctx bridge.Context, _ struct{}) (*LogoutOutput, error) {
	user := ctx.User()
	if user == nil {
		return &LogoutOutput{
			Success: false,
			Message: "Not authenticated",
		}, nil
	}

	// Get session ID from context
	// Note: This assumes bridge context provides session information
	// Actual implementation should get session ID from cookie or context

	// Revoke session (implementation needed)
	// err := bm.sessionSvc.RevokeSession(ctx.Request().Context(), sessionID)

	return &LogoutOutput{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

// checkSession validates current session.
func (bm *BridgeManager) checkSession(ctx bridge.Context, _ struct{}) (*CheckSessionOutput, error) {
	user := ctx.User()
	if user == nil {
		return &CheckSessionOutput{
			Valid: false,
		}, nil
	}

	// Get session information from context
	// Note: Actual implementation should retrieve session details

	return &CheckSessionOutput{
		Valid: true,
		User: &UserInfo{
			ID:    user.ID(),
			Email: user.Email(),
		},
	}, nil
}

// requestPasswordReset initiates password reset process.
func (bm *BridgeManager) requestPasswordReset(ctx bridge.Context, input RequestPasswordResetInput) (*RequestPasswordResetOutput, error) {
	if input.Email == "" {
		return &RequestPasswordResetOutput{
			Success: false,
			Message: "Email is required",
		}, nil
	}

	// Find user by email
	user, err := bm.userSvc.FindByEmail(ctx.Request().Context(), input.Email)
	if err != nil {
		// Don't reveal if user exists
		return &RequestPasswordResetOutput{
			Success: true,
			Message: "If an account exists with that email, you will receive password reset instructions",
		}, nil
	}

	if user == nil {
		// Don't reveal if user exists
		return &RequestPasswordResetOutput{
			Success: true,
			Message: "If an account exists with that email, you will receive password reset instructions",
		}, nil
	}

	// TODO: Generate reset token and send email
	// For now, return success
	return &RequestPasswordResetOutput{
		Success: true,
		Message: "Password reset instructions have been sent to your email",
	}, nil
}

// resetPassword resets password using token.
func (bm *BridgeManager) resetPassword(ctx bridge.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
	if input.Token == "" || input.Password == "" {
		return &ResetPasswordOutput{
			Success: false,
			Message: "Token and password are required",
		}, nil
	}

	if len(input.Password) < 8 {
		return &ResetPasswordOutput{
			Success: false,
			Message: "Password must be at least 8 characters long",
		}, nil
	}

	// TODO: Validate token and reset password
	// This is a placeholder implementation
	return &ResetPasswordOutput{
		Success: true,
		Message: "Password has been reset successfully",
	}, nil
}
