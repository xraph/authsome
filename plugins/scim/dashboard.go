package scim

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"

	scimdash "github.com/xraph/authsome/plugins/scim/dashui"
)

// Compile-time dashboard interface checks.
var (
	_ dashboard.Plugin                  = (*Plugin)(nil)
	_ dashboard.PageContributor         = (*Plugin)(nil)
	_ dashboard.OrgDetailContributor    = (*Plugin)(nil)
	_ dashboard.OrgDetailTabContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns SCIM provisioning widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "scim-overview",
			Title:      "SCIM Provisioning",
			Size:       "md",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				appID := p.resolveAppID(ctx)
				data := scimdash.OverviewWidgetData{}
				if p.scimStore != nil {
					if configs, err := p.scimStore.ListConfigs(ctx, appID); err == nil {
						data.ConfigCount = len(configs)
						for _, cfg := range configs {
							if tokens, err := p.scimStore.ListTokens(ctx, cfg.ID); err == nil {
								data.TokenCount += len(tokens)
							}
						}
					}
					if logs, err := p.scimStore.ListAllLogs(ctx, appID, 100); err == nil {
						data.RecentCount = len(logs)
					}
				}
				return scimdash.OverviewWidget(data)
			},
		},
	}
}

// DashboardSettingsPanel returns the SCIM settings panel.
func (p *Plugin) DashboardSettingsPanel(ctx context.Context) templ.Component {
	appID := p.resolveAppID(ctx)
	opts := settings.ResolveOpts{AppID: appID}

	enabled, _ := settings.Get(ctx, p.settings, SettingSCIMEnabled, opts)
	autoCreate, _ := settings.Get(ctx, p.settings, SettingAutoCreateUsers, opts)
	autoSuspend, _ := settings.Get(ctx, p.settings, SettingAutoSuspendUsers, opts)
	groupSync, _ := settings.Get(ctx, p.settings, SettingGroupSync, opts)
	defaultRole, _ := settings.Get(ctx, p.settings, SettingDefaultRole, opts)
	tokenExpiry, _ := settings.Get(ctx, p.settings, SettingTokenExpiryDays, opts)

	return scimdash.SettingsPanel(scimdash.SettingsPanelData{
		Enabled:         enabled,
		AutoCreate:      autoCreate,
		AutoSuspend:     autoSuspend,
		GroupSync:       groupSync,
		DefaultRole:     defaultRole,
		TokenExpiryDays: tokenExpiry,
	})
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for SCIM pages.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "SCIM",
			Path:     "/scim",
			Icon:     "shield-check",
			Group:    "Provisioning",
			Priority: 0,
		},
		{
			Label:    "SCIM Logs",
			Path:     "/scim-logs",
			Icon:     "scroll-text",
			Group:    "Provisioning",
			Priority: 1,
		},
	}
}

// DashboardRenderPage renders SCIM dashboard pages.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch route {
	case "/scim":
		return p.renderSCIMList(ctx, params)
	case "/scim/detail":
		return p.renderSCIMDetail(ctx, params)
	case "/scim-logs":
		return p.renderSCIMLogsPage(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// OrgDetailContributor implementation
// ──────────────────────────────────────────────────

// DashboardOrgDetailSection returns SCIM info for the org overview tab.
func (p *Plugin) DashboardOrgDetailSection(ctx context.Context, orgID id.OrgID) templ.Component {
	data := scimdash.OrgSectionData{}

	if p.scimStore != nil {
		configs, err := p.scimStore.ListConfigsByOrg(ctx, orgID)
		if err == nil && len(configs) > 0 {
			data.HasConfig = true
			data.ConfigCount = len(configs)
			for _, c := range configs {
				if c.Enabled {
					data.Enabled = true
				}
			}
			// Get last log time for the first config.
			if logs, err := p.scimStore.ListLogs(ctx, configs[0].ID, 1); err == nil && len(logs) > 0 {
				data.LastSync = logs[0].CreatedAt.Format("Jan 02, 2006 15:04")
			}
		}
	}

	return scimdash.OrgSection(data)
}

// ──────────────────────────────────────────────────
// OrgDetailTabContributor implementation
// ──────────────────────────────────────────────────

// DashboardOrgDetailTabs returns SCIM tabs for the org detail page.
func (p *Plugin) DashboardOrgDetailTabs(_ context.Context, _ id.OrgID) []dashboard.OrgDetailTab {
	return []dashboard.OrgDetailTab{
		{
			ID:       "scim",
			Label:    "SCIM",
			Icon:     "shield-check",
			Priority: 15,
			Render:   p.renderOrgSCIMTab,
		},
	}
}

func (p *Plugin) renderOrgSCIMTab(ctx context.Context, orgID id.OrgID) templ.Component {
	data := scimdash.OrgTabData{
		OrgID: orgID.String(),
	}

	if p.scimStore != nil {
		configs, err := p.scimStore.ListConfigsByOrg(ctx, orgID)
		if err == nil {
			for _, cfg := range configs {
				cv := toConfigView(cfg)
				if tokens, err := p.scimStore.ListTokens(ctx, cfg.ID); err == nil {
					cv.TokenCount = len(tokens)
				}
				data.Configs = append(data.Configs, cv)

				// Gather recent logs.
				if logs, err := p.scimStore.ListLogs(ctx, cfg.ID, 10); err == nil {
					for _, l := range logs {
						data.RecentLogs = append(data.RecentLogs, toLogView(l, cfg.Name))
					}
					data.RecentCount += len(logs)
				}
			}
		}
	}

	return scimdash.OrgTab(data)
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderSCIMList(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data scimdash.SCIMListPageData

	// Handle form actions.
	action := params.FormData["action"]
	if action == "create" {
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error, data.Success = p.handleCreateConfig(ctx, appID, params)
		}
	}

	data.FormNonce = dashboard.GenerateNonce()

	// Fetch configs.
	if p.scimStore != nil {
		configs, err := p.scimStore.ListConfigs(ctx, appID)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load SCIM configs: %v", err)
		} else {
			data.TotalConfigs = len(configs)
			data.Configs = make([]scimdash.SCIMConfigView, 0, len(configs))
			for _, cfg := range configs {
				cv := toConfigView(cfg)
				// Resolve org name.
				if !cfg.OrgID.IsNil() && p.authStore != nil {
					if org, err := p.authStore.GetOrganization(ctx, cfg.OrgID); err == nil {
						cv.OrgName = org.Name
					}
				}
				// Count tokens.
				if tokens, err := p.scimStore.ListTokens(ctx, cfg.ID); err == nil {
					cv.TokenCount = len(tokens)
					data.ActiveTokens += len(tokens)
				}
				// Get last activity.
				if logs, err := p.scimStore.ListLogs(ctx, cfg.ID, 1); err == nil && len(logs) > 0 {
					cv.LastActivity = logs[0].CreatedAt.Format("Jan 02 15:04")
				}
				data.Configs = append(data.Configs, cv)
			}
		}

		// Count recent activity.
		if logs, err := p.scimStore.ListAllLogs(ctx, appID, 1000); err == nil {
			data.RecentActivity = len(logs)
		}
	}

	return scimdash.SCIMListPage(data), nil
}

func (p *Plugin) renderSCIMDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	configIDStr := params.QueryParams["id"]
	if configIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	configID, err := id.ParseSCIMConfigID(configIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	cfg, err := p.service.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("scim dashboard: get config: %w", err)
	}

	var data scimdash.SCIMDetailPageData

	// Handle form actions.
	action := params.FormData["action"]
	switch action {
	case "create_token":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error, data.Success, data.NewToken = p.handleCreateToken(ctx, configID, params)
		}
	case "revoke_token":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			p.handleRevokeToken(ctx, params)
		}
	case "update_settings":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error, data.Success = p.handleUpdateConfigSettings(ctx, cfg, params)
			// Re-fetch.
			if updated, fetchErr := p.service.GetConfig(ctx, configID); fetchErr == nil {
				cfg = updated
			}
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.Config = toConfigView(cfg)

	// Resolve org name.
	if !cfg.OrgID.IsNil() && p.authStore != nil {
		if org, orgErr := p.authStore.GetOrganization(ctx, cfg.OrgID); orgErr == nil {
			data.Config.OrgName = org.Name
		}
	}

	// Build endpoint URLs.
	data.BaseURL = p.config.BasePath
	data.UsersEndpoint = p.config.BasePath + "/Users"
	data.GroupsEndpoint = p.config.BasePath + "/Groups"

	// Fetch tokens.
	if tokens, tokErr := p.service.ListTokens(ctx, configID); tokErr == nil {
		data.Tokens = make([]scimdash.SCIMTokenView, 0, len(tokens))
		for _, t := range tokens {
			data.Tokens = append(data.Tokens, toTokenView(t))
		}
	}

	// Fetch log stats.
	success, errors, skipped, err := p.service.CountLogsByStatus(ctx, configID)
	if err == nil {
		data.SuccessCount = success
		data.ErrorCount = errors
		data.SkippedCount = skipped
	}

	// Fetch recent logs.
	if logs, err := p.service.ListLogs(ctx, configID, 20); err == nil {
		data.RecentLogs = make([]scimdash.SCIMLogView, 0, len(logs))
		for _, l := range logs {
			data.RecentLogs = append(data.RecentLogs, toLogView(l, cfg.Name))
		}
	}

	return scimdash.SCIMDetailPage(data), nil
}

func (p *Plugin) renderSCIMLogsPage(ctx context.Context, _ contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data scimdash.SCIMLogsPageData

	if p.scimStore != nil {
		// Fetch all logs.
		logs, err := p.scimStore.ListAllLogs(ctx, appID, 200)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load logs: %v", err)
		} else {
			// Build config name map.
			configNames := make(map[string]string)
			if configs, err := p.scimStore.ListConfigs(ctx, appID); err == nil {
				for _, c := range configs {
					configNames[c.ID.String()] = c.Name
				}
			}

			data.Logs = make([]scimdash.SCIMLogView, 0, len(logs))
			for _, l := range logs {
				lv := toLogView(l, configNames[l.ConfigID.String()])
				data.Logs = append(data.Logs, lv)
			}
		}

		// Fetch log counts.
		success, errors, skipped, err := p.scimStore.CountAllLogsByStatus(ctx, appID)
		if err == nil {
			data.SuccessCount = success
			data.ErrorCount = errors
			data.SkippedCount = skipped
			data.TotalCount = success + errors + skipped
		}
	}

	return scimdash.SCIMLogsPage(data), nil
}

// ──────────────────────────────────────────────────
// Dashboard form handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateConfig(ctx context.Context, appID string, params contributor.Params) (errMsg, successMsg string) {
	name := params.FormData["name"]
	if name == "" {
		return "Name is required.", ""
	}

	parsedAppID, err := id.ParseAppID(appID)
	if err != nil {
		return "Invalid app ID.", ""
	}

	cfg := &SCIMConfig{
		AppID:       parsedAppID,
		Name:        name,
		Enabled:     true,
		AutoCreate:  true,
		AutoSuspend: true,
		DefaultRole: "member",
	}

	orgIDStr := params.FormData["org_id"]
	if orgIDStr != "" {
		orgID, err := id.ParseOrgID(orgIDStr)
		if err != nil {
			return "Invalid organization ID.", ""
		}
		cfg.OrgID = orgID
	}

	if err := p.service.CreateConfig(ctx, cfg); err != nil {
		return fmt.Sprintf("Failed to create config: %v", err), ""
	}

	return "", fmt.Sprintf("SCIM configuration %q created.", name)
}

func (p *Plugin) handleCreateToken(ctx context.Context, configID id.SCIMConfigID, params contributor.Params) (errMsg, successMsg, newToken string) {
	name := params.FormData["token_name"]
	if name == "" {
		return "Token name is required.", "", ""
	}

	var expiresAt *time.Time
	if days := params.FormData["token_expiry_days"]; days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			t := time.Now().AddDate(0, 0, d)
			expiresAt = &t
		}
	}

	plaintext, _, err := p.service.GenerateToken(ctx, configID, name, expiresAt)
	if err != nil {
		return fmt.Sprintf("Failed to generate token: %v", err), "", ""
	}

	return "", "Token generated successfully.", plaintext
}

func (p *Plugin) handleRevokeToken(ctx context.Context, params contributor.Params) {
	tokenIDStr := params.FormData["token_id"]
	if tokenIDStr == "" {
		return
	}
	tokenID, err := id.ParseSCIMTokenID(tokenIDStr)
	if err != nil {
		return
	}
	_ = p.service.RevokeToken(ctx, tokenID)
}

func (p *Plugin) handleUpdateConfigSettings(ctx context.Context, cfg *SCIMConfig, params contributor.Params) (errMsg, successMsg string) {
	cfg.Enabled = params.FormData["enabled"] == "true"
	cfg.AutoCreate = params.FormData["auto_create"] == "true"
	cfg.AutoSuspend = params.FormData["auto_suspend"] == "true"
	cfg.GroupSync = params.FormData["group_sync"] == "true"
	if role := params.FormData["default_role"]; role != "" {
		cfg.DefaultRole = role
	}

	if err := p.service.UpdateConfig(ctx, cfg); err != nil {
		return fmt.Sprintf("Failed to update settings: %v", err), ""
	}

	return "", "Settings updated."
}

// ──────────────────────────────────────────────────
// View conversion helpers
// ──────────────────────────────────────────────────

func (p *Plugin) resolveAppID(ctx context.Context) string {
	appID, ok := dashboard.AppIDFromContext(ctx)
	if ok {
		return appID.String()
	}
	return p.defaultAppID
}

func toConfigView(c *SCIMConfig) scimdash.SCIMConfigView {
	return scimdash.SCIMConfigView{
		ID:          c.ID.String(),
		Name:        c.Name,
		Enabled:     c.Enabled,
		AutoCreate:  c.AutoCreate,
		AutoSuspend: c.AutoSuspend,
		GroupSync:   c.GroupSync,
		DefaultRole: c.DefaultRole,
		OrgID:       c.OrgID.String(),
		AppID:       c.AppID.String(),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toTokenView(t *SCIMToken) scimdash.SCIMTokenView {
	tv := scimdash.SCIMTokenView{
		ID:        t.ID.String(),
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
	if t.LastUsedAt != nil {
		tv.LastUsedAt = t.LastUsedAt.Format("Jan 02, 2006 15:04")
	}
	if t.ExpiresAt != nil {
		tv.ExpiresAt = t.ExpiresAt.Format("Jan 02, 2006")
	}
	return tv
}

func toLogView(l *SCIMProvisionLog, configName string) scimdash.SCIMLogView {
	return scimdash.SCIMLogView{
		ID:           l.ID.String(),
		ConfigID:     l.ConfigID.String(),
		ConfigName:   configName,
		Action:       l.Action,
		ResourceType: l.ResourceType,
		ExternalID:   l.ExternalID,
		InternalID:   l.InternalID,
		Status:       l.Status,
		Detail:       l.Detail,
		CreatedAt:    l.CreatedAt,
	}
}
