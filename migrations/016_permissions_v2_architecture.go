package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("[Migration] 016_permissions_v2_architecture.go: Starting migration...")

		// Phase 1: Add new columns to permission_policies table
		fmt.Println("[Migration] Adding 'app_id' and 'user_organization_id' to 'permission_policies'...")
		_, err := db.Exec(ctx, `
			ALTER TABLE permission_policies
			ADD COLUMN app_id VARCHAR(20) NULL,
			ADD COLUMN user_organization_id VARCHAR(20) NULL;
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id and user_organization_id to permission_policies: %w", err)
		}
		fmt.Println("[Migration] 'app_id' and 'user_organization_id' added to 'permission_policies'.")

		// Phase 2: Migrate existing org_id data to app_id
		fmt.Println("[Migration] Migrating existing 'org_id' to 'app_id' in 'permission_policies'...")
		_, err = db.Exec(ctx, `
			UPDATE permission_policies
			SET app_id = org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to migrate org_id to app_id in permission_policies: %w", err)
		}
		fmt.Println("[Migration] Existing 'org_id' migrated to 'app_id'.")

		// Phase 3: Make app_id NOT NULL and drop old org_id
		fmt.Println("[Migration] Making 'app_id' NOT NULL and dropping 'org_id' from 'permission_policies'...")
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_policies
			ALTER COLUMN app_id SET NOT NULL;
		`)
		if err != nil {
			return fmt.Errorf("failed to set app_id as NOT NULL in permission_policies: %w", err)
		}
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_policies
			DROP COLUMN org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to drop org_id from permission_policies: %w", err)
		}
		fmt.Println("[Migration] 'app_id' is NOT NULL, 'org_id' dropped from 'permission_policies'.")

		// Phase 4: Add new columns to permission_namespaces table
		fmt.Println("[Migration] Adding 'app_id' and 'user_organization_id' to 'permission_namespaces'...")
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_namespaces
			ADD COLUMN app_id VARCHAR(20) NULL,
			ADD COLUMN user_organization_id VARCHAR(20) NULL;
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id and user_organization_id to permission_namespaces: %w", err)
		}
		fmt.Println("[Migration] 'app_id' and 'user_organization_id' added to 'permission_namespaces'.")

		// Phase 5: Migrate existing org_id data to app_id in namespaces
		fmt.Println("[Migration] Migrating existing 'org_id' to 'app_id' in 'permission_namespaces'...")
		_, err = db.Exec(ctx, `
			UPDATE permission_namespaces
			SET app_id = org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to migrate org_id to app_id in permission_namespaces: %w", err)
		}
		fmt.Println("[Migration] Existing 'org_id' migrated to 'app_id' in 'permission_namespaces'.")

		// Phase 6: Make app_id NOT NULL and drop old org_id from namespaces
		fmt.Println("[Migration] Making 'app_id' NOT NULL and dropping 'org_id' from 'permission_namespaces'...")
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_namespaces
			ALTER COLUMN app_id SET NOT NULL,
			DROP COLUMN org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to update permission_namespaces: %w", err)
		}
		fmt.Println("[Migration] 'app_id' is NOT NULL, 'org_id' dropped from 'permission_namespaces'.")

		// Phase 7: Add new columns to permission_audit_events table
		fmt.Println("[Migration] Adding 'app_id' and 'user_organization_id' to 'permission_audit_events'...")
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_audit_events
			ADD COLUMN app_id VARCHAR(20) NULL,
			ADD COLUMN user_organization_id VARCHAR(20) NULL;
		`)
		if err != nil {
			return fmt.Errorf("failed to add app_id and user_organization_id to permission_audit_events: %w", err)
		}
		fmt.Println("[Migration] 'app_id' and 'user_organization_id' added to 'permission_audit_events'.")

		// Phase 8: Migrate existing org_id data to app_id in audit events
		fmt.Println("[Migration] Migrating existing 'org_id' to 'app_id' in 'permission_audit_events'...")
		_, err = db.Exec(ctx, `
			UPDATE permission_audit_events
			SET app_id = org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to migrate org_id to app_id in permission_audit_events: %w", err)
		}
		fmt.Println("[Migration] Existing 'org_id' migrated to 'app_id' in 'permission_audit_events'.")

		// Phase 9: Make app_id NOT NULL and drop old org_id from audit events
		fmt.Println("[Migration] Making 'app_id' NOT NULL and dropping 'org_id' from 'permission_audit_events'...")
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_audit_events
			ALTER COLUMN app_id SET NOT NULL,
			DROP COLUMN org_id;
		`)
		if err != nil {
			return fmt.Errorf("failed to update permission_audit_events: %w", err)
		}
		fmt.Println("[Migration] 'app_id' is NOT NULL, 'org_id' dropped from 'permission_audit_events'.")

		// Phase 10: Add foreign key constraints and indexes
		fmt.Println("[Migration] Adding foreign key constraints and indexes...")

		// Policies table
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_policies
			ADD CONSTRAINT fk_permission_policies_app_id FOREIGN KEY (app_id) REFERENCES apps (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_policies_app_id: %w", err)
		}
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_policies
			ADD CONSTRAINT fk_permission_policies_user_organization_id FOREIGN KEY (user_organization_id) REFERENCES user_organizations (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_policies_user_organization_id: %w", err)
		}

		// Namespaces table
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_namespaces
			ADD CONSTRAINT fk_permission_namespaces_app_id FOREIGN KEY (app_id) REFERENCES apps (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_namespaces_app_id: %w", err)
		}
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_namespaces
			ADD CONSTRAINT fk_permission_namespaces_user_organization_id FOREIGN KEY (user_organization_id) REFERENCES user_organizations (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_namespaces_user_organization_id: %w", err)
		}

		// Audit events table
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_audit_events
			ADD CONSTRAINT fk_permission_audit_events_app_id FOREIGN KEY (app_id) REFERENCES apps (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_audit_events_app_id: %w", err)
		}
		_, err = db.Exec(ctx, `
			ALTER TABLE permission_audit_events
			ADD CONSTRAINT fk_permission_audit_events_user_organization_id FOREIGN KEY (user_organization_id) REFERENCES user_organizations (id) ON DELETE CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to add fk_permission_audit_events_user_organization_id: %w", err)
		}

		// Add indexes for new columns
		_, err = db.Exec(ctx, `
			CREATE INDEX idx_permission_policies_app_id ON permission_policies (app_id);
			CREATE INDEX idx_permission_policies_user_organization_id ON permission_policies (user_organization_id);
			CREATE INDEX idx_permission_policies_app_org ON permission_policies (app_id, user_organization_id);
		`)
		if err != nil {
			return fmt.Errorf("failed to create indexes on permission_policies: %w", err)
		}

		_, err = db.Exec(ctx, `
			CREATE INDEX idx_permission_namespaces_app_id ON permission_namespaces (app_id);
			CREATE INDEX idx_permission_namespaces_user_organization_id ON permission_namespaces (user_organization_id);
		`)
		if err != nil {
			return fmt.Errorf("failed to create indexes on permission_namespaces: %w", err)
		}

		_, err = db.Exec(ctx, `
			CREATE INDEX idx_permission_audit_events_app_id ON permission_audit_events (app_id);
			CREATE INDEX idx_permission_audit_events_user_organization_id ON permission_audit_events (user_organization_id);
		`)
		if err != nil {
			return fmt.Errorf("failed to create indexes on permission_audit_events: %w", err)
		}

		fmt.Println("[Migration] Foreign key constraints and indexes added.")
		fmt.Println("[Migration] 016_permissions_v2_architecture.go: Migration complete!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("[Rollback] 016_permissions_v2_architecture.go: Starting rollback...")

		// Drop foreign key constraints and indexes
		fmt.Println("[Rollback] Dropping foreign key constraints and indexes...")
		db.Exec(ctx, `ALTER TABLE permission_policies DROP CONSTRAINT IF EXISTS fk_permission_policies_app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_policies DROP CONSTRAINT IF EXISTS fk_permission_policies_user_organization_id;`)
		db.Exec(ctx, `ALTER TABLE permission_namespaces DROP CONSTRAINT IF EXISTS fk_permission_namespaces_app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_namespaces DROP CONSTRAINT IF EXISTS fk_permission_namespaces_user_organization_id;`)
		db.Exec(ctx, `ALTER TABLE permission_audit_events DROP CONSTRAINT IF EXISTS fk_permission_audit_events_app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_audit_events DROP CONSTRAINT IF EXISTS fk_permission_audit_events_user_organization_id;`)

		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_policies_app_id;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_policies_user_organization_id;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_policies_app_org;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_namespaces_app_id;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_namespaces_user_organization_id;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_audit_events_app_id;`)
		db.Exec(ctx, `DROP INDEX IF EXISTS idx_permission_audit_events_user_organization_id;`)
		fmt.Println("[Rollback] Foreign key constraints and indexes dropped.")

		// Revert permission_policies table
		fmt.Println("[Rollback] Reverting 'permission_policies' table...")
		db.Exec(ctx, `ALTER TABLE permission_policies ADD COLUMN org_id VARCHAR(20) NULL;`)
		db.Exec(ctx, `UPDATE permission_policies SET org_id = app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_policies DROP COLUMN app_id, DROP COLUMN user_organization_id;`)
		db.Exec(ctx, `ALTER TABLE permission_policies ALTER COLUMN org_id SET NOT NULL;`)
		fmt.Println("[Rollback] 'permission_policies' table reverted.")

		// Revert permission_namespaces table
		fmt.Println("[Rollback] Reverting 'permission_namespaces' table...")
		db.Exec(ctx, `ALTER TABLE permission_namespaces ADD COLUMN org_id VARCHAR(20) NULL;`)
		db.Exec(ctx, `UPDATE permission_namespaces SET org_id = app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_namespaces DROP COLUMN app_id, DROP COLUMN user_organization_id;`)
		db.Exec(ctx, `ALTER TABLE permission_namespaces ALTER COLUMN org_id SET NOT NULL;`)
		fmt.Println("[Rollback] 'permission_namespaces' table reverted.")

		// Revert permission_audit_events table
		fmt.Println("[Rollback] Reverting 'permission_audit_events' table...")
		db.Exec(ctx, `ALTER TABLE permission_audit_events ADD COLUMN org_id VARCHAR(20) NULL;`)
		db.Exec(ctx, `UPDATE permission_audit_events SET org_id = app_id;`)
		db.Exec(ctx, `ALTER TABLE permission_audit_events DROP COLUMN app_id, DROP COLUMN user_organization_id;`)
		db.Exec(ctx, `ALTER TABLE permission_audit_events ALTER COLUMN org_id SET NOT NULL;`)
		fmt.Println("[Rollback] 'permission_audit_events' table reverted.")

		fmt.Println("[Rollback] 016_permissions_v2_architecture.go: Rollback complete!")
		return nil
	})
}
