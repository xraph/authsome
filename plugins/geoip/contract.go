// contract.go: Wires the geoip plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package geoip

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	geoipcontract "github.com/xraph/authsome/plugins/geoip/contract"

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
		return fmt.Errorf("geoip: contract registration requires *authsome.Engine, got %T", engine)
	}
	return geoipcontract.Register(d, reg, wreg, geoipcontract.Deps{Engine: eng})
}
