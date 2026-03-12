package subscription

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"

	"github.com/xraph/ledger/coupon"
	"github.com/xraph/ledger/feature"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	lsub "github.com/xraph/ledger/subscription"
	"github.com/xraph/ledger/types"

	subdash "github.com/xraph/authsome/plugins/subscription/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
	_ dashboard.UserDetailContributor    = (*Plugin)(nil)
	_ dashboard.OrgDetailContributor     = (*Plugin)(nil)
	_ dashboard.OrgDetailTabContributor  = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// DashboardPlugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns subscription widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "subscription-overview",
			Title:      "Subscriptions",
			Size:       "md",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				appID := p.resolveAppID(ctx)
				data := subdash.OverviewWidgetData{}
				if p.ledgerStore != nil {
					if subs, err := p.ledgerStore.ListSubscriptions(ctx, "", appID, lsub.ListOpts{Status: lsub.StatusActive}); err == nil {
						data.ActiveCount = len(subs)
					}
					if subs, err := p.ledgerStore.ListSubscriptions(ctx, "", appID, lsub.ListOpts{Status: lsub.StatusTrialing}); err == nil {
						data.TrialCount = len(subs)
					}
					if subs, err := p.ledgerStore.ListSubscriptions(ctx, "", appID, lsub.ListOpts{Status: lsub.StatusPastDue}); err == nil {
						data.PastDueCount = len(subs)
					}
					if plans, err := p.ledgerStore.ListPlans(ctx, appID, plan.ListOpts{}); err == nil {
						data.PlanCount = len(plans)
					}
					data.MRR = p.service.CalculateMRR(ctx, appID)
				}
				return subdash.OverviewWidget(data)
			},
		},
	}
}

// DashboardSettingsPanel returns the subscription settings panel.
func (p *Plugin) DashboardSettingsPanel(ctx context.Context) templ.Component {
	appID := p.resolveAppID(ctx)
	opts := settings.ResolveOpts{AppID: appID}

	defaultPlan, _ := settings.Get(ctx, p.settings, SettingDefaultPlan, opts)
	tenantMode, _ := settings.Get(ctx, p.settings, SettingTenantMode, opts)
	autoSubOrg, _ := settings.Get(ctx, p.settings, SettingAutoSubscribeOrg, opts)
	autoSubUser, _ := settings.Get(ctx, p.settings, SettingAutoSubscribeUser, opts)
	trialDays, _ := settings.Get(ctx, p.settings, SettingTrialDays, opts)
	selfService, _ := settings.Get(ctx, p.settings, SettingSelfServiceUpgrade, opts)
	graceDays, _ := settings.Get(ctx, p.settings, SettingGracePeriodDays, opts)

	return subdash.SettingsPanel(subdash.SettingsPanelData{
		DefaultPlan:      defaultPlan,
		TenantMode:       tenantMode,
		AutoSubscribeOrg: autoSubOrg,
		AutoSubscribeUsr: autoSubUser,
		TrialDays:        trialDays,
		SelfService:      selfService,
		GracePeriodDays:  graceDays,
	})
}

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for billing pages.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{Label: "Plans", Path: "/plans", Icon: "layers", Group: "Billing", Priority: 0},
		{Label: "Subscriptions", Path: "/subscriptions", Icon: "credit-card", Group: "Billing", Priority: 1},
		{Label: "Invoices", Path: "/invoices", Icon: "file-text", Group: "Billing", Priority: 2},
		{Label: "Coupons", Path: "/coupons", Icon: "ticket", Group: "Billing", Priority: 3},
		{Label: "Features", Path: "/features", Icon: "puzzle", Group: "Billing", Priority: 4},
	}
}

// DashboardRenderPage renders billing pages.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch route {
	case "/plans":
		return p.renderPlansPage(ctx, params)
	case "/plans/detail":
		return p.renderPlanDetail(ctx, params)
	case "/subscriptions":
		return p.renderSubscriptionsPage(ctx, params)
	case "/subscriptions/detail":
		return p.renderSubscriptionDetail(ctx, params)
	case "/invoices":
		return p.renderInvoicesPage(ctx, params)
	case "/invoices/detail":
		return p.renderInvoiceDetail(ctx, params)
	case "/coupons":
		return p.renderCouponsPage(ctx, params)
	case "/features":
		return p.renderFeaturesPage(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// User/Org detail contributors
// ──────────────────────────────────────────────────

func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.ledger == nil {
		return nil
	}
	appID := p.resolveAppID(ctx)
	view := subdash.ActiveSubView{}
	sub, err := p.ledger.GetActiveSubscription(ctx, userID.String(), appID)
	if err == nil && sub != nil {
		view.HasSub = true
		view.Status = string(sub.Status)
		view.PeriodEnd = sub.CurrentPeriodEnd
		if pl, err := p.ledger.GetPlan(ctx, sub.PlanID); err == nil {
			view.PlanName = pl.Name
		}
	}
	return subdash.UserSection(view)
}

func (p *Plugin) DashboardOrgDetailSection(ctx context.Context, orgID id.OrgID) templ.Component {
	if p.ledger == nil {
		return nil
	}
	appID := p.resolveAppID(ctx)
	view := subdash.ActiveSubView{}
	sub, err := p.ledger.GetActiveSubscription(ctx, orgID.String(), appID)
	if err == nil && sub != nil {
		view.HasSub = true
		view.Status = string(sub.Status)
		view.PeriodEnd = sub.CurrentPeriodEnd
		if pl, err := p.ledger.GetPlan(ctx, sub.PlanID); err == nil {
			view.PlanName = pl.Name
		}
	}
	return subdash.OrgSection(view)
}

// ──────────────────────────────────────────────────
// OrgDetailTabContributor
// ──────────────────────────────────────────────────

func (p *Plugin) DashboardOrgDetailTabs(ctx context.Context, orgID id.OrgID) []dashboard.OrgDetailTab {
	return []dashboard.OrgDetailTab{
		{
			ID:       "billing",
			Label:    "Billing",
			Icon:     "credit-card",
			Priority: 50,
			Render: func(ctx context.Context, orgID id.OrgID) templ.Component {
				return p.renderOrgBillingTab(ctx, orgID)
			},
		},
	}
}

func (p *Plugin) renderOrgBillingTab(ctx context.Context, orgID id.OrgID) templ.Component {
	appID := p.resolveAppID(ctx)
	data := subdash.OrgBillingTabData{}

	sub, err := p.ledger.GetActiveSubscription(ctx, orgID.String(), appID)
	if err == nil && sub != nil {
		data.HasSub = true
		sv := toSubView(sub)
		if pl, err := p.ledger.GetPlan(ctx, sub.PlanID); err == nil {
			sv.PlanName = pl.Name
			pv := toPlanView(pl)
			data.Plan = &pv
		}
		data.Subscription = &sv

		// Usage.
		if summaries, err := p.service.GetUsageSummary(ctx, sub.TenantID, sub.AppID); err == nil {
			data.Usage = make([]subdash.UsageView, 0, len(summaries))
			for _, u := range summaries {
				pct := 0
				if u.Limit > 0 {
					pct = int(float64(u.Used) / float64(u.Limit) * 100)
					if pct > 100 {
						pct = 100
					}
				}
				data.Usage = append(data.Usage, subdash.UsageView{
					FeatureKey: u.FeatureKey, FeatureName: u.FeatureName, FeatureType: u.FeatureType,
					Used: u.Used, Limit: u.Limit, Remaining: u.Remaining,
					Period: u.Period, Percentage: pct,
				})
			}
		}

		// Invoices.
		if invoices, err := p.service.ListInvoices(ctx, sub.TenantID, sub.AppID); err == nil {
			data.Invoices = make([]subdash.InvoiceView, 0, len(invoices))
			for _, inv := range invoices {
				data.Invoices = append(data.Invoices, toInvoiceView(inv))
			}
		}
	}

	return subdash.OrgBillingTab(data)
}

// ──────────────────────────────────────────────────
// Page renderers
// ──────────────────────────────────────────────────

func (p *Plugin) renderPlansPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data subdash.PlansPageData

	action := params.FormData["action"]
	switch action {
	case "create":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashCreatePlan(ctx, appID, params)
		}
	case "archive":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashArchivePlan(ctx, params)
		}
	case "activate":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashActivatePlan(ctx, params)
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.Tab = params.QueryParams["tab"]
	if data.Tab == "" {
		data.Tab = "plans"
	}

	if p.ledgerStore != nil {
		plans, err := p.ledgerStore.ListPlans(ctx, appID, plan.ListOpts{})
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load plans: %v", err)
		} else {
			data.Plans = make([]subdash.PlanView, 0, len(plans))
			data.Addons = make([]subdash.PlanView, 0)
			for _, pl := range plans {
				pv := toPlanView(pl)
				data.TotalFeatures += pv.FeaturesCount
				if pv.Status == "active" {
					data.ActiveCount++
				} else if pv.Status == "archived" {
					data.ArchivedCount++
				}
				if pv.IsAddon {
					data.Addons = append(data.Addons, pv)
				} else {
					data.Plans = append(data.Plans, pv)
				}
			}
		}
	}

	return subdash.PlansPage(data), nil
}

func (p *Plugin) renderPlanDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	planIDStr := params.QueryParams["id"]
	if planIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}
	planID, err := ledgerid.ParsePlanID(planIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}
	pl, err := p.service.GetPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("subscription dashboard: get plan: %w", err)
	}

	var data subdash.PlanDetailPageData

	action := params.FormData["action"]
	switch action {
	case "add_feature":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashAddFeature(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "remove_feature":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashRemoveFeature(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "update_pricing":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashUpdatePricing(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "update_info":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashUpdatePlanInfo(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "edit_feature":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashEditFeature(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "add_tier":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashAddTier(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	case "remove_tier":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashRemoveTier(ctx, pl, params)
			if pl2, err := p.service.GetPlan(ctx, planID); err == nil {
				pl = pl2
			}
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.Plan = toPlanDetailView(pl)
	data.Plan.SubscriberCount = p.service.CountSubscribers(ctx, pl.ID, pl.AppID)

	// Populate catalog features for quick-add dropdown.
	if catalogFeatures, err := p.service.ListCatalogFeatures(ctx, pl.AppID); err == nil {
		for _, cf := range catalogFeatures {
			if cf.Status == feature.StatusActive {
				data.CatalogFeatures = append(data.CatalogFeatures, toCatalogFeatureView(cf))
			}
		}
	}

	// Populate feature keys for tier form dropdown.
	for _, f := range pl.Features {
		data.TierForm.FeatureKeys = append(data.TierForm.FeatureKeys, f.Key)
	}

	return subdash.PlanDetailPage(data), nil
}

func (p *Plugin) renderSubscriptionsPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data subdash.SubscriptionsPageData

	action := params.FormData["action"]
	switch action {
	case "cancel":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashAction(ctx, params.FormData["sub_id"], func(subID ledgerid.SubscriptionID) error {
				return p.service.CancelSubscription(ctx, subID, false)
			})
		}
	case "create":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashCreateSub(ctx, appID, params)
		}
	case "pause":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashAction(ctx, params.FormData["sub_id"], func(subID ledgerid.SubscriptionID) error {
				return p.service.PauseSubscription(ctx, subID)
			})
		}
	case "resume":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashAction(ctx, params.FormData["sub_id"], func(subID ledgerid.SubscriptionID) error {
				return p.service.ResumeSubscription(ctx, subID)
			})
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.StatusFilter = params.QueryParams["status"]

	if p.ledgerStore != nil {
		opts := lsub.ListOpts{}
		if data.StatusFilter != "" {
			opts.Status = lsub.Status(data.StatusFilter)
		}
		subs, err := p.ledgerStore.ListSubscriptions(ctx, "", appID, opts)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load subscriptions: %v", err)
		} else {
			data.Subscriptions = make([]subdash.SubscriptionView, 0, len(subs))
			for _, s := range subs {
				sv := toSubView(s)
				if pl, err := p.ledger.GetPlan(ctx, s.PlanID); err == nil {
					sv.PlanName = pl.Name
				}
				data.Subscriptions = append(data.Subscriptions, sv)
				switch sv.Status {
				case "active":
					data.ActiveCount++
				case "trialing":
					data.TrialCount++
				case "past_due":
					data.PastDueCount++
				case "canceled":
					data.CanceledCount++
				}
			}
		}

		if plans, err := p.ledgerStore.ListPlans(ctx, appID, plan.ListOpts{}); err == nil {
			data.Plans = make([]subdash.PlanView, 0, len(plans))
			for _, pl := range plans {
				if pl.Status == plan.StatusActive {
					data.Plans = append(data.Plans, toPlanView(pl))
				}
			}
		}
	}

	return subdash.SubscriptionsPage(data), nil
}

func (p *Plugin) renderSubscriptionDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	subIDStr := params.QueryParams["id"]
	if subIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}
	subID, err := ledgerid.ParseSubscriptionID(subIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}
	sub, err := p.service.GetSubscription(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("subscription dashboard: get subscription: %w", err)
	}

	var data subdash.SubscriptionDetailPageData

	action := params.FormData["action"]
	switch action {
	case "change_plan":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			newPlanIDStr := params.FormData["plan_id"]
			if newPlanID, err := ledgerid.ParsePlanID(newPlanIDStr); err == nil {
				if err := p.service.ChangePlan(ctx, subID, newPlanID); err != nil {
					data.Error = fmt.Sprintf("Failed to change plan: %v", err)
				} else {
					data.Success = "Plan changed."
					if sub2, err := p.service.GetSubscription(ctx, subID); err == nil {
						sub = sub2
					}
				}
			}
		}
	case "cancel":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if err := p.service.CancelSubscription(ctx, subID, false); err != nil {
				data.Error = fmt.Sprintf("Failed to cancel: %v", err)
			} else {
				data.Success = "Subscription canceled."
				if sub2, err := p.service.GetSubscription(ctx, subID); err == nil {
					sub = sub2
				}
			}
		}
	case "pause":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if err := p.service.PauseSubscription(ctx, subID); err != nil {
				data.Error = fmt.Sprintf("Failed to pause: %v", err)
			} else {
				data.Success = "Subscription paused."
				if sub2, err := p.service.GetSubscription(ctx, subID); err == nil {
					sub = sub2
				}
			}
		}
	case "resume":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if err := p.service.ResumeSubscription(ctx, subID); err != nil {
				data.Error = fmt.Sprintf("Failed to resume: %v", err)
			} else {
				data.Success = "Subscription resumed."
				if sub2, err := p.service.GetSubscription(ctx, subID); err == nil {
					sub = sub2
				}
			}
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.ActiveTab = params.QueryParams["tab"]
	if data.ActiveTab == "" {
		data.ActiveTab = "overview"
	}
	data.Subscription = toSubView(sub)

	if pl, err := p.service.GetPlan(ctx, sub.PlanID); err == nil {
		data.Subscription.PlanName = pl.Name
		data.Plan = toPlanDetailView(pl)
	}

	if summaries, err := p.service.GetUsageSummary(ctx, sub.TenantID, sub.AppID); err == nil {
		data.Usage = make([]subdash.UsageView, 0, len(summaries))
		for _, u := range summaries {
			pct := 0
			if u.Limit > 0 {
				pct = int(float64(u.Used) / float64(u.Limit) * 100)
				if pct > 100 {
					pct = 100
				}
			}
			data.Usage = append(data.Usage, subdash.UsageView{
				FeatureKey: u.FeatureKey, FeatureName: u.FeatureName, FeatureType: u.FeatureType,
				Used: u.Used, Limit: u.Limit, Remaining: u.Remaining,
				Period: u.Period, Percentage: pct,
			})
		}
	}

	if invoices, err := p.service.ListInvoices(ctx, sub.TenantID, sub.AppID); err == nil {
		data.Invoices = make([]subdash.InvoiceView, 0, len(invoices))
		for _, inv := range invoices {
			data.Invoices = append(data.Invoices, toInvoiceView(inv))
		}
	}

	appID := p.resolveAppID(ctx)
	if plans, err := p.ledgerStore.ListPlans(ctx, appID, plan.ListOpts{}); err == nil {
		data.Plans = make([]subdash.PlanView, 0, len(plans))
		for _, pl := range plans {
			if pl.Status == plan.StatusActive {
				data.Plans = append(data.Plans, toPlanView(pl))
			}
		}
	}

	return subdash.SubscriptionDetailPage(data), nil
}

func (p *Plugin) renderInvoicesPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data subdash.InvoicesPageData
	data.StatusFilter = params.QueryParams["status"]

	if p.ledgerStore != nil {
		invoices, err := p.ledgerStore.ListInvoices(ctx, "", appID, invoice.ListOpts{})
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load invoices: %v", err)
		} else {
			data.Invoices = make([]subdash.InvoiceView, 0, len(invoices))
			for _, inv := range invoices {
				iv := toInvoiceView(inv)
				data.Invoices = append(data.Invoices, iv)
				switch iv.Status {
				case "paid":
					data.PaidCount++
				case "pending", "draft":
					data.PendingCount++
				case "past_due":
					data.OverdueCount++
				}
			}
		}
	}

	return subdash.InvoicesPage(data), nil
}

func (p *Plugin) renderInvoiceDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	invIDStr := params.QueryParams["id"]
	if invIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}
	invID, err := ledgerid.ParseInvoiceID(invIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}
	inv, err := p.service.GetInvoice(ctx, invID)
	if err != nil {
		return nil, fmt.Errorf("subscription dashboard: get invoice: %w", err)
	}

	var data subdash.InvoiceDetailPageData

	action := params.FormData["action"]
	switch action {
	case "mark_paid":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			ref := params.FormData["payment_ref"]
			if ref == "" {
				ref = "dashboard"
			}
			if err := p.service.MarkInvoicePaid(ctx, invID, ref); err != nil {
				data.Error = fmt.Sprintf("Failed to mark as paid: %v", err)
			} else {
				data.Success = "Invoice marked as paid."
				if inv2, err := p.service.GetInvoice(ctx, invID); err == nil {
					inv = inv2
				}
			}
		}
	case "void":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if err := p.service.MarkInvoiceVoided(ctx, invID, params.FormData["reason"]); err != nil {
				data.Error = fmt.Sprintf("Failed to void invoice: %v", err)
			} else {
				data.Success = "Invoice voided."
				if inv2, err := p.service.GetInvoice(ctx, invID); err == nil {
					inv = inv2
				}
			}
		}
	}

	data.FormNonce = dashboard.GenerateNonce()
	data.Invoice = toInvoiceView(inv)
	data.LineItems = make([]subdash.LineItemView, 0, len(inv.LineItems))
	for _, li := range inv.LineItems {
		data.LineItems = append(data.LineItems, subdash.LineItemView{
			Description: li.Description,
			Type:        string(li.Type),
			FeatureKey:  li.FeatureKey,
			Quantity:    li.Quantity,
			UnitAmount:  li.UnitAmount.FormatMajor(),
			Amount:      li.Amount.FormatMajor(),
		})
	}

	return subdash.InvoiceDetailPage(data), nil
}

func (p *Plugin) renderCouponsPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data subdash.CouponsPageData

	action := params.FormData["action"]
	switch action {
	case "create":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashCreateCoupon(ctx, appID, params)
		}
	case "delete":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			p.handleDashDeleteCoupon(ctx, params)
		}
	}

	data.FormNonce = dashboard.GenerateNonce()

	if p.ledgerStore != nil {
		coupons, err := p.service.ListCoupons(ctx, appID)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load coupons: %v", err)
		} else {
			now := time.Now()
			data.Coupons = make([]subdash.CouponView, 0, len(coupons))
			for _, c := range coupons {
				data.Coupons = append(data.Coupons, toCouponView(c, now))
			}
		}
	}

	return subdash.CouponsPage(data), nil
}

func (p *Plugin) renderFeaturesPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID := p.resolveAppID(ctx)
	var data subdash.CatalogFeaturesPageData

	action := params.FormData["action"]
	switch action {
	case "create":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			data.Error, data.Success = p.handleDashCreateCatalogFeature(ctx, appID, params)
		}
	case "archive":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if fid, err := ledgerid.ParseFeatureID(params.FormData["feature_id"]); err == nil {
				if err := p.service.ArchiveCatalogFeature(ctx, fid); err != nil {
					data.Error = fmt.Sprintf("Failed to archive feature: %v", err)
				} else {
					data.Success = "Feature archived."
				}
			}
		}
	case "activate":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if fid, err := ledgerid.ParseFeatureID(params.FormData["feature_id"]); err == nil {
				f, err := p.service.GetCatalogFeature(ctx, fid)
				if err != nil {
					data.Error = fmt.Sprintf("Failed to get feature: %v", err)
				} else {
					f.Status = feature.StatusActive
					f.Entity.Touch()
					if err := p.service.UpdateCatalogFeature(ctx, f); err != nil {
						data.Error = fmt.Sprintf("Failed to activate feature: %v", err)
					} else {
						data.Success = "Feature activated."
					}
				}
			}
		}
	case "delete":
		if dashboard.ConsumeNonce(params.FormData["nonce"]) {
			if fid, err := ledgerid.ParseFeatureID(params.FormData["feature_id"]); err == nil {
				if err := p.service.DeleteCatalogFeature(ctx, fid); err != nil {
					data.Error = fmt.Sprintf("Failed to delete feature: %v", err)
				} else {
					data.Success = "Feature deleted."
				}
			}
		}
	}

	data.FormNonce = dashboard.GenerateNonce()

	if p.ledgerStore != nil {
		features, err := p.service.ListCatalogFeatures(ctx, appID)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to load features: %v", err)
		} else {
			data.Features = make([]subdash.CatalogFeatureView, 0, len(features))
			for _, f := range features {
				fv := toCatalogFeatureView(f)
				data.Features = append(data.Features, fv)
				switch f.Status {
				case feature.StatusActive:
					data.ActiveCount++
				case feature.StatusDraft:
					data.DraftCount++
				case feature.StatusArchived:
					data.ArchivedCount++
				}
			}
		}
	}

	return subdash.FeaturesPage(data), nil
}

// ──────────────────────────────────────────────────
// Form handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleDashCreatePlan(ctx context.Context, appID string, params contributor.Params) (errMsg, successMsg string) {
	name := params.FormData["name"]
	slug := params.FormData["slug"]
	if name == "" || slug == "" {
		return "Name and slug are required.", ""
	}

	currency := params.FormData["currency"]
	if currency == "" {
		currency = "usd"
	}
	trialDays, _ := strconv.Atoi(params.FormData["trial_days"])

	pl := &plan.Plan{
		Name: name, Slug: slug,
		Description: params.FormData["description"],
		Currency:    currency,
		Status:      plan.StatusDraft,
		TrialDays:   trialDays,
		AppID:       appID,
	}

	if params.FormData["is_addon"] == "true" {
		pl.Metadata = map[string]string{"addon": "true"}
	}

	if baseStr := params.FormData["base_amount"]; baseStr != "" {
		if cents := parseAmountCents(baseStr); cents > 0 {
			period := plan.PeriodMonthly
			if params.FormData["billing_period"] == "yearly" {
				period = plan.PeriodYearly
			}
			pl.Pricing = &plan.Pricing{
				BaseAmount:    types.Money{Amount: cents, Currency: currency},
				BillingPeriod: period,
			}
		}
	}

	if err := p.service.CreatePlan(ctx, pl); err != nil {
		return fmt.Sprintf("Failed to create plan: %v", err), ""
	}
	return "", fmt.Sprintf("Plan %q created as draft.", name)
}

func (p *Plugin) handleDashArchivePlan(ctx context.Context, params contributor.Params) {
	if pid, err := ledgerid.ParsePlanID(params.FormData["plan_id"]); err == nil {
		_ = p.service.ArchivePlan(ctx, pid)
	}
}

func (p *Plugin) handleDashActivatePlan(ctx context.Context, params contributor.Params) {
	if pid, err := ledgerid.ParsePlanID(params.FormData["plan_id"]); err == nil {
		_ = p.service.ActivatePlan(ctx, pid)
	}
}

func (p *Plugin) handleDashAddFeature(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	key := params.FormData["feature_key"]
	name := params.FormData["feature_name"]
	if key == "" || name == "" {
		return "Feature key and name are required.", ""
	}

	f := plan.Feature{
		ID:   ledgerid.NewFeatureID(),
		Key:  key,
		Name: name,
		Type: plan.FeatureType(params.FormData["feature_type"]),
	}
	if f.Type == "" {
		f.Type = plan.FeatureBoolean
	}
	if v := params.FormData["feature_limit"]; v != "" {
		f.Limit, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.FormData["feature_period"]; v != "" {
		f.Period = plan.Period(v)
	}
	if params.FormData["feature_soft_limit"] == "true" {
		f.SoftLimit = true
	}
	if v := params.FormData["catalog_id"]; v != "" {
		if cid, err := ledgerid.ParseFeatureID(v); err == nil {
			f.CatalogID = cid
		}
	}

	pl.Features = append(pl.Features, f)
	pl.Entity.Touch()
	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to add feature: %v", err), ""
		}
	}
	return "", fmt.Sprintf("Feature %q added.", name)
}

func (p *Plugin) handleDashRemoveFeature(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	featureID := params.FormData["feature_id"]
	if featureID == "" {
		return "Feature ID is required.", ""
	}
	newFeatures := make([]plan.Feature, 0, len(pl.Features))
	var removed string
	for _, f := range pl.Features {
		if f.ID.String() == featureID {
			removed = f.Name
			continue
		}
		newFeatures = append(newFeatures, f)
	}
	if removed == "" {
		return "Feature not found.", ""
	}
	pl.Features = newFeatures
	pl.Entity.Touch()
	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to remove feature: %v", err), ""
		}
	}
	return "", fmt.Sprintf("Feature %q removed.", removed)
}

func (p *Plugin) handleDashUpdatePricing(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	cents := parseAmountCents(params.FormData["base_amount"])
	period := plan.PeriodMonthly
	if params.FormData["billing_period"] == "yearly" {
		period = plan.PeriodYearly
	}
	if cents > 0 {
		pl.Pricing = &plan.Pricing{
			BaseAmount:    types.Money{Amount: cents, Currency: pl.Currency},
			BillingPeriod: period,
		}
	} else {
		pl.Pricing = nil
	}
	pl.Entity.Touch()
	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to update pricing: %v", err), ""
		}
	}
	return "", "Pricing updated."
}

func (p *Plugin) handleDashUpdatePlanInfo(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	if v := params.FormData["name"]; v != "" {
		pl.Name = v
	}
	if v := params.FormData["description"]; v != "" {
		pl.Description = v
	}
	if v := params.FormData["trial_days"]; v != "" {
		pl.TrialDays, _ = strconv.Atoi(v)
	}
	pl.Entity.Touch()
	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to update plan: %v", err), ""
		}
	}
	return "", "Plan updated."
}

func (p *Plugin) handleDashCreateSub(ctx context.Context, appID string, params contributor.Params) (errMsg, successMsg string) {
	tenantID := params.FormData["tenant_id"]
	planIDStr := params.FormData["plan_id"]
	if tenantID == "" || planIDStr == "" {
		return "Tenant ID and plan are required.", ""
	}
	planID, err := ledgerid.ParsePlanID(planIDStr)
	if err != nil {
		return "Invalid plan ID.", ""
	}
	sub, err := p.service.Subscribe(ctx, tenantID, planID, appID)
	if err != nil {
		return fmt.Sprintf("Failed to create subscription: %v", err), ""
	}
	return "", fmt.Sprintf("Subscription %s created.", sub.ID.String())
}

func (p *Plugin) handleDashCreateCoupon(ctx context.Context, appID string, params contributor.Params) (errMsg, successMsg string) {
	code := strings.TrimSpace(params.FormData["code"])
	name := strings.TrimSpace(params.FormData["name"])
	if code == "" || name == "" {
		return "Code and name are required.", ""
	}

	currency := params.FormData["currency"]
	if currency == "" {
		currency = "usd"
	}

	c := &coupon.Coupon{
		Code: strings.ToUpper(code), Name: name,
		Currency: currency, AppID: appID,
	}

	switch params.FormData["type"] {
	case "percentage":
		c.Type = coupon.CouponTypePercentage
		c.Percentage, _ = strconv.Atoi(params.FormData["percentage"])
	default:
		c.Type = coupon.CouponTypeAmount
		c.Amount = types.Money{Amount: parseAmountCents(params.FormData["amount"]), Currency: currency}
	}

	if v := params.FormData["max_redemptions"]; v != "" {
		c.MaxRedemptions, _ = strconv.Atoi(v)
	}
	if v := params.FormData["valid_from"]; v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			c.ValidFrom = &t
		}
	}
	if v := params.FormData["valid_until"]; v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			c.ValidUntil = &t
		}
	}

	if err := p.service.CreateCoupon(ctx, c); err != nil {
		return fmt.Sprintf("Failed to create coupon: %v", err), ""
	}
	return "", fmt.Sprintf("Coupon %q created.", code)
}

func (p *Plugin) handleDashDeleteCoupon(ctx context.Context, params contributor.Params) {
	if cid, err := ledgerid.ParseCouponID(params.FormData["coupon_id"]); err == nil {
		_ = p.service.DeleteCoupon(ctx, cid)
	}
}

func (p *Plugin) handleDashAction(_ context.Context, subIDStr string, fn func(ledgerid.SubscriptionID) error) {
	if subID, err := ledgerid.ParseSubscriptionID(subIDStr); err == nil {
		_ = fn(subID)
	}
}

func (p *Plugin) handleDashCreateCatalogFeature(ctx context.Context, appID string, params contributor.Params) (errMsg, successMsg string) {
	key := strings.TrimSpace(params.FormData["feature_key"])
	name := strings.TrimSpace(params.FormData["feature_name"])
	if key == "" || name == "" {
		return "Feature key and name are required.", ""
	}

	f := &feature.Feature{
		ID:     ledgerid.NewFeatureID(),
		Key:    key,
		Name:   name,
		Type:   feature.FeatureType(params.FormData["feature_type"]),
		Status: feature.StatusDraft,
		AppID:  appID,
	}
	if f.Type == "" {
		f.Type = feature.FeatureBoolean
	}
	if v := params.FormData["feature_description"]; v != "" {
		f.Description = v
	}
	if v := params.FormData["feature_default_limit"]; v != "" {
		f.DefaultLimit, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.FormData["feature_period"]; v != "" {
		f.Period = feature.Period(v)
	}
	if params.FormData["feature_soft_limit"] == "true" {
		f.SoftLimit = true
	}

	if err := p.service.CreateCatalogFeature(ctx, f); err != nil {
		return fmt.Sprintf("Failed to create feature: %v", err), ""
	}
	return "", fmt.Sprintf("Feature %q created as draft.", name)
}

func (p *Plugin) handleDashEditFeature(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	featureID := params.FormData["feature_id"]
	if featureID == "" {
		return "Feature ID is required.", ""
	}

	var found bool
	for i, f := range pl.Features {
		if f.ID.String() == featureID {
			found = true
			if v := params.FormData["feature_name"]; v != "" {
				pl.Features[i].Name = v
			}
			if v := params.FormData["feature_type"]; v != "" {
				pl.Features[i].Type = plan.FeatureType(v)
			}
			if v := params.FormData["feature_limit"]; v != "" {
				pl.Features[i].Limit, _ = strconv.ParseInt(v, 10, 64)
			}
			if v := params.FormData["feature_period"]; v != "" {
				pl.Features[i].Period = plan.Period(v)
			}
			pl.Features[i].SoftLimit = params.FormData["feature_soft_limit"] == "true"
			break
		}
	}
	if !found {
		return "Feature not found.", ""
	}

	pl.Entity.Touch()
	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to update feature: %v", err), ""
		}
	}
	return "", "Feature updated."
}

func (p *Plugin) handleDashAddTier(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	featureKey := params.FormData["tier_feature_key"]
	tierType := params.FormData["tier_type"]
	if featureKey == "" || tierType == "" {
		return "Feature key and tier type are required.", ""
	}

	tier := plan.PriceTier{
		FeatureKey: featureKey,
		Type:       plan.TierType(tierType),
	}
	if v := params.FormData["tier_up_to"]; v != "" {
		tier.UpTo, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.FormData["tier_unit_amount"]; v != "" {
		cents := parseAmountCents(v)
		tier.UnitAmount = types.Money{Amount: cents, Currency: pl.Currency}
	}
	if v := params.FormData["tier_flat_amount"]; v != "" {
		cents := parseAmountCents(v)
		tier.FlatAmount = types.Money{Amount: cents, Currency: pl.Currency}
	}

	if pl.Pricing == nil {
		pl.Pricing = &plan.Pricing{
			BaseAmount:    types.Money{Currency: pl.Currency},
			BillingPeriod: plan.PeriodMonthly,
		}
	}
	tier.Priority = len(pl.Pricing.Tiers)
	pl.Pricing.Tiers = append(pl.Pricing.Tiers, tier)
	pl.Entity.Touch()

	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to add tier: %v", err), ""
		}
	}
	return "", fmt.Sprintf("Tier for %q added.", featureKey)
}

func (p *Plugin) handleDashRemoveTier(ctx context.Context, pl *plan.Plan, params contributor.Params) (errMsg, successMsg string) {
	idxStr := params.FormData["tier_index"]
	if idxStr == "" {
		return "Tier index is required.", ""
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || pl.Pricing == nil || idx < 0 || idx >= len(pl.Pricing.Tiers) {
		return "Invalid tier index.", ""
	}

	pl.Pricing.Tiers = append(pl.Pricing.Tiers[:idx], pl.Pricing.Tiers[idx+1:]...)
	pl.Entity.Touch()

	if p.ledgerStore != nil {
		if err := p.ledgerStore.UpdatePlan(ctx, pl); err != nil {
			return fmt.Sprintf("Failed to remove tier: %v", err), ""
		}
	}
	return "", "Tier removed."
}

// ──────────────────────────────────────────────────
// View conversion helpers
// ──────────────────────────────────────────────────

func (p *Plugin) resolveAppID(ctx context.Context) string {
	if appID, ok := dashboard.AppIDFromContext(ctx); ok {
		return appID.String()
	}
	return p.defaultAppID
}

func toPlanView(pl *plan.Plan) subdash.PlanView {
	v := subdash.PlanView{
		ID: pl.ID.String(), Name: pl.Name, Slug: pl.Slug,
		Description: pl.Description, Currency: pl.Currency,
		Status: string(pl.Status), TrialDays: pl.TrialDays,
		FeaturesCount: len(pl.Features), Metadata: pl.Metadata,
	}
	if pl.Metadata != nil && pl.Metadata["addon"] == "true" {
		v.IsAddon = true
	}
	if pl.Pricing != nil {
		v.BaseAmount = pl.Pricing.BaseAmount.FormatMajor()
		v.BillingPeriod = string(pl.Pricing.BillingPeriod)
	}
	return v
}

func toPlanDetailView(pl *plan.Plan) subdash.PlanDetailView {
	v := subdash.PlanDetailView{PlanView: toPlanView(pl)}
	for _, f := range pl.Features {
		v.Features = append(v.Features, subdash.FeatureView{
			ID: f.ID.String(), CatalogID: f.CatalogID.String(),
			Key: f.Key, Name: f.Name,
			Type: string(f.Type), Limit: f.Limit,
			Period: string(f.Period), SoftLimit: f.SoftLimit,
		})
	}
	if pl.Pricing != nil {
		for _, t := range pl.Pricing.Tiers {
			v.Tiers = append(v.Tiers, subdash.TierView{
				FeatureKey: t.FeatureKey, Type: string(t.Type),
				UpTo: t.UpTo, UnitAmount: t.UnitAmount.FormatMajor(),
				FlatAmount: t.FlatAmount.FormatMajor(),
			})
		}
	}
	return v
}

func toSubView(s *lsub.Subscription) subdash.SubscriptionView {
	return subdash.SubscriptionView{
		ID: s.ID.String(), TenantID: s.TenantID, PlanID: s.PlanID.String(),
		Status: string(s.Status), CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd: s.CurrentPeriodEnd, TrialStart: s.TrialStart,
		TrialEnd: s.TrialEnd, CanceledAt: s.CanceledAt, CancelAt: s.CancelAt,
		EndedAt: s.EndedAt, AppID: s.AppID, ProviderName: s.ProviderName,
		Metadata: s.Metadata,
	}
}

func toInvoiceView(inv *invoice.Invoice) subdash.InvoiceView {
	return subdash.InvoiceView{
		ID: inv.ID.String(), TenantID: inv.TenantID,
		SubscriptionID: inv.SubscriptionID.String(),
		Status:         string(inv.Status), Currency: inv.Currency,
		Subtotal: inv.Subtotal.FormatMajor(), TaxAmount: inv.TaxAmount.FormatMajor(),
		DiscountAmount: inv.DiscountAmount.FormatMajor(), Total: inv.Total.FormatMajor(),
		PeriodStart: inv.PeriodStart, PeriodEnd: inv.PeriodEnd,
		DueDate: inv.DueDate, PaidAt: inv.PaidAt,
		PaymentRef: inv.PaymentRef, VoidReason: inv.VoidReason,
	}
}

func toCatalogFeatureView(f *feature.Feature) subdash.CatalogFeatureView {
	return subdash.CatalogFeatureView{
		ID: f.ID.String(), Key: f.Key, Name: f.Name,
		Description: f.Description, Type: string(f.Type),
		DefaultLimit: f.DefaultLimit, Period: string(f.Period),
		SoftLimit: f.SoftLimit, Status: string(f.Status),
	}
}

func toCouponView(c *coupon.Coupon, now time.Time) subdash.CouponView {
	v := subdash.CouponView{
		ID: c.ID.String(), Code: c.Code, Name: c.Name,
		Type: string(c.Type), Currency: c.Currency,
		MaxRedemptions: c.MaxRedemptions, TimesRedeemed: c.TimesRedeemed,
		ValidFrom: c.ValidFrom, ValidUntil: c.ValidUntil,
	}
	if c.Type == coupon.CouponTypePercentage {
		v.Percentage = c.Percentage
	} else {
		v.Amount = c.Amount.FormatMajor()
	}
	if c.ValidUntil != nil && now.After(*c.ValidUntil) {
		v.IsExpired = true
	}
	if c.MaxRedemptions > 0 && c.TimesRedeemed >= c.MaxRedemptions {
		v.IsExhausted = true
	}
	return v
}
