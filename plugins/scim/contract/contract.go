// Package contract wires the scim plugin's deep-link settings page
// into the forge-dashboard contract registry. Settings auto-discover
// via settings.tabs; richer SCIM intents (configs / tokens / logs)
// land in a follow-up.
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
		return fmt.Errorf("scim/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "scim/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("scim/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("scim/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("scim/contract: register manifest: %w", err)
	}
	_ = d
	return nil
}
