package migrations

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Create organizations table
		_, err := db.NewCreateTable().
			Model((*schema.Organization)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create users table
		_, err = db.NewCreateTable().
			Model((*schema.User)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create members table
		_, err = db.NewCreateTable().
			Model((*schema.Member)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create sessions table
		_, err = db.NewCreateTable().
			Model((*schema.Session)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create accounts table
		_, err = db.NewCreateTable().
			Model((*schema.Account)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create verifications table
		_, err = db.NewCreateTable().
			Model((*schema.Verification)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create devices table
		_, err = db.NewCreateTable().
			Model((*schema.Device)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create audit events table
		_, err = db.NewCreateTable().
			Model((*schema.AuditEvent)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create user bans table
		_, err = db.NewCreateTable().
			Model((*schema.UserBan)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create roles table
		_, err = db.NewCreateTable().
			Model((*schema.Role)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create permissions table
		_, err = db.NewCreateTable().
			Model((*schema.Permission)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create user roles table
		_, err = db.NewCreateTable().
			Model((*schema.UserRole)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create policies table
		_, err = db.NewCreateTable().
			Model((*schema.Policy)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create API keys table
		_, err = db.NewCreateTable().
			Model((*schema.APIKey)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create webhooks table
		_, err = db.NewCreateTable().
			Model((*schema.Webhook)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create webhook events table
		_, err = db.NewCreateTable().
			Model((*schema.Event)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create webhook deliveries table
		_, err = db.NewCreateTable().
			Model((*schema.Delivery)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create form schemas table
		_, err = db.NewCreateTable().
			Model((*schema.FormSchema)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create teams table
		_, err = db.NewCreateTable().
			Model((*schema.Team)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create team members table
		_, err = db.NewCreateTable().
			Model((*schema.TeamMember)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes
		_, err = db.NewCreateIndex().
			Model((*schema.User)(nil)).
			Index("idx_users_email").
			Column("email").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_token").
			Column("token").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Member)(nil)).
			Index("idx_members_org_user").
			Column("organization_id", "user_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		// Create webhook indexes
		_, err = db.NewCreateIndex().
			Model((*schema.Event)(nil)).
			Index("idx_webhook_events_org").
			Column("organization_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Delivery)(nil)).
			Index("idx_webhook_deliveries_webhook_id").
			Column("webhook_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Delivery)(nil)).
			Index("idx_webhook_deliveries_event_id").
			Column("event_id").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateIndex().
			Model((*schema.Delivery)(nil)).
			Index("idx_webhook_deliveries_status").
			Column("status", "next_retry_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop all tables
		tables := []string{
			"team_members",
			"teams",
			"form_schemas",
			"webhook_deliveries",
			"webhook_events",
			"webhooks",
			"api_keys",
			"policies",
			"user_roles",
			"permissions",
			"roles",
			"user_bans",
			"audit_events",
			"devices",
			"verifications",
			"accounts",
			"sessions",
			"members",
			"users",
			"organizations",
		}

		for _, table := range tables {
			_, err := db.NewDropTable().
				Table(table).
				IfExists().
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
