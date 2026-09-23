// Package contract wires the notification plugin's deep-link settings
// page into the forge-dashboard contract registry.
package contract

import (
	"bytes"
	_ "embed"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

type Deps struct {
	Engine   *authsome.Engine
	Manager  func() bridge.HeraldTemplateManager
	Sender   func() bridge.Herald
	Mappings func() []MappingSummary
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("notification/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "notification/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("notification/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("notification/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("notification/contract: register manifest: %w", err)
	}
	const c = "notification"
	if err := dispatcher.RegisterQuery(d, c, "notification.templates.list", 1, templatesList(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterQuery(d, c, "notification.templates.detail", 1, templateDetail(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterQuery(d, c, "notification.templates.preview", 1, templatePreview(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.templates.create", 1, templateCreate(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.templates.update", 1, templateUpdate(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.templates.delete", 1, templateDelete(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.versions.create", 1, versionCreate(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.versions.update", 1, versionUpdate(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.versions.delete", 1, versionDelete(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.send", 1, sendNotification(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterCommand(d, c, "notification.templates.resetDefaults", 1, resetDefaultTemplates(deps)); err != nil {
		return err
	}
	if err := dispatcher.RegisterQuery(d, c, "notification.mappings.list", 1, mappingsList(deps)); err != nil {
		return err
	}
	return nil
}
