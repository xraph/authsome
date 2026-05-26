// Package contract wires the subscription plugin's intent surface
// into the forge-dashboard contract registry. The `/plans` and
// `/plans/:id` pages stay declared on the auth contributor; only the
// intent handlers + their declarations move here.
package contract

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"

	authsome "github.com/xraph/authsome"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// SubscriptionService is the surface this package needs from the
// subscription plugin. Declared as an interface here so the contract
// subpackage doesn't import plugins/subscription directly (cycle
// avoidance). The parent contract.go satisfies it by passing the
// plugin's Service().
type SubscriptionService interface {
	ListPlans(ctx context.Context, appID string) ([]*plan.Plan, error)
	GetPlan(ctx context.Context, planID ledgerid.PlanID) (*plan.Plan, error)
	ArchivePlan(ctx context.Context, planID ledgerid.PlanID) error
	ActivatePlan(ctx context.Context, planID ledgerid.PlanID) error
	ListSubscriptions(ctx context.Context, tenantID, appID string, opts subscription.ListOpts) ([]*subscription.Subscription, error)
}

type Deps struct {
	Engine  *authsome.Engine
	Service SubscriptionService
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("subscription/contract: Engine is required")
	}
	if deps.Service == nil {
		return fmt.Errorf("subscription/contract: Service is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("subscription/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("subscription/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("subscription/contract: register manifest: %w", err)
	}

	const c = "subscription"
	if err := dispatcher.RegisterQuery(d, c, "plans.list", 1, plansListHandler(deps)); err != nil {
		return fmt.Errorf("subscription/contract: register plans.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "plans.detail", 1, plansDetailHandler(deps)); err != nil {
		return fmt.Errorf("subscription/contract: register plans.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "plans.archive", 1, plansArchiveHandler(deps)); err != nil {
		return fmt.Errorf("subscription/contract: register plans.archive: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "plans.activate", 1, plansActivateHandler(deps)); err != nil {
		return fmt.Errorf("subscription/contract: register plans.activate: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "subscriptions.list", 1, subscriptionsListHandler(deps)); err != nil {
		return fmt.Errorf("subscription/contract: register subscriptions.list: %w", err)
	}
	return nil
}
