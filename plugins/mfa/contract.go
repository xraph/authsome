// contract.go: Wires the mfa plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package mfa

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	mfacontract "github.com/xraph/authsome/plugins/mfa/contract"

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
		return fmt.Errorf("mfa: contract registration requires *authsome.Engine, got %T", engine)
	}
	return mfacontract.Register(d, reg, wreg, mfacontract.Deps{Engine: eng})
}
