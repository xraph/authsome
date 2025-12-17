package scim

import (
	"fmt"
	"net/http"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Token Management Handlers

// ServeTokensListPage renders the SCIM tokens management page
func (e *DashboardExtension) ServeTokensListPage(c forge.Context) error {
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

	// Get current environment
	currentEnv, err := handler.GetCurrentEnvironment(c, currentApp.ID)
	if err != nil || currentEnv == nil {
		return c.String(http.StatusBadRequest, "Invalid environment context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(c)

	content := e.renderTokensListContent(c, currentApp, currentEnv, orgID)

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "scim-tokens", content)
}

// renderTokensListContent renders the tokens list page content
func (e *DashboardExtension) renderTokensListContent(c forge.Context, currentApp interface{}, currentEnv interface{}, orgID *xid.ID) g.Node {
	ctx := c.Request().Context()
	basePath := e.getBasePath()

	// Fetch tokens from service
	app := currentApp.(*app.App)
	env := currentEnv.(*environment.Environment)
	tokens, err := e.plugin.service.ListTokens(ctx, &app.ID, &env.ID, orgID)
	if err != nil {
		return alertBox("error", "Error", "Failed to load SCIM tokens: "+err.Error())
	}

	mode := e.detectMode()
	scopeLabel := "App"
	if mode == "organization" && orgID != nil {
		scopeLabel = "Organization"
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("SCIM Tokens")),
				P(Class("mt-1 text-slate-600 dark:text-gray-400"),
					g.Textf("Manage bearer tokens for IdP authentication (%s scope)", scopeLabel)),
			),
			Button(
				Type("button"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				g.Attr("onclick", "showCreateTokenModal()"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Token"),
			),
		),

		// Info Alert
		alertBox("info", "Security Notice",
			"SCIM tokens provide full access to provision users and groups. Store them securely and rotate them regularly. Tokens are only shown once upon creation."),

		// Tokens List
		g.If(len(tokens) == 0,
			emptyState(
				lucide.Key(Class("size-12 text-slate-400")),
				"No SCIM Tokens",
				"Create your first SCIM token to enable identity provider integration with Okta, Azure AD, or other IdP systems.",
				"Create Token",
				"#",
			),
		),

		g.If(len(tokens) > 0,
			Div(
				Class("grid gap-4"),
				g.Group(e.renderTokenCards(tokens, basePath, &app.ID)),
			),
		),

		// Create Token Modal (hidden by default)
		e.renderCreateTokenModal(basePath, &app.ID),
	)
}

// renderTokenCards renders token cards
func (e *DashboardExtension) renderTokenCards(tokens []*SCIMToken, basePath string, appID *xid.ID) []g.Node {
	cards := make([]g.Node, len(tokens))
	for i, token := range tokens {
		cards[i] = tokenCard(
			token,
			basePath,
			*appID,
			fmt.Sprintf("revokeToken('%s')", token.ID.String()),
			fmt.Sprintf("rotateToken('%s')", token.ID.String()),
		)
	}
	return cards
}

// renderCreateTokenModal renders the create token modal
func (e *DashboardExtension) renderCreateTokenModal(basePath string, appID *xid.ID) g.Node {
	return Div(
		ID("create-token-modal"),
		Class("hidden fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("aria-labelledby", "modal-title"),
		g.Attr("role", "dialog"),
		g.Attr("aria-modal", "true"),

		Div(
			Class("flex min-h-screen items-end justify-center px-4 pt-4 pb-20 text-center sm:block sm:p-0"),

			// Background overlay
			Div(
				Class("fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"),
				g.Attr("aria-hidden", "true"),
				g.Attr("onclick", "hideCreateTokenModal()"),
			),

			// Modal panel
			Div(
				Class("inline-block align-bottom bg-white dark:bg-gray-900 rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6"),

				Form(
					ID("create-token-form"),
					g.Attr("onsubmit", "return handleCreateToken(event)"),

					Div(
						Class("space-y-4"),

						H3(
							ID("modal-title"),
							Class("text-lg font-medium leading-6 text-slate-900 dark:text-white"),
							g.Text("Create SCIM Token"),
						),

						// Name field
						Div(
							Label(
								For("token-name"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Token Name"),
							),
							Input(
								Type("text"),
								Name("name"),
								ID("token-name"),
								Required(),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								g.Attr("placeholder", "Production Okta"),
							),
						),

						// Description field
						Div(
							Label(
								For("token-description"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Description (optional)"),
							),
							Textarea(
								Name("description"),
								ID("token-description"),
								Rows("3"),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								g.Attr("placeholder", "SCIM token for production Okta integration"),
							),
						),

						// Scopes field
						Div(
							Label(
								For("token-scopes"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Scopes"),
							),
							Select(
								Name("scopes"),
								ID("token-scopes"),
								Multiple(),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								Option(Value("scim:read"), g.Text("scim:read"), g.Attr("selected", "")),
								Option(Value("scim:write"), g.Text("scim:write"), g.Attr("selected", "")),
								Option(Value("scim:users"), g.Text("scim:users")),
								Option(Value("scim:groups"), g.Text("scim:groups")),
							),
							P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
								g.Text("Select one or more scopes (Ctrl/Cmd + Click)")),
						),

						// Expiry field
						Div(
							Label(
								For("token-expiry"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Expires In"),
							),
							Select(
								Name("expires_in"),
								ID("token-expiry"),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								Option(Value("30"), g.Text("30 days")),
								Option(Value("90"), g.Text("90 days"), g.Attr("selected", "")),
								Option(Value("180"), g.Text("180 days")),
								Option(Value("365"), g.Text("1 year")),
								Option(Value(""), g.Text("Never")),
							),
						),
					),

					// Actions
					Div(
						Class("mt-5 sm:mt-6 flex gap-3"),
						Button(
							Type("button"),
							Class("flex-1 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
							g.Attr("onclick", "hideCreateTokenModal()"),
							g.Text("Cancel"),
						),
						Button(
							Type("submit"),
							Class("flex-1 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
							g.Text("Create Token"),
						),
					),
				),
			),
		),

		// Token display modal (shown after creation)
		e.renderTokenDisplayModal(),
	)
}

// renderTokenDisplayModal renders the modal that displays the newly created token
func (e *DashboardExtension) renderTokenDisplayModal() g.Node {
	return Div(
		ID("token-display-modal"),
		Class("hidden fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("aria-labelledby", "token-modal-title"),
		g.Attr("role", "dialog"),
		g.Attr("aria-modal", "true"),

		Div(
			Class("flex min-h-screen items-end justify-center px-4 pt-4 pb-20 text-center sm:block sm:p-0"),

			Div(
				Class("fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"),
				g.Attr("aria-hidden", "true"),
			),

			Div(
				Class("inline-block align-bottom bg-white dark:bg-gray-900 rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6"),

				Div(
					Class("space-y-4"),

					Div(
						Class("flex items-center gap-3"),
						Div(
							Class("rounded-full bg-green-100 p-3 dark:bg-green-900/30"),
							lucide.Check(Class("size-6 text-green-600 dark:text-green-400")),
						),
						H3(
							ID("token-modal-title"),
							Class("text-lg font-medium text-slate-900 dark:text-white"),
							g.Text("Token Created Successfully"),
						),
					),

					Div(
						Class("rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 p-4"),
						Div(
							Class("flex gap-2"),
							lucide.Info(Class("size-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0")),
							P(Class("text-sm text-yellow-800 dark:text-yellow-400"),
								g.Text("Save this token securely. It will not be shown again!")),
						),
					),

					Div(
						Label(
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
							g.Text("Your SCIM Token"),
						),
						Div(
							Class("flex gap-2"),
							Input(
								Type("text"),
								ID("new-token-value"),
								g.Attr("readonly", ""),
								Class("flex-1 block w-full rounded-md border-slate-300 bg-slate-50 font-mono text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								Value(""),
							),
							Button(
								Type("button"),
								Class("inline-flex items-center gap-2 rounded-md bg-violet-600 px-3 py-2 text-sm font-medium text-white hover:bg-violet-700"),
								g.Attr("onclick", "copyTokenToClipboard()"),
								lucide.Copy(Class("size-4")),
								g.Text("Copy"),
							),
						),
					),

					Div(
						Class("rounded-lg bg-slate-50 dark:bg-gray-800 p-4"),
						H4(Class("text-sm font-medium text-slate-900 dark:text-white mb-2"),
							g.Text("Configuration Instructions")),
						P(Class("text-sm text-slate-600 dark:text-gray-400 mb-2"),
							g.Text("Use this token in your IdP configuration:")),
						Ol(
							Class("list-decimal list-inside space-y-1 text-sm text-slate-600 dark:text-gray-400"),
							Li(g.Text("Set SCIM Base URL to: "), Code(Class("text-xs"), g.Text("https://your-domain.com/scim/v2"))),
							Li(g.Text("Set Authentication to: Bearer Token")),
							Li(g.Text("Paste the token above as the bearer token")),
						),
					),
				),

				Div(
					Class("mt-5 sm:mt-6"),
					Button(
						Type("button"),
						Class("w-full rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Attr("onclick", "hideTokenDisplayModal()"),
						g.Text("Done"),
					),
				),
			),
		),
	)
}

// HandleCreateToken handles token creation
func (e *DashboardExtension) HandleCreateToken(c forge.Context) error {
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

	// Get current environment
	currentEnv, err := handler.GetCurrentEnvironment(c, currentApp.ID)
	if err != nil || currentEnv == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid environment context",
		})
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(c)

	// Parse form data
	name := c.FormValue("name")
	description := c.FormValue("description")
	scopesStr := c.FormValue("scopes")
	expiresInStr := c.FormValue("expires_in")

	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	// Parse scopes
	var scopes []string
	if scopesStr != "" {
		scopes = []string{"scim:read", "scim:write"} // Default scopes
	} else {
		scopes = []string{"scim:read", "scim:write"}
	}

	// Parse expiry
	var expiresAt *time.Time
	if expiresInStr != "" {
		days := 90 // Default to 90 days
		switch expiresInStr {
		case "30":
			days = 30
		case "180":
			days = 180
		case "365":
			days = 365
		}
		expiry := time.Now().AddDate(0, 0, days)
		expiresAt = &expiry
	}

	// Create token
	req := &CreateSCIMTokenRequest{
		AppID:          currentApp.ID,
		EnvironmentID:  currentEnv.ID,
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		Scopes:         scopes,
		ExpiresAt:      expiresAt,
	}

	token, err := e.plugin.service.CreateToken(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create token: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token.Token, // Full token only shown once
		"id":      token.ID.String(),
		"message": "Token created successfully",
	})
}

// HandleRotateToken handles token rotation
func (e *DashboardExtension) HandleRotateToken(c forge.Context) error {
	ctx := c.Request().Context()
	tokenID := c.Param("id")

	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Token ID is required",
		})
	}

	parsedTokenID, err := xid.FromString(tokenID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid token ID",
		})
	}

	// Rotate token
	newToken, err := e.plugin.service.RotateToken(ctx, parsedTokenID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to rotate token: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   newToken.Token,
		"message": "Token rotated successfully",
	})
}

// HandleRevokeToken handles token revocation
func (e *DashboardExtension) HandleRevokeToken(c forge.Context) error {
	ctx := c.Request().Context()
	tokenID := c.Param("id")

	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Token ID is required",
		})
	}

	parsedTokenID, err := xid.FromString(tokenID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid token ID",
		})
	}

	// Revoke token
	err = e.plugin.service.RevokeToken(ctx, parsedTokenID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to revoke token: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Token revoked successfully",
	})
}

// HandleTestConnection handles connection testing
func (e *DashboardExtension) HandleTestConnection(c forge.Context) error {
	ctx := c.Request().Context()
	tokenID := c.Param("id")

	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Token ID is required",
		})
	}

	parsedTokenID, err := xid.FromString(tokenID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid token ID",
		})
	}

	// Test connection
	result, err := e.plugin.service.TestConnection(ctx, parsedTokenID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Connection test failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, result)
}
