// Go Client Example
//
// This example demonstrates using the generated AuthSome Go client
// with plugin composition.

package main

import (
	"context"
	"fmt"
	"log"

	authsome "github.com/xraph/authsome/clients/go"
	"github.com/xraph/authsome/clients/go/plugins/social"
	"github.com/xraph/authsome/clients/go/plugins/twofa"
)

func main() {
	ctx := context.Background()

	// Initialize client with plugins
	client := authsome.NewClient("http://localhost:8080",
		authsome.WithPlugins(
			social.NewPlugin(),
			twofa.NewPlugin(),
		),
		authsome.WithHeaders(map[string]string{
			"X-Client-Version": "1.0.0",
		}),
	)

	fmt.Println("AuthSome Go Client Example\n")

	// Example 1: User Registration
	fmt.Println("1. Registering new user...")
	signUpResp, err := client.SignUp(ctx, &authsome.SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		Name:     stringPtr("Test User"),
	})
	if err != nil {
		log.Fatalf("Failed to sign up: %v", err)
	}
	fmt.Printf("✓ User registered: %s\n", signUpResp.User.Email)
	fmt.Printf("✓ Session created: %s\n", signUpResp.Session.Id)

	// Store token for authenticated requests
	client.SetToken(signUpResp.Session.Token)

	// Example 2: Get Current Session
	fmt.Println("\n2. Fetching current session...")
	sessionResp, err := client.GetSession(ctx)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}
	fmt.Printf("✓ Current user: %s\n", sessionResp.User.Email)
	fmt.Printf("✓ Session expires: %s\n", sessionResp.Session.ExpiresAt)

	// Example 3: Update User Profile
	fmt.Println("\n3. Updating user profile...")
	updateResp, err := client.UpdateUser(ctx, &authsome.UpdateUserRequest{
		Name: stringPtr("Updated Test User"),
	})
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	if updateResp.User.Name != nil {
		fmt.Printf("✓ Profile updated: %s\n", *updateResp.User.Name)
	}

	// Example 4: Social OAuth Plugin
	fmt.Println("\n4. Using social OAuth plugin...")
	if socialPlugin, ok := client.GetPlugin("social"); ok {
		// Cast to the specific plugin type if needed
		_ = socialPlugin // Use the plugin
		fmt.Println("✓ Social OAuth plugin available")
	}

	// Example 5: List Devices
	fmt.Println("\n5. Listing devices...")
	devicesResp, err := client.ListDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}
	fmt.Printf("✓ Found %d device(s)\n", len(devicesResp.Devices))

	// Example 6: Sign Out
	fmt.Println("\n6. Signing out...")
	signOutResp, err := client.SignOut(ctx)
	if err != nil {
		log.Fatalf("Failed to sign out: %v", err)
	}
	if signOutResp.Success {
		fmt.Println("✓ Signed out successfully")
	}

	fmt.Println("\n✓ Example completed successfully!")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Example error handling
func handleError(err error) {
	if authErr, ok := err.(*authsome.Error); ok {
		fmt.Printf("❌ API Error: %s\n", authErr.Message)
		fmt.Printf("   Status Code: %d\n", authErr.StatusCode)
		fmt.Printf("   Error Code: %s\n", authErr.Code)
	} else {
		fmt.Printf("❌ Unexpected error: %v\n", err)
	}
}
