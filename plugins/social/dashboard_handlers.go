package social

import (
	"net/http"
	"strings"

	g "maragu.dev/gomponents"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forgeui/router"
)

// parseXID parses an xid string.
func parseXID(s string) (xid.ID, error) {
	return xid.FromString(s)
}

// HandleCreateProvider creates a new social provider configuration.
func (e *DashboardExtension) HandleCreateProvider(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	envID, err := e.getCurrentEnvironmentID(ctx, currentApp.ID)
	if err != nil {
		return nil, errs.BadRequest("Invalid environment context")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()
	redirectURL := basePath + "/app/" + currentApp.ID.String() + "/social"

	// Parse form values
	providerName := ctx.Request.FormValue("provider_name")
	clientID := ctx.Request.FormValue("client_id")
	clientSecret := ctx.Request.FormValue("client_secret")
	customRedirectURL := ctx.Request.FormValue("redirect_url")
	scopesStr := ctx.Request.FormValue("scopes")
	displayName := ctx.Request.FormValue("display_name")
	isEnabled := ctx.Request.FormValue("is_enabled") == "true"

	// Validate required fields
	if providerName == "" || clientID == "" || clientSecret == "" {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Missing+required+fields", http.StatusFound)

		return nil, nil
	}

	// Validate provider name
	if !schema.IsValidProvider(providerName) {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Invalid+provider+name", http.StatusFound)

		return nil, nil
	}

	// Check if provider already exists for this environment
	exists, err := e.configRepo.ExistsByProvider(reqCtx, currentApp.ID, envID, providerName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Failed+to+check+existing+provider", http.StatusFound)

		return nil, nil
	}

	if exists {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Provider+already+configured", http.StatusFound)

		return nil, nil
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

	if err := e.configRepo.Create(reqCtx, config); err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Failed+to+create+provider", http.StatusFound)

		return nil, nil
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(currentApp.ID, envID)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?success=Provider+created+successfully", http.StatusFound)

	return nil, nil
}

// HandleUpdateProvider updates an existing social provider configuration.
func (e *DashboardExtension) HandleUpdateProvider(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	configID, err := parseXID(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid provider ID")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()
	redirectURL := basePath + "/app/" + currentApp.ID.String() + "/social"

	// Get existing config
	config, err := e.configRepo.FindByID(reqCtx, configID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Provider+not+found", http.StatusFound)

		return nil, nil
	}

	// Parse form values
	clientID := ctx.Request.FormValue("client_id")
	clientSecret := ctx.Request.FormValue("client_secret")
	customRedirectURL := ctx.Request.FormValue("redirect_url")
	scopesStr := ctx.Request.FormValue("scopes")
	displayName := ctx.Request.FormValue("display_name")
	isEnabled := ctx.Request.FormValue("is_enabled") == "true"

	// Validate required fields
	if clientID == "" {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Client+ID+is+required", http.StatusFound)

		return nil, nil
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

	if err := e.configRepo.Update(reqCtx, config); err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Failed+to+update+provider", http.StatusFound)

		return nil, nil
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?success=Provider+updated+successfully", http.StatusFound)

	return nil, nil
}

// HandleToggleProvider toggles a social provider's enabled status.
func (e *DashboardExtension) HandleToggleProvider(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	configID, err := parseXID(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid provider ID")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()
	redirectURL := basePath + "/app/" + currentApp.ID.String() + "/social"

	// Get existing config to toggle
	config, err := e.configRepo.FindByID(reqCtx, configID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Provider+not+found", http.StatusFound)

		return nil, nil
	}

	// Toggle enabled status
	newEnabled := !config.IsEnabled
	if err := e.configRepo.SetEnabled(reqCtx, configID, newEnabled); err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Failed+to+toggle+provider", http.StatusFound)

		return nil, nil
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	status := "enabled"
	if !newEnabled {
		status = "disabled"
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?success=Provider+"+status+"+successfully", http.StatusFound)

	return nil, nil
}

// HandleDeleteProvider deletes a social provider configuration.
func (e *DashboardExtension) HandleDeleteProvider(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	configID, err := parseXID(ctx.Param("id"))
	if err != nil {
		return nil, errs.BadRequest("Invalid provider ID")
	}

	basePath := e.getBasePath()
	reqCtx := ctx.Request.Context()
	redirectURL := basePath + "/app/" + currentApp.ID.String() + "/social"

	// Get existing config to retrieve environment ID for cache invalidation
	config, err := e.configRepo.FindByID(reqCtx, configID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Provider+not+found", http.StatusFound)

		return nil, nil
	}

	// Delete the config (soft delete)
	if err := e.configRepo.Delete(reqCtx, configID); err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?error=Failed+to+delete+provider", http.StatusFound)

		return nil, nil
	}

	// Invalidate cache for this environment
	if e.plugin != nil && e.plugin.service != nil {
		e.plugin.service.InvalidateEnvironmentCache(config.AppID, config.EnvironmentID)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL+"?success=Provider+deleted+successfully", http.StatusFound)

	return nil, nil
}
