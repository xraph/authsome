package apikey

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the API key plugin to add its own screens to the dashboard.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension for API keys.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui",
	}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration).
func (e *DashboardExtension) SetRegistry(registry any) {
	// No longer needed - layout handled by ForgeUI
}

// getBasePath returns the dashboard base path.
func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "apikey"
}

// NavigationItems returns navigation items to register (none for settings-only plugin).
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{} // Using settings pages only
}

// Routes returns routes to register under /dashboard/app/:appId/.
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

// SettingsSections returns settings sections (deprecated, using SettingsPages instead).
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{} // Using SettingsPages instead
}

// SettingsPages returns full settings pages for the sidebar layout.
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

// DashboardWidgets returns widgets to show on the main dashboard.
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

// BridgeFunctions returns bridge functions for the apikey plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	// No bridge functions for this plugin yet
	return nil
}

// ServeAPIKeysListPage renders the API keys management page.
func (e *DashboardExtension) ServeAPIKeysListPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get current user from context
	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		return nil, errs.Unauthorized()
	}

	return e.renderAPIKeysListContent(ctx.Request, currentApp, currentUser), nil
}

// ServeAPIKeysConfigPage renders the configuration page.
func (e *DashboardExtension) ServeAPIKeysConfigPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	return e.renderConfigContent(currentApp), nil
}

// ServeAPIKeysSecurityPage renders the security settings page.
func (e *DashboardExtension) ServeAPIKeysSecurityPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	return e.renderSecurityContent(currentApp), nil
}

// CreateAPIKey handles API key creation.
func (e *DashboardExtension) CreateAPIKey(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	w := ctx.ResponseWriter

	// Helper to write JSON response
	writeJSON := func(status int, data map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(data)
	}

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		writeJSON(400, map[string]any{"success": false, "error": "Invalid request"})

		return nil, nil
	}

	appID := currentApp.ID

	// Parse form data
	name := ctx.Request.FormValue("name")
	keyTypeStr := ctx.Request.FormValue("key_type")
	scopesStr := ctx.Request.FormValue("scopes")
	rateLimitStr := ctx.Request.FormValue("rate_limit")
	expiresInStr := ctx.Request.FormValue("expires_in")

	if name == "" {
		writeJSON(400, map[string]any{"success": false, "error": "Name is required"})

		return nil, nil
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
		EnvironmentID: xid.NilID(), // Environment ID if needed
		UserID:        xid.NilID(), // User ID if needed
		Name:          name,
		KeyType:       keyType,
		Scopes:        scopes,
		RateLimit:     rateLimit,
		ExpiresAt:     expiresAt,
		Permissions:   make(map[string]string),
		Metadata:      make(map[string]string),
	}

	key, err := e.plugin.service.CreateAPIKey(reqCtx, req)
	if err != nil {
		writeJSON(500, map[string]any{"success": false, "error": "Failed to create API key: " + err.Error()})

		return nil, nil
	}

	// Return success with the new key
	writeJSON(200, map[string]any{
		"success": true,
		"key":     key.Key, // The actual API key value
		"id":      key.ID.String(),
		"name":    key.Name,
	})

	return nil, nil
}

// RotateAPIKey handles API key rotation.
func (e *DashboardExtension) RotateAPIKey(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	w := ctx.ResponseWriter
	keyID := ctx.Param("keyId")

	// Helper to write JSON response
	writeJSON := func(status int, data map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(data)
	}

	if keyID == "" {
		writeJSON(400, map[string]any{"success": false, "error": "Key ID is required"})

		return nil, nil
	}

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		writeJSON(400, map[string]any{"success": false, "error": "Invalid app context"})

		return nil, nil
	}

	appID := currentApp.ID

	parsedKeyID, err := xid.FromString(keyID)
	if err != nil {
		writeJSON(400, map[string]any{"success": false, "error": "Invalid key ID format"})

		return nil, nil
	}

	req := &apikey.RotateAPIKeyRequest{
		ID:            parsedKeyID,
		AppID:         appID,
		EnvironmentID: xid.NilID(),
		UserID:        xid.NilID(),
	}

	key, err := e.plugin.service.RotateAPIKey(reqCtx, req)
	if err != nil {
		writeJSON(500, map[string]any{"success": false, "error": "Failed to rotate API key: " + err.Error()})

		return nil, nil
	}

	writeJSON(200, map[string]any{
		"success": true,
		"key":     key.Key,
		"id":      key.ID.String(),
	})

	return nil, nil
}

// RevokeAPIKey handles API key revocation.
func (e *DashboardExtension) RevokeAPIKey(ctx *router.PageContext) (g.Node, error) {
	reqCtx := ctx.Request.Context()
	w := ctx.ResponseWriter
	keyID := ctx.Request.FormValue("key_id")

	// Helper to write JSON response
	writeJSON := func(status int, data map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(data)
	}

	if keyID == "" {
		writeJSON(400, map[string]any{"success": false, "error": "Key ID is required"})

		return nil, nil
	}

	// Extract app from URL
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		writeJSON(400, map[string]any{"success": false, "error": "Invalid app context"})

		return nil, nil
	}

	appID := currentApp.ID

	parsedKeyID, err := xid.FromString(keyID)
	if err != nil {
		writeJSON(400, map[string]any{"success": false, "error": "Invalid key ID format"})

		return nil, nil
	}

	// Delete the API key
	err = e.plugin.service.DeleteAPIKey(reqCtx, appID, parsedKeyID, xid.NilID(), nil)
	if err != nil {
		writeJSON(500, map[string]any{"success": false, "error": "Failed to revoke API key: " + err.Error()})

		return nil, nil
	}

	writeJSON(200, map[string]any{"success": true, "message": "API key revoked successfully"})

	return nil, nil
}

// UpdateConfig handles configuration updates.
func (e *DashboardExtension) UpdateConfig(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	defaultRateLimit, _ := strconv.Atoi(ctx.Request.FormValue("default_rate_limit"))
	maxRateLimit, _ := strconv.Atoi(ctx.Request.FormValue("max_rate_limit"))
	maxKeysPerUser, _ := strconv.Atoi(ctx.Request.FormValue("max_keys_per_user"))
	maxKeysPerOrg, _ := strconv.Atoi(ctx.Request.FormValue("max_keys_per_org"))
	keyLength, _ := strconv.Atoi(ctx.Request.FormValue("key_length"))
	autoCleanupEnabled := ctx.Request.FormValue("auto_cleanup_enabled") == "on"
	cleanupIntervalHours, _ := strconv.Atoi(ctx.Request.FormValue("cleanup_interval_hours"))

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

	return nil, nil // Success
}

// UpdateSecurity handles security settings updates.
func (e *DashboardExtension) UpdateSecurity(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	allowQueryParam := ctx.Request.FormValue("allow_query_param") == "on"
	rateLimitingEnabled := ctx.Request.FormValue("rate_limiting_enabled") == "on"
	ipWhitelistingEnabled := ctx.Request.FormValue("ip_whitelisting_enabled") == "on"

	// Update plugin config
	e.plugin.config.AllowQueryParam = allowQueryParam
	e.plugin.config.RateLimiting.Enabled = rateLimitingEnabled
	e.plugin.config.IPWhitelisting.Enabled = ipWhitelistingEnabled

	return nil, nil // Success
}

// RenderDashboardWidget renders the API key stats widget.
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

func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) *user.User {
	// First try to get from PageContext (set by ForgeUI router)
	if userVal, exists := ctx.Get("user"); exists && userVal != nil {
		if u, ok := userVal.(*user.User); ok {
			return u
		}
	}

	// Fallback: try request context
	reqCtx := ctx.Request.Context()
	if u, ok := reqCtx.Value(contexts.UserContextKey).(*user.User); ok {
		return u
	}

	return nil
}

func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	// First try to extract app from request context (set by middleware)
	reqCtx := ctx.Request.Context()

	appVal := reqCtx.Value(contexts.AppContextKey)
	if appVal != nil {
		if currentApp, ok := appVal.(*app.App); ok {
			return currentApp, nil
		}
	}

	// Fallback: try to get from PageContext (set by ForgeUI router)
	if pageAppVal, exists := ctx.Get("currentApp"); exists && pageAppVal != nil {
		if currentApp, ok := pageAppVal.(*app.App); ok {
			return currentApp, nil
		}
	}

	// Final fallback: parse app ID from URL and create minimal app
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.RequiredField("app_id")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	// Return minimal app with ID - the dashboard handler will enrich it
	return &app.App{ID: appID}, nil
}

// KeyStats holds API key statistics.
type KeyStats struct {
	TotalActive     int
	UsedLast24h     int
	AvgRequestRate  float64
	ExpiringSoon    int
	TotalRevoked    int
	MostUsedKeyName string
}

// getKeyStats fetches API key statistics for the app.
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
