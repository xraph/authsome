package handlers

import (
	"time"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/plugins/permissions/core"
)

// =============================================================================
// SHARED RESPONSE TYPES
// =============================================================================

// Use shared response types from core.
type (
	MessageResponse = responses.MessageResponse
	StatusResponse  = responses.StatusResponse
	ErrorResponse   = responses.ErrorResponse
)

// =============================================================================
// POLICY DTOs
// =============================================================================

// CreatePolicyRequest represents a request to create a new policy.
type CreatePolicyRequest struct {
	NamespaceID  string   `json:"namespaceId"  validate:"required"`
	Name         string   `json:"name"         validate:"required,min=3,max=100"`
	Description  string   `json:"description"  validate:"max=500"`
	Expression   string   `json:"expression"   validate:"required"`
	ResourceType string   `json:"resourceType" validate:"required"`
	Actions      []string `json:"actions"      validate:"required,min=1"`
	Priority     int      `json:"priority"     validate:"min=0,max=1000"`
	Enabled      bool     `json:"enabled"`
}

// UpdatePolicyRequest represents a request to update an existing policy.
type UpdatePolicyRequest struct {
	Name         string   `json:"name,omitempty"         validate:"omitempty,min=3,max=100"`
	Description  string   `json:"description,omitempty"  validate:"omitempty,max=500"`
	Expression   string   `json:"expression,omitempty"`
	ResourceType string   `json:"resourceType,omitempty"`
	Actions      []string `json:"actions,omitempty"`
	Priority     int      `json:"priority,omitempty"     validate:"omitempty,min=0,max=1000"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// PolicyResponse represents a single policy response.
type PolicyResponse struct {
	ID                 string    `json:"id"`
	AppID              string    `json:"appId"`
	EnvironmentID      string    `json:"environmentId"`
	UserOrganizationID *string   `json:"userOrganizationId,omitempty"`
	NamespaceID        string    `json:"namespaceId"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Expression         string    `json:"expression"`
	ResourceType       string    `json:"resourceType"`
	Actions            []string  `json:"actions"`
	Priority           int       `json:"priority"`
	Enabled            bool      `json:"enabled"`
	Version            int       `json:"version"`
	CreatedBy          string    `json:"createdBy"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// PoliciesListResponse represents a list of policies.
type PoliciesListResponse struct {
	Policies   []*PolicyResponse `json:"policies"`
	TotalCount int               `json:"totalCount"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
}

// =============================================================================
// RESOURCE DTOs
// =============================================================================

// CreateResourceRequest represents a request to create a resource definition.
type CreateResourceRequest struct {
	NamespaceID string                     `json:"namespaceId" validate:"required"`
	Type        string                     `json:"type"        validate:"required,min=3,max=50"`
	Description string                     `json:"description" validate:"max=500"`
	Attributes  []ResourceAttributeRequest `json:"attributes"  validate:"required,min=1"`
}

// ResourceAttributeRequest represents an attribute in a create/update request.
type ResourceAttributeRequest struct {
	Name        string `json:"name"                  validate:"required,min=1,max=50"`
	Type        string `json:"type"                  validate:"required,oneof=string int bool array object"`
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty" validate:"max=200"`
}

// ResourceAttributeInput is an alias for ResourceAttributeRequest for backwards compatibility.
type ResourceAttributeInput = ResourceAttributeRequest

// ResourceResponse represents a single resource definition response.
type ResourceResponse struct {
	ID          string                   `json:"id"`
	NamespaceID string                   `json:"namespaceId"`
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Attributes  []core.ResourceAttribute `json:"attributes"`
	CreatedAt   time.Time                `json:"createdAt"`
}

// ResourcesListResponse represents a list of resource definitions.
type ResourcesListResponse struct {
	Resources  []*ResourceResponse `json:"resources"`
	TotalCount int                 `json:"totalCount"`
}

// =============================================================================
// ACTION DTOs
// =============================================================================

// CreateActionRequest represents a request to create an action definition.
type CreateActionRequest struct {
	NamespaceID string `json:"namespaceId" validate:"required"`
	Name        string `json:"name"        validate:"required,min=3,max=50"`
	Description string `json:"description" validate:"max=500"`
}

// ActionResponse represents a single action definition response.
type ActionResponse struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespaceId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ActionsListResponse represents a list of action definitions.
type ActionsListResponse struct {
	Actions    []*ActionResponse `json:"actions"`
	TotalCount int               `json:"totalCount"`
}

// =============================================================================
// NAMESPACE DTOs
// =============================================================================

// CreateNamespaceRequest represents a request to create a namespace.
type CreateNamespaceRequest struct {
	Name            string `json:"name"                 validate:"required,min=3,max=100"`
	Description     string `json:"description"          validate:"max=500"`
	TemplateID      string `json:"templateId,omitempty"`
	InheritPlatform bool   `json:"inheritPlatform"`
}

// UpdateNamespaceRequest represents a request to update a namespace.
type UpdateNamespaceRequest struct {
	Name            string `json:"name,omitempty"            validate:"omitempty,min=3,max=100"`
	Description     string `json:"description,omitempty"     validate:"omitempty,max=500"`
	InheritPlatform *bool  `json:"inheritPlatform,omitempty"`
}

// NamespaceResponse represents a single namespace response.
type NamespaceResponse struct {
	ID                 string    `json:"id"`
	AppID              string    `json:"appId"`
	EnvironmentID      string    `json:"environmentId"`
	UserOrganizationID *string   `json:"userOrganizationId,omitempty"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	TemplateID         *string   `json:"templateId,omitempty"`
	InheritPlatform    bool      `json:"inheritPlatform"`
	ResourceCount      int       `json:"resourceCount"`
	ActionCount        int       `json:"actionCount"`
	PolicyCount        int       `json:"policyCount"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// NamespacesListResponse represents a list of namespaces.
type NamespacesListResponse struct {
	Namespaces []*NamespaceResponse `json:"namespaces"`
	TotalCount int                  `json:"totalCount"`
}

// =============================================================================
// EVALUATION DTOs
// =============================================================================

// EvaluateRequest represents a request to evaluate a permission.
type EvaluateRequest struct {
	Principal    map[string]any `json:"principal"            validate:"required"`
	Resource     map[string]any `json:"resource"             validate:"required"`
	Request      map[string]any `json:"request,omitempty"`
	Action       string         `json:"action"               validate:"required"`
	ResourceType string         `json:"resourceType"         validate:"required"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

// EvaluateResponse represents the result of a permission evaluation.
type EvaluateResponse struct {
	Allowed           bool     `json:"allowed"`
	MatchedPolicies   []string `json:"matchedPolicies,omitempty"`
	EvaluatedPolicies int      `json:"evaluatedPolicies"`
	EvaluationTimeMs  float64  `json:"evaluationTimeMs"`
	CacheHit          bool     `json:"cacheHit"`
	Error             string   `json:"error,omitempty"`
	Reason            string   `json:"reason,omitempty"`
}

// BatchEvaluateRequest represents a batch evaluation request.
type BatchEvaluateRequest struct {
	Requests []EvaluateRequest `json:"requests" validate:"required,min=1,max=100"`
}

// BatchEvaluationResult represents a single evaluation in a batch.
type BatchEvaluationResult struct {
	Index            int      `json:"index"`
	ResourceType     string   `json:"resourceType"`
	ResourceID       string   `json:"resourceId,omitempty"`
	Action           string   `json:"action"`
	Allowed          bool     `json:"allowed"`
	Policies         []string `json:"policies,omitempty"`
	Error            string   `json:"error,omitempty"`
	EvaluationTimeMs float64  `json:"evaluationTimeMs"`
}

// BatchEvaluateResponse represents the result of a batch evaluation.
type BatchEvaluateResponse struct {
	Results          []*BatchEvaluationResult `json:"results"`
	TotalEvaluations int                      `json:"totalEvaluations"`
	TotalTimeMs      float64                  `json:"totalTimeMs"`
	SuccessCount     int                      `json:"successCount"`
	FailureCount     int                      `json:"failureCount"`
}

// =============================================================================
// VALIDATION & TESTING DTOs
// =============================================================================

// ValidatePolicyRequest represents a request to validate a policy expression.
type ValidatePolicyRequest struct {
	Expression   string `json:"expression"   validate:"required"`
	ResourceType string `json:"resourceType" validate:"required"`
}

// ValidatePolicyResponse represents the result of policy validation.
type ValidatePolicyResponse struct {
	Valid      bool     `json:"valid"`
	Error      string   `json:"error,omitempty"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Complexity int      `json:"complexity,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// TestPolicyRequest represents a request to test a policy with sample data.
type TestPolicyRequest struct {
	Expression   string     `json:"expression"   validate:"required"`
	ResourceType string     `json:"resourceType" validate:"required"`
	Actions      []string   `json:"actions"      validate:"required,min=1"`
	TestCases    []TestCase `json:"testCases"    validate:"required,min=1"`
}

// TestCase represents a single test case for policy testing.
type TestCase struct {
	Name      string         `json:"name"              validate:"required"`
	Principal map[string]any `json:"principal"         validate:"required"`
	Resource  map[string]any `json:"resource"          validate:"required"`
	Request   map[string]any `json:"request,omitempty"`
	Action    string         `json:"action"            validate:"required"`
	Expected  bool           `json:"expected"`
}

// PolicyTestCase is an alias for TestCase for backwards compatibility.
type PolicyTestCase = TestCase

// TestCaseResult represents the result of a single test case.
type TestCaseResult struct {
	Name             string  `json:"name"`
	Passed           bool    `json:"passed"`
	Actual           bool    `json:"actual"`
	Expected         bool    `json:"expected"`
	Error            string  `json:"error,omitempty"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
}

// PolicyTestResult is an alias for TestCaseResult for backwards compatibility.
type PolicyTestResult = TestCaseResult

// TestPolicyResponse represents the result of policy testing.
type TestPolicyResponse struct {
	Passed  bool             `json:"passed"`
	Results []TestCaseResult `json:"results"`
	Total   int              `json:"total"`
	PassCnt int              `json:"passedCount"`
	FailCnt int              `json:"failedCount"`
	Error   string           `json:"error,omitempty"`
}

// =============================================================================
// TEMPLATE DTOs
// =============================================================================

// TemplateResponse represents a single policy template.
type TemplateResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Category    string                   `json:"category"`
	Expression  string                   `json:"expression"`
	Parameters  []core.TemplateParameter `json:"parameters"`
	Examples    []string                 `json:"examples"`
}

// TemplatesListResponse represents a list of policy templates.
type TemplatesListResponse struct {
	Templates  []*TemplateResponse `json:"templates"`
	TotalCount int                 `json:"totalCount"`
	Categories []string            `json:"categories"`
}

// InstantiateTemplateRequest represents a request to instantiate a template.
type InstantiateTemplateRequest struct {
	NamespaceID  string         `json:"namespaceId"  validate:"required"`
	Name         string         `json:"name"         validate:"required,min=3,max=100"`
	Description  string         `json:"description"  validate:"max=500"`
	Parameters   map[string]any `json:"parameters"   validate:"required"`
	ResourceType string         `json:"resourceType" validate:"required"`
	Actions      []string       `json:"actions"      validate:"required,min=1"`
	Priority     int            `json:"priority"     validate:"min=0,max=1000"`
	Enabled      bool           `json:"enabled"`
}

// =============================================================================
// MIGRATION DTOs
// =============================================================================

// MigrateRBACRequest represents a request to migrate from RBAC to permissions.
type MigrateRBACRequest struct {
	NamespaceID         string `json:"namespaceId"         validate:"required"`
	ValidateEquivalence bool   `json:"validateEquivalence"`
	KeepRBACPolicies    bool   `json:"keepRbacPolicies"`
	DryRun              bool   `json:"dryRun"`
}

// MigrationResponse represents the result of starting a migration.
type MigrationResponse struct {
	MigrationID string    `json:"migrationId"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	StartedAt   time.Time `json:"startedAt"`
}

// MigrationStatusResponse represents the status of a migration.
type MigrationStatusResponse struct {
	AppID              string     `json:"appId"`
	EnvironmentID      string     `json:"environmentId"`
	UserOrganizationID *string    `json:"userOrganizationId,omitempty"`
	Status             string     `json:"status"`
	StartedAt          time.Time  `json:"startedAt"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	TotalPolicies      int        `json:"totalPolicies"`
	MigratedCount      int        `json:"migratedCount"`
	FailedCount        int        `json:"failedCount"`
	ValidationPassed   bool       `json:"validationPassed"`
	Errors             []string   `json:"errors,omitempty"`
	Progress           float64    `json:"progress"`
}

// =============================================================================
// AUDIT & ANALYTICS DTOs
// =============================================================================

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	ID                 string         `json:"id"`
	AppID              string         `json:"appId"`
	EnvironmentID      string         `json:"environmentId"`
	UserOrganizationID *string        `json:"userOrganizationId,omitempty"`
	ActorID            string         `json:"actorId"`
	Action             string         `json:"action"`
	ResourceType       string         `json:"resourceType"`
	ResourceID         string         `json:"resourceId"`
	OldValue           map[string]any `json:"oldValue,omitempty"`
	NewValue           map[string]any `json:"newValue,omitempty"`
	IPAddress          string         `json:"ipAddress"`
	UserAgent          string         `json:"userAgent"`
	Timestamp          time.Time      `json:"timestamp"`
}

// AuditLogResponse represents a list of audit log entries.
type AuditLogResponse struct {
	Entries    []*AuditLogEntry `json:"entries"`
	TotalCount int              `json:"totalCount"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
}

// AnalyticsSummary represents summary analytics data.
type AnalyticsSummary struct {
	TotalPolicies    int                 `json:"totalPolicies"`
	ActivePolicies   int                 `json:"activePolicies"`
	TotalEvaluations int64               `json:"totalEvaluations"`
	AllowedCount     int64               `json:"allowedCount"`
	DeniedCount      int64               `json:"deniedCount"`
	AvgLatencyMs     float64             `json:"avgLatencyMs"`
	CacheHitRate     float64             `json:"cacheHitRate"`
	TopPolicies      []PolicyStats       `json:"topPolicies,omitempty"`
	TopResourceTypes []ResourceTypeStats `json:"topResourceTypes,omitempty"`
}

// PolicyStats represents statistics for a single policy.
type PolicyStats struct {
	PolicyID        string  `json:"policyId"`
	PolicyName      string  `json:"policyName"`
	EvaluationCount int64   `json:"evaluationCount"`
	AllowCount      int64   `json:"allowCount"`
	DenyCount       int64   `json:"denyCount"`
	AvgLatencyMs    float64 `json:"avgLatencyMs"`
}

// ResourceTypeStats represents statistics for a resource type.
type ResourceTypeStats struct {
	ResourceType    string  `json:"resourceType"`
	EvaluationCount int64   `json:"evaluationCount"`
	AllowRate       float64 `json:"allowRate"`
	AvgLatencyMs    float64 `json:"avgLatencyMs"`
}

// AnalyticsResponse represents analytics data response.
type AnalyticsResponse struct {
	Summary   AnalyticsSummary `json:"summary"`
	TimeRange struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"timeRange"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// ToPolicyResponse converts a core.Policy to a PolicyResponse.
func ToPolicyResponse(p *core.Policy) *PolicyResponse {
	resp := &PolicyResponse{
		ID:            p.ID.String(),
		AppID:         p.AppID.String(),
		EnvironmentID: p.EnvironmentID.String(),
		NamespaceID:   p.NamespaceID.String(),
		Name:          p.Name,
		Description:   p.Description,
		Expression:    p.Expression,
		ResourceType:  p.ResourceType,
		Actions:       p.Actions,
		Priority:      p.Priority,
		Enabled:       p.Enabled,
		Version:       p.Version,
		CreatedBy:     p.CreatedBy.String(),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
	if p.UserOrganizationID != nil {
		orgID := p.UserOrganizationID.String()
		resp.UserOrganizationID = &orgID
	}

	return resp
}

// ToNamespaceResponse converts a core.Namespace to a NamespaceResponse.
func ToNamespaceResponse(n *core.Namespace) *NamespaceResponse {
	resp := &NamespaceResponse{
		ID:              n.ID.String(),
		AppID:           n.AppID.String(),
		EnvironmentID:   n.EnvironmentID.String(),
		Name:            n.Name,
		Description:     n.Description,
		InheritPlatform: n.InheritPlatform,
		ResourceCount:   len(n.Resources),
		ActionCount:     len(n.Actions),
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}
	if n.UserOrganizationID != nil {
		orgID := n.UserOrganizationID.String()
		resp.UserOrganizationID = &orgID
	}

	if n.TemplateID != nil {
		templateID := n.TemplateID.String()
		resp.TemplateID = &templateID
	}

	return resp
}

// ToResourceResponse converts a core.ResourceDefinition to a ResourceResponse.
func ToResourceResponse(r *core.ResourceDefinition) *ResourceResponse {
	return &ResourceResponse{
		ID:          r.ID.String(),
		NamespaceID: r.NamespaceID.String(),
		Type:        r.Type,
		Description: r.Description,
		Attributes:  r.Attributes,
		CreatedAt:   r.CreatedAt,
	}
}

// ToActionResponse converts a core.ActionDefinition to an ActionResponse.
func ToActionResponse(a *core.ActionDefinition) *ActionResponse {
	return &ActionResponse{
		ID:          a.ID.String(),
		NamespaceID: a.NamespaceID.String(),
		Name:        a.Name,
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
	}
}

// ToAuditLogEntry converts a core.AuditEvent to an AuditLogEntry.
func ToAuditLogEntry(e *core.AuditEvent) *AuditLogEntry {
	entry := &AuditLogEntry{
		ID:            e.ID.String(),
		AppID:         e.AppID.String(),
		EnvironmentID: e.EnvironmentID.String(),
		ActorID:       e.ActorID.String(),
		Action:        e.Action,
		ResourceType:  e.ResourceType,
		ResourceID:    e.ResourceID.String(),
		OldValue:      e.OldValue,
		NewValue:      e.NewValue,
		IPAddress:     e.IPAddress,
		UserAgent:     e.UserAgent,
		Timestamp:     e.Timestamp,
	}
	if e.UserOrganizationID != nil {
		orgID := e.UserOrganizationID.String()
		entry.UserOrganizationID = &orgID
	}

	return entry
}
