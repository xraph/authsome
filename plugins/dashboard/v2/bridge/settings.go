package bridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui/schema"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// SettingsMetadataKey is the key used in App.Metadata for storing settings
const SettingsMetadataKey = "settings"

// GeneralSettingsInput represents general settings request
type GeneralSettingsInput struct {
	AppID string `json:"appId" validate:"required"`
}

// GeneralSettingsOutput represents general settings response
type GeneralSettingsOutput struct {
	AppName     string `json:"appName"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone"`
	Language    string `json:"language"`
	DateFormat  string `json:"dateFormat,omitempty"`
	SupportEmail string `json:"supportEmail,omitempty"`
	WebsiteUrl  string `json:"websiteUrl,omitempty"`
}

// UpdateGeneralSettingsInput represents general settings update request
type UpdateGeneralSettingsInput struct {
	AppID        string `json:"appId" validate:"required"`
	AppName      string `json:"appName,omitempty"`
	Description  string `json:"description,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	Language     string `json:"language,omitempty"`
	DateFormat   string `json:"dateFormat,omitempty"`
	SupportEmail string `json:"supportEmail,omitempty"`
	WebsiteUrl   string `json:"websiteUrl,omitempty"`
}

// SecuritySettingsOutput represents security settings response
type SecuritySettingsOutput struct {
	PasswordMinLength      int      `json:"passwordMinLength"`
	RequireUppercase       bool     `json:"requireUppercase"`
	RequireLowercase       bool     `json:"requireLowercase"`
	RequireNumbers         bool     `json:"requireNumbers"`
	RequireSpecialChars    bool     `json:"requireSpecialChars"`
	MaxLoginAttempts       int      `json:"maxLoginAttempts"`
	LockoutDuration        int      `json:"lockoutDuration"`
	RequireMFA             bool     `json:"requireMFA"`
	AllowedMFAMethods      []string `json:"allowedMfaMethods,omitempty"`
	AllowedIPAddresses     []string `json:"allowedIpAddresses,omitempty"`
}

// UpdateSecuritySettingsInput represents security settings update request
type UpdateSecuritySettingsInput struct {
	AppID                  string   `json:"appId" validate:"required"`
	PasswordMinLength      *int     `json:"passwordMinLength,omitempty"`
	RequireUppercase       *bool    `json:"requireUppercase,omitempty"`
	RequireLowercase       *bool    `json:"requireLowercase,omitempty"`
	RequireNumbers         *bool    `json:"requireNumbers,omitempty"`
	RequireSpecialChars    *bool    `json:"requireSpecialChars,omitempty"`
	MaxLoginAttempts       *int     `json:"maxLoginAttempts,omitempty"`
	LockoutDuration        *int     `json:"lockoutDuration,omitempty"`
	RequireMFA             *bool    `json:"requireMFA,omitempty"`
	AllowedMFAMethods      []string `json:"allowedMfaMethods,omitempty"`
	AllowedIPAddresses     []string `json:"allowedIpAddresses,omitempty"`
}

// SessionSettingsOutput represents session settings response
type SessionSettingsOutput struct {
	SessionDuration        int    `json:"sessionDuration"`
	RefreshTokenDuration   int    `json:"refreshTokenDuration"`
	IdleTimeout            int    `json:"idleTimeout"`
	AllowMultipleSessions  bool   `json:"allowMultipleSessions"`
	MaxConcurrentSessions  int    `json:"maxConcurrentSessions"`
	RememberMeEnabled      bool   `json:"rememberMeEnabled"`
	RememberMeDuration     int    `json:"rememberMeDuration"`
	CookieSameSite         string `json:"cookieSameSite"`
	CookieSecure           bool   `json:"cookieSecure"`
}

// UpdateSessionSettingsInput represents session settings update request
type UpdateSessionSettingsInput struct {
	AppID                  string  `json:"appId" validate:"required"`
	SessionDuration        *int    `json:"sessionDuration,omitempty"`
	RefreshTokenDuration   *int    `json:"refreshTokenDuration,omitempty"`
	IdleTimeout            *int    `json:"idleTimeout,omitempty"`
	AllowMultipleSessions  *bool   `json:"allowMultipleSessions,omitempty"`
	MaxConcurrentSessions  *int    `json:"maxConcurrentSessions,omitempty"`
	RememberMeEnabled      *bool   `json:"rememberMeEnabled,omitempty"`
	RememberMeDuration     *int    `json:"rememberMeDuration,omitempty"`
	CookieSameSite         *string `json:"cookieSameSite,omitempty"`
	CookieSecure           *bool   `json:"cookieSecure,omitempty"`
}

// SectionSettingsInput represents a request for any section's settings
type SectionSettingsInput struct {
	AppID     string `json:"appId" validate:"required"`
	SectionID string `json:"sectionId" validate:"required"`
}

// SectionSettingsOutput represents any section's settings response
type SectionSettingsOutput struct {
	SectionID string                 `json:"sectionId"`
	Data      map[string]interface{} `json:"data"`
}

// UpdateSectionSettingsInput represents a section settings update request
type UpdateSectionSettingsInput struct {
	AppID     string                 `json:"appId" validate:"required"`
	SectionID string                 `json:"sectionId" validate:"required"`
	Data      map[string]interface{} `json:"data" validate:"required"`
}

// SchemaOutput represents the settings schema response
type SchemaOutput struct {
	Schema *schema.Schema `json:"schema"`
}

// Note: API key types and functions have been moved to apikeys.go

// registerSettingsFunctions registers settings-related bridge functions
func (bm *BridgeManager) registerSettingsFunctions() error {
	// General settings
	if err := bm.bridge.Register("getGeneralSettings", bm.getGeneralSettings,
		bridge.WithDescription("Get general app settings"),
	); err != nil {
		return fmt.Errorf("failed to register getGeneralSettings: %w", err)
	}

	if err := bm.bridge.Register("updateGeneralSettings", bm.updateGeneralSettings,
		bridge.WithDescription("Update general app settings"),
	); err != nil {
		return fmt.Errorf("failed to register updateGeneralSettings: %w", err)
	}

	// Security settings
	if err := bm.bridge.Register("getSecuritySettings", bm.getSecuritySettings,
		bridge.WithDescription("Get security settings"),
	); err != nil {
		return fmt.Errorf("failed to register getSecuritySettings: %w", err)
	}

	if err := bm.bridge.Register("updateSecuritySettings", bm.updateSecuritySettings,
		bridge.WithDescription("Update security settings"),
	); err != nil {
		return fmt.Errorf("failed to register updateSecuritySettings: %w", err)
	}

	// Session settings
	if err := bm.bridge.Register("getSessionSettings", bm.getSessionSettings,
		bridge.WithDescription("Get session settings"),
	); err != nil {
		return fmt.Errorf("failed to register getSessionSettings: %w", err)
	}

	if err := bm.bridge.Register("updateSessionSettings", bm.updateSessionSettings,
		bridge.WithDescription("Update session settings"),
	); err != nil {
		return fmt.Errorf("failed to register updateSessionSettings: %w", err)
	}

	// Generic section settings (for dynamic sections)
	if err := bm.bridge.Register("getSectionSettings", bm.getSectionSettings,
		bridge.WithDescription("Get settings for any section"),
	); err != nil {
		return fmt.Errorf("failed to register getSectionSettings: %w", err)
	}

	if err := bm.bridge.Register("updateSectionSettings", bm.updateSectionSettings,
		bridge.WithDescription("Update settings for any section"),
	); err != nil {
		return fmt.Errorf("failed to register updateSectionSettings: %w", err)
	}

	// Settings schema
	if err := bm.bridge.Register("getSettingsSchema", bm.getSettingsSchema,
		bridge.WithDescription("Get the settings schema for rendering forms"),
	); err != nil {
		return fmt.Errorf("failed to register getSettingsSchema: %w", err)
	}

	// Note: API key functions are registered in apikeys.go

	bm.log.Info("settings bridge functions registered")
	return nil
}

// getAppSettings retrieves settings from app metadata
func (bm *BridgeManager) getAppSettings(ctx context.Context, appID xid.ID) (map[string]map[string]interface{}, error) {
	appData, err := bm.appSvc.FindAppByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	settings := make(map[string]map[string]interface{})

	if appData.Metadata != nil {
		if settingsData, ok := appData.Metadata[SettingsMetadataKey]; ok {
			// Handle different types that might be in metadata
			switch v := settingsData.(type) {
			case map[string]interface{}:
				for sectionID, sectionData := range v {
					if sd, ok := sectionData.(map[string]interface{}); ok {
						settings[sectionID] = sd
					}
				}
			case map[string]map[string]interface{}:
				settings = v
			}
		}
	}

	return settings, nil
}

// getSectionSettings retrieves settings for a specific section
func (bm *BridgeManager) getSectionSettingsData(ctx context.Context, appID xid.ID, sectionID string) (map[string]interface{}, error) {
	allSettings, err := bm.getAppSettings(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Get section defaults
	section := schema.GetGlobalSection(sectionID)
	if section == nil {
		return nil, fmt.Errorf("section not found: %s", sectionID)
	}

	defaults := section.GetDefaults()

	// Merge with stored settings
	sectionSettings := allSettings[sectionID]
	if sectionSettings == nil {
		return defaults, nil
	}

	// Overlay stored settings on defaults
	for k, v := range sectionSettings {
		defaults[k] = v
	}

	return defaults, nil
}

// updateSectionSettingsData updates settings for a specific section with validation
func (bm *BridgeManager) updateSectionSettingsData(ctx context.Context, appID xid.ID, sectionID string, data map[string]interface{}) error {
	// Validate against schema
	section := schema.GetGlobalSection(sectionID)
	if section == nil {
		return fmt.Errorf("section not found: %s", sectionID)
	}

	validationResult := section.Validate(ctx, data)
	if validationResult.HasErrors() {
		return fmt.Errorf("validation failed: %s", validationResult.Error())
	}

	// Get current app
	appData, err := bm.appSvc.FindAppByID(ctx, appID)
	if err != nil {
		return err
	}

	// Get existing settings
	allSettings, err := bm.getAppSettings(ctx, appID)
	if err != nil {
		return err
	}

	// Get existing section data
	existingSection := allSettings[sectionID]
	if existingSection == nil {
		existingSection = make(map[string]interface{})
	}

	// Patch the section
	patchedData, err := section.Patch(existingSection, data)
	if err != nil {
		return err
	}

	// Update the settings map
	allSettings[sectionID] = patchedData

	// Prepare metadata update
	metadata := appData.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata[SettingsMetadataKey] = allSettings

	// Update app
	updateReq := &app.UpdateAppRequest{
		Metadata: metadata,
	}

	_, err = bm.appSvc.UpdateApp(ctx, appID, updateReq)
	return err
}

// getGeneralSettings retrieves general settings
func (bm *BridgeManager) getGeneralSettings(ctx bridge.Context, input GeneralSettingsInput) (*GeneralSettingsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Get app for name
	appData, err := bm.appSvc.FindAppByID(goCtx, appID)
	if err != nil {
		bm.log.Error("failed to find app", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to load app")
	}

	// Get section settings
	settings, err := bm.getSectionSettingsData(goCtx, appID, schema.SectionIDGeneral)
	if err != nil {
		bm.log.Error("failed to get settings", forge.F("error", err.Error()))
		// Return defaults with app name
		return &GeneralSettingsOutput{
			AppName:  appData.Name,
			Timezone: "UTC",
			Language: "en",
		}, nil
	}

	// Build output
	output := &GeneralSettingsOutput{
		AppName:  appData.Name,
		Timezone: "UTC",
		Language: "en",
	}

	// Override with stored settings
	if v, ok := settings["description"].(string); ok {
		output.Description = v
	}
	if v, ok := settings["timezone"].(string); ok {
		output.Timezone = v
	}
	if v, ok := settings["language"].(string); ok {
		output.Language = v
	}
	if v, ok := settings["dateFormat"].(string); ok {
		output.DateFormat = v
	}
	if v, ok := settings["supportEmail"].(string); ok {
		output.SupportEmail = v
	}
	if v, ok := settings["websiteUrl"].(string); ok {
		output.WebsiteUrl = v
	}

	return output, nil
}

// updateGeneralSettings updates general settings
func (bm *BridgeManager) updateGeneralSettings(ctx bridge.Context, input UpdateGeneralSettingsInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Update app name if provided
	if input.AppName != "" {
		updateReq := &app.UpdateAppRequest{
			Name: &input.AppName,
		}
		if _, err := bm.appSvc.UpdateApp(goCtx, appID, updateReq); err != nil {
			bm.log.Error("failed to update app name", forge.F("error", err.Error()))
			return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update app name")
		}
	}

	// Build settings data
	data := make(map[string]interface{})
	if input.Description != "" {
		data["description"] = input.Description
	}
	if input.Timezone != "" {
		data["timezone"] = input.Timezone
	}
	if input.Language != "" {
		data["language"] = input.Language
	}
	if input.DateFormat != "" {
		data["dateFormat"] = input.DateFormat
	}
	if input.SupportEmail != "" {
		data["supportEmail"] = input.SupportEmail
	}
	if input.WebsiteUrl != "" {
		data["websiteUrl"] = input.WebsiteUrl
	}

	// Update section settings
	if len(data) > 0 {
		if err := bm.updateSectionSettingsData(goCtx, appID, schema.SectionIDGeneral, data); err != nil {
			bm.log.Error("failed to update settings", forge.F("error", err.Error()))
			return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update settings")
		}
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "settings.general.updated", "app:"+input.AppID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "General settings updated successfully",
	}, nil
}

// getSecuritySettings retrieves security settings
func (bm *BridgeManager) getSecuritySettings(ctx bridge.Context, input GeneralSettingsInput) (*SecuritySettingsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	settings, err := bm.getSectionSettingsData(goCtx, appID, schema.SectionIDSecurity)
	if err != nil {
		bm.log.Error("failed to get security settings", forge.F("error", err.Error()))
		// Return defaults
		return &SecuritySettingsOutput{
			PasswordMinLength:   8,
			RequireUppercase:    true,
			RequireLowercase:    true,
			RequireNumbers:      true,
			RequireSpecialChars: false,
			MaxLoginAttempts:    5,
			LockoutDuration:     15,
			RequireMFA:          false,
		}, nil
	}

	output := &SecuritySettingsOutput{
		PasswordMinLength:   8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: false,
		MaxLoginAttempts:    5,
		LockoutDuration:     15,
		RequireMFA:          false,
	}

	// Override with stored settings
	if v, ok := toInt(settings["passwordMinLength"]); ok {
		output.PasswordMinLength = v
	}
	if v, ok := settings["requireUppercase"].(bool); ok {
		output.RequireUppercase = v
	}
	if v, ok := settings["requireLowercase"].(bool); ok {
		output.RequireLowercase = v
	}
	if v, ok := settings["requireNumbers"].(bool); ok {
		output.RequireNumbers = v
	}
	if v, ok := settings["requireSpecialChars"].(bool); ok {
		output.RequireSpecialChars = v
	}
	if v, ok := toInt(settings["maxLoginAttempts"]); ok {
		output.MaxLoginAttempts = v
	}
	if v, ok := toInt(settings["lockoutDuration"]); ok {
		output.LockoutDuration = v
	}
	if v, ok := settings["requireMFA"].(bool); ok {
		output.RequireMFA = v
	}
	if v, ok := settings["allowedMFAMethods"].([]interface{}); ok {
		output.AllowedMFAMethods = toStringSlice(v)
	}
	if v, ok := settings["allowedIPAddresses"].([]interface{}); ok {
		output.AllowedIPAddresses = toStringSlice(v)
	}

	return output, nil
}

// updateSecuritySettings updates security settings
func (bm *BridgeManager) updateSecuritySettings(ctx bridge.Context, input UpdateSecuritySettingsInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Build settings data
	data := make(map[string]interface{})
	if input.PasswordMinLength != nil {
		data["passwordMinLength"] = *input.PasswordMinLength
	}
	if input.RequireUppercase != nil {
		data["requireUppercase"] = *input.RequireUppercase
	}
	if input.RequireLowercase != nil {
		data["requireLowercase"] = *input.RequireLowercase
	}
	if input.RequireNumbers != nil {
		data["requireNumbers"] = *input.RequireNumbers
	}
	if input.RequireSpecialChars != nil {
		data["requireSpecialChars"] = *input.RequireSpecialChars
	}
	if input.MaxLoginAttempts != nil {
		data["maxLoginAttempts"] = *input.MaxLoginAttempts
	}
	if input.LockoutDuration != nil {
		data["lockoutDuration"] = *input.LockoutDuration
	}
	if input.RequireMFA != nil {
		data["requireMFA"] = *input.RequireMFA
	}
	if input.AllowedMFAMethods != nil {
		data["allowedMFAMethods"] = input.AllowedMFAMethods
	}
	if input.AllowedIPAddresses != nil {
		data["allowedIPAddresses"] = input.AllowedIPAddresses
	}

	if len(data) > 0 {
		if err := bm.updateSectionSettingsData(goCtx, appID, schema.SectionIDSecurity, data); err != nil {
			bm.log.Error("failed to update security settings", forge.F("error", err.Error()))
			return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update settings")
		}
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "settings.security.updated", "app:"+input.AppID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Security settings updated successfully",
	}, nil
}

// getSessionSettings retrieves session settings
func (bm *BridgeManager) getSessionSettings(ctx bridge.Context, input GeneralSettingsInput) (*SessionSettingsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	settings, err := bm.getSectionSettingsData(goCtx, appID, schema.SectionIDSession)
	if err != nil {
		bm.log.Error("failed to get session settings", forge.F("error", err.Error()))
		// Return defaults
		return &SessionSettingsOutput{
			SessionDuration:       24,
			RefreshTokenDuration:  30,
			IdleTimeout:           0,
			AllowMultipleSessions: true,
			MaxConcurrentSessions: 0,
			RememberMeEnabled:     true,
			RememberMeDuration:    30,
			CookieSameSite:        "lax",
			CookieSecure:          true,
		}, nil
	}

	output := &SessionSettingsOutput{
		SessionDuration:       24,
		RefreshTokenDuration:  30,
		IdleTimeout:           0,
		AllowMultipleSessions: true,
		MaxConcurrentSessions: 0,
		RememberMeEnabled:     true,
		RememberMeDuration:    30,
		CookieSameSite:        "lax",
		CookieSecure:          true,
	}

	// Override with stored settings
	if v, ok := toInt(settings["sessionDuration"]); ok {
		output.SessionDuration = v
	}
	if v, ok := toInt(settings["refreshTokenDuration"]); ok {
		output.RefreshTokenDuration = v
	}
	if v, ok := toInt(settings["idleTimeout"]); ok {
		output.IdleTimeout = v
	}
	if v, ok := settings["allowMultipleSessions"].(bool); ok {
		output.AllowMultipleSessions = v
	}
	if v, ok := toInt(settings["maxConcurrentSessions"]); ok {
		output.MaxConcurrentSessions = v
	}
	if v, ok := settings["rememberMeEnabled"].(bool); ok {
		output.RememberMeEnabled = v
	}
	if v, ok := toInt(settings["rememberMeDuration"]); ok {
		output.RememberMeDuration = v
	}
	if v, ok := settings["cookieSameSite"].(string); ok {
		output.CookieSameSite = v
	}
	if v, ok := settings["cookieSecure"].(bool); ok {
		output.CookieSecure = v
	}

	return output, nil
}

// updateSessionSettings updates session settings
func (bm *BridgeManager) updateSessionSettings(ctx bridge.Context, input UpdateSessionSettingsInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Build settings data
	data := make(map[string]interface{})
	if input.SessionDuration != nil {
		data["sessionDuration"] = *input.SessionDuration
	}
	if input.RefreshTokenDuration != nil {
		data["refreshTokenDuration"] = *input.RefreshTokenDuration
	}
	if input.IdleTimeout != nil {
		data["idleTimeout"] = *input.IdleTimeout
	}
	if input.AllowMultipleSessions != nil {
		data["allowMultipleSessions"] = *input.AllowMultipleSessions
	}
	if input.MaxConcurrentSessions != nil {
		data["maxConcurrentSessions"] = *input.MaxConcurrentSessions
	}
	if input.RememberMeEnabled != nil {
		data["rememberMeEnabled"] = *input.RememberMeEnabled
	}
	if input.RememberMeDuration != nil {
		data["rememberMeDuration"] = *input.RememberMeDuration
	}
	if input.CookieSameSite != nil {
		data["cookieSameSite"] = *input.CookieSameSite
	}
	if input.CookieSecure != nil {
		data["cookieSecure"] = *input.CookieSecure
	}

	if len(data) > 0 {
		if err := bm.updateSectionSettingsData(goCtx, appID, schema.SectionIDSession, data); err != nil {
			bm.log.Error("failed to update session settings", forge.F("error", err.Error()))
			return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update settings")
		}
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "settings.session.updated", "app:"+input.AppID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Session settings updated successfully",
	}, nil
}

// getSectionSettings retrieves settings for any section
func (bm *BridgeManager) getSectionSettings(ctx bridge.Context, input SectionSettingsInput) (*SectionSettingsOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}
	if input.SectionID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "sectionId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	settings, err := bm.getSectionSettingsData(goCtx, appID, input.SectionID)
	if err != nil {
		bm.log.Error("failed to get section settings", forge.F("error", err.Error()), forge.F("sectionId", input.SectionID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to load settings")
	}

	return &SectionSettingsOutput{
		SectionID: input.SectionID,
		Data:      settings,
	}, nil
}

// updateSectionSettings updates settings for any section
func (bm *BridgeManager) updateSectionSettings(ctx bridge.Context, input UpdateSectionSettingsInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}
	if input.SectionID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "sectionId is required")
	}
	if input.Data == nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "data is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx, appID)

	if err := bm.updateSectionSettingsData(goCtx, appID, input.SectionID, input.Data); err != nil {
		bm.log.Error("failed to update section settings", forge.F("error", err.Error()), forge.F("sectionId", input.SectionID))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update settings: "+err.Error())
	}

	// Log audit event
	if bm.auditSvc != nil {
		_ = bm.auditSvc.Log(goCtx, nil, "settings."+input.SectionID+".updated", "app:"+input.AppID, "", "", "")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: fmt.Sprintf("%s settings updated successfully", input.SectionID),
	}, nil
}

// getSettingsSchema returns the settings schema for rendering forms
func (bm *BridgeManager) getSettingsSchema(ctx bridge.Context, input GeneralSettingsInput) (*SchemaOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Get the global schema (includes all registered sections)
	settingsSchema := schema.GetGlobalSchema(schema.DefaultAppSettingsSchemaID, schema.DefaultAppSettingsSchemaName)

	return &SchemaOutput{
		Schema: settingsSchema,
	}, nil
}

// Helper functions

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case json.Number:
		i, err := val.Int64()
		if err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func toStringSlice(v []interface{}) []string {
	result := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// Note: API key functions (getAPIKeys, createAPIKey, revokeAPIKey) have been moved to apikeys.go
