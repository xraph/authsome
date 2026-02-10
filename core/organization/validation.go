package organization

import (
	"fmt"
	"time"
)

// validateRole validates that a role is valid.
func validateRole(role string) error {
	if !IsValidRole(role) {
		return InvalidRole(role)
	}

	return nil
}

// validateStatus validates that a status is valid.
func validateStatus(status string) error {
	if !IsValidStatus(status) {
		return InvalidStatus(status)
	}

	return nil
}

// validateInvitationStatus validates that an invitation status is valid.
func validateInvitationStatus(status string) error {
	if !IsValidInvitationStatus(status) {
		return fmt.Errorf("invalid invitation status: %s", status)
	}

	return nil
}

// validateInvitationExpiry checks if an invitation has expired.
func validateInvitationExpiry(expiresAt time.Time) error {
	if time.Now().After(expiresAt) {
		return InvitationExpired()
	}

	return nil
}

// validateOwnerProtection prevents operations on owner role that shouldn't be allowed.
func validateOwnerProtection(role string) error {
	if role == RoleOwner {
		return CannotRemoveOwner()
	}

	return nil
}

// validateInvitationPending ensures invitation is in pending status.
func validateInvitationPending(status string) error {
	if status != InvitationStatusPending {
		return InvitationNotPending()
	}

	return nil
}

// isAlphanumeric checks if a string contains only alphanumeric characters.
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}

	return true
}
