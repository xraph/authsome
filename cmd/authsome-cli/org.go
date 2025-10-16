package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/rs/xid"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// orgCmd represents the org command
var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Organization management commands",
	Long:  `Commands for managing organizations in SaaS mode.`,
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations",
	Long:  `List all organizations in the system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		orgs, err := listOrganizations(db)
		if err != nil {
			return fmt.Errorf("failed to list organizations: %w", err)
		}

		printOrganizations(orgs)
		return nil
	},
}

var orgCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new organization",
	Long:  `Create a new organization with the specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		name, _ := cmd.Flags().GetString("name")
		slug, _ := cmd.Flags().GetString("slug")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("organization name is required")
		}
		if slug == "" {
			return fmt.Errorf("organization slug is required")
		}

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		org, err := createOrganization(db, name, slug, description)
		if err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		fmt.Printf("Created organization:\n")
		fmt.Printf("  ID: %s\n", org.ID)
		fmt.Printf("  Name: %s\n", org.Name)
		fmt.Printf("  Slug: %s\n", org.Slug)
		fmt.Printf("  Logo: %s\n", org.Logo)
		fmt.Printf("  Created: %s\n", org.CreatedAt.Format(time.RFC3339))

		return nil
	},
}

var orgShowCmd = &cobra.Command{
	Use:   "show [org-id]",
	Short: "Show organization details",
	Long:  `Show detailed information about a specific organization.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		orgID := args[0]

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		org, err := getOrganization(db, orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization: %w", err)
		}

		members, err := getOrganizationMembers(db, orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization members: %w", err)
		}

		printOrganizationDetails(org, members)
		return nil
	},
}

var orgDeleteCmd = &cobra.Command{
	Use:   "delete [org-id]",
	Short: "Delete an organization",
	Long:  `Delete an organization and all its associated data.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		orgID := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("This will permanently delete organization '%s' and all its data.\n", orgID)
			fmt.Println("Use --confirm flag to proceed.")
			return nil
		}

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		if err := deleteOrganization(db, orgID); err != nil {
			return fmt.Errorf("failed to delete organization: %w", err)
		}

		fmt.Printf("Successfully deleted organization: %s\n", orgID)
		return nil
	},
}

var orgMembersCmd = &cobra.Command{
	Use:   "members [org-id]",
	Short: "List organization members",
	Long:  `List all members of a specific organization.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		orgID := args[0]

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		members, err := getOrganizationMembers(db, orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization members: %w", err)
		}

		printMembers(members)
		return nil
	},
}

func init() {
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgShowCmd)
	orgCmd.AddCommand(orgDeleteCmd)
	orgCmd.AddCommand(orgMembersCmd)

	// Common flags
	orgCmd.PersistentFlags().String("database-url", "authsome.db", "Database URL")

	// Create command flags
	orgCreateCmd.Flags().String("name", "", "Organization name (required)")
	orgCreateCmd.Flags().String("slug", "", "Organization slug (required)")
	orgCreateCmd.Flags().String("description", "", "Organization description")

	// Delete command flags
	orgDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion")
}

// connectOrgDB connects to the database for organization operations
func connectOrgDB(dbURL string) (*bun.DB, error) {
	// Use the shared multi-database connection function
	return connectDatabaseMulti()
}

// listOrganizations retrieves all organizations
func listOrganizations(db *bun.DB) ([]*schema.Organization, error) {
	var orgs []*schema.Organization
	err := db.NewSelect().
		Model(&orgs).
		Order("created_at DESC").
		Scan(context.Background())
	return orgs, err
}

// createOrganization creates a new organization
func createOrganization(db *bun.DB, name, slug, description string) (*schema.Organization, error) {
	orgID := xid.New()
	systemID := xid.New() // System user for CLI operations

	org := &schema.Organization{
		ID:   orgID,
		Name: name,
		Slug: slug,
		Logo: description, // Using logo field for description since Description doesn't exist
	}

	// Set audit fields manually for CLI operations
	org.AuditableModel.ID = orgID
	org.AuditableModel.CreatedBy = systemID
	org.AuditableModel.UpdatedBy = systemID

	_, err := db.NewInsert().Model(org).Exec(context.Background())
	if err != nil {
		return nil, err
	}

	return org, nil
}

// getOrganization retrieves a specific organization
func getOrganization(db *bun.DB, orgID string) (*schema.Organization, error) {
	org := &schema.Organization{}
	err := db.NewSelect().
		Model(org).
		Where("id = ? OR slug = ?", orgID, orgID).
		Scan(context.Background())
	return org, err
}

// getOrganizationMembers retrieves members of an organization
func getOrganizationMembers(db *bun.DB, orgID string) ([]*MemberWithUser, error) {
	var members []*MemberWithUser
	err := db.NewSelect().
		Model((*schema.Member)(nil)).
		Column("m.*").
		ColumnExpr("u.email, u.name").
		Join("JOIN users AS u ON u.id = m.user_id").
		Where("m.organization_id = ?", orgID).
		Order("m.created_at DESC").
		Scan(context.Background(), &members)
	return members, err
}

// deleteOrganization deletes an organization and all its data
func deleteOrganization(db *bun.DB, orgID string) error {
	ctx := context.Background()

	// Delete in order to respect foreign key constraints
	tables := []struct {
		model interface{}
		where string
	}{
		{(*schema.AuditEvent)(nil), "organization_id = ?"},
		{(*schema.UserRole)(nil), "organization_id = ?"},
		{(*schema.TeamMember)(nil), "team_id IN (SELECT id FROM teams WHERE organization_id = ?)"},
		{(*schema.Team)(nil), "organization_id = ?"},
		{(*schema.FormSchema)(nil), "organization_id = ?"},
		{(*schema.Webhook)(nil), "organization_id = ?"},
		{(*schema.APIKey)(nil), "organization_id = ?"},
		{(*schema.Policy)(nil), "organization_id = ?"},
		{(*schema.Permission)(nil), "organization_id = ?"},
		{(*schema.Role)(nil), "organization_id = ?"},
		{(*schema.Device)(nil), "user_id IN (SELECT user_id FROM members WHERE organization_id = ?)"},
		{(*schema.Verification)(nil), "user_id IN (SELECT user_id FROM members WHERE organization_id = ?)"},
		{(*schema.Account)(nil), "user_id IN (SELECT user_id FROM members WHERE organization_id = ?)"},
		{(*schema.Session)(nil), "user_id IN (SELECT user_id FROM members WHERE organization_id = ?)"},
		{(*schema.Member)(nil), "organization_id = ?"},
		{(*schema.Organization)(nil), "id = ?"},
	}

	for _, table := range tables {
		_, err := db.NewDelete().
			Model(table.model).
			Where(table.where, orgID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete from table: %w", err)
		}
	}

	return nil
}

// MemberWithUser represents a member with user details
type MemberWithUser struct {
	schema.Member
	Email string `bun:"email"`
	Name  string `bun:"name"`
}

// printOrganizations prints a table of organizations
func printOrganizations(orgs []*schema.Organization) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tCREATED")
	fmt.Fprintln(w, "---\t----\t----\t-------")

	for _, org := range orgs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			org.ID,
			org.Name,
			org.Slug,
			org.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
}

// printOrganizationDetails prints detailed organization information
func printOrganizationDetails(org *schema.Organization, members []*MemberWithUser) {
	fmt.Printf("Organization Details:\n")
	fmt.Printf("  ID: %s\n", org.ID)
	fmt.Printf("  Name: %s\n", org.Name)
	fmt.Printf("  Slug: %s\n", org.Slug)
	fmt.Printf("  Logo: %s\n", org.Logo)
	fmt.Printf("  Created: %s\n", org.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated: %s\n", org.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("  Members: %d\n\n", len(members))

	if len(members) > 0 {
		fmt.Printf("Members:\n")
		printMembers(members)
	}
}

// printMembers prints a table of members
func printMembers(members []*MemberWithUser) {
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
