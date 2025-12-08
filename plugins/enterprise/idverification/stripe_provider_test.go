package idverification

import (
	"context"
	"os"
	"testing"

	"github.com/rs/xid"
)

func TestStripeIdentityProvider_Mock(t *testing.T) {
	config := StripeIdentityConfig{
		APIKey:                "mock",
		UseMock:               true,
		RequireLiveCapture:    true,
		RequireMatchingSelfie: true,
	}

	provider, err := NewStripeIdentityProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.GetProviderName() != "stripe_identity" {
		t.Errorf("Expected provider name 'stripe_identity', got '%s'", provider.GetProviderName())
	}

	t.Run("CreateSession_Mock", func(t *testing.T) {
		ctx := context.Background()

		userID := xid.New()
		appID := xid.New()
		orgID := xid.New()

		req := &ProviderSessionRequest{
			AppID:          appID,
			OrganizationID: orgID,
			UserID:         userID,
			SuccessURL:     "https://example.com/success",
		}

		session, err := provider.CreateSession(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if session == nil {
			t.Fatal("Expected session, got nil")
		}

		if session.ID == "" {
			t.Error("Expected session ID")
		}

		if session.URL == "" {
			t.Error("Expected session URL")
		}

		if session.Token == "" {
			t.Error("Expected session token")
		}

		if session.Status != "requires_input" {
			t.Errorf("Expected status 'requires_input', got '%s'", session.Status)
		}

		// Provider name is returned by GetProviderName(), not stored in session
	})

	t.Run("GetSession_Mock", func(t *testing.T) {
		ctx := context.Background()
		sessionID := "vs_mock_123"

		session, err := provider.GetSession(ctx, sessionID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if session == nil {
			t.Fatal("Expected session, got nil")
		}

		if session.ID != sessionID {
			t.Errorf("Expected session ID '%s', got '%s'", sessionID, session.ID)
		}

		if session.Status != "verified" {
			t.Errorf("Expected status 'verified', got '%s'", session.Status)
		}
	})

	t.Run("GetCheck_Mock", func(t *testing.T) {
		ctx := context.Background()
		sessionID := "vs_mock_123"

		result, err := provider.GetCheck(ctx, sessionID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if result.ID != sessionID {
			t.Errorf("Expected ID '%s', got '%s'", sessionID, result.ID)
		}

		if result.Result != "clear" {
			t.Errorf("Expected Result 'clear', got '%s'", result.Result)
		}

		if !result.IsDocumentValid {
			t.Error("Expected IsDocumentValid to be true")
		}

		if !result.IsLive {
			t.Error("Expected IsLive to be true")
		}

		if result.DateOfBirth == nil {
			t.Error("Expected DateOfBirth to be set")
		}
	})

	t.Run("VerifyWebhook_Mock", func(t *testing.T) {
		config.WebhookSecret = "test_secret"
		provider, _ := NewStripeIdentityProvider(config)

		valid, err := provider.VerifyWebhook("test_signature", "test_payload")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !valid {
			t.Error("Expected webhook to be valid in mock mode")
		}
	})
}

// Integration test with real Stripe API (requires STRIPE_SECRET_KEY env var)
func TestStripeIdentityProvider_Real(t *testing.T) {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: STRIPE_SECRET_KEY not set")
	}

	// Only run with short flag disabled (integration tests)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := StripeIdentityConfig{
		APIKey:                apiKey,
		UseMock:               false,
		RequireLiveCapture:    true,
		RequireMatchingSelfie: false,
		AllowedTypes:          []string{"document"},
	}

	provider, err := NewStripeIdentityProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	t.Run("CreateSession_Real", func(t *testing.T) {
		ctx := context.Background()

		userID := xid.New()
		appID := xid.New()
		orgID := xid.New()

		req := &ProviderSessionRequest{
			AppID:          appID,
			OrganizationID: orgID,
			UserID:         userID,
			SuccessURL:     "https://example.com/success",
			Metadata: map[string]interface{}{
				"test": "true",
			},
		}

		session, err := provider.CreateSession(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session, got nil")
		}

		if session.ID == "" {
			t.Error("Expected session ID")
		}

		if session.URL == "" {
			t.Error("Expected session URL")
		}

		if session.Token == "" {
			t.Error("Expected session token (client_secret)")
		}

		t.Logf("Created Stripe session: %s", session.ID)
		t.Logf("URL: %s", session.URL)
		t.Logf("Status: %s", session.Status)

		// Test GetSession with the created session
		t.Run("GetSession_Real", func(t *testing.T) {
			retrievedSession, err := provider.GetSession(ctx, session.ID)
			if err != nil {
				t.Fatalf("Failed to get session: %v", err)
			}

			if retrievedSession.ID != session.ID {
				t.Errorf("Expected session ID '%s', got '%s'", session.ID, retrievedSession.ID)
			}

			t.Logf("Retrieved session status: %s", retrievedSession.Status)
		})
	})
}

func TestStripeIdentityProvider_ParseWebhook(t *testing.T) {
	config := StripeIdentityConfig{
		APIKey:  "mock",
		UseMock: true,
	}

	provider, _ := NewStripeIdentityProvider(config)

	tests := []struct {
		name        string
		payload     string
		expectError bool
		eventType   string
	}{
		{
			name: "verified event",
			payload: `{
				"id": "evt_test",
				"type": "identity.verification_session.verified",
				"created": 1234567890,
				"data": {
					"object": {
						"id": "vs_test123",
						"status": "verified"
					}
				}
			}`,
			expectError: false,
			eventType:   "identity.verification_session.verified",
		},
		{
			name: "requires_input event",
			payload: `{
				"id": "evt_test",
				"type": "identity.verification_session.requires_input",
				"created": 1234567890,
				"data": {
					"object": {
						"id": "vs_test123",
						"status": "requires_input",
						"last_error": {
							"code": "document_unverified_other"
						}
					}
				}
			}`,
			expectError: false,
			eventType:   "identity.verification_session.requires_input",
		},
		{
			name:        "invalid JSON",
			payload:     `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook, err := provider.ParseWebhook([]byte(tt.payload))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if webhook == nil {
				t.Fatal("Expected webhook, got nil")
			}

			if webhook.EventType != tt.eventType {
				t.Errorf("Expected event type '%s', got '%s'", tt.eventType, webhook.EventType)
			}
		})
	}
}

func TestStripeIdentityProvider_MissingAPIKey(t *testing.T) {
	config := StripeIdentityConfig{
		APIKey: "",
	}

	_, err := NewStripeIdentityProvider(config)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestStripeIdentityProvider_MockToggle(t *testing.T) {
	t.Run("UseMock=true", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey:  "sk_test_123",
			UseMock: true,
		}

		provider, _ := NewStripeIdentityProvider(config)
		if !provider.useMock {
			t.Error("Expected useMock to be true")
		}
	})

	t.Run("APIKey=mock", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey:  "mock",
			UseMock: false,
		}

		provider, _ := NewStripeIdentityProvider(config)
		if !provider.useMock {
			t.Error("Expected useMock to be true when APIKey='mock'")
		}
	})

	t.Run("Real API key", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey:  "sk_test_real_key",
			UseMock: false,
		}

		provider, _ := NewStripeIdentityProvider(config)
		if provider.useMock {
			t.Error("Expected useMock to be false with real API key")
		}
	})
}
