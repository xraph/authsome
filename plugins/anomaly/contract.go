// contract.go: Wires the anomaly plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package anomaly

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	anomalycontract "github.com/xraph/authsome/plugins/anomaly/contract"

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
		return fmt.Errorf("anomaly: contract registration requires *authsome.Engine, got %T", engine)
	}
	return anomalycontract.Register(d, reg, wreg, anomalycontract.Deps{Engine: eng})
}
