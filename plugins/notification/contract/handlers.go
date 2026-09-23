package contract

import (
	"context"
	"errors"
	"strings"

	"github.com/xraph/authsome/bridge"
	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

type templateIDInput struct {
	ID string `json:"id"`
}
type previewInput struct {
	ID     string         `json:"id"`
	Locale string         `json:"locale"`
	Data   map[string]any `json:"data"`
}
type createTemplateInput struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Channel  string `json:"channel"`
	Category string `json:"category"`
	Locale   string `json:"locale"`
	Subject  string `json:"subject"`
	Title    string `json:"title"`
	HTML     string `json:"html"`
	Text     string `json:"text"`
}
type updateTemplateInput struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Enabled  bool   `json:"enabled"`
}
type versionInput struct {
	TemplateID string `json:"templateId"`
	ID         string `json:"id"`
	Locale     string `json:"locale"`
	Subject    string `json:"subject"`
	Title      string `json:"title"`
	HTML       string `json:"html"`
	Text       string `json:"text"`
	Active     bool   `json:"active"`
}
type sendInput struct {
	ID        string         `json:"id"`
	Recipient string         `json:"recipient"`
	Locale    string         `json:"locale"`
	Data      map[string]any `json:"data"`
	Test      bool           `json:"test"`
}
type ack struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}
type templatesResponse struct {
	Templates []*bridge.HeraldTemplate `json:"templates"`
}
type MappingSummary struct {
	Action   string   `json:"action"`
	Template string   `json:"template"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}
type mappingsResponse struct {
	Mappings []MappingSummary `json:"mappings"`
}

func mappingsList(deps Deps) func(context.Context, struct{}, contract.Principal) (mappingsResponse, error) {
	return func(_ context.Context, _ struct{}, _ contract.Principal) (mappingsResponse, error) {
		if deps.Mappings == nil {
			return mappingsResponse{Mappings: []MappingSummary{}}, nil
		}
		return mappingsResponse{Mappings: deps.Mappings()}, nil
	}
}

func invalid(message string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: message}
}
func missing() error {
	return &contract.Error{Code: contract.CodeNotFound, Message: "template not found"}
}
func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "notification service not configured"}
}
func failure(err error) error {
	if err == nil {
		return nil
	}
	var known *contract.Error
	if errors.As(err, &known) {
		return known
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
func manager(deps Deps) (bridge.HeraldTemplateManager, error) {
	if deps.Manager == nil {
		return nil, unavailable()
	}
	m := deps.Manager()
	if m == nil {
		return nil, unavailable()
	}
	return m, nil
}
func appID(deps Deps, p contract.Principal) string {
	return authcontract.AppIDFromPrincipal(p, deps.Engine).String()
}
func scopedTemplate(ctx context.Context, deps Deps, p contract.Principal, id string) (*bridge.HeraldTemplate, error) {
	if strings.TrimSpace(id) == "" {
		return nil, invalid("id is required")
	}
	m, err := manager(deps)
	if err != nil {
		return nil, err
	}
	t, err := m.GetTemplate(ctx, id)
	if err != nil || t == nil {
		return nil, missing()
	}
	if t.AppID != "" && t.AppID != appID(deps, p) {
		return nil, missing()
	}
	return t, nil
}
func templatesList(deps Deps) func(context.Context, struct{}, contract.Principal) (templatesResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (templatesResponse, error) {
		m, err := manager(deps)
		if err != nil {
			return templatesResponse{}, err
		}
		list, err := m.ListTemplates(ctx, appID(deps, p))
		if err != nil {
			return templatesResponse{}, failure(err)
		}
		if list == nil {
			list = []*bridge.HeraldTemplate{}
		}
		return templatesResponse{Templates: list}, nil
	}
}
func templateDetail(deps Deps) func(context.Context, templateIDInput, contract.Principal) (*bridge.HeraldTemplate, error) {
	return func(ctx context.Context, in templateIDInput, p contract.Principal) (*bridge.HeraldTemplate, error) {
		return scopedTemplate(ctx, deps, p, in.ID)
	}
}
func templatePreview(deps Deps) func(context.Context, previewInput, contract.Principal) (*bridge.HeraldRenderedContent, error) {
	return func(ctx context.Context, in previewInput, p contract.Principal) (*bridge.HeraldRenderedContent, error) {
		if _, err := scopedTemplate(ctx, deps, p, in.ID); err != nil {
			return nil, err
		}
		m, _ := manager(deps)
		out, err := m.RenderTemplate(ctx, in.ID, in.Locale, in.Data)
		return out, failure(err)
	}
}
func templateCreate(deps Deps) func(context.Context, createTemplateInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in createTemplateInput, p contract.Principal) (ack, error) {
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Slug) == "" {
			return ack{}, invalid("name and slug are required")
		}
		switch in.Channel {
		case "email", "sms", "inapp", "push":
		default:
			return ack{}, invalid("invalid channel")
		}
		m, err := manager(deps)
		if err != nil {
			return ack{}, err
		}
		locale := strings.TrimSpace(in.Locale)
		if locale == "" {
			locale = "en"
		}
		category := strings.TrimSpace(in.Category)
		if category == "" {
			category = bridge.HeraldCategoryTransactional
		}
		t := &bridge.HeraldTemplate{AppID: appID(deps, p), Name: strings.TrimSpace(in.Name), Slug: strings.TrimSpace(in.Slug), Channel: in.Channel, Category: category, Enabled: true,
			Versions: []bridge.HeraldTemplateVersion{{Locale: locale, Subject: in.Subject, Title: in.Title, HTML: in.HTML, Text: in.Text, Active: true}}}
		if err := m.CreateTemplate(ctx, t); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: t.ID}, nil
	}
}
func templateUpdate(deps Deps) func(context.Context, updateTemplateInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in updateTemplateInput, p contract.Principal) (ack, error) {
		t, err := scopedTemplate(ctx, deps, p, in.ID)
		if err != nil {
			return ack{}, err
		}
		if strings.TrimSpace(in.Name) == "" {
			return ack{}, invalid("name is required")
		}
		t.Name = strings.TrimSpace(in.Name)
		t.Category = strings.TrimSpace(in.Category)
		t.Enabled = in.Enabled
		m, _ := manager(deps)
		if err := m.UpdateTemplate(ctx, t); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: t.ID}, nil
	}
}
func templateDelete(deps Deps) func(context.Context, templateIDInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in templateIDInput, p contract.Principal) (ack, error) {
		t, err := scopedTemplate(ctx, deps, p, in.ID)
		if err != nil {
			return ack{}, err
		}
		if t.IsSystem {
			return ack{}, invalid("system templates cannot be deleted")
		}
		m, _ := manager(deps)
		if err := m.DeleteTemplate(ctx, in.ID); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: in.ID}, nil
	}
}
func versionFor(t *bridge.HeraldTemplate, id string) bool {
	for _, version := range t.Versions {
		if version.ID == id {
			return true
		}
	}
	return false
}
func versionCreate(deps Deps) func(context.Context, versionInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in versionInput, p contract.Principal) (ack, error) {
		if _, err := scopedTemplate(ctx, deps, p, in.TemplateID); err != nil {
			return ack{}, err
		}
		if strings.TrimSpace(in.Locale) == "" {
			return ack{}, invalid("locale is required")
		}
		m, _ := manager(deps)
		v := &bridge.HeraldTemplateVersion{TemplateID: in.TemplateID, Locale: strings.TrimSpace(in.Locale), Subject: in.Subject, Title: in.Title, HTML: in.HTML, Text: in.Text, Active: true}
		if err := m.CreateVersion(ctx, v); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: v.ID}, nil
	}
}
func versionUpdate(deps Deps) func(context.Context, versionInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in versionInput, p contract.Principal) (ack, error) {
		t, err := scopedTemplate(ctx, deps, p, in.TemplateID)
		if err != nil {
			return ack{}, err
		}
		if !versionFor(t, in.ID) {
			return ack{}, missing()
		}
		if strings.TrimSpace(in.Locale) == "" {
			return ack{}, invalid("locale is required")
		}
		m, _ := manager(deps)
		v := &bridge.HeraldTemplateVersion{ID: in.ID, TemplateID: in.TemplateID, Locale: in.Locale, Subject: in.Subject, Title: in.Title, HTML: in.HTML, Text: in.Text, Active: in.Active}
		if err := m.UpdateVersion(ctx, v); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: in.ID}, nil
	}
}
func versionDelete(deps Deps) func(context.Context, versionInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in versionInput, p contract.Principal) (ack, error) {
		t, err := scopedTemplate(ctx, deps, p, in.TemplateID)
		if err != nil {
			return ack{}, err
		}
		if !versionFor(t, in.ID) {
			return ack{}, missing()
		}
		m, _ := manager(deps)
		if err := m.DeleteVersion(ctx, in.ID); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true, ID: in.ID}, nil
	}
}
func sendNotification(deps Deps) func(context.Context, sendInput, contract.Principal) (ack, error) {
	return func(ctx context.Context, in sendInput, p contract.Principal) (ack, error) {
		t, err := scopedTemplate(ctx, deps, p, in.ID)
		if err != nil {
			return ack{}, err
		}
		if !t.Enabled {
			return ack{}, invalid("template is disabled")
		}
		recipient := strings.TrimSpace(in.Recipient)
		if recipient == "" {
			return ack{}, invalid("recipient is required")
		}
		req := &bridge.HeraldSendRequest{AppID: appID(deps, p), Channel: t.Channel, Template: t.Slug, Locale: in.Locale, To: []string{recipient}, Data: in.Data}
		if in.Test {
			m, _ := manager(deps)
			req.Metadata = map[string]string{"test": "true"}
			if err := m.TestSend(ctx, req); err != nil {
				return ack{}, failure(err)
			}
		} else {
			if deps.Sender == nil || deps.Sender() == nil {
				return ack{}, unavailable()
			}
			if err := deps.Sender().Send(ctx, req); err != nil {
				return ack{}, failure(err)
			}
		}
		return ack{OK: true, ID: t.ID}, nil
	}
}
func resetDefaultTemplates(deps Deps) func(context.Context, struct{}, contract.Principal) (ack, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (ack, error) {
		m, err := manager(deps)
		if err != nil {
			return ack{}, err
		}
		if err := m.ResetDefaultTemplates(ctx, appID(deps, p)); err != nil {
			return ack{}, failure(err)
		}
		return ack{OK: true}, nil
	}
}
