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
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("organization not found: %s", orgID)
			}
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

var orgAddMemberCmd = &cobra.Command{
	Use:   "add-member [org-id] [user-id]",
	Short: "Add a member to an organization",
	Long:  `Add a user as a member of an organization with a specified role.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		orgID := args[0]
		userID := args[1]
		role, _ := cmd.Flags().GetString("role")

		if role == "" {
			role = "member"
		}

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		err = addOrganizationMember(db, orgID, userID, role)
		if err != nil {
			return fmt.Errorf("failed to add member: %w", err)
		}

		fmt.Printf("✓ Successfully added user %s to organization %s with role: %s\n", userID, orgID, role)
		return nil
	},
}

var orgRemoveMemberCmd = &cobra.Command{
	Use:   "remove-member [org-id] [user-id]",
	Short: "Remove a member from an organization",
	Long:  `Remove a user from an organization.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("database-url")
		orgID := args[0]
		userID := args[1]

		db, err := connectOrgDB(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		err = removeOrganizationMember(db, orgID, userID)
		if err != nil {
			return fmt.Errorf("failed to remove member: %w", err)
		}

		fmt.Printf("✓ Successfully removed user %s from organization %s\n", userID, orgID)
		return nil
	},
}

func init() {
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgShowCmd)
	orgCmd.AddCommand(orgDeleteCmd)
	orgCmd.AddCommand(orgMembersCmd)
	orgCmd.AddCommand(orgAddMemberCmd)
	orgCmd.AddCommand(orgRemoveMemberCmd)

	// Common flags
	orgCmd.PersistentFlags().String("database-url", "authsome.db", "Database URL")

	// Create command flags
	orgCreateCmd.Flags().String("name", "", "Organization name (required)")
	orgCreateCmd.Flags().String("slug", "", "Organization slug (required)")
	orgCreateCmd.Flags().String("description", "", "Organization description")

	// Delete command flags
	orgDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion")

	// Add member command flags
	orgAddMemberCmd.Flags().String("role", "member", "Member role (default: member)")
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

// addOrganizationMember adds a user to an organization
func addOrganizationMember(db *bun.DB, orgID string, userID string, role string) error {
	ctx := context.Background()

	// Parse IDs
	parsedOrgID, err := xid.FromString(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
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

	// Check if organization exists
	var org schema.Organization
	err = db.NewSelect().Model(&org).Where("id = ?", parsedOrgID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Check if member already exists
	var existingMember schema.Member
	err = db.NewSelect().Model(&existingMember).
		Where("user_id = ? AND organization_id = ?", parsedUserID, parsedOrgID).
		Scan(ctx)
	if err == nil {
		return fmt.Errorf("user is already a member of this organization")
	}

	// Create member
	memberID := xid.New()
	systemID := xid.New() // System user for CLI operations
	member := &schema.Member{
		ID:             memberID,
		UserID:         parsedUserID,
		OrganizationID: parsedOrgID,
		Role:           role,
	}

	// Set audit fields
	member.AuditableModel.ID = memberID
	member.AuditableModel.CreatedBy = systemID
	member.AuditableModel.UpdatedBy = systemID

	_, err = db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// removeOrganizationMember removes a user from an organization
func removeOrganizationMember(db *bun.DB, orgID string, userID string) error {
	ctx := context.Background()

	// Parse IDs
	parsedOrgID, err := xid.FromString(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	parsedUserID, err := xid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if member exists
	var member schema.Member
	err = db.NewSelect().Model(&member).
		Where("user_id = ? AND organization_id = ?", parsedUserID, parsedOrgID).
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

// deleteOrganization deletes an organization and its members
func deleteOrganization(db *bun.DB, orgID string) error {
	ctx := context.Background()

	// Parse organization ID
	parsedOrgID, err := xid.FromString(orgID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}

	// Check if organization exists
	var org schema.Organization
	err = db.NewSelect().Model(&org).Where("id = ?", parsedOrgID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Delete members first (respecting foreign keys)
	_, err = db.NewDelete().
		Model((*schema.Member)(nil)).
		Where("organization_id = ?", parsedOrgID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete members: %w", err)
	}

	// Delete organization
	_, err = db.NewDelete().
		Model((*schema.Organization)(nil)).
		Where("id = ?", parsedOrgID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
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
