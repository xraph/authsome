// Package heraldadapter bridges AuthSome notification requests to the Herald extension.
package heraldadapter

import (
	"context"
	"fmt"

	"github.com/xraph/herald"
	"github.com/xraph/herald/id"
	"github.com/xraph/herald/template"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome notification requests to Herald engine calls.
type Adapter struct {
	h        *herald.Herald
	renderer *template.Renderer
}

// New creates a Herald bridge adapter.
func New(h *herald.Herald) *Adapter {
	return &Adapter{
		h:        h,
		renderer: template.NewRenderer(),
	}
}

// Send implements bridge.Herald.
func (a *Adapter) Send(ctx context.Context, req *bridge.HeraldSendRequest) error {
	_, err := a.h.Send(ctx, &herald.SendRequest{
		AppID:    req.AppID,
		EnvID:    req.EnvID,
		OrgID:    req.OrgID,
		UserID:   req.UserID,
		Channel:  req.Channel,
		Template: req.Template,
		Locale:   req.Locale,
		To:       req.To,
		Data:     req.Data,
		Async:    req.Async,
		Metadata: req.Metadata,
	})
	return err
}

// Notify implements bridge.Herald.
func (a *Adapter) Notify(ctx context.Context, req *bridge.HeraldNotifyRequest) error {
	_, err := a.h.Notify(ctx, &herald.NotifyRequest{
		AppID:    req.AppID,
		EnvID:    req.EnvID,
		OrgID:    req.OrgID,
		UserID:   req.UserID,
		Template: req.Template,
		Locale:   req.Locale,
		To:       req.To,
		Data:     req.Data,
		Channels: req.Channels,
		Async:    req.Async,
		Metadata: req.Metadata,
	})
	return err
}

// ──────────────────────────────────────────────────
// HeraldTemplateManager implementation
// ──────────────────────────────────────────────────

// ListTemplates implements bridge.HeraldTemplateManager.
func (a *Adapter) ListTemplates(ctx context.Context, appID string) ([]*bridge.HeraldTemplate, error) {
	templates, err := a.h.Store().ListTemplates(ctx, appID)
	if err != nil {
		return nil, err
	}
	result := make([]*bridge.HeraldTemplate, len(templates))
	for i, t := range templates {
		result[i] = templateToBridge(t)
	}
	return result, nil
}

// GetTemplate implements bridge.HeraldTemplateManager.
func (a *Adapter) GetTemplate(ctx context.Context, templateID string) (*bridge.HeraldTemplate, error) {
	tid, err := id.ParseTemplateID(templateID)
	if err != nil {
		return nil, fmt.Errorf("heraldadapter: invalid template ID: %w", err)
	}
	t, err := a.h.Store().GetTemplate(ctx, tid)
	if err != nil {
		return nil, err
	}
	bt := templateToBridge(t)
	// Load versions.
	versions, err := a.h.Store().ListVersions(ctx, tid)
	if err != nil {
		return nil, err
	}
	bt.Versions = make([]bridge.HeraldTemplateVersion, len(versions))
	for i, v := range versions {
		bt.Versions[i] = versionToBridge(v)
	}
	return bt, nil
}

// CreateTemplate implements bridge.HeraldTemplateManager.
func (a *Adapter) CreateTemplate(ctx context.Context, t *bridge.HeraldTemplate) error {
	tmpl := templateFromBridge(t)
	if tmpl.ID.IsNil() {
		tmpl.ID = id.NewTemplateID()
	}
	if err := a.h.Store().CreateTemplate(ctx, tmpl); err != nil {
		return err
	}
	t.ID = tmpl.ID.String()
	// Create versions.
	for i := range t.Versions {
		v := versionFromBridge(&t.Versions[i])
		v.TemplateID = tmpl.ID
		if v.ID.IsNil() {
			v.ID = id.NewTemplateVersionID()
		}
		if err := a.h.Store().CreateVersion(ctx, v); err != nil {
			return err
		}
		t.Versions[i].ID = v.ID.String()
		t.Versions[i].TemplateID = tmpl.ID.String()
	}
	return nil
}

// UpdateTemplate implements bridge.HeraldTemplateManager.
func (a *Adapter) UpdateTemplate(ctx context.Context, t *bridge.HeraldTemplate) error {
	return a.h.Store().UpdateTemplate(ctx, templateFromBridge(t))
}

// DeleteTemplate implements bridge.HeraldTemplateManager.
func (a *Adapter) DeleteTemplate(ctx context.Context, templateID string) error {
	tid, err := id.ParseTemplateID(templateID)
	if err != nil {
		return fmt.Errorf("heraldadapter: invalid template ID: %w", err)
	}
	return a.h.Store().DeleteTemplate(ctx, tid)
}

// CreateVersion implements bridge.HeraldTemplateManager.
func (a *Adapter) CreateVersion(ctx context.Context, v *bridge.HeraldTemplateVersion) error {
	ver := versionFromBridge(v)
	if ver.ID.IsNil() {
		ver.ID = id.NewTemplateVersionID()
	}
	if err := a.h.Store().CreateVersion(ctx, ver); err != nil {
		return err
	}
	v.ID = ver.ID.String()
	return nil
}

// UpdateVersion implements bridge.HeraldTemplateManager.
func (a *Adapter) UpdateVersion(ctx context.Context, v *bridge.HeraldTemplateVersion) error {
	return a.h.Store().UpdateVersion(ctx, versionFromBridge(v))
}

// DeleteVersion implements bridge.HeraldTemplateManager.
func (a *Adapter) DeleteVersion(ctx context.Context, versionID string) error {
	vid, err := id.ParseTemplateVersionID(versionID)
	if err != nil {
		return fmt.Errorf("heraldadapter: invalid version ID: %w", err)
	}
	return a.h.Store().DeleteVersion(ctx, vid)
}

// RenderTemplate implements bridge.HeraldTemplateManager.
func (a *Adapter) RenderTemplate(ctx context.Context, templateID, locale string, data map[string]any) (*bridge.HeraldRenderedContent, error) {
	tid, err := id.ParseTemplateID(templateID)
	if err != nil {
		return nil, fmt.Errorf("heraldadapter: invalid template ID: %w", err)
	}
	t, err := a.h.Store().GetTemplate(ctx, tid)
	if err != nil {
		return nil, err
	}
	// Load versions into template for rendering.
	versions, err := a.h.Store().ListVersions(ctx, tid)
	if err != nil {
		return nil, err
	}
	t.Versions = make([]template.Version, len(versions))
	for i, v := range versions {
		t.Versions[i] = *v
	}
	rendered, err := a.renderer.Render(t, locale, data)
	if err != nil {
		return nil, err
	}
	return &bridge.HeraldRenderedContent{
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
		Text:    rendered.Text,
		Title:   rendered.Title,
	}, nil
}

// TestSend implements bridge.HeraldTemplateManager.
func (a *Adapter) TestSend(ctx context.Context, req *bridge.HeraldSendRequest) error {
	return a.Send(ctx, req)
}

// SeedDefaultTemplates implements bridge.HeraldTemplateManager.
func (a *Adapter) SeedDefaultTemplates(ctx context.Context, appID string) error {
	return a.h.SeedDefaultTemplates(ctx, appID)
}

// ResetDefaultTemplates implements bridge.HeraldTemplateManager.
func (a *Adapter) ResetDefaultTemplates(ctx context.Context, appID string) error {
	return a.h.ResetDefaultTemplates(ctx, appID)
}

// ──────────────────────────────────────────────────
// Conversion helpers
// ──────────────────────────────────────────────────

func templateToBridge(t *template.Template) *bridge.HeraldTemplate {
	bt := &bridge.HeraldTemplate{
		ID:        t.ID.String(),
		AppID:     t.AppID,
		Slug:      t.Slug,
		Name:      t.Name,
		Channel:   t.Channel,
		Category:  t.Category,
		IsSystem:  t.IsSystem,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
	if len(t.Variables) > 0 {
		bt.Variables = make([]bridge.HeraldTemplateVariable, len(t.Variables))
		for i, v := range t.Variables {
			bt.Variables[i] = bridge.HeraldTemplateVariable{
				Name:        v.Name,
				Type:        v.Type,
				Required:    v.Required,
				Default:     v.Default,
				Description: v.Description,
			}
		}
	}
	if len(t.Versions) > 0 {
		bt.Versions = make([]bridge.HeraldTemplateVersion, len(t.Versions))
		for i, v := range t.Versions {
			bt.Versions[i] = versionToBridge(&v)
		}
	}
	return bt
}

func versionToBridge(v *template.Version) bridge.HeraldTemplateVersion {
	return bridge.HeraldTemplateVersion{
		ID:         v.ID.String(),
		TemplateID: v.TemplateID.String(),
		Locale:     v.Locale,
		Subject:    v.Subject,
		HTML:       v.HTML,
		Text:       v.Text,
		Title:      v.Title,
		Active:     v.Active,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
}

func templateFromBridge(t *bridge.HeraldTemplate) *template.Template {
	tmpl := &template.Template{
		AppID:    t.AppID,
		Slug:     t.Slug,
		Name:     t.Name,
		Channel:  t.Channel,
		Category: t.Category,
		IsSystem: t.IsSystem,
		Enabled:  t.Enabled,
	}
	if t.ID != "" {
		tmpl.ID, _ = id.ParseTemplateID(t.ID)
	}
	if len(t.Variables) > 0 {
		tmpl.Variables = make([]template.Variable, len(t.Variables))
		for i, v := range t.Variables {
			tmpl.Variables[i] = template.Variable{
				Name:        v.Name,
				Type:        v.Type,
				Required:    v.Required,
				Default:     v.Default,
				Description: v.Description,
			}
		}
	}
	return tmpl
}

func versionFromBridge(v *bridge.HeraldTemplateVersion) *template.Version {
	ver := &template.Version{
		Locale:  v.Locale,
		Subject: v.Subject,
		HTML:    v.HTML,
		Text:    v.Text,
		Title:   v.Title,
		Active:  v.Active,
	}
	if v.ID != "" {
		ver.ID, _ = id.ParseTemplateVersionID(v.ID)
	}
	if v.TemplateID != "" {
		ver.TemplateID, _ = id.ParseTemplateID(v.TemplateID)
	}
	return ver
}

// Compile-time checks.
var (
	_ bridge.Herald                = (*Adapter)(nil)
	_ bridge.HeraldTemplateManager = (*Adapter)(nil)
)
