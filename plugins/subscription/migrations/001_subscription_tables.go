package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// CreateSubscriptionTables creates all subscription-related tables.
func CreateSubscriptionTables(ctx context.Context, db *bun.DB) error {
	models := []any{
		(*schema.SubscriptionPlan)(nil),
		(*schema.SubscriptionPlanFeature)(nil),
		(*schema.SubscriptionPlanTier)(nil),
		(*schema.Subscription)(nil),
		(*schema.SubscriptionAddOn)(nil),
		(*schema.SubscriptionAddOnFeature)(nil),
		(*schema.SubscriptionAddOnTier)(nil),
		(*schema.SubscriptionAddOnItem)(nil),
		(*schema.SubscriptionInvoice)(nil),
		(*schema.SubscriptionInvoiceItem)(nil),
		(*schema.SubscriptionUsageRecord)(nil),
		(*schema.SubscriptionPaymentMethod)(nil),
		(*schema.SubscriptionCustomer)(nil),
		(*schema.SubscriptionEvent)(nil),
	}

	// Create tables in order (respecting foreign key dependencies)
	for _, model := range models {
		if _, err := db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table for %T: %w", model, err)
		}
	}

	// Create indexes for better query performance
	indexes := []struct {
		table string
		name  string
		cols  string
	}{
		// Plan indexes
		{"subscription_plans", "idx_subscription_plans_app_id", "app_id"},
		{"subscription_plans", "idx_subscription_plans_slug", "app_id, slug"},
		{"subscription_plans", "idx_subscription_plans_active", "is_active"},

		// Subscription indexes
		{"subscriptions", "idx_subscriptions_org_id", "organization_id"},
		{"subscriptions", "idx_subscriptions_plan_id", "plan_id"},
		{"subscriptions", "idx_subscriptions_status", "status"},
		{"subscriptions", "idx_subscriptions_provider_sub_id", "provider_sub_id"},

		// Add-on indexes
		{"subscription_addons", "idx_subscription_addons_app_id", "app_id"},
		{"subscription_addons", "idx_subscription_addons_slug", "app_id, slug"},

		// Invoice indexes
		{"subscription_invoices", "idx_subscription_invoices_org_id", "organization_id"},
		{"subscription_invoices", "idx_subscription_invoices_sub_id", "subscription_id"},
		{"subscription_invoices", "idx_subscription_invoices_status", "status"},
		{"subscription_invoices", "idx_subscription_invoices_number", "number"},

		// Usage indexes
		{"subscription_usage_records", "idx_subscription_usage_sub_id", "subscription_id"},
		{"subscription_usage_records", "idx_subscription_usage_org_id", "organization_id"},
		{"subscription_usage_records", "idx_subscription_usage_metric", "metric_key"},
		{"subscription_usage_records", "idx_subscription_usage_reported", "reported"},
		{"subscription_usage_records", "idx_subscription_usage_timestamp", "timestamp"},

		// Payment method indexes
		{"subscription_payment_methods", "idx_subscription_pm_org_id", "organization_id"},
		{"subscription_payment_methods", "idx_subscription_pm_provider_id", "provider_method_id"},

		// Customer indexes
		{"subscription_customers", "idx_subscription_customers_org_id", "organization_id"},
		{"subscription_customers", "idx_subscription_customers_provider_id", "provider_customer_id"},

		// Event indexes
		{"subscription_events", "idx_subscription_events_sub_id", "subscription_id"},
		{"subscription_events", "idx_subscription_events_org_id", "organization_id"},
		{"subscription_events", "idx_subscription_events_type", "event_type"},
	}

	for _, idx := range indexes {
		_, err := db.ExecContext(ctx, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			idx.name, idx.table, idx.cols,
		))
		if err != nil {
			// Log but don't fail - index creation might fail on some DBs
			fmt.Printf("Warning: failed to create index %s: %v\n", idx.name, err)
		}
	}

	return nil
}

// DropSubscriptionTables drops all subscription-related tables (for rollback).
func DropSubscriptionTables(ctx context.Context, db *bun.DB) error {
	// Drop in reverse order of creation (respecting foreign keys)
	models := []any{
		(*schema.SubscriptionEvent)(nil),
		(*schema.SubscriptionCustomer)(nil),
		(*schema.SubscriptionPaymentMethod)(nil),
		(*schema.SubscriptionUsageRecord)(nil),
		(*schema.SubscriptionInvoiceItem)(nil),
		(*schema.SubscriptionInvoice)(nil),
		(*schema.SubscriptionAddOnItem)(nil),
		(*schema.SubscriptionAddOnTier)(nil),
		(*schema.SubscriptionAddOnFeature)(nil),
		(*schema.SubscriptionAddOn)(nil),
		(*schema.Subscription)(nil),
		(*schema.SubscriptionPlanTier)(nil),
		(*schema.SubscriptionPlanFeature)(nil),
		(*schema.SubscriptionPlan)(nil),
	}

	for _, model := range models {
		if _, err := db.NewDropTable().
			Model(model).
			IfExists().
			Cascade().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to drop table for %T: %w", model, err)
		}
	}

	return nil
}
