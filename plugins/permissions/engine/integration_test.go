package engine

import (
	"context"
	"testing"
	
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/engine/providers"
)

// TestEndToEnd_WithAttributeResolution tests the complete flow:
// 1. Compile policy
// 2. Set up attribute providers
// 3. Evaluate with automatic attribute enrichment
func TestEndToEnd_WithAttributeResolution(t *testing.T) {
	// Step 1: Create compiler and evaluator
	compiler, err := NewCompiler(DefaultCompilerConfig())
	require.NoError(t, err)
	
	// Step 2: Set up attribute resolver with mock providers
	cache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(cache)
	
	// Add mock user service
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:         "user_alice",
		Email:      "alice@example.com",
		Name:       "Alice Admin",
		Roles:      []string{"admin", "developer"},
		OrgID:      "org_acme",
		Department: "Engineering",
	})
	userService.AddUser(&providers.User{
		ID:    "user_bob",
		Email: "bob@example.com",
		Name:  "Bob User",
		Roles: []string{"viewer"},
		OrgID: "org_acme",
	})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	// Add mock resource service
	resourceService := providers.NewMockResourceService()
	resourceService.AddResource(&providers.Resource{
		ID:           "doc_123",
		Type:         "document",
		Name:         "Secret Plans",
		Owner:        "user_alice",
		OrgID:        "org_acme",
		Visibility:   "private",
		Confidential: "internal",
	})
	resourceProvider := providers.NewResourceAttributeProvider(resourceService)
	resolver.RegisterProvider(resourceProvider)
	
	// Add context provider
	contextProvider := providers.NewContextAttributeProvider()
	resolver.RegisterProvider(contextProvider)
	
	// Create evaluator with resolver
	evaluatorConfig := DefaultEvaluatorConfig()
	evaluatorConfig.AttributeResolver = resolver
	evaluator := NewEvaluator(evaluatorConfig)
	
	// Step 3: Define policies
	tests := []struct {
		name          string
		policy        *core.Policy
		evalCtx       *EvaluationContext
		expectAllowed bool
		description   string
	}{
		{
			name: "admin_can_access_any_document",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Admin Access",
				Expression:   `principal.roles.exists(r, r == "admin")`,
				ResourceType: "document",
				Actions:      []string{"read"},
			},
			evalCtx: &EvaluationContext{
				// Only provide minimal info - resolver will fetch the rest
				Principal: map[string]interface{}{
					"id": "user_alice", // Resolver will fetch roles, email, etc.
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123", // Resolver will fetch owner, visibility, etc.
				},
				Action: "read",
			},
			expectAllowed: true,
			description:   "Alice is admin (fetched from user service), should be allowed",
		},
		{
			name: "non_admin_cannot_access",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Admin Only",
				Expression:   `principal.roles.exists(r, r == "admin")`,
				ResourceType: "document",
				Actions:      []string{"read"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id": "user_bob", // Bob is viewer, not admin
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123",
				},
				Action: "read",
			},
			expectAllowed: false,
			description:   "Bob is not admin (fetched from user service), should be denied",
		},
		{
			name: "owner_can_access_their_document",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Owner Access",
				Expression:   `resource.owner == principal.id`,
				ResourceType: "document",
				Actions:      []string{"read", "write"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id": "user_alice",
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123", // Owner is user_alice (fetched from resource service)
				},
				Action: "read",
			},
			expectAllowed: true,
			description:   "Alice owns doc_123 (fetched from resource service), should be allowed",
		},
		{
			name: "non_owner_cannot_access",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Owner Only",
				Expression:   `resource.owner == principal.id`,
				ResourceType: "document",
				Actions:      []string{"read"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id": "user_bob",
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123", // Owner is user_alice, not bob
				},
				Action: "read",
			},
			expectAllowed: false,
			description:   "Bob doesn't own doc_123 (fetched from resource service), should be denied",
		},
		{
			name: "same_org_members_can_view_internal_docs",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Org Internal Access",
				Expression:   `resource.confidential == "internal" && principal.org_id == resource.org_id`,
				ResourceType: "document",
				Actions:      []string{"read"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id": "user_bob", // Bob is in org_acme
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123", // Doc is internal and in org_acme
				},
				Action: "read",
			},
			expectAllowed: true,
			description:   "Bob and doc_123 are in same org (fetched from services), should be allowed",
		},
		{
			name: "admin_or_owner_can_write",
			policy: &core.Policy{
				ID:           xid.New(),
				OrgID:        xid.New(),
				NamespaceID:  xid.New(),
				Name:         "Admin or Owner Write",
				Expression:   `principal.roles.exists(r, r == "admin") || resource.owner == principal.id`,
				ResourceType: "document",
				Actions:      []string{"write"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id": "user_bob", // Bob is not admin and not owner
				},
				Resource: map[string]interface{}{
					"type": "document",
					"id":   "doc_123", // Owned by alice
				},
				Action: "write",
			},
			expectAllowed: false,
			description:   "Bob is neither admin nor owner (all fetched), should be denied",
		},
	}
	
	ctx := context.Background()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile policy
			compiled, err := compiler.Compile(tt.policy)
			require.NoError(t, err, "policy should compile")
			
			// Evaluate with automatic attribute enrichment
			decision, err := evaluator.Evaluate(ctx, []*CompiledPolicy{compiled}, tt.evalCtx)
			require.NoError(t, err, "evaluation should not error")
			require.NotNil(t, decision, "decision should not be nil")
			
			// Check result
			assert.Equal(t, tt.expectAllowed, decision.Allowed, tt.description)
			
			// Verify that attributes were enriched
			if tt.expectAllowed {
				// If allowed, at least one policy should have matched
				assert.Greater(t, len(decision.MatchedPolicies), 0, "should have matched policies")
			}
			
			// Verify evaluation metrics
			assert.Greater(t, decision.EvaluatedPolicies, 0, "should have evaluated at least one policy")
			assert.Greater(t, decision.EvaluationTime, int64(0), "evaluation time should be recorded")
		})
	}
}

// TestAttributeEnrichment_OnlyWhenNeeded verifies that attribute enrichment
// only fetches missing attributes, not overwriting existing ones
func TestAttributeEnrichment_OnlyWhenNeeded(t *testing.T) {
	// Setup
	cache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(cache)
	
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:    "user_123",
		Name:  "From Service",
		Email: "service@example.com",
		Roles: []string{"admin"},
	})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	evaluatorConfig := DefaultEvaluatorConfig()
	evaluatorConfig.AttributeResolver = resolver
	evaluator := NewEvaluator(evaluatorConfig)
	
	ctx := context.Background()
	
	// Test 1: Principal has ID but no roles - should fetch from service
	evalCtx1 := &EvaluationContext{
		Principal: map[string]interface{}{
			"id": "user_123",
			// No roles - should be fetched
		},
		Resource: map[string]interface{}{},
		Action:   "read",
	}
	
	err := evaluator.EnrichEvaluationContext(ctx, evalCtx1)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, evalCtx1.Principal["roles"])
	assert.Equal(t, "From Service", evalCtx1.Principal["name"])
	
	// Test 2: Principal has ID and roles - should NOT overwrite
	evalCtx2 := &EvaluationContext{
		Principal: map[string]interface{}{
			"id":    "user_123",
			"name":  "Pre-existing Name",
			"roles": []string{"pre-existing-role"},
			// Already has roles - should NOT fetch
		},
		Resource: map[string]interface{}{},
		Action:   "read",
	}
	
	err = evaluator.EnrichEvaluationContext(ctx, evalCtx2)
	require.NoError(t, err)
	// Should keep pre-existing values
	assert.Equal(t, "Pre-existing Name", evalCtx2.Principal["name"])
	assert.Equal(t, []string{"pre-existing-role"}, evalCtx2.Principal["roles"])
}

// TestAttributeCaching verifies that attribute resolution uses caching
func TestAttributeCaching(t *testing.T) {
	// Setup
	cache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(cache)
	
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:    "user_123",
		Name:  "Alice",
		Roles: []string{"admin"},
	})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	evaluatorConfig := DefaultEvaluatorConfig()
	evaluatorConfig.AttributeResolver = resolver
	evaluator := NewEvaluator(evaluatorConfig)
	
	ctx := context.Background()
	
	// First evaluation - should fetch from service
	evalCtx1 := &EvaluationContext{
		Principal: map[string]interface{}{"id": "user_123"},
		Resource:  map[string]interface{}{},
		Action:    "read",
	}
	
	err := evaluator.EnrichEvaluationContext(ctx, evalCtx1)
	require.NoError(t, err)
	assert.Equal(t, "Alice", evalCtx1.Principal["name"])
	
	// Second evaluation - should use cache (verify by checking cache directly)
	evalCtx2 := &EvaluationContext{
		Principal: map[string]interface{}{"id": "user_123"},
		Resource:  map[string]interface{}{},
		Action:    "read",
	}
	
	err = evaluator.EnrichEvaluationContext(ctx, evalCtx2)
	require.NoError(t, err)
	assert.Equal(t, "Alice", evalCtx2.Principal["name"])
	
	// Verify cache was used by checking cache directly
	cached, found := cache.Get(ctx, "user:user_123")
	assert.True(t, found, "user should be in cache")
	assert.Equal(t, "Alice", cached["name"])
}

