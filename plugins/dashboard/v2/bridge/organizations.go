package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// OrganizationsListInput represents organizations list request
type OrganizationsListInput struct {
	AppID    string `json:"appId" validate:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// OrganizationsListOutput represents organizations list response
type OrganizationsListOutput struct {
	Organizations []OrganizationItem `json:"organizations"`
	Total         int                `json:"total"`
	Page          int                `json:"page"`
	PageSize      int                `json:"pageSize"`
	TotalPages    int                `json:"totalPages"`
}

// OrganizationItem represents an organization in the list
type OrganizationItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MemberCount int    `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
	Status      string `json:"status"`
}

// OrganizationDetailInput represents organization detail request
type OrganizationDetailInput struct {
	OrgID string `json:"orgId" validate:"required"`
}

// OrganizationDetailOutput represents organization detail response
type OrganizationDetailOutput struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Members     []MemberItem `json:"members"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt,omitempty"`
	Status      string       `json:"status"`
}

// MemberItem represents a member in an organization
type MemberItem struct {
	UserID   string   `json:"userId"`
	Email    string   `json:"email"`
	Name     string   `json:"name,omitempty"`
	Roles    []string `json:"roles"`
	JoinedAt string   `json:"joinedAt"`
}

// CreateOrganizationInput represents organization creation request
type CreateOrganizationInput struct {
	AppID       string `json:"appId" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

// UpdateOrganizationInput represents organization update request
type UpdateOrganizationInput struct {
	OrgID       string `json:"orgId" validate:"required"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// DeleteOrganizationInput represents organization delete request
type DeleteOrganizationInput struct {
	OrgID string `json:"orgId" validate:"required"`
}

// registerOrganizationFunctions registers organization management bridge functions
func (bm *BridgeManager) registerOrganizationFunctions() error {
	// List organizations
	if err := bm.bridge.Register("getOrganizationsList", bm.getOrganizationsList,
		bridge.WithDescription("Get list of organizations"),
	); err != nil {
		return fmt.Errorf("failed to register getOrganizationsList: %w", err)
	}

	// Get organization detail
	if err := bm.bridge.Register("getOrganizationDetail", bm.getOrganizationDetail,
		bridge.WithDescription("Get detailed information about an organization"),
	); err != nil {
		return fmt.Errorf("failed to register getOrganizationDetail: %w", err)
	}

	// Create organization
	if err := bm.bridge.Register("createOrganization", bm.createOrganization,
		bridge.WithDescription("Create a new organization"),
	); err != nil {
		return fmt.Errorf("failed to register createOrganization: %w", err)
	}

	// Update organization
	if err := bm.bridge.Register("updateOrganization", bm.updateOrganization,
		bridge.WithDescription("Update organization information"),
	); err != nil {
		return fmt.Errorf("failed to register updateOrganization: %w", err)
	}

	// Delete organization
	if err := bm.bridge.Register("deleteOrganization", bm.deleteOrganization,
		bridge.WithDescription("Delete an organization"),
	); err != nil {
		return fmt.Errorf("failed to register deleteOrganization: %w", err)
	}

	bm.log.Info("organization bridge functions registered")
	return nil
}

// getOrganizationsList retrieves list of organizations
func (bm *BridgeManager) getOrganizationsList(ctx bridge.Context, input OrganizationsListInput) (*OrganizationsListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	// Check if organization service is available
	if bm.orgSvc == nil {
		return &OrganizationsListOutput{
			Organizations: []OrganizationItem{},
			Total:         0,
			Page:          1,
			PageSize:      20,
			TotalPages:    0,
		}, nil
	}

	// Set defaults
	if input.Page == 0 {
		input.Page = 1
	}
	if input.PageSize == 0 {
		input.PageSize = 20
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Get environment ID from context (required for organizations)
	envID, _ := contexts.GetEnvironmentID(goCtx)
	
	// List organizations
	filter := &organization.ListOrganizationsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  input.Page,
			Limit: input.PageSize,
		},
		AppID:         appID,
		EnvironmentID: envID,
	}

	response, err := bm.orgSvc.ListOrganizations(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list organizations", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch organizations")
	}

	// Transform to DTOs
	orgs := make([]OrganizationItem, len(response.Data))
	for i, o := range response.Data {
		// Extract description from metadata if available
		description := ""
		if o.Metadata != nil {
			if desc, ok := o.Metadata["description"].(string); ok {
				description = desc
			}
		}
		
		orgs[i] = OrganizationItem{
			ID:          o.ID.String(),
			Name:        o.Name,
			Description: description,
			MemberCount: 0, // TODO: Count members if needed
			CreatedAt:   o.CreatedAt.Format(time.RFC3339),
			Status:      "active",
		}
	}

	totalPages := 0
	if response.Pagination.Total > 0 && input.PageSize > 0 {
		totalPages = (int(response.Pagination.Total) + input.PageSize - 1) / input.PageSize
	}

	return &OrganizationsListOutput{
		Organizations: orgs,
		Total:         int(response.Pagination.Total),
		Page:          input.Page,
		PageSize:      input.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// getOrganizationDetail retrieves detailed information about an organization
func (bm *BridgeManager) getOrganizationDetail(ctx bridge.Context, input OrganizationDetailInput) (*OrganizationDetailOutput, error) {
	if input.OrgID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "orgId is required")
	}

	// Check if organization service is available
	if bm.orgSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "organization service not available")
	}

	// Parse orgID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid orgId")
	}

	goCtx := bm.buildContext(ctx)

	// Get organization
	o, err := bm.orgSvc.FindOrganizationByID(goCtx, orgID)
	if err != nil {
		bm.log.Error("failed to get organization", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "organization not found")
	}

	// Extract description from metadata if available
	description := ""
	if o.Metadata != nil {
		if desc, ok := o.Metadata["description"].(string); ok {
			description = desc
		}
	}

	return &OrganizationDetailOutput{
		ID:          o.ID.String(),
		Name:        o.Name,
		Description: description,
		Members:     []MemberItem{}, // TODO: List members if needed
		CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   o.UpdatedAt.Format(time.RFC3339),
		Status:      "active",
	}, nil
}

// createOrganization creates a new organization
func (bm *BridgeManager) createOrganization(ctx bridge.Context, input CreateOrganizationInput) (*GenericSuccessOutput, error) {
	if input.AppID == "" || input.Name == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and name are required")
	}

	// Check if organization service is available
	if bm.orgSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "organization service not available")
	}

	// Parse appID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
	}

	goCtx := bm.buildContext(ctx)
	goCtx = contexts.SetAppID(goCtx, appID)

	// Get user ID from bridge context for creator
	userID := xid.NilID() // TODO: Extract from bridge context if available
	
	// Get environment ID from context
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Generate slug from name
	slug := generateSlug(input.Name)

	// Store description in metadata
	metadata := make(map[string]interface{})
	if input.Description != "" {
		metadata["description"] = input.Description
	}

	// Create organization
	createReq := &organization.CreateOrganizationRequest{
		Name:     input.Name,
		Slug:     slug,
		Metadata: metadata,
	}

	_, err = bm.orgSvc.CreateOrganization(goCtx, createReq, userID, appID, envID)
	if err != nil {
		bm.log.Error("failed to create organization", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create organization")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Organization created successfully",
	}, nil
}

// updateOrganization updates organization information
func (bm *BridgeManager) updateOrganization(ctx bridge.Context, input UpdateOrganizationInput) (*GenericSuccessOutput, error) {
	if input.OrgID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "orgId is required")
	}

	// Check if organization service is available
	if bm.orgSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "organization service not available")
	}

	// Parse orgID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid orgId")
	}

	goCtx := bm.buildContext(ctx)

	// Build update request
	updateReq := &organization.UpdateOrganizationRequest{}
	
	if input.Name != "" {
		updateReq.Name = &input.Name
	}
	
	// Store description in metadata
	if input.Description != "" {
		metadata := make(map[string]interface{})
		metadata["description"] = input.Description
		updateReq.Metadata = metadata
	}

	// Update organization
	_, err = bm.orgSvc.UpdateOrganization(goCtx, orgID, updateReq)
	if err != nil {
		bm.log.Error("failed to update organization", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to update organization")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Organization updated successfully",
	}, nil
}

// deleteOrganization deletes an organization
func (bm *BridgeManager) deleteOrganization(ctx bridge.Context, input DeleteOrganizationInput) (*GenericSuccessOutput, error) {
	if input.OrgID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "orgId is required")
	}

	// Check if organization service is available
	if bm.orgSvc == nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "organization service not available")
	}

	// Parse orgID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid orgId")
	}

	goCtx := bm.buildContext(ctx)

	// Delete organization using ForceDeleteOrganization (admin operation)
	err = bm.orgSvc.ForceDeleteOrganization(goCtx, orgID)
	if err != nil {
		bm.log.Error("failed to delete organization", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete organization")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Organization deleted successfully",
	}, nil
}

// generateSlug generates a URL-safe slug from a name
func generateSlug(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (keep alphanumeric and hyphens)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
