package postgres

import (
	"context"

	"github.com/xraph/grove/migrate"
)

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "webhook_relay_endpoint",
		Version: "20260922000005",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			for _, stmt := range []string{
				`ALTER TABLE authsome_webhooks ADD COLUMN IF NOT EXISTS relay_endpoint_id TEXT NOT NULL DEFAULT ''`,
				`ALTER TABLE authsome_webhooks ADD COLUMN IF NOT EXISTS secret_hash TEXT NOT NULL DEFAULT ''`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			for _, stmt := range []string{
				`ALTER TABLE authsome_webhooks DROP COLUMN IF EXISTS secret_hash`,
				`ALTER TABLE authsome_webhooks DROP COLUMN IF EXISTS relay_endpoint_id`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
