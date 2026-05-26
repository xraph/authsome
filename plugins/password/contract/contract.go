// Package contract registers the password plugin's forge-dashboard
// contract surface: three intents (password.settings, password.policy,
// password.settingsUpdate) backed by the settings.Manager, plus a
// standalone /auth/password page that extends the auth contributor's
// /settings page with a password-policy panel.
//
// Wiring lives in plugins/password/dashboard.go's RegisterContract
// method (added when the plugin opts into the contract surface via
// the authsome plugin.ContractContributor interface). The bootstrap
// loop in authsome/extension/extension.go's
// RegisterContractContributor invokes RegisterContract on every plugin
// that implements the interface after the auth contributor itself
// registers.
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

// Deps is the typed dependency bundle the password contract handlers
// need at registration time. Engine is required — it gives access to
// the settings.Manager. AppID is the app whose policy is being read /
// edited; defaults to the engine's platform app when zero.
type Deps struct {
	Engine *authsome.Engine
}

// Register loads the embedded manifest, registers it against the
// dashboard's contract registry (applying any `extends` blocks against
// previously-registered contributors — `auth` must register first),
// and binds the three intent handlers against the dispatcher.
func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("password/contract: Engine is required")
	}

	m, err := loader.Load(bytes.NewReader(manifestYAML), "password/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("password/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("password/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("password/contract: register manifest: %w", err)
	}

	const c = "password"
	// password.policy is the only intent this plugin still owns. Reads
	// and writes of the underlying password.* settings flow through the
	// auth contributor's settings.namespace / settings.update intents —
	// the settings.panel renderer wires them up automatically because
	// password.DeclareSettings registers the keys under the "password"
	// namespace at plugin init time.
	if err := dispatcher.RegisterQuery(d, c, "password.policy", 1, policyHandler(deps)); err != nil {
		return fmt.Errorf("password/contract: register password.policy: %w", err)
	}

	return nil
}
