// contract.go: Wires the phone plugin into the forge-dashboard
// contract surface via plugin.ContractContributor. The manifest +
// handlers live in plugins/phone/contract/.
package phone

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	phonecontract "github.com/xraph/authsome/plugins/phone/contract"

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
		return fmt.Errorf("phone: contract registration requires *authsome.Engine, got %T", engine)
	}
	return phonecontract.Register(d, reg, wreg, phonecontract.Deps{Engine: eng})
}
