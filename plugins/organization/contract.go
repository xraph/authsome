// contract.go: Wires the organization plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package organization

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	orgcontract "github.com/xraph/authsome/plugins/organization/contract"

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
		return fmt.Errorf("organization: contract registration requires *authsome.Engine, got %T", engine)
	}
	return orgcontract.Register(d, reg, wreg, orgcontract.Deps{Engine: eng, Plugin: p})
}
