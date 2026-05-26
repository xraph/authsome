// contract.go: Wires the email plugin into the forge-dashboard
// contract surface. The manifest + handlers live in the subpackage at
// plugins/email/contract/; this file exposes the integration via
// plugin.ContractContributor so authsome's RegisterContractContributor
// loop picks the plugin up automatically.
package email

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	emailcontract "github.com/xraph/authsome/plugins/email/contract"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

// Compile-time assertion that *Plugin satisfies the optional contract
// contributor interface. If the interface signature drifts, the build
// breaks here rather than at runtime in extension.go's iteration loop.
var _ plugin.ContractContributor = (*Plugin)(nil)

// RegisterContract implements plugin.ContractContributor. Type-asserts
// the engine handle to *authsome.Engine and delegates to the subpackage.
func (p *Plugin) RegisterContract(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	engine plugin.Engine,
) error {
	eng, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("email: contract registration requires *authsome.Engine, got %T", engine)
	}
	return emailcontract.Register(d, reg, wreg, emailcontract.Deps{Engine: eng})
}
