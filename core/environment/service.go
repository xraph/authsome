package environment

import (
	"context"
	"slices"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Service handles environment-related business logic.
type Service struct {
	repo   Repository
	config Config
}

// Config holds the environment service configuration.
type Config struct {
	AutoCreateDev                  bool     `json:"autoCreateDev"`
	DefaultDevName                 string   `json:"defaultDevName"`
	AllowPromotion                 bool     `json:"allowPromotion"`
	RequireConfirmationForDataCopy bool     `json:"requireConfirmationForDataCopy"`
	MaxEnvironmentsPerApp          int      `json:"maxEnvironmentsPerApp"`
	AllowedTypes                   []string `json:"allowedTypes"`
}

// NewService creates a new environment service.
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

// CreateEnvironment creates a new environment.
func (s *Service) CreateEnvironment(ctx context.Context, req *CreateEnvironmentRequest) (*Environment, error) {
	// Validate environment type
	if !s.isAllowedType(req.Type) {
		return nil, EnvironmentTypeForbidden(req.Type)
	}

	// Check environment limit
	count, err := s.repo.CountByApp(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if count >= s.config.MaxEnvironmentsPerApp {
		return nil, EnvironmentLimitReached(s.config.MaxEnvironmentsPerApp)
	}

	// Check if slug already exists for this app
	existing, err := s.repo.FindByAppAndSlug(ctx, req.AppID, req.Slug)
	if err == nil && existing != nil {
		return nil, EnvironmentSlugAlreadyExists(req.Slug)
	}

	// Create environment DTO
	now := time.Now().UTC()
	env := &Environment{
		ID:        xid.New(),
		AppID:     req.AppID,
		Name:      req.Name,
		Slug:      req.Slug,
		Type:      req.Type,
		Status:    schema.EnvironmentStatusActive,
		Config:    req.Config,
		IsDefault: false, // Only set via CreateDefaultEnvironment
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to database using schema conversion
	if err := s.repo.Create(ctx, env.ToSchema()); err != nil {
		return nil, err
	}

	return env, nil
}

// CreateDefaultEnvironment creates the default dev environment for an app.
func (s *Service) CreateDefaultEnvironment(ctx context.Context, appID xid.ID) (*Environment, error) {
	// Check if default environment already exists
	existing, err := s.repo.FindDefaultByApp(ctx, appID)
	if err == nil && existing != nil {
		return FromSchemaEnvironment(existing), nil // Already exists, return it
	}

	now := time.Now().UTC()
	env := &Environment{
		ID:        xid.New(),
		AppID:     appID,
		Name:      s.config.DefaultDevName,
		Slug:      "dev",
		Type:      schema.EnvironmentTypeDevelopment,
		Status:    schema.EnvironmentStatusActive,
		Config:    make(map[string]any),
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, env.ToSchema()); err != nil {
		return nil, err
	}

	return env, nil
}

// GetEnvironment retrieves an environment by ID.
func (s *Service) GetEnvironment(ctx context.Context, id xid.ID) (*Environment, error) {
	schemaEnv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, EnvironmentNotFound(id.String())
	}

	return FromSchemaEnvironment(schemaEnv), nil
}

// GetEnvironmentBySlug retrieves an environment by app and slug.
func (s *Service) GetEnvironmentBySlug(ctx context.Context, appID xid.ID, slug string) (*Environment, error) {
	schemaEnv, err := s.repo.FindByAppAndSlug(ctx, appID, slug)
	if err != nil {
		return nil, EnvironmentNotFound(slug)
	}

	return FromSchemaEnvironment(schemaEnv), nil
}

// GetDefaultEnvironment retrieves the default environment for an app.
func (s *Service) GetDefaultEnvironment(ctx context.Context, appID xid.ID) (*Environment, error) {
	schemaEnv, err := s.repo.FindDefaultByApp(ctx, appID)
	if err != nil {
		return nil, DefaultEnvironmentNotFound(appID.String())
	}

	return FromSchemaEnvironment(schemaEnv), nil
}

// ListEnvironments lists environments for an app with pagination.
func (s *Service) ListEnvironments(ctx context.Context, filter *ListEnvironmentsFilter) (*ListEnvironmentsResponse, error) {
	// Get paginated results from repository
	pageResp, err := s.repo.ListEnvironments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema environments to DTOs
	dtoEnvs := FromSchemaEnvironments(pageResp.Data)

	// Return paginated response with DTOs
	return &pagination.PageResponse[*Environment]{
		Data:       dtoEnvs,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
}

// UpdateEnvironment updates an environment.
func (s *Service) UpdateEnvironment(ctx context.Context, id xid.ID, req *UpdateEnvironmentRequest) (*Environment, error) {
	schemaEnv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, EnvironmentNotFound(id.String())
	}

	// Cannot change default environment type
	if schemaEnv.IsDefault && req.Type != nil && *req.Type != schema.EnvironmentTypeDevelopment {
		return nil, CannotModifyDefaultEnvironmentType()
	}

	// Update fields
	if req.Name != nil {
		schemaEnv.Name = *req.Name
	}

	if req.Status != nil {
		schemaEnv.Status = *req.Status
	}

	if req.Config != nil {
		schemaEnv.Config = req.Config
	}

	schemaEnv.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, schemaEnv); err != nil {
		return nil, err
	}

	return FromSchemaEnvironment(schemaEnv), nil
}

// DeleteEnvironment deletes an environment.
func (s *Service) DeleteEnvironment(ctx context.Context, id xid.ID) error {
	schemaEnv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return EnvironmentNotFound(id.String())
	}

	// Cannot delete default environment
	if schemaEnv.IsDefault {
		return CannotDeleteDefaultEnvironment()
	}

	// Cannot delete production without explicit confirmation
	if schemaEnv.IsProduction() {
		return CannotDeleteProductionEnvironment()
	}

	return s.repo.Delete(ctx, id)
}

// PromoteEnvironment promotes/clones one environment to another.
func (s *Service) PromoteEnvironment(ctx context.Context, req *PromoteEnvironmentRequest) (*Promotion, error) {
	if !s.config.AllowPromotion {
		return nil, PromotionNotAllowed()
	}

	// Get source environment
	sourceEnv, err := s.repo.FindByID(ctx, req.SourceEnvID)
	if err != nil {
		return nil, SourceEnvironmentNotFound(req.SourceEnvID.String())
	}

	// Validate target environment details
	if !s.isAllowedType(req.TargetType) {
		return nil, EnvironmentTypeForbidden(req.TargetType)
	}

	// Check if target environment already exists
	targetEnv, err := s.repo.FindByAppAndSlug(ctx, sourceEnv.AppID, req.TargetSlug)
	if err != nil && err.Error() != "environment not found" {
		return nil, err
	}

	// If target doesn't exist, create it
	if targetEnv == nil {
		now := time.Now().UTC()
		targetEnv = &schema.Environment{
			ID:        xid.New(),
			AppID:     sourceEnv.AppID,
			Name:      req.TargetName,
			Slug:      req.TargetSlug,
			Type:      req.TargetType,
			Status:    schema.EnvironmentStatusActive,
			Config:    make(map[string]any),
			IsDefault: false,
			AuditableModel: schema.AuditableModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		if err := s.repo.Create(ctx, targetEnv); err != nil {
			return nil, err
		}
	}

	// Create promotion record
	now := time.Now().UTC()
	promotion := &Promotion{
		ID:          xid.New(),
		AppID:       sourceEnv.AppID,
		SourceEnvID: sourceEnv.ID,
		TargetEnvID: targetEnv.ID,
		PromotedBy:  req.PromotedBy,
		IncludeData: req.IncludeData,
		Status:      schema.PromotionStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreatePromotion(ctx, promotion.ToSchema()); err != nil {
		return nil, err
	}

	// Start promotion process (async in production)
	if err := s.executePromotion(ctx, promotion.ToSchema(), sourceEnv, targetEnv); err != nil {
		promotion.Status = schema.PromotionStatusFailed
		promotion.ErrorMessage = err.Error()
		s.repo.UpdatePromotion(ctx, promotion.ToSchema())

		return nil, PromotionFailed(err.Error())
	}

	// Mark as completed
	promotion.Status = schema.PromotionStatusCompleted
	completedAt := time.Now().UTC()
	promotion.CompletedAt = &completedAt
	s.repo.UpdatePromotion(ctx, promotion.ToSchema())

	return promotion, nil
}

// executePromotion performs the actual promotion/cloning.
func (s *Service) executePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion, source, target *schema.Environment) error {
	// Update status to in progress
	promotion.Status = schema.PromotionStatusInProgress
	if err := s.repo.UpdatePromotion(ctx, promotion); err != nil {
		return err
	}

	// Copy configuration from source to target
	target.Config = source.Config
	if err := s.repo.Update(ctx, target); err != nil {
		return err
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

// GetPromotion retrieves a promotion by ID.
func (s *Service) GetPromotion(ctx context.Context, id xid.ID) (*Promotion, error) {
	schemaPromotion, err := s.repo.FindPromotionByID(ctx, id)
	if err != nil {
		return nil, PromotionNotFound(id.String())
	}

	return FromSchemaPromotion(schemaPromotion), nil
}

// ListPromotions lists promotions for an app with pagination.
func (s *Service) ListPromotions(ctx context.Context, filter *ListPromotionsFilter) (*ListPromotionsResponse, error) {
	// Get paginated results from repository
	pageResp, err := s.repo.ListPromotions(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema promotions to DTOs
	dtoPromotions := FromSchemaPromotions(pageResp.Data)

	// Return paginated response with DTOs
	return &pagination.PageResponse[*Promotion]{
		Data:       dtoPromotions,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
}

// Helper methods

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *Service) isAllowedType(envType string) bool {
	return slices.Contains(s.config.AllowedTypes, envType)
}
