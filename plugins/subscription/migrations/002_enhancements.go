package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

func init() {
	// Register migration for enhancement tables
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Currency tables
		_, err := db.NewCreateTable().
			Model((*schema.SubscriptionCurrency)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionExchangeRate)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Tax tables
		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionTaxRate)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionTaxExemption)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionCustomerTaxID)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Coupon tables
		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionCoupon)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionCouponRedemption)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionPromotionCode)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Alert tables
		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionAlertConfig)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionAlert)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionAlertTemplate)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Analytics tables
		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionBillingMetric)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionMovement)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*schema.SubscriptionCohort)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for common queries
		// Exchange rates
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionExchangeRate)(nil)).
			Index("idx_exchange_rate_currencies").
			Column("from_currency", "to_currency", "valid_from").
			IfNotExists().
			Exec(ctx)

		// Tax rates
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionTaxRate)(nil)).
			Index("idx_tax_rate_location").
			Column("app_id", "country", "state").
			IfNotExists().
			Exec(ctx)

		// Coupons
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionCoupon)(nil)).
			Index("idx_coupon_code").
			Column("app_id", "code").
			Unique().
			IfNotExists().
			Exec(ctx)

		// Alerts
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionAlert)(nil)).
			Index("idx_alert_org_status").
			Column("organization_id", "status").
			IfNotExists().
			Exec(ctx)

		// Billing metrics
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionBillingMetric)(nil)).
			Index("idx_billing_metric_lookup").
			Column("app_id", "type", "period", "date").
			Unique().
			IfNotExists().
			Exec(ctx)

		// Movements
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionMovement)(nil)).
			Index("idx_movement_date").
			Column("app_id", "occurred_at").
			IfNotExists().
			Exec(ctx)

		// Cohorts
		_, _ = db.NewCreateIndex().
			Model((*schema.SubscriptionCohort)(nil)).
			Index("idx_cohort_month").
			Column("app_id", "cohort_month", "month_number").
			Unique().
			IfNotExists().
			Exec(ctx)

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop tables in reverse order
		tables := []interface{}{
			(*schema.SubscriptionCohort)(nil),
			(*schema.SubscriptionMovement)(nil),
			(*schema.SubscriptionBillingMetric)(nil),
			(*schema.SubscriptionAlertTemplate)(nil),
			(*schema.SubscriptionAlert)(nil),
			(*schema.SubscriptionAlertConfig)(nil),
			(*schema.SubscriptionPromotionCode)(nil),
			(*schema.SubscriptionCouponRedemption)(nil),
			(*schema.SubscriptionCoupon)(nil),
			(*schema.SubscriptionCustomerTaxID)(nil),
			(*schema.SubscriptionTaxExemption)(nil),
			(*schema.SubscriptionTaxRate)(nil),
			(*schema.SubscriptionExchangeRate)(nil),
			(*schema.SubscriptionCurrency)(nil),
		}

		for _, table := range tables {
			_, _ = db.NewDropTable().Model(table).IfExists().Exec(ctx)
		}

		return nil
	})
}

// Migrations is the collection of all subscription plugin migrations
var Migrations = migrate.NewMigrations()

