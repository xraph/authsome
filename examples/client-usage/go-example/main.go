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

	// Example 1: User Registration

	signUpResp, err := client.SignUp(ctx, &authsome.SignUpRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		Name:     stringPtr("Test User"),
	})
	if err != nil {
		log.Fatalf("Failed to sign up: %v", err)
	}


	// Store token for authenticated requests
	client.SetToken(signUpResp.Session.Token)

	// Example 2: Get Current Session

	sessionResp, err := client.GetSession(ctx)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}


	// Example 3: Update User Profile

	updateResp, err := client.UpdateUser(ctx, &authsome.UpdateUserRequest{
		Name: stringPtr("Updated Test User"),
	})
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	if updateResp.User.Name != nil {

	}

	// Example 4: Social OAuth Plugin

	if socialPlugin, ok := client.GetPlugin("social"); ok {
		// Cast to the specific plugin type if needed
		_ = socialPlugin // Use the plugin

	}

	// Example 5: List Devices

	devicesResp, err := client.ListDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	// Example 6: Sign Out

	signOutResp, err := client.SignOut(ctx)
	if err != nil {
		log.Fatalf("Failed to sign out: %v", err)
	}
	if signOutResp.Success {

	}

}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Example error handling
func handleError(err error) {
	if authErr, ok := err.(*authsome.Error); ok {


	} else {

	}
}
