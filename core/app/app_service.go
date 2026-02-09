package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
)

// AppService handles app aggregate operations.
type AppService struct {
	repo               AppRepository
	config             Config
	rbacSvc            *rbac.Service
	hookRegistry       *hooks.HookRegistry
	globalCookieConfig *session.CookieConfig // Global cookie configuration
}

// NewAppService creates a new app service.
func NewAppService(repo AppRepository, cfg Config, rbacSvc *rbac.Service) *AppService {
	return &AppService{
		repo:    repo,
		config:  cfg,
		rbacSvc: rbacSvc,
	}
}

// SetGlobalCookieConfig sets the global cookie configuration
// This is called during Auth initialization to provide the global default.
func (s *AppService) SetGlobalCookieConfig(config *session.CookieConfig) {
	s.globalCookieConfig = config
}

// SetHookRegistry sets the hook registry for executing hooks.
func (s *AppService) SetHookRegistry(hookRegistry *hooks.HookRegistry) {
	s.hookRegistry = hookRegistry
}

// UpdateConfig updates the app service configuration.
func (s *AppService) UpdateConfig(cfg Config) {
	s.config = cfg
}

// CreateApp creates a new app.
func (s *AppService) CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error) {
	id := xid.New()
	now := time.Now().UTC()

	app := &App{
		ID:        id,
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateApp(ctx, app.ToSchema()); err != nil {
		return nil, err
	}

	// Execute after app create hooks
	if s.hookRegistry != nil {
		if err := s.hookRegistry.ExecuteAfterAppCreate(ctx, app.ToSchema()); err != nil {
			// Log error but don't fail app creation
			_ = err
		}
	}

	return app, nil
}

// GetPlatformApp retrieves the platform-level app entity.
func (s *AppService) GetPlatformApp(ctx context.Context) (*App, error) {
	schemaApp, err := s.repo.GetPlatformApp(ctx)
	if err != nil {
		return nil, err
	}

	return FromSchemaApp(schemaApp), nil
}

// FindAppByID returns an app by ID.
func (s *AppService) FindAppByID(ctx context.Context, id xid.ID) (*App, error) {
	schemaApp, err := s.repo.FindAppByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return FromSchemaApp(schemaApp), nil
}

// FindAppBySlug returns an app by slug.
func (s *AppService) FindAppBySlug(ctx context.Context, slug string) (*App, error) {
	schemaApp, err := s.repo.FindAppBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return FromSchemaApp(schemaApp), nil
}

// UpdateApp updates an app.
func (s *AppService) UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error) {
	schemaApp, err := s.repo.FindAppByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		schemaApp.Name = *req.Name
	}

	if req.Logo != nil {
		schemaApp.Logo = *req.Logo
	}

	if req.Metadata != nil {
		schemaApp.Metadata = req.Metadata
	}

	schemaApp.UpdatedAt = time.Now()

	if err := s.repo.UpdateApp(ctx, schemaApp); err != nil {
		return nil, err
	}

	return FromSchemaApp(schemaApp), nil
}

// DeleteApp deletes an app by ID
// Platform app cannot be deleted unless another app is made platform first.
func (s *AppService) DeleteApp(ctx context.Context, id xid.ID) error {
	// Check if the app exists and is platform app
	app, err := s.repo.FindAppByID(ctx, id)
	if err != nil {
		return AppNotFound().WithError(err)
	}

	// Prevent deletion of platform app
	if app.IsPlatform {
		return CannotDeletePlatformApp()
	}

	return s.repo.DeleteApp(ctx, id)
}

// SetPlatformApp transfers platform status to the specified app
// Only one app can be platform at a time.
func (s *AppService) SetPlatformApp(ctx context.Context, newPlatformAppID xid.ID) error {
	// Get the target app
	newPlatformApp, err := s.repo.FindAppByID(ctx, newPlatformAppID)
	if err != nil {
		return AppNotFound().WithError(err)
	}

	// If already platform, nothing to do
	if newPlatformApp.IsPlatform {
		return nil
	}

	// Get current platform app
	currentPlatformApp, err := s.repo.GetPlatformApp(ctx)
	if err != nil {
		// No platform app exists yet, this is fine
		currentPlatformApp = nil
	}

	// Unset old platform app (if exists)
	if currentPlatformApp != nil && currentPlatformApp.ID != newPlatformAppID {
		currentPlatformApp.IsPlatform = false

		currentPlatformApp.UpdatedAt = time.Now()
		if err := s.repo.UpdateApp(ctx, currentPlatformApp); err != nil {
			return fmt.Errorf("failed to unset old platform app: %w", err)
		}
	}

	// Set new platform app
	newPlatformApp.IsPlatform = true

	newPlatformApp.UpdatedAt = time.Now()
	if err := s.repo.UpdateApp(ctx, newPlatformApp); err != nil {
		return fmt.Errorf("failed to set new platform app: %w", err)
	}

	return nil
}

// IsPlatformApp checks if the specified app is the platform app.
func (s *AppService) IsPlatformApp(ctx context.Context, appID xid.ID) (bool, error) {
	app, err := s.repo.FindAppByID(ctx, appID)
	if err != nil {
		return false, err
	}

	return app.IsPlatform, nil
}

// ListApps returns a paginated list of apps.
func (s *AppService) ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*App], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListApps(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	// Convert schema apps to DTOs
	apps := FromSchemaApps(response.Data)

	return &pagination.PageResponse[*App]{
		Data:       apps,
		Pagination: response.Pagination,
	}, nil
}

// CountApps returns total number of apps.
func (s *AppService) CountApps(ctx context.Context) (int, error) {
	return s.repo.CountApps(ctx)
}

// GetCookieConfig retrieves the cookie configuration for a specific app
// It merges app-specific overrides from metadata with the global configuration.
func (s *AppService) GetCookieConfig(ctx context.Context, appID xid.ID) (*session.CookieConfig, error) {
	// Start with global config as base
	var baseConfig session.CookieConfig
	if s.globalCookieConfig != nil {
		baseConfig = *s.globalCookieConfig
	} else {
		// Use defaults if no global config is set
		baseConfig = session.DefaultCookieConfig()
	}

	// If no app ID provided, return global config
	if appID.IsNil() {
		return &baseConfig, nil
	}

	// Fetch app from database
	app, err := s.repo.FindAppByID(ctx, appID)
	if err != nil {
		// If app not found, return global config
		return &baseConfig, nil
	}

	// Check if app has cookie config override in metadata
	if app.Metadata == nil {
		return &baseConfig, nil
	}

	cookieConfigData, exists := app.Metadata["sessionCookie"]
	if !exists {
		return &baseConfig, nil
	}

	// Parse the cookie config from metadata
	var appCookieConfig session.CookieConfig

	// Handle different possible types in metadata
	switch v := cookieConfigData.(type) {
	case map[string]any:
		// Convert to JSON and unmarshal
		jsonData, err := json.Marshal(v)
		if err != nil {
			// Invalid format, return base config
			return &baseConfig, nil
		}

		if err := json.Unmarshal(jsonData, &appCookieConfig); err != nil {
			// Invalid format, return base config
			return &baseConfig, nil
		}
	case string:
		// Try to parse as JSON string
		if err := json.Unmarshal([]byte(v), &appCookieConfig); err != nil {
			// Invalid format, return base config
			return &baseConfig, nil
		}
	default:
		// Unknown type, return base config
		return &baseConfig, nil
	}

	// Merge app config with base config
	mergedConfig := baseConfig.Merge(&appCookieConfig)

	return mergedConfig, nil
}

// Type assertion to ensure AppService implements AppOperations.
var _ AppOperations = (*AppService)(nil)
