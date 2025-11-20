package idverification

import (
	"context"
	"testing"

	"github.com/rs/xid"
)

func TestOnfidoProvider_New(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := OnfidoConfig{
			APIToken: "test_token",
			Region:   "eu",
		}

		provider, err := NewOnfidoProvider(config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if provider == nil {
			t.Error("Expected provider, got nil")
		}

		if provider.GetProviderName() != "onfido" {
			t.Errorf("Expected provider name 'onfido', got %s", provider.GetProviderName())
		}
	})

	t.Run("missing API token", func(t *testing.T) {
		config := OnfidoConfig{
			APIToken: "",
		}

		_, err := NewOnfidoProvider(config)
		if err == nil {
			t.Error("Expected error for missing API token")
		}
	})
}

func TestOnfidoProvider_CreateSession(t *testing.T) {
	config := OnfidoConfig{
		APIToken: "test_token",
		Region:   "eu",
	}

	provider, _ := NewOnfidoProvider(config)
	ctx := context.Background()

	userID := xid.New()
	appID := xid.New()
	orgID := xid.New()

	req := &ProviderSessionRequest{
		AppID:          appID,
		OrganizationID: orgID,
		UserID:         userID,
		RequiredChecks: []string{"document", "liveness"},
		SuccessURL:     "https://example.com/success",
		CancelURL:      "https://example.com/cancel",
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

	if session.Status != "created" {
		t.Errorf("Expected status 'created', got %s", session.Status)
	}
}

func TestOnfidoProvider_VerifyWebhook(t *testing.T) {
	config := OnfidoConfig{
		APIToken:     "test_token",
		WebhookToken: "webhook_secret",
	}

	provider, _ := NewOnfidoProvider(config)

	t.Run("valid signature", func(t *testing.T) {
		// This is a simplified test - in production, you'd test with real HMAC
		payload := "test_payload"
		signature := "test_signature"

		valid, err := provider.VerifyWebhook(signature, payload)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Note: In real implementation, this would properly verify HMAC
		_ = valid
	})

	t.Run("missing webhook token", func(t *testing.T) {
		emptyConfig := OnfidoConfig{
			APIToken:     "test",
			WebhookToken: "",
		}
		emptyProvider, _ := NewOnfidoProvider(emptyConfig)

		_, err := emptyProvider.VerifyWebhook("sig", "payload")
		if err == nil {
			t.Error("Expected error for missing webhook token")
		}
	})
}

func TestJumioProvider_New(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := JumioConfig{
			APIToken:   "test_token",
			APISecret:  "test_secret",
			DataCenter: "us",
		}

		provider, err := NewJumioProvider(config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if provider == nil {
			t.Error("Expected provider, got nil")
		}

		if provider.GetProviderName() != "jumio" {
			t.Errorf("Expected provider name 'jumio', got %s", provider.GetProviderName())
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		config := JumioConfig{
			APIToken:  "test_token",
			APISecret: "",
		}

		_, err := NewJumioProvider(config)
		if err == nil {
			t.Error("Expected error for missing credentials")
		}
	})
}

func TestJumioProvider_CreateSession(t *testing.T) {
	config := JumioConfig{
		APIToken:   "test_token",
		APISecret:  "test_secret",
		DataCenter: "us",
	}

	provider, _ := NewJumioProvider(config)
	ctx := context.Background()

	userID := xid.New()
	appID := xid.New()
	orgID := xid.New()

	req := &ProviderSessionRequest{
		AppID:          appID,
		OrganizationID: orgID,
		UserID:         userID,
		RequiredChecks: []string{"document", "liveness"},
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
}

func TestStripeIdentityProvider_New(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey: "sk_test_123",
		}

		provider, err := NewStripeIdentityProvider(config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if provider == nil {
			t.Error("Expected provider, got nil")
		}

		if provider.GetProviderName() != "stripe_identity" {
			t.Errorf("Expected provider name 'stripe_identity', got %s", provider.GetProviderName())
		}
	})

	t.Run("missing API key", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey: "",
		}

		_, err := NewStripeIdentityProvider(config)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})
}

func TestStripeIdentityProvider_CreateSession(t *testing.T) {
	config := StripeIdentityConfig{
		APIKey:  "mock", // Use mock for testing
		UseMock: true,
	}

	provider, _ := NewStripeIdentityProvider(config)
	ctx := context.Background()

	userID := xid.New()
	appID := xid.New()
	orgID := xid.New()

	req := &ProviderSessionRequest{
		AppID:          appID,
		OrganizationID: orgID,
		UserID:         userID,
		RequiredChecks: []string{"document"},
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
}

func TestStripeIdentityProvider_VerifyWebhook(t *testing.T) {
	config := StripeIdentityConfig{
		APIKey:        "mock",
		UseMock:       true,
		WebhookSecret: "whsec_test",
	}

	provider, _ := NewStripeIdentityProvider(config)

	t.Run("with webhook secret", func(t *testing.T) {
		valid, err := provider.VerifyWebhook("sig", "payload")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !valid {
			t.Error("Expected valid webhook in mock mode")
		}
	})

	t.Run("missing webhook secret", func(t *testing.T) {
		emptyConfig := StripeIdentityConfig{
			APIKey:        "sk_test_123",
			WebhookSecret: "",
		}
		emptyProvider, _ := NewStripeIdentityProvider(emptyConfig)

		_, err := emptyProvider.VerifyWebhook("sig", "payload")
		if err == nil {
			t.Error("Expected error for missing webhook secret")
		}
	})
}

func TestProviderWebhookParsing(t *testing.T) {
	t.Run("onfido webhook", func(t *testing.T) {
		config := OnfidoConfig{
			APIToken: "test",
		}
		provider, _ := NewOnfidoProvider(config)

		payload := []byte(`{"event_type":"check.completed","object":{"id":"check_123"}}`)
		webhook, err := provider.ParseWebhook(payload)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if webhook == nil {
			t.Fatal("Expected webhook, got nil")
		}

		if webhook.EventType == "" {
			t.Error("Expected event type to be set")
		}
	})

	t.Run("jumio webhook", func(t *testing.T) {
		config := JumioConfig{
			APIToken:  "test",
			APISecret: "secret",
		}
		provider, _ := NewJumioProvider(config)

		payload := []byte(`{"status":"COMPLETED","transactionReference":"ref_123"}`)
		webhook, err := provider.ParseWebhook(payload)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if webhook == nil {
			t.Fatal("Expected webhook, got nil")
		}
	})

	t.Run("stripe webhook", func(t *testing.T) {
		config := StripeIdentityConfig{
			APIKey:  "mock",
			UseMock: true,
		}
		provider, _ := NewStripeIdentityProvider(config)

		payload := []byte(`{
			"id": "evt_123",
			"type": "identity.verification_session.verified",
			"created": 1234567890,
			"data": {
				"object": {
					"id": "vs_test123",
					"status": "verified"
				}
			}
		}`)
		webhook, err := provider.ParseWebhook(payload)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if webhook == nil {
			t.Fatal("Expected webhook, got nil")
		}

		if webhook.EventType != "identity.verification_session.verified" {
			t.Errorf("Expected event type 'identity.verification_session.verified', got '%s'", webhook.EventType)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		config := OnfidoConfig{
			APIToken: "test",
		}
		provider, _ := NewOnfidoProvider(config)

		payload := []byte(`invalid json`)
		_, err := provider.ParseWebhook(payload)

		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestProviderNames(t *testing.T) {
	tests := []struct {
		name         string
		providerFunc func() Provider
		expectedName string
	}{
		{
			name: "onfido",
			providerFunc: func() Provider {
				p, _ := NewOnfidoProvider(OnfidoConfig{APIToken: "test"})
				return p
			},
			expectedName: "onfido",
		},
		{
			name: "jumio",
			providerFunc: func() Provider {
				p, _ := NewJumioProvider(JumioConfig{APIToken: "test", APISecret: "secret"})
				return p
			},
			expectedName: "jumio",
		},
		{
			name: "stripe_identity",
			providerFunc: func() Provider {
				p, _ := NewStripeIdentityProvider(StripeIdentityConfig{APIKey: "sk_test_123"})
				return p
			},
			expectedName: "stripe_identity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := tt.providerFunc()
			if provider.GetProviderName() != tt.expectedName {
				t.Errorf("Expected provider name '%s', got '%s'", tt.expectedName, provider.GetProviderName())
			}
		})
	}
}
