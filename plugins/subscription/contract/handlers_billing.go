package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/ledger/coupon"
	"github.com/xraph/ledger/feature"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"
	"github.com/xraph/ledger/types"

	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

type planWriteInput struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Slug          string             `json:"slug"`
	Description   string             `json:"description"`
	Currency      string             `json:"currency"`
	TrialDays     int                `json:"trialDays"`
	BaseAmount    int64              `json:"baseAmount"`
	BillingPeriod string             `json:"billingPeriod"`
	Features      []PlanFeature      `json:"features"`
	Tiers         []PriceTierSummary `json:"tiers"`
}
type subscriptionWriteInput struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	PlanID      string `json:"planId"`
	Immediately bool   `json:"immediately"`
}
type invoiceWriteInput struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscriptionId"`
	PaymentRef     string `json:"paymentRef"`
	Reason         string `json:"reason"`
}
type couponWriteInput struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Amount         int64  `json:"amount"`
	Percentage     int    `json:"percentage"`
	Currency       string `json:"currency"`
	MaxRedemptions int    `json:"maxRedemptions"`
	ValidFrom      string `json:"validFrom"`
	ValidUntil     string `json:"validUntil"`
}
type invoicesResponse struct {
	Invoices []invoiceSummary `json:"invoices"`
}
type detailInput struct {
	ID string `json:"id"`
}
type UsageSummary struct {
	FeatureKey  string `json:"featureKey"`
	FeatureName string `json:"featureName"`
	FeatureType string `json:"featureType"`
	Used        int64  `json:"used"`
	Limit       int64  `json:"limit"`
	Remaining   int64  `json:"remaining"`
	Period      string `json:"period"`
}
type subscriptionDetail struct {
	SubscriptionSummary
	PlanName string           `json:"planName"`
	Usage    []UsageSummary   `json:"usage"`
	Invoices []invoiceSummary `json:"invoices"`
}
type invoiceLineSummary struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	FeatureKey  string `json:"featureKey,omitempty"`
	Quantity    int64  `json:"quantity"`
	UnitAmount  int64  `json:"unitAmount"`
	Amount      int64  `json:"amount"`
}
type invoiceDetail struct {
	invoiceSummary
	Subtotal       int64                `json:"subtotal"`
	TaxAmount      int64                `json:"taxAmount"`
	DiscountAmount int64                `json:"discountAmount"`
	PeriodStart    string               `json:"periodStart"`
	PeriodEnd      string               `json:"periodEnd"`
	LineItems      []invoiceLineSummary `json:"lineItems"`
}
type invoiceSummary struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
	Status         string `json:"status"`
	Currency       string `json:"currency"`
	Total          int64  `json:"total"`
	PaymentRef     string `json:"paymentRef,omitempty"`
}

func projectInvoice(item *invoice.Invoice) invoiceSummary {
	return invoiceSummary{ID: item.ID.String(), TenantID: item.TenantID, SubscriptionID: item.SubscriptionID.String(), Status: string(item.Status), Currency: item.Currency, Total: item.Total.Amount, PaymentRef: item.PaymentRef}
}

type couponsResponse struct {
	Coupons []couponSummary `json:"coupons"`
}
type couponSummary struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Amount         int64  `json:"amount"`
	Percentage     int    `json:"percentage"`
	Currency       string `json:"currency"`
	MaxRedemptions int    `json:"maxRedemptions"`
	TimesRedeemed  int    `json:"timesRedeemed"`
	ValidFrom      string `json:"validFrom,omitempty"`
	ValidUntil     string `json:"validUntil,omitempty"`
}

func scopedApp(deps Deps, p contract.Principal) string {
	return authcontract.AppIDFromPrincipal(p, deps.Engine).String()
}
func notFound(resource string) error {
	return &contract.Error{Code: contract.CodeNotFound, Message: resource + " not found"}
}
func scopedPlan(ctx context.Context, deps Deps, p contract.Principal, raw string) (*plan.Plan, error) {
	id, err := parsePlanID(raw)
	if err != nil {
		return nil, err
	}
	item, err := deps.Service.GetPlan(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if item == nil || item.AppID != scopedApp(deps, p) {
		return nil, notFound("plan")
	}
	return item, nil
}
func scopedSubscription(ctx context.Context, deps Deps, p contract.Principal, raw string) (*subscription.Subscription, error) {
	id, err := ledgerid.ParseSubscriptionID(strings.TrimSpace(raw))
	if err != nil {
		return nil, badReq("invalid subscription id")
	}
	item, err := deps.Service.GetSubscription(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if item == nil || item.AppID != scopedApp(deps, p) {
		return nil, notFound("subscription")
	}
	return item, nil
}
func scopedInvoice(ctx context.Context, deps Deps, p contract.Principal, raw string) (*invoice.Invoice, error) {
	id, err := ledgerid.ParseInvoiceID(strings.TrimSpace(raw))
	if err != nil {
		return nil, badReq("invalid invoice id")
	}
	item, err := deps.Service.GetInvoice(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if item == nil || item.AppID != scopedApp(deps, p) {
		return nil, notFound("invoice")
	}
	return item, nil
}
func pricing(amount int64, currency, period string, tiers []PriceTierSummary) (*plan.Pricing, error) {
	if amount < 0 {
		return nil, badReq("base amount cannot be negative")
	}
	if amount == 0 && len(tiers) == 0 {
		return nil, nil
	}
	if period == "" {
		period = string(plan.PeriodMonthly)
	}
	if period != string(plan.PeriodMonthly) && period != string(plan.PeriodYearly) {
		return nil, badReq("invalid billing period")
	}
	out := &plan.Pricing{BaseAmount: types.Money{Amount: amount, Currency: currency}, BillingPeriod: plan.Period(period)}
	for _, item := range tiers {
		if strings.TrimSpace(item.FeatureKey) == "" || item.UpTo < 0 || item.UnitAmount < 0 || item.FlatAmount < 0 {
			return nil, badReq("invalid price tier")
		}
		kind := plan.TierType(item.Type)
		if kind != plan.TierGraduated && kind != plan.TierVolume && kind != plan.TierFlat {
			return nil, badReq("invalid price tier type")
		}
		out.Tiers = append(out.Tiers, plan.PriceTier{FeatureKey: strings.TrimSpace(item.FeatureKey), Type: kind, UpTo: item.UpTo, UnitAmount: types.Money{Amount: item.UnitAmount, Currency: currency}, FlatAmount: types.Money{Amount: item.FlatAmount, Currency: currency}, Priority: len(out.Tiers)})
	}
	return out, nil
}
func planFeatures(input []PlanFeature) ([]plan.Feature, error) {
	out := make([]plan.Feature, 0, len(input))
	seen := map[string]bool{}
	for _, item := range input {
		key := strings.TrimSpace(item.Key)
		if key == "" || seen[key] {
			return nil, badReq("feature keys must be unique and nonempty")
		}
		seen[key] = true
		kind := plan.FeatureType(item.Type)
		if kind != plan.FeatureBoolean && kind != plan.FeatureMetered && kind != plan.FeatureSeat {
			return nil, badReq("invalid feature type")
		}
		f := plan.Feature{ID: ledgerid.NewFeatureID(), Key: key, Name: strings.TrimSpace(item.Name), Type: kind, Limit: item.Limit, Period: plan.Period(item.Period), SoftLimit: item.SoftLimit}
		if item.CatalogID != "" {
			cid, err := ledgerid.ParseFeatureID(item.CatalogID)
			if err != nil {
				return nil, badReq("invalid catalog feature id")
			}
			f.CatalogID = cid
		}
		out = append(out, f)
	}
	return out, nil
}
func validateTierFeatures(price *plan.Pricing, features []plan.Feature) error {
	if price == nil {
		return nil
	}
	keys := make(map[string]bool, len(features))
	for _, item := range features {
		keys[item.Key] = true
	}
	for _, tier := range price.Tiers {
		if !keys[tier.FeatureKey] {
			return badReq("price tier feature is not on the plan")
		}
	}
	return nil
}
func plansCreateHandler(deps Deps) func(context.Context, planWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in planWriteInput, p contract.Principal) (ackResponse, error) {
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Slug) == "" {
			return ackResponse{}, badReq("name and slug are required")
		}
		if in.TrialDays < 0 {
			return ackResponse{}, badReq("trial days cannot be negative")
		}
		currency := strings.ToLower(strings.TrimSpace(in.Currency))
		if currency == "" {
			currency = "usd"
		}
		price, err := pricing(in.BaseAmount, currency, in.BillingPeriod, in.Tiers)
		if err != nil {
			return ackResponse{}, err
		}
		features, err := planFeatures(in.Features)
		if err != nil {
			return ackResponse{}, err
		}
		if err := validateTierFeatures(price, features); err != nil {
			return ackResponse{}, err
		}
		item := &plan.Plan{Name: strings.TrimSpace(in.Name), Slug: strings.TrimSpace(in.Slug), Description: in.Description, Currency: currency, Status: plan.StatusDraft, TrialDays: in.TrialDays, AppID: scopedApp(deps, p), Pricing: price, Features: features}
		if err := deps.Service.CreatePlan(ctx, item); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func plansUpdateHandler(deps Deps) func(context.Context, planWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in planWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedPlan(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if strings.TrimSpace(in.Name) == "" {
			return ackResponse{}, badReq("name is required")
		}
		if in.TrialDays < 0 {
			return ackResponse{}, badReq("trial days cannot be negative")
		}
		price, err := pricing(in.BaseAmount, item.Currency, in.BillingPeriod, in.Tiers)
		if err != nil {
			return ackResponse{}, err
		}
		features, err := planFeatures(in.Features)
		if err != nil {
			return ackResponse{}, err
		}
		if err := validateTierFeatures(price, features); err != nil {
			return ackResponse{}, err
		}
		oldIDs := map[string]ledgerid.FeatureID{}
		for _, f := range item.Features {
			oldIDs[f.Key] = f.ID
		}
		for i := range features {
			if oldID, ok := oldIDs[features[i].Key]; ok {
				features[i].ID = oldID
			}
		}
		item.Name = strings.TrimSpace(in.Name)
		item.Description = in.Description
		item.TrialDays = in.TrialDays
		item.Features = features
		if price != nil && item.Pricing != nil {
			price.ID = item.Pricing.ID
			price.PlanID = item.Pricing.PlanID
		}
		item.Pricing = price
		if err := deps.Service.UpdatePlan(ctx, item); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func subscriptionsAllHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (SubscriptionsListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (SubscriptionsListResponse, error) {
		list, err := deps.Service.ListSubscriptions(ctx, "", scopedApp(deps, p), subscription.ListOpts{})
		if err != nil {
			return SubscriptionsListResponse{}, mapErr(err)
		}
		out := SubscriptionsListResponse{Subscriptions: make([]SubscriptionSummary, 0, len(list))}
		for _, item := range list {
			if item.AppID == scopedApp(deps, p) {
				out.Subscriptions = append(out.Subscriptions, projectSubscription(item))
			}
		}
		return out, nil
	}
}
func subscriptionsDetailHandler(deps Deps) func(context.Context, detailInput, contract.Principal) (subscriptionDetail, error) {
	return func(ctx context.Context, in detailInput, p contract.Principal) (subscriptionDetail, error) {
		item, err := scopedSubscription(ctx, deps, p, in.ID)
		if err != nil {
			return subscriptionDetail{}, err
		}
		out := subscriptionDetail{SubscriptionSummary: projectSubscription(item), Usage: []UsageSummary{}, Invoices: []invoiceSummary{}}
		pl, err := scopedPlan(ctx, deps, p, item.PlanID.String())
		if err == nil {
			out.PlanName = pl.Name
		}
		if deps.Usage != nil {
			active, activeErr := deps.Service.GetActiveSubscription(ctx, item.TenantID, scopedApp(deps, p))
			if activeErr == nil && active != nil && active.ID == item.ID {
				if usage, err := deps.Usage(ctx, item.TenantID, scopedApp(deps, p)); err == nil {
					out.Usage = usage
				}
			}
		}
		invoices, err := deps.Service.ListInvoices(ctx, item.TenantID, scopedApp(deps, p))
		if err != nil {
			return subscriptionDetail{}, mapErr(err)
		}
		for _, inv := range invoices {
			if inv.AppID == scopedApp(deps, p) && inv.SubscriptionID == item.ID {
				out.Invoices = append(out.Invoices, projectInvoice(inv))
			}
		}
		return out, nil
	}
}
func subscriptionsCreateHandler(deps Deps) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in subscriptionWriteInput, p contract.Principal) (ackResponse, error) {
		if strings.TrimSpace(in.TenantID) == "" {
			return ackResponse{}, badReq("tenant id is required")
		}
		pl, err := scopedPlan(ctx, deps, p, in.PlanID)
		if err != nil {
			return ackResponse{}, err
		}
		if pl.Status != plan.StatusActive {
			return ackResponse{}, badReq("plan must be active")
		}
		item, err := deps.Service.Subscribe(ctx, strings.TrimSpace(in.TenantID), pl.ID, scopedApp(deps, p))
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func subscriptionsChangeHandler(deps Deps) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in subscriptionWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedSubscription(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		pl, err := scopedPlan(ctx, deps, p, in.PlanID)
		if err != nil {
			return ackResponse{}, err
		}
		if pl.Status != plan.StatusActive {
			return ackResponse{}, badReq("plan must be active")
		}
		if err := deps.Service.ChangePlan(ctx, item.ID, pl.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func subscriptionAction(deps Deps, action func(context.Context, ledgerid.SubscriptionID) error) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in subscriptionWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedSubscription(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := action(ctx, item.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func subscriptionsPauseHandler(deps Deps) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return subscriptionAction(deps, deps.Service.PauseSubscription)
}
func subscriptionsResumeHandler(deps Deps) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return subscriptionAction(deps, deps.Service.ResumeSubscription)
}
func subscriptionsCancelHandler(deps Deps) func(context.Context, subscriptionWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in subscriptionWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedSubscription(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.CancelSubscription(ctx, item.ID, in.Immediately); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func invoicesListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (invoicesResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (invoicesResponse, error) {
		list, err := deps.Service.ListAllInvoices(ctx, scopedApp(deps, p))
		if err != nil {
			return invoicesResponse{}, mapErr(err)
		}
		out := invoicesResponse{Invoices: make([]invoiceSummary, 0, len(list))}
		for _, item := range list {
			if item.AppID == scopedApp(deps, p) {
				out.Invoices = append(out.Invoices, invoiceSummary{ID: item.ID.String(), TenantID: item.TenantID, SubscriptionID: item.SubscriptionID.String(), Status: string(item.Status), Currency: item.Currency, Total: item.Total.Amount, PaymentRef: item.PaymentRef})
			}
		}
		return out, nil
	}
}
func invoicesDetailHandler(deps Deps) func(context.Context, detailInput, contract.Principal) (invoiceDetail, error) {
	return func(ctx context.Context, in detailInput, p contract.Principal) (invoiceDetail, error) {
		item, err := scopedInvoice(ctx, deps, p, in.ID)
		if err != nil {
			return invoiceDetail{}, err
		}
		out := invoiceDetail{invoiceSummary: projectInvoice(item), Subtotal: item.Subtotal.Amount, TaxAmount: item.TaxAmount.Amount, DiscountAmount: item.DiscountAmount.Amount, PeriodStart: item.PeriodStart.UTC().Format("2006-01-02"), PeriodEnd: item.PeriodEnd.UTC().Format("2006-01-02"), LineItems: make([]invoiceLineSummary, 0, len(item.LineItems))}
		for _, line := range item.LineItems {
			out.LineItems = append(out.LineItems, invoiceLineSummary{Description: line.Description, Type: string(line.Type), FeatureKey: line.FeatureKey, Quantity: line.Quantity, UnitAmount: line.UnitAmount.Amount, Amount: line.Amount.Amount})
		}
		return out, nil
	}
}
func invoicesGenerateHandler(deps Deps) func(context.Context, invoiceWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in invoiceWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedSubscription(ctx, deps, p, in.SubscriptionID)
		if err != nil {
			return ackResponse{}, err
		}
		generated, err := deps.Service.GenerateInvoice(ctx, item.ID)
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: generated.ID.String()}, nil
	}
}
func invoicesMarkPaidHandler(deps Deps) func(context.Context, invoiceWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in invoiceWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedInvoice(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if strings.TrimSpace(in.PaymentRef) == "" {
			return ackResponse{}, badReq("payment reference is required")
		}
		if err := deps.Service.MarkInvoicePaid(ctx, item.ID, strings.TrimSpace(in.PaymentRef)); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func invoicesVoidHandler(deps Deps) func(context.Context, invoiceWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in invoiceWriteInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedInvoice(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.MarkInvoiceVoided(ctx, item.ID, strings.TrimSpace(in.Reason)); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func couponsListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (couponsResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (couponsResponse, error) {
		list, err := deps.Service.ListCoupons(ctx, scopedApp(deps, p))
		if err != nil {
			return couponsResponse{}, mapErr(err)
		}
		out := couponsResponse{Coupons: make([]couponSummary, 0, len(list))}
		for _, item := range list {
			if item.AppID == scopedApp(deps, p) {
				row := couponSummary{ID: item.ID.String(), Code: item.Code, Name: item.Name, Type: string(item.Type), Amount: item.Amount.Amount, Percentage: item.Percentage, Currency: item.Currency, MaxRedemptions: item.MaxRedemptions, TimesRedeemed: item.TimesRedeemed}
				if item.ValidFrom != nil {
					row.ValidFrom = item.ValidFrom.Format(time.DateOnly)
				}
				if item.ValidUntil != nil {
					row.ValidUntil = item.ValidUntil.Format(time.DateOnly)
				}
				out.Coupons = append(out.Coupons, row)
			}
		}
		return out, nil
	}
}
func couponsCreateHandler(deps Deps) func(context.Context, couponWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in couponWriteInput, p contract.Principal) (ackResponse, error) {
		if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" {
			return ackResponse{}, badReq("code and name are required")
		}
		kind := coupon.CouponType(in.Type)
		if kind != coupon.CouponTypePercentage && kind != coupon.CouponTypeAmount {
			return ackResponse{}, badReq("invalid coupon type")
		}
		if kind == coupon.CouponTypePercentage && (in.Percentage <= 0 || in.Percentage > 100) {
			return ackResponse{}, badReq("percentage must be between 1 and 100")
		}
		if kind == coupon.CouponTypeAmount && in.Amount <= 0 {
			return ackResponse{}, badReq("amount must be positive")
		}
		currency := strings.ToLower(strings.TrimSpace(in.Currency))
		if currency == "" {
			currency = "usd"
		}
		item := &coupon.Coupon{Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), Type: kind, Amount: types.Money{Amount: in.Amount, Currency: currency}, Percentage: in.Percentage, Currency: currency, MaxRedemptions: in.MaxRedemptions, AppID: scopedApp(deps, p)}
		if in.MaxRedemptions < 0 {
			return ackResponse{}, badReq("max redemptions cannot be negative")
		}
		if in.ValidFrom != "" {
			date, err := time.Parse(time.DateOnly, in.ValidFrom)
			if err != nil {
				return ackResponse{}, badReq("invalid start date")
			}
			item.ValidFrom = &date
		}
		if in.ValidUntil != "" {
			date, err := time.Parse(time.DateOnly, in.ValidUntil)
			if err != nil {
				return ackResponse{}, badReq("invalid expiry date")
			}
			item.ValidUntil = &date
		}
		if item.ValidFrom != nil && item.ValidUntil != nil && item.ValidUntil.Before(*item.ValidFrom) {
			return ackResponse{}, badReq("expiry must follow start date")
		}
		if err := deps.Service.CreateCoupon(ctx, item); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func couponsDeleteHandler(deps Deps) func(context.Context, couponWriteInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in couponWriteInput, p contract.Principal) (ackResponse, error) {
		list, err := deps.Service.ListCoupons(ctx, scopedApp(deps, p))
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		for _, item := range list {
			if item.ID.String() == in.ID && item.AppID == scopedApp(deps, p) {
				if err := deps.Service.DeleteCoupon(ctx, item.ID); err != nil {
					return ackResponse{}, mapErr(err)
				}
				return ackResponse{OK: true, ID: in.ID}, nil
			}
		}
		return ackResponse{}, notFound("coupon")
	}
}

type catalogFeatureInput struct {
	ID           string `json:"id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	DefaultLimit int64  `json:"defaultLimit"`
	Period       string `json:"period"`
	SoftLimit    bool   `json:"softLimit"`
}
type catalogFeatureSummary struct {
	catalogFeatureInput
	Status string `json:"status"`
}
type catalogFeaturesResponse struct {
	Features []catalogFeatureSummary `json:"features"`
}

func projectCatalogFeature(item *feature.Feature) catalogFeatureSummary {
	return catalogFeatureSummary{catalogFeatureInput: catalogFeatureInput{ID: item.ID.String(), Key: item.Key, Name: item.Name, Description: item.Description, Type: string(item.Type), DefaultLimit: item.DefaultLimit, Period: string(item.Period), SoftLimit: item.SoftLimit}, Status: string(item.Status)}
}
func scopedCatalogFeature(ctx context.Context, deps Deps, p contract.Principal, raw string) (*feature.Feature, error) {
	id, err := ledgerid.ParseFeatureID(strings.TrimSpace(raw))
	if err != nil {
		return nil, badReq("invalid feature id")
	}
	item, err := deps.Service.GetCatalogFeature(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if item == nil || item.AppID != scopedApp(deps, p) {
		return nil, notFound("feature")
	}
	return item, nil
}
func validateCatalogFeature(in catalogFeatureInput) error {
	if strings.TrimSpace(in.Key) == "" || strings.TrimSpace(in.Name) == "" {
		return badReq("key and name are required")
	}
	if in.Type != string(feature.FeatureBoolean) && in.Type != string(feature.FeatureMetered) && in.Type != string(feature.FeatureSeat) {
		return badReq("invalid feature type")
	}
	if in.Period != "" && in.Period != string(feature.PeriodNone) && in.Period != string(feature.PeriodMonthly) && in.Period != string(feature.PeriodYearly) {
		return badReq("invalid feature period")
	}
	return nil
}
func featuresListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (catalogFeaturesResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (catalogFeaturesResponse, error) {
		list, err := deps.Service.ListCatalogFeatures(ctx, scopedApp(deps, p))
		if err != nil {
			return catalogFeaturesResponse{}, mapErr(err)
		}
		out := catalogFeaturesResponse{Features: make([]catalogFeatureSummary, 0, len(list))}
		for _, item := range list {
			if item.AppID == scopedApp(deps, p) {
				out.Features = append(out.Features, projectCatalogFeature(item))
			}
		}
		return out, nil
	}
}
func featuresCreateHandler(deps Deps) func(context.Context, catalogFeatureInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in catalogFeatureInput, p contract.Principal) (ackResponse, error) {
		if err := validateCatalogFeature(in); err != nil {
			return ackResponse{}, err
		}
		item := &feature.Feature{Key: strings.TrimSpace(in.Key), Name: strings.TrimSpace(in.Name), Description: in.Description, Type: feature.FeatureType(in.Type), DefaultLimit: in.DefaultLimit, Period: feature.Period(in.Period), SoftLimit: in.SoftLimit, Status: feature.StatusActive, AppID: scopedApp(deps, p)}
		if err := deps.Service.CreateCatalogFeature(ctx, item); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func featuresUpdateHandler(deps Deps) func(context.Context, catalogFeatureInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in catalogFeatureInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedCatalogFeature(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := validateCatalogFeature(in); err != nil {
			return ackResponse{}, err
		}
		if item.Key != strings.TrimSpace(in.Key) {
			return ackResponse{}, badReq("feature key cannot change")
		}
		item.Name = strings.TrimSpace(in.Name)
		item.Description = in.Description
		item.Type = feature.FeatureType(in.Type)
		item.DefaultLimit = in.DefaultLimit
		item.Period = feature.Period(in.Period)
		item.SoftLimit = in.SoftLimit
		if err := deps.Service.UpdateCatalogFeature(ctx, item); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
func featuresArchiveHandler(deps Deps) func(context.Context, catalogFeatureInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in catalogFeatureInput, p contract.Principal) (ackResponse, error) {
		item, err := scopedCatalogFeature(ctx, deps, p, in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Service.ArchiveCatalogFeature(ctx, item.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: item.ID.String()}, nil
	}
}
