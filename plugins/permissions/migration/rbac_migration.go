package migration

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// RBAC MIGRATION SERVICE
// =============================================================================

// RBACMigrationService handles migration from the legacy RBAC system to
// the new CEL-based permissions system.
type RBACMigrationService struct {
	// Repository for storing migrated policies
	policyRepo PolicyRepository

	// RBAC service for reading existing policies
	rbacService RBACService

	// Logger
	logger Logger

	// Configuration
	config MigrationConfig
}

// MigrationConfig configures the migration service.
type MigrationConfig struct {
	// BatchSize for processing policies
	BatchSize int

	// DryRun mode - log but don't persist
	DryRun bool

	// PreserveOriginal keeps RBAC policies after migration
	PreserveOriginal bool

	// DefaultNamespace for migrated policies
	DefaultNamespace string

	// DefaultPriority for migrated policies
	DefaultPriority int
}

// DefaultMigrationConfig returns default configuration.
func DefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		BatchSize:        100,
		DryRun:           false,
		PreserveOriginal: true,
		DefaultNamespace: "default",
		DefaultPriority:  100,
	}
}

// =============================================================================
// INTERFACES
// =============================================================================

// PolicyRepository interface for storing migrated policies.
type PolicyRepository interface {
	CreatePolicy(ctx context.Context, policy *core.Policy) error
	GetPoliciesByResourceType(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, resourceType string) ([]*core.Policy, error)
}

// RBACService interface for reading existing RBAC data.
type RBACService interface {
	// GetAllPolicies returns all RBAC policies
	GetAllPolicies(ctx context.Context) ([]*RBACPolicy, error)

	// GetRoles returns all roles for an app and environment
	GetRoles(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error)

	// GetRolePermissions returns permissions for a role
	GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error)
}

// RBACPolicy represents a legacy RBAC policy.
type RBACPolicy struct {
	Subject   string   `json:"subject"`   // e.g., "user", "role:admin"
	Actions   []string `json:"actions"`   // e.g., ["read", "write"]
	Resource  string   `json:"resource"`  // e.g., "project:*", "document:123"
	Condition string   `json:"condition"` // e.g., "owner = true"
}

// Logger interface for migration logging.
type Logger interface {
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

// NewRBACMigrationService creates a new RBAC migration service.
func NewRBACMigrationService(
	policyRepo PolicyRepository,
	rbacService RBACService,
	logger Logger,
	config MigrationConfig,
) *RBACMigrationService {
	return &RBACMigrationService{
		policyRepo:  policyRepo,
		rbacService: rbacService,
		logger:      logger,
		config:      config,
	}
}

// =============================================================================
// MIGRATION METHODS
// =============================================================================

// MigrationResult represents the result of a migration operation.
type MigrationResult struct {
	TotalPolicies     int              `json:"totalPolicies"`
	MigratedPolicies  int              `json:"migratedPolicies"`
	SkippedPolicies   int              `json:"skippedPolicies"`
	FailedPolicies    int              `json:"failedPolicies"`
	Errors            []MigrationError `json:"errors,omitempty"`
	ConvertedPolicies []*core.Policy   `json:"convertedPolicies,omitempty"`
	StartedAt         time.Time        `json:"startedAt"`
	CompletedAt       time.Time        `json:"completedAt"`
	DryRun            bool             `json:"dryRun"`
}

// MigrationError represents an error during migration.
type MigrationError struct {
	PolicyIndex int    `json:"policyIndex"`
	Subject     string `json:"subject"`
	Resource    string `json:"resource"`
	Error       string `json:"error"`
}

// MigrateAll migrates all RBAC policies to the permissions system.
func (s *RBACMigrationService) MigrateAll(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, createdBy xid.ID) (*MigrationResult, error) {
	result := &MigrationResult{
		StartedAt: time.Now().UTC(),
		DryRun:    s.config.DryRun,
	}

	// Fetch all RBAC policies
	policies, err := s.rbacService.GetAllPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RBAC policies: %w", err)
	}

	result.TotalPolicies = len(policies)
	s.logger.Info("Starting RBAC migration", "total_policies", len(policies), "dry_run", s.config.DryRun)

	// Convert and store each policy
	for i, rbacPolicy := range policies {
		// Convert RBAC policy to CEL policy
		celPolicy, err := s.ConvertPolicy(ctx, rbacPolicy, appID, envID, userOrgID, createdBy)
		if err != nil {
			result.FailedPolicies++
			result.Errors = append(result.Errors, MigrationError{
				PolicyIndex: i,
				Subject:     rbacPolicy.Subject,
				Resource:    rbacPolicy.Resource,
				Error:       err.Error(),
			})
			s.logger.Warn("Failed to convert policy",
				"index", i,
				"subject", rbacPolicy.Subject,
				"error", err.Error())

			continue
		}

		result.ConvertedPolicies = append(result.ConvertedPolicies, celPolicy)

		// Store policy if not dry run
		if !s.config.DryRun {
			if err := s.policyRepo.CreatePolicy(ctx, celPolicy); err != nil {
				result.FailedPolicies++
				result.Errors = append(result.Errors, MigrationError{
					PolicyIndex: i,
					Subject:     rbacPolicy.Subject,
					Resource:    rbacPolicy.Resource,
					Error:       "storage error: " + err.Error(),
				})
				s.logger.Error("Failed to store migrated policy",
					"index", i,
					"name", celPolicy.Name,
					"error", err.Error())

				continue
			}
		}

		result.MigratedPolicies++
	}

	result.CompletedAt = time.Now().UTC()
	s.logger.Info("RBAC migration completed",
		"migrated", result.MigratedPolicies,
		"skipped", result.SkippedPolicies,
		"failed", result.FailedPolicies,
		"duration", result.CompletedAt.Sub(result.StartedAt).String())

	return result, nil
}

// ConvertPolicy converts a single RBAC policy to a CEL policy.
func (s *RBACMigrationService) ConvertPolicy(
	ctx context.Context,
	rbacPolicy *RBACPolicy,
	appID, envID xid.ID,
	userOrgID *xid.ID,
	createdBy xid.ID,
) (*core.Policy, error) {
	if rbacPolicy == nil {
		return nil, errs.BadRequest("rbac policy is nil")
	}

	// Extract resource type from resource pattern
	resourceType, _ := extractResourceType(rbacPolicy.Resource)
	if resourceType == "" {
		resourceType = "unknown"
	}

	// Convert to CEL expression
	celExpression, err := s.convertToCEL(rbacPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to CEL: %w", err)
	}

	// Generate policy name
	policyName := generatePolicyName(rbacPolicy)

	now := time.Now().UTC()
	policy := &core.Policy{
		ID:                 xid.New(),
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: userOrgID,
		NamespaceID:        xid.NilID(), // Will be set based on namespace lookup
		Name:               policyName,
		Description:        fmt.Sprintf("Migrated from RBAC: %s on %s", rbacPolicy.Subject, rbacPolicy.Resource),
		Expression:         celExpression,
		ResourceType:       resourceType,
		Actions:            rbacPolicy.Actions,
		Priority:           s.config.DefaultPriority,
		Enabled:            true,
		Version:            1,
		CreatedBy:          createdBy,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return policy, nil
}

// =============================================================================
// CEL CONVERSION
// =============================================================================

// convertToCEL converts an RBAC policy to a CEL expression.
func (s *RBACMigrationService) convertToCEL(rbacPolicy *RBACPolicy) (string, error) {
	var conditions []string

	// Convert subject condition
	subjectCond, err := s.convertSubject(rbacPolicy.Subject)
	if err != nil {
		return "", fmt.Errorf("failed to convert subject: %w", err)
	}

	if subjectCond != "" {
		conditions = append(conditions, subjectCond)
	}

	// Convert resource condition
	resourceCond, err := s.convertResource(rbacPolicy.Resource)
	if err != nil {
		return "", fmt.Errorf("failed to convert resource: %w", err)
	}

	if resourceCond != "" {
		conditions = append(conditions, resourceCond)
	}

	// Convert action condition
	actionCond := s.convertActions(rbacPolicy.Actions)
	if actionCond != "" {
		conditions = append(conditions, actionCond)
	}

	// Convert custom condition
	if rbacPolicy.Condition != "" {
		customCond, err := s.convertCondition(rbacPolicy.Condition)
		if err != nil {
			return "", fmt.Errorf("failed to convert condition: %w", err)
		}

		if customCond != "" {
			conditions = append(conditions, customCond)
		}
	}

	// Combine all conditions
	if len(conditions) == 0 {
		return "true", nil // Allow all
	}

	return strings.Join(conditions, " && "), nil
}

// convertSubject converts RBAC subject to CEL condition.
func (s *RBACMigrationService) convertSubject(subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" || subject == "*" {
		return "", nil // No subject restriction
	}

	// Handle role-based subjects: "role:admin" -> has_role("admin")
	if strings.HasPrefix(strings.ToLower(subject), "role:") {
		roleName := strings.TrimPrefix(subject[5:], " ")

		return fmt.Sprintf(`principal.roles.exists(r, r == "%s")`, roleName), nil
	}

	// Handle user-based subjects: "user:123" -> principal.id == "123"
	if strings.HasPrefix(strings.ToLower(subject), "user:") {
		userID := strings.TrimPrefix(subject[5:], " ")
		if userID == "*" {
			return "", nil // Any user
		}

		return fmt.Sprintf(`principal.id == "%s"`, userID), nil
	}

	// Handle group-based subjects: "group:engineering" -> has_group("engineering")
	if strings.HasPrefix(strings.ToLower(subject), "group:") {
		groupName := strings.TrimPrefix(subject[6:], " ")

		return fmt.Sprintf(`principal.groups.exists(g, g == "%s")`, groupName), nil
	}

	// Handle permission-based subjects: "permission:users.read" -> has_permission("users.read")
	if strings.HasPrefix(strings.ToLower(subject), "permission:") {
		permName := strings.TrimPrefix(subject[11:], " ")

		return fmt.Sprintf(`principal.permissions.exists(p, p == "%s")`, permName), nil
	}

	// Default: treat as literal user ID or role
	return fmt.Sprintf(`principal.id == "%s" || principal.roles.exists(r, r == "%s")`, subject, subject), nil
}

// convertResource converts RBAC resource to CEL condition.
func (s *RBACMigrationService) convertResource(resource string) (string, error) {
	resource = strings.TrimSpace(resource)
	if resource == "" || resource == "*" {
		return "", nil // No resource restriction
	}

	// Parse resource pattern: "type:id" or "type:*"
	parts := strings.SplitN(resource, ":", 2)
	if len(parts) == 1 {
		// Just resource type, any ID
		return fmt.Sprintf(`resource.type == "%s"`, resource), nil
	}

	resourceType := parts[0]
	resourceID := parts[1]

	// Wildcard ID
	if resourceID == "*" {
		return fmt.Sprintf(`resource.type == "%s"`, resourceType), nil
	}

	// Prefix wildcard: "doc_*" -> starts with
	if strings.HasSuffix(resourceID, "*") {
		prefix := resourceID[:len(resourceID)-1]

		return fmt.Sprintf(`resource.type == "%s" && resource.id.startsWith("%s")`, resourceType, prefix), nil
	}

	// Specific resource
	return fmt.Sprintf(`resource.type == "%s" && resource.id == "%s"`, resourceType, resourceID), nil
}

// convertActions converts RBAC actions to CEL condition.
func (s *RBACMigrationService) convertActions(actions []string) string {
	if len(actions) == 0 {
		return ""
	}

	// Single action
	if len(actions) == 1 {
		if actions[0] == "*" {
			return "" // Any action allowed
		}

		return fmt.Sprintf(`action == "%s"`, actions[0])
	}

	// Multiple actions
	quotedActions := make([]string, len(actions))
	for i, a := range actions {
		quotedActions[i] = fmt.Sprintf(`"%s"`, a)
	}

	return fmt.Sprintf(`action in [%s]`, strings.Join(quotedActions, ", "))
}

// convertCondition converts RBAC condition string to CEL expression.
func (s *RBACMigrationService) convertCondition(condition string) (string, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return "", nil
	}

	// Common condition patterns:

	// "owner = true" -> resource.owner == principal.id
	if matches := ownerPattern.FindStringSubmatch(condition); len(matches) > 0 {
		value := matches[1]
		if value == "true" {
			return "resource.owner == principal.id", nil
		}

		return "resource.owner != principal.id", nil
	}

	// "team = <team_id>" -> resource.team_id == principal.team_id
	if matches := teamPattern.FindStringSubmatch(condition); len(matches) > 0 {
		teamValue := matches[1]
		if teamValue == "own" || teamValue == "self" {
			return "resource.team_id == principal.team_id", nil
		}

		return fmt.Sprintf(`resource.team_id == "%s"`, teamValue), nil
	}

	// "visibility = public" -> resource.visibility == "public"
	if matches := visibilityPattern.FindStringSubmatch(condition); len(matches) > 0 {
		visibility := matches[1]

		return fmt.Sprintf(`resource.visibility == "%s"`, visibility), nil
	}

	// "org = <org_id>" -> resource.org_id == principal.org_id
	if matches := orgPattern.FindStringSubmatch(condition); len(matches) > 0 {
		orgValue := matches[1]
		if orgValue == "own" || orgValue == "self" || orgValue == "same" {
			return "resource.org_id == principal.org_id", nil
		}

		return fmt.Sprintf(`resource.org_id == "%s"`, orgValue), nil
	}

	// Generic comparison: "field = value" -> resource.field == "value"
	if matches := genericComparePattern.FindStringSubmatch(condition); len(matches) > 0 {
		field := matches[1]
		op := matches[2]
		value := matches[3]

		celOp := "=="

		switch op {
		case "=", "==":
			celOp = "=="
		case "!=", "<>":
			celOp = "!="
		case ">":
			celOp = ">"
		case "<":
			celOp = "<"
		case ">=":
			celOp = ">="
		case "<=":
			celOp = "<="
		}

		// Check if value is a number or boolean
		if value == "true" || value == "false" {
			return fmt.Sprintf("resource.%s %s %s", field, celOp, value), nil
		}
		// Try to detect numbers (simple check)
		if isNumeric(value) {
			return fmt.Sprintf("resource.%s %s %s", field, celOp, value), nil
		}

		return fmt.Sprintf(`resource.%s %s "%s"`, field, celOp, value), nil
	}

	// If no pattern matches, return the condition wrapped for CEL
	// This may not work, but allows manual fixing
	return fmt.Sprintf("/* MANUAL REVIEW: %s */", condition), nil
}

// =============================================================================
// ROLE MIGRATION
// =============================================================================

// MigrateRoles migrates role-based permissions to policies.
func (s *RBACMigrationService) MigrateRoles(ctx context.Context, appID, envID xid.ID, createdBy xid.ID) (*MigrationResult, error) {
	result := &MigrationResult{
		StartedAt: time.Now().UTC(),
		DryRun:    s.config.DryRun,
	}

	// Fetch all roles
	roles, err := s.rbacService.GetRoles(ctx, appID, envID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles: %w", err)
	}

	s.logger.Info("Starting role migration", "total_roles", len(roles), "dry_run", s.config.DryRun)

	for i, role := range roles {
		// Fetch permissions for this role
		permissions, err := s.rbacService.GetRolePermissions(ctx, role.ID)
		if err != nil {
			s.logger.Warn("Failed to fetch role permissions",
				"role_id", role.ID.String(),
				"error", err.Error())

			continue
		}

		// Create a policy for each permission
		for _, perm := range permissions {
			policy, err := s.createRolePermissionPolicy(role, perm, appID, envID, role.OrganizationID, createdBy)
			if err != nil {
				result.FailedPolicies++
				result.Errors = append(result.Errors, MigrationError{
					PolicyIndex: i,
					Subject:     "role:" + role.Name,
					Error:       err.Error(),
				})

				continue
			}

			result.TotalPolicies++
			result.ConvertedPolicies = append(result.ConvertedPolicies, policy)

			if !s.config.DryRun {
				if err := s.policyRepo.CreatePolicy(ctx, policy); err != nil {
					result.FailedPolicies++
					result.Errors = append(result.Errors, MigrationError{
						PolicyIndex: i,
						Subject:     "role:" + role.Name,
						Error:       "storage error: " + err.Error(),
					})

					continue
				}
			}

			result.MigratedPolicies++
		}
	}

	result.CompletedAt = time.Now().UTC()
	s.logger.Info("Role migration completed",
		"migrated", result.MigratedPolicies,
		"failed", result.FailedPolicies)

	return result, nil
}

// createRolePermissionPolicy creates a CEL policy from a role-permission mapping.
func (s *RBACMigrationService) createRolePermissionPolicy(
	role *schema.Role,
	permission *schema.Permission,
	appID, envID xid.ID,
	orgID *xid.ID,
	createdBy xid.ID,
) (*core.Policy, error) {
	// Parse permission name: typically "resource.action" format
	parts := strings.SplitN(permission.Name, ".", 2)
	resourceType := "unknown"
	action := permission.Name

	if len(parts) == 2 {
		resourceType = parts[0]
		action = parts[1]
	}

	// Build CEL expression
	celExpression := fmt.Sprintf(`principal.roles.exists(r, r == "%s")`, role.Name)

	now := time.Now().UTC()
	policy := &core.Policy{
		ID:                 xid.New(),
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		Name:               fmt.Sprintf("role_%s_%s", role.Name, permission.Name),
		Description:        fmt.Sprintf("Role %s has permission %s", role.Name, permission.Name),
		Expression:         celExpression,
		ResourceType:       resourceType,
		Actions:            []string{action},
		Priority:           s.config.DefaultPriority,
		Enabled:            true,
		Version:            1,
		CreatedBy:          createdBy,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return policy, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// Regex patterns for condition parsing.
var (
	ownerPattern          = regexp.MustCompile(`(?i)owner\s*=\s*(true|false)`)
	teamPattern           = regexp.MustCompile(`(?i)team\s*=\s*(\S+)`)
	visibilityPattern     = regexp.MustCompile(`(?i)visibility\s*=\s*(\S+)`)
	orgPattern            = regexp.MustCompile(`(?i)org(?:anization)?\s*=\s*(\S+)`)
	genericComparePattern = regexp.MustCompile(`(\w+)\s*(=|==|!=|<>|>|<|>=|<=)\s*(.+)`)
)

// extractResourceType extracts the resource type from a resource pattern.
func extractResourceType(resource string) (string, string) {
	if resource == "" || resource == "*" {
		return "*", ""
	}

	parts := strings.SplitN(resource, ":", 2)
	if len(parts) == 1 {
		return resource, ""
	}

	return parts[0], parts[1]
}

// generatePolicyName generates a unique policy name from RBAC policy.
func generatePolicyName(rbacPolicy *RBACPolicy) string {
	// Sanitize subject and resource for naming
	subject := sanitizeForName(rbacPolicy.Subject)
	resource := sanitizeForName(rbacPolicy.Resource)

	// Combine with timestamp suffix for uniqueness
	return fmt.Sprintf("migrated_%s_%s_%d", subject, resource, time.Now().UnixNano()%10000)
}

// sanitizeForName removes/replaces invalid characters for policy names.
func sanitizeForName(s string) string {
	// Replace common separators with underscores
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "any")
	s = strings.ReplaceAll(s, " ", "_")

	// Remove other special characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	s = reg.ReplaceAllString(s, "")

	// Truncate if too long
	if len(s) > 50 {
		s = s[:50]
	}

	return strings.ToLower(s)
}

// isNumeric checks if a string represents a number.
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			if c != '.' && c != '-' {
				return false
			}
		}
	}

	return len(s) > 0
}

// =============================================================================
// PREVIEW METHODS
// =============================================================================

// PreviewConversion previews the conversion of an RBAC policy without storing.
func (s *RBACMigrationService) PreviewConversion(ctx context.Context, rbacPolicy *RBACPolicy) (*ConversionPreview, error) {
	celExpr, err := s.convertToCEL(rbacPolicy)
	if err != nil {
		return &ConversionPreview{
			Original: rbacPolicy,
			Success:  false,
			Error:    err.Error(),
		}, nil
	}

	resourceType, resourceID := extractResourceType(rbacPolicy.Resource)

	return &ConversionPreview{
		Original:      rbacPolicy,
		Success:       true,
		CELExpression: celExpr,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		PolicyName:    generatePolicyName(rbacPolicy),
	}, nil
}

// ConversionPreview represents a preview of policy conversion.
type ConversionPreview struct {
	Original      *RBACPolicy `json:"original"`
	Success       bool        `json:"success"`
	CELExpression string      `json:"celExpression,omitempty"`
	ResourceType  string      `json:"resourceType,omitempty"`
	ResourceID    string      `json:"resourceId,omitempty"`
	PolicyName    string      `json:"policyName,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// =============================================================================
// MOCK LOGGER FOR TESTING
// =============================================================================

// NoOpLogger is a logger that does nothing (for testing).
type NoOpLogger struct{}

func (l *NoOpLogger) Info(msg string, fields ...any)  {}
func (l *NoOpLogger) Warn(msg string, fields ...any)  {}
func (l *NoOpLogger) Error(msg string, fields ...any) {}
