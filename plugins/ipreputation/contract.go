// contract.go: Wires the ipreputation plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package ipreputation

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	ipreputcontract "github.com/xraph/authsome/plugins/ipreputation/contract"

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
		return fmt.Errorf("ipreputation: contract registration requires *authsome.Engine, got %T", engine)
	}
	return ipreputcontract.Register(d, reg, wreg, ipreputcontract.Deps{Engine: eng})
}
