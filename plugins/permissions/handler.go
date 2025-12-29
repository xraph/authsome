package permissions

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for the permissions plugin
// V2 Architecture: App → Environment → Organization
type Handler struct {
	service *Service
}

// Response types - use from handlers package
type (
	MessageResponse = handlers.MessageResponse
	StatusResponse  = handlers.StatusResponse
)

// NewHandler creates a new handler instance
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// extractContext extracts V2 context values (app, env, org, user)
func extractContext(c forge.Context) (appID, envID xid.ID, orgID *xid.ID, userID xid.ID, err error) {
	ctx := c.Request().Context()

	// App is required
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return xid.NilID(), xid.NilID(), nil, xid.NilID(), errs.New("APP_CONTEXT_REQUIRED", "App context required", 400)
	}

	// Environment is required
	envID, ok = contexts.GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return xid.NilID(), xid.NilID(), nil, xid.NilID(), errs.New("ENV_CONTEXT_REQUIRED", "Environment context required", 400)
	}

	// Organization is optional
	orgIDVal, _ := contexts.GetOrganizationID(ctx)
	if !orgIDVal.IsNil() {
		orgID = &orgIDVal
	}

	// User is optional for some operations
	userID, _ = contexts.GetUserID(ctx)

	return appID, envID, orgID, userID, nil
}

// getStatusCode extracts HTTP status code from error
func getStatusCode(err error) int {
	return errs.GetHTTPStatus(err)
}

// toErrorResponse converts an error to a JSON-serializable response
func toErrorResponse(err error) interface{} {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return authErr
	}
	// Wrap generic errors
	return errs.InternalError(err)
}

// =============================================================================
// POLICY MANAGEMENT
// =============================================================================

// CreatePolicy handles POST /permissions/policies
func (h *Handler) CreatePolicy(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.CreatePolicyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	policy, err := h.service.CreatePolicy(c.Request().Context(), appID, envID, orgID, userID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(201, handlers.ToPolicyResponse(policy))
}

// ListPolicies handles GET /permissions/policies
func (h *Handler) ListPolicies(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Build filters from query params
	filters := make(map[string]interface{})
	if resourceType := c.Query("resourceType"); resourceType != "" {
		filters["resourceType"] = resourceType
	}
	if namespaceID := c.Query("namespaceId"); namespaceID != "" {
		filters["namespaceId"] = namespaceID
	}
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			filters["enabled"] = enabled
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filters["page"] = page
	filters["pageSize"] = pageSize

	// Call service
	policies, total, err := h.service.ListPolicies(c.Request().Context(), appID, envID, orgID, filters)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	policyResponses := make([]*handlers.PolicyResponse, 0, len(policies))
	for _, p := range policies {
		policyResponses = append(policyResponses, handlers.ToPolicyResponse(p))
	}

	return c.JSON(200, &handlers.PoliciesListResponse{
		Policies:   policyResponses,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	})
}

// GetPolicy handles GET /permissions/policies/:id
func (h *Handler) GetPolicy(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse policy ID from URL param
	policyIDStr := c.Param("id")
	policyID, err := xid.FromString(policyIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_POLICY_ID", "Invalid policy ID", 400))
	}

	// Call service
	policy, err := h.service.GetPolicy(c.Request().Context(), appID, envID, orgID, policyID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, handlers.ToPolicyResponse(policy))
}

// UpdatePolicy handles PUT /permissions/policies/:id
func (h *Handler) UpdatePolicy(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse policy ID from URL param
	policyIDStr := c.Param("id")
	policyID, err := xid.FromString(policyIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_POLICY_ID", "Invalid policy ID", 400))
	}

	// Parse request
	var req handlers.UpdatePolicyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	policy, err := h.service.UpdatePolicy(c.Request().Context(), appID, envID, orgID, userID, policyID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, handlers.ToPolicyResponse(policy))
}

// DeletePolicy handles DELETE /permissions/policies/:id
func (h *Handler) DeletePolicy(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse policy ID from URL param
	policyIDStr := c.Param("id")
	policyID, err := xid.FromString(policyIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_POLICY_ID", "Invalid policy ID", 400))
	}

	// Call service
	if err := h.service.DeletePolicy(c.Request().Context(), appID, envID, orgID, policyID); err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(204, nil)
}

// ValidatePolicy handles POST /permissions/policies/validate
func (h *Handler) ValidatePolicy(c forge.Context) error {
	// Parse request
	var req handlers.ValidatePolicyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	validation, err := h.service.ValidatePolicy(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, validation)
}

// TestPolicy handles POST /permissions/policies/test
func (h *Handler) TestPolicy(c forge.Context) error {
	// Parse request
	var req handlers.TestPolicyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	testResult, err := h.service.TestPolicy(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, testResult)
}

// =============================================================================
// RESOURCE MANAGEMENT
// =============================================================================

// CreateResource handles POST /permissions/resources
func (h *Handler) CreateResource(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.CreateResourceRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	resource, err := h.service.CreateResource(c.Request().Context(), appID, envID, orgID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(201, handlers.ToResourceResponse(resource))
}

// ListResources handles GET /permissions/resources
func (h *Handler) ListResources(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Get optional namespace filter
	var namespaceID xid.ID
	if nsID := c.Query("namespaceId"); nsID != "" {
		namespaceID, err = xid.FromString(nsID)
		if err != nil {
			return c.JSON(400, errs.New("INVALID_NAMESPACE_ID", "Invalid namespace ID", 400))
		}
	}

	// Call service
	resources, err := h.service.ListResources(c.Request().Context(), appID, envID, orgID, namespaceID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	resourceResponses := make([]*handlers.ResourceResponse, 0, len(resources))
	for _, r := range resources {
		resourceResponses = append(resourceResponses, handlers.ToResourceResponse(r))
	}

	return c.JSON(200, &handlers.ResourcesListResponse{
		Resources:  resourceResponses,
		TotalCount: len(resourceResponses),
	})
}

// GetResource handles GET /permissions/resources/:id
func (h *Handler) GetResource(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse resource ID from URL param
	resourceIDStr := c.Param("id")
	resourceID, err := xid.FromString(resourceIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_RESOURCE_ID", "Invalid resource ID", 400))
	}

	// Call service
	resource, err := h.service.GetResource(c.Request().Context(), appID, envID, orgID, resourceID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, handlers.ToResourceResponse(resource))
}

// DeleteResource handles DELETE /permissions/resources/:id
func (h *Handler) DeleteResource(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse resource ID from URL param
	resourceIDStr := c.Param("id")
	resourceID, err := xid.FromString(resourceIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_RESOURCE_ID", "Invalid resource ID", 400))
	}

	// Call service
	if err := h.service.DeleteResource(c.Request().Context(), appID, envID, orgID, resourceID); err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(204, nil)
}

// =============================================================================
// ACTION MANAGEMENT
// =============================================================================

// CreateAction handles POST /permissions/actions
func (h *Handler) CreateAction(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.CreateActionRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	action, err := h.service.CreateAction(c.Request().Context(), appID, envID, orgID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(201, handlers.ToActionResponse(action))
}

// ListActions handles GET /permissions/actions
func (h *Handler) ListActions(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Get optional namespace filter
	var namespaceID xid.ID
	if nsID := c.Query("namespaceId"); nsID != "" {
		namespaceID, err = xid.FromString(nsID)
		if err != nil {
			return c.JSON(400, errs.New("INVALID_NAMESPACE_ID", "Invalid namespace ID", 400))
		}
	}

	// Call service
	actions, err := h.service.ListActions(c.Request().Context(), appID, envID, orgID, namespaceID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	actionResponses := make([]*handlers.ActionResponse, 0, len(actions))
	for _, a := range actions {
		actionResponses = append(actionResponses, handlers.ToActionResponse(a))
	}

	return c.JSON(200, &handlers.ActionsListResponse{
		Actions:    actionResponses,
		TotalCount: len(actionResponses),
	})
}

// DeleteAction handles DELETE /permissions/actions/:id
func (h *Handler) DeleteAction(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse action ID from URL param
	actionIDStr := c.Param("id")
	actionID, err := xid.FromString(actionIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_ACTION_ID", "Invalid action ID", 400))
	}

	// Call service
	if err := h.service.DeleteAction(c.Request().Context(), appID, envID, orgID, actionID); err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(204, nil)
}

// =============================================================================
// NAMESPACE MANAGEMENT
// =============================================================================

// CreateNamespace handles POST /permissions/namespaces
func (h *Handler) CreateNamespace(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.CreateNamespaceRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	namespace, err := h.service.CreateNamespace(c.Request().Context(), appID, envID, orgID, userID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(201, handlers.ToNamespaceResponse(namespace))
}

// ListNamespaces handles GET /permissions/namespaces
func (h *Handler) ListNamespaces(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Call service
	namespaces, err := h.service.ListNamespaces(c.Request().Context(), appID, envID, orgID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	namespaceResponses := make([]*handlers.NamespaceResponse, 0, len(namespaces))
	for _, n := range namespaces {
		namespaceResponses = append(namespaceResponses, handlers.ToNamespaceResponse(n))
	}

	return c.JSON(200, &handlers.NamespacesListResponse{
		Namespaces: namespaceResponses,
		TotalCount: len(namespaceResponses),
	})
}

// GetNamespace handles GET /permissions/namespaces/:id
func (h *Handler) GetNamespace(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse namespace ID from URL param
	namespaceIDStr := c.Param("id")
	namespaceID, err := xid.FromString(namespaceIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_NAMESPACE_ID", "Invalid namespace ID", 400))
	}

	// Call service
	namespace, err := h.service.GetNamespace(c.Request().Context(), appID, envID, orgID, namespaceID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, handlers.ToNamespaceResponse(namespace))
}

// UpdateNamespace handles PUT /permissions/namespaces/:id
func (h *Handler) UpdateNamespace(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse namespace ID from URL param
	namespaceIDStr := c.Param("id")
	namespaceID, err := xid.FromString(namespaceIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_NAMESPACE_ID", "Invalid namespace ID", 400))
	}

	// Parse request
	var req handlers.UpdateNamespaceRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	namespace, err := h.service.UpdateNamespace(c.Request().Context(), appID, envID, orgID, namespaceID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, handlers.ToNamespaceResponse(namespace))
}

// DeleteNamespace handles DELETE /permissions/namespaces/:id
func (h *Handler) DeleteNamespace(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse namespace ID from URL param
	namespaceIDStr := c.Param("id")
	namespaceID, err := xid.FromString(namespaceIDStr)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_NAMESPACE_ID", "Invalid namespace ID", 400))
	}

	// Call service
	if err := h.service.DeleteNamespace(c.Request().Context(), appID, envID, orgID, namespaceID); err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(204, nil)
}

// =============================================================================
// EVALUATION
// =============================================================================

// Evaluate handles POST /permissions/evaluate
func (h *Handler) Evaluate(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.EvaluateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	decision, err := h.service.Evaluate(c.Request().Context(), appID, envID, orgID, userID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	response := &handlers.EvaluateResponse{
		Allowed:           decision.Allowed,
		EvaluatedPolicies: decision.EvaluatedPolicies,
		EvaluationTimeMs:  float64(decision.EvaluationTime.Milliseconds()),
		CacheHit:          decision.CacheHit,
		MatchedPolicies:   decision.MatchedPolicies,
		Error:             decision.Error,
	}

	return c.JSON(200, response)
}

// EvaluateBatch handles POST /permissions/evaluate/batch
func (h *Handler) EvaluateBatch(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.BatchEvaluateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	results, err := h.service.EvaluateBatch(c.Request().Context(), appID, envID, orgID, userID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Calculate totals
	var totalTime float64
	successCount := 0
	failureCount := 0
	for _, r := range results {
		totalTime += r.EvaluationTimeMs
		if r.Allowed {
			successCount++
		} else {
			failureCount++
		}
	}

	return c.JSON(200, &handlers.BatchEvaluateResponse{
		Results:          results,
		TotalEvaluations: len(results),
		TotalTimeMs:      totalTime,
		SuccessCount:     successCount,
		FailureCount:     failureCount,
	})
}

// =============================================================================
// TEMPLATES
// =============================================================================

// ListTemplates handles GET /permissions/templates
func (h *Handler) ListTemplates(c forge.Context) error {
	// Call service
	templates, err := h.service.ListTemplates(c.Request().Context())
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	templateResponses := make([]*handlers.TemplateResponse, 0, len(templates))
	categories := make(map[string]bool)
	for _, t := range templates {
		templateResponses = append(templateResponses, toTemplateResponse(t))
		categories[t.Category] = true
	}

	// Extract unique categories
	categoryList := make([]string, 0, len(categories))
	for cat := range categories {
		categoryList = append(categoryList, cat)
	}

	return c.JSON(200, &handlers.TemplatesListResponse{
		Templates:  templateResponses,
		TotalCount: len(templateResponses),
		Categories: categoryList,
	})
}

// GetTemplate handles GET /permissions/templates/:id
func (h *Handler) GetTemplate(c forge.Context) error {
	templateID := c.Param("id")

	// Call service
	template, err := h.service.GetTemplate(c.Request().Context(), templateID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, toTemplateResponse(template))
}

// InstantiateTemplate handles POST /permissions/templates/:id/instantiate
func (h *Handler) InstantiateTemplate(c forge.Context) error {
	appID, envID, orgID, userID, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Get template ID from URL
	templateID := c.Param("id")

	// Parse request
	var req handlers.InstantiateTemplateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	policy, err := h.service.InstantiateTemplate(c.Request().Context(), appID, envID, orgID, userID, templateID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(201, handlers.ToPolicyResponse(policy))
}

// toTemplateResponse converts a core.PolicyTemplate to a handlers.TemplateResponse
func toTemplateResponse(t *core.PolicyTemplate) *handlers.TemplateResponse {
	return &handlers.TemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		Expression:  t.Expression,
		Parameters:  t.Parameters,
		Examples:    t.Examples,
	}
}

// =============================================================================
// MIGRATION
// =============================================================================

// MigrateFromRBAC handles POST /permissions/migrate/rbac
func (h *Handler) MigrateFromRBAC(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Parse request
	var req handlers.MigrateRBACRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Call service
	status, err := h.service.MigrateFromRBAC(c.Request().Context(), appID, envID, orgID, &req)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, toMigrationStatusResponse(status))
}

// GetMigrationStatus handles GET /permissions/migrate/rbac/status
func (h *Handler) GetMigrationStatus(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Call service
	status, err := h.service.GetMigrationStatus(c.Request().Context(), appID, envID, orgID)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, toMigrationStatusResponse(status))
}

// toMigrationStatusResponse converts a core.MigrationStatus to a handlers.MigrationStatusResponse
func toMigrationStatusResponse(s *core.MigrationStatus) *handlers.MigrationStatusResponse {
	resp := &handlers.MigrationStatusResponse{
		AppID:            s.AppID.String(),
		EnvironmentID:    s.EnvironmentID.String(),
		Status:           s.Status,
		StartedAt:        s.StartedAt,
		CompletedAt:      s.CompletedAt,
		TotalPolicies:    s.TotalPolicies,
		MigratedCount:    s.MigratedCount,
		FailedCount:      s.FailedCount,
		ValidationPassed: s.ValidationPassed,
		Errors:           s.Errors,
	}
	if s.TotalPolicies > 0 {
		resp.Progress = float64(s.MigratedCount) / float64(s.TotalPolicies) * 100
	}
	if s.UserOrganizationID != nil {
		orgID := s.UserOrganizationID.String()
		resp.UserOrganizationID = &orgID
	}
	return resp
}

// =============================================================================
// AUDIT & ANALYTICS
// =============================================================================

// GetAuditLog handles GET /permissions/audit
func (h *Handler) GetAuditLog(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Build filters from query params
	filters := make(map[string]interface{})
	if resourceType := c.Query("resourceType"); resourceType != "" {
		filters["resourceType"] = resourceType
	}
	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}
	if actorID := c.Query("actorId"); actorID != "" {
		filters["actorId"] = actorID
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filters["page"] = page
	filters["pageSize"] = pageSize

	// Call service
	events, total, err := h.service.ListAuditEvents(c.Request().Context(), appID, envID, orgID, filters)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	// Convert to response
	entries := make([]*handlers.AuditLogEntry, 0, len(events))
	for _, e := range events {
		entries = append(entries, handlers.ToAuditLogEntry(e))
	}

	return c.JSON(200, &handlers.AuditLogResponse{
		Entries:    entries,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	})
}

// GetAnalytics handles GET /permissions/analytics
func (h *Handler) GetAnalytics(c forge.Context) error {
	appID, envID, orgID, _, err := extractContext(c)
	if err != nil {
		return c.JSON(400, toErrorResponse(err))
	}

	// Build time range from query params
	timeRange := make(map[string]interface{})
	if startTime := c.Query("startTime"); startTime != "" {
		timeRange["startTime"] = startTime
	}
	if endTime := c.Query("endTime"); endTime != "" {
		timeRange["endTime"] = endTime
	}

	// Call service
	analytics, err := h.service.GetAnalytics(c.Request().Context(), appID, envID, orgID, timeRange)
	if err != nil {
		return c.JSON(getStatusCode(err), toErrorResponse(err))
	}

	return c.JSON(200, analytics)
}
