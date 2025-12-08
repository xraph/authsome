// Package ui provides Pine UI page templates for the subscription plugin dashboard
package ui

import (
	"fmt"
	"time"

	"github.com/xraph/authsome/plugins/subscription/core"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// PageLayout wraps content in a standard page layout
func PageLayout(title string, breadcrumbs []Breadcrumb, actions g.Node, content ...g.Node) g.Node {
	return html.Div(
		html.Class("space-y-6"),
		// Header
		html.Div(
			html.Class("md:flex md:items-center md:justify-between"),
			html.Div(
				html.Class("min-w-0 flex-1"),
				// Breadcrumbs
				g.If(len(breadcrumbs) > 0, renderBreadcrumbs(breadcrumbs)),
				html.H2(
					html.Class("text-2xl font-bold leading-7 text-gray-900 dark:text-white sm:truncate sm:text-3xl sm:tracking-tight"),
					g.Text(title),
				),
			),
			g.If(actions != nil, html.Div(
				html.Class("mt-4 flex md:ml-4 md:mt-0 space-x-3"),
				actions,
			)),
		),
		// Content
		g.Group(content),
	)
}

// Breadcrumb represents a breadcrumb item
type Breadcrumb struct {
	Label string
	Href  string
}

func renderBreadcrumbs(items []Breadcrumb) g.Node {
	nodes := make([]g.Node, 0)
	for i, item := range items {
		if i > 0 {
			nodes = append(nodes, html.Span(
				html.Class("mx-2 text-gray-400"),
				g.Text("/"),
			))
		}
		if item.Href != "" && i < len(items)-1 {
			nodes = append(nodes, html.A(
				html.Href(item.Href),
				html.Class("text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"),
				g.Text(item.Label),
			))
		} else {
			nodes = append(nodes, html.Span(
				html.Class("text-sm text-gray-900 dark:text-white"),
				g.Text(item.Label),
			))
		}
	}

	return html.Nav(
		html.Class("flex mb-2"),
		g.Attr("aria-label", "Breadcrumb"),
		g.Group(nodes),
	)
}

// BillingOverviewPage renders the billing overview dashboard
func BillingOverviewPage(basePath string, metrics *core.DashboardMetrics, recentSubs []*core.Subscription) g.Node {
	return PageLayout(
		"Billing Overview",
		nil,
		nil,
		// Stats Grid
		html.Div(
			html.Class("grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4"),
			StatCard("Monthly Recurring Revenue", formatMoney(metrics.TotalMRR, metrics.Currency),
				formatGrowth(metrics.MRRGrowth), metrics.MRRGrowth >= 0,
				html.Span(html.Class("text-indigo-600"), g.Text("$")),
			),
			StatCard("Annual Recurring Revenue", formatMoney(metrics.TotalARR, metrics.Currency),
				"", true,
				html.Span(html.Class("text-indigo-600"), g.Text("📊")),
			),
			StatCard("Active Subscriptions", fmt.Sprintf("%d", metrics.ActiveSubscriptions),
				formatGrowth(metrics.SubscriptionGrowth), metrics.SubscriptionGrowth >= 0,
				html.Span(html.Class("text-indigo-600"), g.Text("✓")),
			),
			StatCard("Trial Subscriptions", fmt.Sprintf("%d", metrics.TrialingSubscriptions),
				"", true,
				html.Span(html.Class("text-indigo-600"), g.Text("⏱")),
			),
		),
		// MRR Breakdown
		html.Div(
			html.Class("grid grid-cols-1 gap-5 lg:grid-cols-2 mt-6"),
			Card("MRR Breakdown",
				html.Div(
					html.Class("space-y-4"),
					mrrBreakdownRow("New MRR", metrics.NewMRR, metrics.Currency, "text-green-600"),
					mrrBreakdownRow("Expansion MRR", metrics.ExpansionMRR, metrics.Currency, "text-green-600"),
					mrrBreakdownRow("Churned MRR", -metrics.ChurnedMRR, metrics.Currency, "text-red-600"),
				),
			),
			Card("Health Metrics",
				html.Div(
					html.Class("space-y-4"),
					healthMetricRow("Churn Rate", fmt.Sprintf("%.1f%%", metrics.ChurnRate), metrics.ChurnRate < 5),
					healthMetricRow("Trial Conversion", fmt.Sprintf("%.1f%%", metrics.TrialConversionRate), metrics.TrialConversionRate > 20),
					healthMetricRow("Net Revenue Retention", fmt.Sprintf("%.1f%%", metrics.NetRevenueRetention), metrics.NetRevenueRetention >= 100),
				),
			),
		),
		// Recent Subscriptions
		CardWithActions("Recent Subscriptions",
			LinkButton("View All", basePath+"/billing/subscriptions", "secondary"),
			g.If(len(recentSubs) == 0,
				EmptyState("No subscriptions yet", "Create your first plan to get started", LinkButton("Create Plan", basePath+"/billing/plans/create", "primary")),
			),
			g.If(len(recentSubs) > 0, renderSubscriptionTable(recentSubs, basePath)),
		),
	)
}

func mrrBreakdownRow(label string, amount int64, currency, colorClass string) g.Node {
	return html.Div(
		html.Class("flex justify-between items-center"),
		html.Span(html.Class("text-sm text-gray-600 dark:text-gray-400"), g.Text(label)),
		html.Span(html.Class("text-sm font-medium "+colorClass), g.Text(formatMoney(amount, currency))),
	)
}

func healthMetricRow(label, value string, healthy bool) g.Node {
	statusClass := "text-green-600"
	if !healthy {
		statusClass = "text-yellow-600"
	}
	return html.Div(
		html.Class("flex justify-between items-center"),
		html.Span(html.Class("text-sm text-gray-600 dark:text-gray-400"), g.Text(label)),
		html.Span(html.Class("text-sm font-medium "+statusClass), g.Text(value)),
	)
}

// PlansListPage renders the plans list page
func PlansListPage(basePath string, plans []*core.Plan, total, page, pageSize int) g.Node {
	totalPages := (total + pageSize - 1) / pageSize

	return PageLayout(
		"Subscription Plans",
		[]Breadcrumb{{Label: "Billing", Href: basePath + "/billing"}, {Label: "Plans"}},
		LinkButton("Create Plan", basePath+"/billing/plans/create", "primary"),
		g.If(len(plans) == 0,
			Card("", EmptyState("No plans created", "Create your first subscription plan to start accepting payments", nil)),
		),
		g.If(len(plans) > 0, Card("",
			Table(
				[]string{"Plan", "Slug", "Price", "Billing", "Status", "Subscriptions", "Actions"},
				renderPlanRows(plans, basePath)...,
			),
			Pagination(page, totalPages, basePath+"/billing/plans"),
		)),
	)
}

func renderPlanRows(plans []*core.Plan, basePath string) []g.Node {
	rows := make([]g.Node, len(plans))
	for i, plan := range plans {
		statusBadge := Badge("Inactive", "gray")
		if plan.IsActive {
			statusBadge = Badge("Active", "success")
		}

		rows[i] = TableRow(
			TableCell(html.Div(
				html.Span(html.Class("font-medium"), g.Text(plan.Name)),
				g.If(plan.Description != "", html.P(
					html.Class("text-xs text-gray-500 dark:text-gray-400 truncate max-w-xs"),
					g.Text(plan.Description),
				)),
			)),
			TableCellMuted(g.Text(plan.Slug)),
			TableCell(g.Text(formatMoney(plan.BasePrice, plan.Currency))),
			TableCellMuted(g.Text(string(plan.BillingInterval))),
			TableCell(statusBadge),
			TableCellMuted(g.Text("0")), // Would show actual count
			TableCell(html.Div(
				html.Class("flex space-x-2"),
				html.A(html.Href(fmt.Sprintf("%s/billing/plans/%s", basePath, plan.ID.String())), html.Class("text-indigo-600 hover:text-indigo-900 text-sm"), g.Text("View")),
				html.A(html.Href(fmt.Sprintf("%s/billing/plans/%s/edit", basePath, plan.ID.String())), html.Class("text-gray-600 hover:text-gray-900 text-sm"), g.Text("Edit")),
			)),
		)
	}
	return rows
}

// SubscriptionsListPage renders the subscriptions list page
func SubscriptionsListPage(basePath string, subs []*core.Subscription, total, page, pageSize int) g.Node {
	totalPages := (total + pageSize - 1) / pageSize

	return PageLayout(
		"Subscriptions",
		[]Breadcrumb{{Label: "Billing", Href: basePath + "/billing"}, {Label: "Subscriptions"}},
		nil,
		g.If(len(subs) == 0,
			Card("", EmptyState("No subscriptions yet", "Subscriptions will appear here when organizations subscribe to your plans", nil)),
		),
		g.If(len(subs) > 0, Card("",
			renderSubscriptionTable(subs, basePath),
			Pagination(page, totalPages, basePath+"/billing/subscriptions"),
		)),
	)
}

func renderSubscriptionTable(subs []*core.Subscription, basePath string) g.Node {
	rows := make([]g.Node, len(subs))
	for i, sub := range subs {
		planName := "Unknown Plan"
		if sub.Plan != nil {
			planName = sub.Plan.Name
		}

		rows[i] = TableRow(
			TableCell(g.Text(sub.OrganizationID.String()[:8]+"...")),
			TableCell(g.Text(planName)),
			TableCell(StatusBadge(string(sub.Status))),
			TableCellMuted(g.Text(sub.CurrentPeriodEnd.Format("Jan 2, 2006"))),
			TableCell(html.A(
				html.Href(fmt.Sprintf("%s/billing/subscriptions/%s", basePath, sub.ID.String())),
				html.Class("text-indigo-600 hover:text-indigo-900 text-sm"),
				g.Text("View"),
			)),
		)
	}

	return Table(
		[]string{"Organization", "Plan", "Status", "Renews", "Actions"},
		rows...,
	)
}

// CouponsListPage renders the coupons list page
func CouponsListPage(basePath string, coupons []*core.Coupon, total, page, pageSize int) g.Node {
	totalPages := (total + pageSize - 1) / pageSize

	return PageLayout(
		"Coupons & Discounts",
		[]Breadcrumb{{Label: "Billing", Href: basePath + "/billing"}, {Label: "Coupons"}},
		LinkButton("Create Coupon", basePath+"/billing/coupons/create", "primary"),
		g.If(len(coupons) == 0,
			Card("", EmptyState("No coupons created", "Create discount codes to offer promotions to your customers", nil)),
		),
		g.If(len(coupons) > 0, Card("",
			Table(
				[]string{"Code", "Name", "Discount", "Duration", "Redemptions", "Status", "Actions"},
				renderCouponRows(coupons, basePath)...,
			),
			Pagination(page, totalPages, basePath+"/billing/coupons"),
		)),
	)
}

func renderCouponRows(coupons []*core.Coupon, basePath string) []g.Node {
	rows := make([]g.Node, len(coupons))
	for i, coupon := range coupons {
		discount := ""
		switch coupon.Type {
		case core.CouponTypePercentage:
			discount = fmt.Sprintf("%.0f%% off", coupon.PercentOff)
		case core.CouponTypeFixedAmount:
			discount = formatMoney(coupon.AmountOff, coupon.Currency) + " off"
		case core.CouponTypeTrialExtension:
			discount = fmt.Sprintf("+%d trial days", coupon.TrialDays)
		case core.CouponTypeFreeMonths:
			discount = fmt.Sprintf("%d free months", coupon.FreeMonths)
		}

		redemptions := fmt.Sprintf("%d", coupon.TimesRedeemed)
		if coupon.MaxRedemptions > 0 {
			redemptions = fmt.Sprintf("%d / %d", coupon.TimesRedeemed, coupon.MaxRedemptions)
		}

		statusBadge := Badge("Expired", "gray")
		if coupon.IsValid() {
			statusBadge = Badge("Active", "success")
		}

		rows[i] = TableRow(
			TableCell(html.Code(html.Class("text-sm bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded"), g.Text(coupon.Code))),
			TableCell(g.Text(coupon.Name)),
			TableCell(g.Text(discount)),
			TableCellMuted(g.Text(string(coupon.Duration))),
			TableCellMuted(g.Text(redemptions)),
			TableCell(statusBadge),
			TableCell(html.A(
				html.Href(fmt.Sprintf("%s/billing/coupons/%s", basePath, coupon.ID.String())),
				html.Class("text-indigo-600 hover:text-indigo-900 text-sm"),
				g.Text("View"),
			)),
		)
	}
	return rows
}

// InvoicesListPage renders the invoices list page
func InvoicesListPage(basePath string, invoices []*core.Invoice, total, page, pageSize int) g.Node {
	totalPages := (total + pageSize - 1) / pageSize

	return PageLayout(
		"Invoices",
		[]Breadcrumb{{Label: "Billing", Href: basePath + "/billing"}, {Label: "Invoices"}},
		nil,
		g.If(len(invoices) == 0,
			Card("", EmptyState("No invoices yet", "Invoices will be created automatically when subscriptions are billed", nil)),
		),
		g.If(len(invoices) > 0, Card("",
			Table(
				[]string{"Invoice", "Organization", "Amount", "Status", "Due Date", "Actions"},
				renderInvoiceRows(invoices, basePath)...,
			),
			Pagination(page, totalPages, basePath+"/billing/invoices"),
		)),
	)
}

func renderInvoiceRows(invoices []*core.Invoice, basePath string) []g.Node {
	rows := make([]g.Node, len(invoices))
	for i, inv := range invoices {
		statusBadge := Badge(string(inv.Status), "gray")
		switch inv.Status {
		case core.InvoiceStatusPaid:
			statusBadge = Badge("Paid", "success")
		case core.InvoiceStatusOpen:
			statusBadge = Badge("Open", "info")
		case core.InvoiceStatusVoid:
			statusBadge = Badge("Void", "gray")
		case core.InvoiceStatusUncollectible:
			statusBadge = Badge("Uncollectible", "danger")
		}

		dueDate := inv.DueDate.Format("Jan 2, 2006")

		rows[i] = TableRow(
			TableCell(g.Text(inv.Number)),
			TableCellMuted(g.Text(inv.OrganizationID.String()[:8]+"...")),
			TableCell(g.Text(formatMoney(inv.Total, inv.Currency))),
			TableCell(statusBadge),
			TableCellMuted(g.Text(dueDate)),
			TableCell(html.Div(
				html.Class("flex space-x-2"),
				html.A(html.Href(fmt.Sprintf("%s/billing/invoices/%s", basePath, inv.ID.String())), html.Class("text-indigo-600 hover:text-indigo-900 text-sm"), g.Text("View")),
				g.If(inv.ProviderPDFURL != "", html.A(html.Href(inv.ProviderPDFURL), html.Target("_blank"), html.Class("text-gray-600 hover:text-gray-900 text-sm"), g.Text("PDF"))),
			)),
		)
	}
	return rows
}

// UsageDashboardPage renders the usage dashboard
func UsageDashboardPage(basePath string, usage map[string]*core.UsageLimit) g.Node {
	return PageLayout(
		"Usage Dashboard",
		[]Breadcrumb{{Label: "Billing", Href: basePath + "/billing"}, {Label: "Usage"}},
		nil,
		Card("Current Usage",
			g.If(len(usage) == 0,
				EmptyState("No usage data", "Usage metrics will appear here once configured", nil),
			),
			g.If(len(usage) > 0,
				html.Div(
					append([]g.Node{html.Class("space-y-6")}, renderUsageMetrics(usage)...)...,
				),
			),
		),
	)
}

func renderUsageMetrics(usage map[string]*core.UsageLimit) []g.Node {
	nodes := make([]g.Node, 0)
	for _, limit := range usage {
		percent := limit.PercentUsed
		if percent > 100 {
			percent = 100
		}

		colorClass := "bg-green-500"
		if percent > 80 {
			colorClass = "bg-yellow-500"
		}
		if percent > 95 {
			colorClass = "bg-red-500"
		}

		limitText := "Unlimited"
		if limit.Limit != -1 { // -1 means unlimited
			limitText = fmt.Sprintf("%d / %d", limit.CurrentUsage, limit.Limit)
		}

		nodes = append(nodes, html.Div(
			html.Div(
				html.Class("flex justify-between mb-1"),
				html.Span(html.Class("text-sm font-medium text-gray-700 dark:text-gray-300"), g.Text(limit.MetricKey)),
				html.Span(html.Class("text-sm text-gray-500 dark:text-gray-400"), g.Text(limitText)),
			),
			html.Div(
				html.Class("w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5"),
				html.Div(
					html.Class(colorClass+" h-2.5 rounded-full"),
					html.Style(fmt.Sprintf("width: %.0f%%", percent)),
				),
			),
		))
	}
	return nodes
}

// Helper functions

func formatMoney(amount int64, currency string) string {
	symbol := currencySymbol(currency)
	return fmt.Sprintf("%s%.2f", symbol, float64(amount)/100)
}

func formatGrowth(value float64) string {
	if value == 0 {
		return ""
	}
	sign := "+"
	if value < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f%%", sign, value)
}

func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}
