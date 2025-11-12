package environment

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// AppRepository defines minimal interface needed for bootstrap
type AppRepository interface {
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, app *schema.App) error
	FindBySlug(ctx context.Context, slug string) (*schema.App, error)
}

// BootstrapConfig holds configuration for bootstrap process
type BootstrapConfig struct {
	DefaultAppName       string `json:"defaultAppName"`
	DefaultAppSlug       string `json:"defaultAppSlug"`
	AutoCreateDefaultApp bool   `json:"autoCreateDefaultApp"`
	MultitenancyEnabled  bool   `json:"multitenancyEnabled"`
}

// Bootstrap handles initial app and environment setup
type Bootstrap struct {
	appRepo AppRepository
	envRepo Repository
	config  BootstrapConfig
}

// NewBootstrap creates a new bootstrap instance
func NewBootstrap(appRepo AppRepository, envRepo Repository, config BootstrapConfig) *Bootstrap {
	// Set defaults
	if config.DefaultAppName == "" {
		config.DefaultAppName = "Platform App"
	}
	if config.DefaultAppSlug == "" {
		config.DefaultAppSlug = "platform"
	}

	return &Bootstrap{
		appRepo: appRepo,
		envRepo: envRepo,
		config:  config,
	}
}

// EnsureDefaultApp ensures the default app exists (non-multitenancy mode)
func (b *Bootstrap) EnsureDefaultApp(ctx context.Context) (*schema.App, *schema.Environment, error) {
	// Check if auto-create is disabled
	if !b.config.AutoCreateDefaultApp {
		return nil, nil, nil
	}

	// Check how many apps exist
	count, err := b.appRepo.Count(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count apps: %w", err)
	}

	// In non-multitenancy mode, ensure only one app exists
	if !b.config.MultitenancyEnabled && count > 1 {
		return nil, nil, fmt.Errorf("multiple apps detected in non-multitenancy mode")
	}

	// If app already exists, return it
	if count > 0 {
		existingApp, err := b.appRepo.FindBySlug(ctx, b.config.DefaultAppSlug)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find default app: %w", err)
		}

		// Ensure it has a default environment
		env, err := b.ensureDefaultEnvironment(ctx, existingApp.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to ensure default environment: %w", err)
		}

		return existingApp, env, nil
	}

	// Create default app
	app := &schema.App{
		ID:         xid.New(),
		Name:       b.config.DefaultAppName,
		Slug:       b.config.DefaultAppSlug,
		IsPlatform: true, // This is the platform app
		Logo:       "",
		Metadata:   make(map[string]interface{}),
	}

	if err := b.appRepo.Create(ctx, app); err != nil {
		return nil, nil, fmt.Errorf("failed to create default app: %w", err)
	}

	// Create default dev environment
	env, err := b.createDefaultEnvironment(ctx, app.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create default environment: %w", err)
	}

	return app, env, nil
}

// EnsureAppEnvironment ensures an app has a default environment
func (b *Bootstrap) EnsureAppEnvironment(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	return b.ensureDefaultEnvironment(ctx, appID)
}

// ensureDefaultEnvironment checks and creates default environment if needed
func (b *Bootstrap) ensureDefaultEnvironment(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	// Check if default environment exists
	env, err := b.envRepo.FindDefaultByApp(ctx, appID)
	if err == nil && env != nil {
		return env, nil // Already exists
	}

	// Create it
	return b.createDefaultEnvironment(ctx, appID)
}

// createDefaultEnvironment creates the default dev environment
func (b *Bootstrap) createDefaultEnvironment(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	env := &schema.Environment{
		ID:        xid.New(),
		AppID:     appID,
		Name:      "Development",
		Slug:      "dev",
		Type:      schema.EnvironmentTypeDevelopment,
		Status:    schema.EnvironmentStatusActive,
		Config:    make(map[string]interface{}),
		IsDefault: true,
	}

	if err := b.envRepo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create dev environment: %w", err)
	}

	return env, nil
}

// ValidateMultitenancyMode ensures app count respects multitenancy setting
func (b *Bootstrap) ValidateMultitenancyMode(ctx context.Context) error {
	count, err := b.appRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count apps: %w", err)
	}

	// In non-multitenancy mode, only 1 app should exist
	if !b.config.MultitenancyEnabled && count > 1 {
		return fmt.Errorf("non-multitenancy mode enabled but multiple apps detected (%d apps)", count)
	}

	return nil
}
