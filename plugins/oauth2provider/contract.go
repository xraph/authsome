// contract.go: Wires the oauth2provider plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package oauth2provider

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	oauth2contract "github.com/xraph/authsome/plugins/oauth2provider/contract"

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
		return fmt.Errorf("oauth2provider: contract registration requires *authsome.Engine, got %T", engine)
	}
	return oauth2contract.Register(d, reg, wreg, oauth2contract.Deps{Engine: eng})
}
