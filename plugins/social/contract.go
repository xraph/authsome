// contract.go: Wires the social plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package social

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	socialcontract "github.com/xraph/authsome/plugins/social/contract"

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
		return fmt.Errorf("social: contract registration requires *authsome.Engine, got %T", engine)
	}
	return socialcontract.Register(d, reg, wreg, socialcontract.Deps{Engine: eng})
}
