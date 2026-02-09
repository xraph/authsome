package bridge

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// AppsListOutput represents apps list response.
type AppsListOutput struct {
	Apps         []AppItem `json:"apps"`
	IsMultiApp   bool      `json:"isMultiApp"`   // Whether multiapp mode is enabled
	DefaultAppID string    `json:"defaultAppId"` // Default app ID in standalone mode
}

// AppItem represents an app in the list.
type AppItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UserCount   int    `json:"userCount"`
	CreatedAt   string `json:"createdAt"`
	Status      string `json:"status"`
}

// AppDetailInput represents app detail request.
type AppDetailInput struct {
	AppID string `json:"appId" validate:"required"`
}

// AppDetailOutput represents app detail response.
type AppDetailOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UserCount   int    `json:"userCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	Status      string `json:"status"`
}

// CreateAppInput represents app creation request.
type CreateAppInput struct {
	Name        string `json:"name"                  validate:"required"`
	Description string `json:"description,omitempty"`
}

// UpdateAppInput represents app update request.
type UpdateAppInput struct {
	AppID       string `json:"appId"                 validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// DeleteAppInput represents app delete request.
type DeleteAppInput struct {
	AppID string `json:"appId" validate:"required"`
}

// registerAppFunctions registers app management bridge functions.
func (bm *BridgeManager) registerAppFunctions() error {
	// List apps - no auth required as it's read-only and needed for navbar
	if err := bm.bridge.Register("getAppsList", bm.getAppsList,
		bridge.WithDescription("Get list of platform apps"),
	); err != nil {
		return fmt.Errorf("failed to register getAppsList: %w", err)
	}

	// Get app detail
	if err := bm.bridge.Register("getAppDetail", bm.getAppDetail,
		bridge.WithDescription("Get detailed information about an app"),
	); err != nil {
		return fmt.Errorf("failed to register getAppDetail: %w", err)
	}

	// Create app (requires multiapp plugin)
	if err := bm.bridge.Register("createApp", bm.createApp,
		bridge.WithDescription("Create a new app (requires multiapp plugin)"),
	); err != nil {
		return fmt.Errorf("failed to register createApp: %w", err)
	}

	// Update app
	if err := bm.bridge.Register("updateApp", bm.updateApp,
		bridge.WithDescription("Update app information"),
	); err != nil {
		return fmt.Errorf("failed to register updateApp: %w", err)
	}

	// Delete app
	if err := bm.bridge.Register("deleteApp", bm.deleteApp,
		bridge.WithDescription("Delete an app"),
	); err != nil {
		return fmt.Errorf("failed to register deleteApp: %w", err)
	}

	bm.log.Info("app bridge functions registered")

	return nil
}

// getAppsList retrieves list of apps.
func (bm *BridgeManager) getAppsList(ctx bridge.Context, _ struct{}) (*AppsListOutput, error) {
	goCtx := bm.buildContext(ctx)

	// List all apps with pagination
	filter := &app.ListAppsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 100, // Get all apps
		},
	}

	response, err := bm.appSvc.ListApps(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list apps", forge.F("error", err.Error()))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch apps")
	}

	// Transform apps to AppItem DTOs
	apps := make([]AppItem, len(response.Data))
	for i, a := range response.Data {
		// Count users for this app
		userCount := 0

		if bm.userSvc != nil {
			userFilter := &user.CountUsersFilter{
				AppID: a.ID,
			}

			count, err := bm.userSvc.CountUsers(goCtx, userFilter)
			if err == nil {
				userCount = count
			}
		}

		// Extract description from metadata if available
		description := ""

		if a.Metadata != nil {
			if desc, ok := a.Metadata["description"].(string); ok {
				description = desc
			}
		}

		apps[i] = AppItem{
			ID:          a.ID.String(),
			Name:        a.Name,
			Description: description,
			UserCount:   userCount,
			CreatedAt:   a.CreatedAt.Format(time.RFC3339),
			Status:      "active", // Default status
		}
	}

	// Check if multiapp mode is enabled by checking if there are multiple apps
	isMultiApp := len(apps) > 1
	isMultiApp = bm.enabledPlugins["multiapp"]

	defaultAppID := ""
	if !isMultiApp && len(apps) > 0 {
		// In standalone mode, set the first (and typically only) app as default
		defaultAppID = apps[0].ID
	}

	return &AppsListOutput{
		Apps:         apps,
		IsMultiApp:   isMultiApp,
		DefaultAppID: defaultAppID,
	}, nil
}

// getAppDetail retrieves detailed information about an app.
func (bm *BridgeManager) getAppDetail(ctx bridge.Context, input AppDetailInput) (*AppDetailOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Get app from service
	a, err := bm.appSvc.FindAppByID(goCtx, appID)
	if err != nil {
		bm.log.Error("failed to find app", forge.F("error", err.Error()), forge.F("appId", input.AppID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "app not found")
	}

	// Count users for this app
	userCount := 0

	if bm.userSvc != nil {
		userFilter := &user.CountUsersFilter{
			AppID: appID,
		}

		count, err := bm.userSvc.CountUsers(goCtx, userFilter)
		if err == nil {
			userCount = count
		}
	}

	// Extract description from metadata if available
	description := ""

	if a.Metadata != nil {
		if desc, ok := a.Metadata["description"].(string); ok {
			description = desc
		}
	}

	return &AppDetailOutput{
		ID:          a.ID.String(),
		Name:        a.Name,
		Description: description,
		UserCount:   userCount,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
		Status:      "active",
	}, nil
}

// createApp creates a new app.
func (bm *BridgeManager) createApp(ctx bridge.Context, input CreateAppInput) (*GenericSuccessOutput, error) {
	if input.Name == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "name is required")
	}

	goCtx := bm.buildContext(ctx)

	// Check if multiapp mode is enabled by counting existing apps
	count, err := bm.appSvc.CountApps(goCtx)
	if err != nil {
		bm.log.Error("failed to count apps", forge.F("error", err.Error()))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to check multiapp status")
	}

	// Allow creation if there's already more than one app (multiapp mode)
	// or if this is the first app being created
	if count > 1 {
		// Multiapp mode is enabled
	}

	// Create app
	metadata := make(map[string]any)
	if input.Description != "" {
		metadata["description"] = input.Description
	}

	createReq := &app.CreateAppRequest{
		Name:     input.Name,
		Metadata: metadata,
	}

	_, err = bm.appSvc.CreateApp(goCtx, createReq)
	if err != nil {
		bm.log.Error("failed to create app", forge.F("error", err.Error()))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create app")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		metadata, _ := json.Marshal(map[string]string{"name": input.Name})
		_ = bm.auditSvc.Log(goCtx, nil, "app.created", "app", "", "", string(metadata))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "App created successfully",
	}, nil
}

// updateApp updates app information.
func (bm *BridgeManager) updateApp(ctx bridge.Context, input UpdateAppInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Get existing app
	a, err := bm.appSvc.FindAppByID(goCtx, appID)
	if err != nil {
		bm.log.Error("failed to find app", forge.F("error", err.Error()), forge.F("appId", input.AppID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "app not found")
	}

	// Build update request
	updateReq := &app.UpdateAppRequest{}
	if input.Name != "" {
		updateReq.Name = &input.Name
	}

	// Update metadata with description
	if input.Description != "" {
		metadata := a.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}

		metadata["description"] = input.Description
		updateReq.Metadata = metadata
	}

	// Update app
	_, err = bm.appSvc.UpdateApp(goCtx, appID, updateReq)
	if err != nil {
		bm.log.Error("failed to update app", forge.F("error", err.Error()), forge.F("appId", input.AppID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update app")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		metadata, _ := json.Marshal(map[string]string{"name": input.Name, "description": input.Description})
		_ = bm.auditSvc.Log(goCtx, nil, "app.updated", "app:"+input.AppID, "", "", string(metadata))
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "App updated successfully",
	}, nil
}

// deleteApp deletes an app.
func (bm *BridgeManager) deleteApp(ctx bridge.Context, input DeleteAppInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)

	// Check if this is the platform app (cannot be deleted)
	a, err := bm.appSvc.FindAppByID(goCtx, appID)
	if err != nil {
		bm.log.Error("failed to find app", forge.F("error", err.Error()), forge.F("appId", input.AppID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "app not found")
	}

	if a.IsPlatform {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "cannot delete platform app")
	}

	// Delete app
	err = bm.appSvc.DeleteApp(goCtx, appID)
	if err != nil {
		bm.log.Error("failed to delete app", forge.F("error", err.Error()), forge.F("appId", input.AppID))

		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete app")
	}

	// Log audit event if audit service is available
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "app.deleted", "app:"+input.AppID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "App deleted successfully",
	}, nil
}
