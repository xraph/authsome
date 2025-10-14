package validator

import (
    "regexp"
    "strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
    email = strings.TrimSpace(strings.ToLower(email))
    return emailRegex.MatchString(email)
}