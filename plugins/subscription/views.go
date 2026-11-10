package subscription

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/id"
)

// SubscriptionView is the tenant-facing shape of a subscription: everything a
// plan card needs in one payload, with the plan embedded and every time in
// RFC 3339 UTC.
type SubscriptionView struct {
	ID                 string       `json:"id"`
	TenantID           string       `json:"tenant_id"`
	Status             string       `json:"status"`
	Plan               PlanResponse `json:"plan"`
	CurrentPeriodStart string       `json:"current_period_start"`
	CurrentPeriodEnd   string       `json:"current_period_end"`
	TrialStart         *string      `json:"trial_start"`
	TrialEnd           *string      `json:"trial_end"`
	// CancelAt is when a cancellation scheduled for the period end takes
	// effect; the subscription stays usable until then.
	CancelAt   *string    `json:"cancel_at"`
	CanceledAt *string    `json:"canceled_at"`
	PausedAt   *string    `json:"paused_at"`
	Seats      *SeatsView `json:"seats,omitempty"`
	AppID      string     `json:"app_id"`
}

// SeatsView reports seat use for a plan with a seat feature. Used counts the
// organization's members; Limit is the bought quantity, or the plan's limit
// when none was bought, and nil when unlimited.
type SeatsView struct {
	FeatureKey string `json:"feature_key"`
	Used       int64  `json:"used"`
	Limit      *int64 `json:"limit"`
}

// toSubscriptionView builds the tenant view of s, loading its plan and, for a
// plan with a seat feature, counting the organization's members.
func (p *Plugin) toSubscriptionView(ctx context.Context, s *lsub.Subscription) (*SubscriptionView, error) {
	pl, err := p.service.GetPlan(ctx, s.PlanID)
	if err != nil {
		return nil, fmt.Errorf("load plan %s: %w", s.PlanID, err)
	}

	view := &SubscriptionView{
		ID:                 s.ID.String(),
		TenantID:           s.TenantID,
		Status:             string(s.Status),
		Plan:               toPlanResponse(pl),
		CurrentPeriodStart: rfc3339(s.CurrentPeriodStart),
		CurrentPeriodEnd:   rfc3339(s.CurrentPeriodEnd),
		TrialStart:         rfc3339Ptr(s.TrialStart),
		TrialEnd:           rfc3339Ptr(s.TrialEnd),
		CancelAt:           rfc3339Ptr(s.CancelAt),
		CanceledAt:         rfc3339Ptr(s.CanceledAt),
		PausedAt:           rfc3339Ptr(s.PausedAt),
		AppID:              s.AppID,
	}

	seats, err := p.seatsFor(ctx, s, pl)
	if err != nil {
		return nil, err
	}
	view.Seats = seats
	return view, nil
}

// seatsFor answers nil when the plan has no seat feature.
func (p *Plugin) seatsFor(ctx context.Context, s *lsub.Subscription, pl *plan.Plan) (*SeatsView, error) {
	feat := seatFeature(pl)
	if feat == nil {
		return nil, nil
	}

	seats := &SeatsView{FeatureKey: feat.Key}
	if n, ok := s.Quantity[feat.Key]; ok {
		seats.Limit = &n
	} else if feat.Limit >= 0 {
		limit := feat.Limit
		seats.Limit = &limit
	}

	if p.tenantMode(ctx) != "organization" || p.authStore == nil {
		return seats, nil
	}
	orgID, err := id.ParseOrgID(s.TenantID)
	if err != nil {
		return seats, nil //nolint:nilerr // a tenant that isn't an org ID has no members to count
	}
	members, err := p.authStore.ListMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count members of %s: %w", orgID, err)
	}
	seats.Used = int64(len(members))
	return seats, nil
}

// seatFeature returns the plan's seat feature, or nil when it has none.
func seatFeature(pl *plan.Plan) *plan.Feature {
	for i := range pl.Features {
		if pl.Features[i].Type == plan.FeatureSeat {
			return &pl.Features[i]
		}
	}
	return nil
}

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := rfc3339(*t)
	return &s
}

// sortPlansByPrice orders plans the way a pricing table reads them: plans
// without a price first, then cheapest first. Ties keep the store's order.
func sortPlansByPrice(plans []PlanResponse) {
	slices.SortStableFunc(plans, func(a, b PlanResponse) int {
		return cmp.Compare(priceKey(a), priceKey(b))
	})
}

func priceKey(p PlanResponse) int64 {
	if p.AmountMinor == nil {
		return -1
	}
	return *p.AmountMinor
}
