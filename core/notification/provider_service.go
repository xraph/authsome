package notification

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/notification/crypto"
	"github.com/xraph/authsome/schema"
)

// ProviderService handles provider management operations.
type ProviderService struct {
	repo Repository
}

// NewProviderService creates a new provider service.
func NewProviderService(repo Repository) *ProviderService {
	return &ProviderService{repo: repo}
}

// CreateProvider creates a new notification provider.
func (s *ProviderService) CreateProvider(ctx context.Context, appID xid.ID, orgID *xid.ID, providerType, providerName string, config map[string]any, isDefault bool) (*schema.NotificationProvider, error) {
	// Encrypt sensitive configuration fields
	encryptedConfig, err := crypto.EncryptConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt config: %w", err)
	}

	provider := &schema.NotificationProvider{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: orgID,
		ProviderType:   providerType,
		ProviderName:   providerName,
		Config:         encryptedConfig,
		IsActive:       true,
		IsDefault:      isDefault,
	}

	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

// GetProvider retrieves a provider by ID.
func (s *ProviderService) GetProvider(ctx context.Context, id xid.ID) (*schema.NotificationProvider, error) {
	provider, err := s.repo.FindProviderByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	if provider == nil {
		return nil, ProviderNotFound("unknown")
	}

	// Decrypt sensitive configuration fields
	decryptedConfig, err := crypto.DecryptConfig(provider.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	provider.Config = decryptedConfig

	return provider, nil
}

// ListProviders lists all providers for an app/org.
func (s *ProviderService) ListProviders(ctx context.Context, appID xid.ID, orgID *xid.ID) ([]*schema.NotificationProvider, error) {
	providers, err := s.repo.ListProviders(ctx, appID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	return providers, nil
}

// UpdateProvider updates a provider's configuration.
func (s *ProviderService) UpdateProvider(ctx context.Context, id xid.ID, config map[string]any, isActive, isDefault bool) error {
	// Encrypt sensitive configuration fields
	encryptedConfig, err := crypto.EncryptConfig(config)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	if err := s.repo.UpdateProvider(ctx, id, encryptedConfig, isActive, isDefault); err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	return nil
}

// DeleteProvider deletes a provider.
func (s *ProviderService) DeleteProvider(ctx context.Context, id xid.ID) error {
	if err := s.repo.DeleteProvider(ctx, id); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	return nil
}

// ResolveProvider resolves the best provider for a given app/org and type
// Priority: org-specific default > app-level default.
func (s *ProviderService) ResolveProvider(ctx context.Context, appID xid.ID, orgID *xid.ID, providerType string) (*schema.NotificationProvider, error) {
	// Try org-specific provider first if orgID is provided
	if orgID != nil {
		provider, err := s.repo.FindProviderByTypeOrgScoped(ctx, appID, orgID, providerType)
		if err == nil && provider != nil {
			// Decrypt config before returning
			decryptedConfig, err := crypto.DecryptConfig(provider.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt config: %w", err)
			}

			provider.Config = decryptedConfig

			return provider, nil
		}
	}

	// Fall back to app-level provider
	provider, err := s.repo.FindProviderByTypeOrgScoped(ctx, appID, nil, providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}

	if provider == nil {
		return nil, ProviderNotFound(providerType)
	}

	// Decrypt config before returning
	decryptedConfig, err := crypto.DecryptConfig(provider.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	provider.Config = decryptedConfig

	return provider, nil
}
