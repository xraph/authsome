// contract.go: Wires the riskengine plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package riskengine

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	riskcontract "github.com/xraph/authsome/plugins/riskengine/contract"

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
		return fmt.Errorf("riskengine: contract registration requires *authsome.Engine, got %T", engine)
	}
	return riskcontract.Register(d, reg, wreg, riskcontract.Deps{Engine: eng})
}
