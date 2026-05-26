// Package contract wires the ipreputation plugin's deep-link settings page
// into the forge-dashboard contract registry. Settings flow through
// the auth contributor's settings.tabs auto-discovery; no
// plugin-owned intents yet. Add domain-specific intents (e.g. a
// list of recent events) here when surfaces beyond settings need to
// bind to plugin data.
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
		return fmt.Errorf("ipreputation/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "ipreputation/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("ipreputation/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("ipreputation/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("ipreputation/contract: register manifest: %w", err)
	}
	_ = d
	return nil
}
