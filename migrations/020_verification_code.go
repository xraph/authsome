package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add 'code' column to verifications table for 6-digit mobile verification codes
		// This supports mobile-friendly password reset where users can enter a short code
		// instead of clicking a URL link

		// Check if verifications table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", "verifications").
			Scan(ctx, &tableExists)

		if err != nil {
			// Try SQLite approach
			err = db.NewSelect().
				ColumnExpr("EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", "verifications").
				Scan(ctx, &tableExists)
			if err != nil {
				// Table check failed, try migration anyway
				tableExists = true
			}
		}

		if !tableExists {
			// Table doesn't exist, skip migration
			return nil
		}

		// Check if code column already exists
		var columnExists bool
		err = db.NewSelect().
			ColumnExpr("EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = ? AND column_name = ?)", "verifications", "code").
			Scan(ctx, &columnExists)

		if err != nil {
			// Try SQLite approach using pragma
			var columns []struct {
				Name string `bun:"name"`
			}
			err = db.NewRaw("PRAGMA table_info(verifications)").Scan(ctx, &columns)
			if err == nil {
				for _, col := range columns {
					if col.Name == "code" {
						columnExists = true
						break
					}
				}
			}
		}

		if columnExists {
			// Column already exists, skip
			return nil
		}

		// Add the code column
		_, err = db.ExecContext(ctx, `ALTER TABLE verifications ADD COLUMN code VARCHAR(10)`)
		if err != nil {
			return fmt.Errorf("failed to add code column to verifications: %w", err)
		}

		// Add index for code lookups (partial index for non-null codes)
		// Using type + used filters for efficient password reset code lookups
		_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_verification_code ON verifications (code, type) WHERE code IS NOT NULL AND used = false`)
		if err != nil {
			// Try without partial index for SQLite
			_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_verification_code ON verifications (code, type)`)
			if err != nil {
				fmt.Printf("Warning: failed to create verification code index: %v\n", err)
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop the code column and index

		// Drop index first
		_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_verification_code`)

		// Drop column (SQLite doesn't support DROP COLUMN in older versions)
		// For SQLite, we would need to recreate the table, but for simplicity
		// we'll just try the standard ALTER TABLE
		_, err := db.ExecContext(ctx, `ALTER TABLE verifications DROP COLUMN IF EXISTS code`)
		if err != nil {
			// SQLite might not support this, log but don't fail
			fmt.Printf("Warning: could not drop code column (may need manual intervention for SQLite): %v\n", err)
		}

		return nil
	})
}


