package contract

import (
	"bytes"
	"context"
	"errors"
	"testing"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

type managerStub struct {
	bridge.HeraldTemplateManager
	template    *bridge.HeraldTemplate
	testRequest *bridge.HeraldSendRequest
}

func (m *managerStub) GetTemplate(_ context.Context, _ string) (*bridge.HeraldTemplate, error) {
	return m.template, nil
}
func (m *managerStub) TestSend(_ context.Context, req *bridge.HeraldSendRequest) error {
	m.testRequest = req
	return nil
}

type senderStub struct {
	bridge.Herald
	request *bridge.HeraldSendRequest
}

func (s *senderStub) Send(_ context.Context, req *bridge.HeraldSendRequest) error {
	s.request = req
	return nil
}

func TestSendNotificationUsesScopedTemplateAndTestTransport(t *testing.T) {
	app := id.NewAppID().String()
	other := id.NewAppID().String()
	m := &managerStub{template: &bridge.HeraldTemplate{ID: "template-1", AppID: app, Slug: "welcome", Channel: "email", Enabled: true}}
	s := &senderStub{}
	deps := Deps{Engine: &authsome.Engine{}, Manager: func() bridge.HeraldTemplateManager { return m }, Sender: func() bridge.Herald { return s }}
	p := contract.Principal{Claims: map[string]any{"app_id": other}}
	_, err := sendNotification(deps)(context.Background(), sendInput{ID: "template-1", Recipient: "user@example.com"}, p)
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeNotFound {
		t.Fatalf("cross-app template should be hidden, got %v", err)
	}
	if s.request != nil {
		t.Fatal("cross-app notification was sent")
	}
	p.Claims["app_id"] = app
	result, err := sendNotification(deps)(context.Background(), sendInput{ID: "template-1", Recipient: "user@example.com", Locale: "en", Data: map[string]any{"name": "Ada"}, Test: true}, p)
	if err != nil || !result.OK {
		t.Fatalf("test send: result=%+v err=%v", result, err)
	}
	if m.testRequest == nil || m.testRequest.AppID != app || m.testRequest.Template != "welcome" || m.testRequest.Channel != "email" || len(m.testRequest.To) != 1 || m.testRequest.To[0] != "user@example.com" {
		t.Fatalf("unexpected test send request: %+v", m.testRequest)
	}
	if s.request != nil {
		t.Fatal("test send reached live sender")
	}
}

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "notification/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "notification" {
		t.Errorf("contributor name = %q, want notification", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 12 {
		t.Errorf("intents = %d, want 12", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "notification/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}
