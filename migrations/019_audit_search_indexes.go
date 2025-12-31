package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Check if audit_events table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			// Table doesn't exist, skip migration
			return nil
		}

		// ========== Full-Text Search Index ==========
		// Create GIN index for full-text search on action, resource, and metadata
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_fts 
			ON audit_events 
			USING GIN (to_tsvector('english', 
				action || ' ' || resource || ' ' || COALESCE(metadata, '') || ' ' || COALESCE(user_agent, '')
			))
		`)
		if err != nil {
			return fmt.Errorf("failed to create full-text search index: %w", err)
		}

		// ========== Pattern Search Indexes ==========
		// Index for action pattern searches (ILIKE)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_action_pattern 
			ON audit_events (action text_pattern_ops)
		`)
		if err != nil {
			return fmt.Errorf("failed to create action pattern index: %w", err)
		}

		// Index for resource pattern searches (ILIKE)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_resource_pattern 
			ON audit_events (resource text_pattern_ops)
		`)
		if err != nil {
			return fmt.Errorf("failed to create resource pattern index: %w", err)
		}

		// ========== Composite Indexes for Common Query Patterns ==========
		// Index for environment + action queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_env_action 
			ON audit_events (environment_id, action) 
			WHERE environment_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create environment+action index: %w", err)
		}

		// Index for environment + user queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_env_user 
			ON audit_events (environment_id, user_id) 
			WHERE environment_id IS NOT NULL AND user_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create environment+user index: %w", err)
		}

		// Index for environment + created_at (for time-range queries)
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_env_created 
			ON audit_events (environment_id, created_at DESC) 
			WHERE environment_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create environment+created_at index: %w", err)
		}

		// ========== IP Address Indexing ==========
		// Regular index for exact IP match
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_ip_address 
			ON audit_events (ip_address) 
			WHERE ip_address IS NOT NULL AND ip_address != ''
		`)
		if err != nil {
			return fmt.Errorf("failed to create IP address index: %w", err)
		}

		// GiST index for IP range queries (CIDR matching)
		// Note: This requires ip_address to be cast to inet type in queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_ip_range 
			ON audit_events USING GIST (inet(ip_address)) 
			WHERE ip_address IS NOT NULL AND ip_address != ''
		`)
		if err != nil {
			// Log warning but don't fail - this might not work if IP addresses aren't valid
			fmt.Printf("Warning: failed to create IP range index (this is expected if IP addresses aren't valid inet format): %v\n", err)
		}

		// ========== User Action Analysis Index ==========
		// Index for analyzing user actions across time
		_, err = db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_audit_user_action_time 
			ON audit_events (user_id, action, created_at DESC) 
			WHERE user_id IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to create user+action+time index: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback - drop all indexes

		// Check if audit_events table exists
		var tableExists bool
		err := db.NewSelect().
			ColumnExpr("to_regclass(?) IS NOT NULL", "public.audit_events").
			Scan(ctx, &tableExists)

		if err != nil || !tableExists {
			return nil
		}

		// Drop indexes (safe to ignore errors if they don't exist)
		indexes := []string{
			"idx_audit_fts",
			"idx_audit_action_pattern",
			"idx_audit_resource_pattern",
			"idx_audit_env_action",
			"idx_audit_env_user",
			"idx_audit_env_created",
			"idx_audit_ip_address",
			"idx_audit_ip_range",
			"idx_audit_user_action_time",
		}

		for _, index := range indexes {
			_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s", index))
		}

		return nil
	})
}

