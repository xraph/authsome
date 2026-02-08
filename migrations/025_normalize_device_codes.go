package migrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] normalizing existing device codes (remove hyphens)")

		// Get all device codes
		type DeviceCode struct {
			ID       string `bun:"id,pk"`
			UserCode string `bun:"user_code"`
		}

		var codes []DeviceCode
		err := db.NewSelect().
			Model((*DeviceCode)(nil)).
			Column("id", "user_code").
			Scan(ctx, &codes)
		if err != nil {
			return fmt.Errorf("failed to fetch device codes: %w", err)
		}

		// Normalize each code
		for _, code := range codes {
			normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(code.UserCode, " ", ""), "-", ""))
			if normalizedCode != code.UserCode {
				_, err := db.NewUpdate().
					Model((*DeviceCode)(nil)).
					Set("user_code = ?", normalizedCode).
					Where("id = ?", code.ID).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to normalize device code %s: %w", code.ID, err)
				}
			}
		}

		fmt.Println(" [OK]")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Down migration - no-op since we can't reverse normalization
		fmt.Println(" [down migration] skipping device code normalization reversal")
		return nil
	})
}
