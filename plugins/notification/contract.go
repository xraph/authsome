// contract.go: Wires the notification plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package notification

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	notifcontract "github.com/xraph/authsome/plugins/notification/contract"

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
		return fmt.Errorf("notification: contract registration requires *authsome.Engine, got %T", engine)
	}
	return notifcontract.Register(d, reg, wreg, notifcontract.Deps{Engine: eng})
}
