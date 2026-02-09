package user

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// USER-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeUserNotFound          = "USER_NOT_FOUND"
	CodeUserAlreadyExists     = "USER_ALREADY_EXISTS"
	CodeEmailAlreadyExists    = "EMAIL_ALREADY_EXISTS"
	CodeUsernameAlreadyExists = "USERNAME_ALREADY_EXISTS"
	CodeInvalidEmail          = "INVALID_EMAIL"
	CodeInvalidUsername       = "INVALID_USERNAME"
	CodeWeakPassword          = "WEAK_PASSWORD"
	CodeUserCreationFailed    = "USER_CREATION_FAILED"
	CodeUserUpdateFailed      = "USER_UPDATE_FAILED"
	CodeUserDeletionFailed    = "USER_DELETION_FAILED"
	CodeInvalidUserData       = "INVALID_USER_DATA"
	CodeEmailTaken            = "EMAIL_TAKEN"
	CodeUsernameTaken         = "USERNAME_TAKEN"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// UserNotFound returns an error when a user cannot be found.
func UserNotFound(identifier string) *errs.AuthsomeError {
	return errs.New(CodeUserNotFound, "User not found", http.StatusNotFound).
		WithContext("identifier", identifier)
}

// UserAlreadyExists returns an error when attempting to create a duplicate user.
func UserAlreadyExists(email string) *errs.AuthsomeError {
	return errs.New(CodeUserAlreadyExists, "User already exists", http.StatusConflict).
		WithContext("email", email)
}

// EmailAlreadyExists returns an error when an email is already registered.
func EmailAlreadyExists(email string) *errs.AuthsomeError {
	return errs.New(CodeEmailAlreadyExists, "Email address already registered", http.StatusConflict).
		WithContext("email", email)
}

// UsernameAlreadyExists returns an error when a username is already taken.
func UsernameAlreadyExists(username string) *errs.AuthsomeError {
	return errs.New(CodeUsernameAlreadyExists, "Username already taken", http.StatusConflict).
		WithContext("username", username)
}

// InvalidEmail returns an error for invalid email format.
func InvalidEmail(email string) *errs.AuthsomeError {
	return errs.New(CodeInvalidEmail, "Invalid email address format", http.StatusBadRequest).
		WithContext("email", email)
}

// InvalidUsername returns an error for invalid username format.
func InvalidUsername(username, reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidUsername, "Invalid username format", http.StatusBadRequest).
		WithContext("username", username).
		WithContext("reason", reason)
}

// WeakPassword returns an error when password doesn't meet requirements.
func WeakPassword(reason string) *errs.AuthsomeError {
	return errs.New(CodeWeakPassword, "Password does not meet security requirements", http.StatusBadRequest).
		WithContext("reason", reason)
}

// UserCreationFailed returns an error when user creation fails.
func UserCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeUserCreationFailed, "Failed to create user", http.StatusInternalServerError)
}

// UserUpdateFailed returns an error when user update fails.
func UserUpdateFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeUserUpdateFailed, "Failed to update user", http.StatusInternalServerError)
}

// UserDeletionFailed returns an error when user deletion fails.
func UserDeletionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeUserDeletionFailed, "Failed to delete user", http.StatusInternalServerError)
}

// InvalidUserData returns an error for invalid user data.
func InvalidUserData(field, reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidUserData, "Invalid user data", http.StatusBadRequest).
		WithContext("field", field).
		WithContext("reason", reason)
}

// EmailTaken returns an error when email is taken in the same app.
func EmailTaken(email string) *errs.AuthsomeError {
	return errs.New(CodeEmailTaken, "Email already taken", http.StatusConflict).
		WithContext("email", email)
}

// UsernameTaken returns an error when username is taken.
func UsernameTaken(username string) *errs.AuthsomeError {
	return errs.New(CodeUsernameTaken, "Username already taken", http.StatusConflict).
		WithContext("username", username)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrUserNotFound          = &errs.AuthsomeError{Code: CodeUserNotFound}
	ErrUserAlreadyExists     = &errs.AuthsomeError{Code: CodeUserAlreadyExists}
	ErrEmailAlreadyExists    = &errs.AuthsomeError{Code: CodeEmailAlreadyExists}
	ErrUsernameAlreadyExists = &errs.AuthsomeError{Code: CodeUsernameAlreadyExists}
	ErrInvalidEmail          = &errs.AuthsomeError{Code: CodeInvalidEmail}
	ErrInvalidUsername       = &errs.AuthsomeError{Code: CodeInvalidUsername}
	ErrWeakPassword          = &errs.AuthsomeError{Code: CodeWeakPassword}
	ErrUserCreationFailed    = &errs.AuthsomeError{Code: CodeUserCreationFailed}
	ErrUserUpdateFailed      = &errs.AuthsomeError{Code: CodeUserUpdateFailed}
	ErrUserDeletionFailed    = &errs.AuthsomeError{Code: CodeUserDeletionFailed}
	ErrInvalidUserData       = &errs.AuthsomeError{Code: CodeInvalidUserData}
)
