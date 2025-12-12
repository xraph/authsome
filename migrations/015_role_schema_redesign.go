package migrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Step 1: Add environment_id column (nullable initially)
		if _, err := db.ExecContext(ctx, `ALTER TABLE roles ADD COLUMN environment_id VARCHAR(20)`); err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "duplicate column") {
				return fmt.Errorf("failed to add environment_id column: %w", err)
			}
		}

		// Step 2: Add display_name column (nullable initially)
		if _, err := db.ExecContext(ctx, `ALTER TABLE roles ADD COLUMN display_name TEXT`); err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "duplicate column") {
				return fmt.Errorf("failed to add display_name column: %w", err)
			}
		}

		// Step 3: Populate display_name from name with title-case transformation
		// This uses a simple transformation: replace underscores with spaces and title-case each word
		var roles []struct {
			ID   string `bun:"id"`
			Name string `bun:"name"`
		}
		if err := db.NewSelect().
			Table("roles").
			Column("id", "name").
			Scan(ctx, &roles); err != nil {
			return fmt.Errorf("failed to fetch roles for display_name population: %w", err)
		}

		for _, role := range roles {
			displayName := toTitleCase(role.Name)
			if _, err := db.ExecContext(ctx,
				`UPDATE roles SET display_name = ? WHERE id = ?`,
				displayName, role.ID); err != nil {
				return fmt.Errorf("failed to set display_name for role %s: %w", role.ID, err)
			}
		}

		// Step 4 & 5: Get default environment for each app and populate environment_id
		var apps []struct {
			ID string `bun:"id"`
		}
		if err := db.NewSelect().Table("apps").Column("id").Scan(ctx, &apps); err != nil {
			return fmt.Errorf("failed to fetch apps: %w", err)
		}

		for _, app := range apps {
			// Get the default environment for this app
			var envID string
			err := db.NewSelect().
				Table("environments").
				Column("id").
				Where("app_id = ?", app.ID).
				Where("is_default = ?", true).
				Limit(1).
				Scan(ctx, &envID)

			if err != nil {
				// If no default environment found, get the first environment
				err = db.NewSelect().
					Table("environments").
					Column("id").
					Where("app_id = ?", app.ID).
					Order("created_at ASC").
					Limit(1).
					Scan(ctx, &envID)

				if err != nil {
					// If no environment exists, check if this app has any roles
					roleCount, countErr := db.NewSelect().
						Table("roles").
						Where("app_id = ?", app.ID).
						Count(ctx)
					
					if countErr == nil && roleCount > 0 {
						// App has roles but no environment - create a default environment
						fmt.Printf("Creating default environment for app %s (has %d roles)\n", app.ID, roleCount)
						
						// Create default environment
						_, err = db.ExecContext(ctx, `
							INSERT INTO environments (id, app_id, name, slug, type, status, is_default, created_at, updated_at, created_by, updated_by, version)
							VALUES (?, ?, 'Default', 'default', 'production', 'active', true, NOW(), NOW(), ?, ?, 1)
						`, generateXID(), app.ID, app.ID, app.ID)
						
						if err != nil {
							return fmt.Errorf("failed to create default environment for app %s: %w", app.ID, err)
						}
						
						// Now get the newly created environment
						err = db.NewSelect().
							Table("environments").
							Column("id").
							Where("app_id = ?", app.ID).
							Limit(1).
							Scan(ctx, &envID)
							
						if err != nil {
							return fmt.Errorf("failed to retrieve created environment for app %s: %w", app.ID, err)
						}
					} else {
						// No roles for this app, skip
						continue
					}
				}
			}

			// Update all roles for this app to use the default environment
			if _, err := db.ExecContext(ctx,
				`UPDATE roles SET environment_id = ? WHERE app_id = ?`,
				envID, app.ID); err != nil {
				return fmt.Errorf("failed to set environment_id for app %s: %w", app.ID, err)
			}
		}

		// Step 6: Identify and delete duplicate roles
		// For app-level templates (organization_id IS NULL):
		// Keep most recently updated role per (app_id, environment_id, name, is_template)
		duplicatesQuery := `
			DELETE FROM roles
			WHERE id IN (
				SELECT r1.id
				FROM roles r1
				INNER JOIN roles r2 ON
					r1.app_id = r2.app_id AND
					r1.environment_id = r2.environment_id AND
					r1.name = r2.name AND
					r1.is_template = r2.is_template AND
					r1.organization_id IS NULL AND
					r2.organization_id IS NULL AND
					r1.updated_at < r2.updated_at
			)
		`
		result, err := db.ExecContext(ctx, duplicatesQuery)
		if err != nil {
			return fmt.Errorf("failed to delete duplicate app-level roles: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			fmt.Printf("Deleted %d duplicate app-level roles\n", rowsAffected)
		}

		// For org-scoped roles (organization_id IS NOT NULL):
		// Keep most recently updated role per (app_id, environment_id, organization_id, name, is_template)
		orgDuplicatesQuery := `
			DELETE FROM roles
			WHERE id IN (
				SELECT r1.id
				FROM roles r1
				INNER JOIN roles r2 ON
					r1.app_id = r2.app_id AND
					r1.environment_id = r2.environment_id AND
					r1.organization_id = r2.organization_id AND
					r1.name = r2.name AND
					r1.is_template = r2.is_template AND
					r1.organization_id IS NOT NULL AND
					r2.organization_id IS NOT NULL AND
					r1.updated_at < r2.updated_at
			)
		`
		result, err = db.ExecContext(ctx, orgDuplicatesQuery)
		if err != nil {
			return fmt.Errorf("failed to delete duplicate org-level roles: %w", err)
		}
		rowsAffected, _ = result.RowsAffected()
		if rowsAffected > 0 {
			fmt.Printf("Deleted %d duplicate org-level roles\n", rowsAffected)
		}

		// Step 7: Make environment_id and display_name NOT NULL
		// Note: Different databases have different syntax for this
		// PostgreSQL syntax:
		_, err = db.ExecContext(ctx, `ALTER TABLE roles ALTER COLUMN environment_id SET NOT NULL`)
		if err != nil {
			// SQLite doesn't support ALTER COLUMN, so we'll skip this for SQLite
			// The schema definition will enforce it for new tables
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "syntax") && !strings.Contains(errStr, "near") {
				return fmt.Errorf("failed to set environment_id as NOT NULL: %w", err)
			}
		}

		_, err = db.ExecContext(ctx, `ALTER TABLE roles ALTER COLUMN display_name SET NOT NULL`)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "syntax") && !strings.Contains(errStr, "near") {
				return fmt.Errorf("failed to set display_name as NOT NULL: %w", err)
			}
		}

		// Step 8: Create unique indexes with partial WHERE clauses
		// Index 1: App-level templates (non-org-scoped)
		indexQuery1 := `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_app_env_name_template
			ON roles (app_id, environment_id, name, is_template)
			WHERE organization_id IS NULL
		`
		if _, err := db.ExecContext(ctx, indexQuery1); err != nil {
			// For SQLite, WHERE clause might not be fully supported in older versions
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "already exists") {
				return fmt.Errorf("failed to create app-level unique index: %w", err)
			}
		}

		// Index 2: Organization-scoped roles
		indexQuery2 := `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_app_env_org_name_template
			ON roles (app_id, environment_id, organization_id, name, is_template)
			WHERE organization_id IS NOT NULL
		`
		if _, err := db.ExecContext(ctx, indexQuery2); err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "already exists") {
				return fmt.Errorf("failed to create org-level unique index: %w", err)
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback: Drop indexes and columns
		queries := []string{
			`DROP INDEX IF EXISTS idx_roles_app_env_org_name_template`,
			`DROP INDEX IF EXISTS idx_roles_app_env_name_template`,
			`ALTER TABLE roles DROP COLUMN IF EXISTS display_name`,
			`ALTER TABLE roles DROP COLUMN IF EXISTS environment_id`,
		}

		for _, query := range queries {
			if _, err := db.ExecContext(ctx, query); err != nil {
				// Ignore errors on rollback for compatibility
				continue
			}
		}

		return nil
	})
}

// toTitleCase converts a snake_case string to Title Case
// Example: "workspace_owner" -> "Workspace Owner"
func toTitleCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// generateXID generates a new XID and returns it as a string
func generateXID() string {
	return xid.New().String()
}

