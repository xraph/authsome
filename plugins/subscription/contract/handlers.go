// handlers.go: subscription plugin intent handlers. Moved from the
// auth contributor's handlers_subscriptions.go. The plugin's Service
// arrives via Deps (set up in plugins/subscription/contract.go) so
// handlers skip the engine.Plugin("subscription") indirection.
package contract

import (
	"context"
	"strings"
	"time"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"

	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

type PlanSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Status      string `json:"status"`
	TrialDays   int    `json:"trialDays,omitempty"`
}

type SubscriptionSummary struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenantId"`
	PlanID             string `json:"planId"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   string `json:"currentPeriodEnd,omitempty"`
}

type PlanDetail struct {
	PlanSummary
	Features []PlanFeature `json:"features,omitempty"`
}

type PlanFeature struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Limit  int64  `json:"limit"`
	Period string `json:"period"`
}

type PlansListResponse struct {
	Plans []PlanSummary `json:"plans"`
}

type SubscriptionsListResponse struct {
	Subscriptions []SubscriptionSummary `json:"subscriptions"`
}

type SubscriptionsListInput struct {
	TenantID string `json:"tenantId"`
}

type GetPlanInput struct {
	ID string `json:"id"`
}

type ArchivePlanInput struct {
	ID string `json:"id"`
}

type ActivatePlanInput struct {
	ID string `json:"id"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func plansListHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (PlansListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (PlansListResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return PlansListResponse{}, unavailable()
		}
		list, err := deps.Service.ListPlans(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine).String())
		if err != nil {
			return PlansListResponse{}, mapErr(err)
		}
		out := PlansListResponse{Plans: make([]PlanSummary, 0, len(list))}
		for _, pl := range list {
			out.Plans = append(out.Plans, projectPlan(pl))
		}
		return out, nil
	}
}

func plansDetailHandler(deps Deps) func(ctx context.Context, in GetPlanInput, _ contract.Principal) (PlanDetail, error) {
	return func(ctx context.Context, in GetPlanInput, _ contract.Principal) (PlanDetail, error) {
		if deps.Engine == nil || deps.Service == nil {
			return PlanDetail{}, unavailable()
		}
		pid, err := parsePlanID(in.ID)
		if err != nil {
			return PlanDetail{}, err
		}
		pl, err := deps.Service.GetPlan(ctx, pid)
		if err != nil {
			return PlanDetail{}, mapErr(err)
		}
		d := PlanDetail{PlanSummary: projectPlan(pl)}
		for _, f := range pl.Features {
			d.Features = append(d.Features, PlanFeature{
				Key: f.Key, Name: f.Name,
				Type:   string(f.Type),
				Limit:  f.Limit,
				Period: string(f.Period),
			})
		}
		return d, nil
	}
}

func plansArchiveHandler(deps Deps) func(ctx context.Context, in ArchivePlanInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in ArchivePlanInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return ackResponse{}, unavailable()
		}
		pid, err := parsePlanID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.ArchivePlan(ctx, pid); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: pid.String()}, nil
	}
}

func plansActivateHandler(deps Deps) func(ctx context.Context, in ActivatePlanInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in ActivatePlanInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return ackResponse{}, unavailable()
		}
		pid, err := parsePlanID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.ActivatePlan(ctx, pid); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: pid.String()}, nil
	}
}

func subscriptionsListHandler(deps Deps) func(ctx context.Context, in SubscriptionsListInput, p contract.Principal) (SubscriptionsListResponse, error) {
	return func(ctx context.Context, in SubscriptionsListInput, p contract.Principal) (SubscriptionsListResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return SubscriptionsListResponse{}, unavailable()
		}
		tenant := strings.TrimSpace(in.TenantID)
		if tenant == "" {
			return SubscriptionsListResponse{Subscriptions: []SubscriptionSummary{}}, nil
		}
		list, err := deps.Service.ListSubscriptions(ctx, tenant, authcontract.AppIDFromPrincipal(p, deps.Engine).String(), subscription.ListOpts{})
		if err != nil {
			return SubscriptionsListResponse{}, mapErr(err)
		}
		out := SubscriptionsListResponse{Subscriptions: make([]SubscriptionSummary, 0, len(list))}
		for _, s := range list {
			out.Subscriptions = append(out.Subscriptions, projectSubscription(s))
		}
		return out, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func projectPlan(p *plan.Plan) PlanSummary {
	if p == nil {
		return PlanSummary{}
	}
	return PlanSummary{
		ID: p.ID.String(), Name: p.Name, Slug: p.Slug,
		Description: p.Description, Currency: p.Currency,
		Status: string(p.Status), TrialDays: p.TrialDays,
	}
}

func projectSubscription(s *subscription.Subscription) SubscriptionSummary {
	if s == nil {
		return SubscriptionSummary{}
	}
	return SubscriptionSummary{
		ID: s.ID.String(), TenantID: s.TenantID, PlanID: s.PlanID.String(),
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.UTC().Format(time.RFC3339),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.UTC().Format(time.RFC3339),
	}
}

func parsePlanID(s string) (ledgerid.PlanID, error) {
	if strings.TrimSpace(s) == "" {
		return ledgerid.PlanID{}, badReq("id is required")
	}
	pid, err := ledgerid.ParsePlanID(s)
	if err != nil {
		return ledgerid.PlanID{}, badReq("invalid plan id: " + err.Error())
	}
	return pid, nil
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "subscription plugin not enabled"}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*contract.Error); ok {
		return ce
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
