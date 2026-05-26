// contract.go: Wires the password plugin into the forge-dashboard
// contract surface. The actual manifest + handlers live in the
// subpackage at plugins/password/contract/; this file exposes the
// integration via plugin.ContractContributor so authsome's
// RegisterContractContributor loop picks the plugin up automatically.
package password

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	pwcontract "github.com/xraph/authsome/plugins/password/contract"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

// Compile-time assertion that *Plugin satisfies the optional contract
// contributor interface. If the interface signature drifts, the build
// breaks here rather than at runtime in extension.go's iteration loop.
var _ plugin.ContractContributor = (*Plugin)(nil)

// RegisterContract implements plugin.ContractContributor. It type-asserts
// the engine handle to *authsome.Engine (the only concrete type that
// implements plugin.Engine in practice) and delegates to the subpackage's
// Register function. Failed assertion returns a descriptive error rather
// than panicking — keeps non-authsome embedders cleanly opted out.
func (p *Plugin) RegisterContract(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	engine plugin.Engine,
) error {
	eng, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("password: contract registration requires *authsome.Engine, got %T", engine)
	}
	return pwcontract.Register(d, reg, wreg, pwcontract.Deps{Engine: eng})
}
