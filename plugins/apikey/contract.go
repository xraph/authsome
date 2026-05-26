// contract.go: Wires the apikey plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package apikey

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	akcontract "github.com/xraph/authsome/plugins/apikey/contract"

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
		return fmt.Errorf("apikey: contract registration requires *authsome.Engine, got %T", engine)
	}
	return akcontract.Register(d, reg, wreg, akcontract.Deps{Engine: eng})
}
