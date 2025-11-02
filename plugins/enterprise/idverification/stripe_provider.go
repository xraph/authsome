package idverification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/identity/verificationsession"
	"github.com/stripe/stripe-go/v79/webhook"
)

// StripeIdentityProvider implements the Provider interface for Stripe Identity
type StripeIdentityProvider struct {
	config  StripeIdentityConfig
	useMock bool // Toggle between mock and real implementation
}

// NewStripeIdentityProvider creates a new Stripe Identity provider
func NewStripeIdentityProvider(config StripeIdentityConfig) (*StripeIdentityProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Stripe API key is required")
	}

	// Set Stripe API key
	stripe.Key = config.APIKey

	// Determine if we should use mock based on API key prefix
	useMock := false
	if config.UseMock || config.APIKey == "mock" || config.APIKey == "test_mock" {
		useMock = true
	}

	return &StripeIdentityProvider{
		config:  config,
		useMock: useMock,
	}, nil
}

// CreateSession creates a Stripe Identity verification session
func (p *StripeIdentityProvider) CreateSession(ctx context.Context, req *ProviderSessionRequest) (*ProviderSession, error) {
	// Use mock for testing/development
	if p.useMock {
		return p.createMockSession(req)
	}

	// Real Stripe API implementation
	params := &stripe.IdentityVerificationSessionParams{
		Type: stripe.String("document"),
	}

	// Configure document verification options
	params.Options = &stripe.IdentityVerificationSessionOptionsParams{
		Document: &stripe.IdentityVerificationSessionOptionsDocumentParams{
			RequireLiveCapture:   stripe.Bool(p.config.RequireLiveCapture),
			RequireMatchingSelfie: stripe.Bool(p.config.RequireMatchingSelfie),
		},
	}

	// Set allowed document types if configured
	if len(p.config.AllowedTypes) > 0 {
		allowedTypes := make([]*string, len(p.config.AllowedTypes))
		for i, t := range p.config.AllowedTypes {
			allowedTypes[i] = stripe.String(t)
		}
		params.Options.Document.AllowedTypes = allowedTypes
	}

	// Set return URL if provided
	if req.SuccessURL != "" {
		params.ReturnURL = stripe.String(req.SuccessURL)
	}

	// Set metadata
	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}
	// Add standard metadata
	params.AddMetadata("user_id", req.UserID)
	params.AddMetadata("organization_id", req.OrganizationID)

	// Create verification session via Stripe API
	session, err := verificationsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe verification session: %w", err)
	}

	// Convert Stripe session to our ProviderSession format
	// Note: Stripe sessions don't have explicit expiry, they expire after 24 hours
	expiresAt := time.Unix(session.Created, 0).Add(24 * time.Hour)
	
	return &ProviderSession{
		ID:        session.ID,
		URL:       session.URL,
		Token:     session.ClientSecret,
		Status:    string(session.Status),
		ExpiresAt: expiresAt,
		CreatedAt: time.Unix(session.Created, 0),
	}, nil
}

// GetSession retrieves a Stripe Identity verification session status
func (p *StripeIdentityProvider) GetSession(ctx context.Context, sessionID string) (*ProviderSession, error) {
	// Use mock for testing/development
	if p.useMock {
		return p.getMockSession(sessionID)
	}

	// Real Stripe API implementation
	session, err := verificationsession.Get(sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe verification session: %w", err)
	}

	expiresAt := time.Unix(session.Created, 0).Add(24 * time.Hour)
	
	return &ProviderSession{
		ID:        session.ID,
		URL:       session.URL,
		Token:     session.ClientSecret,
		Status:    string(session.Status),
		ExpiresAt: expiresAt,
		CreatedAt: time.Unix(session.Created, 0),
	}, nil
}

// GetCheck retrieves a Stripe Identity verification result
func (p *StripeIdentityProvider) GetCheck(ctx context.Context, sessionID string) (*ProviderCheckResult, error) {
	// Use mock for testing/development
	if p.useMock {
		return p.getMockCheck(sessionID)
	}

	// Real Stripe API implementation
	session, err := verificationsession.Get(sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe verification session: %w", err)
	}

	result := &ProviderCheckResult{
		ID:     session.ID,
		Type:   "document",
		Status: string(session.Status),
		Result: string(session.Status), // verified, requires_input, processing, canceled
	}

	// Parse verified outputs if available
	if session.VerifiedOutputs != nil {
		outputs := session.VerifiedOutputs

		// Document information
		result.IsDocumentValid = session.Status == "verified"
		if outputs.IDNumber != "" {
			result.DocumentNumber = outputs.IDNumber
		}

		// Date of birth
		if outputs.DOB != nil {
			dob := time.Date(
				int(outputs.DOB.Year),
				time.Month(outputs.DOB.Month),
				int(outputs.DOB.Day),
				0, 0, 0, 0, time.UTC,
			)
			result.DateOfBirth = &dob
		}

		// Address information
		if outputs.Address != nil {
			result.DocumentCountry = outputs.Address.Country
		}

		// Name information
		if outputs.FirstName != "" {
			result.FirstName = outputs.FirstName
		}
		if outputs.LastName != "" {
			result.LastName = outputs.LastName
		}
	}

	// Parse last verification error if present
	if session.LastError != nil {
		// Store error code in Properties for consistency with other providers
		if result.Properties == nil {
			result.Properties = make(map[string]interface{})
		}
		result.Properties["error_code"] = string(session.LastError.Code)
	}

	// Determine document type from session data
	if session.Type == "document" {
		result.DocumentType = "identity_document"
	}

	// Liveness check (Stripe automatically does this with RequireLiveCapture)
	if p.config.RequireLiveCapture {
		result.IsLive = session.Status == "verified"
	}

	// Set risk/confidence scores (Stripe doesn't provide these, use defaults)
	if session.Status == "verified" {
		result.RiskScore = 10 // Low risk
		result.ConfidenceScore = 95 // High confidence
	}

	return result, nil
}

// VerifyWebhook verifies a Stripe webhook signature
func (p *StripeIdentityProvider) VerifyWebhook(signature, payload string) (bool, error) {
	if p.config.WebhookSecret == "" {
		return false, fmt.Errorf("webhook secret not configured")
	}

	// Use mock for testing
	if p.useMock {
		return true, nil
	}

	// Stripe provides webhook verification in their SDK
	// We'll verify the signature using Stripe's method
	_, err := webhook.ConstructEvent(
		[]byte(payload),
		signature,
		p.config.WebhookSecret,
	)

	if err != nil {
		return false, fmt.Errorf("webhook verification failed: %w", err)
	}

	return true, nil
}

// ParseWebhook parses a Stripe webhook payload
func (p *StripeIdentityProvider) ParseWebhook(payload []byte) (*WebhookPayload, error) {
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Convert RawMessage to map for RawPayload
	var rawData map[string]interface{}
	if err := json.Unmarshal(event.Data.Raw, &rawData); err != nil {
		rawData = make(map[string]interface{})
	}

	result := &WebhookPayload{
		EventType:  string(event.Type),
		Timestamp:  time.Unix(event.Created, 0),
		RawPayload: rawData,
	}

	// Parse specific event types
	switch event.Type {
	case "identity.verification_session.verified":
		var session stripe.IdentityVerificationSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, fmt.Errorf("failed to parse session: %w", err)
		}
		result.Status = string(session.Status)
		// Store session ID in metadata
		if result.RawPayload == nil {
			result.RawPayload = make(map[string]interface{})
		}
		result.RawPayload["session_id"] = session.ID

	case "identity.verification_session.requires_input":
		var session stripe.IdentityVerificationSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, fmt.Errorf("failed to parse session: %w", err)
		}
		result.Status = string(session.Status)
		if result.RawPayload == nil {
			result.RawPayload = make(map[string]interface{})
		}
		result.RawPayload["session_id"] = session.ID
		if session.LastError != nil {
			result.RawPayload["error_code"] = string(session.LastError.Code)
		}

	case "identity.verification_session.canceled":
		var session stripe.IdentityVerificationSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, fmt.Errorf("failed to parse session: %w", err)
		}
		result.Status = "canceled"
		if result.RawPayload == nil {
			result.RawPayload = make(map[string]interface{})
		}
		result.RawPayload["session_id"] = session.ID

	case "identity.verification_session.processing":
		var session stripe.IdentityVerificationSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, fmt.Errorf("failed to parse session: %w", err)
		}
		result.Status = "processing"
		if result.RawPayload == nil {
			result.RawPayload = make(map[string]interface{})
		}
		result.RawPayload["session_id"] = session.ID
	}

	return result, nil
}

// GetProviderName returns the provider name
func (p *StripeIdentityProvider) GetProviderName() string {
	return "stripe_identity"
}

// Mock implementations for testing/development

func (p *StripeIdentityProvider) createMockSession(req *ProviderSessionRequest) (*ProviderSession, error) {
	return &ProviderSession{
		ID:        fmt.Sprintf("vs_mock_%d", time.Now().Unix()),
		URL:       fmt.Sprintf("https://verify.stripe.com/start/mock_%d", time.Now().Unix()),
		Token:     fmt.Sprintf("vs_mock_secret_%d", time.Now().Unix()),
		Status:    "requires_input",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

func (p *StripeIdentityProvider) getMockSession(sessionID string) (*ProviderSession, error) {
	return &ProviderSession{
		ID:        sessionID,
		URL:       "https://verify.stripe.com/start/" + sessionID,
		Token:     sessionID + "_secret",
		Status:    "verified",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}, nil
}

func (p *StripeIdentityProvider) getMockCheck(sessionID string) (*ProviderCheckResult, error) {
	dob := time.Now().AddDate(-25, 0, 0)

	return &ProviderCheckResult{
		ID:               sessionID,
		Type:             "document",
		Status:           "verified",
		Result:           "clear",
		IsDocumentValid:  true,
		IsLive:           true,
		DocumentType:     "passport",
		DocumentNumber:   "MOCK123456",
		DocumentCountry:  "US",
		FirstName:        "John",
		LastName:         "Doe",
		DateOfBirth:      &dob,
		RiskScore:        10,
		ConfidenceScore:  95,
	}, nil
}

