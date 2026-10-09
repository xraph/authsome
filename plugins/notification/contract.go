// contract.go: Wires the notification plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package notification

import (
	"fmt"
	"sort"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
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
	return notifcontract.Register(d, reg, wreg, notifcontract.Deps{
		Engine:  eng,
		Manager: func() bridge.HeraldTemplateManager { return p.templates },
		Sender:  func() bridge.Herald { return p.herald },
		Mappings: func() []notifcontract.MappingSummary {
			out := make([]notifcontract.MappingSummary, 0, len(p.mappings))
			for action, mapping := range p.mappings {
				if mapping != nil {
					out = append(out, notifcontract.MappingSummary{Action: action, Template: mapping.Template, Channels: append([]string(nil), mapping.Channels...), Enabled: mapping.Enabled})
				}
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Action < out[j].Action })
			return out
		},
	})
}
