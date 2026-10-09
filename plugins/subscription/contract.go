// contract.go: Wires the subscription plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package subscription

import (
	"context"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
	subcontract "github.com/xraph/authsome/plugins/subscription/contract"

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
		return fmt.Errorf("subscription: contract registration requires *authsome.Engine, got %T", engine)
	}
	svc := p.Service()
	if svc == nil {
		return fmt.Errorf("subscription: Service not initialised")
	}
	return subcontract.Register(d, reg, wreg, subcontract.Deps{Engine: eng, Service: svc, Usage: func(ctx context.Context, tenantID, appID string) ([]subcontract.UsageSummary, error) {
		rows, err := svc.GetUsageSummary(ctx, tenantID, appID)
		if err != nil {
			return nil, err
		}
		out := make([]subcontract.UsageSummary, 0, len(rows))
		for _, row := range rows {
			out = append(out, subcontract.UsageSummary{FeatureKey: row.FeatureKey, FeatureName: row.FeatureName, FeatureType: row.FeatureType, Used: row.Used, Limit: row.Limit, Remaining: row.Remaining, Period: row.Period})
		}
		return out, nil
	}})
}
