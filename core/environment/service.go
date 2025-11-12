package environment

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Service handles environment-related business logic
type Service struct {
	repo   Repository
	config Config
}

// Config holds the environment service configuration
type Config struct {
	AutoCreateDev                  bool     `json:"autoCreateDev"`
	DefaultDevName                 string   `json:"defaultDevName"`
	AllowPromotion                 bool     `json:"allowPromotion"`
	RequireConfirmationForDataCopy bool     `json:"requireConfirmationForDataCopy"`
	MaxEnvironmentsPerApp          int      `json:"maxEnvironmentsPerApp"`
	AllowedTypes                   []string `json:"allowedTypes"`
}

// NewService creates a new environment service
func NewService(repo Repository, config Config) *Service {
	// Set defaults
	if config.DefaultDevName == "" {
		config.DefaultDevName = "Development"
	}
	if config.MaxEnvironmentsPerApp == 0 {
		config.MaxEnvironmentsPerApp = 10
	}
	if len(config.AllowedTypes) == 0 {
		config.AllowedTypes = []string{
			schema.EnvironmentTypeDevelopment,
			schema.EnvironmentTypeProduction,
			schema.EnvironmentTypeStaging,
			schema.EnvironmentTypePreview,
			schema.EnvironmentTypeTest,
		}
	}

	return &Service{
		repo:   repo,
		config: config,
	}
}

// CreateEnvironment creates a new environment
func (s *Service) CreateEnvironment(ctx context.Context, req *CreateEnvironmentRequest) (*schema.Environment, error) {
	// Validate environment type
	if !s.isAllowedType(req.Type) {
		return nil, fmt.Errorf("environment type '%s' is not allowed", req.Type)
	}

	// Check environment limit
	count, err := s.repo.CountByApp(ctx, req.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to count environments: %w", err)
	}
	if count >= s.config.MaxEnvironmentsPerApp {
		return nil, fmt.Errorf("maximum environments per app reached (%d)", s.config.MaxEnvironmentsPerApp)
	}

	// Check if slug already exists for this app
	existing, err := s.repo.FindByAppAndSlug(ctx, req.AppID, req.Slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("environment with slug '%s' already exists", req.Slug)
	}

	// Create environment
	env := &schema.Environment{
		ID:        xid.New(),
		AppID:     req.AppID,
		Name:      req.Name,
		Slug:      req.Slug,
		Type:      req.Type,
		Status:    schema.EnvironmentStatusActive,
		Config:    req.Config,
		IsDefault: false, // Only set via CreateDefaultEnvironment
	}

	if err := s.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	return env, nil
}

// CreateDefaultEnvironment creates the default dev environment for an app
func (s *Service) CreateDefaultEnvironment(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	// Check if default environment already exists
	existing, err := s.repo.FindDefaultByApp(ctx, appID)
	if err == nil && existing != nil {
		return existing, nil // Already exists, return it
	}

	env := &schema.Environment{
		ID:        xid.New(),
		AppID:     appID,
		Name:      s.config.DefaultDevName,
		Slug:      "dev",
		Type:      schema.EnvironmentTypeDevelopment,
		Status:    schema.EnvironmentStatusActive,
		Config:    make(map[string]interface{}),
		IsDefault: true,
	}

	if err := s.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create default environment: %w", err)
	}

	return env, nil
}

// GetEnvironment retrieves an environment by ID
func (s *Service) GetEnvironment(ctx context.Context, id xid.ID) (*schema.Environment, error) {
	return s.repo.FindByID(ctx, id)
}

// GetEnvironmentBySlug retrieves an environment by app and slug
func (s *Service) GetEnvironmentBySlug(ctx context.Context, appID xid.ID, slug string) (*schema.Environment, error) {
	return s.repo.FindByAppAndSlug(ctx, appID, slug)
}

// GetDefaultEnvironment retrieves the default environment for an app
func (s *Service) GetDefaultEnvironment(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	return s.repo.FindDefaultByApp(ctx, appID)
}

// ListEnvironments lists environments for an app
func (s *Service) ListEnvironments(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.Environment, error) {
	return s.repo.ListByApp(ctx, appID, limit, offset)
}

// UpdateEnvironment updates an environment
func (s *Service) UpdateEnvironment(ctx context.Context, id xid.ID, req *UpdateEnvironmentRequest) (*schema.Environment, error) {
	env, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Cannot change default environment type
	if env.IsDefault && req.Type != nil && *req.Type != schema.EnvironmentTypeDevelopment {
		return nil, fmt.Errorf("cannot change default environment type")
	}

	// Update fields
	if req.Name != nil {
		env.Name = *req.Name
	}
	if req.Status != nil {
		env.Status = *req.Status
	}
	if req.Config != nil {
		env.Config = req.Config
	}
	env.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}

	return env, nil
}

// DeleteEnvironment deletes an environment
func (s *Service) DeleteEnvironment(ctx context.Context, id xid.ID) error {
	env, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Cannot delete default environment
	if env.IsDefault {
		return fmt.Errorf("cannot delete default environment")
	}

	// Cannot delete production without explicit confirmation
	if env.IsProduction() {
		return fmt.Errorf("cannot delete production environment without explicit confirmation")
	}

	return s.repo.Delete(ctx, id)
}

// PromoteEnvironment promotes/clones one environment to another
func (s *Service) PromoteEnvironment(ctx context.Context, req *PromoteEnvironmentRequest) (*schema.EnvironmentPromotion, error) {
	if !s.config.AllowPromotion {
		return nil, fmt.Errorf("environment promotion is disabled")
	}

	// Get source environment
	sourceEnv, err := s.repo.FindByID(ctx, req.SourceEnvID)
	if err != nil {
		return nil, fmt.Errorf("source environment not found: %w", err)
	}

	// Validate target environment details
	if !s.isAllowedType(req.TargetType) {
		return nil, fmt.Errorf("target environment type '%s' is not allowed", req.TargetType)
	}

	// Check if target environment already exists
	targetEnv, err := s.repo.FindByAppAndSlug(ctx, sourceEnv.AppID, req.TargetSlug)
	if err != nil && err.Error() != "environment not found" {
		return nil, fmt.Errorf("failed to check target environment: %w", err)
	}

	// If target doesn't exist, create it
	if targetEnv == nil {
		targetEnv = &schema.Environment{
			ID:        xid.New(),
			AppID:     sourceEnv.AppID,
			Name:      req.TargetName,
			Slug:      req.TargetSlug,
			Type:      req.TargetType,
			Status:    schema.EnvironmentStatusActive,
			Config:    make(map[string]interface{}),
			IsDefault: false,
		}

		if err := s.repo.Create(ctx, targetEnv); err != nil {
			return nil, fmt.Errorf("failed to create target environment: %w", err)
		}
	}

	// Create promotion record
	promotion := &schema.EnvironmentPromotion{
		ID:          xid.New(),
		AppID:       sourceEnv.AppID,
		SourceEnvID: sourceEnv.ID,
		TargetEnvID: targetEnv.ID,
		PromotedBy:  req.PromotedBy,
		IncludeData: req.IncludeData,
		Status:      schema.PromotionStatusPending,
	}

	if err := s.repo.CreatePromotion(ctx, promotion); err != nil {
		return nil, fmt.Errorf("failed to create promotion record: %w", err)
	}

	// Start promotion process (async in production)
	if err := s.executePromotion(ctx, promotion, sourceEnv, targetEnv); err != nil {
		promotion.Status = schema.PromotionStatusFailed
		promotion.ErrorMessage = err.Error()
		s.repo.UpdatePromotion(ctx, promotion)
		return nil, fmt.Errorf("promotion failed: %w", err)
	}

	// Mark as completed
	promotion.Status = schema.PromotionStatusCompleted
	now := time.Now()
	promotion.CompletedAt = &now
	s.repo.UpdatePromotion(ctx, promotion)

	return promotion, nil
}

// executePromotion performs the actual promotion/cloning
func (s *Service) executePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion, source, target *schema.Environment) error {
	// Update status to in progress
	promotion.Status = schema.PromotionStatusInProgress
	s.repo.UpdatePromotion(ctx, promotion)

	// Copy configuration from source to target
	target.Config = source.Config
	if err := s.repo.Update(ctx, target); err != nil {
		return fmt.Errorf("failed to update target environment config: %w", err)
	}

	// If includeData is true, copy organization data
	// NOTE: This is a simplified version. In production, this would be:
	// 1. Run in background job
	// 2. Copy organizations, members, teams, etc.
	// 3. Handle large datasets efficiently
	// 4. Provide progress updates
	if promotion.IncludeData {
		// TODO: Implement data copying logic
		// - Copy organizations
		// - Copy organization members
		// - Copy organization teams
		// - Copy organization invitations
		// - Update foreign keys to new environment_id
	}

	return nil
}

// GetPromotion retrieves a promotion by ID
func (s *Service) GetPromotion(ctx context.Context, id xid.ID) (*schema.EnvironmentPromotion, error) {
	return s.repo.FindPromotionByID(ctx, id)
}

// ListPromotions lists promotions for an app
func (s *Service) ListPromotions(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.EnvironmentPromotion, error) {
	return s.repo.ListPromotionsByApp(ctx, appID, limit, offset)
}

// Helper methods

func (s *Service) isAllowedType(envType string) bool {
	for _, allowed := range s.config.AllowedTypes {
		if envType == allowed {
			return true
		}
	}
	return false
}

// Request/Response types

// CreateEnvironmentRequest represents the request to create an environment
type CreateEnvironmentRequest struct {
	AppID  xid.ID                 `json:"appId"`
	Name   string                 `json:"name"`
	Slug   string                 `json:"slug"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateEnvironmentRequest represents the request to update an environment
type UpdateEnvironmentRequest struct {
	Name   *string                `json:"name,omitempty"`
	Status *string                `json:"status,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
	Type   *string                `json:"type,omitempty"`
}

// PromoteEnvironmentRequest represents the request to promote an environment
type PromoteEnvironmentRequest struct {
	SourceEnvID xid.ID                 `json:"sourceEnvId"`
	TargetName  string                 `json:"targetName"`
	TargetSlug  string                 `json:"targetSlug"`
	TargetType  string                 `json:"targetType"`
	IncludeData bool                   `json:"includeData"`
	PromotedBy  xid.ID                 `json:"promotedBy"`
	Config      map[string]interface{} `json:"config,omitempty"`
}
