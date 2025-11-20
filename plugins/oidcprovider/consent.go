package oidcprovider

import (
	"context"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// ConsentService handles OAuth2/OIDC consent management
type ConsentService struct {
	consentRepo *repo.OAuthConsentRepository
	clientRepo  *repo.OAuthClientRepository
}

// NewConsentService creates a new consent service
func NewConsentService(consentRepo *repo.OAuthConsentRepository, clientRepo *repo.OAuthClientRepository) *ConsentService {
	return &ConsentService{
		consentRepo: consentRepo,
		clientRepo:  clientRepo,
	}
}

// CheckConsent checks if user has already consented to the requested scopes for a client
func (s *ConsentService) CheckConsent(ctx context.Context, userID xid.ID, clientID string, requestedScopes []string, appID, envID xid.ID, orgID *xid.ID) (bool, error) {
	// Get client to check if consent is required
	client, err := s.clientRepo.FindByClientIDWithContext(ctx, appID, envID, orgID, clientID)
	if err != nil {
		return false, errs.DatabaseError("find client", err)
	}
	if client == nil {
		return false, errs.NotFound("client not found")
	}
	
	// Trusted clients skip consent
	if client.TrustedClient {
		return true, nil
	}
	
	// If client doesn't require consent, skip
	if !client.RequireConsent {
		return true, nil
	}
	
	// Check if user has existing valid consent
	hasConsent, err := s.consentRepo.HasValidConsent(ctx, userID, clientID, requestedScopes, appID, envID, orgID)
	if err != nil {
		return false, errs.DatabaseError("check consent", err)
	}
	
	return hasConsent, nil
}

// GrantConsent stores user's consent decision
func (s *ConsentService) GrantConsent(ctx context.Context, userID xid.ID, clientID string, scopes []string, appID, envID xid.ID, orgID *xid.ID, expiresIn *time.Duration) error {
	// Check if consent already exists
	existing, err := s.consentRepo.FindByUserAndClient(ctx, userID, clientID, appID, envID, orgID)
	if err != nil {
		return errs.DatabaseError("find existing consent", err)
	}
	
	if existing != nil {
		// Update existing consent
		existing.Scopes = scopes
		existing.UpdatedAt = time.Now()
		if expiresIn != nil {
			expiresAt := time.Now().Add(*expiresIn)
			existing.ExpiresAt = &expiresAt
		} else {
			existing.ExpiresAt = nil // Never expires
		}
		
		return s.consentRepo.Update(ctx, existing)
	}
	
	// Create new consent
	consent := &schema.OAuthConsent{
		ID:             xid.New(),
		AppID:          appID,
		EnvironmentID:  envID,
		OrganizationID: orgID,
		UserID:         userID,
		ClientID:       clientID,
		Scopes:         scopes,
	}
	
	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		consent.ExpiresAt = &expiresAt
	}
	
	return s.consentRepo.Create(ctx, consent)
}

// RevokeConsent removes a user's consent for a client
func (s *ConsentService) RevokeConsent(ctx context.Context, userID xid.ID, clientID string) error {
	return s.consentRepo.DeleteByUserAndClient(ctx, userID, clientID)
}

// ListUserConsents retrieves all consents granted by a user
func (s *ConsentService) ListUserConsents(ctx context.Context, userID xid.ID, appID, envID xid.ID, orgID *xid.ID) ([]*schema.OAuthConsent, error) {
	consents, err := s.consentRepo.ListByUser(ctx, userID, appID, envID, orgID)
	if err != nil {
		return nil, errs.DatabaseError("list consents", err)
	}
	return consents, nil
}

// ParseScopes converts a space-separated scope string to a slice
func (s *ConsentService) ParseScopes(scopeString string) []string {
	if scopeString == "" {
		return []string{}
	}
	return strings.Fields(scopeString)
}

// FormatScopes converts a scope slice to a space-separated string
func (s *ConsentService) FormatScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

// GetScopeDescriptions returns user-friendly descriptions for scopes
func (s *ConsentService) GetScopeDescriptions(scopes []string) []ScopeInfo {
	descriptions := map[string]string{
		"openid":         "Verify your identity",
		"profile":        "Access your basic profile information (name, username)",
		"email":          "Access your email address",
		"phone":          "Access your phone number",
		"address":        "Access your address information",
		"offline_access": "Keep you signed in and access your data when you're not using the app",
	}
	
	result := make([]ScopeInfo, 0, len(scopes))
	for _, scope := range scopes {
		description := descriptions[scope]
		if description == "" {
			description = "Access " + scope + " permissions"
		}
		result = append(result, ScopeInfo{
			Name:        scope,
			Description: description,
		})
	}
	
	return result
}

// RequiresConsent checks if the requested scopes require user consent
func (s *ConsentService) RequiresConsent(ctx context.Context, clientID string, scopes []string, appID, envID xid.ID, orgID *xid.ID) (bool, error) {
	// Get client configuration
	client, err := s.clientRepo.FindByClientIDWithContext(ctx, appID, envID, orgID, clientID)
	if err != nil {
		return false, errs.DatabaseError("find client", err)
	}
	if client == nil {
		return false, errs.NotFound("client not found")
	}
	
	// Trusted clients never require consent
	if client.TrustedClient {
		return false, nil
	}
	
	// Check client's consent requirement setting
	return client.RequireConsent, nil
}

