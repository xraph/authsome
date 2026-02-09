package rbac

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// validateUserID validates that a user ID is not nil.
func validateUserID(userID xid.ID) error {
	if userID.IsNil() {
		return errs.RequiredField("user_id")
	}

	return nil
}

// validateRoleID validates that a role ID is not nil.
func validateRoleID(roleID xid.ID) error {
	if roleID.IsNil() {
		return errs.RequiredField("role_id")
	}

	return nil
}

// validateOrgID validates that an organization ID is not nil.
func validateOrgID(orgID xid.ID) error {
	if orgID.IsNil() {
		return errs.RequiredField("organization_id")
	}

	return nil
}

// validateAppID validates that an app ID is not nil.
func validateAppID(appID xid.ID) error {
	if appID.IsNil() {
		return errs.RequiredField("app_id")
	}

	return nil
}

// validateEnvID validates that an environment ID is not nil.
func validateEnvID(envID xid.ID) error {
	if envID.IsNil() {
		return errs.RequiredField("environment_id")
	}

	return nil
}

// validateRoleIDs validates that a slice of role IDs is not empty and contains no nil IDs.
func validateRoleIDs(roleIDs []xid.ID) error {
	if len(roleIDs) == 0 {
		return errs.RequiredField("role_ids")
	}

	for i, roleID := range roleIDs {
		if roleID.IsNil() {
			return errs.InvalidInput("role_id", fmt.Sprintf("nil at index %d", i))
		}
	}

	return nil
}

// validateUserIDs validates that a slice of user IDs is not empty and contains no nil IDs.
func validateUserIDs(userIDs []xid.ID) error {
	if len(userIDs) == 0 {
		return errs.RequiredField("user_ids")
	}

	for i, userID := range userIDs {
		if userID.IsNil() {
			return errs.InvalidInput("user_id", fmt.Sprintf("nil at index %d", i))
		}
	}

	return nil
}
