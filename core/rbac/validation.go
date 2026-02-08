package rbac

import (
	"fmt"

	"github.com/rs/xid"
)

// validateUserID validates that a user ID is not nil
func validateUserID(userID xid.ID) error {
	if userID.IsNil() {
		return fmt.Errorf("user_id is required but was nil")
	}
	return nil
}

// validateRoleID validates that a role ID is not nil
func validateRoleID(roleID xid.ID) error {
	if roleID.IsNil() {
		return fmt.Errorf("role_id is required but was nil")
	}
	return nil
}

// validateOrgID validates that an organization ID is not nil
func validateOrgID(orgID xid.ID) error {
	if orgID.IsNil() {
		return fmt.Errorf("organization_id is required but was nil")
	}
	return nil
}

// validateAppID validates that an app ID is not nil
func validateAppID(appID xid.ID) error {
	if appID.IsNil() {
		return fmt.Errorf("app_id is required but was nil")
	}
	return nil
}

// validateEnvID validates that an environment ID is not nil
func validateEnvID(envID xid.ID) error {
	if envID.IsNil() {
		return fmt.Errorf("environment_id is required but was nil")
	}
	return nil
}

// validateRoleIDs validates that a slice of role IDs is not empty and contains no nil IDs
func validateRoleIDs(roleIDs []xid.ID) error {
	if len(roleIDs) == 0 {
		return fmt.Errorf("at least one role_id is required")
	}
	for i, roleID := range roleIDs {
		if roleID.IsNil() {
			return fmt.Errorf("role_id at index %d is nil", i)
		}
	}
	return nil
}

// validateUserIDs validates that a slice of user IDs is not empty and contains no nil IDs
func validateUserIDs(userIDs []xid.ID) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("at least one user_id is required")
	}
	for i, userID := range userIDs {
		if userID.IsNil() {
			return fmt.Errorf("user_id at index %d is nil", i)
		}
	}
	return nil
}
