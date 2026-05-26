// contract.go: Wires the subscription plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package subscription

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	subcontract "github.com/xraph/authsome/plugins/subscription/contract"

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
		return fmt.Errorf("subscription: contract registration requires *authsome.Engine, got %T", engine)
	}
	svc := p.Service()
	if svc == nil {
		return fmt.Errorf("subscription: Service not initialised")
	}
	return subcontract.Register(d, reg, wreg, subcontract.Deps{Engine: eng, Service: svc})
}
