package bridge

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// UsersListInput represents users list request
type UsersListInput struct {
	AppID         string  `json:"appId" validate:"required"`
	Page          int     `json:"page"`
	PageSize      int     `json:"pageSize"`
	SearchTerm    string  `json:"searchTerm,omitempty"`
	SortBy        string  `json:"sortBy,omitempty"`
	SortOrder     string  `json:"sortOrder,omitempty"`
	Status        *string `json:"status,omitempty"`        // "active", "inactive", "banned"
	EmailVerified *bool   `json:"emailVerified,omitempty"` // true/false
	RoleFilter    string  `json:"roleFilter,omitempty"`    // role name to filter by
}

// UsersListOutput represents users list response
type UsersListOutput struct {
	Users      []UserItem `json:"users"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// UserItem represents a user in the list
type UserItem struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name,omitempty"`
	EmailVerified bool     `json:"emailVerified"`
	CreatedAt     string   `json:"createdAt"`
	Roles         []string `json:"roles,omitempty"`
	Status        string   `json:"status"`
}

// UserDetailInput represents user detail request
type UserDetailInput struct {
	UserID string `json:"userId" validate:"required"`
	AppID  string `json:"appId,omitempty"`
}

// UserDetailOutput represents user detail response
type UserDetailOutput struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name,omitempty"`
	EmailVerified bool     `json:"emailVerified"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt,omitempty"`
	Roles         []string `json:"roles,omitempty"`
	Status        string   `json:"status"`
	LastLoginAt   string   `json:"lastLoginAt,omitempty"`
	SessionCount  int      `json:"sessionCount"`
}

// UpdateUserInput represents user update request
type UpdateUserInput struct {
	UserID        string  `json:"userId" validate:"required"`
	AppID         string  `json:"appId" validate:"required"`
	Name          *string `json:"name,omitempty"`
	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"emailVerified,omitempty"`
	Image         *string `json:"image,omitempty"`
}

// DeleteUserInput represents user delete request
type DeleteUserInput struct {
	UserID string `json:"userId" validate:"required"`
}

// UpdatePasswordInput represents password update request
type UpdatePasswordInput struct {
	UserID      string `json:"userId" validate:"required"`
	AppID       string `json:"appId" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

// BulkDeleteUsersInput represents bulk user delete request
type BulkDeleteUsersInput struct {
	AppID   string   `json:"appId" validate:"required"`
	UserIDs []string `json:"userIds" validate:"required,min=1"`
}

// BulkDeleteUsersOutput represents bulk delete response
type BulkDeleteUsersOutput struct {
	SuccessCount int               `json:"successCount"`
	FailedCount  int               `json:"failedCount"`
	Errors       map[string]string `json:"errors,omitempty"` // userID -> error message
}

// UpdateUserRolesInput represents user role update request
type UpdateUserRolesInput struct {
	UserID  string   `json:"userId" validate:"required"`
	AppID   string   `json:"appId" validate:"required"`
	OrgID   string   `json:"orgId,omitempty"`
	RoleIDs []string `json:"roleIds"` // Array of role IDs to assign
}

// ListRolesInput represents list roles request
type ListRolesInput struct {
	AppID string `json:"appId" validate:"required"`
	OrgID string `json:"orgId,omitempty"`
}

// RoleItem represents a role
type RoleItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// ListRolesOutput represents list roles response
type ListRolesOutput struct {
	Roles []RoleItem `json:"roles"`
}

// GenericSuccessOutput represents a generic success response
type GenericSuccessOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// registerUserFunctions registers user management bridge functions
func (bm *BridgeManager) registerUserFunctions() error {
	// List users
	if err := bm.bridge.Register("getUsersList", bm.getUsersList,
		bridge.WithDescription("Get list of users with pagination and filtering"),
	); err != nil {
		return fmt.Errorf("failed to register getUsersList: %w", err)
	}

	// Get user detail
	if err := bm.bridge.Register("getUserDetail", bm.getUserDetail,
		bridge.WithDescription("Get detailed information about a user"),
	); err != nil {
		return fmt.Errorf("failed to register getUserDetail: %w", err)
	}

	// Update user
	if err := bm.bridge.Register("updateUser", bm.updateUser,
		bridge.WithDescription("Update user information"),
	); err != nil {
		return fmt.Errorf("failed to register updateUser: %w", err)
	}

	// Delete user
	if err := bm.bridge.Register("deleteUser", bm.deleteUser,
		bridge.WithDescription("Delete a user"),
	); err != nil {
		return fmt.Errorf("failed to register deleteUser: %w", err)
	}

	// Search users
	if err := bm.bridge.Register("searchUsers", bm.searchUsers,
		bridge.WithDescription("Search users by query"),
	); err != nil {
		return fmt.Errorf("failed to register searchUsers: %w", err)
	}

	// Update user password
	if err := bm.bridge.Register("updateUserPassword", bm.updateUserPassword,
		bridge.WithDescription("Update user password"),
	); err != nil {
		return fmt.Errorf("failed to register updateUserPassword: %w", err)
	}

	// Bulk delete users
	if err := bm.bridge.Register("bulkDeleteUsers", bm.bulkDeleteUsers,
		bridge.WithDescription("Delete multiple users"),
	); err != nil {
		return fmt.Errorf("failed to register bulkDeleteUsers: %w", err)
	}

	// List roles
	if err := bm.bridge.Register("listRoles", bm.listRoles,
		bridge.WithDescription("Get list of available roles"),
	); err != nil {
		return fmt.Errorf("failed to register listRoles: %w", err)
	}

	// Get user roles
	if err := bm.bridge.Register("getUserRoles", bm.getUserRoles,
		bridge.WithDescription("Get roles assigned to a user"),
	); err != nil {
		return fmt.Errorf("failed to register getUserRoles: %w", err)
	}

	// Update user roles
	if err := bm.bridge.Register("updateUserRoles", bm.updateUserRoles,
		bridge.WithDescription("Update roles assigned to a user"),
	); err != nil {
		return fmt.Errorf("failed to register updateUserRoles: %w", err)
	}

	// Current user profile functions
	if err := bm.bridge.Register("getCurrentUserProfile", bm.getCurrentUserProfile,
		bridge.WithDescription("Get current user's profile"),
	); err != nil {
		return fmt.Errorf("failed to register getCurrentUserProfile: %w", err)
	}

	if err := bm.bridge.Register("updateCurrentUserProfile", bm.updateCurrentUserProfile,
		bridge.WithDescription("Update current user's profile"),
	); err != nil {
		return fmt.Errorf("failed to register updateCurrentUserProfile: %w", err)
	}

	if err := bm.bridge.Register("changeCurrentUserPassword", bm.changeCurrentUserPassword,
		bridge.WithDescription("Change current user's password"),
	); err != nil {
		return fmt.Errorf("failed to register changeCurrentUserPassword: %w", err)
	}

	if err := bm.bridge.Register("getCurrentUserSessions", bm.getCurrentUserSessions,
		bridge.WithDescription("Get current user's active sessions"),
	); err != nil {
		return fmt.Errorf("failed to register getCurrentUserSessions: %w", err)
	}

	if err := bm.bridge.Register("revokeCurrentUserSession", bm.revokeCurrentUserSession,
		bridge.WithDescription("Revoke a session for the current user"),
	); err != nil {
		return fmt.Errorf("failed to register revokeCurrentUserSession: %w", err)
	}

	bm.log.Info("user bridge functions registered")
	return nil
}

// getUsersList retrieves list of users
func (bm *BridgeManager) getUsersList(ctx bridge.Context, input UsersListInput) (*UsersListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Set defaults
	if input.Page == 0 {
		input.Page = 1
	}
	if input.PageSize == 0 {
		input.PageSize = 20
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context with all available context values
	goCtx := bm.buildContext(ctx, appID)

	// Build filter
	filter := &user.ListUsersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  input.Page,
			Limit: input.PageSize,
		},
		AppID: appID,
	}

	// Add search term if provided
	if input.SearchTerm != "" {
		filter.Search = &input.SearchTerm
	}

	// Add email verified filter if provided
	if input.EmailVerified != nil {
		filter.EmailVerified = input.EmailVerified
	}

	// List users from service
	response, err := bm.userSvc.ListUsers(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list users", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch users")
	}

	// Transform users to UserItem DTOs and apply in-memory filters
	users := make([]UserItem, 0, len(response.Data))
	for _, u := range response.Data {
		// Get user roles if RBAC service is available
		roles := []string{}
		if bm.rbacSvc != nil {
			// Get environment ID from context (required for RBAC)
			envID, _ := contexts.GetEnvironmentID(goCtx)
			userRoles, err := bm.rbacSvc.GetUserRolesInApp(goCtx, u.ID, appID, envID)
			if err == nil {
				for _, roleWithPerms := range userRoles {
					if roleWithPerms.Role != nil {
						roles = append(roles, roleWithPerms.Role.Name)
					}
				}
			}
		}

		// Determine status (default to active)
		status := "active"
		if u.DeletedAt != nil {
			status = "inactive"
		}

		// Apply status filter
		if input.Status != nil && *input.Status != "" && *input.Status != "all" {
			if *input.Status != status {
				continue
			}
		}

		// Apply role filter
		if input.RoleFilter != "" && input.RoleFilter != "all" {
			hasRole := false
			for _, role := range roles {
				if role == input.RoleFilter {
					hasRole = true
					break
				}
			}
			if !hasRole {
				continue
			}
		}

		users = append(users, UserItem{
			ID:            u.ID.String(),
			Email:         u.Email,
			Name:          u.Name,
			EmailVerified: u.EmailVerified,
			CreatedAt:     u.CreatedAt.Format(time.RFC3339),
			Roles:         roles,
			Status:        status,
		})
	}

	// Calculate totals based on filtered results
	totalFiltered := len(users)
	totalPages := 0
	if totalFiltered > 0 && input.PageSize > 0 {
		totalPages = (totalFiltered + input.PageSize - 1) / input.PageSize
	}

	return &UsersListOutput{
		Users:      users,
		Total:      totalFiltered,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// getUserDetail retrieves detailed information about a user
func (bm *BridgeManager) getUserDetail(ctx bridge.Context, input UserDetailInput) (*UserDetailOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}

	// Parse userID
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		bm.log.Warn("invalid userId format", forge.F("userId", input.UserID), forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId format")
	}

	// Parse and add appID if provided
	var appID xid.ID
	if input.AppID != "" {
		appID, err = xid.FromString(input.AppID)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
		}
	}

	// Build context with appId, envId, orgId, userId from bridge context
	goCtx := bm.buildContext(ctx, appID)

	// Get user from service
	u, err := bm.userSvc.FindByID(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to find user", forge.F("error", err.Error()), forge.F("userId", input.UserID), forge.F("userIdParsed", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, fmt.Sprintf("user not found: %s", input.UserID))
	}

	if u == nil {
		bm.log.Warn("user is nil after FindByID", forge.F("userId", input.UserID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "user not found")
	}

	// Get user roles if RBAC service is available
	roles := []string{}
	if bm.rbacSvc != nil {
		// Get app and environment IDs from context
		appID, _ := contexts.GetAppID(goCtx)
		envID, _ := contexts.GetEnvironmentID(goCtx)
		userRoles, err := bm.rbacSvc.GetUserRolesInApp(goCtx, u.ID, appID, envID)
		if err == nil {
			for _, roleWithPerms := range userRoles {
				if roleWithPerms.Role != nil {
					roles = append(roles, roleWithPerms.Role.Name)
				}
			}
		}
	}

	// Count user sessions
	sessionCount := 0
	lastLoginAt := ""
	if bm.sessionSvc != nil {
		sessionFilter := &session.ListSessionsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1000,
			},
			UserID: &userID,
		}
		sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
		if err == nil && sessionResponse != nil {
			sessionCount = len(sessionResponse.Data)

			// Find most recent session
			for _, sess := range sessionResponse.Data {
				if lastLoginAt == "" || sess.CreatedAt.After(parseTime(lastLoginAt)) {
					lastLoginAt = sess.CreatedAt.Format(time.RFC3339)
				}
			}
		}
	}

	return &UserDetailOutput{
		ID:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.Format(time.RFC3339),
		Roles:         roles,
		Status:        "active",
		LastLoginAt:   lastLoginAt,
		SessionCount:  sessionCount,
	}, nil
}

// parseTime is a helper to parse RFC3339 time strings
func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}

// updateUser updates user information
func (bm *BridgeManager) updateUser(ctx bridge.Context, input UpdateUserInput) (*GenericSuccessOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse userID
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context with all available context values
	goCtx := bm.buildContext(ctx, appID)

	// Get existing user
	u, err := bm.userSvc.FindByID(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to find user", forge.F("error", err.Error()), forge.F("userId", input.UserID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "user not found")
	}

	// Build update request
	updateReq := &user.UpdateUserRequest{}
	if input.Name != nil {
		updateReq.Name = input.Name
	}
	if input.Email != nil {
		updateReq.Email = input.Email
	}
	if input.EmailVerified != nil {
		updateReq.EmailVerified = input.EmailVerified
	}
	if input.Image != nil {
		updateReq.Image = input.Image
	}

	// Update user
	_, err = bm.userSvc.Update(goCtx, u, updateReq)
	if err != nil {
		bm.log.Error("failed to update user", forge.F("error", err.Error()), forge.F("userId", input.UserID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update user")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		auditData := map[string]interface{}{}
		if input.Name != nil {
			auditData["name"] = *input.Name
		}
		if input.Email != nil {
			auditData["email"] = *input.Email
		}
		if input.EmailVerified != nil {
			auditData["emailVerified"] = *input.EmailVerified
		}
		if input.Image != nil {
			auditData["image"] = *input.Image
		}
		auditJSON, _ := json.Marshal(auditData)
		_ = bm.auditSvc.Log(goCtx, &userID, "user.updated", "user:"+input.UserID, "", "", string(auditJSON))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "User updated successfully",
	}, nil
}

// deleteUser deletes a user
func (bm *BridgeManager) deleteUser(ctx bridge.Context, input DeleteUserInput) (*GenericSuccessOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}

	// Parse userID
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	// Build context with all available context values
	goCtx := bm.buildContext(ctx)

	// Revoke all user sessions first
	if bm.sessionSvc != nil {
		// List all sessions for user and revoke them
		sessionFilter := &session.ListSessionsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1000,
			},
			UserID: &userID,
		}
		sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
		if err == nil && sessionResponse != nil {
			for _, sess := range sessionResponse.Data {
				_ = bm.sessionSvc.RevokeByID(goCtx, sess.ID)
			}
		}
	}

	// Delete user
	err = bm.userSvc.Delete(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to delete user", forge.F("error", err.Error()), forge.F("userId", input.UserID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete user")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, &userID, "user.deleted", "user:"+input.UserID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

// updateUserPassword updates a user's password
func (bm *BridgeManager) updateUserPassword(ctx bridge.Context, input UpdatePasswordInput) (*GenericSuccessOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}
	if input.NewPassword == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "newPassword is required")
	}

	// Parse userID
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context with all available context values
	goCtx := bm.buildContext(ctx, appID)

	// Hash the new password
	hashedPassword, err := hashPassword(input.NewPassword)
	if err != nil {
		bm.log.Error("failed to hash password", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to process password")
	}

	// Update password
	err = bm.userSvc.UpdatePassword(goCtx, userID, hashedPassword)
	if err != nil {
		bm.log.Error("failed to update password", forge.F("error", err.Error()), forge.F("userId", input.UserID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update password")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, &userID, "user.password_updated", "user:"+input.UserID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Password updated successfully",
	}, nil
}

// bulkDeleteUsers deletes multiple users
func (bm *BridgeManager) bulkDeleteUsers(ctx bridge.Context, input BulkDeleteUsersInput) (*BulkDeleteUsersOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}
	if len(input.UserIDs) == 0 {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "at least one userId is required")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context with all available context values
	goCtx := bm.buildContext(ctx, appID)

	output := &BulkDeleteUsersOutput{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make(map[string]string),
	}

	// Delete each user
	for _, userIDStr := range input.UserIDs {
		userID, err := xid.FromString(userIDStr)
		if err != nil {
			output.FailedCount++
			output.Errors[userIDStr] = "invalid userId"
			continue
		}

		// Revoke all user sessions first
		if bm.sessionSvc != nil {
			sessionFilter := &session.ListSessionsFilter{
				PaginationParams: pagination.PaginationParams{
					Page:  1,
					Limit: 1000,
				},
				UserID: &userID,
			}
			sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
			if err == nil && sessionResponse != nil {
				for _, sess := range sessionResponse.Data {
					_ = bm.sessionSvc.RevokeByID(goCtx, sess.ID)
				}
			}
		}

		// Delete user
		err = bm.userSvc.Delete(goCtx, userID)
		if err != nil {
			output.FailedCount++
			output.Errors[userIDStr] = err.Error()
			bm.log.Error("failed to delete user in bulk", forge.F("error", err.Error()), forge.F("userId", userIDStr))
			continue
		}

		// Log audit event if audit service is available
		if bm.auditSvc != nil {
			_ = bm.auditSvc.Log(goCtx, &userID, "user.deleted", "user:"+userIDStr, "", "", "bulk_delete")
		}

		output.SuccessCount++
	}

	return output, nil
}

// listRoles retrieves list of available roles
func (bm *BridgeManager) listRoles(ctx bridge.Context, input ListRolesInput) (*ListRolesOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID and inject into context
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context
	goCtx := bm.buildContext(ctx, appID)

	// Get environment ID from context
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Get role templates for the app
	roles, err := bm.rbacSvc.GetRoleTemplates(goCtx, appID, envID)
	if err != nil {
		bm.log.Error("failed to list roles", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch roles")
	}

	// Transform to RoleItem DTOs
	roleItems := make([]RoleItem, 0, len(roles))
	for _, r := range roles {
		if r != nil {
			roleItems = append(roleItems, RoleItem{
				ID:          r.ID.String(),
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
			})
		}
	}

	return &ListRolesOutput{
		Roles: roleItems,
	}, nil
}

// getUserRoles retrieves roles assigned to a user
func (bm *BridgeManager) getUserRoles(ctx bridge.Context, input UserDetailInput) (*ListRolesOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse IDs
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Build context
	goCtx := bm.buildContext(ctx, appID)

	// Get environment ID from context
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Get user roles
	userRoles, err := bm.rbacSvc.GetUserRolesInApp(goCtx, userID, appID, envID)
	if err != nil {
		bm.log.Error("failed to get user roles", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch user roles")
	}

	// Transform to RoleItem DTOs
	roleItems := make([]RoleItem, 0, len(userRoles))
	for _, r := range userRoles {
		if r.Role != nil {
			roleItems = append(roleItems, RoleItem{
				ID:          r.Role.ID.String(),
				Name:        r.Role.Name,
				DisplayName: r.Role.DisplayName,
				Description: r.Role.Description,
			})
		}
	}

	return &ListRolesOutput{
		Roles: roleItems,
	}, nil
}

// updateUserRoles updates roles assigned to a user
func (bm *BridgeManager) updateUserRoles(ctx bridge.Context, input UpdateUserRolesInput) (*GenericSuccessOutput, error) {
	if input.UserID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "userId is required")
	}
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse IDs
	userID, err := xid.FromString(input.UserID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid userId")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// Parse role IDs
	roleIDs := make([]xid.ID, 0, len(input.RoleIDs))
	for _, roleIDStr := range input.RoleIDs {
		roleID, err := xid.FromString(roleIDStr)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, fmt.Sprintf("invalid roleId: %s", roleIDStr))
		}
		roleIDs = append(roleIDs, roleID)
	}

	// Build context
	goCtx := bm.buildContext(ctx, appID)

	// Get or parse org ID
	var orgID xid.ID
	if input.OrgID != "" {
		parsedOrgID, err := xid.FromString(input.OrgID)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid orgId")
		}
		orgID = parsedOrgID
	} else {
		// Try to get org ID from context
		orgID, _ = contexts.GetOrganizationID(goCtx)
	}

	// If still no org ID, return error
	if orgID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "orgId is required")
	}

	// Replace user roles
	err = bm.rbacSvc.ReplaceUserRoles(goCtx, userID, orgID, roleIDs)
	if err != nil {
		bm.log.Error("failed to update user roles", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update user roles")
	}

	// Log audit event
	if bm.auditSvc != nil {
		auditData := map[string]interface{}{
			"roleIds": input.RoleIDs,
		}
		auditJSON, _ := json.Marshal(auditData)
		_ = bm.auditSvc.Log(goCtx, &userID, "user.roles_updated", "user:"+input.UserID, "", "", string(auditJSON))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "User roles updated successfully",
	}, nil
}

// hashPassword is a helper to hash passwords using bcrypt
func hashPassword(password string) (string, error) {
	return crypto.HashPassword(password)
}

// searchUsers searches for users
func (bm *BridgeManager) searchUsers(ctx bridge.Context, input UsersListInput) (*UsersListOutput, error) {
	// Reuse getUsersList with search filtering
	return bm.getUsersList(ctx, input)
}

// ============================================================================
// Current User Profile Functions
// ============================================================================

// CurrentUserProfileOutput represents the current user's profile
type CurrentUserProfileOutput struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name,omitempty"`
	Image         string `json:"image,omitempty"`
	EmailVerified bool   `json:"emailVerified"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// UpdateCurrentUserProfileInput represents the update profile request
type UpdateCurrentUserProfileInput struct {
	Name  *string `json:"name,omitempty"`
	Image *string `json:"image,omitempty"`
}

// ChangePasswordInput represents the change password request
type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// CurrentUserSessionsOutput represents the current user's sessions
type CurrentUserSessionsOutput struct {
	Sessions []ProfileSessionItem `json:"sessions"`
}

// ProfileSessionItem represents a session in the profile page
type ProfileSessionItem struct {
	ID           string `json:"id"`
	UserAgent    string `json:"userAgent,omitempty"`
	IPAddress    string `json:"ipAddress,omitempty"`
	CreatedAt    string `json:"createdAt"`
	LastActiveAt string `json:"lastActiveAt,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	IsCurrent    bool   `json:"isCurrent"`
}

// ProfileRevokeSessionInput represents the revoke session request for profile page
type ProfileRevokeSessionInput struct {
	SessionID string `json:"sessionId" validate:"required"`
}

// getCurrentUserProfile retrieves the current user's profile
func (bm *BridgeManager) getCurrentUserProfile(ctx bridge.Context, _ struct{}) (*CurrentUserProfileOutput, error) {
	// Build context first to get enriched context from middleware
	goCtx := bm.buildContext(ctx)

	// Get user ID from context (set by BridgeContextMiddleware)
	userID, ok := contexts.GetUserID(goCtx)
	if !ok || userID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeUnauthorized, "not authenticated")
	}

	// Get user from service
	u, err := bm.userSvc.FindByID(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to find user", forge.F("error", err.Error()), forge.F("userId", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to load profile")
	}

	if u == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "user not found")
	}

	return &CurrentUserProfileOutput{
		ID:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		Image:         u.Image,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// updateCurrentUserProfile updates the current user's profile
func (bm *BridgeManager) updateCurrentUserProfile(ctx bridge.Context, input UpdateCurrentUserProfileInput) (*GenericSuccessOutput, error) {
	// Build context first to get enriched context from middleware
	goCtx := bm.buildContext(ctx)

	// Get user ID from context (set by BridgeContextMiddleware)
	userID, ok := contexts.GetUserID(goCtx)
	if !ok || userID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeUnauthorized, "not authenticated")
	}

	// Get existing user
	u, err := bm.userSvc.FindByID(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to find user", forge.F("error", err.Error()), forge.F("userId", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "user not found")
	}

	// Build update request
	updateReq := &user.UpdateUserRequest{}
	if input.Name != nil {
		updateReq.Name = input.Name
	}
	if input.Image != nil {
		updateReq.Image = input.Image
	}

	// Update user
	_, err = bm.userSvc.Update(goCtx, u, updateReq)
	if err != nil {
		bm.log.Error("failed to update user", forge.F("error", err.Error()), forge.F("userId", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update profile")
	}

	// Log audit event
	if bm.auditSvc != nil {
		auditData := map[string]interface{}{}
		if input.Name != nil {
			auditData["name"] = *input.Name
		}
		if input.Image != nil {
			auditData["image"] = *input.Image
		}
		auditJSON, _ := json.Marshal(auditData)
		_ = bm.auditSvc.Log(goCtx, &userID, "user.profile_updated", "user:"+userID.String(), "", "", string(auditJSON))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Profile updated successfully",
	}, nil
}

// changeCurrentUserPassword changes the current user's password
func (bm *BridgeManager) changeCurrentUserPassword(ctx bridge.Context, input ChangePasswordInput) (*GenericSuccessOutput, error) {
	// Build context first to get enriched context from middleware
	goCtx := bm.buildContext(ctx)

	// Get user ID from context (set by BridgeContextMiddleware)
	userID, ok := contexts.GetUserID(goCtx)
	if !ok || userID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeUnauthorized, "not authenticated")
	}

	// Validate input
	if input.CurrentPassword == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "current password is required")
	}
	if input.NewPassword == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "new password is required")
	}
	if len(input.NewPassword) < 8 {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "new password must be at least 8 characters")
	}

	// Get existing user to verify current password
	u, err := bm.userSvc.FindByID(goCtx, userID)
	if err != nil {
		bm.log.Error("failed to find user", forge.F("error", err.Error()), forge.F("userId", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "user not found")
	}

	// Verify current password
	if !crypto.CheckPassword(input.CurrentPassword, u.PasswordHash) {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "current password is incorrect")
	}

	// Hash the new password
	hashedPassword, err := hashPassword(input.NewPassword)
	if err != nil {
		bm.log.Error("failed to hash password", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to process password")
	}

	// Update password
	err = bm.userSvc.UpdatePassword(goCtx, userID, hashedPassword)
	if err != nil {
		bm.log.Error("failed to update password", forge.F("error", err.Error()), forge.F("userId", userID.String()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to change password")
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, &userID, "user.password_changed", "user:"+userID.String(), "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

// getCurrentUserSessions retrieves the current user's active sessions
func (bm *BridgeManager) getCurrentUserSessions(ctx bridge.Context, _ struct{}) (*CurrentUserSessionsOutput, error) {
	// Build context first to get enriched context from middleware
	goCtx := bm.buildContext(ctx)

	// Get user ID from context (set by BridgeContextMiddleware)
	userID, ok := contexts.GetUserID(goCtx)
	if !ok || userID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeUnauthorized, "not authenticated")
	}

	// Get current session token from request to identify the current session
	currentSessionToken := ""
	if req := ctx.Request(); req != nil {
		if cookie, err := req.Cookie("authsome_session"); err == nil && cookie != nil {
			currentSessionToken = cookie.Value
		}
	}

	// Get user sessions
	if bm.sessionSvc == nil {
		return &CurrentUserSessionsOutput{Sessions: []ProfileSessionItem{}}, nil
	}

	sessionFilter := &session.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 100,
		},
		UserID: &userID,
	}

	sessionResponse, err := bm.sessionSvc.ListSessions(goCtx, sessionFilter)
	if err != nil {
		bm.log.Error("failed to list sessions", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to load sessions")
	}

	// Transform sessions
	sessions := make([]ProfileSessionItem, 0, len(sessionResponse.Data))
	for _, sess := range sessionResponse.Data {
		// Check if this is the current session
		isCurrent := currentSessionToken != "" && sess.Token == currentSessionToken

		sessions = append(sessions, ProfileSessionItem{
			ID:           sess.ID.String(),
			UserAgent:    sess.UserAgent,
			IPAddress:    sess.IPAddress,
			CreatedAt:    sess.CreatedAt.Format(time.RFC3339),
			LastActiveAt: sess.UpdatedAt.Format(time.RFC3339),
			ExpiresAt:    sess.ExpiresAt.Format(time.RFC3339),
			IsCurrent:    isCurrent,
		})
	}

	return &CurrentUserSessionsOutput{
		Sessions: sessions,
	}, nil
}

// revokeCurrentUserSession revokes a session for the current user
func (bm *BridgeManager) revokeCurrentUserSession(ctx bridge.Context, input ProfileRevokeSessionInput) (*GenericSuccessOutput, error) {
	// Build context first to get enriched context from middleware
	goCtx := bm.buildContext(ctx)

	// Get user ID from context (set by BridgeContextMiddleware)
	userID, ok := contexts.GetUserID(goCtx)
	if !ok || userID.IsNil() {
		return nil, bridge.NewError(bridge.ErrCodeUnauthorized, "not authenticated")
	}

	if input.SessionID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "sessionId is required")
	}

	// Parse session ID
	sessionID, err := xid.FromString(input.SessionID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid sessionId")
	}

	// Get the session to verify it belongs to the current user
	if bm.sessionSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "session service not available")
	}

	sess, err := bm.sessionSvc.FindByID(goCtx, sessionID)
	if err != nil {
		bm.log.Error("failed to find session", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "session not found")
	}

	// Verify the session belongs to the current user
	if sess.UserID != userID {
		return nil, bridge.NewError(bridge.ErrCodeForbidden, "cannot revoke sessions for other users")
	}

	// Revoke the session
	err = bm.sessionSvc.RevokeByID(goCtx, sessionID)
	if err != nil {
		bm.log.Error("failed to revoke session", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to revoke session")
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, &userID, "session.revoked", "session:"+input.SessionID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Session revoked successfully",
	}, nil
}
