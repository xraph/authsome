// contract.go: Wires the sso plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package sso

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	ssocontract "github.com/xraph/authsome/plugins/sso/contract"

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
		return fmt.Errorf("sso: contract registration requires *authsome.Engine, got %T", engine)
	}
	return ssocontract.Register(d, reg, wreg, ssocontract.Deps{Engine: eng})
}
