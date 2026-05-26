// contract.go: Wires the passkey plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package passkey

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	pkcontract "github.com/xraph/authsome/plugins/passkey/contract"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

var _ plugin.ContractContributor = (*Plugin)(nil)

func (p *Plugin) RegisterContract(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	engine plugin.Engine,
) error {
	eng, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("passkey: contract registration requires *authsome.Engine, got %T", engine)
	}
	return pkcontract.Register(d, reg, wreg, pkcontract.Deps{Engine: eng})
}
