package environment

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// ENVIRONMENT-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeEnvironmentNotFound                = "ENVIRONMENT_NOT_FOUND"
	CodeEnvironmentAlreadyExists           = "ENVIRONMENT_ALREADY_EXISTS"
	CodeEnvironmentSlugAlreadyExists       = "ENVIRONMENT_SLUG_ALREADY_EXISTS"
	CodeDefaultEnvironmentNotFound         = "DEFAULT_ENVIRONMENT_NOT_FOUND"
	CodeCannotDeleteDefaultEnvironment     = "CANNOT_DELETE_DEFAULT_ENVIRONMENT"
	CodeCannotDeleteProductionEnvironment  = "CANNOT_DELETE_PRODUCTION_ENVIRONMENT"
	CodeCannotModifyDefaultEnvironmentType = "CANNOT_MODIFY_DEFAULT_ENVIRONMENT_TYPE"
	CodeEnvironmentTypeForbidden           = "ENVIRONMENT_TYPE_FORBIDDEN"
	CodeEnvironmentLimitReached            = "ENVIRONMENT_LIMIT_REACHED"
	CodePromotionNotAllowed                = "PROMOTION_NOT_ALLOWED"
	CodePromotionFailed                    = "PROMOTION_FAILED"
	CodePromotionNotFound                  = "PROMOTION_NOT_FOUND"
	CodePromotionInProgress                = "PROMOTION_IN_PROGRESS"
	CodeInvalidEnvironmentStatus           = "INVALID_ENVIRONMENT_STATUS"
	CodeInvalidEnvironmentType             = "INVALID_ENVIRONMENT_TYPE"
	CodeSourceEnvironmentNotFound          = "SOURCE_ENVIRONMENT_NOT_FOUND"
	CodeTargetEnvironmentNotFound          = "TARGET_ENVIRONMENT_NOT_FOUND"
	CodeInvalidSlug                        = "INVALID_SLUG"
	CodeInvalidConfig                      = "INVALID_CONFIG"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// Environment CRUD errors
func EnvironmentNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeEnvironmentNotFound, "Environment not found", http.StatusNotFound).
		WithContext("environment_id", id)
}

func EnvironmentAlreadyExists(name string) *errs.AuthsomeError {
	return errs.New(CodeEnvironmentAlreadyExists, "Environment already exists", http.StatusConflict).
		WithContext("name", name)
}

func EnvironmentSlugAlreadyExists(slug string) *errs.AuthsomeError {
	return errs.New(CodeEnvironmentSlugAlreadyExists, "Environment with this slug already exists", http.StatusConflict).
		WithContext("slug", slug)
}

func DefaultEnvironmentNotFound(appID string) *errs.AuthsomeError {
	return errs.New(CodeDefaultEnvironmentNotFound, "Default environment not found for app", http.StatusNotFound).
		WithContext("app_id", appID)
}

// Environment deletion errors
func CannotDeleteDefaultEnvironment() *errs.AuthsomeError {
	return errs.New(CodeCannotDeleteDefaultEnvironment, "Cannot delete default environment", http.StatusForbidden)
}

func CannotDeleteProductionEnvironment() *errs.AuthsomeError {
	return errs.New(CodeCannotDeleteProductionEnvironment, "Cannot delete production environment without explicit confirmation", http.StatusForbidden)
}

// Environment modification errors
func CannotModifyDefaultEnvironmentType() *errs.AuthsomeError {
	return errs.New(CodeCannotModifyDefaultEnvironmentType, "Cannot change default environment type", http.StatusForbidden)
}

func EnvironmentTypeForbidden(envType string) *errs.AuthsomeError {
	return errs.New(CodeEnvironmentTypeForbidden, "Environment type is not allowed", http.StatusForbidden).
		WithContext("type", envType)
}

func EnvironmentLimitReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeEnvironmentLimitReached, "Maximum environments per app reached", http.StatusForbidden).
		WithContext("limit", limit)
}

// Promotion errors
func PromotionNotAllowed() *errs.AuthsomeError {
	return errs.New(CodePromotionNotAllowed, "Environment promotion is disabled", http.StatusForbidden)
}

func PromotionFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodePromotionFailed, "Environment promotion failed", http.StatusInternalServerError).
		WithContext("reason", reason)
}

func PromotionNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodePromotionNotFound, "Promotion not found", http.StatusNotFound).
		WithContext("promotion_id", id)
}

func PromotionInProgress(id string) *errs.AuthsomeError {
	return errs.New(CodePromotionInProgress, "Promotion is already in progress", http.StatusConflict).
		WithContext("promotion_id", id)
}

// Validation errors
func InvalidEnvironmentStatus(status string) *errs.AuthsomeError {
	return errs.New(CodeInvalidEnvironmentStatus, "Invalid environment status", http.StatusBadRequest).
		WithContext("status", status)
}

func InvalidEnvironmentType(envType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidEnvironmentType, "Invalid environment type", http.StatusBadRequest).
		WithContext("type", envType)
}

func SourceEnvironmentNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeSourceEnvironmentNotFound, "Source environment not found", http.StatusNotFound).
		WithContext("source_env_id", id)
}

func TargetEnvironmentNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeTargetEnvironmentNotFound, "Target environment not found", http.StatusNotFound).
		WithContext("target_env_id", id)
}

func InvalidSlug(slug string) *errs.AuthsomeError {
	return errs.New(CodeInvalidSlug, "Invalid environment slug", http.StatusBadRequest).
		WithContext("slug", slug)
}

func InvalidConfig(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidConfig, "Invalid environment configuration", http.StatusBadRequest).
		WithContext("reason", reason)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrEnvironmentNotFound                = &errs.AuthsomeError{Code: CodeEnvironmentNotFound}
	ErrEnvironmentAlreadyExists           = &errs.AuthsomeError{Code: CodeEnvironmentAlreadyExists}
	ErrEnvironmentSlugAlreadyExists       = &errs.AuthsomeError{Code: CodeEnvironmentSlugAlreadyExists}
	ErrDefaultEnvironmentNotFound         = &errs.AuthsomeError{Code: CodeDefaultEnvironmentNotFound}
	ErrCannotDeleteDefaultEnvironment     = &errs.AuthsomeError{Code: CodeCannotDeleteDefaultEnvironment}
	ErrCannotDeleteProductionEnvironment  = &errs.AuthsomeError{Code: CodeCannotDeleteProductionEnvironment}
	ErrCannotModifyDefaultEnvironmentType = &errs.AuthsomeError{Code: CodeCannotModifyDefaultEnvironmentType}
	ErrEnvironmentTypeForbidden           = &errs.AuthsomeError{Code: CodeEnvironmentTypeForbidden}
	ErrEnvironmentLimitReached            = &errs.AuthsomeError{Code: CodeEnvironmentLimitReached}
	ErrPromotionNotAllowed                = &errs.AuthsomeError{Code: CodePromotionNotAllowed}
	ErrPromotionFailed                    = &errs.AuthsomeError{Code: CodePromotionFailed}
	ErrPromotionNotFound                  = &errs.AuthsomeError{Code: CodePromotionNotFound}
	ErrPromotionInProgress                = &errs.AuthsomeError{Code: CodePromotionInProgress}
	ErrInvalidEnvironmentStatus           = &errs.AuthsomeError{Code: CodeInvalidEnvironmentStatus}
	ErrInvalidEnvironmentType             = &errs.AuthsomeError{Code: CodeInvalidEnvironmentType}
	ErrSourceEnvironmentNotFound          = &errs.AuthsomeError{Code: CodeSourceEnvironmentNotFound}
	ErrTargetEnvironmentNotFound          = &errs.AuthsomeError{Code: CodeTargetEnvironmentNotFound}
	ErrInvalidSlug                        = &errs.AuthsomeError{Code: CodeInvalidSlug}
	ErrInvalidConfig                      = &errs.AuthsomeError{Code: CodeInvalidConfig}
)
