package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create notification_templates table
		_, err := db.NewCreateTable().
			Model((*schema.NotificationTemplate)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create notifications table
		_, err = db.NewCreateTable().
			Model((*schema.Notification)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for notification_templates
		_, err = db.NewCreateIndex().
			Model((*schema.NotificationTemplate)(nil)).
			Index("idx_notification_templates_org_key_type_lang").
			Column("organization_id", "template_key", "type", "language").
			Unique().
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.NotificationTemplate)(nil)).
			Index("idx_notification_templates_org_active").
			Column("organization_id", "active").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for notifications
		_, err = db.NewCreateIndex().
			Model((*schema.Notification)(nil)).
			Index("idx_notifications_org_status").
			Column("organization_id", "status").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Notification)(nil)).
			Index("idx_notifications_org_recipient").
			Column("organization_id", "recipient").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Notification)(nil)).
			Index("idx_notifications_created_at").
			Column("created_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Drop indexes
		_, _ = db.NewDropIndex().
			Index("idx_notifications_created_at").
			IfExists().
			Exec(ctx)

		_, _ = db.NewDropIndex().
			Index("idx_notifications_org_recipient").
			IfExists().
			Exec(ctx)

		_, _ = db.NewDropIndex().
			Index("idx_notifications_org_status").
			IfExists().
			Exec(ctx)

		_, _ = db.NewDropIndex().
			Index("idx_notification_templates_org_active").
			IfExists().
			Exec(ctx)

		_, _ = db.NewDropIndex().
			Index("idx_notification_templates_org_key_type_lang").
			IfExists().
			Exec(ctx)

		// Drop tables
		_, err := db.NewDropTable().
			Model((*schema.Notification)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*schema.NotificationTemplate)(nil)).
			IfExists().
			Exec(ctx)
		return err
	})
}
