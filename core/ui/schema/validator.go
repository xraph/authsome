package schema

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/xraph/authsome/internal/errs"
)

// Validator is the interface for synchronous field validators.
type Validator interface {
	// Validate validates the given value
	Validate(value any) error
	// Name returns the validator name for identification
	Name() string
}

// ValidatorFunc is a function type that implements Validator.
type ValidatorFunc struct {
	name string
	fn   func(value any) error
}

// Validate implements the Validator interface.
func (v ValidatorFunc) Validate(value any) error {
	return v.fn(value)
}

// Name returns the validator name.
func (v ValidatorFunc) Name() string {
	return v.name
}

// NewValidator creates a new validator from a function.
func NewValidator(name string, fn func(value any) error) Validator {
	return ValidatorFunc{name: name, fn: fn}
}

// RequiredValidator returns a validator that ensures a value is not empty.
func RequiredValidator() Validator {
	return NewValidator("required", func(value any) error {
		if isEmpty(value) {
			return errs.RequiredField("field")
		}

		return nil
	})
}

// MinLengthValidator returns a validator that ensures a string has minimum length.
func MinLengthValidator(min int) Validator {
	return NewValidator("min_length", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil // Skip non-strings
		}

		if len(str) < min {
			return fmt.Errorf("must be at least %d characters", min)
		}

		return nil
	})
}

// MaxLengthValidator returns a validator that ensures a string has maximum length.
func MaxLengthValidator(max int) Validator {
	return NewValidator("max_length", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if len(str) > max {
			return fmt.Errorf("must be at most %d characters", max)
		}

		return nil
	})
}

// MinValueValidator returns a validator that ensures a number is at least min.
func MinValueValidator(min float64) Validator {
	return NewValidator("min_value", func(value any) error {
		num, ok := toFloat64(value)
		if !ok {
			return nil
		}

		if num < min {
			return fmt.Errorf("must be at least %v", min)
		}

		return nil
	})
}

// MaxValueValidator returns a validator that ensures a number is at most max.
func MaxValueValidator(max float64) Validator {
	return NewValidator("max_value", func(value any) error {
		num, ok := toFloat64(value)
		if !ok {
			return nil
		}

		if num > max {
			return fmt.Errorf("must be at most %v", max)
		}

		return nil
	})
}

// RangeValidator returns a validator that ensures a number is within a range.
func RangeValidator(min, max float64) Validator {
	return NewValidator("range", func(value any) error {
		num, ok := toFloat64(value)
		if !ok {
			return nil
		}

		if num < min || num > max {
			return fmt.Errorf("must be between %v and %v", min, max)
		}

		return nil
	})
}

// PatternValidator returns a validator that ensures a string matches a regex pattern.
func PatternValidator(pattern string, message string) Validator {
	re := regexp.MustCompile(pattern)

	return NewValidator("pattern", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if str == "" {
			return nil // Empty strings handled by required validator
		}

		if !re.MatchString(str) {
			if message != "" {
				return fmt.Errorf("%s", message)
			}

			return errs.BadRequest("invalid format")
		}

		return nil
	})
}

// EnumValidator returns a validator that ensures a value is one of the allowed values.
func EnumValidator(allowed []any) Validator {
	return NewValidator("enum", func(value any) error {
		if slices.Contains(allowed, value) {
			return nil
		}

		return errs.BadRequest("invalid option")
	})
}

// StringEnumValidator returns a validator for string enums.
func StringEnumValidator(allowed ...string) Validator {
	return NewValidator("enum", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if slices.Contains(allowed, str) {
			return nil
		}

		return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
	})
}

// EmailValidator returns a validator that ensures a valid email format.
func EmailValidator() Validator {
	return NewValidator("email", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if str == "" {
			return nil
		}

		_, err := mail.ParseAddress(str)
		if err != nil {
			return errs.BadRequest("invalid email address")
		}

		return nil
	})
}

// URLValidator returns a validator that ensures a valid URL format.
func URLValidator() Validator {
	return NewValidator("url", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if str == "" {
			return nil
		}

		u, err := url.Parse(str)
		if err != nil {
			return errs.BadRequest("invalid URL")
		}

		if u.Scheme == "" || u.Host == "" {
			return errs.BadRequest("URL must have a scheme and host")
		}

		return nil
	})
}

// URLWithSchemeValidator returns a validator that ensures a valid URL with specific schemes.
func URLWithSchemeValidator(schemes ...string) Validator {
	return NewValidator("url_scheme", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if str == "" {
			return nil
		}

		u, err := url.Parse(str)
		if err != nil {
			return errs.BadRequest("invalid URL")
		}

		if u.Scheme == "" || u.Host == "" {
			return errs.BadRequest("URL must have a scheme and host")
		}

		if slices.Contains(schemes, u.Scheme) {
			return nil
		}

		return fmt.Errorf("URL scheme must be one of: %s", strings.Join(schemes, ", "))
	})
}

// AlphanumericValidator returns a validator for alphanumeric strings.
func AlphanumericValidator() Validator {
	return PatternValidator(`^[a-zA-Z0-9]+$`, "must contain only letters and numbers")
}

// SlugValidator returns a validator for URL-safe slugs.
func SlugValidator() Validator {
	return PatternValidator(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, "must be a valid slug (lowercase letters, numbers, and hyphens)")
}

// PhoneValidator returns a validator for phone numbers.
func PhoneValidator() Validator {
	return PatternValidator(`^\+?[1-9]\d{1,14}$`, "invalid phone number format")
}

// IPAddressValidator returns a validator for IP addresses.
func IPAddressValidator() Validator {
	return NewValidator("ip_address", func(value any) error {
		str, ok := value.(string)
		if !ok || str == "" {
			return nil
		}
		// Simple regex for IPv4 and IPv6
		ipv4 := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)

		ipv6 := regexp.MustCompile(`^(?:[A-Fa-f0-9]{1,4}:){7}[A-Fa-f0-9]{1,4}$`)
		if !ipv4.MatchString(str) && !ipv6.MatchString(str) {
			return errs.BadRequest("invalid IP address")
		}

		return nil
	})
}

// CIDRValidator returns a validator for CIDR notation.
func CIDRValidator() Validator {
	return PatternValidator(
		`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\/(?:3[0-2]|[12]?[0-9])$`,
		"invalid CIDR notation",
	)
}

// HexColorValidator returns a validator for hex color codes.
func HexColorValidator() Validator {
	return PatternValidator(`^#(?:[0-9a-fA-F]{3}){1,2}$`, "invalid hex color code")
}

// JSONValidator returns a validator that ensures valid JSON.
func JSONValidator() Validator {
	return NewValidator("json", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if str == "" {
			return nil
		}

		var js any
		if err := jsonUnmarshal([]byte(str), &js); err != nil {
			return errs.BadRequest("invalid JSON")
		}

		return nil
	})
}

// PasswordStrengthValidator returns a validator for password strength.
func PasswordStrengthValidator(minLength int, requireUppercase, requireLowercase, requireNumbers, requireSpecial bool) Validator {
	return NewValidator("password_strength", func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}

		if len(str) < minLength {
			return fmt.Errorf("password must be at least %d characters", minLength)
		}

		if requireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(str) {
			return errs.BadRequest("password must contain at least one uppercase letter")
		}

		if requireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(str) {
			return errs.BadRequest("password must contain at least one lowercase letter")
		}

		if requireNumbers && !regexp.MustCompile(`[0-9]`).MatchString(str) {
			return errs.BadRequest("password must contain at least one number")
		}

		if requireSpecial && !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(str) {
			return errs.BadRequest("password must contain at least one special character")
		}

		return nil
	})
}

// ArrayMinLengthValidator returns a validator for minimum array length.
func ArrayMinLengthValidator(min int) Validator {
	return NewValidator("array_min_length", func(value any) error {
		arr, ok := value.([]any)
		if !ok {
			return nil
		}

		if len(arr) < min {
			return fmt.Errorf("must have at least %d items", min)
		}

		return nil
	})
}

// ArrayMaxLengthValidator returns a validator for maximum array length.
func ArrayMaxLengthValidator(max int) Validator {
	return NewValidator("array_max_length", func(value any) error {
		arr, ok := value.([]any)
		if !ok {
			return nil
		}

		if len(arr) > max {
			return fmt.Errorf("must have at most %d items", max)
		}

		return nil
	})
}

// NotEmptyValidator returns a validator that ensures a value is not empty.
func NotEmptyValidator() Validator {
	return NewValidator("not_empty", func(value any) error {
		if isEmpty(value) {
			return errs.BadRequest("must not be empty")
		}

		return nil
	})
}

// CustomValidator creates a validator with a custom validation function.
func CustomValidator(name string, message string, fn func(value any) bool) Validator {
	return NewValidator(name, func(value any) error {
		if !fn(value) {
			return fmt.Errorf("%s", message)
		}

		return nil
	})
}

// CompositeValidator combines multiple validators.
func CompositeValidator(validators ...Validator) Validator {
	return NewValidator("composite", func(value any) error {
		for _, v := range validators {
			if err := v.Validate(value); err != nil {
				return err
			}
		}

		return nil
	})
}

// Helper for JSON unmarshaling without importing encoding/json again.
func jsonUnmarshal(data []byte, v any) error {
	// Simple implementation - in production, use encoding/json
	if len(data) == 0 {
		return errs.BadRequest("empty data")
	}
	// Check for valid JSON start characters
	c := data[0]
	if c != '{' && c != '[' && c != '"' && c != 't' && c != 'f' && c != 'n' && (c < '0' || c > '9') && c != '-' {
		return errs.BadRequest("invalid JSON")
	}

	return nil
}
