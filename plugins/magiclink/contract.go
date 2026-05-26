// contract.go: Wires the magiclink plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package magiclink

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	mlcontract "github.com/xraph/authsome/plugins/magiclink/contract"

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
		return fmt.Errorf("magiclink: contract registration requires *authsome.Engine, got %T", engine)
	}
	return mlcontract.Register(d, reg, wreg, mlcontract.Deps{Engine: eng})
}
