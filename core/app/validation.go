package app

import (
	"time"

	"github.com/xraph/authsome/schema"
)

// validateOwnerProtection ensures that the owner cannot be removed or demoted
// Package-level function usable by all services
func validateOwnerProtection(member *schema.Member) error {
	if member.Role == schema.MemberRoleOwner {
		return CannotRemoveOwner()
	}
	return nil
}

// validateInvitationExpiry checks if an invitation has expired
// Package-level function usable by all services
func validateInvitationExpiry(inv *schema.Invitation) error {
	if time.Now().After(inv.ExpiresAt) {
		return InvitationExpired()
	}
	return nil
}

// validateInvitationStatus checks if an invitation is in the expected status
// Package-level function usable by all services
func validateInvitationStatus(inv *schema.Invitation, expectedStatus schema.InvitationStatus) error {
	if inv.Status != expectedStatus {
		return InvitationInvalidStatus(string(expectedStatus), string(inv.Status))
	}
	return nil
}

// validateRole checks if a role is valid
func validateRole(role schema.MemberRole) error {
	if !role.IsValid() {
		return InvalidRole(string(role))
	}
	return nil
}

// validateStatus checks if a status is valid
func validateStatus(status schema.MemberStatus) error {
	if !status.IsValid() {
		return InvalidStatus(string(status))
	}
	return nil
}
