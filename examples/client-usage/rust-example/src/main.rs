// Rust Client Example
//
// This example demonstrates using the generated AuthSome Rust client
// with plugin composition.

use authsome_client::{AuthsomeClient, AuthsomeError, Result};
use authsome_client::client::{SignUpRequest, UpdateUserRequest};

#[tokio::main]
async fn main() -> Result<()> {
    println!("AuthSome Rust Client Example\n");

    // Initialize client
    let mut client = AuthsomeClient::builder()
        .base_url("http://localhost:8080")
        .build()?;

    // Example 1: User Registration
    println!("1. Registering new user...");
    let signup_response = client.sign_up(SignUpRequest {
        email: "test@example.com".to_string(),
        password: "SecurePassword123!".to_string(),
        name: Some("Test User".to_string()),
    }).await?;
    
    println!("✓ User registered: {}", signup_response.user.email);
    println!("✓ Session created: {}", signup_response.session.id);

    // Store token for authenticated requests
    client.set_token(signup_response.session.token.clone());

    // Example 2: Get Current Session
    println!("\n2. Fetching current session...");
    let session_response = client.get_session().await?;
    println!("✓ Current user: {}", session_response.user.email);
    println!("✓ Session expires: {}", session_response.session.expires_at);

    // Example 3: Update User Profile
    println!("\n3. Updating user profile...");
    let update_response = client.update_user(UpdateUserRequest {
        name: Some("Updated Test User".to_string()),
        email: None,
    }).await?;
    
    if let Some(name) = &update_response.user.name {
        println!("✓ Profile updated: {}", name);
    }

    // Example 4: List Devices
    println!("\n4. Listing devices...");
    let devices_response = client.list_devices().await?;
    println!("✓ Found {} device(s)", devices_response.devices.len());

    // Example 5: Sign Out
    println!("\n5. Signing out...");
    client.sign_out().await?;
    println!("✓ Signed out successfully");

    println!("\n✓ Example completed successfully!");
    
    Ok(())
}

// Example error handling
fn handle_error(error: AuthsomeError) {
    match error {
        AuthsomeError::Unauthorized(msg) => {
            eprintln!("❌ Unauthorized: {}", msg);
        }
        AuthsomeError::Validation(msg) => {
            eprintln!("❌ Validation error: {}", msg);
        }
        AuthsomeError::NotFound(msg) => {
            eprintln!("❌ Not found: {}", msg);
        }
        AuthsomeError::Api { status, message } => {
            eprintln!("❌ API Error ({}): {}", status, message);
        }
        _ => {
            eprintln!("❌ Error: {:?}", error);
        }
    }
}

