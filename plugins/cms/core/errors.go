package core

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// Error codes for the CMS plugin
const (
	// Content Type errors
	ErrCodeContentTypeNotFound    = "CMS_CONTENT_TYPE_NOT_FOUND"
	ErrCodeContentTypeExists      = "CMS_CONTENT_TYPE_EXISTS"
	ErrCodeContentTypeHasEntries  = "CMS_CONTENT_TYPE_HAS_ENTRIES"
	ErrCodeInvalidContentTypeSlug = "CMS_INVALID_CONTENT_TYPE_SLUG"
	
	// Content Field errors
	ErrCodeFieldNotFound    = "CMS_FIELD_NOT_FOUND"
	ErrCodeFieldExists      = "CMS_FIELD_EXISTS"
	ErrCodeInvalidFieldSlug = "CMS_INVALID_FIELD_SLUG"
	ErrCodeInvalidFieldType = "CMS_INVALID_FIELD_TYPE"
	ErrCodeFieldRequired    = "CMS_FIELD_REQUIRED"
	
	// Component Schema errors
	ErrCodeComponentSchemaNotFound   = "CMS_COMPONENT_SCHEMA_NOT_FOUND"
	ErrCodeComponentSchemaExists     = "CMS_COMPONENT_SCHEMA_EXISTS"
	ErrCodeComponentSchemaInUse      = "CMS_COMPONENT_SCHEMA_IN_USE"
	ErrCodeInvalidComponentSchemaSlug = "CMS_INVALID_COMPONENT_SCHEMA_SLUG"
	ErrCodeCircularComponentRef      = "CMS_CIRCULAR_COMPONENT_REF"
	
	// Content Entry errors
	ErrCodeEntryNotFound         = "CMS_ENTRY_NOT_FOUND"
	ErrCodeEntryValidationFailed = "CMS_ENTRY_VALIDATION_FAILED"
	ErrCodeEntryAlreadyPublished = "CMS_ENTRY_ALREADY_PUBLISHED"
	ErrCodeEntryNotPublished     = "CMS_ENTRY_NOT_PUBLISHED"
	ErrCodeEntryLimitReached     = "CMS_ENTRY_LIMIT_REACHED"
	ErrCodeUniqueConstraint      = "CMS_UNIQUE_CONSTRAINT_VIOLATION"
	
	// Revision errors
	ErrCodeRevisionNotFound = "CMS_REVISION_NOT_FOUND"
	ErrCodeRollbackFailed   = "CMS_ROLLBACK_FAILED"
	
	// Relation errors
	ErrCodeRelationNotFound      = "CMS_RELATION_NOT_FOUND"
	ErrCodeInvalidRelation       = "CMS_INVALID_RELATION"
	ErrCodeCircularRelation      = "CMS_CIRCULAR_RELATION"
	ErrCodeRelationConstraint    = "CMS_RELATION_CONSTRAINT_VIOLATION"
	
	// Query errors
	ErrCodeInvalidQuery    = "CMS_INVALID_QUERY"
	ErrCodeInvalidFilter   = "CMS_INVALID_FILTER"
	ErrCodeInvalidSort     = "CMS_INVALID_SORT"
	ErrCodeInvalidOperator = "CMS_INVALID_OPERATOR"
	
	// General errors
	ErrCodeAccessDenied     = "CMS_ACCESS_DENIED"
	ErrCodeInvalidRequest   = "CMS_INVALID_REQUEST"
	ErrCodeAppContextMissing = "CMS_APP_CONTEXT_MISSING"
	ErrCodeEnvContextMissing = "CMS_ENV_CONTEXT_MISSING"
)

// =============================================================================
// Content Type Errors
// =============================================================================

// ErrContentTypeNotFound returns a not found error for a content type
func ErrContentTypeNotFound(identifier string) error {
	return errs.New(ErrCodeContentTypeNotFound, "content type not found: "+identifier, http.StatusNotFound)
}

// ErrContentTypeExists returns a conflict error when a content type already exists
func ErrContentTypeExists(slug string) error {
	return errs.New(ErrCodeContentTypeExists, "content type already exists with slug: "+slug, http.StatusConflict)
}

// ErrContentTypeHasEntries returns an error when trying to delete a content type with entries
func ErrContentTypeHasEntries(slug string, entryCount int) error {
	return errs.New(
		ErrCodeContentTypeHasEntries,
		fmt.Sprintf("cannot delete content type '%s': has %d entries", slug, entryCount),
		http.StatusConflict,
	)
}

// ErrInvalidContentTypeSlug returns a bad request error for invalid slug format
func ErrInvalidContentTypeSlug(slug, reason string) error {
	msg := "invalid content type slug: " + slug
	if reason != "" {
		msg += " (" + reason + ")"
	}
	return errs.New(ErrCodeInvalidContentTypeSlug, msg, http.StatusBadRequest)
}

// =============================================================================
// Content Field Errors
// =============================================================================

// ErrFieldNotFound returns a not found error for a content field
func ErrFieldNotFound(identifier string) error {
	return errs.New(ErrCodeFieldNotFound, "field not found: "+identifier, http.StatusNotFound)
}

// ErrFieldExists returns a conflict error when a field already exists
func ErrFieldExists(slug string) error {
	return errs.New(ErrCodeFieldExists, "field already exists with slug: "+slug, http.StatusConflict)
}

// ErrInvalidFieldSlug returns a bad request error for invalid field slug
func ErrInvalidFieldSlug(slug, reason string) error {
	msg := "invalid field slug: " + slug
	if reason != "" {
		msg += " (" + reason + ")"
	}
	return errs.New(ErrCodeInvalidFieldSlug, msg, http.StatusBadRequest)
}

// ErrInvalidFieldType returns a bad request error for invalid field type
func ErrInvalidFieldType(fieldType string) error {
	return errs.New(
		ErrCodeInvalidFieldType,
		"invalid field type: "+fieldType,
		http.StatusBadRequest,
	)
}

// ErrFieldRequired returns a bad request error when a required field is missing
func ErrFieldRequired(fieldName string) error {
	return errs.New(
		ErrCodeFieldRequired,
		"required field missing: "+fieldName,
		http.StatusBadRequest,
	)
}

// =============================================================================
// Component Schema Errors
// =============================================================================

// ErrComponentSchemaNotFound returns a not found error for a component schema
func ErrComponentSchemaNotFound(identifier string) error {
	return errs.New(ErrCodeComponentSchemaNotFound, "component schema not found: "+identifier, http.StatusNotFound)
}

// ErrComponentSchemaExists returns a conflict error when a component schema already exists
func ErrComponentSchemaExists(slug string) error {
	return errs.New(ErrCodeComponentSchemaExists, "component schema already exists with slug: "+slug, http.StatusConflict)
}

// ErrComponentSchemaInUse returns an error when trying to delete a component schema that is in use
func ErrComponentSchemaInUse(slug string, usageCount int) error {
	return errs.New(
		ErrCodeComponentSchemaInUse,
		fmt.Sprintf("cannot delete component schema '%s': used by %d field(s)", slug, usageCount),
		http.StatusConflict,
	)
}

// ErrInvalidComponentSchemaSlug returns a bad request error for invalid component schema slug format
func ErrInvalidComponentSchemaSlug(slug, reason string) error {
	msg := "invalid component schema slug: " + slug
	if reason != "" {
		msg += " (" + reason + ")"
	}
	return errs.New(ErrCodeInvalidComponentSchemaSlug, msg, http.StatusBadRequest)
}

// ErrCircularComponentRef returns an error when a circular component reference is detected
func ErrCircularComponentRef(path string) error {
	return errs.New(
		ErrCodeCircularComponentRef,
		"circular component reference detected: "+path,
		http.StatusBadRequest,
	)
}

// =============================================================================
// Content Entry Errors
// =============================================================================

// ErrEntryNotFound returns a not found error for a content entry
func ErrEntryNotFound(identifier string) error {
	return errs.New(ErrCodeEntryNotFound, "content entry not found: "+identifier, http.StatusNotFound)
}

// ErrEntryValidationFailed returns a bad request error when entry validation fails
func ErrEntryValidationFailed(errors map[string]string) error {
	err := errs.New(
		ErrCodeEntryValidationFailed,
		"entry validation failed",
		http.StatusBadRequest,
	)
	return err.WithDetails(errors)
}

// ErrEntryValidationFailedSingle returns a bad request error for a single validation failure
func ErrEntryValidationFailedSingle(field, reason string) error {
	err := errs.New(
		ErrCodeEntryValidationFailed,
		fmt.Sprintf("validation failed for field '%s': %s", field, reason),
		http.StatusBadRequest,
	)
	return err.WithContext("field", field).WithContext("reason", reason)
}

// ErrEntryAlreadyPublished returns a conflict error when entry is already published
func ErrEntryAlreadyPublished() error {
	return errs.New(ErrCodeEntryAlreadyPublished, "entry is already published", http.StatusConflict)
}

// ErrEntryNotPublished returns a bad request error when entry is not published
func ErrEntryNotPublished() error {
	return errs.New(ErrCodeEntryNotPublished, "entry is not published", http.StatusBadRequest)
}

// ErrEntryLimitReached returns an error when entry limit is reached for a content type
func ErrEntryLimitReached(contentType string, limit int) error {
	return errs.New(
		ErrCodeEntryLimitReached,
		fmt.Sprintf("entry limit (%d) reached for content type '%s'", limit, contentType),
		http.StatusForbidden,
	)
}

// ErrUniqueConstraint returns an error when a unique constraint is violated
func ErrUniqueConstraint(field, value string) error {
	return errs.New(
		ErrCodeUniqueConstraint,
		fmt.Sprintf("unique constraint violation: field '%s' already has value '%s'", field, value),
		http.StatusConflict,
	)
}

// =============================================================================
// Revision Errors
// =============================================================================

// ErrRevisionNotFound returns a not found error for a revision
func ErrRevisionNotFound(entryID string, version int) error {
	return errs.New(
		ErrCodeRevisionNotFound,
		fmt.Sprintf("revision v%d not found for entry %s", version, entryID),
		http.StatusNotFound,
	)
}

// ErrRollbackFailed returns an error when rollback fails
func ErrRollbackFailed(reason string, cause error) error {
	return errs.Wrap(cause, ErrCodeRollbackFailed, "rollback failed: "+reason, http.StatusInternalServerError)
}

// ErrRevisionsNotEnabled returns an error when revisions are not enabled
func ErrRevisionsNotEnabled() error {
	return errs.New(
		ErrCodeRollbackFailed,
		"revisions are not enabled for this content type",
		http.StatusBadRequest,
	)
}

// =============================================================================
// Relation Errors
// =============================================================================

// ErrRelationNotFound returns a not found error for a relation
func ErrRelationNotFound(identifier string) error {
	return errs.New(ErrCodeRelationNotFound, "relation not found: "+identifier, http.StatusNotFound)
}

// ErrTypeRelationNotFound returns a not found error for a type relation
func ErrTypeRelationNotFound(identifier string) error {
	return errs.New(ErrCodeRelationNotFound, "type relation not found: "+identifier, http.StatusNotFound)
}

// ErrInvalidRelation returns an error for invalid relation configuration
func ErrInvalidRelation(reason string) error {
	return errs.New(ErrCodeInvalidRelation, "invalid relation: "+reason, http.StatusBadRequest)
}

// ErrCircularRelation returns an error when a circular relation is detected
func ErrCircularRelation(path string) error {
	return errs.New(
		ErrCodeCircularRelation,
		"circular relation detected: "+path,
		http.StatusBadRequest,
	)
}

// ErrRelationConstraint returns an error when a relation constraint is violated
func ErrRelationConstraint(reason string) error {
	return errs.New(
		ErrCodeRelationConstraint,
		"relation constraint violation: "+reason,
		http.StatusConflict,
	)
}

// =============================================================================
// Query Errors
// =============================================================================

// ErrInvalidQuery returns an error for invalid query syntax
func ErrInvalidQuery(reason string) error {
	return errs.New(ErrCodeInvalidQuery, "invalid query: "+reason, http.StatusBadRequest)
}

// ErrInvalidFilter returns an error for invalid filter expression
func ErrInvalidFilter(filter, reason string) error {
	return errs.New(
		ErrCodeInvalidFilter,
		fmt.Sprintf("invalid filter '%s': %s", filter, reason),
		http.StatusBadRequest,
	)
}

// ErrInvalidSort returns an error for invalid sort expression
func ErrInvalidSort(sort, reason string) error {
	return errs.New(
		ErrCodeInvalidSort,
		fmt.Sprintf("invalid sort '%s': %s", sort, reason),
		http.StatusBadRequest,
	)
}

// ErrInvalidOperator returns an error for invalid query operator
func ErrInvalidOperator(operator string) error {
	return errs.New(
		ErrCodeInvalidOperator,
		"invalid operator: "+operator,
		http.StatusBadRequest,
	)
}

// =============================================================================
// General Errors
// =============================================================================

// ErrAccessDenied returns a forbidden error when access is denied
func ErrAccessDenied(reason string) error {
	return errs.New(ErrCodeAccessDenied, "access denied: "+reason, http.StatusForbidden)
}

// ErrInvalidRequest returns a bad request error for generic invalid requests
func ErrInvalidRequest(reason string) error {
	return errs.New(ErrCodeInvalidRequest, "invalid request: "+reason, http.StatusBadRequest)
}

// ErrAppContextMissing returns a bad request error when app context is missing
func ErrAppContextMissing() error {
	return errs.New(
		ErrCodeAppContextMissing,
		"app context is required for this operation",
		http.StatusBadRequest,
	)
}

// ErrEnvContextMissing returns a bad request error when environment context is missing
func ErrEnvContextMissing() error {
	return errs.New(
		ErrCodeEnvContextMissing,
		"environment context is required for this operation",
		http.StatusBadRequest,
	)
}

// InternalError returns an internal server error with the given message
func InternalError(message string, cause error) error {
	return errs.Wrap(cause, "CMS_INTERNAL_ERROR", message, http.StatusInternalServerError)
}

// ErrInternalError is an alias for InternalError for consistency
func ErrInternalError(message string, cause error) error {
	return InternalError(message, cause)
}

// ErrDatabaseError returns an internal server error for database issues
func ErrDatabaseError(message string, cause error) error {
	return errs.Wrap(cause, "CMS_DATABASE_ERROR", message, http.StatusInternalServerError)
}

