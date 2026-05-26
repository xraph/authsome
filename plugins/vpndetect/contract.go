// contract.go: Wires the vpndetect plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package vpndetect

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	vpndetectcontract "github.com/xraph/authsome/plugins/vpndetect/contract"

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
		return fmt.Errorf("vpndetect: contract registration requires *authsome.Engine, got %T", engine)
	}
	return vpndetectcontract.Register(d, reg, wreg, vpndetectcontract.Deps{Engine: eng})
}
