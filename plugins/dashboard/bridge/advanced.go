package bridge

import (
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// PluginsListInput represents plugins list request
type PluginsListInput struct {
	AppID string `json:"appId" validate:"required"`
}

// PluginsListOutput represents plugins list response
type PluginsListOutput struct {
	Plugins []PluginItem `json:"plugins"`
}

// PluginItem represents a plugin
type PluginItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
}

// TogglePluginInput represents plugin toggle request
type TogglePluginInput struct {
	AppID    string `json:"appId" validate:"required"`
	PluginID string `json:"pluginId" validate:"required"`
	Enabled  bool   `json:"enabled"`
}

// EnvironmentsListInput represents environments list request
type EnvironmentsListInput struct {
	AppID string `json:"appId" validate:"required"`
}

// EnvironmentsListOutput represents environments list response
type EnvironmentsListOutput struct {
	Environments []EnvironmentItem `json:"environments"`
	Current      *EnvironmentItem  `json:"current,omitempty"`
}

// EnvironmentItem represents an environment
type EnvironmentItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
}

// SwitchEnvironmentInput represents environment switch request
type SwitchEnvironmentInput struct {
	AppID string `json:"appId" validate:"required"`
	EnvID string `json:"envId" validate:"required"`
}

// EnvironmentDetailInput represents environment detail request
type EnvironmentDetailInput struct {
	AppID string `json:"appId" validate:"required"`
	EnvID string `json:"envId" validate:"required"`
}

// EnvironmentDetailOutput represents environment detail response
type EnvironmentDetailOutput struct {
	ID         string                 `json:"id"`
	AppID      string                 `json:"appId"`
	Name       string                 `json:"name"`
	Slug       string                 `json:"slug"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Config     map[string]interface{} `json:"config"`
	IsDefault  bool                   `json:"isDefault"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
	Promotions []PromotionItem        `json:"promotions"`
}

// PromotionItem represents a promotion history item
type PromotionItem struct {
	ID          string `json:"id"`
	FromEnvID   string `json:"fromEnvId"`
	FromEnvName string `json:"fromEnvName"`
	ToEnvID     string `json:"toEnvId"`
	ToEnvName   string `json:"toEnvName"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
}

// CreateEnvironmentInput represents environment creation request
type CreateEnvironmentInput struct {
	AppID       string `json:"appId" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug"`
	Type        string `json:"type"` // development, staging, production
	Description string `json:"description"`
}

// DeleteEnvironmentInput represents environment delete request
type DeleteEnvironmentInput struct {
	AppID string `json:"appId" validate:"required"`
	EnvID string `json:"envId" validate:"required"`
}

// PromoteEnvironmentInput represents environment promotion request
type PromoteEnvironmentInput struct {
	AppID       string `json:"appId" validate:"required"`
	SourceEnvID string `json:"sourceEnvId" validate:"required"`
	TargetEnvID string `json:"targetEnvId" validate:"required"`
}

// SystemConfigInput represents system config request
type SystemConfigInput struct {
	// Empty struct - no parameters needed but required for bridge signature
}

// SystemConfigOutput represents system config
type SystemConfigOutput struct {
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
	Features    map[string]bool        `json:"features"`
	Settings    map[string]interface{} `json:"settings"`
}

// AuditLogsInput represents audit logs request
type AuditLogsInput struct {
	AppID     string `json:"appId" validate:"required"`
	Action    string `json:"action,omitempty"`
	User      string `json:"user,omitempty"`
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
}

// AuditLogsOutput represents audit logs response
type AuditLogsOutput struct {
	Logs []AuditLogItem `json:"logs"`
}

// AuditLogItem represents an audit log entry
type AuditLogItem struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	UserEmail string `json:"userEmail,omitempty"`
	Resource  string `json:"resource"`
	Status    string `json:"status"`
	Details   string `json:"details,omitempty"`
}

// registerAdvancedFunctions registers advanced feature bridge functions
func (bm *BridgeManager) registerAdvancedFunctions() error {
	// Plugins management
	if err := bm.bridge.Register("getPluginsList", bm.getPluginsList,
		bridge.WithDescription("Get list of available plugins"),
	); err != nil {
		return fmt.Errorf("failed to register getPluginsList: %w", err)
	}

	if err := bm.bridge.Register("togglePlugin", bm.togglePlugin,
		bridge.WithDescription("Enable or disable a plugin"),
	); err != nil {
		return fmt.Errorf("failed to register togglePlugin: %w", err)
	}

	// Environments management
	if err := bm.bridge.Register("getEnvironmentsList", bm.getEnvironmentsList,
		bridge.WithDescription("Get list of environments"),
	); err != nil {
		return fmt.Errorf("failed to register getEnvironmentsList: %w", err)
	}

	if err := bm.bridge.Register("getEnvironmentDetail", bm.getEnvironmentDetail,
		bridge.WithDescription("Get environment details"),
	); err != nil {
		return fmt.Errorf("failed to register getEnvironmentDetail: %w", err)
	}

	if err := bm.bridge.Register("switchEnvironment", bm.switchEnvironment,
		bridge.WithDescription("Switch to a different environment"),
	); err != nil {
		return fmt.Errorf("failed to register switchEnvironment: %w", err)
	}

	if err := bm.bridge.Register("createEnvironment", bm.createEnvironment,
		bridge.WithDescription("Create a new environment"),
	); err != nil {
		return fmt.Errorf("failed to register createEnvironment: %w", err)
	}

	if err := bm.bridge.Register("deleteEnvironment", bm.deleteEnvironment,
		bridge.WithDescription("Delete an environment"),
	); err != nil {
		return fmt.Errorf("failed to register deleteEnvironment: %w", err)
	}

	if err := bm.bridge.Register("promoteEnvironment", bm.promoteEnvironment,
		bridge.WithDescription("Promote environment configuration to another environment"),
	); err != nil {
		return fmt.Errorf("failed to register promoteEnvironment: %w", err)
	}

	// System config
	if err := bm.bridge.Register("getSystemConfig", bm.getSystemConfig,
		bridge.WithDescription("Get system configuration"),
	); err != nil {
		return fmt.Errorf("failed to register getSystemConfig: %w", err)
	}

	// Audit logs
	if err := bm.bridge.Register("getAuditLogs", bm.getAuditLogs,
		bridge.WithDescription("Get audit logs"),
	); err != nil {
		return fmt.Errorf("failed to register getAuditLogs: %w", err)
	}

	bm.log.Info("advanced feature bridge functions registered")
	return nil
}

// getPluginsList retrieves list of plugins
func (bm *BridgeManager) getPluginsList(ctx bridge.Context, input PluginsListInput) (*PluginsListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Use enabled plugins map from bridge manager
	enabledPlugins := bm.enabledPlugins

	// Build comprehensive list of all available plugins
	allPlugins := []PluginItem{
		// Core/Administration
		{ID: "dashboard", Name: "Dashboard", Description: "Administrative dashboard for managing authentication", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("dashboard", enabledPlugins)},
		{ID: "multiapp", Name: "Multi-App", Description: "Manage multiple applications within a single platform", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("multiapp", enabledPlugins)},
		{ID: "admin", Name: "Admin", Description: "Administrative tools and user management", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("admin", enabledPlugins)},
		{ID: "cms", Name: "CMS", Description: "Content management system integration", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("cms", enabledPlugins)},

		// Authentication Methods
		{ID: "oidcprovider", Name: "OIDC Provider", Description: "OpenID Connect provider implementation", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("oidcprovider", enabledPlugins)},
		{ID: "social", Name: "Social Login", Description: "Third-party social authentication providers", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("social", enabledPlugins)},
		{ID: "username", Name: "Username/Password", Description: "Traditional username and password authentication", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("username", enabledPlugins)},
		{ID: "emailverification", Name: "Email Verification", Description: "Email-based account verification", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("emailverification", enabledPlugins)},
		{ID: "emailotp", Name: "Email OTP", Description: "One-time password via email", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("emailotp", enabledPlugins)},
		{ID: "phone", Name: "Phone Auth", Description: "SMS-based authentication", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("phone", enabledPlugins)},
		{ID: "passkey", Name: "Passkeys", Description: "WebAuthn/FIDO2 passwordless authentication", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("passkey", enabledPlugins)},
		{ID: "magiclink", Name: "Magic Link", Description: "Passwordless authentication via email links", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("magiclink", enabledPlugins)},
		{ID: "anonymous", Name: "Anonymous Auth", Description: "Anonymous user sessions", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("anonymous", enabledPlugins)},
		{ID: "sso", Name: "SSO", Description: "Single Sign-On integration", Version: "1.0.0", Category: "Authentication", Enabled: isPluginEnabled("sso", enabledPlugins)},

		// Security & MFA
		{ID: "mfa", Name: "Multi-Factor Auth", Description: "Multi-factor authentication framework", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("mfa", enabledPlugins)},
		{ID: "twofa", Name: "Two-Factor Auth", Description: "TOTP-based 2FA", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("twofa", enabledPlugins)},
		{ID: "bearer", Name: "Bearer Token", Description: "Bearer token authentication", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("bearer", enabledPlugins)},
		{ID: "jwt", Name: "JWT", Description: "JSON Web Token authentication", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("jwt", enabledPlugins)},
		{ID: "apikey", Name: "API Keys", Description: "API key management for service accounts", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("apikey", enabledPlugins)},
		{ID: "secrets", Name: "Secrets Manager", Description: "Secure secrets and credentials management", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("secrets", enabledPlugins)},

		// Session Management
		{ID: "multisession", Name: "Multi-Session", Description: "Multiple concurrent session support", Version: "1.0.0", Category: "Session", Enabled: isPluginEnabled("multisession", enabledPlugins)},

		// Organization & Permissions
		{ID: "organization", Name: "Organizations", Description: "Multi-tenant organization management", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("organization", enabledPlugins)},
		{ID: "permissions", Name: "Permissions", Description: "Fine-grained permission system", Version: "1.0.0", Category: "Security", Enabled: isPluginEnabled("permissions", enabledPlugins)},
		{ID: "impersonation", Name: "User Impersonation", Description: "Admin user impersonation capabilities", Version: "1.0.0", Category: "Administration", Enabled: isPluginEnabled("impersonation", enabledPlugins)},

		// Communication
		{ID: "notification", Name: "Notifications", Description: "Multi-channel notification system", Version: "1.0.0", Category: "Communication", Enabled: isPluginEnabled("notification", enabledPlugins)},

		// Billing & Subscription
		{ID: "subscription", Name: "Subscriptions", Description: "Subscription and billing management", Version: "1.0.0", Category: "Commerce", Enabled: isPluginEnabled("subscription", enabledPlugins)},

		// Integration
		{ID: "mcp", Name: "MCP", Description: "Model Context Protocol integration", Version: "1.0.0", Category: "Integration", Enabled: isPluginEnabled("mcp", enabledPlugins)},

		// Enterprise Features
		{ID: "compliance", Name: "Compliance", Description: "Regulatory compliance and audit trails", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("compliance", enabledPlugins)},
		{ID: "geofence", Name: "Geofencing", Description: "Geographic access restrictions", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("geofence", enabledPlugins)},
		{ID: "scim", Name: "SCIM", Description: "System for Cross-domain Identity Management", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("scim", enabledPlugins)},
		{ID: "stepup", Name: "Step-Up Auth", Description: "Progressive authentication for sensitive operations", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("stepup", enabledPlugins)},
		{ID: "idverification", Name: "ID Verification", Description: "Identity verification services", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("idverification", enabledPlugins)},
		{ID: "mtls", Name: "mTLS", Description: "Mutual TLS authentication", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("mtls", enabledPlugins)},
		{ID: "consent", Name: "Consent Management", Description: "User consent and privacy management", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("consent", enabledPlugins)},
		{ID: "backupauth", Name: "Backup Auth", Description: "Backup authentication methods", Version: "1.0.0", Category: "Enterprise", Enabled: isPluginEnabled("backupauth", enabledPlugins)},
	}

	return &PluginsListOutput{
		Plugins: allPlugins,
	}, nil
}

// isPluginEnabled checks if a plugin is in the enabled plugins map
func isPluginEnabled(pluginID string, enabledPlugins map[string]bool) bool {
	if enabledPlugins == nil {
		return false
	}
	return enabledPlugins[pluginID]
}

// togglePlugin enables or disables a plugin
func (bm *BridgeManager) togglePlugin(ctx bridge.Context, input TogglePluginInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.PluginID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and pluginId are required")
	}

	// TODO: Implement actual plugin toggle
	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("Plugin %s successfully", map[bool]string{true: "enabled", false: "disabled"}[input.Enabled]),
	}, nil
}

// getEnvironmentsList retrieves list of environments
func (bm *BridgeManager) getEnvironmentsList(ctx bridge.Context, input EnvironmentsListInput) (*EnvironmentsListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// If environment service is not available, return empty list
	if bm.envSvc == nil {
		bm.log.Warn("environment service not available")
		return &EnvironmentsListOutput{
			Environments: []EnvironmentItem{},
			Current:      nil,
		}, nil
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// List environments from service
	filter := &environment.ListEnvironmentsFilter{
		AppID: appID,
	}

	response, err := bm.envSvc.ListEnvironments(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list environments", forge.F("error", err.Error()), forge.F("appId", input.AppID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch environments")
	}

	// Transform environments to EnvironmentItem DTOs
	environments := make([]EnvironmentItem, len(response.Data))
	for i, env := range response.Data {
		environments[i] = EnvironmentItem{
			ID:        env.ID.String(),
			Name:      env.Name,
			Type:      env.Type,
			CreatedAt: env.CreatedAt.Format(time.RFC3339),
		}
	}

	// Get current/default environment
	var current *EnvironmentItem
	defaultEnv, err := bm.envSvc.GetDefaultEnvironment(goCtx, appID)
	if err == nil && defaultEnv != nil {
		current = &EnvironmentItem{
			ID:        defaultEnv.ID.String(),
			Name:      defaultEnv.Name,
			Type:      defaultEnv.Type,
			CreatedAt: defaultEnv.CreatedAt.Format(time.RFC3339),
		}
	}

	return &EnvironmentsListOutput{
		Environments: environments,
		Current:      current,
	}, nil
}

// switchEnvironment switches to a different environment
func (bm *BridgeManager) switchEnvironment(ctx bridge.Context, input SwitchEnvironmentInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.EnvID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and envId are required")
	}

	// Parse appID and envID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	envID, err := xid.FromString(input.EnvID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid envId")
	}

	// If environment service is not available, return error
	if bm.envSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment service not available")
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// Validate environment exists
	env, err := bm.envSvc.GetEnvironment(goCtx, envID)
	if err != nil {
		bm.log.Error("failed to get environment", forge.F("error", err.Error()), forge.F("envId", input.EnvID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment not found")
	}

	// Verify environment belongs to the app
	if env.AppID != appID {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "environment does not belong to this app")
	}

	// TODO: Set environment cookie in HTTP response
	// This should be handled at a higher level or through middleware
	// since the bridge context doesn't expose ResponseWriter directly

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("Switched to %s environment", env.Name),
	}, nil
}

// getEnvironmentDetail retrieves detailed information about an environment
func (bm *BridgeManager) getEnvironmentDetail(ctx bridge.Context, input EnvironmentDetailInput) (*EnvironmentDetailOutput, error) {
	if input.AppID == "" || input.EnvID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and envId are required")
	}

	// Parse appID and envID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	envID, err := xid.FromString(input.EnvID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid envId")
	}

	// If environment service is not available, return error
	if bm.envSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment service not available")
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// Get environment details
	env, err := bm.envSvc.GetEnvironment(goCtx, envID)
	if err != nil {
		bm.log.Error("failed to get environment", forge.F("error", err.Error()), forge.F("envId", input.EnvID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment not found")
	}

	// Verify environment belongs to the app
	if env.AppID != appID {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "environment does not belong to this app")
	}

	// Get promotion history for this environment
	var promotions []PromotionItem
	promotionsFilter := &environment.ListPromotionsFilter{
		AppID:       appID,
		SourceEnvID: &envID,
	}
	promotionsResp, err := bm.envSvc.ListPromotions(goCtx, promotionsFilter)
	if err == nil && promotionsResp != nil && len(promotionsResp.Data) > 0 {
		// Transform promotions
		for _, p := range promotionsResp.Data {
			// Get source and target environment names
			sourceEnv, _ := bm.envSvc.GetEnvironment(goCtx, p.SourceEnvID)
			targetEnv, _ := bm.envSvc.GetEnvironment(goCtx, p.TargetEnvID)

			sourceName := "Unknown"
			targetName := "Unknown"
			if sourceEnv != nil {
				sourceName = sourceEnv.Name
			}
			if targetEnv != nil {
				targetName = targetEnv.Name
			}

			promotions = append(promotions, PromotionItem{
				ID:          p.ID.String(),
				FromEnvID:   p.SourceEnvID.String(),
				FromEnvName: sourceName,
				ToEnvID:     p.TargetEnvID.String(),
				ToEnvName:   targetName,
				Status:      p.Status,
				CreatedAt:   p.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return &EnvironmentDetailOutput{
		ID:         env.ID.String(),
		AppID:      env.AppID.String(),
		Name:       env.Name,
		Slug:       env.Slug,
		Type:       env.Type,
		Status:     env.Status,
		Config:     env.Config,
		IsDefault:  env.IsDefault,
		CreatedAt:  env.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  env.UpdatedAt.Format(time.RFC3339),
		Promotions: promotions,
	}, nil
}

// createEnvironment creates a new environment
func (bm *BridgeManager) createEnvironment(ctx bridge.Context, input CreateEnvironmentInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.Name == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and name are required")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	// If environment service is not available, return error
	if bm.envSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment service not available")
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// Set defaults
	envType := input.Type
	if envType == "" {
		envType = "development"
	}

	// Create environment request
	req := &environment.CreateEnvironmentRequest{
		AppID: appID,
		Name:  input.Name,
		Slug:  input.Slug,
		Type:  envType,
	}

	_, err = bm.envSvc.CreateEnvironment(goCtx, req)
	if err != nil {
		bm.log.Error("failed to create environment", forge.F("error", err.Error()), forge.F("appId", input.AppID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create environment: "+err.Error())
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("Environment '%s' created successfully", input.Name),
	}, nil
}

// deleteEnvironment deletes an environment
func (bm *BridgeManager) deleteEnvironment(ctx bridge.Context, input DeleteEnvironmentInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.EnvID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and envId are required")
	}

	// Parse appID and envID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	envID, err := xid.FromString(input.EnvID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid envId")
	}

	// If environment service is not available, return error
	if bm.envSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment service not available")
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// Verify environment exists and belongs to the app
	env, err := bm.envSvc.GetEnvironment(goCtx, envID)
	if err != nil {
		bm.log.Error("failed to get environment", forge.F("error", err.Error()), forge.F("envId", input.EnvID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment not found")
	}

	if env.AppID != appID {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "environment does not belong to this app")
	}

	// Check if it's the default environment
	if env.IsDefault {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "cannot delete the default environment")
	}

	// Delete environment
	err = bm.envSvc.DeleteEnvironment(goCtx, envID)
	if err != nil {
		bm.log.Error("failed to delete environment", forge.F("error", err.Error()), forge.F("envId", input.EnvID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete environment: "+err.Error())
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("Environment '%s' deleted successfully", env.Name),
	}, nil
}

// promoteEnvironment promotes configuration from one environment to another
func (bm *BridgeManager) promoteEnvironment(ctx bridge.Context, input PromoteEnvironmentInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.SourceEnvID == "" || input.TargetEnvID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId, sourceEnvId, and targetEnvId are required")
	}

	// Parse IDs
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	sourceEnvID, err := xid.FromString(input.SourceEnvID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid sourceEnvId")
	}

	targetEnvID, err := xid.FromString(input.TargetEnvID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid targetEnvId")
	}

	// If environment service is not available, return error
	if bm.envSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "environment service not available")
	}

	// Build context with appId
	goCtx := bm.buildContext(ctx, appID)

	// Get target environment details to use for the promotion
	targetEnv, err := bm.envSvc.GetEnvironment(goCtx, targetEnvID)
	if err != nil {
		bm.log.Error("failed to get target environment", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "target environment not found")
	}

	// Create promotion request using target environment's details
	// This effectively copies the source config to the target environment
	req := &environment.PromoteEnvironmentRequest{
		SourceEnvID: sourceEnvID,
		TargetName:  targetEnv.Name,
		TargetSlug:  targetEnv.Slug,
		TargetType:  targetEnv.Type,
		IncludeData: false,
		Config:      targetEnv.Config,
	}

	_, err = bm.envSvc.PromoteEnvironment(goCtx, req)
	if err != nil {
		bm.log.Error("failed to promote environment", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to promote environment: "+err.Error())
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("Configuration promoted to %s successfully", targetEnv.Name),
	}, nil
}

// getSystemConfig retrieves system configuration
func (bm *BridgeManager) getSystemConfig(ctx bridge.Context, input SystemConfigInput) (*SystemConfigOutput, error) {
	// TODO: Implement actual config retrieval
	return &SystemConfigOutput{
		Version:     "2.0.0",
		Environment: "production",
		Features: map[string]bool{
			"oauth":        true,
			"twofa":        true,
			"multiapp":     true,
			"environments": true,
		},
		Settings: map[string]interface{}{
			"sessionDuration":   24,
			"maxLoginAttempts":  5,
			"passwordMinLength": 8,
		},
	}, nil
}

// getAuditLogs retrieves audit logs
func (bm *BridgeManager) getAuditLogs(ctx bridge.Context, input AuditLogsInput) (*AuditLogsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// TODO: Implement actual audit logs retrieval from audit service
	return &AuditLogsOutput{
		Logs: []AuditLogItem{
			{ID: "1", Timestamp: "2025-01-05T10:30:00Z", Action: "user.login", UserEmail: "admin@example.com", Resource: "user:admin", Status: "success"},
			{ID: "2", Timestamp: "2025-01-05T10:25:00Z", Action: "user.create", UserEmail: "admin@example.com", Resource: "user:newuser", Status: "success"},
			{ID: "3", Timestamp: "2025-01-05T10:20:00Z", Action: "settings.update", UserEmail: "admin@example.com", Resource: "settings:security", Status: "success"},
		},
	}, nil
}
