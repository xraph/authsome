package schema

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	FieldID     string `json:"fieldId"`
	ValidatorID string `json:"validatorId"`
	Message     string `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.FieldID, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(fieldID, validatorID, format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		FieldID:     fieldID,
		ValidatorID: validatorID,
		Message:     fmt.Sprintf(format, args...),
	}
}

// ValidationResult holds the result of validating a section or schema
type ValidationResult struct {
	Valid        bool                          `json:"valid"`
	FieldErrors  map[string][]*ValidationError `json:"fieldErrors,omitempty"`
	GlobalErrors []string                      `json:"globalErrors,omitempty"`
}

// NewValidationResult creates a new valid validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:       true,
		FieldErrors: make(map[string][]*ValidationError),
	}
}

// AddFieldError adds a field-level error
func (r *ValidationResult) AddFieldError(fieldID, validatorID, message string) {
	r.Valid = false
	r.FieldErrors[fieldID] = append(r.FieldErrors[fieldID], &ValidationError{
		FieldID:     fieldID,
		ValidatorID: validatorID,
		Message:     message,
	})
}

// AddGlobalError adds a schema-level error
func (r *ValidationResult) AddGlobalError(message string) {
	r.Valid = false
	r.GlobalErrors = append(r.GlobalErrors, message)
}

// HasErrors returns true if there are any errors
func (r *ValidationResult) HasErrors() bool {
	return !r.Valid
}

// HasFieldError checks if a specific field has errors
func (r *ValidationResult) HasFieldError(fieldID string) bool {
	errors, ok := r.FieldErrors[fieldID]
	return ok && len(errors) > 0
}

// GetFieldErrors returns all errors for a specific field
func (r *ValidationResult) GetFieldErrors(fieldID string) []*ValidationError {
	return r.FieldErrors[fieldID]
}

// GetFirstFieldError returns the first error for a specific field
func (r *ValidationResult) GetFirstFieldError(fieldID string) *ValidationError {
	errors := r.FieldErrors[fieldID]
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

// AllErrors returns all errors as a flat slice
func (r *ValidationResult) AllErrors() []*ValidationError {
	var all []*ValidationError
	for _, errors := range r.FieldErrors {
		all = append(all, errors...)
	}
	return all
}

// ErrorMessages returns all error messages as strings
func (r *ValidationResult) ErrorMessages() []string {
	var messages []string
	for _, errors := range r.FieldErrors {
		for _, err := range errors {
			messages = append(messages, err.Error())
		}
	}
	messages = append(messages, r.GlobalErrors...)
	return messages
}

// Error returns a combined error string
func (r *ValidationResult) Error() string {
	if r.Valid {
		return ""
	}
	messages := r.ErrorMessages()
	return strings.Join(messages, "; ")
}

// Merge combines another validation result into this one
func (r *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}
	if !other.Valid {
		r.Valid = false
	}
	for fieldID, errors := range other.FieldErrors {
		r.FieldErrors[fieldID] = append(r.FieldErrors[fieldID], errors...)
	}
	r.GlobalErrors = append(r.GlobalErrors, other.GlobalErrors...)
}

// ToMap converts the validation result to a map for JSON responses
func (r *ValidationResult) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"valid": r.Valid,
	}
	if len(r.FieldErrors) > 0 {
		fieldErrors := make(map[string][]string)
		for fieldID, errors := range r.FieldErrors {
			for _, err := range errors {
				fieldErrors[fieldID] = append(fieldErrors[fieldID], err.Message)
			}
		}
		result["fieldErrors"] = fieldErrors
	}
	if len(r.GlobalErrors) > 0 {
		result["globalErrors"] = r.GlobalErrors
	}
	return result
}

// SchemaError represents an error in schema definition
type SchemaError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *SchemaError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Common schema errors
var (
	ErrSectionNotFound    = &SchemaError{Type: "section_not_found", Message: "Section not found"}
	ErrFieldNotFound      = &SchemaError{Type: "field_not_found", Message: "Field not found"}
	ErrInvalidFieldType   = &SchemaError{Type: "invalid_field_type", Message: "Invalid field type"}
	ErrDuplicateSectionID = &SchemaError{Type: "duplicate_section", Message: "Section ID already exists"}
	ErrDuplicateFieldID   = &SchemaError{Type: "duplicate_field", Message: "Field ID already exists in section"}
	ErrSchemaNotFound     = &SchemaError{Type: "schema_not_found", Message: "Schema not found"}
)

// NewSchemaError creates a new schema error with optional details
func NewSchemaError(errType, message, details string) *SchemaError {
	return &SchemaError{
		Type:    errType,
		Message: message,
		Details: details,
	}
}
