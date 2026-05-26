// contract.go: Wires the impossibletravel plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package impossibletravel

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	imptravelcontract "github.com/xraph/authsome/plugins/impossibletravel/contract"

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
		return fmt.Errorf("impossibletravel: contract registration requires *authsome.Engine, got %T", engine)
	}
	return imptravelcontract.Register(d, reg, wreg, imptravelcontract.Deps{Engine: eng})
}
