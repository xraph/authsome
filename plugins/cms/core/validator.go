package core

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// FieldValidator validates a field value against its type and options
type FieldValidator struct {
	FieldName    string
	FieldType    FieldType
	Required     bool
	Unique       bool
	Options      *FieldOptionsDTO
}

// ValidationError represents a validation error for a field
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []ValidationError  `json:"errors,omitempty"`
}

// NewFieldValidator creates a new field validator
func NewFieldValidator(name string, fieldType FieldType, required, unique bool, options *FieldOptionsDTO) *FieldValidator {
	return &FieldValidator{
		FieldName: name,
		FieldType: fieldType,
		Required:  required,
		Unique:    unique,
		Options:   options,
	}
}

// Validate validates a value against the field's type and options
func (v *FieldValidator) Validate(value interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check required
	if v.Required && isEmptyValue(value) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   v.FieldName,
			Message: "this field is required",
			Code:    "required",
		})
		return result
	}

	// Skip further validation if value is empty (and not required)
	if isEmptyValue(value) {
		return result
	}

	// Type-specific validation
	var err error
	switch v.FieldType {
	case FieldTypeText, FieldTypeTextarea:
		err = v.validateText(value)
	case FieldTypeRichText, FieldTypeMarkdown:
		err = v.validateRichText(value)
	case FieldTypeNumber, FieldTypeFloat, FieldTypeDecimal:
		err = v.validateNumber(value)
	case FieldTypeInteger, FieldTypeBigInteger:
		err = v.validateInteger(value)
	case FieldTypeBoolean:
		err = v.validateBoolean(value)
	case FieldTypeDate:
		err = v.validateDate(value)
	case FieldTypeDateTime:
		err = v.validateDateTime(value)
	case FieldTypeTime:
		err = v.validateTime(value)
	case FieldTypeEmail:
		err = v.validateEmail(value)
	case FieldTypeURL:
		err = v.validateURL(value)
	case FieldTypePhone:
		err = v.validatePhone(value)
	case FieldTypeSlug:
		err = v.validateSlug(value)
	case FieldTypeUUID:
		err = v.validateUUID(value)
	case FieldTypeColor:
		err = v.validateColor(value)
	case FieldTypeJSON:
		err = v.validateJSON(value)
	case FieldTypeSelect, FieldTypeEnumeration:
		err = v.validateSelect(value)
	case FieldTypeMultiSelect:
		err = v.validateMultiSelect(value)
	case FieldTypeRelation:
		err = v.validateRelation(value)
	case FieldTypeMedia:
		err = v.validateMedia(value)
	case FieldTypeObject:
		err = v.validateObject(value)
	case FieldTypeArray:
		err = v.validateArray(value)
	}

	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   v.FieldName,
			Message: err.Error(),
			Code:    "invalid",
		})
	}

	return result
}

// =============================================================================
// Type-specific validators
// =============================================================================

func (v *FieldValidator) validateText(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if v.Options != nil {
		if v.Options.MinLength > 0 && len(str) < v.Options.MinLength {
			return fmt.Errorf("must be at least %d characters", v.Options.MinLength)
		}
		if v.Options.MaxLength > 0 && len(str) > v.Options.MaxLength {
			return fmt.Errorf("must be at most %d characters", v.Options.MaxLength)
		}
		if v.Options.Pattern != "" {
			matched, err := regexp.MatchString(v.Options.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern: %v", err)
			}
			if !matched {
				return fmt.Errorf("does not match required pattern")
			}
		}
	}

	return nil
}

func (v *FieldValidator) validateRichText(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if v.Options != nil {
		if v.Options.MaxWords > 0 {
			words := strings.Fields(stripHTML(str))
			if len(words) > v.Options.MaxWords {
				return fmt.Errorf("must be at most %d words", v.Options.MaxWords)
			}
		}
	}

	return nil
}

func (v *FieldValidator) validateNumber(value interface{}) error {
	num, ok := toFloat64(value)
	if !ok {
		return fmt.Errorf("must be a number")
	}

	if v.Options != nil {
		if v.Options.Min != nil && num < *v.Options.Min {
			return fmt.Errorf("must be at least %v", *v.Options.Min)
		}
		if v.Options.Max != nil && num > *v.Options.Max {
			return fmt.Errorf("must be at most %v", *v.Options.Max)
		}
		if v.Options.Step != nil && *v.Options.Step > 0 {
			// Check if value is a multiple of step
			remainder := num - (*v.Options.Min)
			if v.Options.Min == nil {
				remainder = num
			}
			if remainder != 0 && int(remainder*1000000)%int(*v.Options.Step*1000000) != 0 {
				return fmt.Errorf("must be a multiple of %v", *v.Options.Step)
			}
		}
	}

	return nil
}

func (v *FieldValidator) validateInteger(value interface{}) error {
	// First validate as number
	if err := v.validateNumber(value); err != nil {
		return err
	}

	// Then check it's an integer
	num, _ := toFloat64(value)
	if num != float64(int64(num)) {
		return fmt.Errorf("must be a whole number")
	}

	return nil
}

func (v *FieldValidator) validateBoolean(value interface{}) error {
	switch value.(type) {
	case bool:
		return nil
	case string:
		str := strings.ToLower(value.(string))
		if str == "true" || str == "false" || str == "1" || str == "0" {
			return nil
		}
	case int, int64, float64:
		return nil // 0 or 1 are valid
	}
	return fmt.Errorf("must be a boolean")
}

func (v *FieldValidator) validateDate(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a date string")
	}

	// Try parsing date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"02-01-2006",
	}

	var parsedDate time.Time
	var parseErr error
	for _, format := range formats {
		parsedDate, parseErr = time.Parse(format, str)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return fmt.Errorf("invalid date format")
	}

	if v.Options != nil {
		if v.Options.MinDate != nil && parsedDate.Before(*v.Options.MinDate) {
			return fmt.Errorf("must be on or after %s", v.Options.MinDate.Format("2006-01-02"))
		}
		if v.Options.MaxDate != nil && parsedDate.After(*v.Options.MaxDate) {
			return fmt.Errorf("must be on or before %s", v.Options.MaxDate.Format("2006-01-02"))
		}
	}

	return nil
}

func (v *FieldValidator) validateDateTime(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a datetime string")
	}

	// Try parsing datetime formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	var parsedTime time.Time
	var parseErr error
	for _, format := range formats {
		parsedTime, parseErr = time.Parse(format, str)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return fmt.Errorf("invalid datetime format")
	}

	if v.Options != nil {
		if v.Options.MinDate != nil && parsedTime.Before(*v.Options.MinDate) {
			return fmt.Errorf("must be on or after %s", v.Options.MinDate.Format(time.RFC3339))
		}
		if v.Options.MaxDate != nil && parsedTime.After(*v.Options.MaxDate) {
			return fmt.Errorf("must be on or before %s", v.Options.MaxDate.Format(time.RFC3339))
		}
	}

	return nil
}

func (v *FieldValidator) validateTime(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a time string")
	}

	// Try parsing time formats
	formats := []string{
		"15:04:05",
		"15:04",
		"3:04 PM",
		"3:04PM",
	}

	var parseErr error
	for _, format := range formats {
		_, parseErr = time.Parse(format, str)
		if parseErr == nil {
			return nil
		}
	}

	return fmt.Errorf("invalid time format")
}

func (v *FieldValidator) validateEmail(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	_, err := mail.ParseAddress(str)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}

	return nil
}

func (v *FieldValidator) validateURL(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	u, err := url.Parse(str)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https")
	}

	return nil
}

func (v *FieldValidator) validatePhone(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Basic phone number validation (allows +, digits, spaces, dashes, parentheses)
	phonePattern := regexp.MustCompile(`^[\+]?[(]?[0-9]{1,4}[)]?[-\s\./0-9]*$`)
	if !phonePattern.MatchString(str) {
		return fmt.Errorf("invalid phone number format")
	}

	// Check minimum digits
	digits := regexp.MustCompile(`\d`).FindAllString(str, -1)
	if len(digits) < 7 {
		return fmt.Errorf("phone number must have at least 7 digits")
	}

	return nil
}

func (v *FieldValidator) validateSlug(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	slugPattern := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	if !slugPattern.MatchString(str) {
		return fmt.Errorf("must be lowercase with hyphens only (e.g., my-slug)")
	}

	return nil
}

func (v *FieldValidator) validateUUID(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidPattern.MatchString(str) {
		return fmt.Errorf("invalid UUID format")
	}

	return nil
}

func (v *FieldValidator) validateColor(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Allow hex colors (#RGB, #RRGGBB, #RGBA, #RRGGBBAA)
	hexPattern := regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	// Allow rgb/rgba
	rgbPattern := regexp.MustCompile(`^rgba?\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*(,\s*(0|1|0?\.\d+))?\s*\)$`)
	// Allow hsl/hsla
	hslPattern := regexp.MustCompile(`^hsla?\(\s*\d{1,3}\s*,\s*\d{1,3}%\s*,\s*\d{1,3}%\s*(,\s*(0|1|0?\.\d+))?\s*\)$`)

	if !hexPattern.MatchString(str) && !rgbPattern.MatchString(str) && !hslPattern.MatchString(str) {
		return fmt.Errorf("invalid color format (use hex, rgb, or hsl)")
	}

	return nil
}

func (v *FieldValidator) validateJSON(value interface{}) error {
	// If it's already a map or slice, it's valid JSON structure
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		// TODO: Validate against JSON schema if provided in options
		return nil
	case string:
		var js interface{}
		if err := json.Unmarshal([]byte(value.(string)), &js); err != nil {
			return fmt.Errorf("invalid JSON")
		}
		return nil
	default:
		return fmt.Errorf("must be a valid JSON object or array")
	}
}

func (v *FieldValidator) validateSelect(value interface{}) error {
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if v.Options == nil || len(v.Options.Choices) == 0 {
		return nil // No choices defined, accept any value
	}

	for _, choice := range v.Options.Choices {
		if choice.Value == str {
			if choice.Disabled {
				return fmt.Errorf("this option is not available")
			}
			return nil
		}
	}

	return fmt.Errorf("invalid selection")
}

func (v *FieldValidator) validateMultiSelect(value interface{}) error {
	// Value should be a slice/array
	var values []string

	switch val := value.(type) {
	case []interface{}:
		for _, item := range val {
			str, ok := toString(item)
			if !ok {
				return fmt.Errorf("each selection must be a string")
			}
			values = append(values, str)
		}
	case []string:
		values = val
	default:
		return fmt.Errorf("must be an array of values")
	}

	if v.Options == nil || len(v.Options.Choices) == 0 {
		return nil
	}

	// Validate each selection
	validValues := make(map[string]bool)
	for _, choice := range v.Options.Choices {
		if !choice.Disabled {
			validValues[choice.Value] = true
		}
	}

	for _, val := range values {
		if !validValues[val] {
			return fmt.Errorf("invalid selection: %s", val)
		}
	}

	return nil
}

func (v *FieldValidator) validateRelation(value interface{}) error {
	// Relation value should be an ID or array of IDs
	switch val := value.(type) {
	case string:
		if val == "" {
			return nil // Empty is okay for optional relations
		}
		// Single relation - just needs to be a non-empty string ID
		return nil
	case []interface{}:
		// Multiple relations
		for _, item := range val {
			str, ok := toString(item)
			if !ok || str == "" {
				return fmt.Errorf("each related item must be a valid ID")
			}
		}
		return nil
	case []string:
		return nil
	default:
		return fmt.Errorf("must be an ID or array of IDs")
	}
}

func (v *FieldValidator) validateMedia(value interface{}) error {
	// Media value should be a file reference (URL or ID)
	str, ok := toString(value)
	if !ok {
		return fmt.Errorf("must be a file reference")
	}

	// TODO: Validate mime type if provided in options
	_ = str

	return nil
}

func (v *FieldValidator) validateObject(value interface{}) error {
	// Object value should be a map
	obj, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("must be an object")
	}

	// Validate nested fields if defined
	if v.Options != nil && len(v.Options.NestedFields) > 0 {
		return v.validateNestedFields(obj, v.Options.NestedFields)
	}

	return nil
}

func (v *FieldValidator) validateArray(value interface{}) error {
	// Array value should be a slice
	arr, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("must be an array")
	}

	// Validate array constraints
	if v.Options != nil {
		if v.Options.MinItems != nil && len(arr) < *v.Options.MinItems {
			return fmt.Errorf("must have at least %d items", *v.Options.MinItems)
		}
		if v.Options.MaxItems != nil && len(arr) > *v.Options.MaxItems {
			return fmt.Errorf("must have at most %d items", *v.Options.MaxItems)
		}

		// Validate each item if nested fields are defined
		if len(v.Options.NestedFields) > 0 {
			for i, item := range arr {
				obj, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d must be an object", i+1)
				}
				if err := v.validateNestedFields(obj, v.Options.NestedFields); err != nil {
					return fmt.Errorf("item %d: %w", i+1, err)
				}
			}
		}
	}

	return nil
}

// validateNestedFields validates nested field values against their definitions
func (v *FieldValidator) validateNestedFields(data map[string]interface{}, fields []NestedFieldDefDTO) error {
	for _, field := range fields {
		value, exists := data[field.Slug]

		// Check required
		if field.Required && (!exists || isEmptyValue(value)) {
			return fmt.Errorf("field '%s' is required", field.Name)
		}

		// Skip validation if empty and not required
		if !exists || isEmptyValue(value) {
			continue
		}

		// Create validator for nested field
		var options *FieldOptionsDTO
		if field.Options != nil {
			options = field.Options
		}

		nestedValidator := NewFieldValidator(
			field.Slug,
			FieldType(field.Type),
			field.Required,
			false, // Unique not supported in nested
			options,
		)

		result := nestedValidator.Validate(value)
		if !result.Valid && len(result.Errors) > 0 {
			return fmt.Errorf("field '%s': %s", field.Name, result.Errors[0].Message)
		}
	}

	return nil
}

// ValidateNestedData validates data against nested field definitions (exported for use by services)
func ValidateNestedData(data map[string]interface{}, fields []NestedFieldDefDTO) *ValidationResult {
	result := &ValidationResult{Valid: true}

	validator := &FieldValidator{}
	if err := validator.validateNestedFields(data, fields); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "data",
			Message: err.Error(),
			Code:    "invalid",
		})
	}

	return result
}

// ValidateArrayData validates an array of data against nested field definitions
func ValidateArrayData(items []map[string]interface{}, fields []NestedFieldDefDTO, minItems, maxItems *int) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate array constraints
	if minItems != nil && len(items) < *minItems {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "items",
			Message: fmt.Sprintf("must have at least %d items", *minItems),
			Code:    "min_items",
		})
		return result
	}
	if maxItems != nil && len(items) > *maxItems {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "items",
			Message: fmt.Sprintf("must have at most %d items", *maxItems),
			Code:    "max_items",
		})
		return result
	}

	// Validate each item
	validator := &FieldValidator{}
	for i, item := range items {
		if err := validator.validateNestedFields(item, fields); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("items[%d]", i),
				Message: err.Error(),
				Code:    "invalid",
			})
		}
	}

	return result
}

// =============================================================================
// Helper functions
// =============================================================================

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float64:
		return fmt.Sprintf("%v", v), true
	case int:
		return fmt.Sprintf("%d", v), true
	case int64:
		return fmt.Sprintf("%d", v), true
	case bool:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

func stripHTML(s string) string {
	// Simple HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

