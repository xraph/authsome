package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/rs/xid"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// appCmd represents the app command.
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "App management commands",
	Long:  `Commands for managing platform-level apps (tenants).`,
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all apps",
	Long:  `List all apps in the system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		apps, err := listApps(db)
		if err != nil {
			return fmt.Errorf("failed to list apps: %w", err)
		}

		printApps(apps)

		return nil
	},
}

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new app",
	Long:  `Create a new app with the specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		name, _ := cmd.Flags().GetString("name")
		slug, _ := cmd.Flags().GetString("slug")
		logo, _ := cmd.Flags().GetString("logo")

		if name == "" {
			return errs.New(errs.CodeInvalidInput, "app name is required", http.StatusBadRequest)
		}
		if slug == "" {
			return errs.New(errs.CodeInvalidInput, "app slug is required", http.StatusBadRequest)
		}

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		app, err := createApp(db, name, slug, logo)
		if err != nil {
			return fmt.Errorf("failed to create app: %w", err)
		}

		fmt.Printf("Created app:\n")
		fmt.Printf("  ID: %s\n", app.ID)
		fmt.Printf("  Name: %s\n", app.Name)
		fmt.Printf("  Slug: %s\n", app.Slug)
		fmt.Printf("  Logo: %s\n", app.Logo)
		fmt.Printf("  Created: %s\n", app.CreatedAt.Format(time.RFC3339))

		return nil
	},
}

var appShowCmd = &cobra.Command{
	Use:   "show [app-id]",
	Short: "Show app details",
	Long:  `Show detailed information about a specific app.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		appID := args[0]

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		app, err := getApp(db, appID)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("app not found: %s", appID)
			}

			return fmt.Errorf("failed to get app: %w", err)
		}

		members, err := getAppMembers(db, appID)
		if err != nil {
			return fmt.Errorf("failed to get app members: %w", err)
		}

		printAppDetails(app, members)

		return nil
	},
}

var appDeleteCmd = &cobra.Command{
	Use:   "delete [app-id]",
	Short: "Delete an app",
	Long:  `Delete an app and all its associated data.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		appID := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("This will permanently delete app '%s' and all its data.\n", appID)
			fmt.Println("Use --confirm flag to proceed.")

			return nil
		}

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		if err := deleteApp(db, appID); err != nil {
			return fmt.Errorf("failed to delete app: %w", err)
		}

		fmt.Printf("Successfully deleted app: %s\n", appID)

		return nil
	},
}

var appMembersCmd = &cobra.Command{
	Use:   "members [app-id]",
	Short: "List app members",
	Long:  `List all members of a specific app.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		appID := args[0]

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		members, err := getAppMembers(db, appID)
		if err != nil {
			return fmt.Errorf("failed to get app members: %w", err)
		}

		printAppMembersList(members)

		return nil
	},
}

var appAddMemberCmd = &cobra.Command{
	Use:   "add-member [app-id] [user-id]",
	Short: "Add a member to an app",
	Long:  `Add a user as a member of an app with a specified role.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		appID := args[0]
		userID := args[1]
		role, _ := cmd.Flags().GetString("role")

		if role == "" {
			role = "member"
		}

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		err = addAppMember(db, appID, userID, role)
		if err != nil {
			return fmt.Errorf("failed to add member: %w", err)
		}

		fmt.Printf("✓ Successfully added user %s to app %s with role: %s\n", userID, appID, role)

		return nil
	},
}

var appRemoveMemberCmd = &cobra.Command{
	Use:   "remove-member [app-id] [user-id]",
	Short: "Remove a member from an app",
	Long:  `Remove a user from an app.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		appID := args[0]
		userID := args[1]

		db, err := connectAppDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		err = removeAppMember(db, appID, userID)
		if err != nil {
			return fmt.Errorf("failed to remove member: %w", err)
		}

		fmt.Printf("✓ Successfully removed user %s from app %s\n", userID, appID)

		return nil
	},
}

func init() {
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appCreateCmd)
	appCmd.AddCommand(appShowCmd)
	appCmd.AddCommand(appDeleteCmd)
	appCmd.AddCommand(appMembersCmd)
	appCmd.AddCommand(appAddMemberCmd)
	appCmd.AddCommand(appRemoveMemberCmd)

	// Common flags
	appCmd.PersistentFlags().String("database-url", "authsome.db", "Database URL")

	// Create command flags
	appCreateCmd.Flags().String("name", "", "App name (required)")
	appCreateCmd.Flags().String("slug", "", "App slug (required)")
	appCreateCmd.Flags().String("logo", "", "App logo URL")

	// Delete command flags
	appDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion")

	// Add member command flags
	appAddMemberCmd.Flags().String("role", "member", "Member role (default: member)")
}

// connectAppDB connects to the database for app operations.
func connectAppDB(dbURL string) (*bun.DB, error) {
	// Use the shared multi-database connection function
	return connectDatabaseMulti()
}

// listApps retrieves all apps.
func listApps(db *bun.DB) ([]*schema.App, error) {
	var apps []*schema.App

	err := db.NewSelect().
		Model(&apps).
		Order("created_at DESC").
		Scan(context.Background())

	return apps, err
}

// createApp creates a new app.
func createApp(db *bun.DB, name, slug, logo string) (*schema.App, error) {
	appID := xid.New()
	systemID := xid.New() // System user for CLI operations

	app := &schema.App{
		ID:   appID,
		Name: name,
		Slug: slug,
		Logo: logo,
	}

	// Set audit fields manually for CLI operations
	app.AuditableModel.ID = appID
	app.CreatedBy = systemID
	app.UpdatedBy = systemID

	_, err := db.NewInsert().Model(app).Exec(context.Background())
	if err != nil {
		return nil, err
	}

	return app, nil
}

// getApp retrieves a specific app.
func getApp(db *bun.DB, appID string) (*schema.App, error) {
	app := &schema.App{}
	err := db.NewSelect().
		Model(app).
		Where("id = ? OR slug = ?", appID, appID).
		Scan(context.Background())

	return app, err
}

// getAppMembers retrieves members of an app.
func getAppMembers(db *bun.DB, appID string) ([]*AppMemberWithUser, error) {
	var members []*AppMemberWithUser

	err := db.NewSelect().
		Model((*schema.Member)(nil)).
		Column("m.*").
		ColumnExpr("u.email, u.name").
		Join("JOIN users AS u ON u.id = m.user_id").
		Where("m.app_id = ?", appID).
		Order("m.created_at DESC").
		Scan(context.Background(), &members)

	return members, err
}

// addAppMember adds a user to an app.
func addAppMember(db *bun.DB, appID string, userID string, role string) error {
	ctx := context.Background()

	// Parse IDs
	parsedAppID, err := xid.FromString(appID)
	if err != nil {
		return fmt.Errorf("invalid app ID: %w", err)
	}

	parsedUserID, err := xid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user exists
	var user schema.User

	err = db.NewSelect().Model(&user).Where("id = ?", parsedUserID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if app exists
	var app schema.App

	err = db.NewSelect().Model(&app).Where("id = ?", parsedAppID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	// Check if member already exists
	var existingMember schema.Member

	err = db.NewSelect().Model(&existingMember).
		Where("user_id = ? AND app_id = ?", parsedUserID, parsedAppID).
		Scan(ctx)
	if err == nil {
		return errs.New(errs.CodeInvalidInput, "user is already a member of this app", http.StatusBadRequest)
	}

	// Create member
	memberID := xid.New()
	systemID := xid.New() // System user for CLI operations
	member := &schema.Member{
		ID:     memberID,
		UserID: parsedUserID,
		AppID:  parsedAppID,
		Role:   schema.MemberRole(role),
	}

	// Set audit fields
	member.AuditableModel.ID = memberID
	member.CreatedBy = systemID
	member.UpdatedBy = systemID

	_, err = db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// removeAppMember removes a user from an app.
func removeAppMember(db *bun.DB, appID string, userID string) error {
	ctx := context.Background()

	// Parse IDs
	parsedAppID, err := xid.FromString(appID)
	if err != nil {
		return fmt.Errorf("invalid app ID: %w", err)
	}

	parsedUserID, err := xid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if member exists
	var member schema.Member

	err = db.NewSelect().Model(&member).
		Where("user_id = ? AND app_id = ?", parsedUserID, parsedAppID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	// Delete member
	_, err = db.NewDelete().Model(&member).Where("id = ?", member.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}

// deleteApp deletes an app and its members.
func deleteApp(db *bun.DB, appID string) error {
	ctx := context.Background()

	// Parse app ID
	parsedAppID, err := xid.FromString(appID)
	if err != nil {
		return fmt.Errorf("invalid app ID: %w", err)
	}

	// Check if app exists
	var app schema.App

	err = db.NewSelect().Model(&app).Where("id = ?", parsedAppID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	// Delete members first (respecting foreign keys)
	_, err = db.NewDelete().
		Model((*schema.Member)(nil)).
		Where("app_id = ?", parsedAppID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete members: %w", err)
	}

	// Delete app
	_, err = db.NewDelete().
		Model((*schema.App)(nil)).
		Where("id = ?", parsedAppID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	return nil
}

// AppMemberWithUser represents a member with user details.
type AppMemberWithUser struct {
	schema.Member

	Email string `bun:"email"`
	Name  string `bun:"name"`
}

// printApps prints a table of apps.
func printApps(apps []*schema.App) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tCREATED")
	fmt.Fprintln(w, "---\t----\t----\t-------")

	for _, app := range apps {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			app.ID,
			app.Name,
			app.Slug,
			app.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
}

// printAppDetails prints detailed app information.
func printAppDetails(app *schema.App, members []*AppMemberWithUser) {
	fmt.Printf("App Details:\n")
	fmt.Printf("  ID: %s\n", app.ID)
	fmt.Printf("  Name: %s\n", app.Name)
	fmt.Printf("  Slug: %s\n", app.Slug)
	fmt.Printf("  Logo: %s\n", app.Logo)
	fmt.Printf("  Created: %s\n", app.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated: %s\n", app.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("  Members: %d\n\n", len(members))

	if len(members) > 0 {
		fmt.Printf("Members:\n")
		printAppMembersList(members)
	}
}

// printAppMembersList prints a table of members.
func printAppMembersList(members []*AppMemberWithUser) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EMAIL\tNAME\tROLE\tJOINED")
	fmt.Fprintln(w, "-----\t----\t----\t------")

	for _, member := range members {
		name := member.Name
		if name == "" {
			name = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			member.Email,
			name,
			member.Role,
			member.CreatedAt.Format("2006-01-02"),
		)
	}

	w.Flush()
}
