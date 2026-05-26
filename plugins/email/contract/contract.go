// Package contract wires the email plugin's deep-link settings page
// into the forge-dashboard contract registry. The plugin owns no
// intents of its own — its settings (email.from_address, email.app_name,
// email.base_url) flow through the auth contributor's
// settings.namespace / settings.update intents via auto-discovery.
//
// Wired from plugins/email/contract.go (Plugin.RegisterContract method),
// which authsome's RegisterContractContributor loop invokes after the
// auth contributor itself registers.
package contract

import (
	"bytes"
	_ "embed"
	"fmt"

	authsome "github.com/xraph/authsome"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// Deps is the typed dependency bundle. Engine is required for symmetry
// with other plugin contracts even though the email surface doesn't
// touch it directly — future intents (e.g. email.lastSend, a status
// query) will need it.
type Deps struct {
	Engine *authsome.Engine
}

// Register loads the manifest, validates it, registers it against the
// dashboard's contract registry, and (since this plugin currently has
// no intents) returns. Future intent registrations belong here.
func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("email/contract: Engine is required")
	}

	m, err := loader.Load(bytes.NewReader(manifestYAML), "email/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("email/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("email/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("email/contract: register manifest: %w", err)
	}

	// No intent handlers — settings flow through the auth contributor.
	_ = d
	return nil
}
