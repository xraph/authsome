// handlers.go: subscription plugin intent handlers. Moved from the
// auth contributor's handlers_subscriptions.go. The plugin's Service
// arrives via Deps (set up in plugins/subscription/contract.go) so
// handlers skip the engine.Plugin("subscription") indirection.
package contract

import (
	"context"
	"errors"
	"strings"
	"time"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"

	"github.com/xraph/forge/extensions/dashboard/contract"

	authcontract "github.com/xraph/authsome/extension/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

type PlanSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Description   string `json:"description,omitempty"`
	Currency      string `json:"currency,omitempty"`
	Status        string `json:"status"`
	TrialDays     int    `json:"trialDays,omitempty"`
	BaseAmount    int64  `json:"baseAmount"`
	BillingPeriod string `json:"billingPeriod,omitempty"`
}

type SubscriptionSummary struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenantId"`
	PlanID             string `json:"planId"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   string `json:"currentPeriodEnd,omitempty"`
	CancelAt           string `json:"cancelAt,omitempty"`
}

type PlanDetail struct {
	PlanSummary
	Features []PlanFeature      `json:"features,omitempty"`
	Tiers    []PriceTierSummary `json:"tiers,omitempty"`
}

type PlanFeature struct {
	ID        string `json:"id,omitempty"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Limit     int64  `json:"limit"`
	Period    string `json:"period"`
	SoftLimit bool   `json:"softLimit"`
	CatalogID string `json:"catalogId,omitempty"`
}

type PriceTierSummary struct {
	FeatureKey string `json:"featureKey"`
	Type       string `json:"type"`
	UpTo       int64  `json:"upTo"`
	UnitAmount int64  `json:"unitAmount"`
	FlatAmount int64  `json:"flatAmount"`
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

func plansDetailHandler(deps Deps) func(ctx context.Context, in GetPlanInput, p contract.Principal) (PlanDetail, error) {
	return func(ctx context.Context, in GetPlanInput, p contract.Principal) (PlanDetail, error) {
		if deps.Engine == nil || deps.Service == nil {
			return PlanDetail{}, unavailable()
		}
		pl, err := scopedPlan(ctx, deps, p, in.ID)
		if err != nil {
			return PlanDetail{}, mapErr(err)
		}
		d := PlanDetail{PlanSummary: projectPlan(pl)}
		for _, f := range pl.Features {
			d.Features = append(d.Features, PlanFeature{
				ID: f.ID.String(), Key: f.Key, Name: f.Name,
				Type:      string(f.Type),
				Limit:     f.Limit,
				Period:    string(f.Period),
				SoftLimit: f.SoftLimit,
				CatalogID: f.CatalogID.String(),
			})
		}
		if pl.Pricing != nil {
			for _, tier := range pl.Pricing.Tiers {
				d.Tiers = append(d.Tiers, PriceTierSummary{FeatureKey: tier.FeatureKey, Type: string(tier.Type), UpTo: tier.UpTo, UnitAmount: tier.UnitAmount.Amount, FlatAmount: tier.FlatAmount.Amount})
			}
		}
		return d, nil
	}
}

func plansArchiveHandler(deps Deps) func(ctx context.Context, in ArchivePlanInput, p contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in ArchivePlanInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return ackResponse{}, unavailable()
		}
		pl, err := scopedPlan(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.ArchivePlan(ctx, pl.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: pl.ID.String()}, nil
	}
}

func plansActivateHandler(deps Deps) func(ctx context.Context, in ActivatePlanInput, p contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in ActivatePlanInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Service == nil {
			return ackResponse{}, unavailable()
		}
		pl, err := scopedPlan(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.ActivatePlan(ctx, pl.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: pl.ID.String()}, nil
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
			if s.AppID == authcontract.AppIDFromPrincipal(p, deps.Engine).String() {
				out.Subscriptions = append(out.Subscriptions, projectSubscription(s))
			}
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
	out := PlanSummary{
		ID: p.ID.String(), Name: p.Name, Slug: p.Slug,
		Description: p.Description, Currency: p.Currency,
		Status: string(p.Status), TrialDays: p.TrialDays,
	}
	if p.Pricing != nil {
		out.BaseAmount = p.Pricing.BaseAmount.Amount
		out.BillingPeriod = string(p.Pricing.BillingPeriod)
	}
	return out
}

func projectSubscription(s *subscription.Subscription) SubscriptionSummary {
	if s == nil {
		return SubscriptionSummary{}
	}
	out := SubscriptionSummary{
		ID: s.ID.String(), TenantID: s.TenantID, PlanID: s.PlanID.String(),
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.UTC().Format(time.RFC3339),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.UTC().Format(time.RFC3339),
	}
	if s.CancelAt != nil {
		out.CancelAt = s.CancelAt.UTC().Format(time.RFC3339)
	}
	return out
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
	var ce *contract.Error
	if errors.As(err, &ce) {
		return ce
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
