package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/migrations"
)

func init() {
	migrations.Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add new WebAuthn fields to passkeys table
		migrations := []string{
			// Add public key field (required for WebAuthn verification)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS public_key BYTEA`,

			// Add AAGUID field (authenticator identifier)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS aaguid BYTEA`,

			// Add sign count for replay attack detection
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS sign_count INTEGER DEFAULT 0`,

			// Add authenticator type (platform vs cross-platform)
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS authenticator_type VARCHAR(50)`,

			// Add name field for user-friendly device naming
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS name VARCHAR(255)`,

			// Add resident key flag
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS is_resident_key BOOLEAN DEFAULT FALSE`,

			// Add last used timestamp
			`ALTER TABLE passkeys ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP`,
		}

		for _, migration := range migrations {
			if _, err := db.ExecContext(ctx, migration); err != nil {
				return fmt.Errorf("failed to execute migration: %w", err)
			}
		}

		// For SQLite, use different syntax
		// Check if this is SQLite and execute alternate migrations if needed
		// This is simplified; in production you'd check the driver type

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - remove the added columns
		rollbacks := []string{
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS last_used_at`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS is_resident_key`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS name`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS authenticator_type`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS sign_count`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS aaguid`,
			`ALTER TABLE passkeys DROP COLUMN IF EXISTS public_key`,
		}

		for _, rollback := range rollbacks {
			if _, err := db.ExecContext(ctx, rollback); err != nil {
				// Log error but continue with other rollbacks
				fmt.Printf("rollback error: %v\n", err)
			}
		}

		return nil
	})
}
