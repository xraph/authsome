package apikey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	akdash "github.com/xraph/authsome/plugins/apikey/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin                = (*Plugin)(nil)
	_ dashboard.PageContributor       = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns API key widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "apikey-count",
			Title:      "API Keys",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(_ context.Context) templ.Component {
				return akdash.CountWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the API key settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	var expiryStr string
	if p.config.DefaultExpiry > 0 {
		expiryStr = p.config.DefaultExpiry.String()
	}
	return akdash.SettingsPanel(p.config.MaxKeysPerUser, expiryStr)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the API keys page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "API Keys",
			Path:     "/api-keys",
			Icon:     "key",
			Group:    "Developer",
			Priority: 0,
		},
	}
}

// DashboardRenderPage renders the API keys management page with form handling.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	if route != "/api-keys" {
		return nil, contributor.ErrPageNotFound
	}
	return p.renderKeysPage(ctx, params)
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderKeysPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	if p.store == nil {
		return akdash.KeysPage(akdash.KeysPageData{
			Error: "API key store is not configured.",
		}), nil
	}

	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.defaultAppID) //nolint:errcheck // best-effort parse
	}

	var data akdash.KeysPageData

	// Resolve the actor's session ID once; the scoped nonce binds to
	// (sessionID, scope) so a stolen nonce from another admin's session
	// can't be replayed via CSRF (Phase 2C.2 sweep).
	//
	// The keys page hosts create + revoke forms; both share scope
	// "apikey.write" so a single hidden nonce field on the templ
	// continues to work. Cross-action replay within the same admin's
	// session window is acceptable — both actions are write-class on
	// the same resource type.
	sessionID, _ := middleware.SessionIDFrom(ctx)
	sessIDStr := sessionID.String()
	const formScope = "apikey.write"

	// Handle form actions (POST).
	action := params.FormData["action"]
	switch action {
	case "create":
		nonce := params.FormData["nonce"]
		if !dashboard.ConsumeScopedNonce(sessIDStr, formScope, nonce) {
			// Nonce was already consumed (duplicate submission / refresh)
			// or didn't match the actor's session. Skip silently and
			// show the list.
			break
		}
		data.CreatedKey, data.Error = p.handleDashboardCreate(ctx, appID, params)
	case "revoke":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeScopedNonce(sessIDStr, formScope, nonce) {
			p.handleDashboardRevoke(ctx, params)
		}
	}

	// Generate a fresh nonce for the next form render.
	data.FormNonce = dashboard.GenerateScopedNonce(sessIDStr, formScope)

	// Fetch all keys for the app.
	keys, err := p.store.ListAPIKeysByApp(ctx, appID)
	if err != nil {
		data.Error = fmt.Sprintf("Failed to load API keys: %v", err)
		keys = nil
	}

	data.Keys = make([]akdash.APIKeyView, 0, len(keys))
	for _, k := range keys {
		data.Keys = append(data.Keys, akdash.APIKeyView{
			ID:              k.ID.String(),
			Name:            k.Name,
			KeyPrefix:       k.KeyPrefix,
			PublicKey:       k.PublicKey,
			PublicKeyPrefix: k.PublicKeyPrefix,
			Revoked:         k.Revoked,
			CreatedAt:       k.CreatedAt,
			LastUsedAt:      k.LastUsedAt,
		})
	}

	return akdash.KeysPage(data), nil
}

// handleDashboardCreate creates a new API key from form data.
//
// The new key MUST be bound to a UserID — the API-key auth
// strategy refuses to authenticate keys whose UserID is the zero
// value (resolveUser("") fails, surfacing as a generic
// "authentication required" 401 that's near-impossible to debug
// from the client side). UserID is sourced from, in order:
//
//  1. middleware.UserIDFrom(ctx) — populated by the standard auth
//     middleware when the upstream dashboard forwards the user's
//     Authorization header on its server-to-server call to
//     authsome's contributor protocol. Most local installs hit
//     this path.
//
//  2. params.FormData["actor_user_id"] — for upstream dashboards
//     (e.g. a remote forge Portal) that don't forward auth
//     headers but DO know the logged-in user's authsome user_id.
//     The host opts in by injecting this field into the form.
//
// If neither yields a non-zero UserID, the dashboard cannot
// safely create a key (creating one without a UserID produces a
// row that authenticates against nothing, which is exactly the
// historical bug that this guard exists to prevent). We surface
// an actionable error pointing operators at POST /v1/keys, which
// requires user_id explicitly.
func (p *Plugin) handleDashboardCreate(ctx context.Context, appID id.AppID, params contributor.Params) (created *akdash.CreatedKeyView, errMsg string) {
	name := strings.TrimSpace(params.FormData["name"])
	if name == "" {
		return nil, "Please provide a key name."
	}

	userID, _ := middleware.UserIDFrom(ctx)
	if userID.IsNil() {
		// Fallback: the upstream host can forward the actor's
		// authsome user_id via a form field. Lets remote-dashboard
		// installs work without round-tripping the user's bearer
		// token to authsome.
		if forwarded := strings.TrimSpace(params.FormData["actor_user_id"]); forwarded != "" {
			parsed, err := id.ParseUserID(forwarded)
			if err == nil {
				userID = parsed
			}
		}
	}
	if userID.IsNil() {
		return nil, "Could not identify the dashboard user (no auth context and no actor_user_id in form). Mint the key via POST /v1/keys instead, which accepts user_id explicitly."
	}

	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Sprintf("Failed to generate key pair: %v", err)
	}

	now := time.Now()
	key := &apikey.APIKey{
		ID:              id.NewAPIKeyID(),
		AppID:           appID,
		UserID:          userID,
		Name:            name,
		KeyHash:         secretHash,
		KeyPrefix:       secretPrefix,
		PublicKey:       publicKey,
		PublicKeyPrefix: publicPrefix,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Apply default expiry if configured.
	if p.config.DefaultExpiry > 0 {
		exp := now.Add(p.config.DefaultExpiry)
		key.ExpiresAt = &exp
	}

	if err := p.store.CreateAPIKey(ctx, key); err != nil {
		return nil, fmt.Sprintf("Failed to create API key: %v", err)
	}

	return &akdash.CreatedKeyView{
		Name:      name,
		PublicKey: publicKey,
		SecretKey: secretKey,
		KeyPrefix: secretPrefix,
	}, ""
}

// handleDashboardRevoke revokes an API key from form data.
func (p *Plugin) handleDashboardRevoke(ctx context.Context, params contributor.Params) {
	keyIDStr := params.FormData["key_id"]
	if keyIDStr == "" {
		return
	}

	keyID, err := id.ParseAPIKeyID(keyIDStr)
	if err != nil {
		return
	}

	key, err := p.store.GetAPIKey(ctx, keyID)
	if err != nil {
		return
	}

	key.Revoked = true
	key.UpdatedAt = time.Now()
	_ = p.store.UpdateAPIKey(ctx, key) //nolint:errcheck // best-effort update
}

// ──────────────────────────────────────────────────
// User detail section
// ──────────────────────────────────────────────────

// DashboardUserDetailSection returns the user-specific API keys section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.store == nil {
		return nil
	}
	// Use app ID from dashboard context to scope the query.
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.defaultAppID) //nolint:errcheck // best-effort parse
	}
	keys, err := p.store.ListAPIKeysByUser(ctx, appID, userID)
	if err != nil {
		keys = nil
	}

	// Convert to view models.
	views := make([]akdash.APIKeyView, len(keys))
	for i, k := range keys {
		views[i] = akdash.APIKeyView{
			ID:              k.ID.String(),
			Name:            k.Name,
			KeyPrefix:       k.KeyPrefix,
			PublicKey:       k.PublicKey,
			PublicKeyPrefix: k.PublicKeyPrefix,
			Revoked:         k.Revoked,
			CreatedAt:       k.CreatedAt,
			LastUsedAt:      k.LastUsedAt,
		}
	}
	return akdash.UserSection(views)
}
