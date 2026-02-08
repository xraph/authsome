package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] creating device_codes table")

		// Create device_codes table
		_, err := db.NewCreateTable().
			Model((*schema.DeviceCode)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create device_codes table: %w", err)
		}

		// Create indexes for performance
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_device_codes_device_code ON device_codes(device_code)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_user_code ON device_codes(user_code)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_client_id ON device_codes(client_id)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_status ON device_codes(status)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_expires_at ON device_codes(expires_at)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_user_id ON device_codes(user_id)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_session_id ON device_codes(session_id)",
			"CREATE INDEX IF NOT EXISTS idx_device_codes_app_env ON device_codes(app_id, environment_id)",
		}

		for _, idx := range indexes {
			if _, err := db.ExecContext(ctx, idx); err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}

		fmt.Println(" [OK]")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] dropping device_codes table")

		_, err := db.NewDropTable().
			Model((*schema.DeviceCode)(nil)).
			IfExists().
			Cascade().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to drop device_codes table: %w", err)
		}

		fmt.Println(" [OK]")
		return nil
	})
}
