// contract.go: Wires the geofence plugin into the forge-dashboard contract
// surface via plugin.ContractContributor.
package geofence

import (
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	geofencecontract "github.com/xraph/authsome/plugins/geofence/contract"

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
		return fmt.Errorf("geofence: contract registration requires *authsome.Engine, got %T", engine)
	}
	return geofencecontract.Register(d, reg, wreg, geofencecontract.Deps{Engine: eng})
}
