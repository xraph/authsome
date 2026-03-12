package notification

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/dashboard"
	notifydash "github.com/xraph/authsome/plugins/notification/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns no widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the notification settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return notifydash.SettingsPanel(p.config.AppName, p.config.BaseURL, p.config.DefaultLocale, p.config.Async)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the notifications page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Notifications",
			Path:     "/notifications",
			Icon:     "bell",
			Group:    "Configuration",
			Priority: 20,
		},
	}
}

// DashboardRenderPage renders notification dashboard pages.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch route {
	case "/notifications":
		return p.renderOverview(ctx, params)
	case "/notifications/templates":
		return p.renderTemplateList(ctx, params)
	case "/notifications/templates/detail":
		return p.renderTemplateDetail(ctx, params)
	case "/notifications/templates/create":
		return p.renderTemplateCreate(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderOverview(ctx context.Context, _ contributor.Params) (templ.Component, error) {
	// Count templates by channel if template manager is available.
	var stats notifydash.OverviewStats
	stats.MappingsCount = len(p.mappings)

	if p.templates != nil {
		appID := p.appIDFromContext(ctx)
		templates, err := p.templates.ListTemplates(ctx, appID)
		if err == nil {
			stats.TotalTemplates = len(templates)
			channels := make(map[string]int)
			for _, t := range templates {
				channels[t.Channel]++
			}
			stats.EmailCount = channels["email"]
			stats.SMSCount = channels["sms"]
			stats.InAppCount = channels["inapp"]
			stats.PushCount = channels["push"]
		}
	}

	mappingNames := make([]string, 0, len(p.mappings))
	for action := range p.mappings {
		mappingNames = append(mappingNames, action)
	}
	sort.Strings(mappingNames)

	return notifydash.OverviewPage(notifydash.OverviewData{
		AppName:      p.config.AppName,
		BaseURL:      p.config.BaseURL,
		Locale:       p.config.DefaultLocale,
		Async:        p.config.Async,
		MappingNames: mappingNames,
		Stats:        stats,
	}), nil
}

func (p *Plugin) renderTemplateList(ctx context.Context, params contributor.Params) (templ.Component, error) {
	if p.templates == nil {
		return notifydash.TemplatesListPage(notifydash.TemplatesListData{}), nil
	}

	appID := p.appIDFromContext(ctx)

	// Handle reset defaults action (POST).
	var successMsg, errorMsg string
	action := params.FormData["action"]
	if action == "reset_defaults" {
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			if err := p.templates.ResetDefaultTemplates(ctx, appID); err != nil {
				errorMsg = fmt.Sprintf("Failed to reset templates: %v", err)
			} else {
				successMsg = "Default templates restored successfully."
			}
		}
	}

	channel := params.QueryParams["channel"]

	templates, err := p.templates.ListTemplates(ctx, appID)
	if err != nil {
		return notifydash.TemplatesListPage(notifydash.TemplatesListData{
			Error: fmt.Sprintf("Failed to load templates: %v", err),
		}), nil
	}

	// Filter by channel if specified.
	if channel != "" && channel != "all" {
		filtered := make([]*bridge.HeraldTemplate, 0, len(templates))
		for _, t := range templates {
			if t.Channel == channel {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}

	return notifydash.TemplatesListPage(notifydash.TemplatesListData{
		Templates:     templates,
		ActiveChannel: channel,
		SuccessMsg:    successMsg,
		ErrorMsg:      errorMsg,
		FormNonce:     dashboard.GenerateNonce(),
	}), nil
}

func (p *Plugin) renderTemplateDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	templateID := params.QueryParams["id"]
	if templateID == "" {
		return nil, contributor.ErrPageNotFound
	}

	if p.templates == nil {
		return nil, bridge.ErrHeraldNotAvailable
	}

	// Handle form actions (POST).
	action := params.FormData["action"]
	var actionMsg, actionErr string

	switch action {
	case "update_version":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			actionErr = p.handleUpdateVersion(ctx, params)
			if actionErr == "" {
				actionMsg = "Version updated successfully."
			}
		}
	case "create_version":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			actionErr = p.handleCreateVersion(ctx, params, templateID)
			if actionErr == "" {
				actionMsg = "Version created successfully."
			}
		}
	case "delete_version":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			versionID := params.FormData["version_id"]
			if err := p.templates.DeleteVersion(ctx, versionID); err != nil {
				actionErr = fmt.Sprintf("Failed to delete version: %v", err)
			} else {
				actionMsg = "Version deleted."
			}
		}
	case "toggle_enabled":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			actionErr = p.handleToggleEnabled(ctx, templateID)
			if actionErr == "" {
				actionMsg = "Template status updated."
			}
		}
	case "test_send":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			actionErr = p.handleTestSend(ctx, params, templateID)
			if actionErr == "" {
				actionMsg = "Test notification sent."
			}
		}
	}

	tmpl, err := p.templates.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("notification dashboard: resolve template: %w", err)
	}

	// Preview rendering (non-destructive, uses sample data).
	var preview *bridge.HeraldRenderedContent
	activeLocale := params.QueryParams["locale"]
	if activeLocale == "" && len(tmpl.Versions) > 0 {
		activeLocale = tmpl.Versions[0].Locale
	}
	sampleData := buildSampleData(tmpl.Variables)
	preview, _ = p.templates.RenderTemplate(ctx, templateID, activeLocale, sampleData)

	return notifydash.TemplateDetailPage(notifydash.TemplateDetailData{
		Template:     tmpl,
		Preview:      preview,
		ActiveLocale: activeLocale,
		FormNonce:    dashboard.GenerateNonce(),
		SuccessMsg:   actionMsg,
		ErrorMsg:     actionErr,
	}), nil
}

func (p *Plugin) renderTemplateCreate(ctx context.Context, params contributor.Params) (templ.Component, error) {
	if p.templates == nil {
		return nil, bridge.ErrHeraldNotAvailable
	}

	var data notifydash.TemplateCreateData
	data.FormNonce = dashboard.GenerateNonce()

	action := params.FormData["action"]
	if action == "create" {
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			created, errMsg := p.handleCreateTemplate(ctx, params)
			if errMsg != "" {
				data.Error = errMsg
			} else {
				data.CreatedID = created.ID
				data.SuccessMsg = fmt.Sprintf("Template %q created.", created.Name)
			}
			// Regenerate nonce for next submit.
			data.FormNonce = dashboard.GenerateNonce()
		}
	}

	return notifydash.TemplateCreatePage(data), nil
}

// ──────────────────────────────────────────────────
// Form action handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleUpdateVersion(ctx context.Context, params contributor.Params) string {
	versionID := params.FormData["version_id"]
	if versionID == "" {
		return "Version ID is required."
	}

	v := &bridge.HeraldTemplateVersion{
		ID:         versionID,
		TemplateID: params.FormData["template_id"],
		Locale:     params.FormData["locale"],
		Subject:    params.FormData["subject"],
		HTML:       params.FormData["html"],
		Text:       params.FormData["text"],
		Title:      params.FormData["title"],
		Active:     params.FormData["active"] == "on",
	}

	if err := p.templates.UpdateVersion(ctx, v); err != nil {
		return fmt.Sprintf("Failed to update version: %v", err)
	}
	return ""
}

func (p *Plugin) handleCreateVersion(ctx context.Context, params contributor.Params, templateID string) string {
	locale := strings.TrimSpace(params.FormData["locale"])
	if locale == "" {
		return "Locale is required."
	}

	v := &bridge.HeraldTemplateVersion{
		TemplateID: templateID,
		Locale:     locale,
		Subject:    params.FormData["subject"],
		HTML:       params.FormData["html"],
		Text:       params.FormData["text"],
		Title:      params.FormData["title"],
		Active:     true,
	}

	if err := p.templates.CreateVersion(ctx, v); err != nil {
		return fmt.Sprintf("Failed to create version: %v", err)
	}
	return ""
}

func (p *Plugin) handleToggleEnabled(ctx context.Context, templateID string) string {
	tmpl, err := p.templates.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Sprintf("Failed to load template: %v", err)
	}
	tmpl.Enabled = !tmpl.Enabled
	if err := p.templates.UpdateTemplate(ctx, tmpl); err != nil {
		return fmt.Sprintf("Failed to update template: %v", err)
	}
	return ""
}

func (p *Plugin) handleTestSend(ctx context.Context, params contributor.Params, templateID string) string {
	recipient := strings.TrimSpace(params.FormData["recipient"])
	if recipient == "" {
		return "Recipient is required."
	}

	tmpl, err := p.templates.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Sprintf("Failed to load template: %v", err)
	}

	sampleData := buildSampleData(tmpl.Variables)

	appID := p.appIDFromContext(ctx)
	if err := p.templates.TestSend(ctx, &bridge.HeraldSendRequest{
		AppID:    appID,
		Channel:  tmpl.Channel,
		Template: tmpl.Slug,
		To:       []string{recipient},
		Data:     sampleData,
		Metadata: map[string]string{"test": "true"},
	}); err != nil {
		return fmt.Sprintf("Test send failed: %v", err)
	}
	return ""
}

func (p *Plugin) handleCreateTemplate(ctx context.Context, params contributor.Params) (*bridge.HeraldTemplate, string) {
	name := strings.TrimSpace(params.FormData["name"])
	slug := strings.TrimSpace(params.FormData["slug"])
	channel := strings.TrimSpace(params.FormData["channel"])
	category := strings.TrimSpace(params.FormData["category"])

	if name == "" || slug == "" || channel == "" {
		return nil, "Name, slug, and channel are required."
	}
	if category == "" {
		category = bridge.HeraldCategoryTransactional
	}

	appID := p.appIDFromContext(ctx)
	tmpl := &bridge.HeraldTemplate{
		AppID:    appID,
		Slug:     slug,
		Name:     name,
		Channel:  channel,
		Category: category,
		Enabled:  true,
		Versions: []bridge.HeraldTemplateVersion{{
			Locale:  "en",
			Subject: params.FormData["subject"],
			HTML:    params.FormData["html"],
			Text:    params.FormData["text"],
			Title:   params.FormData["title"],
			Active:  true,
		}},
	}

	if err := p.templates.CreateTemplate(ctx, tmpl); err != nil {
		return nil, fmt.Sprintf("Failed to create template: %v", err)
	}
	return tmpl, ""
}

// appIDFromContext extracts the app ID string from the dashboard context.
func (p *Plugin) appIDFromContext(ctx context.Context) string {
	if appID, ok := dashboard.AppIDFromContext(ctx); ok {
		return appID.String()
	}
	return ""
}

// buildSampleData generates sample data for template preview based on variable definitions.
func buildSampleData(vars []bridge.HeraldTemplateVariable) map[string]any {
	data := make(map[string]any, len(vars)+2)
	for _, v := range vars {
		if v.Default != "" {
			data[v.Name] = v.Default
		} else {
			switch v.Type {
			case "url":
				data[v.Name] = "https://example.com/" + v.Name
			default:
				data[v.Name] = "Sample " + v.Name
			}
		}
	}
	// Always include common variables.
	if _, ok := data["app_name"]; !ok {
		data["app_name"] = "MyApp"
	}
	return data
}
