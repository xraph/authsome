// contract.go: Wires the deviceverify plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package deviceverify

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	devverifycontract "github.com/xraph/authsome/plugins/deviceverify/contract"

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
		return fmt.Errorf("deviceverify: contract registration requires *authsome.Engine, got %T", engine)
	}
	return devverifycontract.Register(d, reg, wreg, devverifycontract.Deps{Engine: eng})
}
