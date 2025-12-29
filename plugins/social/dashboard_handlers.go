package social

import (
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// parseXID parses an xid string
func parseXID(s string) (xid.ID, error) {
	return xid.FromString(s)
}

// HandleCreateProvider creates a new social provider configuration
func (e *DashboardExtension) HandleCreateProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	envID, err := e.getCurrentEnvironmentID(c, currentApp.ID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment context")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()
	redirectURL := basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/social"

	// Parse form values
	providerName := c.FormValue("provider_name")
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")
	customRedirectURL := c.FormValue("redirect_url")
	scopesStr := c.FormValue("scopes")
	displayName := c.FormValue("display_name")
	isEnabled := c.FormValue("is_enabled") == "true"

	// Validate required fields
	if providerName == "" || clientID == "" || clientSecret == "" {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Missing+required+fields")
	}

	// Validate provider name
	if !schema.IsValidProvider(providerName) {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Invalid+provider+name")
	}

	// Check if provider already exists for this environment
	exists, err := e.configRepo.ExistsByProvider(ctx, currentApp.ID, envID, providerName)
	if err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Failed+to+check+existing+provider")
	}
	if exists {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Provider+already+configured")
	}

	// Parse scopes
	var scopes []string
	if scopesStr != "" {
		scopes = strings.Fields(scopesStr)
	}

	// Create config
	config := &schema.SocialProviderConfig{
		AppID:         currentApp.ID,
		EnvironmentID: envID,
		ProviderName:  providerName,
		ClientID:      clientID,
		ClientSecret:  clientSecret, // TODO: Encrypt before storing
		RedirectURL:   customRedirectURL,
		Scopes:        scopes,
		IsEnabled:     isEnabled,
		DisplayName:   displayName,
	}

	if err := e.configRepo.Create(ctx, config); err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Failed+to+create+provider")
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(currentApp.ID, envID)
	}

	return c.Redirect(http.StatusFound, redirectURL+"?success=Provider+created+successfully")
}

// HandleUpdateProvider updates an existing social provider configuration
func (e *DashboardExtension) HandleUpdateProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	configID, err := parseXID(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid provider ID")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()
	redirectURL := basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/social"

	// Get existing config
	config, err := e.configRepo.FindByID(ctx, configID)
	if err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Provider+not+found")
	}

	// Parse form values
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")
	customRedirectURL := c.FormValue("redirect_url")
	scopesStr := c.FormValue("scopes")
	displayName := c.FormValue("display_name")
	isEnabled := c.FormValue("is_enabled") == "true"

	// Validate required fields
	if clientID == "" {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Client+ID+is+required")
	}

	// Update config fields
	config.ClientID = clientID

	// Only update secret if provided
	if clientSecret != "" {
		config.ClientSecret = clientSecret // TODO: Encrypt before storing
	}

	config.RedirectURL = customRedirectURL
	config.DisplayName = displayName
	config.IsEnabled = isEnabled

	// Parse scopes
	if scopesStr != "" {
		config.Scopes = strings.Fields(scopesStr)
	} else {
		config.Scopes = nil
	}

	if err := e.configRepo.Update(ctx, config); err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Failed+to+update+provider")
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	return c.Redirect(http.StatusFound, redirectURL+"?success=Provider+updated+successfully")
}

// HandleToggleProvider toggles a social provider's enabled status
func (e *DashboardExtension) HandleToggleProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	configID, err := parseXID(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid provider ID")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()
	redirectURL := basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/social"

	// Get existing config to toggle
	config, err := e.configRepo.FindByID(ctx, configID)
	if err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Provider+not+found")
	}

	// Toggle enabled status
	newEnabled := !config.IsEnabled
	if err := e.configRepo.SetEnabled(ctx, configID, newEnabled); err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Failed+to+toggle+provider")
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	status := "enabled"
	if !newEnabled {
		status = "disabled"
	}

	return c.Redirect(http.StatusFound, redirectURL+"?success=Provider+"+status+"+successfully")
}

// HandleDeleteProvider deletes a social provider configuration
func (e *DashboardExtension) HandleDeleteProvider(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	configID, err := parseXID(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid provider ID")
	}

	basePath := handler.GetBasePath()
	ctx := c.Request().Context()
	redirectURL := basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/social"

	// Get existing config to retrieve environment ID for cache invalidation
	config, err := e.configRepo.FindByID(ctx, configID)
	if err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Provider+not+found")
	}

	// Delete the config (soft delete)
	if err := e.configRepo.Delete(ctx, configID); err != nil {
		return c.Redirect(http.StatusFound, redirectURL+"?error=Failed+to+delete+provider")
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	return c.Redirect(http.StatusFound, redirectURL+"?success=Provider+deleted+successfully")
}
