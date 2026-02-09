package permissions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/engine"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/authsome/plugins/permissions/migration"
	"github.com/xraph/authsome/plugins/permissions/storage"
	"github.com/xraph/forge"
)

// Service is the main permissions service
// V2 Architecture: App → Environment → Organization.
type Service struct {
	config    *Config
	db        *bun.DB
	repo      storage.Repository
	cache     storage.Cache
	logger    forge.Logger
	compiler  *engine.Compiler
	evaluator *engine.Evaluator

	// Migration service for RBAC to CEL migration
	migrationService *migration.RBACMigrationService

	// Compiled policies cache (in-memory for fast evaluation)
	compiledPolicies   map[string]*engine.CompiledPolicy
	compiledPoliciesMu sync.RWMutex
	policyIndex        map[string][]*engine.CompiledPolicy // Indexed by resource type
	policyIndexMu      sync.RWMutex
}

// SetMigrationService sets the migration service.
func (s *Service) SetMigrationService(svc *migration.RBACMigrationService) {
	s.migrationService = svc
}

// NewService creates a new permissions service with all dependencies.
func NewService(db *bun.DB, config *Config, logger forge.Logger) *Service {
	repo := storage.NewRepository(db)
	cache := storage.NewMemoryCache(config.Cache)

	// Create CEL compiler with config
	compilerConfig := engine.CompilerConfig{
		MaxComplexity: config.Engine.MaxPolicyComplexity,
	}

	compiler, err := engine.NewCompiler(compilerConfig)
	if err != nil {
		logger.Error("failed to create CEL compiler", forge.F("error", err.Error()))

		compiler = nil
	}

	// Create evaluator
	evaluatorConfig := engine.EvaluatorConfig{
		Timeout:             config.Engine.EvaluationTimeout,
		ParallelEvaluations: config.Engine.MaxParallelEvaluations,
		EnableParallel:      config.Engine.ParallelEvaluation,
	}
	evaluator := engine.NewEvaluator(evaluatorConfig)

	return &Service{
		config:           config,
		db:               db,
		repo:             repo,
		cache:            cache,
		logger:           logger,
		compiler:         compiler,
		evaluator:        evaluator,
		compiledPolicies: make(map[string]*engine.CompiledPolicy),
		policyIndex:      make(map[string][]*engine.CompiledPolicy),
	}
}

// =============================================================================
// POLICY OPERATIONS
// =============================================================================

// CreatePolicy creates a new permission policy.
func (s *Service) CreatePolicy(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, req *handlers.CreatePolicyRequest) (*core.Policy, error) {
	if s.compiler == nil {
		return nil, errs.InternalServerErrorWithMessage("CEL compiler not initialized")
	}

	// 1. Validate the CEL expression
	if err := s.compiler.Validate(req.Expression); err != nil {
		return nil, fmt.Errorf("invalid policy expression: %w", err)
	}

	// 2. Check expression complexity
	complexity, err := s.compiler.EstimateComplexity(req.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate expression complexity: %w", err)
	}

	if complexity > s.compiler.GetMaxComplexity() {
		return nil, fmt.Errorf("expression complexity %d exceeds maximum %d", complexity, s.compiler.GetMaxComplexity())
	}

	// 3. Parse namespace ID
	namespaceID, err := xid.FromString(req.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace ID: %w", err)
	}

	// 4. Create policy object
	now := time.Now()
	policy := &core.Policy{
		ID:                 xid.New(),
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		NamespaceID:        namespaceID,
		Name:               req.Name,
		Description:        req.Description,
		Expression:         req.Expression,
		ResourceType:       req.ResourceType,
		Actions:            req.Actions,
		Priority:           req.Priority,
		Enabled:            true,
		Version:            1,
		CreatedBy:          userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// 5. Store in database
	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	// 6. Compile and cache the policy
	compiled, err := s.compiler.Compile(policy)
	if err != nil {
		s.logger.Warn("failed to compile policy after creation", forge.F("policyId", policy.ID.String()), forge.F("error", err.Error()))
	} else {
		s.cacheCompiledPolicy(compiled)
	}

	// 7. Create audit log entry
	s.createAuditEvent(ctx, appID, envID, orgID, userID, "create_policy", "policy", policy.ID, nil, map[string]any{
		"name":         policy.Name,
		"resourceType": policy.ResourceType,
		"actions":      policy.Actions,
	})

	s.logger.Info("policy created",
		forge.F("policyId", policy.ID.String()),
		forge.F("name", policy.Name),
		forge.F("resourceType", policy.ResourceType),
	)

	return policy, nil
}

// GetPolicy retrieves a policy by ID.
func (s *Service) GetPolicy(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, policyID xid.ID) (*core.Policy, error) {
	policy, err := s.repo.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy == nil {
		return nil, errs.NotFound("policy not found")
	}

	// Verify policy belongs to app/env/org
	if policy.AppID != appID || policy.EnvironmentID != envID {
		return nil, errs.NotFound("policy not found")
	}

	if orgID != nil && policy.UserOrganizationID != nil && *policy.UserOrganizationID != *orgID {
		return nil, errs.NotFound("policy not found")
	}

	return policy, nil
}

// ListPolicies lists policies for an app/env/org.
func (s *Service) ListPolicies(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, filters map[string]any) ([]*core.Policy, int, error) {
	// Convert filters to PolicyFilters
	pf := storage.PolicyFilters{
		Limit:  100,
		Offset: 0,
	}

	if limit, ok := filters["limit"].(int); ok {
		pf.Limit = limit
	}

	if offset, ok := filters["offset"].(int); ok {
		pf.Offset = offset
	}

	if resourceType, ok := filters["resourceType"].(string); ok {
		pf.ResourceType = &resourceType
	}

	if actions, ok := filters["actions"].([]string); ok {
		pf.Actions = actions
	}

	if enabled, ok := filters["enabled"].(bool); ok {
		pf.Enabled = &enabled
	}

	if namespaceID, ok := filters["namespaceId"].(xid.ID); ok {
		pf.NamespaceID = &namespaceID
	}

	policies, err := s.repo.ListPolicies(ctx, appID, envID, orgID, pf)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list policies: %w", err)
	}

	return policies, len(policies), nil
}

// UpdatePolicy updates an existing policy.
func (s *Service) UpdatePolicy(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, policyID xid.ID, req *handlers.UpdatePolicyRequest) (*core.Policy, error) {
	// 1. Get existing policy
	policy, err := s.GetPolicy(ctx, appID, envID, orgID, policyID)
	if err != nil {
		return nil, err
	}

	oldValue := map[string]any{
		"name":         policy.Name,
		"description":  policy.Description,
		"expression":   policy.Expression,
		"resourceType": policy.ResourceType,
		"actions":      policy.Actions,
		"priority":     policy.Priority,
		"enabled":      policy.Enabled,
	}

	// 2. Update fields
	if req.Name != "" {
		policy.Name = req.Name
	}

	if req.Description != "" {
		policy.Description = req.Description
	}

	if req.Expression != "" {
		// Validate new expression
		if err := s.compiler.Validate(req.Expression); err != nil {
			return nil, fmt.Errorf("invalid policy expression: %w", err)
		}

		policy.Expression = req.Expression
	}

	if req.ResourceType != "" {
		policy.ResourceType = req.ResourceType
	}

	if len(req.Actions) > 0 {
		policy.Actions = req.Actions
	}

	if req.Priority != 0 {
		policy.Priority = req.Priority
	}

	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	// 3. Increment version and update timestamp
	policy.Version++
	policy.UpdatedAt = time.Now()

	// 4. Update in database
	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	// 5. Recompile and update cache
	if policy.Enabled {
		compiled, err := s.compiler.Compile(policy)
		if err != nil {
			s.logger.Warn("failed to recompile policy after update", forge.F("policyId", policy.ID.String()), forge.F("error", err.Error()))
		} else {
			s.cacheCompiledPolicy(compiled)
		}
	} else {
		// Remove from cache if disabled
		s.removeCompiledPolicy(policy.ID.String())
	}

	// 6. Create audit log entry
	newValue := map[string]any{
		"name":         policy.Name,
		"description":  policy.Description,
		"expression":   policy.Expression,
		"resourceType": policy.ResourceType,
		"actions":      policy.Actions,
		"priority":     policy.Priority,
		"enabled":      policy.Enabled,
	}
	s.createAuditEvent(ctx, appID, envID, orgID, userID, "update_policy", "policy", policy.ID, oldValue, newValue)

	return policy, nil
}

// DeletePolicy deletes a policy.
func (s *Service) DeletePolicy(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, policyID xid.ID) error {
	// 1. Verify policy exists and belongs to scope
	policy, err := s.GetPolicy(ctx, appID, envID, orgID, policyID)
	if err != nil {
		return err
	}

	// 2. Delete from database
	if err := s.repo.DeletePolicy(ctx, policyID); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	// 3. Remove from cache
	s.removeCompiledPolicy(policyID.String())

	// 4. Create audit log entry
	s.createAuditEvent(ctx, appID, envID, orgID, xid.NilID(), "delete_policy", "policy", policyID, map[string]any{
		"name":         policy.Name,
		"resourceType": policy.ResourceType,
	}, nil)

	s.logger.Info("policy deleted", forge.F("policyId", policyID.String()))

	return nil
}

// ValidatePolicy validates a policy expression.
func (s *Service) ValidatePolicy(ctx context.Context, req *handlers.ValidatePolicyRequest) (*handlers.ValidatePolicyResponse, error) {
	if s.compiler == nil {
		return nil, errs.InternalServerErrorWithMessage("CEL compiler not initialized")
	}

	response := &handlers.ValidatePolicyResponse{
		Valid:    true,
		Warnings: []string{},
	}

	// Validate expression
	if err := s.compiler.Validate(req.Expression); err != nil {
		response.Valid = false
		response.Error = err.Error()

		return response, nil
	}

	// Estimate complexity
	complexity, err := s.compiler.EstimateComplexity(req.Expression)
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("failed to estimate complexity: %v", err)

		return response, nil
	}

	response.Complexity = complexity

	// Add warning if approaching limit
	maxComplexity := s.compiler.GetMaxComplexity()
	if complexity > maxComplexity*80/100 {
		response.Warnings = append(response.Warnings, fmt.Sprintf("expression complexity (%d) is approaching the limit (%d)", complexity, maxComplexity))
	}

	return response, nil
}

// TestPolicy tests a policy against test cases.
func (s *Service) TestPolicy(ctx context.Context, req *handlers.TestPolicyRequest) (*handlers.TestPolicyResponse, error) {
	if s.compiler == nil {
		return nil, errs.InternalServerErrorWithMessage("CEL compiler not initialized")
	}

	// Create a temporary policy for testing
	testPolicy := &core.Policy{
		ID:           xid.New(),
		Name:         "test_policy",
		Expression:   req.Expression,
		ResourceType: req.ResourceType,
		Actions:      req.Actions,
		Priority:     100,
		Enabled:      true,
	}

	// Compile the test policy
	compiled, err := s.compiler.Compile(testPolicy)
	if err != nil {
		return &handlers.TestPolicyResponse{
			Passed: false,
			Error:  fmt.Sprintf("failed to compile expression: %v", err),
		}, nil
	}

	// Run test cases
	response := &handlers.TestPolicyResponse{
		Passed:  true,
		Results: make([]handlers.TestCaseResult, len(req.TestCases)),
	}

	for i, tc := range req.TestCases {
		evalCtx := &engine.EvaluationContext{
			Principal: tc.Principal,
			Resource:  tc.Resource,
			Request:   tc.Request,
			Action:    tc.Action,
		}

		decision, err := s.evaluator.Evaluate(ctx, []*engine.CompiledPolicy{compiled}, evalCtx)

		result := handlers.TestCaseResult{
			Name:     tc.Name,
			Expected: tc.Expected,
		}

		if err != nil {
			result.Passed = false
			result.Error = err.Error()
			response.Passed = false
		} else {
			result.Actual = decision.Allowed
			result.Passed = decision.Allowed == tc.Expected

			result.EvaluationTimeMs = float64(decision.EvaluationTime.Microseconds()) / 1000
			if !result.Passed {
				response.Passed = false
			}
		}

		response.Results[i] = result
	}

	return response, nil
}

// =============================================================================
// RESOURCE OPERATIONS
// =============================================================================

// CreateResource creates a new resource definition.
func (s *Service) CreateResource(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, req *handlers.CreateResourceRequest) (*core.ResourceDefinition, error) {
	namespaceID, err := xid.FromString(req.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace ID: %w", err)
	}

	now := time.Now()
	res := &core.ResourceDefinition{
		ID:          xid.New(),
		NamespaceID: namespaceID,
		Type:        req.Type,
		Description: req.Description,
		Attributes:  convertToResourceAttributes(req.Attributes),
		CreatedAt:   now,
	}

	if err := s.repo.CreateResourceDefinition(ctx, res); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return res, nil
}

// GetResource retrieves a resource definition by ID.
func (s *Service) GetResource(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, resourceID xid.ID) (*core.ResourceDefinition, error) {
	return s.repo.GetResourceDefinition(ctx, resourceID)
}

// ListResources lists resource definitions for a namespace.
func (s *Service) ListResources(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, namespaceID xid.ID) ([]*core.ResourceDefinition, error) {
	return s.repo.ListResourceDefinitions(ctx, namespaceID)
}

// DeleteResource deletes a resource definition.
func (s *Service) DeleteResource(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, resourceID xid.ID) error {
	return s.repo.DeleteResourceDefinition(ctx, resourceID)
}

// =============================================================================
// ACTION OPERATIONS
// =============================================================================

// CreateAction creates a new action definition.
func (s *Service) CreateAction(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, req *handlers.CreateActionRequest) (*core.ActionDefinition, error) {
	namespaceID, err := xid.FromString(req.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace ID: %w", err)
	}

	now := time.Now()
	action := &core.ActionDefinition{
		ID:          xid.New(),
		NamespaceID: namespaceID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
	}

	if err := s.repo.CreateActionDefinition(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// ListActions lists action definitions for a namespace.
func (s *Service) ListActions(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, namespaceID xid.ID) ([]*core.ActionDefinition, error) {
	return s.repo.ListActionDefinitions(ctx, namespaceID)
}

// DeleteAction deletes an action definition.
func (s *Service) DeleteAction(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, actionID xid.ID) error {
	return s.repo.DeleteActionDefinition(ctx, actionID)
}

// =============================================================================
// NAMESPACE OPERATIONS
// =============================================================================

// CreateNamespace creates a new namespace.
func (s *Service) CreateNamespace(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, req *handlers.CreateNamespaceRequest) (*core.Namespace, error) {
	now := time.Now()
	ns := &core.Namespace{
		ID:                 xid.New(),
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		Name:               req.Name,
		Description:        req.Description,
		InheritPlatform:    req.InheritPlatform,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if req.TemplateID != "" {
		templateID, err := xid.FromString(req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("invalid template ID: %w", err)
		}

		ns.TemplateID = &templateID
	}

	if err := s.repo.CreateNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	s.createAuditEvent(ctx, appID, envID, orgID, userID, "create_namespace", "namespace", ns.ID, nil, map[string]any{
		"name": ns.Name,
	})

	return ns, nil
}

// GetNamespace retrieves a namespace by ID.
func (s *Service) GetNamespace(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, namespaceID xid.ID) (*core.Namespace, error) {
	ns, err := s.repo.GetNamespace(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	if ns == nil {
		return nil, errs.NotFound("namespace not found")
	}

	// Verify namespace belongs to scope
	if ns.AppID != appID || ns.EnvironmentID != envID {
		return nil, errs.NotFound("namespace not found")
	}

	return ns, nil
}

// ListNamespaces lists namespaces for an app/env/org.
func (s *Service) ListNamespaces(ctx context.Context, appID, envID xid.ID, orgID *xid.ID) ([]*core.Namespace, error) {
	return s.repo.ListNamespaces(ctx, appID, envID, orgID)
}

// UpdateNamespace updates an existing namespace.
func (s *Service) UpdateNamespace(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, namespaceID xid.ID, req *handlers.UpdateNamespaceRequest) (*core.Namespace, error) {
	ns, err := s.GetNamespace(ctx, appID, envID, orgID, namespaceID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		ns.Name = req.Name
	}

	if req.Description != "" {
		ns.Description = req.Description
	}

	ns.UpdatedAt = time.Now()

	if err := s.repo.UpdateNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("failed to update namespace: %w", err)
	}

	return ns, nil
}

// DeleteNamespace deletes a namespace.
func (s *Service) DeleteNamespace(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, namespaceID xid.ID) error {
	// Check if namespace has policies
	policies, err := s.repo.ListPolicies(ctx, appID, envID, orgID, storage.PolicyFilters{
		NamespaceID: &namespaceID,
		Limit:       1,
	})
	if err != nil {
		return fmt.Errorf("failed to check namespace policies: %w", err)
	}

	if len(policies) > 0 {
		return errs.BadRequest("cannot delete namespace with existing policies")
	}

	return s.repo.DeleteNamespace(ctx, namespaceID)
}

// CreateDefaultNamespace creates a default namespace for a new app/env or organization.
func (s *Service) CreateDefaultNamespace(ctx context.Context, appID, envID xid.ID, orgID *xid.ID) error {
	req := &handlers.CreateNamespaceRequest{
		Name:            "default",
		Description:     "Default permission namespace",
		InheritPlatform: true,
	}

	_, err := s.CreateNamespace(ctx, appID, envID, orgID, xid.NilID(), req)

	return err
}

// =============================================================================
// EVALUATION OPERATIONS (THE HEART OF PERMISSIONS)
// =============================================================================

// Evaluate evaluates a permission check - THE CORE FEATURE.
func (s *Service) Evaluate(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, req *handlers.EvaluateRequest) (*engine.Decision, error) {
	startTime := time.Now()

	// 1. Build evaluation context
	evalCtx := &engine.EvaluationContext{
		Principal: req.Principal,
		Resource:  req.Resource,
		Request:   req.Request,
		Action:    req.Action,
	}

	// Add user ID to principal if not present
	if evalCtx.Principal == nil {
		evalCtx.Principal = make(map[string]any)
	}

	if _, ok := evalCtx.Principal["id"]; !ok && !userID.IsNil() {
		evalCtx.Principal["id"] = userID.String()
	}

	// 2. Find applicable policies
	policies, err := s.getApplicablePolicies(ctx, appID, envID, orgID, req.ResourceType, req.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable policies: %w", err)
	}

	if len(policies) == 0 {
		// No policies = default deny
		return &engine.Decision{
			Allowed:           false,
			EvaluatedPolicies: 0,
			EvaluationTime:    time.Since(startTime),
		}, nil
	}

	// 3. Evaluate policies
	decision, err := s.evaluator.Evaluate(ctx, policies, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	decision.EvaluationTime = time.Since(startTime)

	s.logger.Debug("permission evaluated",
		forge.F("allowed", decision.Allowed),
		forge.F("resourceType", req.ResourceType),
		forge.F("action", req.Action),
		forge.F("evaluatedPolicies", decision.EvaluatedPolicies),
		forge.F("evaluationTimeMs", float64(decision.EvaluationTime.Microseconds())/1000),
	)

	return decision, nil
}

// EvaluateBatch evaluates multiple permission checks efficiently.
func (s *Service) EvaluateBatch(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, req *handlers.BatchEvaluateRequest) ([]*handlers.BatchEvaluationResult, error) {
	results := make([]*handlers.BatchEvaluationResult, len(req.Requests))

	// Use goroutines for parallel evaluation
	var wg sync.WaitGroup

	errCh := make(chan error, len(req.Requests))

	for i, evalReq := range req.Requests {
		wg.Add(1)

		go func(idx int, r handlers.EvaluateRequest) {
			defer wg.Done()

			decision, err := s.Evaluate(ctx, appID, envID, orgID, userID, &r)
			if err != nil {
				errCh <- fmt.Errorf("request %d failed: %w", idx, err)

				results[idx] = &handlers.BatchEvaluationResult{
					ResourceType: r.ResourceType,
					ResourceID:   r.ResourceID,
					Action:       r.Action,
					Allowed:      false,
					Error:        err.Error(),
				}

				return
			}

			results[idx] = &handlers.BatchEvaluationResult{
				ResourceType:     r.ResourceType,
				ResourceID:       r.ResourceID,
				Action:           r.Action,
				Allowed:          decision.Allowed,
				EvaluationTimeMs: float64(decision.EvaluationTime.Microseconds()) / 1000,
			}
		}(i, evalReq)
	}

	wg.Wait()
	close(errCh)

	return results, nil
}

// getApplicablePolicies retrieves and compiles policies applicable to the request.
func (s *Service) getApplicablePolicies(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, resourceType, action string) ([]*engine.CompiledPolicy, error) {
	// Build cache key
	cacheKey := buildPolicyCacheKey(appID, envID, orgID, resourceType, action)

	// Check in-memory compiled cache first
	s.compiledPoliciesMu.RLock()

	if policies, ok := s.policyIndex[cacheKey]; ok {
		s.compiledPoliciesMu.RUnlock()

		return policies, nil
	}

	s.compiledPoliciesMu.RUnlock()

	// Fetch from database
	dbPolicies, err := s.repo.GetPoliciesByResourceType(ctx, appID, envID, orgID, resourceType)
	if err != nil {
		return nil, err
	}

	// Filter by action and compile
	var compiled []*engine.CompiledPolicy

	for _, policy := range dbPolicies {
		if !policy.Enabled {
			continue
		}

		// Check if action matches
		actionMatches := false

		for _, a := range policy.Actions {
			if a == "*" || a == action {
				actionMatches = true

				break
			}
		}

		if !actionMatches {
			continue
		}

		// Compile policy
		cp, err := s.compiler.Compile(policy)
		if err != nil {
			s.logger.Warn("failed to compile policy",
				forge.F("policyId", policy.ID.String()),
				forge.F("error", err.Error()),
			)

			continue
		}

		compiled = append(compiled, cp)
	}

	// Cache compiled policies
	if len(compiled) > 0 {
		s.compiledPoliciesMu.Lock()
		s.policyIndex[cacheKey] = compiled
		s.compiledPoliciesMu.Unlock()
	}

	return compiled, nil
}

// =============================================================================
// TEMPLATE OPERATIONS
// =============================================================================

// ListTemplates lists available policy templates.
func (s *Service) ListTemplates(ctx context.Context) ([]*core.PolicyTemplate, error) {
	return getBuiltInTemplates(), nil
}

// GetTemplate retrieves a specific policy template.
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*core.PolicyTemplate, error) {
	templates := getBuiltInTemplates()
	for _, t := range templates {
		if t.ID == templateID {
			return t, nil
		}
	}

	return nil, fmt.Errorf("template not found: %s", templateID)
}

// InstantiateTemplate creates a policy from a template.
func (s *Service) InstantiateTemplate(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, userID xid.ID, templateID string, req *handlers.InstantiateTemplateRequest) (*core.Policy, error) {
	// Get template
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Substitute parameters in expression
	expression := template.Expression
	for _, param := range template.Parameters {
		if value, ok := req.Parameters[param.Name]; ok {
			expression = substituteParameter(expression, param.Name, value)
		} else if param.Required {
			return nil, fmt.Errorf("missing required parameter: %s", param.Name)
		} else if param.DefaultValue != nil {
			expression = substituteParameter(expression, param.Name, param.DefaultValue)
		}
	}

	// Create policy from template
	createReq := &handlers.CreatePolicyRequest{
		NamespaceID:  req.NamespaceID,
		Name:         req.Name,
		Description:  template.Description,
		Expression:   expression,
		ResourceType: req.ResourceType,
		Actions:      req.Actions,
		Priority:     req.Priority,
	}

	return s.CreatePolicy(ctx, appID, envID, orgID, userID, createReq)
}

// =============================================================================
// MIGRATION OPERATIONS
// =============================================================================

// MigrateFromRBAC migrates RBAC policies to permissions.
func (s *Service) MigrateFromRBAC(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, req *handlers.MigrateRBACRequest) (*core.MigrationStatus, error) {
	if s.migrationService == nil {
		return nil, errs.InternalServerErrorWithMessage("migration service not configured")
	}

	startTime := time.Now()

	// Get the user ID from context for audit purposes
	var userID xid.ID
	// Note: userID should be passed from handler if needed

	// Execute the migration
	result, err := s.migrationService.MigrateAll(ctx, appID, envID, orgID, userID)
	if err != nil {
		return &core.MigrationStatus{
			AppID:              appID,
			EnvironmentID:      envID,
			UserOrganizationID: orgID,
			Status:             "failed",
			StartedAt:          startTime,
			Errors:             []string{err.Error()},
		}, err
	}

	// Convert migration result to MigrationStatus
	completedAt := time.Now()

	// Convert migration errors to string slice
	var errorStrings []string
	for _, e := range result.Errors {
		errorStrings = append(errorStrings, e.Error)
	}

	status := &core.MigrationStatus{
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		Status:             "completed",
		StartedAt:          startTime,
		CompletedAt:        &completedAt,
		TotalPolicies:      result.TotalPolicies,
		MigratedCount:      result.MigratedPolicies,
		FailedCount:        result.FailedPolicies,
		ValidationPassed:   result.FailedPolicies == 0,
		Errors:             errorStrings,
	}

	s.logger.Info("RBAC migration completed",
		forge.F("app_id", appID.String()),
		forge.F("env_id", envID.String()),
		forge.F("total", result.TotalPolicies),
		forge.F("migrated", result.MigratedPolicies),
		forge.F("failed", result.FailedPolicies),
	)

	return status, nil
}

// GetMigrationStatus retrieves migration status.
func (s *Service) GetMigrationStatus(ctx context.Context, appID, envID xid.ID, orgID *xid.ID) (*core.MigrationStatus, error) {
	// Check if there's an ongoing or completed migration
	// For now, return a basic status based on existing policies
	policies, total, err := s.ListPolicies(ctx, appID, envID, orgID, map[string]any{"limit": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check migration status: %w", err)
	}

	status := "not_started"
	if total > 0 && len(policies) > 0 {
		status = "completed"
	}

	return &core.MigrationStatus{
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		Status:             status,
		TotalPolicies:      total,
		MigratedCount:      total,
	}, nil
}

// =============================================================================
// AUDIT & ANALYTICS OPERATIONS
// =============================================================================

// ListAuditEvents lists audit log entries.
func (s *Service) ListAuditEvents(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, filters map[string]any) ([]*core.AuditEvent, int, error) {
	af := storage.AuditFilters{
		Limit:  100,
		Offset: 0,
	}

	if limit, ok := filters["limit"].(int); ok {
		af.Limit = limit
	}

	if offset, ok := filters["offset"].(int); ok {
		af.Offset = offset
	}

	events, err := s.repo.ListAuditEvents(ctx, appID, envID, orgID, af)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit events: %w", err)
	}

	return events, len(events), nil
}

// GetAnalytics retrieves analytics data.
func (s *Service) GetAnalytics(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, timeRange map[string]any) (*handlers.AnalyticsSummary, error) {
	// Get policy counts
	policies, totalPolicies, err := s.ListPolicies(ctx, appID, envID, orgID, map[string]any{})
	if err != nil {
		s.logger.Warn("failed to get policies for analytics", forge.F("error", err.Error()))
	}

	// Count active policies
	activePolicies := 0

	for _, p := range policies {
		if p.Enabled {
			activePolicies++
		}
	}

	// Get evaluation stats from repository
	stats, err := s.repo.GetEvaluationStats(ctx, appID, envID, orgID, timeRange)
	if err != nil {
		s.logger.Warn("failed to get evaluation stats", forge.F("error", err.Error()))
		// Return basic analytics without stats
		return &handlers.AnalyticsSummary{
			TotalPolicies:    totalPolicies,
			ActivePolicies:   activePolicies,
			TotalEvaluations: 0,
			AllowedCount:     0,
			DeniedCount:      0,
			AvgLatencyMs:     0,
			CacheHitRate:     0,
		}, nil
	}

	// Calculate cache hit rate
	var cacheHitRate float64
	if stats.TotalEvaluations > 0 {
		cacheHitRate = float64(stats.CacheHits) / float64(stats.TotalEvaluations) * 100
	}

	return &handlers.AnalyticsSummary{
		TotalPolicies:    totalPolicies,
		ActivePolicies:   activePolicies,
		TotalEvaluations: stats.TotalEvaluations,
		AllowedCount:     stats.AllowedCount,
		DeniedCount:      stats.DeniedCount,
		AvgLatencyMs:     stats.AvgLatencyMs,
		CacheHitRate:     cacheHitRate,
	}, nil
}

// =============================================================================
// CACHE OPERATIONS
// =============================================================================

// WarmCache pre-loads and compiles all active policies for a given scope into the cache
// This is called during plugin initialization to ensure fast permission evaluation from the start.
func (s *Service) WarmCache(ctx context.Context, appID, envID xid.ID, orgID *xid.ID) error {
	if s.compiler == nil {
		return errs.InternalServerErrorWithMessage("CEL compiler not initialized")
	}

	// Fetch all enabled policies for the scope
	enabledFilter := true

	policies, err := s.repo.ListPolicies(ctx, appID, envID, orgID, storage.PolicyFilters{
		Enabled: &enabledFilter,
		Limit:   1000, // Reasonable limit for initial cache warm
	})
	if err != nil {
		return fmt.Errorf("failed to fetch policies for cache warming: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Debug("no policies found for cache warming")

		return nil
	}

	// Compile and cache each policy
	compiledCount := 0
	failedCount := 0

	for _, policy := range policies {
		// Compile the policy (core.Policy is already the correct type)
		compiled, err := s.compiler.Compile(policy)
		if err != nil {
			s.logger.Warn("failed to compile policy during cache warming",
				forge.F("policy_id", policy.ID.String()),
				forge.F("error", err.Error()),
			)

			failedCount++

			continue
		}

		// Add to cache
		s.cacheCompiledPolicy(compiled)

		compiledCount++
	}

	s.logger.Debug("cache warming completed",
		forge.F("total_policies", len(policies)),
		forge.F("compiled", compiledCount),
		forge.F("failed", failedCount),
		forge.F("app_id", appID.String()),
		forge.F("env_id", envID.String()),
	)

	return nil
}

// WarmCacheForAllApps warms the cache for all apps and environments
// This should be called sparingly as it can be resource intensive.
func (s *Service) WarmCacheForAllApps(ctx context.Context) error {
	// Fetch all enabled policies without scope restriction
	enabledFilter := true

	policies, err := s.repo.ListPolicies(ctx, xid.NilID(), xid.NilID(), nil, storage.PolicyFilters{
		Enabled: &enabledFilter,
		Limit:   5000, // Higher limit for global warm
	})
	if err != nil {
		return fmt.Errorf("failed to fetch policies for global cache warming: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Debug("no policies found for global cache warming")

		return nil
	}

	compiledCount := 0
	failedCount := 0

	for _, policy := range policies {
		compiled, err := s.compiler.Compile(policy)
		if err != nil {
			failedCount++

			continue
		}

		s.cacheCompiledPolicy(compiled)

		compiledCount++
	}

	s.logger.Debug("global cache warming completed",
		forge.F("total_policies", len(policies)),
		forge.F("compiled", compiledCount),
		forge.F("failed", failedCount),
	)

	return nil
}

// InvalidateUserCache invalidates the cache for a specific user.
func (s *Service) InvalidateUserCache(ctx context.Context, userID xid.ID) error {
	// Clear compiled policies that might reference this user
	// In practice, you'd have a user->policies mapping
	return nil
}

// InvalidateAppCache invalidates the cache for a specific app.
func (s *Service) InvalidateAppCache(ctx context.Context, appID xid.ID) error {
	s.compiledPoliciesMu.Lock()
	defer s.compiledPoliciesMu.Unlock()

	// Clear all compiled policies for this app
	for key := range s.policyIndex {
		if keyBelongsToApp(key, appID) {
			delete(s.policyIndex, key)
		}
	}

	if s.cache != nil {
		return s.cache.DeleteByApp(ctx, appID)
	}

	return nil
}

// InvalidateEnvironmentCache invalidates the cache for a specific environment.
func (s *Service) InvalidateEnvironmentCache(ctx context.Context, appID, envID xid.ID) error {
	s.compiledPoliciesMu.Lock()
	defer s.compiledPoliciesMu.Unlock()

	// Clear all compiled policies for this environment
	for key := range s.policyIndex {
		if keyBelongsToEnv(key, appID, envID) {
			delete(s.policyIndex, key)
		}
	}

	if s.cache != nil {
		return s.cache.DeleteByEnvironment(ctx, appID, envID)
	}

	return nil
}

// InvalidateOrganizationCache invalidates the cache for a specific organization.
func (s *Service) InvalidateOrganizationCache(ctx context.Context, appID, envID, orgID xid.ID) error {
	s.compiledPoliciesMu.Lock()
	defer s.compiledPoliciesMu.Unlock()

	// Clear all compiled policies for this org
	for key := range s.policyIndex {
		if keyBelongsToOrg(key, appID, envID, orgID) {
			delete(s.policyIndex, key)
		}
	}

	if s.cache != nil {
		return s.cache.DeleteByOrganization(ctx, appID, envID, orgID)
	}

	return nil
}

// =============================================================================
// LIFECYCLE OPERATIONS
// =============================================================================

// Migrate runs database migrations.
func (s *Service) Migrate(ctx context.Context) error {
	// Migrations are handled by the plugin's Migrate method
	return nil
}

// Shutdown gracefully shuts down the service.
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down permissions service")

	return nil
}

// Health checks service health.
func (s *Service) Health(ctx context.Context) error {
	if s.compiler == nil {
		return errs.InternalServerErrorWithMessage("CEL compiler not initialized")
	}

	if s.db == nil {
		return errs.InternalServerErrorWithMessage("database not initialized")
	}

	return nil
}

// =============================================================================
// INTERNAL HELPER METHODS
// =============================================================================

// cacheCompiledPolicy adds a compiled policy to the in-memory cache.
func (s *Service) cacheCompiledPolicy(policy *engine.CompiledPolicy) {
	s.compiledPoliciesMu.Lock()
	defer s.compiledPoliciesMu.Unlock()

	s.compiledPolicies[policy.PolicyID.String()] = policy

	// Also add to index by resource type
	for _, action := range policy.Actions {
		var orgIDStr string
		if policy.UserOrganizationID != nil {
			orgIDStr = policy.UserOrganizationID.String()
		}

		key := fmt.Sprintf("%s:%s:%s:%s:%s",
			policy.AppID.String(),
			policy.EnvironmentID.String(),
			orgIDStr,
			policy.ResourceType,
			action,
		)
		s.policyIndex[key] = append(s.policyIndex[key], policy)
	}
}

// removeCompiledPolicy removes a policy from the in-memory cache.
func (s *Service) removeCompiledPolicy(policyID string) {
	s.compiledPoliciesMu.Lock()
	defer s.compiledPoliciesMu.Unlock()

	policy, ok := s.compiledPolicies[policyID]
	if !ok {
		return
	}

	delete(s.compiledPolicies, policyID)

	// Remove from index
	for key, policies := range s.policyIndex {
		var filtered []*engine.CompiledPolicy

		for _, p := range policies {
			if p.PolicyID.String() != policyID {
				filtered = append(filtered, p)
			}
		}

		if len(filtered) == 0 {
			delete(s.policyIndex, key)
		} else {
			s.policyIndex[key] = filtered
		}
	}

	_ = policy // Suppress unused variable warning
}

// createAuditEvent creates an audit log entry.
func (s *Service) createAuditEvent(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, actorID xid.ID, action, resourceType string, resourceID xid.ID, oldValue, newValue map[string]any) {
	event := &core.AuditEvent{
		ID:                 xid.New(),
		AppID:              appID,
		EnvironmentID:      envID,
		UserOrganizationID: orgID,
		ActorID:            actorID,
		Action:             action,
		ResourceType:       resourceType,
		ResourceID:         resourceID,
		OldValue:           oldValue,
		NewValue:           newValue,
		Timestamp:          time.Now(),
	}

	if err := s.repo.CreateAuditEvent(ctx, event); err != nil {
		s.logger.Warn("failed to create audit event", forge.F("error", err.Error()))
	}
}

// buildPolicyCacheKey builds a cache key for policies.
func buildPolicyCacheKey(appID, envID xid.ID, orgID *xid.ID, resourceType, action string) string {
	var orgIDStr string
	if orgID != nil {
		orgIDStr = orgID.String()
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s", appID.String(), envID.String(), orgIDStr, resourceType, action)
}

// keyBelongsToApp checks if a cache key belongs to an app.
func keyBelongsToApp(key string, appID xid.ID) bool {
	return len(key) >= 20 && key[:20] == appID.String()
}

// keyBelongsToEnv checks if a cache key belongs to an environment.
func keyBelongsToEnv(key string, appID, envID xid.ID) bool {
	prefix := appID.String() + ":" + envID.String()

	return len(key) >= len(prefix) && key[:len(prefix)] == prefix
}

// keyBelongsToOrg checks if a cache key belongs to an organization.
func keyBelongsToOrg(key string, appID, envID, orgID xid.ID) bool {
	prefix := appID.String() + ":" + envID.String() + ":" + orgID.String()

	return len(key) >= len(prefix) && key[:len(prefix)] == prefix
}

// convertToResourceAttributes converts request attributes to core attributes.
func convertToResourceAttributes(attrs []handlers.ResourceAttributeRequest) []core.ResourceAttribute {
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

// substituteParameter replaces a parameter in an expression.
func substituteParameter(expression, paramName string, value any) string {
	// Simple string replacement - in production, use proper templating
	placeholder := fmt.Sprintf("{{%s}}", paramName)

	return replaceAll(expression, placeholder, fmt.Sprintf("%v", value))
}

// replaceAll replaces all occurrences of old with new in s.
func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}

		s = s[:idx] + new + s[idx+len(old):]
	}

	return s
}

// indexOf finds the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// getBuiltInTemplates returns the built-in policy templates.
func getBuiltInTemplates() []*core.PolicyTemplate {
	return []*core.PolicyTemplate{
		{
			ID:          "owner_only",
			Name:        "Owner Only",
			Description: "Only the resource owner can perform the action",
			Category:    "ownership",
			Expression:  `resource.owner == principal.id`,
			Parameters:  []core.TemplateParameter{},
			Examples:    []string{`resource.owner == principal.id`},
		},
		{
			ID:          "admin_or_owner",
			Name:        "Admin or Owner",
			Description: "Admin users or the resource owner can perform the action",
			Category:    "ownership",
			Expression:  `has_role("admin") || resource.owner == principal.id`,
			Parameters:  []core.TemplateParameter{},
			Examples:    []string{`has_role("admin") || resource.owner == principal.id`},
		},
		{
			ID:          "role_based",
			Name:        "Role Based",
			Description: "Users with a specific role can perform the action",
			Category:    "role",
			Expression:  `has_role("{{role}}")`,
			Parameters: []core.TemplateParameter{
				{
					Name:        "role",
					Type:        "string",
					Description: "The required role name",
					Required:    true,
				},
			},
			Examples: []string{`has_role("editor")`, `has_role("moderator")`},
		},
		{
			ID:          "any_role",
			Name:        "Any Role",
			Description: "Users with any of the specified roles can perform the action",
			Category:    "role",
			Expression:  `has_any_role({{roles}})`,
			Parameters: []core.TemplateParameter{
				{
					Name:        "roles",
					Type:        "array",
					Description: "List of role names (any matches)",
					Required:    true,
				},
			},
			Examples: []string{`has_any_role(["admin", "moderator"])`},
		},
		{
			ID:          "team_members",
			Name:        "Team Members Only",
			Description: "Only team members can access the resource",
			Category:    "team",
			Expression:  `principal.team_id == resource.team_id`,
			Parameters:  []core.TemplateParameter{},
			Examples:    []string{`principal.team_id == resource.team_id`},
		},
		{
			ID:          "public_or_owner",
			Name:        "Public or Owner",
			Description: "Anyone can access public resources, or the owner can access private ones",
			Category:    "visibility",
			Expression:  `resource.visibility == "public" || resource.owner == principal.id`,
			Parameters:  []core.TemplateParameter{},
			Examples:    []string{`resource.visibility == "public" || resource.owner == principal.id`},
		},
		{
			ID:          "business_hours",
			Name:        "Business Hours Only",
			Description: "Access is only allowed during business hours",
			Category:    "time",
			Expression:  `is_weekday() && in_time_range("{{start}}", "{{end}}")`,
			Parameters: []core.TemplateParameter{
				{
					Name:         "start",
					Type:         "string",
					Description:  "Start time (HH:MM format)",
					Required:     false,
					DefaultValue: "09:00",
				},
				{
					Name:         "end",
					Type:         "string",
					Description:  "End time (HH:MM format)",
					Required:     false,
					DefaultValue: "17:00",
				},
			},
			Examples: []string{`is_weekday() && in_time_range("09:00", "17:00")`},
		},
		{
			ID:          "ip_allowlist",
			Name:        "IP Allowlist",
			Description: "Access is restricted to specific IP ranges",
			Category:    "network",
			Expression:  `ip_in_range({{cidrs}})`,
			Parameters: []core.TemplateParameter{
				{
					Name:        "cidrs",
					Type:        "array",
					Description: "List of allowed CIDR ranges",
					Required:    true,
				},
			},
			Examples: []string{`ip_in_range(["10.0.0.0/8", "192.168.0.0/16"])`},
		},
	}
}
