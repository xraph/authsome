package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/schema"
)

// Repository interface defined in interfaces.go

// bunRepository is a Bun-based repository implementation
// V2 Architecture: App → Environment → Organization.
type bunRepository struct {
	db *bun.DB
}

// NewRepository creates a new Bun repository.
func NewRepository(db *bun.DB) Repository {
	return &bunRepository{db: db}
}

// =============================================================================
// POLICY OPERATIONS
// =============================================================================

// CreatePolicy creates a new policy in the database.
func (r *bunRepository) CreatePolicy(ctx context.Context, policy *core.Policy) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbPolicy := &schema.PermissionPolicy{
		ID:                 policy.ID,
		AppID:              policy.AppID,
		EnvironmentID:      policy.EnvironmentID,
		UserOrganizationID: policy.UserOrganizationID,
		NamespaceID:        policy.NamespaceID,
		Name:               policy.Name,
		Description:        policy.Description,
		Expression:         policy.Expression,
		ResourceType:       policy.ResourceType,
		Actions:            policy.Actions,
		Priority:           policy.Priority,
		Enabled:            policy.Enabled,
		Version:            policy.Version,
		CreatedBy:          policy.CreatedBy,
		CreatedAt:          policy.CreatedAt,
		UpdatedAt:          policy.UpdatedAt,
	}

	_, err := r.db.NewInsert().Model(dbPolicy).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert policy: %w", err)
	}

	return nil
}

// GetPolicy retrieves a policy by ID.
func (r *bunRepository) GetPolicy(ctx context.Context, id xid.ID) (*core.Policy, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbPolicy := new(schema.PermissionPolicy)

	err := r.db.NewSelect().
		Model(dbPolicy).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return toCorePolicy(dbPolicy), nil
}

// ListPolicies lists policies with filters.
func (r *bunRepository) ListPolicies(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, filters PolicyFilters) ([]*core.Policy, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbPolicies []schema.PermissionPolicy

	query := r.db.NewSelect().
		Model(&dbPolicies).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	// Handle organization scope
	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("user_organization_id = ?", *userOrgID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	// Apply filters
	if filters.ResourceType != nil && *filters.ResourceType != "" {
		query = query.Where("resource_type = ?", *filters.ResourceType)
	}

	if filters.Enabled != nil {
		query = query.Where("enabled = ?", *filters.Enabled)
	}

	if filters.NamespaceID != nil && !filters.NamespaceID.IsNil() {
		query = query.Where("namespace_id = ?", *filters.NamespaceID)
	}

	// Pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by priority (highest first) then by created_at
	query = query.Order("priority DESC", "created_at DESC")

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	// Convert to core types
	policies := make([]*core.Policy, len(dbPolicies))
	for i, p := range dbPolicies {
		policies[i] = toCorePolicy(&p)
	}

	return policies, nil
}

// UpdatePolicy updates an existing policy.
func (r *bunRepository) UpdatePolicy(ctx context.Context, policy *core.Policy) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbPolicy := &schema.PermissionPolicy{
		ID:                 policy.ID,
		AppID:              policy.AppID,
		EnvironmentID:      policy.EnvironmentID,
		UserOrganizationID: policy.UserOrganizationID,
		NamespaceID:        policy.NamespaceID,
		Name:               policy.Name,
		Description:        policy.Description,
		Expression:         policy.Expression,
		ResourceType:       policy.ResourceType,
		Actions:            policy.Actions,
		Priority:           policy.Priority,
		Enabled:            policy.Enabled,
		Version:            policy.Version,
		CreatedBy:          policy.CreatedBy,
		CreatedAt:          policy.CreatedAt,
		UpdatedAt:          policy.UpdatedAt,
	}

	_, err := r.db.NewUpdate().
		Model(dbPolicy).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

// DeletePolicy deletes a policy.
func (r *bunRepository) DeletePolicy(ctx context.Context, id xid.ID) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	_, err := r.db.NewDelete().
		Model((*schema.PermissionPolicy)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

// GetPoliciesByResourceType retrieves active policies for a specific resource type.
func (r *bunRepository) GetPoliciesByResourceType(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, resourceType string) ([]*core.Policy, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbPolicies []schema.PermissionPolicy

	query := r.db.NewSelect().
		Model(&dbPolicies).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("resource_type = ?", resourceType).
		Where("enabled = ?", true)

	// Handle organization scope - include both org-specific and environment-level policies
	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("(user_organization_id = ? OR user_organization_id IS NULL)", *userOrgID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	query = query.Order("priority DESC")

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by resource type: %w", err)
	}

	policies := make([]*core.Policy, len(dbPolicies))
	for i, p := range dbPolicies {
		policies[i] = toCorePolicy(&p)
	}

	return policies, nil
}

// GetActivePolicies retrieves all active policies for a scope.
func (r *bunRepository) GetActivePolicies(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID) ([]*core.Policy, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	enabled := true

	return r.ListPolicies(ctx, appID, envID, userOrgID, PolicyFilters{
		Enabled: &enabled,
		Limit:   1000, // Reasonable limit
	})
}

// =============================================================================
// NAMESPACE OPERATIONS
// =============================================================================

// CreateNamespace creates a new namespace.
func (r *bunRepository) CreateNamespace(ctx context.Context, ns *core.Namespace) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbNs := &schema.PermissionNamespace{
		ID:                 ns.ID,
		AppID:              ns.AppID,
		EnvironmentID:      ns.EnvironmentID,
		UserOrganizationID: ns.UserOrganizationID,
		Name:               ns.Name,
		Description:        ns.Description,
		TemplateID:         ns.TemplateID,
		InheritPlatform:    ns.InheritPlatform,
		CreatedAt:          ns.CreatedAt,
		UpdatedAt:          ns.UpdatedAt,
	}

	_, err := r.db.NewInsert().Model(dbNs).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert namespace: %w", err)
	}

	return nil
}

// GetNamespace retrieves a namespace by ID.
func (r *bunRepository) GetNamespace(ctx context.Context, id xid.ID) (*core.Namespace, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbNs := new(schema.PermissionNamespace)

	err := r.db.NewSelect().
		Model(dbNs).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return toCoreNamespace(dbNs), nil
}

// GetNamespaceByScope retrieves a namespace by scope.
func (r *bunRepository) GetNamespaceByScope(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID) (*core.Namespace, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbNs := new(schema.PermissionNamespace)
	query := r.db.NewSelect().
		Model(dbNs).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("user_organization_id = ?", *userOrgID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	err := query.Limit(1).Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get namespace by scope: %w", err)
	}

	return toCoreNamespace(dbNs), nil
}

// ListNamespaces lists namespaces for a scope.
func (r *bunRepository) ListNamespaces(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID) ([]*core.Namespace, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbNamespaces []schema.PermissionNamespace

	query := r.db.NewSelect().
		Model(&dbNamespaces).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("(user_organization_id = ? OR user_organization_id IS NULL)", *userOrgID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	query = query.Order("name ASC")

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := make([]*core.Namespace, len(dbNamespaces))
	for i, ns := range dbNamespaces {
		namespaces[i] = toCoreNamespace(&ns)
	}

	return namespaces, nil
}

// UpdateNamespace updates a namespace.
func (r *bunRepository) UpdateNamespace(ctx context.Context, ns *core.Namespace) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbNs := &schema.PermissionNamespace{
		ID:                 ns.ID,
		AppID:              ns.AppID,
		EnvironmentID:      ns.EnvironmentID,
		UserOrganizationID: ns.UserOrganizationID,
		Name:               ns.Name,
		Description:        ns.Description,
		TemplateID:         ns.TemplateID,
		InheritPlatform:    ns.InheritPlatform,
		CreatedAt:          ns.CreatedAt,
		UpdatedAt:          ns.UpdatedAt,
	}

	_, err := r.db.NewUpdate().
		Model(dbNs).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update namespace: %w", err)
	}

	return nil
}

// DeleteNamespace deletes a namespace.
func (r *bunRepository) DeleteNamespace(ctx context.Context, id xid.ID) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	_, err := r.db.NewDelete().
		Model((*schema.PermissionNamespace)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// =============================================================================
// RESOURCE DEFINITION OPERATIONS
// =============================================================================

// CreateResourceDefinition creates a new resource definition.
func (r *bunRepository) CreateResourceDefinition(ctx context.Context, res *core.ResourceDefinition) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbRes := &schema.PermissionResource{
		ID:          res.ID,
		NamespaceID: res.NamespaceID,
		Type:        res.Type,
		Description: res.Description,
		Attributes:  toSchemaAttributes(res.Attributes),
		CreatedAt:   res.CreatedAt,
	}

	_, err := r.db.NewInsert().Model(dbRes).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert resource definition: %w", err)
	}

	return nil
}

// GetResourceDefinition retrieves a resource definition by ID.
func (r *bunRepository) GetResourceDefinition(ctx context.Context, id xid.ID) (*core.ResourceDefinition, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbRes := new(schema.PermissionResource)

	err := r.db.NewSelect().
		Model(dbRes).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get resource definition: %w", err)
	}

	return toCoreResourceDefinition(dbRes), nil
}

// ListResourceDefinitions lists resource definitions for a namespace.
func (r *bunRepository) ListResourceDefinitions(ctx context.Context, namespaceID xid.ID) ([]*core.ResourceDefinition, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbResources []schema.PermissionResource

	err := r.db.NewSelect().
		Model(&dbResources).
		Where("namespace_id = ?", namespaceID).
		Order("type ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource definitions: %w", err)
	}

	resources := make([]*core.ResourceDefinition, len(dbResources))
	for i, res := range dbResources {
		resources[i] = toCoreResourceDefinition(&res)
	}

	return resources, nil
}

// DeleteResourceDefinition deletes a resource definition.
func (r *bunRepository) DeleteResourceDefinition(ctx context.Context, id xid.ID) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	_, err := r.db.NewDelete().
		Model((*schema.PermissionResource)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete resource definition: %w", err)
	}

	return nil
}

// =============================================================================
// ACTION DEFINITION OPERATIONS
// =============================================================================

// CreateActionDefinition creates a new action definition.
func (r *bunRepository) CreateActionDefinition(ctx context.Context, action *core.ActionDefinition) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbAction := &schema.PermissionAction{
		ID:          action.ID,
		NamespaceID: action.NamespaceID,
		Name:        action.Name,
		Description: action.Description,
		CreatedAt:   action.CreatedAt,
	}

	_, err := r.db.NewInsert().Model(dbAction).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert action definition: %w", err)
	}

	return nil
}

// GetActionDefinition retrieves an action definition by ID.
func (r *bunRepository) GetActionDefinition(ctx context.Context, id xid.ID) (*core.ActionDefinition, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbAction := new(schema.PermissionAction)

	err := r.db.NewSelect().
		Model(dbAction).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get action definition: %w", err)
	}

	return toCoreActionDefinition(dbAction), nil
}

// ListActionDefinitions lists action definitions for a namespace.
func (r *bunRepository) ListActionDefinitions(ctx context.Context, namespaceID xid.ID) ([]*core.ActionDefinition, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbActions []schema.PermissionAction

	err := r.db.NewSelect().
		Model(&dbActions).
		Where("namespace_id = ?", namespaceID).
		Order("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list action definitions: %w", err)
	}

	actions := make([]*core.ActionDefinition, len(dbActions))
	for i, action := range dbActions {
		actions[i] = toCoreActionDefinition(&action)
	}

	return actions, nil
}

// DeleteActionDefinition deletes an action definition.
func (r *bunRepository) DeleteActionDefinition(ctx context.Context, id xid.ID) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	_, err := r.db.NewDelete().
		Model((*schema.PermissionAction)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete action definition: %w", err)
	}

	return nil
}

// =============================================================================
// AUDIT OPERATIONS
// =============================================================================

// CreateAuditEvent creates an audit event.
func (r *bunRepository) CreateAuditEvent(ctx context.Context, event *core.AuditEvent) error {
	if r.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	dbEvent := &schema.PermissionAuditLog{
		ID:                 event.ID,
		AppID:              event.AppID,
		EnvironmentID:      event.EnvironmentID,
		UserOrganizationID: event.UserOrganizationID,
		ActorID:            event.ActorID,
		Action:             event.Action,
		ResourceType:       event.ResourceType,
		ResourceID:         event.ResourceID,
		OldValue:           event.OldValue,
		NewValue:           event.NewValue,
		IPAddress:          event.IPAddress,
		UserAgent:          event.UserAgent,
		Timestamp:          event.Timestamp,
	}

	_, err := r.db.NewInsert().Model(dbEvent).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert audit event: %w", err)
	}

	return nil
}

// ListAuditEvents lists audit events with filters.
func (r *bunRepository) ListAuditEvents(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, filters AuditFilters) ([]*core.AuditEvent, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	var dbEvents []schema.PermissionAuditLog

	query := r.db.NewSelect().
		Model(&dbEvents).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	// Handle organization scope
	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("(user_organization_id = ? OR user_organization_id IS NULL)", *userOrgID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	// Apply filters
	if filters.ActorID != nil && !filters.ActorID.IsNil() {
		query = query.Where("actor_id = ?", *filters.ActorID)
	}

	if filters.Action != nil && *filters.Action != "" {
		query = query.Where("action = ?", *filters.Action)
	}

	if filters.ResourceType != nil && *filters.ResourceType != "" {
		query = query.Where("resource_type = ?", *filters.ResourceType)
	}

	if filters.StartTime != nil {
		query = query.Where("timestamp >= ?", *filters.StartTime)
	}

	if filters.EndTime != nil {
		query = query.Where("timestamp <= ?", *filters.EndTime)
	}

	// Pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by timestamp descending (most recent first)
	query = query.Order("timestamp DESC")

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	events := make([]*core.AuditEvent, len(dbEvents))
	for i, e := range dbEvents {
		events[i] = toCoreAuditEvent(&e)
	}

	return events, nil
}

// =============================================================================
// TYPE CONVERSION HELPERS
// =============================================================================

// toCorePolicy converts a schema policy to a core policy.
func toCorePolicy(p *schema.PermissionPolicy) *core.Policy {
	return &core.Policy{
		ID:                 p.ID,
		AppID:              p.AppID,
		EnvironmentID:      p.EnvironmentID,
		UserOrganizationID: p.UserOrganizationID,
		NamespaceID:        p.NamespaceID,
		Name:               p.Name,
		Description:        p.Description,
		Expression:         p.Expression,
		ResourceType:       p.ResourceType,
		Actions:            p.Actions,
		Priority:           p.Priority,
		Enabled:            p.Enabled,
		Version:            p.Version,
		CreatedBy:          p.CreatedBy,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

// toCoreNamespace converts a schema namespace to a core namespace.
func toCoreNamespace(ns *schema.PermissionNamespace) *core.Namespace {
	return &core.Namespace{
		ID:                 ns.ID,
		AppID:              ns.AppID,
		EnvironmentID:      ns.EnvironmentID,
		UserOrganizationID: ns.UserOrganizationID,
		Name:               ns.Name,
		Description:        ns.Description,
		TemplateID:         ns.TemplateID,
		InheritPlatform:    ns.InheritPlatform,
		CreatedAt:          ns.CreatedAt,
		UpdatedAt:          ns.UpdatedAt,
	}
}

// toCoreResourceDefinition converts a schema resource to a core resource definition.
func toCoreResourceDefinition(res *schema.PermissionResource) *core.ResourceDefinition {
	return &core.ResourceDefinition{
		ID:          res.ID,
		NamespaceID: res.NamespaceID,
		Type:        res.Type,
		Description: res.Description,
		Attributes:  toCoreAttributes(res.Attributes),
		CreatedAt:   res.CreatedAt,
	}
}

// toCoreActionDefinition converts a schema action to a core action definition.
func toCoreActionDefinition(action *schema.PermissionAction) *core.ActionDefinition {
	return &core.ActionDefinition{
		ID:          action.ID,
		NamespaceID: action.NamespaceID,
		Name:        action.Name,
		Description: action.Description,
		CreatedAt:   action.CreatedAt,
	}
}

// toCoreAuditEvent converts a schema audit log to a core audit event.
func toCoreAuditEvent(e *schema.PermissionAuditLog) *core.AuditEvent {
	return &core.AuditEvent{
		ID:                 e.ID,
		AppID:              e.AppID,
		EnvironmentID:      e.EnvironmentID,
		UserOrganizationID: e.UserOrganizationID,
		ActorID:            e.ActorID,
		Action:             e.Action,
		ResourceType:       e.ResourceType,
		ResourceID:         e.ResourceID,
		OldValue:           e.OldValue,
		NewValue:           e.NewValue,
		IPAddress:          e.IPAddress,
		UserAgent:          e.UserAgent,
		Timestamp:          e.Timestamp,
	}
}

// toSchemaAttributes converts core attributes to schema attributes.
func toSchemaAttributes(attrs []core.ResourceAttribute) []schema.ResourceAttribute {
	result := make([]schema.ResourceAttribute, len(attrs))
	for i, a := range attrs {
		result[i] = schema.ResourceAttribute{
			Name:        a.Name,
			Type:        a.Type,
			Required:    a.Required,
			Default:     a.Default,
			Description: a.Description,
		}
	}

	return result
}

// toCoreAttributes converts schema attributes to core attributes.
func toCoreAttributes(attrs []schema.ResourceAttribute) []core.ResourceAttribute {
	result := make([]core.ResourceAttribute, len(attrs))
	for i, a := range attrs {
		result[i] = core.ResourceAttribute{
			Name:        a.Name,
			Type:        a.Type,
			Required:    a.Required,
			Default:     a.Default,
			Description: a.Description,
		}
	}

	return result
}

// =============================================================================
// ANALYTICS OPERATIONS
// =============================================================================

// GetEvaluationStats retrieves aggregated evaluation statistics.
func (r *bunRepository) GetEvaluationStats(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, timeRange map[string]any) (*EvaluationStats, error) {
	if r.db == nil {
		return nil, errs.InternalServerErrorWithMessage("database not initialized")
	}

	// Check if the stats table exists - for now return empty stats
	// In a full implementation, you would query the PermissionEvaluationStats table
	stats := &EvaluationStats{
		TotalEvaluations: 0,
		AllowedCount:     0,
		DeniedCount:      0,
		AvgLatencyMs:     0,
		CacheHits:        0,
		CacheMisses:      0,
	}

	// Try to query the stats table if it exists
	var result struct {
		TotalEvaluations int64   `bun:"total_evaluations"`
		AllowedCount     int64   `bun:"allowed_count"`
		DeniedCount      int64   `bun:"denied_count"`
		AvgLatencyMs     float64 `bun:"avg_latency_ms"`
		CacheHits        int64   `bun:"cache_hits"`
		CacheMisses      int64   `bun:"cache_misses"`
	}

	query := r.db.NewSelect().
		TableExpr("permission_evaluation_stats").
		ColumnExpr("SUM(evaluation_count) as total_evaluations").
		ColumnExpr("SUM(allowed_count) as allowed_count").
		ColumnExpr("SUM(denied_count) as denied_count").
		ColumnExpr("AVG(avg_latency_ms) as avg_latency_ms").
		ColumnExpr("SUM(cache_hits) as cache_hits").
		ColumnExpr("SUM(cache_misses) as cache_misses").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	// Handle organization scope
	if userOrgID != nil && !userOrgID.IsNil() {
		query = query.Where("(user_organization_id = ? OR user_organization_id IS NULL)", *userOrgID)
	}

	// Apply time range filter if provided
	if startTime, ok := timeRange["startTime"].(string); ok && startTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, startTime)
		if err == nil {
			query = query.Where("created_at >= ?", parsedTime)
		}
	}

	if endTime, ok := timeRange["endTime"].(string); ok && endTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, endTime)
		if err == nil {
			query = query.Where("created_at <= ?", parsedTime)
		}
	}

	err := query.Scan(ctx, &result)
	if err != nil {
		// Table might not exist yet, return empty stats
		return stats, nil
	}

	stats.TotalEvaluations = result.TotalEvaluations
	stats.AllowedCount = result.AllowedCount
	stats.DeniedCount = result.DeniedCount
	stats.AvgLatencyMs = result.AvgLatencyMs
	stats.CacheHits = result.CacheHits
	stats.CacheMisses = result.CacheMisses

	return stats, nil
}

// Ensure unused import is used.
var _ = time.Now
