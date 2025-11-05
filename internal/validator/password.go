package validator

import (
	"unicode"
)

// PasswordRequirements defines password requirements
type PasswordRequirements struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
}

// DefaultPasswordRequirements returns default password requirements
func DefaultPasswordRequirements() PasswordRequirements {
	return PasswordRequirements{
		MinLength:      8,
		RequireUpper:   false,
		RequireLower:   false,
		RequireNumber:  false,
		RequireSpecial: false,
	}
}

// ValidatePassword validates a password against requirements
func ValidatePassword(password string, reqs PasswordRequirements) (bool, string) {
	if len(password) < reqs.MinLength {
		return false, "password too short"
	}

	if reqs.RequireUpper && !hasUpper(password) {
		return false, "password must contain uppercase letter"
	}

	if reqs.RequireLower && !hasLower(password) {
		return false, "password must contain lowercase letter"
	}

	if reqs.RequireNumber && !hasNumber(password) {
		return false, "password must contain number"
	}

	if reqs.RequireSpecial && !hasSpecial(password) {
		return false, "password must contain special character"
	}

	return true, ""
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func hasLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func hasNumber(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasSpecial(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
