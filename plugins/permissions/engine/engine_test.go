package engine

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/plugins/permissions/core"
)

func TestCompilerAndEvaluator_EndToEnd(t *testing.T) {
	// Create compiler
	compiler, err := NewCompiler(DefaultCompilerConfig())
	require.NoError(t, err)

	// Create evaluator
	evaluator := NewEvaluator(DefaultEvaluatorConfig())

	// Test cases
	tests := []struct {
		name        string
		policy      *core.Policy
		evalCtx     *EvaluationContext
		expectAllow bool
	}{
		{
			name: "owner can access",
			policy: &core.Policy{
				ID:                 xid.New(),
				AppID:              xid.New(),
				UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
				NamespaceID:        xid.New(),
				Name:               "Owner access",
				Expression:         `resource.owner == principal.id`,
				ResourceType:       "document",
				Actions:            []string{"read", "write"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{"id": "user_123"},
				Resource:  map[string]interface{}{"owner": "user_123"},
				Action:    "read",
			},
			expectAllow: true,
		},
		{
			name: "non-owner cannot access",
			policy: &core.Policy{
				ID:                 xid.New(),
				AppID:              xid.New(),
				UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
				NamespaceID:        xid.New(),
				Name:               "Owner access",
				Expression:         `resource.owner == principal.id`,
				ResourceType:       "document",
				Actions:            []string{"read", "write"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{"id": "user_456"},
				Resource:  map[string]interface{}{"owner": "user_123"},
				Action:    "read",
			},
			expectAllow: false,
		},
		{
			name: "admin or owner can access",
			policy: &core.Policy{
				ID:                 xid.New(),
				AppID:              xid.New(),
				UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
				NamespaceID:        xid.New(),
				Name:               "Admin or owner",
				// Use CEL's native .exists() instead of has_role() for now
				Expression:   `principal.roles.exists(r, r == "admin") || resource.owner == principal.id`,
				ResourceType: "document",
				Actions:      []string{"read", "write"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id":    "user_456",
					"roles": []string{"admin"},
				},
				Resource: map[string]interface{}{"owner": "user_123"},
				Action:   "read",
			},
			expectAllow: true,
		},
		{
			name: "team members can access",
			policy: &core.Policy{
				ID:                 xid.New(),
				AppID:              xid.New(),
				UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
				NamespaceID:        xid.New(),
				Name:               "Team access",
				Expression:         `principal.team_id == resource.team_id`,
				ResourceType:       "project",
				Actions:            []string{"read"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{
					"id":      "user_123",
					"team_id": "team_456",
				},
				Resource: map[string]interface{}{
					"id":      "project_789",
					"team_id": "team_456",
				},
				Action: "read",
			},
			expectAllow: true,
		},
		{
			name: "public resources accessible",
			policy: &core.Policy{
				ID:                 xid.New(),
				AppID:              xid.New(),
				UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
				NamespaceID:        xid.New(),
				Name:               "Public access",
				Expression:         `resource.visibility == "public"`,
				ResourceType:       "document",
				Actions:            []string{"read"},
			},
			evalCtx: &EvaluationContext{
				Principal: map[string]interface{}{"id": "user_123"},
				Resource: map[string]interface{}{
					"id":         "doc_456",
					"visibility": "public",
				},
				Action: "read",
			},
			expectAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile policy
			compiled, err := compiler.Compile(tt.policy)
			require.NoError(t, err)
			assert.NotNil(t, compiled)

			// Evaluate
			ctx := context.Background()
			decision, err := evaluator.Evaluate(ctx, []*CompiledPolicy{compiled}, tt.evalCtx)
			require.NoError(t, err)
			assert.NotNil(t, decision)
			assert.Equal(t, tt.expectAllow, decision.Allowed)

			if tt.expectAllow {
				assert.NotEmpty(t, decision.MatchedPolicies)
			}

			// Check performance
			assert.Less(t, decision.EvaluationTime, 10*time.Millisecond, "Evaluation should be fast")
		})
	}
}

func TestCompiler_Validate(t *testing.T) {
	compiler, err := NewCompiler(DefaultCompilerConfig())
	require.NoError(t, err)

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid expression",
			expression: `resource.owner == principal.id`,
			wantErr:    false,
		},
		{
			name:       "invalid syntax",
			expression: `resource.owner ==`,
			wantErr:    true,
		},
		{
			name:       "non-boolean return",
			expression: `principal.id`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.Validate(tt.expression)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompiler_CompileBatch(t *testing.T) {
	compiler, err := NewCompiler(DefaultCompilerConfig())
	require.NoError(t, err)

	policies := []*core.Policy{
		{
			ID:                 xid.New(),
			AppID:              xid.New(),
			UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
			NamespaceID:        xid.New(),
			Expression:         `resource.owner == principal.id`,
			ResourceType:       "document",
			Actions:            []string{"read"},
		},
		{
			ID:                 xid.New(),
			AppID:              xid.New(),
			UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
			NamespaceID:        xid.New(),
			Expression:         `has_role("admin")`,
			ResourceType:       "document",
			Actions:            []string{"write"},
		},
	}

	compiled, err := compiler.CompileBatch(policies)
	require.NoError(t, err)
	assert.Len(t, compiled, 2)
}

func TestEvaluator_MultiplePolicies(t *testing.T) {
	compiler, err := NewCompiler(DefaultCompilerConfig())
	require.NoError(t, err)

	evaluator := NewEvaluator(DefaultEvaluatorConfig())

	// Create multiple policies
	policies := []*core.Policy{
		{
			ID:                 xid.New(),
			AppID:              xid.New(),
			UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
			NamespaceID:        xid.New(),
			Expression:         `resource.owner == principal.id`,
			ResourceType:       "document",
			Actions:            []string{"read"},
			Priority:           100,
		},
		{
			ID:                 xid.New(),
			AppID:              xid.New(),
			UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
			NamespaceID:        xid.New(),
			// Use CEL's native .exists() instead of has_role() for now
			Expression:   `principal.roles.exists(r, r == "admin")`,
			ResourceType: "document",
			Actions:      []string{"read"},
			Priority:     200, // Higher priority
		},
	}

	// Compile policies
	compiled := make([]*CompiledPolicy, 0, len(policies))
	for _, p := range policies {
		c, err := compiler.Compile(p)
		require.NoError(t, err)
		compiled = append(compiled, c)
	}

	// Test: admin role matches (higher priority policy)
	evalCtx := &EvaluationContext{
		Principal: map[string]interface{}{
			"id":    "user_123",
			"roles": []string{"admin"},
		},
		Resource: map[string]interface{}{"owner": "user_456"},
		Action:   "read",
	}

	decision, err := evaluator.Evaluate(context.Background(), compiled, evalCtx)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.NotEmpty(t, decision.MatchedPolicies)
}

func BenchmarkEvaluator_SimplePolicy(b *testing.B) {
	compiler, _ := NewCompiler(DefaultCompilerConfig())
	evaluator := NewEvaluator(DefaultEvaluatorConfig())

	policy := &core.Policy{
		ID:                 xid.New(),
		AppID:              xid.New(),
		UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
		NamespaceID:        xid.New(),
		Expression:         `resource.owner == principal.id`,
		ResourceType:       "document",
		Actions:            []string{"read"},
	}

	compiled, _ := compiler.Compile(policy)

	evalCtx := &EvaluationContext{
		Principal: map[string]interface{}{"id": "user_123"},
		Resource:  map[string]interface{}{"owner": "user_123"},
		Action:    "read",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(ctx, []*CompiledPolicy{compiled}, evalCtx)
	}
}

func BenchmarkEvaluator_ComplexPolicy(b *testing.B) {
	compiler, _ := NewCompiler(DefaultCompilerConfig())
	evaluator := NewEvaluator(DefaultEvaluatorConfig())

	policy := &core.Policy{
		ID:                 xid.New(),
		AppID:              xid.New(),
		UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
		NamespaceID:        xid.New(),
		Expression:         `(resource.owner == principal.id || has_role("admin")) && resource.enabled == true && principal.department == "engineering"`,
		ResourceType:       "document",
		Actions:            []string{"read"},
	}

	compiled, _ := compiler.Compile(policy)

	evalCtx := &EvaluationContext{
		Principal: map[string]interface{}{
			"id":         "user_123",
			"roles":      []string{"member"},
			"department": "engineering",
		},
		Resource: map[string]interface{}{
			"owner":   "user_123",
			"enabled": true,
		},
		Action: "read",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(ctx, []*CompiledPolicy{compiled}, evalCtx)
	}
}

func BenchmarkEvaluator_1000Policies(b *testing.B) {
	compiler, _ := NewCompiler(DefaultCompilerConfig())
	evaluator := NewEvaluator(DefaultEvaluatorConfig())

	// Create 1000 policies
	policies := make([]*CompiledPolicy, 1000)
	for i := 0; i < 1000; i++ {
		policy := &core.Policy{
			ID:                 xid.New(),
			AppID:              xid.New(),
			UserOrganizationID: func() *xid.ID { id := xid.New(); return &id }(),
			NamespaceID:        xid.New(),
			Expression:         `resource.owner == principal.id`,
			ResourceType:       "document",
			Actions:            []string{"read"},
		}
		compiled, _ := compiler.Compile(policy)
		policies[i] = compiled
	}

	evalCtx := &EvaluationContext{
		Principal: map[string]interface{}{"id": "user_123"},
		Resource:  map[string]interface{}{"owner": "user_456"}, // No match
		Action:    "read",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(ctx, policies, evalCtx)
	}
}
