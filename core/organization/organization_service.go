package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/internal/errs"
)

// OrganizationService handles organization aggregate operations
type OrganizationService struct {
	repo    OrganizationRepository
	config  Config
	rbacSvc *rbac.Service
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(repo OrganizationRepository, cfg Config, rbacSvc *rbac.Service) *OrganizationService {
	return &OrganizationService{
		repo:    repo,
		config:  cfg,
		rbacSvc: rbacSvc,
	}
}

// CreateOrganization creates a new user-created organization
func (s *OrganizationService) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*Organization, error) {
	// Check if user creation is enabled
	if !s.config.EnableUserCreation {
		return nil, OrganizationCreationDisabled()
	}

	if environmentID.IsNil() {
		envId, err := contexts.RequireEnvironmentID(ctx)
		if err != nil {
			return nil, errs.InternalServerError("failed to get environment ID", err)
		}
		environmentID = envId
	}

	// Check user's organization limit
	count, err := s.repo.CountByUser(ctx, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user organizations: %w", err)
	}
	if count >= s.config.MaxOrganizationsPerUser {
		return nil, MaxOrganizationsReached(s.config.MaxOrganizationsPerUser)
	}

	// Check if slug is already taken within this app+environment
	existing, err := s.repo.FindBySlug(ctx, appID, environmentID, req.Slug)
	if err == nil && existing != nil {
		return nil, OrganizationSlugExists(req.Slug)
	}

	// Handle logo (convert pointer to string)
	logo := ""
	if req.Logo != nil {
		logo = *req.Logo
	}

	// Create organization DTO
	now := time.Now().UTC()
	org := &Organization{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: environmentID,
		Name:          req.Name,
		Slug:          req.Slug,
		Logo:          logo,
		Metadata:      req.Metadata,
		CreatedBy:     creatorUserID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Bootstrap roles from templates if RBAC service is available
	if s.rbacSvc != nil {
		err = s.rbacSvc.BootstrapOrgRoles(ctx, org.ID, req.RoleTemplateIDs, req.RoleCustomizations)
		if err != nil {
			// Log error but don't fail org creation - roles can be added later
			fmt.Printf("[OrgService] Warning: failed to bootstrap org roles for %s: %v\n", org.ID.String(), err)
		} else {
			// Auto-assign owner role to creator
			err = s.rbacSvc.AssignOwnerRole(ctx, creatorUserID, org.ID)
			if err != nil {
				// Log error but don't fail - owner can be assigned manually
				fmt.Printf("[OrgService] Warning: failed to assign owner role to creator for %s: %v\n", org.ID.String(), err)
			}
		}
	}

	return org, nil
}

// FindOrganizationByID retrieves an organization by ID
func (s *OrganizationService) FindOrganizationByID(ctx context.Context, id xid.ID) (*Organization, error) {
	org, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, OrganizationNotFound()
	}
	return org, nil
}

// FindOrganizationBySlug retrieves an organization by slug
func (s *OrganizationService) FindOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*Organization, error) {
	org, err := s.repo.FindBySlug(ctx, appID, environmentID, slug)
	if err != nil {
		return nil, OrganizationNotFound()
	}
	return org, nil
}

// ListOrganizations lists organizations with pagination and filtering
func (s *OrganizationService) ListOrganizations(ctx context.Context, filter *ListOrganizationsFilter) (*pagination.PageResponse[*Organization], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByApp(ctx, filter)
}

// ListUserOrganizations lists organizations a user is a member of
func (s *OrganizationService) ListUserOrganizations(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Organization], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByUser(ctx, userID, filter)
}

// UpdateOrganization updates an organization
func (s *OrganizationService) UpdateOrganization(ctx context.Context, id xid.ID, req *UpdateOrganizationRequest) (*Organization, error) {
	org, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, OrganizationNotFound()
	}

	// Update fields
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	if req.Metadata != nil {
		org.Metadata = req.Metadata
	}
	org.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization deletes an organization (owner only - authorization check should be done before calling)
func (s *OrganizationService) DeleteOrganization(ctx context.Context, id, userID xid.ID) error {
	// Verify organization exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return OrganizationNotFound()
	}

	// Note: Authorization check (IsOwner) should be performed by the caller
	// This keeps the organization service focused on organization operations

	return s.repo.Delete(ctx, id)
}

// Type assertion to ensure OrganizationService implements OrganizationOperations
var _ OrganizationOperations = (*OrganizationService)(nil)
