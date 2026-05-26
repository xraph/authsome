// Package contract wires the apikey plugin's intent surface into the
// forge-dashboard contract registry.
//
// The /apikeys list+detail page stays on the auth contributor (it's a
// platform-level navigation entry, not a plugin-deep-link page). This
// package only owns the four intent handlers + their manifest
// declarations. Cross-contributor invocations are name-only on the
// wire so the auth contributor's manifest can reference apikeys.list
// without re-declaring it.
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
		return fmt.Errorf("apikey/contract: Engine is required")
	}

	m, err := loader.Load(bytes.NewReader(manifestYAML), "apikey/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("apikey/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("apikey/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("apikey/contract: register manifest: %w", err)
	}

	const c = "apikey"
	if err := dispatcher.RegisterQuery(d, c, "apikeys.list", 1, apikeysListHandler(deps)); err != nil {
		return fmt.Errorf("apikey/contract: register apikeys.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "apikeys.detail", 1, apikeysDetailHandler(deps)); err != nil {
		return fmt.Errorf("apikey/contract: register apikeys.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apikeys.create", 1, apikeysCreateHandler(deps)); err != nil {
		return fmt.Errorf("apikey/contract: register apikeys.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apikeys.revoke", 1, apikeysRevokeHandler(deps)); err != nil {
		return fmt.Errorf("apikey/contract: register apikeys.revoke: %w", err)
	}
	return nil
}
