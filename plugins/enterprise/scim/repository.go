package scim

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Repository handles SCIM data persistence
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new SCIM repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Provisioning Token operations

// CreateProvisioningToken creates a new provisioning token
func (r *Repository) CreateProvisioningToken(ctx context.Context, token *ProvisioningToken) error {
	_, err := r.db.NewInsert().
		Model(token).
		Exec(ctx)
	return err
}

// FindProvisioningTokenByPrefix finds a token by its prefix
func (r *Repository) FindProvisioningTokenByPrefix(ctx context.Context, prefix string) (*ProvisioningToken, error) {
	var token ProvisioningToken
	err := r.db.NewSelect().
		Model(&token).
		Where("token_prefix = ?", prefix).
		Where("revoked_at IS NULL").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	return &token, nil
}

// FindProvisioningTokenByID finds a token by ID
func (r *Repository) FindProvisioningTokenByID(ctx context.Context, id xid.ID) (*ProvisioningToken, error) {
	var token ProvisioningToken
	err := r.db.NewSelect().
		Model(&token).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	return &token, nil
}

// ListProvisioningTokens lists all provisioning tokens for an organization
// Updated for 3-tier architecture: App → Environment → Organization
func (r *Repository) ListProvisioningTokens(ctx context.Context, appID, envID, orgID xid.ID, limit, offset int) ([]*ProvisioningToken, error) {
	var tokens []*ProvisioningToken
	err := r.db.NewSelect().
		Model(&tokens).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("revoked_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	return tokens, err
}

// UpdateProvisioningToken updates a provisioning token
func (r *Repository) UpdateProvisioningToken(ctx context.Context, token *ProvisioningToken) error {
	token.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(token).
		WherePK().
		Exec(ctx)

	return err
}

// RevokeProvisioningToken revokes a provisioning token
func (r *Repository) RevokeProvisioningToken(ctx context.Context, id xid.ID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*ProvisioningToken)(nil)).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// CountProvisioningTokens counts active tokens for an organization
// Updated for 3-tier architecture
func (r *Repository) CountProvisioningTokens(ctx context.Context, appID, envID, orgID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*ProvisioningToken)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("revoked_at IS NULL").
		Count(ctx)

	return count, err
}

// Provisioning Log operations

// CreateProvisioningLog creates a new provisioning log entry
func (r *Repository) CreateProvisioningLog(ctx context.Context, log *ProvisioningLog) error {
	_, err := r.db.NewInsert().
		Model(log).
		Exec(ctx)
	return err
}

// ListProvisioningLogs lists provisioning logs with filtering
// Updated for 3-tier architecture
func (r *Repository) ListProvisioningLogs(ctx context.Context, appID, envID, orgID xid.ID, filters map[string]interface{}, limit, offset int) ([]*ProvisioningLog, error) {
	query := r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID)

	// Apply filters
	if operation, ok := filters["operation"].(string); ok && operation != "" {
		query = query.Where("operation = ?", operation)
	}
	if resourceType, ok := filters["resource_type"].(string); ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if success, ok := filters["success"].(bool); ok {
		query = query.Where("success = ?", success)
	}
	if startDate, ok := filters["start_date"].(time.Time); ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		query = query.Where("created_at <= ?", endDate)
	}

	var logs []*ProvisioningLog
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &logs)

	return logs, err
}

// CountProvisioningLogs counts provisioning logs with filtering
func (r *Repository) CountProvisioningLogs(ctx context.Context, appID, envID, orgID xid.ID, filters map[string]interface{}) (int, error) {
	query := r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID)

	// Apply same filters as ListProvisioningLogs
	if operation, ok := filters["operation"].(string); ok && operation != "" {
		query = query.Where("operation = ?", operation)
	}
	if resourceType, ok := filters["resource_type"].(string); ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if success, ok := filters["success"].(bool); ok {
		query = query.Where("success = ?", success)
	}
	if startDate, ok := filters["start_date"].(time.Time); ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		query = query.Where("created_at <= ?", endDate)
	}

	return query.Count(ctx)
}

// GetProvisioningStats returns provisioning statistics
func (r *Repository) GetProvisioningStats(ctx context.Context, appID, envID, orgID xid.ID, startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total operations
	totalCount, err := r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Count(ctx)

	if err != nil {
		return nil, err
	}
	stats["total_operations"] = totalCount

	// Success rate
	successCount, err := r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Where("success = ?", true).
		Count(ctx)

	if err != nil {
		return nil, err
	}
	stats["successful_operations"] = successCount
	stats["failed_operations"] = totalCount - successCount

	if totalCount > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalCount) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	// Operations by type
	type OperationCount struct {
		Operation string `bun:"operation"`
		Count     int    `bun:"count"`
	}

	var operationCounts []OperationCount
	err = r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		Column("operation").
		ColumnExpr("COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Group("operation").
		Scan(ctx, &operationCounts)

	if err != nil {
		return nil, err
	}

	operationStats := make(map[string]int)
	for _, oc := range operationCounts {
		operationStats[oc.Operation] = oc.Count
	}
	stats["operations_by_type"] = operationStats

	// Average duration
	var avgDuration float64
	err = r.db.NewSelect().
		Model((*ProvisioningLog)(nil)).
		ColumnExpr("AVG(duration_ms) as avg_duration").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Scan(ctx, &avgDuration)

	if err != nil {
		return nil, err
	}
	stats["average_duration_ms"] = avgDuration

	return stats, nil
}

// Attribute Mapping operations

// CreateAttributeMapping creates a new attribute mapping
func (r *Repository) CreateAttributeMapping(ctx context.Context, mapping *AttributeMapping) error {
	_, err := r.db.NewInsert().
		Model(mapping).
		Exec(ctx)
	return err
}

// GetAttributeMapping gets attribute mapping for an organization
func (r *Repository) GetAttributeMapping(ctx context.Context, appID, envID, orgID xid.ID) (*AttributeMapping, error) {
	var mapping AttributeMapping
	err := r.db.NewSelect().
		Model(&mapping).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("attribute mapping not found: %w", err)
	}

	return &mapping, nil
}

// UpdateAttributeMapping updates attribute mapping
func (r *Repository) UpdateAttributeMapping(ctx context.Context, mapping *AttributeMapping) error {
	mapping.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(mapping).
		WherePK().
		Exec(ctx)

	return err
}

// Group Mapping operations

// CreateGroupMapping creates a new group mapping
func (r *Repository) CreateGroupMapping(ctx context.Context, mapping *GroupMapping) error {
	_, err := r.db.NewInsert().
		Model(mapping).
		Exec(ctx)
	return err
}

// FindGroupMapping finds a group mapping by SCIM group ID
func (r *Repository) FindGroupMapping(ctx context.Context, appID, envID, orgID xid.ID, scimGroupID string) (*GroupMapping, error) {
	var mapping GroupMapping
	err := r.db.NewSelect().
		Model(&mapping).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("scim_group_id = ?", scimGroupID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("group mapping not found: %w", err)
	}

	return &mapping, nil
}

// ListGroupMappings lists all group mappings for an organization
func (r *Repository) ListGroupMappings(ctx context.Context, appID, envID, orgID xid.ID) ([]*GroupMapping, error) {
	var mappings []*GroupMapping
	err := r.db.NewSelect().
		Model(&mappings).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)

	return mappings, err
}

// UpdateGroupMapping updates a group mapping
func (r *Repository) UpdateGroupMapping(ctx context.Context, mapping *GroupMapping) error {
	mapping.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(mapping).
		WherePK().
		Exec(ctx)

	return err
}

// DeleteGroupMapping deletes a group mapping
func (r *Repository) DeleteGroupMapping(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*GroupMapping)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Migrate runs database migrations
func (r *Repository) Migrate(ctx context.Context) error {
	// Create provisioning_tokens table
	if _, err := r.db.NewCreateTable().
		Model((*ProvisioningToken)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create provisioning_tokens table: %w", err)
	}

	// Create indexes for provisioning_tokens (3-tier architecture)
	if _, err := r.db.NewCreateIndex().
		Model((*ProvisioningToken)(nil)).
		Index("idx_provisioning_tokens_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if _, err := r.db.NewCreateIndex().
		Model((*ProvisioningToken)(nil)).
		Index("idx_provisioning_tokens_prefix").
		Column("token_prefix").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create provisioning_logs table
	if _, err := r.db.NewCreateTable().
		Model((*ProvisioningLog)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create provisioning_logs table: %w", err)
	}

	// Create indexes for provisioning_logs (3-tier architecture)
	if _, err := r.db.NewCreateIndex().
		Model((*ProvisioningLog)(nil)).
		Index("idx_provisioning_logs_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if _, err := r.db.NewCreateIndex().
		Model((*ProvisioningLog)(nil)).
		Index("idx_provisioning_logs_created_at").
		Column("created_at").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if _, err := r.db.NewCreateIndex().
		Model((*ProvisioningLog)(nil)).
		Index("idx_provisioning_logs_operation").
		Column("operation").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create attribute_mappings table
	if _, err := r.db.NewCreateTable().
		Model((*AttributeMapping)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create attribute_mappings table: %w", err)
	}

	// Create indexes for attribute_mappings (3-tier architecture)
	// Unique constraint on app/env/org combination
	if _, err := r.db.NewCreateIndex().
		Model((*AttributeMapping)(nil)).
		Index("idx_attribute_mappings_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		Unique().
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create group_mappings table
	if _, err := r.db.NewCreateTable().
		Model((*GroupMapping)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create group_mappings table: %w", err)
	}

	// Create indexes for group_mappings (3-tier architecture)
	if _, err := r.db.NewCreateIndex().
		Model((*GroupMapping)(nil)).
		Index("idx_group_mappings_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Unique constraint on app/env/org/scim_group_id combination
	if _, err := r.db.NewCreateIndex().
		Model((*GroupMapping)(nil)).
		Index("idx_group_mappings_scim_group_id").
		Column("app_id", "environment_id", "organization_id", "scim_group_id").
		Unique().
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	fmt.Println("[SCIM] Database migrations completed successfully")
	return nil
}

// Ping checks database connectivity
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.Ping()
}

// FindGroupMappingByTargetID finds a group mapping by target team ID
func (r *Repository) FindGroupMappingByTargetID(ctx context.Context, targetID xid.ID) (*GroupMapping, error) {
	var mapping GroupMapping
	err := r.db.NewSelect().
		Model(&mapping).
		Where("target_id = ?", targetID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("group mapping not found: %w", err)
	}

	return &mapping, nil
}

// FindGroupMappingBySCIMID finds a group mapping by SCIM group ID
// Updated for 3-tier architecture
func (r *Repository) FindGroupMappingBySCIMID(ctx context.Context, appID, envID, orgID xid.ID, scimGroupID string) (*GroupMapping, error) {
	var mapping GroupMapping
	err := r.db.NewSelect().
		Model(&mapping).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Where("scim_group_id = ?", scimGroupID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("group mapping not found: %w", err)
	}

	return &mapping, nil
}

// FindAttributeMappingByOrgID finds attribute mapping by organization ID
func (r *Repository) FindAttributeMappingByOrganization(ctx context.Context, appID, envID, orgID xid.ID) (*AttributeMapping, error) {
	var mapping AttributeMapping
	err := r.db.NewSelect().
		Model(&mapping).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("attribute mapping not found: %w", err)
	}

	return &mapping, nil
}

// UpdateTeamProvisioningInfo updates team provisioning tracking fields
// This method updates both app teams and organization teams
func (r *Repository) UpdateTeamProvisioningInfo(ctx context.Context, teamID xid.ID, provisionedBy, externalID *string) error {
	// Try updating app teams first
	result, err := r.db.NewUpdate().
		Table("teams").
		Set("provisioned_by = ?", provisionedBy).
		Set("external_id = ?", externalID).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", teamID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update app team: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no app team found, try organization teams
	if rowsAffected == 0 {
		result, err = r.db.NewUpdate().
			Table("organization_teams").
			Set("provisioned_by = ?", provisionedBy).
			Set("external_id = ?", externalID).
			Set("updated_at = ?", time.Now()).
			Where("id = ?", teamID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("failed to update organization team: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("team not found: %s", teamID)
		}
	}

	return nil
}

// UpdateTeamMemberProvisioningInfo updates team member provisioning tracking field
// This method updates both app team members and organization team members
func (r *Repository) UpdateTeamMemberProvisioningInfo(ctx context.Context, teamID, memberID xid.ID, provisionedBy *string) error {
	// Try updating app team members first
	result, err := r.db.NewUpdate().
		Table("team_members").
		Set("provisioned_by = ?", provisionedBy).
		Set("updated_at = ?", time.Now()).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update app team member: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no app team member found, try organization team members
	if rowsAffected == 0 {
		result, err = r.db.NewUpdate().
			Table("organization_team_members").
			Set("provisioned_by = ?", provisionedBy).
			Set("updated_at = ?", time.Now()).
			Where("team_id = ? AND member_id = ?", teamID, memberID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("failed to update organization team member: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("team member not found: team=%s, member=%s", teamID, memberID)
		}
	}

	return nil
}
