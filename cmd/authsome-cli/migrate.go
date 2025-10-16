package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/xraph/authsome/migrations"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
	Long:  `Commands for managing database migrations including up, down, status, and reset operations.`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	Long:  `Run all pending database migrations to bring the database up to date.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := connectDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		migrator := migrate.NewMigrator(db, migrations.Migrations)

		if err := migrator.Init(context.Background()); err != nil {
			return fmt.Errorf("failed to initialize migrator: %w", err)
		}

		group, err := migrator.Migrate(context.Background())
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		if group.IsZero() {
			fmt.Println("No new migrations to run")
		} else {
			fmt.Printf("Migrated to %s\n", group)
		}

		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Long:  `Rollback the last applied migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := connectDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		migrator := migrate.NewMigrator(db, migrations.Migrations)

		group, err := migrator.Rollback(context.Background())
		if err != nil {
			return fmt.Errorf("failed to rollback migration: %w", err)
		}

		if group.IsZero() {
			fmt.Println("No migrations to rollback")
		} else {
			fmt.Printf("Rolled back %s\n", group)
		}

		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  `Show the status of all migrations including which have been applied.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := connectDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		migrator := migrate.NewMigrator(db, migrations.Migrations)

		if err := migrator.Init(context.Background()); err != nil {
			return fmt.Errorf("failed to initialize migrator: %w", err)
		}

		ms, err := migrator.MigrationsWithStatus(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		fmt.Printf("Migration Status:\n")
		fmt.Printf("%-20s %-10s %s\n", "ID", "Status", "Name")
		fmt.Printf("%-20s %-10s %s\n", "----", "------", "----")

		for _, m := range ms {
			status := "pending"
			if m.GroupID != 0 {
				status = "applied"
			}
			fmt.Printf("%-20d %-10s %s\n", m.ID, status, m.Name)
		}

		return nil
	},
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset database (drop all tables and re-run migrations)",
	Long:  `Drop all tables and re-run all migrations. WARNING: This will delete all data!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Confirm before proceeding
		fmt.Print("This will delete all data in the database. Are you sure? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return nil
		}

		db, err := connectDB()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		migrator := migrate.NewMigrator(db, migrations.Migrations)

		// Reset the database
		if err := migrator.Reset(context.Background()); err != nil {
			return fmt.Errorf("failed to reset database: %w", err)
		}

		// Run all migrations
		group, err := migrator.Migrate(context.Background())
		if err != nil {
			return fmt.Errorf("failed to run migrations after reset: %w", err)
		}

		fmt.Printf("Database reset and migrated to %s\n", group)
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateResetCmd)
}

// connectDB creates a database connection using configuration
// Now supports PostgreSQL, MySQL, and SQLite
func connectDB() (*bun.DB, error) {
	return connectDatabaseMulti()
}
