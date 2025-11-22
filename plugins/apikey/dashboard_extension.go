package apikey

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the API key plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
}

// NewDashboardExtension creates a new dashboard extension for API keys
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "apikey"
}

// NavigationItems returns navigation items to register (none for settings-only plugin)
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{} // Using settings pages only
}

// Routes returns routes to register under /dashboard/app/:appId/
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// API Keys Management Page
		{
			Method:       "GET",
			Path:         "/settings/api-keys",
			Handler:      e.ServeAPIKeysListPage,
			Name:         "dashboard.settings.api-keys",
			Summary:      "API keys management page",
			Description:  "View and manage API keys for programmatic access",
			Tags:         []string{"Dashboard", "Settings", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create API Key
		{
			Method:       "POST",
			Path:         "/settings/api-keys/create",
			Handler:      e.CreateAPIKey,
			Name:         "dashboard.settings.api-keys.create",
			Summary:      "Create new API key",
			Description:  "Create a new API key with specified scopes and permissions",
			Tags:         []string{"Dashboard", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Rotate API Key
		{
			Method:       "POST",
			Path:         "/settings/api-keys/rotate/:keyId",
			Handler:      e.RotateAPIKey,
			Name:         "dashboard.settings.api-keys.rotate",
			Summary:      "Rotate API key",
			Description:  "Rotate an existing API key",
			Tags:         []string{"Dashboard", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Revoke API Key
		{
			Method:       "POST",
			Path:         "/settings/api-keys/revoke",
			Handler:      e.RevokeAPIKey,
			Name:         "dashboard.settings.api-keys.revoke",
			Summary:      "Revoke API key",
			Description:  "Revoke an API key",
			Tags:         []string{"Dashboard", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Configuration Page
		{
			Method:       "GET",
			Path:         "/settings/api-keys-config",
			Handler:      e.ServeAPIKeysConfigPage,
			Name:         "dashboard.settings.api-keys-config",
			Summary:      "API keys configuration page",
			Description:  "Configure API key defaults and limits",
			Tags:         []string{"Dashboard", "Settings", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Configuration
		{
			Method:       "POST",
			Path:         "/settings/api-keys-config/update",
			Handler:      e.UpdateConfig,
			Name:         "dashboard.settings.api-keys-config.update",
			Summary:      "Update API keys configuration",
			Description:  "Update API key configuration settings",
			Tags:         []string{"Dashboard", "API Keys"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Security Page
		{
			Method:       "GET",
			Path:         "/settings/api-keys-security",
			Handler:      e.ServeAPIKeysSecurityPage,
			Name:         "dashboard.settings.api-keys-security",
			Summary:      "API keys security page",
			Description:  "Configure API key security settings",
			Tags:         []string{"Dashboard", "Settings", "API Keys", "Security"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Security
		{
			Method:       "POST",
			Path:         "/settings/api-keys-security/update",
			Handler:      e.UpdateSecurity,
			Name:         "dashboard.settings.api-keys-security.update",
			Summary:      "Update API keys security",
			Description:  "Update API key security settings",
			Tags:         []string{"Dashboard", "API Keys", "Security"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated, using SettingsPages instead)
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{} // Using SettingsPages instead
}

// SettingsPages returns full settings pages for the sidebar layout
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "api-keys",
			Label:         "API Keys",
			Description:   "Manage API keys for programmatic access",
			Icon:          lucide.Key(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         10,
			Path:          "api-keys",
			RequirePlugin: "apikey",
			RequireAdmin:  true,
		},
		{
			ID:            "api-keys-config",
			Label:         "API Key Configuration",
			Description:   "Configure API key defaults and limits",
			Icon:          lucide.Settings(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         11,
			Path:          "api-keys-config",
			RequirePlugin: "apikey",
			RequireAdmin:  true,
		},
		{
			ID:            "api-keys-security",
			Label:         "API Key Security",
			Description:   "Configure API key security settings",
			Icon:          lucide.Shield(Class("h-5 w-5")),
			Category:      "security",
			Order:         25,
			Path:          "api-keys-security",
			RequirePlugin: "apikey",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "apikey-stats",
			Title: "API Keys",
			Icon: lucide.Key(
				Class("size-5"),
			),
			Order: 40,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// ServeAPIKeysListPage renders the API keys management page
func (e *DashboardExtension) ServeAPIKeysListPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	pageData := components.PageData{
		Title:      "API Keys",
		User:       currentUser,
		ActivePage: "settings-api-keys",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderAPIKeysListContent(c, currentApp, currentUser)
	return handler.RenderWithLayout(c, pageData, content)
}

// ServeAPIKeysConfigPage renders the configuration page
func (e *DashboardExtension) ServeAPIKeysConfigPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	pageData := components.PageData{
		Title:      "API Key Configuration",
		User:       currentUser,
		ActivePage: "settings-api-keys-config",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderConfigContent(c, currentApp)
	return handler.RenderWithLayout(c, pageData, content)
}

// ServeAPIKeysSecurityPage renders the security settings page
func (e *DashboardExtension) ServeAPIKeysSecurityPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	pageData := components.PageData{
		Title:      "API Key Security",
		User:       currentUser,
		ActivePage: "settings-api-keys-security",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderSecurityContent(c, currentApp)
	return handler.RenderWithLayout(c, pageData, content)
}

// CreateAPIKey handles API key creation
func (e *DashboardExtension) CreateAPIKey(c forge.Context) error {
	ctx := c.Request().Context()

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app context",
		})
	}

	// Get current user
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	// Get handler to access current environment
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Dashboard handler not available",
		})
	}

	// Get current environment from handler
	currentEnv, err := handler.GetCurrentEnvironment(c, currentApp.ID)
	if err != nil || currentEnv == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid environment context",
		})
	}

	appID := currentApp.ID
	envID := currentEnv.ID
	userID := currentUser.ID

	// Parse form data
	name := c.FormValue("name")
	keyTypeStr := c.FormValue("key_type")
	scopesStr := c.FormValue("scopes")
	rateLimitStr := c.FormValue("rate_limit")
	expiresInStr := c.FormValue("expires_in")

	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	// Parse key type
	var keyType apikey.KeyType
	switch keyTypeStr {
	case "pk":
		keyType = apikey.KeyTypePublishable
	case "sk":
		keyType = apikey.KeyTypeSecret
	case "rk":
		keyType = apikey.KeyTypeRestricted
	default:
		keyType = apikey.KeyTypeRestricted // Default to restricted if not specified
	}

	// Parse scopes
	var scopes []string
	if scopesStr != "" {
		scopes = strings.Split(strings.ReplaceAll(scopesStr, " ", ""), ",")
	}
	if len(scopes) == 0 {
		// Set default scopes based on key type
		switch keyType {
		case apikey.KeyTypePublishable:
			scopes = []string{"app:identify", "sessions:create", "users:verify"}
		case apikey.KeyTypeSecret:
			scopes = []string{"admin:full"}
		default:
			scopes = []string{"read"} // Default scope for restricted keys
		}
	}

	// Parse rate limit
	rateLimit := e.plugin.config.DefaultRateLimit
	if rateLimitStr != "" {
		if rl, err := strconv.Atoi(rateLimitStr); err == nil {
			rateLimit = rl
		}
	}

	// Parse expiry
	var expiresAt *time.Time
	if expiresInStr != "" {
		if days, err := strconv.Atoi(expiresInStr); err == nil && days > 0 {
			expiry := time.Now().AddDate(0, 0, days)
			expiresAt = &expiry
		}
	}

	// Create API key
	req := &apikey.CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envID,
		UserID:        userID,
		Name:          name,
		KeyType:       keyType,
		Scopes:        scopes,
		RateLimit:     rateLimit,
		ExpiresAt:     expiresAt,
		Permissions:   make(map[string]string),
		Metadata:      make(map[string]string),
	}

	key, err := e.plugin.service.CreateAPIKey(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create API key: %v", err),
		})
	}

	// Return success with the new key
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"key":     key.Key, // Full key only shown once
		"message": "API key created successfully. Save it securely - it won't be shown again!",
	})
}

// RotateAPIKey handles API key rotation
func (e *DashboardExtension) RotateAPIKey(c forge.Context) error {
	ctx := c.Request().Context()
	keyID := c.Param("keyId")

	if keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Key ID is required",
		})
	}

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Get handler to access current environment
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	// Get current environment
	currentEnv, err := handler.GetCurrentEnvironment(c, currentApp.ID)
	if err != nil || currentEnv == nil {
		return c.String(http.StatusBadRequest, "Invalid environment context")
	}

	appID := currentApp.ID
	envID := currentEnv.ID

	// Get current user
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}
	userID := currentUser.ID

	parsedKeyID, err := xid.FromString(keyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid key ID",
		})
	}

	req := &apikey.RotateAPIKeyRequest{
		ID:            parsedKeyID,
		AppID:         appID,
		EnvironmentID: envID,
		UserID:        userID,
	}

	newKey, err := e.plugin.service.RotateAPIKey(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to rotate API key: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"key":     newKey.Key,
		"message": "API key rotated successfully",
	})
}

// RevokeAPIKey handles API key revocation
func (e *DashboardExtension) RevokeAPIKey(c forge.Context) error {
	ctx := c.Request().Context()
	keyID := c.FormValue("key_id")

	if keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Key ID is required",
		})
	}

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app context",
		})
	}

	// Get handler to access current environment
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Dashboard handler not available",
		})
	}

	// Get current environment
	currentEnv, err := handler.GetCurrentEnvironment(c, currentApp.ID)
	if err != nil || currentEnv == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid environment context",
		})
	}

	// Get current user
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	appID := currentApp.ID
	envID := currentEnv.ID
	userID := currentUser.ID

	parsedKeyID, err := xid.FromString(keyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid key ID",
		})
	}

	var orgIDPtr *xid.ID
	if !envID.IsNil() {
		orgIDPtr = &envID // Using envID as placeholder, adjust if org logic is different
	}
	err = e.plugin.service.DeleteAPIKey(ctx, appID, parsedKeyID, userID, orgIDPtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to revoke API key: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "API key revoked successfully",
	})
}

// UpdateConfig handles configuration updates
func (e *DashboardExtension) UpdateConfig(c forge.Context) error {
	// Parse form data
	defaultRateLimit, _ := strconv.Atoi(c.FormValue("default_rate_limit"))
	maxRateLimit, _ := strconv.Atoi(c.FormValue("max_rate_limit"))
	maxKeysPerUser, _ := strconv.Atoi(c.FormValue("max_keys_per_user"))
	maxKeysPerOrg, _ := strconv.Atoi(c.FormValue("max_keys_per_org"))
	keyLength, _ := strconv.Atoi(c.FormValue("key_length"))
	autoCleanupEnabled := c.FormValue("auto_cleanup_enabled") == "on"
	cleanupIntervalHours, _ := strconv.Atoi(c.FormValue("cleanup_interval_hours"))

	// Update plugin config (in production, this should persist to config store)
	e.plugin.config.DefaultRateLimit = defaultRateLimit
	e.plugin.config.MaxRateLimit = maxRateLimit
	e.plugin.config.MaxKeysPerUser = maxKeysPerUser
	e.plugin.config.MaxKeysPerOrg = maxKeysPerOrg
	e.plugin.config.KeyLength = keyLength
	e.plugin.config.AutoCleanup.Enabled = autoCleanupEnabled
	e.plugin.config.AutoCleanup.Interval = time.Duration(cleanupIntervalHours) * time.Hour

	// Restart cleanup scheduler if settings changed
	if autoCleanupEnabled {
		e.plugin.StopCleanupScheduler()
		e.plugin.startCleanupScheduler()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
	})
}

// UpdateSecurity handles security settings updates
func (e *DashboardExtension) UpdateSecurity(c forge.Context) error {
	// Parse form data
	allowQueryParam := c.FormValue("allow_query_param") == "on"
	rateLimitingEnabled := c.FormValue("rate_limiting_enabled") == "on"
	ipWhitelistingEnabled := c.FormValue("ip_whitelisting_enabled") == "on"

	// Update plugin config
	e.plugin.config.AllowQueryParam = allowQueryParam
	e.plugin.config.RateLimiting.Enabled = rateLimitingEnabled
	e.plugin.config.IPWhitelisting.Enabled = ipWhitelistingEnabled

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Security settings updated successfully",
	})
}

// RenderDashboardWidget renders the API key stats widget
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	if currentApp == nil {
		return Div(Class("text-gray-500"), g.Text("No app context"))
	}

	// Fetch stats (using a simple approach, could be enhanced with caching)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, _ := e.getKeyStats(ctx, currentApp.ID)

	return e.renderKeyStatsWidget(stats)
}

// Helper methods

func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil
	}
	return handler.GetUserFromContext(c)
}

func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil, fmt.Errorf("handler not available")
	}
	return handler.GetCurrentApp(c)
}

// KeyStats holds API key statistics
type KeyStats struct {
	TotalActive     int
	UsedLast24h     int
	AvgRequestRate  float64
	ExpiringSoon    int
	TotalRevoked    int
	MostUsedKeyName string
}

// getKeyStats fetches API key statistics for the app
func (e *DashboardExtension) getKeyStats(ctx context.Context, appID xid.ID) (KeyStats, error) {
	stats := KeyStats{}

	// List all keys for the app
	filter := &apikey.ListAPIKeysFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  1000,
			Offset: 0,
		},
		AppID: appID,
	}

	keysResp, err := e.plugin.service.ListAPIKeys(ctx, filter)
	if err != nil {
		return stats, err
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	weekFromNow := now.AddDate(0, 0, 7)

	maxUsageCount := int64(0)

	for _, key := range keysResp.Data {
		if key.Active {
			stats.TotalActive++

			// Check if used in last 24h
			if key.LastUsedAt != nil && key.LastUsedAt.After(yesterday) {
				stats.UsedLast24h++
			}

			// Check if expiring soon
			if key.ExpiresAt != nil && key.ExpiresAt.Before(weekFromNow) && key.ExpiresAt.After(now) {
				stats.ExpiringSoon++
			}

			// Track most used key
			if key.UsageCount > maxUsageCount {
				maxUsageCount = key.UsageCount
				stats.MostUsedKeyName = key.Name
			}
		} else {
			stats.TotalRevoked++
		}
	}

	// Calculate average request rate (simplified)
	if stats.TotalActive > 0 {
		totalRequests := int64(0)
		for _, key := range keysResp.Data {
			if key.Active {
				totalRequests += key.UsageCount
			}
		}
		stats.AvgRequestRate = float64(totalRequests) / float64(stats.TotalActive)
	}

	return stats, nil
}
