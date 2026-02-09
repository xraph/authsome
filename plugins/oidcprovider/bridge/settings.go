package bridge

import (
	"encoding/json"
	"time"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Input/Output Types
// =============================================================================

// GetSettingsInput is the input for getting OIDC settings.
type GetSettingsInput struct {
	AppID string `json:"appId"`
}

// GetSettingsOutput is the output for getting OIDC settings.
type GetSettingsOutput struct {
	Data SettingsDTO `json:"data"`
}

// SettingsDTO represents OIDC provider configuration.
type SettingsDTO struct {
	Issuer        string           `json:"issuer"`
	DiscoveryURL  string           `json:"discoveryUrl"`
	JWKSURL       string           `json:"jwksUrl"`
	TokenSettings TokenSettingsDTO `json:"tokenSettings"`
	KeySettings   KeySettingsDTO   `json:"keySettings"`
	DeviceFlow    DeviceFlowDTO    `json:"deviceFlow"`
}

// TokenSettingsDTO represents token lifetime settings.
type TokenSettingsDTO struct {
	AccessTokenExpiry  string `json:"accessTokenExpiry"`  // Duration string (e.g., "1h")
	IDTokenExpiry      string `json:"idTokenExpiry"`      // Duration string
	RefreshTokenExpiry string `json:"refreshTokenExpiry"` // Duration string
}

// KeySettingsDTO represents key management settings.
type KeySettingsDTO struct {
	RotationInterval string `json:"rotationInterval"` // Duration string
	KeyLifetime      string `json:"keyLifetime"`      // Duration string
	LastRotation     string `json:"lastRotation"`     // Timestamp
	CurrentKeyID     string `json:"currentKeyId"`
}

// DeviceFlowDTO represents device flow configuration.
type DeviceFlowDTO struct {
	Enabled         bool   `json:"enabled"`
	CodeExpiry      string `json:"codeExpiry"` // Duration string
	UserCodeLength  int    `json:"userCodeLength"`
	UserCodeFormat  string `json:"userCodeFormat"`
	PollingInterval int    `json:"pollingInterval"` // Seconds
	VerificationURI string `json:"verificationUri"`
	MaxPollAttempts int    `json:"maxPollAttempts"`
	CleanupInterval string `json:"cleanupInterval"` // Duration string
}

// UpdateTokenSettingsInput is the input for updating token settings.
type UpdateTokenSettingsInput struct {
	AccessTokenExpiry  string `json:"accessTokenExpiry,omitempty"`
	IDTokenExpiry      string `json:"idTokenExpiry,omitempty"`
	RefreshTokenExpiry string `json:"refreshTokenExpiry,omitempty"`
}

// UpdateTokenSettingsOutput is the output for updating token settings.
type UpdateTokenSettingsOutput struct {
	Success bool `json:"success"`
}

// UpdateDeviceFlowSettingsInput is the input for updating device flow settings.
type UpdateDeviceFlowSettingsInput struct {
	Enabled         bool   `json:"enabled,omitempty"`
	CodeExpiry      string `json:"codeExpiry,omitempty"`
	UserCodeLength  int    `json:"userCodeLength,omitempty"`
	UserCodeFormat  string `json:"userCodeFormat,omitempty"`
	PollingInterval int    `json:"pollingInterval,omitempty"`
	VerificationURI string `json:"verificationUri,omitempty"`
	MaxPollAttempts int    `json:"maxPollAttempts,omitempty"`
	CleanupInterval string `json:"cleanupInterval,omitempty"`
}

// UpdateDeviceFlowSettingsOutput is the output for updating device flow settings.
type UpdateDeviceFlowSettingsOutput struct {
	Success bool `json:"success"`
}

// RotateKeysInput is the input for rotating JWT keys.
type RotateKeysInput struct{}

// RotateKeysOutput is the output for rotating JWT keys.
type RotateKeysOutput struct {
	Success  bool   `json:"success"`
	NewKeyID string `json:"newKeyId"`
}

// =============================================================================
// Bridge Functions
// =============================================================================

// GetSettings retrieves current OIDC provider configuration.
func (bm *BridgeManager) GetSettings(ctx bridge.Context, input GetSettingsInput) (*GetSettingsOutput, error) {
	_, _, _, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get service config (type assert to expected structure)
	serviceConfigIface := bm.service.GetConfig()

	// Type assert to extract config values
	// Using reflection/interface{} to access config without importing parent package
	configMap := convertConfigToMap(serviceConfigIface)

	// Get current key info
	keyID, _ := bm.service.GetCurrentKeyID()
	lastRotation := bm.service.GetLastKeyRotation()

	// Extract config values from map (using lowercase JSON keys)
	issuer := getString(configMap, "issuer")
	tokensMap := getMap(configMap, "tokens")
	keysMap := getMap(configMap, "keys")
	deviceFlowMap := getMap(configMap, "deviceFlow")

	settings := SettingsDTO{
		Issuer:       issuer,
		DiscoveryURL: issuer + "/.well-known/openid-configuration",
		JWKSURL:      issuer + "/oauth2/jwks",
		TokenSettings: TokenSettingsDTO{
			AccessTokenExpiry:  getString(tokensMap, "accessTokenExpiry"),
			IDTokenExpiry:      getString(tokensMap, "idTokenExpiry"),
			RefreshTokenExpiry: getString(tokensMap, "refreshTokenExpiry"),
		},
		KeySettings: KeySettingsDTO{
			RotationInterval: getString(keysMap, "rotationInterval"),
			KeyLifetime:      getString(keysMap, "keyLifetime"),
			LastRotation:     lastRotation.Format(time.RFC3339),
			CurrentKeyID:     keyID,
		},
		DeviceFlow: DeviceFlowDTO{
			Enabled:         getBool(deviceFlowMap, "enabled"),
			CodeExpiry:      getString(deviceFlowMap, "codeExpiry"),
			UserCodeLength:  getInt(deviceFlowMap, "userCodeLength"),
			UserCodeFormat:  getString(deviceFlowMap, "userCodeFormat"),
			PollingInterval: getInt(deviceFlowMap, "pollingInterval"),
			VerificationURI: getString(deviceFlowMap, "verificationUri"),
			MaxPollAttempts: getInt(deviceFlowMap, "maxPollAttempts"),
			CleanupInterval: getString(deviceFlowMap, "cleanupInterval"),
		},
	}

	return &GetSettingsOutput{
		Data: settings,
	}, nil
}

// UpdateTokenSettings updates token lifetime configuration.
func (bm *BridgeManager) UpdateTokenSettings(ctx bridge.Context, input UpdateTokenSettingsInput) (*UpdateTokenSettingsOutput, error) {
	_, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate duration strings
	if err := validateDuration(input.AccessTokenExpiry); err != nil {
		return nil, errs.BadRequest("invalid accessTokenExpiry: " + err.Error())
	}

	if err := validateDuration(input.IDTokenExpiry); err != nil {
		return nil, errs.BadRequest("invalid idTokenExpiry: " + err.Error())
	}

	if err := validateDuration(input.RefreshTokenExpiry); err != nil {
		return nil, errs.BadRequest("invalid refreshTokenExpiry: " + err.Error())
	}

	// Update configuration
	// Note: In a real implementation, you'd persist this to a config store
	// For now, we'll just log the change
	bm.logger.Info("token settings updated",
		forge.F("accessTokenExpiry", input.AccessTokenExpiry),
		forge.F("idTokenExpiry", input.IDTokenExpiry),
		forge.F("refreshTokenExpiry", input.RefreshTokenExpiry))

	// TODO: Implement config persistence
	// This would involve:
	// 1. Updating the config in the database or config store
	// 2. Reloading the service configuration
	// 3. Possibly triggering a service restart or hot reload

	return &UpdateTokenSettingsOutput{
		Success: true,
	}, nil
}

// UpdateDeviceFlowSettings updates device flow configuration.
func (bm *BridgeManager) UpdateDeviceFlowSettings(ctx bridge.Context, input UpdateDeviceFlowSettingsInput) (*UpdateDeviceFlowSettingsOutput, error) {
	_, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate duration strings
	if err := validateDuration(input.CodeExpiry); err != nil {
		return nil, errs.BadRequest("invalid codeExpiry: " + err.Error())
	}

	if err := validateDuration(input.CleanupInterval); err != nil {
		return nil, errs.BadRequest("invalid cleanupInterval: " + err.Error())
	}

	// Validate numeric values
	if input.UserCodeLength < 4 || input.UserCodeLength > 20 {
		return nil, errs.BadRequest("userCodeLength must be between 4 and 20")
	}

	if input.PollingInterval < 1 || input.PollingInterval > 60 {
		return nil, errs.BadRequest("pollingInterval must be between 1 and 60 seconds")
	}

	if input.MaxPollAttempts < 10 || input.MaxPollAttempts > 1000 {
		return nil, errs.BadRequest("maxPollAttempts must be between 10 and 1000")
	}

	// Update configuration
	bm.logger.Info("device flow settings updated",
		forge.F("enabled", input.Enabled),
		forge.F("codeExpiry", input.CodeExpiry),
		forge.F("userCodeLength", input.UserCodeLength))

	// TODO: Implement config persistence
	// This would involve:
	// 1. Updating the config in the database or config store
	// 2. Reloading the service configuration
	// 3. If enabling/disabling, start/stop the device flow service

	return &UpdateDeviceFlowSettingsOutput{
		Success: true,
	}, nil
}

// RotateKeys triggers a manual JWT key rotation.
func (bm *BridgeManager) RotateKeys(ctx bridge.Context, input RotateKeysInput) (*RotateKeysOutput, error) {
	_, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Trigger key rotation
	if err := bm.service.RotateKeys(); err != nil {
		bm.logger.Error("failed to rotate JWT keys",
			forge.F("error", err.Error()))

		return nil, errs.InternalServerError("failed to rotate keys", err)
	}

	// Get new key ID
	newKeyID, _ := bm.service.GetCurrentKeyID()

	bm.logger.Info("JWT keys rotated manually",
		forge.F("newKeyId", newKeyID))

	return &RotateKeysOutput{
		Success:  true,
		NewKeyID: newKeyID,
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// validateDuration validates a duration string.
func validateDuration(s string) error {
	_, err := time.ParseDuration(s)

	return err
}

// Helper functions for safe config value extraction using reflection.
func convertConfigToMap(config any) map[string]any {
	// Convert to map using JSON marshaling
	data, err := json.Marshal(config)
	if err != nil {
		return make(map[string]any)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return make(map[string]any)
	}

	return result
}

func getString(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}

func getBool(m map[string]any, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}

	return false
}

func getInt(m map[string]any, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}

	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if val, ok := m[key]; ok {
		if subMap, ok := val.(map[string]any); ok {
			return subMap
		}
	}

	return make(map[string]any)
}
