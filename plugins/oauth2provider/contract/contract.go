// Package contract wires the oauth2provider plugin's deep-link
// settings page into the forge-dashboard contract registry.
//
// Future iterations should add intents for managing OAuth2 client
// records (oauth2.clients, oauth2.clientCreate, oauth2.clientUpdate,
// oauth2.clientDelete) — those are distinct from settings because each
// client is a row in the engine's oauth2 client store, not a knob in
// the settings manager.
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

type Deps struct {
	Engine *authsome.Engine
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("oauth2provider/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "oauth2provider/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("oauth2provider/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("oauth2provider/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("oauth2provider/contract: register manifest: %w", err)
	}
	_ = d
	return nil
}
