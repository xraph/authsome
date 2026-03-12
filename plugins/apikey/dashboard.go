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
			Render: func(ctx context.Context) templ.Component {
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
		appID, _ = id.ParseAppID(p.defaultAppID)
	}

	var data akdash.KeysPageData

	// Handle form actions (POST).
	action := params.FormData["action"]
	switch action {
	case "create":
		nonce := params.FormData["nonce"]
		if !dashboard.ConsumeNonce(nonce) {
			// Nonce was already consumed (duplicate submission / refresh).
			// Skip creation silently and just show the list.
			break
		}
		data.CreatedKey, data.Error = p.handleDashboardCreate(ctx, appID, params)
	case "revoke":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			p.handleDashboardRevoke(ctx, params)
		}
	}

	// Generate a fresh nonce for the next form render.
	data.FormNonce = dashboard.GenerateNonce()

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
func (p *Plugin) handleDashboardCreate(ctx context.Context, appID id.AppID, params contributor.Params) (created *akdash.CreatedKeyView, errMsg string) {
	name := strings.TrimSpace(params.FormData["name"])
	if name == "" {
		return nil, "Please provide a key name."
	}

	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Sprintf("Failed to generate key pair: %v", err)
	}

	now := time.Now()
	key := &apikey.APIKey{
		ID:              id.NewAPIKeyID(),
		AppID:           appID,
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
	_ = p.store.UpdateAPIKey(ctx, key)
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
		appID, _ = id.ParseAppID(p.defaultAppID)
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
