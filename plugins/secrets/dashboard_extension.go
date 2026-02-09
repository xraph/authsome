package secrets

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/secrets/core"
	"github.com/xraph/authsome/plugins/secrets/pages"
	"github.com/xraph/forgeui/bridge"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the secrets plugin.
type DashboardExtension struct {
	plugin *Plugin
}

// NewDashboardExtension creates a new dashboard extension.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "secrets"
}

// NavigationItems returns navigation items for the dashboard.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "secrets",
			Label:    "Secrets",
			Icon:     lucide.KeyRound(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    60,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/secrets"
				}

				return basePath + "/app/" + currentApp.ID.String() + "/secrets"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "secrets" || activePage == "secret-detail" ||
					activePage == "secret-create" || activePage == "secret-edit" ||
					activePage == "secret-history"
			},
			RequiresPlugin: "secrets",
		},
	}
}

// Routes returns dashboard routes
// Note: All secrets routes use /secrets/ prefix (not /settings/secrets/) to ensure
// they get the dashboard layout instead of settings layout.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Secrets list page
		{
			Method:       "GET",
			Path:         "/secrets",
			Handler:      e.ServeSecretsListPage,
			Name:         "secrets.dashboard.list",
			Summary:      "Secrets management",
			Description:  "View and manage application secrets",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create secret page
		{
			Method:       "GET",
			Path:         "/secrets/create",
			Handler:      e.ServeCreateSecretPage,
			Name:         "secrets.dashboard.create",
			Summary:      "Create secret",
			Description:  "Create a new secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create secret action
		{
			Method:       "POST",
			Path:         "/secrets/create",
			Handler:      e.HandleCreateSecret,
			Name:         "secrets.dashboard.create.submit",
			Summary:      "Submit create secret",
			Description:  "Process secret creation form",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Secret detail page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId",
			Handler:      e.ServeSecretDetailPage,
			Name:         "secrets.dashboard.detail",
			Summary:      "Secret details",
			Description:  "View secret details",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit secret page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId/edit",
			Handler:      e.ServeEditSecretPage,
			Name:         "secrets.dashboard.edit",
			Summary:      "Edit secret",
			Description:  "Edit an existing secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update secret action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/update",
			Handler:      e.HandleUpdateSecret,
			Name:         "secrets.dashboard.update",
			Summary:      "Update secret",
			Description:  "Process secret update form",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete secret action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/delete",
			Handler:      e.HandleDeleteSecret,
			Name:         "secrets.dashboard.delete",
			Summary:      "Delete secret",
			Description:  "Delete a secret",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Version history page
		{
			Method:       "GET",
			Path:         "/secrets/:secretId/history",
			Handler:      e.ServeVersionHistoryPage,
			Name:         "secrets.dashboard.history",
			Summary:      "Version history",
			Description:  "View secret version history",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Rollback action
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/rollback/:version",
			Handler:      e.HandleRollback,
			Name:         "secrets.dashboard.rollback",
			Summary:      "Rollback secret",
			Description:  "Rollback to a previous version",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Reveal value (legacy POST endpoint - prefer using bridge)
		{
			Method:       "POST",
			Path:         "/secrets/:secretId/reveal",
			Handler:      e.HandleRevealValue,
			Name:         "secrets.dashboard.reveal",
			Summary:      "Reveal secret value",
			Description:  "Get decrypted secret value",
			Tags:         []string{"Dashboard", "Secrets"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated).
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages
// Note: Secrets is a main navigation item (not a settings page), so we return nil here.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return nil
}

// DashboardWidgets returns dashboard widgets.
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "secrets-count",
			Title: "Secrets",
			Icon:  lucide.KeyRound(Class("size-5")),
			Order: 50,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderSecretsWidget(currentApp)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for the secrets plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		{
			Name:        "revealSecret",
			Handler:     e.bridgeRevealSecret,
			Description: "Reveal a secret's decrypted value",
		},
		{
			Name:        "getSecrets",
			Handler:     e.bridgeGetSecrets,
			Description: "List secrets with pagination",
		},
		{
			Name:        "getSecret",
			Handler:     e.bridgeGetSecret,
			Description: "Get secret details by ID",
		},
		{
			Name:        "createSecret",
			Handler:     e.bridgeCreateSecret,
			Description: "Create a new secret",
		},
		{
			Name:        "updateSecret",
			Handler:     e.bridgeUpdateSecret,
			Description: "Update an existing secret",
		},
		{
			Name:        "deleteSecret",
			Handler:     e.bridgeDeleteSecret,
			Description: "Delete a secret",
		},
	}
}

// =============================================================================
// Bridge Handler Types
// =============================================================================

// RevealSecretInput is the input for revealing a secret.
type RevealSecretInput struct {
	AppID    string `json:"appId"`
	SecretID string `json:"secretId"`
}

// RevealSecretOutput is the output for revealing a secret.
type RevealSecretOutput struct {
	Value     any    `json:"value"`
	ValueType string `json:"valueType"`
}

// GetSecretsInput is the input for listing secrets.
type GetSecretsInput struct {
	AppID    string `json:"appId"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Search   string `json:"search"`
}

// SecretItem represents a secret in the list.
type SecretItem struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Key         string   `json:"key"`
	Description string   `json:"description"`
	ValueType   string   `json:"valueType"`
	Tags        []string `json:"tags"`
	Version     int      `json:"version"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// GetSecretsOutput is the output for listing secrets.
type GetSecretsOutput struct {
	Secrets    []SecretItem `json:"secrets"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalPages int          `json:"totalPages"`
}

// GetSecretInput is the input for getting a secret.
type GetSecretInput struct {
	AppID    string `json:"appId"`
	SecretID string `json:"secretId"`
}

// GetSecretOutput is the output for getting a secret.
type GetSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

// CreateSecretInput is the input for creating a secret.
type CreateSecretInput struct {
	AppID       string   `json:"appId"`
	Path        string   `json:"path"`
	Value       any      `json:"value"`
	Description string   `json:"description"`
	ValueType   string   `json:"valueType"`
	Tags        []string `json:"tags"`
}

// CreateSecretOutput is the output for creating a secret.
type CreateSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

// UpdateSecretInput is the input for updating a secret.
type UpdateSecretInput struct {
	AppID        string   `json:"appId"`
	SecretID     string   `json:"secretId"`
	Value        any      `json:"value"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	ChangeReason string   `json:"changeReason"`
}

// UpdateSecretOutput is the output for updating a secret.
type UpdateSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

// DeleteSecretInput is the input for deleting a secret.
type DeleteSecretInput struct {
	AppID    string `json:"appId"`
	SecretID string `json:"secretId"`
}

// DeleteSecretOutput is the output for deleting a secret.
type DeleteSecretOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// =============================================================================
// Bridge Handler Implementations
// =============================================================================

// bridgeRevealSecret handles the revealSecret bridge call.
func (e *DashboardExtension) bridgeRevealSecret(ctx bridge.Context, input RevealSecretInput) (*RevealSecretOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse secret ID
	secretID, err := xid.FromString(input.SecretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid secretId")
	}

	// Get the decrypted value
	value, err := e.plugin.Service().GetValue(goCtx, secretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to retrieve secret value")
	}

	// Get secret details for value type
	secret, err := e.plugin.Service().Get(goCtx, secretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to retrieve secret")
	}

	return &RevealSecretOutput{
		Value:     value,
		ValueType: secret.ValueType,
	}, nil
}

// bridgeGetSecrets handles the getSecrets bridge call.
func (e *DashboardExtension) bridgeGetSecrets(ctx bridge.Context, input GetSecretsInput) (*GetSecretsOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	// Build query
	query := &core.ListSecretsQuery{
		Page:     page,
		PageSize: pageSize,
	}

	if input.Search != "" {
		query.Search = input.Search
	}

	// Fetch secrets
	secretList, pag, err := e.plugin.Service().List(goCtx, query)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch secrets")
	}

	// Transform to output
	secrets := make([]SecretItem, 0, len(secretList))
	for _, s := range secretList {
		secrets = append(secrets, SecretItem{
			ID:          s.ID,
			Path:        s.Path,
			Key:         s.Key,
			Description: s.Description,
			ValueType:   s.ValueType,
			Tags:        s.Tags,
			Version:     s.Version,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	totalPages := pag.TotalItems / pageSize
	if pag.TotalItems%pageSize > 0 {
		totalPages++
	}

	return &GetSecretsOutput{
		Secrets:    secrets,
		Total:      int64(pag.TotalItems),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// bridgeGetSecret handles the getSecret bridge call.
func (e *DashboardExtension) bridgeGetSecret(ctx bridge.Context, input GetSecretInput) (*GetSecretOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse secret ID
	secretID, err := xid.FromString(input.SecretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid secretId")
	}

	// Fetch secret
	secret, err := e.plugin.Service().Get(goCtx, secretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "secret not found")
	}

	return &GetSecretOutput{
		Secret: SecretItem{
			ID:          secret.ID,
			Path:        secret.Path,
			Key:         secret.Key,
			Description: secret.Description,
			ValueType:   secret.ValueType,
			Tags:        secret.Tags,
			Version:     secret.Version,
			CreatedAt:   secret.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   secret.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

// bridgeCreateSecret handles the createSecret bridge call.
func (e *DashboardExtension) bridgeCreateSecret(ctx bridge.Context, input CreateSecretInput) (*CreateSecretOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Create secret (app/env IDs are extracted from context by the service)
	createReq := &core.CreateSecretRequest{
		Path:        input.Path,
		Value:       input.Value,
		Description: input.Description,
		ValueType:   input.ValueType,
		Tags:        input.Tags,
	}

	secret, err := e.plugin.Service().Create(goCtx, createReq)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create secret: "+err.Error())
	}

	return &CreateSecretOutput{
		Secret: SecretItem{
			ID:          secret.ID,
			Path:        secret.Path,
			Key:         secret.Key,
			Description: secret.Description,
			ValueType:   secret.ValueType,
			Tags:        secret.Tags,
			Version:     secret.Version,
			CreatedAt:   secret.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   secret.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

// bridgeUpdateSecret handles the updateSecret bridge call.
func (e *DashboardExtension) bridgeUpdateSecret(ctx bridge.Context, input UpdateSecretInput) (*UpdateSecretOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse secret ID
	secretID, err := xid.FromString(input.SecretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid secretId")
	}

	// Update secret
	updateReq := &core.UpdateSecretRequest{
		Description:  input.Description,
		Tags:         input.Tags,
		ChangeReason: input.ChangeReason,
	}

	// Only update value if provided
	if input.Value != nil {
		updateReq.Value = input.Value
	}

	secret, err := e.plugin.Service().Update(goCtx, secretID, updateReq)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update secret: "+err.Error())
	}

	return &UpdateSecretOutput{
		Secret: SecretItem{
			ID:          secret.ID,
			Path:        secret.Path,
			Key:         secret.Key,
			Description: secret.Description,
			ValueType:   secret.ValueType,
			Tags:        secret.Tags,
			Version:     secret.Version,
			CreatedAt:   secret.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   secret.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

// bridgeDeleteSecret handles the deleteSecret bridge call.
func (e *DashboardExtension) bridgeDeleteSecret(ctx bridge.Context, input DeleteSecretInput) (*DeleteSecretOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse secret ID
	secretID, err := xid.FromString(input.SecretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid secretId")
	}

	// Delete secret
	err = e.plugin.Service().Delete(goCtx, secretID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete secret: "+err.Error())
	}

	return &DeleteSecretOutput{
		Success: true,
		Message: "Secret deleted successfully",
	}, nil
}

// buildContextFromBridge creates a Go context from a bridge context with app/env IDs.
func (e *DashboardExtension) buildContextFromBridge(ctx bridge.Context, appIDStr string) (context.Context, error) {
	// Get the context from the HTTP request (already enriched by middleware)
	var goCtx context.Context
	if req := ctx.Request(); req != nil {
		goCtx = req.Context()
	} else {
		goCtx = ctx.Context()
	}

	// Parse and set app ID if provided
	if appIDStr != "" {
		appID, err := xid.FromString(appIDStr)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
		}

		goCtx = contexts.SetAppID(goCtx, appID)

		// Ensure environment ID is set
		if _, ok := contexts.GetEnvironmentID(goCtx); !ok {
			// Try to get default environment
			if e.plugin.authInst != nil {
				if envSvc := e.plugin.authInst.GetServiceRegistry().EnvironmentService(); envSvc != nil {
					if defaultEnv, err := envSvc.GetDefaultEnvironment(goCtx, appID); err == nil && defaultEnv != nil {
						goCtx = contexts.SetEnvironmentID(goCtx, defaultEnv.ID)
					}
				}
			}
		}
	}

	return goCtx, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// getUserFromContext extracts the current user from the request context.
func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) *user.User {
	reqCtx := ctx.Request.Context()
	if u, ok := reqCtx.Value("user").(*user.User); ok {
		return u
	}

	return nil
}

// extractAppFromURL extracts the app from the URL parameter.
func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.RequiredField("appId")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path.
func (e *DashboardExtension) getBasePath() string {
	return "/api/identity/ui"
}

// injectContext returns the Go context from the HTTP request.
// The context is already enriched by the dashboard v2 AppContextMiddleware with:
// - App ID (from URL parameter)
// - Environment ID (from cookie or default environment)
// - User ID (from session via AuthMiddleware)
//
// This method provides a fallback for cases where the middleware hasn't enriched the context.
func (e *DashboardExtension) injectContext(ctx *router.PageContext) context.Context {
	// Get the context that's already been enriched by dashboard middleware
	reqCtx := ctx.Request.Context()

	// Check if app ID is already set (by dashboard middleware)
	if appID, ok := contexts.GetAppID(reqCtx); ok && !appID.IsNil() {
		// Check if environment ID is also set
		if envID, ok := contexts.GetEnvironmentID(reqCtx); ok && !envID.IsNil() {
			// Context is fully enriched, return as-is
			return reqCtx
		}
	}

	// Fallback: Manually enrich context if middleware didn't do it
	// This handles edge cases or direct API access
	var appID xid.ID

	if appIDStr := ctx.Param("appId"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			appID = id
			reqCtx = contexts.SetAppID(reqCtx, appID)
		}
	}

	// Try to get environment ID from cookie
	if envCookie, err := ctx.Request.Cookie("authsome_environment"); err == nil && envCookie != nil && envCookie.Value != "" {
		if envID, err := xid.FromString(envCookie.Value); err == nil && !envID.IsNil() {
			reqCtx = contexts.SetEnvironmentID(reqCtx, envID)

			return reqCtx
		}
	}

	// Fall back to default environment for the app
	if !appID.IsNil() && e.plugin.authInst != nil {
		if envSvc := e.plugin.authInst.GetServiceRegistry().EnvironmentService(); envSvc != nil {
			if defaultEnv, err := envSvc.GetDefaultEnvironment(reqCtx, appID); err == nil && defaultEnv != nil {
				reqCtx = contexts.SetEnvironmentID(reqCtx, defaultEnv.ID)
			}
		}
	}

	return reqCtx
}

// parseSecretID parses a secret ID from URL parameter.
func (e *DashboardExtension) parseSecretID(ctx *router.PageContext) (xid.ID, error) {
	idStr := ctx.Param("secretId")
	if idStr == "" {
		return xid.NilID(), errs.RequiredField("secretId")
	}

	return xid.FromString(idStr)
}

// =============================================================================
// Widget Renderer
// =============================================================================

func (e *DashboardExtension) renderSecretsWidget(currentApp *app.App) g.Node {
	// Create a background context with app context
	ctx := context.Background()
	if currentApp != nil {
		ctx = contexts.SetAppID(ctx, currentApp.ID)
	}

	// Try to get stats - use default context handling
	stats, err := e.plugin.Service().GetStats(ctx)

	count := 0
	if err == nil && stats != nil {
		count = stats.TotalSecrets
	}

	return Div(
		Class("text-center"),
		Div(
			Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text(strconv.Itoa(count)),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Total Secrets"),
		),
	)
}

// =============================================================================
// Common UI Components
// =============================================================================

// statsCard renders a statistics card.
func (e *DashboardExtension) statsCard(title, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("mt-1 text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
			),
			Div(
				Class("rounded-full bg-violet-100 p-3 dark:bg-violet-900/30"),
				icon,
			),
		),
	)
}

// statusBadge renders a status badge.
func (e *DashboardExtension) statusBadge(status string) g.Node {
	var classes string

	switch status {
	case "active", "success":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	case "expired", "error":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
	case "expiring":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
	default:
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}

	return Span(Class(classes), g.Text(status))
}

// valueTypeBadge renders a value type badge.
func (e *DashboardExtension) valueTypeBadge(valueType string) g.Node {
	var classes, icon string

	switch valueType {
	case "json":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
		icon = "{}"
	case "yaml":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
		icon = "---"
	case "binary":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400"
		icon = "01"
	default:
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-gray-700 dark:text-gray-300"
		icon = "Aa"
	}

	return Span(
		Class(classes),
		Span(Class("font-mono"), g.Text(icon)),
		g.Text(valueType),
	)
}

// renderPagination renders pagination controls.
func (e *DashboardExtension) renderPagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	items := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Previous"),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 border border-violet-600 rounded-md"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(
				Class("px-2 py-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Next"),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(items),
	)
}

// =============================================================================
// Secrets Sub-navigation
// =============================================================================

func (e *DashboardExtension) renderSecretsNav(currentApp *app.App, basePath, activePage string) g.Node {
	type navItem struct {
		label string
		path  string
		page  string
		icon  g.Node
	}

	items := []navItem{
		{"All Secrets", "/secrets", "secrets", lucide.List(Class("size-4"))},
	}

	navItems := make([]g.Node, 0, len(items))
	for _, item := range items {
		isActive := activePage == item.page

		classes := "inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors "
		if isActive {
			classes += "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
		} else {
			classes += "text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800"
		}

		navItems = append(navItems, A(
			Href(basePath+"/app/"+currentApp.ID.String()+item.path),
			Class(classes),
			item.icon,
			g.Text(item.label),
		))
	}

	return Nav(
		Class("flex flex-wrap gap-2 mb-6 p-2 bg-slate-50 dark:bg-gray-800/50 rounded-lg"),
		g.Group(navItems),
	)
}

// =============================================================================
// Page Handlers - Placeholders (to be implemented in pages/ folder)
// =============================================================================

// ServeSecretsListPage serves the secrets list page.
func (e *DashboardExtension) ServeSecretsListPage(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get query parameters
	query := &core.ListSecretsQuery{
		Prefix: func() string {
			v := ctx.Request.URL.Query().Get("prefix")
			if v == "" {
				return ""
			}

			return v
		}(),
		Search: func() string {
			v := ctx.Request.URL.Query().Get("search")
			if v == "" {
				return ""
			}

			return v
		}(),
		PageSize: 20,
		Page:     1,
	}

	if p := func() string {
		v := ctx.Request.URL.Query().Get("page")
		if v == "" {
			return ""
		}

		return v
	}(); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			query.Page = parsed
		}
	}

	// Get secrets
	secrets, pag, err := e.plugin.Service().List(reqCtx, query)
	if err != nil {
		secrets = []*core.SecretDTO{}
		pag = nil
	}

	basePath := e.getBasePath()
	content := e.renderSecretsListContent(currentApp, basePath, secrets, pag, query)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeCreateSecretPage serves the create secret page.
func (e *DashboardExtension) ServeCreateSecretPage(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()
	content := e.renderCreateSecretForm(currentApp, basePath, nil, "")

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// HandleCreateSecret handles the create secret form submission.
func (e *DashboardExtension) HandleCreateSecret(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Parse form
	req := &core.CreateSecretRequest{
		Path:        ctx.Request.FormValue("path"),
		Value:       ctx.Request.FormValue("value"),
		ValueType:   ctx.Request.FormValue("valueType"),
		Schema:      ctx.Request.FormValue("schema"),
		Description: ctx.Request.FormValue("description"),
	}

	if tags := ctx.Request.FormValue("tags"); tags != "" {
		req.Tags = splitTags(tags)
	}

	// Create secret
	_, err = e.plugin.Service().Create(reqCtx, req)
	if err != nil {
		basePath := e.getBasePath()
		content := e.renderCreateSecretForm(currentApp, basePath, req, err.Error())

		return content, nil
	}

	// Redirect to list
	http.Redirect(ctx.ResponseWriter, ctx.Request, e.getBasePath()+"/app/"+currentApp.ID.String()+"/secrets", http.StatusFound)

	return nil, nil
}

// ServeSecretDetailPage serves the secret detail page.
func (e *DashboardExtension) ServeSecretDetailPage(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	// Get secret
	secret, err := e.plugin.Service().Get(reqCtx, secretID)
	if err != nil {
		return nil, errs.NotFound("Secret not found")
	}

	// Get recent versions
	versions, _, _ := e.plugin.Service().GetVersions(reqCtx, secretID, 1, 5)

	basePath := e.getBasePath()
	content := e.renderSecretDetailContent(currentApp, basePath, secret, versions)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// ServeEditSecretPage serves the edit secret page.
func (e *DashboardExtension) ServeEditSecretPage(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	secret, err := e.plugin.Service().Get(reqCtx, secretID)
	if err != nil {
		return nil, errs.NotFound("Secret not found")
	}

	basePath := e.getBasePath()
	content := e.renderEditSecretForm(currentApp, basePath, secret, "")

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// HandleUpdateSecret handles the update secret form submission.
func (e *DashboardExtension) HandleUpdateSecret(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	// Parse form
	req := &core.UpdateSecretRequest{
		Description:  ctx.Request.FormValue("description"),
		ChangeReason: ctx.Request.FormValue("changeReason"),
	}

	if value := ctx.Request.FormValue("value"); value != "" {
		req.Value = value
	}

	if valueType := ctx.Request.FormValue("valueType"); valueType != "" {
		req.ValueType = valueType
	}

	if tags := ctx.Request.FormValue("tags"); tags != "" {
		req.Tags = splitTags(tags)
	}

	// Update secret
	_, err = e.plugin.Service().Update(reqCtx, secretID, req)
	if err != nil {
		secret, _ := e.plugin.Service().Get(reqCtx, secretID)
		basePath := e.getBasePath()
		content := e.renderEditSecretForm(currentApp, basePath, secret, err.Error())

		return content, nil
	}

	// Redirect to detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, e.getBasePath()+"/app/"+currentApp.ID.String()+"/secrets/"+secretID.String(), http.StatusFound)

	return nil, nil
}

// HandleDeleteSecret handles the delete secret action.
func (e *DashboardExtension) HandleDeleteSecret(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	if err := e.plugin.Service().Delete(reqCtx, secretID); err != nil {
		return nil, errs.InternalServerError("Failed to delete secret", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, e.getBasePath()+"/app/"+currentApp.ID.String()+"/secrets", http.StatusFound)

	return nil, nil
}

// ServeVersionHistoryPage serves the version history page.
func (e *DashboardExtension) ServeVersionHistoryPage(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	secret, err := e.plugin.Service().Get(reqCtx, secretID)
	if err != nil {
		return nil, errs.NotFound("Secret not found")
	}

	page := 1

	if p := func() string {
		v := ctx.Request.URL.Query().Get("page")
		if v == "" {
			return ""
		}

		return v
	}(); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	versions, pag, _ := e.plugin.Service().GetVersions(reqCtx, secretID, page, 20)

	basePath := e.getBasePath()
	content := e.renderVersionHistoryContent(currentApp, basePath, secret, versions, pag)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// HandleRollback handles the rollback action.
func (e *DashboardExtension) HandleRollback(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid secret ID")
	}

	versionStr := ctx.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid version number")
	}

	_, err = e.plugin.Service().Rollback(reqCtx, secretID, version, "Rollback from dashboard")
	if err != nil {
		return nil, errs.InternalServerError("Rollback failed", err)
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, e.getBasePath()+"/app/"+currentApp.ID.String()+"/secrets/"+secretID.String(), http.StatusFound)

	return nil, nil
}

// HandleRevealValue handles the reveal value AJAX request.
func (e *DashboardExtension) HandleRevealValue(ctx *router.PageContext) (g.Node, error) {
	// Use injectContext to properly set app/environment IDs in context
	reqCtx := e.injectContext(ctx)

	secretID, err := e.parseSecretID(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid request")
	}

	value, err := e.plugin.Service().GetValue(reqCtx, secretID)
	if err != nil {
		return nil, errs.InternalServerError("Operation failed", nil)
	}

	secret, _ := e.plugin.Service().Get(reqCtx, secretID)
	_ = value
	_ = secret

	return nil, nil // Success
}

// =============================================================================
// Page Rendering Methods
// =============================================================================

// =============================================================================
// Page Rendering Methods
// =============================================================================

// renderSecretsListContent renders the secrets list page content.
func (e *DashboardExtension) renderSecretsListContent(
	currentApp *app.App,
	basePath string,
	secrets []*core.SecretDTO,
	pag *pagination.Pagination,
	query *core.ListSecretsQuery,
) g.Node {
	return pages.SecretsListPage(currentApp, basePath, secrets, pag, query)
}

// renderCreateSecretForm renders the create secret form.
func (e *DashboardExtension) renderCreateSecretForm(
	currentApp *app.App,
	basePath string,
	prefill *core.CreateSecretRequest,
	errorMsg string,
) g.Node {
	return pages.CreateSecretPage(currentApp, basePath, prefill, errorMsg)
}

// renderSecretDetailContent renders the secret detail page content.
func (e *DashboardExtension) renderSecretDetailContent(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
) g.Node {
	return pages.SecretDetailPage(currentApp, basePath, secret, versions)
}

// renderEditSecretForm renders the edit secret form.
func (e *DashboardExtension) renderEditSecretForm(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	errorMsg string,
) g.Node {
	return pages.EditSecretPage(currentApp, basePath, secret, errorMsg)
}

// renderVersionHistoryContent renders the version history page content.
func (e *DashboardExtension) renderVersionHistoryContent(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
	pag *pagination.Pagination,
) g.Node {
	return pages.VersionHistoryPage(currentApp, basePath, secret, versions, pag)
}
