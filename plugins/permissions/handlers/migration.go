package handlers

import (
	"encoding/json"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/plugins/permissions/migration"
	"github.com/xraph/forge"
)

// =============================================================================
// MIGRATION HANDLER
// =============================================================================

// MigrationHandler handles RBAC migration API endpoints.
type MigrationHandler struct {
	migrationService *migration.RBACMigrationService
}

// NewMigrationHandler creates a new migration handler.
func NewMigrationHandler(migrationService *migration.RBACMigrationService) *MigrationHandler {
	return &MigrationHandler{
		migrationService: migrationService,
	}
}

// =============================================================================
// MIGRATION DTOs
// =============================================================================

// MigrateAllRequest is the request to migrate all RBAC policies.
type MigrateAllRequest struct {
	DryRun           bool `json:"dryRun"`
	PreserveOriginal bool `json:"preserveOriginal"`
}

// MigrateAllResponse is the response from migrating all RBAC policies.
type MigrateAllResponse struct {
	TotalPolicies     int                      `json:"totalPolicies"`
	MigratedPolicies  int                      `json:"migratedPolicies"`
	SkippedPolicies   int                      `json:"skippedPolicies"`
	FailedPolicies    int                      `json:"failedPolicies"`
	Errors            []MigrationErrorResponse `json:"errors,omitempty"`
	ConvertedPolicies []PolicyPreviewResponse  `json:"convertedPolicies,omitempty"`
	StartedAt         string                   `json:"startedAt"`
	CompletedAt       string                   `json:"completedAt"`
	DryRun            bool                     `json:"dryRun"`
}

// MigrationErrorResponse represents a migration error in API response.
type MigrationErrorResponse struct {
	PolicyIndex int    `json:"policyIndex"`
	Subject     string `json:"subject"`
	Resource    string `json:"resource"`
	Error       string `json:"error"`
}

// PolicyPreviewResponse represents a preview of a converted policy.
type PolicyPreviewResponse struct {
	Name        string   `json:"name"`
	Expression  string   `json:"expression"`
	Resource    string   `json:"resourceType"`
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
}

// MigrateRolesRequest is the request to migrate role-based permissions.
type MigrateRolesRequest struct {
	DryRun bool `json:"dryRun"`
}

// MigrateRolesResponse is the response from migrating roles.
type MigrateRolesResponse = MigrateAllResponse

// PreviewConversionRequest is the request to preview an RBAC policy conversion.
type PreviewConversionRequest struct {
	Subject   string   `json:"subject"             validate:"required"`
	Actions   []string `json:"actions"             validate:"required,min=1"`
	Resource  string   `json:"resource"            validate:"required"`
	Condition string   `json:"condition,omitempty"`
}

// PreviewConversionResponse is the response from previewing a conversion.
type PreviewConversionResponse struct {
	Success       bool   `json:"success"`
	CELExpression string `json:"celExpression,omitempty"`
	ResourceType  string `json:"resourceType,omitempty"`
	ResourceID    string `json:"resourceId,omitempty"`
	PolicyName    string `json:"policyName,omitempty"`
	Error         string `json:"error,omitempty"`
}

// GetMigrationStatusRequest is the request to get migration status.
type GetMigrationStatusRequest struct {
	// No fields needed
}

// GetMigrationStatusResponse is the response with migration status.
type GetMigrationStatusResponse struct {
	HasMigratedPolicies bool   `json:"hasMigratedPolicies"`
	MigratedCount       int    `json:"migratedCount"`
	LastMigrationAt     string `json:"lastMigrationAt,omitempty"`
	PendingRBACPolicies int    `json:"pendingRbacPolicies"`
}

// =============================================================================
// HANDLERS
// =============================================================================

// MigrateAll migrates all RBAC policies to the permissions system.
func (h *MigrationHandler) MigrateAll(c forge.Context) error {
	// Extract context
	appID, envID, orgID, userID, err := extractMigrationContext(c)
	if err != nil {
		return c.JSON(400, ErrorResponse{
			Code:    "invalid_context",
			Message: err.Error(),
		})
	}

	// Bind request
	var req MigrateAllRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		// Use defaults if no body provided
		req = MigrateAllRequest{
			DryRun:           false,
			PreserveOriginal: true,
		}
	}

	// Check if migration service is available
	if h.migrationService == nil {
		return c.JSON(501, ErrorResponse{
			Code:    "not_implemented",
			Message: "RBAC migration service not configured",
		})
	}

	// Execute migration
	result, err := h.migrationService.MigrateAll(c.Request().Context(), appID, envID, orgID, userID)
	if err != nil {
		return c.JSON(500, ErrorResponse{
			Code:    "migration_failed",
			Message: err.Error(),
		})
	}

	// Convert to response
	response := convertMigrationResult(result)

	return c.JSON(200, response)
}

// MigrateRoles migrates role-based permissions to policies.
func (h *MigrationHandler) MigrateRoles(c forge.Context) error {
	// Extract context
	appID, envID, _, userID, err := extractMigrationContext(c)
	if err != nil {
		return c.JSON(400, ErrorResponse{
			Code:    "invalid_context",
			Message: err.Error(),
		})
	}

	// Check if migration service is available
	if h.migrationService == nil {
		return c.JSON(501, ErrorResponse{
			Code:    "not_implemented",
			Message: "RBAC migration service not configured",
		})
	}

	// Execute migration
	result, err := h.migrationService.MigrateRoles(c.Request().Context(), appID, envID, userID)
	if err != nil {
		return c.JSON(500, ErrorResponse{
			Code:    "migration_failed",
			Message: err.Error(),
		})
	}

	// Convert to response
	response := convertMigrationResult(result)

	return c.JSON(200, response)
}

// PreviewConversion previews the conversion of an RBAC policy.
func (h *MigrationHandler) PreviewConversion(c forge.Context) error {
	// Bind request
	var req PreviewConversionRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, ErrorResponse{
			Code:    "invalid_request",
			Message: err.Error(),
		})
	}

	// Check if migration service is available
	if h.migrationService == nil {
		return c.JSON(501, ErrorResponse{
			Code:    "not_implemented",
			Message: "RBAC migration service not configured",
		})
	}

	// Convert to RBAC policy structure
	rbacPolicy := &migration.RBACPolicy{
		Subject:   req.Subject,
		Actions:   req.Actions,
		Resource:  req.Resource,
		Condition: req.Condition,
	}

	// Preview conversion
	preview, err := h.migrationService.PreviewConversion(c.Request().Context(), rbacPolicy)
	if err != nil {
		return c.JSON(500, ErrorResponse{
			Code:    "preview_failed",
			Message: err.Error(),
		})
	}

	// Convert to response
	response := PreviewConversionResponse{
		Success:       preview.Success,
		CELExpression: preview.CELExpression,
		ResourceType:  preview.ResourceType,
		ResourceID:    preview.ResourceID,
		PolicyName:    preview.PolicyName,
		Error:         preview.Error,
	}

	return c.JSON(200, response)
}

// =============================================================================
// HELPERS
// =============================================================================

// extractMigrationContext extracts app, env, org, user IDs from context.
func extractMigrationContext(c forge.Context) (appID, envID xid.ID, orgID *xid.ID, userID xid.ID, err error) {
	ctx := c.Request().Context()

	// Get app ID
	appID, _ = contexts.GetAppID(ctx)

	// Get environment ID
	envID, _ = contexts.GetEnvironmentID(ctx)

	// Get organization ID (optional)
	orgIDVal, ok := contexts.GetOrganizationID(ctx)
	if ok && !orgIDVal.IsNil() {
		orgID = &orgIDVal
	}

	// Get user ID
	userID, _ = contexts.GetUserID(ctx)

	return appID, envID, orgID, userID, nil
}

// convertMigrationResult converts internal result to API response.
func convertMigrationResult(result *migration.MigrationResult) *MigrateAllResponse {
	response := &MigrateAllResponse{
		TotalPolicies:    result.TotalPolicies,
		MigratedPolicies: result.MigratedPolicies,
		SkippedPolicies:  result.SkippedPolicies,
		FailedPolicies:   result.FailedPolicies,
		StartedAt:        result.StartedAt.Format("2006-01-02T15:04:05Z"),
		CompletedAt:      result.CompletedAt.Format("2006-01-02T15:04:05Z"),
		DryRun:           result.DryRun,
	}

	// Convert errors
	for _, e := range result.Errors {
		response.Errors = append(response.Errors, MigrationErrorResponse{
			PolicyIndex: e.PolicyIndex,
			Subject:     e.Subject,
			Resource:    e.Resource,
			Error:       e.Error,
		})
	}

	// Convert previews (only include in dry run for brevity)
	if result.DryRun && len(result.ConvertedPolicies) > 0 {
		for _, p := range result.ConvertedPolicies {
			response.ConvertedPolicies = append(response.ConvertedPolicies, PolicyPreviewResponse{
				Name:        p.Name,
				Expression:  p.Expression,
				Resource:    p.ResourceType,
				Actions:     p.Actions,
				Description: p.Description,
			})
		}
	}

	return response
}
