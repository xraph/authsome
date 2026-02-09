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
	"golang.org/x/crypto/bcrypt"
)

// userCmd represents the user command.
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  `Commands for managing users in the AuthSome system.`,
}

// User list command.
var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  `List all users or users in a specific organization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, _ := cmd.Flags().GetString("app")

		return listUsers(appID)
	},
}

// User create command.
var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  `Create a new user with email and password.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		appID, _ := cmd.Flags().GetString("app")
		role, _ := cmd.Flags().GetString("role")
		verified, _ := cmd.Flags().GetBool("verified")

		if email == "" || password == "" || appID == "" {
			return errs.New(errs.CodeInvalidInput, "email, password, and app are required", http.StatusBadRequest)
		}

		user, err := createUser(email, password, firstName, lastName, appID, role, verified)
		if err != nil {
			return err
		}

		fmt.Printf("User created successfully:\n")
		fmt.Printf("  ID: %s\n", user.ID)
		fmt.Printf("  Email: %s\n", user.Email)
		fmt.Printf("  Name: %s\n", user.Name)
		fmt.Printf("  Verified: %t\n", user.EmailVerified)
		fmt.Printf("  Created: %s\n", user.CreatedAt.Format(time.RFC3339))

		return nil
	},
}

// User show command.
var userShowCmd = &cobra.Command{
	Use:   "show [user-id]",
	Short: "Show user details",
	Long:  `Show detailed information about a specific user.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]

		return showUser(userID)
	},
}

// User delete command.
var userDeleteCmd = &cobra.Command{
	Use:   "delete [user-id]",
	Short: "Delete a user",
	Long:  `Delete a user from the system.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete user %s? (y/N): ", userID)
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil {
				fmt.Println("Error reading input:", err)
				return err
			}
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled")

				return nil
			}
		}

		return deleteUser(userID)
	},
}

// User password command.
var userPasswordCmd = &cobra.Command{
	Use:   "password [user-id]",
	Short: "Update user password",
	Long:  `Update a user's password.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		password, _ := cmd.Flags().GetString("password")

		if password == "" {
			return errs.New(errs.CodeInvalidInput, "password is required", http.StatusBadRequest)
		}

		return updateUserPassword(userID, password)
	},
}

// User verify command.
var userVerifyCmd = &cobra.Command{
	Use:   "verify [user-id]",
	Short: "Verify user email",
	Long:  `Mark a user's email as verified.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]

		return verifyUser(userID)
	},
}

func init() {
	// Add subcommands
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userPasswordCmd)
	userCmd.AddCommand(userVerifyCmd)

	// List flags
	userListCmd.Flags().StringP("app", "a", "", "Filter by app ID")

	// Create flags
	userCreateCmd.Flags().StringP("email", "e", "", "User email (required)")
	userCreateCmd.Flags().StringP("password", "p", "", "User password (required)")
	userCreateCmd.Flags().String("first-name", "", "User first name")
	userCreateCmd.Flags().String("last-name", "", "User last name")
	userCreateCmd.Flags().StringP("app", "a", "", "App ID (required)")
	userCreateCmd.Flags().StringP("role", "r", "member", "User role in app")
	userCreateCmd.Flags().Bool("verified", false, "Mark email as verified")

	// Delete flags
	userDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Password flags
	userPasswordCmd.Flags().StringP("password", "p", "", "New password (required)")
}

// connectUserDB connects to the database (now supports PostgreSQL, MySQL, SQLite).
func connectUserDB() (*bun.DB, error) {
	return connectDatabaseMulti()
}

// listUsers lists all users with their app memberships.
func listUsers(appID string) error {
	db, err := connectUserDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	var users []schema.User

	query := db.NewSelect().Model(&users)

	if appID != "" {
		// Filter by app through members table
		query = query.Join("JOIN members m ON m.user_id = u.id").
			Where("m.app_id = ?", appID)
	}

	err = query.Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")

		return nil
	}

	fmt.Printf("Found %d users:\n\n", len(users))

	// Use tabwriter for better formatting
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tNAME\tVERIFIED\tCREATED")
	fmt.Fprintln(w, "---\t-----\t----\t--------\t-------")

	for _, user := range users {
		verified := "No"
		if user.EmailVerified {
			verified = "Yes"
		}

		name := user.Name
		if name == "" {
			name = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			user.ID,
			user.Email,
			name,
			verified,
			user.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()

	return nil
}

// createUser creates a new user.
func createUser(email, password, firstName, lastName, appID, role string, verified bool) (*schema.User, error) {
	db, err := connectUserDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ctx := context.Background()

	// Check if app exists
	var app schema.App

	err = db.NewSelect().Model(&app).Where("id = ?", appID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	// Check if user already exists
	var existingUser schema.User

	err = db.NewSelect().Model(&existingUser).Where("email = ?", email).Scan(ctx)
	if err == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	userID := xid.New()
	systemID := xid.New() // System user for CLI operations

	user := &schema.User{
		ID:            userID,
		Email:         email,
		Username:      email, // Use email as default username
		PasswordHash:  string(hashedPassword),
		Name:          firstName + " " + lastName,
		EmailVerified: verified,
	}

	// Set audit fields manually for CLI operations
	user.AuditableModel.ID = userID
	user.CreatedBy = systemID
	user.UpdatedBy = systemID

	_, err = db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Parse app ID
	appXID, err := xid.FromString(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	// Create membership
	memberID := xid.New()

	member := &schema.Member{
		ID:     memberID,
		UserID: user.ID,
		AppID:  appXID,
		Role:   schema.MemberRole(role),
	}

	// Set audit fields manually for CLI operations
	member.AuditableModel.ID = memberID
	member.CreatedBy = systemID
	member.UpdatedBy = systemID

	_, err = db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create membership: %w", err)
	}

	return user, nil
}

// showUser shows detailed information about a user.
func showUser(userID string) error {
	db, err := connectUserDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	// Get user
	var user schema.User

	err = db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get memberships
	var memberships []schema.Member

	err = db.NewSelect().Model(&memberships).
		Relation("App").
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get memberships: %w", err)
	}

	// Print user details
	fmt.Printf("User Details:\n")
	fmt.Printf("  ID: %s\n", user.ID)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Verified: %t\n", user.EmailVerified)
	fmt.Printf("  Created: %s\n", user.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated: %s\n", user.UpdatedAt.Format(time.RFC3339))

	if len(memberships) > 0 {
		fmt.Printf("\nApp Memberships:\n")

		for _, membership := range memberships {
			appName := "Unknown"
			if membership.App != nil {
				appName = membership.App.Name
			}

			fmt.Printf("  - %s (%s) - Role: %s\n", appName, membership.AppID, membership.Role)
		}
	}

	return nil
}

// deleteUser deletes a user.
func deleteUser(userID string) error {
	db, err := connectUserDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	// Check if user exists
	var user schema.User

	err = db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Delete memberships first
	_, err = db.NewDelete().Model((*schema.Member)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete memberships: %w", err)
	}

	// Delete user
	_, err = db.NewDelete().Model(&user).Where("id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("User %s deleted successfully\n", userID)

	return nil
}

// updateUserPassword updates a user's password.
func updateUserPassword(userID, password string) error {
	db, err := connectUserDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	// Check if user exists
	var user schema.User

	err = db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	_, err = db.NewUpdate().Model(&user).
		Set("password_hash = ?", string(hashedPassword)).
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	fmt.Printf("Password updated successfully for user %s\n", userID)

	return nil
}

// verifyUser marks a user's email as verified.
func verifyUser(userID string) error {
	db, err := connectUserDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	// Check if user exists
	var user schema.User

	err = db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update verification status
	now := time.Now()

	_, err = db.NewUpdate().Model(&user).
		Set("email_verified = ?", true).
		Set("email_verified_at = ?", now).
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	fmt.Printf("User %s email verified successfully\n", userID)

	return nil
}
